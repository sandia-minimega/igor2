// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"context"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

// IHandlerChain is an interface defining operations that build up and execute a chain of HTTP handlers
// that can be processed by httprouter.Router
type IHandlerChain interface {
	Add(h ChainedHandler)
	AddFunc(h http.Handler)
	Extend(hc *HandlerChain)
	ApplyTo(h http.HandlerFunc) httprouter.Handle
}

// ChainedHandler is an adapter type that accepts a function that takes a http.Handler and returns a http.Handler
type ChainedHandler func(http.Handler) http.Handler

// HandlerChain bundles common handlers
type HandlerChain struct {
	paramsHandler func(http.Handler) httprouter.Handle
	handlerList   []ChainedHandler
}

// NewHandlerChain creates a new HandlerChain struct with the option to add a comma-delimited
// list of ChainedHandlers. It will always include a special handler that is executed first and copies
// httprouter.Params into a new context that is passed to the next handler. This allows
// access to params at any stage in the handler chain via the httprouter.ParamsFromContext
// method.
func NewHandlerChain(handlers ...ChainedHandler) *HandlerChain {

	chs := make([]ChainedHandler, 0, len(handlers))
	chs = append(chs, handlers...)

	return &HandlerChain{
		paramsHandler: func(handler http.Handler) httprouter.Handle {
			return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
				if len(p) > 0 {
					ctx := r.Context()
					ctx = context.WithValue(ctx, httprouter.ParamsKey, p)
					rCopy := r.WithContext(ctx)
					handler.ServeHTTP(w, rCopy)
				} else {
					handler.ServeHTTP(w, r)
				}
			}
		},
		handlerList: chs,
	}
}

// Add appends a new ChainedHandler to the handler list.
func (s *HandlerChain) Add(h ChainedHandler) {
	s.handlerList = append(s.handlerList, h)
}

// AddFunc adapts and appends any function to the handler list that returns type http.Handler. It should
// be of the form:
//
//	 func name(any) http.HandlerFunc {
//		   return func(w http.ResponseWriter, r *http.Request) {
//		  	 // do something with any
//		   }
//	 }
func (s *HandlerChain) AddFunc(h http.Handler) {
	wrappedFunc := func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r)
			handler.ServeHTTP(w, r)
		})
	}

	s.Add(wrappedFunc)
}

// Extend appends an existing HandlerChain to the calling HandlerChain. This can be used
// to add groups of handlers with a common execution flow so they don't have to be re-declared for each route
// that needs them.
func (s *HandlerChain) Extend(hc *HandlerChain) {
	s.handlerList = append(s.handlerList, hc.handlerList...)
}

// ApplyTo associates the handler chain with the application handler, which is the last handler to be executed and
// presumably where the business logic for the route resides. The httprouter.Handle it returns is the handler that
// stores httprouter.Params in a new context that is passed to all subsequent handlers.
func (s *HandlerChain) ApplyTo(appHandler http.HandlerFunc) httprouter.Handle {

	l := len(s.handlerList)
	if l == 0 {
		return s.paramsHandler(appHandler)
	}

	var handler http.Handler
	handler = s.handlerList[l-1](appHandler)

	for i := l - 2; i >= 0; i-- {
		handler = s.handlerList[i](handler)
	}

	return s.paramsHandler(handler)
}
