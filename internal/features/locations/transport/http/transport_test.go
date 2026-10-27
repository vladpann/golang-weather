package locations_transport_http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/vladpann/backend/golang-weather/internal/core/domain"
	core_logger "github.com/vladpann/backend/golang-weather/internal/core/logger"
	core_http_middleware "github.com/vladpann/backend/golang-weather/internal/core/transport/http/middleware"
	core_http_server "github.com/vladpann/backend/golang-weather/internal/core/transport/http/server"
	locations_repository_postgres "github.com/vladpann/backend/golang-weather/internal/features/locations/repository/postgres"
	locations_service "github.com/vladpann/backend/golang-weather/internal/features/locations/service"
	locations_transport_http "github.com/vladpann/backend/golang-weather/internal/features/locations/transport/http"
	sessions_repository_postgres "github.com/vladpann/backend/golang-weather/internal/features/sessions/repository/postgres"
	sessions_service "github.com/vladpann/backend/golang-weather/internal/features/sessions/service"
	users_repository_postgres "github.com/vladpann/backend/golang-weather/internal/features/users/repository/postgres"
	users_service "github.com/vladpann/backend/golang-weather/internal/features/users/service"
	users_transport_http "github.com/vladpann/backend/golang-weather/internal/features/users/transport/http"
	"github.com/vladpann/backend/golang-weather/internal/testutil"
)

type locationsTestEnv struct {
	usersService     *users_service.UsersService
	usersHandler     *users_transport_http.UsersHTTPHandler
	locationsService *locations_service.LocationsService
	router           *core_http_server.APIVersionRouter
	logger           *core_logger.Logger
}

func setupLocationsTestEnv(t *testing.T) *locationsTestEnv {
	t.Helper()

	pool := testutil.SetupTestDB(t)
	t.Cleanup(func() { testutil.CleanupTestDB(t, pool) })

	usersRepository := users_repository_postgres.NewUsersRepository(pool)
	passwordHasher := users_service.NewPasswordHasher()
	usersService := users_service.NewUsersService(usersRepository, passwordHasher)

	sessionsRepository := sessions_repository_postgres.NewSessionsRepository(pool)
	sessionsService := sessions_service.NewSessionsService(
		sessionsRepository,
		sessions_service.Config{TTL: 24 * time.Hour},
	)

	authMiddleware := core_http_middleware.Auth(sessionsService)

	usersHandler := users_transport_http.NewUsersHTTPHandler(
		usersService,
		sessionsService,
		func(next http.Handler) http.Handler { return next },
	)

	locationsRepository := locations_repository_postgres.NewLocationsRepository(pool)
	locationsService := locations_service.NewLocationsService(locationsRepository)
	locationsHandler := locations_transport_http.NewLocationsHTTPHandler(locationsService, authMiddleware)

	zapLogger, err := zap.NewDevelopment()
	require.NoError(t, err)
	logger := &core_logger.Logger{Logger: zapLogger}

	router := core_http_server.NewAPIVersionRouter(core_http_server.ApiVersion1)
	router.RegisterRoutes(locationsHandler.Routes()...)

	return &locationsTestEnv{
		usersService:     usersService,
		usersHandler:     usersHandler,
		locationsService: locationsService,
		router:           router,
		logger:           logger,
	}
}

// registerAndSignIn создаёт пользователя и возвращает его ID и cookie session_id.
func (e *locationsTestEnv) registerAndSignIn(t *testing.T, login string) (int64, *http.Cookie) {
	t.Helper()

	ctx := context.Background()

	user, err := e.usersService.CreateUser(ctx, login, "password123", "password123")
	require.NoError(t, err)

	body, err := json.Marshal(map[string]string{"login": login, "password": "password123"})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/auth/sign-in", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(core_logger.ToContext(ctx, e.logger))

	rr := httptest.NewRecorder()
	e.usersHandler.SignInUser(rr, req)
	require.Equal(t, http.StatusNoContent, rr.Code, "авторизация %s должна пройти успешно", login)

	cookies := rr.Result().Cookies()
	require.Len(t, cookies, 1)

	return user.ID, cookies[0]
}

// createLocation создаёт локацию от имени пользователя userID.
func (e *locationsTestEnv) createLocation(t *testing.T, userID int64, name string) domain.Location {
	t.Helper()

	location, err := e.locationsService.CreateLocation(context.Background(), domain.NewLocation(
		0,
		userID,
		name,
		55.7558,
		37.6173,
	))
	require.NoError(t, err)

	return location
}

func (e *locationsTestEnv) deleteLocation(t *testing.T, id int64, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/locations/%d", id), nil)
	req = req.WithContext(core_logger.ToContext(context.Background(), e.logger))
	if cookie != nil {
		req.AddCookie(cookie)
	}

	rr := httptest.NewRecorder()
	e.router.ServeHTTP(rr, req)

	return rr
}

func TestLocationsHTTPHandler_DeleteLocation(t *testing.T) {
	env := setupLocationsTestEnv(t)

	t.Run("авторизованный пользователь не может удалить чужую локацию", func(t *testing.T) {
		testutil.ClearTables(t, testutil.SetupTestDB(t))

		ownerID, _ := env.registerAndSignIn(t, "owner")
		_, attackerCookie := env.registerAndSignIn(t, "attacker")

		location := env.createLocation(t, ownerID, "Moscow")

		// Атакующий (вошёл под своим аккаунтом) пытается удалить локацию владельца
		rr := env.deleteLocation(t, location.ID, attackerCookie)

		require.NotEqual(t, http.StatusNoContent, rr.Code,
			"уязвимость: пользователь удалил чужую локацию")
		// Репозиторий: 0 затронутых строк -> ErrNotFound -> 404 (не раскрываем существование чужой локации)
		assert.Equal(t, http.StatusNotFound, rr.Code)

		// Локация должна остаться у владельца
		_, err := env.locationsService.GetLocation(context.Background(), location.ID, ownerID)
		assert.NoError(t, err, "чужая локация не должна быть удалена")
	})

	t.Run("владелец может удалить свою локацию", func(t *testing.T) {
		testutil.ClearTables(t, testutil.SetupTestDB(t))

		ownerID, ownerCookie := env.registerAndSignIn(t, "owner2")
		location := env.createLocation(t, ownerID, "Berlin")

		rr := env.deleteLocation(t, location.ID, ownerCookie)

		assert.Equal(t, http.StatusNoContent, rr.Code)

		_, err := env.locationsService.GetLocation(context.Background(), location.ID, ownerID)
		assert.Error(t, err, "локация должна быть удалена")
	})

	t.Run("без авторизации удалить локацию нельзя", func(t *testing.T) {
		testutil.ClearTables(t, testutil.SetupTestDB(t))

		ownerID, _ := env.registerAndSignIn(t, "owner3")
		location := env.createLocation(t, ownerID, "Paris")

		rr := env.deleteLocation(t, location.ID, nil)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)

		_, err := env.locationsService.GetLocation(context.Background(), location.ID, ownerID)
		assert.NoError(t, err, "локация не должна быть удалена без авторизации")
	})

	t.Run("удаление несуществующей локации возвращает 404", func(t *testing.T) {
		testutil.ClearTables(t, testutil.SetupTestDB(t))

		_, cookie := env.registerAndSignIn(t, "owner4")

		rr := env.deleteLocation(t, 999999, cookie)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})
}
