#!/bin/bash

# Скрипт для проверки состояния тестовой базы данных

set -e

DB_NAME=${POSTGRES_DB:-weather_db_test}
DB_USER=${POSTGRES_USER:-weather_user}

echo "🔍 Проверка тестовой базы данных: $DB_NAME"
echo ""

# Цвета
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Функция для выполнения SQL запроса
run_query() {
    local query=$1
    psql -U "$DB_USER" -d "$DB_NAME" -t -A -c "$query"
}

# Проверка подключения
echo -e "${BLUE}Проверка подключения...${NC}"
if psql -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Подключение успешно${NC}"
else
    echo -e "${RED}✗ Не удалось подключиться к БД${NC}"
    exit 1
fi
echo ""

# Список таблиц
echo -e "${BLUE}Таблицы в схеме weather:${NC}"
psql -U "$DB_USER" -d "$DB_NAME" -c "\dt weather.*"
echo ""

# Количество пользователей
echo -e "${BLUE}Количество пользователей:${NC}"
user_count=$(run_query "SELECT COUNT(*) FROM weather.users;")
echo "  $user_count пользователей"
echo ""

# Количество сессий
echo -e "${BLUE}Количество сессий:${NC}"
session_count=$(run_query "SELECT COUNT(*) FROM weather.sessions;")
echo "  $session_count сессий"
echo ""

# Активные сессии
echo -e "${BLUE}Активные сессии (не истекшие):${NC}"
active_sessions=$(run_query "SELECT COUNT(*) FROM weather.sessions WHERE expires_at > NOW();")
echo "  $active_sessions активных сессий"
echo ""

# Истекшие сессии
echo -e "${BLUE}Истекшие сессии:${NC}"
expired_sessions=$(run_query "SELECT COUNT(*) FROM weather.sessions WHERE expires_at <= NOW();")
echo "  $expired_sessions истекших сессий"
echo ""

# Список пользователей (если есть)
if [ "$user_count" -gt 0 ]; then
    echo -e "${BLUE}Список пользователей:${NC}"
    psql -U "$DB_USER" -d "$DB_NAME" -c "SELECT id, login, created_at FROM weather.users ORDER BY created_at DESC LIMIT 10;"
    echo ""
fi

# Список сессий (если есть)
if [ "$session_count" -gt 0 ]; then
    echo -e "${BLUE}Последние сессии:${NC}"
    psql -U "$DB_USER" -d "$DB_NAME" -c "SELECT id, user_id, expires_at, created_at FROM weather.sessions ORDER BY created_at DESC LIMIT 10;"
    echo ""
fi

echo -e "${GREEN}✅ Проверка завершена${NC}"
