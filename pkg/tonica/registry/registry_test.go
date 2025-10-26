package registry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tonica-go/tonica/pkg/tonica/consumer"
	"github.com/tonica-go/tonica/pkg/tonica/service"
	"github.com/tonica-go/tonica/pkg/tonica/worker"
)

func TestNewRegistry(t *testing.T) {
	reg := NewRegistry()

	assert.NotNil(t, reg)

	// Should be empty initially
	services, err := reg.GetAllServices()
	assert.NoError(t, err)
	assert.Empty(t, services)

	workers, err := reg.GetAllWorkers()
	assert.NoError(t, err)
	assert.Empty(t, workers)

	consumers, err := reg.GetAllConsumers()
	assert.NoError(t, err)
	assert.Empty(t, consumers)
}

func TestRegistry_Services(t *testing.T) {
	reg := NewRegistry()

	t.Run("RegisterService", func(t *testing.T) {
		svc := service.NewService(service.WithName("test-service"))

		err := reg.RegisterService(svc)
		assert.NoError(t, err)

		// Get the service back
		retrieved, err := reg.GetService("test-service")
		assert.NoError(t, err)
		assert.Same(t, svc, retrieved)
	})

	t.Run("RegisterService duplicate", func(t *testing.T) {
		svc1 := service.NewService(service.WithName("duplicate-service"))
		svc2 := service.NewService(service.WithName("duplicate-service"))

		err := reg.RegisterService(svc1)
		assert.NoError(t, err)

		err = reg.RegisterService(svc2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("MustRegisterService", func(t *testing.T) {
		svc := service.NewService(service.WithName("must-service"))

		assert.NotPanics(t, func() {
			reg.MustRegisterService(svc)
		})
	})

	t.Run("MustRegisterService panic on duplicate", func(t *testing.T) {
		svc1 := service.NewService(service.WithName("panic-service"))
		svc2 := service.NewService(service.WithName("panic-service"))

		reg.MustRegisterService(svc1)

		assert.Panics(t, func() {
			reg.MustRegisterService(svc2)
		})
	})

	t.Run("GetService not found", func(t *testing.T) {
		_, err := reg.GetService("non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("GetAllServices", func(t *testing.T) {
		reg := NewRegistry()

		svc1 := service.NewService(service.WithName("service-1"))
		svc2 := service.NewService(service.WithName("service-2"))
		svc3 := service.NewService(service.WithName("service-3"))

		reg.RegisterService(svc1)
		reg.RegisterService(svc2)
		reg.RegisterService(svc3)

		services, err := reg.GetAllServices()
		assert.NoError(t, err)
		assert.Len(t, services, 3)

		// Verify all services are present
		names := make(map[string]bool)
		for _, svc := range services {
			names[svc.GetName()] = true
		}

		assert.True(t, names["service-1"])
		assert.True(t, names["service-2"])
		assert.True(t, names["service-3"])
	})
}

func TestRegistry_Workers(t *testing.T) {
	reg := NewRegistry()

	t.Run("RegisterWorker", func(t *testing.T) {
		wrk := worker.NewWorker(worker.WithName("test-worker"))

		err := reg.RegisterWorker("test-worker", wrk)
		assert.NoError(t, err)

		// Get the worker back
		retrieved, err := reg.GetWorker("test-worker")
		assert.NoError(t, err)
		assert.Same(t, wrk, retrieved)
	})

	t.Run("RegisterWorker duplicate", func(t *testing.T) {
		wrk1 := worker.NewWorker(worker.WithName("duplicate-worker"))
		wrk2 := worker.NewWorker(worker.WithName("duplicate-worker"))

		err := reg.RegisterWorker("duplicate-worker", wrk1)
		assert.NoError(t, err)

		err = reg.RegisterWorker("duplicate-worker", wrk2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("GetWorker not found", func(t *testing.T) {
		_, err := reg.GetWorker("non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("GetAllWorkers", func(t *testing.T) {
		reg := NewRegistry()

		wrk1 := worker.NewWorker(worker.WithName("worker-1"))
		wrk2 := worker.NewWorker(worker.WithName("worker-2"))

		reg.RegisterWorker("worker-1", wrk1)
		reg.RegisterWorker("worker-2", wrk2)

		workers, err := reg.GetAllWorkers()
		assert.NoError(t, err)
		assert.Len(t, workers, 2)
	})
}

func TestRegistry_Consumers(t *testing.T) {
	reg := NewRegistry()

	t.Run("RegisterConsumer", func(t *testing.T) {
		cons := consumer.NewConsumer(consumer.WithName("test-consumer"))

		err := reg.RegisterConsumer("test-consumer", cons)
		assert.NoError(t, err)

		// Get the consumer back
		retrieved, err := reg.GetConsumer("test-consumer")
		assert.NoError(t, err)
		assert.Same(t, cons, retrieved)
	})

	t.Run("RegisterConsumer duplicate", func(t *testing.T) {
		cons1 := consumer.NewConsumer(consumer.WithName("duplicate-consumer"))
		cons2 := consumer.NewConsumer(consumer.WithName("duplicate-consumer"))

		err := reg.RegisterConsumer("duplicate-consumer", cons1)
		assert.NoError(t, err)

		err = reg.RegisterConsumer("duplicate-consumer", cons2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("MustRegisterConsumer", func(t *testing.T) {
		cons := consumer.NewConsumer(consumer.WithName("must-consumer"))

		assert.NotPanics(t, func() {
			reg.MustRegisterConsumer(cons)
		})
	})

	t.Run("MustRegisterConsumer panic on duplicate", func(t *testing.T) {
		cons1 := consumer.NewConsumer(consumer.WithName("panic-consumer"))
		cons2 := consumer.NewConsumer(consumer.WithName("panic-consumer"))

		reg.MustRegisterConsumer(cons1)

		assert.Panics(t, func() {
			reg.MustRegisterConsumer(cons2)
		})
	})

	t.Run("GetConsumer not found", func(t *testing.T) {
		_, err := reg.GetConsumer("non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("GetAllConsumers", func(t *testing.T) {
		reg := NewRegistry()

		cons1 := consumer.NewConsumer(consumer.WithName("consumer-1"))
		cons2 := consumer.NewConsumer(consumer.WithName("consumer-2"))
		cons3 := consumer.NewConsumer(consumer.WithName("consumer-3"))

		reg.RegisterConsumer("consumer-1", cons1)
		reg.RegisterConsumer("consumer-2", cons2)
		reg.RegisterConsumer("consumer-3", cons3)

		consumers, err := reg.GetAllConsumers()
		assert.NoError(t, err)
		assert.Len(t, consumers, 3)

		// Verify all consumers are present
		names := make(map[string]bool)
		for _, cons := range consumers {
			names[cons.GetName()] = true
		}

		assert.True(t, names["consumer-1"])
		assert.True(t, names["consumer-2"])
		assert.True(t, names["consumer-3"])
	})
}

func TestRegistry_Mixed(t *testing.T) {
	t.Run("should handle multiple types", func(t *testing.T) {
		reg := NewRegistry()

		// Register different types
		svc := service.NewService(service.WithName("service-1"))
		wrk := worker.NewWorker(worker.WithName("worker-1"))
		cons := consumer.NewConsumer(consumer.WithName("consumer-1"))

		reg.RegisterService(svc)
		reg.RegisterWorker("worker-1", wrk)
		reg.RegisterConsumer("consumer-1", cons)

		// Verify counts
		services, _ := reg.GetAllServices()
		workers, _ := reg.GetAllWorkers()
		consumers, _ := reg.GetAllConsumers()

		assert.Len(t, services, 1)
		assert.Len(t, workers, 1)
		assert.Len(t, consumers, 1)
	})

	t.Run("should isolate namespaces", func(t *testing.T) {
		reg := NewRegistry()

		// Same name for different types - should not conflict
		svc := service.NewService(service.WithName("same-name"))
		wrk := worker.NewWorker(worker.WithName("same-name"))
		cons := consumer.NewConsumer(consumer.WithName("same-name"))

		err := reg.RegisterService(svc)
		assert.NoError(t, err)

		err = reg.RegisterWorker("same-name", wrk)
		assert.NoError(t, err)

		err = reg.RegisterConsumer("same-name", cons)
		assert.NoError(t, err)

		// All should be retrievable
		_, err = reg.GetService("same-name")
		assert.NoError(t, err)

		_, err = reg.GetWorker("same-name")
		assert.NoError(t, err)

		_, err = reg.GetConsumer("same-name")
		assert.NoError(t, err)
	})
}

func TestRegistry_EmptyResults(t *testing.T) {
	reg := NewRegistry()

	t.Run("GetAllServices empty", func(t *testing.T) {
		services, err := reg.GetAllServices()
		assert.NoError(t, err)
		assert.Empty(t, services)
	})

	t.Run("GetAllWorkers empty", func(t *testing.T) {
		workers, err := reg.GetAllWorkers()
		assert.NoError(t, err)
		assert.Empty(t, workers)
	})

	t.Run("GetAllConsumers empty", func(t *testing.T) {
		consumers, err := reg.GetAllConsumers()
		assert.NoError(t, err)
		assert.Empty(t, consumers)
	})
}
