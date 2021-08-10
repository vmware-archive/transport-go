// Copyright 2019-2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package middleware

import (
	"net/http"
)

type basicSecurityHeadersMiddleware struct {
	name string
}

func NewBasicSecurityHeadersMiddleware() Middleware {
	return &basicSecurityHeadersMiddleware{
		name: "BasicSecurityHeadersMiddleware",
	}
}

func (m *basicSecurityHeadersMiddleware) Name() string {
	return m.name
}

func (m *basicSecurityHeadersMiddleware) Intercept(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "no-sniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("X-Xss-Protection", "1; mode=block")
		h.ServeHTTP(w, r)
	})
}
