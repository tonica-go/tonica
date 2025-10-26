# Tonica CLI

Tonica includes a command-line tool to help you quickly scaffold projects and generate boilerplate code from Protocol Buffers.

## Installation

Install the Tonica CLI tool:

```bash
go install github.com/tonica-go/tonica@latest
```

Make sure `$GOPATH/bin` is in your `PATH`.

## Prerequisites

Before using the CLI, install the required tools:

```bash
# Install buf (Protocol Buffer tool)
brew install buf

# Install protoc plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
```

## Commands

### `tonica init`

Initialize a new Tonica project with boilerplate structure.

**Usage:**
```bash
tonica init --name=PROJECT_NAME
```

**Options:**
- `--name` (required) - Name of your project

**Example:**
```bash
mkdir myservice && cd myservice
tonica init --name=myservice
```

**What it creates:**
```
myservice/
├── cmd/
│   └── server/
│       └── main.go           # Application entrypoint
├── internal/
│   └── service/              # Service implementations
├── proto/                    # Protocol buffer definitions
│   └── myservice/
│       └── v1/
│           └── myservice.proto
├── buf.gen.yaml             # Buf configuration
├── buf.yaml                 # Buf workspace
├── go.mod
└── README.md
```

### `tonica proto`

Generate Protocol Buffer code and boilerplate for a new service.

**Usage:**
```bash
tonica proto --name=SERVICE_NAME
```

**Options:**
- `--name` (required) - Name of the service

**Example:**
```bash
tonica proto --name=userservice
```

**What it generates:**
- `.proto` file with basic service definition
- gRPC service interface
- Gateway registration code
- OpenAPI specification
- Helper constants (ServiceName, ServiceAddrEnvName)

**Generated files:**
```
proto/
└── userservice/
    └── v1/
        ├── userservice.proto              # Proto definition
        ├── userservice.pb.go              # Generated protobuf
        ├── userservice_grpc.pb.go         # Generated gRPC server/client
        ├── userservice.pb.gw.go           # Generated gateway
        ├── userservice_grpc.go            # Tonica wrapper with helpers
        └── userservice.swagger.json       # OpenAPI spec
```

### `wrap` (Advanced)

Generate Tonica wrapper code from existing `.proto` files. This is useful when you have existing proto definitions.

**Usage:**
```bash
go run ./pkg/tonica/cmd/wrap --proto path/to/service.proto
```

**Options:**
- `--proto` (required) - Path to the `.proto` file

**Example:**
```bash
go run ./pkg/tonica/cmd/wrap --proto proto/payment/v1/payment.proto
```

**What it generates:**
Adds a `*_grpc.go` file with helper functions and constants:
- `ServiceName` - Service name constant
- `ServiceAddrEnvName` - Environment variable name for service address
- Registration helpers

## Quick Start Workflow

Here's a complete workflow for creating a new service:

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

## Proto File Example

When you run `tonica proto --name=userservice`, it generates a proto file like this:

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

## Generated Helper Constants

The `*_grpc.go` wrapper file includes useful constants:

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

## Working with Existing Protos

If you have existing `.proto` files:

1. **Option 1: Use `buf generate`**
   ```bash
   buf generate
   ```

2. **Option 2: Generate Tonica wrappers**
   ```bash
   go run ./pkg/tonica/cmd/wrap --proto proto/myservice/v1/myservice.proto
   ```

3. **Option 3: Manual setup**
   - Run protoc with the required plugins
   - Create wrapper functions manually

## Buf Configuration

Tonica projects use `buf` for proto management. The generated `buf.gen.yaml`:

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

## Regenerating Code

After modifying `.proto` files, regenerate code:

```bash
# Using buf (recommended)
buf generate

# Or using protoc directly
protoc --go_out=. --go-grpc_out=. --grpc-gateway_out=. \
  --openapiv2_out=openapi \
  proto/myservice/v1/*.proto
```

## Troubleshooting

### Command not found: tonica

Make sure `$GOPATH/bin` is in your PATH:
```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

### Protoc plugins not found

Install all required plugins:
```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
```

### Import errors in generated code

Run `go mod tidy` to download dependencies:
```bash
go mod tidy
```

### Buf errors

Ensure `buf.yaml` and `buf.gen.yaml` are properly configured. Check the [Buf documentation](https://buf.build/docs) for details.

## Next Steps

- [Getting Started](./getting-started.md) - Build your first service
- [Architecture](./architecture.md) - Understand the framework
- [Configuration](./configuration.md) - Configure your services
- [Best Practices](./best-practices.md) - Production patterns
