# Загрузка переменных из .env файла
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

.PHONY: run help test test-integration test-users test-sessions test-all setup-test-db

help: ## Показать справку
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

run: ## Запустить проект
	@echo "🚀 Запуск проекта..."
	@go run ./cmd/weather/main.go
	@echo "✅ Проект запущен"

setup-db: ## Настроить production базу данных
	@echo "🗄️ Настройка production базы данных..."
	@echo "Создание пользователя и базы данных..."
	@psql postgres -c "CREATE USER $(POSTGRES_USER) WITH PASSWORD '$(POSTGRES_PASSWORD)';" || echo "Пользователь уже существует"
	@psql postgres -c "CREATE DATABASE $(POSTGRES_DB) OWNER $(POSTGRES_USER);" || echo "База данных уже существует"
	@echo "✅ База данных создана"
	@echo "Применение миграций..."
	@export DATABASE_URL='postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable' && \
		migrate -path migrations -database "$$DATABASE_URL" up
	@echo "✅ Production база данных настроена"

migrate-up: ## Применить миграции
	@echo "⬆️  Применение миграций..."
	@export DATABASE_URL='postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable' && \
		migrate -path migrations -database "$$DATABASE_URL" up
	@echo "✅ Миграции применены"

migrate-down: ## Откатить миграции
	@echo "⬇️  Откат миграций..."
	@export DATABASE_URL='postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable' && \
		migrate -path migrations -database "$$DATABASE_URL" down
	@echo "✅ Миграции откачены"

migrate-force: ## Принудительно установить версию миграции (использовать: make migrate-force VERSION=1)
	@echo "🔧 Установка версии миграции: $(VERSION)..."
	@export DATABASE_URL='postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable' && \
		migrate -path migrations -database "$$DATABASE_URL" force $(VERSION)
	@echo "✅ Версия миграции установлена"

drop-db: ## Удалить production базу данных (ОСТОРОЖНО!)
	@echo "⚠️  ВНИМАНИЕ: Удаление production базы данных!"
	@read -p "Вы уверены? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		dropdb $(POSTGRES_DB) || echo "База данных не найдена"; \
		psql postgres -c "DROP USER IF EXISTS $(POSTGRES_USER);"; \
		echo "✅ Production база данных и пользователь удалены"; \
	else \
		echo "❌ Отменено"; \
	fi

test: ## Запустить все тесты
	@go test ./... -v

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

test-all: ## Запустить все тесты с покрытием
	@go test ./... -v -cover -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@echo "📊 Отчет о покрытии сохранен в coverage.html"

setup-test-db: ## Настроить тестовую базу данных
	@echo "🗄️ Настройка тестовой базы данных..."
	@./scripts/setup_test_db_full.sh
	@echo "✅ Тестовая база данных настроена"

clean-test-db: ## Очистить тестовую базу данных
	@echo "🧹 Очистка тестовой базы данных..."
	@psql -d weather_db_test -c "TRUNCATE TABLE weather.sessions CASCADE"
	@psql -d weather_db_test -c "TRUNCATE TABLE weather.users CASCADE"
	@echo "✅ Тестовая база данных очищена"

drop-test-db: ## Удалить тестовую базу данных
	@echo "⚠️  Удаление тестовой базы данных..."
	@dropdb weather_db_test || echo "База данных не найдена"
	@echo "✅ Тестовая база данных удалена"


