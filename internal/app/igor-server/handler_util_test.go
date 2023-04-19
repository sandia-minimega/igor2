// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"net/http"
	"strings"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
)

func TestEmptyJsonBody(t *testing.T) {

	hc := NewHandlerChain(storeJSONBodyHandler)

	assert.Equalf(t, 1, len(hc.handlerList), "There should only be 1 chained handler, found %d", len(hc.handlerList))
	assert.NotNil(t, hc.paramsHandler, "Params handler in handle chain should not be nil")

	var routed = false
	var jBodyMap map[string]interface{} = nil
	var expected map[string]interface{} = nil

	appHandler := func(w http.ResponseWriter, r *http.Request) {
		routed = true
		jBodyMap = getBodyFromContext(r)
	}

	router := httprouter.New()
	router.Handle(http.MethodPost, "/testJsonBody", hc.ApplyTo(appHandler))

	w := new(mockResponseWriter)

	req, _ := http.NewRequest(http.MethodPost, "/testJsonBody", http.NoBody)
	router.ServeHTTP(w, req)

	assert.True(t, routed, "routing failed")
	assert.Equalf(t, expected, jBodyMap, "jsonBody is not empty")

}

func TestGetJSONBodyFromContext(t *testing.T) {

	hc := NewHandlerChain(storeJSONBodyHandler)

	assert.Equalf(t, 1, len(hc.handlerList), "There should only be 1 chained handler, found %d", len(hc.handlerList))
	assert.NotNil(t, hc.paramsHandler, "Params handler in handle chain should not be nil")

	var routed = false
	var jBodyMap map[string]interface{}

	appHandler := func(w http.ResponseWriter, r *http.Request) {
		routed = true
		jBodyMap = getBodyFromContext(r)
	}

	router := httprouter.New()
	router.Handle(http.MethodPost, "/testJsonBody", hc.ApplyTo(appHandler))

	w := new(mockResponseWriter)
	expected := "{\"one\": 1, \"two\": 2}"

	req, _ := http.NewRequest(http.MethodPost, "/testJsonBody", strings.NewReader(expected))
	router.ServeHTTP(w, req)

	assert.True(t, routed, "routing failed")
	assert.Contains(t, jBodyMap, "one", "did not contain a key named 'one'")

}
