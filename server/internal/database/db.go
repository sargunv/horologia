package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"

	"github.com/pressly/goose/v3"
	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"

	"github.com/sargunv/tend/server/internal/database/gen"
	"github.com/sargunv/tend/server/internal/types"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Open opens a SQLite database at the given path and configures pragmas
// (foreign keys, WAL mode, busy timeout). It does not run migrations.
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
			_ = db.Close()
			return nil, fmt.Errorf("pragma %q: %w", p, err)
		}
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

	provider, err := goose.NewProvider(goose.DialectSQLite3, db, migrations)
	if err != nil {
		return nil, fmt.Errorf("goose provider: %w", err)
	}

	return provider, nil
}

var defaultStatuses = []gen.CreateTaskStatusParams{
	{Name: "todo", Category: "initial", Position: 0},
	{Name: "done", Category: "completion", Position: 1},
}

var defaultEffortLevels = []gen.CreateTaskEffortLevelParams{
	{Name: "small", Position: 0},
	{Name: "medium", Position: 1},
	{Name: "large", Position: 2},
}

var defaultPriorityLevels = []gen.CreateTaskPriorityLevelParams{
	{Name: "low", Position: 0},
	{Name: "medium", Position: 1},
	{Name: "high", Position: 2},
}

// CreateSpaceWithDefaults creates a space, its default statuses, effort levels,
// and priority levels, and adds the creator as an admin member, all in a single transaction.
func CreateSpaceWithDefaults(ctx context.Context, db *sql.DB, slug, name, description string, creatorUserID int64) (gen.Space, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return gen.Space{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	q := gen.New(tx)
	now := types.Now()

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

	for _, e := range defaultEffortLevels {
		e.SpaceSlug = space.Slug
		if _, err := q.CreateTaskEffortLevel(ctx, e); err != nil {
			return gen.Space{}, fmt.Errorf("create default effort level %q: %w", e.Name, err)
		}
	}

	for _, p := range defaultPriorityLevels {
		p.SpaceSlug = space.Slug
		if _, err := q.CreateTaskPriorityLevel(ctx, p); err != nil {
			return gen.Space{}, fmt.Errorf("create default priority level %q: %w", p.Name, err)
		}
	}

	// Add creator as admin.
	if _, err := q.CreateSpaceMember(ctx, gen.CreateSpaceMemberParams{
		SpaceSlug: space.Slug,
		UserID:    creatorUserID,
		Role:      "admin",
		CreatedAt: now,
	}); err != nil {
		return gen.Space{}, fmt.Errorf("create admin member: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return gen.Space{}, fmt.Errorf("commit: %w", err)
	}

	return space, nil
}

// CreateUserWithPassword creates a user with a bcrypt-hashed password.
func CreateUserWithPassword(ctx context.Context, db *sql.DB, email, name, password string, isOwner bool) (gen.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return gen.User{}, fmt.Errorf("hash password: %w", err)
	}
	hashStr := string(hash)
	now := types.Now()

	var ownerFlag int64
	if isOwner {
		ownerFlag = 1
	}

	q := gen.New(db)
	return q.CreateUser(ctx, gen.CreateUserParams{
		Email:        email,
		Name:         name,
		PasswordHash: &hashStr,
		IsOwner:      ownerFlag,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
}
