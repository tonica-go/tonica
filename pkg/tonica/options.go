package tonica

import (
	"log"

	"github.com/tonica-go/tonica/pkg/tonica/config"
	"github.com/tonica-go/tonica/pkg/tonica/registry"
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

func WithSpecUrl(spec string) AppOption {
	return func(a *App) {
		a.specUrl = spec
	}
}

func WithAPIPrefix(prefix string) AppOption {
	return func(a *App) {
		a.apiPrefix = prefix
	}
}

func WithWorkflowService(namespace string) AppOption {
	return func(a *App) {
		a.isWorkflowService = true
		a.workflowNamespace = namespace
	}
}

func WithEntityService(definitionsPath, dbDriver, dsn string) AppOption {
	return func(a *App) {
		a.isEntityService = true
		a.entityDefinitions = definitionsPath
		a.entityDriver = dbDriver
		a.entityDSN = dsn
	}
}
