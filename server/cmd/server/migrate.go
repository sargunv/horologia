package main

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/sargunv/tend/server/internal/database"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Database migration commands",
}

var migrateUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Apply all pending migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		log, err := newLogger(cfg)
		if err != nil {
			return err
		}

		db, err := database.OpenSQL(context.Background(), cfg.DB)
		if err != nil {
			return err
		}
		defer func() { _ = db.Close() }()

		migrator, err := database.NewMigrator(db)
		if err != nil {
			return err
		}

		results, err := migrator.Up(context.Background())
		if err != nil {
			return err
		}

		if len(results) == 0 {
			log.Info("no pending migrations")
			return nil
		}

		for _, r := range results {
			log.Info("applied migration", "path", r.Source.Path, "duration", r.Duration)
		}
		return nil
	},
}

var migrateStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Print migration status",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		log, err := newLogger(cfg)
		if err != nil {
			return err
		}

		db, err := database.OpenSQL(context.Background(), cfg.DB)
		if err != nil {
			return err
		}
		defer func() { _ = db.Close() }()

		migrator, err := database.NewMigrator(db)
		if err != nil {
			return err
		}

		statuses, err := migrator.Status(context.Background())
		if err != nil {
			return err
		}

		for _, s := range statuses {
			path := "<missing>"
			if s.Source != nil {
				path = s.Source.Path
			}
			log.Info("migration", "state", s.State, "path", path)
		}
		return nil
	},
}

func init() {
	migrateCmd.AddCommand(migrateUpCmd, migrateStatusCmd)
}
