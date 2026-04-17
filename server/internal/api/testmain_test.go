package api_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sargunv/horologia/server/internal/database"
	dbgen "github.com/sargunv/horologia/server/internal/database/gen"
	"github.com/sargunv/horologia/server/internal/taskengine"
)

// testDSN is the connection string for the shared embedded PG instance.
// It points to the default database; tests create per-test databases from the template.
var testDSN string

// testPort is the port the embedded PG instance is listening on.
var testPort uint32

// testAdminPool is a shared admin connection used to create and drop per-test databases.
var testAdminPool *pgxpool.Pool

// testTemplateName is the template database with migrations already applied.
// Tests use CREATE DATABASE ... TEMPLATE to get a fresh copy instantly.
const testTemplateName = "horologia_template"

func freePort(ctx context.Context) (uint32, error) {
	var lc net.ListenConfig
	l, err := lc.Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	_ = l.Close()
	tcpAddr, ok := l.Addr().(*net.TCPAddr)
	if !ok {
		return 0, fmt.Errorf("unexpected address type: %T", l.Addr())
	}
	if tcpAddr.Port < 0 || tcpAddr.Port > 65535 {
		return 0, fmt.Errorf("port %d out of range", tcpAddr.Port)
	}
	return uint32(tcpAddr.Port), nil //#nosec G115 -- bounds checked above
}

func TestMain(m *testing.M) {
	port, err := freePort(context.Background()) // TestMain receives *testing.M, no context available
	if err != nil {
		fmt.Fprintf(os.Stderr, "find free port: %v\n", err)
		os.Exit(1)
	}

	runtimeDir, err := os.MkdirTemp("", "horologia-embedded-postgres-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "create embedded postgres temp dir: %v\n", err)
		os.Exit(1)
	}

	pg := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().
		Port(port).
		RuntimePath(runtimeDir).
		DataPath(filepath.Join(runtimeDir, "data")).
		Database("horologia_test"))

	if err := pg.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "start embedded postgres: %v\n", err)
		os.Exit(1)
	}

	testPort = port
	testDSN = fmt.Sprintf("postgres://postgres:postgres@localhost:%d/horologia_test?sslmode=disable", testPort) //nolint:gosec // test credentials for embedded postgres

	testAdminPool, err = pgxpool.New(context.Background(), testDSN)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect admin pool: %v\n", err)
		_ = pg.Stop()
		_ = os.RemoveAll(runtimeDir)
		os.Exit(1)
	}

	// Create and migrate the template database once.
	if err := setupTemplate(); err != nil {
		fmt.Fprintf(os.Stderr, "setup template: %v\n", err)
		testAdminPool.Close()
		_ = pg.Stop()
		_ = os.RemoveAll(runtimeDir)
		os.Exit(1)
	}

	code := m.Run()

	testAdminPool.Close()

	if err := pg.Stop(); err != nil {
		fmt.Fprintf(os.Stderr, "stop embedded postgres: %v\n", err)
	}
	if err := os.RemoveAll(runtimeDir); err != nil {
		fmt.Fprintf(os.Stderr, "remove embedded postgres temp dir: %v\n", err)
	}

	os.Exit(code)
}

// setupTemplate creates the template database and runs migrations on it.
// All per-test databases are created as copies of this template.
func setupTemplate() error {
	ctx := context.Background()

	if _, err := testAdminPool.Exec(ctx, `CREATE DATABASE "`+testTemplateName+`"`); err != nil {
		return fmt.Errorf("create template db: %w", err)
	}

	templateDSN := fmt.Sprintf("postgres://postgres:postgres@localhost:%d/%s?sslmode=disable", testPort, testTemplateName)
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

	pool, err := database.OpenPool(ctx, templateDSN)
	if err != nil {
		return fmt.Errorf("open template pool: %w", err)
	}
	defer pool.Close()

	user, err := taskengine.CreateUserWithPassword(ctx, pool, testOwnerEmail, testOwnerName, testOwnerPassword, true, nil, time.Now())
	if err != nil {
		return fmt.Errorf("seed template owner: %w", err)
	}

	hash := sha256.Sum256([]byte(testSessionToken))
	tokenHash := hex.EncodeToString(hash[:])
	q := dbgen.New(pool)
	_, err = q.CreateAuthToken(ctx, dbgen.CreateAuthTokenParams{
		UserID:    user.ID,
		TokenHash: tokenHash,
		Name:      "test",
		Kind:      dbgen.AuthTokenKindSession,
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})
	if err != nil {
		return fmt.Errorf("seed template token: %w", err)
	}

	return nil
}
