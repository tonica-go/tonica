# Руководство по тестированию

Это руководство описывает, как писать тесты для приложений, построенных с использованием фреймворка Tonica, включая модульные тесты, интеграционные тесты и end-to-end тесты.

## Философия тестирования

Tonica спроектирована для удобного тестирования:

- **Внедрение зависимостей**: Легко создавать моки для зависимостей
- **Интерфейсный подход**: Сервисы реализуют интерфейсы
- **Модульная архитектура**: Тестирование компонентов в изоляции
- **Стандартная библиотека**: Использует стандартный пакет `testing` из Go

## Структура тестов

Рекомендуемая организация тестов:

```
myservice/
├── internal/
│   ├── service/
│   │   ├── user_service.go
│   │   └── user_service_test.go      # Модульные тесты
│   └── repository/
│       ├── user_repository.go
│       └── user_repository_test.go
├── tests/
│   ├── integration/                   # Интеграционные тесты
│   │   └── user_api_test.go
│   └── e2e/                          # End-to-end тесты
│       └── complete_flow_test.go
└── main_test.go                       # Тесты основного пакета
```

## Модульное тестирование

### Тестирование реализации сервиса

```go
package service

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    hellov1 "github.com/yourusername/myservice/proto/hello/v1"
)

// Мок репозитория
type MockUserRepo struct {
    mock.Mock
}

func (m *MockUserRepo) GetUser(ctx context.Context, id string) (*User, error) {
    args := m.Called(ctx, id)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*User), args.Error(1)
}

// Тест
func TestHelloService_SayHello(t *testing.T) {
    t.Run("should return greeting with name", func(t *testing.T) {
        svc := &HelloServiceImpl{}

        resp, err := svc.SayHello(context.Background(), &hellov1.SayHelloRequest{
            Name: "Alice",
        })

        assert.NoError(t, err)
        assert.NotNil(t, resp)
        assert.Equal(t, "Hello, Alice!", resp.Message)
    })

    t.Run("should return error for empty name", func(t *testing.T) {
        svc := &HelloServiceImpl{}

        resp, err := svc.SayHello(context.Background(), &hellov1.SayHelloRequest{
            Name: "",
        })

        assert.Error(t, err)
        assert.Nil(t, resp)
    })
}

func TestUserService_GetUser(t *testing.T) {
    mockRepo := new(MockUserRepo)
    svc := &UserService{
        repo: mockRepo,
    }

    t.Run("should return user when found", func(t *testing.T) {
        expectedUser := &User{ID: "123", Name: "Alice"}
        mockRepo.On("GetUser", mock.Anything, "123").Return(expectedUser, nil)

        user, err := svc.GetUser(context.Background(), "123")

        assert.NoError(t, err)
        assert.Equal(t, expectedUser, user)
        mockRepo.AssertExpectations(t)
    })

    t.Run("should return error when user not found", func(t *testing.T) {
        mockRepo.On("GetUser", mock.Anything, "999").Return(nil, errors.New("not found"))

        user, err := svc.GetUser(context.Background(), "999")

        assert.Error(t, err)
        assert.Nil(t, user)
        mockRepo.AssertExpectations(t)
    })
}
```

### Тестирование пользовательских маршрутов

```go
package main

import (
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/assert"
    "github.com/tonica-go/tonica/pkg/tonica"
)

func TestHealthEndpoint(t *testing.T) {
    gin.SetMode(gin.TestMode)
    app := tonica.NewApp()

    // Регистрируем маршрут
    tonica.NewRoute(app).
        GET("/health").
        Handle(func(c *gin.Context) {
            c.JSON(200, gin.H{"status": "healthy"})
        })

    // Тестируем его
    w := httptest.NewRecorder()
    req, _ := http.NewRequest("GET", "/health", nil)
    app.GetRouter().ServeHTTP(w, req)

    assert.Equal(t, 200, w.Code)
    assert.JSONEq(t, `{"status":"healthy"}`, w.Body.String())
}

func TestUserEndpoints(t *testing.T) {
    gin.SetMode(gin.TestMode)
    app := tonica.NewApp()

    // Регистрируем маршруты
    setupUserRoutes(app)

    t.Run("GET /users/:id", func(t *testing.T) {
        w := httptest.NewRecorder()
        req, _ := http.NewRequest("GET", "/users/123", nil)
        app.GetRouter().ServeHTTP(w, req)

        assert.Equal(t, 200, w.Code)
        assert.Contains(t, w.Body.String(), "123")
    })

    t.Run("POST /users", func(t *testing.T) {
        body := strings.NewReader(`{"name":"Alice","email":"alice@example.com"}`)
        w := httptest.NewRecorder()
        req, _ := http.NewRequest("POST", "/users", body)
        req.Header.Set("Content-Type", "application/json")
        app.GetRouter().ServeHTTP(w, req)

        assert.Equal(t, 201, w.Code)
    })
}
```

### Тестирование воркеров

```go
package worker

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "go.temporal.io/sdk/testsuite"
)

type EmailActivitySuite struct {
    testsuite.WorkflowTestSuite
}

func TestEmailActivity(t *testing.T) {
    suite.Run(t, new(EmailActivitySuite))
}

func (s *EmailActivitySuite) TestSendEmailActivity() {
    env := s.NewTestActivityEnvironment()

    // Регистрируем активность
    env.RegisterActivity(SendEmailActivity)

    // Выполняем активность
    val, err := env.ExecuteActivity(SendEmailActivity, "test@example.com", "Hello")

    s.NoError(err)
    s.NotNil(val)

    var result string
    s.NoError(val.Get(&result))
    s.Equal("Email sent to test@example.com", result)
}

func (s *EmailActivitySuite) TestEmailWorkflow() {
    env := s.NewTestWorkflowEnvironment()

    // Мокируем активность
    env.OnActivity(SendEmailActivity, mock.Anything, "test@example.com", "Hello").
        Return("sent", nil)

    // Выполняем workflow
    env.ExecuteWorkflow(EmailWorkflow, "test@example.com", "Hello")

    s.True(env.IsWorkflowCompleted())
    s.NoError(env.GetWorkflowError())
}
```

### Тестирование консьюмеров

```go
package consumer

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/tonica-go/tonica/pkg/tonica/consumer"
)

type MockPubSubClient struct {
    messages chan *pubsub.Message
    errors   chan error
}

func (m *MockPubSubClient) Subscribe(ctx context.Context, topic string) (*pubsub.Message, error) {
    select {
    case msg := <-m.messages:
        return msg, nil
    case err := <-m.errors:
        return nil, err
    case <-ctx.Done():
        return nil, ctx.Err()
    }
}

func TestConsumer_ProcessMessage(t *testing.T) {
    t.Run("should process message successfully", func(t *testing.T) {
        mockClient := &MockPubSubClient{
            messages: make(chan *pubsub.Message, 1),
            errors:   make(chan error),
        }

        processed := false
        handler := func(ctx context.Context, msg *pubsub.Message) error {
            processed = true
            return nil
        }

        cons := consumer.NewConsumer(
            consumer.WithName("test-consumer"),
            consumer.WithTopic("test-topic"),
            consumer.WithClient(mockClient),
            consumer.WithHandler(handler),
        )

        // Отправляем тестовое сообщение
        mockClient.messages <- &pubsub.Message{Value: []byte("test")}

        // Запускаем консьюмер с таймаутом
        ctx, cancel := context.WithTimeout(context.Background(), time.Second)
        defer cancel()

        go cons.Start(ctx)

        // Ждем обработки
        time.Sleep(100 * time.Millisecond)
        assert.True(t, processed)
    })
}
```

## Интеграционное тестирование

### Тестирование с реальной базой данных

```go
package integration

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/tonica-go/tonica/pkg/tonica/service"
)

func TestUserRepository_Integration(t *testing.T) {
    // Пропускаем, если не запускаются интеграционные тесты
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    // Настраиваем тестовую базу данных
    db := service.NewDB(
        service.WithDriver(service.Sqlite),
        service.WithDSN("file::memory:?cache=shared"),
    )
    client := db.GetClient()

    // Запускаем миграции
    err := runMigrations(client)
    require.NoError(t, err)

    // Создаем репозиторий
    repo := NewUserRepository(client)

    t.Run("should create and retrieve user", func(t *testing.T) {
        user := &User{
            Name:  "Alice",
            Email: "alice@example.com",
        }

        // Создаем
        err := repo.Create(context.Background(), user)
        require.NoError(t, err)
        assert.NotEmpty(t, user.ID)

        // Получаем
        retrieved, err := repo.GetByID(context.Background(), user.ID)
        require.NoError(t, err)
        assert.Equal(t, user.Name, retrieved.Name)
        assert.Equal(t, user.Email, retrieved.Email)
    })

    // Очистка
    defer client.Close()
}
```

### Тестирование HTTP сервера

```go
package integration

import (
    "context"
    "net/http"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/tonica-go/tonica/pkg/tonica"
)

func TestHTTPServer_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    // Создаем приложение
    app := tonica.NewApp(
        tonica.WithName("test-service"),
    )

    // Регистрируем маршруты
    setupRoutes(app)

    // Запускаем сервер в фоне
    go func() {
        app.Run()
    }()

    // Ждем запуска сервера
    time.Sleep(500 * time.Millisecond)

    // Тестовые запросы
    t.Run("GET /health", func(t *testing.T) {
        resp, err := http.Get("http://localhost:8080/health")
        assert.NoError(t, err)
        assert.Equal(t, 200, resp.StatusCode)
        defer resp.Body.Close()
    })

    t.Run("GET /users", func(t *testing.T) {
        resp, err := http.Get("http://localhost:8080/users")
        assert.NoError(t, err)
        assert.Equal(t, 200, resp.StatusCode)
        defer resp.Body.Close()
    })

    // Очистка: отменяем контекст для остановки сервера
    cancel()
    time.Sleep(100 * time.Millisecond)
}
```

### Тестирование с Docker Compose

```go
package integration

import (
    "context"
    "testing"
    "time"

    "github.com/stretchr/testify/require"
    "github.com/tonica-go/tonica/pkg/tonica/service"
)

func TestWithRealPostgres(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    // Предполагается, что postgres запущен через docker-compose
    db := service.NewDB(
        service.WithDriver(service.Postgres),
        service.WithDSN("postgres://test:test@localhost:5432/testdb?sslmode=disable"),
    )

    ctx := context.Background()
    client := db.GetClient()

    // Тестируем подключение
    err := client.PingContext(ctx)
    require.NoError(t, err)

    // Запускаем ваши тесты...
    t.Run("should perform database operations", func(t *testing.T) {
        // Реализация теста
    })

    // Очистка
    defer client.Close()
}
```

**docker-compose.test.yml:**
```yaml
version: '3.8'
services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_USER: test
      POSTGRES_PASSWORD: test
      POSTGRES_DB: testdb
    ports:
      - "5432:5432"

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
```

**Запуск тестов:**
```bash
# Запускаем зависимости
docker-compose -f docker-compose.test.yml up -d

# Запускаем тесты
go test ./tests/integration/... -v

# Очистка
docker-compose -f docker-compose.test.yml down
```

## End-to-End тестирование

### Полный пользовательский сценарий

```go
package e2e

import (
    "bytes"
    "encoding/json"
    "net/http"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestUserFlow_E2E(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping e2e test")
    }

    baseURL := "http://localhost:8080"
    var userID string

    t.Run("1. Create user", func(t *testing.T) {
        payload := map[string]string{
            "name":  "Alice",
            "email": "alice@example.com",
        }
        body, _ := json.Marshal(payload)

        resp, err := http.Post(baseURL+"/users", "application/json", bytes.NewReader(body))
        require.NoError(t, err)
        defer resp.Body.Close()

        assert.Equal(t, 201, resp.StatusCode)

        var result map[string]interface{}
        json.NewDecoder(resp.Body).Decode(&result)
        userID = result["id"].(string)
        assert.NotEmpty(t, userID)
    })

    t.Run("2. Get user", func(t *testing.T) {
        resp, err := http.Get(baseURL + "/users/" + userID)
        require.NoError(t, err)
        defer resp.Body.Close()

        assert.Equal(t, 200, resp.StatusCode)

        var user map[string]interface{}
        json.NewDecoder(resp.Body).Decode(&user)
        assert.Equal(t, "Alice", user["name"])
        assert.Equal(t, "alice@example.com", user["email"])
    })

    t.Run("3. Update user", func(t *testing.T) {
        payload := map[string]string{
            "name": "Alice Smith",
        }
        body, _ := json.Marshal(payload)

        req, _ := http.NewRequest("PUT", baseURL+"/users/"+userID, bytes.NewReader(body))
        req.Header.Set("Content-Type", "application/json")

        resp, err := http.DefaultClient.Do(req)
        require.NoError(t, err)
        defer resp.Body.Close()

        assert.Equal(t, 200, resp.StatusCode)
    })

    t.Run("4. Delete user", func(t *testing.T) {
        req, _ := http.NewRequest("DELETE", baseURL+"/users/"+userID, nil)
        resp, err := http.DefaultClient.Do(req)
        require.NoError(t, err)
        defer resp.Body.Close()

        assert.Equal(t, 204, resp.StatusCode)
    })

    t.Run("5. Verify deletion", func(t *testing.T) {
        resp, err := http.Get(baseURL + "/users/" + userID)
        require.NoError(t, err)
        defer resp.Body.Close()

        assert.Equal(t, 404, resp.StatusCode)
    })
}
```

## Вспомогательные функции для тестов

### Фабрика тестового приложения

```go
package testutil

import (
    "github.com/tonica-go/tonica/pkg/tonica"
    "github.com/tonica-go/tonica/pkg/tonica/service"
)

func NewTestApp() *tonica.App {
    app := tonica.NewApp(
        tonica.WithName("test-app"),
    )

    // Используем базу данных в памяти
    db := service.NewDB(
        service.WithDriver(service.Sqlite),
        service.WithDSN("file::memory:?cache=shared"),
    )

    // Добавляем в приложение (при необходимости)
    // ...

    return app
}
```

### Тестовые фикстуры

```go
package testutil

type UserFixture struct {
    ID    string
    Name  string
    Email string
}

func CreateUserFixture(t *testing.T, repo UserRepository) *UserFixture {
    user := &User{
        Name:  "Test User",
        Email: "test@example.com",
    }

    err := repo.Create(context.Background(), user)
    if err != nil {
        t.Fatalf("Failed to create user fixture: %v", err)
    }

    return &UserFixture{
        ID:    user.ID,
        Name:  user.Name,
        Email: user.Email,
    }
}
```

### Моки компонентов Tonica

```go
package testutil

import "github.com/tonica-go/tonica/pkg/tonica/storage"

type MockStorage struct {
    data map[string][]byte
}

func NewMockStorage() *MockStorage {
    return &MockStorage{
        data: make(map[string][]byte),
    }
}

func (m *MockStorage) Set(ctx context.Context, key string, value []byte) error {
    m.data[key] = value
    return nil
}

func (m *MockStorage) Get(ctx context.Context, key string) ([]byte, error) {
    if val, ok := m.data[key]; ok {
        return val, nil
    }
    return nil, storage.ErrNotFound
}
```

## Запуск тестов

### Запуск всех тестов

```bash
go test ./...
```

### Запуск конкретного пакета

```bash
go test ./internal/service
```

### Запуск с покрытием

```bash
go test ./... -cover

# Детальное покрытие
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Запуск только модульных тестов (пропуск интеграционных)

```bash
go test ./... -short
```

### Запуск только интеграционных тестов

```bash
go test ./tests/integration/... -v
```

### Запуск с детектором гонок

```bash
go test ./... -race
```

### Параллельное выполнение

```bash
# Запуск тестов параллельно
go test ./... -parallel 4

# В коде теста
func TestSomething(t *testing.T) {
    t.Parallel()  // Включаем параллельное выполнение
    // ...
}
```

### Подробный вывод

```bash
go test ./... -v
```

## Непрерывная интеграция

### GitHub Actions

```.github/workflows/test.yml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest

    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_USER: test
          POSTGRES_PASSWORD: test
          POSTGRES_DB: testdb
        ports:
          - 5432:5432

      redis:
        image: redis:7-alpine
        ports:
          - 6379:6379

    steps:
    - uses: actions/checkout@v3

    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'

    - name: Install dependencies
      run: go mod download

    - name: Run unit tests
      run: go test ./... -short -cover

    - name: Run integration tests
      run: go test ./tests/integration/... -v
      env:
        DB_DSN: postgres://test:test@localhost:5432/testdb?sslmode=disable
        REDIS_ADDR: localhost:6379

    - name: Run with race detector
      run: go test ./... -race -short
```

## Лучшие практики тестирования

### 1. Используйте табличные тесты

```go
func TestValidateEmail(t *testing.T) {
    tests := []struct {
        name    string
        email   string
        want    bool
        wantErr bool
    }{
        {"valid email", "user@example.com", true, false},
        {"invalid email", "invalid", false, true},
        {"empty email", "", false, true},
        {"email without domain", "user@", false, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ValidateEmail(tt.email)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateEmail() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("ValidateEmail() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### 2. Настройка и очистка

```go
func TestMain(m *testing.M) {
    // Настройка
    setup()

    // Запуск тестов
    code := m.Run()

    // Очистка
    teardown()

    os.Exit(code)
}

func TestWithSetup(t *testing.T) {
    // Настройка для отдельного теста
    db := setupTestDB(t)
    defer db.Close()

    // Код теста...
}
```

### 3. Используйте подтесты для организации

```go
func TestUserService(t *testing.T) {
    svc := setupService(t)

    t.Run("Create", func(t *testing.T) {
        // Тест создания пользователя
    })

    t.Run("Get", func(t *testing.T) {
        // Тест получения пользователя
    })

    t.Run("Update", func(t *testing.T) {
        // Тест обновления пользователя
    })
}
```

### 4. Мокируйте внешние зависимости

```go
// Хорошо: Мокируем внешний API
type MockPaymentAPI struct {
    mock.Mock
}

func (m *MockPaymentAPI) Charge(amount int) error {
    args := m.Called(amount)
    return args.Error(0)
}

// Плохо: Вызываем реальный внешний API в тестах
func TestPayment(t *testing.T) {
    api := NewRealPaymentAPI() // ❌ Не делайте так
    // ...
}
```

### 5. Тестируйте случаи ошибок

```go
func TestGetUser(t *testing.T) {
    t.Run("success case", func(t *testing.T) {
        // Тест успешного сценария
    })

    t.Run("user not found", func(t *testing.T) {
        // Тест случая ошибки
    })

    t.Run("database error", func(t *testing.T) {
        // Тест случая ошибки
    })

    t.Run("invalid input", func(t *testing.T) {
        // Тест валидации
    })
}
```

## Распространенные паттерны тестирования

### Тестирование с контекстом

```go
func TestWithContext(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    result, err := svc.DoSomething(ctx)
    assert.NoError(t, err)
    assert.NotNil(t, result)
}
```

### Тестирование конкурентного кода

```go
func TestConcurrent(t *testing.T) {
    svc := NewService()

    var wg sync.WaitGroup
    errors := make(chan error, 10)

    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            if err := svc.Process(id); err != nil {
                errors <- err
            }
        }(i)
    }

    wg.Wait()
    close(errors)

    for err := range errors {
        t.Errorf("Error occurred: %v", err)
    }
}
```

### Тестирование с таймаутами

```go
func TestWithTimeout(t *testing.T) {
    done := make(chan bool, 1)

    go func() {
        // Долгая операция
        svc.LongOperation()
        done <- true
    }()

    select {
    case <-done:
        // Успех
    case <-time.After(5 * time.Second):
        t.Fatal("Test timed out")
    }
}
```

## Следующие шаги

- [Лучшие практики](./best-practices.md) - Паттерны для продакшена
- [Архитектура](./architecture.md) - Понимание дизайна компонентов
- [Конфигурация](./configuration.md) - Настройка тестового окружения
