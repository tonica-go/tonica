# Custom Routes with OpenAPI Documentation

Tonica's fluent API allows you to add custom HTTP routes that are not defined in your Protocol Buffers. These routes are automatically documented in your OpenAPI specification and appear in the Scalar UI alongside your proto-generated endpoints.

## Why Custom Routes?

While proto-first is powerful, sometimes you need routes that don't fit the RPC model:

- Health checks (`/health`, `/ready`)
- Static file serving
- Legacy endpoints
- Internal utilities
- Webhooks
- Special auth endpoints

Custom routes let you add these without modifying your proto files.

## Quick Start

### Basic Example

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

This creates:
1. A GET endpoint at `/health`
2. OpenAPI documentation for the endpoint
3. Automatic appearance in Scalar UI at `/docs`

### Viewing Documentation

After starting your app:

- **OpenAPI Spec**: `http://localhost:8080/openapi.json`
- **Scalar UI**: `http://localhost:8080/docs`

## HTTP Methods

All standard HTTP methods are supported:

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

## Parameters

### Query Parameters

Query parameters are passed in the URL: `/endpoint?name=value`

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

**Required vs Optional:**
```go
// Required query parameter
.QueryParam("name", "string", "User name", true)

// Optional query parameter
.QueryParam("lang", "string", "Language code", false)
```

### Path Parameters

Path parameters are part of the URL: `/users/:id`

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

**Note:** Path parameters are always required.

### Header Parameters

Read values from HTTP headers:

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

### Body Parameters

For POST, PUT, PATCH requests:

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

## Documentation

### Summary and Description

```go
tonica.NewRoute(app).
    GET("/stats").
    Summary("Get statistics").  // Short one-line description
    Description(`
        Returns comprehensive statistics about the system including:
        - Total users
        - Active sessions
        - Request rates
        - Error rates

        This endpoint is cached for 5 minutes.
    `).  // Long detailed description
    Handle(func(c *gin.Context) {
        // ...
    })
```

### Tags

Tags group endpoints in the documentation UI:

```go
// Single tag
tonica.NewRoute(app).
    GET("/health").
    Tag("Monitoring").
    Handle(func(c *gin.Context) { /* ... */ })

// Multiple tags
tonica.NewRoute(app).
    GET("/users").
    Tags("Users", "Public", "v1").
    Handle(func(c *gin.Context) { /* ... */ })

// Adding tags one by one
tonica.NewRoute(app).
    GET("/orders").
    Tag("Orders").
    Tag("Commerce").
    Tag("v2").
    Handle(func(c *gin.Context) { /* ... */ })
```

**Result in Scalar UI:**
```
└─ Monitoring
   └─ GET /health

└─ Users
   └─ GET /users

└─ Orders
   └─ GET /orders
```

## Responses

### Simple Response

```go
tonica.NewRoute(app).
    GET("/ping").
    Response(200, "Pong response", tonica.StringSchema()).
    Handle(func(c *gin.Context) {
        c.String(200, "pong")
    })
```

### Multiple Responses

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

### Response with Complex Schema

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

## Schema Helpers

Tonica provides helper functions for creating OpenAPI schemas:

### StringSchema

```go
tonica.StringSchema()  // {"type": "string"}
```

### InlineObjectSchema

Quick way to define an object:

```go
tonica.InlineObjectSchema(map[string]string{
    "name":   "string",
    "age":    "integer",
    "email":  "string",
    "active": "boolean",
})
```

Result:
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

Full control over schema:

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
// Array of strings
tonica.ArraySchema(map[string]string{"type": "string"})

// Array of objects
tonica.ArraySchema(map[string]interface{}{
    "type": "object",
    "properties": map[string]interface{}{
        "id":   map[string]string{"type": "string"},
        "name": map[string]string{"type": "string"},
    },
})
```

### RefSchema

Reference a definition from your proto-generated spec:

```go
// Reference to #/definitions/User
tonica.RefSchema("User")
```

This is useful when you want to reuse schemas from your proto definitions:

```go
tonica.NewRoute(app).
    POST("/users").
    BodyParam("User data", tonica.RefSchema("UserCreateRequest")).
    Response(200, "Created user", tonica.RefSchema("User")).
    Handle(func(c *gin.Context) {
        // ...
    })
```

## Security

Add authentication requirements to your routes:

### Bearer Authentication

```go
tonica.NewRoute(app).
    GET("/protected").
    Summary("Protected endpoint").
    Security("bearer").  // Requires Bearer token
    Response(200, "Success", tonica.StringSchema()).
    Handle(func(c *gin.Context) {
        // Check authorization header
        auth := c.GetHeader("Authorization")
        if auth == "" {
            c.JSON(401, gin.H{"error": "unauthorized"})
            return
        }
        // Validate token...
        c.JSON(200, gin.H{"message": "authorized"})
    })
```

### API Key

```go
tonica.NewRoute(app).
    GET("/api-endpoint").
    Security("apiKey").
    Handle(func(c *gin.Context) {
        // Check API key from header or query param
        apiKey := c.GetHeader("X-API-Key")
        // Validate...
        c.JSON(200, gin.H{"data": "..."})
    })
```

### OAuth2 with Scopes

```go
tonica.NewRoute(app).
    POST("/users/:id/delete").
    Summary("Delete user (requires admin access)").
    Security("oauth2", "admin", "users:delete").  // Requires specific scopes
    Handle(func(c *gin.Context) {
        // Check OAuth2 token and scopes
        // ...
    })
```

**Note:** Security definitions must be defined in your OpenAPI spec or main app configuration.

## Complete Examples

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

### Health Check Endpoints

```go
// Basic health check
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

// Detailed health check
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

// Readiness check (Kubernetes)
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

// Liveness check (Kubernetes)
tonica.NewRoute(app).
    GET("/alive").
    Summary("Liveness probe").
    Tags("Monitoring").
    Response(200, "Service is alive", nil).
    Handle(func(c *gin.Context) {
        c.Status(200)
    })
```

### Webhook Endpoint

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

        // Verify signature
        if !verifyStripeSignature(signature) {
            c.JSON(400, gin.H{"error": "invalid signature"})
            return
        }

        // Process webhook
        var event map[string]interface{}
        if err := c.BindJSON(&event); err != nil {
            c.JSON(400, gin.H{"error": err.Error()})
            return
        }

        // Handle event...
        c.Status(200)
    })
```

### File Upload

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

        // Save file...
        fileID := saveFile(file)

        c.JSON(200, gin.H{
            "fileId": fileID,
            "url":    "/files/" + fileID,
        })
    })
```

## Fluent API Chaining

All methods return the RouteBuilder, allowing you to chain calls:

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
        // Handler implementation
    })
```

## Best Practices

### 1. Always Document Your Routes

```go
// ❌ Bad: No documentation
tonica.NewRoute(app).
    GET("/data").
    Handle(handler)

// ✅ Good: Well-documented
tonica.NewRoute(app).
    GET("/data").
    Summary("Fetch data").
    Description("Retrieves aggregated data for the dashboard").
    Tags("Analytics").
    Response(200, "Data payload", schema).
    Handle(handler)
```

### 2. Use Consistent Tags

```go
// Group related endpoints
// Users
tonica.NewRoute(app).GET("/users").Tags("Users")...
tonica.NewRoute(app).POST("/users").Tags("Users")...

// Orders
tonica.NewRoute(app).GET("/orders").Tags("Orders")...
tonica.NewRoute(app).POST("/orders").Tags("Orders")...
```

### 3. Document All Responses

```go
// Include success and error responses
tonica.NewRoute(app).
    POST("/users").
    Response(201, "Created", schema).
    Response(400, "Invalid input", errorSchema).
    Response(409, "Already exists", errorSchema).
    Response(500, "Server error", errorSchema).
    Handle(handler)
```

### 4. Reuse Schemas

```go
// Define once
errorSchema := tonica.InlineObjectSchema(map[string]string{
    "error":   "string",
    "code":    "string",
    "details": "string",
})

// Reuse everywhere
tonica.NewRoute(app).POST("/users").Response(400, "Error", errorSchema)...
tonica.NewRoute(app).POST("/orders").Response(400, "Error", errorSchema)...
```

### 5. Validate Input

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

        // Validate business rules
        if user.Email == "" {
            c.JSON(400, gin.H{"error": "email is required"})
            return
        }

        // Process...
    })
```

## Troubleshooting

### Routes Not Appearing in OpenAPI Spec

**Problem:** Custom routes don't show up in `/openapi.json`

**Solution:**
1. Make sure you call `.Handle()` - this registers the route
2. Verify `WithSpec()` is set correctly in app initialization
3. Check app logs for errors during spec merging

### Scalar UI Not Loading

**Problem:** `/docs` returns 404

**Solution:**
1. Ensure you've set a spec path: `WithSpec("path/to/spec.json")`
2. Check the spec file exists at the specified path
3. Verify HTTP server is running on correct port

### Wrong Schema in Documentation

**Problem:** Parameters or responses show incorrect types

**Solution:**
```go
// Use correct type strings
"string", "integer", "number", "boolean", "array", "object"

// Not: "int", "str", "bool"
```

### Route Handler Not Called

**Problem:** Route registered but handler doesn't execute

**Solution:**
1. Check the path matches exactly (case-sensitive)
2. Verify HTTP method is correct
3. Check if middleware is blocking the request
4. Look for errors in console/logs

## Next Steps

- [Testing](./testing.md) - Test your custom routes
- [Best Practices](./best-practices.md) - API design patterns
- [Configuration](./configuration.md) - Configure OpenAPI spec serving
