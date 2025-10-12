package storage

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"

	"github.com/devdudeio/verus-gateway/internal/domain"
)

// Decompressor implements file decompression
type Decompressor struct {
	maxSize int64 // Maximum decompressed size to prevent zip bombs
}

// DecompressorConfig holds configuration for the decompressor
type DecompressorConfig struct {
	MaxSize int64 // Maximum decompressed size (default: 100MB)
}

// NewDecompressor creates a new decompressor
func NewDecompressor(cfg DecompressorConfig) *Decompressor {
	// Set defaults
	if cfg.MaxSize == 0 {
		cfg.MaxSize = 100 * 1024 * 1024 // 100MB default
	}

	return &Decompressor{
		maxSize: cfg.MaxSize,
	}
}

// Decompress attempts to decompress gzipped content
// Returns the decompressed data, or the original data if not gzipped
func (d *Decompressor) Decompress(content []byte) ([]byte, error) {
	// Check if content is gzipped
	if !d.isGzipped(content) {
		return content, nil
	}

	// Decompress
	decompressed, err := d.decompressGzip(content)
	if err != nil {
		return nil, domain.NewDecompressionError(fmt.Sprintf("gzip decompression failed: %v", err))
	}

	return decompressed, nil
}

// isGzipped checks if content is gzip-compressed
func (d *Decompressor) isGzipped(content []byte) bool {
	if len(content) < 2 {
		return false
	}
	return content[0] == 0x1F && content[1] == 0x8B
}

// decompressGzip decompresses gzip data with size limit protection
func (d *Decompressor) decompressGzip(content []byte) ([]byte, error) {
	gr, err := gzip.NewReader(bytes.NewReader(content))
	if err != nil {
		return nil, err
	}
	defer func() { _ = gr.Close() }()

	// Pre-allocate buffer (estimate 2x compressed size)
	var out bytes.Buffer
	out.Grow(len(content) * 2)

	// Use limited writer to prevent zip bombs
	lim := &limitedWriter{
		W: &out,
		N: d.maxSize,
	}

	// Copy with size limit
	if _, err := io.Copy(lim, gr); err != nil && err != io.EOF {
		if err == errSizeLimitExceeded {
			return nil, domain.NewDecompressionError(
				fmt.Sprintf("decompressed size exceeds limit of %d bytes", d.maxSize),
			)
		}
		return nil, err
	}

	return out.Bytes(), nil
}

// limitedWriter wraps an io.Writer and limits the number of bytes written
type limitedWriter struct {
	W io.Writer
	N int64 // Remaining bytes allowed
}

var errSizeLimitExceeded = fmt.Errorf("size limit exceeded")

func (l *limitedWriter) Write(p []byte) (int, error) {
	if l.N <= 0 {
		return 0, errSizeLimitExceeded
	}

	// Limit the write to the remaining allowed bytes
	if int64(len(p)) > l.N {
		p = p[:l.N]
	}

	n, err := l.W.Write(p)
	l.N -= int64(n)

	// Return error if we've exceeded the limit
	if l.N <= 0 && err == nil {
		return n, errSizeLimitExceeded
	}

	return n, err
}

// MustDecompress attempts decompression but returns original data on failure
// This is useful when you want to handle both compressed and uncompressed data
func (d *Decompressor) MustDecompress(content []byte) []byte {
	decompressed, err := d.Decompress(content)
	if err != nil {
		return content
	}
	return decompressed
}
