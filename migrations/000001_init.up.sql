CREATE SCHEMA weather;

CREATE TABLE weather.users (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    login         VARCHAR(255)    NOT NULL    UNIQUE,
    password      VARCHAR(255)    NOT NULL
);

CREATE TABLE weather.locations (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name          VARCHAR(255)    NOT NULL,
    user_id       BIGINT          NOT NULL REFERENCES weather.users(id) ON DELETE CASCADE,
    latitude      NUMERIC(9, 6)   NOT NULL,
    longitude     NUMERIC(9, 6)   NOT NULL
);

CREATE TABLE weather.sessions (
    id            UUID                        PRIMARY KEY,
    user_id       BIGINT          NOT NULL REFERENCES weather.users(id) ON DELETE CASCADE,
    expires_at    TIMESTAMPTZ     NOT NULL
);