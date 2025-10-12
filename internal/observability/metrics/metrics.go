package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics
type Metrics struct {
	// HTTP Metrics
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec
	HTTPRequestSize     *prometheus.HistogramVec
	HTTPResponseSize    *prometheus.HistogramVec

	// Cache Metrics
	CacheHits       prometheus.Counter
	CacheMisses     prometheus.Counter
	CacheSize       prometheus.Gauge
	CacheItems      prometheus.Gauge
	CacheOperations *prometheus.CounterVec

	// RPC Metrics
	RPCRequestsTotal   *prometheus.CounterVec
	RPCRequestDuration *prometheus.HistogramVec
	RPCErrors          *prometheus.CounterVec

	// Business Metrics
	FilesServed        prometheus.Counter
	BytesTransferred   prometheus.Counter
	DecryptionsTotal   *prometheus.CounterVec
	DecompressionTotal *prometheus.CounterVec
}

// New creates and registers all Prometheus metrics
func New(namespace string) *Metrics {
	if namespace == "" {
		namespace = "verus_gateway"
	}

	m := &Metrics{
		// HTTP Metrics
		HTTPRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "http_requests_total",
				Help:      "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		HTTPRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_request_duration_seconds",
				Help:      "HTTP request latency in seconds",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"method", "path", "status"},
		),
		HTTPRequestSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_request_size_bytes",
				Help:      "HTTP request size in bytes",
				Buckets:   prometheus.ExponentialBuckets(100, 10, 8), // 100B to ~10MB
			},
			[]string{"method", "path"},
		),
		HTTPResponseSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_response_size_bytes",
				Help:      "HTTP response size in bytes",
				Buckets:   prometheus.ExponentialBuckets(100, 10, 8), // 100B to ~10MB
			},
			[]string{"method", "path", "status"},
		),

		// Cache Metrics
		CacheHits: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "cache_hits_total",
				Help:      "Total number of cache hits",
			},
		),
		CacheMisses: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "cache_misses_total",
				Help:      "Total number of cache misses",
			},
		),
		CacheSize: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "cache_size_bytes",
				Help:      "Current cache size in bytes",
			},
		),
		CacheItems: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "cache_items",
				Help:      "Current number of items in cache",
			},
		),
		CacheOperations: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "cache_operations_total",
				Help:      "Total number of cache operations",
			},
			[]string{"operation", "status"},
		),

		// RPC Metrics
		RPCRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "rpc_requests_total",
				Help:      "Total number of RPC requests",
			},
			[]string{"chain", "method", "status"},
		),
		RPCRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "rpc_request_duration_seconds",
				Help:      "RPC request latency in seconds",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"chain", "method"},
		),
		RPCErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "rpc_errors_total",
				Help:      "Total number of RPC errors",
			},
			[]string{"chain", "method", "error_type"},
		),

		// Business Metrics
		FilesServed: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "files_served_total",
				Help:      "Total number of files served",
			},
		),
		BytesTransferred: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "bytes_transferred_total",
				Help:      "Total number of bytes transferred",
			},
		),
		DecryptionsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "decryptions_total",
				Help:      "Total number of decryption operations",
			},
			[]string{"chain", "status"},
		),
		DecompressionTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "decompressions_total",
				Help:      "Total number of decompression operations",
			},
			[]string{"status"},
		),
	}

	return m
}

// RecordHTTPRequest records an HTTP request metric
func (m *Metrics) RecordHTTPRequest(method, path, status string, duration float64, requestSize, responseSize int64) {
	m.HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
	m.HTTPRequestDuration.WithLabelValues(method, path, status).Observe(duration)

	if requestSize > 0 {
		m.HTTPRequestSize.WithLabelValues(method, path).Observe(float64(requestSize))
	}
	if responseSize > 0 {
		m.HTTPResponseSize.WithLabelValues(method, path, status).Observe(float64(responseSize))
	}
}

// RecordCacheHit records a cache hit
func (m *Metrics) RecordCacheHit() {
	m.CacheHits.Inc()
	m.CacheOperations.WithLabelValues("get", "hit").Inc()
}

// RecordCacheMiss records a cache miss
func (m *Metrics) RecordCacheMiss() {
	m.CacheMisses.Inc()
	m.CacheOperations.WithLabelValues("get", "miss").Inc()
}

// UpdateCacheStats updates cache size and items metrics
func (m *Metrics) UpdateCacheStats(sizeBytes, items int64) {
	m.CacheSize.Set(float64(sizeBytes))
	m.CacheItems.Set(float64(items))
}

// RecordRPCRequest records an RPC request metric
func (m *Metrics) RecordRPCRequest(chain, method, status string, duration float64) {
	m.RPCRequestsTotal.WithLabelValues(chain, method, status).Inc()
	m.RPCRequestDuration.WithLabelValues(chain, method).Observe(duration)
}

// RecordRPCError records an RPC error
func (m *Metrics) RecordRPCError(chain, method, errorType string) {
	m.RPCErrors.WithLabelValues(chain, method, errorType).Inc()
}

// RecordFileServed records a file served
func (m *Metrics) RecordFileServed(sizeBytes int64) {
	m.FilesServed.Inc()
	m.BytesTransferred.Add(float64(sizeBytes))
}

// RecordDecryption records a decryption operation
func (m *Metrics) RecordDecryption(chain, status string) {
	m.DecryptionsTotal.WithLabelValues(chain, status).Inc()
}

// RecordDecompression records a decompression operation
func (m *Metrics) RecordDecompression(status string) {
	m.DecompressionTotal.WithLabelValues(status).Inc()
}
