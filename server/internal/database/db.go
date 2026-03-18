package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"time"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"

	"github.com/sargunv/tend/server/internal/database/gen"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Open opens a SQLite database at the given path, configures pragmas
// (foreign keys, WAL mode, busy timeout), and runs pending migrations.
func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	pragmas := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
		"PRAGMA busy_timeout = 5000",
		"PRAGMA synchronous = NORMAL",
	}
	for _, p := range pragmas {
		if _, err := db.ExecContext(context.Background(), p); err != nil {
			return nil, fmt.Errorf("pragma %q: %w", p, err)
		}
	}

	migrations, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("migrations fs: %w", err)
	}

	provider, err := goose.NewProvider(goose.DialectSQLite3, db, migrations)
	if err != nil {
		return nil, fmt.Errorf("goose provider: %w", err)
	}
	if _, err := provider.Up(context.Background()); err != nil {
		return nil, fmt.Errorf("goose up: %w", err)
	}

	return db, nil
}

var defaultStatuses = []gen.CreateTaskStatusParams{
	{Name: "todo", Category: "initial", Position: 0},
	{Name: "done", Category: "completion", Position: 1},
}

// CreateSpaceWithDefaults creates a space and its default statuses
// ("todo" and "done") in a single transaction.
func CreateSpaceWithDefaults(ctx context.Context, db *sql.DB, slug, name, description string) (gen.Space, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return gen.Space{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	q := gen.New(tx)
	now := time.Now().UTC().Format(time.RFC3339)

	space, err := q.CreateSpace(ctx, gen.CreateSpaceParams{
		Slug:        slug,
		Name:        name,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		return gen.Space{}, fmt.Errorf("create space: %w", err)
	}

	for _, s := range defaultStatuses {
		s.SpaceSlug = space.Slug
		if _, err := q.CreateTaskStatus(ctx, s); err != nil {
			return gen.Space{}, fmt.Errorf("create default status %q: %w", s.Name, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return gen.Space{}, fmt.Errorf("commit: %w", err)
	}

	return space, nil
}
