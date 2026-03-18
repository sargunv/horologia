package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
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

		db, migrator, err := openDB(cfg)
		if err != nil {
			return err
		}
		defer func() { _ = db.Close() }()

		results, err := migrator.Up(context.Background())
		if err != nil {
			return err
		}

		if len(results) == 0 {
			fmt.Println("no pending migrations")
			return nil
		}

		for _, r := range results {
			fmt.Printf("applied: %s (%s)\n", r.Source.Path, r.Duration)
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

		db, migrator, err := openDB(cfg)
		if err != nil {
			return err
		}
		defer func() { _ = db.Close() }()

		statuses, err := migrator.Status(context.Background())
		if err != nil {
			return err
		}

		for _, s := range statuses {
			path := "<missing>"
			if s.Source != nil {
				path = s.Source.Path
			}
			fmt.Printf("%-10s %s\n", s.State, path)
		}
		return nil
	},
}

func init() {
	migrateCmd.AddCommand(migrateUpCmd, migrateStatusCmd)
}
