# Getting Started with Tonica

This guide will help you create your first Tonica application in just a few minutes.

## Prerequisites

- Go 1.25 or higher
- Protocol Buffers compiler (`protoc`)
- Buf CLI (recommended for proto management)
- Docker (optional, for running dependencies)

## Installation

```bash
go get github.com/tonica-go/tonica
```

## Quick Start: Hello World Service

Let's create a simple gRPC service with REST API in 5 minutes.

### Step 1: Initialize Your Project

```bash
mkdir myservice
cd myservice
go mod init github.com/yourusername/myservice
```

### Step 2: Define Your Proto

Create `proto/hello/v1/hello.proto`:

```protobuf
syntax = "proto3";

package hello.v1;

import "google/api/annotations.proto";

option go_package = "github.com/yourusername/myservice/proto/hello/v1;hellov1";

service HelloService {
  rpc SayHello(SayHelloRequest) returns (SayHelloResponse) {
    option (google.api.http) = {
      post: "/v1/hello"
      body: "*"
    };
  }
}

message SayHelloRequest {
  string name = 1;
}

message SayHelloResponse {
  string message = 1;
}
```

### Step 3: Generate Code

Create `buf.gen.yaml`:

```yaml
version: v1
managed:
  enabled: true
  go_package_prefix:
    default: github.com/yourusername/myservice
plugins:
  - plugin: buf.build/protocolbuffers/go
    out: .
    opt: paths=source_relative
  - plugin: buf.build/grpc/go
    out: .
    opt: paths=source_relative
  - plugin: buf.build/grpc-ecosystem/gateway
    out: .
    opt:
      - paths=source_relative
      - generate_unbound_methods=true
  - plugin: buf.build/grpc-ecosystem/openapiv2
    out: openapi
```

Generate code:

```bash
buf generate proto
```

### Step 4: Implement Your Service

Create `main.go`:

```go
package main

import (
    "context"
    "log"

    "github.com/tonica-go/tonica/pkg/tonica"
    hellov1 "github.com/yourusername/myservice/proto/hello/v1"
)

// HelloServiceImpl implements the HelloService
type HelloServiceImpl struct {
    hellov1.UnimplementedHelloServiceServer
}

func (s *HelloServiceImpl) SayHello(ctx context.Context, req *hellov1.SayHelloRequest) (*hellov1.SayHelloResponse, error) {
    return &hellov1.SayHelloResponse{
        Message: "Hello, " + req.Name + "!",
    }, nil
}

func main() {
    // Create Tonica app
    app := tonica.NewApp(
        tonica.WithName("hello-service"),
        tonica.WithSpec("openapi/hello/v1/hello.swagger.json"),
    )

    // Register your service
    svc := tonica.NewService(
        tonica.WithServiceName("HelloService"),
    )

    // Register gRPC service handler
    hellov1.RegisterHelloServiceServer(svc.GetGRPCServer(), &HelloServiceImpl{})

    // Register with app
    app.GetRegistry().MustRegisterService(svc)

    // Run in AIO mode (gRPC + REST)
    if err := app.Run(context.Background(), tonica.ModeAio); err != nil {
        log.Fatal(err)
    }
}
```

### Step 5: Run Your Service

```bash
go run main.go
```

Your service is now running! 🎉

- **gRPC**: `localhost:50051`
- **REST API**: `http://localhost:8080/v1/hello`
- **OpenAPI UI**: `http://localhost:8080/docs`
- **OpenAPI Spec**: `http://localhost:8080/openapi.json`
- **Metrics**: `http://localhost:9090/metrics`

### Step 6: Test Your Service

Test with REST:

```bash
curl -X POST http://localhost:8080/v1/hello \
  -H "Content-Type: application/json" \
  -d '{"name": "World"}'
```

Response:
```json
{
  "message": "Hello, World!"
}
```

Test with gRPC (using grpcurl):

```bash
grpcurl -plaintext -d '{"name": "World"}' \
  localhost:50051 hello.v1.HelloService/SayHello
```

## Next Steps

### Add Custom Routes

Add a custom health check endpoint:

```go
tonica.NewRoute(app).
    GET("/health").
    Summary("Health check endpoint").
    Tags("Monitoring").
    Response(200, "Service is healthy", tonica.InlineObjectSchema(map[string]string{
        "status": "string",
        "version": "string",
    })).
    Handle(func(c *gin.Context) {
        c.JSON(200, gin.H{
            "status": "healthy",
            "version": "1.0.0",
        })
    })
```

See [Custom Routes](./custom-routes.md) for more details.

### Add Database

Connect to PostgreSQL:

```go
db := tonica.NewDB(
    tonica.WithDriver(tonica.Postgres),
    tonica.WithDSN("postgres://user:pass@localhost:5432/mydb?sslmode=disable"),
)

// Use the database
client := db.GetClient()
```

See [Configuration](./configuration.md) for all database options.

### Add Redis Cache

```go
redis := tonica.NewRedis(
    tonica.WithRedisAddr("localhost:6379"),
    tonica.WithRedisPassword(""),
    tonica.WithRedisDB(0),
)

// Use Redis
cache := redis.GetClient()
cache.Set(ctx, "key", "value", 0)
```

### Add Temporal Workers

```go
// Create worker
worker := tonica.NewWorker(
    tonica.WithWorkerName("my-worker"),
    tonica.WithTaskQueue("my-task-queue"),
)

// Register activities
worker.GetWorker().RegisterActivity(MyActivity)

// Register with app
app.GetRegistry().MustRegisterWorker("my-worker", worker)

// Run in worker mode
app.Run(context.Background(), tonica.ModeWorker)
```

See [Run Modes](./run-modes.md) for more about different modes.

### Add Message Consumers

```go
// Create consumer
consumer := tonica.NewConsumer(
    tonica.WithConsumerName("order-consumer"),
    tonica.WithTopic("orders"),
    tonica.WithConsumerGroup("order-processors"),
    tonica.WithHandler(func(ctx context.Context, msg *pubsub.Message) error {
        // Process message
        log.Printf("Received order: %s", msg.Value)
        return nil
    }),
)

// Register with app
app.GetRegistry().MustRegisterConsumer(consumer)

// Run in consumer mode
app.Run(context.Background(), tonica.ModeConsumer)
```

## Configuration via Environment Variables

Tonica supports configuration through environment variables:

```bash
# App configuration
export APP_NAME="hello-service"
export APP_PORT="8080"
export GRPC_PORT="50051"
export METRICS_PORT="9090"

# Database
export DB_DRIVER="postgres"
export DB_DSN="postgres://user:pass@localhost/mydb"

# Redis
export REDIS_ADDR="localhost:6379"
export REDIS_PASSWORD=""
export REDIS_DB="0"

# Temporal
export TEMPORAL_HOST="localhost:7233"
export TEMPORAL_NAMESPACE="default"

# Observability
export OTEL_ENABLED="true"
export OTEL_ENDPOINT="localhost:4317"
```

See [Configuration](./configuration.md) for all available options.

## Project Structure

Recommended project structure:

```
myservice/
├── cmd/
│   └── server/
│       └── main.go          # Application entrypoint
├── internal/
│   ├── service/             # Service implementations
│   │   └── hello.go
│   ├── repository/          # Data access layer
│   │   └── user.go
│   └── models/              # Domain models
│       └── user.go
├── proto/                   # Protocol buffer definitions
│   └── hello/
│       └── v1/
│           └── hello.proto
├── openapi/                 # Generated OpenAPI specs
├── buf.gen.yaml            # Buf configuration
├── go.mod
└── go.sum
```

## Common Patterns

### Dependency Injection

```go
type UserService struct {
    db    *bun.DB
    redis *redis.Client
    logger *slog.Logger
}

func NewUserService(app *tonica.App) *UserService {
    return &UserService{
        db:     app.GetDB().GetClient(),
        redis:  app.GetRedis().GetClient(),
        logger: app.GetLogger(),
    }
}
```

### Error Handling

```go
func (s *HelloServiceImpl) SayHello(ctx context.Context, req *hellov1.SayHelloRequest) (*hellov1.SayHelloResponse, error) {
    if req.Name == "" {
        return nil, status.Error(codes.InvalidArgument, "name is required")
    }

    // Your logic here

    return &hellov1.SayHelloResponse{
        Message: "Hello, " + req.Name + "!",
    }, nil
}
```

### Logging

```go
app.GetLogger().Info("Processing request",
    "user_id", userID,
    "action", "create_order",
)
```

### Metrics

```go
counter := app.GetMetricManager().NewCounter(
    "orders_created_total",
    "Total number of orders created",
)

// Increment counter
counter.Inc()
```

## Troubleshooting

### Port Already in Use

If you see "address already in use" error, change the ports:

```bash
export APP_PORT="8081"
export GRPC_PORT="50052"
export METRICS_PORT="9091"
```

### Proto Generation Fails

Make sure you have all dependencies:

```bash
# Install buf
go install github.com/bufbuild/buf/cmd/buf@latest

# Install protoc-gen-go plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### Database Connection Fails

Check your DSN format:

- PostgreSQL: `postgres://user:pass@host:port/dbname?sslmode=disable`
- MySQL: `user:pass@tcp(host:port)/dbname?parseTime=true`
- SQLite: `file:path/to/database.db?cache=shared&mode=rwc`

## Next Topics

- [Architecture Overview](./architecture.md) - Understand Tonica's design
- [Run Modes](./run-modes.md) - Learn about different deployment modes
- [Configuration](./configuration.md) - Complete configuration reference
- [Testing](./testing.md) - Write tests for your services
- [Best Practices](./best-practices.md) - Production-ready patterns

## Examples

Check out complete working examples in the [example directory](../example/):

- Simple REST service
- gRPC service with database
- Worker with Temporal
- Consumer with Kafka
- All-in-one application

Happy coding! 🚀
