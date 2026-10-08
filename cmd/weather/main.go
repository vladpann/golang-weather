package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	core_logger "github.com/vladpann/golang-weather/internal/core/logger"
	core_pgx_pool "github.com/vladpann/golang-weather/internal/core/repository/postgres/pool/pgx"
	core_http_middleware "github.com/vladpann/golang-weather/internal/core/transport/http/middleware"
	core_http_server "github.com/vladpann/golang-weather/internal/core/transport/http/server"
	sessions_repository_postgres "github.com/vladpann/golang-weather/internal/features/sessions/repository/postgres"
	sessions_service "github.com/vladpann/golang-weather/internal/features/sessions/service"
	users_repository_postgres "github.com/vladpann/golang-weather/internal/features/users/repository/postgres"
	users_service "github.com/vladpann/golang-weather/internal/features/users/service"
	users_transport_http "github.com/vladpann/golang-weather/internal/features/users/transport/http"
	"go.uber.org/zap"
)

func main() {
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

	usersTransportHTTP := users_transport_http.NewUsersHTTPHandler(usersService, sessionsService)

	logger.Debug("initializing HTTP server")
	httpConfig := core_http_server.NewConfigMust()
	httpServer := core_http_server.NewHTTPServer(
		httpConfig,
		logger,
		core_http_middleware.RequestID(),
		core_http_middleware.HTTPLogger(logger),
	)

	apiVersionRouter := core_http_server.NewAPIVersionRouter(core_http_server.ApiVersion1)
	apiVersionRouter.RegisterRoutes(usersTransportHTTP.Routes()...)

	httpServer.RegisterAPIRouters(
		apiVersionRouter,
	)

	if err := httpServer.Run(ctx); err != nil {
		logger.Error("HTTP server run error", zap.Error(err))
	}
}
