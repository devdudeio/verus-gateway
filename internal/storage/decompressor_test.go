package storage

import (
	"bytes"
	"compress/gzip"
	"errors"
	"strings"
	"testing"

	"github.com/devdudeio/verus-gateway/internal/domain"
)

func TestNewDecompressor(t *testing.T) {
	tests := []struct {
		name     string
		cfg      DecompressorConfig
		wantSize int64
	}{
		{
			name:     "default config",
			cfg:      DecompressorConfig{},
			wantSize: 100 * 1024 * 1024, // 100MB default
		},
		{
			name:     "custom max size",
			cfg:      DecompressorConfig{MaxSize: 50 * 1024 * 1024},
			wantSize: 50 * 1024 * 1024,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDecompressor(tt.cfg)
			if d.maxSize != tt.wantSize {
				t.Errorf("expected maxSize %d, got %d", tt.wantSize, d.maxSize)
			}
		})
	}
}

func TestDecompressor_Decompress_UncompressedData(t *testing.T) {
	d := NewDecompressor(DecompressorConfig{})

	// Test with regular uncompressed data
	input := []byte("hello world")
	output, err := d.Decompress(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !bytes.Equal(output, input) {
		t.Errorf("expected output to equal input for uncompressed data")
	}
}

func TestDecompressor_Decompress_GzipData(t *testing.T) {
	d := NewDecompressor(DecompressorConfig{})

	// Create gzipped data
	original := []byte("hello world, this is compressed data!")
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write(original); err != nil {
		t.Fatalf("failed to write gzip data: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("failed to close gzip writer: %v", err)
	}
	compressed := buf.Bytes()

	// Decompress
	output, err := d.Decompress(compressed)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !bytes.Equal(output, original) {
		t.Errorf("expected %q, got %q", string(original), string(output))
	}
}

func TestDecompressor_Decompress_SizeLimit(t *testing.T) {
	// Create decompressor with small size limit
	d := NewDecompressor(DecompressorConfig{MaxSize: 100})

	// Create data that will exceed the limit when decompressed
	original := bytes.Repeat([]byte("x"), 200) // 200 bytes
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write(original); err != nil {
		t.Fatalf("failed to write gzip data: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("failed to close gzip writer: %v", err)
	}
	compressed := buf.Bytes()

	// Decompress should fail due to size limit
	_, err := d.Decompress(compressed)
	if err == nil {
		t.Fatal("expected error due to size limit, got nil")
	}

	// Check if it's a decompression error
	var domainErr *domain.Error
	if !errors.As(err, &domainErr) {
		t.Errorf("expected domain.Error, got %T", err)
	}
}

func TestDecompressor_Decompress_InvalidGzip(t *testing.T) {
	d := NewDecompressor(DecompressorConfig{})

	// Create data with gzip magic bytes but invalid content
	invalid := []byte{0x1F, 0x8B, 0x00, 0x00, 0x00, 0x00}

	_, err := d.Decompress(invalid)
	if err == nil {
		t.Fatal("expected error for invalid gzip data, got nil")
	}
}

func TestDecompressor_isGzipped(t *testing.T) {
	d := NewDecompressor(DecompressorConfig{})

	tests := []struct {
		name     string
		content  []byte
		expected bool
	}{
		{
			name:     "empty content",
			content:  []byte{},
			expected: false,
		},
		{
			name:     "too short",
			content:  []byte{0x1F},
			expected: false,
		},
		{
			name:     "gzip magic bytes",
			content:  []byte{0x1F, 0x8B},
			expected: true,
		},
		{
			name:     "gzip with more data",
			content:  []byte{0x1F, 0x8B, 0x08, 0x00},
			expected: true,
		},
		{
			name:     "not gzipped",
			content:  []byte("hello world"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := d.isGzipped(tt.content)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestDecompressor_MustDecompress(t *testing.T) {
	d := NewDecompressor(DecompressorConfig{})

	t.Run("successful decompression", func(t *testing.T) {
		// Create gzipped data
		original := []byte("test data")
		var buf bytes.Buffer
		gw := gzip.NewWriter(&buf)
		gw.Write(original)
		gw.Close()

		output := d.MustDecompress(buf.Bytes())
		if !bytes.Equal(output, original) {
			t.Errorf("expected %q, got %q", string(original), string(output))
		}
	})

	t.Run("failed decompression returns original", func(t *testing.T) {
		// Invalid gzip data
		invalid := []byte{0x1F, 0x8B, 0x00}

		output := d.MustDecompress(invalid)
		if !bytes.Equal(output, invalid) {
			t.Error("expected original data on decompression failure")
		}
	})

	t.Run("uncompressed data returns as-is", func(t *testing.T) {
		original := []byte("plain text")

		output := d.MustDecompress(original)
		if !bytes.Equal(output, original) {
			t.Error("expected original data for uncompressed input")
		}
	})
}

func TestDecompressor_LargeData(t *testing.T) {
	d := NewDecompressor(DecompressorConfig{MaxSize: 10 * 1024 * 1024}) // 10MB

	// Create large but compressible data (1MB of repeated text)
	original := []byte(strings.Repeat("Hello World! ", 80000)) // ~1MB
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write(original); err != nil {
		t.Fatalf("failed to write gzip data: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("failed to close gzip writer: %v", err)
	}
	compressed := buf.Bytes()

	t.Logf("Original size: %d bytes", len(original))
	t.Logf("Compressed size: %d bytes", len(compressed))
	t.Logf("Compression ratio: %.2f%%", float64(len(compressed))/float64(len(original))*100)

	// Decompress
	output, err := d.Decompress(compressed)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !bytes.Equal(output, original) {
		t.Error("decompressed data does not match original")
	}
}

func TestLimitedWriter_Write(t *testing.T) {
	tests := []struct {
		name      string
		limit     int64
		writes    [][]byte
		expectErr bool
	}{
		{
			name:      "within limit",
			limit:     100,
			writes:    [][]byte{[]byte("hello"), []byte("world")},
			expectErr: false,
		},
		{
			name:      "exact limit",
			limit:     10,
			writes:    [][]byte{[]byte("hello"), []byte("world")},
			expectErr: true, // Second write will exceed
		},
		{
			name:      "exceed limit",
			limit:     5,
			writes:    [][]byte{[]byte("hello world")},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			lw := &limitedWriter{W: &buf, N: tt.limit}

			var gotErr bool
			for _, data := range tt.writes {
				_, err := lw.Write(data)
				if err == errSizeLimitExceeded {
					gotErr = true
					break
				}
			}

			if gotErr != tt.expectErr {
				t.Errorf("expected error: %v, got error: %v", tt.expectErr, gotErr)
			}
		})
	}
}
