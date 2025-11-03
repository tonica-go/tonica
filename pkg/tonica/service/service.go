package service

import (
	"context"
	"log"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/redis/go-redis/v9"
	"github.com/tonica-go/tonica/pkg/tonica/grpc/serviceconfig"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bunotel"
	"github.com/uptrace/opentelemetry-go-extra/otelsql"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
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
	if d.db != nil {
		return d.db
	}

	if d.driver == Postgres || d.driver == "" {
		// Wrap connector with OpenTelemetry tracing
		connector := pgdriver.NewConnector(pgdriver.WithDSN(d.dsn))
		sqldb := otelsql.OpenDB(connector,
			otelsql.WithAttributes(semconv.DBSystemKey.String("postgresql")),
			otelsql.WithDBName("postgres"),
		)
		d.db = bun.NewDB(sqldb, pgdialect.New())
		// Add Bun query hook for additional query formatting
		d.db.AddQueryHook(bunotel.NewQueryHook(
			bunotel.WithDBName("postgres"),
			bunotel.WithFormattedQueries(true),
		))
		return d.db
	}

	if d.driver == Mysql {
		// Use otelsql.Open to get traced sql.DB
		sqldb, err := otelsql.Open("mysql", d.dsn,
			otelsql.WithAttributes(semconv.DBSystemKey.String("mysql")),
			otelsql.WithDBName("mysql"),
		)
		if err != nil {
			panic(err)
		}
		d.db = bun.NewDB(sqldb, mysqldialect.New())
		// Add Bun query hook for additional query formatting
		d.db.AddQueryHook(bunotel.NewQueryHook(
			bunotel.WithDBName("mysql"),
			bunotel.WithFormattedQueries(true),
		))
		return d.db
	}

	if d.driver == Sqlite {
		// Use otelsql.Open to get traced sql.DB
		sqldb, err := otelsql.Open(sqliteshim.ShimName, d.dsn,
			otelsql.WithAttributes(semconv.DBSystemKey.String("sqlite")),
			otelsql.WithDBName("sqlite"),
		)
		if err != nil {
			panic(err)
		}
		d.db = bun.NewDB(sqldb, sqlitedialect.New())
		// Add Bun query hook for additional query formatting
		d.db.AddQueryHook(bunotel.NewQueryHook(
			bunotel.WithDBName("sqlite"),
			bunotel.WithFormattedQueries(true),
		))
		return d.db
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
		r.conn = redis.NewClient(&redis.Options{
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
