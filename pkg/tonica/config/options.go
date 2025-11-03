package config

import "slices"

type Option func(config *Config)

func WithName(name string) Option {
	return func(cfg *Config) {
		cfg.appName = name
	}
}

func WithVersion(version string) Option {
	return func(cfg *Config) {
		cfg.version = version
	}
}

func WithDebugMode(debug bool) Option {
	return func(cfg *Config) {
		cfg.debugMode = debug
	}
}

func WithRunMode(mode string) Option {
	available := []string{ModeAIO, ModeService, ModeWorker, ModeConsumer}
	if !slices.Contains(available, mode) {
		mode = ModeAIO
	}
	return func(cfg *Config) {
		cfg.runMode = mode
	}
}

func WithServices(services []string) Option {
	return func(cfg *Config) {
		cfg.services = services
	}
}

func WithWorkers(workers []string) Option {
	return func(cfg *Config) {
		cfg.workers = workers
	}
}

func WithConsumers(consumers []string) Option {
	return func(cfg *Config) {
		cfg.consumers = consumers
	}
}
