# Tonica Configuration Guide

This guide covers all configuration options available in Tonica, including environment variables, code-based configuration, and best practices.

## Configuration Priority

Tonica follows this priority order (highest to lowest):

1. **Code** - Options passed to constructors
2. **Environment Variables** - Environment-based configuration
3. **Defaults** - Built-in sensible defaults

Example:
```go
// Priority 1: Code (highest)
app := tonica.NewApp(tonica.WithName("my-service"))

// Priority 2: Environment variable
// export APP_NAME="env-service"

// Priority 3: Default (lowest)
// Falls back to "tonica-app"
```

## Application Configuration

### App Options

Configure the main application:

```go
app := tonica.NewApp(
    tonica.WithName("my-service"),              // Application name
    tonica.WithSpec("openapi/spec.json"),       // OpenAPI spec path
    tonica.WithSpecUrl("/swagger.json"),        // Custom spec URL
    tonica.WithConfig(customConfig),            // Custom config object
    tonica.WithLogger(customLogger),            // Custom logger
    tonica.WithRegistry(customRegistry),        // Custom registry
)
```

#### WithName

Sets the application name (used in metrics, logging, etc.).

```go
tonica.WithName("user-service")
```

**Environment Variable:**
```bash
export APP_NAME="user-service"
```

**Default:** `"tonica-app"`

#### WithSpec

Sets the path to the OpenAPI specification file.

```go
tonica.WithSpec("openapi/myservice/v1/myservice.swagger.json")
```

**Environment Variable:**
```bash
export OPENAPI_SPEC="openapi/myservice/v1/myservice.swagger.json"
```

**Default:** `""` (no spec loaded)

#### WithSpecUrl

Sets the URL path where the OpenAPI spec will be served.

```go
tonica.WithSpecUrl("/api-spec.json")
```

**Default:** `"/openapi.json"`

### Server Ports

Configure HTTP, gRPC, and metrics ports:

**Environment Variables:**
```bash
# HTTP/REST server port
export APP_PORT="8080"          # Default: 8080

# gRPC server port
export GRPC_PORT="50051"        # Default: 50051

# Metrics endpoint port
export METRICS_PORT="9090"      # Default: 9090
```

**Example:**
```bash
# Run on custom ports
export APP_PORT="3000"
export GRPC_PORT="9000"
export METRICS_PORT="9100"
```

### CORS Configuration

Configure Cross-Origin Resource Sharing:

**Environment Variables:**
```bash
# Allow all origins (default)
# No configuration needed

# Restrict to specific origins
export APP_CORS_ORIGINS="https://myapp.com,https://admin.myapp.com"
```

**Default:** Allows all origins

**Allowed Methods:** GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS

**Allowed Headers:** Origin, Content-Length, Content-Type, Authorization

## Service Configuration

Configure gRPC services:

```go
svc := tonica.NewService(
    tonica.WithServiceName("UserService"),      // Service name
    tonica.WithDB(db),                          // Database client
    tonica.WithRedis(redis),                    // Redis client
)
```

### WithServiceName

Sets the service name for registration.

```go
tonica.WithServiceName("UserService")
```

**Required:** Yes (for registration)

### WithDB

Attaches a database client to the service.

```go
db := tonica.NewDB(...)
svc := tonica.NewService(tonica.WithDB(db))
```

### WithRedis

Attaches a Redis client to the service.

```go
redis := tonica.NewRedis(...)
svc := tonica.NewService(tonica.WithRedis(redis))
```

## Database Configuration

Tonica supports PostgreSQL, MySQL, and SQLite via Bun ORM.

### PostgreSQL

```go
db := tonica.NewDB(
    tonica.WithDriver(tonica.Postgres),
    tonica.WithDSN("postgres://user:password@localhost:5432/dbname?sslmode=disable"),
)
```

**Environment Variables:**
```bash
export DB_DRIVER="postgres"
export DB_DSN="postgres://user:password@localhost:5432/dbname?sslmode=disable"
```

**DSN Format:**
```
postgres://username:password@host:port/database?sslmode=disable
```

**Options:**
- `sslmode`: `disable`, `require`, `verify-ca`, `verify-full`
- `connect_timeout`: Connection timeout in seconds
- `application_name`: Application name for logging

**Example:**
```bash
export DB_DSN="postgres://myuser:mypass@db.example.com:5432/mydb?sslmode=require&connect_timeout=10"
```

### MySQL

```go
db := tonica.NewDB(
    tonica.WithDriver(tonica.Mysql),
    tonica.WithDSN("user:password@tcp(localhost:3306)/dbname?parseTime=true"),
)
```

**Environment Variables:**
```bash
export DB_DRIVER="mysql"
export DB_DSN="user:password@tcp(localhost:3306)/dbname?parseTime=true"
```

**DSN Format:**
```
username:password@tcp(host:port)/database?parseTime=true
```

**Important Options:**
- `parseTime=true`: **Required** for proper time handling
- `charset=utf8mb4`: Character set (recommended)
- `loc=Local`: Timezone location

**Example:**
```bash
export DB_DSN="myuser:mypass@tcp(mysql.example.com:3306)/mydb?parseTime=true&charset=utf8mb4"
```

### SQLite

```go
db := tonica.NewDB(
    tonica.WithDriver(tonica.Sqlite),
    tonica.WithDSN("file:./data/mydb.db?cache=shared&mode=rwc"),
)
```

**Environment Variables:**
```bash
export DB_DRIVER="sqlite"
export DB_DSN="file:./data/mydb.db?cache=shared&mode=rwc"
```

**DSN Format:**
```
file:path/to/database.db?cache=shared&mode=rwc
```

**Options:**
- `cache`: `shared` (multi-connection) or `private`
- `mode`: `ro` (read-only), `rw` (read-write), `rwc` (read-write-create)

**Example:**
```bash
export DB_DSN="file:/var/lib/myapp/data.db?cache=shared&mode=rwc"
```

### Database Options

#### WithDriver

Sets the database driver.

```go
tonica.WithDriver(tonica.Postgres)  // PostgreSQL
tonica.WithDriver(tonica.Mysql)     // MySQL
tonica.WithDriver(tonica.Sqlite)    // SQLite
```

**Environment Variable:**
```bash
export DB_DRIVER="postgres"  # or "mysql", "sqlite"
```

**Default:** `"postgres"`

#### WithDSN

Sets the database connection string.

```go
tonica.WithDSN("postgres://localhost/mydb")
```

**Environment Variable:**
```bash
export DB_DSN="postgres://localhost/mydb"
```

**Required:** Yes (if using database)

### Connection Pooling

Configure connection pooling (via Bun):

```go
db := tonica.NewDB(...)
client := db.GetClient()

// Configure pool
client.SetMaxOpenConns(25)                        // Max open connections
client.SetMaxIdleConns(10)                        // Max idle connections
client.SetConnMaxLifetime(5 * time.Minute)        // Connection lifetime
client.SetConnMaxIdleTime(10 * time.Minute)       // Max idle time
```

**Recommended Settings:**

**For API services (high concurrency):**
```go
client.SetMaxOpenConns(100)
client.SetMaxIdleConns(25)
client.SetConnMaxLifetime(5 * time.Minute)
```

**For workers (low concurrency):**
```go
client.SetMaxOpenConns(10)
client.SetMaxIdleConns(5)
client.SetConnMaxLifetime(10 * time.Minute)
```

## Redis Configuration

Configure Redis connection:

```go
redis := tonica.NewRedis(
    tonica.WithRedisAddr("localhost:6379"),     // Redis address
    tonica.WithRedisPassword("secret"),         // Password (optional)
    tonica.WithRedisDB(0),                      // Database number
)
```

### Redis Options

#### WithRedisAddr

Sets the Redis server address.

```go
tonica.WithRedisAddr("redis.example.com:6379")
```

**Environment Variable:**
```bash
export REDIS_ADDR="redis.example.com:6379"
```

**Default:** `"localhost:6379"`

#### WithRedisPassword

Sets the Redis password (if required).

```go
tonica.WithRedisPassword("my-secret-password")
```

**Environment Variable:**
```bash
export REDIS_PASSWORD="my-secret-password"
```

**Default:** `""` (no password)

#### WithRedisDB

Sets the Redis database number (0-15).

```go
tonica.WithRedisDB(2)
```

**Environment Variable:**
```bash
export REDIS_DB="2"
```

**Default:** `0`

### Redis Connection Pooling

Configure connection pool (via go-redis):

```go
client := redis.GetClient()

// Configure pool options
client.Options().PoolSize = 10           // Pool size
client.Options().MinIdleConns = 5        // Min idle connections
client.Options().MaxConnAge = 0          // Max connection age (0 = no limit)
client.Options().PoolTimeout = 4 * time.Second
client.Options().IdleTimeout = 5 * time.Minute
```

## Temporal Configuration

Configure Temporal workers:

```go
worker := tonica.NewWorker(
    tonica.WithWorkerName("email-worker"),                    // Worker name
    tonica.WithTaskQueue("email-tasks"),                      // Task queue
    tonica.WithTemporalHost("temporal.example.com:7233"),     // Temporal host
    tonica.WithTemporalNamespace("production"),               // Namespace
    tonica.WithMaxConcurrentActivities(10),                   // Concurrency
)
```

### Temporal Options

#### WithWorkerName

Sets the worker name.

```go
tonica.WithWorkerName("report-generator")
```

**Required:** Yes

#### WithTaskQueue

Sets the Temporal task queue name.

```go
tonica.WithTaskQueue("report-tasks")
```

**Required:** Yes

#### WithTemporalHost

Sets the Temporal server address.

```go
tonica.WithTemporalHost("temporal.example.com:7233")
```

**Environment Variable:**
```bash
export TEMPORAL_HOST="temporal.example.com:7233"
```

**Default:** `"localhost:7233"`

#### WithTemporalNamespace

Sets the Temporal namespace.

```go
tonica.WithTemporalNamespace("production")
```

**Environment Variable:**
```bash
export TEMPORAL_NAMESPACE="production"
```

**Default:** `"default"`

#### WithMaxConcurrentActivities

Sets max concurrent activity executions.

```go
tonica.WithMaxConcurrentActivities(20)
```

**Default:** `10`

**Recommendations:**
- I/O-bound activities (emails, API calls): 20-100
- CPU-bound activities (image processing, reports): 2-5
- Memory-intensive activities: 1-3

## Consumer Configuration

Configure message consumers:

```go
consumer := tonica.NewConsumer(
    tonica.WithConsumerName("order-processor"),       // Consumer name
    tonica.WithTopic("orders"),                       // Topic name
    tonica.WithConsumerGroup("order-handlers"),       // Consumer group
    tonica.WithPubSubClient(client),                  // PubSub client
    tonica.WithHandler(handleOrder),                  // Message handler
)
```

### Consumer Options

#### WithConsumerName

Sets the consumer name.

```go
tonica.WithConsumerName("payment-processor")
```

**Required:** Yes

#### WithTopic

Sets the topic to consume from.

```go
tonica.WithTopic("payments")
```

**Required:** Yes

#### WithConsumerGroup

Sets the consumer group name.

```go
tonica.WithConsumerGroup("payment-handlers")
```

**Default:** `""` (no consumer group)

#### WithPubSubClient

Sets the PubSub/Kafka client.

```go
tonica.WithPubSubClient(pubsubClient)
```

**Required:** Yes

#### WithHandler

Sets the message handler function.

```go
tonica.WithHandler(func(ctx context.Context, msg *pubsub.Message) error {
    // Process message
    return nil
})
```

**Required:** Yes

## Observability Configuration

### OpenTelemetry

Configure distributed tracing:

**Environment Variables:**
```bash
# Enable/disable tracing
export OTEL_ENABLED="true"          # Default: false

# OTLP endpoint
export OTEL_ENDPOINT="localhost:4317"

# Service name (overrides APP_NAME)
export OTEL_SERVICE_NAME="user-service"

# Sampling rate (0.0 to 1.0)
export OTEL_TRACE_SAMPLING="1.0"    # 100% sampling
```

**Example (Jaeger):**
```bash
export OTEL_ENABLED="true"
export OTEL_ENDPOINT="jaeger:4317"
export OTEL_SERVICE_NAME="user-service"
```

**Example (Honeycomb):**
```bash
export OTEL_ENABLED="true"
export OTEL_ENDPOINT="api.honeycomb.io:443"
export OTEL_SERVICE_NAME="user-service"
export OTEL_HEADERS="x-honeycomb-team=YOUR_API_KEY"
```

### Logging

Configure structured logging:

**Log Levels:**
```bash
# Set log level
export LOG_LEVEL="debug"     # debug, info, warn, error
```

**Log Format:**
```bash
# JSON format (for production)
export LOG_FORMAT="json"

# Text format (for development)
export LOG_FORMAT="text"
```

**Example:**
```bash
# Development
export LOG_LEVEL="debug"
export LOG_FORMAT="text"

# Production
export LOG_LEVEL="info"
export LOG_FORMAT="json"
```

### Metrics

Metrics are always enabled on port 9090 by default.

**Customize port:**
```bash
export METRICS_PORT="9100"
```

**Disable metrics:**
```go
// Not recommended - metrics are lightweight
// To disable, don't expose port 9090 externally
```

## Complete Configuration Examples

### Development Environment

```bash
# Application
export APP_NAME="myservice-dev"
export APP_PORT="8080"
export GRPC_PORT="50051"
export METRICS_PORT="9090"

# Database
export DB_DRIVER="sqlite"
export DB_DSN="file:./dev.db?cache=shared&mode=rwc"

# Redis
export REDIS_ADDR="localhost:6379"
export REDIS_PASSWORD=""
export REDIS_DB="0"

# Temporal
export TEMPORAL_HOST="localhost:7233"
export TEMPORAL_NAMESPACE="default"

# Logging
export LOG_LEVEL="debug"
export LOG_FORMAT="text"

# Tracing (disabled)
export OTEL_ENABLED="false"
```

### Production Environment

```bash
# Application
export APP_NAME="myservice"
export APP_PORT="8080"
export GRPC_PORT="50051"
export METRICS_PORT="9090"

# Database
export DB_DRIVER="postgres"
export DB_DSN="postgres://user:pass@postgres.prod:5432/mydb?sslmode=require"

# Redis
export REDIS_ADDR="redis.prod:6379"
export REDIS_PASSWORD="${REDIS_SECRET}"
export REDIS_DB="0"

# Temporal
export TEMPORAL_HOST="temporal.prod:7233"
export TEMPORAL_NAMESPACE="production"

# CORS
export APP_CORS_ORIGINS="https://myapp.com,https://admin.myapp.com"

# Logging
export LOG_LEVEL="info"
export LOG_FORMAT="json"

# Tracing
export OTEL_ENABLED="true"
export OTEL_ENDPOINT="tempo.prod:4317"
export OTEL_SERVICE_NAME="myservice"
export OTEL_TRACE_SAMPLING="0.1"  # 10% sampling
```

### Docker Compose

```yaml
version: '3.8'

services:
  app:
    image: myservice:latest
    environment:
      # Application
      APP_NAME: myservice
      APP_PORT: 8080
      GRPC_PORT: 50051
      METRICS_PORT: 9090

      # Database
      DB_DRIVER: postgres
      DB_DSN: postgres://myuser:mypass@postgres:5432/mydb?sslmode=disable

      # Redis
      REDIS_ADDR: redis:6379
      REDIS_PASSWORD: ""
      REDIS_DB: 0

      # Temporal
      TEMPORAL_HOST: temporal:7233
      TEMPORAL_NAMESPACE: default

      # Observability
      LOG_LEVEL: info
      LOG_FORMAT: json
      OTEL_ENABLED: "true"
      OTEL_ENDPOINT: jaeger:4317

    ports:
      - "8080:8080"    # HTTP
      - "50051:50051"  # gRPC
      - "9090:9090"    # Metrics

    depends_on:
      - postgres
      - redis
      - temporal

  postgres:
    image: postgres:15
    environment:
      POSTGRES_USER: myuser
      POSTGRES_PASSWORD: mypass
      POSTGRES_DB: mydb
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine

  temporal:
    image: temporalio/auto-setup:latest

volumes:
  postgres_data:
```

### Kubernetes ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: myservice-config
data:
  APP_NAME: "myservice"
  APP_PORT: "8080"
  GRPC_PORT: "50051"
  METRICS_PORT: "9090"

  DB_DRIVER: "postgres"

  REDIS_DB: "0"

  TEMPORAL_NAMESPACE: "production"

  LOG_LEVEL: "info"
  LOG_FORMAT: "json"

  OTEL_ENABLED: "true"
  OTEL_TRACE_SAMPLING: "0.1"

---
apiVersion: v1
kind: Secret
metadata:
  name: myservice-secrets
type: Opaque
stringData:
  DB_DSN: "postgres://user:pass@postgres:5432/db"
  REDIS_ADDR: "redis:6379"
  REDIS_PASSWORD: "secret"
  TEMPORAL_HOST: "temporal:7233"

---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myservice
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: app
        image: myservice:latest
        envFrom:
        - configMapRef:
            name: myservice-config
        - secretRef:
            name: myservice-secrets
```

## Configuration Best Practices

### 1. Never Hardcode Secrets

❌ **Bad:**
```go
db := tonica.NewDB(
    tonica.WithDSN("postgres://admin:password123@localhost/db"),
)
```

✅ **Good:**
```go
dsn := os.Getenv("DB_DSN")
if dsn == "" {
    log.Fatal("DB_DSN is required")
}
db := tonica.NewDB(tonica.WithDSN(dsn))
```

### 2. Validate Configuration

```go
func validateConfig() error {
    if os.Getenv("DB_DSN") == "" {
        return errors.New("DB_DSN is required")
    }
    if os.Getenv("REDIS_ADDR") == "" {
        return errors.New("REDIS_ADDR is required")
    }
    return nil
}

func main() {
    if err := validateConfig(); err != nil {
        log.Fatal(err)
    }
    // ...
}
```

### 3. Use Environment-Specific Files

```bash
# .env.development
APP_NAME=myservice-dev
DB_DRIVER=sqlite
DB_DSN=file:./dev.db

# .env.production
APP_NAME=myservice
DB_DRIVER=postgres
DB_DSN=postgres://...
```

### 4. Document Required Variables

Create a `.env.example`:

```bash
# Application
APP_NAME=myservice
APP_PORT=8080
GRPC_PORT=50051

# Database (required)
DB_DRIVER=postgres
DB_DSN=postgres://user:pass@host:5432/db

# Redis (optional)
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# Temporal (required for workers)
TEMPORAL_HOST=localhost:7233
TEMPORAL_NAMESPACE=default
```

### 5. Use Defaults Wisely

```go
func getEnvOrDefault(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}

appPort := getEnvOrDefault("APP_PORT", "8080")
grpcPort := getEnvOrDefault("GRPC_PORT", "50051")
```

## Troubleshooting

### Database Connection Issues

**Error:** `connection refused`
```bash
# Check if database is running
docker ps | grep postgres

# Test connection
psql postgres://user:pass@localhost:5432/db
```

**Error:** `authentication failed`
```bash
# Verify credentials
echo $DB_DSN

# Check PostgreSQL logs
docker logs postgres-container
```

### Redis Connection Issues

**Error:** `dial tcp: connection refused`
```bash
# Check if Redis is running
docker ps | grep redis

# Test connection
redis-cli -h localhost -p 6379 ping
```

### Port Conflicts

**Error:** `bind: address already in use`
```bash
# Find what's using the port
lsof -i :8080

# Use different port
export APP_PORT="8081"
```

## Next Steps

- [Custom Routes](./custom-routes.md) - Add custom HTTP routes
- [Testing](./testing.md) - Test your configuration
- [Best Practices](./best-practices.md) - Production configuration patterns
