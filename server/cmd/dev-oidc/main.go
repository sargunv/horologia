// dev-oidc runs a minimal OIDC provider for local development.
// It uses the zitadel/oidc example OP with an in-memory user store.
//
// Default users (username / password):
//
//	id1 / verysecure
//	id2 / verysecure
//
// Usage:
//
//	go run ./cmd/dev-oidc
package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/zitadel/oidc/v3/example/server/exampleop"
	"github.com/zitadel/oidc/v3/example/server/storage"
)

func main() {
	port := "5556"
	if p := os.Getenv("DEV_OIDC_PORT"); p != "" {
		port = p
	}

	issuer := fmt.Sprintf("http://localhost:%s/", port)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Register the Tend client.
	storage.RegisterClients(
		storage.WebClient("tend", "tend-dev-secret",
			"http://localhost:8080/auth/oidc/callback",
		),
	)

	userStore := storage.NewUserStore(issuer)
	stor := storage.NewStorage(userStore)
	router := exampleop.SetupServer(issuer, stor, logger, false)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	logger.Info("dev OIDC provider listening", "issuer", issuer, "users", "id1/verysecure, id2/verysecure")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}
