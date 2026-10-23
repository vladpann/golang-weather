package sessions_service_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vladpann/backend/golang-weather/internal/core/domain"
	sessions_service "github.com/vladpann/backend/golang-weather/internal/features/sessions/service"
)

// mockSessionsRepository - мок репозитория для юнит-тестов
type mockSessionsRepository struct {
	createSessionFunc func(ctx context.Context, session domain.Session) (domain.Session, error)
	getSessionFunc    func(ctx context.Context, sessionID uuid.UUID) (domain.Session, error)
}

func (m *mockSessionsRepository) CreateSession(ctx context.Context, session domain.Session) (domain.Session, error) {
	if m.createSessionFunc != nil {
		return m.createSessionFunc(ctx, session)
	}
	return session, nil
}

func (m *mockSessionsRepository) GetSession(ctx context.Context, sessionID uuid.UUID) (domain.Session, error) {
	if m.getSessionFunc != nil {
		return m.getSessionFunc(ctx, sessionID)
	}
	return domain.Session{}, nil
}

// TestNewConfig проверяет создание конфигурации из переменных окружения
func TestNewConfig(t *testing.T) {
	t.Run("успешное создание конфигурации с дефолтными значениями", func(t *testing.T) {
		// Очищаем переменные окружения
		os.Unsetenv("SESSION_TTL")

		config, err := sessions_service.NewConfig()

		require.NoError(t, err)
		assert.Equal(t, 24*time.Hour, config.TTL, "TTL должен быть 24 часа по умолчанию")
	})

	t.Run("успешное создание конфигурации с кастомным TTL", func(t *testing.T) {
		// Устанавливаем переменную окружения
		os.Setenv("SESSION_TTL", "2h")
		defer os.Unsetenv("SESSION_TTL")

		config, err := sessions_service.NewConfig()

		require.NoError(t, err)
		assert.Equal(t, 2*time.Hour, config.TTL, "TTL должен быть 2 часа")
	})

	t.Run("ошибка при невалидном TTL", func(t *testing.T) {
		// Устанавливаем невалидное значение
		os.Setenv("SESSION_TTL", "invalid")
		defer os.Unsetenv("SESSION_TTL")

		_, err := sessions_service.NewConfig()

		assert.Error(t, err)
	})
}

// TestNewConfigMust проверяет создание конфигурации с паникой при ошибке
func TestNewConfigMust(t *testing.T) {
	t.Run("успешное создание конфигурации", func(t *testing.T) {
		os.Unsetenv("SESSION_TTL")

		config := sessions_service.NewConfigMust()

		assert.Equal(t, 24*time.Hour, config.TTL)
	})

	t.Run("паника при невалидной конфигурации", func(t *testing.T) {
		os.Setenv("SESSION_TTL", "invalid")
		defer os.Unsetenv("SESSION_TTL")

		assert.Panics(t, func() {
			sessions_service.NewConfigMust()
		})
	})
}

// TestCreateSession_Unit юнит-тест для CreateSession
func TestCreateSession_Unit(t *testing.T) {
	ctx := context.Background()

	t.Run("успешное создание сессии", func(t *testing.T) {
		userID := int64(123)
		expectedSession := domain.NewSession(
			uuid.New(),
			userID,
			time.Now().Add(1*time.Hour),
		)

		mockRepo := &mockSessionsRepository{
			createSessionFunc: func(ctx context.Context, session domain.Session) (domain.Session, error) {
				return expectedSession, nil
			},
		}

		config := sessions_service.Config{TTL: 1 * time.Hour}
		service := sessions_service.NewSessionsService(mockRepo, config)

		session, err := service.CreateSession(ctx, userID)

		require.NoError(t, err)
		assert.Equal(t, expectedSession.ID, session.ID)
		assert.Equal(t, userID, session.UserID)
	})

	t.Run("ошибка при сохранении сессии в репозитории", func(t *testing.T) {
		userID := int64(456)
		expectedError := errors.New("database error")

		mockRepo := &mockSessionsRepository{
			createSessionFunc: func(ctx context.Context, session domain.Session) (domain.Session, error) {
				return domain.Session{}, expectedError
			},
		}

		config := sessions_service.Config{TTL: 1 * time.Hour}
		service := sessions_service.NewSessionsService(mockRepo, config)

		_, err := service.CreateSession(ctx, userID)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "create session")
	})

	t.Run("сессия создается с правильным TTL", func(t *testing.T) {
		userID := int64(789)
		customTTL := 3 * time.Hour

		mockRepo := &mockSessionsRepository{
			createSessionFunc: func(ctx context.Context, session domain.Session) (domain.Session, error) {
				return session, nil
			},
		}

		config := sessions_service.Config{TTL: customTTL}
		service := sessions_service.NewSessionsService(mockRepo, config)

		beforeCreate := time.Now()
		session, err := service.CreateSession(ctx, userID)
		afterCreate := time.Now()

		require.NoError(t, err)

		// Проверяем, что ExpiresAt находится в правильном диапазоне
		expectedMinExpiry := beforeCreate.Add(customTTL)
		expectedMaxExpiry := afterCreate.Add(customTTL)

		assert.True(t,
			session.ExpiresAt.After(expectedMinExpiry) || session.ExpiresAt.Equal(expectedMinExpiry),
			"ExpiresAt должен быть больше или равен beforeCreate + TTL",
		)
		assert.True(t,
			session.ExpiresAt.Before(expectedMaxExpiry) || session.ExpiresAt.Equal(expectedMaxExpiry),
			"ExpiresAt должен быть меньше или равен afterCreate + TTL",
		)
	})
}

// TestGetSession_Unit юнит-тест для GetSession
func TestGetSession_Unit(t *testing.T) {
	ctx := context.Background()

	t.Run("успешное получение сессии", func(t *testing.T) {
		sessionID := uuid.New()
		expectedSession := domain.NewSession(
			sessionID,
			123,
			time.Now().Add(1*time.Hour),
		)

		mockRepo := &mockSessionsRepository{
			getSessionFunc: func(ctx context.Context, id uuid.UUID) (domain.Session, error) {
				assert.Equal(t, sessionID, id)
				return expectedSession, nil
			},
		}

		config := sessions_service.Config{TTL: 1 * time.Hour}
		service := sessions_service.NewSessionsService(mockRepo, config)

		session, err := service.GetSession(ctx, sessionID)

		require.NoError(t, err)
		assert.Equal(t, expectedSession.ID, session.ID)
		assert.Equal(t, expectedSession.UserID, session.UserID)
		assert.Equal(t, expectedSession.ExpiresAt.Unix(), session.ExpiresAt.Unix())
	})

	t.Run("ошибка при получении сессии из репозитория", func(t *testing.T) {
		sessionID := uuid.New()
		expectedError := errors.New("session not found")

		mockRepo := &mockSessionsRepository{
			getSessionFunc: func(ctx context.Context, id uuid.UUID) (domain.Session, error) {
				return domain.Session{}, expectedError
			},
		}

		config := sessions_service.Config{TTL: 1 * time.Hour}
		service := sessions_service.NewSessionsService(mockRepo, config)

		_, err := service.GetSession(ctx, sessionID)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "get session from repository")
	})

	t.Run("получение несуществующей сессии", func(t *testing.T) {
		nonExistentID := uuid.New()

		mockRepo := &mockSessionsRepository{
			getSessionFunc: func(ctx context.Context, id uuid.UUID) (domain.Session, error) {
				return domain.Session{}, errors.New("not found")
			},
		}

		config := sessions_service.Config{TTL: 1 * time.Hour}
		service := sessions_service.NewSessionsService(mockRepo, config)

		_, err := service.GetSession(ctx, nonExistentID)

		assert.Error(t, err)
	})
}

// TestNewSessionsService проверяет конструктор сервиса
func TestNewSessionsService(t *testing.T) {
	t.Run("создание сервиса с валидными параметрами", func(t *testing.T) {
		mockRepo := &mockSessionsRepository{}
		config := sessions_service.Config{TTL: 1 * time.Hour}

		service := sessions_service.NewSessionsService(mockRepo, config)

		assert.NotNil(t, service)
	})
}
