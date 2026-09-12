// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// stageTestImagePair points the image staging and store directories at temp dirs and writes
// a kernel/initrd pair into staging. It returns the temp base names and the tempFiles slice
// in the form processImage expects.
func stageTestImagePair(t *testing.T) (image *DistroImage, tempFiles []string) {
	t.Helper()

	stage := t.TempDir()
	tftp := t.TempDir()

	prevStage, prevTFTP, prevStore := igor.Server.ImageStagePath, igor.TFTPPath, igor.ImageStoreDir
	igor.Server.ImageStagePath = stage
	igor.TFTPPath = tftp
	igor.ImageStoreDir = "images"
	t.Cleanup(func() {
		igor.Server.ImageStagePath, igor.TFTPPath, igor.ImageStoreDir = prevStage, prevTFTP, prevStore
	})

	// Base names must not end in a character that processImageFiles' TrimRight would eat.
	const tempK, tempI = "stagedk1", "stagedi1"
	require.NoError(t, os.WriteFile(filepath.Join(stage, tempK), []byte("test kernel payload"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(stage, tempI), []byte("test initrd payload"), 0600))

	image = &DistroImage{
		Type:   DistroKI,
		Kernel: "vmlinuz-test",
		Initrd: "initrd-test.img",
	}
	return image, []string{tempK + ".kernel", tempI + ".initrd"}
}

// fillInitrdQueue installs a small initrd queue filled to capacity, with no worker draining
// it, and restores the previous queue afterwards. This is the state that used to wedge the
// server: any enqueue attempted from here on blocks until something drains.
func fillInitrdQueue(t *testing.T, capacity int) *InitrdJobQueue {
	t.Helper()

	q := &InitrdJobQueue{queue: make(chan InitrdJob, capacity)}
	for i := 0; i < capacity; i++ {
		q.queue <- InitrdJob{Image: &DistroImage{ImageID: "backlog"}}
	}

	prev := initrdQueue
	initrdQueue = q
	t.Cleanup(func() { initrdQueue = prev })
	return q
}

// Regression test for BUG-017. processImage used to call enqueueInitrdJob from inside its
// lockedDbWrite region, while a write transaction was also open. Enqueue blocks on a full
// queue, and the only goroutine that drains that queue needs dbAccess and a transaction of
// its own to record each result -- so a registration landing on a full queue held the mutex
// forever and every write in the process stopped.
//
// With a full queue and no worker at all, registering an image must still complete.
func TestProcessImageCompletesWithFullInitrdQueue(t *testing.T) {
	newTestDbBackend(t)
	captureLog(t)

	const capacity = 4
	q := fillInitrdQueue(t, capacity)
	image, tempFiles := stageTestImagePair(t)

	type result struct {
		image   *DistroImage
		created bool
		err     error
	}
	done := make(chan result, 1)

	go func() {
		var res result
		res.err = performDbTx(func(tx *gorm.DB) error {
			var pErr error
			res.image, res.created, pErr = processImage(image, tempFiles, tx)
			return pErr
		})
		done <- res
	}()

	select {
	case res := <-done:
		require.NoError(t, res.err, "registering an image against a full initrd queue must not fail")
		require.NotNil(t, res.image)
		assert.True(t, res.created, "a brand new image must report created")
		assert.NotEmpty(t, res.image.ImageID, "the image row must carry its content hash")

	case <-time.After(10 * time.Second):
		// Pre-fix this is where it ends: the goroutine is blocked on a full queue while
		// holding dbAccess, so it never returns and the mutex is gone for good.
		t.Fatal("BUG-017 regression: processImage blocked on a full initrd queue. " +
			"The hand-off to the initrd worker has moved back inside the locked region " +
			"or the transaction.")
	}

	assert.Len(t, q.queue, capacity,
		"processImage must not enqueue the initrd job itself; its caller does so after the transaction commits")

	// The mutex must be free for everyone else, which is what the bug actually cost.
	assertDbAccessFree(t, "dbAccess still held after registering an image against a full initrd queue")
}

// An image whose content hash already exists is returned as-is, and owes no initrd job --
// the first registration already queued one.
func TestProcessImageExistingImageOwesNoInitrdJob(t *testing.T) {
	newTestDbBackend(t)
	captureLog(t)

	const capacity = 2
	q := fillInitrdQueue(t, capacity)

	first, firstFiles := stageTestImagePair(t)
	var firstID string
	require.NoError(t, performDbTx(func(tx *gorm.DB) error {
		created, wasNew, err := processImage(first, firstFiles, tx)
		if err != nil {
			return err
		}
		require.True(t, wasNew)
		firstID = created.ImageID
		return nil
	}))

	// Same payload staged again produces the same hash, so no new row is written.
	second, secondFiles := stageTestImagePair(t)
	require.NoError(t, performDbTx(func(tx *gorm.DB) error {
		found, wasNew, err := processImage(second, secondFiles, tx)
		if err != nil {
			return err
		}
		assert.False(t, wasNew, "an already-registered image must not report created")
		assert.Equal(t, firstID, found.ImageID, "the existing row must be returned")
		return nil
	}))

	assert.Len(t, q.queue, capacity, "neither call may enqueue from inside processImage")
	assertDbAccessFree(t, "dbAccess still held after re-registering an existing image")
}

// The other half of the BUG-017 fix lives in the callers: registerImage's contract is that
// they enqueue only once their transaction has committed. A functional test cannot easily
// reach doRegisterImage and doCreateDistro without a full multipart request, so the
// placement is checked structurally instead.
func TestInitrdJobNotEnqueuedInsideLockOrTransaction(t *testing.T) {
	goFiles, err := filepath.Glob("*.go")
	require.NoError(t, err)
	require.NotEmpty(t, goFiles)

	// Calls whose function-literal argument runs with dbAccess held, a transaction open, or
	// both. enqueueInitrdJob blocks, so it must not appear inside any of them.
	guarded := map[string]bool{
		"lockedDbWrite": true,
		"performDbTx":   true,
	}

	fset := token.NewFileSet()
	for _, path := range goFiles {
		file, parseErr := parser.ParseFile(fset, path, nil, 0)
		require.NoError(t, parseErr, "parsing %s", path)

		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			ident, ok := call.Fun.(*ast.Ident)
			if !ok || !guarded[ident.Name] {
				return true
			}

			// Anything reached from inside this call's arguments runs under the guard.
			for _, arg := range call.Args {
				ast.Inspect(arg, func(inner ast.Node) bool {
					innerCall, isCall := inner.(*ast.CallExpr)
					if !isCall {
						return true
					}
					if innerIdent, isIdent := innerCall.Fun.(*ast.Ident); isIdent &&
						innerIdent.Name == "enqueueInitrdJob" {
						t.Errorf("%s: enqueueInitrdJob called inside %s - it blocks on a full "+
							"queue whose drain needs dbAccess and a transaction, which deadlocks "+
							"the server (BUG-017). Enqueue after the transaction commits.",
							fset.Position(innerCall.Pos()), ident.Name)
					}
					return true
				})
			}
			return true
		})
	}
}
