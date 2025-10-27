# Руководство по конфигурации Tonica

Это руководство охватывает все параметры конфигурации, доступные в Tonica, включая переменные окружения, конфигурацию на основе кода и лучшие практики.

## Приоритет конфигурации

Tonica следует этому порядку приоритета (от высшего к низшему):

1. **Код** - Опции, переданные в конструкторы
2. **Переменные окружения** - Конфигурация на основе окружения
3. **Значения по умолчанию** - Встроенные разумные значения по умолчанию

Пример:
```go
// Приоритет 1: Код (высший)
app := tonica.NewApp(tonica.WithName("my-service"))

// Приоритет 2: Переменная окружения
// export APP_NAME="env-service"

// Приоритет 3: По умолчанию (низший)
// Откатывается к "tonica-app"
```

## Конфигурация приложения

### Опции приложения

Настройка основного приложения:

```go
app := tonica.NewApp(
    tonica.WithName("my-service"),              // Имя приложения
    tonica.WithSpec("openapi/spec.json"),       // Путь к OpenAPI спецификации
    tonica.WithSpecUrl("/swagger.json"),        // Пользовательский URL спецификации
    tonica.WithConfig(customConfig),            // Пользовательский объект конфигурации
    tonica.WithLogger(customLogger),            // Пользовательский логгер
    tonica.WithRegistry(customRegistry),        // Пользовательский реестр
)
```

#### WithName

Устанавливает имя приложения (используется в метриках, логировании и т.д.).

```go
tonica.WithName("user-service")
```

**Переменная окружения:**
```bash
export APP_NAME="user-service"
```

**По умолчанию:** `"tonica-app"`

#### WithSpec

Устанавливает путь к файлу спецификации OpenAPI.

```go
tonica.WithSpec("openapi/myservice/v1/myservice.swagger.json")
```

**Переменная окружения:**
```bash
export OPENAPI_SPEC="openapi/myservice/v1/myservice.swagger.json"
```

**По умолчанию:** `""` (спецификация не загружена)

#### WithSpecUrl

Устанавливает URL-путь, по которому будет доступна спецификация OpenAPI.

```go
tonica.WithSpecUrl("/api-spec.json")
```

**По умолчанию:** `"/openapi.json"`

### Порты сервера

Настройка портов HTTP, gRPC и метрик:

**Переменные окружения:**
```bash
# Порт HTTP/REST сервера
export APP_PORT="8080"          # По умолчанию: 8080

# Порт gRPC сервера
export GRPC_PORT="50051"        # По умолчанию: 50051

# Порт для метрик
export METRICS_PORT="9090"      # По умолчанию: 9090
```

**Пример:**
```bash
# Запуск на пользовательских портах
export APP_PORT="3000"
export GRPC_PORT="9000"
export METRICS_PORT="9100"
```

### Конфигурация CORS

Настройка совместного использования ресурсов между источниками (Cross-Origin Resource Sharing):

**Переменные окружения:**
```bash
# Разрешить все источники (по умолчанию)
# Конфигурация не требуется

# Ограничить конкретными источниками
export APP_CORS_ORIGINS="https://myapp.com,https://admin.myapp.com"
```

**По умолчанию:** Разрешены все источники

**Разрешенные методы:** GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS

**Разрешенные заголовки:** Origin, Content-Length, Content-Type, Authorization

## Конфигурация сервисов

Настройка gRPC сервисов:

```go
svc := tonica.NewService(
    tonica.WithServiceName("UserService"),      // Имя сервиса
    tonica.WithDB(db),                          // Клиент базы данных
    tonica.WithRedis(redis),                    // Клиент Redis
)
```

### WithServiceName

Устанавливает имя сервиса для регистрации.

```go
tonica.WithServiceName("UserService")
```

**Обязательно:** Да (для регистрации)

### WithDB

Присоединяет клиент базы данных к сервису.

```go
db := tonica.NewDB(...)
svc := tonica.NewService(tonica.WithDB(db))
```

### WithRedis

Присоединяет клиент Redis к сервису.

```go
redis := tonica.NewRedis(...)
svc := tonica.NewService(tonica.WithRedis(redis))
```

## Конфигурация базы данных

Tonica поддерживает PostgreSQL, MySQL и SQLite через Bun ORM.

### PostgreSQL

```go
db := tonica.NewDB(
    tonica.WithDriver(tonica.Postgres),
    tonica.WithDSN("postgres://user:password@localhost:5432/dbname?sslmode=disable"),
)
```

**Переменные окружения:**
```bash
export DB_DRIVER="postgres"
export DB_DSN="postgres://user:password@localhost:5432/dbname?sslmode=disable"
```

**Формат DSN:**
```
postgres://username:password@host:port/database?sslmode=disable
```

**Опции:**
- `sslmode`: `disable`, `require`, `verify-ca`, `verify-full`
- `connect_timeout`: Таймаут подключения в секундах
- `application_name`: Имя приложения для логирования

**Пример:**
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

**Переменные окружения:**
```bash
export DB_DRIVER="mysql"
export DB_DSN="user:password@tcp(localhost:3306)/dbname?parseTime=true"
```

**Формат DSN:**
```
username:password@tcp(host:port)/database?parseTime=true
```

**Важные опции:**
- `parseTime=true`: **Обязательно** для правильной обработки времени
- `charset=utf8mb4`: Набор символов (рекомендуется)
- `loc=Local`: Часовой пояс

**Пример:**
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

**Переменные окружения:**
```bash
export DB_DRIVER="sqlite"
export DB_DSN="file:./data/mydb.db?cache=shared&mode=rwc"
```

**Формат DSN:**
```
file:path/to/database.db?cache=shared&mode=rwc
```

**Опции:**
- `cache`: `shared` (множественные подключения) или `private`
- `mode`: `ro` (только чтение), `rw` (чтение-запись), `rwc` (чтение-запись-создание)

**Пример:**
```bash
export DB_DSN="file:/var/lib/myapp/data.db?cache=shared&mode=rwc"
```

### Опции базы данных

#### WithDriver

Устанавливает драйвер базы данных.

```go
tonica.WithDriver(tonica.Postgres)  // PostgreSQL
tonica.WithDriver(tonica.Mysql)     // MySQL
tonica.WithDriver(tonica.Sqlite)    // SQLite
```

**Переменная окружения:**
```bash
export DB_DRIVER="postgres"  # или "mysql", "sqlite"
```

**По умолчанию:** `"postgres"`

#### WithDSN

Устанавливает строку подключения к базе данных.

```go
tonica.WithDSN("postgres://localhost/mydb")
```

**Переменная окружения:**
```bash
export DB_DSN="postgres://localhost/mydb"
```

**Обязательно:** Да (если используется база данных)

### Пул соединений

Настройка пула соединений (через Bun):

```go
db := tonica.NewDB(...)
client := db.GetClient()

// Настройка пула
client.SetMaxOpenConns(25)                        // Максимальное количество открытых соединений
client.SetMaxIdleConns(10)                        // Максимальное количество неактивных соединений
client.SetConnMaxLifetime(5 * time.Minute)        // Время жизни соединения
client.SetConnMaxIdleTime(10 * time.Minute)       // Максимальное время простоя
```

**Рекомендуемые настройки:**

**Для API сервисов (высокая конкурентность):**
```go
client.SetMaxOpenConns(100)
client.SetMaxIdleConns(25)
client.SetConnMaxLifetime(5 * time.Minute)
```

**Для воркеров (низкая конкурентность):**
```go
client.SetMaxOpenConns(10)
client.SetMaxIdleConns(5)
client.SetConnMaxLifetime(10 * time.Minute)
```

## Конфигурация Redis

Настройка подключения к Redis:

```go
redis := tonica.NewRedis(
    tonica.WithRedisAddr("localhost:6379"),     // Адрес Redis
    tonica.WithRedisPassword("secret"),         // Пароль (опционально)
    tonica.WithRedisDB(0),                      // Номер базы данных
)
```

### Опции Redis

#### WithRedisAddr

Устанавливает адрес сервера Redis.

```go
tonica.WithRedisAddr("redis.example.com:6379")
```

**Переменная окружения:**
```bash
export REDIS_ADDR="redis.example.com:6379"
```

**По умолчанию:** `"localhost:6379"`

#### WithRedisPassword

Устанавливает пароль Redis (если требуется).

```go
tonica.WithRedisPassword("my-secret-password")
```

**Переменная окружения:**
```bash
export REDIS_PASSWORD="my-secret-password"
```

**По умолчанию:** `""` (без пароля)

#### WithRedisDB

Устанавливает номер базы данных Redis (0-15).

```go
tonica.WithRedisDB(2)
```

**Переменная окружения:**
```bash
export REDIS_DB="2"
```

**По умолчанию:** `0`

### Пул соединений Redis

Настройка пула соединений (через go-redis):

```go
client := redis.GetClient()

// Настройка опций пула
client.Options().PoolSize = 10           // Размер пула
client.Options().MinIdleConns = 5        // Минимальное количество неактивных соединений
client.Options().MaxConnAge = 0          // Максимальный возраст соединения (0 = без ограничений)
client.Options().PoolTimeout = 4 * time.Second
client.Options().IdleTimeout = 5 * time.Minute
```

## Конфигурация Temporal

Настройка воркеров Temporal:

```go
worker := tonica.NewWorker(
    tonica.WithWorkerName("email-worker"),                    // Имя воркера
    tonica.WithTaskQueue("email-tasks"),                      // Очередь задач
    tonica.WithTemporalHost("temporal.example.com:7233"),     // Хост Temporal
    tonica.WithTemporalNamespace("production"),               // Пространство имен
    tonica.WithMaxConcurrentActivities(10),                   // Конкурентность
)
```

### Опции Temporal

#### WithWorkerName

Устанавливает имя воркера.

```go
tonica.WithWorkerName("report-generator")
```

**Обязательно:** Да

#### WithTaskQueue

Устанавливает имя очереди задач Temporal.

```go
tonica.WithTaskQueue("report-tasks")
```

**Обязательно:** Да

#### WithTemporalHost

Устанавливает адрес сервера Temporal.

```go
tonica.WithTemporalHost("temporal.example.com:7233")
```

**Переменная окружения:**
```bash
export TEMPORAL_HOST="temporal.example.com:7233"
```

**По умолчанию:** `"localhost:7233"`

#### WithTemporalNamespace

Устанавливает пространство имен Temporal.

```go
tonica.WithTemporalNamespace("production")
```

**Переменная окружения:**
```bash
export TEMPORAL_NAMESPACE="production"
```

**По умолчанию:** `"default"`

#### WithMaxConcurrentActivities

Устанавливает максимальное количество одновременно выполняемых активностей.

```go
tonica.WithMaxConcurrentActivities(20)
```

**По умолчанию:** `10`

**Рекомендации:**
- Активности с вводом-выводом (email, API-вызовы): 20-100
- Активности с нагрузкой на CPU (обработка изображений, отчеты): 2-5
- Активности с большим потреблением памяти: 1-3

## Конфигурация консьюмеров

Настройка консьюмеров сообщений:

```go
consumer := tonica.NewConsumer(
    tonica.WithConsumerName("order-processor"),       // Имя консьюмера
    tonica.WithTopic("orders"),                       // Имя топика
    tonica.WithConsumerGroup("order-handlers"),       // Группа консьюмеров
    tonica.WithPubSubClient(client),                  // Клиент PubSub
    tonica.WithHandler(handleOrder),                  // Обработчик сообщений
)
```

### Опции консьюмера

#### WithConsumerName

Устанавливает имя консьюмера.

```go
tonica.WithConsumerName("payment-processor")
```

**Обязательно:** Да

#### WithTopic

Устанавливает топик для потребления.

```go
tonica.WithTopic("payments")
```

**Обязательно:** Да

#### WithConsumerGroup

Устанавливает имя группы консьюмеров.

```go
tonica.WithConsumerGroup("payment-handlers")
```

**По умолчанию:** `""` (без группы консьюмеров)

#### WithPubSubClient

Устанавливает клиент PubSub/Kafka.

```go
tonica.WithPubSubClient(pubsubClient)
```

**Обязательно:** Да

#### WithHandler

Устанавливает функцию-обработчик сообщений.

```go
tonica.WithHandler(func(ctx context.Context, msg *pubsub.Message) error {
    // Обработка сообщения
    return nil
})
```

**Обязательно:** Да

## Конфигурация наблюдаемости

### OpenTelemetry

Настройка распределенной трассировки:

**Переменные окружения:**
```bash
# Включить/отключить трассировку
export OTEL_ENABLED="true"          # По умолчанию: false

# OTLP endpoint
export OTEL_ENDPOINT="localhost:4317"

# Имя сервиса (переопределяет APP_NAME)
export OTEL_SERVICE_NAME="user-service"

# Частота сэмплирования (0.0 до 1.0)
export OTEL_TRACE_SAMPLING="1.0"    # 100% сэмплирование
```

**Пример (Jaeger):**
```bash
export OTEL_ENABLED="true"
export OTEL_ENDPOINT="jaeger:4317"
export OTEL_SERVICE_NAME="user-service"
```

**Пример (Honeycomb):**
```bash
export OTEL_ENABLED="true"
export OTEL_ENDPOINT="api.honeycomb.io:443"
export OTEL_SERVICE_NAME="user-service"
export OTEL_HEADERS="x-honeycomb-team=YOUR_API_KEY"
```

### Логирование

Настройка структурированного логирования:

**Уровни логирования:**
```bash
# Установить уровень логирования
export LOG_LEVEL="debug"     # debug, info, warn, error
```

**Формат логов:**
```bash
# JSON формат (для продакшена)
export LOG_FORMAT="json"

# Текстовый формат (для разработки)
export LOG_FORMAT="text"
```

**Пример:**
```bash
# Разработка
export LOG_LEVEL="debug"
export LOG_FORMAT="text"

# Продакшен
export LOG_LEVEL="info"
export LOG_FORMAT="json"
```

### Метрики

Метрики всегда включены на порту 9090 по умолчанию.

**Настройка порта:**
```bash
export METRICS_PORT="9100"
```

**Отключение метрик:**
```go
// Не рекомендуется - метрики легковесны
// Чтобы отключить, не открывайте порт 9090 извне
```

## Полные примеры конфигурации

### Окружение разработки

```bash
# Приложение
export APP_NAME="myservice-dev"
export APP_PORT="8080"
export GRPC_PORT="50051"
export METRICS_PORT="9090"

# База данных
export DB_DRIVER="sqlite"
export DB_DSN="file:./dev.db?cache=shared&mode=rwc"

# Redis
export REDIS_ADDR="localhost:6379"
export REDIS_PASSWORD=""
export REDIS_DB="0"

# Temporal
export TEMPORAL_HOST="localhost:7233"
export TEMPORAL_NAMESPACE="default"

# Логирование
export LOG_LEVEL="debug"
export LOG_FORMAT="text"

# Трассировка (отключена)
export OTEL_ENABLED="false"
```

### Окружение продакшена

```bash
# Приложение
export APP_NAME="myservice"
export APP_PORT="8080"
export GRPC_PORT="50051"
export METRICS_PORT="9090"

# База данных
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

# Логирование
export LOG_LEVEL="info"
export LOG_FORMAT="json"

# Трассировка
export OTEL_ENABLED="true"
export OTEL_ENDPOINT="tempo.prod:4317"
export OTEL_SERVICE_NAME="myservice"
export OTEL_TRACE_SAMPLING="0.1"  # 10% сэмплирование
```

### Docker Compose

```yaml
version: '3.8'

services:
  app:
    image: myservice:latest
    environment:
      # Приложение
      APP_NAME: myservice
      APP_PORT: 8080
      GRPC_PORT: 50051
      METRICS_PORT: 9090

      # База данных
      DB_DRIVER: postgres
      DB_DSN: postgres://myuser:mypass@postgres:5432/mydb?sslmode=disable

      # Redis
      REDIS_ADDR: redis:6379
      REDIS_PASSWORD: ""
      REDIS_DB: 0

      # Temporal
      TEMPORAL_HOST: temporal:7233
      TEMPORAL_NAMESPACE: default

      # Наблюдаемость
      LOG_LEVEL: info
      LOG_FORMAT: json
      OTEL_ENABLED: "true"
      OTEL_ENDPOINT: jaeger:4317

    ports:
      - "8080:8080"    # HTTP
      - "50051:50051"  # gRPC
      - "9090:9090"    # Метрики

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

## Лучшие практики конфигурации

### 1. Никогда не используйте секреты в коде

❌ **Плохо:**
```go
db := tonica.NewDB(
    tonica.WithDSN("postgres://admin:password123@localhost/db"),
)
```

✅ **Хорошо:**
```go
dsn := os.Getenv("DB_DSN")
if dsn == "" {
    log.Fatal("DB_DSN is required")
}
db := tonica.NewDB(tonica.WithDSN(dsn))
```

### 2. Валидируйте конфигурацию

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

### 3. Используйте файлы для разных окружений

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

### 4. Документируйте обязательные переменные

Создайте файл `.env.example`:

```bash
# Приложение
APP_NAME=myservice
APP_PORT=8080
GRPC_PORT=50051

# База данных (обязательно)
DB_DRIVER=postgres
DB_DSN=postgres://user:pass@host:5432/db

# Redis (опционально)
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# Temporal (обязательно для воркеров)
TEMPORAL_HOST=localhost:7233
TEMPORAL_NAMESPACE=default
```

### 5. Разумно используйте значения по умолчанию

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

## Устранение неполадок

### Проблемы с подключением к базе данных

**Ошибка:** `connection refused`
```bash
# Проверьте, запущена ли база данных
docker ps | grep postgres

# Проверьте подключение
psql postgres://user:pass@localhost:5432/db
```

**Ошибка:** `authentication failed`
```bash
# Проверьте учетные данные
echo $DB_DSN

# Проверьте логи PostgreSQL
docker logs postgres-container
```

### Проблемы с подключением к Redis

**Ошибка:** `dial tcp: connection refused`
```bash
# Проверьте, запущен ли Redis
docker ps | grep redis

# Проверьте подключение
redis-cli -h localhost -p 6379 ping
```

### Конфликты портов

**Ошибка:** `bind: address already in use`
```bash
# Узнайте, что использует порт
lsof -i :8080

# Используйте другой порт
export APP_PORT="8081"
```

## Следующие шаги

- [Пользовательские маршруты](./custom-routes.md) - Добавление пользовательских HTTP маршрутов
- [Тестирование](./testing.md) - Тестирование вашей конфигурации
- [Лучшие практики](./best-practices.md) - Паттерны конфигурации для продакшена
