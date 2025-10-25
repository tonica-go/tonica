package service

import (
	"context"
	"log"

	"github.com/alexrett/tonica/pkg/tonica/grpc/serviceconfig"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

type GRPCRegistrar func(*grpc.Server, *Service)
type GRPCClient func(conn *grpc.ClientConn)

type GatewayRegistrar func(ctx context.Context, mux *runtime.ServeMux, target string, dialOpts []grpc.DialOption) error
type NamedGatewayRegistrar struct {
	Name      string
	Registrar GatewayRegistrar
}

type Service struct {
	config  *Config
	storage *Storage

	logger *log.Logger

	clientConnections serviceconfig.ClientConnections

	grpc       GRPCRegistrar
	grpcClient GRPCClient

	gatewayRegistrar GatewayRegistrar
	isGatewayEnabled bool
}

type Config struct {
	Name     string
	GrpcAddr string
}

type Storage struct {
	db     *DB
	rdb    *Redis
	pubsub *PubSub
}

type DB struct {
	dsn string
}

type Redis struct {
	dsn string
}

type PubSub struct {
	dsn string
}

func NewService(options ...Option) *Service {
	app := &Service{
		storage:           &Storage{},
		config:            &Config{},
		clientConnections: serviceconfig.ClientConnections{},
	}

	for _, option := range options {
		option(app)
	}

	return app
}
