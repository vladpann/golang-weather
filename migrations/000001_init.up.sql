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
    longitude     NUMERIC(9, 6)   NOT NULL,

    CONSTRAINT locations_user_id_coords_key UNIQUE (user_id, latitude, longitude),
    CONSTRAINT locations_name_not_blank     CHECK (btrim(name) <> ''),
    CONSTRAINT locations_latitude_range     CHECK (latitude  BETWEEN -90  AND 90),
    CONSTRAINT locations_longitude_range    CHECK (longitude BETWEEN -180 AND 180)
);

CREATE TABLE weather.sessions (
    id            UUID                        PRIMARY KEY,
    user_id       BIGINT          NOT NULL REFERENCES weather.users(id) ON DELETE CASCADE,
    expires_at    TIMESTAMPTZ     NOT NULL
);