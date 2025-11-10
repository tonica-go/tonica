# Пользовательские маршруты с документацией OpenAPI

Fluent API библиотеки Tonica позволяет добавлять пользовательские HTTP маршруты, которые не определены в ваших Protocol Buffers. Эти маршруты автоматически документируются в спецификации OpenAPI и отображаются в интерфейсе Scalar вместе с вашими эндпоинтами, сгенерированными из proto.

## Зачем нужны пользовательские маршруты?

Хотя proto-first подход очень мощный, иногда вам нужны маршруты, которые не вписываются в модель RPC:

- Проверки работоспособности (`/health`, `/ready`)
- Отдача статических файлов
- Устаревшие эндпоинты
- Внутренние утилиты
- Вебхуки
- Специальные эндпоинты для аутентификации

Пользовательские маршруты позволяют добавлять их без изменения ваших proto файлов.

## Два подхода к созданию маршрутов

В Tonica есть два способа добавления пользовательских HTTP-маршрутов:

1.  **Fluent API (`tonica.NewRoute`)**: Основной и рекомендуемый способ. Он позволяет создавать маршруты и одновременно генерировать для них документацию OpenAPI.
2.  **Прямой доступ к роутеру (`app.GetRouter()`)**: Быстрый способ добавить стандартные обработчики [Gin](https://gin-gonic.com/), когда вам не нужна автоматическая документация.

---

## Прямой доступ к роутеру (без OpenAPI)

Иногда вам может понадобиться добавить эндпоинт очень быстро, и для него не требуется документация. Например, для внутренних нужд или быстрых тестов. В этом случае вы можете получить доступ к базовому роутеру Gin и использовать его стандартный API.

**Эти маршруты не будут добавлены в вашу спецификацию OpenAPI.**

```go
// Получаем экземпляр роутера Gin
router := app.GetRouter()

// Используем стандартные методы Gin
router.GET("/ping", func(c *gin.Context) {
    c.String(200, "pong")
})

router.POST("/internal/sync", func(c *gin.Context) {
    // какая-то внутренняя логика
    c.JSON(200, gin.H{"status": "sync started"})
})
```

Этот способ полезен для:
- Эндпоинтов, которые не должны быть видны в публичной документации.
- Быстрого прототипирования и отладки.
- Интеграции с существующими обработчиками Gin.

Далее в этом руководстве мы сосредоточимся на **Fluent API**, который является предпочтительным способом для создания документированных и поддерживаемых маршрутов.

---

## Быстрый старт с Fluent API

### Простой пример

```go
tonica.NewRoute(app).
    GET("/health").
    Summary("Health check endpoint").
    Description("Returns the health status of the service").
    Tags("Monitoring").
    Response(200, "Service is healthy", tonica.InlineObjectSchema(map[string]string{
        "status": "string",
    })).
    Handle(func(c *gin.Context) {
        c.JSON(200, gin.H{"status": "healthy"})
    })
```

Это создает:
1. GET эндпоинт по адресу `/health`
2. OpenAPI документацию для эндпоинта
3. Автоматическое отображение в интерфейсе Scalar по адресу `/docs`

### Просмотр документации

После запуска вашего приложения:

- **OpenAPI спецификация**: `http://localhost:8080/openapi.json`
- **Интерфейс Scalar**: `http://localhost:8080/docs`

## HTTP методы

Поддерживаются все стандартные HTTP методы:

### GET

```go
tonica.NewRoute(app).
    GET("/users/:id").
    Summary("Get user by ID").
    Handle(func(c *gin.Context) {
        id := c.Param("id")
        // Fetch user...
        c.JSON(200, gin.H{"id": id})
    })
```

### POST

```go
tonica.NewRoute(app).
    POST("/users").
    Summary("Create new user").
    BodyParam("User data", tonica.InlineObjectSchema(map[string]string{
        "name":  "string",
        "email": "string",
    })).
    Handle(func(c *gin.Context) {
        var user map[string]interface{}
        if err := c.BindJSON(&user); err != nil {
            c.JSON(400, gin.H{"error": err.Error()})
            return
        }
        // Create user...
        c.JSON(201, gin.H{"id": "123"})
    })
```

### PUT

```go
tonica.NewRoute(app).
    PUT("/users/:id").
    Summary("Update user").
    PathParam("id", "string", "User ID").
    BodyParam("Updated user data", tonica.InlineObjectSchema(map[string]string{
        "name":  "string",
        "email": "string",
    })).
    Handle(func(c *gin.Context) {
        id := c.Param("id")
        // Update user...
        c.JSON(200, gin.H{"id": id})
    })
```

### PATCH

```go
tonica.NewRoute(app).
    PATCH("/users/:id").
    Summary("Partially update user").
    PathParam("id", "string", "User ID").
    BodyParam("Fields to update", tonica.ObjectSchema(map[string]interface{}{
        "name":  map[string]string{"type": "string"},
        "email": map[string]string{"type": "string"},
    })).
    Handle(func(c *gin.Context) {
        id := c.Param("id")
        // Patch user...
        c.JSON(200, gin.H{"id": id})
    })
```

### DELETE

```go
tonica.NewRoute(app).
    DELETE("/users/:id").
    Summary("Delete user").
    PathParam("id", "string", "User ID").
    Response(204, "User deleted successfully", nil).
    Handle(func(c *gin.Context) {
        id := c.Param("id")
        // Delete user...
        c.Status(204)
    })
```

## Параметры

### Query параметры

Query параметры передаются в URL: `/endpoint?name=value`

```go
tonica.NewRoute(app).
    GET("/users").
    Summary("List users").
    QueryParam("page", "integer", "Page number", false).
    QueryParam("limit", "integer", "Items per page", false).
    QueryParam("status", "string", "Filter by status", false).
    Handle(func(c *gin.Context) {
        page := c.DefaultQuery("page", "1")
        limit := c.DefaultQuery("limit", "10")
        status := c.Query("status")

        // Fetch users with pagination...
        c.JSON(200, gin.H{
            "page":  page,
            "limit": limit,
            "users": []interface{}{},
        })
    })
```

**Обязательные и необязательные:**
```go
// Required query parameter
.QueryParam("name", "string", "User name", true)

// Optional query parameter
.QueryParam("lang", "string", "Language code", false)
```

### Path параметры

Path параметры являются частью URL: `/users/:id`

```go
tonica.NewRoute(app).
    GET("/users/:id/orders/:orderId").
    Summary("Get user order").
    PathParam("id", "string", "User ID").
    PathParam("orderId", "string", "Order ID").
    Handle(func(c *gin.Context) {
        userID := c.Param("id")
        orderID := c.Param("orderId")

        // Fetch order...
        c.JSON(200, gin.H{
            "userId":  userID,
            "orderId": orderID,
        })
    })
```

**Примечание:** Path параметры всегда обязательны.

### Header параметры

Чтение значений из HTTP заголовков:

```go
tonica.NewRoute(app).
    GET("/protected").
    Summary("Protected endpoint").
    HeaderParam("Authorization", "string", "Bearer token", true).
    HeaderParam("X-Request-ID", "string", "Request ID for tracing", false).
    Handle(func(c *gin.Context) {
        auth := c.GetHeader("Authorization")
        requestID := c.GetHeader("X-Request-ID")

        // Validate auth...
        c.JSON(200, gin.H{"authenticated": true})
    })
```

### Body параметры

Для POST, PUT, PATCH запросов:

```go
tonica.NewRoute(app).
    POST("/users").
    Summary("Create user").
    BodyParam("User data", tonica.InlineObjectSchema(map[string]string{
        "name":     "string",
        "email":    "string",
        "age":      "integer",
        "active":   "boolean",
    })).
    Handle(func(c *gin.Context) {
        var user struct {
            Name   string `json:"name"`
            Email  string `json:"email"`
            Age    int    `json:"age"`
            Active bool   `json:"active"`
        }

        if err := c.BindJSON(&user); err != nil {
            c.JSON(400, gin.H{"error": "invalid request body"})
            return
        }

        // Create user...
        c.JSON(201, user)
    })
```

## Документация

### Краткое описание и подробное описание

```go
tonica.NewRoute(app).
    GET("/stats").
    Summary("Get statistics").  // Краткое однострочное описание
    Description(`
        Returns comprehensive statistics about the system including:
        - Total users
        - Active sessions
        - Request rates
        - Error rates

        This endpoint is cached for 5 minutes.
    `).  // Длинное подробное описание
    Handle(func(c *gin.Context) {
        // ...
    })
```

### Теги

Теги группируют эндпоинты в интерфейсе документации:

```go
// Один тег
tonica.NewRoute(app).
    GET("/health").
    Tag("Monitoring").
    Handle(func(c *gin.Context) { /* ... */ })

// Несколько тегов
tonica.NewRoute(app).
    GET("/users").
    Tags("Users", "Public", "v1").
    Handle(func(c *gin.Context) { /* ... */ })

// Добавление тегов по одному
tonica.NewRoute(app).
    GET("/orders").
    Tag("Orders").
    Tag("Commerce").
    Tag("v2").
    Handle(func(c *gin.Context) { /* ... */ })
```

**Результат в интерфейсе Scalar:**
```
└─ Monitoring
   └─ GET /health

└─ Users
   └─ GET /users

└─ Orders
   └─ GET /orders
```

## Ответы

### Простой ответ

```go
tonica.NewRoute(app).
    GET("/ping").
    Response(200, "Pong response", tonica.StringSchema()).
    Handle(func(c *gin.Context) {
        c.String(200, "pong")
    })
```

### Несколько ответов

```go
tonica.NewRoute(app).
    POST("/users").
    Response(201, "User created successfully", tonica.InlineObjectSchema(map[string]string{
        "id":      "string",
        "message": "string",
    })).
    Response(400, "Invalid request data", tonica.InlineObjectSchema(map[string]string{
        "error": "string",
    })).
    Response(409, "User already exists", tonica.InlineObjectSchema(map[string]string{
        "error": "string",
    })).
    Response(500, "Internal server error", tonica.InlineObjectSchema(map[string]string{
        "error": "string",
    })).
    Handle(func(c *gin.Context) {
        // Handler implementation...
    })
```

### Ответ со сложной схемой

```go
tonica.NewRoute(app).
    GET("/users/:id").
    Response(200, "User information", tonica.ObjectSchema(map[string]interface{}{
        "id":    map[string]string{"type": "string"},
        "name":  map[string]string{"type": "string"},
        "email": map[string]string{"type": "string", "format": "email"},
        "age":   map[string]string{"type": "integer", "format": "int32"},
        "roles": map[string]interface{}{
            "type": "array",
            "items": map[string]string{"type": "string"},
        },
    })).
    Response(404, "User not found", nil).
    Handle(func(c *gin.Context) {
        // ...
    })
```

## Вспомогательные функции для схем

Tonica предоставляет вспомогательные функции для создания OpenAPI схем:

### StringSchema

```go
tonica.StringSchema()  // {"type": "string"}
```

### InlineObjectSchema

Быстрый способ определения объекта:

```go
tonica.InlineObjectSchema(map[string]string{
    "name":   "string",
    "age":    "integer",
    "email":  "string",
    "active": "boolean",
})
```

Результат:
```json
{
  "type": "object",
  "properties": {
    "name": {"type": "string"},
    "age": {"type": "integer"},
    "email": {"type": "string"},
    "active": {"type": "boolean"}
  }
}
```

### ObjectSchema

Полный контроль над схемой:

```go
tonica.ObjectSchema(map[string]interface{}{
    "name": map[string]string{
        "type": "string",
        "minLength": "1",
        "maxLength": "100",
    },
    "age": map[string]string{
        "type": "integer",
        "minimum": "0",
        "maximum": "150",
    },
    "email": map[string]string{
        "type": "string",
        "format": "email",
    },
})
```

### ArraySchema

```go
// Массив строк
tonica.ArraySchema(map[string]string{"type": "string"})

// Массив объектов
tonica.ArraySchema(map[string]interface{}{
    "type": "object",
    "properties": map[string]interface{}{
        "id":   map[string]string{"type": "string"},
        "name": map[string]string{"type": "string"},
    },
})
```

### RefSchema

Ссылка на определение из вашей proto-сгенерированной спецификации:

```go
// Ссылка на #/definitions/User
tonica.RefSchema("User")
```

Это полезно, когда вы хотите повторно использовать схемы из ваших proto определений:

```go
tonica.NewRoute(app).
    POST("/users").
    BodyParam("User data", tonica.RefSchema("UserCreateRequest")).
    Response(200, "Created user", tonica.RefSchema("User")).
    Handle(func(c *gin.Context) {
        // ...
    })
```

## Безопасность

Добавление требований аутентификации к вашим маршрутам:

### Bearer аутентификация

```go
tonica.NewRoute(app).
    GET("/protected").
    Summary("Protected endpoint").
    Security("bearer").  // Требует Bearer токен
    Response(200, "Success", tonica.StringSchema()).
    Handle(func(c *gin.Context) {
        // Проверка заголовка authorization
        auth := c.GetHeader("Authorization")
        if auth == "" {
            c.JSON(401, gin.H{"error": "unauthorized"})
            return
        }
        // Валидация токена...
        c.JSON(200, gin.H{"message": "authorized"})
    })
```

### API ключ

```go
tonica.NewRoute(app).
    GET("/api-endpoint").
    Security("apiKey").
    Handle(func(c *gin.Context) {
        // Проверка API ключа из заголовка или query параметра
        apiKey := c.GetHeader("X-API-Key")
        // Валидация...
        c.JSON(200, gin.H{"data": "..."})
    })
```

### OAuth2 с областями доступа

```go
tonica.NewRoute(app).
    POST("/users/:id/delete").
    Summary("Delete user (requires admin access)").
    Security("oauth2", "admin", "users:delete").  // Требует определенные области доступа
    Handle(func(c *gin.Context) {
        // Проверка OAuth2 токена и областей доступа
        // ...
    })
```

**Примечание:** Определения безопасности должны быть определены в вашей OpenAPI спецификации или конфигурации основного приложения.

## Полные примеры

### RESTful CRUD API

```go
// List users
tonica.NewRoute(app).
    GET("/api/v1/users").
    Summary("List all users").
    Tags("Users", "v1").
    QueryParam("page", "integer", "Page number", false).
    QueryParam("limit", "integer", "Items per page", false).
    Response(200, "List of users", tonica.InlineObjectSchema(map[string]string{
        "users": "array",
        "total": "integer",
        "page":  "integer",
    })).
    Handle(listUsersHandler)

// Get user
tonica.NewRoute(app).
    GET("/api/v1/users/:id").
    Summary("Get user by ID").
    Tags("Users", "v1").
    PathParam("id", "string", "User ID").
    Response(200, "User information", tonica.RefSchema("User")).
    Response(404, "User not found", nil).
    Handle(getUserHandler)

// Create user
tonica.NewRoute(app).
    POST("/api/v1/users").
    Summary("Create new user").
    Tags("Users", "v1").
    BodyParam("User data", tonica.InlineObjectSchema(map[string]string{
        "name":  "string",
        "email": "string",
    })).
    Response(201, "User created", tonica.RefSchema("User")).
    Response(400, "Invalid input", nil).
    Handle(createUserHandler)

// Update user
tonica.NewRoute(app).
    PUT("/api/v1/users/:id").
    Summary("Update user").
    Tags("Users", "v1").
    PathParam("id", "string", "User ID").
    BodyParam("Updated data", tonica.RefSchema("UserUpdateRequest")).
    Response(200, "User updated", tonica.RefSchema("User")).
    Response(404, "User not found", nil).
    Handle(updateUserHandler)

// Delete user
tonica.NewRoute(app).
    DELETE("/api/v1/users/:id").
    Summary("Delete user").
    Tags("Users", "v1").
    Security("bearer").
    PathParam("id", "string", "User ID").
    Response(204, "User deleted", nil).
    Response(404, "User not found", nil).
    Handle(deleteUserHandler)
```

### Эндпоинты проверки работоспособности

```go
// Базовая проверка работоспособности
tonica.NewRoute(app).
    GET("/health").
    Summary("Basic health check").
    Tags("Monitoring").
    Response(200, "Service is healthy", tonica.InlineObjectSchema(map[string]string{
        "status": "string",
    })).
    Handle(func(c *gin.Context) {
        c.JSON(200, gin.H{"status": "healthy"})
    })

// Детальная проверка работоспособности
tonica.NewRoute(app).
    GET("/health/detailed").
    Summary("Detailed health check").
    Description("Returns health status of all service dependencies").
    Tags("Monitoring").
    Response(200, "Health status", tonica.InlineObjectSchema(map[string]string{
        "status":   "string",
        "database": "string",
        "redis":    "string",
        "temporal": "string",
    })).
    Handle(func(c *gin.Context) {
        health := gin.H{
            "status":   "healthy",
            "database": checkDatabase(),
            "redis":    checkRedis(),
            "temporal": checkTemporal(),
        }
        c.JSON(200, health)
    })

// Проверка готовности (Kubernetes)
tonica.NewRoute(app).
    GET("/ready").
    Summary("Readiness probe").
    Tags("Monitoring").
    Response(200, "Service is ready", nil).
    Response(503, "Service not ready", nil).
    Handle(func(c *gin.Context) {
        if !isReady() {
            c.Status(503)
            return
        }
        c.Status(200)
    })

// Проверка живучести (Kubernetes)
tonica.NewRoute(app).
    GET("/alive").
    Summary("Liveness probe").
    Tags("Monitoring").
    Response(200, "Service is alive", nil).
    Handle(func(c *gin.Context) {
        c.Status(200)
    })
```

### Эндпоинт вебхука

```go
tonica.NewRoute(app).
    POST("/webhooks/stripe").
    Summary("Stripe webhook handler").
    Description("Receives webhook events from Stripe payment processor").
    Tags("Webhooks", "Payments").
    HeaderParam("Stripe-Signature", "string", "Webhook signature", true).
    BodyParam("Webhook payload", tonica.ObjectSchema(map[string]interface{}{
        "id":   map[string]string{"type": "string"},
        "type": map[string]string{"type": "string"},
        "data": map[string]string{"type": "object"},
    })).
    Response(200, "Webhook processed", nil).
    Response(400, "Invalid signature", nil).
    Handle(func(c *gin.Context) {
        signature := c.GetHeader("Stripe-Signature")

        // Проверка подписи
        if !verifyStripeSignature(signature) {
            c.JSON(400, gin.H{"error": "invalid signature"})
            return
        }

        // Обработка вебхука
        var event map[string]interface{}
        if err := c.BindJSON(&event); err != nil {
            c.JSON(400, gin.H{"error": err.Error()})
            return
        }

        // Обработка события...
        c.Status(200)
    })
```

### Загрузка файлов

```go
tonica.NewRoute(app).
    POST("/upload").
    Summary("Upload file").
    Tags("Files").
    Security("bearer").
    Response(200, "File uploaded successfully", tonica.InlineObjectSchema(map[string]string{
        "fileId": "string",
        "url":    "string",
    })).
    Response(400, "Invalid file", nil).
    Handle(func(c *gin.Context) {
        file, err := c.FormFile("file")
        if err != nil {
            c.JSON(400, gin.H{"error": "no file provided"})
            return
        }

        // Сохранение файла...
        fileID := saveFile(file)

        c.JSON(200, gin.H{
            "fileId": fileID,
            "url":    "/files/" + fileID,
        })
    })
```

## Цепочка вызовов Fluent API

Все методы возвращают RouteBuilder, что позволяет вам создавать цепочки вызовов:

```go
tonica.NewRoute(app).
    GET("/users/:id/orders").
    Summary("Get user orders").
    Description("Returns all orders for a specific user with optional filtering").
    Tags("Users", "Orders").
    PathParam("id", "string", "User ID").
    QueryParam("status", "string", "Filter by order status", false).
    QueryParam("limit", "integer", "Max number of results", false).
    Security("bearer").
    Response(200, "List of orders", tonica.ArraySchema(tonica.RefSchema("Order"))).
    Response(404, "User not found", nil).
    Handle(func(c *gin.Context) {
        // Реализация обработчика
    })
```

## Лучшие практики

### 1. Всегда документируйте ваши маршруты

```go
// ❌ Плохо: Нет документации
tonica.NewRoute(app).
    GET("/data").
    Handle(handler)

// ✅ Хорошо: Хорошо задокументировано
tonica.NewRoute(app).
    GET("/data").
    Summary("Fetch data").
    Description("Retrieves aggregated data for the dashboard").
    Tags("Analytics").
    Response(200, "Data payload", schema).
    Handle(handler)
```

### 2. Используйте согласованные теги

```go
// Группируйте связанные эндпоинты
// Пользователи
tonica.NewRoute(app).GET("/users").Tags("Users")...
tonica.NewRoute(app).POST("/users").Tags("Users")...

// Заказы
tonica.NewRoute(app).GET("/orders").Tags("Orders")...
tonica.NewRoute(app).POST("/orders").Tags("Orders")...
```

### 3. Документируйте все ответы

```go
// Включайте успешные ответы и ответы с ошибками
tonica.NewRoute(app).
    POST("/users").
    Response(201, "Created", schema).
    Response(400, "Invalid input", errorSchema).
    Response(409, "Already exists", errorSchema).
    Response(500, "Server error", errorSchema).
    Handle(handler)
```

### 4. Переиспользуйте схемы

```go
// Определите один раз
errorSchema := tonica.InlineObjectSchema(map[string]string{
    "error":   "string",
    "code":    "string",
    "details": "string",
})

// Используйте везде
tonica.NewRoute(app).POST("/users").Response(400, "Error", errorSchema)...
tonica.NewRoute(app).POST("/orders").Response(400, "Error", errorSchema)...
```

### 5. Валидируйте входные данные

```go
tonica.NewRoute(app).
    POST("/users").
    BodyParam("User data", schema).
    Handle(func(c *gin.Context) {
        var user User
        if err := c.ShouldBindJSON(&user); err != nil {
            c.JSON(400, gin.H{"error": "invalid input"})
            return
        }

        // Валидация бизнес-правил
        if user.Email == "" {
            c.JSON(400, gin.H{"error": "email is required"})
            return
        }

        // Обработка...
    })
```

## Решение проблем

### Маршруты не появляются в OpenAPI спецификации

**Проблема:** Пользовательские маршруты не отображаются в `/openapi.json`

**Решение:**
1. Убедитесь, что вы вызываете `.Handle()` - это регистрирует маршрут
2. Проверьте, что `WithSpec()` правильно установлен при инициализации приложения
3. Проверьте логи приложения на наличие ошибок при объединении спецификаций

### Интерфейс Scalar не загружается

**Проблема:** `/docs` возвращает 404

**Решение:**
1. Убедитесь, что вы установили путь к спецификации: `WithSpec("path/to/spec.json")`
2. Проверьте, что файл спецификации существует по указанному пути
3. Проверьте, что HTTP сервер запущен на правильном порту

### Неправильная схема в документации

**Проблема:** Параметры или ответы показывают некорректные типы

**Решение:**
```go
// Используйте правильные строки типов
"string", "integer", "number", "boolean", "array", "object"

// Не используйте: "int", "str", "bool"
```

### Обработчик маршрута не вызывается

**Проблема:** Маршрут зарегистрирован, но обработчик не выполняется

**Решение:**
1. Проверьте, что путь совпадает точно (с учетом регистра)
2. Проверьте правильность HTTP метода
3. Проверьте, не блокирует ли middleware запрос
4. Поищите ошибки в консоли/логах

## Следующие шаги

- [Testing](./testing.md) - Тестирование ваших пользовательских маршрутов
- [Best Practices](./best-practices.md) - Паттерны проектирования API
- [Configuration](./configuration.md) - Настройка отдачи OpenAPI спецификации
