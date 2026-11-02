# Руководство по тестированию HTTP handlers

## Обзор

Этот документ объясняет, как правильно создавать и тестировать HTTP handlers в проекте, включая:

- Обработку ошибок и их маппинг на HTTP статус-коды
- Использование тестовых хелперов для handlers
- Создание тестов, которые не паникуют из-за отсутствующего логгера

## Таблица ошибок и HTTP статус-кодов

`core_http_response.HTTPResponseHandler.ErrorResponse` автоматически преобразует sentinel errors в HTTP коды:

| Sentinel Error             | HTTP Status | Когда использовать                  |
| -------------------------- | ----------- | ----------------------------------- |
| `ErrInvalidArgument`       | 400         | Некорректные входные данные         |
| `ErrLoginRequired`         | 400         | Пустой логин                        |
| `ErrPasswordRequired`      | 400         | Пустой пароль                       |
| `ErrPasswordDoNotMatch`    | 400         | Пароли не совпадают                 |
| `ErrPasswordConstrains`    | 400         | Пароль не соответствует требованиям |
| `ErrCityNameRequired`      | 400         | Пустое название города              |
| `ErrInvalidCredentials`    | 401         | Неверный логин или пароль           |
| `ErrNotFound`              | 404         | Ресурс не найден                    |
| `ErrConflict`              | 409         | Конфликт ресурсов (generic)         |
| `ErrLocationAlreadyExists` | 409         | Локация уже добавлена               |
| Любая другая ошибка        | 500         | Внутренняя ошибка сервера           |

## Создание ошибок валидации

### Рекомендуемый способ: `NewValidationError`

Используйте `core_errors.NewValidationError` для создания ошибок валидации:

```go
func ValidateOrder(quantity int, productID string) error {
    if quantity <= 0 {
        return core_errors.NewValidationError("quantity must be positive")
    }
    if productID == "" {
        return core_errors.NewValidationError("product ID is required")
    }
    return nil
}
```

**Почему так?**

- Автоматически оборачивает ошибку в `ErrInvalidArgument`
- Гарантирует HTTP 400 в `ErrorResponse`
- Единообразный стиль в кодовой базе

### С форматированием: `NewValidationErrorf`

```go
func ValidateAge(age int) error {
    if age < 0 || age > 150 {
        return core_errors.NewValidationErrorf(
            "age must be between 0 and 150, got: %d", age,
        )
    }
    return nil
}
```

### Возврат существующих sentinel errors

Если у ошибки есть специальный sentinel, возвращайте его напрямую:

```go
func CheckLocationExists(location string) error {
    exists, err := repo.LocationExists(location)
    if err != nil {
        return fmt.Errorf("check location existence: %w", err)
    }
    if exists {
        return core_errors.ErrConflict  // HTTP 409
    }
    return nil
}
```

## Тестирование HTTP handlers

### Проблема: FromContext паникует без логгера

`core_logger.FromContext` паникует, если в контексте нет логгера:

```go
func (h *Handler) CreateUser(rw http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    log := core_logger.FromContext(ctx)  // ⚠️ PANIC если логгер не установлен
    // ...
}
```

### Решение: тестовые хелперы

#### 1. Тестирование простого handler

Когда нужно проверить только HTTP статус и тело ответа:

```go
func TestMyHandler(t *testing.T) {
    rr := httptest.NewRecorder()
    handler := core_http_response.NewTestHandler(t, rr)

    handler.ErrorResponse(core_errors.ErrNotFound, "user not found")

    assert.Equal(t, http.StatusNotFound, rr.Code)
}
```

#### 2. Тестирование с проверкой логов

Когда нужно проверить, что handler правильно логирует:

```go
func TestMyHandlerLogs(t *testing.T) {
    rr := httptest.NewRecorder()
    handler, logs := core_http_response.NewTestHandlerWithLogs(t, rr)

    handler.ErrorResponse(errors.New("db error"), "failed to save user")

    assert.Equal(t, http.StatusInternalServerError, rr.Code)
    assert.Equal(t, 1, logs.Len())
    assert.Contains(t, logs.All()[0].Message, "failed to save user")
}
```

#### 3. Тестирование handler, использующего `FromContext`

Когда ваш handler вызывает `core_logger.FromContext`:

```go
func TestHTTPHandler(t *testing.T) {
    req := httptest.NewRequest("POST", "/users", body)
    ctx := core_http_response.NewTestContext(t)
    req = req.WithContext(ctx)

    rr := httptest.NewRecorder()
    handler.ServeHTTP(rr, req)

    assert.Equal(t, http.StatusCreated, rr.Code)
}
```

#### 4. Тестирование с контекстом и логами

Полный контроль над логированием и контекстом:

```go
func TestHTTPHandlerWithLogs(t *testing.T) {
    req := httptest.NewRequest("POST", "/users", body)
    ctx, logs := core_http_response.NewTestContextWithLogs(t)
    req = req.WithContext(ctx)

    rr := httptest.NewRecorder()
    handler.ServeHTTP(rr, req)

    assert.Equal(t, 1, logs.Len())
    assert.Equal(t, "user created", logs.All()[0].Message)
}
```

## Полный пример: добавление нового handler

Предположим, вам нужно добавить endpoint для создания заказа с валидацией.

### 1. Создайте валидатор

```go
// internal/features/orders/service/validators.go
package orders_service

import (
    core_errors "github.com/vladpann/backend/golang-weather/internal/core/errors"
)

type OrderInput struct {
    ProductID string
    Quantity  int
    UserID    int64
}

func (i *OrderInput) Validate() error {
    if i.ProductID == "" {
        return core_errors.NewValidationError("product ID is required")
    }
    if i.Quantity <= 0 {
        return core_errors.NewValidationError("quantity must be positive")
    }
    if i.UserID <= 0 {
        return core_errors.NewValidationError("user ID must be positive")
    }
    return nil
}
```

### 2. Реализуйте handler

```go
// internal/features/orders/transport/http/create_order.go
package orders_transport_http

import (
    "net/http"

    core_logger "github.com/vladpann/backend/golang-weather/internal/core/logger"
    core_http_request "github.com/vladpann/backend/golang-weather/internal/core/transport/http/request"
    core_http_response "github.com/vladpann/backend/golang-weather/internal/core/transport/http/response"
)

type CreateOrderRequest struct {
    ProductID string `json:"product_id"`
    Quantity  int    `json:"quantity"`
}

func (h *OrdersHTTPHandler) CreateOrder(rw http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    log := core_logger.FromContext(ctx)
    responseHandler := core_http_response.NewHTTPResponseHandler(log, rw)

    var request CreateOrderRequest
    if err := core_http_request.DecodeAndValidateRequest(r, &request); err != nil {
        responseHandler.ErrorResponse(err, "failed to decode request")
        return
    }

    // Валидация
    input := OrderInput{
        ProductID: request.ProductID,
        Quantity:  request.Quantity,
        UserID:    getUserIDFromContext(ctx),
    }
    if err := input.Validate(); err != nil {
        responseHandler.ErrorResponse(err, "validation failed")
        return
    }

    // Бизнес-логика
    order, err := h.ordersService.CreateOrder(ctx, input)
    if err != nil {
        responseHandler.ErrorResponse(err, "failed to create order")
        return
    }

    responseHandler.JSONResponse(order, http.StatusCreated)
}
```

### 3. Напишите тесты

```go
// internal/features/orders/transport/http/create_order_test.go
package orders_transport_http_test

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/stretchr/testify/assert"

    core_http_response "github.com/vladpann/backend/golang-weather/internal/core/transport/http/response"
)

func TestCreateOrder(t *testing.T) {
    t.Run("успешное создание заказа возвращает 201", func(t *testing.T) {
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
    })

    t.Run("пустой product_id возвращает 400", func(t *testing.T) {
        reqBody := map[string]interface{}{
            "product_id": "",  // Невалидно
            "quantity":   5,
        }

        body, _ := json.Marshal(reqBody)
        req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
        req.Header.Set("Content-Type", "application/json")

        ctx := core_http_response.NewTestContext(t)
        req = req.WithContext(ctx)

        rr := httptest.NewRecorder()
        handler.CreateOrder(rr, req)

        // Ожидаем HTTP 400, так как NewValidationError оборачивает в ErrInvalidArgument
        assert.Equal(t, http.StatusBadRequest, rr.Code)
    })

    t.Run("отрицательный quantity возвращает 400", func(t *testing.T) {
        reqBody := map[string]interface{}{
            "product_id": "prod-123",
            "quantity":   -1,  // Невалидно
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
}
```

## Чеклист для добавления нового ошибочного сценария

✅ **Определите тип ошибки:**

- Ошибка валидации (400) → используйте `NewValidationError`
- Конкретная ошибка с существующим sentinel → возвращайте sentinel напрямую
- Новый тип ошибки → создайте sentinel в `core_errors/common.go` и добавьте case в `ErrorResponse`

✅ **Создайте валидатор:**

- Используйте `NewValidationError` для ошибок валидации
- Документируйте условия валидации

✅ **Напишите тест:**

- Используйте `NewTestContext` чтобы избежать паники
- Проверьте ожидаемый HTTP статус-код
- Проверьте структуру ответа

✅ **Проверьте логирование (опционально):**

- Используйте `NewTestHandlerWithLogs` если нужно проверить логи
- Проверьте правильный уровень логирования (Error, Warn, Debug)

## Справочник по тестовым хелперам

### `core_http_response.NewTestHandler(t, rw)`

Создаёт `HTTPResponseHandler` с Nop-логгером для простых тестов.

### `core_http_response.NewTestHandlerWithLogs(t, rw)`

Создаёт `HTTPResponseHandler` с наблюдаемыми логами для проверки логирования.

### `core_http_response.NewTestContext(t)`

Создаёт контекст с тестовым логгером. **Обязательно** для handlers, использующих `FromContext`.

### `core_http_response.NewTestContextWithLogs(t)`

Создаёт контекст с наблюдаемым логгером для проверки логирования из контекста.

## Часто задаваемые вопросы

**Q: Мой тест паникует с "no logger in context"**  
A: Используйте `NewTestContext` для создания контекста request:

```go
ctx := core_http_response.NewTestContext(t)
req = req.WithContext(ctx)
```

**Q: Как узнать, какой HTTP код вернёт моя ошибка?**  
A: Смотрите таблицу в начале документа или `internal/core/errors/doc.go`.

**Q: Нужно ли мне создавать новый sentinel error для каждой ошибки валидации?**  
A: Нет! Используйте `NewValidationError` — все они вернут HTTP 400.

**Q: Как проверить, что handler правильно логирует ошибки?**  
A: Используйте `NewTestHandlerWithLogs`:

```go
handler, logs := core_http_response.NewTestHandlerWithLogs(t, rr)
// ... вызовите handler ...
assert.Equal(t, 1, logs.Len())
```

**Q: Можно ли использовать эти хелперы в integration тестах?**  
A: Да, но для integration тестов с реальной БД смотрите существующие примеры в `transport_test.go`.
