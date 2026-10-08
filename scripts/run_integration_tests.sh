#!/bin/bash

# Скрипт для запуска интеграционных тестов
# Убедитесь, что тестовая база данных настроена перед запуском

set -e

echo "🧪 Запуск интеграционных тестов..."
echo ""

# Проверяем наличие .env.test
if [ ! -f ".env.test" ]; then
    echo "❌ Ошибка: файл .env.test не найден"
    echo "Скопируйте .env.example в .env.test и настройте параметры тестовой БД"
    exit 1
fi

# Цвета для вывода
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Тесты сервиса пользователей...${NC}"
go test ./internal/features/users/service -v -count=1

echo ""
echo -e "${YELLOW}Тесты сервиса сессий...${NC}"
go test ./internal/features/sessions/service -v -count=1

echo ""
echo -e "${GREEN}✅ Все интеграционные тесты успешно пройдены!${NC}"
