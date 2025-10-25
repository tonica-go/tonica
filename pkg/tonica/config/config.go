package config

const (
	DefaultAppName   = "Tonica"
	DefaultVersion   = "1.0.0"
	DefaultDebugMode = false

	DefaultWorkerName      = "DefaultWorker"
	DefaultWorkerNamespace = "default"
	DefaultWorkerQueueName = "default"

	DefaultServiceHost = "localhost"
	DefaultServicePort = 8080

	DefaultConsumerName = "DefaultConsumer"

	ModeAIO      = "aio"
	ModeService  = "service"
	ModeWorker   = "worker"
	ModeConsumer = "consumer"
)

type Config struct {
	appName string
	version string

	debugMode bool
	runMode   string

	services  []string
	workers   []WorkerConfig
	consumers []ConsumerConfig
}

func (c *Config) AppName() string {
	return c.appName
}

func (c *Config) Version() string {
	return c.version
}
func (c *Config) DebugMode() bool {
	return c.debugMode
}

func (c *Config) RunMode() string {
	return c.runMode
}

func (c *Config) Workers() []WorkerConfig {
	return c.workers
}

func (c *Config) Consumers() []ConsumerConfig {
	return c.consumers
}

func (c *Config) Services() []string {
	return c.services
}

type ServiceConfig struct {
	Name string
	Host string
	Port int
}

type WorkerConfig struct {
	WorkerName string
	Namespace  string
	QueueName  string
	Workflows  []string
	Activities []string
}

type ConsumerConfig struct {
	ConsumerName string
	Func         *func() error
}

func NewConfig(options ...Option) *Config {
	cfg := &Config{
		appName: DefaultAppName,
	}

	for _, option := range options {
		option(cfg)
	}

	return cfg
}
