// +build !js
// +build !wasm

package metrics

import "github.com/prometheus/client_golang/prometheus"

var PageViewCounter = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "page_views_count",
		Help: "How many times a page was viewed",
	},
	[]string{"group", "uri"})
