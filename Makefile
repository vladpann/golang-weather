# Загрузка переменных из .env файла
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

.PHONY: run help test test-integration test-users test-sessions test-all setup-test-db docker-up docker-down docker-logs docker-clean docker-migrate dev migrate-up migrate-up-test migrate-down migrate-force setup-db drop-db clean-test-db drop-test-db

help: ## Показать справку
	@grep -hE '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ": *## *"}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ============================================
# Приложение
# ============================================
dev: ## Запустить полное окружение (Docker + миграции + приложение)
	@docker-up docker-migrate run 

run: ## Запустить приложение
	@go run ./cmd/weather/main.go

# ============================================
# Docker команды
# ============================================

docker-up: ## Запустить Docker контейнеры (PostgreSQL dev + test)
	@./scripts/docker-setup.sh

docker-down: ## Остановить Docker контейнеры
	@echo "🛑 Остановка Docker контейнеров..."
	@docker compose down
	@echo "✅ Docker контейнеры остановлены"

docker-logs: ## Показать логи Docker контейнеров
	@docker compose logs -f

docker-logs-dev: ## Показать логи dev базы данных
	@docker compose logs -f postgres

docker-logs-test: ## Показать логи test базы данных
	@docker compose logs -f postgres_test

docker-clean: ## Удалить Docker контейнеры и volumes (ОСТОРОЖНО!)
	@echo "⚠️  ВНИМАНИЕ: Удаление всех Docker контейнеров и данных!"
	@read -p "Вы уверены? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		docker compose down -v; \
		echo "✅ Docker контейнеры и volumes удалены"; \
	else \
		echo "❌ Отменено"; \
	fi

docker-restart: ## Перезапустить Docker контейнеры
	@echo "🔄 Перезапуск Docker контейнеров..."
	@docker compose down
	@docker compose up -d --build
	@echo "✅ Docker контейнеры перезапущены"

docker-ps: ## Показать статус Docker контейнеров
	@docker compose ps

# ============================================
# Миграции (Docker)
# ============================================

docker-migrate: ## Применить миграции к обеим базам данных в Docker
	@echo "⬆️  Применение миграций к dev базе..."
	@export DATABASE_URL='postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable' && \
		migrate -path migrations -database "$$DATABASE_URL" up
	@echo "✅ Миграции применены к dev базе"
	@echo "⬆️  Применение миграций к test базе..."
	@export DATABASE_URL='postgres://$(POSTGRES_USER_TEST):$(POSTGRES_PASSWORD_TEST)@localhost:$(POSTGRES_PORT_TEST)/$(POSTGRES_DB_TEST)?sslmode=disable' && \
		migrate -path migrations -database "$$DATABASE_URL" up
	@echo "✅ Миграции применены к test базе"

migrate-up: ## Применить миграции к dev базе
	@echo "⬆️  Применение миграций..."
	@export DATABASE_URL='postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable' && \
		migrate -path migrations -database "$$DATABASE_URL" up
	@echo "✅ Миграции применены"

migrate-up-test: ## Применить миграции к test базе
	@echo "⬆️  Применение миграций к test базе..."
	@export DATABASE_URL='postgres://$(POSTGRES_USER_TEST):$(POSTGRES_PASSWORD_TEST)@$(POSTGRES_HOST_TEST):$(POSTGRES_PORT_TEST)/$(POSTGRES_DB_TEST)?sslmode=disable' && \
		migrate -path migrations -database "$$DATABASE_URL" up
	@echo "✅ Миграции применены к test базе"

migrate-down: ## Откатить миграции dev базы
	@echo "⬇️  Откат миграций..."
	@export DATABASE_URL='postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable' && \
		migrate -path migrations -database "$$DATABASE_URL" down
	@echo "✅ Миграции откачены"

migrate-force: ## Принудительно установить версию миграции (использовать: make migrate-force VERSION=1)
	@echo "🔧 Установка версии миграции: $(VERSION)..."
	@export DATABASE_URL='postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable' && \
		migrate -path migrations -database "$$DATABASE_URL" force $(VERSION)
	@echo "✅ Версия миграции установлена"


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
	@echo "🧪 Запуск интеграционных тестов..."
	@go test ./internal/features/users/service -v -count=1
	@go test ./internal/features/sessions/service -v -count=1

test-users: ## Запустить тесты сервиса пользователей
	@echo "👤 Тесты сервиса пользователей..."
	@go test ./internal/features/users/service -v -count=1

test-sessions: ## Запустить тесты сервиса сессий
	@echo "🔑 Тесты сервиса сессий..."
	@go test ./internal/features/sessions/service -v -count=1

test-openweather: ## Запустить тесты OpenWeather HTTP транспорта
	@echo "🌤️  Тесты OpenWeather HTTP транспорта..."
	@go test ./internal/features/openweather/transport/http -v -count=1
