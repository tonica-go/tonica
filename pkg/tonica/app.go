package tonica

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/MarceloPetrucio/go-scalar-api-reference"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/tonica-go/tonica/pkg/tonica/config"
	"github.com/tonica-go/tonica/pkg/tonica/logger"
	"github.com/tonica-go/tonica/pkg/tonica/metrics"
	"github.com/tonica-go/tonica/pkg/tonica/metrics/exporters"
	"github.com/tonica-go/tonica/pkg/tonica/modules/entities"
	"github.com/tonica-go/tonica/pkg/tonica/modules/workflows"
	obs "github.com/tonica-go/tonica/pkg/tonica/observabillity"
	"github.com/tonica-go/tonica/pkg/tonica/registry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type App struct {
	Name     string
	registry registry.Registry
	logger   *log.Logger
	cfg      *config.Config

	spec         string
	specUrl      string
	customRoutes []RouteMetadata
	apiPrefix    string

	isWorkflowService bool
	workflowNamespace string

	isEntityService   bool
	entityDefinitions string
	entityDriver      string
	entityDSN         string

	router       *gin.Engine
	metricRouter *gin.Engine

	metricsManager metrics.Manager
	shutdown       *Shutdown
}

func NewApp(options ...AppOption) *App {
	l := logger.New()
	app := &App{
		Name:           config.DefaultAppName,
		registry:       registry.NewRegistry(),
		logger:         l,
		metricsManager: metrics.NewMetricsManager(exporters.Prometheus(config.DefaultAppName, "0.0.0")),
		router:         gin.New(),
		metricRouter:   gin.New(),
		shutdown:       NewShutdown(),
		apiPrefix:      "/v1", // default prefix for backward compatibility
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
	slog.Info(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
	return obs.Init(ctx, obs.Config{
		ServiceName:    service,
		ServiceVersion: "v0.1.0",
		OTLPEndpoint:   os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		LogLevel:       config.GetEnv("LOG_LEVEL", "info"),
	})
}

func (a *App) registerGateway(ctx context.Context) *runtime.ServeMux {
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
		runtime.WithMetadata(func(ctx context.Context, r *http.Request) metadata.MD {
			md := metadata.MD{}

			// Извлекаем identity из контекста и добавляем в metadata
			if identity, ok := ctx.Value("identity").(map[string]interface{}); ok && identity != nil {
				ib, err := json.Marshal(identity)
				if err != nil {
					slog.Error("Failed to marshal identity", "error", err)
					return metadata.MD{}
				}
				md.Set("x-identity", string(ib))
			}

			return md
		}),
	)
	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		obs.GRPCClientStats(),
		//grpc.WithUnaryInterceptor(ClientContextInterceptor()),
	}

	services, err := a.GetRegistry().GetAllServices()
	if err != nil {
		a.GetLogger().Fatal(err)
	}
	for _, service := range services {
		if service.GetIsGatewayEnabled() {
			registerGw := service.GetGateway()
			if err := registerGw(ctx, gwmux, service.GetGRPCAddr(), dialOpts); err != nil {
				a.GetLogger().Fatal(err)
			}
		}
	}

	return gwmux
}

func (a *App) registerMetrics(ctx context.Context, o *obs.Observability) {
	router := a.metricRouter

	router.Use(gin.Recovery())
	router.Use(obs.HTTPLogger())

	// Use OpenTelemetry metrics handler instead of old metrics.Manager
	if o != nil && o.MetricsHandler != nil {
		router.GET("/metrics", gin.WrapH(o.MetricsHandler))
	} else {
		// Fallback to old handler if obs not available
		metrics.GetHandler(a.GetMetricManager(), router)
	}

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
		//todo need to add ready checks for services, consumers, workers, gateway
		c.JSON(code, gin.H{
			"status": func() string {
				//if overall {
				//	return "ok"
				//}
				return "ok"
			}(),
			"now":    time.Now().UTC().Format(time.RFC3339),
			"checks": "results",
		})
	})

	addr := config.GetEnv("APP_METRIC_ADDR", ":2121")
	a.GetLogger().Println("metrics server running, listening addr", addr)
	router.Run(addr)
}

func (a *App) registerAPI(ctx context.Context) {
	router := a.router
	router.Use(gin.Recovery())
	router.Use(obs.HTTPTracing(a.Name + "-http"))
	router.Use(obs.RequestID())
	router.Use(obs.HTTPLogger())
	router.Use(cors.New(buildCORSConfig()))

	if a.spec != "" {
		specContent := func() ([]byte, error) {
			specBytes, err := os.ReadFile(a.spec)
			if err != nil {
				a.GetLogger().Printf("failed to read spec file %s: %v", a.spec, err)
				//c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read spec file"})
				return nil, err
			}

			// Merge custom routes into the spec
			mergedSpec, err := mergeCustomRoutesIntoSpec(specBytes, a.customRoutes)
			if err != nil {
				a.GetLogger().Printf("failed to merge custom routes: %v", err)
				//c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to merge custom routes"})
				return nil, err
			}

			return mergedSpec, nil
		}

		// Serve merged OpenAPI spec with custom routes at /openapi.json
		router.GET("/openapi.json", func(c *gin.Context) {
			specBytes, err := specContent()
			if err != nil {
				a.GetLogger().Printf("failed to read spec file %s: %v", a.spec, err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read spec file"})
				return
			}

			c.Data(http.StatusOK, "application/json", specBytes)
		})
	}
	if a.specUrl != "" {

		docs := func(w http.ResponseWriter, r *http.Request) {
			htmlContent, err := scalar.ApiReferenceHTML(&scalar.Options{
				SpecURL:  a.specUrl,
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

	router.Any(a.apiPrefix+"/*any", WrapH(a.registerGateway(ctx)))

	addr := config.GetEnv("APP_HTTP_ADDR", ":8080")
	a.GetLogger().Println("http server running, listening addr", addr)
	router.Run(addr)
}

func WrapH(h http.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, exist := c.Get("identity")
		if exist {

			r := c.Request.WithContext(context.WithValue(c.Request.Context(), "identity", id))
			h.ServeHTTP(c.Writer, r)
			return
		}
		h.ServeHTTP(c.Writer, c.Request)
	}
}

func (a *App) registerConsumers(ctx context.Context) {
	consumers, err := a.GetRegistry().GetAllConsumers()
	if err != nil {
		a.GetLogger().Fatal(err)
	}
	for _, consumer := range consumers {
		go func() {
			err := consumer.Start(ctx)
			if err != nil {
				a.GetLogger().Fatal(err)
			}
		}()
	}
}

func (a *App) registerWorkers(_ context.Context) {
	workers, err := a.GetRegistry().GetAllWorkers()
	if err != nil {
		a.GetLogger().Fatal(err)
	}
	for _, w := range workers {
		go func() {
			err := w.Start()
			if err != nil {
				a.GetLogger().Fatal(err)
			}
		}()
	}
}

// UnaryInterceptor returns a gRPC unary interceptor that authenticates incoming requests.
func UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Allowlist unauthenticated methods (e.g., login, session refresh, health checks)
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return handler(ctx, req)
		}

		// Извлекаем identity из metadata
		if identityJSON := md.Get("x-identity"); len(identityJSON) > 0 {
			var identity map[string]interface{}
			if err := json.Unmarshal([]byte(identityJSON[0]), &identity); err != nil {
				slog.Error("Failed to unmarshal identity", "error", err)
			} else {
				// Добавляем identity в контекст
				ctx = context.WithValue(ctx, "identity", identity)
			}
		}

		return handler(ctx, req)
	}
}

func (a *App) registerServices(_ context.Context, errCh chan error) {
	if a.isEntityService {
		// Register Entities service
		entitiesService := entities.NewTonicaService(a.entityDriver, a.entityDSN)
		a.GetRegistry().MustRegisterService(entitiesService)
		slog.Info("registered entities service")
	}

	if a.isWorkflowService {
		// Temporal client configuration
		temporalClient, err := GetTemporalClient("")
		if err != nil {
			a.GetLogger().Fatal(err)
		}
		// Register Workflows service
		workflowsService := workflows.NewTonicaService(temporalClient)
		a.GetRegistry().MustRegisterService(workflowsService)
		slog.Info("Registered workflows")
	}

	services, err := a.GetRegistry().GetAllServices()
	if err != nil {
		a.GetLogger().Fatal(err)
	}
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
				UnaryInterceptor(),
				obs.GRPCRecoverUnary(),
				obs.GRPCLoggingUnary(),
			),
			grpc.ChainStreamInterceptor(
				obs.GRPCRecoverStream(),
				obs.GRPCLoggingStream(),
			),
		)

		// Register gRPC server for graceful shutdown
		a.shutdown.RegisterGRPCServer(grpcSrv)

		srvGrpc(grpcSrv, service)

		go func(srv *grpc.Server, addr string) {
			a.GetLogger().Println("gRPC listening", "addr", addr)
			if err := srv.Serve(grpcLis); err != nil {
				errCh <- err
			}
		}(grpcSrv, service.GetGRPCAddr())
	}
}

func (a *App) run(ctx context.Context, errCh chan error) {
	select {
	case <-ctx.Done():
		a.GetLogger().Println("shutdown signal received, starting graceful shutdown...")

		// Graceful shutdown with 30 second timeout
		if err := a.shutdown.Execute(30 * time.Second); err != nil {
			a.GetLogger().Printf("graceful shutdown error: %v", err)
		}

		a.GetLogger().Println("shutdown complete")
		return
	case err := <-errCh:
		a.GetLogger().Fatal(err)
	}
}

const gatewayCount = 1

func (a *App) runAio(ctx context.Context, o *obs.Observability) {
	go a.registerMetrics(ctx, o)
	errCh := make(chan error, gatewayCount+a.GetRegistry().GetCountWorkers()+a.GetRegistry().GetCountConsumers()+a.GetRegistry().GetCountServices()+a.GetRegistry().GetCountServices())
	a.registerServices(ctx, errCh)
	go a.registerAPI(ctx)
	go a.registerWorkers(ctx)
	go a.registerConsumers(ctx)

	a.run(ctx, errCh)
}

func (a *App) runService(ctx context.Context, o *obs.Observability) {
	go a.registerMetrics(ctx, o)
	errCh := make(chan error, a.GetRegistry().GetCountServices())
	a.registerServices(ctx, errCh)
	a.run(ctx, errCh)
}

func (a *App) runWorker(ctx context.Context, o *obs.Observability) {
	go a.registerMetrics(ctx, o)
	go a.registerWorkers(ctx)
	errCh := make(chan error, a.GetRegistry().GetCountWorkers())
	a.run(ctx, errCh)
}

func (a *App) runConsumer(ctx context.Context, o *obs.Observability) {
	go a.registerMetrics(ctx, o)
	go a.registerConsumers(ctx)
	errCh := make(chan error, a.GetRegistry().GetCountConsumers())
	a.run(ctx, errCh)
}

func (a *App) runGateway(ctx context.Context, o *obs.Observability) {
	go a.registerMetrics(ctx, o)
	go a.registerAPI(ctx)
	errCh := make(chan error, gatewayCount)
	a.run(ctx, errCh)
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
	if v := strings.TrimSpace(os.Getenv("APP_CORS_ORIGINS")); v != "" {
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
