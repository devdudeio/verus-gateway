package service

import (
	"context"
	"fmt"
	"time"

	"github.com/devdudeio/verus-gateway/internal/chain"
	"github.com/devdudeio/verus-gateway/internal/crypto"
	"github.com/devdudeio/verus-gateway/internal/domain"
	"github.com/devdudeio/verus-gateway/internal/storage"
)

// FileService handles file retrieval, decryption, and processing
type FileService struct {
	chainManager *chain.Manager
	cache        domain.Cache
	decompressor *storage.Decompressor
	detector     *storage.Detector
}

// NewFileService creates a new file service
func NewFileService(
	chainManager *chain.Manager,
	cache domain.Cache,
) *FileService {
	return &FileService{
		chainManager: chainManager,
		cache:        cache,
		decompressor: storage.NewDecompressor(storage.DecompressorConfig{
			MaxSize: 100 * 1024 * 1024, // 100MB
		}),
		detector: storage.NewDetector(),
	}
}

// GetFile retrieves a file by TXID and EVK, with caching
func (s *FileService) GetFile(ctx context.Context, req *domain.FileRequest) (*domain.File, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Check cache first if enabled
	if req.UseCache && s.cache != nil {
		cacheKey := req.CacheKey()
		if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
			return cached, nil
		}
	}

	// Get RPC client for the chain
	client, err := s.getClient(req.ChainID)
	if err != nil {
		return nil, err
	}

	// Create decryptor with the client
	decryptor := crypto.NewDecryptor(client)

	// Decrypt data from blockchain
	encryptedData, err := decryptor.DecryptData(ctx, req.TXID, req.EVK)
	if err != nil {
		return nil, err
	}

	// Decompress if needed
	data, err := s.decompressor.Decompress(encryptedData)
	if err != nil {
		// Non-fatal: return encrypted data if decompression fails
		data = encryptedData
	}

	// Detect file type
	metadata, err := s.detector.DetectType(data, req.Filename)
	if err != nil {
		return nil, fmt.Errorf("failed to detect file type: %w", err)
	}

	// Create file object
	file := &domain.File{
		TXID:        req.TXID,
		ChainID:     req.ChainID,
		Content:     data,
		Metadata:    metadata,
		RetrievedAt: time.Now(),
	}

	// Cache the file if caching is enabled
	if req.UseCache && s.cache != nil {
		cacheKey := req.CacheKey()
		// Fire and forget - don't fail the request if caching fails
		go func() {
			// Use background context since original might be canceled
			cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := s.cache.Set(cacheCtx, cacheKey, file, 24*time.Hour); err != nil {
				fmt.Printf("[WARN] Failed to cache file %s: %v\n", req.TXID, err)
			}
		}()
	}

	return file, nil
}

// GetMetadata retrieves only the metadata for a file (without full content)
func (s *FileService) GetMetadata(ctx context.Context, req *domain.FileRequest) (*domain.FileMetadata, error) {
	// For now, we need to fetch the full file to get metadata
	// In the future, we could optimize this by caching metadata separately
	file, err := s.GetFile(ctx, req)
	if err != nil {
		return nil, err
	}

	return file.Metadata, nil
}

// getClient retrieves the RPC client for a chain
func (s *FileService) getClient(chainID string) (crypto.RPCClient, error) {
	if chainID == "" {
		// Use default chain
		client, err := s.chainManager.GetDefaultChain()
		if err != nil {
			return nil, err
		}
		return client, nil
	}

	// Get specific chain
	client, err := s.chainManager.GetChain(chainID)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// ClearCache clears the entire cache
func (s *FileService) ClearCache(ctx context.Context) error {
	if s.cache == nil {
		return fmt.Errorf("cache not configured")
	}
	return s.cache.Clear(ctx)
}

// GetCacheStats returns cache statistics
func (s *FileService) GetCacheStats(ctx context.Context) (*domain.CacheStats, error) {
	if s.cache == nil {
		return nil, fmt.Errorf("cache not configured")
	}
	return s.cache.Stats(ctx)
}

// DeleteFromCache removes a specific file from cache
func (s *FileService) DeleteFromCache(ctx context.Context, cacheKey string) error {
	if s.cache == nil {
		return fmt.Errorf("cache not configured")
	}
	return s.cache.Delete(ctx, cacheKey)
}
