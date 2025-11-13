package tonica

import (
	"log"

	"github.com/gin-gonic/gin"
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

// WithGatewayProtoMessages enables the use of proto messages fields in the API Gateway (snakecase instead of camelCase)
func WithGatewayProtoMessages() AppOption {
	return func(a *App) {
		a.useGatewayProtoMessages = true
	}
}

// WithRouteMiddleware adds middleware for specific route patterns
// Example:
//
//	WithRouteMiddleware([]string{"/public"}, authMiddleware1, authMiddleware2)
//	WithRouteMiddleware([]string{"/api/v1", "/api/v2"}, rateLimitMiddleware)
func WithRouteMiddleware(pathPrefixes []string, middlewares ...gin.HandlerFunc) AppOption {
	return func(a *App) {
		a.routeMiddlewares = append(a.routeMiddlewares, RouteMiddleware{
			PathPrefixes: pathPrefixes,
			Middlewares:  middlewares,
		})
	}
}

func WithCustomGrpcHeaders(headers map[string]string) AppOption {
	return func(a *App) {
		if a.customGrpcHeaders == nil {
			a.customGrpcHeaders = make(map[string]string)
		}

		for k, v := range headers {
			a.customGrpcHeaders[k] = v
		}
	}
}
