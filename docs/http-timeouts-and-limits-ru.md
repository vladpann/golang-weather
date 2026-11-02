# HTTP Timeouts и Resource Limits

## Обзор

Система HTTP timeout'ов и лимитов ресурсов обеспечивает защиту от:
- Медленных клиентов (slowloris атаки)
- Исчерпания памяти (memory exhaustion)
- Неограниченного накопления запросов в очереди
- Таймаутов на уровне отдельных операций (DB, rate limiter, upstream API)

## Архитектура защиты

```
┌─────────────────────────────────────────────────────────────────┐
│                    Защитные слои                                 │
└─────────────────────────────────────────────────────────────────┘

1. HTTP Server Level (http.Server timeouts)
   ├── ReadHeaderTimeout: 10s    - Чтение заголовков
   ├── ReadTimeout: 15s           - Полное чтение запроса
   ├── WriteTimeout: 30s          - Запись ответа
   └── IdleTimeout: 120s          - Keep-alive соединения

2. Middleware Level (request lifecycle)
   ├── LoadShed                   - Отклонение при перегрузке
   ├── MaxBodySize: 1MB           - Ограничение размера тела
   ├── RequestDeadline: 25s       - Общий deadline запроса
   └── Logger                     - Безопасное логирование

3. Application Level
   ├── Rate Limiter Queue         - Использует context deadline
   ├── Database Queries           - Используют context deadline
   └── Upstream API Calls         - Используют context deadline
```

## Конфигурация

### 1. HTTP Server Timeouts

#### ReadHeaderTimeout (по умолчанию: 10s)
```bash
HTTP_READ_HEADER_TIMEOUT=10s
```

**Назначение**: Максимальное время на чтение заголовков HTTP запроса.

**Защищает от**:
- Slowloris атак (медленная отправка заголовков)
- Зависших соединений

**Поведение при превышении**: 
- Соединение закрывается
- Клиент не получает ответа

**Выбор значения**:
- 10s — разумное значение для большинства случаев
- Увеличить, если клиенты находятся в медленных сетях

---

#### ReadTimeout (по умолчанию: 15s)
```bash
HTTP_READ_TIMEOUT=15s
```

**Назначение**: Максимальное время на чтение полного запроса (заголовки + тело).

**Защищает от**:
- Медленной отправки тела запроса
- Зависания при чтении больших payload'ов

**Поведение при превышении**:
- Соединение закрывается
- Клиент может получить EOF

**Важно**: Должно быть больше ReadHeaderTimeout

---

#### WriteTimeout (по умолчанию: 30s)
```bash
HTTP_WRITE_TIMEOUT=30s
```

**Назначение**: Максимальное время на запись ответа клиенту.

**Защищает от**:
- Медленных клиентов, которые не читают ответ
- Зависших записей в медленные сети

**Поведение при превышении**:
- Соединение закрывается
- Handler может не завершиться корректно

**Важно**: Должно быть больше самого долгого ожидаемого времени обработки запроса + времени передачи ответа.

---

#### IdleTimeout (по умолчанию: 120s)
```bash
HTTP_IDLE_TIMEOUT=120s
```

**Назначение**: Максимальное время ожидания между keep-alive запросами.

**Защищает от**:
- Накопления неиспользуемых соединений
- Исчерпания файловых дескрипторов

**Поведение при превышении**:
- Keep-alive соединение закрывается
- Клиент может переиспользовать для следующего запроса

---

### 2. Request Limits

#### MaxBodyBytes (по умолчанию: 1MB)
```bash
HTTP_MAX_BODY_BYTES=1048576
```

**Назначение**: Максимальный размер тела запроса в байтах.

**Защищает от**:
- Memory exhaustion от больших payload'ов
- Неограниченного `io.ReadAll` в логгере

**Поведение при превышении**:
- HTTP 413 Request Entity Too Large
- Чтение прерывается через `http.MaxBytesReader`

**Применяется**: Middleware `MaxBodySize` вызывается **перед** Logger

```go
// Порядок middleware критичен:
httpServer := core_http_server.NewHTTPServer(
    // ...
    core_http_middleware.MaxBodySize(httpConfig.MaxBodyBytes),  // ← ПЕРЕД
    core_http_middleware.Logger(logger),                        // ← Logger
)
```

---

#### RequestDeadline (по умолчанию: 25s)
```bash
HTTP_REQUEST_DEADLINE=25s
```

**Назначение**: Общий deadline для всего запроса, охватывает:
- Ожидание в очереди rate limiter
- Запросы к базе данных
- Вызовы upstream API (OpenWeather)
- Обработку в handler'е

**Защищает от**:
- Бесконечного накопления запросов в очереди
- Зависших DB запросов
- Медленных upstream API

**Поведение при превышении**:
- `context.DeadlineExceeded` распространяется по всему стеку
- Rate limiter возвращает HTTP 408
- DB драйвер отменяет запрос
- HTTP client отменяет upstream запрос

**Выбор значения**:
```
RequestDeadline > WriteTimeout - 5s (запас на запись ответа)

Пример:
- RequestDeadline: 25s
- WriteTimeout: 30s
- Запас: 5s на запись ответа после deadline
```

---

#### MaxConcurrentRequests (по умолчанию: 500)
```bash
HTTP_MAX_CONCURRENT_REQUESTS=500
```

**Назначение**: Максимальное количество одновременно обрабатываемых запросов.

**Защищает от**:
- Неограниченного накопления горутин
- Исчерпания памяти при высокой нагрузке
- Каскадных отказов

**Поведение при превышении**:
- HTTP 503 Service Unavailable
- `Retry-After: 5` header
- Запрос отклоняется **до** любой обработки

**Middleware LoadShed**:
```go
core_http_middleware.LoadShed(
    core_http_middleware.LoadShedConfig{
        MaxConcurrentRequests: httpConfig.MaxConcurrentRequests,
    },
    logger,
)
```

**Выбор значения**:
- Зависит от доступной памяти и CPU
- Мониторить метрики под нагрузкой
- 500 — консервативное значение
- 0 — отключает load shedding (не рекомендуется)

---

## Middleware Ordering (критично!)

Порядок применения middleware влияет на безопасность:

```go
httpServer := core_http_server.NewHTTPServer(
    httpConfig,
    logger,
    // 1. LoadShed - ПЕРВЫМ отклоняет перегрузку
    core_http_middleware.LoadShed(...),
    
    // 2. CORS и RequestID - минимальная настройка
    core_http_middleware.CORS(...),
    core_http_middleware.RequestID(),
    
    // 3. RequestDeadline - устанавливает deadline ПЕРЕД I/O
    core_http_middleware.RequestDeadline(httpConfig.RequestDeadline),
    
    // 4. MaxBodySize - ограничивает чтение ПЕРЕД Logger
    core_http_middleware.MaxBodySize(httpConfig.MaxBodyBytes),
    
    // 5. Logger - теперь безопасно читает тело (уже ограничено)
    core_http_middleware.Logger(logger),
    
    // 6. Trace и Panic - обработка ошибок
    core_http_middleware.Trace(),
    core_http_middleware.Panic(),
)
```

**Почему этот порядок**:
1. LoadShed: Отклонение перегрузки до любых вычислений
2. RequestDeadline: Deadline применяется ко всем I/O операциям ниже
3. MaxBodySize: Ограничение до чтения в Logger
4. Logger: Безопасное чтение уже ограниченного тела

---

## Взаимодействие с Rate Limiter

Rate limiter использует context из RequestDeadline middleware:

```go
// В handler'е OpenWeather
func (h *OpenWeatherHTTPHandler) GetOpenWeather(rw http.ResponseWriter, r *http.Request) {
    ctx := r.Context() // ← содержит deadline из middleware
    
    // Rate limiter ждёт токена с учётом deadline
    if err := h.rateLimiter.Allow(ctx); err != nil {
        if errors.Is(err, context.DeadlineExceeded) {
            // HTTP 408 Request Timeout
            return
        }
        // ...
    }
    
    // Upstream запрос также использует ctx с deadline
    req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
    resp, err := httpClient.Do(req)
    // ...
}
```

**Сценарии**:

### 1. Токен доступен сразу
```
Request → RequestDeadline(25s) → Allow() → [token available] → Upstream(2s) → Response
Total: ~2s
```

### 2. Ожидание в очереди (success)
```
Request → RequestDeadline(25s) → Allow() → [queue 5s] → [token] → Upstream(2s) → Response
Total: ~7s (в рамках deadline)
```

### 3. Ожидание в очереди (timeout)
```
Request → RequestDeadline(25s) → Allow() → [queue 26s] → context.DeadlineExceeded → HTTP 408
                                              ↑
                                       Deadline превышен
```

---

## Рекомендуемые значения для разных сценариев

### Development (локальная разработка)
```bash
HTTP_READ_HEADER_TIMEOUT=30s
HTTP_READ_TIMEOUT=60s
HTTP_WRITE_TIMEOUT=60s
HTTP_IDLE_TIMEOUT=300s
HTTP_MAX_BODY_BYTES=10485760      # 10MB
HTTP_REQUEST_DEADLINE=55s
HTTP_MAX_CONCURRENT_REQUESTS=0     # Без ограничений
```

### Production (за Nginx reverse proxy)
```bash
HTTP_READ_HEADER_TIMEOUT=10s
HTTP_READ_TIMEOUT=15s
HTTP_WRITE_TIMEOUT=30s
HTTP_IDLE_TIMEOUT=120s
HTTP_MAX_BODY_BYTES=1048576       # 1MB
HTTP_REQUEST_DEADLINE=25s
HTTP_MAX_CONCURRENT_REQUESTS=500
```

### High Load Production
```bash
HTTP_READ_HEADER_TIMEOUT=5s
HTTP_READ_TIMEOUT=10s
HTTP_WRITE_TIMEOUT=20s
HTTP_IDLE_TIMEOUT=60s
HTTP_MAX_BODY_BYTES=524288        # 512KB
HTTP_REQUEST_DEADLINE=15s
HTTP_MAX_CONCURRENT_REQUESTS=1000
```

---

## Мониторинг и Метрики

### Логирование

Load shedding события логируются:
```json
{
  "level": "warn",
  "msg": "load shedding: rejecting request",
  "active_requests": 501,
  "max_concurrent": 500,
  "path": "/api/v1/open-weather/get-weather"
}
```

Timeout события:
```json
{
  "level": "warn",
  "msg": "request timed out while waiting in queue"
}
```

Body size limit события:
```json
{
  "level": "warn",
  "msg": "failed to read request body for logging",
  "error": "http: request body too large"
}
```

### Рекомендуемые метрики (Prometheus)

```go
// Добавить в будущем:
http_requests_rejected_total{reason="load_shed"}
http_requests_rejected_total{reason="body_too_large"}
http_requests_timeout_total{stage="queue|db|upstream"}
http_active_requests_gauge
http_request_duration_seconds{stage="total|queue|handler"}
```

---

## Безопасность

### 1. Защита от Slowloris
```
ReadHeaderTimeout=10s
→ Клиент должен отправить заголовки за 10 секунд
→ Иначе соединение закрывается
```

### 2. Защита от Memory Exhaustion
```
MaxBodyBytes=1MB
→ Тело запроса ограничено 1MB
→ Logger не может прочитать > 1MB в память
```

### 3. Защита от Queue Buildup
```
RequestDeadline=25s
→ Запрос не может находиться в очереди rate limiter дольше 25s
→ Предотвращает бесконечное накопление
```

### 4. Защита от Cascading Failures
```
MaxConcurrentRequests=500
→ При перегрузке новые запросы получают 503 немедленно
→ Сервер не пытается обработать больше, чем может
```

---

## Интеграция с Nginx

Nginx также должен иметь свои таймауты (первая линия защиты):

```nginx
http {
    # Клиентские таймауты
    client_body_timeout 12s;
    client_header_timeout 12s;
    client_max_body_size 1m;
    
    # Upstream таймауты (к Go backend)
    proxy_connect_timeout 10s;
    proxy_send_timeout 30s;
    proxy_read_timeout 30s;
    
    # Keep-alive
    keepalive_timeout 120s;
    
    # Rate limiting (дополнительный слой)
    limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;
    limit_req zone=api burst=20 nodelay;
}
```

**Соотношение таймаутов**:
```
Nginx proxy_read_timeout (30s) > Go WriteTimeout (30s)
Nginx client_body_timeout (12s) < Go ReadTimeout (15s)
Nginx client_max_body_size (1m) = Go MaxBodyBytes (1MB)
```

---

## Тестирование

### 1. Тест ReadHeaderTimeout
```bash
# Медленная отправка заголовков
(echo -n "GET /api/v1/health HTTP/1.1\r\n"; sleep 15; echo "Host: localhost\r\n\r\n") | nc localhost 5050
# Ожидается: соединение закроется через ~10s
```

### 2. Тест MaxBodyBytes
```bash
# Отправка тела > 1MB
dd if=/dev/zero bs=1M count=2 | curl -X POST http://localhost:5050/api/v1/users/register \
  -H "Content-Type: application/json" --data-binary @-
# Ожидается: HTTP 413 Request Entity Too Large
```

### 3. Тест RequestDeadline
```bash
# Симуляция медленного upstream (мок сервер с задержкой 30s)
# Запрос должен вернуть 408 через ~25s
```

### 4. Тест LoadShedding
```bash
# Запустить 1000 параллельных запросов
ab -n 1000 -c 1000 http://localhost:5050/api/v1/open-weather/get-weather
# Часть запросов должна получить HTTP 503
```

---

## Troubleshooting

### Проблема: Клиенты получают 408 Timeout
**Причины**:
- RequestDeadline слишком мал для медленных операций
- Upstream API медленный
- DB запросы долгие

**Решение**:
- Увеличить `HTTP_REQUEST_DEADLINE`
- Оптимизировать DB запросы (индексы)
- Добавить кеширование

### Проблема: 503 Service Unavailable при нормальной нагрузке
**Причины**:
- `MaxConcurrentRequests` слишком мал
- Обработка запросов медленная

**Решение**:
- Увеличить `HTTP_MAX_CONCURRENT_REQUESTS`
- Оптимизировать handler'ы
- Горизонтальное масштабирование

### Проблема: 413 Request Entity Too Large
**Причины**:
- Клиент отправляет тело > `MaxBodyBytes`

**Решение**:
- Увеличить `HTTP_MAX_BODY_BYTES` (осторожно!)
- Клиент должен отправлять меньше данных
- Использовать pagination/chunking

---

## Заключение

Система timeouts и limits обеспечивает:
✅ Защиту от медленных клиентов  
✅ Ограничение использования памяти  
✅ Контролируемую деградацию при перегрузке  
✅ Предсказуемое поведение под нагрузкой  

Эти гарантии дополняют Nginx, но критичны для самого backend сервиса.
