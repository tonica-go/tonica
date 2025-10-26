package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedis_GetClient(t *testing.T) {
	t.Run("should create and cache connection", func(t *testing.T) {
		redis := &Redis{
			addr:     "localhost:6379",
			password: "",
			database: 0,
		}

		// First call should create connection
		client1 := redis.GetClient()
		require.NotNil(t, client1, "first call should return non-nil client")

		// Second call should return the same instance
		client2 := redis.GetClient()
		require.NotNil(t, client2, "second call should return non-nil client")

		assert.Same(t, client1, client2, "should return the same client instance")
	})

	t.Run("should not create multiple connections", func(t *testing.T) {
		redis := &Redis{
			addr:     "localhost:6379",
			password: "secret",
			database: 1,
		}

		// Multiple calls
		clients := make([]interface{}, 10)
		for i := 0; i < 10; i++ {
			clients[i] = redis.GetClient()
		}

		// All should be the same instance
		first := clients[0]
		for i := 1; i < len(clients); i++ {
			assert.Same(t, first, clients[i], "all calls should return the same instance")
		}
	})
}

func TestDB_GetClient_Postgres(t *testing.T) {
	t.Run("should create and cache postgres connection", func(t *testing.T) {
		db := &DB{
			dsn:    "postgres://localhost:5432/test",
			driver: Postgres,
		}

		// First call should create connection
		client1 := db.GetClient()
		require.NotNil(t, client1, "first call should return non-nil client")

		// Second call should return the same instance
		client2 := db.GetClient()
		require.NotNil(t, client2, "second call should return non-nil client")

		assert.Same(t, client1, client2, "should return the same client instance")
	})

	t.Run("should use postgres by default", func(t *testing.T) {
		db := &DB{
			dsn:    "postgres://localhost:5432/test",
			driver: "", // empty driver
		}

		client := db.GetClient()
		require.NotNil(t, client, "should create client with empty driver")
	})

	t.Run("should not create multiple connections", func(t *testing.T) {
		db := &DB{
			dsn:    "postgres://localhost:5432/test",
			driver: Postgres,
		}

		// Multiple calls
		clients := make([]interface{}, 10)
		for i := 0; i < 10; i++ {
			clients[i] = db.GetClient()
		}

		// All should be the same instance
		first := clients[0]
		for i := 1; i < len(clients); i++ {
			assert.Same(t, first, clients[i], "all calls should return the same instance")
		}
	})
}

func TestDB_GetClient_Sqlite(t *testing.T) {
	t.Run("should create and cache sqlite connection", func(t *testing.T) {
		db := &DB{
			dsn:    ":memory:",
			driver: Sqlite,
		}

		// First call should create connection
		client1 := db.GetClient()
		require.NotNil(t, client1, "first call should return non-nil client")

		// Second call should return the same instance
		client2 := db.GetClient()
		require.NotNil(t, client2, "second call should return non-nil client")

		assert.Same(t, client1, client2, "should return the same client instance")
	})
}

func TestDB_GetClient_Mysql(t *testing.T) {
	t.Run("should create and cache mysql connection", func(t *testing.T) {
		// MySQL requires a more specific DSN format
		db := &DB{
			dsn:    "user:password@tcp(localhost:3306)/dbname",
			driver: Mysql,
		}

		// First call should create connection
		client1 := db.GetClient()
		require.NotNil(t, client1, "first call should return non-nil client")

		// Second call should return the same instance
		client2 := db.GetClient()
		require.NotNil(t, client2, "second call should return non-nil client")

		assert.Same(t, client1, client2, "should return the same client instance")
	})
}

func TestNewService(t *testing.T) {
	t.Run("should create service with default values", func(t *testing.T) {
		svc := NewService()

		assert.NotNil(t, svc)
		assert.NotNil(t, svc.storage)
		assert.NotNil(t, svc.config)
		assert.NotNil(t, svc.clientConnections)
	})

	t.Run("should apply options", func(t *testing.T) {
		svc := NewService(
			WithName("test-service"),
			WithGRPCAddr(":9999"),
		)

		assert.Equal(t, "test-service", svc.config.Name)
		assert.Equal(t, ":9999", svc.config.GrpcAddr)
	})
}
