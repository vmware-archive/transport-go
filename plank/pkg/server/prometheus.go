// Copyright 2019-2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

//go:build !js && !wasm
// +build !js,!wasm

package server

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/vmware/transport-go/plank/pkg/middleware"
)

// enablePrometheus sets up /prometheus endpoint for metrics
func enablePrometheus(ps *platformServer) {
	ps.router.Path("/prometheus").Handler(
		middleware.BasicSecurityHeaderMiddleware()(promhttp.HandlerFor(
			prometheus.DefaultGatherer,
			promhttp.HandlerOpts{
				EnableOpenMetrics: true,
			})))
}
