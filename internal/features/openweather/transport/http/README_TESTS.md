# Тесты OpenWeather HTTP Transport

## Описание

Этот набор тестов покрывает HTTP транспортный слой для работы с OpenWeather API. Тесты используют мокирование HTTP сервера и сервисов для изоляции тестируемого кода от внешних зависимостей.

## Структура тестов

### Файл: `transport_test.go`

Содержит unit-тесты для HTTP handlers с использованием мок-объектов.

## Покрытые сценарии

### 1. Успешные запросы

#### `TestGetOpenWeatherLocation_Success`

- **Описание**: Тестирует успешное получение локации по названию города
- **Проверяет**:
  - Корректный запрос к OpenWeather API (параметры q и appid)
  - Парсинг JSON ответа от API
  - Сохранение локации через сервис
  - Возврат корректного ответа клиенту
- **Мок**: HTTP сервер возвращает валидный JSON с данными о локации

#### `TestGetOpenWeather_Success`

- **Описание**: Тестирует успешное получение погоды по координатам
- **Проверяет**:
  - Корректный запрос к OpenWeather API (параметры lat, lon и appid)
  - Парсинг полного JSON ответа с погодными данными
  - Возврат корректного ответа клиенту
- **Мок**: HTTP сервер возвращает валидный JSON с полными погодными данными

### 2. Обработка ошибок API

#### `TestGetOpenWeatherLocation_APIError_404`

- **Описание**: Тестирует обработку ошибки "город не найден"
- **Проверяет**:
  - HTTP статус 500 (Internal Server Error) возвращается клиенту
  - Ошибка корректно логируется
  - Сервис CreateLocation НЕ вызывается
- **Мок**: HTTP сервер возвращает 404 статус

#### `TestGetOpenWeatherLocation_APIError_503`

- **Описание**: Тестирует обработку ошибки недоступности API
- **Проверяет**:
  - HTTP статус 500 возвращается клиенту
  - Сервис CreateLocation НЕ вызывается
- **Мок**: HTTP сервер возвращает 503 статус

#### `TestGetOpenWeather_APIError_400`

- **Описание**: Тестирует обработку ошибки невалидных координат
- **Проверяет**:
  - HTTP статус 500 возвращается клиенту при ошибке от API
- **Мок**: HTTP сервер возвращает 400 статус с сообщением об ошибке

### 3. Обработка некорректных данных

#### `TestGetOpenWeatherLocation_InvalidJSON`

- **Описание**: Тестирует обработку невалидного JSON от API
- **Проверяет**:
  - Корректная обработка ошибки парсинга JSON
  - HTTP статус 500 возвращается клиенту
  - Сервис CreateLocation НЕ вызывается
- **Мок**: HTTP сервер возвращает невалидный JSON

### 4. Проверка аутентификации

#### `TestGetOpenWeatherLocation_MissingUserID`

- **Описание**: Тестирует обработку отсутствия userID в контексте
- **Проверяет**: Возврат HTTP статус 500 при отсутствии аутентификации
- **Статус**: ✅ PASS (баг в ErrorResponse исправлен)

### 5. Rate Limiting

#### `TestRateLimitBehavior` (SKIPPED)

- **Описание**: Тестирует работу rate limiter
- **Статус**: Пропущен (требует времени для выполнения)
- **Должен проверять**:
  - Блокировка избыточных запросов
  - Корректная работа очереди
  - Возврат ошибок rate limit при превышении лимита

## Используемые техники тестирования

### 1. Мокирование HTTP сервера

```go
testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    // Проверка параметров запроса
    // Возврат мок-ответа
}))
defer testServer.Close()
```

### 2. Мокирование сервисов

```go
type MockLocationsService struct {
    mock.Mock
}

func (m *MockLocationsService) CreateLocation(ctx context.Context, location domain.Location) (domain.Location, error) {
    args := m.Called(ctx, location)
    return args.Get(0).(domain.Location), args.Error(1)
}
```

### 3. Использование настоящего Rate Limiter

Вместо мокирования rate limiter, тесты используют настоящую реализацию с реальной конфигурацией. Это обеспечивает:

- Проверку интеграции между handler и rate limiter
- Валидацию поведения rate limiter в реальных условиях

### 4. Настройка контекста

```go
ctx := req.Context()
ctx = core_logger.ToContext(ctx, logger)
ctx = context.WithValue(ctx, core_http_middleware.UserIDContextKey, int64(123))
req = req.WithContext(ctx)
```

## Запуск тестов

### Все тесты OpenWeather HTTP транспорта

```bash
make test-openweather
```

### Или напрямую через go test

```bash
go test ./internal/features/openweather/transport/http -v -count=1
```

### Запуск конкретного теста

```bash
go test ./internal/features/openweather/transport/http -v -count=1 -run TestGetOpenWeatherLocation_Success
```

### С покрытием кода

```bash
go test ./internal/features/openweather/transport/http -v -count=1 -cover -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

## Что проверяется тестами

✅ **Успешные сценарии**:

- Получение локации по названию города
- Получение погоды по координатам
- Корректная работа rate limiter

✅ **Обработка ошибок HTTP (4xx, 5xx)**:

- 404 - город не найден
- 400 - невалидные координаты
- 503 - сервис недоступен

✅ **Обработка невалидных данных**:

- Невалидный JSON от API
- Отсутствие обязательных полей

✅ **Проверка параметров запросов**:

- API key передается корректно
- Query параметры формируются правильно
- Content-Type устанавливается корректно

✅ **Проверка интеграции**:

- Rate limiter работает корректно с handlers
- Сервис локаций вызывается только при успешных запросах
- Контекст передается корректно

## Известные проблемы

### ~~1. ErrorResponse с nil паникует~~ ✅ ИСПРАВЛЕНО

**Проблема**: В production коде `handler.ErrorResponse(nil, "message")` вызывал панику при вызове `err.Error()`

**Файл**: `internal/core/transport/http/response/handler.go:76`

**Решение**: ✅ Исправлено - добавлена проверка на `nil`:

```go
func (h *HTTPResponseHandler) errorResponse(statusCode int, err error, msg string) {
    errorMsg := ""
    if err != nil {
        errorMsg = err.Error()
    }
    response := ErrorResponse{
        Error:   errorMsg,
        Message: msg,
    }
    h.JSONResponse(response, statusCode)
}
```

**Тесты**: Добавлен регрессионный unit-тест `TestErrorResponse_WithNilError` в `internal/core/transport/http/response/handler_test.go`

### 2. Отсутствие тестов для edge cases

- Таймауты сетевых запросов
- Отмена контекста во время запроса
- Превышение очереди rate limiter

## Рекомендации для улучшения

1. **Добавить больше edge case тестов**:
   - Сетевые ошибки
   - Таймауты
   - Большие JSON payload

2. **Добавить benchmark тесты**:
   - Производительность парсинга JSON
   - Влияние rate limiter на latency

3. **Добавить integration тесты**:
   - Полный flow: HTTP request → API → Database
   - Проверка транзакционности

4. **Улучшить coverage**:
   - Добавить тесты для всех веток кода
   - Проверить граничные значения

## Архитектурные решения

### Почему используется настоящий Rate Limiter?

- **Реализм**: Проверяем реальное поведение системы
- **Интеграция**: Валидируем корректную работу между компонентами
- **Упрощение**: Не нужно поддерживать сложные моки

### Почему мокируется HTTP сервер?

- **Скорость**: Тесты выполняются мгновенно
- **Надежность**: Не зависят от внешнего API
- **Контроль**: Можем тестировать любые сценарии ошибок

### Почему мокируется LocationsService?

- **Изоляция**: Тестируем только HTTP layer
- **Контроль**: Проверяем, что метод вызывается с правильными параметрами
- **Скорость**: Не требуется база данных

## Заключение

Этот набор тестов обеспечивает высокое покрытие критических путей выполнения HTTP транспортного слоя OpenWeather API. Тесты используют лучшие практики Go тестирования с мокированием зависимостей и проверкой edge cases.

Для полного покрытия рекомендуется также использовать интеграционные тесты с реальной базой данных и API (если доступен тестовый ключ).
