package api_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sargunv/tend/server/internal/database"
)

// testDSN is the connection string for the shared embedded PG instance.
// It points to the default database; tests create per-test databases from the template.
var testDSN string

// testTemplateName is the template database with migrations already applied.
// Tests use CREATE DATABASE ... TEMPLATE to get a fresh copy instantly.
const testTemplateName = "tend_template"

func TestMain(m *testing.M) {
	pg := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().
		Port(15432).
		Database("tend_test"))

	if err := pg.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "start embedded postgres: %v\n", err)
		os.Exit(1)
	}

	testDSN = "postgres://postgres:postgres@localhost:15432/tend_test?sslmode=disable" //nolint:gosec // test credentials for embedded postgres

	// Create and migrate the template database once.
	if err := setupTemplate(); err != nil {
		fmt.Fprintf(os.Stderr, "setup template: %v\n", err)
		_ = pg.Stop()
		os.Exit(1)
	}

	code := m.Run()

	if err := pg.Stop(); err != nil {
		fmt.Fprintf(os.Stderr, "stop embedded postgres: %v\n", err)
	}

	os.Exit(code)
}

// setupTemplate creates the template database and runs migrations on it.
// All per-test databases are created as copies of this template.
func setupTemplate() error {
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, testDSN)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer pool.Close()

	if _, err := pool.Exec(ctx, "CREATE DATABASE "+testTemplateName); err != nil {
		return fmt.Errorf("create template db: %w", err)
	}

	templateDSN := fmt.Sprintf("postgres://postgres:postgres@localhost:15432/%s?sslmode=disable", testTemplateName)
	sqlDB, err := database.OpenSQL(ctx, templateDSN)
	if err != nil {
		return fmt.Errorf("open template sql: %w", err)
	}
	defer func() { _ = sqlDB.Close() }()

	migrator, err := database.NewMigrator(sqlDB)
	if err != nil {
		return fmt.Errorf("new migrator: %w", err)
	}
	if _, err := migrator.Up(ctx); err != nil {
		return fmt.Errorf("migrate template: %w", err)
	}

	return nil
}
