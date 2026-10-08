#!/bin/bash

# Полный скрипт настройки тестовой БД
# Автоматически определяет, нужны ли права суперпользователя

set -e

DB_NAME=${POSTGRES_DB:-weather_db_test}
DB_USER=${POSTGRES_USER:-weather_user}
SUPERUSER=${POSTGRES_SUPERUSER:-$USER}  # Используем текущего пользователя по умолчанию

echo "🗄️  Настройка тестовой базы данных: $DB_NAME"
echo ""

# Цвета
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Проверка существования БД
if ! psql -U "$SUPERUSER" -lqt | cut -d \| -f 1 | grep -qw "$DB_NAME"; then
    echo -e "${YELLOW}База данных $DB_NAME не найдена. Создаем...${NC}"
    createdb -U "$SUPERUSER" "$DB_NAME"
    echo -e "${GREEN}✓ База данных создана${NC}"
else
    echo -e "${GREEN}✓ База данных $DB_NAME существует${NC}"
fi

# Настройка прав (от суперпользователя)
echo ""
echo -e "${YELLOW}Настройка прав доступа...${NC}"
psql -U "$SUPERUSER" -d "$DB_NAME" << EOF
-- Даем права на создание объектов
GRANT CREATE ON DATABASE $DB_NAME TO $DB_USER;

-- Создаем схему weather
CREATE SCHEMA IF NOT EXISTS weather AUTHORIZATION $DB_USER;

-- Даем все права на схему
GRANT ALL ON SCHEMA weather TO $DB_USER;

-- Права по умолчанию
ALTER DEFAULT PRIVILEGES IN SCHEMA weather 
    GRANT ALL PRIVILEGES ON TABLES TO $DB_USER;

ALTER DEFAULT PRIVILEGES IN SCHEMA weather 
    GRANT ALL PRIVILEGES ON SEQUENCES TO $DB_USER;
EOF

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Права настроены${NC}"
else
    echo -e "${RED}✗ Ошибка настройки прав${NC}"
    exit 1
fi

# Создание таблиц (от имени weather_user)
echo ""
echo -e "${YELLOW}Создание таблиц...${NC}"
psql -U "$DB_USER" -d "$DB_NAME" -f scripts/setup_test_db.sql

if [ $? -eq 0 ]; then
    echo ""
    echo -e "${GREEN}✅ Тестовая база данных успешно настроена!${NC}"
    echo ""
    echo "Запустите тесты:"
    echo "  make test-integration"
else
    echo -e "${RED}✗ Ошибка создания таблиц${NC}"
    exit 1
fi
