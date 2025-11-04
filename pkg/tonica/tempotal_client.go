package tonica

import (
	"log/slog"
	"os"

	"github.com/tonica-go/tonica/pkg/tonica/config"
	"github.com/tonica-go/tonica/pkg/tonica/identity"
	"go.opentelemetry.io/otel"
	"go.temporal.io/sdk/client"
	oteltemporal "go.temporal.io/sdk/contrib/opentelemetry"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/workflow"
)

// GetTemporalClient creates a Temporal client with observability and identity propagation
// The client automatically propagates OpenTelemetry traces, metrics, and user identity
// through workflows and activities
func GetTemporalClient(namespace string) (client.Client, error) {
	temporalAddr := config.GetEnv("TEMPORAL_ADDR", "localhost:7233")
	if namespace == "" {
		namespace = config.GetEnv("TEMPORAL_NAMESPACE", "default")
	}

	// Create OpenTelemetry tracing interceptor for trace propagation
	ti, err := oteltemporal.NewTracingInterceptor(oteltemporal.TracerOptions{
		TextMapPropagator: otel.GetTextMapPropagator(),
	})
	if err != nil {
		slog.Warn("failed to create temporal tracing interceptor", "error", err)
	}

	// Create OpenTelemetry metrics handler
	metricsHandler := oteltemporal.NewMetricsHandler(oteltemporal.MetricsHandlerOptions{
		Meter: otel.GetMeterProvider().Meter("temporal-client"),
	})

	opts := client.Options{
		HostPort:  temporalAddr,
		Namespace: namespace,
	}

	// Add interceptor if successfully created
	if ti != nil {
		opts.Interceptors = []interceptor.ClientInterceptor{ti.(interceptor.ClientInterceptor)}
	}

	// Add metrics handler
	opts.MetricsHandler = metricsHandler

	// Add identity context propagator for automatic identity propagation
	// This allows user authentication info to flow through workflows and activities
	opts.ContextPropagators = []workflow.ContextPropagator{
		identity.NewIdentityContextPropagator(),
	}

	return client.Dial(opts)
}

func MustGetTemporalClient(namespace string) client.Client {
	temporalClient, err := GetTemporalClient(namespace)
	if err != nil {
		slog.Error("Error getting temporal client", "error", err)
		os.Exit(1)
	}

	return temporalClient
}
