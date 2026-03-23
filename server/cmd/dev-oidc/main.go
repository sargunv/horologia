// dev-oidc runs a minimal OIDC provider for local development.
// It uses the zitadel/oidc example OP with a custom user store.
//
// Users (username / password):
//
//	admin@localhost / password
//	test@localhost  / password
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
	"golang.org/x/text/language"
)

type devUserStore struct {
	users    map[string]*storage.User
	clientID string
}

func (s *devUserStore) GetUserByID(id string) *storage.User      { return s.users[id] }
func (s *devUserStore) GetUserByUsername(u string) *storage.User { return s.users[u] }
func (s *devUserStore) ExampleClientID() string                  { return s.clientID }

func main() {
	port := "5556"
	if p := os.Getenv("DEV_OIDC_PORT"); p != "" {
		port = p
	}

	issuer := fmt.Sprintf("http://localhost:%s/", port)
	callbackURL := "http://localhost:5173/api/auth/oidc/callback"
	if u := os.Getenv("DEV_OIDC_CALLBACK_URL"); u != "" {
		callbackURL = u
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	storage.RegisterClients(
		storage.WebClient("tend", "tend-dev-secret", callbackURL),
	)

	admin := &storage.User{
		ID:                "admin@localhost",
		Username:          "admin@localhost",
		Password:          "password",
		FirstName:         "Admin",
		LastName:          "User",
		Email:             "admin@localhost",
		EmailVerified:     true,
		PreferredLanguage: language.English,
		IsAdmin:           true,
	}
	test := &storage.User{
		ID:                "test@localhost",
		Username:          "test@localhost",
		Password:          "password",
		FirstName:         "Test",
		LastName:          "User",
		Email:             "test@localhost",
		EmailVerified:     true,
		PreferredLanguage: language.English,
	}

	userStore := &devUserStore{
		users: map[string]*storage.User{
			admin.ID: admin,
			test.ID:  test,
		},
		clientID: "tend",
	}

	stor := storage.NewStorage(userStore)
	router := exampleop.SetupServer(issuer, stor, logger, false)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	logger.Info("dev OIDC provider listening", "issuer", issuer, "users", "admin@localhost/password, test@localhost/password")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}
