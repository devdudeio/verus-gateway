package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewAdminHandler(t *testing.T) {
	handler := NewAdminHandler(nil, nil, nil, "v1.0.0")
	if handler == nil {
		t.Fatal("NewAdminHandler returned nil")
	}
	if handler.version != "v1.0.0" {
		t.Errorf("expected version v1.0.0, got %s", handler.version)
	}
}

func TestHealth(t *testing.T) {
	handler := NewAdminHandler(nil, nil, nil, "v1.2.3")

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/health", nil)

	handler.Health(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if status, ok := response["status"].(string); !ok || status != "healthy" {
		t.Errorf("expected status 'healthy', got %v", response["status"])
	}

	if version, ok := response["version"].(string); !ok || version != "v1.2.3" {
		t.Errorf("expected version 'v1.2.3', got %v", response["version"])
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}
}

func TestHealthResponseFormat(t *testing.T) {
	tests := []struct {
		name    string
		version string
	}{
		{"with semver", "v1.2.3"},
		{"with dev version", "dev"},
		{"with git hash", "abc123"},
		{"empty version", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewAdminHandler(nil, nil, nil, tt.version)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/health", nil)

			handler.Health(w, r)

			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if response["version"] != tt.version {
				t.Errorf("expected version %q, got %v", tt.version, response["version"])
			}

			// Verify required fields exist
			if _, ok := response["status"]; !ok {
				t.Error("response missing 'status' field")
			}
		})
	}
}
