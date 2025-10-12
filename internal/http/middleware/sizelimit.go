package middleware

import (
	"net/http"
)

// MaxBodySize limits the request body size
func MaxBodySize(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Limit the request body size
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

			next.ServeHTTP(w, r)
		})
	}
}

// MaxURILength limits the URI length
func MaxURILength(maxLength int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check URI length
			if len(r.RequestURI) > maxLength {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusRequestURITooLong)
				w.Write([]byte(`{"error":"URI_TOO_LONG","message":"Request URI too long"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
