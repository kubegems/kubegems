package auth

import (
	"net/http"
	"strings"
)

type MiddlewareFunc func(http.Handler) http.Handler

func NewWhitelistMiddleware(whitelist []string, onWhite http.Handler) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions {
				onWhite.ServeHTTP(w, r)
				return
			}
			for _, path := range whitelist {
				if strings.HasPrefix(r.URL.Path, path) {
					onWhite.ServeHTTP(w, r)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
