package storage

import (
	"bytes"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/devdudeio/verus-gateway/internal/domain"
)

// Detector implements file type detection
type Detector struct{}

// NewDetector creates a new file detector
func NewDetector() *Detector {
	return &Detector{}
}

// DetectType detects the file type from content and optional filename
func (d *Detector) DetectType(content []byte, filename string) (*domain.FileMetadata, error) {
	metadata := &domain.FileMetadata{
		Filename: filename,
		Size:     int64(len(content)),
	}

	// Detect MIME type from content
	metadata.ContentType = d.DetectMIME(content)

	// Detect extension
	if filename != "" {
		ext := filepath.Ext(filename)
		if ext != "" {
			metadata.Extension = strings.TrimPrefix(ext, ".")
		}
	}

	// If no extension from filename, try to detect from content
	if metadata.Extension == "" {
		metadata.Extension = d.DetectExtension(content)
	}

	// Detect if compressed
	metadata.Compressed = d.isGzipCompressed(content)

	return metadata, nil
}

// DetectMIME detects MIME type from file content
func (d *Detector) DetectMIME(content []byte) string {
	if len(content) == 0 {
		return "application/octet-stream"
	}

	// Use http.DetectContentType for basic detection
	mimeType := http.DetectContentType(content)

	// Enhanced detection for specific formats
	if mimeType == "application/octet-stream" || mimeType == "text/plain; charset=utf-8" {
		// Check for specific file signatures
		if detected := d.detectBySignature(content); detected != "" {
			return detected
		}
	}

	return mimeType
}

// DetectExtension detects file extension from content
func (d *Detector) DetectExtension(content []byte) string {
	mime := d.DetectMIME(content)

	// Map common MIME types to extensions
	extMap := map[string]string{
		"image/jpeg":               "jpg",
		"image/png":                "png",
		"image/gif":                "gif",
		"image/webp":               "webp",
		"image/svg+xml":            "svg",
		"image/bmp":                "bmp",
		"video/mp4":                "mp4",
		"video/webm":               "webm",
		"video/mpeg":               "mpeg",
		"audio/mpeg":               "mp3",
		"audio/ogg":                "ogg",
		"audio/wav":                "wav",
		"application/pdf":          "pdf",
		"application/zip":          "zip",
		"application/x-gzip":       "gz",
		"application/json":         "json",
		"application/xml":          "xml",
		"text/html":                "html",
		"text/css":                 "css",
		"text/javascript":          "js",
		"text/plain":               "txt",
		"application/octet-stream": "bin",
	}

	if ext, ok := extMap[mime]; ok {
		return ext
	}

	return "bin"
}

// detectBySignature detects file type by magic bytes
func (d *Detector) detectBySignature(content []byte) string {
	if len(content) < 16 {
		return ""
	}

	// Check for common file signatures
	signatures := []struct {
		magic []byte
		mime  string
	}{
		// Images
		{[]byte{0xFF, 0xD8, 0xFF}, "image/jpeg"},
		{[]byte{0x89, 0x50, 0x4E, 0x47}, "image/png"},
		{[]byte("GIF87a"), "image/gif"},
		{[]byte("GIF89a"), "image/gif"},
		{[]byte("RIFF"), "image/webp"}, // Needs more specific check
		{[]byte{0x42, 0x4D}, "image/bmp"},

		// Videos
		{[]byte{0x00, 0x00, 0x00, 0x18, 0x66, 0x74, 0x79, 0x70}, "video/mp4"},
		{[]byte{0x1A, 0x45, 0xDF, 0xA3}, "video/webm"},

		// Audio
		{[]byte("ID3"), "audio/mpeg"},
		{[]byte{0xFF, 0xFB}, "audio/mpeg"},
		{[]byte("OggS"), "audio/ogg"},
		{[]byte("RIFF"), "audio/wav"}, // Needs more specific check

		// Documents
		{[]byte("%PDF"), "application/pdf"},
		{[]byte("PK\x03\x04"), "application/zip"},

		// Archives
		{[]byte{0x1F, 0x8B}, "application/x-gzip"},
		{[]byte("Rar!"), "application/x-rar-compressed"},
		{[]byte("7z\xBC\xAF\x27\x1C"), "application/x-7z-compressed"},
	}

	for _, sig := range signatures {
		if bytes.HasPrefix(content, sig.magic) {
			// Special handling for RIFF files (WAV, WEBP, AVI)
			if bytes.HasPrefix(content, []byte("RIFF")) && len(content) > 12 {
				if bytes.Contains(content[8:12], []byte("WAVE")) {
					return "audio/wav"
				}
				if bytes.Contains(content[8:12], []byte("WEBP")) {
					return "image/webp"
				}
				if bytes.Contains(content[8:12], []byte("AVI ")) {
					return "video/x-msvideo"
				}
			}

			return sig.mime
		}
	}

	return ""
}

// isGzipCompressed checks if content is gzip-compressed
func (d *Detector) isGzipCompressed(content []byte) bool {
	if len(content) < 2 {
		return false
	}
	return content[0] == 0x1F && content[1] == 0x8B
}

// IsTextLike checks if content appears to be text
func (d *Detector) IsTextLike(content []byte) bool {
	if len(content) == 0 {
		return false
	}

	// Check first 512 bytes for non-printable characters
	limit := 512
	if len(content) < limit {
		limit = len(content)
	}

	nonPrintable := 0
	for i := 0; i < limit; i++ {
		b := content[i]
		// Check for common control characters (except whitespace)
		if b < 0x20 && b != 0x09 && b != 0x0A && b != 0x0D {
			nonPrintable++
		}
		// Check for high bytes (likely binary)
		if b > 0x7E {
			nonPrintable++
		}
	}

	// If more than 30% non-printable, probably binary
	return float64(nonPrintable)/float64(limit) < 0.3
}
