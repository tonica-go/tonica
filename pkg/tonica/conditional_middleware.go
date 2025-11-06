package tonica

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// ConditionalMiddleware creates a middleware that applies different middleware based on path patterns
// This is an alternative approach to route groups, useful when you want all logic in one place
type ConditionalMiddleware struct {
	rules []MiddlewareRule
}

// MiddlewareRule defines when to apply specific middleware
type MiddlewareRule struct {
	PathPrefixes []string          // paths that match these prefixes
	Middlewares  []gin.HandlerFunc // middleware to apply
}

// NewConditionalMiddleware creates a new conditional middleware
func NewConditionalMiddleware() *ConditionalMiddleware {
	return &ConditionalMiddleware{
		rules: make([]MiddlewareRule, 0),
	}
}

// AddRule adds a middleware rule
func (cm *ConditionalMiddleware) AddRule(pathPrefixes []string, middlewares ...gin.HandlerFunc) *ConditionalMiddleware {
	cm.rules = append(cm.rules, MiddlewareRule{
		PathPrefixes: pathPrefixes,
		Middlewares:  middlewares,
	})
	return cm
}

// Handler returns a gin middleware that conditionally applies middleware based on path
func (cm *ConditionalMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// Find matching rules and apply their middleware
		for _, rule := range cm.rules {
			if cm.pathMatches(path, rule.PathPrefixes) {
				// Create a chain of middleware for this rule
				handlers := append(rule.Middlewares, func(c *gin.Context) {
					c.Next()
				})

				// Execute middleware chain
				for _, handler := range handlers {
					if c.IsAborted() {
						return
					}
					handler(c)
				}
				return
			}
		}

		// No matching rule, continue without additional middleware
		c.Next()
	}
}

// pathMatches checks if path matches any of the prefixes
func (cm *ConditionalMiddleware) pathMatches(path string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// Example usage:
/*
func setupConditionalMiddleware() gin.HandlerFunc {
	cm := NewConditionalMiddleware()

	// Public routes - no auth
	cm.AddRule(
		[]string{"/public", "/health"},
		publicMiddleware(),
	)

	// API v1 - JWT auth
	cm.AddRule(
		[]string{"/api/v1"},
		jwtAuthMiddleware(),
		identityMiddleware(),
	)

	// Internal - API key auth
	cm.AddRule(
		[]string{"/internal"},
		apiKeyAuthMiddleware(),
	)

	// Admin - JWT + admin check
	cm.AddRule(
		[]string{"/admin"},
		jwtAuthMiddleware(),
		identityMiddleware(),
		adminOnlyMiddleware(),
	)

	return cm.Handler()
}

// Then in app.go:
router.Use(setupConditionalMiddleware())
router.Any("/*any", WrapH(a.registerGateway(ctx)))
*/
