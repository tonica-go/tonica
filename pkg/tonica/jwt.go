package tonica

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/tonica-go/tonica/pkg/tonica/config"
)

func newIdentityMapFromClaims(claims jwt.Claims) map[string]any {
	mapClaims := claims.(jwt.MapClaims)
	return mapClaims
}

func UseBetterAuthMiddleware(ctx *gin.Context) {
	if ctx.Request.RequestURI == "/openapi.json" {
		ctx.Next()
		return
	}
	if ctx.Request.RequestURI == "/metrics" {
		ctx.Next()
		return
	}
	if ctx.Request.RequestURI == "/docs" {
		ctx.Next()
		return
	}
	if ctx.Request.Method == "OPTIONS" {
		ctx.Next()
		return
	}
	auth := ctx.Request.Header.Get("Authorization")
	if auth == "" {
		ctx.AbortWithStatus(401)
	}
	auth = strings.TrimPrefix(auth, "Bearer ")
	token, err := jwt.Parse(auth, func(token *jwt.Token) (interface{}, error) {
		// Verify algorithm
		if token.Method.Alg() != "EdDSA" {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Get kid from header
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, errors.New("missing kid in token header")
		}

		// Get public key for this kid
		publicKey, err := getPublicKey(ctx, kid)
		if err != nil {
			return nil, err
		}

		return publicKey, nil
	})
	if err != nil {
		ctx.AbortWithStatus(401)
		return
	}
	identityVal := newIdentityMapFromClaims(token.Claims)
	ctx.Set("identity", identityVal)
	ctx.Next()
}

// getPublicKey retrieves the public key for the given kid.
func getPublicKey(ctx context.Context, kid string) (ed25519.PublicKey, error) {
	// Check cache
	//if key, ok := s.jwksCache.keys[kid]; ok && time.Now().Before(s.jwksCache.expiresAt) {
	//	return key, nil
	//}

	// Fetch JWKS
	keys, err := refreshJWKS(ctx)
	if err != nil {
		return nil, err
	}

	// Try again from cache
	key, ok := keys[kid]
	if !ok {
		return nil, fmt.Errorf("key %s not found in JWKS", kid)
	}

	return key, nil
}

type JWKS struct {
	Keys []JWK `json:"keys"`
}

// JWK represents a single JSON Web Key.
type JWK struct {
	Kty string `json:"kty"` // Key type
	Use string `json:"use"` // Public key use
	Kid string `json:"kid"` // Key ID
	Alg string `json:"alg"` // Algorithm
	Crv string `json:"crv"` // Curve (for Ed25519: "Ed25519")
	X   string `json:"x"`   // Public key (base64url encoded)
}

func refreshJWKS(ctx context.Context) (map[string]ed25519.PublicKey, error) {
	betterAuthUrl := config.GetEnv("BETTER_AUTH_URL", "http://localhost:3000")
	url := fmt.Sprintf("%s%s", betterAuthUrl, "/api/auth/jwks")
	newKeys := make(map[string]ed25519.PublicKey)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return newKeys, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return newKeys, fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return newKeys, fmt.Errorf("JWKS endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var jwks JWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return newKeys, fmt.Errorf("failed to decode JWKS: %w", err)
	}

	// Parse keys

	for _, key := range jwks.Keys {
		if key.Kty != "OKP" || key.Crv != "Ed25519" {
			continue // Skip non-Ed25519 keys
		}

		// Decode base64url encoded public key
		pubKeyBytes, err := base64.RawURLEncoding.DecodeString(key.X)
		if err != nil {
			return newKeys, fmt.Errorf("failed to decode public key for kid %s: %w", key.Kid, err)
		}

		if len(pubKeyBytes) != ed25519.PublicKeySize {
			return newKeys, fmt.Errorf("invalid public key size for kid %s", key.Kid)
		}

		newKeys[key.Kid] = pubKeyBytes
	}

	// Update cache
	//s.jwksCache.keys = newKeys
	//s.jwksCache.expiresAt = time.Now().Add(24 * time.Hour) // Cache for 24 hours

	return newKeys, nil
}
