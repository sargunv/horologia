package main

import (
	"github.com/spf13/cobra"

	"github.com/sargunv/horologia/server/internal/config"
	"github.com/sargunv/horologia/server/internal/database"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Database migration commands",
}

var migrateUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Apply all pending migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		log, err := newLogger(cfg)
		if err != nil {
			return err
		}

		db, err := database.OpenSQL(cmd.Context(), cfg.DB)
		if err != nil {
			return err
		}
		defer func() { _ = db.Close() }()

		migrator, err := database.NewMigrator(db)
		if err != nil {
			return err
		}

		results, err := migrator.Up(cmd.Context())
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
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		log, err := newLogger(cfg)
		if err != nil {
			return err
		}

		db, err := database.OpenSQL(cmd.Context(), cfg.DB)
		if err != nil {
			return err
		}
		defer func() { _ = db.Close() }()

		migrator, err := database.NewMigrator(db)
		if err != nil {
			return err
		}

		statuses, err := migrator.Status(cmd.Context())
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
