# Погода

Реализация проекта "Погода" из [Golang Роадмап Сергея Жукова](https://zhukovsd.github.io/golang-backend-learning-course/projects/weather-viewer/)

Представляет собой REST API приложение для получения информации о погоде с веб-интерфейсом.

## Демонстрация

Проект доступен по ссылке:

http://77.223.99.64

Username
```
Demo
```

Password
```
Demo123
```

## Архитектура

- **Backend**: Go (REST API)
- **Frontend**: Vanilla JS (SPA)
- **Web Server**: Nginx (reverse proxy + static files)
- **Database**: PostgreSQL
- **Migrations**: golang-migrate
- **Weather Provider**: OpenWeather API

## API Endpoints

### Users

- `POST /api/v1/users` - создать пользователя
- `POST /api/v1/users/login` - авторизация пользователя

### Sessions

- `GET /api/v1/sessions` - получить текущую сессию
- `DELETE /api/v1/sessions` - удалить сессию (logout)

### Locations

- `GET /api/v1/locations` - получить сохраненные локации пользователя
- `POST /api/v1/locations` - добавить локацию
- `DELETE /api/v1/locations/{id}` - удалить локацию

### OpenWeather

- `GET /api/v1/open-weather/search-location?city_name={city}` - поиск локаций
- `GET /api/v1/open-weather/get-weather?latitude={lat}&longitude={lon}` - получить погоду по координатам

## Структура проекта

```
.
├── cmd/weather/          # Точка входа приложения
├── internal/
│   ├── core/             # Общие компоненты (config, logger, transport, repository, etc.)
│   │   ├── config/       # Конфигурация приложения
│   │   ├── domain/       # Доменные модели
│   │   ├── errors/       # Общие ошибки
│   │   ├── logger/       # Логирование (zap)
│   │   ├── repository/   # Работа с БД (pgx pool)
│   │   └── transport/    # HTTP транспорт (middleware, request, response, server)
│   └── features/         # Бизнес-логика
│       ├── users/        # Управление пользователями
│       ├── sessions/     # Управление сессиями
│       ├── locations/    # Управление локациями
│       └── openweather/  # Интеграция с OpenWeather API
├── migrations/           # SQL миграции
├── public/               # Статические файлы (frontend)
├── docs/                 # Документация
├── nginx.conf            # Конфигурация Nginx
└── docker-compose.yml    # Docker Compose конфигурация
```

## Быстрый старт

1. Скопируйте `.env.example` в `.env`:

```bash
cp .env.example .env
```

2. Добавьте свой API ключ OpenWeather в `.env`:

```bash
OPENWEATHER_API_KEY=your_openweather_api_key_here
```

3. Запустите PostgreSQL:

```bash
make env-up
```

4. Пробросить порты для локального подключения

```bash
make env-port-forward
```

5. Примените миграции:

```bash
make migrate-up
```

6. Запустите приложение с Nginx:

```bash
make weather-deploy
```

6. Откройте в браузере: http://localhost

## Доступные команды

### Окружение

- `make env-up` - запустить PostgreSQL (dev + test)
- `make env-down` - остановить PostgreSQL (dev + test)
- `make env-cleanup` - удалить данные PostgreSQL
- `make env-port-forward` - пробросить порты БД на localhost (dev 5432, test 5433)
- `make env-port-close` - закрыть проброс портов

### Миграции

- `make migrate-up` - применить все миграции к dev базе
- `make migrate-down` - откатить последнюю миграцию dev базы
- `make migrate-create seq=<name>` - создать новую миграцию
- `make migrate-up-test` - применить миграции к test базе
- `make migrate-down-test` - откатить миграции test базы

### Приложение

- `make weather-deploy` - деплой с локально собранным бинарником
- `make weather-undeploy` - остановить приложение
- `make weather-run` - запустить backend локально (для разработки)
- `make build-local` - собрать бинарник локально для Linux

### Тесты

- `make test` - запустить все тесты
- `make test-all` - запустить все тесты с покрытием
- `make test-integration` - запустить только интеграционные тесты
- `make test-users` - запустить тесты сервиса пользователей
- `make test-sessions` - запустить тесты сервиса сессий
- `make test-openweather` - запустить тесты OpenWeather HTTP транспорта

### Утилиты

- `make ps` - показать статус контейнеров
- `make logs-cleanup` - очистить логи

## Разработка

### Работа с миграциями

Создать новую миграцию:

```bash
make migrate-create seq=add_new_table
```

Применить миграции:

```bash
make migrate-up
```

Откатить последнюю миграцию:

```bash
make migrate-down
```

### Запуск тестов

Запустить все тесты:

```bash
make test
```

Запустить тесты с отчетом о покрытии:

```bash
make test-all
```

Отчет будет сохранен в `coverage.html`.

## Особенности

- **Аутентификация**: Сессии на основе cookies с bcrypt хешированием паролей
- **Rate Limiting**: Ограничение запросов к OpenWeather API (60 req/min по умолчанию)
- **ON DELETE CASCADE**: При удалении пользователя автоматически удаляются все его сессии и локации
- **CORS**: Настроен для работы с frontend
- **Логирование**: Структурированные логи с помощью zap
- **Graceful Shutdown**: Корректное завершение работы сервера
- **Middleware**: Request ID, Logger, Trace, Panic Recovery, CORS, Auth
- **Валидация**: Валидация входных данных с помощью go-playground/validator
- **Connection Pooling**: Эффективное управление подключениями к БД с помощью pgx/v5

## Конфигурация

Основные параметры конфигурации (см. `.env.example`):

- `HTTP_ADDR` - адрес HTTP сервера (по умолчанию :5050)
- `POSTGRES_HOST`, `POSTGRES_PORT`, `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB` - настройки подключения к PostgreSQL
- `SESSION_TTL` - время жизни сессии (по умолчанию 24h)
- `OPENWEATHER_API_KEY` - API ключ OpenWeather
- `OPENWEATHER_REQUESTS_PER_MINUTE` - лимит запросов к OpenWeather API (по умолчанию 60)
- `LOGGER_LEVEL` - уровень логирования (DEBUG, INFO, WARN, ERROR)
- `ALLOWED_ORIGINS` - разрешенные origins для CORS

## Технологии

- **Go 1.26.1**
- **Chi Router** - легкий и быстрый HTTP роутер
- **pgx/v5** - PostgreSQL драйвер и connection pool
- **Zap** - быстрое структурированное логирование
- **golang-migrate** - управление миграциями БД
- **bcrypt** - хеширование паролей
- **UUID** - генерация уникальных идентификаторов
- **validator** - валидация структур

## Получение API ключа OpenWeather

1. Зарегистрируйтесь на [OpenWeather](https://openweathermap.org/)
2. Перейдите в раздел [API keys](https://home.openweathermap.org/api_keys)
3. Создайте новый API ключ
4. Добавьте его в `.env` файл

## Лицензия

MIT
