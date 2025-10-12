package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/devdudeio/verus-gateway/internal/domain"
	"github.com/redis/go-redis/v9"
)

// RedisCache implements cache using Redis
type RedisCache struct {
	client *redis.Client
	ttl    time.Duration

	// Metrics
	hits   atomic.Uint64
	misses atomic.Uint64
}

// RedisCacheConfig holds configuration for Redis cache
type RedisCacheConfig struct {
	Addresses  []string
	Password   string
	DB         int
	MaxRetries int
	PoolSize   int
	Timeout    time.Duration
	TTL        time.Duration
}

// NewRedisCache creates a new Redis cache
func NewRedisCache(cfg RedisCacheConfig) (*RedisCache, error) {
	// Set defaults
	if cfg.Timeout == 0 {
		cfg.Timeout = 5 * time.Second
	}
	if cfg.TTL == 0 {
		cfg.TTL = 24 * time.Hour
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}
	if cfg.PoolSize == 0 {
		cfg.PoolSize = 10
	}

	// Use first address (for single-instance mode)
	addr := "localhost:6379"
	if len(cfg.Addresses) > 0 {
		addr = cfg.Addresses[0]
	}

	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		MaxRetries:   cfg.MaxRetries,
		PoolSize:     cfg.PoolSize,
		DialTimeout:  cfg.Timeout,
		ReadTimeout:  cfg.Timeout,
		WriteTimeout: cfg.Timeout,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisCache{
		client: client,
		ttl:    cfg.TTL,
	}, nil
}

// cacheEntry is the structure stored in Redis
type cacheEntry struct {
	Content     []byte               `json:"content"`
	Metadata    *domain.FileMetadata `json:"metadata,omitempty"`
	RetrievedAt time.Time            `json:"retrieved_at"`
}

// Get retrieves a file from cache
func (c *RedisCache) Get(ctx context.Context, key string) (*domain.File, error) {
	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		c.misses.Add(1)
		return nil, domain.ErrCacheMiss
	}
	if err != nil {
		c.misses.Add(1)
		return nil, fmt.Errorf("redis get failed: %w", err)
	}

	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		c.misses.Add(1)
		return nil, fmt.Errorf("failed to unmarshal cache entry: %w", err)
	}

	c.hits.Add(1)

	return &domain.File{
		Content:     entry.Content,
		Metadata:    entry.Metadata,
		RetrievedAt: entry.RetrievedAt,
	}, nil
}

// Set stores a file in cache
func (c *RedisCache) Set(ctx context.Context, key string, file *domain.File, ttl time.Duration) error {
	if ttl == 0 {
		ttl = c.ttl
	}

	entry := cacheEntry{
		Content:     file.Content,
		Metadata:    file.Metadata,
		RetrievedAt: time.Now(),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal cache entry: %w", err)
	}

	if err := c.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("redis set failed: %w", err)
	}

	return nil
}

// Delete removes a file from cache
func (c *RedisCache) Delete(ctx context.Context, key string) error {
	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis del failed: %w", err)
	}
	return nil
}

// Clear removes all files from cache
func (c *RedisCache) Clear(ctx context.Context) error {
	if err := c.client.FlushDB(ctx).Err(); err != nil {
		return fmt.Errorf("redis flushdb failed: %w", err)
	}
	return nil
}

// Stats returns cache statistics
func (c *RedisCache) Stats(ctx context.Context) (*domain.CacheStats, error) {
	// Get Redis info (not currently parsing detailed stats)
	_, err := c.client.Info(ctx, "stats", "keyspace").Result()
	if err != nil {
		return nil, fmt.Errorf("redis info failed: %w", err)
	}

	// Parse info (simplified - just get basic stats)
	dbSize, err := c.client.DBSize(ctx).Result()
	if err != nil {
		dbSize = 0
	}

	hits := c.hits.Load()
	misses := c.misses.Load()

	var hitRate float64
	if total := hits + misses; total > 0 {
		hitRate = float64(hits) / float64(total)
	}

	// Note: Redis doesn't easily provide total size, so we set it to 0
	// In a production system, you might track this separately
	return &domain.CacheStats{
		Hits:    int64(hits),
		Misses:  int64(misses),
		Size:    0, // Not easily available from Redis
		Items:   dbSize,
		HitRate: hitRate,
	}, nil
}

// Close closes the Redis connection
func (c *RedisCache) Close() error {
	return c.client.Close()
}
