package testutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"
	core_pgx_pool "github.com/vladpann/backend/golang-weather/internal/core/repository/postgres/pool/pgx"
)

func findEnvTest() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		candidate := filepath.Join(dir, ".env.test")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf(".env.test not found")
		}
		dir = parent
	}
}

// SetupTestDB connection to test DB
func SetupTestDB(t *testing.T) *core_pgx_pool.Pool {
	t.Helper()

	path, err := findEnvTest()
	if err == nil {
		require.NoError(t, godotenv.Overload(path), "failed to load .env.test")
		t.Logf("Loaded config from: %s", path)
	} else {
		require.NotEmpty(t, os.Getenv("POSTGRES_DB"),
			".env.test not found and POSTGRES_* env vars are not set")
		t.Log("Using POSTGRES_* from environment (CI)")
	}

	ctx := context.Background()
	config := core_pgx_pool.NewConfigMust()

	pool, err := core_pgx_pool.NewPool(ctx, config)
	require.NoError(t, err, "failed to create test database pool")

	return pool
}

// CleanupTestDB закрывает подключение к базе данных
func CleanupTestDB(t *testing.T, pool *core_pgx_pool.Pool) {
	t.Helper()
	pool.Close()
}

// ClearTables очищает таблицы в тестовой базе данных
func ClearTables(t *testing.T, pool *core_pgx_pool.Pool) {
	t.Helper()

	ctx := context.Background()

	// Очищаем в обратном порядке зависимостей:
	// сначала зависимые таблицы, потом главные
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

// GetTestDBName возвращает имя тестовой базы данных
func GetTestDBName() string {
	dbName := os.Getenv("POSTGRES_DB")
	if dbName == "" {
		return "weather_db_test"
	}
	return dbName
}
