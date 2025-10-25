package proto_init

const protoTpl = `syntax = "proto3";

package {{ .Name }}.v1;

option go_package = "proto/gen/{{ .Name }}/v1;{{ .Name }}v1";

import "google/api/annotations.proto";
import "protoc-gen-openapiv2/options/annotations.proto";

option (grpc.gateway.protoc_gen_openapiv2.options.openapiv2_swagger) = {
  info: {
    title: "Example System API";
    version: "v1";
    description: "Demo example";
  };
  schemes: HTTP;
  consumes: "application/json";
  produces: "application/json";
  security_definitions: {
    security: {
      key: "bearer";
      value: {
        type: TYPE_API_KEY;
        name: "Authorization";
        in: IN_HEADER;
        description: "Use 'Bearer <token>'";
      }
    }
  };
};

service {{ .NameFirstUpper }}Service {
  // Public example endpoint
  rpc Hello(HelloRequest) returns (HelloResponse) {
    option (google.api.http) = {
      post: "/v1/hello"
      body: "*"
    };
  }
}

message HelloRequest {
  string name = 1;
}

message HelloResponse {
  string data = 1;
}

`

const serviceTpl = `package {{ .Name }};

import (
	"context"

	"github.com/tonica-go/tonica/pkg/tonica/service"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

type {{ .NameFirstUpper }}ServiceServer struct {
	{{ .Name }}v1.Unimplemented{{ .NameFirstUpper }}ServiceServer
	srv *service.Service
}

func RegisterGRPC(s *grpc.Server, srv *service.Service) {
	{{ .Name }}v1.Register{{ .NameFirstUpper }}ServiceServer(s, &{{ .NameFirstUpper }}ServiceServer{srv: srv})
}

func RegisterGateway(ctx context.Context, mux *runtime.ServeMux, target string, dialOpts []grpc.DialOption) error {
	return {{ .Name }}v1.Register{{ .NameFirstUpper }}ServiceHandlerFromEndpoint(ctx, mux, target, dialOpts)
}

func GetClient(s *grpc.ClientConn) {{ .Name }}v1.{{ .NameFirstUpper }}ServiceClient {
	return {{ .Name }}v1.New{{ .NameFirstUpper }}ServiceClient(s)
}

// implement all necessary methods, better in separated file, cause this one probably could be regenerated

`
