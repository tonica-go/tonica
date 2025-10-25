# Tonica framework

> ### ВСЕ ЕЩЕ В РАЗРАБОТКЕ! 
> Не готово для полноценного использования.


ToDo:

- [ ] Консьюмеры (kafka, pubsub)
- [ ] Темпорал воркеры
- [ ] Коннекторы к БД (bun, sqlc)
- [ ] Миграции (bun)
- [ ] Редис
- [ ] Рейтлимитер
- [ ] GCS, S3 клиенты
- [ ] Аутентификация
- [ ] Документация
- [ ] Примеры
- [ ] air
- [ ] Docker
- [ ] Helm
- [ ] Compose



<img alt="tonica mascot" src="docs/tonica-gofer.webp" />

Вдохновлено gofr.dev. Tonica — легкий фреймворк на Go для сервисной архитектуры. Он позволяет запускать несколько микросервисов в одном бинаре (AIO) для локальной разработки и предпросмотра, а также изолированно — для продакшена и масштабирования.

Код разделен на четкие компоненты (сервисы, реестр, наблюдаемость, метрики), поэтому проекты остаются читаемыми и расширяемыми.

## Возможности

- HTTP API на базе `gin`
- gRPC‑сервисы с включенной наблюдаемостью
- Автоматический REST‑шлюз для gRPC через `grpc-gateway` (`/v1/*`)
- Документация `/docs` из OpenAPI‑спеки (`tonica.WithSpec`)
- Отдельный HTTP сервер метрик и профайлинга (`/metrics`, pprof)
- Встроенные метрики и трассировка (Prometheus, OpenTelemetry)
- Регистрация и запуск нескольких сервисов через единый реестр
- CLI‑утилита для генерации обвязки по `.proto`

## Установка

- Требуется Go `1.25`.
- Подключить модуль: `go get github.com/tonica-go/tonica@latest`.

## Как это работает

Архитектура строится вокруг `App`, `Registry` и описаний сервисов:

- `App` — оболочка запуска: HTTP, gRPC, метрики, трассировка.
- `Registry` — хранит зарегистрированные сервисы, воркеры и консьюмеры.
- `service.Service` — описание конкретного сервиса: имя, gRPC‑регистратор, адрес, опциональный REST‑шлюз.

В AIO‑режиме Tonica поднимает:

- gRPC сервер(а) для каждого сервиса (адрес задается на сервисе),
- HTTP сервер приложение: `/v1/*` (REST‑шлюз к gRPC), `/docs` (если задана спека),
- отдельный HTTP сервер метрик: `/metrics`, `/healthz`, `/readyz`, pprof.

Смотрите рабочий пример: `example/dev/main.go:1`,
реализации сервисов: `example/dev/services/payment/service.go:1`, `example/dev/services/report/service.go:1`.

## Быстрый старт

1) Создайте `App` и конфигурацию:
- `config.WithRunMode("aio"|"service"|"worker"|"consumer")` — режим работы (по умолчанию `aio`).
- `config.WithServices([]string{"paymentservice-service"})` — список сервисов для режима `service`.
- `tonica.WithSpec("path/to/openapi.json")` — подключить `/docs` (опционально).

2) Зарегистрируйте сервисы в реестре:
- `service.WithName(paymentv1.ServiceName)` — уникальное имя (генерируется из `.proto`).
- `service.WithGRPC(payment.RegisterGRPC)` — функция, которая регистрирует gRPC эндпоинты в `*grpc.Server`.
- `service.WithGRPCAddr(":9000")` — адрес gRPC сервера сервиса.
- `service.WithGateway(payment.RegisterGateway)` — регистрация REST‑шлюза (если нужен HTTP доступ).

3) Запустите приложение: `app.Run()`.

Минимально воспроизводимый пример есть в `example/dev`.

## Режимы запуска

- `aio` — все сервисы + HTTP‑шлюз + метрики в одном процессе.
- `service` — поднимаются только сервисы, перечисленные в `config.WithServices` или переменной окружения `APP_SERVICES` (через запятую).
- `worker`, `consumer` — заготовки под Temporal/стриминг (в разработке).

Переменная окружения `APP_MODE` может переопределить режим, например: `APP_MODE=service`.

## HTTP и REST‑шлюз

- HTTP‑сервер приложения слушает `APP_HTTP_ADDR` (по умолчанию `:8080`).
- CORS по умолчанию открыт; можно задать разрешенные источники через `PS_CORS_ORIGINS` (через запятую).
- REST‑шлюз проксирует в gRPC на `/v1/*` и пробрасывает заголовки `authorization`, `traceparent`, `tracestate`, `x-request-id`.
- Документация доступна по `/docs`, если задан `tonica.WithSpec(".../openapi.swagger.json")`.

## gRPC

- Каждый сервис поднимает собственный gRPC‑сервер на своём адресе (см. `service.WithGRPCAddr`).
- Генератор обвязки по `.proto` создаёт полезные константы, например `ServiceName` и `ServiceAddrEnvName` для адреса сервиса.
- Пример сгенерированных файлов: `example/dev/proto/payment/v1/paymentservice_grpc.go:1`.

## Наблюдаемость и метрики

- Трассировка: задайте `OTEL_EXPORTER_OTLP_ENDPOINT` (gRPC), уровень логов — `LOG_LEVEL` (`debug|info|warn|error`).
- Логи: `slog` (локально — текст при `PS_APP_ENV=local`, иначе JSON).
- Метрики и профайлинг: отдельный HTTP‑сервер на `APP_METRIC_ADDR` (по умолчанию `:2121`), ручки `/metrics`, `/healthz`, `/readyz` и pprof.
- Из коробки регистрируются гистограммы и счётчики для HTTP, gRPC, Redis/SQL и Pub/Sub, например:
  - `app_http_response`, `app_http_service_response`
  - `app_sql_stats`, `app_redis_stats`
  - `app_pubsub_*`

## CLI: генерация кода по .proto

- Команда: `go run ./pkg/tonica/cmd/wrap --proto <path/to/service.proto>`.
- Генерирует рядом с `.proto` вспомогательные файлы для клиента/сервера и конфигурации адресов.
- В примере уже сгенерированы: `example/dev/proto/payment/v1/paymentservice_grpc.go:1`, `example/dev/proto/reports/v1/reportsservice_grpc.go:1`.

## Переменные окружения (основные)

- `APP_MODE` — `aio|service|worker|consumer`.
- `APP_HTTP_ADDR` — адрес HTTP‑сервера приложения, по умолчанию `:8080`.
- `APP_METRIC_ADDR` — адрес сервера метрик, по умолчанию `:2121`.
- `APP_SERVICES` — список сервисов для режима `service`, например `paymentservice-service,reportsservice-service`.
- `OTEL_EXPORTER_OTLP_ENDPOINT` — endpoint OTLP для трассировки.
- `LOG_LEVEL` — `debug|info|warn|error`.
- `PS_APP_ENV` — формат логов (`local` — текстовый).
- `PS_CORS_ORIGINS` — разрешенные источники CORS (через запятую).

## Технологии

- `gin`, `grpc`, `grpc-gateway`
- OpenTelemetry (`otel`), Prometheus
- Kafka/PubSub, Temporal (подключаемо)

## Ограничения

Фреймворк ориентирован на практичные кейсы сервисной архитектуры и быстрый запуск окружений. Универсальность не была целью; часть направлений (воркеры/консьюмеры) помечены как WIP.

---

Если хотите, можно начать с примера: `go run example/dev/main.go` и открыть `http://localhost:8080/docs` (при наличии `example/dev/openapi/openapi.swagger.json`).

Тулинг для локальной разработки

```bash
brew install buf

go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest

go install github.com/tonica-go/tonica@latest

mkdir example1 && cd example1
tonica init --name=example1
tonica proto --name=example1
go mod init example1
go mod tidy
go run .

```