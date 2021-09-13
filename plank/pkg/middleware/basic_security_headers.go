// Copyright 2019-2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package middleware

import (
	"github.com/gorilla/mux"
	"net/http"
)

func BasicSecurityHeaderMiddleware() mux.MiddlewareFunc {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "no-sniff")
			w.Header().Set("X-Frame-Options", "SAMEORIGIN")
			w.Header().Set("X-Xss-Protection", "1; mode=block")
			handler.ServeHTTP(w, r)
		})
	}
}
