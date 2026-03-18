package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/caarlos0/env/v11"
	"github.com/pressly/goose/v3"
	"github.com/spf13/cobra"

	"github.com/sargunv/tend/server/internal/database"
)

type config struct {
	DB   string `env:"TEND_DB,required"`
	Addr string `env:"TEND_ADDR" envDefault:":8080"`
}

func loadConfig() (config, error) {
	return env.ParseAs[config]()
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

		db, migrator, err := openDB(cfg)
		if err != nil {
			return err
		}
		defer func() { _ = db.Close() }()

		if _, err := migrator.Up(context.Background()); err != nil {
			return fmt.Errorf("auto-migrate: %w", err)
		}

		fmt.Printf("tend-server: ready on %s\n", cfg.Addr)
		return nil
	},
}

var createAdminCmd = &cobra.Command{
	Use:   "create-admin",
	Short: "Create an admin user",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		db, err := database.Open(cfg.DB)
		if err != nil {
			return err
		}
		defer func() { _ = db.Close() }()

		// TODO: implement
		fmt.Println("create-admin: not yet implemented")
		return nil
	},
}

func main() {
	rootCmd.AddCommand(serveCmd, migrateCmd, createAdminCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
