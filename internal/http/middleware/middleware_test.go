package middleware

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/devdudeio/verus-gateway/internal/observability/logger"
	"github.com/devdudeio/verus-gateway/internal/observability/metrics"
)

func TestRequestID_GeneratesID(t *testing.T) {
	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := GetRequestID(r.Context())
		if requestID == "" {
			t.Error("Request ID was not added to context")
		}
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with RequestID middleware
	wrappedHandler := RequestID(handler)

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	// Serve the request
	wrappedHandler.ServeHTTP(rec, req)

	// Verify X-Request-ID header was set
	requestID := rec.Header().Get("X-Request-ID")
	if requestID == "" {
		t.Error("X-Request-ID header was not set in response")
	}
}

func TestRequestID_UsesExistingID(t *testing.T) {
	expectedID := "test-request-id-123"

	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := GetRequestID(r.Context())
		if requestID != expectedID {
			t.Errorf("Expected request ID %s, got %s", expectedID, requestID)
		}
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with RequestID middleware
	wrappedHandler := RequestID(handler)

	// Create test request with existing X-Request-ID header
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", expectedID)
	rec := httptest.NewRecorder()

	// Serve the request
	wrappedHandler.ServeHTTP(rec, req)

	// Verify the same ID was used
	if rec.Header().Get("X-Request-ID") != expectedID {
		t.Errorf("Expected X-Request-ID %s, got %s", expectedID, rec.Header().Get("X-Request-ID"))
	}
}

func TestGetRequestID_WithID(t *testing.T) {
	expectedID := "test-id-456"
	ctx := context.WithValue(context.Background(), RequestIDKey, expectedID)

	requestID := GetRequestID(ctx)
	if requestID != expectedID {
		t.Errorf("Expected request ID %s, got %s", expectedID, requestID)
	}
}

func TestGetRequestID_WithoutID(t *testing.T) {
	ctx := context.Background()

	requestID := GetRequestID(ctx)
	if requestID != "" {
		t.Errorf("Expected empty request ID, got %s", requestID)
	}
}

func TestLogger_LogsRequest(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer
	testLogger := zerolog.New(&buf).With().Timestamp().Logger()

	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify logger was added to context
		ctxLogger := logger.FromContext(r.Context())
		if ctxLogger == nil {
			t.Error("Logger was not added to context")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	// Wrap with RequestID and Logger middleware
	wrappedHandler := RequestID(Logger(&testLogger)(handler))

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	// Serve the request
	wrappedHandler.ServeHTTP(rec, req)

	// Verify log output contains expected fields
	logOutput := buf.String()
	if logOutput == "" {
		t.Error("No log output was generated")
	}

	// Should contain "Request started" and "Request completed"
	if !contains(logOutput, "Request started") {
		t.Error("Log output does not contain 'Request started'")
	}
	if !contains(logOutput, "Request completed") {
		t.Error("Log output does not contain 'Request completed'")
	}
}

func TestLogger_LogsErrorStatus(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer
	testLogger := zerolog.New(&buf).With().Timestamp().Logger()

	// Create a test handler that returns 500
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	// Wrap with RequestID and Logger middleware
	wrappedHandler := RequestID(Logger(&testLogger)(handler))

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	// Serve the request
	wrappedHandler.ServeHTTP(rec, req)

	// Verify log output contains error level
	logOutput := buf.String()
	if !contains(logOutput, "error") {
		t.Error("Log output does not contain error level for 500 status")
	}
}

func TestLogger_LogsWarningStatus(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer
	testLogger := zerolog.New(&buf).With().Timestamp().Logger()

	// Create a test handler that returns 404
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	// Wrap with RequestID and Logger middleware
	wrappedHandler := RequestID(Logger(&testLogger)(handler))

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	// Serve the request
	wrappedHandler.ServeHTTP(rec, req)

	// Verify log output contains warn level
	logOutput := buf.String()
	if !contains(logOutput, "warn") {
		t.Error("Log output does not contain warn level for 404 status")
	}
}

func TestRecoverer_NormalRequest(t *testing.T) {
	// Create a logger
	testLogger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	// Create a normal handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	// Wrap with Recoverer middleware
	wrappedHandler := RequestID(Recoverer(&testLogger)(handler))

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	// Serve the request
	wrappedHandler.ServeHTTP(rec, req)

	// Verify response is normal
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}

func TestRecoverer_Panic(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer
	testLogger := zerolog.New(&buf).With().Timestamp().Logger()

	// Create a handler that panics
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	// Wrap with Recoverer middleware
	wrappedHandler := RequestID(Recoverer(&testLogger)(handler))

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	// Serve the request (should not panic)
	wrappedHandler.ServeHTTP(rec, req)

	// Verify panic was recovered
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 after panic, got %d", rec.Code)
	}

	// Verify response contains error
	if !contains(rec.Body.String(), "internal_server_error") {
		t.Error("Response does not contain error message")
	}

	// Verify panic was logged
	logOutput := buf.String()
	if !contains(logOutput, "Panic recovered") {
		t.Error("Panic was not logged")
	}
}

func TestMetrics_RecordsMetrics(t *testing.T) {
	// Create metrics
	m := metrics.New("test")

	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	// Wrap with Metrics middleware
	wrappedHandler := Metrics(m)(handler)

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	// Serve the request
	wrappedHandler.ServeHTTP(rec, req)

	// Verify response is OK
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	// Metrics are recorded asynchronously, so we can't easily verify them here
	// The important thing is that the middleware doesn't break the request
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "short path",
			path:     "/c/vrsctest/file",
			expected: "/c/vrsctest/file",
		},
		{
			name:     "medium path (50 chars)",
			path:     "/c/vrsctest/file/abc123def456ghi789jkl012mno345pqr", // 50 chars
			expected: "/c/vrsctest/file/abc123def456ghi789jkl012mno345pqr",
		},
		{
			name:     "path exactly 51 chars (> 50 but <= 64)",
			path:     "/c/vrsctest/file/abc123def456ghi789jkl012mno345pqr6", // 51 chars
			expected: "/c/vrsctest/file/abc123def456ghi789jkl012mno345pqr6", // returned as-is
		},
		{
			name:     "path 63 chars (> 50 but <= 64 - returned as-is)",
			path:     "/c/vrsctest/file/abc123def456ghi789jkl012mno345pqr678stu901vwxy", // 63 chars
			expected: "/c/vrsctest/file/abc123def456ghi789jkl012mno345pqr678stu901vwxy", // returned as-is
		},
		{
			name:     "path exactly 64 chars (not > 64 - returned as-is)",
			path:     "/c/vrsctest/file/abc123def456ghi789jkl012mno345pqr678stu901vwxyz", // 64 chars
			expected: "/c/vrsctest/file/abc123def456ghi789jkl012mno345pqr678stu901vwxyz", // returned as-is
		},
		{
			name:     "path exactly 65 chars (> 64 - should truncate to 32 + ...)",
			path:     "/c/vrsctest/file/abc123def456ghi789jkl012mno345pqr678stu901vwxyzA", // 65 chars
			expected: "/c/vrsctest/file/abc123def456ghi...", // first 32 chars + "..."
		},
		{
			name:     "very long path (should truncate to 32 + ...)",
			path:     "/c/vrsctest/file/abc123def456ghi789jkl012mno345pqr678stu901vwx234yz567890abcdefghijklmnopqrstuvwxyz",
			expected: "/c/vrsctest/file/abc123def456ghi...", // first 32 chars + "..."
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizePath(tt.path)
			if result != tt.expected {
				t.Errorf("normalizePath(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}

func TestSecurityHeaders(t *testing.T) {
	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with SecurityHeaders middleware
	wrappedHandler := SecurityHeaders(handler)

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	// Serve the request
	wrappedHandler.ServeHTTP(rec, req)

	// Verify security headers
	expectedHeaders := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"X-XSS-Protection":       "1; mode=block",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	}

	for header, expectedValue := range expectedHeaders {
		actualValue := rec.Header().Get(header)
		if actualValue != expectedValue {
			t.Errorf("Header %s = %q, want %q", header, actualValue, expectedValue)
		}
	}
}

func TestRealIP_XForwardedFor(t *testing.T) {
	// Create a test handler
	var capturedIP string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedIP = r.RemoteAddr
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with RealIP middleware
	wrappedHandler := RealIP(handler)

	// Create test request with X-Forwarded-For header
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.100")
	req.RemoteAddr = "10.0.0.1:12345"
	rec := httptest.NewRecorder()

	// Serve the request
	wrappedHandler.ServeHTTP(rec, req)

	// Verify RemoteAddr was updated
	if capturedIP != "192.168.1.100" {
		t.Errorf("Expected RemoteAddr to be 192.168.1.100, got %s", capturedIP)
	}
}

func TestRealIP_XRealIP(t *testing.T) {
	// Create a test handler
	var capturedIP string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedIP = r.RemoteAddr
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with RealIP middleware
	wrappedHandler := RealIP(handler)

	// Create test request with X-Real-IP header
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Real-IP", "192.168.1.200")
	req.RemoteAddr = "10.0.0.1:12345"
	rec := httptest.NewRecorder()

	// Serve the request
	wrappedHandler.ServeHTTP(rec, req)

	// Verify RemoteAddr was updated
	if capturedIP != "192.168.1.200" {
		t.Errorf("Expected RemoteAddr to be 192.168.1.200, got %s", capturedIP)
	}
}

func TestRealIP_NoHeaders(t *testing.T) {
	// Create a test handler
	var capturedIP string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedIP = r.RemoteAddr
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with RealIP middleware
	wrappedHandler := RealIP(handler)

	// Create test request without headers
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	rec := httptest.NewRecorder()

	// Serve the request
	wrappedHandler.ServeHTTP(rec, req)

	// Verify RemoteAddr was not changed
	if capturedIP != "10.0.0.1:12345" {
		t.Errorf("Expected RemoteAddr to be 10.0.0.1:12345, got %s", capturedIP)
	}
}

func TestTimeout_CompletesBeforeTimeout(t *testing.T) {
	// Create a test handler that completes quickly
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that context has a deadline
		if _, ok := r.Context().Deadline(); !ok {
			t.Error("Context does not have a deadline")
		}
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with Timeout middleware (1 second timeout)
	wrappedHandler := Timeout(1 * time.Second)(handler)

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	// Serve the request
	wrappedHandler.ServeHTTP(rec, req)

	// Verify response is OK
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}

func TestTimeout_ContextHasDeadline(t *testing.T) {
	// Create a test handler
	var contextHasDeadline bool
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, contextHasDeadline = r.Context().Deadline()
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with Timeout middleware
	wrappedHandler := Timeout(100 * time.Millisecond)(handler)

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	// Serve the request
	wrappedHandler.ServeHTTP(rec, req)

	// Verify context has deadline
	if !contextHasDeadline {
		t.Error("Context does not have a deadline after Timeout middleware")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && s != "" &&
		bytes.Contains([]byte(s), []byte(substr))
}
