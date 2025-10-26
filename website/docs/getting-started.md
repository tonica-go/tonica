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

    "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
    "github.com/tonica-go/tonica/pkg/tonica"
    "github.com/tonica-go/tonica/pkg/tonica/config"
    "github.com/tonica-go/tonica/pkg/tonica/service"
    hellov1 "github.com/yourusername/myservice/proto/hello/v1"
    "google.golang.org/grpc"
)

// HelloServiceImpl implements the HelloService
type HelloServiceImpl struct {
    hellov1.UnimplementedHelloServiceServer
    srv *service.Service
}

func (s *HelloServiceImpl) SayHello(ctx context.Context, req *hellov1.SayHelloRequest) (*hellov1.SayHelloResponse, error) {
    return &hellov1.SayHelloResponse{
        Message: "Hello, " + req.Name + "!",
    }, nil
}

// RegisterGRPC registers the gRPC service
func RegisterGRPC(s *grpc.Server, srv *service.Service) {
    hellov1.RegisterHelloServiceServer(s, &HelloServiceImpl{srv: srv})
}

// RegisterGateway registers the HTTP gateway
func RegisterGateway(ctx context.Context, mux *runtime.ServeMux, target string, dialOpts []grpc.DialOption) error {
    return hellov1.RegisterHelloServiceHandlerFromEndpoint(ctx, mux, target, dialOpts)
}

func main() {
    // Create Tonica app
    app := tonica.NewApp(
        tonica.WithSpec("openapi/hello/v1/hello.swagger.json"),
        tonica.WithConfig(
            config.NewConfig(
                config.WithRunMode(config.ModeAIO),
            ),
        ),
    )

    // Create and register your service
    svc := service.NewService(
        service.WithName("HelloService"),
        service.WithGRPC(RegisterGRPC),
        service.WithGateway(RegisterGateway),
        service.WithGRPCAddr(":9000"),
    )
    app.GetRegistry().MustRegisterService(svc)

    // Run the application
    if err := app.Run(); err != nil {
        app.GetLogger().Fatal(err)
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

Connect to PostgreSQL by adding it to your service:

```go
import "github.com/tonica-go/tonica/pkg/tonica/service"

svc := service.NewService(
    service.WithName("HelloService"),
    service.WithGRPC(RegisterGRPC),
    service.WithGateway(RegisterGateway),
    service.WithGRPCAddr(":9000"),
    service.WithDB(
        "postgres://user:pass@localhost:5432/mydb?sslmode=disable",
        service.Postgres, // or service.Mysql, service.Sqlite
    ),
)

// Access database in your service implementation
func RegisterGRPC(s *grpc.Server, srv *service.Service) {
    impl := &HelloServiceImpl{
        srv: srv,
        db:  srv.GetDB().GetClient(), // *bun.DB
    }
    hellov1.RegisterHelloServiceServer(s, impl)
}
```

See [Configuration](./configuration.md) for all database options.

### Add Redis Cache

Add Redis to your service:

```go
svc := service.NewService(
    service.WithName("HelloService"),
    service.WithGRPC(RegisterGRPC),
    service.WithGateway(RegisterGateway),
    service.WithGRPCAddr(":9000"),
    service.WithRedis("localhost:6379", "", 0), // addr, password, db
)

// Access Redis in your service implementation
func RegisterGRPC(s *grpc.Server, srv *service.Service) {
    impl := &HelloServiceImpl{
        srv:   srv,
        redis: srv.GetRedis().GetClient(), // *redis.Client
    }
    hellov1.RegisterHelloServiceServer(s, impl)
}
```

### Add Temporal Workers

```go
import (
    "github.com/tonica-go/tonica/pkg/tonica"
    "github.com/tonica-go/tonica/pkg/tonica/config"
    "github.com/tonica-go/tonica/pkg/tonica/worker"
    "go.temporal.io/sdk/client"
)

// Create Temporal client
temporalClient, err := client.Dial(client.Options{
    HostPort: "localhost:7233",
})
if err != nil {
    log.Fatal(err)
}

// Create worker
w := worker.NewWorker(
    worker.WithName("my-worker"),
    worker.WithQueue("my-task-queue"),
    worker.WithClient(temporalClient),
    worker.WithActivities([]interface{}{MyActivity}),
    worker.WithWorkflows([]interface{}{MyWorkflow}),
)

// Create app and register worker
app := tonica.NewApp(
    tonica.WithConfig(
        config.NewConfig(
            config.WithRunMode(config.ModeWorker),
        ),
    ),
)
app.GetRegistry().MustRegisterWorker(w)

// Run in worker mode
err = app.Run()
```

See [Run Modes](./run-modes.md) for more about different modes.

### Add Message Consumers

```go
import (
    "context"

    "github.com/tonica-go/tonica/pkg/tonica"
    "github.com/tonica-go/tonica/pkg/tonica/config"
    "github.com/tonica-go/tonica/pkg/tonica/consumer"
    "github.com/tonica-go/tonica/pkg/tonica/storage/pubsub"
    "github.com/tonica-go/tonica/pkg/tonica/storage/pubsub/kafka"
)

// Create Kafka client
kafkaClient := kafka.New(&kafka.Config{
    Brokers:         []string{"localhost:9092"},
    ConsumerGroupID: "order-processors",
}, nil)

// Create consumer
c := consumer.NewConsumer(
    consumer.WithName("order-consumer"),
    consumer.WithTopic("orders"),
    consumer.WithConsumerGroup("order-processors"),
    consumer.WithClient(kafkaClient),
    consumer.WithHandler(func(ctx context.Context, msg *pubsub.Message) error {
        // Process message
        log.Printf("Received order: %s", msg.Value)
        return nil
    }),
)

// Create app and register consumer
app := tonica.NewApp(
    tonica.WithConfig(
        config.NewConfig(
            config.WithRunMode(config.ModeConsumer),
        ),
    ),
)
app.GetRegistry().MustRegisterConsumer(c)

// Run in consumer mode
err := app.Run()
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
import (
    "log/slog"
    "github.com/redis/go-redis/v9"
    "github.com/uptrace/bun"
    "github.com/tonica-go/tonica/pkg/tonica/service"
)

type UserService struct {
    db     *bun.DB
    redis  *redis.Client
    logger *slog.Logger
}

// Dependencies are injected from the service
func NewUserService(srv *service.Service) *UserService {
    return &UserService{
        db:     srv.GetDB().GetClient(),
        redis:  srv.GetRedis().GetClient(),
        logger: slog.Default(), // or pass logger separately
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
import "log/slog"

// Use slog for structured logging
slog.Info("Processing request",
    "user_id", userID,
    "action", "create_order",
)

// Or use app's logger (standard log.Logger)
app.GetLogger().Printf("Processing request for user %s", userID)
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

Check out complete working examples on [GitHub](https://github.com/tonica-go/tonica/tree/main/example):

- Simple REST service
- gRPC service with database
- Worker with Temporal
- Consumer with Kafka
- All-in-one application

Happy coding! 🚀
