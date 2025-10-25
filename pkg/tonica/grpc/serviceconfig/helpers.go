package serviceconfig

import (
	"log"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
)

type ServiceConfig struct {
	Address string
	Name    string
}

const maxDelay = 10 * time.Second

func CreateDefaultConnectionParams() grpc.ConnectParams {
	bc := backoff.DefaultConfig
	bc.MaxDelay = maxDelay
	return grpc.ConnectParams{Backoff: bc}
}

func MustCreateNewNonBlockingServiceConnection(config ServiceConfig, opts ...grpc.DialOption) *grpc.ClientConn {
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	opts = append(opts, grpc.WithConnectParams(CreateDefaultConnectionParams()))
	opts = append(opts, grpc.WithChainUnaryInterceptor(
		retry.UnaryClientInterceptor(),
	))
	serviceConn, err := grpc.NewClient(config.Address, opts...)
	if err != nil {
		log.Panicf("unrecoverable error occurred whiule starting grpc dial context - %v", err)
	}

	return serviceConn
}

type ClientConnections struct {
	Connections []*grpc.ClientConn
}

func (c *ClientConnections) CreateNewConnection(serviceConfig ServiceConfig) *grpc.ClientConn {
	serviceConn, err := grpc.NewClient(serviceConfig.Address, grpc.WithConnectParams(CreateDefaultConnectionParams()), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	c.Connections = append(c.Connections, serviceConn)
	return serviceConn
}
