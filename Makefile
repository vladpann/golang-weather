include .env
export

export PROJECT_ROOT=$(shell pwd)

.PHONY: help env-up env-down env-cleanup env-port-forward env-port-close \
	migrate-create migrate-up migrate-down migrate-action \
	migrate-up-test migrate-down-test migrate-action-test \
	logs-cleanup weather-run weather-deploy weather-undeploy ps \
	test test-all test-integration test-users test-sessions test-openweather

help: ## Показать справку
	@grep -hE '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ": *## *"}; {printf "\033[36m%-22s\033[0m %s\n", $$1, $$2}'

# ============================================
# Приложение
# ============================================

build-local: ## Собрать бинарник локально для Linux
	@echo "Building binary for Linux..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
		-ldflags="-w -s" \
		-o ${PROJECT_ROOT}/exe \
		${PROJECT_ROOT}/cmd/weather/main.go
	@echo "Binary built: ${PROJECT_ROOT}/exe"

weather-run: ## Запустить приложение локально
	@export LOGGER_FOLDER=${PROJECT_ROOT}/out/logs && \
	export POSTGRES_HOST=localhost && \
	go mod tidy && \
	go run ${PROJECT_ROOT}/cmd/weather/main.go

weather-deploy: ## Собрать и запустить приложение + nginx в Docker
	@docker compose up -d --build weather nginx

weather-deploy-fast: ## Деплой с локально собранным бинарником (быстрее)
	@echo "Building locally..."
	@$(MAKE) build-local
	@echo "Deploying to Docker..."
	@docker compose -f docker-compose.yml build --build-arg USE_LOCAL_BINARY=1 weather
	@docker compose up -d weather nginx
	@echo "Deployment complete!"

weather-undeploy: ## Остановить приложение + nginx
	@docker compose down weather nginx

logs-cleanup: ## Удалить файлы логов (ОСТОРОЖНО!)
	@read -p "Delete all log volume files? Risk of data loss. [y/N]: " ans; \
	if [ "$$ans" = "y" ]; then \
		rm -rf ${PROJECT_ROOT}/out/logs && \
		echo "Log files have been cleared"; \
	else \
		echo "Log clean-up cancelled"; \
	fi;

ps: ## Показать статус контейнеров
	@docker compose ps

# ============================================
# Окружение (PostgreSQL dev + test)
# ============================================

env-up: ## Запустить PostgreSQL (dev + test)
	@docker compose up -d weather-postgres weather-postgres-test

env-down: ## Остановить PostgreSQL (dev + test)
	@docker compose down weather-postgres weather-postgres-test

env-cleanup: ## Удалить данные PostgreSQL (ОСТОРОЖНО!)
	@read -p "Delete all environment volume files? Risk of data loss. [y/N]: " ans; \
	if [ "$$ans" = "y" ]; then \
		docker compose down weather-postgres weather-postgres-test port-forwarder port-forwarder-test && \
		rm -rf ${PROJECT_ROOT}/out/pgdata ${PROJECT_ROOT}/out/pgdata-test && \
		echo "Environment files have been cleared"; \
	else \
		echo "Environment clean-up cancelled"; \
	fi;

env-port-forward: ## Пробросить порты БД на localhost (dev 5432, test 5433)
	@docker compose up -d port-forwarder port-forwarder-test

env-port-close: ## Закрыть проброшенные порты БД
	@docker compose down port-forwarder port-forwarder-test

# ============================================
# Миграции
# ============================================

migrate-create: ## Создать миграцию (make migrate-create seq=init)
	@if [ -z "$(seq)" ]; then \
		echo "The required \"seq\" parameter is missing. Example: make migrate-create seq=init"; \
		exit 1; \
	fi; \
	docker compose run --rm weather-postgres-migrate \
		create \
		-ext sql \
		-dir /migrations \
		-seq "$(seq)"

migrate-up: ## Применить миграции к dev базе
	@make migrate-action action=up

migrate-down: ## Откатить миграции dev базы
	@make migrate-action action=down

migrate-action: ## Выполнить действие миграции на dev базе (make migrate-action action=up)
	@if [ -z "$(action)" ]; then \
		echo "The required \"action\" parameter is missing. Example: make migrate-action action=up"; \
		exit 1; \
	fi; \
	docker compose run --rm weather-postgres-migrate \
		-path /migrations \
		-database "postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@weather-postgres:5432/${POSTGRES_DB}?sslmode=disable" \
		"$(action)"

migrate-up-test: ## Применить миграции к test базе
	@make migrate-action-test action=up

migrate-down-test: ## Откатить миграции test базы
	@make migrate-action-test action=down

migrate-action-test: ## Выполнить действие миграции на test базе
	@if [ -z "$(action)" ]; then \
		echo "The required \"action\" parameter is missing. Example: make migrate-action-test action=up"; \
		exit 1; \
	fi; \
	docker compose run --rm weather-postgres-migrate \
		-path /migrations \
		-database "postgres://${POSTGRES_USER_TEST}:${POSTGRES_PASSWORD_TEST}@weather-postgres-test:5432/${POSTGRES_DB_TEST}?sslmode=disable" \
		"$(action)"

# ============================================
# Тесты
# ============================================

test: ## Запустить все тесты
	@go test ./... -v

test-all: ## Запустить все тесты с покрытием
	@go test ./... -v -cover -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@echo "📊 Отчет о покрытии сохранен в coverage.html"

test-integration: ## Запустить только интеграционные тесты
	@go test ./internal/features/users/service -v -count=1
	@go test ./internal/features/sessions/service -v -count=1

test-users: ## Запустить тесты сервиса пользователей
	@go test ./internal/features/users/service -v -count=1

test-sessions: ## Запустить тесты сервиса сессий
	@go test ./internal/features/sessions/service -v -count=1

test-openweather: ## Запустить тесты OpenWeather HTTP транспорта
	@go test ./internal/features/openweather/transport/http -v -count=1