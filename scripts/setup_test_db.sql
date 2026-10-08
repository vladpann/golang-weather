-- Создание тестовой базы данных
-- Запустите этот скрипт для подготовки окружения для интеграционных тестов

-- ВАЖНО: Этот скрипт должен быть запущен от имени суперпользователя (postgres)
-- или владельца базы данных

-- Подключитесь к тестовой базе данных:
-- \c weather_db_test

-- Выдаем права на создание схем (если запущено от суперпользователя)
-- GRANT CREATE ON DATABASE weather_db_test TO weather_user;

-- Создаем схему weather
CREATE SCHEMA IF NOT EXISTS weather AUTHORIZATION weather_user;

-- Создаем таблицу пользователей
CREATE TABLE IF NOT EXISTS weather.users (
    id BIGSERIAL PRIMARY KEY,
    login VARCHAR(255) NOT NULL UNIQUE,
    password BYTEA NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Создаем индекс для быстрого поиска по логину
CREATE INDEX IF NOT EXISTS idx_users_login ON weather.users(login);

-- Создаем таблицу сессий
CREATE TABLE IF NOT EXISTS weather.sessions (
    id UUID PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES weather.users(id) ON DELETE CASCADE,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Создаем индекс для быстрого поиска сессий по user_id
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON weather.sessions(user_id);

-- Создаем индекс для очистки истекших сессий
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON weather.sessions(expires_at);

-- Комментарии к таблицам
COMMENT ON TABLE weather.users IS 'Таблица пользователей приложения';
COMMENT ON TABLE weather.sessions IS 'Таблица активных сессий пользователей';

-- Выдаем права пользователю (замените weather_user на вашего пользователя)
GRANT USAGE ON SCHEMA weather TO weather_user;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA weather TO weather_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA weather TO weather_user;

-- Устанавливаем права по умолчанию для новых объектов
ALTER DEFAULT PRIVILEGES IN SCHEMA weather GRANT ALL PRIVILEGES ON TABLES TO weather_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA weather GRANT ALL PRIVILEGES ON SEQUENCES TO weather_user;

-- Вывод информации о созданных таблицах
\dt weather.*
