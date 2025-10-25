package service

import (
	"context"
	"database/sql"
	"log"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/redis/go-redis/v9"
	"github.com/tonica-go/tonica/pkg/tonica/grpc/serviceconfig"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"google.golang.org/grpc"

	_ "github.com/go-sql-driver/mysql"
	"github.com/uptrace/bun/dialect/mysqldialect"

	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
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
	dsn    string
	driver string
	db     *bun.DB
}

func (d *DB) GetClient() *bun.DB {
	if d.driver == Postgres || d.driver == "" {
		sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(d.dsn)))
		if d.db == nil {
			d.db = bun.NewDB(sqldb, pgdialect.New())
		}
	}

	if d.driver == Mysql {
		sqldb, err := sql.Open("mysql", d.dsn)
		if err != nil {
			panic(err)
		}
		if d.db == nil {
			d.db = bun.NewDB(sqldb, mysqldialect.New())
		}
	}

	if d.driver == Sqlite {
		sqldb, err := sql.Open(sqliteshim.ShimName, d.dsn)
		if err != nil {
			panic(err)
		}
		if d.db == nil {
			d.db = bun.NewDB(sqldb, sqlitedialect.New())
		}
	}

	return d.db
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
