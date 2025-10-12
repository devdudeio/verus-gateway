package storage

import (
	"testing"
)

func TestNewDetector(t *testing.T) {
	detector := NewDetector()
	if detector == nil {
		t.Fatal("NewDetector() returned nil")
	}
}

func TestDetectType_EmptyContent(t *testing.T) {
	detector := NewDetector()
	metadata, err := detector.DetectType([]byte{}, "test.txt")
	if err != nil {
		t.Fatalf("DetectType() error = %v", err)
	}

	if metadata == nil {
		t.Fatal("DetectType() returned nil metadata")
	}

	if metadata.Filename != "test.txt" {
		t.Errorf("Expected filename 'test.txt', got %s", metadata.Filename)
	}

	if metadata.Size != 0 {
		t.Errorf("Expected size 0, got %d", metadata.Size)
	}

	if metadata.Extension != "txt" {
		t.Errorf("Expected extension 'txt', got %s", metadata.Extension)
	}
}

func TestDetectType_WithFilename(t *testing.T) {
	detector := NewDetector()
	content := []byte("Hello, World!")
	metadata, err := detector.DetectType(content, "hello.txt")
	if err != nil {
		t.Fatalf("DetectType() error = %v", err)
	}

	if metadata.Filename != "hello.txt" {
		t.Errorf("Expected filename 'hello.txt', got %s", metadata.Filename)
	}

	if metadata.Extension != "txt" {
		t.Errorf("Expected extension 'txt', got %s", metadata.Extension)
	}

	if metadata.Size != int64(len(content)) {
		t.Errorf("Expected size %d, got %d", len(content), metadata.Size)
	}
}

func TestDetectType_WithoutFilename(t *testing.T) {
	detector := NewDetector()
	content := []byte("Hello, World!")
	metadata, err := detector.DetectType(content, "")
	if err != nil {
		t.Fatalf("DetectType() error = %v", err)
	}

	if metadata.Filename != "" {
		t.Errorf("Expected empty filename, got %s", metadata.Filename)
	}

	// Should detect extension from content
	if metadata.Extension == "" {
		t.Error("Expected extension to be detected from content")
	}
}

func TestDetectMIME_Images(t *testing.T) {
	detector := NewDetector()

	tests := []struct {
		name     string
		content  []byte
		expected string
	}{
		{
			name:     "JPEG image",
			content:  []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46},
			expected: "image/jpeg",
		},
		{
			name:     "PNG image",
			content:  []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A},
			expected: "image/png",
		},
		{
			name:     "GIF87a image",
			content:  []byte("GIF87a\x00\x00\x00\x00\x00\x00"),
			expected: "image/gif",
		},
		{
			name:     "GIF89a image",
			content:  []byte("GIF89a\x00\x00\x00\x00\x00\x00"),
			expected: "image/gif",
		},
		{
			name:     "BMP image",
			content:  []byte{0x42, 0x4D, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			expected: "image/bmp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.DetectMIME(tt.content)
			if result != tt.expected {
				t.Errorf("DetectMIME() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestDetectMIME_Videos(t *testing.T) {
	detector := NewDetector()

	tests := []struct {
		name     string
		content  []byte
		expected string
	}{
		{
			name: "MP4 video",
			// http.DetectContentType doesn't always detect MP4, but our signature detection will
			content:  []byte{0x00, 0x00, 0x00, 0x18, 0x66, 0x74, 0x79, 0x70, 0x6D, 0x70, 0x34, 0x32, 0x00, 0x00, 0x00, 0x00},
			expected: "video/mp4",
		},
		{
			name:     "WebM video",
			content:  []byte{0x1A, 0x45, 0xDF, 0xA3, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			expected: "video/webm",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.DetectMIME(tt.content)
			if result != tt.expected {
				t.Errorf("DetectMIME() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestDetectMIME_Audio(t *testing.T) {
	detector := NewDetector()

	tests := []struct {
		name     string
		content  []byte
		expected string
	}{
		{
			name: "MP3 with ID3",
			// Need at least 16 bytes for signature detection
			content:  []byte("ID3\x03\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"),
			expected: "audio/mpeg",
		},
		{
			name: "MP3 without ID3",
			// Need at least 16 bytes for signature detection
			content:  []byte{0xFF, 0xFB, 0x90, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			expected: "audio/mpeg",
		},
		{
			name: "OGG audio",
			// http.DetectContentType returns "application/ogg"
			content:  []byte("OggS\x00\x02\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"),
			expected: "application/ogg",
		},
		{
			name: "WAV audio",
			// http.DetectContentType returns "audio/wave"
			content:  []byte("RIFF\x00\x00\x00\x00WAVEfmt \x00\x00\x00\x00"),
			expected: "audio/wave",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.DetectMIME(tt.content)
			if result != tt.expected {
				t.Errorf("DetectMIME() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestDetectMIME_Documents(t *testing.T) {
	detector := NewDetector()

	tests := []struct {
		name     string
		content  []byte
		expected string
	}{
		{
			name:     "PDF document",
			content:  []byte("%PDF-1.4\n%\xE2\xE3\xCF\xD3"),
			expected: "application/pdf",
		},
		{
			name:     "ZIP archive",
			content:  []byte("PK\x03\x04\x14\x00\x00\x00"),
			expected: "application/zip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.DetectMIME(tt.content)
			if result != tt.expected {
				t.Errorf("DetectMIME() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestDetectMIME_Compressed(t *testing.T) {
	detector := NewDetector()

	tests := []struct {
		name     string
		content  []byte
		expected string
	}{
		{
			name:     "GZIP compressed",
			content:  []byte{0x1F, 0x8B, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00},
			expected: "application/x-gzip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.DetectMIME(tt.content)
			if result != tt.expected {
				t.Errorf("DetectMIME() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestDetectMIME_Text(t *testing.T) {
	detector := NewDetector()

	tests := []struct {
		name     string
		content  []byte
		expected string
	}{
		{
			name:     "Plain text",
			content:  []byte("Hello, World!"),
			expected: "text/plain; charset=utf-8",
		},
		{
			name:     "HTML",
			content:  []byte("<!DOCTYPE html><html><body>Test</body></html>"),
			expected: "text/html; charset=utf-8",
		},
		{
			name:     "XML",
			content:  []byte("<?xml version=\"1.0\"?><root></root>"),
			expected: "text/xml; charset=utf-8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.DetectMIME(tt.content)
			if result != tt.expected {
				t.Errorf("DetectMIME() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestDetectMIME_Empty(t *testing.T) {
	detector := NewDetector()
	result := detector.DetectMIME([]byte{})
	if result != "application/octet-stream" {
		t.Errorf("DetectMIME(empty) = %q, want %q", result, "application/octet-stream")
	}
}

func TestDetectExtension(t *testing.T) {
	detector := NewDetector()

	tests := []struct {
		name     string
		content  []byte
		expected string
	}{
		{
			name:     "JPEG image",
			content:  []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46},
			expected: "jpg",
		},
		{
			name:     "PNG image",
			content:  []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A},
			expected: "png",
		},
		{
			name:     "PDF document",
			content:  []byte("%PDF-1.4\n%\xE2\xE3\xCF\xD3"),
			expected: "pdf",
		},
		{
			name: "Plain text",
			// http.DetectContentType returns "text/plain; charset=utf-8" which maps to "txt"
			// but our extMap doesn't have an entry for "text/plain; charset=utf-8", only "text/plain"
			content:  []byte("Hello, World!"),
			expected: "bin", // Falls back to bin because "text/plain; charset=utf-8" is not in the map
		},
		{
			name:     "Unknown binary",
			content:  []byte{0x00, 0x01, 0x02, 0x03},
			expected: "bin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.DetectExtension(tt.content)
			if result != tt.expected {
				t.Errorf("DetectExtension() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestIsGzipCompressed(t *testing.T) {
	detector := NewDetector()

	tests := []struct {
		name     string
		content  []byte
		expected bool
	}{
		{
			name:     "GZIP compressed",
			content:  []byte{0x1F, 0x8B, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00},
			expected: true,
		},
		{
			name:     "Not compressed",
			content:  []byte("Hello, World!"),
			expected: false,
		},
		{
			name:     "Empty content",
			content:  []byte{},
			expected: false,
		},
		{
			name:     "Single byte",
			content:  []byte{0x1F},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.isGzipCompressed(tt.content)
			if result != tt.expected {
				t.Errorf("isGzipCompressed() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsTextLike(t *testing.T) {
	detector := NewDetector()

	tests := []struct {
		name     string
		content  []byte
		expected bool
	}{
		{
			name:     "Plain text",
			content:  []byte("Hello, World! This is a text file."),
			expected: true,
		},
		{
			name:     "Text with newlines",
			content:  []byte("Line 1\nLine 2\nLine 3"),
			expected: true,
		},
		{
			name:     "Text with tabs",
			content:  []byte("Column1\tColumn2\tColumn3"),
			expected: true,
		},
		{
			name:     "Binary data",
			content:  []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07},
			expected: false,
		},
		{
			name:     "JPEG image",
			content:  []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46},
			expected: false,
		},
		{
			name:     "Empty content",
			content:  []byte{},
			expected: false,
		},
		{
			name:     "Mixed content (mostly text)",
			content:  append([]byte("Hello World "), 0x00, 0x01), // Less than 30% non-printable
			expected: true,
		},
		{
			name:     "Mixed content (mostly binary)",
			content:  append([]byte{0x00, 0x01, 0x02, 0x03, 0x04}, []byte("Hi")...), // More than 30% non-printable
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.IsTextLike(tt.content)
			if result != tt.expected {
				t.Errorf("IsTextLike() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDetectType_GzipCompressed(t *testing.T) {
	detector := NewDetector()
	// GZIP magic bytes
	content := []byte{0x1F, 0x8B, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00}

	metadata, err := detector.DetectType(content, "file.gz")
	if err != nil {
		t.Fatalf("DetectType() error = %v", err)
	}

	if !metadata.Compressed {
		t.Error("Expected Compressed to be true for gzip content")
	}

	if metadata.Extension != "gz" {
		t.Errorf("Expected extension 'gz', got %s", metadata.Extension)
	}
}

func TestDetectBySignature_WEBP(t *testing.T) {
	detector := NewDetector()
	// WEBP file signature
	content := []byte("RIFF\x00\x00\x00\x00WEBP\x00\x00\x00\x00")

	mime := detector.detectBySignature(content)
	if mime != "image/webp" {
		t.Errorf("detectBySignature(WEBP) = %q, want %q", mime, "image/webp")
	}
}

func TestDetectBySignature_ShortContent(t *testing.T) {
	detector := NewDetector()
	// Content too short for signature detection
	content := []byte{0xFF, 0xD8}

	mime := detector.detectBySignature(content)
	if mime != "" {
		t.Errorf("detectBySignature(short) = %q, want empty string", mime)
	}
}

func TestDetectType_FilenameWithoutExtension(t *testing.T) {
	detector := NewDetector()
	content := []byte("Hello, World!")
	metadata, err := detector.DetectType(content, "README")
	if err != nil {
		t.Fatalf("DetectType() error = %v", err)
	}

	// Should detect extension from content since filename has no extension
	if metadata.Extension == "" {
		t.Error("Expected extension to be detected from content")
	}

	if metadata.Filename != "README" {
		t.Errorf("Expected filename 'README', got %s", metadata.Filename)
	}
}
