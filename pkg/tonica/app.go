package tonica

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/MarceloPetrucio/go-scalar-api-reference"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/tonica-go/tonica/pkg/tonica/config"
	"github.com/tonica-go/tonica/pkg/tonica/logger"
	"github.com/tonica-go/tonica/pkg/tonica/metrics"
	"github.com/tonica-go/tonica/pkg/tonica/metrics/exporters"
	obs "github.com/tonica-go/tonica/pkg/tonica/observabillity"
	"github.com/tonica-go/tonica/pkg/tonica/registry"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type App struct {
	Name     string
	registry registry.Registry
	logger   *log.Logger
	cfg      *config.Config

	spec string

	router       *gin.Engine
	metricRouter *gin.Engine

	metricsManager metrics.Manager
}

func NewApp(options ...AppOption) *App {
	l := logger.New()
	app := &App{
		Name:           config.DefaultAppName,
		registry:       registry.NewRegistry(),
		logger:         l,
		metricsManager: metrics.NewMetricsManager(exporters.Prometheus("aoo", "0.0.0")),
		router:         gin.New(),
		metricRouter:   gin.New(),
	}

	for _, option := range options {
		option(app)
	}

	app.registerFrameworkMetrics()

	return app
}

func (a *App) GetLogger() *log.Logger {
	return a.logger
}

func (a *App) GetMetricManager() metrics.Manager {
	return a.metricsManager
}

func (a *App) GetRegistry() registry.Registry {
	return a.registry
}

func (a *App) GetMetricRouter() *gin.Engine {
	return a.metricRouter
}

func (a *App) GetRouter() *gin.Engine {
	return a.router
}

// initObs initializes OpenTelemetry + Prometheus for a given service name.
func initObs(ctx context.Context, service string) (*obs.Observability, error) {
	return obs.Init(ctx, obs.Config{
		ServiceName:    service,
		ServiceVersion: "v0.1.0",
		OTLPEndpoint:   os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		LogLevel:       config.GetEnv("LOG_LEVEL", "info"),
	})
}

func (a *App) runAio() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Observability
	o, err := initObs(ctx, a.cfg.AppName())
	if err != nil {
		a.logger.Fatal(err)
	}
	defer func() { _ = o.Shutdown(context.Background()) }()

	workers, err := a.GetRegistry().GetAllWorkers()
	if err != nil {
		a.GetLogger().Fatal(err)
	}
	consumers, err := a.GetRegistry().GetAllConsumers()
	if err != nil {
		a.GetLogger().Fatal(err)
	}
	services, err := a.GetRegistry().GetAllServices()
	if err != nil {
		a.GetLogger().Fatal(err)
	}

	gwmux := runtime.NewServeMux(
		runtime.WithIncomingHeaderMatcher(func(key string) (string, bool) {
			// Forward selected headers to gRPC metadata
			switch strings.ToLower(key) {
			case "authorization", "traceparent", "tracestate", "x-request-id":
				return key, true
			default:
				return runtime.DefaultHeaderMatcher(key)
			}
		}),
	)
	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		obs.GRPCClientStats(),
	}

	errCh := make(chan error, len(services))
	for _, service := range services {
		var grpcSrv *grpc.Server
		var grpcLis net.Listener
		srvGrpc := service.GetGRPC()

		grpcLis, err = net.Listen("tcp", service.GetGRPCAddr())
		if err != nil {
			a.GetLogger().Fatal(err)
		}
		grpcSrv = grpc.NewServer(
			obs.GRPCServerStats(),
			grpc.ChainUnaryInterceptor(
				obs.GRPCRecoverUnary(),
				obs.GRPCLoggingUnary(),
			),
			grpc.ChainStreamInterceptor(
				obs.GRPCRecoverStream(),
				obs.GRPCLoggingStream(),
			),
		)

		srvGrpc(grpcSrv, service)
		if service.GetIsGatewayEnabled() {
			registerGw := service.GetGateway()
			if err := registerGw(ctx, gwmux, service.GetGRPCAddr(), dialOpts); err != nil {
				a.GetLogger().Fatal(err)
			}
		}
		go func() {
			a.GetLogger().Println("gRPC listening", "addr", service.GetGRPCAddr())
			if err := grpcSrv.Serve(grpcLis); err != nil {
				errCh <- err
			}
		}()

	}

	go func() {
		router := a.metricRouter

		router.Use(gin.Recovery())
		router.Use(obs.HTTPLogger())
		metrics.GetHandler(a.GetMetricManager(), router)

		router.GET("/healthz", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status": "ok",
				"now":    time.Now().UTC().Format(time.RFC3339),
			})
		})
		router.GET("/readyz", func(c *gin.Context) {
			//results, overall := ReadinessChecks(c.Request.Context(), opts.Infra)
			code := http.StatusOK
			//if !overall {
			//	code = http.StatusServiceUnavailable
			//}
			c.JSON(code, gin.H{
				"status": func() string {
					//if overall {
					//	return "ok"
					//}
					return "degraded"
				}(),
				"now":    time.Now().UTC().Format(time.RFC3339),
				"checks": "results",
			})
		})

		addr := config.GetEnv("APP_METRIC_ADDR", ":2121")
		a.GetLogger().Println("metrics server running, listening addr", addr)
		router.Run(addr)
	}()

	go func() {
		router := a.router

		router.Use(gin.Recovery())
		router.Use(otelgin.Middleware(a.Name + "-http"))
		router.Use(obs.RequestID())
		router.Use(obs.HTTPLogger())
		router.Use(cors.New(buildCORSConfig()))

		router.GET("/ping", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "pong",
			})
		})
		if a.spec != "" {
			docs := func(w http.ResponseWriter, r *http.Request) {
				htmlContent, err := scalar.ApiReferenceHTML(&scalar.Options{
					SpecURL:  a.spec,
					DarkMode: true,
				})

				if err != nil {
					fmt.Printf("%v", err)
				}

				_, err = fmt.Fprintln(w, htmlContent)
				if err != nil {
					a.GetLogger().Fatal(err)
				}
			}

			router.GET("/docs", gin.WrapF(docs))
		}

		router.Any("/v1/*any", gin.WrapH(gwmux))

		addr := config.GetEnv("APP_HTTP_ADDR", ":8080")
		a.GetLogger().Println("http server running, listening addr", addr)
		router.Run(addr)
	}()

	for _, consumer := range consumers {
		go func() {
			err := consumer.Start(ctx)
			if err != nil {
				a.GetLogger().Fatal(err)
			}
		}()
	}

	for _, w := range workers {
		go func() {
			err := w.Start()
			if err != nil {
				a.GetLogger().Fatal(err)
			}
		}()
	}

	select {
	case <-ctx.Done():
		a.GetLogger().Println("context canceled")
		os.Exit(1)
	case err := <-errCh:
		a.GetLogger().Fatal(err)
	}
}

func (a *App) runService() {
	ctx := context.Background()
	services, err := a.GetRegistry().GetAllServices()
	if err != nil {
		a.GetLogger().Fatal(err)
	}
	cnt := 0
	for _, service := range services {
		if !slices.Contains(a.cfg.Services(), service.GetName()) {
			continue
		}
		cnt++
	}

	errCh := make(chan error, cnt)
	for _, service := range services {
		if !slices.Contains(a.cfg.Services(), service.GetName()) {
			continue
		}
		var grpcSrv *grpc.Server
		var grpcLis net.Listener
		srvGrpc := service.GetGRPC()

		grpcLis, err = net.Listen("tcp", service.GetGRPCAddr())
		if err != nil {
			a.GetLogger().Fatal(err)
		}
		grpcSrv = grpc.NewServer(
			grpc.ChainUnaryInterceptor(),
			grpc.ChainStreamInterceptor(),
		)

		srvGrpc(grpcSrv, service)
		go func() {
			a.GetLogger().Println("gRPC listening", "addr", service.GetGRPCAddr())
			if err := grpcSrv.Serve(grpcLis); err != nil {
				errCh <- err
			}
		}()

	}

	select {
	case <-ctx.Done():
		a.GetLogger().Println("context canceled")
		os.Exit(1)
	case err := <-errCh:
		a.GetLogger().Fatal(err)
	}
}

func (a *App) registerFrameworkMetrics() {
	// system info metrics
	a.GetMetricManager().NewGauge("app_info", "Info for app_name, app_version and framework_version.")
	a.GetMetricManager().NewGauge("app_go_routines", "Number of Go routines running.")
	a.GetMetricManager().NewGauge("app_sys_memory_alloc", "Number of bytes allocated for heap objects.")
	a.GetMetricManager().NewGauge("app_sys_total_alloc", "Number of cumulative bytes allocated for heap objects.")
	a.GetMetricManager().NewGauge("app_go_numGC", "Number of completed Garbage Collector cycles.")
	a.GetMetricManager().NewGauge("app_go_sys", "Number of total bytes of memory.")

	{ // HTTP metrics
		httpBuckets := []float64{.001, .003, .005, .01, .02, .03, .05, .1, .2, .3, .5, .75, 1, 2, 3, 5, 10, 30}
		a.GetMetricManager().NewHistogram("app_http_response", "Response time of HTTP requests in seconds.", httpBuckets...)
		a.GetMetricManager().NewHistogram("app_http_service_response", "Response time of HTTP service requests in seconds.", httpBuckets...)
	}

	{ // Redis metrics
		redisBuckets := []float64{.05, .075, .1, .125, .15, .2, .3, .5, .75, 1, 1.25, 1.5, 2, 2.5, 3}
		a.GetMetricManager().NewHistogram("app_redis_stats", "Response time of Redis commands in milliseconds.", redisBuckets...)
	}

	{ // SQL metrics
		sqlBuckets := []float64{.05, .075, .1, .125, .15, .2, .3, .5, .75, 1, 2, 3, 4, 5, 7.5, 10}
		a.GetMetricManager().NewHistogram("app_sql_stats", "Response time of SQL queries in milliseconds.", sqlBuckets...)
		a.GetMetricManager().NewGauge("app_sql_open_connections", "Number of open SQL connections.")
		a.GetMetricManager().NewGauge("app_sql_inUse_connections", "Number of inUse SQL connections.")
	}

	// pubsub metrics
	a.GetMetricManager().NewCounter("app_pubsub_publish_total_count", "Number of total publish operations.")
	a.GetMetricManager().NewCounter("app_pubsub_publish_success_count", "Number of successful publish operations.")
	a.GetMetricManager().NewCounter("app_pubsub_subscribe_total_count", "Number of total subscribe operations.")
	a.GetMetricManager().NewCounter("app_pubsub_subscribe_success_count", "Number of successful subscribe operations.")
}

func buildCORSConfig() cors.Config {
	cfg := cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type", "X-Request-ID"},
		ExposeHeaders:    []string{"Content-Length", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	if v := strings.TrimSpace(os.Getenv("PS_CORS_ORIGINS")); v != "" {
		cfg.AllowAllOrigins = false
		cfg.AllowOrigins = splitAndTrim(v)
	}
	return cfg
}

func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
