# Конфигурация

В Tonica конфигурация вашего приложения выполняется в коде с помощью функциональных опций. Такой подход дает вам полный контроль над тем, как ваше приложение собирается и запускается. Основная идея заключается в том, чтобы считывать значения из переменных окружения (или других источников, таких как файлы) и передавать их в качестве опций при создании экземпляра приложения и его сервисов.

## Философия конфигурации

Tonica придерживается принципа "явное лучше неявного". Фреймворк не считывает автоматически переменные окружения. Вместо этого он предоставляет удобные хелперы, такие как `config.GetEnv`, а вы сами решаете, какие переменные окружения использовать и каким опциям их передать.

**Порядок приоритета:**

1.  **Код:** Значения, жестко закодированные или переданные в функции опций, имеют наивысший приоритет.
2.  **Переменные окружения:** Ваша логика в `main.go` считывает переменные окружения и передает их в код.
3.  **Значения по умолчанию:** Хелпер `config.GetEnv` и некоторые опции имеют запасные значения.

## Быстрый старт: Пример конфигурации

Вот как выглядит типичная структура конфигурации в `main.go`:

```go
package main

import (
    "github.com/tonica-go/tonica/pkg/tonica"
    "github.com/tonica-go/tonica/pkg/tonica/config"
    "github.com/tonica-go/tonica/pkg/tonica/service"
    // ... импорты ваших сервисов
)

func main() {
    // 1. Создаем конфигурацию приложения
    appConfig := config.NewConfig(
        // Устанавливаем режим запуска из переменной окружения APP_MODE
        config.WithRunMode(config.GetEnv("APP_MODE", config.ModeAIO)),
    )

    // 2. Создаем экземпляр приложения
    app := tonica.NewApp(
        // Передаем созданную конфигурацию
        tonica.WithConfig(appConfig),
        // Устанавливаем имя приложения
        tonica.WithName(config.GetEnv("APP_NAME", "MyApp")),
    )

    // 3. Конфигурируем и регистрируем сервисы
    
    // Считываем DSN и драйвер из переменных окружения
    dbDSN := config.GetEnv("DB_DSN", "user:pass@tcp(localhost:3306)/db?parseTime=true")
    dbDriver := config.GetEnv("DB_DRIVER", service.Mysql)

    // Считываем адрес Redis
    redisAddr := config.GetEnv("REDIS_ADDR", "localhost:6379")

    // Создаем сервис с подключением к БД и Redis
    paymentSvc := service.NewService(
        service.WithName("payment-service"),
        service.WithDB(dbDSN, dbDriver),
        service.WithRedis(redisAddr, "", 0),
        service.WithGRPC(payment.RegisterGRPC), // ваш gRPC регистратор
        service.WithGateway(payment.RegisterGateway), // ваш Gateway регистратор
    )
    
    // Регистрируем сервис в приложении
    app.GetRegistry().MustRegisterService(paymentSvc)

    // 4. Запускаем приложение
    if err := app.Run(); err != nil {
        app.GetLogger().Fatal(err)
    }
}
```

## Конфигурация приложения (`tonica.App`)

Основной экземпляр приложения создается с помощью `tonica.NewApp(options ...AppOption)`.

### Основные опции `AppOption`

| Опция | Описание | Пример |
| --- | --- | --- |
| `WithName(string)` | Устанавливает имя приложения. Используется для логирования и метрик. | `tonica.WithName("user-service")` |
| `WithConfig(*config.Config)` | Применяет конфигурацию запуска (режим, список сервисов). **Очень важная опция.** | `tonica.WithConfig(appConfig)` |
| `WithSpec(string)` | Указывает путь к файлу спецификации OpenAPI. | `tonica.WithSpec("openapi/spec.json")` |
| `WithSpecUrl(string)` | Задает URL, по которому будет доступна спецификация. | `tonica.WithSpecUrl("/swagger.json")` |
| `WithAPIPrefix(string)` | Добавляет глобальный префикс ко всем HTTP-маршрутам. | `tonica.WithAPIPrefix("/api/v1")` |
| `WithLogger(*log.Logger)` | Позволяет использовать собственный логгер. | `tonica.WithLogger(myLogger)` |

### Конфигурация запуска (`config.Config`)

Эта конфигурация определяет, *как* ваше приложение будет работать. Она создается с помощью `config.NewConfig(options ...Option)`.

| Опция | Описание | Переменная окружения | Пример |
| --- | --- | --- | --- |
| `WithRunMode(string)` | Устанавливает режим запуска приложения. | `APP_MODE` | `config.WithRunMode(config.ModeService)` |
| `WithServices([]string)` | В режиме `service` указывает, какие именно сервисы запускать. | `APP_SERVICES` | `config.WithServices([]string{"auth", "users"})` |
| `WithWorkers([]string)` | В режиме `worker` указывает, какие воркеры запускать. | `APP_WORKERS` | `config.WithWorkers([]string{"emails", "reports"})` |
| `WithConsumers([]string)` | В режиме `consumer` указывает, какие консьюмеры запускать. | `APP_CONSUMERS` | `config.WithConsumers([]string{"orders"})` |
| `WithDebugMode(bool)` | Включает/выключает режим отладки. | `APP_DEBUG` | `config.WithDebugMode(true)` |

## Режимы запуска (`Run Modes`)

Режим запуска — это ключевая концепция в Tonica, которая позволяет вам использовать одну и ту же кодовую базу для разных типов развертывания. Режим устанавливается через опцию `config.WithRunMode` и обычно контролируется переменной окружения `APP_MODE`.

| Режим | `APP_MODE` | Описание |
| --- | --- | --- |
| **All-In-One** | `aio` | **(По умолчанию)**. Запускает все зарегистрированные компоненты (сервисы, воркеры, консьюмеры) в одном процессе. Идеально для разработки и простых развертываний. |
| **Service** | `service` | Запускает только указанные gRPC сервисы и их HTTP-шлюзы. Используйте `APP_SERVICES` (через запятую), чтобы указать, какие именно. |
| **Worker** | `worker` | Запускает только указанные воркеры Temporal. Используйте `APP_WORKERS` для выбора. |
| **Consumer** | `consumer` | Запускает только указанные консьюмеры сообщений (например, Kafka). Используйте `APP_CONSUMERS` для выбора. |
| **Gateway** | `gateway` | Запускает только HTTP-шлюзы для всех зарегистрированных gRPC сервисов, но не сами gRPC серверы. Полезно для развертывания API Gateway как отдельного компонента. |

## Конфигурация сервиса (`service.Service`)

Каждый сервис в вашем приложении создается с помощью `service.NewService(options ...Option)`.

### Основные опции `service.Option`

| Опция | Описание | Пример |
| --- | --- | --- |
| `WithName(string)` | **Обязательно.** Уникальное имя сервиса. | `service.WithName("payment-service")` |
| `WithGRPC(GRPCRegistrar)` | **Обязательно.** Регистрирует вашу реализацию gRPC сервера. | `service.WithGRPC(RegisterPaymentService)` |
| `WithGateway(GatewayRegistrar)` | Регистрирует HTTP-шлюз (gRPC-Gateway) для вашего сервиса. | `service.WithGateway(RegisterPaymentGateway)` |
| `WithGRPCAddr(string)` | Устанавливает адрес для gRPC сервера (`host:port`). | `service.WithGRPCAddr(":9001")` |

### Подключение к базам данных и кэшу

Подключения настраиваются для каждого сервиса индивидуально.

#### База данных

Для подключения к базе данных используется опция `WithDB`. Tonica "из коробки" поддерживает PostgreSQL, MySQL и SQLite с помощью `bun` ORM и автоматически интегрирует OpenTelemetry для трассировки запросов.

```go
// Считываем DSN и драйвер из переменных окружения
dbDSN := config.GetEnv("DB_DSN", "user:pass@tcp(localhost:3306)/db?parseTime=true")
dbDriver := config.GetEnv("DB_DRIVER", service.Postgres) // service.Postgres, service.Mysql, service.Sqlite

svc := service.NewService(
    // ...другие опции
    service.WithDB(dbDSN, dbDriver),
)
```

| Драйвер | Константа | Пример DSN |
| --- | --- | --- |
| PostgreSQL | `service.Postgres` | `postgres://user:pass@host:5432/db?sslmode=disable` |
| MySQL | `service.Mysql` | `user:pass@tcp(host:3306)/db?parseTime=true` |
| SQLite | `service.Sqlite` | `file:data.db?cache=shared` |

#### Redis

Для подключения к Redis используется опция `WithRedis`.

```go
redisAddr := config.GetEnv("REDIS_ADDR", "localhost:6379")
redisPassword := config.GetEnv("REDIS_PASSWORD", "")
redisDB := config.GetEnvInt("REDIS_DB", 0) // Используем GetEnvInt для числовых значений

svc := service.NewService(
    // ...другие опции
    service.WithRedis(redisAddr, redisPassword, redisDB),
)
```

## Переменные окружения

Вот сводка наиболее часто используемых переменных окружения.

| Переменная | Описание | Значение по умолчанию |
| --- | --- | --- |
| `APP_NAME` | Имя вашего приложения. | `"Tonica"` |
| `APP_MODE` | Режим запуска приложения. | `"aio"` |
| `APP_SERVICES` | Список сервисов для запуска в режиме `service`. | `""` |
| `APP_WORKERS` | Список воркеров для запуска в режиме `worker`. | `""` |
| `APP_CONSUMERS` | Список консьюмеров для запуска в режиме `consumer`. | `""` |
| `APP_PORT` | Порт для основного HTTP сервера (шлюзы, кастомные роуты). | `"8080"` |
| `GRPC_PORT` | Порт для gRPC сервера (если не задан `WithGRPCAddr`). | `"50051"` |
| `METRICS_PORT` | Порт для эндпоинта метрик Prometheus. | `"9090"` |
| `DB_DSN` | Data Source Name для подключения к БД. | `""` |
| `DB_DRIVER` | Драйвер базы данных (`postgres`, `mysql`, `sqlite`). | `"postgres"` |
| `REDIS_ADDR` | Адрес сервера Redis (`host:port`). | `"localhost:6379"` |
| `REDIS_PASSWORD` | Пароль для Redis. | `""` |
| `REDIS_DB` | Номер базы данных Redis. | `0` |
| `LOG_LEVEL` | Уровень логирования (`debug`, `info`, `warn`, `error`). | `"info"` |
| `LOG_FORMAT` | Формат логов (`text` или `json`). | `"text"` |

## Наблюдаемость (Observability)

Tonica имеет встроенную поддержку логирования, метрик и трассировки.

### Логирование

Логирование настраивается через переменные окружения:
- `LOG_LEVEL`: Установите `debug` для разработки и `info` для продакшена.
- `LOG_FORMAT`: Установите `text` для локальной разработки и `json` для продакшена, чтобы логи было легко парсить.

### Метрики

Метрики в формате Prometheus доступны по умолчанию на порту, заданном переменной `METRICS_PORT` (по умолчанию `:9090`).

### Трассировка (OpenTelemetry)

Трассировка включается и настраивается через стандартные переменные окружения OpenTelemetry:

| Переменная | Описание | Пример |
| --- | --- | --- |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Адрес коллектора OpenTelemetry (например, Jaeger, Tempo). | `localhost:4317` |
| `OTEL_SERVICE_NAME` | Имя сервиса для трассировки (обычно совпадает с `APP_NAME`). | `payment-service` |
| `OTEL_TRACES_EXPORTER` | Укажите `otlp` для экспорта. | `otlp` |
| `OTEL_EXPORTER_OTLP_PROTOCOL` | Протокол экспортера (`grpc` или `http/protobuf`). | `grpc` |
| `OTEL_SDK_DISABLED` | Установите в `true`, чтобы полностью отключить трассировку. | `false` |