package domain

import (
	"context"
	"time"
)

// Cache defines the interface for file caching
type Cache interface {
	// Get retrieves a file from cache
	Get(ctx context.Context, key string) (*File, error)

	// Set stores a file in cache with TTL
	Set(ctx context.Context, key string, file *File, ttl time.Duration) error

	// Delete removes a file from cache
	Delete(ctx context.Context, key string) error

	// Clear removes all files from cache
	Clear(ctx context.Context) error

	// Stats returns cache statistics
	Stats(ctx context.Context) (*CacheStats, error)

	// Close closes the cache connection
	Close() error
}

// CacheStats contains cache statistics
type CacheStats struct {
	// Hits is the number of cache hits
	Hits int64

	// Misses is the number of cache misses
	Misses int64

	// Size is the current cache size in bytes
	Size int64

	// Items is the number of items in cache
	Items int64

	// HitRate is the cache hit rate (0.0 to 1.0)
	HitRate float64
}

// RPCClient defines the interface for blockchain RPC calls
type RPCClient interface {
	// DecryptData calls the decryptdata RPC method
	DecryptData(ctx context.Context, txid, evk string) (string, error)

	// GetInfo calls the getinfo RPC method (for health checks)
	GetInfo(ctx context.Context) (*ChainInfo, error)

	// Close closes the RPC connection
	Close() error
}

// ChainInfo contains blockchain information
type ChainInfo struct {
	// Chain is the chain name
	Chain string

	// Blocks is the current block height
	Blocks int64

	// Version is the daemon version
	Version int

	// Connections is the number of peer connections
	Connections int

	// Synced indicates if the chain is fully synced
	Synced bool
}

// FileService defines the interface for file operations
type FileService interface {
	// GetFile retrieves a file by request
	GetFile(ctx context.Context, req *FileRequest) (*File, error)

	// GetFileMetadata retrieves only file metadata
	GetFileMetadata(ctx context.Context, req *FileRequest) (*FileMetadata, error)
}

// ChainManager defines the interface for managing blockchain connections
type ChainManager interface {
	// GetChain returns a chain client by ID
	GetChain(chainID string) (RPCClient, error)

	// GetDefaultChain returns the default chain client
	GetDefaultChain() (RPCClient, error)

	// ListChains returns all configured chain IDs
	ListChains() []string

	// HealthCheck checks if a chain is healthy
	HealthCheck(ctx context.Context, chainID string) error

	// Close closes all chain connections
	Close() error
}

// Decryptor defines the interface for data decryption
type Decryptor interface {
	// Decrypt decrypts encrypted data
	Decrypt(ctx context.Context, encryptedData string) ([]byte, error)
}

// FileDetector defines the interface for file type detection
type FileDetector interface {
	// DetectType detects the file type from content
	DetectType(content []byte, filename string) (*FileMetadata, error)

	// DetectMIME detects MIME type from content
	DetectMIME(content []byte) string

	// DetectExtension detects file extension from content
	DetectExtension(content []byte) string
}

// Decompressor defines the interface for data decompression
type Decompressor interface {
	// Decompress decompresses gzip-compressed data
	Decompress(data []byte, maxSize int64) ([]byte, error)

	// IsCompressed checks if data is compressed
	IsCompressed(data []byte) bool
}
