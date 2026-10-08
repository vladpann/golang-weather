-- Скрипт для настройки тестовой БД от имени суперпользователя
-- Запускать: psql -U postgres -d weather_db_test -f scripts/setup_test_db_as_superuser.sql

-- Даем права пользователю weather_user на создание объектов в БД
GRANT CREATE ON DATABASE weather_db_test TO weather_user;

-- Создаем схему weather и делаем weather_user владельцем
CREATE SCHEMA IF NOT EXISTS weather AUTHORIZATION weather_user;

-- Даем все права на схему
GRANT ALL ON SCHEMA weather TO weather_user;

-- Устанавливаем права по умолчанию для будущих объектов
ALTER DEFAULT PRIVILEGES IN SCHEMA weather 
    GRANT ALL PRIVILEGES ON TABLES TO weather_user;

ALTER DEFAULT PRIVILEGES IN SCHEMA weather 
    GRANT ALL PRIVILEGES ON SEQUENCES TO weather_user;

-- Теперь можно запустить основной скрипт от имени weather_user:
-- psql -U weather_user -d weather_db_test -f scripts/setup_test_db.sql
