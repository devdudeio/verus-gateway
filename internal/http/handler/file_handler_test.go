package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/devdudeio/verus-gateway/internal/domain"
)

// Test helper functions
func TestSetFileHeaders(t *testing.T) {
	handler := &FileHandler{}

	tests := []struct {
		name        string
		file        *domain.File
		wantHeaders map[string]string
	}{
		{
			name: "sets all headers correctly",
			file: &domain.File{
				TXID:    "abc123",
				Content: []byte("test"),
				Metadata: &domain.FileMetadata{
					Filename:    "test.txt",
					ContentType: "text/plain",
					Size:        4,
				},
			},
			wantHeaders: map[string]string{
				"Content-Type":        "text/plain",
				"Content-Disposition": `inline; filename="test.txt"`,
				"Content-Length":      "4",
				"Cache-Control":       "public, max-age=31536000, immutable",
				"ETag":                `"abc123"`,
			},
		},
		{
			name: "handles missing content type",
			file: &domain.File{
				TXID:    "abc123",
				Content: []byte("test"),
				Metadata: &domain.FileMetadata{
					Filename: "test.bin",
					Size:     4,
				},
			},
			wantHeaders: map[string]string{
				"Content-Type": "application/octet-stream",
			},
		},
		{
			name: "sanitizes filename with quotes",
			file: &domain.File{
				TXID:    "abc123",
				Content: []byte("test"),
				Metadata: &domain.FileMetadata{
					Filename:    `test"file.txt`,
					ContentType: "text/plain",
					Size:        4,
				},
			},
			wantHeaders: map[string]string{
				"Content-Disposition": `inline; filename="test\"file.txt"`,
			},
		},
		{
			name: "handles no filename",
			file: &domain.File{
				TXID:    "abc123",
				Content: []byte("test"),
				Metadata: &domain.FileMetadata{
					ContentType: "text/plain",
					Size:        4,
				},
			},
			wantHeaders: map[string]string{
				"Content-Type": "text/plain",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			handler.setFileHeaders(w, tt.file)

			for key, want := range tt.wantHeaders {
				got := w.Header().Get(key)
				if got != want {
					t.Errorf("header %s = %q, want %q", key, got, want)
				}
			}
		})
	}
}

func TestWriteJSON(t *testing.T) {
	handler := &FileHandler{}

	tests := []struct {
		name       string
		statusCode int
		data       interface{}
		wantStatus int
		checkBody  bool
	}{
		{
			name:       "writes valid JSON",
			statusCode: http.StatusOK,
			data:       map[string]string{"key": "value"},
			wantStatus: http.StatusOK,
			checkBody:  true,
		},
		{
			name:       "writes error JSON",
			statusCode: http.StatusBadRequest,
			data:       map[string]string{"error": "invalid request"},
			wantStatus: http.StatusBadRequest,
			checkBody:  true,
		},
		{
			name:       "writes complex object",
			statusCode: http.StatusOK,
			data: map[string]interface{}{
				"txid":     "abc123",
				"chain":    "vrsctest",
				"size":     1024,
				"filename": "test.txt",
			},
			wantStatus: http.StatusOK,
			checkBody:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			handler.writeJSON(w, tt.statusCode, tt.data)

			if w.Code != tt.wantStatus {
				t.Errorf("status code = %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.checkBody {
				var got map[string]interface{}
				if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
					t.Errorf("failed to decode JSON: %v", err)
				}
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Content-Type = %q, want %q", contentType, "application/json")
			}
		})
	}
}

func TestWriteError(t *testing.T) {
	handler := &FileHandler{}

	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{
			name:       "domain not found error",
			err:        domain.NewNotFoundError("file", "abc123"),
			wantStatus: http.StatusNotFound,
			wantCode:   "NOT_FOUND",
		},
		{
			name:       "generic error",
			err:        errors.New("something went wrong"),
			wantStatus: http.StatusInternalServerError,
			wantCode:   "INTERNAL_ERROR",
		},
		{
			name:       "invalid input error",
			err:        domain.NewInvalidInputError("txid", "txid is required"),
			wantStatus: http.StatusBadRequest,
			wantCode:   "INVALID_INPUT",
		},
		{
			name:       "rate limit error",
			err:        domain.NewRateLimitError(100, "1m"),
			wantStatus: http.StatusTooManyRequests,
			wantCode:   "RATE_LIMIT_EXCEEDED",
		},
		{
			name:       "rpc error",
			err:        domain.NewRPCError("getdata", errors.New("connection refused")),
			wantStatus: http.StatusBadGateway,
			wantCode:   "RPC_ERROR",
		},
		{
			name:       "decryption error",
			err:        domain.NewDecryptionError("abc123", errors.New("invalid key")),
			wantStatus: http.StatusInternalServerError,
			wantCode:   "DECRYPTION_FAILED",
		},
		{
			name:       "chain error",
			err:        domain.NewChainError("vrsc", "chain not configured"),
			wantStatus: http.StatusBadRequest,
			wantCode:   "CHAIN_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/test", nil)

			handler.writeError(w, r, tt.err)

			if w.Code != tt.wantStatus {
				t.Errorf("status code = %d, want %d", w.Code, tt.wantStatus)
			}

			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Errorf("failed to decode error response: %v", err)
			}

			if got, ok := response["error"].(string); !ok || got != tt.wantCode {
				t.Errorf("error code = %q, want %q", got, tt.wantCode)
			}

			if _, ok := response["message"]; !ok {
				t.Error("error response missing message field")
			}

			if _, ok := response["request_id"]; !ok {
				t.Error("error response missing request_id field")
			}

			// Verify Content-Type
			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Content-Type = %q, want %q", contentType, "application/json")
			}
		})
	}
}

func TestNewFileHandler(t *testing.T) {
	handler := NewFileHandler(nil)
	if handler == nil {
		t.Fatal("NewFileHandler returned nil")
	}
	if handler.fileService != nil {
		t.Error("expected fileService to be nil when passed nil")
	}
}
