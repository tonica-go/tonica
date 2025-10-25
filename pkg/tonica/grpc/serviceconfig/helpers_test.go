package serviceconfig //nolint:testpackage // tests

import (
	"context"
	"net"
	"testing"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener //nolint:gochecknoglobals // required for testing

func init() { //nolint:gochecknoinits // required for testing
	lis = bufconn.Listen(bufSize)
}

func bufDialer(_ context.Context, _ string) (net.Conn, error) {
	return lis.Dial()
}

func TestMustCreateNewNonBlockingServiceConnection_Success(t *testing.T) {
	config := ServiceConfig{
		Address: "bufconn",
		Name:    "test-service",
	}

	// Устанавливаем флаг для перехвата вызова log.Panicf
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("function panicked: %v", r)
		}
	}()

	// Вызываем функцию
	conn := MustCreateNewNonBlockingServiceConnection(config, grpc.WithContextDialer(bufDialer))

	// Проверяем, что соединение создано успешно
	require.NotNil(t, conn)

	// Закрываем соединение
	require.NoError(t, conn.Close())
}

func TestCreateDefaultConnectionParams(t *testing.T) {
	// Вызываем функцию
	params := CreateDefaultConnectionParams()

	// Проверяем, что параметры были созданы с правильным значением maxDelay
	require.Equal(t, maxDelay, params.Backoff.MaxDelay)
}

func TestMustCreateNewNonBlockingServiceConnection_Options(t *testing.T) {
	config := ServiceConfig{
		Address: "bufconn",
		Name:    "test-service",
	}

	// Устанавливаем флаг для перехвата вызова log.Panicf
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("function panicked: %v", r)
		}
	}()

	// Вызываем функцию с доп. опциями
	conn := MustCreateNewNonBlockingServiceConnection(config, grpc.WithChainUnaryInterceptor(retry.UnaryClientInterceptor()))

	// Проверяем, что соединение успешно установлено
	require.NotNil(t, conn)

	// Закрываем соединение
	require.NoError(t, conn.Close())
}
