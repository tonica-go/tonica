package registry

import (
	"fmt"

	"github.com/tonica-go/tonica/pkg/tonica/consumer"
	"github.com/tonica-go/tonica/pkg/tonica/service"
	"github.com/tonica-go/tonica/pkg/tonica/worker"
)

type Registry interface {
	RegisterService(item *service.Service) error
	MustRegisterService(item *service.Service)
	GetService(name string) (*service.Service, error)
	GetAllServices() ([]*service.Service, error)

	RegisterWorker(name string, item *worker.Worker) error
	GetWorker(name string) (*worker.Worker, error)
	GetAllWorkers() ([]*worker.Worker, error)

	RegisterConsumer(name string, item *consumer.Consumer) error
	GetConsumer(name string) (*consumer.Consumer, error)
	GetAllConsumers() ([]*consumer.Consumer, error)
}

type AppRegistry struct {
	services  map[string]*service.Service
	workers   map[string]*worker.Worker
	consumers map[string]*consumer.Consumer
}

func (r *AppRegistry) RegisterService(item *service.Service) error {
	name := item.GetName()
	if _, ok := r.services[name]; ok {
		return fmt.Errorf("service %s already exists", name)
	}
	r.services[name] = item
	return nil
}

func (r *AppRegistry) MustRegisterService(item *service.Service) {
	name := item.GetName()
	if _, ok := r.services[name]; ok {
		panic(fmt.Errorf("service %s already exists", name))
	}
	r.services[name] = item
}

func (r *AppRegistry) GetService(name string) (*service.Service, error) {
	if item, ok := r.services[name]; ok {
		return item, nil
	}
	return nil, fmt.Errorf("service %s not found", name)
}

func (r *AppRegistry) GetAllServices() ([]*service.Service, error) {
	services := make([]*service.Service, 0, len(r.services))
	for _, src := range r.services {
		services = append(services, src)
	}
	return services, nil
}

func (r *AppRegistry) RegisterWorker(name string, item *worker.Worker) error {
	if _, ok := r.workers[name]; ok {
		return fmt.Errorf("worker %s already exists", name)
	}
	r.workers[name] = item
	return nil
}

func (r *AppRegistry) GetWorker(name string) (*worker.Worker, error) {
	if item, ok := r.workers[name]; ok {
		return item, nil
	}
	return nil, fmt.Errorf("worker %s not found", name)
}

func (r *AppRegistry) GetAllWorkers() ([]*worker.Worker, error) {
	workers := make([]*worker.Worker, 0, len(r.workers))
	for _, src := range r.workers {
		workers = append(workers, src)
	}
	return workers, nil
}

func (r *AppRegistry) RegisterConsumer(name string, item *consumer.Consumer) error {
	if _, ok := r.consumers[name]; ok {
		return fmt.Errorf("consumer %s already exists", name)
	}
	r.consumers[name] = item
	return nil
}
func (r *AppRegistry) GetConsumer(name string) (*consumer.Consumer, error) {
	if item, ok := r.consumers[name]; ok {
		return item, nil
	}
	return nil, fmt.Errorf("consumer %s not found", name)
}

func (r *AppRegistry) GetAllConsumers() ([]*consumer.Consumer, error) {
	consumers := make([]*consumer.Consumer, 0, len(r.consumers))
	for _, src := range r.consumers {
		consumers = append(consumers, src)
	}
	return consumers, nil
}

func NewRegistry() Registry {
	return &AppRegistry{
		services:  make(map[string]*service.Service),
		workers:   make(map[string]*worker.Worker),
		consumers: make(map[string]*consumer.Consumer),
	}
}
