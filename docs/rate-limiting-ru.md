# Документация: Ограничение частоты запросов (Rate Limiting)

## Оглавление

1. [Обзор](#обзор)
2. [Архитектура и принципы работы](#архитектура-и-принципы-работы)
3. [Конфигурация](#конфигурация)
4. [Использование](#использование)
5. [Обработка ошибок](#обработка-ошибок)
6. [Тестирование](#тестирование)
7. [Производительность и масштабирование](#производительность-и-масштабирование)
8. [Примеры использования](#примеры-использования)

---

## Обзор

Система ограничения частоты запросов (Rate Limiting) в приложении golang-weather представляет собой **кастомную реализацию** алгоритма Token Bucket (Корзина токенов), разработанную специально для контроля частоты обращений к внешнему API OpenWeather.

### Основные характеристики

- **Алгоритм**: Token Bucket с пропорциональным пополнением токенов
- **Реализация**: Полностью custom, без использования внешних библиотек
- **Область применения**: Защита эндпоинтов OpenWeather API
- **Хранилище**: In-memory (только в оперативной памяти)
- **Режим работы**: Очередь запросов с graceful degradation

### Зачем нужен Rate Limiting?

1. **Соблюдение лимитов внешнего API**: OpenWeather API имеет ограничения на количество запросов в минуту
2. **Защита от злоупотреблений**: Предотвращение DoS-атак и случайных перегрузок
3. **Контроль затрат**: Минимизация риска превышения платных квот API
4. **Graceful degradation**: Плавная деградация сервиса при высокой нагрузке вместо отказа в обслуживании

---

## Архитектура и принципы работы

### 1. Алгоритм Token Bucket

Rate Limiter использует классический алгоритм "Корзина токенов":

```
┌─────────────────────────────────────────┐
│         Token Bucket                     │
│  ┌────────────────────────────────────┐ │
│  │  Токены: ████████░░░░░░░░░░░       │ │
│  │  Доступно: 8 / 60                   │ │
│  └────────────────────────────────────┘ │
│                                          │
│  Параметры:                              │
│  • Максимум токенов: 60                  │
│  • Окно времени: 1 минута                │
│  • Режим пополнения: пропорциональный    │
└─────────────────────────────────────────┘
```

#### Основные принципы:

1. **Инициализация**: При запуске корзина полностью заполнена (все токены доступны)
2. **Потребление**: Каждый запрос забирает 1 токен
3. **Пополнение**: Токены пополняются двумя способами:
   - **Полное пополнение**: Если прошла полная минута, корзина полностью заполняется
   - **Пропорциональное пополнение**: Токены добавляются пропорционально прошедшему времени

#### Математика пополнения:

```go
// Если прошла полная минута
if elapsed >= 1 minute {
    tokens = RequestsPerMinute
}

// Иначе пропорциональное пополнение
tokensToAdd = (RequestsPerMinute × elapsed_seconds) / 60
tokens = min(tokens + tokensToAdd, RequestsPerMinute)
```

**Пример**: При лимите 60 запросов/минуту:

- После 10 секунд добавится 10 токенов
- После 30 секунд добавится 30 токенов
- После 60 секунд корзина будет полностью заполнена

### 2. Система очередей

Когда токены недоступны, запросы не отклоняются сразу, а помещаются в очередь:

```
┌─────────────────────────────────────────────────────────┐
│                    Rate Limiter Flow                     │
└─────────────────────────────────────────────────────────┘

  Входящий запрос
       │
       ▼
  ┌─────────────────┐
  │ Токен доступен? │
  └─────────────────┘
       │
       ├─── Да ──────────────────────────┐
       │                                  ▼
       │                         ┌──────────────────┐
       │                         │ Забрать токен    │
       │                         │ Пропустить запрос│
       │                         └──────────────────┘
       │
       └─── Нет ────────────────────────┐
                                         ▼
                                ┌──────────────────┐
                                │ Есть место       │
                                │ в очереди?       │
                                └──────────────────┘
                                         │
                    ┌────────────────────┴──────────────────┐
                    │                                        │
                   Да                                       Нет
                    │                                        │
                    ▼                                        ▼
           ┌─────────────────┐                    ┌──────────────────┐
           │ Добавить в       │                    │ Вернуть ошибку   │
           │ очередь          │                    │ ErrQueueFull     │
           │ Ждать токена     │                    │ (HTTP 503)       │
           └─────────────────┘                    └──────────────────┘
                    │
                    ▼
           ┌─────────────────┐
           │ Background worker│
           │ обрабатывает     │
           │ очередь каждые   │
           │ 100ms            │
           └─────────────────┘
```

#### Параметры очереди:

- **QueueSize**: Максимальное количество запросов в очереди (по умолчанию: 100)
- **Время обработки**: Фоновая горутина проверяет очередь каждые 100 миллисекунд
- **Context awareness**: Запросы с истекшим контекстом пропускаются, токен возвращается

### 3. Структура компонентов

```
┌────────────────────────────────────────────────────────┐
│                   HTTP Handler Layer                    │
│  • OpenWeatherHTTPHandler                              │
│  • Endpoints: SearchLocation, GetOpenWeather           │
└────────────────────────────────────────────────────────┘
                          │
                          ▼
┌────────────────────────────────────────────────────────┐
│                  Rate Limiter Core                      │
│  • RateLimiter struct                                  │
│  • Allow(ctx) method                                   │
│  • Token management                                    │
└────────────────────────────────────────────────────────┘
                          │
          ┌───────────────┴───────────────┐
          ▼                               ▼
┌──────────────────────┐      ┌──────────────────────┐
│  Token Bucket        │      │  Request Queue       │
│  • tokens: int       │      │  • chan requests     │
│  • lastRefill: time  │      │  • queueSize: int    │
│  • refillTokens()    │      │  • processQueue()    │
└──────────────────────┘      └──────────────────────┘
```

---

## Конфигурация

### 1. Переменные окружения

Rate Limiter настраивается через переменные окружения с префиксом `OPENWEATHER_`:

```bash
# Максимальное количество запросов в минуту
OPENWEATHER_REQUESTS_PER_MINUTE=60

# Размер очереди ожидания
OPENWEATHER_QUEUE_SIZE=100

# Также требуются для работы с OpenWeather API
OPENWEATHER_API_BASE_URL=https://api.openweathermap.org
OPENWEATHER_API_KEY=your_api_key_here
```

### 2. Конфигурационная структура

```go
type Config struct {
    APIBaseUrl        string `envconfig:"API_BASE_URL" required:"true"`
    APIKey            string `envconfig:"API_KEY" required:"true"`
    RequestsPerMinute int    `envconfig:"REQUESTS_PER_MINUTE" default:"60"`
    QueueSize         int    `envconfig:"QUEUE_SIZE" default:"100"`
}
```

### 3. Правила валидации

При создании Rate Limiter выполняются следующие проверки:

```go
// Валидация конфигурации
func (c Config) Validate() error {
    if c.RequestsPerMinute <= 0 {
        return errors.New("requests per minute must be greater than 0")
    }
    if c.QueueSize < 0 {
        return errors.New("queue size must be greater than or equal to 0")
    }
    return nil
}
```

### 4. Примеры конфигураций

#### Строгий режим (минимальная очередь):

```bash
OPENWEATHER_REQUESTS_PER_MINUTE=60
OPENWEATHER_QUEUE_SIZE=10
```

#### Стандартный режим:

```bash
OPENWEATHER_REQUESTS_PER_MINUTE=60
OPENWEATHER_QUEUE_SIZE=100
```

#### Агрессивный режим (большая очередь):

```bash
OPENWEATHER_REQUESTS_PER_MINUTE=100
OPENWEATHER_QUEUE_SIZE=500
```

---

## Использование

### 1. Инициализация в приложении

Rate Limiter создается один раз при запуске приложения:

```go
// cmd/weather/main.go

// 1. Загрузить конфигурацию
config := openweather_service.NewConfigMust()

// 2. Создать Rate Limiter
rateLimiter, err := ratelimiter.NewRateLimiter(
    ratelimiter.Config{
        RequestsPerMinute: config.RequestsPerMinute,
        QueueSize:         config.QueueSize,
    },
    zapLogger,
)
if err != nil {
    log.Fatal(err)
}

// 3. Зарегистрировать cleanup при завершении
defer rateLimiter.Close()

// 4. Передать в HTTP handler
openWeatherHandler := openweather_transport_http.NewOpenWeatherHTTPHandler(
    locationsService,
    config,
    authMiddleware,
    rateLimiter, // <- инъекция зависимости
)
```

### 2. Использование в HTTP handlers

Rate Limiter применяется **перед каждым запросом** к внешнему API:

```go
func (h *OpenWeatherHTTPHandler) GetOpenWeather(rw http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    log := core_logger.FromContext(ctx)
    responseHandler := core_http_response.NewHTTPResponseHandler(log, rw)

    // 1. Декодировать запрос
    var request GetOpenWeatherRequest
    if err := core_http_request.DecodeAndValidateRequest(r, &request); err != nil {
        responseHandler.ErrorResponse(err, "failed to decode request")
        return
    }

    // 2. ⚠️ КРИТИЧНО: Проверить rate limit ПЕРЕД вызовом внешнего API
    if err := h.rateLimiter.Allow(ctx); err != nil {
        log.Warn("rate limit check failed", zap.Error(err))
        h.handleRateLimitError(responseHandler, err)
        return
    }

    // 3. Теперь безопасно делать запрос к внешнему API
    resp, err := httpClient.Do(req)
    // ... обработка ответа
}
```

### 3. Защищенные эндпоинты

В настоящий момент Rate Limiting применяется к двум эндпоинтам:

| Эндпоинт                               | Метод | Описание                                 | Файл                    |
| -------------------------------------- | ----- | ---------------------------------------- | ----------------------- |
| `/api/v1/open-weather/search-location` | GET   | Поиск локаций по названию города         | `search_location.go:38` |
| `/api/v1/open-weather/get-weather`     | POST  | Получение данных о погоде по координатам | `get_openweather.go:36` |

---

## Обработка ошибок

### 1. Типы ошибок

Rate Limiter может вернуть следующие ошибки:

```go
// Собственная ошибка rate limiter
var ErrQueueFull = errors.New("rate limit queue is full")

// Стандартные ошибки контекста
context.Canceled           // Клиент отменил запрос
context.DeadlineExceeded   // Истек таймаут ожидания в очереди
```

### 2. HTTP статусы и ответы

Каждая ошибка преобразуется в соответствующий HTTP ответ:

#### **ErrQueueFull → HTTP 503 Service Unavailable**

```json
{
  "error": "Service temporarily unavailable",
  "message": "Rate limit queue is full (max queue size: 100). Please try again in a few seconds.",
  "rate_limit": {
    "requests_per_minute": 60,
    "retry_after_seconds": 5
  }
}
```

**Когда происходит**:

- Очередь полностью заполнена
- Нет свободных токенов
- Новый запрос не может быть добавлен в очередь

**Рекомендация клиенту**: Повторить запрос через 5 секунд

---

#### **context.DeadlineExceeded → HTTP 408 Request Timeout**

```json
{
  "error": "Request timeout",
  "message": "Request timed out while waiting for rate limit. Please try again.",
  "rate_limit": {
    "requests_per_minute": 60,
    "retry_after_seconds": 5
  }
}
```

**Когда происходит**:

- Запрос находился в очереди слишком долго
- Истек таймаут контекста запроса
- Токены не освободились вовремя

**Рекомендация клиенту**: Повторить запрос с бóльшим таймаутом или позже

---

#### **context.Canceled → HTTP 408 Request Timeout**

```json
{
  "error": "Request canceled",
  "message": "Request was canceled"
}
```

**Когда происходит**:

- Клиент отменил запрос (закрыл соединение)
- Приложение завершает работу
- Контекст был явно отменен

**Рекомендация**: Нет, запрос был отменен намеренно

---

#### **Другие ошибки → HTTP 429 Too Many Requests**

```json
{
  "error": "Rate limit exceeded",
  "message": "Too many requests. Rate limit: 60 requests per minute. Please try again in a few seconds.",
  "rate_limit": {
    "requests_per_minute": 60,
    "retry_after_seconds": 5
  }
}
```

**Когда происходит**:

- Любая другая ошибка rate limiter'а
- Неизвестный сценарий

---

### 3. Логирование

Rate Limiter использует структурированное логирование (zap):

```go
// Уровень DEBUG - операции с токенами
log.Debug("token acquired", zap.Int("remaining_tokens", 59))
log.Debug("token acquired immediately")
log.Debug("tokens refilled", zap.Int("old_tokens", 0), zap.Int("new_tokens", 10))

// Уровень INFO - жизненный цикл
log.Info("rate limiter initialized",
    zap.Int("requests_per_minute", 60),
    zap.Int("queue_size", 100))

// Уровень WARN - проблемы
log.Warn("rate limit queue is full")
log.Warn("request timed out while waiting in queue")
log.Warn("queued request context expired, skipping")
```

---

## Тестирование

### 1. Покрытие тестами

Система имеет комплексное тестовое покрытие:

| Компонент             | Тесты              | Описание                                       |
| --------------------- | ------------------ | ---------------------------------------------- |
| **Rate Limiter Core** | 12 unit-тестов     | Проверка алгоритма, очередей, обработки ошибок |
| **HTTP Integration**  | 1 integration-тест | Проверка интеграции с HTTP handlers            |
| **Всего**             | 13 тестов          | Полное покрытие функциональности               |

### 2. Unit-тесты Rate Limiter

#### Основные тесты:

```go
// 1. Немедленное разрешение запроса
TestRateLimiter_AllowImmediate()

// 2. Множественные запросы в рамках лимита
TestRateLimiter_MultipleRequestsWithinLimit()

// 3. Превышение лимита + очередь
TestRateLimiter_ExceedLimit_Queued()

// 4. Переполнение очереди
TestRateLimiter_QueueFull()

// 5. Отмена контекста
TestRateLimiter_ContextCanceled()

// 6. Таймаут контекста
TestRateLimiter_ContextTimeout()

// 7. Пополнение токенов
TestRateLimiter_TokenRefill()

// 8. Конкурентные запросы
TestRateLimiter_ConcurrentRequests()

// 9. Закрытие rate limiter
TestRateLimiter_Close()

// 10. Валидация конфигурации
TestRateLimiter_ConfigValidation()

// 11. Обработка очереди
TestRateLimiter_QueueProcessing()
```

### 3. Запуск тестов

```bash
# Все тесты rate limiter
go test ./internal/features/openweather/ratelimiter/ -v

# Конкретный тест
go test ./internal/features/openweather/ratelimiter/ -v -run TestRateLimiter_QueueFull

# С покрытием
go test ./internal/features/openweather/ratelimiter/ -cover

# HTTP integration тесты
go test ./internal/features/openweather/transport/http/ -v -run TestGetOpenWeather_RateLimitRejected
```

### 4. Пример теста

```go
func TestRateLimiter_QueueFull(t *testing.T) {
    requestsPerMin := 5
    queueSize := 2
    rl := newTestRateLimiter(t, requestsPerMin, queueSize)

    ctx := context.Background()

    // Использовать все токены
    for i := 0; i < requestsPerMin; i++ {
        err := rl.Allow(ctx)
        require.NoError(t, err)
    }

    // Заполнить очередь
    var wg sync.WaitGroup
    longCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    for i := 0; i < queueSize; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            _ = rl.Allow(longCtx)
        }()
    }

    time.Sleep(100 * time.Millisecond)

    // Следующий запрос должен вернуть ErrQueueFull
    err := rl.Allow(ctx)
    assert.ErrorIs(t, err, ratelimiter.ErrQueueFull)

    cancel()
    wg.Wait()
}
```

---

## Производительность и масштабирование

### 1. Характеристики производительности

| Параметр                            | Значение   | Комментарий                              |
| ----------------------------------- | ---------- | ---------------------------------------- |
| **Overhead на запрос**              | ~10-50 мкс | Минимальная задержка при наличии токенов |
| **Память на limiter**               | ~1 KB      | Фиксированный размер структуры           |
| **Горутины**                        | 1          | Один background worker на весь limiter   |
| **Concurrent safety**               | ✅ Да      | Использует sync.Mutex                    |
| **Периодичность обработки очереди** | 100 ms     | Фоновая горутина                         |

### 2. Ограничения

#### In-Memory хранение:

✅ **Преимущества**:

- Очень высокая скорость
- Нет сетевых задержек
- Простота реализации

❌ **Недостатки**:

- Лимиты НЕ общие между инстансами приложения
- Сброс счетчиков при перезапуске
- Не подходит для горизонтального масштабирования

#### Масштабирование:

```
┌──────────────────────────────────────────────────────┐
│         Сценарий: 1 инстанс приложения               │
│                                                       │
│  Лимит: 60 запросов/минуту                           │
│  Результат: ✅ Работает корректно                     │
└──────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────┐
│         Сценарий: 3 инстанса приложения              │
│                                                       │
│  Лимит на инстанс: 60 запросов/минуту                │
│  Общий лимит: 180 запросов/минуту ⚠️                 │
│  Результат: ❌ Каждый инстанс считает независимо      │
└──────────────────────────────────────────────────────┘
```

**Решение для кластера**: Использовать Redis-based rate limiting (например, библиотека `go-redis/redis_rate`)

### 3. Конкурентность

Rate Limiter полностью thread-safe:

```go
type RateLimiter struct {
    mu sync.Mutex // Защита от одновременного доступа
    // ... другие поля
}

func (rl *RateLimiter) tryAcquireToken() bool {
    rl.mu.Lock()         // Блокировка
    defer rl.mu.Unlock() // Разблокировка

    rl.refillTokens()
    if rl.tokens > 0 {
        rl.tokens--
        return true
    }
    return false
}
```

### 4. Оптимизация для высоких нагрузок

```go
// Фоновая обработка очереди (каждые 100ms)
func (rl *RateLimiter) processQueue() {
    ticker := time.NewTicker(100 * time.Millisecond)
    defer ticker.Stop()

    for {
        select {
        case <-rl.stopChan:
            return
        case <-ticker.C:
            rl.processQueuedRequests()
        }
    }
}
```

**Компромисс**:

- Частая проверка (100ms) = меньшая задержка для запросов в очереди
- Редкая проверка (1s) = меньшая нагрузка на CPU

---

## Примеры использования

### 1. Базовый пример клиента (cURL)

```bash
# Успешный запрос
curl -X POST http://localhost:8080/api/v1/open-weather/get-weather \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "latitude": 55.7522,
    "longitude": 37.6156
  }'

# Ответ при превышении лимита (429)
{
  "error": "Rate limit exceeded",
  "message": "Too many requests. Rate limit: 60 requests per minute. Please try again in a few seconds.",
  "rate_limit": {
    "requests_per_minute": 60,
    "retry_after_seconds": 5
  }
}
```

### 2. Пример клиента с повторными попытками (Go)

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

func getWeatherWithRetry(lat, lon float64, maxRetries int) error {
    request := map[string]float64{
        "latitude":  lat,
        "longitude": lon,
    }

    body, _ := json.Marshal(request)

    for attempt := 0; attempt < maxRetries; attempt++ {
        resp, err := http.Post(
            "http://localhost:8080/api/v1/open-weather/get-weather",
            "application/json",
            bytes.NewReader(body),
        )
        if err != nil {
            return err
        }
        defer resp.Body.Close()

        switch resp.StatusCode {
        case http.StatusOK:
            // Успех!
            return nil

        case http.StatusTooManyRequests, http.StatusServiceUnavailable:
            // Rate limit, повторить через 5 секунд
            fmt.Printf("Rate limited (attempt %d/%d), retrying in 5s...\n",
                attempt+1, maxRetries)
            time.Sleep(5 * time.Second)
            continue

        default:
            // Другая ошибка
            return fmt.Errorf("unexpected status: %d", resp.StatusCode)
        }
    }

    return fmt.Errorf("max retries exceeded")
}
```

### 3. Пример клиента (JavaScript/TypeScript)

```typescript
async function getWeather(
  latitude: number,
  longitude: number,
): Promise<WeatherData> {
  const maxRetries = 3;
  const retryDelay = 5000; // 5 секунд

  for (let attempt = 0; attempt < maxRetries; attempt++) {
    const response = await fetch('/api/v1/open-weather/get-weather', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({ latitude, longitude }),
    });

    if (response.ok) {
      return await response.json();
    }

    if (response.status === 429 || response.status === 503) {
      // Rate limit exceeded
      const errorData = await response.json();
      const retryAfter = errorData.rate_limit?.retry_after_seconds || 5;

      console.warn(
        `Rate limited. Retrying in ${retryAfter}s (attempt ${attempt + 1}/${maxRetries})...`,
      );

      if (attempt < maxRetries - 1) {
        await new Promise((resolve) => setTimeout(resolve, retryAfter * 1000));
        continue;
      }
    }

    throw new Error(`Request failed: ${response.status}`);
  }

  throw new Error('Max retries exceeded');
}
```

### 4. Нагрузочное тестирование

```bash
# Использование Apache Bench для проверки rate limiting
ab -n 100 -c 10 -T 'application/json' -p request.json \
  http://localhost:8080/api/v1/open-weather/get-weather

# request.json
{
  "latitude": 55.7522,
  "longitude": 37.6156
}

# Ожидаемый результат:
# - Первые 60 запросов: HTTP 200
# - Следующие запросы: HTTP 429 или HTTP 503 (в зависимости от очереди)
```

---

## Заключение

### Основные преимущества реализации

✅ **Простота**: Минимальные зависимости, понятный код  
✅ **Производительность**: In-memory, низкие задержки  
✅ **Graceful degradation**: Очередь вместо немедленного отказа  
✅ **Context-aware**: Корректная обработка отмены и таймаутов  
✅ **Наблюдаемость**: Подробное логирование всех операций

### Возможные улучшения

1. **Distributed rate limiting**: Использовать Redis для общих лимитов между инстансами
2. **Metrics**: Добавить метрики Prometheus (счетчики отклоненных запросов, размер очереди)
3. **Adaptive limits**: Динамическая подстройка лимитов на основе нагрузки
4. **Per-user limits**: Индивидуальные лимиты для разных пользователей
5. **Rate limit headers**: Добавить заголовки `X-RateLimit-*` в ответы

### Дальнейшее чтение

- [Token Bucket Algorithm](https://en.wikipedia.org/wiki/Token_bucket)
- [Rate Limiting Strategies](https://cloud.google.com/architecture/rate-limiting-strategies-techniques)
- [OpenWeather API Limits](https://openweathermap.org/price)
- [Демонстрация](rate-limit-test.html)

---
