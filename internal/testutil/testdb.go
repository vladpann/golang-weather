package testutil

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"
	core_pgx_pool "github.com/vladpann/golang-weather/internal/core/repository/postgres/pool/pgx"
)

// SetupTestDB создает подключение к тестовой базе данных
func SetupTestDB(t *testing.T) *core_pgx_pool.Pool {
	t.Helper()

	// Загружаем тестовую конфигурацию из нескольких возможных мест
	// Пробуем загрузить .env.test, если не получается - используем .env
	loaded := false
	paths := []string{
		".env.test",
		"../.env.test",
		"../../.env.test",
		"../../../.env.test",
		"../../../../.env.test",
	}
	
	for _, path := range paths {
		if err := godotenv.Load(path); err == nil {
			loaded = true
			t.Logf("Loaded config from: %s", path)
			break
		}
	}
	
	if !loaded {
		t.Log("Warning: could not load .env.test, using default environment")
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

	queries := []string{
		"TRUNCATE TABLE weather.sessions CASCADE",
		"TRUNCATE TABLE weather.users CASCADE",
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
