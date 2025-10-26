# Testing Guide

This guide covers how to write tests for applications built with the Tonica framework, including unit tests, integration tests, and end-to-end tests.

## Testing Philosophy

Tonica is designed to be testable:

- **Dependency Injection**: Easy to mock dependencies
- **Interface-Based**: Services implement interfaces
- **Modular Architecture**: Test components in isolation
- **Standard Library**: Uses Go's standard `testing` package

## Test Structure

Recommended test organization:

```
myservice/
├── internal/
│   ├── service/
│   │   ├── user_service.go
│   │   └── user_service_test.go      # Unit tests
│   └── repository/
│       ├── user_repository.go
│       └── user_repository_test.go
├── tests/
│   ├── integration/                   # Integration tests
│   │   └── user_api_test.go
│   └── e2e/                          # End-to-end tests
│       └── complete_flow_test.go
└── main_test.go                       # Main package tests
```

## Unit Testing

### Testing Service Implementation

```go
package service

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    hellov1 "github.com/yourusername/myservice/proto/hello/v1"
)

// Mock repository
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

// Test
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

### Testing Custom Routes

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

    // Register the route
    tonica.NewRoute(app).
        GET("/health").
        Handle(func(c *gin.Context) {
            c.JSON(200, gin.H{"status": "healthy"})
        })

    // Test it
    w := httptest.NewRecorder()
    req, _ := http.NewRequest("GET", "/health", nil)
    app.GetRouter().ServeHTTP(w, req)

    assert.Equal(t, 200, w.Code)
    assert.JSONEq(t, `{"status":"healthy"}`, w.Body.String())
}

func TestUserEndpoints(t *testing.T) {
    gin.SetMode(gin.TestMode)
    app := tonica.NewApp()

    // Register routes
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

### Testing Workers

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

    // Register activity
    env.RegisterActivity(SendEmailActivity)

    // Execute activity
    val, err := env.ExecuteActivity(SendEmailActivity, "test@example.com", "Hello")

    s.NoError(err)
    s.NotNil(val)

    var result string
    s.NoError(val.Get(&result))
    s.Equal("Email sent to test@example.com", result)
}

func (s *EmailActivitySuite) TestEmailWorkflow() {
    env := s.NewTestWorkflowEnvironment()

    // Mock activity
    env.OnActivity(SendEmailActivity, mock.Anything, "test@example.com", "Hello").
        Return("sent", nil)

    // Execute workflow
    env.ExecuteWorkflow(EmailWorkflow, "test@example.com", "Hello")

    s.True(env.IsWorkflowCompleted())
    s.NoError(env.GetWorkflowError())
}
```

### Testing Consumers

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

        // Send test message
        mockClient.messages <- &pubsub.Message{Value: []byte("test")}

        // Start consumer with timeout
        ctx, cancel := context.WithTimeout(context.Background(), time.Second)
        defer cancel()

        go cons.Start(ctx)

        // Wait for processing
        time.Sleep(100 * time.Millisecond)
        assert.True(t, processed)
    })
}
```

## Integration Testing

### Testing with Real Database

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
    // Skip if not running integration tests
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    // Setup test database
    db := service.NewDB(
        service.WithDriver(service.Sqlite),
        service.WithDSN("file::memory:?cache=shared"),
    )
    client := db.GetClient()

    // Run migrations
    err := runMigrations(client)
    require.NoError(t, err)

    // Create repository
    repo := NewUserRepository(client)

    t.Run("should create and retrieve user", func(t *testing.T) {
        user := &User{
            Name:  "Alice",
            Email: "alice@example.com",
        }

        // Create
        err := repo.Create(context.Background(), user)
        require.NoError(t, err)
        assert.NotEmpty(t, user.ID)

        // Retrieve
        retrieved, err := repo.GetByID(context.Background(), user.ID)
        require.NoError(t, err)
        assert.Equal(t, user.Name, retrieved.Name)
        assert.Equal(t, user.Email, retrieved.Email)
    })

    // Cleanup
    defer client.Close()
}
```

### Testing HTTP Server

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

    // Create app
    app := tonica.NewApp(
        tonica.WithName("test-service"),
    )

    // Register routes
    setupRoutes(app)

    // Start server in background
    go func() {
        app.Run()
    }()

    // Wait for server to start
    time.Sleep(500 * time.Millisecond)

    // Test requests
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

    // Cleanup: cancel context to stop server
    cancel()
    time.Sleep(100 * time.Millisecond)
}
```

### Testing with Docker Compose

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

    // Assumes postgres is running via docker-compose
    db := service.NewDB(
        service.WithDriver(service.Postgres),
        service.WithDSN("postgres://test:test@localhost:5432/testdb?sslmode=disable"),
    )

    ctx := context.Background()
    client := db.GetClient()

    // Test connection
    err := client.PingContext(ctx)
    require.NoError(t, err)

    // Run your tests...
    t.Run("should perform database operations", func(t *testing.T) {
        // Test implementation
    })

    // Cleanup
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

**Run tests:**
```bash
# Start dependencies
docker-compose -f docker-compose.test.yml up -d

# Run tests
go test ./tests/integration/... -v

# Cleanup
docker-compose -f docker-compose.test.yml down
```

## End-to-End Testing

### Complete User Flow

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

## Test Helpers

### Test Application Factory

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

    // Use in-memory database
    db := service.NewDB(
        service.WithDriver(service.Sqlite),
        service.WithDSN("file::memory:?cache=shared"),
    )

    // Add to app (if needed)
    // ...

    return app
}
```

### Test Fixtures

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

### Mock Tonica Components

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

## Running Tests

### Run All Tests

```bash
go test ./...
```

### Run Specific Package

```bash
go test ./internal/service
```

### Run with Coverage

```bash
go test ./... -cover

# Detailed coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Run Only Unit Tests (Skip Integration)

```bash
go test ./... -short
```

### Run Only Integration Tests

```bash
go test ./tests/integration/... -v
```

### Run with Race Detector

```bash
go test ./... -race
```

### Parallel Execution

```bash
# Run tests in parallel
go test ./... -parallel 4

# In test code
func TestSomething(t *testing.T) {
    t.Parallel()  // Enable parallel execution
    // ...
}
```

### Verbose Output

```bash
go test ./... -v
```

## Continuous Integration

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

## Testing Best Practices

### 1. Use Table-Driven Tests

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

### 2. Setup and Teardown

```go
func TestMain(m *testing.M) {
    // Setup
    setup()

    // Run tests
    code := m.Run()

    // Teardown
    teardown()

    os.Exit(code)
}

func TestWithSetup(t *testing.T) {
    // Per-test setup
    db := setupTestDB(t)
    defer db.Close()

    // Test code...
}
```

### 3. Use Subtests for Organization

```go
func TestUserService(t *testing.T) {
    svc := setupService(t)

    t.Run("Create", func(t *testing.T) {
        // Test user creation
    })

    t.Run("Get", func(t *testing.T) {
        // Test user retrieval
    })

    t.Run("Update", func(t *testing.T) {
        // Test user update
    })
}
```

### 4. Mock External Dependencies

```go
// Good: Mock external API
type MockPaymentAPI struct {
    mock.Mock
}

func (m *MockPaymentAPI) Charge(amount int) error {
    args := m.Called(amount)
    return args.Error(0)
}

// Bad: Call real external API in tests
func TestPayment(t *testing.T) {
    api := NewRealPaymentAPI() // ❌ Don't do this
    // ...
}
```

### 5. Test Error Cases

```go
func TestGetUser(t *testing.T) {
    t.Run("success case", func(t *testing.T) {
        // Test happy path
    })

    t.Run("user not found", func(t *testing.T) {
        // Test error case
    })

    t.Run("database error", func(t *testing.T) {
        // Test error case
    })

    t.Run("invalid input", func(t *testing.T) {
        // Test validation
    })
}
```

## Common Testing Patterns

### Testing with Context

```go
func TestWithContext(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    result, err := svc.DoSomething(ctx)
    assert.NoError(t, err)
    assert.NotNil(t, result)
}
```

### Testing Concurrent Code

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

### Testing with Timeouts

```go
func TestWithTimeout(t *testing.T) {
    done := make(chan bool, 1)

    go func() {
        // Long-running operation
        svc.LongOperation()
        done <- true
    }()

    select {
    case <-done:
        // Success
    case <-time.After(5 * time.Second):
        t.Fatal("Test timed out")
    }
}
```

## Next Steps

- [Best Practices](./best-practices.md) - Production patterns
- [Architecture](./architecture.md) - Understand component design
- [Configuration](./configuration.md) - Configure test environments
