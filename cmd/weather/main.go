package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	core_config "github.com/vladpann/backend/golang-weather/internal/core/config"
	core_logger "github.com/vladpann/backend/golang-weather/internal/core/logger"
	core_pgx_pool "github.com/vladpann/backend/golang-weather/internal/core/repository/postgres/pool/pgx"
	core_http_middleware "github.com/vladpann/backend/golang-weather/internal/core/transport/http/middleware"
	core_http_server "github.com/vladpann/backend/golang-weather/internal/core/transport/http/server"
	locations_repository_postgres "github.com/vladpann/backend/golang-weather/internal/features/locations/repository/postgres"
	locations_service "github.com/vladpann/backend/golang-weather/internal/features/locations/service"
	locations_transport_http "github.com/vladpann/backend/golang-weather/internal/features/locations/transport/http"
	"github.com/vladpann/backend/golang-weather/internal/features/openweather/ratelimiter"
	openweather_service "github.com/vladpann/backend/golang-weather/internal/features/openweather/service"
	openweather_transport_http "github.com/vladpann/backend/golang-weather/internal/features/openweather/transport/http"
	sessions_repository_postgres "github.com/vladpann/backend/golang-weather/internal/features/sessions/repository/postgres"
	sessions_service "github.com/vladpann/backend/golang-weather/internal/features/sessions/service"
	users_repository_postgres "github.com/vladpann/backend/golang-weather/internal/features/users/repository/postgres"
	users_service "github.com/vladpann/backend/golang-weather/internal/features/users/service"
	users_transport_http "github.com/vladpann/backend/golang-weather/internal/features/users/transport/http"
	"go.uber.org/zap"
)

func main() {
	cfg := core_config.NewConfigMust()
	time.Local = cfg.TimeZone

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT, syscall.SIGTERM,
	)
	defer cancel()

	if err := godotenv.Load(); err != nil {
		fmt.Println("failed to init env variables:", err)
	}

	logger, err := core_logger.NewLogger(core_logger.NewConfigMust())
	if err != nil {
		fmt.Println("failed to init application logger:", err)
		os.Exit(1)
	}
	defer logger.Close()

	logger.Debug("application time zone", zap.Any("zone", time.Local))

	logger.Debug("initializing postgres connection pool")
	pool, err := core_pgx_pool.NewPool(
		ctx,
		core_pgx_pool.NewConfigMust(),
	)
	if err != nil {
		logger.Fatal("failed to init postgres connection pool", zap.Error(err))
	}
	defer pool.Close()

	logger.Debug("initializing feature", zap.String("feature", "users"))
	usersRepository := users_repository_postgres.NewUsersRepository(pool)
	passwordHasher := users_service.NewPasswordHasher()
	usersService := users_service.NewUsersService(usersRepository, passwordHasher)

	logger.Debug("initializing feature", zap.String("feature", "sessions"))
	sessionsRepository := sessions_repository_postgres.NewSessionsRepository(pool)
	sessionsConfig := sessions_service.NewConfigMust()
	sessionsService := sessions_service.NewSessionsService(sessionsRepository, sessionsConfig)

	authMiddleware := core_http_middleware.Auth(sessionsService)
	usersTransportHTTP := users_transport_http.NewUsersHTTPHandler(usersService, sessionsService, authMiddleware)

	logger.Debug("initializing feature", zap.String("feature", "locations"))
	locationsRepository := locations_repository_postgres.NewLocationsRepository(pool)
	locationsService := locations_service.NewLocationsService(locationsRepository)
	locationsTransportHTTP := locations_transport_http.NewLocationsHTTPHandler(locationsService, authMiddleware)

	logger.Debug("initializing feature", zap.String("feature", "openweather"))
	openWeatherConfig := openweather_service.NewConfigMust()

	// Initialize rate limiter
	rateLimiterConfig := ratelimiter.Config{
		RequestsPerMinute: openWeatherConfig.RequestsPerMinute,
		QueueSize:         openWeatherConfig.QueueSize,
	}
	rateLimiter, err := ratelimiter.NewRateLimiter(rateLimiterConfig, logger.Logger)
	if err != nil {
		logger.Fatal("failed to init rate limiter", zap.Error(err))
	}
	defer rateLimiter.Close()

	openWeatherTransportHTTP := openweather_transport_http.NewOpenWeatherHTTPHandler(
		locationsService,
		openWeatherConfig,
		authMiddleware,
		rateLimiter,
	)

	logger.Debug("initializing HTTP server")
	httpConfig := core_http_server.NewConfigMust()
	httpServer := core_http_server.NewHTTPServer(
		httpConfig,
		logger,
		core_http_middleware.CORS(httpConfig.AllowedOrigins),
		core_http_middleware.RequestID(),
		core_http_middleware.Logger(logger),
		core_http_middleware.Trace(),
		core_http_middleware.Panic(),
	)

	apiVersionRouter := core_http_server.NewAPIVersionRouter(core_http_server.ApiVersion1)
	apiVersionRouter.RegisterRoutes(usersTransportHTTP.Routes()...)
	apiVersionRouter.RegisterRoutes(locationsTransportHTTP.Routes()...)
	apiVersionRouter.RegisterRoutes(openWeatherTransportHTTP.Routes()...)

	httpServer.RegisterAPIRouters(
		apiVersionRouter,
	)

	if err := httpServer.Run(ctx); err != nil {
		logger.Error("HTTP server run error", zap.Error(err))
	}
}
