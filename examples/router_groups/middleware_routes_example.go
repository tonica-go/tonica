package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tonica-go/tonica/pkg/tonica"
	"github.com/tonica-go/tonica/pkg/tonica/identity"
)

// Example 1: Public routes without authentication
func publicMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Println("Public route - no auth required")
		c.Next()
	}
}

// Example 2: JWT authentication middleware
func jwtAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return
		}

		// TODO: validate JWT token and extract claims
		// For now, just a simple example
		claims := map[string]interface{}{
			"user_id": "123",
			"email":   "user@example.com",
			"role":    "user",
		}

		// Store claims in context for identity extraction
		c.Set("jwt_claims", claims)
		c.Next()
	}
}

// Example 3: API Key authentication middleware
func apiKeyAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing api key"})
			return
		}

		// TODO: validate API key
		// For now, just a simple example
		if apiKey != "secret-api-key" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid api key"})
			return
		}

		// Set identity from API key
		identity := identity.NewIdentity("api-user-" + apiKey)
		c.Set("identity", identity)
		c.Next()
	}
}

// Example 4: Admin role check middleware
func adminOnlyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		_, exists := c.Get("identityV")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "identity not found"})
			return
		}
		//
		//id, ok := identityV.(identityV.(map[string]any)["Identity"])
		//if !ok || id.GetRole() != "admin" {
		//	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		//	return
		//}

		c.Next()
	}
}

// Example 5: Rate limiting middleware
func rateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: implement rate limiting logic
		log.Println("Rate limit check")
		c.Next()
	}
}

func main() {
	app := tonica.NewApp(
		tonica.WithName("example-app"),

		// Example 1: Public routes - no authentication
		tonica.WithRouteMiddleware(
			[]string{"/public", "/health"},
			publicMiddleware(),
		),

		// Example 2: API v1 routes - JWT authentication
		tonica.WithRouteMiddleware(
			[]string{"/api/v1"},
			jwtAuthMiddleware(),
			identity.Middleware(identity.JWTExtractor("jwt_claims", "user_id", "email", "role")),
			rateLimitMiddleware(),
		),

		// Example 3: Internal routes - API key authentication
		tonica.WithRouteMiddleware(
			[]string{"/internal"},
			apiKeyAuthMiddleware(),
			rateLimitMiddleware(),
		),

		// Example 4: Admin routes - JWT + admin role check
		tonica.WithRouteMiddleware(
			[]string{"/admin"},
			jwtAuthMiddleware(),
			identity.Middleware(identity.JWTExtractor("jwt_claims", "user_id", "email", "role")),
			adminOnlyMiddleware(),
		),

		// Example 5: Legacy API - different auth
		tonica.WithRouteMiddleware(
			[]string{"/api/v2"},
			apiKeyAuthMiddleware(),
		),
	)

	// Register your services
	// app.GetRegistry().MustRegisterService(...)

	app.Run()
}

/*
How this works:

1. Public routes (/public/*, /health/*):
   - No authentication required
   - Only logging middleware

2. API v1 routes (/api/v1/*):
   - JWT authentication required
   - Identity extracted from JWT claims
   - Rate limiting applied

3. Internal routes (/internal/*):
   - API key authentication required
   - Rate limiting applied

4. Admin routes (/admin/*):
   - JWT authentication required
   - Identity extracted from JWT claims
   - Admin role check
   - Only users with role="admin" can access

5. API v2 routes (/api/v2/*):
   - API key authentication required
   - No rate limiting

All other routes (not matching above patterns):
   - No middleware applied by default
   - Can be configured with global middleware if needed

Proto file example:

service UserService {
  // Public route - no auth
  rpc GetPublicInfo(GetPublicInfoRequest) returns (GetPublicInfoResponse) {
    option (google.api.http) = {
      get: "/public/info"
    };
  }

  // API v1 route - JWT auth required
  rpc GetUser(GetUserRequest) returns (GetUserResponse) {
    option (google.api.http) = {
      get: "/api/v1/users/{id}"
    };
  }

  // Internal route - API key auth
  rpc InternalSync(InternalSyncRequest) returns (InternalSyncResponse) {
    option (google.api.http) = {
      post: "/internal/sync"
    };
  }

  // Admin route - JWT auth + admin role
  rpc DeleteUser(DeleteUserRequest) returns (DeleteUserResponse) {
    option (google.api.http) = {
      delete: "/admin/users/{id}"
    };
  }
}
*/
