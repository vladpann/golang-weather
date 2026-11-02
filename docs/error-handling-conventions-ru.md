# Соглашения по обработке ошибок

## Обзор

Этот документ описывает явные соглашения по обработке ошибок в проекте, которые обеспечивают:

- Предсказуемый маппинг ошибок на HTTP статус-коды
- Единообразный способ создания ошибок валидации
- Простое тестирование HTTP handlers без скрытых требований

## Проблемы, которые решены

### До

1. **Неявный маппинг**: чтобы понять, какой HTTP код вернёт ошибка, нужно было искать `case` в `ErrorResponse`
2. **Разные стили валидации**: `fmt.Errorf("...: %w", ErrInvalidArgument)` vs прямой return sentinel error
3. **Паники в тестах**: `FromContext` паникует без логгера, требуя скрытого знания о структуре зависимостей
4. **Нет документации**: новый разработчик не мог быстро добавить ошибочный сценарий

### После

1. **Явная таблица**: все sentinel errors и их HTTP коды документированы в `core_errors/doc.go`
2. **Единый стиль**: `core_errors.NewValidationError()` для всех ошибок валидации (HTTP 400)
3. **Тестовые хелперы**: `NewTestContext()`, `NewTestHandler()` — никаких паник
4. **Полное руководство**: коллега может добавить новый сценарий за 5 минут

## Быстрый старт

### Создание ошибки валидации (HTTP 400)

```go
import core_errors "github.com/vladpann/backend/golang-weather/internal/core/errors"

func ValidateOrder(productID string, quantity int) error {
    if productID == "" {
        return core_errors.NewValidationError("product ID is required")
    }
    if quantity <= 0 {
        return core_errors.NewValidationErrorf("quantity must be positive, got: %d", quantity)
    }
    return nil
}
```

### Использование sentinel errors (HTTP 409, 404, и т.д.)

```go
func CreateLocation(location string) error {
    exists, err := repo.LocationExists(location)
    if err != nil {
        return fmt.Errorf("check location: %w", err)
    }
    if exists {
        return core_errors.ErrConflict  // HTTP 409
    }
    return nil
}

func GetLocation(id int64) (*Location, error) {
    location, err := repo.FindByID(id)
    if errors.Is(err, pgx.ErrNoRows) {
        return nil, core_errors.ErrNotFound  // HTTP 404
    }
    return location, err
}
```

### Тестирование handler

```go
func TestCreateOrder(t *testing.T) {
    t.Run("пустой product_id возвращает 400", func(t *testing.T) {
        reqBody := map[string]interface{}{
            "product_id": "",
            "quantity":   5,
        }

        body, _ := json.Marshal(reqBody)
        req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))

        // ⚠️ Обязательно! Без этого FromContext паникует
        ctx := core_http_response.NewTestContext(t)
        req = req.WithContext(ctx)

        rr := httptest.NewRecorder()
        handler.CreateOrder(rr, req)

        assert.Equal(t, http.StatusBadRequest, rr.Code)
    })
}
```

## Таблица ошибок и HTTP статус-кодов

| Sentinel Error             | HTTP Status | Использование                                 |
| -------------------------- | ----------- | --------------------------------------------- |
| `ErrInvalidArgument`       | 400         | Ошибки валидации (через `NewValidationError`) |
| `ErrLoginRequired`         | 400         | Пустой логин                                  |
| `ErrPasswordRequired`      | 400         | Пустой пароль                                 |
| `ErrPasswordDoNotMatch`    | 400         | Пароли не совпадают                           |
| `ErrPasswordConstrains`    | 400         | Пароль не соответствует требованиям           |
| `ErrCityNameRequired`      | 400         | Пустое название города                        |
| `ErrInvalidCredentials`    | 401         | Неверный логин или пароль                     |
| `ErrNotFound`              | 404         | Ресурс не найден                              |
| `ErrConflict`              | 409         | Конфликт ресурсов (generic)                   |
| `ErrLocationAlreadyExists` | 409         | Локация уже добавлена                         |
| **Любая другая ошибка**    | 500         | Внутренняя ошибка сервера                     |

**Полная документация:** `internal/core/errors/doc.go`

## API Reference

### Создание ошибок

| Функция                                | Назначение                         | HTTP код |
| -------------------------------------- | ---------------------------------- | -------- |
| `NewValidationError(msg)`              | Ошибка валидации                   | 400      |
| `NewValidationErrorf(format, args...)` | Ошибка валидации с форматированием | 400      |
| `ErrNotFound`                          | Ресурс не найден                   | 404      |
| `ErrConflict`                          | Конфликт ресурсов                  | 409      |

### Проверка ошибок

| Функция                       | Назначение                           |
| ----------------------------- | ------------------------------------ |
| `errors.Is(err, ErrNotFound)` | Проверка конкретного sentinel error  |
| `IsValidationError(err)`      | Проверка ошибки валидации (HTTP 400) |

### Тестовые хелперы

| Функция                         | Назначение                                      |
| ------------------------------- | ----------------------------------------------- |
| `NewTestHandler(t, rw)`         | Handler с Nop-логгером                          |
| `NewTestHandlerWithLogs(t, rw)` | Handler + наблюдаемые логи                      |
| `NewTestContext(t)`             | Контекст с логгером (обязательно для handlers!) |
| `NewTestContextWithLogs(t)`     | Контекст + наблюдаемые логи                     |

## Примеры использования

### Пример 1: Валидатор с несколькими проверками

```go
type ProductInput struct {
    Name     string
    Price    float64
    Category string
}

func (i *ProductInput) Validate() error {
    if i.Name == "" {
        return core_errors.NewValidationError("product name is required")
    }
    if i.Price <= 0 {
        return core_errors.NewValidationErrorf("price must be positive, got: %.2f", i.Price)
    }
    if i.Category == "" {
        return core_errors.NewValidationError("category is required")
    }
    return nil
}
```

### Пример 2: Service с бизнес-логикой

```go
func (s *OrdersService) CreateOrder(ctx context.Context, input OrderInput) (*Order, error) {
    // Валидация
    if err := input.Validate(); err != nil {
        return nil, fmt.Errorf("validate input: %w", err)
    }

    // Проверка существования продукта
    product, err := s.productsRepo.FindByID(ctx, input.ProductID)
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, core_errors.ErrNotFound  // HTTP 404
        }
        return nil, fmt.Errorf("find product: %w", err)
    }

    // Проверка на дубликаты
    exists, err := s.ordersRepo.OrderExists(ctx, input.ProductID, input.UserID)
    if err != nil {
        return nil, fmt.Errorf("check order existence: %w", err)
    }
    if exists {
        return nil, core_errors.ErrConflict  // HTTP 409
    }

    // Создание
    order, err := s.ordersRepo.Create(ctx, input)
    if err != nil {
        return nil, fmt.Errorf("create order: %w", err)
    }

    return order, nil
}
```

### Пример 3: HTTP Handler

```go
func (h *OrdersHTTPHandler) CreateOrder(rw http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    log := core_logger.FromContext(ctx)
    responseHandler := core_http_response.NewHTTPResponseHandler(log, rw)

    var request CreateOrderRequest
    if err := core_http_request.DecodeAndValidateRequest(r, &request); err != nil {
        responseHandler.ErrorResponse(err, "failed to decode request")
        return
    }

    input := OrderInput{
        ProductID: request.ProductID,
        Quantity:  request.Quantity,
        UserID:    getUserIDFromContext(ctx),
    }

    order, err := h.ordersService.CreateOrder(ctx, input)
    if err != nil {
        // ErrorResponse автоматически определит HTTP код:
        // - ErrInvalidArgument → 400
        // - ErrNotFound → 404
        // - ErrConflict → 409
        // - Любая другая → 500
        responseHandler.ErrorResponse(err, "failed to create order")
        return
    }

    responseHandler.JSONResponse(order, http.StatusCreated)
}
```

### Пример 4: Полный тест handler

```go
func TestCreateOrder(t *testing.T) {
    t.Run("успешное создание возвращает 201", func(t *testing.T) {
        reqBody := map[string]interface{}{
            "product_id": "prod-123",
            "quantity":   5,
        }

        body, _ := json.Marshal(reqBody)
        req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
        req.Header.Set("Content-Type", "application/json")

        ctx := core_http_response.NewTestContext(t)
        req = req.WithContext(ctx)

        rr := httptest.NewRecorder()
        handler.CreateOrder(rr, req)

        assert.Equal(t, http.StatusCreated, rr.Code)

        var response map[string]interface{}
        err := json.NewDecoder(rr.Body).Decode(&response)
        require.NoError(t, err)
        assert.Contains(t, response, "id")
    })

    t.Run("пустой product_id возвращает 400", func(t *testing.T) {
        reqBody := map[string]interface{}{
            "product_id": "",
            "quantity":   5,
        }

        body, _ := json.Marshal(reqBody)
        req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
        req.Header.Set("Content-Type", "application/json")

        ctx := core_http_response.NewTestContext(t)
        req = req.WithContext(ctx)

        rr := httptest.NewRecorder()
        handler.CreateOrder(rr, req)

        assert.Equal(t, http.StatusBadRequest, rr.Code)
    })

    t.Run("несуществующий продукт возвращает 404", func(t *testing.T) {
        // Mock service для возврата ErrNotFound
        mockService := &MockOrdersService{
            CreateOrderFunc: func(ctx context.Context, input OrderInput) (*Order, error) {
                return nil, core_errors.ErrNotFound
            },
        }

        handler := NewOrdersHTTPHandler(mockService)

        reqBody := map[string]interface{}{
            "product_id": "nonexistent",
            "quantity":   5,
        }

        body, _ := json.Marshal(reqBody)
        req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
        ctx := core_http_response.NewTestContext(t)
        req = req.WithContext(ctx)

        rr := httptest.NewRecorder()
        handler.CreateOrder(rr, req)

        assert.Equal(t, http.StatusNotFound, rr.Code)
    })
}
```

## Чеклист: Добавление нового ошибочного сценария

Используйте этот чеклист, когда добавляете новую обработку ошибок:

- [ ] **Определите тип ошибки:**
  - Ошибка валидации (400)? → используйте `NewValidationError`
  - Конфликт ресурсов (409)? → используйте существующий sentinel (например, `ErrConflict`)
  - Новый тип ошибки? → добавьте sentinel в `core_errors/common.go` и case в `ErrorResponse`

- [ ] **Создайте валидатор (если нужен):**
  - Используйте `NewValidationError` для ошибок валидации
  - Документируйте условия валидации

- [ ] **Добавьте обработку в service:**
  - Возвращайте правильный sentinel error
  - Оборачивайте ошибки для контекста: `fmt.Errorf("operation: %w", err)`

- [ ] **Обновите handler:**
  - Используйте `ErrorResponse` — он автоматически определит HTTP код
  - Добавьте понятное сообщение ошибки

- [ ] **Напишите тест:**
  - Используйте `NewTestContext` для создания контекста
  - Проверьте ожидаемый HTTP статус-код
  - Проверьте структуру ответа

- [ ] **Проверьте логирование (опционально):**
  - Используйте `NewTestHandlerWithLogs` если нужно проверить логи

## FAQ

**Q: Нужно ли создавать новый sentinel error для каждой ошибки валидации?**  
A: Нет! Используйте `NewValidationError` — все они автоматически вернут HTTP 400.

**Q: Мой тест паникует с "no logger in context". Что делать?**  
A: Используйте `NewTestContext(t)` для создания контекста request:

```go
ctx := core_http_response.NewTestContext(t)
req = req.WithContext(ctx)
```

**Q: Как узнать, какой HTTP код вернёт моя ошибка?**  
A: Смотрите таблицу выше или документацию в `internal/core/errors/doc.go`.

**Q: Когда использовать `NewTestHandlerWithLogs`?**  
A: Когда нужно проверить, что handler правильно логирует ошибки:

```go
handler, logs := core_http_response.NewTestHandlerWithLogs(t, rr)
// ... вызовите handler ...
assert.Equal(t, 1, logs.Len())
assert.Contains(t, logs.All()[0].Message, "expected log message")
```

**Q: Можно ли комбинировать sentinel errors?**  
A: Да, используйте `errors.Is` для проверки:

```go
err := service.DoSomething()
if errors.Is(err, core_errors.ErrNotFound) || errors.Is(err, core_errors.ErrInvalidArgument) {
    // Обработка
}
```

## Дополнительные ресурсы

- **[docs/testing-http-handlers-ru.md](./testing-http-handlers-ru.md)** — подробное руководство по тестированию

## Примеры в коде

Примеры рабочего кода можно найти в:

- `internal/features/users/service/validators.go` — валидатор с `NewValidationError`
- `internal/features/users/transport/http/create_user.go` — handler с обработкой ошибок
- `internal/features/users/transport/http/transport_test.go` — тесты с `NewTestContext`
- `internal/core/transport/http/response/testing_test.go` — примеры использования всех хелперов
