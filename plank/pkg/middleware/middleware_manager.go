package middleware

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/jooskim/plank/utils"
	"net/http"
	"strings"
	"sync"
)

// reusable basic security headers. TODO: make it configurable
var BasicSecurityHeaderMiddleware = NewBasicSecurityHeadersMiddleware()

type MiddlewareManager interface {
	SetNewMiddleware(route *mux.Route, middleware []Middleware) error
	RemoveMiddleware(route *mux.Route) error
}

type Middleware interface {
	Intercept(h http.Handler) http.Handler
	Name() string
}

type middlewareManager struct {
	endpointHandlerMap  *map[string]http.HandlerFunc
	originalHandlersMap map[string]http.HandlerFunc
	mu                  sync.Mutex
}

func (m *middlewareManager) SetNewMiddleware(route *mux.Route, middleware []Middleware) error {
	uri, method := m.extractUriVerbFromMuxRoute(route)
	if route == nil {
		return fmt.Errorf("failed to set a new middleware. route does not exist at %s (%s)", uri, method)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	key := uri + "-" + method

	// find if base handler exists first. if not, error out
	original, exists := (*m.endpointHandlerMap)[key]
	if !exists {
		return fmt.Errorf("cannot set middleware. REST bridge handler does not exist at %s (%s)", uri, method)
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
		utils.Log.Debugf("middleware '%s' registered for %s (%s)", mw.Name(), uri, method)
	}

	utils.Log.Infof("New middleware configured for REST bridge at %s (%s)", uri, method)

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

func (m *middlewareManager) buildMiddlewareChain(handlers []Middleware, originalHandler http.Handler) http.Handler {
	var idx = len(handlers) - 1
	var finalHandler http.Handler

	for idx >= 0 {
		var currHandler http.Handler
		if idx == len(handlers)-1 {
			currHandler = originalHandler
		} else {
			currHandler = finalHandler
		}
		finalHandler = handlers[idx].Intercept(currHandler)
		idx--
	}
	return finalHandler
}

// extractUriVerbFromMuxRoute takes *mux.Route and returns URI and verb as string values
func (m *middlewareManager) extractUriVerbFromMuxRoute(route *mux.Route) (string, string) {
	opRawString := strings.Replace(route.GetName(), "bridge-", "", 1)
	delimiterIdx := strings.LastIndex(opRawString, "-")
	return opRawString[:delimiterIdx], opRawString[delimiterIdx+1:]
}

// NewMiddlewareManager sets up a new middleware manager singleton instance
func NewMiddlewareManager(endpointHandlerMapPtr *map[string]http.HandlerFunc) MiddlewareManager {
	return &middlewareManager{
		endpointHandlerMap:  endpointHandlerMapPtr,
		originalHandlersMap: make(map[string]http.HandlerFunc),
	}
}
