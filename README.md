# Tonica framework

> ## Still in development!
> Not ready for full production use.


<img alt="tonica mascot" src="docs/tonica-gofer.webp" />

Inspired by gofr.dev. Tonica is a lightweight Go framework for service-oriented architecture. It lets you run multiple microservices in a single binary (AIO) for local development and preview, as well as run them in isolation for production and scaling.

The code is split into clear components (services, registry, observability, metrics) so projects remain readable and extensible.

## Features

- HTTP API powered by `gin`
- gRPC services with observability enabled
- Automatic REST gateway for gRPC via `grpc-gateway` (`/v1/*`)
- `/docs` API documentation from an OpenAPI spec (`tonica.WithSpec`)
- Separate HTTP server for metrics and profiling (`/metrics`, pprof)
- Built-in metrics and tracing (Prometheus, OpenTelemetry)
- Register and run multiple services via a single registry
- CLI tool to generate glue code from `.proto`

## Installation

- Requires Go `1.25`.
- Add the module: `go get github.com/tonica-go/tonica@latest`.

## How it works

The architecture centers around `App`, `Registry`, and service definitions:

- `App` — the runtime shell: HTTP, gRPC, metrics, tracing.
- `Registry` — stores registered services, workers, and consumers.
- `service.Service` — describes a specific service: name, gRPC registrar, address, optional REST gateway.

In AIO mode Tonica starts:

- A gRPC server for each service (address is set on the service),
- The application HTTP server: `/v1/*` (REST gateway to gRPC), `/docs` (if a spec is provided),
- A separate HTTP server for metrics: `/metrics`, `/healthz`, `/readyz`, and pprof.

See the working example: `example/dev/main.go:1`,
service implementations: `example/dev/services/payment/service.go:1`, `example/dev/services/report/service.go:1`.

## Quick start

1) Create an `App` and its configuration:
- `config.WithRunMode("aio"|"service"|"worker"|"consumer")` — run mode (defaults to `aio`).
- `config.WithServices([]string{"paymentservice-service"})` — services list for `service` mode.
- `tonica.WithSpec("path/to/openapi.json")` — enable `/docs` (optional).

2) Register services in the registry:
- `service.WithName(paymentv1.ServiceName)` — unique name (generated from `.proto`).
- `service.WithGRPC(payment.RegisterGRPC)` — function that registers gRPC endpoints on `*grpc.Server`.
- `service.WithGRPCAddr(":9000")` — service's gRPC server address.
- `service.WithGateway(payment.RegisterGateway)` — register the REST gateway (if HTTP access is needed).

3) Run the application: `app.Run()`.

A minimal reproducible example is in `example/dev`.

## Run modes

- `aio` — all services + HTTP gateway + metrics in a single process.
- `service` — only services specified by `config.WithServices` or the `APP_SERVICES` env var (comma-separated) are started.
- `worker`, `consumer` — stubs for Temporal/streaming (work in progress).

The `APP_MODE` environment variable can override the mode, for example: `APP_MODE=service`.

## HTTP and REST gateway

- The application HTTP server listens on `APP_HTTP_ADDR` (defaults to `:8080`).
- CORS is open by default; allowed origins can be set via `PS_CORS_ORIGINS` (comma-separated).
- The REST gateway proxies to gRPC on `/v1/*` and forwards headers `authorization`, `traceparent`, `tracestate`, `x-request-id`.
- Documentation is available at `/docs` when `tonica.WithSpec(".../openapi.swagger.json")` is provided.

## gRPC

- Each service starts its own gRPC server at its address (see `service.WithGRPCAddr`).
- The `.proto` wrapper generator creates useful constants, for example `ServiceName` and `ServiceAddrEnvName` for the service address.
- Example generated files: `example/dev/proto/payment/v1/paymentservice_grpc.go:1`.

## Observability and metrics

- Tracing: set `OTEL_EXPORTER_OTLP_ENDPOINT` (gRPC). Log level is controlled by `LOG_LEVEL` (`debug|info|warn|error`).
- Logging: `slog` (text locally when `PS_APP_ENV=local`, otherwise JSON).
- Metrics and profiling: separate HTTP server on `APP_METRIC_ADDR` (defaults to `:2121`) with `/metrics`, `/healthz`, `/readyz`, and pprof.
- Out of the box histograms and counters for HTTP, gRPC, Redis/SQL, and Pub/Sub, for example:
  - `app_http_response`, `app_http_service_response`
  - `app_sql_stats`, `app_redis_stats`
  - `app_pubsub_*`

## CLI: code generation from .proto

- Command: `go run ./pkg/tonica/cmd/wrap --proto <path/to/service.proto>`.
- Generates helper files next to the `.proto` for client/server and address configuration.
- Already generated in the example: `example/dev/proto/payment/v1/paymentservice_grpc.go:1`, `example/dev/proto/reports/v1/reportsservice_grpc.go:1`.

## Environment variables (core)

- `APP_MODE` — `aio|service|worker|consumer`.
- `APP_HTTP_ADDR` — application HTTP server address, defaults to `:8080`.
- `APP_METRIC_ADDR` — metrics server address, defaults to `:2121`.
- `APP_SERVICES` — services list for `service` mode, e.g. `paymentservice-service,reportsservice-service`.
- `OTEL_EXPORTER_OTLP_ENDPOINT` — OTLP endpoint for tracing.
- `LOG_LEVEL` — `debug|info|warn|error`.
- `PS_APP_ENV` — logging format (`local` — text output).
- `PS_CORS_ORIGINS` — allowed CORS origins (comma-separated).

## Technologies

- `gin`, `grpc`, `grpc-gateway`
- OpenTelemetry (`otel`), Prometheus
- Kafka/PubSub, Temporal (pluggable)

## Limitations

The framework focuses on practical service-architecture use cases and fast environment bootstrapping. General-purpose universality was not a goal; some areas (workers/consumers) are marked as WIP.

---

You can start with the example: `go run example/dev/main.go` and open `http://localhost:8080/docs` (when `example/dev/openapi/openapi.swagger.json` is present).

Local development tooling

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
