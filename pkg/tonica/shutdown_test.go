package tonica

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestNewShutdown(t *testing.T) {
	s := NewShutdown()

	assert.NotNil(t, s)
	assert.NotNil(t, s.httpServers)
	assert.NotNil(t, s.grpcServers)
	assert.NotNil(t, s.cleanupFns)
	assert.Empty(t, s.httpServers)
	assert.Empty(t, s.grpcServers)
	assert.Empty(t, s.cleanupFns)
}

func TestShutdown_RegisterHTTPServer(t *testing.T) {
	s := NewShutdown()

	srv1 := &http.Server{Addr: ":8080"}
	srv2 := &http.Server{Addr: ":8081"}

	s.RegisterHTTPServer(srv1)
	s.RegisterHTTPServer(srv2)

	assert.Len(t, s.httpServers, 2)
	assert.Contains(t, s.httpServers, srv1)
	assert.Contains(t, s.httpServers, srv2)
}

func TestShutdown_RegisterGRPCServer(t *testing.T) {
	s := NewShutdown()

	srv1 := grpc.NewServer()
	srv2 := grpc.NewServer()

	s.RegisterGRPCServer(srv1)
	s.RegisterGRPCServer(srv2)

	assert.Len(t, s.grpcServers, 2)
	assert.Contains(t, s.grpcServers, srv1)
	assert.Contains(t, s.grpcServers, srv2)
}

func TestShutdown_RegisterCleanup(t *testing.T) {
	s := NewShutdown()

	called := false
	cleanup := func(ctx context.Context) error {
		called = true
		return nil
	}

	s.RegisterCleanup(cleanup)

	assert.Len(t, s.cleanupFns, 1)

	// Execute to verify function is registered
	err := s.Execute(1 * time.Second)
	assert.NoError(t, err)
	assert.True(t, called, "cleanup function should be called")
}

func TestShutdown_Execute_HTTPServer(t *testing.T) {
	t.Run("should shutdown HTTP server gracefully", func(t *testing.T) {
		s := NewShutdown()

		// Create and start HTTP server
		srv := &http.Server{Addr: ":0"} // random port
		listener, err := net.Listen("tcp", srv.Addr)
		require.NoError(t, err)
		defer listener.Close()

		go func() {
			srv.Serve(listener)
		}()

		s.RegisterHTTPServer(srv)

		// Execute shutdown
		err = s.Execute(1 * time.Second)
		assert.NoError(t, err)
	})

	// Note: Testing timeout scenarios is complex and time-dependent
	// We've verified graceful shutdown works, which is the primary requirement
}

func TestShutdown_Execute_GRPCServer(t *testing.T) {
	t.Run("should shutdown gRPC server gracefully", func(t *testing.T) {
		s := NewShutdown()

		// Create and start gRPC server
		srv := grpc.NewServer()
		listener, err := net.Listen("tcp", ":0")
		require.NoError(t, err)

		go func() {
			srv.Serve(listener)
		}()

		s.RegisterGRPCServer(srv)

		// Execute shutdown
		err = s.Execute(1 * time.Second)
		assert.NoError(t, err)
	})

	// Note: Testing timeout scenarios is complex and time-dependent
	// We've verified graceful shutdown works, which is the primary requirement
}

func TestShutdown_Execute_CleanupFunctions(t *testing.T) {
	t.Run("should execute all cleanup functions", func(t *testing.T) {
		s := NewShutdown()

		called := make([]bool, 3)

		for i := 0; i < 3; i++ {
			index := i
			s.RegisterCleanup(func(ctx context.Context) error {
				called[index] = true
				return nil
			})
		}

		err := s.Execute(1 * time.Second)
		assert.NoError(t, err)

		for i, wasCalled := range called {
			assert.True(t, wasCalled, "cleanup function %d should be called", i)
		}
	})

	t.Run("should return first error from cleanup", func(t *testing.T) {
		s := NewShutdown()

		testErr := errors.New("cleanup error")

		s.RegisterCleanup(func(ctx context.Context) error {
			return nil
		})

		s.RegisterCleanup(func(ctx context.Context) error {
			return testErr
		})

		s.RegisterCleanup(func(ctx context.Context) error {
			return nil
		})

		err := s.Execute(1 * time.Second)
		assert.Error(t, err)
		assert.Equal(t, testErr, err)
	})

	t.Run("should timeout cleanup functions", func(t *testing.T) {
		s := NewShutdown()

		s.RegisterCleanup(func(ctx context.Context) error {
			time.Sleep(100 * time.Millisecond)
			return nil
		})

		err := s.Execute(10 * time.Millisecond)
		assert.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
	})
}

func TestShutdown_Execute_Multiple(t *testing.T) {
	t.Run("should shutdown all components in parallel", func(t *testing.T) {
		s := NewShutdown()

		// HTTP server
		httpSrv := &http.Server{Addr: ":0"}
		httpListener, err := net.Listen("tcp", httpSrv.Addr)
		require.NoError(t, err)
		defer httpListener.Close()

		go func() {
			httpSrv.Serve(httpListener)
		}()

		// gRPC server
		grpcSrv := grpc.NewServer()
		grpcListener, err := net.Listen("tcp", ":0")
		require.NoError(t, err)

		go func() {
			grpcSrv.Serve(grpcListener)
		}()

		// Cleanup function
		cleanupCalled := false
		cleanup := func(ctx context.Context) error {
			cleanupCalled = true
			return nil
		}

		s.RegisterHTTPServer(httpSrv)
		s.RegisterGRPCServer(grpcSrv)
		s.RegisterCleanup(cleanup)

		// Execute shutdown
		start := time.Now()
		err = s.Execute(2 * time.Second)
		duration := time.Since(start)

		assert.NoError(t, err)
		assert.True(t, cleanupCalled)
		// Should complete quickly since all shutdowns happen in parallel
		assert.Less(t, duration, 1*time.Second)
	})
}

func TestShutdown_Execute_Empty(t *testing.T) {
	t.Run("should complete immediately with no registered components", func(t *testing.T) {
		s := NewShutdown()

		start := time.Now()
		err := s.Execute(1 * time.Second)
		duration := time.Since(start)

		assert.NoError(t, err)
		assert.Less(t, duration, 100*time.Millisecond)
	})
}

func TestShutdown_Concurrency(t *testing.T) {
	t.Run("should be safe for concurrent registration", func(t *testing.T) {
		s := NewShutdown()

		done := make(chan bool, 3)

		// Concurrent HTTP server registration
		go func() {
			for i := 0; i < 10; i++ {
				s.RegisterHTTPServer(&http.Server{})
			}
			done <- true
		}()

		// Concurrent gRPC server registration
		go func() {
			for i := 0; i < 10; i++ {
				s.RegisterGRPCServer(grpc.NewServer())
			}
			done <- true
		}()

		// Concurrent cleanup registration
		go func() {
			for i := 0; i < 10; i++ {
				s.RegisterCleanup(func(ctx context.Context) error { return nil })
			}
			done <- true
		}()

		// Wait for all goroutines
		for i := 0; i < 3; i++ {
			<-done
		}

		assert.Len(t, s.httpServers, 10)
		assert.Len(t, s.grpcServers, 10)
		assert.Len(t, s.cleanupFns, 10)
	})
}
