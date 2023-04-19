// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
	"net/http"
	"strconv"
	"testing"
)

func mw1(s string) ChainedHandler {
	// Do something
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Println("mw1 says, \"" + s + "\"")
			h.ServeHTTP(w, r)
		})
	}
}

func mw2(s string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("mw2 says, \"" + s + "\"")
	}
}

type testHandler struct {
	someString string
	someInt    int
}

func (th *testHandler) mw3(_ http.ResponseWriter, _ *http.Request) {
	fmt.Println("mw3 says, \"" + th.someString + " " + strconv.Itoa(th.someInt) + " is the ultimate answer!\"")
}

type mockResponseWriter struct{}

func (m *mockResponseWriter) Header() (h http.Header) {
	return http.Header{}
}

func (m *mockResponseWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func (m *mockResponseWriter) WriteString(s string) (n int, err error) {
	return len(s), nil
}

func (m *mockResponseWriter) WriteHeader(int) {}

func TestAdd(t *testing.T) {
	hc := NewHandlerChain()
	hc.Add(mw1("Hi, how are ya?"))

	assert.Equalf(t, 1, len(hc.handlerList), "There should only be 1 chained handler, found %d", len(hc.handlerList))
	assert.NotNil(t, hc.paramsHandler, "Params handler in handle chain should not be nil")

	var routed = false
	appHandler := func(w http.ResponseWriter, r *http.Request) {
		routed = true
	}

	router := httprouter.New()
	router.Handle("GET", "/", hc.ApplyTo(appHandler))
	w := new(mockResponseWriter)

	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	router.ServeHTTP(w, req)

	assert.True(t, routed, "routing failed")
}

func TestAddFunc(t *testing.T) {
	hc := NewHandlerChain()
	hc.AddFunc(mw2("Hello!"))

	assert.Equalf(t, 1, len(hc.handlerList), "There should only be 1 chained handler, found %d", len(hc.handlerList))
	assert.NotNil(t, hc.paramsHandler, "Params handler in handle chain should not be nil")

	var routed = false
	appHandler := func(w http.ResponseWriter, r *http.Request) {
		routed = true
	}

	router := httprouter.New()
	router.Handle("GET", "/", hc.ApplyTo(appHandler))
	w := new(mockResponseWriter)

	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	router.ServeHTTP(w, req)

	assert.True(t, routed, "routing failed")
}

// Tests another way of creating a route from a func that gets its dependencies from a struct instead of the
// method signature.
func TestAddFuncFromStruct(t *testing.T) {
	hc := NewHandlerChain()
	h := &testHandler{someString: "Hello World!", someInt: 42}
	myTestHandler := http.HandlerFunc(h.mw3)
	hc.AddFunc(myTestHandler)

	assert.Equalf(t, 1, len(hc.handlerList), "There should only be 1 chained handler, found %d", len(hc.handlerList))
	assert.NotNil(t, hc.paramsHandler, "Params handler in handle chain should not be nil")

	var routed = false
	appHandler := func(w http.ResponseWriter, r *http.Request) {
		routed = true
	}

	router := httprouter.New()
	router.Handle("GET", "/", hc.ApplyTo(appHandler))
	w := new(mockResponseWriter)

	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	router.ServeHTTP(w, req)

	assert.True(t, routed, "routing failed")
}

func TestExtend(t *testing.T) {
	hcList := NewHandlerChain(mw1("I was added by extension"), mw1("so was I"), mw1("and so was I"))
	hc := NewHandlerChain(mw1("I got added directly to the original HC"))
	hc.Extend(hcList)

	assert.Equalf(t, 4, len(hc.handlerList), "There should only be 4 chained handlers, found %d", len(hc.handlerList))
	assert.NotNil(t, hc.paramsHandler, "Params handler in handle chain should not be nil")

	var routed = false
	appHandler := func(w http.ResponseWriter, r *http.Request) {
		routed = true
	}

	router := httprouter.New()
	router.Handle("GET", "/", hc.ApplyTo(appHandler))
	w := new(mockResponseWriter)

	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	router.ServeHTTP(w, req)

	assert.True(t, routed, "routing failed")
}

func TestGetParamsByContext(t *testing.T) {

	handler1 := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r)
		})
	}

	hc := NewHandlerChain(handler1)

	assert.Equalf(t, 1, len(hc.handlerList), "There should only be 1 chained handler, found %d", len(hc.handlerList))
	assert.NotNil(t, hc.paramsHandler, "Params handler in handle chain should not be nil")

	var routed = false
	var shape = ""

	appHandler := func(w http.ResponseWriter, r *http.Request) {
		routed = true
		ps := httprouter.ParamsFromContext(r.Context())
		shape = ps.ByName("shape")
	}

	router := httprouter.New()
	router.Handle("GET", "/test/:shape", hc.ApplyTo(appHandler))

	w := new(mockResponseWriter)

	req, _ := http.NewRequest(http.MethodGet, "/test/square", nil)
	router.ServeHTTP(w, req)

	assert.True(t, routed, "routing failed")
	assert.Equalf(t, "square", shape, "returned param was not correct")

}
