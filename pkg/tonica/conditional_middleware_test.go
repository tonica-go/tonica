package tonica

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestConditionalMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("no matching rule - passes through", func(t *testing.T) {
		cm := NewConditionalMiddleware()
		router := gin.New()
		router.Use(cm.Handler())
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "ok", w.Body.String())
	})

	t.Run("matching rule - applies middleware", func(t *testing.T) {
		cm := NewConditionalMiddleware()

		// Add a rule that sets a header
		cm.AddRule([]string{"/api"}, func(c *gin.Context) {
			c.Header("X-Test", "applied")
			c.Next()
		})

		router := gin.New()
		router.Use(cm.Handler())
		router.GET("/api/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "applied", w.Header().Get("X-Test"))
	})

	t.Run("matching rule - middleware can abort", func(t *testing.T) {
		cm := NewConditionalMiddleware()

		// Add a rule that aborts the request
		cm.AddRule([]string{"/protected"}, func(c *gin.Context) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		})

		router := gin.New()
		router.Use(cm.Handler())
		router.GET("/protected/resource", func(c *gin.Context) {
			c.String(http.StatusOK, "should not reach here")
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/protected/resource", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "unauthorized")
	})

	t.Run("multiple rules - first match wins", func(t *testing.T) {
		cm := NewConditionalMiddleware()

		// Add two rules with overlapping paths
		cm.AddRule([]string{"/api"}, func(c *gin.Context) {
			c.Header("X-Test", "api")
			c.Next()
		})

		cm.AddRule([]string{"/api/v1"}, func(c *gin.Context) {
			c.Header("X-Test", "api-v1")
			c.Next()
		})

		router := gin.New()
		router.Use(cm.Handler())
		router.GET("/api/v1/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		// First rule matches /api, so it should be applied
		assert.Equal(t, "api", w.Header().Get("X-Test"))
	})

	t.Run("multiple middleware in one rule", func(t *testing.T) {
		cm := NewConditionalMiddleware()

		// Add a rule with multiple middleware
		cm.AddRule(
			[]string{"/api"},
			func(c *gin.Context) {
				c.Header("X-Test-1", "first")
				c.Next()
			},
			func(c *gin.Context) {
				c.Header("X-Test-2", "second")
				c.Next()
			},
		)

		router := gin.New()
		router.Use(cm.Handler())
		router.GET("/api/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "first", w.Header().Get("X-Test-1"))
		assert.Equal(t, "second", w.Header().Get("X-Test-2"))
	})

	t.Run("multiple path prefixes in one rule", func(t *testing.T) {
		cm := NewConditionalMiddleware()

		// Add a rule with multiple path prefixes
		cm.AddRule(
			[]string{"/public", "/health"},
			func(c *gin.Context) {
				c.Header("X-Public", "true")
				c.Next()
			},
		)

		router := gin.New()
		router.Use(cm.Handler())
		router.GET("/public/info", func(c *gin.Context) {
			c.String(http.StatusOK, "public")
		})
		router.GET("/health/check", func(c *gin.Context) {
			c.String(http.StatusOK, "health")
		})

		// Test /public
		w1 := httptest.NewRecorder()
		req1 := httptest.NewRequest("GET", "/public/info", nil)
		router.ServeHTTP(w1, req1)
		assert.Equal(t, http.StatusOK, w1.Code)
		assert.Equal(t, "true", w1.Header().Get("X-Public"))

		// Test /health
		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest("GET", "/health/check", nil)
		router.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusOK, w2.Code)
		assert.Equal(t, "true", w2.Header().Get("X-Public"))
	})
}

func TestPathMatches(t *testing.T) {
	cm := NewConditionalMiddleware()

	tests := []struct {
		name     string
		path     string
		prefixes []string
		expected bool
	}{
		{
			name:     "exact match",
			path:     "/api",
			prefixes: []string{"/api"},
			expected: true,
		},
		{
			name:     "prefix match",
			path:     "/api/v1/users",
			prefixes: []string{"/api"},
			expected: true,
		},
		{
			name:     "no match",
			path:     "/public/info",
			prefixes: []string{"/api"},
			expected: false,
		},
		{
			name:     "multiple prefixes - first matches",
			path:     "/api/test",
			prefixes: []string{"/api", "/public"},
			expected: true,
		},
		{
			name:     "multiple prefixes - second matches",
			path:     "/public/test",
			prefixes: []string{"/api", "/public"},
			expected: true,
		},
		{
			name:     "multiple prefixes - none match",
			path:     "/admin/test",
			prefixes: []string{"/api", "/public"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cm.pathMatches(tt.path, tt.prefixes)
			assert.Equal(t, tt.expected, result)
		})
	}
}
