package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/devdudeio/verus-gateway/internal/domain"
	"github.com/go-chi/chi/v5"
)

// Mock FileService for testing
type mockFileService struct {
	getFileFunc     func(ctx context.Context, req *domain.FileRequest) (*domain.File, error)
	getMetadataFunc func(ctx context.Context, req *domain.FileRequest) (*domain.FileMetadata, error)
}

func (m *mockFileService) GetFile(ctx context.Context, req *domain.FileRequest) (*domain.File, error) {
	if m.getFileFunc != nil {
		return m.getFileFunc(ctx, req)
	}
	return nil, errors.New("not implemented")
}

func (m *mockFileService) GetMetadata(ctx context.Context, req *domain.FileRequest) (*domain.FileMetadata, error) {
	if m.getMetadataFunc != nil {
		return m.getMetadataFunc(ctx, req)
	}
	return nil, errors.New("not implemented")
}

// newTestHandler creates a FileHandler with a mock service for testing
func newTestHandler(mockService *mockFileService) *FileHandler {
	return &FileHandler{
		fileService: mockService,
	}
}

func TestGetFile_TXID(t *testing.T) {
	tests := []struct {
		name       string
		txid       string
		evk        string
		mockFile   *domain.File
		mockError  error
		wantStatus int
		wantBody   string
	}{
		{
			name: "successful file retrieval by TXID",
			txid: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			mockFile: &domain.File{
				TXID:    "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
				ChainID: "vrsctest",
				Content: []byte("test file content"),
				Metadata: &domain.FileMetadata{
					Filename:    "test.txt",
					ContentType: "text/plain",
					Size:        17,
				},
			},
			wantStatus: http.StatusOK,
			wantBody:   "test file content",
		},
		{
			name:       "file not found",
			txid:       "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			mockError:  domain.NewNotFoundError("file", "0123456789abcdef..."),
			wantStatus: http.StatusNotFound,
		},
		{
			name: "encrypted file with EVK",
			txid: "abc123def456abc123def456abc123def456abc123def456abc123def456abc1",
			evk:  "zxviews1qtest123456789012345678901234567890123456789012345678901234567890123456789012345678",
			mockFile: &domain.File{
				TXID:    "abc123def456...",
				ChainID: "vrsctest",
				Content: []byte("decrypted content"),
				Metadata: &domain.FileMetadata{
					Filename:    "secret.txt",
					ContentType: "text/plain",
					Size:        17,
					Encrypted:   true,
				},
			},
			wantStatus: http.StatusOK,
			wantBody:   "decrypted content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock service
			mockService := &mockFileService{
				getFileFunc: func(ctx context.Context, req *domain.FileRequest) (*domain.File, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return tt.mockFile, nil
				},
			}

			handler := newTestHandler(mockService)

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/c/vrsctest/file/"+tt.txid, nil)
			if tt.evk != "" {
				q := req.URL.Query()
				q.Set("evk", tt.evk)
				req.URL.RawQuery = q.Encode()
			}

			// Add route params
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("chain", "vrsctest")
			rctx.URLParams.Add("txid", tt.txid)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			// Execute request
			w := httptest.NewRecorder()
			handler.GetFile(w, req)

			// Verify status code
			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			// Verify body if success
			if tt.wantStatus == http.StatusOK && tt.wantBody != "" {
				if got := w.Body.String(); got != tt.wantBody {
					t.Errorf("body = %q, want %q", got, tt.wantBody)
				}
			}

			// Verify error response
			if tt.wantStatus >= 400 {
				var errResp map[string]interface{}
				if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
					t.Errorf("failed to decode error response: %v", err)
				}
				if _, ok := errResp["error"]; !ok {
					t.Error("error response missing 'error' field")
				}
			}
		})
	}
}

func TestGetFile_Filename(t *testing.T) {
	mockService := &mockFileService{
		getFileFunc: func(ctx context.Context, req *domain.FileRequest) (*domain.File, error) {
			return &domain.File{
				TXID:    req.TXID,
				ChainID: req.ChainID,
				Content: []byte("file content"),
				Metadata: &domain.FileMetadata{
					Filename:    req.Filename,
					ContentType: "application/pdf",
					Size:        12,
				},
			}, nil
		},
	}

	handler := newTestHandler(mockService)

	// Create request with filename
	req := httptest.NewRequest(http.MethodGet, "/c/vrsctest/file/document.pdf?txid=abc123def456abc123def456abc123def456abc123def456abc123def456abc1", nil)
	q := req.URL.Query()
	q.Set("txid", "abc123def456abc123def456abc123def456abc123def456abc123def456abc1")
	req.URL.RawQuery = q.Encode()

	// Add route params
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("chain", "vrsctest")
	rctx.URLParams.Add("txid", "document.pdf")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	// Execute
	w := httptest.NewRecorder()
	handler.GetFile(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	if got := w.Body.String(); got != "file content" {
		t.Errorf("body = %q, want %q", got, "file content")
	}

	// Verify Content-Disposition header has filename
	disp := w.Header().Get("Content-Disposition")
	if disp == "" {
		t.Error("Content-Disposition header not set")
	}
}

func TestHeadFile(t *testing.T) {
	tests := []struct {
		name         string
		mockMetadata *domain.FileMetadata
		mockError    error
		wantStatus   int
		wantHeaders  map[string]string
	}{
		{
			name: "successful metadata retrieval",
			mockMetadata: &domain.FileMetadata{
				Filename:    "test.txt",
				ContentType: "text/plain",
				Size:        1024,
			},
			wantStatus: http.StatusOK,
			wantHeaders: map[string]string{
				"Content-Type":        "text/plain",
				"Content-Length":      "1024",
				"Content-Disposition": `inline; filename="test.txt"`,
			},
		},
		{
			name:       "file not found",
			mockError:  domain.NewNotFoundError("file", "abc123"),
			wantStatus: http.StatusNotFound,
		},
		{
			name: "metadata without filename",
			mockMetadata: &domain.FileMetadata{
				ContentType: "application/octet-stream",
				Size:        2048,
			},
			wantStatus: http.StatusOK,
			wantHeaders: map[string]string{
				"Content-Type":   "application/octet-stream",
				"Content-Length": "2048",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockFileService{
				getMetadataFunc: func(ctx context.Context, req *domain.FileRequest) (*domain.FileMetadata, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return tt.mockMetadata, nil
				},
			}

			handler := newTestHandler(mockService)

			req := httptest.NewRequest(http.MethodHead, "/c/vrsctest/file/abc123", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("chain", "vrsctest")
			rctx.URLParams.Add("txid", "abc123def456abc123def456abc123def456abc123def456abc123def456abc1")
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()
			handler.HeadFile(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusOK {
				for key, want := range tt.wantHeaders {
					got := w.Header().Get(key)
					if got != want {
						t.Errorf("header %s = %q, want %q", key, got, want)
					}
				}
			}

			// HEAD should not have body
			if w.Body.Len() > 0 && tt.wantStatus >= 400 {
				// Error responses can have body
			} else if w.Body.Len() > 0 && tt.wantStatus == http.StatusOK {
				t.Error("HEAD request should not have body on success")
			}
		})
	}
}

func TestGetMeta(t *testing.T) {
	tests := []struct {
		name         string
		mockMetadata *domain.FileMetadata
		mockError    error
		wantStatus   int
		checkFields  bool
	}{
		{
			name: "successful metadata JSON response",
			mockMetadata: &domain.FileMetadata{
				Filename:    "test.pdf",
				Size:        102400,
				ContentType: "application/pdf",
				Extension:   ".pdf",
				Compressed:  false,
			},
			wantStatus:  http.StatusOK,
			checkFields: true,
		},
		{
			name:       "file not found",
			mockError:  domain.NewNotFoundError("file", "abc123"),
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "decryption error",
			mockError:  domain.NewDecryptionError("abc123", errors.New("invalid key")),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockFileService{
				getMetadataFunc: func(ctx context.Context, req *domain.FileRequest) (*domain.FileMetadata, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return tt.mockMetadata, nil
				},
			}

			handler := newTestHandler(mockService)

			req := httptest.NewRequest(http.MethodGet, "/c/vrsctest/meta/abc123", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("chain", "vrsctest")
			rctx.URLParams.Add("txid", "abc123def456abc123def456abc123def456abc123def456abc123def456abc1")
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()
			handler.GetMeta(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			// Check JSON response
			var resp map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode JSON: %v", err)
			}

			if tt.checkFields {
				// Verify required fields
				if _, ok := resp["txid"]; !ok {
					t.Error("response missing 'txid' field")
				}
				if _, ok := resp["chain"]; !ok {
					t.Error("response missing 'chain' field")
				}
				if _, ok := resp["filename"]; !ok {
					t.Error("response missing 'filename' field")
				}
				if _, ok := resp["size"]; !ok {
					t.Error("response missing 'size' field")
				}
				if _, ok := resp["content_type"]; !ok {
					t.Error("response missing 'content_type' field")
				}

				// Verify values
				if got := resp["chain"].(string); got != "vrsctest" {
					t.Errorf("chain = %q, want %q", got, "vrsctest")
				}
				if got := resp["filename"].(string); got != tt.mockMetadata.Filename {
					t.Errorf("filename = %q, want %q", got, tt.mockMetadata.Filename)
				}
			}
		})
	}
}

func TestIsHexString(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"0123456789abcdef", true},
		{"0123456789ABCDEF", true},
		{"0123456789abcdefABCDEF", true},
		{"", true}, // empty string is valid hex
		{"abc123", true},
		{"xyz", false},
		{"abc123g", false},
		{"123xyz", false},
		{"abc-def", false},
		{"abc def", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isHexString(tt.input)
			if got != tt.want {
				t.Errorf("isHexString(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
