package tonica

import (
	"log/slog"

	"github.com/tonica-go/tonica/pkg/tonica/config"
	"go.temporal.io/sdk/client"

	"os"
)

func GetTemporalClient(namespace string) (client.Client, error) {
	temporalAddr := config.GetEnv("TEMPORAL_ADDR", "localhost:7233")
	if namespace == "" {
		namespace = config.GetEnv("TEMPORAL_NAMESPACE", "default")
	}
	return client.Dial(client.Options{
		HostPort:  temporalAddr,
		Namespace: namespace,
	})
}

func MustGetTemporalClient(namespace string) client.Client {
	temporalClient, err := GetTemporalClient(namespace)
	if err != nil {
		slog.Error("Error getting temporal client", "error", err)
		os.Exit(1)
	}

	return temporalClient
}
