// Copyright 2019-2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package middleware

import (
	"fmt"
	"github.com/gobwas/glob"
	"github.com/gorilla/mux"
	"github.com/vmware/transport-go/plank/utils"
	"net/http"
	"strings"
	"time"
)

// CacheControlRulePair is a container that lumps together the glob pattern, cache control rule that should be applied to
// for the matching pattern and the compiled glob pattern for use in runtime. see https://github.com/gobwas/glob for
// detailed examples of glob patterns.
type CacheControlRulePair struct {
	GlobPattern         string
	CacheControlRule    string
	CompiledGlobPattern glob.Glob
}

// NewCacheControlRulePair returns a new CacheControlRulePair with the provided text glob pattern and its compiled
// counterpart and the cache control rule.
func NewCacheControlRulePair(globPattern string, cacheControlRule string) (CacheControlRulePair, error) {
	var err error
	pair := CacheControlRulePair{
		GlobPattern:      globPattern,
		CacheControlRule: cacheControlRule,
	}

	pair.CompiledGlobPattern, err = glob.Compile(globPattern)
	return pair, err
}

// CacheControlDirective is a data structure that represents the entry for Cache-Control rules
type CacheControlDirective struct {
	directives []string
}

// NewCacheControlDirective returns creates a new instance of CacheControlDirective and returns its pointer
func NewCacheControlDirective() *CacheControlDirective {
	return &CacheControlDirective{}
}

func (c *CacheControlDirective) Public() *CacheControlDirective {
	c.directives = append(c.directives, "public")
	return c
}

func (c *CacheControlDirective) Private() *CacheControlDirective {
	c.directives = append(c.directives, "private")
	return c
}

func (c *CacheControlDirective) NoCache() *CacheControlDirective {
	c.directives = append(c.directives, "no-cache")
	return c
}

func (c *CacheControlDirective) NoStore() *CacheControlDirective {
	c.directives = append(c.directives, "no-store")
	return c
}

func (c *CacheControlDirective) MaxAge(t time.Duration) *CacheControlDirective {
	c.directives = append(c.directives, fmt.Sprintf("max-age=%d", int64(t.Seconds())))
	return c
}

func (c *CacheControlDirective) SharedMaxAge(t time.Duration) *CacheControlDirective {
	c.directives = append(c.directives, fmt.Sprintf("s-maxage=%d", int64(t.Seconds())))
	return c
}

func (c *CacheControlDirective) MaxStale(t time.Duration) *CacheControlDirective {
	c.directives = append(c.directives, fmt.Sprintf("max-stale=%d", int64(t.Seconds())))
	return c
}

func (c *CacheControlDirective) MinFresh(t time.Duration) *CacheControlDirective {
	c.directives = append(c.directives, fmt.Sprintf("min-fresh=%d", int64(t.Seconds())))
	return c
}

func (c *CacheControlDirective) MustRevalidate() *CacheControlDirective {
	c.directives = append(c.directives, "must-revalidate")
	return c
}

func (c *CacheControlDirective) ProxyRevalidate() *CacheControlDirective {
	c.directives = append(c.directives, "proxy-revalidate")
	return c
}

func (c *CacheControlDirective) Immutable() *CacheControlDirective {
	c.directives = append(c.directives, "immutable")
	return c
}

func (c *CacheControlDirective) NoTransform() *CacheControlDirective {
	c.directives = append(c.directives, "no-transform")
	return c
}

func (c *CacheControlDirective) OnlyIfCached() *CacheControlDirective {
	c.directives = append(c.directives, "only-if-cached")
	return c
}

func (c *CacheControlDirective) String() string {
	return strings.Join(c.directives, ", ")
}

// CacheControlMiddleware is the middleware function to be provided as a parameter to mux.Handler()
func CacheControlMiddleware(globPatterns []string, directive *CacheControlDirective) mux.MiddlewareFunc {
	parsed := parseGlobPatterns(globPatterns)
	return func(handler http.Handler) http.Handler {
		return cacheControlWrapper(handler, parsed, directive)
	}
}

// parseGlobPatterns takes an array of glob patterns and returns an array of glob.Glob instances
func parseGlobPatterns(globPatterns []string) []glob.Glob {
	results := make([]glob.Glob, 0)
	for _, exp := range globPatterns {
		globP, err := glob.Compile(exp)
		if err != nil {
			utils.Log.Errorln("Ignoring invalid glob pattern provided as cache control matcher rule", err)
			continue
		}
		results = append(results, globP)
	}
	return results
}

// cacheControlWrapper is the internal function that actually performs the adding of cache control rules based on
// glob pattern matching and the rules provided.
func cacheControlWrapper(h http.Handler, globs []glob.Glob, directive *CacheControlDirective) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(globs) == 0 {
			h.ServeHTTP(w, r)
			return
		}

		uriMatches := false
		for _, glob := range globs {
			if uriMatches = glob.Match(r.RequestURI); uriMatches {
				break
			}
		}

		if !uriMatches {
			h.ServeHTTP(w, r)
		}

		w.Header().Set("Cache-Control", directive.String())
		h.ServeHTTP(w, r)
	})
}
