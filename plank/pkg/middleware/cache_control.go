// Copyright 2019-2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package middleware

import (
	"fmt"
	"net/http"
	"time"
)

type CacheControlWrapperMiddleware struct {
	name   string
	maxAge time.Duration
}

func NewCacheControlWrapperMiddleware(maxAge time.Duration) Middleware {
	return &CacheControlWrapperMiddleware{
		name:   "CacheControlWrapperMiddleware",
		maxAge: maxAge,
	}
}

func (m *CacheControlWrapperMiddleware) Name() string {
	return m.name
}

func (m *CacheControlWrapperMiddleware) Intercept(h http.Handler) http.Handler {
	return cacheControlWrapper(h, m.maxAge)
}

func cacheControlWrapper(h http.Handler, maxAge time.Duration) http.Handler {
	maxAgeInt := (int64)(maxAge.Seconds())
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cacheStr := ""
		if maxAge == 0 {
			cacheStr = "no-cache"
		} else {
			cacheStr = "max-age=" + fmt.Sprintf("%d", maxAgeInt)
		}
		w.Header().Set("Cache-Control", fmt.Sprintf("public, %s", cacheStr))
		h.ServeHTTP(w, r)
	})
}
