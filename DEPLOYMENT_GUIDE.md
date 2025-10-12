# Verus Gateway - Deployment Guide

This guide will help you deploy the Verus Gateway to production.

## Quick Start Options

### Option 1: Docker (Recommended for Testing)

```bash
# 1. Pull the Docker image from GitHub Container Registry
docker pull ghcr.io/devdudeio/verus-gateway:latest

# 2. Run the container
docker run -d \
  --name verus-gateway \
  -p 8080:8080 \
  -e VERUS_RPC_HOST=your-verus-node:27486 \
  -e VERUS_RPC_USER=your-rpc-user \
  -e VERUS_RPC_PASSWORD=your-rpc-password \
  ghcr.io/devdudeio/verus-gateway:latest

# 3. Check logs
docker logs -f verus-gateway

# 4. Test the endpoint
curl http://localhost:8080/health
```

**Note:** Images are hosted on GitHub Container Registry (GHCR). Available tags:
- `latest` - Latest stable release
- `0.2.3`, `0.2`, `0` - Semantic version tags

### Option 2: Docker Compose (Recommended for Production)

```bash
# 1. Copy and configure the environment file
cp .env.example .env
# Edit .env with your configuration

# 2. Start the stack
docker compose -f docker-compose.production.yml up -d

# 3. Check status
docker compose -f docker-compose.production.yml ps

# 4. View logs
docker compose -f docker-compose.production.yml logs -f gateway
```

**Note:** Use `docker compose` (v2, newer) instead of `docker-compose` (v1, older) if available.

### Option 3: Systemd (Recommended for Production on Linux)

```bash
# 1. Build the binary
make build

# 2. Copy binary to system location
sudo cp bin/verus-gateway /usr/local/bin/

# 3. Create configuration directory
sudo mkdir -p /etc/verus-gateway
sudo cp config.example.yaml /etc/verus-gateway/config.yaml
# Edit /etc/verus-gateway/config.yaml with your settings

# 4. Install systemd service
sudo cp deployments/systemd/verus-gateway.service /etc/systemd/system/
sudo systemctl daemon-reload

# 5. Start the service
sudo systemctl enable verus-gateway
sudo systemctl start verus-gateway

# 6. Check status
sudo systemctl status verus-gateway
sudo journalctl -u verus-gateway -f
```

## Production Deployment Steps

### Step 1: Prerequisites

**Required:**
- Verus node (VRSC or VRSCTEST) running and accessible
- Redis server (recommended) or in-memory cache
- Reverse proxy (nginx or Caddy) for HTTPS
- Valid TLS/SSL certificate (Let's Encrypt recommended)
- Firewall configured

**Recommended:**
- Monitoring stack (Prometheus + Grafana)
- Log aggregation (ELK stack, Loki, etc.)
- Alerting system
- Backup solution

### Step 2: Configuration

Create your configuration file based on `config.example.yaml`:

```yaml
server:
  host: "127.0.0.1"  # Bind to localhost, use reverse proxy
  port: 8080
  read_timeout: 60s    # Increased for file decryption operations
  write_timeout: 120s  # Increased for large file transfers
  idle_timeout: 120s
  shutdown_timeout: 30s

chains:
  - id: "VRSC"
    name: "Verus"
    rpc_host: "verus-mainnet:27486"
    rpc_user: "${VRSC_RPC_USER}"
    rpc_password: "${VRSC_RPC_PASSWORD}"
    rpc_timeout: 30s
  - id: "vrsctest"
    name: "Verus Testnet"
    rpc_host: "verus-testnet:27486"
    rpc_user: "${VRSCTEST_RPC_USER}"
    rpc_password: "${VRSCTEST_RPC_PASSWORD}"
    rpc_timeout: 30s

cache:
  type: "redis"  # or "memory"
  redis:
    address: "redis:6379"
    password: "${REDIS_PASSWORD}"
    db: 0
    max_retries: 3
    pool_size: 10
  memory:
    max_size_mb: 1024
    max_items: 10000
  ttl: 24h

logging:
  level: "info"  # debug, info, warn, error
  format: "json"
  output: "stdout"

metrics:
  enabled: true
  path: "/metrics"

# Security settings (optional but recommended)
security:
  api_keys:
    - "${API_KEY_1}"
    - "${API_KEY_2}"

# Rate limiting
rate_limit:
  requests_per_window: 100
  window: 1m
  cleanup_interval: 5m

# CORS (configure for your domain)
cors:
  allowed_origins:
    - "https://your-domain.com"
    - "https://app.your-domain.com"
  allowed_methods: ["GET", "HEAD", "OPTIONS"]
  allowed_headers: ["Accept", "Content-Type", "X-API-Key"]
  allow_credentials: false
```

### Step 3: Reverse Proxy Setup

**Nginx Example:**

```nginx
# /etc/nginx/sites-available/verus-gateway
upstream verus_gateway {
    server 127.0.0.1:8080;
    keepalive 32;
}

# HTTP -> HTTPS redirect
server {
    listen 80;
    listen [::]:80;
    server_name gateway.your-domain.com;

    location / {
        return 301 https://$host$request_uri;
    }
}

# HTTPS server
server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name gateway.your-domain.com;

    # TLS configuration
    ssl_certificate /etc/letsencrypt/live/gateway.your-domain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/gateway.your-domain.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;

    # Security headers (additional to application headers)
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains; preload" always;

    # Rate limiting
    limit_req_zone $binary_remote_addr zone=gateway_limit:10m rate=10r/s;
    limit_req zone=gateway_limit burst=20 nodelay;

    # Request size limits
    client_max_body_size 10M;
    client_body_timeout 30s;

    # Logging
    access_log /var/log/nginx/gateway-access.log;
    error_log /var/log/nginx/gateway-error.log;

    # Proxy to gateway
    location / {
        proxy_pass http://verus_gateway;
        proxy_http_version 1.1;

        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Connection "";

        proxy_connect_timeout 30s;
        proxy_send_timeout 120s;  # Allow time for large file transfers
        proxy_read_timeout 60s;   # Allow time for file decryption
    }

    # Health check (internal only)
    location /health {
        proxy_pass http://verus_gateway;
        access_log off;
    }
}
```

**Caddy Example (easier):**

```caddyfile
# /etc/caddy/Caddyfile
gateway.your-domain.com {
    # Automatic HTTPS with Let's Encrypt

    # Rate limiting
    rate_limit {
        zone dynamic {
            key {remote_host}
            events 100
            window 1m
        }
    }

    # Reverse proxy
    reverse_proxy localhost:8080 {
        header_up Host {host}
        header_up X-Real-IP {remote_host}
        header_up X-Forwarded-For {remote_host}
        header_up X-Forwarded-Proto {scheme}
    }

    # Logging
    log {
        output file /var/log/caddy/gateway-access.log
    }
}
```

### Step 4: Firewall Configuration

```bash
# Allow HTTPS only (reverse proxy)
sudo ufw allow 443/tcp comment 'HTTPS'

# Block direct access to gateway port
sudo ufw deny 8080/tcp comment 'Block direct gateway access'

# Allow SSH (if needed)
sudo ufw allow 22/tcp comment 'SSH'

# Enable firewall
sudo ufw enable
```

### Step 5: Monitoring Setup

**Prometheus Configuration:**

```yaml
# /etc/prometheus/prometheus.yml
scrape_configs:
  - job_name: 'verus-gateway'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
    scrape_interval: 15s
```

**Grafana Dashboard:**

Import or create a dashboard with panels for:
- Request rate
- Response time (P50, P95, P99)
- Error rate
- Cache hit rate
- Active connections
- HTTP status codes

### Step 6: Deployment

**Using Docker Compose:**

```bash
# 1. Set environment variables
export VRSC_RPC_USER="your-rpc-user"
export VRSC_RPC_PASSWORD="your-rpc-password"
export REDIS_PASSWORD="your-redis-password"
export API_KEY_1="your-api-key"

# 2. Start the stack
docker compose -f docker-compose.production.yml up -d

# 3. Wait for services to be ready
sleep 10

# 4. Check health
curl http://localhost:8080/health
curl http://localhost:8080/ready

# 5. Check logs
docker compose -f docker-compose.production.yml logs -f
```

**Using Systemd:**

```bash
# 1. Configure environment variables in systemd
sudo systemctl edit verus-gateway

# Add these lines:
# [Service]
# Environment="VRSC_RPC_USER=your-rpc-user"
# Environment="VRSC_RPC_PASSWORD=your-rpc-password"
# Environment="REDIS_PASSWORD=your-redis-password"

# 2. Start the service
sudo systemctl start verus-gateway

# 3. Enable on boot
sudo systemctl enable verus-gateway

# 4. Check status
sudo systemctl status verus-gateway

# 5. View logs
sudo journalctl -u verus-gateway -f
```

### Step 7: Verification

**Test All Endpoints:**

```bash
# Health check
curl https://gateway.your-domain.com/health

# Readiness check
curl https://gateway.your-domain.com/ready

# Chains list
curl https://gateway.your-domain.com/chains

# Metrics
curl https://gateway.your-domain.com/metrics

# File retrieval (use a real TXID from your chain)
curl https://gateway.your-domain.com/c/vrsctest/file/YOUR_TXID_HERE
```

**Test Security Features:**

```bash
# Test rate limiting (should get 429 after limit)
for i in {1..150}; do curl -s -o /dev/null -w "%{http_code}\n" https://gateway.your-domain.com/health; done

# Test API key auth (if enabled)
curl -H "X-API-Key: your-api-key" https://gateway.your-domain.com/c/vrsctest/file/YOUR_TXID

# Test CORS
curl -H "Origin: https://example.com" -v https://gateway.your-domain.com/health

# Check security headers
curl -I https://gateway.your-domain.com/health
```

### Step 8: Monitoring and Alerting

**Set up alerts for:**

- High error rate (5xx responses)
- High latency (P95 > 1s)
- Rate limit exceeded frequently
- Service down (health check failing)
- Cache connection failures
- Unauthorized access attempts

**Example Prometheus Alert:**

```yaml
groups:
  - name: verus_gateway
    rules:
      - alert: HighErrorRate
        expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.05
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High error rate detected"
          description: "Error rate is {{ $value }} errors/sec"

      - alert: ServiceDown
        expr: up{job="verus-gateway"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Verus Gateway is down"
```

## Maintenance

### Regular Tasks

**Daily:**
```bash
# Check logs for errors
sudo journalctl -u verus-gateway --since today | grep -i error

# Check metrics
curl http://localhost:8080/metrics | grep -E "(error|cache)"
```

**Weekly:**
```bash
# Review access logs
sudo tail -n 1000 /var/log/nginx/gateway-access.log | grep -v "GET /health"

# Check for unauthorized access
sudo journalctl -u verus-gateway | grep "unauthorized_access"
```

**Monthly:**
```bash
# Rotate API keys
# Update configuration and restart service

# Update dependencies
make deps
make test
make build
```

### Troubleshooting

**Gateway not starting:**
```bash
# Check configuration
./bin/verus-gateway --config /etc/verus-gateway/config.yaml --validate

# Check logs
sudo journalctl -u verus-gateway -n 100

# Check port availability
sudo lsof -i :8080
```

**Cache not working:**
```bash
# Test Redis connection
redis-cli -h localhost -p 6379 ping

# Check cache metrics
curl http://localhost:8080/metrics | grep cache
```

**High latency:**
```bash
# Check Verus node latency
time verus getblockchaininfo

# Check cache hit rate
curl http://localhost:8080/metrics | grep cache_hits

# Check connection pool
curl http://localhost:8080/metrics | grep http_connections
```

## Rollback Procedure

**Docker Compose:**
```bash
# Stop current version
docker compose -f docker-compose.production.yml down

# Pull previous version from GHCR
docker pull ghcr.io/devdudeio/verus-gateway:0.2.2  # or desired version

# Update docker-compose.production.yml to use previous tag
# Example: image: ghcr.io/devdudeio/verus-gateway:0.2.2
# Start previous version
docker compose -f docker-compose.production.yml up -d
```

**Systemd:**
```bash
# Stop service
sudo systemctl stop verus-gateway

# Restore previous binary
sudo cp /usr/local/bin/verus-gateway.backup /usr/local/bin/verus-gateway

# Start service
sudo systemctl start verus-gateway
```

## Security Checklist

Before going to production, ensure:

- [ ] HTTPS enabled with valid certificate
- [ ] Rate limiting configured (infrastructure + application)
- [ ] API key authentication enabled (if not fully public)
- [ ] CORS configured for specific origins only
- [ ] Audit logging enabled
- [ ] Firewall rules configured
- [ ] Reverse proxy configured
- [ ] Security headers verified
- [ ] Request size limits set
- [ ] Monitoring and alerting configured
- [ ] Log retention configured (90+ days for security logs)
- [ ] Incident response procedures documented
- [ ] Backup procedures in place

## Support

For issues or questions:
- Check logs: `sudo journalctl -u verus-gateway -f`
- Review metrics: `curl http://localhost:8080/metrics`
- Consult documentation: `docs/security.md`, `README.md`

---

**Deployment Status:** Ready for Production
**Last Updated:** 2025-10-11
