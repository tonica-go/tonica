# Начало работы с Tonica

Это руководство поможет вам создать первое приложение на Tonica за несколько минут.

## Требования

- Go 1.25 или выше
- Компилятор Protocol Buffers (`protoc`)
- Buf CLI (рекомендуется для управления proto файлами)
- Docker (опционально, для запуска зависимостей)

## Установка

```bash
go get github.com/tonica-go/tonica
```

## Быстрый старт: Hello World сервис

Давайте создадим простой gRPC сервис с REST API за 5 минут.

### Шаг 1: Инициализация проекта

```bash
mkdir myservice
cd myservice
go mod init github.com/yourusername/myservice
```

### Шаг 2: Определение Proto файла

Создайте `proto/hello/v1/hello.proto`:

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

### Шаг 3: Генерация кода

Создайте `buf.gen.yaml`:

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

Сгенерируйте код:

```bash
buf generate proto
```

### Шаг 4: Реализация сервиса

Создайте `main.go`:

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

### Шаг 5: Запуск сервиса

```bash
go run main.go
```

Ваш сервис запущен!

- **gRPC**: `localhost:50051`
- **REST API**: `http://localhost:8080/v1/hello`
- **OpenAPI UI**: `http://localhost:8080/docs`
- **OpenAPI Spec**: `http://localhost:8080/openapi.json`
- **Метрики**: `http://localhost:9090/metrics`

### Шаг 6: Тестирование сервиса

Тестирование через REST:

```bash
curl -X POST http://localhost:8080/v1/hello \
  -H "Content-Type: application/json" \
  -d '{"name": "World"}'
```

Ответ:
```json
{
  "message": "Hello, World!"
}
```

Тестирование через gRPC (используя grpcurl):

```bash
grpcurl -plaintext -d '{"name": "World"}' \
  localhost:50051 hello.v1.HelloService/SayHello
```

## Следующие шаги

### Добавление пользовательских маршрутов

Добавьте пользовательский endpoint для проверки здоровья:

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

Подробнее см. [Пользовательские маршруты](./custom-routes.md).

### Добавление базы данных

Подключите PostgreSQL, добавив его в ваш сервис:

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

Подробнее обо всех опциях базы данных см. [Конфигурация](./configuration.md).

### Добавление Redis кэша

Добавьте Redis в ваш сервис:

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

### Добавление Temporal воркеров

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

Подробнее о различных режимах см. [Режимы запуска](./run-modes.md).

### Добавление обработчиков сообщений

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

## Конфигурация через переменные окружения

Tonica поддерживает конфигурацию через переменные окружения:

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

Подробнее обо всех доступных опциях см. [Конфигурация](./configuration.md).

## Структура проекта

Рекомендуемая структура проекта:

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

## Общие паттерны

### Внедрение зависимостей

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

### Обработка ошибок

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

### Логирование

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

### Метрики

```go
counter := app.GetMetricManager().NewCounter(
    "orders_created_total",
    "Total number of orders created",
)

// Increment counter
counter.Inc()
```

## Устранение неполадок

### Порт уже используется

Если вы видите ошибку "address already in use", измените порты:

```bash
export APP_PORT="8081"
export GRPC_PORT="50052"
export METRICS_PORT="9091"
```

### Ошибка генерации Proto

Убедитесь, что у вас установлены все зависимости:

```bash
# Install buf
go install github.com/bufbuild/buf/cmd/buf@latest

# Install protoc-gen-go plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### Ошибка подключения к базе данных

Проверьте формат вашего DSN:

- PostgreSQL: `postgres://user:pass@host:port/dbname?sslmode=disable`
- MySQL: `user:pass@tcp(host:port)/dbname?parseTime=true`
- SQLite: `file:path/to/database.db?cache=shared&mode=rwc`

## Следующие темы

- [Обзор архитектуры](./architecture.md) - Понимание дизайна Tonica
- [Режимы запуска](./run-modes.md) - Узнайте о различных режимах развертывания
- [Конфигурация](./configuration.md) - Полный справочник по конфигурации
- [Тестирование](./testing.md) - Напишите тесты для ваших сервисов
- [Лучшие практики](./best-practices.md) - Паттерны для продакшена

## Примеры

Ознакомьтесь с полными рабочими примерами на [GitHub](https://github.com/tonica-go/tonica/tree/main/example):

- Простой REST сервис
- gRPC сервис с базой данных
- Воркер с Temporal
- Обработчик с Kafka
- All-in-one приложение

Успешного кодирования!
