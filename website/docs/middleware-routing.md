# Middleware Routing Guide

This guide explains how to apply different middleware to different routes in the Tonica gateway.

## Problem

When using a gRPC gateway with proto annotations, all HTTP routes are registered through a single wildcard route. That means all middleware would be applied globally to every route.

## Technical solution

Tonica uses a combination of **Gin Route Groups** for specific prefixes and a **NoRoute** handler for all other routes.

`NoRoute()` is a special handler in Gin that is called only when no other route matched. This avoids conflicts between the wildcard route and specific paths (for example `/openapi.json`, `/docs`).

**Request handling order:**
1. Special routes (`/openapi.json`, `/docs`, `/healthz`, etc.)
2. Route groups with middleware for specific prefixes (`/api`, `/public`, etc.)
3. NoRoute handler — catch-all for all remaining routes

Common requirements:
- Public routes without authentication (`/public/*`)
- API routes with JWT authentication (`/api/v1/*`)
- Internal routes with API key authentication (`/internal/*`)
- Admin routes with additional role checks (`/admin/*`)

## Solution 1: Route Groups (Recommended)

The cleanest and most performant approach is to use Gin route groups with different middleware.

### Advantages:
- ✅ Clean and readable code
- ✅ Good performance (Gin routing)
- ✅ Easy to test
- ✅ Declarative configuration

### Disadvantages:
- ❌ The order of route registration matters
- ❌ More complex configuration for dynamic rules

### Usage:

```go
app := tonica.NewApp(
    tonica.WithName("my-app"),

    // Public routes - no authentication
    tonica.WithRouteMiddleware(
        []string{"/public", "/health"},
        loggingMiddleware(),
    ),

    // API v1 - JWT authentication
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

    // Admin - JWT + admin role
    tonica.WithRouteMiddleware(
        []string{"/admin"},
        jwtAuthMiddleware(),
        identity.Middleware(identity.JWTExtractor("jwt_claims", "user_id", "email", "role")),
        adminOnlyMiddleware(),
    ),
)
```

### Example proto file:

```protobuf
service UserService {
  // Public route - no auth
  rpc GetPublicInfo(GetPublicInfoRequest) returns (GetPublicInfoResponse) {
    option (google.api.http) = {
      get: "/public/info"
    };
  }

  // API v1 - requires JWT
  rpc GetUser(GetUserRequest) returns (GetUserResponse) {
    option (google.api.http) = {
      get: "/api/v1/users/{id}"
    };
  }

  // Internal - requires API key
  rpc InternalSync(InternalSyncRequest) returns (InternalSyncResponse) {
    option (google.api.http) = {
      post: "/internal/sync"
    };
  }

  // Admin - requires JWT + admin role
  rpc DeleteUser(DeleteUserRequest) returns (DeleteUserResponse) {
    option (google.api.http) = {
      delete: "/admin/users/{id}"
    };
  }
}
```

## Solution 2: Conditional Middleware

An alternative approach is a single middleware that checks the request path and applies the necessary logic.

### Advantages:
- ✅ All logic is in one place
- ✅ Easy to add dynamic rules
- ✅ Simpler for small use cases

### Disadvantages:
- ❌ Worse performance (checks all rules on every request)
- ❌ Harder to test
- ❌ Imperative style

### Usage:

```go
// In app.go or a separate file
func setupConditionalMiddleware() gin.HandlerFunc {
    cm := tonica.NewConditionalMiddleware()

    // Public routes
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

// Then in registerAPI:
router.Use(setupConditionalMiddleware())
router.Any("/*any", WrapH(a.registerGateway(ctx)))
```

## Solution 3: Path-specific Middleware in Handler

The simplest approach for straightforward cases is to check the path inside a middleware.

### Usage:

```go
func smartAuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        path := c.Request.URL.Path

        // Public routes - skip
        if strings.HasPrefix(path, "/public") || strings.HasPrefix(path, "/health") {
            c.Next()
            return
        }

        // Internal - check API key
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

        // All others - require JWT
        token := c.GetHeader("Authorization")
        if token == "" {
            c.AbortWithStatusJSON(401, gin.H{"error": "auth required"})
            return
        }
        // validate JWT...

        // Admin routes - additional role check
        if strings.HasPrefix(path, "/admin") {
            // check admin role...
        }

        c.Next()
    }
}

// In registerAPI:
router.Use(smartAuthMiddleware())
router.Any("/*any", WrapH(a.registerGateway(ctx)))
```

## Middleware examples

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

// Usage:
tonica.WithRouteMiddleware(
    []string{"/admin"},
    jwtAuthMiddleware(),
    identity.Middleware(identity.JWTExtractor(...)),
    requireRole("admin"),
)
```

## Recommendations

1. **For production applications** - use Route Groups (Solution 1)
2. **For simple cases** - use Path-specific Middleware (Solution 3)
3. **For dynamic rules** - use Conditional Middleware (Solution 2)

## Execution order

When using Route Groups, middleware run in the following order:

1. Global middleware (from `router.Use()`)
2. Route group middleware (from `WithRouteMiddleware()`)
3. Handler (gateway)

Example:
```go
router.Use(globalLogging(), globalCORS())  // 1. Always executed

tonica.WithRouteMiddleware(
    []string{"/api/v1"},
    jwtAuth(),           // 2. Only for /api/v1/*
    rateLimit(),         // 3. Only for /api/v1/*
)

// 4. Gateway handler
```

## Full example

See `examples/router_groups/middleware_routes_example.go` for a complete working example.
