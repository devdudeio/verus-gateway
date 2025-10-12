package middleware

import (
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

// AuditLogger creates middleware for security audit logging
type AuditLogger struct {
	logger *zerolog.Logger
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(logger *zerolog.Logger) *AuditLogger {
	return &AuditLogger{
		logger: logger,
	}
}

// Log returns middleware that logs security-relevant events
func (a *AuditLogger) Log() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Capture security-relevant information
			auditEvent := a.logger.Info().
				Str("event", "http_request").
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Str("remote_addr", r.RemoteAddr).
				Str("user_agent", r.Header.Get("User-Agent")).
				Str("referer", r.Header.Get("Referer"))

			// Log if authentication is present
			if r.Header.Get("Authorization") != "" {
				auditEvent = auditEvent.Bool("has_auth", true)
			}

			// Log if API key is present
			if r.Header.Get("X-API-Key") != "" {
				auditEvent = auditEvent.Bool("has_api_key", true)
			}

			// Wrap response writer to capture status
			ww := &statusResponseWriter{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(ww, r)

			// Log completion with status and duration
			duration := time.Since(start)
			auditEvent.
				Int("status", ww.status).
				Dur("duration", duration).
				Msg("Request processed")

			// Log security events
			if ww.status == http.StatusUnauthorized {
				a.logger.Warn().
					Str("event", "unauthorized_access").
					Str("path", r.URL.Path).
					Str("remote_addr", r.RemoteAddr).
					Str("user_agent", r.Header.Get("User-Agent")).
					Msg("Unauthorized access attempt")
			}

			if ww.status == http.StatusTooManyRequests {
				a.logger.Warn().
					Str("event", "rate_limit_exceeded").
					Str("remote_addr", r.RemoteAddr).
					Msg("Rate limit exceeded")
			}

			if ww.status >= 500 {
				a.logger.Error().
					Str("event", "server_error").
					Str("path", r.URL.Path).
					Int("status", ww.status).
					Msg("Server error occurred")
			}
		})
	}
}

// statusResponseWriter wraps http.ResponseWriter to capture status code
type statusResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusResponseWriter) Write(b []byte) (int, error) {
	return w.ResponseWriter.Write(b)
}
