package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	mu       sync.RWMutex
	visitors map[string]*visitor
	rate     int           // requests per window
	window   time.Duration // time window
	cleanup  time.Duration // cleanup interval
}

// visitor tracks requests from a single IP
type visitor struct {
	tokens     int
	lastSeen   time.Time
	lastRefill time.Time
}

// RateLimitConfig configures the rate limiter
type RateLimitConfig struct {
	RequestsPerWindow int           // Number of requests allowed per window
	Window            time.Duration // Time window (e.g., 1 minute)
	CleanupInterval   time.Duration // How often to clean up old visitors
}

// NewRateLimiter creates a new rate limiter middleware
func NewRateLimiter(config RateLimitConfig) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     config.RequestsPerWindow,
		window:   config.Window,
		cleanup:  config.CleanupInterval,
	}

	// Start cleanup goroutine
	go rl.cleanupVisitors()

	return rl
}

// RateLimit returns a middleware that enforces rate limits
func (rl *RateLimiter) RateLimit() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get client IP
			ip := getClientIP(r)

			// Check if allowed
			if !rl.allow(ip) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(rl.window.Seconds())))
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error":"RATE_LIMIT_EXCEEDED","message":"Rate limit exceeded. Please try again later."}`))
				return
			}

			// Continue to next handler
			next.ServeHTTP(w, r)
		})
	}
}

// allow checks if a request from the given IP is allowed
func (rl *RateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	now := time.Now()

	if !exists {
		// New visitor
		rl.visitors[ip] = &visitor{
			tokens:     rl.rate - 1,
			lastSeen:   now,
			lastRefill: now,
		}
		return true
	}

	// Update last seen
	v.lastSeen = now

	// Refill tokens if window has passed
	if now.Sub(v.lastRefill) >= rl.window {
		v.tokens = rl.rate
		v.lastRefill = now
	}

	// Check if tokens available
	if v.tokens > 0 {
		v.tokens--
		return true
	}

	return false
}

// cleanupVisitors periodically removes old visitors
func (rl *RateLimiter) cleanupVisitors() {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, v := range rl.visitors {
			// Remove visitors not seen in 2x the window
			if now.Sub(v.lastSeen) > rl.window*2 {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// getClientIP extracts the client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (from reverse proxy)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// Stats returns current rate limiter statistics
func (rl *RateLimiter) Stats() map[string]interface{} {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	return map[string]interface{}{
		"total_visitors":      len(rl.visitors),
		"requests_per_window": rl.rate,
		"window_seconds":      rl.window.Seconds(),
	}
}
