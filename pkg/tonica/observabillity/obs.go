package obs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"os"
	"time"

	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	promclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	otelgrpcpkg "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	promexp "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Config struct {
	ServiceName    string
	ServiceVersion string
	OTLPEndpoint   string // e.g., "localhost:4317" (gRPC)
	LogLevel       string // debug, info, warn, error
}

type Observability struct {
	MetricsHandler http.Handler
	Shutdown       func(ctx context.Context) error
}

// Init sets up OpenTelemetry (traces + Prometheus metrics) and global propagation.
func Init(ctx context.Context, cfg Config) (*Observability, error) {
	// Configure slog default logger first, so everything logs with desired level
	setupLogger(cfg.LogLevel)
	if cfg.ServiceName == "" {
		cfg.ServiceName = "service"
	}
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			attribute.String("service.version", cfg.ServiceVersion),
		),
	)
	if err != nil {
		return nil, err
	}

	// Traces: use OTLP gRPC exporter if endpoint set; otherwise, a noop provider.
	var tp *sdktrace.TracerProvider
	if cfg.OTLPEndpoint != "" {
		exp, err := otlptracegrpc.New(ctx, otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint), otlptracegrpc.WithInsecure())
		if err != nil {
			return nil, err
		}
		tp = sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exp),
			sdktrace.WithResource(res),
		)
	} else {
		tp = sdktrace.NewTracerProvider(sdktrace.WithResource(res))
	}
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// Metrics: Prometheus exporter mounted at /metrics
	reg := promclient.NewRegistry()
	// Add Go and process collectors for baseline metrics
	reg.MustRegister(collectors.NewGoCollector())
	reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	prom, err := promexp.New(promexp.WithRegisterer(reg))
	if err != nil {
		return nil, err
	}
	// Wire the exporter into the OTel metrics SDK
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(prom),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(mp)
	handler := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})

	return &Observability{
		MetricsHandler: handler,
		Shutdown:       func(ctx context.Context) error { return tp.Shutdown(ctx) },
	}, nil
}

// RequestID adds/propagates X-Request-ID.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader("X-Request-ID")
		if rid == "" {
			rid = randomID()
			c.Writer.Header().Set("X-Request-ID", rid)
		}
		c.Set("request_id", rid)
		c.Next()
	}
}

// HTTPLogger logs request/response with slog and trace id.
func HTTPLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip noisy endpoints from logging/metrics
		if isObsPath(c.Request.URL.Path) {
			c.Next()
			return
		}
		start := time.Now()
		c.Next()
		latency := time.Since(start)
		rid, _ := c.Get("request_id")
		sc := trace.SpanContextFromContext(c.Request.Context())
		// Record OTel metrics for HTTP
		recordHTTPMetrics(c, latency)
		// Gather error info from Gin context
		var errMsg string
		if len(c.Errors) > 0 {
			errMsg = c.Errors.String()
		}
		// Choose level by logStatus and errors
		logStatus := c.Writer.Status()
		logFn := slog.Info
		if logStatus >= 500 || errMsg != "" {
			logFn = slog.Error
		} else if logStatus >= 400 {
			logFn = slog.Warn
		}
		logFn(
			"trace_id", sc.TraceID().String(),
			"span_id", sc.SpanID().String(),
			"http",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"logStatus", logStatus,
			"duration_ms", latency.Milliseconds(),
			"request_id", rid,
			"ip", c.ClientIP(),
			"user_agent", c.Request.UserAgent(),
			"error", errMsg,
		)
	}
}

var (
	httpMetricsOnce sync.Once
	httpReqCounter  metric.Int64Counter
	httpLatencyMs   metric.Float64Histogram
)

func initHTTPInstruments() {
	meter := otel.Meter("tonica/http")
	httpReqCounter, _ = meter.Int64Counter(
		"http_requests_total",
		metric.WithDescription("Total number of HTTP requests"),
	)
	httpLatencyMs, _ = meter.Float64Histogram(
		"http_request_duration_ms",
		metric.WithUnit("ms"),
		metric.WithDescription("HTTP request duration in milliseconds"),
	)
}

func recordHTTPMetrics(c *gin.Context, d time.Duration) {
	if isObsPath(c.Request.URL.Path) {
		return
	}
	httpMetricsOnce.Do(initHTTPInstruments)
	attrs := []attribute.KeyValue{
		attribute.String("method", c.Request.Method),
		attribute.String("route", c.FullPath()),
		attribute.Int("status", c.Writer.Status()),
	}
	ctx := c.Request.Context()
	if httpReqCounter != nil {
		httpReqCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
	if httpLatencyMs != nil {
		httpLatencyMs.Record(ctx, float64(d.Milliseconds()), metric.WithAttributes(attrs...))
	}
}

func isObsPath(p string) bool {
	if p == "/healthz" || p == "/readyz" || p == "/metrics" || p == "/docs" {
		return true
	}
	if strings.HasPrefix(p, "/debug/pprof") {
		return true
	}
	return false
}

// GRPCServerStats gRPC OTel stats handlers (preferred over deprecated interceptors)
func GRPCServerStats() grpc.ServerOption { return grpc.StatsHandler(otelgrpcpkg.NewServerHandler()) }
func GRPCClientStats() grpc.DialOption   { return grpc.WithStatsHandler(otelgrpcpkg.NewClientHandler()) }

func GRPCRecoverUnary() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("grpc panic", "method", info.FullMethod, "panic", r)
				err = status.Error(codes.Internal, "internal")
			}
		}()
		return handler(ctx, req)
	}
}
func GRPCRecoverStream() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("grpc panic", "method", info.FullMethod, "panic", r)
				err = status.Error(codes.Internal, "internal")
			}
		}()
		return handler(srv, ss)
	}
}

func GRPCLoggingUnary() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		grpcLog(ctx, info.FullMethod, status.Code(err), start, err)
		return resp, err
	}
}
func GRPCLoggingStream() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		err := handler(srv, ss)
		grpcLog(context.Background(), info.FullMethod, status.Code(err), start, err)
		return err
	}
}

func grpcLog(ctx context.Context, method string, code codes.Code, start time.Time, err error) {
	// Level by status code
	sc := trace.SpanContextFromContext(ctx)
	logFn := slog.Info
	if code == codes.Unknown || code == codes.DeadlineExceeded || code == codes.Unimplemented || code == codes.Internal || code == codes.Unavailable || code == codes.DataLoss {
		logFn = slog.Error
	} else if code != codes.OK {
		logFn = slog.Warn
	}
	logFn(

		"trace_id", sc.TraceID().String(),
		"span_id", sc.SpanID().String(),
		"grpc",
		"method", method,
		"code", code.String(),
		"duration_ms", time.Since(start).Milliseconds(),
		"error", errString(err),
	)
}

func randomID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// setupLogger configures the default slog logger with JSON handler and level
func setupLogger(levelStr string) {
	lvl := new(slog.LevelVar)
	// default info
	lvl.Set(slog.LevelInfo)
	switch strings.ToLower(strings.TrimSpace(levelStr)) {
	case "debug":
		lvl.Set(slog.LevelDebug)
	case "info", "":
		lvl.Set(slog.LevelInfo)
	case "warn", "warning":
		lvl.Set(slog.LevelWarn)
	case "error":
		lvl.Set(slog.LevelError)
	}
	// Choose handler format by PS_APP_ENV: local -> text, otherwise JSON
	// todo use config
	appEnv := strings.ToLower(strings.TrimSpace(os.Getenv("PS_APP_ENV")))
	var h slog.Handler
	if appEnv == "local" {
		// Plain text for local runs
		h = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	} else {
		// JSON for docker/other envs
		h = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	}
	slog.SetDefault(slog.New(h))
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
