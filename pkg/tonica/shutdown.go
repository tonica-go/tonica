package tonica

import (
	"context"
	"net/http"
	"sync"
	"time"

	"google.golang.org/grpc"
)

// Shutdown coordinates graceful shutdown of all app components
type Shutdown struct {
	httpServers []*http.Server
	grpcServers []*grpc.Server
	cleanupFns  []func(context.Context) error
	mu          sync.Mutex
}

// NewShutdown creates a new shutdown coordinator
func NewShutdown() *Shutdown {
	return &Shutdown{
		httpServers: make([]*http.Server, 0),
		grpcServers: make([]*grpc.Server, 0),
		cleanupFns:  make([]func(context.Context) error, 0),
	}
}

// RegisterHTTPServer adds an HTTP server to be gracefully stopped
func (s *Shutdown) RegisterHTTPServer(srv *http.Server) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.httpServers = append(s.httpServers, srv)
}

// RegisterGRPCServer adds a gRPC server to be gracefully stopped
func (s *Shutdown) RegisterGRPCServer(srv *grpc.Server) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.grpcServers = append(s.grpcServers, srv)
}

// RegisterCleanup adds a cleanup function to be called during shutdown
func (s *Shutdown) RegisterCleanup(fn func(context.Context) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupFns = append(s.cleanupFns, fn)
}

// Execute performs graceful shutdown of all registered components
func (s *Shutdown) Execute(timeout time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var wg sync.WaitGroup
	errCh := make(chan error, len(s.httpServers)+len(s.grpcServers)+len(s.cleanupFns))

	// Shutdown HTTP servers
	for _, srv := range s.httpServers {
		wg.Add(1)
		go func(server *http.Server) {
			defer wg.Done()
			if err := server.Shutdown(ctx); err != nil {
				errCh <- err
			}
		}(srv)
	}

	// Shutdown gRPC servers
	for _, srv := range s.grpcServers {
		wg.Add(1)
		go func(server *grpc.Server) {
			defer wg.Done()
			// GracefulStop stops accepting new requests and waits for existing to complete
			stopped := make(chan struct{})
			go func() {
				server.GracefulStop()
				close(stopped)
			}()

			select {
			case <-stopped:
				// Graceful stop completed
			case <-ctx.Done():
				// Timeout reached, force stop
				server.Stop()
			}
		}(srv)
	}

	// Run cleanup functions
	for _, fn := range s.cleanupFns {
		wg.Add(1)
		go func(cleanup func(context.Context) error) {
			defer wg.Done()
			if err := cleanup(ctx); err != nil {
				errCh <- err
			}
		}(fn)
	}

	// Wait for all shutdowns to complete
	doneCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneCh)
	}()

	select {
	case <-doneCh:
		close(errCh)
		// Collect all errors
		var firstErr error
		for err := range errCh {
			if firstErr == nil {
				firstErr = err
			}
		}
		return firstErr
	case <-ctx.Done():
		return ctx.Err()
	}
}
