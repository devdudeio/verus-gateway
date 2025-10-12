package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/devdudeio/verus-gateway/internal/chain"
	"github.com/devdudeio/verus-gateway/internal/observability/metrics"
	"github.com/devdudeio/verus-gateway/internal/service"
)

// AdminHandler handles admin-related HTTP requests
type AdminHandler struct {
	fileService  *service.FileService
	chainManager *chain.Manager
	metrics      *metrics.Metrics
	version      string
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(fileService *service.FileService, chainManager *chain.Manager, m *metrics.Metrics, version string) *AdminHandler {
	return &AdminHandler{
		fileService:  fileService,
		chainManager: chainManager,
		metrics:      m,
		version:      version,
	}
}

// Health handles GET /health (liveness probe)
func (h *AdminHandler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "healthy",
		"version": h.version,
	})
}

// Ready handles GET /ready (readiness probe)
func (h *AdminHandler) Ready(w http.ResponseWriter, r *http.Request) {
	// Create a separate context with 30s timeout for health checks
	// (independent of the HTTP request timeout)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Check if at least one chain is healthy
	results := h.chainManager.HealthCheckAll(ctx)

	healthy := false
	errors := make(map[string]string)
	for chainID, err := range results {
		if err == nil {
			healthy = true
		} else {
			errors[chainID] = err.Error()
		}
	}

	if !healthy {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "unhealthy",
			"reason": "no healthy chains available",
			"chains": errors,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ready",
		"version": h.version,
	})
}

// ListChains handles GET /chains
func (h *AdminHandler) ListChains(w http.ResponseWriter, r *http.Request) {
	chains := h.chainManager.ListChains()
	defaultChain := h.chainManager.GetDefaultChainID()

	chainList := make([]map[string]interface{}, 0, len(chains))
	for _, chainID := range chains {
		chainInfo, err := h.chainManager.GetChainInfo(chainID)
		if err != nil {
			continue
		}

		chainList = append(chainList, map[string]interface{}{
			"id":      chainInfo.ID,
			"name":    chainInfo.Name,
			"default": chainInfo.ID == defaultChain,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"chains": chainList,
		"count":  len(chainList),
	})
}

// GetCacheStats handles GET /admin/cache/stats
func (h *AdminHandler) GetCacheStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.fileService.GetCacheStats(r.Context())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "failed to get cache stats",
			"message": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(stats)
}

// ClearCache handles DELETE /admin/cache
func (h *AdminHandler) ClearCache(w http.ResponseWriter, r *http.Request) {
	if err := h.fileService.ClearCache(r.Context()); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "failed to clear cache",
			"message": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "cache cleared successfully",
	})
}

// DeleteCacheEntry handles DELETE /admin/cache/{key}
func (h *AdminHandler) DeleteCacheEntry(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")

	if err := h.fileService.DeleteFromCache(r.Context(), key); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "failed to delete cache entry",
			"message": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": fmt.Sprintf("cache entry %s deleted successfully", key),
	})
}

// PrometheusMetrics handles GET /metrics (Prometheus metrics endpoint)
func (h *AdminHandler) PrometheusMetrics(w http.ResponseWriter, r *http.Request) {
	// Update cache stats in metrics before serving
	if h.metrics != nil {
		stats, err := h.fileService.GetCacheStats(r.Context())
		if err == nil && stats != nil {
			h.metrics.UpdateCacheStats(stats.Size, stats.Items)
		}
	}

	// Serve Prometheus metrics
	promhttp.Handler().ServeHTTP(w, r)
}
