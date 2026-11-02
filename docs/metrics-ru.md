# Prometheus метрики

## Обзор

Приложение экспонирует метрики в формате Prometheus на эндпойнте `/metrics`. Метрики охватывают HTTP-запросы, вызовы upstream API (OpenWeather), очередь rate limiter и runtime Go.

## Доступ к метрикам

```bash
curl http://localhost/metrics
```

## HTTP метрики

### `http_requests_total`
**Тип:** Counter  
**Описание:** Общее количество обработанных HTTP-запросов  
**Labels:**
- `method` — HTTP метод (GET, POST, etc.)
- `status_class` — класс статус-кода (2xx, 3xx, 4xx, 5xx)

**Пример:**
```
http_requests_total{method="GET",status_class="2xx"} 1234
http_requests_total{method="POST",status_class="4xx"} 45
```

### `http_request_duration_seconds`
**Тип:** Histogram  
**Описание:** Распределение латентности HTTP-запросов в секундах  
**Labels:**
- `method` — HTTP метод
- `status_class` — класс статус-кода

**Buckets:** 0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0

**Пример:**
```
http_request_duration_seconds_bucket{method="GET",status_class="2xx",le="0.1"} 980
http_request_duration_seconds_bucket{method="GET",status_class="2xx",le="0.5"} 1200
http_request_duration_seconds_sum{method="GET",status_class="2xx"} 156.789
http_request_duration_seconds_count{method="GET",status_class="2xx"} 1234
```

### `http_requests_in_flight`
**Тип:** Gauge  
**Описание:** Текущее количество обрабатываемых HTTP-запросов

**Пример:**
```
http_requests_in_flight 5
```

## Upstream метрики (OpenWeather API)

### `upstream_requests_total`
**Тип:** Counter  
**Описание:** Общее количество запросов к upstream API  
**Labels:**
- `endpoint_group` — группа эндпойнтов (`weather`, `geocoding`)
- `status` — результат (`success`, `error`)

**Пример:**
```
upstream_requests_total{endpoint_group="weather",status="success"} 5678
upstream_requests_total{endpoint_group="weather",status="error"} 23
```

### `upstream_request_duration_seconds`
**Тип:** Histogram  
**Описание:** Распределение латентности запросов к upstream API  
**Labels:**
- `endpoint_group` — группа эндпойнтов

**Buckets:** 0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0

### `upstream_errors_total`
**Тип:** Counter  
**Описание:** Общее количество ошибок upstream API по типам  
**Labels:**
- `endpoint_group` — группа эндпойнтов
- `error_type` — тип ошибки:
  - `timeout` — таймаут запроса
  - `auth` — ошибка аутентификации (401, 403)
  - `rate_limit` — превышен rate limit (429)
  - `server_error` — ошибка сервера (5xx)
  - `client_error` — ошибка клиента (4xx, кроме auth и rate_limit)
  - `network` — сетевая ошибка

**Пример:**
```
upstream_errors_total{endpoint_group="weather",error_type="timeout"} 5
upstream_errors_total{endpoint_group="weather",error_type="rate_limit"} 12
```

## Метрики очереди (Rate Limiter)

### `queue_depth`
**Тип:** Gauge  
**Описание:** Текущее количество запросов, ожидающих в очереди rate limiter

**Пример:**
```
queue_depth 3
```

### `queue_wait_duration_seconds`
**Тип:** Histogram  
**Описание:** Время ожидания в очереди rate limiter  
**Buckets:** 0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0

### `queue_rejections_total`
**Тип:** Counter  
**Описание:** Количество запросов, отклоненных из-за переполнения очереди

**Пример:**
```
queue_rejections_total 42
```

## Runtime метрики Go

Автоматически экспонируются стандартные метрики Go:

### Память
- `go_memstats_alloc_bytes` — выделенная память
- `go_memstats_heap_alloc_bytes` — heap memory
- `go_memstats_heap_inuse_bytes` — используемая heap память
- `go_memstats_stack_inuse_bytes` — используемая stack память

### Garbage Collector
- `go_gc_duration_seconds` — длительность GC pause
- `go_memstats_gc_cpu_fraction` — доля CPU времени, потраченная на GC

### Goroutines
- `go_goroutines` — текущее количество goroutines

### Process метрики
- `process_cpu_seconds_total` — CPU time
- `process_open_fds` — открытые file descriptors
- `process_resident_memory_bytes` — resident memory

## Принципы ограничения cardinality

Метрики спроектированы с ограниченной кардинальностью labels:

❌ **НЕ используются в labels:**
- User IDs
- Полные URL
- Произвольные строки
- IP адреса

✅ **Используются только:**
- HTTP методы (GET, POST, etc.)
- Классы статус-кодов (2xx, 3xx, 4xx, 5xx)
- Группы эндпойнтов (weather, geocoding)
- Предопределенные типы ошибок

Это предотвращает взрывной рост количества временных рядов в Prometheus.

## Примеры запросов PromQL

### Средняя латентность HTTP-запросов за 5 минут
```promql
rate(http_request_duration_seconds_sum[5m]) / rate(http_request_duration_seconds_count[5m])
```

### P95 латентность HTTP-запросов
```promql
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))
```

### Rate ошибок upstream API
```promql
rate(upstream_requests_total{status="error"}[5m])
```

### Процент ошибок
```promql
sum(rate(upstream_requests_total{status="error"}[5m])) / sum(rate(upstream_requests_total[5m])) * 100
```

### Средняя глубина очереди
```promql
avg_over_time(queue_depth[5m])
```

### Количество отклоненных запросов в секунду
```promql
rate(queue_rejections_total[1m])
```

## Интеграция с Grafana

Рекомендуемые дашборды:

1. **HTTP Overview**
   - Requests per second по методам и статус-кодам
   - P50/P95/P99 латентность
   - Количество in-flight requests
   - Error rate

2. **Upstream API**
   - Requests per second к weather и geocoding API
   - Латентность вызовов
   - Error rate по типам
   - Распределение ошибок timeout/rate_limit/server

3. **Queue Health**
   - Текущая глубина очереди
   - Время ожидания (P50/P95/P99)
   - Rate отклонений
   - Throughput

4. **Go Runtime**
   - Memory usage
   - GC frequency и duration
   - Goroutine count
   - CPU usage

## Алерты

Примеры правил алертинга для Prometheus:

```yaml
groups:
  - name: weather_app
    rules:
      # High error rate
      - alert: HighErrorRate
        expr: |
          sum(rate(http_requests_total{status_class="5xx"}[5m])) 
          / sum(rate(http_requests_total[5m])) > 0.05
        for: 5m
        annotations:
          summary: "High HTTP error rate (> 5%)"

      # Upstream API errors
      - alert: UpstreamAPIErrors
        expr: rate(upstream_errors_total[5m]) > 1
        for: 5m
        annotations:
          summary: "Upstream API errors detected"

      # Queue saturation
      - alert: QueueNearFull
        expr: queue_depth > 80
        for: 2m
        annotations:
          summary: "Rate limiter queue near capacity"

      # High latency
      - alert: HighLatency
        expr: |
          histogram_quantile(0.95, 
            rate(http_request_duration_seconds_bucket[5m])
          ) > 1
        for: 5m
        annotations:
          summary: "P95 latency > 1 second"
```

## Troubleshooting

### Метрики не отображаются

Проверьте, что эндпойнт доступен:
```bash
curl http://localhost/metrics
```

### Высокая cardinality

Если количество временных рядов растет слишком быстро:
1. Проверьте, что в labels не попадают динамические значения
2. Используйте фиксированный набор значений для labels
3. Группируйте схожие метрики

### Производительность

Сбор метрик имеет минимальный overhead:
- Counters: ~50ns
- Histograms: ~300ns
- Gauges: ~20ns

При высоком RPS рекомендуется:
- Использовать меньше buckets в histograms
- Уменьшить scrape interval в Prometheus
