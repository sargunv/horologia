package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// OpenPool opens a pgx connection pool for the given connection string.
// The connStr should be a PostgreSQL URI, e.g. "postgres://user:pass@host/dbname".
func OpenPool(ctx context.Context, connStr string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, fmt.Errorf("open pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return pool, nil
}

// OpenSQL opens a database/sql connection for use with goose migrations.
// The connStr should be a PostgreSQL URI, e.g. "postgres://user:pass@host/dbname".
func OpenSQL(ctx context.Context, connStr string) (*sql.DB, error) {
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, fmt.Errorf("open sql: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sql: %w", err)
	}
	return db, nil
}

// NewMigrator returns a goose provider for the given database.
// The caller controls when migrations are applied via provider.Up(), etc.
func NewMigrator(db *sql.DB) (*goose.Provider, error) {
	migrations, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("migrations fs: %w", err)
	}

	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrations)
	if err != nil {
		return nil, fmt.Errorf("goose provider: %w", err)
	}

	return provider, nil
}
