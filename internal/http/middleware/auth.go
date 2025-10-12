package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// APIKeyAuth creates middleware for API key authentication
type APIKeyAuth struct {
	apiKeys map[string]bool
	header  string
}

// NewAPIKeyAuth creates a new API key auth middleware
func NewAPIKeyAuth(apiKeys []string, headerName string) *APIKeyAuth {
	keyMap := make(map[string]bool)
	for _, key := range apiKeys {
		if key != "" {
			keyMap[key] = true
		}
	}

	if headerName == "" {
		headerName = "X-API-Key"
	}

	return &APIKeyAuth{
		apiKeys: keyMap,
		header:  headerName,
	}
}

// Require returns middleware that requires a valid API key
func (a *APIKeyAuth) Require() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip if no API keys configured (auth disabled)
			if len(a.apiKeys) == 0 {
				next.ServeHTTP(w, r)
				return
			}

			// Get API key from header
			apiKey := r.Header.Get(a.header)
			if apiKey == "" {
				// Also check Authorization header with Bearer token
				auth := r.Header.Get("Authorization")
				if strings.HasPrefix(auth, "Bearer ") {
					apiKey = strings.TrimPrefix(auth, "Bearer ")
				}
			}

			// Validate API key
			if apiKey == "" || !a.validateKey(apiKey) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("WWW-Authenticate", `Bearer realm="Verus Gateway"`)
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"UNAUTHORIZED","message":"Valid API key required"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// validateKey checks if the API key is valid (constant-time comparison)
func (a *APIKeyAuth) validateKey(provided string) bool {
	for validKey := range a.apiKeys {
		// Use constant-time comparison to prevent timing attacks
		if subtle.ConstantTimeCompare([]byte(provided), []byte(validKey)) == 1 {
			return true
		}
	}
	return false
}

// Optional returns middleware that optionally validates API keys
// If no API key is provided, the request continues
// If an API key is provided, it must be valid
func (a *APIKeyAuth) Optional() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip if no API keys configured
			if len(a.apiKeys) == 0 {
				next.ServeHTTP(w, r)
				return
			}

			// Get API key from header
			apiKey := r.Header.Get(a.header)
			if apiKey == "" {
				auth := r.Header.Get("Authorization")
				if strings.HasPrefix(auth, "Bearer ") {
					apiKey = strings.TrimPrefix(auth, "Bearer ")
				}
			}

			// If API key provided, it must be valid
			if apiKey != "" && !a.validateKey(apiKey) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"UNAUTHORIZED","message":"Invalid API key"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
