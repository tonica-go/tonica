# Обзор архитектуры Tonica

Этот документ объясняет внутреннюю архитектуру и принципы проектирования фреймворка Tonica.

## Философия дизайна

Tonica построен на нескольких основных принципах:

1. **Proto-First**: API определяются с использованием Protocol Buffers, обеспечивая типобезопасность и контрактный дизайн
2. **Соглашения вместо конфигурации**: Разумные значения по умолчанию с возможностями для настройки
3. **Модульность**: Используйте только то, что нужно - запускайте как единый сервис или отдельные компоненты
4. **Наблюдаемость по умолчанию**: Встроенные метрики, трассировка и логирование
5. **Готовность к production**: Корректное завершение, проверки работоспособности и обработка ошибок из коробки

## Высокоуровневая архитектура

```mermaid
graph TB
    subgraph "Tonica Application"
        subgraph "Application Core"
            LIFECYCLE[App Lifecycle]
            CONFIG[Configuration Management]
            REGISTRY[Component Registry]
            SHUTDOWN[Shutdown Coordinator]
        end

        subgraph "Service Layer"
            GRPC[gRPC Server]
            REST[REST Gateway]
            CUSTOM[Custom Routes]
        end

        subgraph "Worker Layer"
            TEMPORAL[Temporal Workers]
            ACTIVITIES[Activities]
            WORKFLOWS[Workflows]
        end

        subgraph "Consumer Layer"
            KAFKA[Kafka Consumers]
            PUBSUB[PubSub Consumers]
            HANDLERS[Message Handlers]
        end

        subgraph "Infrastructure Layer"
            DB[(Database - Bun ORM)]
            REDIS[(Redis Cache)]
            MQ[Message Queue Clients]
        end

        subgraph "Observability Layer"
            OTEL[OpenTelemetry Tracing]
            PROM[Prometheus Metrics]
            LOGS[Structured Logging]
        end

        REGISTRY --> GRPC
        REGISTRY --> TEMPORAL
        REGISTRY --> KAFKA

        GRPC --> DB
        GRPC --> REDIS
        TEMPORAL --> DB
        KAFKA --> DB

        GRPC --> OTEL
        TEMPORAL --> OTEL
        KAFKA --> OTEL
    end
```

## Основные компоненты

### 1. Ядро приложения (`App`)

Структура `App` - это сердце Tonica. Она управляет жизненным циклом приложения и координирует все компоненты.

**Обязанности:**
- Инициализация и настройка всех подсистем
- Управление реестром компонентов
- Координация корректного завершения
- Предоставление доступа к общим ресурсам (логгер, метрики и т.д.)

**Ключевые поля:**
```go
type App struct {
    Name           string
    cfg            *config.Config
    logger         *log.Logger      // Standard logger
    router         *gin.Engine      // HTTP router
    metricRouter   *gin.Engine      // Metrics endpoint
    metricsManager *metrics.Manager
    registry       *Registry
    shutdown       *Shutdown
    customRoutes   []RouteMetadata  // Custom route definitions
}
```

**Жизненный цикл:**
```
NewApp() → Configure() → RegisterComponents() → Run() → Shutdown()
```

### 2. Реестр (Registry)

Реестр - это центральный репозиторий для всех компонентов приложения (сервисы, воркеры, консьюмеры).

**Назначение:**
- Сохранение и получение сервисов по имени
- Сохранение и получение воркеров по имени
- Сохранение и получение консьюмеров по имени
- Предотвращение дублирующихся регистраций
- Обеспечение обнаружения компонентов во время запуска

**Реализация:**
```go
type Registry struct {
    services  map[string]*Service
    workers   map[string]*Worker
    consumers map[string]*Consumer
    mu        sync.RWMutex  // Thread-safe access
}
```

**Потокобезопасность:**
Все операции реестра защищены мьютексом чтения-записи, что позволяет безопасный конкурентный доступ.

### 3. Слой сервисов

Сервисы обрабатывают gRPC и REST API запросы.

**Структура сервиса:**
```go
type Service struct {
    name       string
    grpcServer *grpc.Server
    db         *DB
    redis      *Redis
}
```

**Поток запросов:**

```mermaid
sequenceDiagram
    participant Client
    participant gRPC_Server_50051 as gRPC Server (port 50051)
    participant Service_Implementation as Service Implementation
    participant gRPC_Gateway as gRPC-Gateway
    participant HTTP_Router_8080 as HTTP Router (port 8080)

    Client->>gRPC_Server_50051: gRPC Request
    gRPC_Server_50051->>Service_Implementation: Process
    Service_Implementation-->>gRPC_Server_50051: Response

    Note over gRPC_Gateway: If REST Request
    Client->>HTTP_Router_8080: HTTP Request
    HTTP_Router_8080->>gRPC_Gateway: Route
    gRPC_Gateway->>Service_Implementation: Convert to gRPC
    Service_Implementation-->>gRPC_Gateway: gRPC Response
    gRPC_Gateway-->>HTTP_Router_8080: Convert to HTTP
    HTTP_Router_8080-->>Client: HTTP Response
```

**Возможности:**
- Автоматическая генерация REST API через gRPC-Gateway
- Генерация OpenAPI спецификации из proto аннотаций
- Встроенный middleware (CORS, логирование, метрики)
- Поддержка пользовательских маршрутов

### 4. Слой воркеров

Воркеры обрабатывают фоновые задачи с использованием Temporal.

**Структура воркера:**
```go
type Worker struct {
    name        string
    taskQueue   string
    client      client.Client
    worker      worker.Worker
}
```

**Поток выполнения:**

```mermaid
sequenceDiagram
    participant Temporal Server
    participant Worker
    participant Activity/Workflow

    Worker->>Temporal Server: Poll task queue
    Temporal Server-->>Worker: Task received
    Worker->>Activity/Workflow: Execute
    Activity/Workflow-->>Worker: Result
    Worker->>Temporal Server: Report result
```

**Возможности:**
- Регистрация активностей
- Регистрация workflow
- Автоматические повторные попытки и обработка ошибок
- Интеграция распределенной трассировки

### 5. Слой консьюмеров

Консьюмеры обрабатывают сообщения из Kafka, PubSub или других очередей сообщений.

**Структура консьюмера:**
```go
type Consumer struct {
    name          string
    topic         string
    consumerGroup string
    client        storage.Client
    handler       func(context.Context, *pubsub.Message) error
}
```

**Поток обработки:**

```mermaid
sequenceDiagram
    participant Message Queue
    participant Consumer
    participant Handler

    Consumer->>Message Queue: Subscribe to topic
    Message Queue-->>Consumer: Message received
    Consumer->>Handler: Execute handler
    Handler-->>Consumer: Result
    alt Success
        Consumer->>Message Queue: Acknowledge
    else Error
        Consumer->>Message Queue: Retry/Nack
    end
```

**Возможности:**
- Отмена на основе контекста
- Автоматическая обработка ошибок и логирование
- Корректное завершение (ожидает завершения обработки сообщения)
- Поддержка групп консьюмеров

### 6. Инфраструктурный слой

#### База данных (Bun ORM)

Tonica использует [Bun](https://bun.uptrace.dev/) в качестве ORM, поддерживая PostgreSQL, MySQL и SQLite.

**Управление соединениями:**
```go
type DB struct {
    driver string
    dsn    string
    db     *bun.DB  // Cached connection
}

func (d *DB) GetClient() *bun.DB {
    if d.db != nil {
        return d.db  // Return cached connection
    }
    // Create new connection
    // ...
}
```

**Возможности:**
- Пулинг соединений
- Автоматическое определение драйвера
- Поддержка миграций (через Bun)
- Построитель запросов

#### Redis

Обертка клиента Redis с кэшированием соединений.

**Управление соединениями:**
```go
type Redis struct {
    addr     string
    password string
    database int
    conn     *redis.Client  // Cached connection
}

func (r *Redis) GetClient() *redis.Client {
    if r.conn == nil {
        r.conn = redis.NewClient(&redis.Options{
            Addr:     r.addr,
            Password: r.password,
            DB:       r.database,
        })
    }
    return r.conn
}
```

**Варианты использования:**
- Кэширование
- Хранение сессий
- Ограничение скорости
- Распределенные блокировки

### 7. Слой наблюдаемости

#### Трассировка OpenTelemetry

Автоматическая распределенная трассировка для всех запросов.

**Точки инструментации:**
- HTTP запросы (входящие/исходящие)
- gRPC вызовы (входящие/исходящие)
- Запросы к базе данных
- Операции Redis
- Пользовательские span'ы (определяемые пользователем)

**Конфигурация:**
```bash
OTEL_ENABLED=true
OTEL_ENDPOINT=localhost:4317
OTEL_SERVICE_NAME=myservice
```

#### Метрики Prometheus

Встроенный сбор и экспорт метрик.

**Метрики по умолчанию:**
- `app_info` - Метаданные приложения
- Метрики HTTP запросов (длительность, количество, статус)
- Метрики gRPC запросов
- Метрики Go runtime (горутины, память, GC)

**Пользовательские метрики:**
```go
counter := app.GetMetricManager().NewCounter("orders_total", "Total orders")
gauge := app.GetMetricManager().NewGauge("active_connections", "Active connections")
histogram := app.GetMetricManager().NewHistogram("request_duration", "Request duration")
```

**Эндпоинт метрик:**
```
http://localhost:9090/metrics
```

#### Структурированное логирование

Использует стандартный `log/slog` Go для структурированного логирования.

**Уровни логов:**
- `DEBUG` - Детальная отладочная информация
- `INFO` - Общие информационные сообщения
- `WARN` - Предупреждающие сообщения
- `ERROR` - Сообщения об ошибках

**Использование:**
```go
import "log/slog"

// Use slog for structured logging
slog.Info("User created", "user_id", userID, "email", email)
slog.Error("Failed to process order", "order_id", orderID, "error", err)

// Or use app's logger (*log.Logger)
app.GetLogger().Printf("User created: %s", userID)
```

### 8. Координатор завершения

Координатор завершения обеспечивает корректное завершение работы всех компонентов.

**Поток завершения:**

```mermaid
graph TD
    A[Signal Received<br/>SIGINT/SIGTERM] --> B[Create context with timeout 30s]
    B --> C{Parallel Shutdown}
    C --> D[HTTP Servers<br/>Graceful Shutdown]
    C --> E[gRPC Servers<br/>Graceful Stop]
    C --> F[Cleanup Functions<br/>User-defined]
    D --> G{Wait for Completion}
    E --> G
    F --> G
    G -->|Completed| H[Exit]
    G -->|Timeout| I[Force Stop]
    I --> H
```

**Регистрация:**
```go
// Register HTTP server
app.shutdown.RegisterHTTPServer(httpServer)

// Register gRPC server
app.shutdown.RegisterGRPCServer(grpcServer)

// Register cleanup function
app.shutdown.RegisterCleanup(func(ctx context.Context) error {
    // Close database connections, flush metrics, etc.
    return db.Close()
})
```

**Возможности:**
- Параллельное завершение (быстро)
- Защита по таймауту (без зависаний)
- Агрегация ошибок
- Потокобезопасная регистрация

## Режимы запуска

Tonica поддерживает четыре режима запуска, каждый из которых активирует разные компоненты:

### ModeAio (Все-в-одном)
Запускает всё: gRPC + REST + Воркеры + Консьюмеры

```mermaid
graph LR
    subgraph "Port 8080"
        HTTP[HTTP/REST + OpenAPI]
    end
    subgraph "Port 50051"
        GRPC[gRPC Server]
    end
    subgraph "Background"
        TEMPORAL[Temporal Workers]
        CONSUMERS[Message Consumers]
    end
    subgraph "Port 9090"
        METRICS[Metrics Endpoint]
    end

    HTTP -.runs together.- GRPC
    GRPC -.runs together.- TEMPORAL
    TEMPORAL -.runs together.- CONSUMERS
    CONSUMERS -.runs together.- METRICS
```

### ModeService
Запускает только gRPC и REST API

### ModeWorker
Запускает только Temporal воркеры (+ метрики)

### ModeConsumer
Запускает только консьюмеры сообщений (+ метрики)

См. [Режимы запуска](./run-modes.md) для детального сравнения.

## Примеры потоков запросов

### gRPC запрос

```mermaid
sequenceDiagram
    participant Client
    participant gRPC Middleware
    participant Service Implementation
    participant Database/Redis

    Client->>gRPC Middleware: gRPC Request :50051
    Note over gRPC Middleware: Logging, Tracing, Metrics
    gRPC Middleware->>Service Implementation: Forward Request
    Service Implementation->>Database/Redis: Query (if needed)
    Database/Redis-->>Service Implementation: Result
    Service Implementation-->>gRPC Middleware: Response
    Note over gRPC Middleware: Record metrics, close span
    gRPC Middleware-->>Client: Response
```

### REST запрос (сгенерированный из Proto)

```mermaid
sequenceDiagram
    participant Client
    participant HTTP Middleware
    participant gRPC-Gateway
    participant gRPC Service
    participant Service Implementation

    Client->>HTTP Middleware: POST :8080/api/v1/users
    Note over HTTP Middleware: CORS, Logging, Tracing
    HTTP Middleware->>gRPC-Gateway: Forward Request
    gRPC-Gateway->>gRPC-Gateway: Translate HTTP → gRPC
    gRPC-Gateway->>gRPC Service: Internal gRPC Call
    gRPC Service->>Service Implementation: Execute
    Service Implementation-->>gRPC Service: gRPC Response
    gRPC Service-->>gRPC-Gateway: Response
    gRPC-Gateway->>gRPC-Gateway: Translate gRPC → HTTP
    gRPC-Gateway-->>HTTP Middleware: JSON Response
    HTTP Middleware-->>Client: JSON Response
```

### REST запрос (пользовательский маршрут)

```mermaid
sequenceDiagram
    participant Client
    participant HTTP Middleware
    participant Gin Router
    participant Handler Function

    Client->>HTTP Middleware: GET :8080/health
    Note over HTTP Middleware: CORS, Logging, Tracing
    HTTP Middleware->>Gin Router: Forward Request
    Gin Router->>Gin Router: Match custom route
    Gin Router->>Handler Function: Execute
    Handler Function-->>Gin Router: Response
    Gin Router-->>HTTP Middleware: JSON Response
    HTTP Middleware-->>Client: JSON Response
```

### Обработка задач воркером

```mermaid
sequenceDiagram
    participant Workflow
    participant Temporal Server
    participant Worker
    participant Activity Function
    participant Database

    Workflow->>Temporal Server: Trigger activity
    Temporal Server->>Worker: Assign task
    Worker->>Worker: Pick up from queue
    Worker->>Activity Function: Execute
    Activity Function->>Database: Operations (if needed)
    Database-->>Activity Function: Result
    Activity Function-->>Worker: Result
    Worker->>Temporal Server: Report result
    Temporal Server->>Workflow: Continue
```

### Обработка сообщений консьюмером

```mermaid
sequenceDiagram
    participant Message Queue
    participant Consumer
    participant Handler Function
    participant Database

    Message Queue->>Consumer: Publish message to topic
    Consumer->>Consumer: Receive message
    Consumer->>Handler Function: Call with context
    Handler Function->>Handler Function: Execute business logic
    Handler Function->>Database: Operations (if needed)
    Database-->>Handler Function: Result
    alt Success
        Handler Function-->>Consumer: Success
        Consumer->>Message Queue: Acknowledge
    else Error
        Handler Function-->>Consumer: Error
        Consumer->>Consumer: Log error
        Consumer->>Message Queue: Retry/Nack
    end
```

## Управление конфигурацией

Конфигурация следует этому порядку приоритетов (от высшего к низшему):

1. **Код** - Опции, переданные в конструкторы
2. **Переменные окружения** - `APP_NAME`, `DB_DSN` и т.д.
3. **Значения по умолчанию** - Встроенные разумные значения

Пример:
```go
// 1. Code (highest priority)
app := tonica.NewApp(tonica.WithName("myservice"))

// 2. Environment variable
// export APP_NAME="envservice"

// 3. Default (lowest priority)
// Falls back to "tonica-app" if nothing specified
```

См. [Конфигурация](./configuration.md) для всех опций.

## Стратегия обработки ошибок

Tonica использует многоуровневый подход к обработке ошибок:

### Уровень 1: Ошибки приложения
```go
if err != nil {
    return nil, status.Error(codes.InvalidArgument, "invalid user ID")
}
```

### Уровень 2: Ошибки middleware
Автоматически обрабатываются фреймворком:
- HTTP 500 для внутренних ошибок
- HTTP 4xx для ошибок клиента
- Логируются с контекстом

### Уровень 3: Ошибки инфраструктуры
База данных, Redis и т.д.:
```go
import "log/slog"

user, err := db.GetUser(ctx, userID)
if err != nil {
    slog.Error("Failed to get user", "error", err, "user_id", userID)
    return nil, status.Error(codes.Internal, "database error")
}
```

### Уровень 4: Восстановление после паники
Автоматическое восстановление после паники в HTTP/gRPC обработчиках:
- Логирует панику со стеком вызовов
- Возвращает ошибку 500 клиенту
- Не приводит к падению приложения

## Модель конкурентности

### Гарантии потокобезопасности

**App**:
- Безопасно вызывать геттеры конкурентно
- Доступ к реестру защищен мьютексом
- Менеджер метрик потокобезопасен

**Registry**:
- Все операции используют RWMutex
- Разрешено несколько читателей
- Эксклюзивная блокировка на запись

**Services/Workers/Consumers**:
- Каждый запрос/задача/сообщение в отдельной горутине
- Нет общего изменяемого состояния (если не спроектировано явно)
- Отмена на основе контекста

### Управление горутинами

**HTTP сервер**:
- Одна горутина на запрос
- Автоматически очищается после ответа

**gRPC сервер**:
- Одна горутина на поток
- Очищается при закрытии потока

**Воркеры**:
- Настраиваемое количество конкурентных активностей
- Управляются Temporal воркером

**Консьюмеры**:
- Одна горутина на консьюмер
- Обработка сообщений может быть последовательной или параллельной (выбор пользователя)

## Соображения по производительности

### Пулинг соединений

**База данных**:
```go
// Bun handles connection pooling internally
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(10)
db.SetConnMaxLifetime(5 * time.Minute)
```

**Redis**:
```go
// Redis client has built-in connection pool
// Configured via redis.Options
&redis.Options{
    PoolSize:     10,
    MinIdleConns: 5,
}
```

### Стратегия кэширования

1. **Уровень приложения** - Кэши в памяти в вашем коде
2. **Redis** - Распределенный кэш для многоэкземплярных развертываний
3. **База данных** - Кэширование результатов запросов через Bun

### Накладные расходы метрик

Сбор метрик легковесен:
- Инкремент счетчика: ~100нс
- Наблюдение гистограммы: ~500нс
- Незначительное влияние на производительность

## Соображения безопасности

### Аутентификация и авторизация

Не встроена - реализуйте как middleware:

```go
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        if !validateToken(token) {
            c.AbortWithStatus(401)
            return
        }
        c.Next()
    }
}

app.GetRouter().Use(AuthMiddleware())
```

### CORS

Встроенная поддержка CORS:
```bash
# Allow all origins (default)
# No configuration needed

# Restrict origins
export APP_CORS_ORIGINS="https://myapp.com,https://api.myapp.com"
```

### TLS/HTTPS

Настройка на уровне развертывания:
- Используйте обратный прокси (nginx, Traefik) с терминацией TLS
- Или настройте TLS непосредственно в Gin роутере

## Расширение Tonica

### Пользовательский Middleware

**HTTP Middleware**:
```go
app.GetRouter().Use(MyCustomMiddleware())
```

**gRPC Interceptors**:
```go
grpcServer := grpc.NewServer(
    grpc.UnaryInterceptor(MyInterceptor),
)
```

### Пользовательские метрики

```go
myCounter := app.GetMetricManager().NewCounter("my_metric", "Description")
myCounter.Inc()
```

### Пользовательские проверки работоспособности

```go
tonica.NewRoute(app).
    GET("/health/detailed").
    Handle(func(c *gin.Context) {
        health := map[string]string{
            "database": checkDB(),
            "redis":    checkRedis(),
            "workers":  checkWorkers(),
        }
        c.JSON(200, health)
    })
```

## Следующие шаги

- [Режимы запуска](./run-modes.md) - Узнайте, когда использовать каждый режим
- [Конфигурация](./configuration.md) - Настройте ваше приложение
- [Тестирование](./testing.md) - Пишите тесты для ваших сервисов
- [Лучшие практики](./best-practices.md) - Паттерны для production

## Дополнительное чтение

- [Protocol Buffers](https://protobuf.dev/)
- [gRPC](https://grpc.io/)
- [gRPC-Gateway](https://grpc-ecosystem.github.io/grpc-gateway/)
- [Temporal](https://temporal.io/)
- [Bun ORM](https://bun.uptrace.dev/)
- [OpenTelemetry](https://opentelemetry.io/)
