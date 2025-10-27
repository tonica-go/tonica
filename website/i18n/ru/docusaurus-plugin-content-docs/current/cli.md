# Tonica CLI

Tonica включает инструмент командной строки, который помогает быстро создавать проекты и генерировать шаблонный код из Protocol Buffers.

## Установка

Установите инструмент Tonica CLI:

```bash
go install github.com/tonica-go/tonica@latest
```

Убедитесь, что `$GOPATH/bin` находится в вашем `PATH`.

## Предварительные требования

Перед использованием CLI установите необходимые инструменты:

```bash
# Install buf (Protocol Buffer tool)
brew install buf

# Install protoc plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
```

## Команды

### `tonica init`

Инициализирует новый проект Tonica с шаблонной структурой.

**Использование:**
```bash
tonica init --name=PROJECT_NAME
```

**Опции:**
- `--name` (обязательно) - Название вашего проекта

**Пример:**
```bash
mkdir myservice && cd myservice
tonica init --name=myservice
```

**Что создается:**
```
myservice/
├── cmd/
│   └── server/
│       └── main.go           # Точка входа приложения
├── internal/
│   └── service/              # Реализации сервисов
├── proto/                    # Определения Protocol Buffer
│   └── myservice/
│       └── v1/
│           └── myservice.proto
├── buf.gen.yaml             # Конфигурация Buf
├── buf.yaml                 # Рабочее пространство Buf
├── go.mod
└── README.md
```

### `tonica proto`

Генерирует код Protocol Buffer и шаблонный код для нового сервиса.

**Использование:**
```bash
tonica proto --name=SERVICE_NAME
```

**Опции:**
- `--name` (обязательно) - Название сервиса

**Пример:**
```bash
tonica proto --name=userservice
```

**Что генерируется:**
- Файл `.proto` с базовым определением сервиса
- Интерфейс gRPC сервиса
- Код регистрации Gateway
- Спецификация OpenAPI
- Вспомогательные константы (ServiceName, ServiceAddrEnvName)

**Сгенерированные файлы:**
```
proto/
└── userservice/
    └── v1/
        ├── userservice.proto              # Определение Proto
        ├── userservice.pb.go              # Сгенерированный protobuf
        ├── userservice_grpc.pb.go         # Сгенерированный gRPC сервер/клиент
        ├── userservice.pb.gw.go           # Сгенерированный gateway
        ├── userservice_grpc.go            # Обертка Tonica с вспомогательными функциями
        └── userservice.swagger.json       # Спецификация OpenAPI
```

### `wrap` (Продвинутый)

Генерирует код-обертку Tonica из существующих `.proto` файлов. Это полезно, когда у вас есть существующие proto определения.

**Использование:**
```bash
go run ./pkg/tonica/cmd/wrap --proto path/to/service.proto
```

**Опции:**
- `--proto` (обязательно) - Путь к файлу `.proto`

**Пример:**
```bash
go run ./pkg/tonica/cmd/wrap --proto proto/payment/v1/payment.proto
```

**Что генерируется:**
Добавляет файл `*_grpc.go` с вспомогательными функциями и константами:
- `ServiceName` - Константа имени сервиса
- `ServiceAddrEnvName` - Название переменной окружения для адреса сервиса
- Вспомогательные функции регистрации

## Процесс быстрого старта

Вот полный процесс создания нового сервиса:

```bash
# 1. Create project directory
mkdir myservice && cd myservice

# 2. Initialize Tonica project
tonica init --name=myservice

# 3. Generate your first service
tonica proto --name=myservice

# 4. Initialize Go module
go mod init github.com/yourusername/myservice
go mod tidy

# 5. Run the service
go run ./cmd/server

# 6. Test it
curl http://localhost:8080/docs
```

## Пример Proto файла

Когда вы запускаете `tonica proto --name=userservice`, генерируется proto файл такого вида:

```protobuf
syntax = "proto3";

package userservice.v1;

import "google/api/annotations.proto";

option go_package = "github.com/yourorg/myservice/proto/userservice/v1;userservicev1";

service UserService {
  rpc GetUser(GetUserRequest) returns (GetUserResponse) {
    option (google.api.http) = {
      get: "/v1/users/{id}"
    };
  }

  rpc CreateUser(CreateUserRequest) returns (CreateUserResponse) {
    option (google.api.http) = {
      post: "/v1/users"
      body: "*"
    };
  }
}

message GetUserRequest {
  string id = 1;
}

message GetUserResponse {
  string id = 1;
  string name = 2;
  string email = 3;
}

message CreateUserRequest {
  string name = 1;
  string email = 2;
}

message CreateUserResponse {
  string id = 1;
  string name = 2;
  string email = 3;
}
```

## Сгенерированные вспомогательные константы

Файл-обертка `*_grpc.go` включает полезные константы:

```go
package userservicev1

const (
    // ServiceName is the name of the service for registration
    ServiceName = "userservice-service"

    // ServiceAddrEnvName is the environment variable name for the service address
    ServiceAddrEnvName = "USERSERVICE_SERVICE_ADDR"
)

// RegisterGRPC registers the gRPC service
func RegisterGRPC(server *grpc.Server, impl UserServiceServer) {
    RegisterUserServiceServer(server, impl)
}

// RegisterGateway registers the HTTP gateway
func RegisterGateway(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
    return RegisterUserServiceHandler(ctx, mux, conn)
}
```

## Работа с существующими Proto

Если у вас есть существующие `.proto` файлы:

1. **Вариант 1: Используйте `buf generate`**
   ```bash
   buf generate
   ```

2. **Вариант 2: Сгенерируйте обертки Tonica**
   ```bash
   go run ./pkg/tonica/cmd/wrap --proto proto/myservice/v1/myservice.proto
   ```

3. **Вариант 3: Ручная настройка**
   - Запустите protoc с необходимыми плагинами
   - Создайте функции-обертки вручную

## Конфигурация Buf

Проекты Tonica используют `buf` для управления proto. Сгенерированный `buf.gen.yaml`:

```yaml
version: v1
managed:
  enabled: true
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

## Регенерация кода

После изменения `.proto` файлов регенерируйте код:

```bash
# Using buf (recommended)
buf generate

# Or using protoc directly
protoc --go_out=. --go-grpc_out=. --grpc-gateway_out=. \
  --openapiv2_out=openapi \
  proto/myservice/v1/*.proto
```

## Устранение неполадок

### Команда не найдена: tonica

Убедитесь, что `$GOPATH/bin` находится в вашем PATH:
```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

### Плагины Protoc не найдены

Установите все необходимые плагины:
```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
```

### Ошибки импорта в сгенерированном коде

Запустите `go mod tidy` для загрузки зависимостей:
```bash
go mod tidy
```

### Ошибки Buf

Убедитесь, что `buf.yaml` и `buf.gen.yaml` правильно настроены. Проверьте [документацию Buf](https://buf.build/docs) для подробностей.

## Следующие шаги

- [Начало работы](./getting-started.md) - Создайте свой первый сервис
- [Архитектура](./architecture.md) - Поймите фреймворк
- [Конфигурация](./configuration.md) - Настройте ваши сервисы
- [Лучшие практики](./best-practices.md) - Паттерны для продакшена
