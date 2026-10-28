package testutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	core_pgx_pool "github.com/vladpann/backend/golang-weather/internal/core/repository/postgres/pool/pgx"
)

const (
	pgImage    = "postgres:18.1-bookworm"
	pgDBName   = "weather_db_test"
	pgUser     = "test"
	pgPassword = "test"
)

var (
	pgOnce sync.Once
	pgHost string
	pgPort string
	pgErr  error
)

func findMigrationsDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		candidate := filepath.Join(dir, "migrations")
		if st, err := os.Stat(candidate); err == nil && st.IsDir() {
			return candidate, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("migrations dir not found")
		}
		dir = parent
	}
}

func startPostgres() {
	ctx := context.Background()

	migrationsDir, err := findMigrationsDir()
	if err != nil {
		pgErr = err
		return
	}

	scripts, err := filepath.Glob(filepath.Join(migrationsDir, "*.up.sql"))
	if err != nil {
		pgErr = err
		return
	}
	sort.Strings(scripts)

	container, err := postgres.Run(ctx, pgImage,
		postgres.WithDatabase(pgDBName),
		postgres.WithUsername(pgUser),
		postgres.WithPassword(pgPassword),
		postgres.WithInitScripts(scripts...),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		pgErr = fmt.Errorf("start postgres container: %w", err)
		return
	}

	if pgHost, err = container.Host(ctx); err != nil {
		pgErr = err
		return
	}

	port, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		pgErr = err
		return
	}
	pgPort = port.Port()
}

func SetupTestDB(t *testing.T) *core_pgx_pool.Pool {
	t.Helper()

	pgOnce.Do(startPostgres)
	require.NoError(t, pgErr, "failed to start postgres testcontainer")

	t.Setenv("POSTGRES_HOST", pgHost)
	t.Setenv("POSTGRES_PORT", pgPort)
	t.Setenv("POSTGRES_USER", pgUser)
	t.Setenv("POSTGRES_PASSWORD", pgPassword)
	t.Setenv("POSTGRES_DB", pgDBName)
	if os.Getenv("POSTGRES_TIMEOUT") == "" {
		t.Setenv("POSTGRES_TIMEOUT", (10 * time.Second).String())
	}

	ctx := context.Background()
	config := core_pgx_pool.NewConfigMust()

	pool, err := core_pgx_pool.NewPool(ctx, config)
	require.NoError(t, err, "failed to create test database pool")

	t.Cleanup(pool.Close)

	return pool
}

func CleanupTestDB(t *testing.T, pool *core_pgx_pool.Pool) {
	t.Helper()
	pool.Close()
}

func ClearTables(t *testing.T, pool *core_pgx_pool.Pool) {
	t.Helper()

	ctx := context.Background()

	queries := []string{
		"TRUNCATE TABLE weather.locations RESTART IDENTITY CASCADE",
		"TRUNCATE TABLE weather.sessions RESTART IDENTITY CASCADE",
		"TRUNCATE TABLE weather.users RESTART IDENTITY CASCADE",
	}

	for _, query := range queries {
		_, err := pool.Exec(ctx, query)
		require.NoError(t, err, fmt.Sprintf("failed to clear table: %s", query))
	}
}

func GetTestDBName() string {
	return pgDBName
}
