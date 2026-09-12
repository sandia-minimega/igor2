// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// A request that never completes must not be able to block shutdown indefinitely. This is
// what made every wedge recoverable only by SIGKILL, which destroyed the goroutine state
// that would have identified the cause.
func TestShutdownIsBoundedByAStuckRequest(t *testing.T) {
	stall := make(chan struct{})
	defer close(stall)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)

	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			<-stall // simulates a handler parked on dbAccess or an unbounded external call
		}),
	}
	go func() { _ = srv.Serve(ln) }()

	// get one request in flight and confirm the handler has been entered
	reqStarted := make(chan struct{})
	go func() {
		close(reqStarted)
		c := &http.Client{Timeout: 30 * time.Second}
		resp, rErr := c.Get("http://" + ln.Addr().String())
		if resp != nil {
			_ = resp.Body.Close()
		}
		_ = rErr
	}()
	<-reqStarted
	time.Sleep(200 * time.Millisecond)

	// Shutdown with an unbounded context would never return here. Bounded, it must give up.
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- srv.Shutdown(ctx) }()

	select {
	case sErr := <-done:
		assert.True(t, errors.Is(sErr, context.DeadlineExceeded),
			"a stuck request should make Shutdown hit its deadline, got: %v", sErr)
	case <-time.After(10 * time.Second):
		t.Fatal("Shutdown never returned; it is not bounded")
	}

	assert.NoError(t, srv.Close(), "force close must succeed after the deadline")
}

// withTestShutdownState shortens the grace period for the duration of a test and hands
// back a private shutdown channel and WaitGroup. Both are per-test: sharing the package
// globals would let one test's lingering wg.Wait() race the next test's wg.Add.
func withTestShutdownState(t *testing.T, grace time.Duration) (chan struct{}, *sync.WaitGroup) {
	t.Helper()

	prevGrace := shutdownGrace
	shutdownGrace = grace
	t.Cleanup(func() { shutdownGrace = prevGrace })

	return make(chan struct{}), &sync.WaitGroup{}
}

// waitForWorkers is what keeps runServer alive for the life of the process, so on a
// healthy server it must not return at all.
//
// A previous version bounded the wait unconditionally. It therefore returned after the
// grace period even with nothing shutting down, runServer fell through to closing the
// database and exiting, and Restart=always turned that into a restart loop once per grace
// period. This test exists because that regression reached a testbed.
func TestWaitForWorkersBlocksWhileServerIsRunning(t *testing.T) {
	testChan, workers := withTestShutdownState(t, 200*time.Millisecond)

	release := make(chan struct{})
	workers.Add(1)
	go func() {
		defer workers.Done()
		<-release
	}()
	defer close(release)

	returned := make(chan struct{})
	go func() {
		waitForWorkers(workers, testChan)
		close(returned)
	}()

	// Well past the grace period. The buggy version returned after 200ms.
	select {
	case <-returned:
		t.Fatal("waitForWorkers returned with no shutdown requested; runServer would fall " +
			"through to closing the database and exit, and systemd would restart it in a loop")
	case <-time.After(2 * time.Second):
		// correct: still blocked
	}

	// and it must return promptly once shutdown really is requested
	shutdownDeadline = time.Now().Add(shutdownGrace)
	close(testChan)

	select {
	case <-returned:
	case <-time.After(5 * time.Second):
		t.Fatal("waitForWorkers did not return after shutdown was requested")
	}
}

// Once shutdown is under way, waitForWorkers must give up on a manager that is parked and
// never reaches its shutdownChan select -- otherwise the database session is never closed.
func TestWaitForWorkersGivesUpOnAStuckWorker(t *testing.T) {
	testChan, workers := withTestShutdownState(t, 500*time.Millisecond)

	release := make(chan struct{})
	workers.Add(1)
	go func() {
		defer workers.Done()
		<-release // never signalled until after the assertion below
	}()
	defer close(release)

	shutdownDeadline = time.Now().Add(shutdownGrace)
	close(testChan)

	start := time.Now()
	done := make(chan struct{})
	go func() {
		waitForWorkers(workers, testChan)
		close(done)
	}()

	select {
	case <-done:
		assert.GreaterOrEqual(t, time.Since(start), shutdownGrace,
			"waitForWorkers returned early; it should have waited out its grace period")
	case <-time.After(shutdownGrace + 10*time.Second):
		t.Fatal("waitForWorkers never returned; shutdown is still unbounded")
	}
}

// The stages of shutdown share one budget. If the HTTP servers consume it, waitForWorkers
// must not then start a fresh grace period of its own.
func TestWaitForWorkersHonoursAnAlreadySpentBudget(t *testing.T) {
	testChan, workers := withTestShutdownState(t, 10*time.Second)

	release := make(chan struct{})
	workers.Add(1)
	go func() {
		defer workers.Done()
		<-release
	}()
	defer close(release)

	// deadline already in the past, as if both Shutdown calls had used the whole budget
	shutdownDeadline = time.Now().Add(-time.Second)
	close(testChan)

	done := make(chan struct{})
	go func() {
		waitForWorkers(workers, testChan)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("waitForWorkers waited again on an exhausted budget; shutdown stages are not sharing one deadline")
	}
}
