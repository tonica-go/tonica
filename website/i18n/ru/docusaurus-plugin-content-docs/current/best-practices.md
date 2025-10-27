# Лучшие практики для приложений на Tonica

Это руководство охватывает готовые к продакшену паттерны, антипаттерны и рекомендации для создания надежных приложений с использованием Tonica.

## Содержание

- [Структура приложения](#application-structure)
- [Обработка ошибок](#error-handling)
- [Логирование](#logging)
- [Управление конфигурацией](#configuration-management)
- [База данных](#database)
- [Дизайн API](#api-design)
- [Производительность](#performance)
- [Безопасность](#security)
- [Наблюдаемость](#observability)
- [Развертывание](#deployment)
- [Тестирование](#testing)

## Структура приложения

### Рекомендуемая структура проекта

```
myservice/
├── cmd/
│   └── server/
│       └── main.go                 # Точка входа в приложение
├── internal/                       # Приватный код приложения
│   ├── domain/                     # Доменные модели
│   │   └── user.go
│   ├── repository/                 # Слой доступа к данным
│   │   ├── user_repository.go
│   │   └── user_repository_test.go
│   ├── service/                    # Бизнес-логика
│   │   ├── user_service.go
│   │   └── user_service_test.go
│   └── handler/                    # RPC обработчики
│       ├── user_handler.go
│       └── user_handler_test.go
├── pkg/                            # Публичный переиспользуемый код
│   └── validator/
│       └── email.go
├── proto/                          # Определения Protocol Buffer
│   └── user/
│       └── v1/
│           └── user.proto
├── openapi/                        # Сгенерированные OpenAPI спецификации
├── migrations/                     # Миграции базы данных
│   ├── 001_create_users.sql
│   └── 002_add_users_email_index.sql
├── tests/
│   ├── integration/
│   └── e2e/
├── configs/                        # Конфигурационные файлы
│   ├── dev.env
│   └── prod.env
├── scripts/                        # Скрипты сборки и развертывания
├── .env.example                    # Пример переменных окружения
├── buf.gen.yaml                    # Конфигурация Buf
├── docker-compose.yml
├── Dockerfile
├── Makefile
├── go.mod
└── go.sum
```

### Организация пакетов

✅ **Хорошо: Четкое разделение обязанностей**
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

❌ **Плохо: Все в одном пакете**
```go
// main.go (5000 строк смешанной логики)
```

### Внедрение зависимостей

✅ **Хорошо: Внедрение через конструктор**
```go
import "log/slog"

type UserService struct {
    repo   UserRepository
    cache  *redis.Client
    logger *slog.Logger  // slog.Logger из стандартной библиотеки
}

func NewUserService(repo UserRepository, cache *redis.Client, logger *slog.Logger) *UserService {
    return &UserService{
        repo:   repo,
        cache:  cache,
        logger: logger,
    }
}
```

❌ **Плохо: Глобальные переменные**
```go
var globalDB *bun.DB
var globalCache *redis.Client

func GetUser(id string) (*User, error) {
    // Использует глобальные переменные
    return globalDB.Query(...)
}
```

## Обработка ошибок

### Ошибки gRPC

✅ **Хорошо: Используйте правильные коды статуса**
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

❌ **Плохо: Общие ошибки**
```go
func (h *UserHandler) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
    user, err := h.service.GetUser(ctx, req.Id)
    if err != nil {
        return nil, err  // ❌ Нет контекста, неправильный код
    }
    return &pb.GetUserResponse{User: user}, nil
}
```

### Типы ошибок

✅ **Хорошо: Определите пользовательские ошибки**
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

### Оборачивание ошибок

✅ **Хорошо: Добавляйте контекст к ошибкам**
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

## Логирование

### Структурированное логирование

✅ **Хорошо: Используйте структурированные поля**
```go
logger.Info("user created",
    "user_id", user.ID,
    "email", user.Email,
    "duration_ms", elapsed.Milliseconds(),
)
```

❌ **Плохо: Форматирование строк**
```go
logger.Info(fmt.Sprintf("User %s created with email %s", user.ID, user.Email))
```

### Уровни логирования

```go
// DEBUG - Детальная информация для отладки
logger.Debug("cache miss", "key", key)

// INFO - Общие информационные сообщения
logger.Info("user logged in", "user_id", userID)

// WARN - Предупреждающие сообщения (восстанавливаемые проблемы)
logger.Warn("rate limit approaching", "user_id", userID, "requests", count)

// ERROR - Сообщения об ошибках (что-то не удалось)
logger.Error("failed to send email", "error", err, "recipient", email)
```

### Что логировать

✅ **Логируйте:**
- Метаданные запросов/ответов (ID, длительность, статус)
- Бизнес-события (пользователь создан, заказ размещен)
- Ошибки с контекстом
- Метрики производительности
- События безопасности

❌ **Не логируйте:**
- Пароли или секреты
- Персональные данные (в продакшене)
- Полные тела запросов/ответов (кроме отладки)
- Избыточную отладочную информацию в продакшене

### Пример логирования

```go
import (
    "context"
    "log/slog"
    "time"
)

func (s *UserService) CreateUser(ctx context.Context, user *User) error {
    start := time.Now()

    // slog использует пары ключ-значение для структурированного логирования
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

## Управление конфигурацией

### Переменные окружения

✅ **Хорошо: Валидируйте при запуске**
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

### Управление секретами

❌ **Плохо: Жестко закодированные секреты**
```go
redis := tonica.NewRedis(
    tonica.WithRedisPassword("hardcoded-password"),  // ❌
)
```

✅ **Хорошо: Переменные окружения или менеджер секретов**
```go
// Из окружения
password := os.Getenv("REDIS_PASSWORD")
redis := tonica.NewRedis(tonica.WithRedisPassword(password))

// Или из менеджера секретов (AWS, GCP, и т.д.)
secret, err := secretsManager.GetSecret("redis-password")
redis := tonica.NewRedis(tonica.WithRedisPassword(secret))
```

## База данных

### Управление подключениями

✅ **Хорошо: Настройте пул подключений**
```go
db := service.NewDB(
    service.WithDriver(service.Postgres),
    service.WithDSN(dsn),
)

client := db.GetClient()

// Настройка пула для API сервиса
client.SetMaxOpenConns(25)
client.SetMaxIdleConns(10)
client.SetConnMaxLifetime(5 * time.Minute)
client.SetConnMaxIdleTime(10 * time.Minute)
```

### Паттерны запросов

✅ **Хорошо: Используйте контекст и таймауты**
```go
func (r *UserRepository) GetUser(ctx context.Context, id string) (*User, error) {
    // Добавляем таймаут к контексту
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    var user User
    err := r.db.NewSelect().
        Model(&user).
        Where("id = ?", id).
        Scan(ctx)  // Используем контекст

    return &user, err
}
```

✅ **Хорошо: Используйте подготовленные выражения (встроено в Bun)**
```go
// Bun автоматически использует подготовленные выражения
err := db.NewSelect().
    Model(&user).
    Where("email = ?", email).  // Защищено от SQL-инъекций
    Scan(ctx)
```

❌ **Плохо: Конкатенация строк**
```go
query := fmt.Sprintf("SELECT * FROM users WHERE email = '%s'", email)  // ❌ SQL-инъекция
```

### Транзакции

✅ **Хорошо: Используйте транзакции для многошаговых операций**
```go
func (r *UserRepository) CreateUserWithProfile(ctx context.Context, user *User, profile *Profile) error {
    return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
        // Создаем пользователя
        if _, err := tx.NewInsert().Model(user).Exec(ctx); err != nil {
            return err
        }

        // Создаем профиль
        profile.UserID = user.ID
        if _, err := tx.NewInsert().Model(profile).Exec(ctx); err != nil {
            return err
        }

        return nil
    })
}
```

### Миграции

✅ **Хорошо: Версионируемые миграции**
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

Используйте инструменты миграций, такие как:
- [golang-migrate](https://github.com/golang-migrate/migrate)
- [goose](https://github.com/pressly/goose)
- Bun migrations

## Дизайн API

### Принципы RESTful

✅ **Хорошо: URL на основе ресурсов**
```
GET    /api/v1/users           # Список пользователей
GET    /api/v1/users/:id       # Получить пользователя
POST   /api/v1/users           # Создать пользователя
PUT    /api/v1/users/:id       # Обновить пользователя
DELETE /api/v1/users/:id       # Удалить пользователя

GET    /api/v1/users/:id/orders  # Получить заказы пользователя
```

❌ **Плохо: URL на основе действий**
```
POST   /api/v1/getUser
POST   /api/v1/createUser
POST   /api/v1/deleteUser
```

### Версионирование API

✅ **Хорошо: Версия в пути**
```
/api/v1/users
/api/v2/users
```

### Валидация ввода

✅ **Хорошо: Валидируйте на ранней стадии**
```go
func (h *UserHandler) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
    // Валидация ввода
    if req.Email == "" {
        return nil, status.Error(codes.InvalidArgument, "email is required")
    }
    if !isValidEmail(req.Email) {
        return nil, status.Error(codes.InvalidArgument, "invalid email format")
    }
    if req.Name == "" {
        return nil, status.Error(codes.InvalidArgument, "name is required")
    }

    // Обработка запроса
    user, err := h.service.CreateUser(ctx, req)
    // ...
}
```

### Пагинация

✅ **Хорошо: Всегда используйте пагинацию для списковых эндпоинтов**
```go
message ListUsersRequest {
    int32 page = 1;    // Номер страницы (по умолчанию: 1)
    int32 limit = 2;   // Элементов на странице (по умолчанию: 10, макс: 100)
}

message ListUsersResponse {
    repeated User users = 1;
    int32 total = 2;
    int32 page = 3;
    int32 limit = 4;
}
```

## Производительность

### Стратегия кэширования

✅ **Хорошо: Кэшируйте дорогие операции**
```go
func (s *UserService) GetUser(ctx context.Context, id string) (*User, error) {
    // Сначала пробуем кэш
    cacheKey := "user:" + id
    if cached, err := s.cache.Get(ctx, cacheKey).Bytes(); err == nil {
        var user User
        if err := json.Unmarshal(cached, &user); err == nil {
            return &user, nil
        }
    }

    // Промах кэша - получаем из базы данных
    user, err := s.repo.GetUser(ctx, id)
    if err != nil {
        return nil, err
    }

    // Обновляем кэш
    if data, err := json.Marshal(user); err == nil {
        s.cache.Set(ctx, cacheKey, data, 10*time.Minute)
    }

    return user, nil
}
```

### Проблема N+1 запросов

❌ **Плохо: N+1 запросов**
```go
users, _ := repo.GetUsers(ctx)
for _, user := range users {
    orders, _ := repo.GetUserOrders(ctx, user.ID)  // ❌ Запрос в цикле
    // ...
}
```

✅ **Хорошо: Жадная загрузка**
```go
var users []User
err := db.NewSelect().
    Model(&users).
    Relation("Orders").  // Загружаем заказы одним запросом
    Scan(ctx)
```

### Индексы базы данных

✅ **Хорошо: Индексируйте часто запрашиваемые столбцы**
```sql
-- Индекс на email для поиска
CREATE INDEX idx_users_email ON users(email);

-- Составной индекс для многостолбцовых запросов
CREATE INDEX idx_orders_user_status ON orders(user_id, status);

-- Частичный индекс для специфичных условий
CREATE INDEX idx_active_users ON users(id) WHERE active = true;
```

### Пулинг подключений

✅ **Хорошо: Настройте под вашу нагрузку**
```go
// API сервис (высокая конкурентность)
db.SetMaxOpenConns(100)
db.SetMaxIdleConns(25)

// Воркер (низкая конкурентность, CPU-bound)
db.SetMaxOpenConns(10)
db.SetMaxIdleConns(5)
```

## Безопасность

### Аутентификация

✅ **Хорошо: Валидируйте токены**
```go
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        if token == "" {
            c.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
            return
        }

        // Валидация токена
        claims, err := validateJWT(token)
        if err != nil {
            c.AbortWithStatusJSON(401, gin.H{"error": "invalid token"})
            return
        }

        // Сохраняем информацию о пользователе в контексте
        c.Set("user_id", claims.UserID)
        c.Next()
    }
}
```

### Очистка ввода

✅ **Хорошо: Очищайте и валидируйте**
```go
func (s *UserService) CreateUser(ctx context.Context, email, name string) error {
    // Валидация
    email = strings.TrimSpace(strings.ToLower(email))
    name = strings.TrimSpace(name)

    if !isValidEmail(email) {
        return ErrInvalidEmail
    }

    if len(name) < 2 || len(name) > 100 {
        return ErrInvalidName
    }

    // Продолжаем...
}
```

### Ограничение скорости

✅ **Хорошо: Реализуйте ограничение скорости**
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

// Использование
limiter := rate.NewLimiter(rate.Every(time.Second), 10)  // 10 запросов/сек
app.GetRouter().Use(RateLimitMiddleware(limiter))
```

### Только HTTPS

✅ **Хорошо: Принудительное использование HTTPS в продакшене**
```go
// В продакшене используйте TLS
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

## Наблюдаемость

### Метрики

✅ **Хорошо: Инструментируйте критические пути**
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

### Трассировка

✅ **Хорошо: Добавляйте пользовательские спаны**
```go
import "go.opentelemetry.io/otel"

func (s *UserService) CreateUser(ctx context.Context, user *User) error {
    tracer := otel.Tracer("user-service")
    ctx, span := tracer.Start(ctx, "CreateUser")
    defer span.End()

    // Добавляем атрибуты
    span.SetAttributes(
        attribute.String("user.email", user.Email),
    )

    // Ваша логика
    if err := s.repo.Create(ctx, user); err != nil {
        span.RecordError(err)
        return err
    }

    return nil
}
```

### Проверки здоровья

✅ **Хорошо: Всеобъемлющие проверки здоровья**
```go
tonica.NewRoute(app).
    GET("/health").
    Handle(func(c *gin.Context) {
        health := gin.H{
            "status": "healthy",
        }

        // Проверка базы данных
        if err := db.PingContext(c.Request.Context()); err != nil {
            health["database"] = "unhealthy"
            health["status"] = "unhealthy"
            c.JSON(503, health)
            return
        }
        health["database"] = "healthy"

        // Проверка Redis
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

## Развертывание

### Лучшие практики Docker

✅ **Хорошо: Многоэтапная сборка**
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

### Развертывание в Kubernetes

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

        # Проверки здоровья
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

        # Ресурсы
        resources:
          requests:
            cpu: "500m"
            memory: "512Mi"
          limits:
            cpu: "1000m"
            memory: "1Gi"

        # Переменные окружения
        envFrom:
        - configMapRef:
            name: myservice-config
        - secretRef:
            name: myservice-secrets
```

### Graceful Shutdown

✅ **Хорошо: Правильно обрабатывайте сигналы**
```go
func main() {
    app := tonica.NewApp()

    // Запуск приложения (обработка сигналов автоматическая)
    if err := app.Run(); err != nil {
        app.GetLogger().Fatal(err)
    }

    log.Println("Application shutdown complete")
}
```

## Тестирование

Смотрите [Руководство по тестированию](./testing.md) для полного описания лучших практик тестирования.

### Ключевые принципы тестирования

✅ **Делайте:**
- Пишите тесты перед исправлением багов
- Тестируйте случаи ошибок и граничные случаи
- Используйте табличные тесты
- Мокируйте внешние зависимости
- Запускайте тесты в CI/CD

❌ **Не делайте:**
- Не тестируйте детали реализации
- Не пишите нестабильные тесты
- Не пропускайте интеграционные тесты
- Не игнорируйте сбои тестов
- Не добавляйте медленные тесты в набор unit-тестов

## Распространенные антипаттерны

### 1. God Objects (Божественные объекты)

❌ **Плохо:**
```go
type Manager struct {
    // Делает всё
}

func (m *Manager) CreateUser() {}
func (m *Manager) SendEmail() {}
func (m *Manager) ProcessPayment() {}
func (m *Manager) GenerateReport() {}
```

✅ **Хорошо:** Единая ответственность
```go
type UserService struct { /* связано с пользователями */ }
type EmailService struct { /* связано с email */ }
type PaymentService struct { /* связано с платежами */ }
```

### 2. Преждевременная оптимизация

❌ **Плохо:** Оптимизация до измерения
```go
// Сложное кэширование без понимания, нужно ли оно
```

✅ **Хорошо:** Сначала измерьте, затем оптимизируйте на основе данных
```go
// Профилируйте, определите узкое место, затем оптимизируйте
```

### 3. Отсутствие обработки ошибок

❌ **Плохо:**
```go
user, _ := repo.GetUser(ctx, id)  // Игнорирование ошибки
```

✅ **Хорошо:**
```go
user, err := repo.GetUser(ctx, id)
if err != nil {
    return nil, fmt.Errorf("failed to get user: %w", err)
}
```

## Чек-лист для продакшена

Перед развертыванием в продакшен:

- [ ] Все секреты в переменных окружения или менеджере секретов
- [ ] Настроен пул подключений к базе данных
- [ ] Правильная обработка ошибок и логирование
- [ ] Реализованы эндпоинты проверки здоровья
- [ ] Включены метрики и трассировка
- [ ] Ограничение скорости на публичных эндпоинтах
- [ ] Валидация ввода на всех эндпоинтах
- [ ] Включен TLS/HTTPS
- [ ] Протестированы миграции базы данных
- [ ] Проходят интеграционные тесты
- [ ] Выполнено нагрузочное тестирование
- [ ] Реализовано корректное завершение работы
- [ ] Настроены ограничения ресурсов (CPU, память)
- [ ] Настроен мониторинг и алертинг
- [ ] Обновлена документация

## Следующие шаги

- [Начало работы](./getting-started.md) - Создайте свое первое приложение
- [Архитектура](./architecture.md) - Поймите фреймворк
- [Тестирование](./testing.md) - Пишите всеобъемлющие тесты
- [Конфигурация](./configuration.md) - Настройте для продакшена
