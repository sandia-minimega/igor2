// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// panicHandler must return normally. It is registered as httprouter's PanicHandler and runs
// inside the deferred recover that caught the original panic; a second panic raised there is
// not caught by that same recover, so it escapes ServeHTTP entirely. Everything after the log
// call is then skipped -- no response is written -- and net/http closes the connection.
func TestPanicHandlerReturnsNormally(t *testing.T) {
	captureLog(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/distros", nil)

	require.NotPanics(t, func() {
		panicHandler(w, r, "simulated handler panic")
	}, "panicHandler must not panic; a panic here escapes httprouter's recover and drops the connection")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// End to end: a panicking handler must produce a readable 500, not a dropped connection.
// This is what an operator sees, and the reason a full disk reached one as
// "use of closed network connection" rather than a server error message.
func TestPanickingHandlerReturns500ToClient(t *testing.T) {
	captureLog(t)

	router := &httprouter.Router{PanicHandler: panicHandler}
	router.Handle(http.MethodGet, "/boom", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		panic("simulated handler panic")
	})
	srv := httptest.NewServer(router)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/boom")
	require.NoError(t, err, "the client must receive a response, not a closed connection")
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	var rb struct {
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(body, &rb), "the 500 must carry igor's JSON body")
	assert.Contains(t, strings.ToLower(rb.Message), "notify admins",
		"the response should tell the caller what to do")
}

// The panic value is a raw runtime message -- "index out of range [3] with length 2" and the
// like. It belongs in the log, not in the response.
func TestPanicHandlerDoesNotEchoPanicValueToClient(t *testing.T) {
	logBuf := captureLog(t)

	const secret = "sql: SELECT pass_hash FROM users WHERE name = 'igor-admin'"

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/distros", nil)
	panicHandler(w, r, secret)

	assert.NotContains(t, w.Body.String(), secret,
		"the panic value must not be echoed to the client")
	assert.Contains(t, logBuf.String(), secret,
		"the panic value must still reach the log, or the incident is undiagnosable")
}

// The stack trace has to land in igor's own log. Splitting it between igor.log and stderr
// leaves an operator reading either one with an incomplete picture.
func TestPanicHandlerLogsPathAndStack(t *testing.T) {
	logBuf := captureLog(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/distros/foo", nil)
	panicHandler(w, r, "simulated handler panic")

	logged := logBuf.String()
	assert.Contains(t, logged, "/distros/foo", "the log must identify the request that panicked")
	assert.Contains(t, logged, "simulated handler panic", "the log must carry the panic value")
	assert.Contains(t, logged, "panic_handler_test.go", "the log must carry a usable stack trace")
}

// net/http writes its own errors -- accept failures, TLS handshake errors, the "panic
// serving" line -- through Server.ErrorLog. Unset, that goes to stderr and, under systemd,
// to the journal rather than igor.log. Both servers now route it into zerolog.
func TestHttpErrorLogWritesIntoIgorLog(t *testing.T) {
	logBuf := captureLog(t)

	newHttpErrorLog("api server").Printf("http: TLS handshake error from 10.0.0.1:1234: EOF")

	logged := logBuf.String()
	assert.Contains(t, logged, "api server", "the entry should say which server reported it")
	assert.Contains(t, logged, "TLS handshake error", "net/http's message must reach igor.log")
	assert.NotContains(t, logged, "EOF\\n", "the trailing newline should be trimmed, not logged")
}

// Both servers must actually be wired to it; an adapter nothing uses fixes nothing.
func TestBothServersRouteErrorsToIgorLog(t *testing.T) {
	src, err := os.ReadFile("server.go")
	require.NoError(t, err)

	count := strings.Count(string(src), "ErrorLog:  newHttpErrorLog(") +
		strings.Count(string(src), "ErrorLog: newHttpErrorLog(")
	assert.Equal(t, 2, count,
		"both http.Server instances in runServer must set ErrorLog, or their errors go to stderr")
}
