// Copyright 2019-2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package server

import (
	"github.com/gorilla/mux"
	"github.com/vmware/transport-go/plank/pkg/middleware"
	"github.com/vmware/transport-go/plank/utils"
	"net/http"
	"regexp"
)

// SpaConfig shorthand for SinglePageApplication Config is used to configure routes for your SPAs like
// Angular or React. for example if your app index.html is served at /app and static contents like JS/CSS
// are served from /app/static, BaseUri can be set to /app and StaticAssets to "/app/assets". see config.json
// for details.
type SpaConfig struct {
	RootFolder        string            `json:"root_folder"`   // location where Plank will serve SPA
	BaseUri           string            `json:"base_uri"`      // base URI for the SPA
	StaticAssets      []string          `json:"static_assets"` // locations for static assets used by the SPA
	CacheControlRules map[string]string `json:"cache_control_rules"` // map holding glob pattern - cache-control header value

	cacheControlRulePairs []middleware.CacheControlRulePair
}

type regexCacheControlRulePair struct {
	regex      *regexp.Regexp
	cacheControlRule string
}

// NewSpaConfig takes location to where the SPA content is as an input and returns a sanitized
// instance of *SpaConfig.
func NewSpaConfig(input string) (spaConfig *SpaConfig, err error) {
	p, uri := utils.DeriveStaticURIFromPath(input)
	spaConfig = &SpaConfig{
		RootFolder:                p,
		BaseUri:                   uri,
		CacheControlRules:         make(map[string]string),
		cacheControlRulePairs: make([]middleware.CacheControlRulePair, 0),
	}

	spaConfig.CollateCacheControlRules()
	return spaConfig, err
}

// CollateCacheControlRules compiles glob patterns and stores them as an array.
func (s *SpaConfig) CollateCacheControlRules() {
	for globP, rule := range s.CacheControlRules {
		pair, err := middleware.NewCacheControlRulePair(globP, rule)
		if err != nil {
			utils.Log.Errorln("Ignoring invalid glob pattern provided as cache control matcher rule", err)
			continue
		}

		s.cacheControlRulePairs = append(s.cacheControlRulePairs, pair)
	}
}

// CacheControlMiddleware returns the middleware func to be used in route configuration
func (s *SpaConfig) CacheControlMiddleware() mux.MiddlewareFunc {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// apply cache control rule that matches first
			for _, pair := range s.cacheControlRulePairs {
				if pair.CompiledGlobPattern.Match(r.RequestURI) {
					w.Header().Set("Cache-Control", pair.CacheControlRule)
					break
				}
			}
			handler.ServeHTTP(w, r)
		})
	}
}