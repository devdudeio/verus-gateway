package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/devdudeio/verus-gateway/internal/chain"
	"github.com/devdudeio/verus-gateway/internal/domain"
)

// Mock cache implementation
type mockCache struct {
	getFunc    func(ctx context.Context, key string) (*domain.File, error)
	setFunc    func(ctx context.Context, key string, file *domain.File, ttl time.Duration) error
	deleteFunc func(ctx context.Context, key string) error
	clearFunc  func(ctx context.Context) error
	statsFunc  func(ctx context.Context) (*domain.CacheStats, error)
	closeFunc  func() error
}

func (m *mockCache) Get(ctx context.Context, key string) (*domain.File, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, key)
	}
	return nil, errors.New("cache miss")
}

func (m *mockCache) Set(ctx context.Context, key string, file *domain.File, ttl time.Duration) error {
	if m.setFunc != nil {
		return m.setFunc(ctx, key, file, ttl)
	}
	return nil
}

func (m *mockCache) Delete(ctx context.Context, key string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, key)
	}
	return nil
}

func (m *mockCache) Clear(ctx context.Context) error {
	if m.clearFunc != nil {
		return m.clearFunc(ctx)
	}
	return nil
}

func (m *mockCache) Stats(ctx context.Context) (*domain.CacheStats, error) {
	if m.statsFunc != nil {
		return m.statsFunc(ctx)
	}
	return &domain.CacheStats{}, nil
}

func (m *mockCache) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

// Helper to create test service
func newTestFileService(cache domain.Cache, chainMgr *chain.Manager) *FileService {
	if chainMgr == nil {
		// Create empty chain manager for tests that don't need it
		chainMgr = &chain.Manager{}
	}
	return NewFileService(chainMgr, cache)
}

func TestNewFileService(t *testing.T) {
	cache := &mockCache{}
	chainMgr := &chain.Manager{}

	service := NewFileService(chainMgr, cache)

	if service == nil {
		t.Fatal("NewFileService returned nil")
	}

	if service.cache == nil {
		t.Error("cache not set correctly")
	}

	if service.chainManager != chainMgr {
		t.Error("chainManager not set correctly")
	}

	if service.decompressor == nil {
		t.Error("decompressor not initialized")
	}

	if service.detector == nil {
		t.Error("detector not initialized")
	}
}

func TestNewFileService_NilCache(t *testing.T) {
	chainMgr := &chain.Manager{}

	service := NewFileService(chainMgr, nil)

	if service == nil {
		t.Fatal("NewFileService returned nil")
	}

	if service.cache != nil {
		t.Error("expected cache to be nil")
	}
}

func TestClearCache(t *testing.T) {
	tests := []struct {
		name      string
		cache     domain.Cache
		wantError bool
	}{
		{
			name: "successful cache clear",
			cache: &mockCache{
				clearFunc: func(ctx context.Context) error {
					return nil
				},
			},
			wantError: false,
		},
		{
			name: "cache clear error",
			cache: &mockCache{
				clearFunc: func(ctx context.Context) error {
					return errors.New("cache error")
				},
			},
			wantError: true,
		},
		{
			name:      "no cache configured",
			cache:     nil,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := newTestFileService(tt.cache, nil)
			err := service.ClearCache(context.Background())

			if tt.wantError && err == nil {
				t.Error("expected error, got nil")
			}

			if !tt.wantError && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}

func TestGetCacheStats(t *testing.T) {
	tests := []struct {
		name      string
		cache     domain.Cache
		wantError bool
	}{
		{
			name: "successful stats retrieval",
			cache: &mockCache{
				statsFunc: func(ctx context.Context) (*domain.CacheStats, error) {
					return &domain.CacheStats{
						Hits:   100,
						Misses: 20,
						Size:   1024,
					}, nil
				},
			},
			wantError: false,
		},
		{
			name: "stats error",
			cache: &mockCache{
				statsFunc: func(ctx context.Context) (*domain.CacheStats, error) {
					return nil, errors.New("stats error")
				},
			},
			wantError: true,
		},
		{
			name:      "no cache configured",
			cache:     nil,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := newTestFileService(tt.cache, nil)
			stats, err := service.GetCacheStats(context.Background())

			if tt.wantError && err == nil {
				t.Error("expected error, got nil")
			}

			if !tt.wantError && err != nil {
				t.Errorf("expected no error, got %v", err)
			}

			if !tt.wantError && stats == nil {
				t.Error("expected stats, got nil")
			}
		})
	}
}

func TestDeleteFromCache(t *testing.T) {
	tests := []struct {
		name      string
		cache     domain.Cache
		cacheKey  string
		wantError bool
	}{
		{
			name: "successful deletion",
			cache: &mockCache{
				deleteFunc: func(ctx context.Context, key string) error {
					return nil
				},
			},
			cacheKey:  "test-key",
			wantError: false,
		},
		{
			name: "deletion error",
			cache: &mockCache{
				deleteFunc: func(ctx context.Context, key string) error {
					return errors.New("delete error")
				},
			},
			cacheKey:  "test-key",
			wantError: true,
		},
		{
			name:      "no cache configured",
			cache:     nil,
			cacheKey:  "test-key",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := newTestFileService(tt.cache, nil)
			err := service.DeleteFromCache(context.Background(), tt.cacheKey)

			if tt.wantError && err == nil {
				t.Error("expected error, got nil")
			}

			if !tt.wantError && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}

func TestGetFile_InvalidRequest(t *testing.T) {
	service := newTestFileService(nil, nil)

	tests := []struct {
		name string
		req  *domain.FileRequest
	}{
		{
			name: "empty TXID",
			req: &domain.FileRequest{
				ChainID: "vrsctest",
				TXID:    "",
			},
		},
		{
			name: "invalid TXID length",
			req: &domain.FileRequest{
				ChainID: "vrsctest",
				TXID:    "short",
			},
		},
		{
			name: "invalid TXID format",
			req: &domain.FileRequest{
				ChainID: "vrsctest",
				TXID:    "zzzz456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.GetFile(context.Background(), tt.req)
			if err == nil {
				t.Error("expected validation error, got nil")
			}
		})
	}
}

func TestGetMetadata_InvalidRequest(t *testing.T) {
	service := newTestFileService(nil, nil)

	req := &domain.FileRequest{
		ChainID: "vrsctest",
		TXID:    "",
	}

	_, err := service.GetMetadata(context.Background(), req)
	if err == nil {
		t.Error("expected validation error, got nil")
	}
}
