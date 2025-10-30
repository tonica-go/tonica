package service

import (
	"github.com/tonica-go/tonica/pkg/tonica/grpc/serviceconfig"
	"github.com/uptrace/bun"
	"google.golang.org/grpc"
)

func (s *Service) GetName() string {
	return s.config.Name
}

func (s *Service) GetClientConnections() serviceconfig.ClientConnections {
	return s.clientConnections
}

func (s *Service) GetGRPC() GRPCRegistrar {
	return s.grpc
}

func (s *Service) GetGRPCAddr() string {
	return s.config.GrpcAddr
}

func (s *Service) GetClientConn() *grpc.ClientConn {
	return s.clientConnections.CreateNewConnection(serviceconfig.ServiceConfig{
		Address: s.config.GrpcAddr,
		Name:    s.config.Name,
	})
}

func (s *Service) GetGateway() GatewayRegistrar {
	return s.gatewayRegistrar
}

func (s *Service) GetIsGatewayEnabled() bool {
	return s.isGatewayEnabled
}

func (s *Service) GetDB() *DB {
	if s.storage == nil || s.storage.db == nil {
		return nil
	}
	return s.storage.db
}

func (s *Service) GetDBClient() *bun.DB {
	db := s.GetDB()
	if db == nil {
		return nil
	}
	return db.GetClient()
}
