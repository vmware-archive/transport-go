// Copyright 2019-2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

// +build !js
// +build !wasm

package middleware

import (
	"github.com/gorilla/mux"
	"github.com/vmware/transport-go/plank/pkg/metrics"
	"net/http"
	"strings"
)

func SamplePrometheusMetricsMiddleware() mux.MiddlewareFunc {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			split := strings.Split(r.URL.Path, "/")
			metrics.PageViewCounter.WithLabelValues(split[1], r.URL.Path).Inc()
			handler.ServeHTTP(w, r)
		})
	}
}
