// +build !js
// +build !wasm

package server

import (
	"github.com/vmware/transport-go/plank/pkg/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// enablePrometheus sets up /prometheus endpoint for metrics
func enablePrometheus(ps *platformServer) {
	ps.router.Path("/prometheus").Handler(
		middleware.BasicSecurityHeaderMiddleware.Intercept(promhttp.HandlerFor(
			prometheus.DefaultGatherer,
			promhttp.HandlerOpts{
				EnableOpenMetrics: true,
			})))
}
