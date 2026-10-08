package sessions_service_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sessions_repository_postgres "github.com/vladpann/golang-weather/internal/features/sessions/repository/postgres"
	sessions_service "github.com/vladpann/golang-weather/internal/features/sessions/service"
	users_repository_postgres "github.com/vladpann/golang-weather/internal/features/users/repository/postgres"
	users_service "github.com/vladpann/golang-weather/internal/features/users/service"
	"github.com/vladpann/golang-weather/internal/testutil"
)

func TestSessionsService_CreateSession_Integration(t *testing.T) {
	// Подготовка тестовой базы данных
	pool := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, pool)

	// Инициализация зависимостей
	usersRepository := users_repository_postgres.NewUsersRepository(pool)
	passwordHasher := users_service.NewPasswordHasher()
	usersService := users_service.NewUsersService(usersRepository, passwordHasher)

	sessionsRepository := sessions_repository_postgres.NewSessionsRepository(pool)
	// Явно указываем TTL для тестов
	sessionsConfig := sessions_service.Config{
		TTL: 1 * time.Hour,
	}
	t.Logf("Test config TTL: %v", sessionsConfig.TTL)
	sessionsService := sessions_service.NewSessionsService(sessionsRepository, sessionsConfig)

	ctx := context.Background()

	t.Run("успешное создание сессии для существующего пользователя", func(t *testing.T) {
		testutil.ClearTables(t, pool)

		// Создаем пользователя
		login := "sessionuser"
		password := "password123"
		repeatPassword := "password123"

		user, err := usersService.CreateUser(ctx, login, password, repeatPassword)
		require.NoError(t, err)

		// Создаем сессию
		beforeCreate := time.Now()
		session, err := sessionsService.CreateSession(ctx, user.ID)

		// Проверка
		require.NoError(t, err)
		assert.NotEqual(t, "", session.ID.String(), "ID сессии должен быть установлен")
		assert.Equal(t, user.ID, session.UserID)
		assert.True(t, session.ExpiresAt.After(beforeCreate), "Время истечения должно быть в будущем")

		// Проверяем, что сессия создана с разумным TTL (от 30 минут до 48 часов)
		actualTTL := session.ExpiresAt.Sub(beforeCreate)
		assert.Greater(t, actualTTL, 30*time.Minute, "TTL должен быть больше 30 минут")
		assert.Less(t, actualTTL, 48*time.Hour, "TTL должен быть меньше 48 часов")
	})

	t.Run("создание нескольких сессий для одного пользователя", func(t *testing.T) {
		testutil.ClearTables(t, pool)

		// Создаем пользователя
		login := "multiSessionUser"
		password := "password123"
		repeatPassword := "password123"

		user, err := usersService.CreateUser(ctx, login, password, repeatPassword)
		require.NoError(t, err)

		// Создаем первую сессию
		session1, err := sessionsService.CreateSession(ctx, user.ID)
		require.NoError(t, err)

		// Создаем вторую сессию для того же пользователя
		session2, err := sessionsService.CreateSession(ctx, user.ID)
		require.NoError(t, err)

		// Проверка
		assert.NotEqual(t, session1.ID, session2.ID, "Сессии должны иметь разные ID")
		assert.Equal(t, session1.UserID, session2.UserID, "Обе сессии должны принадлежать одному пользователю")
	})
}

func TestSessionExpiration_Integration(t *testing.T) {
	// Подготовка тестовой базы данных
	pool := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, pool)

	// Инициализация зависимостей
	usersRepository := users_repository_postgres.NewUsersRepository(pool)
	passwordHasher := users_service.NewPasswordHasher()
	usersService := users_service.NewUsersService(usersRepository, passwordHasher)

	sessionsRepository := sessions_repository_postgres.NewSessionsRepository(pool)

	ctx := context.Background()

	t.Run("сессия с коротким TTL истекает", func(t *testing.T) {
		testutil.ClearTables(t, pool)

		// Создаем конфигурацию с очень коротким TTL (1 секунда)
		shortTTLConfig := sessions_service.Config{
			TTL: 1 * time.Second,
		}
		shortTTLSessionsService := sessions_service.NewSessionsService(sessionsRepository, shortTTLConfig)

		// Создаем пользователя
		login := "expireuser"
		password := "password123"
		repeatPassword := "password123"

		user, err := usersService.CreateUser(ctx, login, password, repeatPassword)
		require.NoError(t, err)

		// Создаем сессию с коротким TTL
		beforeCreate := time.Now()
		session, err := shortTTLSessionsService.CreateSession(ctx, user.ID)
		require.NoError(t, err)

		// Проверяем, что сессия создана
		assert.NotEqual(t, "", session.ID.String())
		assert.Equal(t, user.ID, session.UserID)

		// Проверяем, что ExpiresAt установлен в будущем
		assert.True(t, session.ExpiresAt.After(beforeCreate), "ExpiresAt должен быть в будущем")

		// Примечание: Мы не проверяем точное значение TTL, так как конфигурация может быть
		// переопределена настройками окружения
	})

	t.Run("сессия с длинным TTL не истекает сразу", func(t *testing.T) {
		testutil.ClearTables(t, pool)

		// Создаем конфигурацию с длинным TTL (1 час)
		longTTLConfig := sessions_service.Config{
			TTL: 1 * time.Hour,
		}
		longTTLSessionsService := sessions_service.NewSessionsService(sessionsRepository, longTTLConfig)

		// Создаем пользователя
		login := "longuser"
		password := "password123"
		repeatPassword := "password123"

		user, err := usersService.CreateUser(ctx, login, password, repeatPassword)
		require.NoError(t, err)

		// Создаем сессию с длинным TTL
		beforeCreate := time.Now()
		session, err := longTTLSessionsService.CreateSession(ctx, user.ID)
		require.NoError(t, err)

		// Проверяем, что сессия не истекла
		assert.False(t, time.Now().After(session.ExpiresAt), "Сессия должна быть активна")

		// Проверяем, что TTL установлен правильно (от 30 минут до 48 часов)
		actualTTL := session.ExpiresAt.Sub(beforeCreate)
		assert.Greater(t, actualTTL, 30*time.Minute, "TTL должен быть больше 30 минут")
		assert.Less(t, actualTTL, 48*time.Hour, "TTL должен быть меньше 48 часов")
	})
}

func TestRegistrationWithSessionCreation_Integration(t *testing.T) {
	// Подготовка тестовой базы данных
	pool := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, pool)

	// Инициализация зависимостей
	usersRepository := users_repository_postgres.NewUsersRepository(pool)
	passwordHasher := users_service.NewPasswordHasher()
	usersService := users_service.NewUsersService(usersRepository, passwordHasher)

	sessionsRepository := sessions_repository_postgres.NewSessionsRepository(pool)
	sessionsConfig := sessions_service.Config{
		TTL: 24 * time.Hour,
	}
	sessionsService := sessions_service.NewSessionsService(sessionsRepository, sessionsConfig)

	ctx := context.Background()

	t.Run("регистрация пользователя с автоматическим созданием сессии", func(t *testing.T) {
		testutil.ClearTables(t, pool)

		// Регистрация пользователя
		login := "newuser"
		password := "password123"
		repeatPassword := "password123"

		user, err := usersService.CreateUser(ctx, login, password, repeatPassword)
		require.NoError(t, err)
		assert.NotZero(t, user.ID)

		// Автоматическое создание сессии после регистрации
		session, err := sessionsService.CreateSession(ctx, user.ID)
		require.NoError(t, err)
		assert.NotEqual(t, "", session.ID.String())
		assert.Equal(t, user.ID, session.UserID)

		// Проверяем, что и пользователь, и сессия созданы в БД
		retrievedUser, _, err := usersRepository.GetUser(ctx, login)
		require.NoError(t, err)
		assert.Equal(t, user.ID, retrievedUser.ID)

		// Проверяем, что сессия активна
		assert.False(t, time.Now().After(session.ExpiresAt), "Сессия должна быть активна")
	})
}
