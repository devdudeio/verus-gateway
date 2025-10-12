package domain

import (
	"regexp"
	"strings"
	"time"
)

// File represents a file retrieved from the blockchain
type File struct {
	// TXID is the transaction ID containing the file
	TXID string

	// ChainID is the blockchain identifier
	ChainID string

	// Content is the raw file content
	Content []byte

	// Metadata contains file metadata
	Metadata *FileMetadata

	// RetrievedAt is when the file was retrieved
	RetrievedAt time.Time
}

// FileMetadata contains metadata about a file
type FileMetadata struct {
	// Filename is the original filename (may be empty)
	Filename string

	// Size is the file size in bytes
	Size int64

	// ContentType is the MIME type
	ContentType string

	// Extension is the file extension (without dot)
	Extension string

	// Hash is the SHA256 hash of the content
	Hash string

	// Compressed indicates if the content was compressed
	Compressed bool

	// Encrypted indicates if the content was encrypted
	Encrypted bool

	// CreatedAt is when the file was stored on chain (if available)
	CreatedAt *time.Time
}

// FileRequest represents a request to retrieve a file
type FileRequest struct {
	// TXID is the transaction ID
	TXID string

	// EVK is the encryption viewing key (optional)
	EVK string

	// ChainID is the blockchain identifier
	ChainID string

	// Filename is the expected filename (for v2 API)
	Filename string

	// UseCache indicates whether to use cached version
	UseCache bool
}

var (
	// txidPattern matches valid transaction IDs (64 hex characters)
	txidPattern = regexp.MustCompile(`^[a-fA-F0-9]{64}$`)

	// evkPattern matches valid viewing keys (starts with zxviews)
	evkPattern = regexp.MustCompile(`^zxviews[a-zA-Z0-9]{90,}$`)

	// filenamePattern matches safe filenames (alphanumeric, dots, dashes, underscores, spaces, parentheses, brackets)
	filenamePattern = regexp.MustCompile(`^[a-zA-Z0-9._\-() \[\]]+$`)

	// chainIDPattern matches valid chain IDs (alphanumeric, dashes, underscores)
	chainIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_\-]+$`)
)

// Validate validates the file request
func (r *FileRequest) Validate() error {
	// Validate TXID
	if r.TXID == "" {
		return NewInvalidInputError("txid", "txid is required")
	}

	if len(r.TXID) != 64 {
		return NewInvalidInputError("txid", "txid must be exactly 64 characters")
	}

	if !txidPattern.MatchString(r.TXID) {
		return NewInvalidInputError("txid", "txid must be valid hex (0-9, a-f)")
	}

	// Validate ChainID
	if r.ChainID == "" {
		return NewInvalidInputError("chain_id", "chain_id is required")
	}

	if len(r.ChainID) > 32 {
		return NewInvalidInputError("chain_id", "chain_id too long (max 32 characters)")
	}

	if !chainIDPattern.MatchString(r.ChainID) {
		return NewInvalidInputError("chain_id", "chain_id contains invalid characters")
	}

	// Validate filename if provided
	if r.Filename != "" {
		if len(r.Filename) > 255 {
			return NewInvalidInputError("filename", "filename too long (max 255 characters)")
		}

		// Check for path traversal attempts
		if strings.Contains(r.Filename, "..") || strings.Contains(r.Filename, "/") || strings.Contains(r.Filename, "\\") {
			return NewInvalidInputError("filename", "filename contains invalid path characters")
		}

		// Check for safe characters only
		if !filenamePattern.MatchString(r.Filename) {
			return NewInvalidInputError("filename", "filename contains invalid characters")
		}
	}

	// Validate EVK if provided
	if r.EVK != "" {
		if len(r.EVK) < 95 || len(r.EVK) > 500 {
			return NewInvalidInputError("evk", "viewing key has invalid length (must be 95-500 characters)")
		}

		if !evkPattern.MatchString(r.EVK) {
			return NewInvalidInputError("evk", "viewing key has invalid format (must start with 'zxviews')")
		}
	}

	return nil
}

// CacheKey returns the cache key for this file
func (r *FileRequest) CacheKey() string {
	// Include EVK in cache key for encrypted files
	if r.EVK != "" {
		// Use a hash of EVK to avoid storing sensitive data in key
		return r.ChainID + ":" + r.TXID + ":encrypted"
	}
	return r.ChainID + ":" + r.TXID
}
