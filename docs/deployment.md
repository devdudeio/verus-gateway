# Verus Gateway - Deployment Guide

This guide covers various deployment options for the Verus Gateway, from local development to production environments.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Configuration](#configuration)
- [Docker Deployment](#docker-deployment)
- [Docker Compose](#docker-compose)
- [Systemd Service](#systemd-service)
- [Kubernetes](#kubernetes)
- [Production Considerations](#production-considerations)
- [Monitoring](#monitoring)
- [Troubleshooting](#troubleshooting)

## Prerequisites

### Required

- Access to a Verus node with RPC enabled
  - **Testnet**: Port 18843 (default)
  - **Mainnet**: Port 27486 (default)
- Verus RPC credentials (username and password)

### Optional

- **Docker** (recommended for production)
- **Redis** (for distributed caching)
- **Prometheus & Grafana** (for monitoring)
- **Reverse proxy** (nginx/Caddy for HTTPS)

## Configuration

### 1. Create Configuration File

Copy the example configuration:

```bash
cp config.example.yaml config.yaml
```

### 2. Configure Verus RPC Connection

Edit `config.yaml`:

```yaml
chains:
  default: vrsctest  # or 'vrsc' for mainnet
  chains:
    vrsctest:
      name: "Verus Testnet"
      enabled: true
      rpc_url: "http://localhost:18843"  # Your Verus node
      rpc_user: "your-rpc-username"
      rpc_password: "your-rpc-password"
      rpc_timeout: 30s
      max_retries: 3
      retry_delay: 500ms
```

### 3. Configure Cache (Optional)

**Filesystem Cache** (default):

```yaml
cache:
  type: filesystem
  dir: ./cache
  ttl: 24h
  max_size: 10737418240  # 10GB
  cleanup_interval: 1h
```

**Redis Cache** (recommended for production):

```yaml
cache:
  type: redis
  ttl: 24h
  redis:
    addresses:
      - localhost:6379
    password: ""  # if required
    db: 0
    pool_size: 10
    timeout: 5s
```

## Docker Deployment

### Build Docker Image

```bash
# Build the image
docker build -t verus-gateway:latest .

# Or with version tag
docker build -t verus-gateway:1.0.0 .
```

### Run Container

**With config file**:

```bash
docker run -d \
  --name verus-gateway \
  -p 8080:8080 \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  -v $(pwd)/cache:/app/cache \
  verus-gateway:latest
```

**With environment variables**:

```bash
docker run -d \
  --name verus-gateway \
  -p 8080:8080 \
  -e VERUS_GATEWAY_SERVER_PORT=8080 \
  -e VERUS_GATEWAY_CHAINS_DEFAULT=vrsctest \
  -e VERUS_GATEWAY_CHAINS_CHAINS_VRSCTEST_RPC_URL=http://host.docker.internal:18843 \
  -e VERUS_GATEWAY_CHAINS_CHAINS_VRSCTEST_RPC_USER=user \
  -e VERUS_GATEWAY_CHAINS_CHAINS_VRSCTEST_RPC_PASSWORD=password \
  verus-gateway:latest
```

### Docker Network Considerations

If your Verus node is running on the host:

```bash
# Linux
docker run --network=host ...

# macOS/Windows
# Use host.docker.internal in rpc_url:
# rpc_url: "http://host.docker.internal:18843"
```

### View Logs

```bash
# Follow logs
docker logs -f verus-gateway

# Last 100 lines
docker logs --tail 100 verus-gateway
```

### Stop Container

```bash
docker stop verus-gateway
docker rm verus-gateway
```

## Docker Compose

### Local Development

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  gateway:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - ./config.yaml:/app/config.yaml:ro
      - ./cache:/app/cache
    environment:
      - VERUS_GATEWAY_OBSERVABILITY_LOGGING_LEVEL=debug
    restart: unless-stopped
```

Run:

```bash
docker-compose up -d
```

### Production with Redis

Create `docker-compose.production.yml`:

```yaml
version: '3.8'

services:
  gateway:
    image: verus-gateway:latest
    ports:
      - "127.0.0.1:8080:8080"  # Bind to localhost only
    volumes:
      - ./config.production.yaml:/app/config.yaml:ro
    environment:
      - VERUS_GATEWAY_CACHE_TYPE=redis
      - VERUS_GATEWAY_CACHE_REDIS_ADDRESSES=redis:6379
    depends_on:
      - redis
    restart: always
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s

  redis:
    image: redis:7-alpine
    volumes:
      - redis-data:/data
    command: redis-server --appendonly yes
    restart: always
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 3s
      retries: 3

volumes:
  redis-data:
```

Run:

```bash
docker-compose -f docker-compose.production.yml up -d
```

## Systemd Service

### 1. Install Binary

```bash
# Build the binary
make build

# Copy to system location
sudo cp bin/verus-gateway /usr/local/bin/
sudo chmod +x /usr/local/bin/verus-gateway
```

### 2. Create System User

```bash
sudo useradd --system --no-create-home --shell /bin/false verus-gateway
```

### 3. Create Directories

```bash
sudo mkdir -p /etc/verus-gateway
sudo mkdir -p /var/lib/verus-gateway/cache
sudo mkdir -p /var/log/verus-gateway

sudo chown -R verus-gateway:verus-gateway /var/lib/verus-gateway
sudo chown -R verus-gateway:verus-gateway /var/log/verus-gateway
```

### 4. Install Configuration

```bash
sudo cp config.yaml /etc/verus-gateway/config.yaml
sudo chown verus-gateway:verus-gateway /etc/verus-gateway/config.yaml
sudo chmod 600 /etc/verus-gateway/config.yaml  # Protect RPC credentials
```

### 5. Create Systemd Service

Create `/etc/systemd/system/verus-gateway.service`:

```ini
[Unit]
Description=Verus Gateway
Documentation=https://github.com/devdudeio/verus-gateway
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=verus-gateway
Group=verus-gateway
ExecStart=/usr/local/bin/verus-gateway -config /etc/verus-gateway/config.yaml
Restart=on-failure
RestartSec=5s

# Security
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/verus-gateway /var/log/verus-gateway

# Limits
LimitNOFILE=65536

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=verus-gateway

[Install]
WantedBy=multi-user.target
```

### 6. Enable and Start Service

```bash
# Reload systemd
sudo systemctl daemon-reload

# Enable service (start on boot)
sudo systemctl enable verus-gateway

# Start service
sudo systemctl start verus-gateway

# Check status
sudo systemctl status verus-gateway
```

### 7. View Logs

```bash
# Follow logs
sudo journalctl -u verus-gateway -f

# Last 100 lines
sudo journalctl -u verus-gateway -n 100

# Logs from today
sudo journalctl -u verus-gateway --since today
```

### 8. Manage Service

```bash
# Stop
sudo systemctl stop verus-gateway

# Restart
sudo systemctl restart verus-gateway

# Reload configuration
sudo systemctl reload verus-gateway

# Disable (prevent start on boot)
sudo systemctl disable verus-gateway
```

## Kubernetes

### Basic Deployment

Create `deployment.yaml`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: verus-gateway-config
data:
  config.yaml: |
    server:
      port: 8080
      host: "0.0.0.0"
    chains:
      default: vrsctest
      chains:
        vrsctest:
          name: "Verus Testnet"
          enabled: true
          rpc_url: "http://verus-node:18843"
          rpc_user: "user"
          rpc_password: "password"
          rpc_timeout: 30s
    cache:
      type: redis
      redis:
        addresses:
          - redis:6379
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: verus-gateway
  labels:
    app: verus-gateway
spec:
  replicas: 3
  selector:
    matchLabels:
      app: verus-gateway
  template:
    metadata:
      labels:
        app: verus-gateway
    spec:
      containers:
      - name: verus-gateway
        image: verus-gateway:latest
        ports:
        - containerPort: 8080
          name: http
        volumeMounts:
        - name: config
          mountPath: /app/config.yaml
          subPath: config.yaml
          readOnly: true
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 500m
            memory: 512Mi
      volumes:
      - name: config
        configMap:
          name: verus-gateway-config
---
apiVersion: v1
kind: Service
metadata:
  name: verus-gateway
spec:
  selector:
    app: verus-gateway
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
  type: ClusterIP
```

Deploy:

```bash
kubectl apply -f deployment.yaml
```

## Production Considerations

### 1. HTTPS/TLS

Always use HTTPS in production. Use a reverse proxy:

**Nginx**:

```nginx
server {
    listen 443 ssl http2;
    server_name gateway.example.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

**Caddy** (automatic HTTPS):

```
gateway.example.com {
    reverse_proxy localhost:8080
}
```

### 2. Rate Limiting

Configure rate limiting at the reverse proxy level:

**Nginx**:

```nginx
limit_req_zone $binary_remote_addr zone=gateway:10m rate=10r/s;

location / {
    limit_req zone=gateway burst=20 nodelay;
    proxy_pass http://localhost:8080;
}
```

### 3. Security Headers

Add security headers in your reverse proxy:

```nginx
add_header X-Frame-Options "DENY" always;
add_header X-Content-Type-Options "nosniff" always;
add_header X-XSS-Protection "1; mode=block" always;
add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
```

### 4. Firewall

Only expose necessary ports:

```bash
# Allow HTTP/HTTPS from anywhere
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# Allow gateway port only from localhost
sudo ufw allow from 127.0.0.1 to any port 8080

# Enable firewall
sudo ufw enable
```

### 5. Monitoring

Set up Prometheus and Grafana (see [Monitoring](#monitoring) section).

### 6. Backup

Backup configuration and cache metadata:

```bash
# Backup script
#!/bin/bash
tar -czf backup-$(date +%Y%m%d).tar.gz \
  /etc/verus-gateway/config.yaml \
  /var/lib/verus-gateway/cache
```

## Monitoring

### Prometheus

Create `prometheus.yml`:

```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'verus-gateway'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
```

Run Prometheus:

```bash
docker run -d \
  --name prometheus \
  -p 9090:9090 \
  -v $(pwd)/prometheus.yml:/etc/prometheus/prometheus.yml \
  prom/prometheus
```

### Grafana

Run Grafana:

```bash
docker run -d \
  --name grafana \
  -p 3000:3000 \
  grafana/grafana
```

Access Grafana at `http://localhost:3000` (default login: admin/admin).

Add Prometheus as a data source and import dashboard from `docs/grafana-dashboard.json`.

## Troubleshooting

### Gateway won't start

Check logs:

```bash
# Systemd
sudo journalctl -u verus-gateway -n 100

# Docker
docker logs verus-gateway
```

Common issues:

- **Port already in use**: Change `server.port` in config
- **Can't connect to Verus node**: Check `rpc_url`, firewall, node status
- **Permission denied**: Check file permissions, user/group ownership

### Poor performance

1. **Enable Redis caching**
2. **Increase cache size** (`cache.max_size`)
3. **Check Verus node performance**
4. **Monitor metrics** at `/metrics`

### Cache issues

Clear cache:

```bash
curl -X DELETE http://localhost:8080/admin/cache
```

Check cache stats:

```bash
curl http://localhost:8080/admin/cache/stats
```

### Connection refused

Check if service is running:

```bash
# Systemd
sudo systemctl status verus-gateway

# Docker
docker ps | grep verus-gateway

# Check port
netstat -tulpn | grep 8080
```

### High memory usage

1. Reduce cache size
2. Adjust `cache.cleanup_interval`
3. Monitor with Prometheus metrics

---

For more help, see:
- [GitHub Issues](https://github.com/devdudeio/verus-gateway/issues)
- [API Documentation](openapi.yaml)
- [Architecture Documentation](architecture.md)
