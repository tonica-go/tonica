package service

import "log"

type Option func(*Service)

func WithName(name string) Option {
	return func(a *Service) {
		a.config.Name = name
	}
}

func WithDB(dsn string) Option {
	return func(a *Service) {
		a.storage.db = &DB{
			dsn: dsn,
		}
	}
}

func WithRedis(dsn string) Option {
	return func(a *Service) {
		a.storage.rdb = &Redis{
			dsn: dsn,
		}
	}
}

func WithPubSub(dsn string) Option {
	return func(a *Service) {
		a.storage.pubsub = &PubSub{
			dsn: dsn,
		}
	}
}

func WithLogger(logger *log.Logger) Option {
	return func(a *Service) {
		a.logger = logger
	}
}

func WithGRPC(grpc GRPCRegistrar) Option {
	return func(a *Service) {
		a.grpc = grpc
	}
}

func WithGateway(gw GatewayRegistrar) Option {
	return func(a *Service) {
		a.gatewayRegistrar = gw
		a.isGatewayEnabled = true
	}
}

func WithGRPCAddr(grpcAddr string) Option {
	if grpcAddr == "" {
		grpcAddr = ":9000"
	}
	return func(a *Service) {
		a.config.GrpcAddr = grpcAddr
	}
}

func WithGRPClient(client GRPCClient) Option {
	return func(a *Service) {
		a.grpcClient = client
	}
}
