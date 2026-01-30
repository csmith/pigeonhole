package main

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
)

var contextKeyCubby = struct{}{}

func requireToken(tokens map[string]string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			if !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
			cubby := r.Context().Value(contextKeyCubby)

			if tokens[token] != cubby && tokens[token] != "*" {
				slog.Info("Denied access to cubby", "token", token, "cubby", cubby, "client", r.RemoteAddr)
				http.Error(w, "Unauthorized", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func extractCubby() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cubby := strings.TrimPrefix(r.URL.Path, "/")
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), contextKeyCubby, cubby)))
		})
	}
}
