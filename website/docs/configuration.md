# Configuration

In Tonica, configuring your application is done in code using functional options. This approach gives you full control over how your application is assembled and started. The core idea is to read values from environment variables (or other sources like files) and pass them as options when creating the application instance and its services.

## Configuration Philosophy

Tonica follows the "explicit is better than implicit" principle. The framework does not automatically read environment variables. Instead, it provides convenient helpers like `config.GetEnv`, and you decide which environment variables to use and which options to pass them to.

**Priority Order:**

1.  **Code:** Hard-coded values or those passed into option functions have the highest priority.
2.  **Environment Variables:** Your logic in `main.go` reads environment variables and passes them into the code.
3.  **Defaults:** The `config.GetEnv` helper and some options have fallback values.

## Quick Start: Configuration Example

Here is what a typical configuration structure looks like in `main.go`:

```go
package main

import (
    "github.com/tonica-go/tonica/pkg/tonica"
    "github.com/tonica-go/tonica/pkg/tonica/config"
    "github.com/tonica-go/tonica/pkg/tonica/service"
    // ... imports for your services
)

func main() {
    // 1. Create the application configuration
    appConfig := config.NewConfig(
        // Set the run mode from the APP_MODE environment variable
        config.WithRunMode(config.GetEnv("APP_MODE", config.ModeAIO)),
    )

    // 2. Create the application instance
    app := tonica.NewApp(
        // Pass the created configuration
        tonica.WithConfig(appConfig),
        // Set the application name
        tonica.WithName(config.GetEnv("APP_NAME", "MyApp")),
    )

    // 3. Configure and register services
    
    // Read DSN and driver from environment variables
    dbDSN := config.GetEnv("DB_DSN", "user:pass@tcp(localhost:3306)/db?parseTime=true")
    dbDriver := config.GetEnv("DB_DRIVER", service.Mysql)

    // Read Redis address
    redisAddr := config.GetEnv("REDIS_ADDR", "localhost:6379")

    // Create a service with DB and Redis connections
    paymentSvc := service.NewService(
        service.WithName("payment-service"),
        service.WithDB(dbDSN, dbDriver),
        service.WithRedis(redisAddr, "", 0),
        service.WithGRPC(payment.RegisterGRPC), // your gRPC registrar
        service.WithGateway(payment.RegisterGateway), // your Gateway registrar
    )
    
    // Register the service with the application
    app.GetRegistry().MustRegisterService(paymentSvc)

    // 4. Run the application
    if err := app.Run(); err != nil {
        app.GetLogger().Fatal(err)
    }
}
```

## Application Configuration (`tonica.App`)

The main application instance is created using `tonica.NewApp(options ...AppOption)`.

### Core `AppOption` Options

| Option | Description | Example |
| --- | --- | --- |
| `WithName(string)` | Sets the application name. Used for logging and metrics. | `tonica.WithName("user-service")` |
| `WithConfig(*config.Config)` | Applies the startup configuration (run mode, list of services). **A very important option.** | `tonica.WithConfig(appConfig)` |
| `WithSpec(string)` | Specifies the path to the OpenAPI specification file. | `tonica.WithSpec("openapi/spec.json")` |
| `WithSpecUrl(string)` | Sets the URL where the specification will be available. | `tonica.WithSpecUrl("/swagger.json")` |
| `WithAPIPrefix(string)` | Adds a global prefix to all HTTP routes. | `tonica.WithAPIPrefix("/api/v1")` |
| `WithLogger(*log.Logger)` | Allows you to use a custom logger. | `tonica.WithLogger(myLogger)` |

### Startup Configuration (`config.Config`)

This configuration defines *how* your application will run. It is created using `config.NewConfig(options ...Option)`.

| Option | Description | Environment Variable | Example |
| --- | --- | --- | --- |
| `WithRunMode(string)` | Sets the application's run mode. | `APP_MODE` | `config.WithRunMode(config.ModeService)` |
| `WithServices([]string)` | In `service` mode, specifies which services to run. | `APP_SERVICES` | `config.WithServices([]string{"auth", "users"})` |
| `WithWorkers([]string)` | In `worker` mode, specifies which workers to run. | `APP_WORKERS` | `config.WithWorkers([]string{"emails", "reports"})` |
| `WithConsumers([]string)` | In `consumer` mode, specifies which consumers to run. | `APP_CONSUMERS` | `config.WithConsumers([]string{"orders"})` |
| `WithDebugMode(bool)` | Enables/disables debug mode. | `APP_DEBUG` | `config.WithDebugMode(true)` |

## Run Modes

The run mode is a key concept in Tonica that allows you to use the same codebase for different deployment types. The mode is set via the `config.WithRunMode` option and is usually controlled by the `APP_MODE` environment variable.

| Mode | `APP_MODE` | Description |
| --- | --- | --- |
| **All-In-One** | `aio` | **(Default)**. Runs all registered components (services, workers, consumers) in a single process. Ideal for development and simple deployments. |
| **Service** | `service` | Runs only the specified gRPC services and their HTTP gateways. Use `APP_SERVICES` (comma-separated) to specify which ones. |
| **Worker** | `worker` | Runs only the specified Temporal workers. Use `APP_WORKERS` to select them. |
| **Consumer** | `consumer` | Runs only the specified message consumers (e.g., Kafka). Use `APP_CONSUMERS` to select them. |
| **Gateway** | `gateway` | Runs only the HTTP gateways for all registered gRPC services, but not the gRPC servers themselves. Useful for deploying the API Gateway as a separate component. |

## Service Configuration (`service.Service`)

Each service in your application is created using `service.NewService(options ...Option)`.

### Core `service.Option` Options

| Option | Description | Example |
| --- | --- | --- |
| `WithName(string)` | **Required.** A unique name for the service. | `service.WithName("payment-service")` |
| `WithGRPC(GRPCRegistrar)` | **Required.** Registers your gRPC server implementation. | `service.WithGRPC(RegisterPaymentService)` |
| `WithGateway(GatewayRegistrar)` | Registers the HTTP gateway (gRPC-Gateway) for your service. | `service.WithGateway(RegisterPaymentGateway)` |
| `WithGRPCAddr(string)` | Sets the address for the gRPC server (`host:port`). | `service.WithGRPCAddr(":9001")` |

### Connecting to Databases & Caches

Connections are configured for each service individually.

#### Database

The `WithDB` option is used to connect to a database. Tonica supports PostgreSQL, MySQL, and SQLite out-of-the-box using the `bun` ORM and automatically integrates OpenTelemetry for query tracing.

```go
// Read DSN and driver from environment variables
dbDSN := config.GetEnv("DB_DSN", "user:pass@tcp(localhost:3306)/db?parseTime=true")
dbDriver := config.GetEnv("DB_DRIVER", service.Postgres) // service.Postgres, service.Mysql, service.Sqlite

svc := service.NewService(
    // ...other options
    service.WithDB(dbDSN, dbDriver),
)
```

| Driver | Constant | Example DSN |
| --- | --- | --- |
| PostgreSQL | `service.Postgres` | `postgres://user:pass@host:5432/db?sslmode=disable` |
| MySQL | `service.Mysql` | `user:pass@tcp(host:3306)/db?parseTime=true` |
| SQLite | `service.Sqlite` | `file:data.db?cache=shared` |

#### Redis

The `WithRedis` option is used to connect to Redis.

```go
redisAddr := config.GetEnv("REDIS_ADDR", "localhost:6379")
redisPassword := config.GetEnv("REDIS_PASSWORD", "")
redisDB := config.GetEnvInt("REDIS_DB", 0) // Use GetEnvInt for numeric values

svc := service.NewService(
    // ...other options
    service.WithRedis(redisAddr, redisPassword, redisDB),
)
```

## Environment Variables

Here is a summary of the most commonly used environment variables.

| Variable | Description | Default Value |
| --- | --- | --- |
| `APP_NAME` | The name of your application. | `"Tonica"` |
| `APP_MODE` | The application's run mode. | `"aio"` |
| `APP_SERVICES` | List of services to run in `service` mode. | `""` |
| `APP_WORKERS` | List of workers to run in `worker` mode. | `""` |
| `APP_CONSUMERS` | List of consumers to run in `consumer` mode. | `""` |
| `APP_PORT` | Port for the main HTTP server (gateways, custom routes). | `"8080"` |
| `GRPC_PORT` | Port for the gRPC server (if `WithGRPCAddr` is not set). | `"50051"` |
| `METRICS_PORT` | Port for the Prometheus metrics endpoint. | `"9090"` |
| `DB_DSN` | Data Source Name for the database connection. | `""` |
| `DB_DRIVER` | Database driver (`postgres`, `mysql`, `sqlite`). | `"postgres"` |
| `REDIS_ADDR` | Address of the Redis server (`host:port`). | `"localhost:6379"` |
| `REDIS_PASSWORD` | Password for Redis. | `""` |
| `REDIS_DB` | Redis database number. | `0` |
| `LOG_LEVEL` | Logging level (`debug`, `info`, `warn`, `error`). | `"info"` |
| `LOG_FORMAT` | Log format (`text` or `json`). | `"text"` |

## Observability

Tonica has built-in support for logging, metrics, and tracing.

### Logging

Logging is configured via environment variables:
- `LOG_LEVEL`: Set to `debug` for development and `info` for production.
- `LOG_FORMAT`: Set to `text` for local development and `json` for production to make logs easy to parse.

### Metrics

Prometheus-formatted metrics are available by default on the port specified by the `METRICS_PORT` variable (default `:9090`).

### Tracing (OpenTelemetry)

Tracing is enabled and configured via standard OpenTelemetry environment variables:

| Variable | Description | Example |
| --- | --- | --- |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | The address of the OpenTelemetry collector (e.g., Jaeger, Tempo). | `localhost:4317` |
| `OTEL_SERVICE_NAME` | The service name for tracing (usually matches `APP_NAME`). | `payment-service` |
| `OTEL_TRACES_EXPORTER` | Specify `otlp` to export traces. | `otlp` |
| `OTEL_EXPORTER_OTLP_PROTOCOL` | The exporter protocol (`grpc` or `http/protobuf`). | `grpc` |
| `OTEL_SDK_DISABLED` | Set to `true` to completely disable tracing. | `false` |