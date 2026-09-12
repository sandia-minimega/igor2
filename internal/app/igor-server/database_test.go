// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// assertDbAccessFree fails the test if the global write mutex is still held, and leaves
// it unlocked either way so later tests are unaffected.
func assertDbAccessFree(t *testing.T, msg string) {
	t.Helper()
	if !dbAccess.TryLock() {
		t.Fatal(msg)
	}
	dbAccess.Unlock()
}

func TestLockedDbWriteReleasesOnNormalReturn(t *testing.T) {
	ran := false
	lockedDbWrite(func() { ran = true })

	assert.True(t, ran, "the function must actually run")
	assertDbAccessFree(t, "dbAccess still held after a normal return")
}

// The reason this helper exists. handleCreateReservations used to unlock with a bare
// statement after the call, which a panic unwound straight past -- leaving the global
// write mutex held by a goroutine that no longer existed. httprouter recovers the panic,
// so the server stayed up serving reads while every write blocked forever.
func TestLockedDbWriteReleasesOnPanic(t *testing.T) {
	func() {
		defer func() {
			assert.NotNil(t, recover(), "the panic must propagate to the caller")
		}()
		lockedDbWrite(func() {
			panic("simulated panic inside the locked region")
		})
	}()

	assertDbAccessFree(t, "dbAccess leaked: a panic inside the locked region did not release it")
}

// A leaked mutex is only observable through its effect on the next writer, so check that
// a second call can still acquire it after the first one panicked.
func TestLockedDbWriteUsableAfterPanic(t *testing.T) {
	func() {
		defer func() { _ = recover() }()
		lockedDbWrite(func() { panic("first call panics") })
	}()

	done := make(chan struct{})
	go func() {
		lockedDbWrite(func() { close(done) })
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("a later write could not acquire dbAccess; the mutex was leaked")
	}
}

// TestNoDirectDbAccessLocking enforces the convention that dbAccess is only ever taken
// through lockedDbWrite. Locking inline is easy to write and easy to get wrong: the unlock
// can be missed on a panic, and a function-scoped 'defer dbAccess.Unlock()' silently
// commits the whole rest of the function to the locked region, which self-deadlocks as soon
// as anything downstream re-acquires the non-reentrant mutex. Reviewing that hazard once
// per call site does not scale, so it is checked here instead.
//
// This walks the package's own syntax trees rather than grepping, so a mention of
// dbAccess.Unlock() in a comment does not trip it.
func TestNoDirectDbAccessLocking(t *testing.T) {
	// database.go defines the helper. This file needs raw access to assert the mutex was
	// released, which is the one thing lockedDbWrite cannot check about itself.
	exempt := map[string]bool{
		"database.go":      true,
		"database_test.go": true,
	}

	goFiles, err := filepath.Glob("*.go")
	require.NoError(t, err)
	require.NotEmpty(t, goFiles, "found no package sources to scan")

	fset := token.NewFileSet()
	for _, path := range goFiles {
		if exempt[filepath.Base(path)] {
			continue
		}

		file, parseErr := parser.ParseFile(fset, path, nil, 0)
		require.NoError(t, parseErr, "parsing %s", path)

		ast.Inspect(file, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if ident, isIdent := sel.X.(*ast.Ident); isIdent && ident.Name == "dbAccess" {
				t.Errorf("%s: dbAccess.%s used directly - take the write mutex through lockedDbWrite instead",
					fset.Position(sel.Pos()), sel.Sel.Name)
			}
			return true
		})
	}
}
