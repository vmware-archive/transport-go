// Copyright 2019-2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package middleware

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/vmware/transport-go/plank/utils"
	"net/http"
	"strings"
	"sync"
)

type MiddlewareManager interface {
	SetGlobalMiddleware(middleware []mux.MiddlewareFunc) error
	SetNewMiddleware(route *mux.Route, middleware []mux.MiddlewareFunc) error
	RemoveMiddleware(route *mux.Route) error
	GetRouteByUriAndMethod(uri, method string) (*mux.Route, error)
	GetRouteByUri(uri string) (*mux.Route, error)
	GetStaticRoute(prefix string) (*mux.Route, error)
}

type Middleware interface {
	//Intercept(h http.Handler) http.Handler
	Interceptor() mux.MiddlewareFunc
	Name() string
}

type middlewareManager struct {
	endpointHandlerMap  *map[string]http.HandlerFunc
	originalHandlersMap map[string]http.HandlerFunc
	router              *mux.Router
	mu                  sync.Mutex
}

func (m *middlewareManager) SetGlobalMiddleware(middleware []mux.MiddlewareFunc) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.router.Use(middleware...)
	return nil
}

func (m *middlewareManager) SetNewMiddleware(route *mux.Route, middleware []mux.MiddlewareFunc) error {
	var key string
	// expection is that a route's name ending with '*' means it's a prefix route
	isPrefixRoute := route.GetName()[len(route.GetName())-1] == '*'

	if !isPrefixRoute {
		uri, method := m.extractUriVerbFromMuxRoute(route)
		if route == nil {
			return fmt.Errorf("failed to set a new middleware. route does not exist at %s (%s)", uri, method)
		}
		// for REST-bridge service a key is in the format of {uri}-{verb}
		key = uri + "-" + method
	} else {
		// if the route instance is a prefix route use the route name as-is
		key = route.GetName()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// find if base handler exists first. if not, error out
	original, exists := (*m.endpointHandlerMap)[key]
	if !exists {
		return fmt.Errorf("cannot set middleware. handler does not exist at %s", key)
	}

	// make a backup of the original handler that has no other middleware attached to it
	if _, exists := m.originalHandlersMap[key]; !exists {
		m.originalHandlersMap[key] = original
	}

	// build a new middleware chain and apply it
	handler := m.buildMiddlewareChain(middleware, original).(http.HandlerFunc)
	(*m.endpointHandlerMap)[key] = handler
	route.Handler(handler)

	for _, mw := range middleware {
		utils.Log.Debugf("middleware '%v' registered for %s", mw, key)
	}

	utils.Log.Infof("New middleware configured for REST bridge at %s", key)

	return nil
}

func (m *middlewareManager) RemoveMiddleware(route *mux.Route) error {
	uri, method := m.extractUriVerbFromMuxRoute(route)
	if route == nil {
		return fmt.Errorf("failed to remove middleware. route does not exist at %s (%s)", uri, method)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	key := uri + "-" + method
	if _, found := (*m.endpointHandlerMap)[key]; !found {
		return fmt.Errorf("failed to remove handler. REST bridge handler does not exist at %s (%s)", uri, method)
	}
	defer func() {
		if r := recover(); r != nil {
			utils.Log.Errorln(r)
		}
	}()

	(*m.endpointHandlerMap)[key] = m.originalHandlersMap[key]
	route.Handler(m.originalHandlersMap[key])
	utils.Log.Debugf("All middleware have been stripped from %s (%s)", uri, method)

	return nil
}

func (m *middlewareManager) GetRouteByUriAndMethod(uri, method string) (*mux.Route, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	route := m.router.Get(fmt.Sprintf("%s-%s", uri, method))
	if route == nil {
		return nil, fmt.Errorf("no route found at %s (%s)", uri, method)
	}
	return route, nil
}

func (m *middlewareManager) GetStaticRoute(prefix string) (*mux.Route, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	routeName := prefix + "*"
	route := m.router.Get(routeName)
	if route == nil {
		return nil, fmt.Errorf("no route found at static prefix %s", routeName)
	}
	return route, nil
}

func (m *middlewareManager) GetRouteByUri(uri string) (*mux.Route, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	route := m.router.Get(uri)
	if route == nil {
		return nil, fmt.Errorf("no route found at %s", uri)
	}
	return route, nil
}

func (m *middlewareManager) buildMiddlewareChain(handlers []mux.MiddlewareFunc, originalHandler http.Handler) http.Handler {
	var idx = len(handlers) - 1
	var finalHandler http.Handler

	for idx >= 0 {
		var currHandler http.Handler
		if idx == len(handlers)-1 {
			currHandler = originalHandler
		} else {
			currHandler = finalHandler
		}
		middlewareFn := handlers[idx]
		finalHandler = middlewareFn(currHandler)
		idx--
	}

	return finalHandler
}

// extractUriVerbFromMuxRoute takes *mux.Route and returns URI and verb as string values
func (m *middlewareManager) extractUriVerbFromMuxRoute(route *mux.Route) (string, string) {
	opRawString := route.GetName()
	delimiterIdx := strings.LastIndex(opRawString, "-")
	return opRawString[:delimiterIdx], opRawString[delimiterIdx+1:]
}

// NewMiddlewareManager sets up a new middleware manager singleton instance
func NewMiddlewareManager(endpointHandlerMapPtr *map[string]http.HandlerFunc, router *mux.Router) MiddlewareManager {
	return &middlewareManager{
		endpointHandlerMap:  endpointHandlerMapPtr,
		originalHandlersMap: make(map[string]http.HandlerFunc),
		router:              router,
	}
}
