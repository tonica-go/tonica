package tonica

import (
	"log"

	"github.com/alexrett/tonica/pkg/tonica/config"
	"github.com/alexrett/tonica/pkg/tonica/registry"
)

type AppOption func(*App)

func WithName(name string) AppOption {
	return func(a *App) {
		a.Name = name
	}
}

func WithRegistry(r registry.Registry) AppOption {
	return func(a *App) {
		a.registry = r
	}
}

func WithLogger(l *log.Logger) AppOption {
	return func(a *App) {
		a.logger = l
	}
}

func WithConfig(cfg *config.Config) AppOption {
	return func(a *App) {
		a.cfg = cfg
	}
}

func WithSpec(spec string) AppOption {
	return func(a *App) {
		a.spec = spec
	}
}
