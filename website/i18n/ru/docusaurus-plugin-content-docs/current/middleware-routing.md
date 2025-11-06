# Middleware Routing Guide

Это руководство объясняет, как применять разные middleware к разным маршрутам в Tonica gateway.

## Проблема

При использовании gRPC gateway с proto аннотациями, все HTTP маршруты регистрируются через один wildcard route. Это означает, что все middleware применяются глобально ко всем маршрутам.

## Техническое решение

Tonica использует комбинацию **Gin Route Groups** для специфичных префиксов и **NoRoute** handler для всех остальных маршрутов.

`NoRoute()` - это специальный handler в Gin, который вызывается только когда ни один другой маршрут не совпал. Это позволяет избежать конфликтов между wildcard маршрутами и конкретными путями (например, `/openapi.json`, `/docs`).

**Порядок обработки запросов:**
1. Специальные маршруты (`/openapi.json`, `/docs`, `/healthz`, etc.)
2. Route groups с middleware для специфичных префиксов (`/api`, `/public`, etc.)
3. NoRoute handler - catch-all для всех остальных маршрутов

Часто требуется:
- Публичные маршруты без аутентификации (`/public/*`)
- API маршруты с JWT аутентификацией (`/api/v1/*`)
- Внутренние маршруты с API key аутентификацией (`/internal/*`)
- Админские маршруты с дополнительной проверкой ролей (`/admin/*`)

## Решение 1: Route Groups (Рекомендуется)

Самый чистый и производительный подход - использовать Gin route groups с разными middleware.

### Преимущества:
- ✅ Чистый и понятный код
- ✅ Хорошая производительность (Gin routing)
- ✅ Легко тестировать
- ✅ Декларативная конфигурация

### Недостатки:
- ❌ Порядок регистрации маршрутов важен
- ❌ Более сложная конфигурация для динамических правил

### Использование:

```go
app := tonica.NewApp(
    tonica.WithName("my-app"),

    // Публичные маршруты - без аутентификации
    tonica.WithRouteMiddleware(
        []string{"/public", "/health"},
        loggingMiddleware(),
    ),

    // API v1 - JWT аутентификация
    tonica.WithRouteMiddleware(
        []string{"/api/v1"},
        jwtAuthMiddleware(),
        identity.Middleware(identity.JWTExtractor("jwt_claims", "user_id", "email", "role")),
        rateLimitMiddleware(),
    ),

    // Internal API - API key
    tonica.WithRouteMiddleware(
        []string{"/internal"},
        apiKeyAuthMiddleware(),
    ),

    // Admin - JWT + роль admin
    tonica.WithRouteMiddleware(
        []string{"/admin"},
        jwtAuthMiddleware(),
        identity.Middleware(identity.JWTExtractor("jwt_claims", "user_id", "email", "role")),
        adminOnlyMiddleware(),
    ),
)
```

### Proto файл пример:

```protobuf
service UserService {
  // Публичный маршрут - без auth
  rpc GetPublicInfo(GetPublicInfoRequest) returns (GetPublicInfoResponse) {
    option (google.api.http) = {
      get: "/public/info"
    };
  }

  // API v1 - требует JWT
  rpc GetUser(GetUserRequest) returns (GetUserResponse) {
    option (google.api.http) = {
      get: "/api/v1/users/{id}"
    };
  }

  // Internal - требует API key
  rpc InternalSync(InternalSyncRequest) returns (InternalSyncResponse) {
    option (google.api.http) = {
      post: "/internal/sync"
    };
  }

  // Admin - требует JWT + admin роль
  rpc DeleteUser(DeleteUserRequest) returns (DeleteUserResponse) {
    option (google.api.http) = {
      delete: "/admin/users/{id}"
    };
  }
}
```

## Решение 2: Conditional Middleware

Альтернативный подход - один middleware, который проверяет путь и применяет нужную логику.

### Преимущества:
- ✅ Вся логика в одном месте
- ✅ Легко добавлять динамические правила
- ✅ Проще для простых кейсов

### Недостатки:
- ❌ Хуже производительность (проверка всех правил на каждый запрос)
- ❌ Сложнее тестировать
- ❌ Императивный подход

### Использование:

```go
// В app.go или отдельном файле
func setupConditionalMiddleware() gin.HandlerFunc {
    cm := tonica.NewConditionalMiddleware()

    // Публичные маршруты
    cm.AddRule(
        []string{"/public", "/health"},
        loggingMiddleware(),
    )

    // API v1
    cm.AddRule(
        []string{"/api/v1"},
        jwtAuthMiddleware(),
        identityMiddleware(),
        rateLimitMiddleware(),
    )

    // Internal
    cm.AddRule(
        []string{"/internal"},
        apiKeyAuthMiddleware(),
    )

    // Admin
    cm.AddRule(
        []string{"/admin"},
        jwtAuthMiddleware(),
        identityMiddleware(),
        adminOnlyMiddleware(),
    )

    return cm.Handler()
}

// Затем в registerAPI:
router.Use(setupConditionalMiddleware())
router.Any("/*any", WrapH(a.registerGateway(ctx)))
```

## Решение 3: Path-specific Middleware в Handler

Самый простой подход для простых случаев - проверка пути внутри middleware.

### Использование:

```go
func smartAuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        path := c.Request.URL.Path

        // Публичные маршруты - пропускаем
        if strings.HasPrefix(path, "/public") || strings.HasPrefix(path, "/health") {
            c.Next()
            return
        }

        // Internal - проверяем API key
        if strings.HasPrefix(path, "/internal") {
            apiKey := c.GetHeader("X-API-Key")
            if apiKey == "" {
                c.AbortWithStatusJSON(401, gin.H{"error": "api key required"})
                return
            }
            // validate API key...
            c.Next()
            return
        }

        // Все остальные - требуют JWT
        token := c.GetHeader("Authorization")
        if token == "" {
            c.AbortWithStatusJSON(401, gin.H{"error": "auth required"})
            return
        }
        // validate JWT...

        // Admin маршруты - дополнительная проверка роли
        if strings.HasPrefix(path, "/admin") {
            // check admin role...
        }

        c.Next()
    }
}

// В registerAPI:
router.Use(smartAuthMiddleware())
router.Any("/*any", WrapH(a.registerGateway(ctx)))
```

## Примеры Middleware

### JWT Authentication

```go
func jwtAuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        if token == "" {
            c.AbortWithStatusJSON(http.StatusUnauthorized,
                gin.H{"error": "missing authorization header"})
            return
        }

        // Parse and validate JWT
        claims, err := validateJWT(token)
        if err != nil {
            c.AbortWithStatusJSON(http.StatusUnauthorized,
                gin.H{"error": "invalid token"})
            return
        }

        c.Set("jwt_claims", claims)
        c.Next()
    }
}
```

### API Key Authentication

```go
func apiKeyAuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        apiKey := c.GetHeader("X-API-Key")
        if apiKey == "" {
            c.AbortWithStatusJSON(http.StatusUnauthorized,
                gin.H{"error": "missing api key"})
            return
        }

        // Validate API key
        if !isValidAPIKey(apiKey) {
            c.AbortWithStatusJSON(http.StatusUnauthorized,
                gin.H{"error": "invalid api key"})
            return
        }

        c.Next()
    }
}
```

### Role Check Middleware

```go
func requireRole(role string) gin.HandlerFunc {
    return func(c *gin.Context) {
        identity, exists := c.Get("identity")
        if !exists {
            c.AbortWithStatusJSON(http.StatusForbidden,
                gin.H{"error": "identity not found"})
            return
        }

        id, ok := identity.(identity.Identity)
        if !ok || id.GetRole() != role {
            c.AbortWithStatusJSON(http.StatusForbidden,
                gin.H{"error": fmt.Sprintf("%s access required", role)})
            return
        }

        c.Next()
    }
}

// Использование:
tonica.WithRouteMiddleware(
    []string{"/admin"},
    jwtAuthMiddleware(),
    identity.Middleware(identity.JWTExtractor(...)),
    requireRole("admin"),
)
```

## Рекомендации

1. **Для production приложений** - используйте Route Groups (Решение 1)
2. **Для простых случаев** - используйте Path-specific Middleware (Решение 3)
3. **Для динамических правил** - используйте Conditional Middleware (Решение 2)

## Порядок выполнения

При использовании Route Groups, middleware выполняются в следующем порядке:

1. Глобальные middleware (из `router.Use()`)
2. Route group middleware (из `WithRouteMiddleware()`)
3. Handler (gateway)

Пример:
```go
router.Use(globalLogging(), globalCORS())  // 1. Выполнятся всегда

tonica.WithRouteMiddleware(
    []string{"/api/v1"},
    jwtAuth(),           // 2. Только для /api/v1/*
    rateLimit(),         // 3. Только для /api/v1/*
)

// 4. Gateway handler
```

## Полный пример

См. `examples/router_groups/middleware_routes_example.go` для полного рабочего примера.
