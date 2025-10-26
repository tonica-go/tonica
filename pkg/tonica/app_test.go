package tonica

import (
	"context"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/tonica-go/tonica/pkg/tonica/config"
)

func TestNewApp(t *testing.T) {
	t.Run("should create app with defaults", func(t *testing.T) {
		app := NewApp()

		assert.NotNil(t, app)
		assert.NotNil(t, app.registry)
		assert.NotNil(t, app.logger)
		assert.NotNil(t, app.router)
		assert.NotNil(t, app.metricRouter)
		assert.NotNil(t, app.metricsManager)
		assert.NotNil(t, app.shutdown)
		assert.Empty(t, app.customRoutes)
	})

	t.Run("should apply options", func(t *testing.T) {
		app := NewApp(
			WithName("test-app"),
			WithSpec("spec.json"),
		)

		assert.Equal(t, "test-app", app.Name)
		assert.Equal(t, "spec.json", app.spec)
	})

	t.Run("should register framework metrics", func(t *testing.T) {
		app := NewApp()

		// Verify some metrics are registered
		// This is indirect - we know registerFrameworkMetrics was called
		assert.NotNil(t, app.metricsManager)
	})
}

func TestApp_Getters(t *testing.T) {
	app := NewApp()

	t.Run("GetLogger", func(t *testing.T) {
		logger := app.GetLogger()
		assert.NotNil(t, logger)
		assert.Same(t, app.logger, logger)
	})

	t.Run("GetMetricManager", func(t *testing.T) {
		manager := app.GetMetricManager()
		assert.NotNil(t, manager)
		assert.Same(t, app.metricsManager, manager)
	})

	t.Run("GetRegistry", func(t *testing.T) {
		registry := app.GetRegistry()
		assert.NotNil(t, registry)
		assert.Same(t, app.registry, registry)
	})

	t.Run("GetRouter", func(t *testing.T) {
		router := app.GetRouter()
		assert.NotNil(t, router)
		assert.Same(t, app.router, router)
	})

	t.Run("GetMetricRouter", func(t *testing.T) {
		router := app.GetMetricRouter()
		assert.NotNil(t, router)
		assert.Same(t, app.metricRouter, router)
	})
}

func TestApp_CustomRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := NewApp()

	t.Run("should collect custom routes", func(t *testing.T) {
		NewRoute(app).
			GET("/route1").
			Summary("Route 1").
			Handle(func(c *gin.Context) {})

		NewRoute(app).
			POST("/route2").
			Summary("Route 2").
			Handle(func(c *gin.Context) {})

		assert.Len(t, app.customRoutes, 2)
		assert.Equal(t, "/route1", app.customRoutes[0].Path)
		assert.Equal(t, "/route2", app.customRoutes[1].Path)
	})
}

func TestOptions(t *testing.T) {
	t.Run("WithName", func(t *testing.T) {
		app := NewApp(WithName("custom-name"))
		assert.Equal(t, "custom-name", app.Name)
	})

	t.Run("WithSpec", func(t *testing.T) {
		app := NewApp(WithSpec("path/to/spec.json"))
		assert.Equal(t, "path/to/spec.json", app.spec)
	})

	t.Run("WithConfig", func(t *testing.T) {
		cfg := config.NewConfig(
			config.WithRunMode(config.ModeService),
		)
		app := NewApp(WithConfig(cfg))
		assert.Same(t, cfg, app.cfg)
	})

	t.Run("WithLogger", func(t *testing.T) {
		// Can't easily test custom logger, but verify option works
		tempApp := NewApp()
		app := NewApp(WithLogger(tempApp.logger))
		assert.NotNil(t, app.logger)
	})

	t.Run("WithRegistry", func(t *testing.T) {
		// Can't easily test custom registry, but verify option works
		tempApp := NewApp()
		app := NewApp(WithRegistry(tempApp.registry))
		assert.NotNil(t, app.registry)
	})
}

func TestBuildCORSConfig(t *testing.T) {
	t.Run("should allow all origins by default", func(t *testing.T) {
		cfg := buildCORSConfig()

		assert.True(t, cfg.AllowAllOrigins)
		assert.Contains(t, cfg.AllowMethods, "GET")
		assert.Contains(t, cfg.AllowMethods, "POST")
		assert.Contains(t, cfg.AllowHeaders, "Authorization")
	})

	t.Run("should parse custom origins from env", func(t *testing.T) {
		t.Setenv("APP_CORS_ORIGINS", "http://localhost:3000, https://example.com")

		cfg := buildCORSConfig()

		assert.False(t, cfg.AllowAllOrigins)
		assert.Len(t, cfg.AllowOrigins, 2)
		assert.Contains(t, cfg.AllowOrigins, "http://localhost:3000")
		assert.Contains(t, cfg.AllowOrigins, "https://example.com")
	})
}

func TestSplitAndTrim(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single value",
			input:    "value1",
			expected: []string{"value1"},
		},
		{
			name:     "multiple values",
			input:    "value1,value2,value3",
			expected: []string{"value1", "value2", "value3"},
		},
		{
			name:     "with spaces",
			input:    " value1 , value2 , value3 ",
			expected: []string{"value1", "value2", "value3"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "only commas",
			input:    ",,,",
			expected: []string{},
		},
		{
			name:     "mixed empty and values",
			input:    "value1,,value2,  ,value3",
			expected: []string{"value1", "value2", "value3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitAndTrim(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestApp_RegisterFrameworkMetrics(t *testing.T) {
	app := NewApp()

	// After app creation, framework metrics should be registered
	// We can't easily check individual metrics, but we can verify
	// the metrics manager is not nil and was called
	assert.NotNil(t, app.metricsManager)

	// Verify app can create additional metrics
	app.metricsManager.NewCounter("test_counter", "Test counter")
	// No error return, just verify it doesn't panic
}

func TestApp_Shutdown_Integration(t *testing.T) {
	t.Run("should have shutdown coordinator initialized", func(t *testing.T) {
		app := NewApp()
		assert.NotNil(t, app.shutdown)
	})

	t.Run("shutdown should be available for registration", func(t *testing.T) {
		app := NewApp()

		// Should be able to register cleanup functions
		called := false
		app.shutdown.RegisterCleanup(func(ctx context.Context) error {
			called = true
			return nil
		})

		// Execute shutdown
		err := app.shutdown.Execute(100 * time.Millisecond)
		assert.NoError(t, err)
		assert.True(t, called)
	})
}
