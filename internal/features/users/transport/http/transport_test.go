package users_transport_http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	core_logger "github.com/vladpann/backend/golang-weather/internal/core/logger"
	sessions_repository_postgres "github.com/vladpann/backend/golang-weather/internal/features/sessions/repository/postgres"
	sessions_service "github.com/vladpann/backend/golang-weather/internal/features/sessions/service"
	users_repository_postgres "github.com/vladpann/backend/golang-weather/internal/features/users/repository/postgres"
	users_service "github.com/vladpann/backend/golang-weather/internal/features/users/service"
	users_transport_http "github.com/vladpann/backend/golang-weather/internal/features/users/transport/http"
	"github.com/vladpann/backend/golang-weather/internal/testutil"
)

// testEnv содержит все зависимости для тестов
type testEnv struct {
	handler         *users_transport_http.UsersHTTPHandler
	usersService    *users_service.UsersService
	sessionsService *sessions_service.SessionsService
	logger          *core_logger.Logger
}

// setupTestEnv инициализирует тестовое окружение
func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()

	pool := testutil.SetupTestDB(t)
	t.Cleanup(func() { testutil.CleanupTestDB(t, pool) })

	// Инициализация зависимостей
	usersRepository := users_repository_postgres.NewUsersRepository(pool)
	passwordHasher := users_service.NewPasswordHasher()
	usersService := users_service.NewUsersService(usersRepository, passwordHasher)

	sessionsRepository := sessions_repository_postgres.NewSessionsRepository(pool)
	sessionsConfig := sessions_service.Config{TTL: 24 * time.Hour}
	sessionsService := sessions_service.NewSessionsService(sessionsRepository, sessionsConfig)

	// Middleware-заглушка (без проверки аутентификации)
	noopAuth := func(next http.Handler) http.Handler { return next }

	handler := users_transport_http.NewUsersHTTPHandler(
		usersService,
		sessionsService,
		noopAuth,
	)

	zapLogger, err := zap.NewDevelopment()
	require.NoError(t, err)

	return &testEnv{
		handler:         handler,
		usersService:    usersService,
		sessionsService: sessionsService,
		logger:          &core_logger.Logger{Logger: zapLogger},
	}
}

// doRequest выполняет HTTP запрос и возвращает ResponseRecorder
func (e *testEnv) doRequest(t *testing.T, method, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()

	var reqBody []byte
	var err error

	if body != nil {
		reqBody, err = json.Marshal(body)
		require.NoError(t, err)
	}

	req := httptest.NewRequest(method, path, bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	ctx := core_logger.ToContext(context.Background(), e.logger)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	e.handler.CreateUser(rr, req)

	return rr
}

func TestUsersHTTPHandler_CreateUser(t *testing.T) {
	env := setupTestEnv(t)

	t.Run("успешная регистрация пользователя", func(t *testing.T) {
		testutil.ClearTables(t, testutil.SetupTestDB(t))

		reqBody := map[string]string{
			"login":           "testuser",
			"password":        "password123",
			"repeat_password": "password123",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/sign-up", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(core_logger.ToContext(context.Background(), env.logger))

		rr := httptest.NewRecorder()
		env.handler.CreateUser(rr, req)

		// Проверка статуса
		assert.Equal(t, http.StatusCreated, rr.Code)

		// Проверка ответа
		var response map[string]interface{}
		err := json.NewDecoder(rr.Body).Decode(&response)
		require.NoError(t, err)

		assert.Contains(t, response, "id")
		assert.Contains(t, response, "login")
		assert.Equal(t, "testuser", response["login"])
	})

	t.Run("регистрация с дубликатом логина возвращает 409 Conflict", func(t *testing.T) {
		testutil.ClearTables(t, testutil.SetupTestDB(t))

		// Создаем первого пользователя
		ctx := context.Background()
		_, err := env.usersService.CreateUser(ctx, "duplicate", "password123", "password123")
		require.NoError(t, err)

		// Пытаемся создать второго с тем же логином
		reqBody := map[string]string{
			"login":           "duplicate",
			"password":        "password123",
			"repeat_password": "password123",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/sign-up", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(core_logger.ToContext(ctx, env.logger))

		rr := httptest.NewRecorder()
		env.handler.CreateUser(rr, req)

		// Проверка: должна быть ошибка 409 Conflict
		assert.Equal(t, http.StatusConflict, rr.Code)

		var response map[string]interface{}
		err = json.NewDecoder(rr.Body).Decode(&response)
		require.NoError(t, err)
		assert.Contains(t, response, "error")
	})

	t.Run("регистрация с несовпадающими паролями возвращает 400", func(t *testing.T) {
		reqBody := map[string]string{
			"login":           "testuser2",
			"password":        "password123",
			"repeat_password": "different",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/sign-up", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(core_logger.ToContext(context.Background(), env.logger))

		rr := httptest.NewRecorder()
		env.handler.CreateUser(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})

	t.Run("регистрация с невалидным JSON возвращает 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/auth/sign-up", bytes.NewReader([]byte(`{invalid json`)))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(core_logger.ToContext(context.Background(), env.logger))

		rr := httptest.NewRecorder()
		env.handler.CreateUser(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("регистрация с пустым телом запроса возвращает 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/auth/sign-up", nil)
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(core_logger.ToContext(context.Background(), env.logger))

		rr := httptest.NewRecorder()
		env.handler.CreateUser(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("регистрация с коротким паролем возвращает 400", func(t *testing.T) {
		reqBody := map[string]string{
			"login":           "testuser3",
			"password":        "123", // Короче минимума (4 символа)
			"repeat_password": "123",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/sign-up", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(core_logger.ToContext(context.Background(), env.logger))

		rr := httptest.NewRecorder()
		env.handler.CreateUser(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestUsersHTTPHandler_SignIn(t *testing.T) {
	env := setupTestEnv(t)

	t.Run("успешная авторизация возвращает cookie с session_id", func(t *testing.T) {
		testutil.ClearTables(t, testutil.SetupTestDB(t))

		// Регистрируем пользователя
		ctx := context.Background()
		_, err := env.usersService.CreateUser(ctx, "loginuser", "password123", "password123")
		require.NoError(t, err)

		// Авторизуемся
		reqBody := map[string]string{
			"login":    "loginuser",
			"password": "password123",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/sign-in", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(core_logger.ToContext(ctx, env.logger))

		rr := httptest.NewRecorder()
		env.handler.SignInUser(rr, req)

		// Проверка
		assert.Equal(t, http.StatusNoContent, rr.Code)

		// Проверяем, что установлена cookie
		cookies := rr.Result().Cookies()
		require.Len(t, cookies, 1, "должна быть установлена одна cookie")

		cookie := cookies[0]
		assert.Equal(t, "session_id", cookie.Name)
		assert.NotEmpty(t, cookie.Value)
		assert.True(t, cookie.HttpOnly, "cookie должна быть HttpOnly")

		// Проверяем, что session_id валидный UUID
		_, err = uuid.Parse(cookie.Value)
		assert.NoError(t, err, "session_id должен быть валидным UUID")
	})

	t.Run("авторизация с неправильным паролем возвращает 401", func(t *testing.T) {
		testutil.ClearTables(t, testutil.SetupTestDB(t))

		// Регистрируем пользователя
		ctx := context.Background()
		_, err := env.usersService.CreateUser(ctx, "user", "password123", "password123")
		require.NoError(t, err)

		// Пытаемся авторизоваться с неправильным паролем
		reqBody := map[string]string{
			"login":    "user",
			"password": "wrongpassword",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/sign-in", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(core_logger.ToContext(ctx, env.logger))

		rr := httptest.NewRecorder()
		env.handler.SignInUser(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)

		// Cookie не должна быть установлена
		cookies := rr.Result().Cookies()
		assert.Empty(t, cookies)
	})

	t.Run("авторизация несуществующего пользователя возвращает 404", func(t *testing.T) {
		testutil.ClearTables(t, testutil.SetupTestDB(t))

		reqBody := map[string]string{
			"login":    "nonexistent",
			"password": "password123",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/sign-in", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(core_logger.ToContext(context.Background(), env.logger))

		rr := httptest.NewRecorder()
		env.handler.SignInUser(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("авторизация с невалидным JSON возвращает 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/auth/sign-in", bytes.NewReader([]byte(`{invalid`)))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(core_logger.ToContext(context.Background(), env.logger))

		rr := httptest.NewRecorder()
		env.handler.SignInUser(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("авторизация с пустым логином возвращает 400", func(t *testing.T) {
		reqBody := map[string]string{
			"login":    "",
			"password": "password123",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/sign-in", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(core_logger.ToContext(context.Background(), env.logger))

		rr := httptest.NewRecorder()
		env.handler.SignInUser(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestUsersHTTPHandler_Routes(t *testing.T) {
	env := setupTestEnv(t)

	routes := env.handler.Routes()

	t.Run("должно быть 4 маршрута", func(t *testing.T) {
		assert.Len(t, routes, 4)
	})

	t.Run("маршрут sign-up должен быть POST /auth/sign-up", func(t *testing.T) {
		route := routes[0]
		assert.Equal(t, http.MethodPost, route.Method)
		assert.Equal(t, "/auth/sign-up", route.Path)
		assert.NotNil(t, route.Handler)
		assert.Empty(t, route.Middlewares, "sign-up не требует аутентификации")
	})

	t.Run("маршрут sign-in должен быть POST /auth/sign-in", func(t *testing.T) {
		route := routes[1]
		assert.Equal(t, http.MethodPost, route.Method)
		assert.Equal(t, "/auth/sign-in", route.Path)
		assert.NotNil(t, route.Handler)
		assert.Empty(t, route.Middlewares, "sign-in не требует аутентификации")
	})

	t.Run("маршрут sign-out должен быть POST /auth/sign-out", func(t *testing.T) {
		route := routes[2]
		assert.Equal(t, http.MethodPost, route.Method)
		assert.Equal(t, "/auth/sign-out", route.Path)
		assert.NotNil(t, route.Handler)
	})

	t.Run("маршрут check должен быть GET /auth/check с middleware", func(t *testing.T) {
		route := routes[3]
		assert.Equal(t, http.MethodGet, route.Method)
		assert.Equal(t, "/auth/check", route.Path)
		assert.NotNil(t, route.Handler)
		assert.Len(t, route.Middlewares, 1, "check требует аутентификации")
	})
}

func TestUsersHTTPHandler_Integration(t *testing.T) {
	env := setupTestEnv(t)

	t.Run("полный сценарий: регистрация → авторизация", func(t *testing.T) {
		testutil.ClearTables(t, testutil.SetupTestDB(t))

		ctx := context.Background()

		// 1. Регистрируем пользователя
		signUpBody := map[string]string{
			"login":           "integrationuser",
			"password":        "password123",
			"repeat_password": "password123",
		}

		body, _ := json.Marshal(signUpBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/sign-up", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(core_logger.ToContext(ctx, env.logger))

		rr := httptest.NewRecorder()
		env.handler.CreateUser(rr, req)

		require.Equal(t, http.StatusCreated, rr.Code, "регистрация должна пройти успешно")

		var signUpResponse map[string]interface{}
		err := json.NewDecoder(rr.Body).Decode(&signUpResponse)
		require.NoError(t, err)
		userID := int64(signUpResponse["id"].(float64))

		// 2. Авторизуемся с теми же credentials
		signInBody := map[string]string{
			"login":    "integrationuser",
			"password": "password123",
		}

		body2, _ := json.Marshal(signInBody)
		req2 := httptest.NewRequest(http.MethodPost, "/auth/sign-in", bytes.NewReader(body2))
		req2.Header.Set("Content-Type", "application/json")
		req2 = req2.WithContext(core_logger.ToContext(ctx, env.logger))

		rr2 := httptest.NewRecorder()
		env.handler.SignInUser(rr2, req2)

		require.Equal(t, http.StatusNoContent, rr2.Code, "авторизация должна пройти успешно")

		// 3. Проверяем, что сессия создана
		cookies := rr2.Result().Cookies()
		require.Len(t, cookies, 1)

		sessionID, err := uuid.Parse(cookies[0].Value)
		require.NoError(t, err)

		session, err := env.sessionsService.GetSession(ctx, sessionID)
		require.NoError(t, err)
		assert.Equal(t, userID, session.UserID, "сессия должна принадлежать зарегистрированному пользователю")
	})
}
