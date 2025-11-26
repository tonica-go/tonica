package obs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"os"
	"strconv"
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

	// Get histogram buckets from env or use defaults
	// In milliseconds: 10ms, 50ms, 100ms, 250ms, 500ms, 1s, 2.5s, 5s, 10s, 30s, 60s
	buckets := getHistogramBuckets()

	// Create a view to apply custom buckets to all histograms
	customView := sdkmetric.NewView(
		sdkmetric.Instrument{Kind: sdkmetric.InstrumentKindHistogram},
		sdkmetric.Stream{
			Aggregation: sdkmetric.AggregationExplicitBucketHistogram{
				Boundaries: buckets,
			},
		},
	)

	// Wire the exporter into the OTel metrics SDK with custom views
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(prom),
		sdkmetric.WithResource(res),
		sdkmetric.WithView(customView),
	)
	otel.SetMeterProvider(mp)
	handler := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})

	return &Observability{
		MetricsHandler: handler,
		Shutdown:       func(ctx context.Context) error { return tp.Shutdown(ctx) },
	}, nil
}

// getHistogramBuckets returns histogram buckets from env or defaults
func getHistogramBuckets() []float64 {
	// Default buckets in milliseconds: 10ms, 50ms, 100ms, 250ms, 500ms, 1s, 2.5s, 5s, 10s, 30s, 60s
	defaultBuckets := []float64{10, 50, 100, 250, 500, 1000, 2500, 5000, 10000, 30000, 60000}

	bucketsStr := strings.TrimSpace(os.Getenv("OTEL_HISTOGRAM_BUCKETS_MS"))
	if bucketsStr == "" {
		return defaultBuckets
	}

	// Parse comma-separated buckets from env
	parts := strings.Split(bucketsStr, ",")
	buckets := make([]float64, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		val, err := strconv.ParseFloat(p, 64)
		if err != nil {
			slog.Warn("failed to parse histogram bucket value, using defaults", "value", p, "error", err)
			return defaultBuckets
		}
		buckets = append(buckets, val)
	}

	if len(buckets) == 0 {
		return defaultBuckets
	}

	return buckets
}

// HTTPTracing creates a middleware that starts a span with the actual URL path.
// Unlike otelgin.Middleware which uses c.FullPath() (route pattern like "/api/v1/*any"),
// this uses c.Request.URL.Path to get the actual path like "/api/v1/workflows/search".
func HTTPTracing(serviceName string) gin.HandlerFunc {
	tracer := otel.Tracer("github.com/gin-gonic/gin/otelgin")
	return func(c *gin.Context) {
		// Skip observability endpoints and OPTIONS requests
		if isObsPath(c.Request.URL.Path) || c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}

		ctx := c.Request.Context()
		// Extract propagated context from HTTP headers
		ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(c.Request.Header))

		// Use actual URL path instead of route pattern
		spanName := c.Request.Method + " " + c.Request.URL.Path
		opts := []trace.SpanStartOption{
			trace.WithAttributes(
				semconv.HTTPRequestMethodKey.String(c.Request.Method),
				semconv.URLPath(c.Request.URL.Path),
				semconv.URLFull(c.Request.URL.String()),
				semconv.URLScheme(c.Request.URL.Scheme),
				attribute.String("http.route", c.FullPath()),
				attribute.String("server.address", serviceName),
			),
			trace.WithSpanKind(trace.SpanKindServer),
		}

		ctx, span := tracer.Start(ctx, spanName, opts...)
		defer span.End()

		// Add trace_id to context for downstream usage similar to identity
		sc := trace.SpanContextFromContext(ctx)
		if sc.IsValid() {
			ctx = context.WithValue(ctx, "trace_id", sc.TraceID().String())
			c.Set("trace_id", sc.TraceID().String())
		}

		// Update request context with span context (and trace_id)
		c.Request = c.Request.WithContext(ctx)
		c.Next()

		// Record response status
		status := c.Writer.Status()
		span.SetAttributes(semconv.HTTPResponseStatusCode(status))
		if status >= 400 {
			span.SetAttributes(attribute.Bool("error", true))
		}
	}
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
		// Skip noisy endpoints and OPTIONS requests from logging/metrics
		if isObsPath(c.Request.URL.Path) || c.Request.Method == http.MethodOptions {
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
	// Get histogram buckets from env or use defaults
	buckets := getHistogramBuckets()
	httpLatencyMs, _ = meter.Float64Histogram(
		"http_request_duration_ms",
		metric.WithUnit("ms"),
		metric.WithDescription("HTTP request duration in milliseconds"),
		metric.WithExplicitBucketBoundaries(buckets...),
	)
}

func recordHTTPMetrics(c *gin.Context, d time.Duration) {
	// Skip observability endpoints and OPTIONS requests
	if isObsPath(c.Request.URL.Path) || c.Request.Method == http.MethodOptions {
		return
	}
	httpMetricsOnce.Do(initHTTPInstruments)
	// Use actual URL path for metrics instead of route pattern
	attrs := []attribute.KeyValue{
		attribute.String("method", c.Request.Method),
		attribute.String("route", c.Request.URL.Path),
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
