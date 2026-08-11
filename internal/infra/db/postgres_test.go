package db

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/pj-hoakari/go-service-template/internal/domain"
	"github.com/pj-hoakari/go-service-template/internal/tenantctx"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var testDB *sqlx.DB

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	container, err := postgres.Run(ctx, "postgres:17-alpine",
		postgres.WithDatabase("go_service_template"),
		postgres.WithUsername("go_service_template"),
		postgres.WithPassword("go_service_template"),
		postgres.WithInitScripts(migrationPath()),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(2*time.Minute),
		),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "start PostgreSQL test container: %v\n", err)
		os.Exit(1)
	}

	databaseURL, err := container.ConnectionString(ctx, "sslmode=disable")
	if err == nil {
		testDB, err = sqlx.ConnectContext(ctx, "pgx", databaseURL)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "connect to PostgreSQL test container: %v\n", err)

		_ = container.Terminate(context.Background())

		os.Exit(1)
	}

	code := m.Run()
	_ = testDB.Close()
	_ = container.Terminate(context.Background())

	os.Exit(code)
}

func TestPostgresGreetingRepositoryRecord(t *testing.T) {
	repository := newTestRepository(t)
	ctx := tenantctx.WithTenantPublicID(context.Background(), "a1b2c3d4e5f60718")

	greeting, err := domain.NewGreeting("Ada")
	if err != nil {
		t.Fatalf("NewGreeting() error = %v", err)
	}

	if err := repository.Record(ctx, greeting); err != nil {
		t.Fatalf("Record() error = %v", err)
	}

	var rows []greetingRow
	if err := testDB.SelectContext(ctx, &rows, `SELECT tenant_public_id, name FROM greetings ORDER BY id`); err != nil {
		t.Fatalf("select recorded greetings: %v", err)
	}

	if want := []greetingRow{{TenantPublicID: "a1b2c3d4e5f60718", Name: "Ada"}}; !slices.Equal(rows, want) {
		t.Errorf("recorded greetings = %#v, want %#v", rows, want)
	}
}

func TestPostgresGreetingRepositoryRecordRequiresTenant(t *testing.T) {
	repository := newTestRepository(t)
	ctx := context.Background()

	greeting, err := domain.NewGreeting("Ada")
	if err != nil {
		t.Fatalf("NewGreeting() error = %v", err)
	}

	// The context carries no authenticated tenant, so recording must fail
	// closed instead of persisting an ownerless greeting.
	if err := repository.Record(ctx, greeting); !errors.Is(err, tenantctx.ErrMissing) {
		t.Fatalf("Record() error = %v, want %v", err, tenantctx.ErrMissing)
	}

	var count int
	if err := testDB.GetContext(ctx, &count, `SELECT COUNT(*) FROM greetings`); err != nil {
		t.Fatalf("count greetings: %v", err)
	}

	if count != 0 {
		t.Errorf("greetings count = %d, want 0", count)
	}
}

type greetingRow struct {
	TenantPublicID string `db:"tenant_public_id"`
	Name           string `db:"name"`
}

func newTestRepository(t *testing.T) *PostgresGreetingRepository {
	t.Helper()

	if _, err := testDB.Exec(`TRUNCATE greetings`); err != nil {
		t.Fatalf("truncate test database: %v", err)
	}

	return NewPostgresGreetingRepository(testDB)
}

func migrationPath() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("locate test source file")
	}

	return filepath.Join(filepath.Dir(filename), "..", "..", "..", "migrations", "000001_init.up.sql")
}
