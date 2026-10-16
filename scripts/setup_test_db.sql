-- Создание тестовой базы данных
-- Запустите этот скрипт для подготовки окружения для интеграционных тестов

-- ВАЖНО: Этот скрипт должен быть запущен от имени пользователя, который является
-- владельцем базы данных (в CI/CD это POSTGRES_USER из секретов)

-- Удаляем схему если она существует (для чистого тестирования)
DROP SCHEMA IF EXISTS weather CASCADE;

-- Создаем схему weather (владелец - текущий пользователь)
CREATE SCHEMA weather;

-- Создаем таблицу пользователей (без IF NOT EXISTS для гарантии)
CREATE TABLE weather.users (
    id BIGSERIAL PRIMARY KEY,
    login VARCHAR(255) NOT NULL UNIQUE,
    password BYTEA NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Создаем индекс для быстрого поиска по логину
CREATE INDEX idx_users_login ON weather.users(login);

-- Создаем таблицу сессий (без IF NOT EXISTS для гарантии)
CREATE TABLE weather.sessions (
    id UUID PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES weather.users(id) ON DELETE CASCADE,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Создаем индекс для быстрого поиска сессий по user_id
CREATE INDEX idx_sessions_user_id ON weather.sessions(user_id);

-- Создаем индекс для очистки истекших сессий
CREATE INDEX idx_sessions_expires_at ON weather.sessions(expires_at);

-- Комментарии к таблицам
COMMENT ON TABLE weather.users IS 'Таблица пользователей приложения';
COMMENT ON TABLE weather.sessions IS 'Таблица активных сессий пользователей';

-- Права уже установлены, так как текущий пользователь является владельцем схемы

-- Вывод информации о созданных таблицах
\dt weather.*
