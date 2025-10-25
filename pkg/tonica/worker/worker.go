package worker

import (
	"go.opentelemetry.io/otel"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/worker"

	oteltemporal "go.temporal.io/sdk/contrib/opentelemetry"
)

type Worker struct {
	activities []interface{}
	workflows  []interface{}
	queue      string
	name       string
	client     client.Client
}

func NewWorker(options ...Option) *Worker {
	app := &Worker{
		activities: []interface{}{},
		workflows:  []interface{}{},
	}

	for _, option := range options {
		option(app)
	}

	return app
}

func (app *Worker) Activities() []interface{} {
	return app.activities
}

func (app *Worker) Workflows() []interface{} {
	return app.workflows
}

func (app *Worker) Name() string {
	return app.name
}

func (app *Worker) Client() client.Client {
	return app.client
}

func (app *Worker) GetQueue() string {
	return app.queue
}

func (app *Worker) Start() error {
	ti, _ := oteltemporal.NewTracingInterceptor(oteltemporal.TracerOptions{TextMapPropagator: otel.GetTextMapPropagator()})
	w := worker.New(app.client, app.queue, worker.Options{Interceptors: []interceptor.WorkerInterceptor{ti.(interceptor.WorkerInterceptor)}})

	for _, activity := range app.activities {
		w.RegisterActivity(activity)
	}

	for _, workflow := range app.workflows {
		w.RegisterWorkflow(workflow)
	}

	return w.Run(worker.InterruptCh())
}

type Option func(worker *Worker)

func WithName(name string) Option {
	return func(a *Worker) {
		a.name = name
	}
}
func WithQueue(name string) Option {
	return func(a *Worker) {
		a.queue = name
	}
}

func WithClient(client client.Client) Option {
	return func(a *Worker) {
		a.client = client
	}
}

func WithActivities(activities []interface{}) Option {
	return func(a *Worker) {
		a.activities = activities
	}
}

func WithWorkflows(workflows []interface{}) Option {
	return func(a *Worker) {
		a.workflows = workflows
	}
}
