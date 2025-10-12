package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/devdudeio/verus-gateway/internal/domain"
	"github.com/devdudeio/verus-gateway/internal/http/middleware"
	"github.com/devdudeio/verus-gateway/internal/service"
	"github.com/go-chi/chi/v5"
)

// FileServiceInterface defines the interface for file operations
type FileServiceInterface interface {
	GetFile(ctx context.Context, req *domain.FileRequest) (*domain.File, error)
	GetMetadata(ctx context.Context, req *domain.FileRequest) (*domain.FileMetadata, error)
}

// FileHandler handles file-related HTTP requests
type FileHandler struct {
	fileService FileServiceInterface
}

// NewFileHandler creates a new file handler
func NewFileHandler(fileService *service.FileService) *FileHandler {
	h := &FileHandler{}
	if fileService != nil {
		h.fileService = fileService
	}
	return h
}

// GetFile handles GET /c/{chain}/file/{txid_or_filename}?txid=xxx&evk=xxx
// Supports both TXID-based and filename-based retrieval:
// - If path param is 64 hex chars: treated as TXID
// - Otherwise: treated as filename (requires txid query param)
func (h *FileHandler) GetFile(w http.ResponseWriter, r *http.Request) {
	chainID := chi.URLParam(r, "chain")
	pathParam := chi.URLParam(r, "txid")
	evk := r.URL.Query().Get("evk")

	// Determine if path param is TXID or filename
	// TXID is always 64 hex characters
	var req *domain.FileRequest
	if len(pathParam) == 64 && isHexString(pathParam) {
		// Path param is a TXID
		req = &domain.FileRequest{
			TXID:     pathParam,
			EVK:      evk,
			ChainID:  chainID,
			UseCache: true,
		}
	} else {
		// Path param is a filename, get TXID from query
		txid := r.URL.Query().Get("txid")
		req = &domain.FileRequest{
			TXID:     txid,
			EVK:      evk,
			ChainID:  chainID,
			Filename: pathParam,
			UseCache: true,
		}
	}

	// Get file
	file, err := h.fileService.GetFile(r.Context(), req)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	// Override filename from URL if metadata doesn't have it
	if req.Filename != "" && file.Metadata.Filename == "" {
		file.Metadata.Filename = req.Filename
	}

	// Set headers
	h.setFileHeaders(w, file)

	// Write content
	w.WriteHeader(http.StatusOK)
	w.Write(file.Content)
}

// isHexString checks if a string contains only hexadecimal characters
func isHexString(s string) bool {
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}
	return true
}

// HeadFile handles HEAD /c/{chain}/file/{txid}?evk=xxx
func (h *FileHandler) HeadFile(w http.ResponseWriter, r *http.Request) {
	chainID := chi.URLParam(r, "chain")
	txid := chi.URLParam(r, "txid")
	evk := r.URL.Query().Get("evk")

	// Build request
	req := &domain.FileRequest{
		TXID:     txid,
		EVK:      evk,
		ChainID:  chainID,
		UseCache: true,
	}

	// Get metadata only
	metadata, err := h.fileService.GetMetadata(r.Context(), req)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	// Set headers
	if metadata.ContentType != "" {
		w.Header().Set("Content-Type", metadata.ContentType)
	}
	if metadata.Size > 0 {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", metadata.Size))
	}
	if metadata.Filename != "" {
		w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, metadata.Filename))
	}

	w.WriteHeader(http.StatusOK)
}

// GetMeta handles GET /c/{chain}/meta/{txid}?evk=xxx
func (h *FileHandler) GetMeta(w http.ResponseWriter, r *http.Request) {
	chainID := chi.URLParam(r, "chain")
	txid := chi.URLParam(r, "txid")
	evk := r.URL.Query().Get("evk")

	// Build request
	req := &domain.FileRequest{
		TXID:     txid,
		EVK:      evk,
		ChainID:  chainID,
		UseCache: true,
	}

	// Get metadata
	metadata, err := h.fileService.GetMetadata(r.Context(), req)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	// Write JSON response
	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"txid":         txid,
		"chain":        chainID,
		"filename":     metadata.Filename,
		"size":         metadata.Size,
		"content_type": metadata.ContentType,
		"extension":    metadata.Extension,
		"compressed":   metadata.Compressed,
	})
}

// setFileHeaders sets appropriate HTTP headers for file responses
func (h *FileHandler) setFileHeaders(w http.ResponseWriter, file *domain.File) {
	if file.Metadata.ContentType != "" {
		w.Header().Set("Content-Type", file.Metadata.ContentType)
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	if file.Metadata.Filename != "" {
		// Sanitize filename for header
		filename := strings.ReplaceAll(file.Metadata.Filename, `"`, `\"`)
		w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, filename))
	}

	if file.Metadata.Size > 0 {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", file.Metadata.Size))
	}

	// Cache headers
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("ETag", fmt.Sprintf(`"%s"`, file.TXID))
}

// writeJSON writes a JSON response
func (h *FileHandler) writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		fmt.Printf("[ERROR] Failed to encode JSON response: %v\n", err)
	}
}

// writeError writes an error response
func (h *FileHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	requestID := middleware.GetRequestID(r.Context())

	// Determine status code and error details from domain error
	var domainErr *domain.Error
	var statusCode int
	var errorCode string
	var errorMessage string

	if e, ok := err.(*domain.Error); ok {
		domainErr = e
		statusCode = e.HTTPStatus
		errorCode = e.Code
		errorMessage = e.Message
	} else {
		// Generic error
		statusCode = http.StatusInternalServerError
		errorCode = "INTERNAL_ERROR"
		errorMessage = "An internal error occurred"
	}

	// Log the error
	fmt.Printf("[ERROR] Request failed: %v (request_id=%s)\n", err, requestID)

	// Write error response
	response := map[string]interface{}{
		"error":      errorCode,
		"message":    errorMessage,
		"request_id": requestID,
	}

	// Add details if available
	if domainErr != nil && len(domainErr.Details) > 0 {
		response["details"] = domainErr.Details
	}

	h.writeJSON(w, statusCode, response)
}
