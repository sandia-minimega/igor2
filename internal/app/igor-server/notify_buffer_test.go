// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// shrinkNotifyChannels replaces the three notify channels with small ones filled to
// capacity, so the next send blocks, and restores the originals afterwards.
func shrinkNotifyChannels(t *testing.T, capacity int) {
	t.Helper()

	prevRes, prevAcct, prevGroup := resNotifyChan, acctNotifyChan, groupNotifyChan
	resNotifyChan = make(chan ResNotifyEvent, capacity)
	acctNotifyChan = make(chan AcctNotifyEvent, capacity)
	groupNotifyChan = make(chan GroupNotifyEvent, capacity)
	for i := 0; i < capacity; i++ {
		resNotifyChan <- ResNotifyEvent{}
		acctNotifyChan <- AcctNotifyEvent{}
		groupNotifyChan <- GroupNotifyEvent{}
	}
	t.Cleanup(func() {
		resNotifyChan, acctNotifyChan, groupNotifyChan = prevRes, prevAcct, prevGroup
	})
}

// Regression test for the notify lock inversion. Every notification used to be sent from
// inside the caller's dbAccess region. One goroutine serves all three channels and
// processResNotifyEvent takes dbAccess, so a producer blocking on a full channel was waiting
// on a consumer that could not drain without the lock the producer held.
//
// The locked region must now end regardless of how full the channels are.
func TestNotifyBufferReleasesLockBeforeSending(t *testing.T) {
	captureLog(t)
	shrinkNotifyChannels(t, 1) // holds one event, and starts full

	dequeued := make(chan struct{})
	releaseConsumer := make(chan struct{})
	producerHasLock := make(chan struct{})
	lockReleased := make(chan struct{})
	flushDone := make(chan struct{})

	// The consumer, mirroring processResNotifyEvent: take an event, then acquire dbAccess to
	// record it. It is parked between those two steps so the test controls the interleaving,
	// which is the state that made the old code deadlock.
	go func() {
		<-resNotifyChan
		close(dequeued)
		<-releaseConsumer

		lockedDbWrite(func() {})
		for range resNotifyChan { // keep draining so a correct producer can finish
			lockedDbWrite(func() {})
		}
	}()

	<-dequeued
	resNotifyChan <- ResNotifyEvent{} // full again, so the next send must block

	go func() {
		var notices notifyBuffer
		lockedDbWrite(func() {
			close(producerHasLock)
			notices.addRes(&ResNotifyEvent{})
		})
		close(lockReleased)

		notices.flush()
		close(flushDone)
	}()

	<-producerHasLock
	time.Sleep(100 * time.Millisecond) // give the producer time to reach any send
	close(releaseConsumer)             // the consumer now contends for dbAccess

	select {
	case <-lockReleased:
	case <-time.After(10 * time.Second):
		t.Fatal("notify lock inversion regression: the locked region never ended while the " +
			"notify channel was full. A send has moved back inside lockedDbWrite, so the " +
			"producer is blocked on a consumer that needs the lock the producer holds.")
	}

	select {
	case <-flushDone:
	case <-time.After(10 * time.Second):
		t.Fatal("flush never completed even though the consumer was draining")
	}

	assertDbAccessFree(t, "dbAccess still held after flushing notifications")
}

// A nil event means the email configuration disables that notification. The constructors
// return nil in that case, so the add helpers must swallow it rather than queue a zero value
// that the consumer would try to render.
func TestNotifyBufferIgnoresNilEvents(t *testing.T) {
	captureLog(t)
	shrinkNotifyChannels(t, 1)

	var notices notifyBuffer
	notices.addRes(nil)
	notices.addAcct(nil)
	notices.addGroup(nil)

	assert.Empty(t, notices.res)
	assert.Empty(t, notices.acct)
	assert.Empty(t, notices.group)

	// Flushing an empty buffer must not block on the already-full channels.
	done := make(chan struct{})
	go func() { notices.flush(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("flushing an empty buffer blocked; it must be a no-op")
	}
}

// The inverted sends were almost all indirect -- inside do* functions called from a handler's
// lockedDbWrite, not lexically inside it -- so checking for sends nested in a locked region
// would have caught none of them. Confine the sends instead: every notification must go
// through notifyBuffer.flush, which is documented to run with no lock held.
//
// sendExpirationWarnings is the one exception. It is reached from the scheduler
// (server.go, manageReservations) without dbAccess, so its direct send cannot invert.
func TestNotifyChannelSendsAreConfinedToFlush(t *testing.T) {
	channels := map[string]bool{
		"resNotifyChan":   true,
		"acctNotifyChan":  true,
		"groupNotifyChan": true,
	}
	// function name -> why it is allowed to send directly
	allowed := map[string]string{
		"flush":                  "the buffer's own dispatch point, called after the lock is released",
		"sendExpirationWarnings": "runs on the scheduler with no dbAccess held",
	}

	goFiles, err := filepath.Glob("*.go")
	require.NoError(t, err)
	require.NotEmpty(t, goFiles)

	fset := token.NewFileSet()
	scanned := 0
	for _, path := range goFiles {
		// Test fixtures legitimately fill the channels to set up a full-queue state; the
		// invariant being enforced here is about the server's own code.
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		scanned++

		file, parseErr := parser.ParseFile(fset, path, nil, 0)
		require.NoError(t, parseErr, "parsing %s", path)

		// Walk each top-level function separately so a violation can be attributed to one.
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			ast.Inspect(fn, func(n ast.Node) bool {
				send, isSend := n.(*ast.SendStmt)
				if !isSend {
					return true
				}
				ident, isIdent := send.Chan.(*ast.Ident)
				if !isIdent || !channels[ident.Name] {
					return true
				}
				if _, ok := allowed[fn.Name.Name]; ok {
					return true
				}
				t.Errorf("%s: %s sends directly on %s. Queue it with notifyBuffer.add* and let "+
					"the caller flush after its dbAccess region ends -- the notify consumer takes "+
					"dbAccess, so a send under the lock deadlocks the server.",
					fset.Position(send.Pos()), fn.Name.Name, ident.Name)
				return true
			})
		}
	}
	require.NotZero(t, scanned, "found no non-test sources to scan")
}
