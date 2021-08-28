// Copyright 2019-2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package middleware

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/vmware/transport-go/plank/utils"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type RegexCacheControlRulePair struct {
	RegexpString      string
	CacheControlRule string
	compiledRegexp   *regexp.Regexp
}

type CacheControlDirective struct {
	directives []string
}

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

func CacheControlWrapperMiddleware(regExpressions []string, directive *CacheControlDirective) mux.MiddlewareFunc {
	parsed := parseRegexpString(regExpressions)
	return func(handler http.Handler) http.Handler {
		return cacheControlWrapper(handler, parsed, directive)
	}
}

func parseRegexpString(regExpressions []string) []*regexp.Regexp {
	results := make([]*regexp.Regexp, 0)
	for _, exp := range regExpressions {
		regex, err := regexp.Compile(exp)
		if err != nil {
			utils.Log.Errorln("Invalid regular expression provided as cache control matcher rule", err)
			continue
		}
		results = append(results, regex)
	}
	return results
}

func cacheControlWrapper(h http.Handler, regexps []*regexp.Regexp, directive *CacheControlDirective) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(regexps) == 0 {
			h.ServeHTTP(w, r)
			return
		}

		uriMatches := false
		for _, exp := range regexps {
			if uriMatches = exp.MatchString(r.RequestURI); uriMatches {
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
