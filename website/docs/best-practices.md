# Best Practices for Tonica Applications

This guide covers production-ready patterns, anti-patterns, and recommendations for building robust applications with Tonica.

## Table of Contents

- [Application Structure](#application-structure)
- [Error Handling](#error-handling)
- [Logging](#logging)
- [Configuration Management](#configuration-management)
- [Database](#database)
- [API Design](#api-design)
- [Performance](#performance)
- [Security](#security)
- [Observability](#observability)
- [Deployment](#deployment)
- [Testing](#testing)

## Application Structure

### Recommended Project Layout

```
myservice/
├── cmd/
│   └── server/
│       └── main.go                 # Application entrypoint
├── internal/                       # Private application code
│   ├── domain/                     # Domain models
│   │   └── user.go
│   ├── repository/                 # Data access layer
│   │   ├── user_repository.go
│   │   └── user_repository_test.go
│   ├── service/                    # Business logic
│   │   ├── user_service.go
│   │   └── user_service_test.go
│   └── handler/                    # RPC handlers
│       ├── user_handler.go
│       └── user_handler_test.go
├── pkg/                            # Public reusable code
│   └── validator/
│       └── email.go
├── proto/                          # Protocol buffer definitions
│   └── user/
│       └── v1/
│           └── user.proto
├── openapi/                        # Generated OpenAPI specs
├── migrations/                     # Database migrations
│   ├── 001_create_users.sql
│   └── 002_add_users_email_index.sql
├── tests/
│   ├── integration/
│   └── e2e/
├── configs/                        # Configuration files
│   ├── dev.env
│   └── prod.env
├── scripts/                        # Build and deployment scripts
├── .env.example                    # Example environment variables
├── buf.gen.yaml                    # Buf configuration
├── docker-compose.yml
├── Dockerfile
├── Makefile
├── go.mod
└── go.sum
```

### Package Organization

✅ **Good: Clear separation of concerns**
```go
// internal/repository/user_repository.go
type UserRepository interface {
    Create(ctx context.Context, user *domain.User) error
    GetByID(ctx context.Context, id string) (*domain.User, error)
}

// internal/service/user_service.go
type UserService struct {
    repo UserRepository
    logger *slog.Logger
}

// internal/handler/user_handler.go
type UserHandler struct {
    service *UserService
}
```

❌ **Bad: Everything in one package**
```go
// main.go (5000 lines of mixed concerns)
```

### Dependency Injection

✅ **Good: Constructor injection**
```go
import "log/slog"

type UserService struct {
    repo   UserRepository
    cache  *redis.Client
    logger *slog.Logger  // slog.Logger from standard library
}

func NewUserService(repo UserRepository, cache *redis.Client, logger *slog.Logger) *UserService {
    return &UserService{
        repo:   repo,
        cache:  cache,
        logger: logger,
    }
}
```

❌ **Bad: Global variables**
```go
var globalDB *bun.DB
var globalCache *redis.Client

func GetUser(id string) (*User, error) {
    // Uses globals
    return globalDB.Query(...)
}
```

## Error Handling

### gRPC Errors

✅ **Good: Use proper status codes**
```go
import (
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

func (h *UserHandler) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
    if req.Id == "" {
        return nil, status.Error(codes.InvalidArgument, "user ID is required")
    }

    user, err := h.service.GetUser(ctx, req.Id)
    if err != nil {
        if errors.Is(err, repository.ErrNotFound) {
            return nil, status.Error(codes.NotFound, "user not found")
        }
        h.logger.Error("failed to get user", "error", err, "id", req.Id)
        return nil, status.Error(codes.Internal, "internal server error")
    }

    return &pb.GetUserResponse{User: user}, nil
}
```

❌ **Bad: Generic errors**
```go
func (h *UserHandler) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
    user, err := h.service.GetUser(ctx, req.Id)
    if err != nil {
        return nil, err  // ❌ No context, wrong code
    }
    return &pb.GetUserResponse{User: user}, nil
}
```

### Error Types

✅ **Good: Define custom errors**
```go
package repository

import "errors"

var (
    ErrNotFound      = errors.New("entity not found")
    ErrAlreadyExists = errors.New("entity already exists")
    ErrInvalidInput  = errors.New("invalid input")
)

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
    var user User
    err := r.db.NewSelect().
        Model(&user).
        Where("email = ?", email).
        Scan(ctx)

    if err == sql.ErrNoRows {
        return nil, ErrNotFound
    }
    return &user, err
}
```

### Error Wrapping

✅ **Good: Add context to errors**
```go
func (s *UserService) CreateUser(ctx context.Context, user *User) error {
    if err := s.validate(user); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }

    if err := s.repo.Create(ctx, user); err != nil {
        return fmt.Errorf("failed to create user in database: %w", err)
    }

    return nil
}
```

## Logging

### Structured Logging

✅ **Good: Use structured fields**
```go
logger.Info("user created",
    "user_id", user.ID,
    "email", user.Email,
    "duration_ms", elapsed.Milliseconds(),
)
```

❌ **Bad: String formatting**
```go
logger.Info(fmt.Sprintf("User %s created with email %s", user.ID, user.Email))
```

### Log Levels

```go
// DEBUG - Detailed information for debugging
logger.Debug("cache miss", "key", key)

// INFO - General informational messages
logger.Info("user logged in", "user_id", userID)

// WARN - Warning messages (recoverable issues)
logger.Warn("rate limit approaching", "user_id", userID, "requests", count)

// ERROR - Error messages (something failed)
logger.Error("failed to send email", "error", err, "recipient", email)
```

### What to Log

✅ **Log:**
- Request/response metadata (IDs, duration, status)
- Business events (user created, order placed)
- Errors with context
- Performance metrics
- Security events

❌ **Don't log:**
- Passwords or secrets
- Personal data (in production)
- Full request/response bodies (unless debugging)
- Excessive debug info in production

### Logging Example

```go
import (
    "context"
    "log/slog"
    "time"
)

func (s *UserService) CreateUser(ctx context.Context, user *User) error {
    start := time.Now()

    // slog uses key-value pairs for structured logging
    s.logger.Info("creating user",
        "email", user.Email,
    )

    if err := s.repo.Create(ctx, user); err != nil {
        s.logger.Error("failed to create user",
            "error", err,
            "email", user.Email,
            "duration_ms", time.Since(start).Milliseconds(),
        )
        return err
    }

    s.logger.Info("user created successfully",
        "user_id", user.ID,
        "email", user.Email,
        "duration_ms", time.Since(start).Milliseconds(),
    )

    return nil
}
```

## Configuration Management

### Environment Variables

✅ **Good: Validate at startup**
```go
func loadConfig() (*Config, error) {
    cfg := &Config{
        DBHost:     os.Getenv("DB_HOST"),
        DBPort:     os.Getenv("DB_PORT"),
        DBName:     os.Getenv("DB_NAME"),
        RedisAddr:  os.Getenv("REDIS_ADDR"),
    }

    if err := cfg.Validate(); err != nil {
        return nil, fmt.Errorf("invalid configuration: %w", err)
    }

    return cfg, nil
}

func (c *Config) Validate() error {
    if c.DBHost == "" {
        return errors.New("DB_HOST is required")
    }
    if c.DBPort == "" {
        return errors.New("DB_PORT is required")
    }
    return nil
}

func main() {
    cfg, err := loadConfig()
    if err != nil {
        log.Fatal(err)
    }
    // ...
}
```

### Secrets Management

❌ **Bad: Hardcoded secrets**
```go
redis := tonica.NewRedis(
    tonica.WithRedisPassword("hardcoded-password"),  // ❌
)
```

✅ **Good: Environment variables or secrets manager**
```go
// From environment
password := os.Getenv("REDIS_PASSWORD")
redis := tonica.NewRedis(tonica.WithRedisPassword(password))

// Or from secrets manager (AWS, GCP, etc.)
secret, err := secretsManager.GetSecret("redis-password")
redis := tonica.NewRedis(tonica.WithRedisPassword(secret))
```

## Database

### Connection Management

✅ **Good: Configure connection pool**
```go
db := service.NewDB(
    service.WithDriver(service.Postgres),
    service.WithDSN(dsn),
)

client := db.GetClient()

// Configure pool for API service
client.SetMaxOpenConns(25)
client.SetMaxIdleConns(10)
client.SetConnMaxLifetime(5 * time.Minute)
client.SetConnMaxIdleTime(10 * time.Minute)
```

### Query Patterns

✅ **Good: Use context and timeouts**
```go
func (r *UserRepository) GetUser(ctx context.Context, id string) (*User, error) {
    // Add timeout to context
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    var user User
    err := r.db.NewSelect().
        Model(&user).
        Where("id = ?", id).
        Scan(ctx)  // Use context

    return &user, err
}
```

✅ **Good: Use prepared statements (built into Bun)**
```go
// Bun automatically uses prepared statements
err := db.NewSelect().
    Model(&user).
    Where("email = ?", email).  // Safe from SQL injection
    Scan(ctx)
```

❌ **Bad: String concatenation**
```go
query := fmt.Sprintf("SELECT * FROM users WHERE email = '%s'", email)  // ❌ SQL injection
```

### Transactions

✅ **Good: Use transactions for multi-step operations**
```go
func (r *UserRepository) CreateUserWithProfile(ctx context.Context, user *User, profile *Profile) error {
    return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
        // Create user
        if _, err := tx.NewInsert().Model(user).Exec(ctx); err != nil {
            return err
        }

        // Create profile
        profile.UserID = user.ID
        if _, err := tx.NewInsert().Model(profile).Exec(ctx); err != nil {
            return err
        }

        return nil
    })
}
```

### Migrations

✅ **Good: Version-controlled migrations**
```sql
-- migrations/001_create_users.sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
```

Use migration tools like:
- [golang-migrate](https://github.com/golang-migrate/migrate)
- [goose](https://github.com/pressly/goose)
- Bun migrations

## API Design

### RESTful Principles

✅ **Good: Resource-based URLs**
```
GET    /api/v1/users           # List users
GET    /api/v1/users/:id       # Get user
POST   /api/v1/users           # Create user
PUT    /api/v1/users/:id       # Update user
DELETE /api/v1/users/:id       # Delete user

GET    /api/v1/users/:id/orders  # Get user's orders
```

❌ **Bad: Action-based URLs**
```
POST   /api/v1/getUser
POST   /api/v1/createUser
POST   /api/v1/deleteUser
```

### API Versioning

✅ **Good: Version in path**
```
/api/v1/users
/api/v2/users
```

### Input Validation

✅ **Good: Validate early**
```go
func (h *UserHandler) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
    // Validate input
    if req.Email == "" {
        return nil, status.Error(codes.InvalidArgument, "email is required")
    }
    if !isValidEmail(req.Email) {
        return nil, status.Error(codes.InvalidArgument, "invalid email format")
    }
    if req.Name == "" {
        return nil, status.Error(codes.InvalidArgument, "name is required")
    }

    // Process request
    user, err := h.service.CreateUser(ctx, req)
    // ...
}
```

### Pagination

✅ **Good: Always paginate list endpoints**
```go
message ListUsersRequest {
    int32 page = 1;    // Page number (default: 1)
    int32 limit = 2;   // Items per page (default: 10, max: 100)
}

message ListUsersResponse {
    repeated User users = 1;
    int32 total = 2;
    int32 page = 3;
    int32 limit = 4;
}
```

## Performance

### Caching Strategy

✅ **Good: Cache expensive operations**
```go
func (s *UserService) GetUser(ctx context.Context, id string) (*User, error) {
    // Try cache first
    cacheKey := "user:" + id
    if cached, err := s.cache.Get(ctx, cacheKey).Bytes(); err == nil {
        var user User
        if err := json.Unmarshal(cached, &user); err == nil {
            return &user, nil
        }
    }

    // Cache miss - fetch from database
    user, err := s.repo.GetUser(ctx, id)
    if err != nil {
        return nil, err
    }

    // Update cache
    if data, err := json.Marshal(user); err == nil {
        s.cache.Set(ctx, cacheKey, data, 10*time.Minute)
    }

    return user, nil
}
```

### N+1 Query Problem

❌ **Bad: N+1 queries**
```go
users, _ := repo.GetUsers(ctx)
for _, user := range users {
    orders, _ := repo.GetUserOrders(ctx, user.ID)  // ❌ Query in loop
    // ...
}
```

✅ **Good: Eager loading**
```go
var users []User
err := db.NewSelect().
    Model(&users).
    Relation("Orders").  // Load orders in single query
    Scan(ctx)
```

### Database Indexes

✅ **Good: Index frequently queried columns**
```sql
-- Index on email for lookups
CREATE INDEX idx_users_email ON users(email);

-- Composite index for multi-column queries
CREATE INDEX idx_orders_user_status ON orders(user_id, status);

-- Partial index for specific conditions
CREATE INDEX idx_active_users ON users(id) WHERE active = true;
```

### Connection Pooling

✅ **Good: Tune for your workload**
```go
// API service (high concurrency)
db.SetMaxOpenConns(100)
db.SetMaxIdleConns(25)

// Worker (low concurrency, CPU-bound)
db.SetMaxOpenConns(10)
db.SetMaxIdleConns(5)
```

## Security

### Authentication

✅ **Good: Validate tokens**
```go
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        if token == "" {
            c.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
            return
        }

        // Validate token
        claims, err := validateJWT(token)
        if err != nil {
            c.AbortWithStatusJSON(401, gin.H{"error": "invalid token"})
            return
        }

        // Store user info in context
        c.Set("user_id", claims.UserID)
        c.Next()
    }
}
```

### Input Sanitization

✅ **Good: Sanitize and validate**
```go
func (s *UserService) CreateUser(ctx context.Context, email, name string) error {
    // Validate
    email = strings.TrimSpace(strings.ToLower(email))
    name = strings.TrimSpace(name)

    if !isValidEmail(email) {
        return ErrInvalidEmail
    }

    if len(name) < 2 || len(name) > 100 {
        return ErrInvalidName
    }

    // Proceed...
}
```

### Rate Limiting

✅ **Good: Implement rate limiting**
```go
func RateLimitMiddleware(limiter *rate.Limiter) gin.HandlerFunc {
    return func(c *gin.Context) {
        if !limiter.Allow() {
            c.AbortWithStatusJSON(429, gin.H{
                "error": "rate limit exceeded",
            })
            return
        }
        c.Next()
    }
}

// Usage
limiter := rate.NewLimiter(rate.Every(time.Second), 10)  // 10 req/sec
app.GetRouter().Use(RateLimitMiddleware(limiter))
```

### HTTPS Only

✅ **Good: Enforce HTTPS in production**
```go
// In production, use TLS
if os.Getenv("ENV") == "production" {
    router.Use(func(c *gin.Context) {
        if c.Request.Header.Get("X-Forwarded-Proto") != "https" {
            c.Redirect(301, "https://"+c.Request.Host+c.Request.RequestURI)
            return
        }
        c.Next()
    })
}
```

## Observability

### Metrics

✅ **Good: Instrument critical paths**
```go
var (
    requestsTotal = app.GetMetricManager().NewCounter(
        "http_requests_total",
        "Total HTTP requests",
    )
    requestDuration = app.GetMetricManager().NewHistogram(
        "http_request_duration_seconds",
        "HTTP request duration",
    )
)

func (h *UserHandler) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
    start := time.Now()
    requestsTotal.Inc()

    user, err := h.service.CreateUser(ctx, req)

    duration := time.Since(start).Seconds()
    requestDuration.Observe(duration)

    if err != nil {
        return nil, err
    }
    return &pb.CreateUserResponse{User: user}, nil
}
```

### Tracing

✅ **Good: Add custom spans**
```go
import "go.opentelemetry.io/otel"

func (s *UserService) CreateUser(ctx context.Context, user *User) error {
    tracer := otel.Tracer("user-service")
    ctx, span := tracer.Start(ctx, "CreateUser")
    defer span.End()

    // Add attributes
    span.SetAttributes(
        attribute.String("user.email", user.Email),
    )

    // Your logic
    if err := s.repo.Create(ctx, user); err != nil {
        span.RecordError(err)
        return err
    }

    return nil
}
```

### Health Checks

✅ **Good: Comprehensive health checks**
```go
tonica.NewRoute(app).
    GET("/health").
    Handle(func(c *gin.Context) {
        health := gin.H{
            "status": "healthy",
        }

        // Check database
        if err := db.PingContext(c.Request.Context()); err != nil {
            health["database"] = "unhealthy"
            health["status"] = "unhealthy"
            c.JSON(503, health)
            return
        }
        health["database"] = "healthy"

        // Check Redis
        if err := redis.Ping(c.Request.Context()).Err(); err != nil {
            health["redis"] = "unhealthy"
            health["status"] = "degraded"
        } else {
            health["redis"] = "healthy"
        }

        statusCode := 200
        if health["status"] == "unhealthy" {
            statusCode = 503
        }
        c.JSON(statusCode, health)
    })
```

## Deployment

### Docker Best Practices

✅ **Good: Multi-stage build**
```dockerfile
# Builder stage
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /myservice ./cmd/server

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /myservice .
COPY openapi/ ./openapi/

EXPOSE 8080 50051 9090
CMD ["./myservice"]
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myservice
spec:
  replicas: 3
  selector:
    matchLabels:
      app: myservice
  template:
    metadata:
      labels:
        app: myservice
    spec:
      containers:
      - name: myservice
        image: myservice:latest
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 50051
          name: grpc
        - containerPort: 9090
          name: metrics

        # Health checks
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10

        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5

        # Resources
        resources:
          requests:
            cpu: "500m"
            memory: "512Mi"
          limits:
            cpu: "1000m"
            memory: "1Gi"

        # Environment
        envFrom:
        - configMapRef:
            name: myservice-config
        - secretRef:
            name: myservice-secrets
```

### Graceful Shutdown

✅ **Good: Handle signals properly**
```go
func main() {
    app := tonica.NewApp()

    // Run app (signal handling is automatic)
    if err := app.Run(); err != nil {
        app.GetLogger().Fatal(err)
    }

    log.Println("Application shutdown complete")
}
```

## Testing

See [Testing Guide](./testing.md) for comprehensive testing best practices.

### Key Testing Principles

✅ **Do:**
- Write tests before fixing bugs
- Test error cases and edge cases
- Use table-driven tests
- Mock external dependencies
- Run tests in CI/CD

❌ **Don't:**
- Test implementation details
- Write flaky tests
- Skip integration tests
- Ignore test failures
- Have slow tests in unit test suite

## Common Anti-Patterns

### 1. God Objects

❌ **Bad:**
```go
type Manager struct {
    // Does everything
}

func (m *Manager) CreateUser() {}
func (m *Manager) SendEmail() {}
func (m *Manager) ProcessPayment() {}
func (m *Manager) GenerateReport() {}
```

✅ **Good:** Single responsibility
```go
type UserService struct { /* user-related */ }
type EmailService struct { /* email-related */ }
type PaymentService struct { /* payment-related */ }
```

### 2. Premature Optimization

❌ **Bad:** Optimizing before measuring
```go
// Complex caching without knowing if it's needed
```

✅ **Good:** Measure first, optimize based on data
```go
// Profile, identify bottleneck, then optimize
```

### 3. No Error Handling

❌ **Bad:**
```go
user, _ := repo.GetUser(ctx, id)  // Ignoring error
```

✅ **Good:**
```go
user, err := repo.GetUser(ctx, id)
if err != nil {
    return nil, fmt.Errorf("failed to get user: %w", err)
}
```

## Checklist for Production

Before deploying to production:

- [ ] All secrets in environment variables or secrets manager
- [ ] Database connection pooling configured
- [ ] Proper error handling and logging
- [ ] Health check endpoints implemented
- [ ] Metrics and tracing enabled
- [ ] Rate limiting on public endpoints
- [ ] Input validation on all endpoints
- [ ] TLS/HTTPS enabled
- [ ] Database migrations tested
- [ ] Integration tests passing
- [ ] Load testing performed
- [ ] Graceful shutdown implemented
- [ ] Resource limits configured (CPU, memory)
- [ ] Monitoring and alerting set up
- [ ] Documentation updated

## Next Steps

- [Getting Started](./getting-started.md) - Build your first app
- [Architecture](./architecture.md) - Understand the framework
- [Testing](./testing.md) - Write comprehensive tests
- [Configuration](./configuration.md) - Configure for production
