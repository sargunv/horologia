package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/pressly/goose/v3"
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/server/internal/api"
	"github.com/sargunv/tend/server/internal/database"
	"github.com/sargunv/tend/server/internal/taskengine"
)

type config struct {
	DB        string `env:"TEND_DB,required"`
	Addr      string `env:"TEND_ADDR" envDefault:":8080"`
	LogFormat string `env:"TEND_LOG_FORMAT" envDefault:"text"`
	LogLevel  string `env:"TEND_LOG_LEVEL" envDefault:"info"`

	// OIDC config — all optional. If OIDCIssuer is empty, OIDC is disabled.
	OIDCIssuer       string `env:"TEND_OIDC_ISSUER"`
	OIDCClientID     string `env:"TEND_OIDC_CLIENT_ID"`
	OIDCClientSecret string `env:"TEND_OIDC_CLIENT_SECRET"`
	OIDCRedirectURL  string `env:"TEND_OIDC_REDIRECT_URL"`
}

func loadConfig() (config, error) {
	return env.ParseAs[config]()
}

func newLogger(cfg config) (*slog.Logger, error) {
	var level slog.Level
	if err := level.UnmarshalText([]byte(cfg.LogLevel)); err != nil {
		return nil, fmt.Errorf("invalid TEND_LOG_LEVEL %q: %w", cfg.LogLevel, err)
	}

	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	switch cfg.LogFormat {
	case "text":
		handler = slog.NewTextHandler(os.Stderr, opts)
	case "json":
		handler = slog.NewJSONHandler(os.Stderr, opts)
	default:
		return nil, fmt.Errorf("invalid TEND_LOG_FORMAT %q: expected text or json", cfg.LogFormat)
	}

	return slog.New(handler), nil
}

func openDB(cfg config) (*sql.DB, *goose.Provider, error) {
	db, err := database.Open(cfg.DB)
	if err != nil {
		return nil, nil, err
	}

	migrator, err := database.NewMigrator(db)
	if err != nil {
		_ = db.Close()
		return nil, nil, err
	}

	return db, migrator, nil
}

var rootCmd = &cobra.Command{
	Use:           "tend-server",
	Short:         "Tend backend server",
	SilenceUsage:  true,
	SilenceErrors: true,
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run the HTTP server",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		log, err := newLogger(cfg)
		if err != nil {
			return err
		}

		db, migrator, err := openDB(cfg)
		if err != nil {
			return err
		}
		defer func() { _ = db.Close() }()

		if _, err := migrator.Up(context.Background()); err != nil {
			return fmt.Errorf("auto-migrate: %w", err)
		}

		engine := &taskengine.Engine{
			DB:               db,
			Log:              log,
			CopyOnSpawnKinds: api.StoredKindCopyOnSpawn(),
		}
		handler := &api.Handler{DB: db, Log: log, Engine: engine}

		// Start the fixed_accumulating cron job.
		cronCtx, cronCancel := context.WithCancel(context.Background())
		defer cronCancel()
		go engine.RunAccumulatingCron(cronCtx, time.Minute)

		h, err := api.NewServer(handler, log)
		if err != nil {
			return fmt.Errorf("create server: %w", err)
		}

		// Mount OIDC routes if configured.
		finalHandler := h
		if cfg.OIDCIssuer != "" {
			oidcHandler, err := api.NewOIDCHandler(context.Background(), api.OIDCConfig{
				Issuer:       cfg.OIDCIssuer,
				ClientID:     cfg.OIDCClientID,
				ClientSecret: cfg.OIDCClientSecret,
				RedirectURL:  cfg.OIDCRedirectURL,
			}, handler)
			if err != nil {
				return fmt.Errorf("setup oidc: %w", err)
			}
			finalHandler = api.MountOIDC(h, oidcHandler, log)
		}

		// Mount web auth routes (cookie login/logout) and cookie-to-bearer middleware.
		finalHandler = api.MountWebAuth(finalHandler, handler)

		ln, err := net.Listen("tcp", cfg.Addr)
		if err != nil {
			return fmt.Errorf("listen: %w", err)
		}

		srv := &http.Server{Handler: finalHandler}

		errCh := make(chan error, 1)
		go func() { errCh <- srv.Serve(ln) }()

		log.Info("server listening", "addr", ln.Addr().String())

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		select {
		case err := <-errCh:
			return err
		case sig := <-sigCh:
			log.Info("shutting down", "signal", sig.String())
		}

		if err := srv.Shutdown(context.Background()); err != nil {
			return fmt.Errorf("shutdown: %w", err)
		}

		// ListenAndServe returns ErrServerClosed after Shutdown; drain it.
		if err := <-errCh; err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}

		return nil
	},
}

var createAdminCmd = &cobra.Command{
	Use:   "create-admin",
	Short: "Create a global owner user",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		log, err := newLogger(cfg)
		if err != nil {
			return err
		}

		db, migrator, err := openDB(cfg)
		if err != nil {
			return err
		}
		defer func() { _ = db.Close() }()

		if _, err := migrator.Up(context.Background()); err != nil {
			return fmt.Errorf("auto-migrate: %w", err)
		}

		email, _ := cmd.Flags().GetString("email")
		name, _ := cmd.Flags().GetString("name")
		password, _ := cmd.Flags().GetString("password")

		user, err := api.CreateUserWithPassword(context.Background(), db, email, name, password, true)
		if err != nil {
			return fmt.Errorf("create admin: %w", err)
		}

		log.Info("admin created", "id", user.ID, "email", user.Email, "name", user.Name)
		return nil
	},
}

func init() {
	createAdminCmd.Flags().String("email", "", "admin email (required)")
	createAdminCmd.Flags().String("name", "", "admin display name (required)")
	createAdminCmd.Flags().String("password", "", "admin password (required)")
	_ = createAdminCmd.MarkFlagRequired("email")
	_ = createAdminCmd.MarkFlagRequired("name")
	_ = createAdminCmd.MarkFlagRequired("password")
}

func main() {
	rootCmd.AddCommand(serveCmd, migrateCmd, createAdminCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
