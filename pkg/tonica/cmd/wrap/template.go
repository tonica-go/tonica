package wrap

const (
	wrapperTemplate = `// Code generated. DO NOT EDIT.
// versions:
// 	source: {{ .Source }}

{{- $hasUnary := false }}
{{- range .Methods }}
    {{- if and (not .StreamsRequest) (not .StreamsResponse) }}
        {{- $hasUnary = true }}
    {{- end }}
{{- end }}

package {{ .Package }}

import (
	"context"
	//"time"
	
	"github.com/tonica-go/tonica/pkg/tonica"

	//tonicagRPC "github.com/tonica-go/tonica/pkg/tonica/grpc"
	"google.golang.org/grpc"

	{{- if $hasUnary }}
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	{{- end }}
	
	//healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// New{{ .Service }}GoFrServer creates a new instance of {{ .Service }}GoFrServer
func New{{ .Service }}GoFrServer() *{{ .Service }}GoFrServer {
	return &{{ .Service }}GoFrServer{
		//health: getOrCreateHealthServer(), // Initialize the health server
	}
}

// {{ .Service }}ServerWithGofr is the interface for the server implementation
type {{ .Service }}ServerWithGofr interface {
{{- range .Methods }}
{{- if or .StreamsRequest .StreamsResponse }}
	{{ .Name }}(*tonica.Context, {{ $.Service }}_{{ .Name }}Server) error
{{- else }}
	{{ .Name }}(*tonica.Context) (any, error)
{{- end }}
{{- end }}
}

// {{ .Service }}ServerWrapper wraps the server and handles request and response logic
type {{ .Service }}ServerWrapper struct {
	{{ .Service }}Server
	//*healthServer
	//Container *container.Container
	server    {{ .Service }}ServerWithGofr
}

{{- $hasStream := false }}
{{- range .Methods }}
    {{- if or .StreamsRequest .StreamsResponse }}
        {{- $hasStream = true }}
    {{- end }}
{{- end }}

{{- if $hasStream }}
// Base instrumented stream
type instrumentedStream struct {
	grpc.ServerStream
	ctx    *tonica.Context
	method string
}

func (s *instrumentedStream) Context() context.Context {
	return s.ctx
}

func (s *instrumentedStream) SendMsg(m interface{}) error {
	start := time.Now()
	span := s.ctx.Trace(s.method + "/SendMsg")
	defer span.End()

	err := s.ServerStream.SendMsg(m)

	logger := tonicagRPC.NewgRPCLogger()
	logger.DocumentRPCLog(s.ctx, s.ctx.Logger, s.ctx.Metrics(), start, err,
		s.method+"/SendMsg", "app_gRPC-Stream_stats")

	return err
}

func (s *instrumentedStream) RecvMsg(m interface{}) error {
	start := time.Now()
	span := s.ctx.Trace(s.method + "/RecvMsg")
	defer span.End()

	err := s.ServerStream.RecvMsg(m)

	logger := tonicagRPC.NewgRPCLogger()
	logger.DocumentRPCLog(s.ctx, s.ctx.Logger, s.ctx.Metrics(), start, err,
		s.method+"/RecvMsg", "app_gRPC-Stream_stats")

	return err
}
{{- end }}

{{ range .Methods }}
{{- if .StreamsRequest }}
// Client-side streaming specific wrapper
type clientStreamWrapper{{ .Name }} struct {
	*instrumentedStream
}

func (w *clientStreamWrapper{{ .Name }}) SendAndClose(m *{{ .Response }}) error {
	start := time.Now()
	span := w.ctx.Trace(w.method + "/SendAndClose")
	defer span.End()
	
	err := w.ServerStream.SendMsg(m)
	
	logger := tonicagRPC.NewgRPCLogger()
	logger.DocumentRPCLog(w.ctx, w.ctx.Logger, w.ctx.Metrics(), start, err,
	w.method+"/SendAndClose", "app_gRPC-Stream_stats")
	
	return err
}

func (w *clientStreamWrapper{{ .Name }}) Recv() (*{{ .Request }}, error) {
	start := time.Now()
	span := w.ctx.Trace(w.method + "/Recv")
	defer span.End()
	
	var req {{ .Request }}
	err := w.ServerStream.RecvMsg(&req)
	
	logger := tonicagRPC.NewgRPCLogger()
	logger.DocumentRPCLog(w.ctx, w.ctx.Logger, w.ctx.Metrics(), start, err,
	w.method+"/Recv", "app_gRPC-Stream_stats")
	
	return &req, err
}
{{- end }}

{{- if .StreamsResponse }}
{{- if not .StreamsRequest }}
// Server-side streaming specific wrapper
type serverStreamWrapper{{ .Name }} struct {
	*instrumentedStream
}

func (w *serverStreamWrapper{{ .Name }}) Send(m *{{ .Response }}) error {
	start := time.Now()
	span := w.ctx.Trace(w.method + "/Send")
	defer span.End()
	
	err := w.ServerStream.SendMsg(m)
	
	logger := tonicagRPC.NewgRPCLogger()
	logger.DocumentRPCLog(w.ctx, w.ctx.Logger, w.ctx.Metrics(), start, err,
	w.method+"/Send", "app_gRPC-Stream_stats")
	
	return err
}
{{- else }}
// Bidirectional streaming wrapper
type bidiStreamWrapper{{ .Name }} struct {
	*instrumentedStream
}

func (w *bidiStreamWrapper{{ .Name }}) Send(m *{{ .Response }}) error {
	start := time.Now()
	span := w.ctx.Trace(w.method + "/Send")
	defer span.End()
	
	err := w.ServerStream.SendMsg(m)
	
	logger := tonicagRPC.NewgRPCLogger()
	logger.DocumentRPCLog(w.ctx, w.ctx.Logger, w.ctx.Metrics(), start, err,
	w.method+"/Send", "app_gRPC-Stream_stats")
	
	return err
}

func (w *bidiStreamWrapper{{ .Name }}) Recv() (*{{ .Request }}, error) {
	start := time.Now()
	span := w.ctx.Trace(w.method + "/Recv")
	defer span.End()
	
	var req {{ .Request }}
	err := w.ServerStream.RecvMsg(&req)
	
	logger := tonicagRPC.NewgRPCLogger()
	logger.DocumentRPCLog(w.ctx, w.ctx.Logger, w.ctx.Metrics(), start, err,
	w.method+"/Recv", "app_gRPC-Stream_stats")
	
	return &req, err
}

func (w *bidiStreamWrapper{{ .Name }}) CloseSend() error {
	start := time.Now()
	span := w.ctx.Trace(w.method + "/CloseSend")
	defer span.End()

	err := w.ServerStream.(grpc.ClientStream).CloseSend()

	logger := tonicagRPC.NewgRPCLogger()
	logger.DocumentRPCLog(w.ctx, w.ctx.Logger, w.ctx.Metrics(), start, err,
		w.method+"/CloseSend", "app_gRPC-Stream_stats")

	return err
}
{{- end }}
{{- end }}
{{- end }}

{{ range .Methods }}
{{- if .StreamsResponse }}
{{- if not .StreamsRequest }}
// Server-side streaming handler for {{ .Name }}
func (h *{{ $.Service }}ServerWrapper) {{ .Name }}(req *{{ .Request }}, stream {{ $.Service }}_{{ .Name }}Server) error {
	ctx := stream.Context()
	gctx := h.getGofrContext(ctx, &{{ .Request }}Wrapper{ctx: ctx, {{ .Request }}: req})
	
	is := &instrumentedStream{
		ServerStream: stream,
		ctx:        gctx,
		method:     "/{{ $.Service }}/{{ .Name }}",
	}
	
	wrappedStream := &serverStreamWrapper{{ .Name }}{instrumentedStream: is}
	return h.server.{{ .Name }}(gctx, wrappedStream)
}
{{- else }}
// Bidirectional streaming handler for {{ .Name }}
func (h *{{ $.Service }}ServerWrapper) {{ .Name }}(stream {{ $.Service }}_{{ .Name }}Server) error {
	ctx := stream.Context()
	gctx := h.getGofrContext(ctx, nil)
	
	is := &instrumentedStream{
		ServerStream: stream,
		ctx:        gctx,
		method:     "/{{ $.Service }}/{{ .Name }}",
	}
	
	wrappedStream := &bidiStreamWrapper{{ .Name }}{instrumentedStream: is}
	return h.server.{{ .Name }}(gctx, wrappedStream)
}
{{- end }}
{{- else if .StreamsRequest }}
// Client-side streaming handler for {{ .Name }}
func (h *{{ $.Service }}ServerWrapper) {{ .Name }}(stream {{ $.Service }}_{{ .Name }}Server) error {
	ctx := stream.Context()
	gctx := h.getGofrContext(ctx, nil)
	
	is := &instrumentedStream{
		ServerStream: stream,
		ctx:        gctx,
		method:     "/{{ $.Service }}/{{ .Name }}",
	}
	
	wrappedStream := &clientStreamWrapper{{ .Name }}{instrumentedStream: is}
	return h.server.{{ .Name }}(gctx, wrappedStream)
}
{{- else }}
// Unary method handler for {{ .Name }}
func (h *{{ $.Service }}ServerWrapper) {{ .Name }}(ctx context.Context, req *{{ .Request }}) (*{{ .Response }}, error) {
	gctx := h.getGofrContext(ctx, &{{ .Request }}Wrapper{ctx: ctx, {{ .Request }}: req})
	
	res, err := h.server.{{ .Name }}(gctx)
	if err != nil {
		return nil, err
	}

	resp, ok := res.(*{{ .Response }})
	if !ok {
		return nil, status.Errorf(codes.Unknown, "unexpected response type %T", res)
	}
	
	return resp, nil
}
{{- end }}
{{- end }}

// mustEmbedUnimplemented{{ .Service }}Server ensures implementation
func (h *{{ .Service }}ServerWrapper) mustEmbedUnimplemented{{ .Service }}Server() {}

// Register{{ .Service }}ServerWithGofr registers the server
func Register{{ .Service }}ServerWithGofr(app *tonica.App, srv {{ .Service }}ServerWithGofr) {
	registerServerWithGofr(app, srv, func(s grpc.ServiceRegistrar, srv any) {
		wrapper := &{{ .Service }}ServerWrapper{
			server: srv.({{ .Service }}ServerWithGofr),
			//healthServer: getOrCreateHealthServer(),
		}

		Register{{ .Service }}Server(s, wrapper)

		//wrapper.Server.SetServingStatus("Hello", healthpb.HealthCheckResponse_SERVING)
	})
}

// getGofrContext creates GoFr context
func (h *{{ .Service }}ServerWrapper) getGofrContext(ctx context.Context, req tonica.Request) *tonica.Context {
	return &tonica.Context{
		Context:   ctx,
		//Container: h.Container,
		Request:   req,
	}
}
`

	messageTemplate = `// Code generated by tonica.dev/cli/tonica. DO NOT EDIT.
// versions:
// 	tonica-cli v0.7.0
// 	tonica.dev v1.39.0
// 	source: {{ .Source }}


package {{ .Package }}

import (
	"context"
	"fmt"
	"reflect"
)

// Request Wrappers
{{- range $request := .Requests }}
type {{ $request }}Wrapper struct {
	ctx context.Context
	*{{ $request }}
}

func (h *{{ $request }}Wrapper) Context() context.Context {
	return h.ctx
}

func (h *{{ $request }}Wrapper) Param(s string) string {
	return ""
}

func (h *{{ $request }}Wrapper) PathParam(s string) string {
	return ""
}

func (h *{{ $request }}Wrapper) Bind(p interface{}) error {
	ptr := reflect.ValueOf(p)
	if ptr.Kind() != reflect.Ptr {
		return fmt.Errorf("expected a pointer, got %T", p)
	}

	hValue := reflect.ValueOf(h.{{ $request }}).Elem()
	ptrValue := ptr.Elem()

	for i := 0; i < hValue.NumField(); i++ {
		field := hValue.Type().Field(i)
		if field.Name == "state" || field.Name == "sizeCache" || field.Name == "unknownFields" {
			continue
		}

		if field.IsExported() {
			ptrValue.Field(i).Set(hValue.Field(i))
		}
	}

	return nil
}

func (h *{{ $request }}Wrapper) HostName() string {
	return ""
}

func (h *{{ $request }}Wrapper) Params(s string) []string {
	return nil
}
{{- end }}`

	serverTemplate = `// versions:
// 	tonica-cli v0.6.0
// 	tonica.dev v1.37.0
// 	source: {{ .Source }}

package {{ .Package }}

import "github.com/tonica-go/tonica/pkg/tonica"

// Register the gRPC service in your app using the following code in your main.go:
//
// {{ .Package }}.Register{{ $.Service }}ServerWithGofr(app, &{{ .Package }}.New{{ $.Service }}GoFrServer())
//
// {{ $.Service }}GoFrServer defines the gRPC server implementation.
// Customize the struct with required dependencies and fields as needed.

type {{ $.Service }}GoFrServer struct {
 health *healthServer
}

{{- range .Methods }}
{{- if .StreamsRequest }}
func (s *{{ $.Service }}GoFrServer) {{ .Name }}(ctx *tonica.Context, stream {{ $.Service }}_{{ .Name }}Server) error {
	// Implementation here
	return nil
}
{{- else if .StreamsResponse }}
func (s *{{ $.Service }}GoFrServer) {{ .Name }}(ctx *tonica.Context, stream {{ $.Service }}_{{ .Name }}Server) error {
	// Implementation here
	return nil
}
{{- else }}
func (s *{{ $.Service }}GoFrServer) {{ .Name }}(ctx *tonica.Context) (any, error) {
	return &{{ .Response }}{}, nil
}
{{- end }}
{{- end }}
`
	clientTemplate = `// Code generated by tonica.dev/cli/tonica. DO NOT EDIT.
// versions:
// 	tonica-cli v0.7.0
// 	tonica.dev v1.39.0
// 	source: {{ .Source }}

package {{ .Package }}

import (
	"github.com/tonica-go/tonica/pkg/tonica"
	//"github.com/tonica-go/tonica/pkg/tonica/metrics"
	"google.golang.org/grpc"
)

type {{ .Service }}GoFrClient interface {
{{- range .Methods }}
{{- if and .StreamsResponse (not .StreamsRequest) }}
	{{ .Name }}(ctx *tonica.Context, req *{{ .Request }}, opts ...grpc.CallOption) (grpc.ServerStreamingClient[{{ .Response }}], error)
{{- else if and .StreamsRequest (not .StreamsResponse) }}
	{{ .Name }}(ctx *tonica.Context, opts ...grpc.CallOption) (grpc.ClientStreamingClient[{{ .Request }}, {{ .Response }}], error)
{{- else if and .StreamsRequest .StreamsResponse }}
	{{ .Name }}(ctx *tonica.Context, opts ...grpc.CallOption) (grpc.BidiStreamingClient[{{ .Request }}, {{ .Response }}], error)
{{- else }}
	{{ .Name }}(ctx *tonica.Context, req *{{ .Request }}, opts ...grpc.CallOption) (*{{ .Response }}, error)
{{- end }}
{{- end }}
	//HealthClient
}

type {{ .Service }}ClientWrapper struct {
	client {{ .Service }}Client
	//HealthClient
}

func New{{ .Service }}GoFrClient(host string, dialOptions ...grpc.DialOption) ({{ .Service }}GoFrClient, error) {
	conn, err := createGRPCConn(host, "{{ .Service }}", dialOptions...)
	if err != nil {
		return &{{ .Service }}ClientWrapper{
			client:       nil,
			//HealthClient: &HealthClientWrapper{client: nil},
		}, err
	}

	//metricsOnce.Do(func() {
	//	metrics.NewHistogram("app_gRPC-Client_stats", "Response time of gRPC client in milliseconds.", gRPCBuckets...)
	//})

	res := New{{ .Service }}Client(conn)
	//healthClient := NewHealthClient(conn)

	return &{{ .Service }}ClientWrapper{
		client: res,
		//HealthClient: healthClient,
	}, nil
}

{{ range .Methods }}
{{- if and .StreamsResponse (not .StreamsRequest) }}
func (h *{{ $.Service }}ClientWrapper) {{ .Name }}(ctx *tonica.Context, req *{{ .Request }}, 
	opts ...grpc.CallOption) (grpc.ServerStreamingClient[{{ .Response }}], error) {
	result, err := invokeRPC(ctx, "/{{ $.Service }}/{{ .Name }}", func() (interface{}, error) {
		return h.client.{{ .Name }}(ctx.Context, req, opts...)
	}, "app_gRPC-Stream_stats")

	if err != nil {
		return nil, err
	}
	return result.(grpc.ServerStreamingClient[{{ .Response }}]), nil
}
{{- else if and .StreamsRequest (not .StreamsResponse) }}
func (h *{{ $.Service }}ClientWrapper) {{ .Name }}(ctx *tonica.Context, 
	opts ...grpc.CallOption) (grpc.ClientStreamingClient[{{ .Request }}, {{ .Response }}], error) {
	result, err := invokeRPC(ctx, "/{{ $.Service }}/{{ .Name }}", func() (interface{}, error) {
		return h.client.{{ .Name }}(ctx.Context, opts...)
	}, "app_gRPC-Stream_stats")

	if err != nil {
		return nil, err
	}
	return result.(grpc.ClientStreamingClient[{{ .Request }}, {{ .Response }}]), nil
}
{{- else if and .StreamsRequest .StreamsResponse }}
func (h *{{ $.Service }}ClientWrapper) {{ .Name }}(ctx *tonica.Context, 
	opts ...grpc.CallOption) (grpc.BidiStreamingClient[{{ .Request }}, {{ .Response }}], error) {
	result, err := invokeRPC(ctx, "/{{ $.Service }}/{{ .Name }}", func() (interface{}, error) {
		return h.client.{{ .Name }}(ctx.Context, opts...)
	}, "app_gRPC-Stream_stats")

	if err != nil {
		return nil, err
	}
	return result.(grpc.BidiStreamingClient[{{ .Request }}, {{ .Response }}]), nil
}
{{- else }}
func (h *{{ $.Service }}ClientWrapper) {{ .Name }}(ctx *tonica.Context, req *{{ .Request }}, 
	opts ...grpc.CallOption) (*{{ .Response }}, error) {
	result, err := invokeRPC(ctx, "/{{ $.Service }}/{{ .Name }}", func() (interface{}, error) {
		return h.client.{{ .Name }}(ctx.Context, req, opts...)
	}, "app_gRPC-Client_stats")

	if err != nil {
		return nil, err
	}
	return result.(*{{ .Response }}), nil
}
{{- end }}
{{- end }}
`

	healthServerTemplate = `// Code generated by tonica.dev/cli/tonica. DO NOT EDIT.
// versions:
// 	tonica-cli v0.7.0
// 	tonica.dev v1.39.0
// 	source: {{ .Source }}

package {{ .Package }}

import (
	//"fmt"
	"google.golang.org/grpc"
	//"time"

	"github.com/tonica-go/tonica/pkg/tonica"

	//tonicaGRPC "github.com/tonica-go/tonica/pkg/tonica/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type healthServer struct {
	*health.Server
}

var globalHealthServer *healthServer
var healthServerRegistered bool // Global flag to track if health server is registered

// getOrCreateHealthServer ensures only one health server is created and reused.
func getOrCreateHealthServer() *healthServer {
	if globalHealthServer == nil {
		globalHealthServer = &healthServer{health.NewServer()}
	}
	return globalHealthServer
}

func registerServerWithGofr(app *tonica.App, srv any, registerFunc func(grpc.ServiceRegistrar, any)) {
	var s grpc.ServiceRegistrar = app
	h := getOrCreateHealthServer()

	// Register metrics and health server only once
	if !healthServerRegistered {
		//gRPCBuckets := []float64{0.005, 0.01, .05, .075, .1, .125, .15, .2, .3, .5, .75, 1, 2, 3, 4, 5, 7.5, 10}
		//app.Metrics().NewHistogram("app_gRPC-Server_stats", "Response time of gRPC server in milliseconds.", gRPCBuckets...)
		//app.Metrics().NewHistogram("app_gRPC-Stream_stats", "Duration of gRPC stream in milliseconds.", gRPCBuckets...)

		healthpb.RegisterHealthServer(s, h.Server)
		h.Server.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
		healthServerRegistered = true
	}

	// Register the provided server
	registerFunc(s, srv)
}

func (h *healthServer) Check(ctx *tonica.Context, req *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	//start := time.Now()
	span := ctx.Trace("/grpc.health.v1.Health/Check")
	res, err := h.Server.Check(ctx.Context, req)
	//logger := tonicaGRPC.NewgRPCLogger()
	//logger.DocumentRPCLog(ctx.Context, ctx.Logger, ctx.Metrics(), start, err,
	//fmt.Sprintf("/grpc.health.v1.Health/Check	Service: %q", req.Service), "app_gRPC-Server_stats")
	span.End()
	return res, err
}

func (h *healthServer) Watch(ctx *tonica.Context, in *healthpb.HealthCheckRequest, stream healthpb.Health_WatchServer) error {
	//start := time.Now()
	span := ctx.Trace("/grpc.health.v1.Health/Watch")
	err := h.Server.Watch(in, stream)
	//logger := tonicaGRPC.NewgRPCLogger()
	//logger.DocumentRPCLog(ctx.Context, ctx.Logger, ctx.Metrics(), start, err,
	//fmt.Sprintf("/grpc.health.v1.Health/Watch	Service: %q", in.Service), "app_gRPC-Server_stats")
	span.End()
	return err
}

func (h *healthServer) SetServingStatus(ctx *tonica.Context, service string, servingStatus healthpb.HealthCheckResponse_ServingStatus) {
	//start := time.Now()
	span := ctx.Trace("/grpc.health.v1.Health/SetServingStatus")
	h.Server.SetServingStatus(service, servingStatus)
	//logger := tonicaGRPC.NewgRPCLogger()
	//logger.DocumentRPCLog(ctx.Context, ctx.Logger, ctx.Metrics(), start, nil,
	//fmt.Sprintf("/grpc.health.v1.Health/SetServingStatus	Service: %q", service), "app_gRPC-Server_stats")
	span.End()
}

func (h *healthServer) Shutdown(ctx *tonica.Context) {
	//start := time.Now()
	span := ctx.Trace("/grpc.health.v1.Health/Shutdown")
	h.Server.Shutdown()
	//logger := tonicaGRPC.NewgRPCLogger()
	//logger.DocumentRPCLog(ctx.Context, ctx.Logger, ctx.Metrics(), start, nil,
	//"/grpc.health.v1.Health/Shutdown", "app_gRPC-Server_stats")
	span.End()
}

func (h *healthServer) Resume(ctx *tonica.Context) {
	//start := time.Now()
	span := ctx.Trace("/grpc.health.v1.Health/Resume")
	h.Server.Resume()
	//logger := tonicaGRPC.NewgRPCLogger()
	//logger.DocumentRPCLog(ctx.Context, ctx.Logger, ctx.Metrics(), start, nil,
	//"/grpc.health.v1.Health/Resume", "app_gRPC-Server_stats")
	span.End()
}
`

	clientHealthTemplate = `// Code generated by tonica.dev/cli/tonica. DO NOT EDIT.
// versions:
// 	tonica-cli v0.6.0
// 	tonica.dev v1.37.0
// 	source: {{ .Source }}

package {{ .Package }}

import (
	"fmt"
	"sync"
	//"time"

	"github.com/tonica-go/tonica/pkg/tonica"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"

	//tonicagRPC "github.com/tonica-go/tonica/pkg/tonica/grpc"
)

var (
	metricsOnce sync.Once
	gRPCBuckets = []float64{0.005, 0.01, .05, .075, .1, .125, .15, .2, .3, .5, .75, 1, 2, 3, 4, 5, 7.5, 10}
)

type HealthClient interface {
	Check(ctx *tonica.Context, in *grpc_health_v1.HealthCheckRequest, opts ...grpc.CallOption) (*grpc_health_v1.HealthCheckResponse, error)
	Watch(ctx *tonica.Context, in *grpc_health_v1.HealthCheckRequest, opts ...grpc.CallOption) (
	grpc.ServerStreamingClient[grpc_health_v1.HealthCheckResponse], error)
}

type HealthClientWrapper struct {
	client grpc_health_v1.HealthClient
}

func NewHealthClient(conn *grpc.ClientConn) HealthClient {
	return &HealthClientWrapper{
		client: grpc_health_v1.NewHealthClient(conn),
	}
}

func createGRPCConn(host string, serviceName string, dialOptions ...grpc.DialOption) (*grpc.ClientConn, error) {
	serviceConfig := ` + "`{\"loadBalancingPolicy\": \"round_robin\"}`" + `

	defaultOpts := []grpc.DialOption{
		grpc.WithDefaultServiceConfig(serviceConfig),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	// Developer Note: If the user provides custom DialOptions, they will override the default options due to 
	// the ordering of dialOptions. This behavior is intentional to ensure the gRPC client connection is properly 
	// configured even when the user does not specify any DialOptions.
	dialOptions = append(defaultOpts, dialOptions...)

	conn, err := grpc.NewClient(host, dialOptions...)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func invokeRPC(ctx *tonica.Context, rpcName string, rpcFunc func() (interface{}, error), metricName string) (interface{}, error) {
	span := ctx.Trace("gRPC-srv-call: " + rpcName)
	defer span.End()

	traceID := span.SpanContext().TraceID().String()
	spanID := span.SpanContext().SpanID().String()
	md := metadata.Pairs("x-tonica-traceid", traceID, "x-tonica-spanid", spanID)

	ctx.Context = metadata.NewOutgoingContext(ctx.Context, md)
	//transactionStartTime := time.Now()

	res, err := rpcFunc()
	//logger := tonicagRPC.NewgRPCLogger()
	//logger.DocumentRPCLog(ctx.Context, ctx.Logger, ctx.Metrics(), transactionStartTime, err, rpcName, metricName)

	return res, err
}

func (h *HealthClientWrapper) Check(ctx *tonica.Context, in *grpc_health_v1.HealthCheckRequest, 
	opts ...grpc.CallOption) (*grpc_health_v1.HealthCheckResponse, error) {
	result, err := invokeRPC(ctx, fmt.Sprintf("/grpc.health.v1.Health/Check	Service: %q", in.Service), func() (interface{}, error) {
		return h.client.Check(ctx, in, opts...)
	}, "app_gRPC-Client_stats")

	if err != nil {
		return nil, err
	}
	return result.(*grpc_health_v1.HealthCheckResponse), nil
}

func (h *HealthClientWrapper) Watch(ctx *tonica.Context, in *grpc_health_v1.HealthCheckRequest, 
	opts ...grpc.CallOption) (grpc.ServerStreamingClient[grpc_health_v1.HealthCheckResponse], error) {
	result, err := invokeRPC(ctx, fmt.Sprintf("/grpc.health.v1.Health/Watch	Service: %q", in.Service), func() (interface{}, error) {
		return h.client.Watch(ctx, in, opts...)
	}, "app_gRPC-Stream_stats")

	if err != nil {
		return nil, err
	}

	return result.(grpc.ServerStreamingClient[grpc_health_v1.HealthCheckResponse]), nil
}
`

	clientGRPCTemplate = `// Code generated by tonica.dev/cli/tonica. DO NOT EDIT.
package {{ .Package }}

import (
	"net"

	"google.golang.org/grpc"

	"github.com/tonica-go/tonica/pkg/tonica/grpc/serviceconfig"
)

const ServiceName = "{{ $.ServiceLower }}-service"
const ServiceAddrEnvName = "{{ $.ServiceUpper }}_SERVICE_GRPC_ADDR"

type {{ $.Service }}AddressConfig struct {
	{{ $.Service }}Address  string ` + "`" + `env:"{{ $.ServiceUpper }}_SERVICE_GRPC_ADDR,required"` + "`" + `
	{{ $.Service }}GrpcPort string ` + "`" + `env:"{{ $.ServiceUpper }}_SERVICE_GRPC_PORT,required"` + "`" + `
}

func (cfg *{{ $.Service }}AddressConfig) Create{{ $.Service }}Connection() *grpc.ClientConn {
	return serviceconfig.MustCreateNewNonBlockingServiceConnection(cfg.Get{{ $.Service }}Config())
}

func (cfg *{{ $.Service }}AddressConfig) Get{{ $.Service }}Config() serviceconfig.ServiceConfig {
	return serviceconfig.ServiceConfig{
		Address: net.JoinHostPort(cfg.{{ $.Service }}Address, cfg.{{ $.Service }}GrpcPort),
		Name:    ServiceName,
	}
}

func (cfg *{{ $.Service }}AddressConfig) Create{{ $.Service }}Client() {{ $.Service }}Client {
	return New{{ $.Service }}Client(cfg.Create{{ $.Service }}Connection())
}

`
)
