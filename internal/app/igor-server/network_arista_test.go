// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// resetAristaClient clears the memoised client so a test can build one against its own
// stub server. Tests using it must not run in parallel with each other.
func resetAristaClient() {
	aristaClientOnce = sync.Once{}
	aristaClient = nil
}

// withAristaStub points igor.Vlan at a stub server and restores the previous config and
// client when the test finishes.
func withAristaStub(t *testing.T, timeoutSecs int, h http.HandlerFunc) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(h)
	prev := igor.Vlan

	igor.Vlan.Network = "arista"
	igor.Vlan.NetworkUser = "igor"
	igor.Vlan.NetworkPassword = ""
	igor.Vlan.NetworkURL = strings.TrimPrefix(srv.URL, "http://")
	igor.Vlan.NetworkTimeout = timeoutSecs
	resetAristaClient()

	t.Cleanup(func() {
		srv.Close()
		igor.Vlan = prev
		resetAristaClient()
	})

	return srv
}

// A switch that accepts the connection and never answers must not block the caller
// forever. This is the failure that stops every write on the server, because the call is
// made while dbAccess is held.
func TestAristaJSONRPCTimesOutOnSilentSwitch(t *testing.T) {
	stall := make(chan struct{})
	srv := withAristaStub(t, 1, func(w http.ResponseWriter, r *http.Request) {
		<-stall
	})
	t.Cleanup(func() { close(stall) })

	done := make(chan error, 1)
	go func() {
		_, err := aristaJSONRPC(igor.Vlan.NetworkUser, igor.Vlan.NetworkPassword, igor.Vlan.NetworkURL,
			[]string{"show vlan 100-200"})
		done <- err
	}()

	select {
	case err := <-done:
		assert.Error(t, err, "a silent switch must produce an error, not a nil response")
	case <-time.After(10 * time.Second):
		t.Fatal("aristaJSONRPC did not return; the client has no effective timeout")
	}

	_ = srv
}

// The per-host loops in aristaSet/aristaClear call this once per node. Each call must
// reuse the pooled connection rather than opening and abandoning a new socket.
func TestAristaJSONRPCReusesConnections(t *testing.T) {
	var mu sync.Mutex
	conns := make(map[string]struct{})

	srv := withAristaStub(t, 5, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		conns[r.RemoteAddr] = struct{}{}
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":[{"vlans":{}}]}`))
	})
	_ = srv

	for i := 0; i < 10; i++ {
		_, err := aristaJSONRPC(igor.Vlan.NetworkUser, igor.Vlan.NetworkPassword, igor.Vlan.NetworkURL,
			[]string{"show vlan 100-200"})
		assert.NoError(t, err)
	}

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 1, len(conns),
		"10 sequential calls must share one pooled connection, not open %d", len(conns))
}

// Credentials belong in a header, not the request URL. In the URL they reach err.Error(),
// which is what forced this code to scrub its own error text.
func TestAristaJSONRPCSendsBasicAuthHeader(t *testing.T) {
	type creds struct {
		user, pass string
		ok         bool
	}
	got := make(chan creds, 1)

	srv := withAristaStub(t, 5, func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		got <- creds{u, p, ok}
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":[{"vlans":{}}]}`))
	})
	_ = srv

	_, err := aristaJSONRPC("igor", "", igor.Vlan.NetworkURL, []string{"show vlan 100-200"})
	assert.NoError(t, err)

	c := <-got
	assert.True(t, c.ok, "request must carry an Authorization header")
	assert.Equal(t, "igor", c.user)
	assert.Equal(t, "", c.pass, "an empty password is a supported configuration")
}

// A transport error must surface intact. The previous implementation ran the message
// through strings.Replace with the password as the pattern, which for the empty password
// the shipped config permits inserted a placeholder between every rune.
func TestAristaJSONRPCErrorTextIsNotMangled(t *testing.T) {
	srv := withAristaStub(t, 5, func(w http.ResponseWriter, r *http.Request) {})
	addr := igor.Vlan.NetworkURL
	srv.Close() // nothing is listening now, so the POST fails at the transport

	_, err := aristaJSONRPC("igor", "", addr, []string{"show vlan 100-200"})

	assert.Error(t, err)
	assert.NotContains(t, err.Error(), "<PASSWORD>",
		"an empty password must not cause placeholder insertion")
	assert.Contains(t, err.Error(), "post failed:")
}
