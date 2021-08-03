package server

import (
	"net/http"
)

func SampleCustomHeaderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("X-The-Number", "42")
		next.ServeHTTP(w, r)
	})
}
