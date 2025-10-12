# Verus Gateway

[![Go Version](https://img.shields.io/badge/Go-1.23%2B-blue.svg)](https://golang.org)
[![Test Coverage](https://img.shields.io/badge/coverage-54.8%25-brightgreen.svg)](https://github.com/devdudeio/verus-gateway)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

A high-performance, production-ready HTTP gateway for accessing files stored on the Verus blockchain. Built with Go for speed, reliability, and scalability.

## 🚀 Features

- **Multi-Chain Support**: VRSC mainnet, VRSCTEST, and PBaaS chains - all API calls require chain specification
- **High Performance**: Built-in caching (filesystem & Redis) for sub-100ms response times
- **Privacy First**: Full support for encrypted files with viewing keys (EVK)
- **Production Ready**: Comprehensive logging, metrics, and health checks
- **RESTful API**: Clean, intuitive API endpoints with mandatory chain routing
- **Observability**: Prometheus metrics, structured logging (JSON/text), request tracing
- **Flexible Deployment**: Docker, systemd, or standalone binary
- **Well Tested**: 54.8% overall coverage, 80-100% on core packages

## 📋 Table of Contents

- [Quick Start](#-quick-start)
- [Configuration](#-configuration)
- [API Documentation](#-api-documentation)
- [Deployment](#-deployment)
- [Development](#-development)
- [Architecture](#-architecture)
- [Contributing](#-contributing)

## ⚡ Quick Start

### Option 1: Using Docker (Recommended)

```bash
# Pull and run the gateway
docker run -p 8080:8080 \
  -v $(pwd)/config.yaml:/app/config.yaml \
  ghcr.io/devdudeio/verus-gateway:latest
```

### Option 2: Build from Source

**Prerequisites:**
- Go 1.23 or higher
- Access to a Verus node with RPC enabled

```bash
# Clone the repository
git clone https://github.com/devdudeio/verus-gateway.git
cd verus-gateway

# Copy example configuration
cp config.example.yaml config.yaml
# Edit config.yaml with your Verus RPC credentials

# Build the gateway
make build

# Run the gateway
./bin/verus-gateway -config config.yaml
```

The gateway will start on `http://localhost:8080` by default.

## ⚙️ Configuration

The gateway uses YAML configuration files with environment variable support.

### Minimal Configuration

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
```

### Full Configuration Example

See [`config.example.yaml`](config.example.yaml) for all available options including:
- Server settings (host, port, timeouts)
- Multiple chain configurations
- Cache settings (filesystem or Redis)
- Logging configuration
- CORS settings

**Important:** For production deployments with large files, increase timeouts:
```yaml
server:
  read_timeout: 60s   # Allow time for file decryption
  write_timeout: 120s # Allow time for sending large files
```

### Environment Variables

You can override any configuration using environment variables:

```bash
export VERUS_GATEWAY_SERVER_PORT=9090
export VERUS_GATEWAY_CHAINS_DEFAULT=vrsc
export VERUS_GATEWAY_CACHE_TYPE=redis
export VERUS_GATEWAY_CACHE_REDIS_ADDRESSES=localhost:6379
```

Or use a `.env` file:

```bash
cp .env.example .env
# Edit .env with your settings
```

## 📚 API Documentation

### Base URL

```
http://localhost:8080
```

### Important: Chain Specification Required

**All API endpoints require the chain to be specified in the URL path using `/c/{chain}/` prefix.**

Supported chains:
- `vrsc` - Verus mainnet
- `vrsctest` - Verus testnet
- Any configured PBaaS chain

### Migration from Old Gateway

If you're migrating from an older gateway version, note the URL format changes:

**Old format:**
```
/v2/file/document.pdf?txid=abc123...&evk=zxviews...
```

**New format:**
```
/c/vrsctest/file/document.pdf?txid=abc123...&evk=zxviews...
```

Key changes:
- Chain must be explicitly specified: `/c/{chain}/` prefix is now required
- The `/v2/` prefix has been removed
- TXID can be used directly in the path (if 64 hex chars) without requiring filename

### API Endpoints

#### Get File

```http
GET /c/{chain}/file/{txid_or_filename}?txid={txid}&evk={evk}
```

**Supports two retrieval modes:**
1. **By TXID**: If path parameter is a 64-character hex string, it's treated as a TXID
2. **By Filename**: Otherwise, it's treated as a filename (requires `txid` query parameter)

**Parameters:**
- `{chain}`: Chain identifier (e.g., `vrsc`, `vrsctest`)
- `{txid_or_filename}`: Either a TXID (64 hex chars) or filename
- `txid`: Transaction ID (query parameter, required for filename mode)
- `evk`: Viewing key for encrypted files (optional query parameter)

**Examples:**

```bash
# Get encrypted file by filename with TXID and viewing key (working example on vrsctest)
curl "http://localhost:8080/c/vrsctest/file/lee.gif?txid=004b2d1e74351bf361f2555e4254481a3aee9f5db173ff2eeff07e6ae540ba47&evk=zxviews1qdugfjmfqyqqpqxv03ees2eymyvvfa8uhhjcfkezhsleu9686l92w6cycx8jazta4metc3lx7jjly7um6vxujtzj2dt7xw8m7gd0suw56pshraqf34s3ltww9tvr049h4j78duw7w7gvkzfmwvk6k00zgpynq8pwr8h9wk0f47v5cjaczq9y3dndtcsntszt5rl2qsage9dc7ctuevhnvhynex44fnqy0wde3xppuzp2qfdg3tgnp2sn6pajxjfqy355eutvdgsl77sddcuep"

# Get file by TXID (64 hex characters)
curl "http://localhost:8080/c/vrsctest/file/004b2d1e74351bf361f2555e4254481a3aee9f5db173ff2eeff07e6ae540ba47"

# Get file by filename with TXID as query param
curl "http://localhost:8080/c/vrsctest/file/document.pdf?txid=abc123...&evk=zxviews..."

# Get file from vrsc mainnet
curl "http://localhost:8080/c/vrsc/file/image.jpg?txid=def456..."
```

#### Get File Metadata

```http
GET /c/{chain}/meta/{txid}?evk={evk}
```

**Parameters:**
- `{chain}`: Chain identifier (e.g., `vrsc`, `vrsctest`)
- `{txid}`: Transaction ID containing the file
- `evk`: Viewing key for encrypted files (optional query parameter)

**Example:**
```bash
curl "http://localhost:8080/c/vrsctest/meta/abc123def456..."
```

**Response:**
```json
{
  "txid": "abc123def456...",
  "chain": "vrsctest",
  "filename": "document.pdf",
  "size": 102400,
  "content_type": "application/pdf",
  "extension": ".pdf",
  "compressed": false
}
```

#### Head Request (Check File Existence)

```http
HEAD /c/{chain}/file/{txid}?evk={evk}
```

Returns file metadata in headers without downloading the content.

**Parameters:**
- `{chain}`: Chain identifier (e.g., `vrsc`, `vrsctest`)
- `{txid}`: Transaction ID containing the file
- `evk`: Viewing key for encrypted files (optional query parameter)

**Example:**
```bash
curl -I "http://localhost:8080/c/vrsctest/file/abc123def456..."
```

### Admin Endpoints

#### Health Check (Liveness)

```http
GET /health
```

Returns `200 OK` if the gateway is running.

#### Readiness Check

```http
GET /ready
```

Returns `200 OK` if the gateway can connect to at least one Verus node.

#### List Chains

```http
GET /chains
```

Returns configured blockchain networks.

#### Prometheus Metrics

```http
GET /metrics
```

Returns Prometheus-formatted metrics.

#### Cache Management (Admin)

```http
GET /admin/cache/stats        # Get cache statistics
DELETE /admin/cache            # Clear entire cache
DELETE /admin/cache/{key}      # Delete specific cache entry
```

### Path and Query Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `chain` | Path | Yes | Chain identifier (e.g., `vrsc`, `vrsctest`) |
| `txid_or_filename` | Path | Yes | Either TXID (64 hex chars) or filename |
| `txid` | Query | Conditional | Required when using filename in path |
| `evk` | Query | No | Viewing key for encrypted files |

### Response Headers

- `Content-Type`: Detected MIME type (e.g., `image/jpeg`, `application/pdf`)
- `Content-Disposition`: Suggested filename
- `Content-Length`: File size in bytes
- `X-Request-ID`: Unique request identifier for tracing
- `X-Cache-Status`: `HIT` or `MISS`

### Error Responses

All errors return JSON with the following structure:

```json
{
  "error": "error_code",
  "message": "Human-readable error message",
  "request_id": "uuid"
}
```

**Common Error Codes:**
- `invalid_request`: Missing or invalid parameters
- `file_not_found`: TXID not found on the blockchain
- `decryption_failed`: Invalid viewing key
- `chain_not_found`: Unknown blockchain network
- `internal_error`: Server error

## 🚀 Deployment

### Docker

```bash
# Build image
docker build -t verus-gateway .

# Run with config file
docker run -d \
  --name verus-gateway \
  -p 8080:8080 \
  -v $(pwd)/config.yaml:/app/config.yaml \
  verus-gateway
```

### Docker Compose

```bash
# Development (with auto-reload)
docker-compose up

# Production (with Redis cache)
docker-compose -f docker-compose.production.yml up -d
```

### Systemd Service

```bash
# Install binary
sudo cp verus-gateway /usr/local/bin/

# Install service file
sudo cp deployments/systemd/verus-gateway.service /etc/systemd/system/

# Start service
sudo systemctl enable verus-gateway
sudo systemctl start verus-gateway
```

### Kubernetes

See [`deployments/kubernetes/`](deployments/kubernetes/) for Helm charts and manifests.

## 🛠️ Development

### Prerequisites

- Go 1.23+
- Make
- Docker (optional)

### Build

```bash
# Build binary
make build

# Build for all platforms
make build-all

# Run tests
make test

# Run tests with coverage
make test-coverage

# Run linter
make lint

# Format code
make fmt
```

### Project Structure

```
verus-gateway/
├── cmd/gateway/              # Main application entry point
├── internal/
│   ├── cache/               # Cache implementations (filesystem, Redis)
│   ├── chain/               # Multi-chain manager
│   ├── config/              # Configuration loading and validation
│   ├── crypto/              # File decryption
│   ├── domain/              # Domain models and interfaces
│   ├── http/
│   │   ├── handler/        # HTTP request handlers
│   │   ├── middleware/     # HTTP middleware
│   │   └── server/         # HTTP server setup
│   ├── observability/
│   │   ├── logger/         # Structured logging
│   │   └── metrics/        # Prometheus metrics
│   ├── service/            # Business logic layer
│   └── storage/            # File type detection and processing
├── pkg/verusrpc/           # Verus RPC client
├── docs/                   # Documentation
└── deployments/            # Deployment configs
```

### Running Tests

```bash
# All tests
go test ./...

# With coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Specific package
go test -v ./internal/config

# With race detector
go test -race ./...
```

### Test Coverage

- **Overall**: 54.8% (+9.2% from v0.4.0)
- **Core Packages** (80-100%):
  - `internal/crypto`: 100%
  - `internal/observability/logger`: 100%
  - `internal/storage`: 93.4%
  - `internal/config`: 87.5%
  - `internal/domain`: 81.0%
  - `pkg/verusrpc`: 80.2%
  - `internal/chain`: 62.9%
  - `internal/http/handler`: 56.9%
  - `internal/cache`: 55.7%

## 🏗️ Architecture

```
┌─────────────┐
│   Client    │
│  (Browser)  │
└──────┬──────┘
       │ HTTP/REST
       ▼
┌─────────────────────────────────────┐
│      Verus Gateway (Go)              │
│  ┌───────────────────────────────┐  │
│  │  HTTP Server (Chi Router)     │  │
│  │  - Middleware (Auth, CORS)    │  │
│  │  - Metrics (Prometheus)       │  │
│  │  - Logging (Zerolog)          │  │
│  └────────────┬──────────────────┘  │
│               ▼                      │
│  ┌───────────────────────────────┐  │
│  │  File Service Layer           │  │
│  │  - Request validation         │  │
│  │  - Cache lookup               │  │
│  │  - Decryption                 │  │
│  │  - File type detection        │  │
│  └────────────┬──────────────────┘  │
│               ▼                      │
│  ┌───────────────────────────────┐  │
│  │  Cache Layer (Pluggable)      │  │
│  │  - Filesystem Cache           │  │
│  │  - Redis Cache                │  │
│  └────────────┬──────────────────┘  │
│               ▼                      │
│  ┌───────────────────────────────┐  │
│  │  Chain Manager                │  │
│  │  - Multi-chain routing        │  │
│  │  - Health monitoring          │  │
│  └────────────┬──────────────────┘  │
│               ▼                      │
│  ┌───────────────────────────────┐  │
│  │  Verus RPC Client             │  │
│  │  - Connection pooling         │  │
│  │  - Retry logic                │  │
│  └────────────┬──────────────────┘  │
└───────────────┼──────────────────────┘
                │ JSON-RPC
                ▼
        ┌───────────────┐
        │  Verus Node   │
        │  (vrsctest/   │
        │   VRSC)       │
        └───────────────┘
```

### Key Design Principles

- **Clean Architecture**: Separation of concerns with clear boundaries
- **Pluggable Components**: Easy to swap cache/storage implementations
- **Observability First**: Comprehensive logging, metrics, and tracing
- **Performance**: Built-in caching, connection pooling, async operations
- **Reliability**: Retry logic, health checks, graceful shutdown

## 🤝 Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Development Workflow

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests (`make test`)
5. Run linter (`make lint`)
6. Commit your changes (`git commit -m 'Add amazing feature'`)
7. Push to the branch (`git push origin feature/amazing-feature`)
8. Open a Pull Request

### Code Standards

- Follow Go best practices and idioms
- Write tests for new features (aim for 80%+ coverage)
- Use `gofmt` for formatting
- Add godoc comments for public APIs
- Update documentation for user-facing changes

## 📊 Monitoring

### Prometheus Metrics

The gateway exposes Prometheus metrics at `/metrics`:

- `verus_gateway_http_requests_total`: Total HTTP requests
- `verus_gateway_http_request_duration_seconds`: Request latency
- `verus_gateway_cache_hits_total`: Cache hit count
- `verus_gateway_cache_misses_total`: Cache miss count
- `verus_gateway_rpc_requests_total`: RPC call count
- `verus_gateway_files_served_total`: Files served count

See [`docs/monitoring.md`](docs/monitoring.md) for Grafana dashboard setup.

## 🔒 Security

- **HTTPS**: Always use HTTPS in production
- **RPC Credentials**: Store securely, never commit to git
- **Viewing Keys**: Treat as passwords, never log or expose
- **Rate Limiting**: Configure for public deployments
- **CORS**: Restrict to trusted origins in production

**Report security vulnerabilities**: Create a GitHub Security Advisory

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- The Verus community for building an innovative blockchain
- All contributors and maintainers

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/devdudeio/verus-gateway/issues)
- **Discussions**: [GitHub Discussions](https://github.com/devdudeio/verus-gateway/discussions)
- **Verus Discord**: [Join the community](https://discord.gg/VRKMP2S)

---

**Made with ❤️ for the Verus community**
