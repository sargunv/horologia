package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pressly/goose/v3"
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/server/internal/api"
	"github.com/sargunv/tend/server/internal/config"
	"github.com/sargunv/tend/server/internal/cron"
	"github.com/sargunv/tend/server/internal/database"
	dbgen "github.com/sargunv/tend/server/internal/database/gen"
	"github.com/sargunv/tend/server/internal/pwdcheck"
	"github.com/sargunv/tend/server/internal/taskengine"
)

func newLogger(cfg config.Config) (*slog.Logger, error) {
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

func openMigrationDB(ctx context.Context, cfg config.Config) (*sql.DB, *goose.Provider, error) {
	db, err := database.OpenSQL(ctx, cfg.DB)
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

// migrateAndOpenPool runs database migrations and returns an application pool.
func migrateAndOpenPool(ctx context.Context, cfg config.Config) (*pgxpool.Pool, error) {
	migrationDB, migrator, err := openMigrationDB(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if _, err := migrator.Up(ctx); err != nil {
		_ = migrationDB.Close()
		return nil, fmt.Errorf("auto-migrate: %w", err)
	}
	_ = migrationDB.Close()

	pool, err := database.OpenPool(ctx, cfg.DB)
	if err != nil {
		return nil, err
	}
	return pool, nil
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
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		log, err := newLogger(cfg)
		if err != nil {
			return err
		}

		pool, err := migrateAndOpenPool(cmd.Context(), cfg)
		if err != nil {
			return err
		}
		defer pool.Close()

		checker := buildPasswordChecker(cfg)

		handler := &api.Handler{
			Pool:                   pool,
			Log:                    log,
			SecureCookies:          cfg.SecureCookies,
			OIDCEnabled:            cfg.OIDCIssuer != "",
			OIDCLabel:              cfg.OIDCLabel,
			OIDCAutoRedirect:       cfg.OIDCAutoRedirect,
			OIDCLinkConsentEnabled: cfg.OIDCIssuer != "" && cfg.PasswordAuthEnabled && cfg.OIDCLinkConsent,
			PasswordAuthEnabled:    cfg.PasswordAuthEnabled,
			PasswordChecker:        checker,
		}

		if handler.OIDCLinkConsentEnabled {
			linkCH, err := api.NewLinkCookieHandler(cfg.SecureCookies)
			if err != nil {
				return fmt.Errorf("setup link consent cookie: %w", err)
			}
			handler.LinkCookieHandler = linkCH
		}

		// Start the fixed_accumulating cron job.
		cronCtx, cronCancel := context.WithCancel(cmd.Context())
		defer cronCancel()
		go cron.RunAccumulatingCron(cronCtx, pool, log, time.Minute)

		h, err := api.NewServer(handler, log)
		if err != nil {
			return fmt.Errorf("create server: %w", err)
		}

		// Mount OIDC routes if configured.
		finalHandler := h
		if cfg.OIDCIssuer != "" {
			oidcHandler, err := api.NewOIDCHandler(cmd.Context(), api.OIDCConfig{
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

		// Mount health check at /healthz, API under /api prefix, and SPA at root.
		finalHandler = api.MountRoot(finalHandler, pool, log)

		// Bootstrap initial owner if configured and no users exist yet.
		if cfg.InitOwnerEmail != "" {
			if err := bootstrapOwner(cmd.Context(), pool, log, cfg, checker); err != nil {
				return fmt.Errorf("bootstrap owner: %w", err)
			}
		}

		ln, err := net.Listen("tcp", cfg.Addr)
		if err != nil {
			return fmt.Errorf("listen: %w", err)
		}

		srv := &http.Server{Handler: finalHandler, ReadHeaderTimeout: 10 * time.Second}

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

		shutdownCtx, shutdownCancel := context.WithTimeout(context.WithoutCancel(cmd.Context()), 30*time.Second)
		defer shutdownCancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown: %w", err)
		}

		// Serve returns ErrServerClosed after Shutdown; drain it.
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
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		log, err := newLogger(cfg)
		if err != nil {
			return err
		}

		pool, err := migrateAndOpenPool(cmd.Context(), cfg)
		if err != nil {
			return err
		}
		defer pool.Close()

		email, _ := cmd.Flags().GetString("email")
		name, _ := cmd.Flags().GetString("name")
		password, _ := cmd.Flags().GetString("password")

		checker := buildPasswordChecker(cfg)
		user, err := taskengine.CreateUserWithPassword(cmd.Context(), pool, email, name, password, true, checker, time.Now())
		if err != nil {
			return fmt.Errorf("create admin: %w", err)
		}

		log.Info("admin created", "id", user.ID, "email", user.Email, "name", user.Name)
		return nil
	},
}

var healthcheckCmd = &cobra.Command{
	Use:   "healthcheck",
	Short: "Check server health (for use in Docker HEALTHCHECK)",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		addr := cfg.Addr
		// Default host to localhost if addr is just a port (e.g. ":8080").
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return fmt.Errorf("parse TEND_ADDR %q: %w", addr, err)
		}
		if host == "" {
			host = "localhost"
		}

		url := fmt.Sprintf("http://%s/healthz", net.JoinHostPort(host, port))

		ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("health check failed: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
			return fmt.Errorf("health check returned status %d: %s", resp.StatusCode, string(body))
		}

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

func buildPasswordChecker(cfg config.Config) pwdcheck.Checker {
	if !cfg.HIBPEnabled {
		return nil
	}
	return pwdcheck.NewHIBPChecker(&http.Client{Timeout: 5 * time.Second})
}

// bootstrapOwner creates the initial owner user if no users exist yet.
// This is a no-op when users are already present in the database.
// Handles concurrent starts gracefully: if two instances race, the unique
// constraint on email causes the second insert to fail, which is treated as success.
func bootstrapOwner(ctx context.Context, pool *pgxpool.Pool, log *slog.Logger, cfg config.Config, checker pwdcheck.Checker) error {
	var count int64
	err := pool.QueryRow(ctx, "SELECT count(*) FROM users WHERE is_owner = true").Scan(&count)
	if err != nil {
		return fmt.Errorf("count users: %w", err)
	}

	if count > 0 {
		log.Info("skipping owner bootstrap: owner already exists")
		return nil
	}

	var user dbgen.User
	var createErr error
	if cfg.PasswordAuthEnabled {
		user, createErr = taskengine.CreateUserWithPassword(ctx, pool, cfg.InitOwnerEmail, cfg.InitOwnerName, cfg.InitOwnerPassword, true, checker, time.Now())
	} else {
		user, createErr = taskengine.CreateUserWithoutPassword(ctx, pool, cfg.InitOwnerEmail, cfg.InitOwnerName, true, time.Now())
	}
	if createErr != nil {
		// Unique violation means another instance already created the user.
		var pgErr *pgconn.PgError
		if errors.As(createErr, &pgErr) && pgErr.Code == "23505" {
			log.Info("owner already created by another instance")
			return nil
		}
		return fmt.Errorf("create owner: %w", createErr)
	}

	log.Info("created initial owner", "id", user.ID, "email", user.Email, "name", user.Name)
	return nil
}

func main() {
	migrateCmd.AddCommand(migrateUpCmd, migrateStatusCmd)
	rootCmd.AddCommand(serveCmd, migrateCmd, createAdminCmd, healthcheckCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
