package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/devdudeio/verus-gateway/internal/domain"
)

// FilesystemCache implements cache using the local filesystem
type FilesystemCache struct {
	baseDir         string
	maxSize         int64
	ttl             time.Duration
	cleanupInterval time.Duration

	// Metrics
	hits   atomic.Uint64
	misses atomic.Uint64
	size   atomic.Int64
	items  atomic.Int64

	// Cleanup goroutine management
	stopCleanup chan struct{}
	wg          sync.WaitGroup
	mu          sync.RWMutex
}

// FilesystemCacheConfig holds configuration for filesystem cache
type FilesystemCacheConfig struct {
	BaseDir         string
	MaxSize         int64
	TTL             time.Duration
	CleanupInterval time.Duration
}

// NewFilesystemCache creates a new filesystem cache
func NewFilesystemCache(cfg FilesystemCacheConfig) (*FilesystemCache, error) {
	// Set defaults
	if cfg.MaxSize == 0 {
		cfg.MaxSize = 1024 * 1024 * 1024 // 1GB
	}
	if cfg.TTL == 0 {
		cfg.TTL = 24 * time.Hour
	}
	if cfg.CleanupInterval == 0 {
		cfg.CleanupInterval = 1 * time.Hour
	}

	// Create base directory
	if err := os.MkdirAll(cfg.BaseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	cache := &FilesystemCache{
		baseDir:         cfg.BaseDir,
		maxSize:         cfg.MaxSize,
		ttl:             cfg.TTL,
		cleanupInterval: cfg.CleanupInterval,
		stopCleanup:     make(chan struct{}),
	}

	// Calculate initial size and items
	cache.calculateSize()

	// Start cleanup goroutine
	cache.wg.Add(1)
	go cache.cleanupLoop()

	return cache, nil
}

// Get retrieves a file from cache
func (c *FilesystemCache) Get(ctx context.Context, key string) (*domain.File, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	// Generate file paths
	contentPath, metaPath := c.getPaths(key)

	// Check if files exist
	if _, err := os.Stat(contentPath); os.IsNotExist(err) {
		c.misses.Add(1)
		return nil, domain.ErrCacheMiss
	}

	// Check if expired
	info, err := os.Stat(contentPath)
	if err != nil {
		c.misses.Add(1)
		return nil, domain.ErrCacheMiss
	}

	if time.Since(info.ModTime()) > c.ttl {
		c.misses.Add(1)
		// File is expired, delete it
		_ = os.Remove(contentPath)
		_ = os.Remove(metaPath)
		return nil, domain.ErrCacheMiss
	}

	// Read content
	content, err := os.ReadFile(contentPath)
	if err != nil {
		c.misses.Add(1)
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	// Read metadata
	var metadata domain.FileMetadata
	metaBytes, err := os.ReadFile(metaPath)
	if err == nil {
		// Metadata is optional, ignore errors
		_ = json.Unmarshal(metaBytes, &metadata)
	}

	c.hits.Add(1)

	file := &domain.File{
		Content:     content,
		Metadata:    &metadata,
		RetrievedAt: time.Now(),
	}

	return file, nil
}

// Set stores a file in cache
func (c *FilesystemCache) Set(ctx context.Context, key string, file *domain.File, ttl time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Check size limit
	fileSize := int64(len(file.Content))
	if c.size.Load()+fileSize > c.maxSize {
		// Run eviction
		if err := c.evictOldest(fileSize); err != nil {
			return fmt.Errorf("failed to evict old entries: %w", err)
		}
	}

	// Generate file paths
	contentPath, metaPath := c.getPaths(key)

	// Create directory if needed
	dir := filepath.Dir(contentPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Write content
	if err := os.WriteFile(contentPath, file.Content, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	// Write metadata
	if file.Metadata != nil {
		metaBytes, err := json.Marshal(file.Metadata)
		if err == nil {
			_ = os.WriteFile(metaPath, metaBytes, 0644)
		}
	}

	// Update metrics
	c.size.Add(fileSize)
	c.items.Add(1)

	return nil
}

// Delete removes a file from cache
func (c *FilesystemCache) Delete(ctx context.Context, key string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	contentPath, metaPath := c.getPaths(key)

	// Get file size before deletion
	if info, err := os.Stat(contentPath); err == nil {
		c.size.Add(-info.Size())
		c.items.Add(-1)
	}

	// Delete files (ignore errors if files don't exist)
	_ = os.Remove(contentPath)
	_ = os.Remove(metaPath)

	return nil
}

// Clear removes all files from cache
func (c *FilesystemCache) Clear(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Remove all files in base directory
	if err := os.RemoveAll(c.baseDir); err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}

	// Recreate base directory
	if err := os.MkdirAll(c.baseDir, 0755); err != nil {
		return fmt.Errorf("failed to recreate cache directory: %w", err)
	}

	// Reset metrics
	c.size.Store(0)
	c.items.Store(0)

	return nil
}

// Stats returns cache statistics
func (c *FilesystemCache) Stats(ctx context.Context) (*domain.CacheStats, error) {
	hits := c.hits.Load()
	misses := c.misses.Load()

	var hitRate float64
	if total := hits + misses; total > 0 {
		hitRate = float64(hits) / float64(total)
	}

	return &domain.CacheStats{
		Hits:    int64(hits),
		Misses:  int64(misses),
		Size:    c.size.Load(),
		Items:   c.items.Load(),
		HitRate: hitRate,
	}, nil
}

// Close closes the cache and stops cleanup goroutine
func (c *FilesystemCache) Close() error {
	close(c.stopCleanup)
	c.wg.Wait()
	return nil
}

// getPaths returns the content and metadata file paths for a key
func (c *FilesystemCache) getPaths(key string) (string, string) {
	// Hash the key to create a filename
	hash := sha256.Sum256([]byte(key))
	filename := hex.EncodeToString(hash[:])

	// Use first 2 chars as subdirectory for better distribution
	subdir := filename[:2]

	contentPath := filepath.Join(c.baseDir, subdir, filename+".bin")
	metaPath := filepath.Join(c.baseDir, subdir, filename+".meta")

	return contentPath, metaPath
}

// calculateSize calculates the current cache size and items
func (c *FilesystemCache) calculateSize() {
	var totalSize int64
	var totalItems int64

	_ = filepath.Walk(c.baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		// Only count .bin files
		if filepath.Ext(path) == ".bin" {
			totalSize += info.Size()
			totalItems++
		}

		return nil
	})

	c.size.Store(totalSize)
	c.items.Store(totalItems)
}

// evictOldest evicts oldest files to make room for new file
func (c *FilesystemCache) evictOldest(neededSize int64) error {
	type fileInfo struct {
		path    string
		size    int64
		modTime time.Time
	}

	var files []fileInfo

	// Collect all cache files
	_ = filepath.Walk(c.baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		if filepath.Ext(path) == ".bin" {
			files = append(files, fileInfo{
				path:    path,
				size:    info.Size(),
				modTime: info.ModTime(),
			})
		}

		return nil
	})

	// Sort by modification time (oldest first)
	// Simple bubble sort for small datasets
	for i := 0; i < len(files)-1; i++ {
		for j := 0; j < len(files)-i-1; j++ {
			if files[j].modTime.After(files[j+1].modTime) {
				files[j], files[j+1] = files[j+1], files[j]
			}
		}
	}

	// Evict until we have enough space
	var freedSize int64
	for _, f := range files {
		if freedSize >= neededSize {
			break
		}

		// Delete file and its metadata
		_ = os.Remove(f.path)
		_ = os.Remove(f.path[:len(f.path)-4] + ".meta")

		freedSize += f.size
		c.size.Add(-f.size)
		c.items.Add(-1)
	}

	return nil
}

// cleanupLoop runs periodic cleanup of expired entries
func (c *FilesystemCache) cleanupLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCleanup:
			return
		case <-ticker.C:
			c.cleanup()
		}
	}
}

// cleanup removes expired cache entries
func (c *FilesystemCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()

	_ = filepath.Walk(c.baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		if filepath.Ext(path) != ".bin" {
			return nil
		}

		// Check if expired
		if now.Sub(info.ModTime()) > c.ttl {
			// Delete file and metadata
			size := info.Size()
			_ = os.Remove(path)
			_ = os.Remove(path[:len(path)-4] + ".meta")

			c.size.Add(-size)
			c.items.Add(-1)
		}

		return nil
	})
}
