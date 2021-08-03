// +build !js
// +build !wasm

package middleware

import (
	"github.com/jooskim/plank/pkg/metrics"
	"net/http"
	"strings"
	"time"
)

type SamplePrometheusMetricsMiddleware struct {
	name string
}

func NewSamplePrometheusMetricsMiddleware(maxAge time.Duration) Middleware {
	return &SamplePrometheusMetricsMiddleware{
		name: "SamplePrometheusMetricsMiddleware",
	}
}

func (m *SamplePrometheusMetricsMiddleware) Name() string {
	return m.name
}

func (m *SamplePrometheusMetricsMiddleware) Intercept(h http.Handler) http.Handler {
	return pageViewsCountMetricsWrapper(h)
}

func pageViewsCountMetricsWrapper(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		split := strings.Split(r.URL.Path, "/")
		metrics.PageViewCounter.WithLabelValues(split[1], r.URL.Path).Inc()
		h.ServeHTTP(w, r)
	})
}
