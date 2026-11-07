package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequireToken(t *testing.T) {
	tokens := map[string]string{
		"cubby-token": "a",
		"admin-token": "*",
	}

	handler := extractCubby()(
		requireToken(tokens)(
			http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusTeapot)
				},
			),
		),
	)

	tests := []struct {
		name   string
		path   string
		auth   string
		status int
	}{
		{"no header", "/a", "", http.StatusUnauthorized},
		{"not bearer", "/a", "Basic cubby-token", http.StatusUnauthorized},
		{"unknown token on root", "/", "Bearer garbage", http.StatusForbidden},
		{"unknown token on cubby", "/a", "Bearer garbage", http.StatusForbidden},
		{"empty token on root", "/", "Bearer ", http.StatusForbidden},
		{"cubby token on own cubby", "/a", "Bearer cubby-token", http.StatusTeapot},
		{"cubby token on other cubby", "/b", "Bearer cubby-token", http.StatusForbidden},
		{"cubby token on root", "/", "Bearer cubby-token", http.StatusForbidden},
		{"token prefix", "/a", "Bearer cubby-tok", http.StatusForbidden},
		{"wildcard token on cubby", "/b", "Bearer admin-token", http.StatusTeapot},
		{"wildcard token on root", "/", "Bearer admin-token", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, tt.path, nil)
				if tt.auth != "" {
					req.Header.Set("Authorization", tt.auth)
				}
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, req)
				assert.Equal(t, tt.status, rec.Code)
			},
		)
	}
}
