# Настройка Grafana для визуализации метрик

## Быстрый старт

### 1. Запуск мониторинга

```bash
# Запустить Prometheus и Grafana
docker compose up -d prometheus grafana

# Проверить статус
docker ps | grep -E "(prometheus|grafana)"
```

### 2. Доступ к Grafana

Откройте в браузере: **http://localhost:3000**

**Логин:** `admin`  
**Пароль:** `admin` (или значение из `GRAFANA_ADMIN_PASSWORD` в `.env`)

При первом входе Grafana предложит сменить пароль (можно пропустить).

### 3. Открыть дашборд

После логина вы автоматически увидите дашборд **"Weather App - Metrics Overview"**.

Или через меню:
1. Нажмите на **☰ (меню слева)**
2. **Dashboards**
3. **Weather App - Metrics Overview**

## Что показывает дашборд

### HTTP Metrics
1. **HTTP Request Rate** — запросы в секунду по методам и статус-кодам
2. **Requests In-Flight** — текущее количество активных запросов (gauge)
3. **HTTP Request Latency** — P50, P95, P99 латентность

### Queue Metrics
4. **Rate Limiter Queue Depth** — глубина очереди rate limiter
5. **Queue Rejections Rate** — частота отклонений запросов

### Runtime Metrics
6. **Goroutines** — количество goroutines
7. **Memory Usage** — использование памяти (allocated, heap in use)

## Полезные фичи Grafana

### Временной диапазон
Правый верхний угол — выбор периода времени:
- Last 5 minutes
- Last 15 minutes (по умолчанию)
- Last 1 hour
- Last 24 hours
- Custom range

### Auto-refresh
Рядом с временем — кнопка **Refresh**:
- Обновление каждые 5 секунд (по умолчанию)
- Можно изменить на 10s, 30s, 1m

### Zoom
- **Кликнуть и потянуть** на графике — приблизить выбранную область
- **Двойной клик** — вернуть zoom назад

### Legend
- **Клик на legend** — скрыть/показать серию
- **Shift + клик** — показать только эту серию

### Inspect
- **Три точки** в заголовке панели → **Inspect** → **Data** — увидеть сырые данные
- **Query** — увидеть PromQL запрос

## Создание своих панелей

### Добавить панель
1. Нажмите **Add panel** (верхний правый угол)
2. Выберите тип визуализации (Time series, Gauge, Stat, etc.)
3. Добавьте PromQL запрос в **Query**

### Примеры запросов

**Error rate (%)**
```promql
sum(rate(http_requests_total{status_class="5xx"}[5m])) / 
sum(rate(http_requests_total[5m])) * 100
```

**Upstream API успешность (%)**
```promql
sum(rate(upstream_requests_total{status="success"}[5m])) / 
sum(rate(upstream_requests_total[5m])) * 100
```

**Средняя латентность**
```promql
rate(http_request_duration_seconds_sum[5m]) / 
rate(http_request_duration_seconds_count[5m])
```

**Queue wait time P95**
```promql
histogram_quantile(0.95, rate(queue_wait_duration_seconds_bucket[5m]))
```

## Настройка алертов

### Создать alert
1. Откройте панель → **Edit**
2. Вкладка **Alert**
3. **Create alert rule**
4. Настройте условие (например, `http_requests_in_flight > 100`)
5. Добавьте notification channel (Email, Slack, etc.)

### Пример alert rule

**High error rate**
```
Condition: 
sum(rate(http_requests_total{status_class="5xx"}[5m])) > 10

For: 5m

Annotations:
Summary: High HTTP error rate detected
Description: More than 10 errors/sec for 5 minutes
```

## Доступ к Prometheus

Prometheus UI доступен на **http://localhost:9090**

Полезные разделы:
- **Graph** — выполнить PromQL запросы
- **Status > Targets** — статус scraping (должен быть UP для weather-app)
- **Status > Configuration** — текущая конфигурация

## Остановка мониторинга

```bash
# Остановить только мониторинг
docker compose stop prometheus grafana

# Удалить контейнеры (данные сохранятся в volumes)
docker compose down prometheus grafana

# Удалить всё вместе с данными
docker compose down prometheus grafana -v
```

## Troubleshooting

### Grafana не показывает данные

1. **Проверьте Prometheus:**
   ```bash
   curl http://localhost:9090/api/v1/targets
   ```
   Должен показать `weather:5050` в состоянии `up`

2. **Проверьте метрики:**
   ```bash
   curl http://localhost/metrics | grep http_requests_total
   ```

3. **Проверьте datasource в Grafana:**
   - Настройки → Data sources → Prometheus
   - Нажмите **Test** — должно быть "Data source is working"

### Prometheus не собирает метрики

Проверьте логи:
```bash
docker logs weather-prometheus
```

Проверьте, что приложение экспонирует метрики:
```bash
docker exec weather wget -q -O- http://localhost:5050/metrics | head
```

### Дашборд пустой

1. Убедитесь, что приложение получает трафик:
   ```bash
   curl http://localhost/healthz
   curl http://localhost/api/v1/auth/check
   ```

2. Подождите 15-30 секунд — Prometheus собирает метрики каждые 15 секунд

3. Проверьте временной диапазон в Grafana (должен включать "now")

## Production deployment

Для production:

1. **Измените пароль Grafana** в `.env`:
   ```bash
   GRAFANA_ADMIN_PASSWORD=your_secure_password
   ```

2. **Увеличьте retention в Prometheus** (в `prometheus.yml`):
   ```yaml
   --storage.tsdb.retention.time=90d
   ```

3. **Настройте persistent volumes** для данных

4. **Настройте alert notifications:**
   - Email
   - Slack
   - PagerDuty
   - Telegram

5. **Добавьте аутентификацию** для Prometheus (basic auth через nginx)

## Полезные ссылки

- [Prometheus Query Language](https://prometheus.io/docs/prometheus/latest/querying/basics/)
- [Grafana Dashboards](https://grafana.com/grafana/dashboards/)
- [PromQL Examples](https://prometheus.io/docs/prometheus/latest/querying/examples/)
