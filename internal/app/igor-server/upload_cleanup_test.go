// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// uploadBody builds a multipart body with one file part of the given size.
func uploadBody(t *testing.T, size int) (*bytes.Buffer, string) {
	t.Helper()

	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	fw, err := w.CreateFormFile("kernelFile", "vmlinuz-test")
	require.NoError(t, err)
	_, err = fw.Write(bytes.Repeat([]byte("k"), size))
	require.NoError(t, err)
	require.NoError(t, w.Close())

	return body, w.FormDataContentType()
}

// postUpload serves one multipart POST through a router carrying igor's real panicHandler,
// with TMPDIR pointed at a scratch directory, and reports what is left behind in it.
func postUpload(t *testing.T, handler http.HandlerFunc, partSize int) (leftovers []string, resp *http.Response) {
	t.Helper()

	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)

	chain := hlog.NewHandler(zerolog.Nop())(handler)
	router := &httprouter.Router{PanicHandler: panicHandler}
	router.Handle(http.MethodPost, "/upload", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		chain.ServeHTTP(w, r)
	})
	srv := httptest.NewServer(router)
	defer srv.Close()

	body, contentType := uploadBody(t, partSize)
	resp, err := http.Post(srv.URL+"/upload", contentType, body)
	if err == nil {
		t.Cleanup(func() { _ = resp.Body.Close() })
	}

	entries, readErr := os.ReadDir(tmp)
	require.NoError(t, readErr)
	for _, e := range entries {
		leftovers = append(leftovers, e.Name())
	}
	return leftovers, resp
}

// Regression test for the /tmp leak. net/http removes the files a multipart upload spools to
// disk in (*response).finishRequest, but that call is not deferred, so a panic escaping
// ServeHTTP unwinds past it. igor guarantees such a panic: panicHandler logs with
// logger.Panic(), which panics again. Every panicking upload therefore used to
// strand a whole kernel or initrd in the temp directory until it filled.
//
// removeUploadTempFiles is deferred in the parsing frame, so it runs during the unwind.
func TestUploadTempFilesRemovedWhenHandlerPanics(t *testing.T) {
	captureLog(t)

	leftovers, _ := postUpload(t, func(w http.ResponseWriter, r *http.Request) {
		defer removeUploadTempFiles(r)
		if err := r.ParseMultipartForm(1 << 10); err != nil {
			return
		}
		panic("simulated panic after a successful upload -- test artifact only!!!")
	}, 64<<10)

	assert.Empty(t, leftovers,
		"upload temp files leaked after a panic; removeUploadTempFiles must be deferred in the frame that parses")
}

// The main event: an ordinary, entirely successful upload must leave nothing behind.
func TestUploadTempFilesRemovedOnNormalReturn(t *testing.T) {
	captureLog(t)

	leftovers, resp := postUpload(t, func(w http.ResponseWriter, r *http.Request) {
		defer removeUploadTempFiles(r)
		if err := r.ParseMultipartForm(1 << 10); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}, 64<<10)

	require.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Empty(t, leftovers, "upload temp files left behind on the normal path")
}

// removeUploadTempFiles must tolerate a request that never parsed a form -- every validator
// defers it before knowing whether the body is multipart at all.
func TestRemoveUploadTempFilesHandlesUnparsedRequest(t *testing.T) {
	captureLog(t)

	req := httptest.NewRequest(http.MethodGet, "/distros", nil)
	require.Nil(t, req.MultipartForm)
	assert.NotPanics(t, func() { removeUploadTempFiles(req) })
}

// A server that cannot write its temp directory is not a bad request. Reporting 400 with the
// raw error sent whoever was debugging a full disk looking at their own command instead, and
// echoed server-side paths back to the client.
func TestUploadParseErrorOnUnwritableTempDirIsNotClientError(t *testing.T) {
	captureLog(t)

	readOnly := filepath.Join(t.TempDir(), "readonly")
	require.NoError(t, os.Mkdir(readOnly, 0500))
	t.Setenv("TMPDIR", readOnly)

	router := &httprouter.Router{PanicHandler: panicHandler}
	router.Handle(http.MethodPost, "/upload", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		defer removeUploadTempFiles(r)
		if err := r.ParseMultipartForm(1 << 10); err != nil {
			respondUploadParseErr(w, hlog.FromRequest(r), err)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(router)
	defer srv.Close()

	body, contentType := uploadBody(t, 64<<10)
	resp, err := http.Post(srv.URL+"/upload", contentType, body)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode,
		"a temp directory the server cannot write is a server fault, not a malformed request")

	var rb struct {
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(raw, &rb))
	assert.Contains(t, strings.ToLower(rb.Message), "administrator",
		"the message should tell the caller who can fix it")
	assert.NotContains(t, rb.Message, readOnly,
		"the server's temp path must stay in the log, not the response body")
}

// A genuinely malformed body is still the client's fault.
func TestUploadParseErrorOnMalformedBodyStaysClientError(t *testing.T) {
	captureLog(t)

	router := &httprouter.Router{PanicHandler: panicHandler}
	router.Handle(http.MethodPost, "/upload", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		defer removeUploadTempFiles(r)
		if err := r.ParseMultipartForm(1 << 10); err != nil {
			respondUploadParseErr(w, hlog.FromRequest(r), err)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(router)
	defer srv.Close()

	// Declares multipart but the body is not.
	resp, err := http.Post(srv.URL+"/upload", "multipart/form-data; boundary=xyz",
		strings.NewReader("this is not a multipart body"))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode,
		"a malformed multipart body is a client error")
}

// Every frame that parses an upload must also defer the cleanup. A new upload route that
// forgets it reintroduces the leak silently, and the failure only shows up as a full disk
// weeks later, so it is checked rather than left to review.
func TestEveryMultipartParserDefersCleanup(t *testing.T) {
	goFiles, err := filepath.Glob("*.go")
	require.NoError(t, err)
	require.NotEmpty(t, goFiles)

	fset := token.NewFileSet()
	checked := 0

	for _, path := range goFiles {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, parseErr := parser.ParseFile(fset, path, nil, 0)
		require.NoError(t, parseErr, "parsing %s", path)

		// Look at every function literal and declaration independently: the parse and the
		// defer have to share a frame, because the defer must see the same *http.Request the
		// parse wrote MultipartForm onto.
		check := func(name string, pos token.Pos, body *ast.BlockStmt) {
			if body == nil {
				return
			}
			parses, defers := false, false

			walk := func(n ast.Node) bool {
				// Do not descend into nested function bodies; they are their own frames.
				if _, isLit := n.(*ast.FuncLit); isLit && n.Pos() != pos {
					return false
				}
				if call, ok := n.(*ast.CallExpr); ok {
					if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "ParseMultipartForm" {
						parses = true
					}
				}
				if d, ok := n.(*ast.DeferStmt); ok {
					if ident, ok := d.Call.Fun.(*ast.Ident); ok && ident.Name == "removeUploadTempFiles" {
						defers = true
					}
				}
				return true
			}
			for _, stmt := range body.List {
				ast.Inspect(stmt, walk)
			}

			if parses {
				checked++
				if !defers {
					t.Errorf("%s: %s calls ParseMultipartForm without 'defer removeUploadTempFiles(r)' "+
						"in the same frame; a panic below it will strand the spooled upload in the "+
						"server's temp directory", fset.Position(pos), name)
				}
			}
		}

		ast.Inspect(file, func(n ast.Node) bool {
			switch fn := n.(type) {
			case *ast.FuncDecl:
				check(fn.Name.Name, fn.Pos(), fn.Body)
			case *ast.FuncLit:
				check("func literal", fn.Pos(), fn.Body)
			}
			return true
		})
	}

	require.NotZero(t, checked, "found no ParseMultipartForm call sites to check")
}
