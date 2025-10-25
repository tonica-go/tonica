package service

import (
	"context"
	"log"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/redis/go-redis/v9"
	"github.com/tonica-go/tonica/pkg/tonica/grpc/serviceconfig"
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
	addr     string
	password string
	database int
	conn     *redis.Client
}

func (r *Redis) GetClient() *redis.Client {
	if r.conn == nil {
		redis.NewClient(&redis.Options{
			Addr:     r.addr,
			Password: r.password, // no password set
			DB:       r.database, // use default DB
		})
	}
	return r.conn
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
