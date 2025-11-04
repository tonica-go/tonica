package identity

import (
	"github.com/gin-gonic/gin"
)

// IdentityExtractor is a function that extracts identity from gin.Context
// Implement this based on your authentication mechanism (JWT, session, etc.)
type IdentityExtractor func(*gin.Context) Identity

// Middleware creates a Gin middleware that extracts and stores identity in context
// The extractor function is responsible for extracting identity from the request
// (e.g., from JWT token, session, or headers)
func Middleware(extractor IdentityExtractor) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract identity using the provided extractor
		identity := extractor(c)

		// If identity was extracted, add it to the request context
		if identity != nil {
			ctx := ToContext(c.Request.Context(), identity)
			c.Request = c.Request.WithContext(ctx)
		}

		c.Next()
	}
}

// DefaultExtractor is a basic identity extractor that looks for "identity" in gin.Context
// This is useful when you already have identity stored in gin.Context by your auth middleware
func DefaultExtractor(c *gin.Context) Identity {
	if identity, exists := c.Get(IdentityContextKey); exists {
		if identityMap, ok := identity.(map[string]interface{}); ok {
			return Identity(identityMap)
		}
		if identityMap, ok := identity.(Identity); ok {
			return identityMap
		}
	}
	return nil
}

// HeaderExtractor creates an extractor that reads identity from request headers
// This is useful for simple authentication where user info is passed in headers
func HeaderExtractor(userIDHeader, emailHeader, roleHeader string) IdentityExtractor {
	return func(c *gin.Context) Identity {
		userID := c.GetHeader(userIDHeader)
		if userID == "" {
			return nil
		}

		identity := NewIdentity(userID)

		if email := c.GetHeader(emailHeader); email != "" {
			identity = identity.WithEmail(email)
		}

		if role := c.GetHeader(roleHeader); role != "" {
			identity = identity.WithRole(role)
		}

		return identity
	}
}

// JWTExtractor creates an extractor that reads identity from JWT claims
// The claimsGetter should extract claims from the JWT token stored in gin.Context
func JWTExtractor(claimsKey string, userIDField, emailField, roleField string) IdentityExtractor {
	return func(c *gin.Context) Identity {
		claims, exists := c.Get(claimsKey)
		if !exists {
			return nil
		}

		claimsMap, ok := claims.(map[string]interface{})
		if !ok {
			return nil
		}

		userID, ok := claimsMap[userIDField].(string)
		if !ok || userID == "" {
			return nil
		}

		identity := NewIdentity(userID)

		if email, ok := claimsMap[emailField].(string); ok && email != "" {
			identity = identity.WithEmail(email)
		}

		if role, ok := claimsMap[roleField].(string); ok && role != "" {
			identity = identity.WithRole(role)
		}

		// Add all claims as additional fields
		for k, v := range claimsMap {
			if k != userIDField && k != emailField && k != roleField {
				identity[k] = v
			}
		}

		return identity
	}
}

// ChainExtractors chains multiple extractors, returning the first non-nil identity
// This is useful when you have multiple authentication methods
func ChainExtractors(extractors ...IdentityExtractor) IdentityExtractor {
	return func(c *gin.Context) Identity {
		for _, extractor := range extractors {
			if identity := extractor(c); identity != nil {
				return identity
			}
		}
		return nil
	}
}
