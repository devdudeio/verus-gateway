package middleware

import (
	"net/http"
	"strings"
)

// CORSConfig configures CORS behavior
type CORSConfig struct {
	AllowedOrigins   []string // List of allowed origins, or ["*"] for all
	AllowedMethods   []string // Allowed HTTP methods
	AllowedHeaders   []string // Allowed headers
	ExposedHeaders   []string // Headers exposed to client
	AllowCredentials bool     // Whether to allow credentials
	MaxAge           int      // Preflight cache duration in seconds
}

// DefaultCORSConfig returns a secure CORS configuration
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins:   []string{}, // No origins by default (most secure)
		AllowedMethods:   []string{"GET", "HEAD", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"X-Request-ID", "X-Cache-Status"},
		AllowCredentials: false,
		MaxAge:           3600, // 1 hour
	}
}

// CORS creates a CORS middleware with the given configuration
func CORS(config CORSConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			if origin != "" && isOriginAllowed(origin, config.AllowedOrigins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)

				if config.AllowCredentials {
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}

				// Set exposed headers
				if len(config.ExposedHeaders) > 0 {
					w.Header().Set("Access-Control-Expose-Headers", strings.Join(config.ExposedHeaders, ", "))
				}
			}

			// Handle preflight OPTIONS request
			if r.Method == "OPTIONS" {
				// Set allowed methods
				if len(config.AllowedMethods) > 0 {
					w.Header().Set("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))
				}

				// Set allowed headers
				if len(config.AllowedHeaders) > 0 {
					w.Header().Set("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))
				}

				// Set max age
				if config.MaxAge > 0 {
					w.Header().Set("Access-Control-Max-Age", string(rune(config.MaxAge)))
				}

				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isOriginAllowed checks if an origin is in the allowed list
func isOriginAllowed(origin string, allowed []string) bool {
	// If no origins configured, deny all (most secure)
	if len(allowed) == 0 {
		return false
	}

	// Check for wildcard
	for _, o := range allowed {
		if o == "*" {
			return true
		}
		if o == origin {
			return true
		}
		// Support wildcard subdomains (e.g., "*.example.com")
		if strings.HasPrefix(o, "*.") {
			domain := strings.TrimPrefix(o, "*.")
			if strings.HasSuffix(origin, domain) {
				return true
			}
		}
	}

	return false
}
