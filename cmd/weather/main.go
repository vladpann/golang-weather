package main

import (
	"context"
	"fmt"
	"log"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	core_pgx_pool "github.com/vladpann/golang-weather/internal/core/repository/postgres/pool/pgx"
	core_http_server "github.com/vladpann/golang-weather/internal/core/transport/http/server"
	users_repository_postgres "github.com/vladpann/golang-weather/internal/features/users/repository/postgres"
	users_service "github.com/vladpann/golang-weather/internal/features/users/service"
	users_transport_http "github.com/vladpann/golang-weather/internal/features/users/transport/http"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal(err)
	}

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT, syscall.SIGTERM,
	)
	defer cancel()

	pool, err := core_pgx_pool.NewPool(
		ctx,
		core_pgx_pool.NewConfigMust(),
	)
	if err != nil {
		fmt.Println("failed to init postgres connection pool", err.Error())
	}
	defer pool.Close()

	usersRepository := users_repository_postgres.NewUsersRepository(pool)
	passwordHasher := users_service.NewPasswordHasher()
	usersService := users_service.NewUsersService(usersRepository, passwordHasher)
	usersTransportHTTP := users_transport_http.NewUsersHTTPHandler(usersService)

	httpConfig := core_http_server.NewConfigMust()
	httpServer := core_http_server.NewHTTPServer(
		httpConfig,
	)

	apiVersionRouter := core_http_server.NewAPIVersionRouter(core_http_server.ApiVersion1)
	apiVersionRouter.RegisterRoutes(usersTransportHTTP.Routes()...)

	httpServer.RegisterAPIRouters(
		apiVersionRouter,
	)

	if err := httpServer.Run(ctx); err != nil {
		fmt.Println("HTTP server run error", err.Error())
	}
}
