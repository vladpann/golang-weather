package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	core_config "github.com/vladpann/backend/golang-weather/internal/core/config"
	core_health "github.com/vladpann/backend/golang-weather/internal/core/health"
	core_logger "github.com/vladpann/backend/golang-weather/internal/core/logger"
	core_metrics "github.com/vladpann/backend/golang-weather/internal/core/metrics"
	core_pgx_pool "github.com/vladpann/backend/golang-weather/internal/core/repository/postgres/pool/pgx"
	core_http_middleware "github.com/vladpann/backend/golang-weather/internal/core/transport/http/middleware"
	core_http_server "github.com/vladpann/backend/golang-weather/internal/core/transport/http/server"
	locations_repository_postgres "github.com/vladpann/backend/golang-weather/internal/features/locations/repository/postgres"
	locations_service "github.com/vladpann/backend/golang-weather/internal/features/locations/service"
	locations_transport_http "github.com/vladpann/backend/golang-weather/internal/features/locations/transport/http"
	openweather_client "github.com/vladpann/backend/golang-weather/internal/features/openweather/client"
	openweather_ratelimiter "github.com/vladpann/backend/golang-weather/internal/features/openweather/ratelimiter"
	openweather_transport_http "github.com/vladpann/backend/golang-weather/internal/features/openweather/transport/http"
	sessions_repository_postgres "github.com/vladpann/backend/golang-weather/internal/features/sessions/repository/postgres"
	sessions_service "github.com/vladpann/backend/golang-weather/internal/features/sessions/service"
	sessions_worker "github.com/vladpann/backend/golang-weather/internal/features/sessions/worker"
	users_repository_postgres "github.com/vladpann/backend/golang-weather/internal/features/users/repository/postgres"
	users_service "github.com/vladpann/backend/golang-weather/internal/features/users/service"
	users_transport_http "github.com/vladpann/backend/golang-weather/internal/features/users/transport/http"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

func main() {
	cfg := core_config.NewConfigMust()
	time.Local = cfg.TimeZone

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT, syscall.SIGTERM,
	)
	defer cancel()

	logger, err := core_logger.NewLogger(core_logger.NewConfigMust())
	if err != nil {
		fmt.Println("failed to init application logger:", err)
		os.Exit(1)
	}
	defer logger.Close()

	logger.Debug("application time zone", zap.Any("zone", time.Local))

	logger.Debug("initializing metrics")
	metricsRegistry := prometheus.NewRegistry()
	metrics := core_metrics.NewMetrics(metricsRegistry)

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
	usersTransportHTTPConfig := users_transport_http.NewConfigMust()

	logger.Debug("initializing feature", zap.String("feature", "sessions"))
	sessionsRepository := sessions_repository_postgres.NewSessionsRepository(pool)
	sessionsConfig := sessions_service.NewConfigMust()
	sessionsService := sessions_service.NewSessionsService(sessionsConfig, sessionsRepository)
	sessionsWorkerConfig := sessions_worker.NewConfigMust()
	sessionsWorkerCleaner := sessions_worker.NewSessionsCleaner(sessionsWorkerConfig, sessionsService, logger)

	authMiddleware := core_http_middleware.Auth(sessionsService)
	usersTransportHTTP := users_transport_http.NewUsersHTTPHandler(
		usersTransportHTTPConfig,
		usersService,
		sessionsService,
		authMiddleware,
	)

	logger.Debug("initializing feature", zap.String("feature", "locations"))
	locationsRepository := locations_repository_postgres.NewLocationsRepository(pool)
	locationsService := locations_service.NewLocationsService(locationsRepository)
	locationsTransportHTTP := locations_transport_http.NewLocationsHTTPHandler(locationsService, authMiddleware)

	logger.Debug("initializing feature", zap.String("feature", "openweather"))
	openWeatherClientConfig := openweather_client.NewConfigMust()

	openWeatherClient, err := openweather_client.NewClient(openWeatherClientConfig, logger, metrics)
	if err != nil {
		logger.Fatal("failed to init open weather client", zap.Error(err))
	}
	defer openWeatherClient.Close()

	openWeatherRateLimiterConfig := openweather_ratelimiter.Config{
		RequestsPerMinute: openWeatherClientConfig.RequestsPerMinute,
		QueueSize:         openWeatherClientConfig.QueueSize,
	}
	openWeatherRateLimiter, err := openweather_ratelimiter.NewRateLimiter(
		openWeatherRateLimiterConfig,
		logger,
		metrics,
	)
	if err != nil {
		logger.Fatal("failed to init open weather rate limiter", zap.Error(err))
	}
	defer openWeatherRateLimiter.Close()

	openWeatherTransportHTTP := openweather_transport_http.NewOpenWeatherHTTPHandler(
		openWeatherClient,
		authMiddleware,
		openWeatherRateLimiter,
	)

	logger.Debug("initializing HTTP server")
	httpConfig := core_http_server.NewConfigMust()
	httpServer := core_http_server.NewHTTPServer(
		httpConfig,
		logger,
		// Load shedding MUST be first to reject overload before any processing
		core_http_middleware.LoadShed(
			core_http_middleware.LoadShedConfig{
				MaxConcurrentRequests: httpConfig.MaxConcurrentRequests,
			},
			logger,
		),
		// CORS and basic setup
		core_http_middleware.CORS(httpConfig.AllowedOrigins),
		core_http_middleware.RequestID(),
		// Request deadline MUST be before any I/O operations
		core_http_middleware.RequestDeadline(httpConfig.RequestDeadline),
		// MaxBodySize MUST be before Logger to prevent unbounded reads
		core_http_middleware.MaxBodySize(httpConfig.MaxBodyBytes),
		// Logger reads body (now size-limited)
		core_http_middleware.Logger(logger),
		// Metrics middleware - records HTTP request duration and status
		core_http_middleware.Metrics(metrics),
		// Tracing and recovery
		core_http_middleware.Trace(),
		core_http_middleware.Panic(),
	)

	healthChecker := core_health.New(pool.Ping, 2*time.Second, func(err error) {
		logger.Warn("readiness check failed", zap.Error(err))
	})
	context.AfterFunc(ctx, healthChecker.SetShuttingDown)
	httpServer.RegisterHealth(healthChecker.Healthz, healthChecker.Readyz)

	// Register Prometheus metrics endpoint
	httpServer.RegisterMetrics(promhttp.HandlerFor(metricsRegistry, promhttp.HandlerOpts{
		Registry: metricsRegistry,
	}))

	apiVersionRouter := core_http_server.NewAPIVersionRouter(core_http_server.ApiVersion1)
	apiVersionRouter.RegisterRoutes(usersTransportHTTP.Routes()...)
	apiVersionRouter.RegisterRoutes(locationsTransportHTTP.Routes()...)
	apiVersionRouter.RegisterRoutes(openWeatherTransportHTTP.Routes()...)

	httpServer.RegisterAPIRouters(
		apiVersionRouter,
	)

	egroup, ectx := errgroup.WithContext(ctx)

	egroup.Go(func() error { return httpServer.Run(ectx) })
	egroup.Go(func() error { return sessionsWorkerCleaner.Run(ectx) })

	if err := egroup.Wait(); err != nil {
		logger.Error("app stopped with error", zap.Error(err))
		os.Exit(1)
	}
}
