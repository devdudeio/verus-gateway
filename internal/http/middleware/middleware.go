package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/devdudeio/verus-gateway/internal/observability/logger"
	"github.com/devdudeio/verus-gateway/internal/observability/metrics"
)

// RequestIDKey is the context key for request IDs
type contextKey string

const RequestIDKey contextKey = "request_id"

// RequestID middleware adds a unique request ID to each request
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Add to response header
		w.Header().Set("X-Request-ID", requestID)

		// Add to context
		ctx := context.WithValue(r.Context(), RequestIDKey, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID retrieves the request ID from context
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		return requestID
	}
	return ""
}

// Logger middleware logs HTTP requests using zerolog
func Logger(baseLogger *zerolog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status code
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			// Get request ID from context
			requestID := GetRequestID(r.Context())

			// Create request-scoped logger
			reqLogger := baseLogger.With().
				Str("request_id", requestID).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Str("remote_addr", r.RemoteAddr).
				Logger()

			// Add logger to context
			ctx := logger.WithContext(r.Context(), &reqLogger)
			r = r.WithContext(ctx)

			// Log request started
			reqLogger.Info().Msg("Request started")

			defer func() {
				duration := time.Since(start)
				status := ww.Status()

				// Log request completed
				logEvent := reqLogger.Info()
				if status >= 500 {
					logEvent = reqLogger.Error()
				} else if status >= 400 {
					logEvent = reqLogger.Warn()
				}

				logEvent.
					Int("status", status).
					Dur("duration", duration).
					Int("bytes", ww.BytesWritten()).
					Msg("Request completed")
			}()

			next.ServeHTTP(ww, r)
		})
	}
}

// Recoverer middleware recovers from panics using zerolog
func Recoverer(baseLogger *zerolog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rvr := recover(); rvr != nil {
					requestID := GetRequestID(r.Context())

					// Log panic
					baseLogger.Error().
						Str("request_id", requestID).
						Str("method", r.Method).
						Str("path", r.URL.Path).
						Interface("panic", rvr).
						Msg("Panic recovered")

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					fmt.Fprintf(w, `{"error":"internal_server_error","message":"An internal error occurred","request_id":"%s"}`, requestID)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// Metrics middleware records Prometheus metrics
func Metrics(m *metrics.Metrics) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status code and bytes
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			// Get request size
			requestSize := r.ContentLength

			next.ServeHTTP(ww, r)

			// Calculate duration
			duration := time.Since(start).Seconds()

			// Get status and response size
			status := ww.Status()
			responseSize := int64(ww.BytesWritten())

			// Normalize path (remove IDs and hashes for better cardinality)
			path := normalizePath(r.URL.Path)

			// Record metrics
			m.RecordHTTPRequest(
				r.Method,
				path,
				fmt.Sprintf("%d", status),
				duration,
				requestSize,
				responseSize,
			)
		})
	}
}

// normalizePath normalizes URL paths to reduce metric cardinality
func normalizePath(path string) string {
	// Replace UUIDs, TXIDs, and other variable parts
	// This is a simple implementation - could be improved with regex
	switch {
	case len(path) > 50: // Likely contains a TXID or hash
		if len(path) > 64 {
			return path[:32] + "..."
		}
		return path
	default:
		return path
	}
}

// SecurityHeaders adds security-related HTTP headers
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content Security Policy
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")

		// Permissions Policy (formerly Feature-Policy)
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		// Strict Transport Security (HSTS) - only if HTTPS
		if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}

		next.ServeHTTP(w, r)
	})
}

// RealIP middleware extracts the real client IP from headers
func RealIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check X-Forwarded-For header
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			// Take the first IP in the list
			r.RemoteAddr = xff
		} else if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
			r.RemoteAddr = xrip
		}

		next.ServeHTTP(w, r)
	})
}

// Timeout middleware adds a timeout to requests
func Timeout(timeout time.Duration) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}
