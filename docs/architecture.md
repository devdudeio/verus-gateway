# Verus Gateway - Architecture Documentation

## Overview

The Verus Gateway is a high-performance HTTP gateway service that provides REST API access to files stored on the Verus blockchain. Built with Go, it emphasizes clean architecture, performance, observability, and production readiness.

## Table of Contents

- [System Architecture](#system-architecture)
- [Component Overview](#component-overview)
- [Request Lifecycle](#request-lifecycle)
- [Design Decisions](#design-decisions)
- [Data Flow](#data-flow)
- [Caching Strategy](#caching-strategy)
- [Error Handling](#error-handling)
- [Observability](#observability)
- [Scalability](#scalability)
- [Security Considerations](#security-considerations)

## System Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Clients                                  │
│  (Web Browsers, Mobile Apps, CDN, External Services)           │
└────────────────────────┬────────────────────────────────────────┘
                         │ HTTP/HTTPS
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Reverse Proxy (Optional)                      │
│           (nginx, Caddy, Traefik, Kubernetes Ingress)           │
│               - TLS termination                                  │
│               - Rate limiting                                    │
│               - Load balancing                                   │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Verus Gateway                               │
│                                                                  │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │              HTTP Server Layer (Chi Router)                │ │
│  │  - Routing                                                 │ │
│  │  - Middleware Pipeline                                     │ │
│  └────────────────────┬───────────────────────────────────────┘ │
│                       │                                          │
│  ┌────────────────────▼───────────────────────────────────────┐ │
│  │                  Middleware Layer                          │ │
│  │  - Request ID Generation                                   │ │
│  │  - Structured Logging                                      │ │
│  │  - Panic Recovery                                          │ │
│  │  - Metrics Collection                                      │ │
│  │  - Security Headers                                        │ │
│  │  - CORS Handling                                           │ │
│  └────────────────────┬───────────────────────────────────────┘ │
│                       │                                          │
│  ┌────────────────────▼───────────────────────────────────────┐ │
│  │                   Handler Layer                            │ │
│  │  - Request Validation                                      │ │
│  │  - Parameter Extraction                                    │ │
│  │  - Response Formatting                                     │ │
│  └────────────────────┬───────────────────────────────────────┘ │
│                       │                                          │
│  ┌────────────────────▼───────────────────────────────────────┐ │
│  │                   Service Layer                            │ │
│  │  - Business Logic                                          │ │
│  │  - File Retrieval Orchestration                           │ │
│  │  - Decryption                                              │ │
│  │  - File Type Detection                                     │ │
│  └────────┬──────────────────────────┬────────────────────────┘ │
│           │                          │                           │
│  ┌────────▼─────────┐      ┌────────▼──────────┐               │
│  │  Cache Layer     │      │  Chain Manager    │               │
│  │  - Filesystem    │      │  - Multi-chain    │               │
│  │  - Redis         │      │  - Routing        │               │
│  │  - LRU Eviction  │      │  - Health Check   │               │
│  └──────────────────┘      └────────┬──────────┘               │
│                                      │                           │
│                             ┌────────▼──────────┐               │
│                             │  Verus RPC Client │               │
│                             │  - HTTP Client    │               │
│                             │  - Retry Logic    │               │
│                             │  - Timeout        │               │
│                             └────────┬──────────┘               │
└──────────────────────────────────────┼──────────────────────────┘
                                       │ JSON-RPC
                                       ▼
                        ┌──────────────────────────┐
                        │     Verus Nodes          │
                        │  - vrsctest (Testnet)    │
                        │  - vrsc (Mainnet)        │
                        │  - PBaaS chains          │
                        └──────────────────────────┘
```

### C4 Model - Container Diagram

```
                         ┌──────────────┐
                         │   Browser    │
                         │   Client     │
                         └──────┬───────┘
                                │ HTTPS
                                ▼
┌────────────────────────────────────────────────────────────────┐
│                        API Gateway                              │
│                                                                 │
│  ┌───────────────┐     ┌──────────────┐     ┌───────────────┐ │
│  │  HTTP Server  │────▶│   Service    │────▶│ Chain Manager │ │
│  │  (Chi)        │     │   Layer      │     │               │ │
│  └───────────────┘     └──────┬───────┘     └───────┬───────┘ │
│                               │                      │          │
│                               ▼                      ▼          │
│                        ┌──────────────┐     ┌───────────────┐ │
│                        │Cache Manager │     │  RPC Client   │ │
│                        │              │     │               │ │
│                        └──────┬───────┘     └───────┬───────┘ │
└───────────────────────────────┼─────────────────────┼─────────┘
                                │                      │
                                ▼                      ▼
                         ┌──────────────┐     ┌───────────────┐
                         │    Redis     │     │  Verus Node   │
                         │   (Cache)    │     │   (RPC API)   │
                         └──────────────┘     └───────────────┘
```

## Component Overview

### 1. HTTP Server Layer (`internal/http/server`)

**Purpose**: Entry point for all HTTP requests.

**Responsibilities**:
- HTTP server configuration and lifecycle management
- Router setup (Chi)
- Middleware pipeline registration
- Graceful shutdown handling

**Key Technologies**:
- `github.com/go-chi/chi/v5` - HTTP router with middleware support
- Go's `net/http` - Standard HTTP server

**Configuration**:
```go
type ServerConfig struct {
    Host         string        // Bind address
    Port         int           // Listen port
    ReadTimeout  time.Duration // Request read timeout
    WriteTimeout time.Duration // Response write timeout
    IdleTimeout  time.Duration // Keep-alive timeout
}
```

### 2. Middleware Layer (`internal/http/middleware`)

**Purpose**: Cross-cutting concerns applied to all requests.

**Middleware Stack** (in order):
1. **RequestID** - Generates unique request identifier
2. **Logger** - Structured request/response logging
3. **Recoverer** - Panic recovery with error reporting
4. **Metrics** - Prometheus metrics collection
5. **SecurityHeaders** - Security HTTP headers
6. **CORS** - Cross-Origin Resource Sharing

**Key Features**:
- Request context enrichment
- Automatic metric recording
- Panic-safe with stack traces
- Configurable CORS policies

### 3. Handler Layer (`internal/http/handler`)

**Purpose**: HTTP request handling and response formatting.

**Handlers**:
- `FileHandler` - File retrieval with chain specification
  - `GetFile` - Unified file retrieval (TXID or filename)
  - `HeadFile` - File metadata headers
  - `GetMeta` - File metadata JSON
- `AdminHandler` - Health checks, metrics, and cache management

**Responsibilities**:
- Request validation
- Parameter extraction (path, query)
- Service layer invocation
- HTTP response formatting
- Error translation to HTTP status codes

### 4. Service Layer (`internal/service`)

**Purpose**: Business logic orchestration.

**File Service** (`FileService`):
```go
type FileService interface {
    GetFile(ctx context.Context, chainID, txid string, evk *string) (*File, error)
    GetMetadata(ctx context.Context, chainID, txid string, evk *string) (*FileMetadata, error)
}
```

**Operations**:
1. Cache lookup (if enabled)
2. Chain resolution
3. RPC call to Verus node
4. Decryption (if EVK provided)
5. File type detection
6. Cache storage
7. Response preparation

**Design Pattern**: Dependency injection with interfaces for testability.

### 5. Cache Layer (`internal/cache`)

**Purpose**: High-performance file caching.

**Interface**:
```go
type Cache interface {
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
    Clear(ctx context.Context) error
    Stats(ctx context.Context) (*Stats, error)
}
```

**Implementations**:
- **Filesystem Cache** (`FilesystemCache`)
  - LRU eviction
  - Configurable max size
  - Background cleanup
  - Metadata tracking (`.meta` files)

- **Redis Cache** (`RedisCache`)
  - Distributed caching
  - Automatic TTL expiration
  - Connection pooling
  - High availability support

**Cache Key Strategy**:
```
{chainID}:{txid}:{evk_hash}
```

### 6. Chain Manager (`internal/chain`)

**Purpose**: Multi-chain configuration and routing.

**Responsibilities**:
- Chain configuration management
- Default chain resolution
- Chain health monitoring
- RPC client provisioning

**Configuration**:
```yaml
chains:
  default: vrsctest
  chains:
    vrsctest:
      name: "Verus Testnet"
      enabled: true
      rpc_url: "http://localhost:18843"
      rpc_user: "user"
      rpc_password: "password"
      rpc_timeout: 30s
      max_retries: 3
      retry_delay: 500ms
```

### 7. Verus RPC Client (`pkg/verusrpc`)

**Purpose**: Communication with Verus blockchain nodes.

**Key Methods**:
```go
type Client interface {
    GetRawTransaction(ctx context.Context, txid string) (*RawTransaction, error)
    GetTransaction(ctx context.Context, txid string) (*Transaction, error)
    GetBlockchainInfo(ctx context.Context) (*BlockchainInfo, error)
    ZViewTransaction(ctx context.Context, txid, evk string) (*ZTransaction, error)
}
```

**Features**:
- HTTP/JSON-RPC communication
- Automatic retry with exponential backoff
- Configurable timeouts
- Error classification (temporary vs permanent)

### 8. Crypto Layer (`internal/crypto`)

**Purpose**: Encrypted file decryption.

**Decryption Process**:
1. Parse viewing key (EVK)
2. Call `z_viewtransaction` RPC
3. Extract decrypted vdata
4. Decode base64 content
5. Return plaintext

**Supported Encryption**: Sapling shielded transactions

### 9. Storage Layer (`internal/storage`)

**Purpose**: File type detection and metadata extraction.

**Detector** (`Detector`):
```go
type Detector interface {
    DetectMIME(data []byte, filename string) string
    DetectExtension(mime string) string
    IsGzipCompressed(data []byte) bool
    IsTextLike(mime string) bool
}
```

**Detection Strategy**:
1. Check magic bytes (first 512 bytes)
2. Use Go's `http.DetectContentType`
3. Fallback to file extension mapping
4. Default to `application/octet-stream`

**Supported Types**:
- Images: JPEG, PNG, GIF, BMP, WebP, SVG
- Videos: MP4, WebM, AVI, MOV, MKV
- Audio: MP3, OGG, WAV, FLAC
- Documents: PDF, Office files, text formats
- Archives: ZIP, TAR, GZIP, BZIP2
- Code: JSON, XML, HTML, CSS, JavaScript

### 10. Observability Layer

**Logger** (`internal/observability/logger`):
- Structured logging with Zerolog
- JSON and text formats
- Context-aware logging
- Sensitive data masking
- Log levels: debug, info, warn, error

**Metrics** (`internal/observability/metrics`):
- Prometheus exposition format
- Request counters and histograms
- Cache hit/miss tracking
- RPC call metrics
- Custom business metrics

**Key Metrics**:
```
verus_gateway_http_requests_total
verus_gateway_http_request_duration_seconds
verus_gateway_cache_hits_total
verus_gateway_cache_misses_total
verus_gateway_rpc_requests_total
verus_gateway_files_served_total
```

## Request Lifecycle

### Example: GET /c/{chain}/file/{txid}?evk={viewing_key}

**Note**: All API endpoints require chain specification via the `/c/{chain}/` prefix.

```
1. Request arrives at HTTP server
   │
   ▼
2. Middleware Pipeline
   ├─ RequestID: Generate UUID (e.g., 550e8400-e29b-41d4-a716-446655440000)
   ├─ Logger: Log request start
   ├─ Recoverer: Wrap in panic handler
   ├─ Metrics: Start timer
   ├─ SecurityHeaders: Add security headers
   └─ CORS: Validate origin
   │
   ▼
3. Router matches /c/{chain}/file/{txid}
   │
   ▼
4. FileHandler.GetFile()
   ├─ Extract chain from path
   ├─ Extract txid_or_filename from path
   ├─ Extract evk from query
   ├─ Determine if path param is TXID (64 hex) or filename
   ├─ Validate parameters
   └─ Call service.GetFile(ctx, &FileRequest{...})
   │
   ▼
5. FileService.GetFile()
   ├─ Generate cache key: "{chainID}:{txid}:{evk_hash}"
   ├─ Check cache
   │  ├─ HIT: Return cached file ✓
   │  └─ MISS: Continue to RPC
   ├─ Get chain configuration
   ├─ Call RPC client
   │  ├─ If evk: ZViewTransaction()
   │  └─ Else: GetRawTransaction()
   ├─ Decrypt if needed
   ├─ Detect file type
   ├─ Store in cache
   └─ Return file data
   │
   ▼
6. Handler formats response
   ├─ Set Content-Type header
   ├─ Set Content-Disposition header
   ├─ Set Content-Length header
   ├─ Set X-Request-ID header
   ├─ Set X-Cache-Status header
   └─ Write body
   │
   ▼
7. Middleware cleanup
   ├─ Metrics: Record duration and status
   ├─ Logger: Log request completion
   └─ Response sent
```

**Timing Example**:
- Cache HIT: ~5-20ms
- Cache MISS (Verus RPC): ~200-500ms
- Cache MISS + Decryption: ~300-700ms

## Design Decisions

### 1. Clean Architecture

**Decision**: Separate concerns into layers with clear dependencies.

**Rationale**:
- **Testability**: Each layer can be tested independently
- **Maintainability**: Changes are localized
- **Flexibility**: Easy to swap implementations (e.g., cache backends)
- **Clarity**: Clear responsibility boundaries

**Dependency Rule**: Inner layers don't depend on outer layers.
```
Handlers → Services → Domain
   ↓          ↓
Middleware  Cache/Chain/RPC
```

### 2. Interface-Based Design

**Decision**: Use interfaces for all major components.

**Rationale**:
- Enables dependency injection
- Facilitates testing with mocks
- Allows multiple implementations (e.g., filesystem vs Redis cache)
- Supports future extensibility

**Example**:
```go
// Domain interface
type Cache interface {
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
}

// Implementations
type FilesystemCache struct { ... }
type RedisCache struct { ... }
```

### 3. Context Propagation

**Decision**: Pass `context.Context` as first parameter to all operations.

**Rationale**:
- Request cancellation support
- Timeout enforcement
- Request-scoped values (request ID, logger)
- Distributed tracing support (future)

**Pattern**:
```go
func (s *FileService) GetFile(ctx context.Context, chainID, txid string, evk *string) (*File, error) {
    logger := logger.FromContext(ctx)
    // ...
}
```

### 4. Error Handling Strategy

**Decision**: Use typed errors with error wrapping.

**Rationale**:
- Preserve error context
- Enable error classification
- Support debugging with stack traces
- Consistent error responses

**Pattern**:
```go
var (
    ErrNotFound = errors.New("file not found")
    ErrInvalidTxid = errors.New("invalid transaction ID")
    ErrDecryptionFailed = errors.New("decryption failed")
)

// Usage
if err != nil {
    return fmt.Errorf("failed to get file from RPC: %w", err)
}
```

**HTTP Mapping**:
```go
switch {
case errors.Is(err, ErrNotFound):
    http.Error(w, "file not found", http.StatusNotFound)
case errors.Is(err, ErrInvalidTxid):
    http.Error(w, "invalid txid", http.StatusBadRequest)
default:
    http.Error(w, "internal error", http.StatusInternalServerError)
}
```

### 5. Configuration Management

**Decision**: YAML config files with environment variable overrides.

**Rationale**:
- Human-readable configuration
- Environment-specific overrides (dev/staging/prod)
- 12-factor app compliance
- Kubernetes/Docker friendly

**Hierarchy**:
1. Default values
2. Config file (`config.yaml`)
3. Environment variables (`VERUS_GATEWAY_*`)
4. Command-line flags

### 6. Caching Strategy

**Decision**: Pluggable cache with LRU eviction and TTL.

**Rationale**:
- Reduce Verus node load
- Sub-100ms response times
- Support both single-node (filesystem) and distributed (Redis)
- Configurable TTL for different use cases

**Cache Invalidation**: TTL-based (no active invalidation)

**Why no invalidation?**
- Blockchain data is immutable
- TXIDs never change
- Simple and reliable

### 7. Observability First

**Decision**: Built-in logging, metrics, and health checks.

**Rationale**:
- Production readiness
- Operational visibility
- Performance monitoring
- Incident debugging

**Tools**:
- Logging: Zerolog (structured, fast)
- Metrics: Prometheus (industry standard)
- Tracing: Request IDs (with future support for OpenTelemetry)

## Data Flow

### File Retrieval Flow

```
┌─────────┐
│ Client  │
└────┬────┘
     │ GET /c/{chain}/file/{txid}
     ▼
┌─────────────────┐
│   HTTP Server   │
└────┬────────────┘
     │
     ▼
┌─────────────────┐
│   Middleware    │ ─── Request ID, Logging, Metrics
└────┬────────────┘
     │
     ▼
┌─────────────────┐
│    Handler      │ ─── Validation, Parameter Extraction
└────┬────────────┘
     │
     ▼
┌─────────────────┐
│  File Service   │
└────┬────────────┘
     │
     ├──────────────────┐
     │                  │
     ▼                  ▼
┌─────────┐      ┌──────────────┐
│  Cache  │      │Chain Manager │
└────┬────┘      └──────┬───────┘
     │                  │
     │ HIT              │ MISS
     │                  ▼
     │           ┌──────────────┐
     │           │  RPC Client  │
     │           └──────┬───────┘
     │                  │
     │                  ▼
     │           ┌──────────────┐
     │           │ Verus Node   │
     │           └──────┬───────┘
     │                  │
     │                  ▼
     │           ┌──────────────┐
     │           │   Decrypt    │ (if EVK provided)
     │           └──────┬───────┘
     │                  │
     │                  ▼
     │           ┌──────────────┐
     │           │File Detection│
     │           └──────┬───────┘
     │                  │
     │◄─────────────────┘
     │ Store in cache
     ▼
┌─────────────────┐
│  Response       │ ─── Headers, Body
└─────────────────┘
```

### Cache Storage Flow

```
┌──────────────┐
│  File Data   │
└──────┬───────┘
       │
       ▼
┌──────────────────────────────┐
│  Generate Cache Key          │
│  {chain}:{txid}:{evk_hash}   │
└──────┬───────────────────────┘
       │
       ▼
┌──────────────────────────────┐
│  Serialize Metadata          │
│  - filename                  │
│  - content_type              │
│  - size                      │
└──────┬───────────────────────┘
       │
       ├─────────────────┬──────────────────┐
       │                 │                  │
       ▼                 ▼                  ▼
┌────────────┐    ┌────────────┐    ┌────────────┐
│ Filesystem │    │   Redis    │    │   Future   │
│   Cache    │    │   Cache    │    │  (Memcached│
└────────────┘    └────────────┘    └────────────┘
  • File: key      • SET key
  • Meta: key.meta   value
  • LRU tracking     EX ttl
```

## Caching Strategy

### Cache Key Design

**Format**: `{chainID}:{txid}:{evk_hash}`

**Examples**:
- Public file: `vrsctest:abc123...:none`
- Encrypted file: `vrsctest:abc123...:sha256(evk)`

**Why hash EVK?**
- Viewing keys can be 100+ characters
- Hashing ensures fixed-length keys
- Preserves uniqueness

### Cache Backends

#### Filesystem Cache

**Use Case**: Single-instance deployments, local development

**Structure**:
```
cache/
├── vrsctest_abc123..._none          # File data
├── vrsctest_abc123..._none.meta     # Metadata
├── vrsctest_def456..._evkhash       # Encrypted file
└── vrsctest_def456..._evkhash.meta
```

**Features**:
- LRU eviction based on access time
- Configurable max size (e.g., 10GB)
- Background cleanup goroutine
- Atomic writes (temp file + rename)

**Configuration**:
```yaml
cache:
  type: filesystem
  dir: ./cache
  ttl: 24h
  max_size: 10737418240  # 10GB
  cleanup_interval: 1h
```

#### Redis Cache

**Use Case**: Multi-instance deployments, horizontal scaling

**Features**:
- Distributed caching
- Automatic TTL expiration
- Connection pooling
- Cluster support

**Configuration**:
```yaml
cache:
  type: redis
  ttl: 24h
  redis:
    addresses:
      - redis-node1:6379
      - redis-node2:6379
    password: "secret"
    db: 0
    pool_size: 10
    timeout: 5s
```

### Cache TTL Strategy

**Default**: 24 hours

**Rationale**:
- Blockchain data is immutable
- Balance between freshness and storage
- Long enough for repeated access
- Short enough to avoid unbounded growth

**Overrides**:
```yaml
cache:
  ttl: 168h  # 7 days for production
```

## Error Handling

### Error Types

1. **Client Errors** (4xx)
   - Invalid TXID format
   - Missing required parameters
   - Chain not found
   - File not found

2. **Server Errors** (5xx)
   - RPC connection failure
   - Cache unavailable
   - Decryption failure
   - Internal errors

### Error Response Format

```json
{
  "error": "error_code",
  "message": "Human-readable description",
  "request_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

### Error Handling Pattern

```go
// Service layer
func (s *FileService) GetFile(ctx context.Context, chainID, txid string, evk *string) (*File, error) {
    // ... operation ...
    if err != nil {
        return nil, fmt.Errorf("failed to retrieve file: %w", err)
    }
    return file, nil
}

// Handler layer
func (h *FileHandler) Get(w http.ResponseWriter, r *http.Request) {
    file, err := h.service.GetFile(r.Context(), chainID, txid, evk)
    if err != nil {
        h.handleError(w, r, err)
        return
    }
    // ... write response ...
}

func (h *FileHandler) handleError(w http.ResponseWriter, r *http.Request, err error) {
    reqID := middleware.GetRequestID(r.Context())

    switch {
    case errors.Is(err, domain.ErrNotFound):
        h.errorResponse(w, r, http.StatusNotFound, "file_not_found", err.Error(), reqID)
    case errors.Is(err, domain.ErrInvalidTxid):
        h.errorResponse(w, r, http.StatusBadRequest, "invalid_request", err.Error(), reqID)
    default:
        logger.FromContext(r.Context()).Error().Err(err).Msg("internal error")
        h.errorResponse(w, r, http.StatusInternalServerError, "internal_error", "An internal error occurred", reqID)
    }
}
```

## Observability

### Logging Strategy

**Structured Logging** with Zerolog:
```json
{
  "level": "info",
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "method": "GET",
  "path": "/c/vrsctest/file/abc123",
  "chain": "vrsctest",
  "status": 200,
  "duration_ms": 45,
  "cache_status": "HIT",
  "timestamp": "2025-10-10T10:15:30Z"
}
```

**Log Levels**:
- **DEBUG**: Detailed debugging information
- **INFO**: Normal operations, request/response
- **WARN**: Recoverable errors, degraded state
- **ERROR**: Errors requiring attention

**Sensitive Data**: Automatically masked in logs (viewing keys, RPC passwords)

### Metrics Collection

**Prometheus Metrics**:

1. **HTTP Request Metrics**
   ```
   verus_gateway_http_requests_total{method="GET",path="/c/{chain}/file",status="200",chain="vrsctest"}
   verus_gateway_http_request_duration_seconds{method="GET",path="/c/{chain}/file",chain="vrsctest"}
   ```

2. **Cache Metrics**
   ```
   verus_gateway_cache_hits_total{chain="vrsctest"}
   verus_gateway_cache_misses_total{chain="vrsctest"}
   verus_gateway_cache_size_bytes
   ```

3. **RPC Metrics**
   ```
   verus_gateway_rpc_requests_total{chain="vrsctest",method="getrawtransaction"}
   verus_gateway_rpc_errors_total{chain="vrsctest"}
   ```

4. **Business Metrics**
   ```
   verus_gateway_files_served_total{chain="vrsctest",content_type="image/jpeg"}
   ```

### Health Checks

**Liveness Probe** (`/health`):
- Returns 200 if the service is running
- Doesn't check dependencies
- Used by Kubernetes liveness probe

**Readiness Probe** (`/ready`):
- Returns 200 if service can serve traffic
- Checks Verus node connectivity
- Used by Kubernetes readiness probe

## Scalability

### Horizontal Scaling

**Stateless Design**:
- No in-memory state (except metrics)
- All state in cache/database
- Safe to run multiple instances

**Load Balancing**:
```
       ┌──────────┐
       │   LB     │
       └────┬─────┘
            │
    ┌───────┼───────┐
    ▼       ▼       ▼
┌────────┐ ┌────────┐ ┌────────┐
│Gateway1│ │Gateway2│ │Gateway3│
└───┬────┘ └───┬────┘ └───┬────┘
    │          │          │
    └──────────┼──────────┘
               ▼
        ┌──────────────┐
        │    Redis     │
        │   (Cache)    │
        └──────────────┘
```

### Performance Optimizations

1. **Connection Pooling**
   - HTTP client reuse for RPC calls
   - Redis connection pooling

2. **Caching**
   - Filesystem or Redis cache
   - Sub-100ms cache hits

3. **Async Operations**
   - Non-blocking cache cleanup
   - Background metrics collection

4. **Resource Limits**
   - Configurable timeouts
   - Request size limits
   - Cache size limits

### Capacity Planning

**Typical Performance**:
- Cache HIT: 5-20ms, ~1000-5000 req/s per instance
- Cache MISS: 200-500ms, ~50-200 req/s per instance
- Cache hit ratio: 80-95% (typical)

**Scaling Recommendations**:
- **< 1000 req/s**: Single instance + filesystem cache
- **1000-10000 req/s**: 3-5 instances + Redis cache
- **> 10000 req/s**: 10+ instances + Redis cluster + CDN

## Security Considerations

### Input Validation

- TXID format validation (64-char hex)
- Chain ID whitelist
- Request size limits
- Path traversal prevention

### Authentication & Authorization

**Current**: No authentication (public gateway)

**Future Options**:
- API key authentication
- JWT tokens
- Rate limiting per key

### Sensitive Data Handling

1. **Viewing Keys**
   - Never logged
   - Not stored in cache keys (hashed)
   - Not included in metrics

2. **RPC Credentials**
   - Stored in config file (file permissions)
   - Environment variables (preferred)
   - Never logged or exposed

3. **Security Headers**
   ```
   X-Content-Type-Options: nosniff
   X-Frame-Options: DENY
   X-XSS-Protection: 1; mode=block
   Strict-Transport-Security: max-age=31536000
   ```

### HTTPS/TLS

**Production Requirement**: Always use HTTPS

**Options**:
1. Reverse proxy (nginx, Caddy) - **Recommended**
2. Let's Encrypt automatic certificates
3. Cloud load balancer TLS termination

### Rate Limiting

**Implemented At**: Reverse proxy level

**Example** (nginx):
```nginx
limit_req_zone $binary_remote_addr zone=gateway:10m rate=10r/s;
```

---

For deployment configurations and operational details, see:
- [Deployment Guide](deployment.md)
- [Monitoring Setup](monitoring.md)
- [API Documentation](openapi.yaml)
