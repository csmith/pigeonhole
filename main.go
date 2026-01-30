package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/csmith/envflag/v2"
	"github.com/csmith/middleware"
	"github.com/csmith/slogflags"
)

var (
	authTokens = flag.String("tokens", "", "Bearer tokens (format: token:cubby1;token2:*)")
	dbPath     = flag.String("db", "pigeonhole.db", "Path to bbolt database file")
	listenAddr = flag.String("listen", ":8080", "HTTP listen address")
)

func main() {
	envflag.Parse()
	_ = slogflags.Logger(slogflags.WithSetDefault(true))

	if *authTokens == "" {
		slog.Error("No auth tokens specified")
		os.Exit(1)
	}

	tokens, err := parseTokens(*authTokens)
	if err != nil {
		slog.Error("Failed to parse auth tokens", "error", err)
		os.Exit(1)
	}

	slog.Info("Loaded auth tokens", "count", len(tokens))

	// Open database
	store, err := newStore(*dbPath)
	if err != nil {
		slog.Error("Failed to open database", "path", *dbPath, "error", err)
		os.Exit(1)
	}
	defer store.close()

	slog.Info("Opened database", "path", *dbPath)

	mux := http.NewServeMux()
	mux.Handle("GET /", handleGet(store))
	mux.Handle("POST /", handlePost(store))
	mux.Handle("DELETE /", handleDelete(store))

	server := &http.Server{
		Addr: *listenAddr,
		Handler: middleware.Chain(
			middleware.WithMiddleware(
				requireToken(tokens),
				extractCubby(),
				middleware.RealAddress(),
				middleware.Compress(),
				middleware.CrossOriginProtection(),
				middleware.Recover(middleware.WithPanicLogger(func(r *http.Request, err any) {
					slog.Error("Panic recovered", "error", err)
				})),
			),
		)(mux),
	}

	go func() {
		slog.Info("Starting server", "address", *listenAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server error", "error", err)
			os.Exit(1)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	slog.Info("Shutting down server...")
	if err := server.Shutdown(context.Background()); err != nil {
		slog.Error("Error during shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("Server stopped")
}

// parseTokens parses a token string in the format "token1:cubby1;token2:cubby2;token3:*"
// Each token maps to a single cubby, or "*" for wildcard access to all cubbies.
// Returns a map of token to cubby.
func parseTokens(s string) (map[string]string, error) {
	tokens := make(map[string]string)

	if s == "" {
		return nil, fmt.Errorf("no auth tokens specified")
	}

	pairs := strings.SplitSeq(s, ";")
	for pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		parts := strings.SplitN(pair, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid auth token '%s', expected token:cubby", pair)
		}

		token := strings.TrimSpace(parts[0])
		cubby := strings.TrimSpace(parts[1])

		if token == "" || cubby == "" {
			return nil, fmt.Errorf("invalid auth token '%s', token and cubby cannot be empty", pair)
		}

		tokens[token] = cubby
	}

	return tokens, nil
}
