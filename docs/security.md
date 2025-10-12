# Verus Gateway - Security Guide

## Table of Contents

- [Overview](#overview)
- [Security Architecture](#security-architecture)
- [Input Validation](#input-validation)
- [Authentication & Authorization](#authentication--authorization)
- [Rate Limiting](#rate-limiting)
- [Security Headers](#security-headers)
- [CORS Configuration](#cors-configuration)
- [Encryption & Data Protection](#encryption--data-protection)
- [Audit Logging](#audit-logging)
- [Deployment Security](#deployment-security)
- [Security Best Practices](#security-best-practices)
- [Incident Response](#incident-response)
- [Security Checklist](#security-checklist)

## Overview

The Verus Gateway implements defense-in-depth security principles with multiple layers of protection:

1. **Input Validation**: Strict validation of all user inputs
2. **Rate Limiting**: Protection against abuse and DoS attacks
3. **Authentication**: Optional API key authentication
4. **Security Headers**: Comprehensive HTTP security headers
5. **CORS**: Configurable Cross-Origin Resource Sharing
6. **Audit Logging**: Security event tracking
7. **Encryption**: Support for encrypted file retrieval with viewing keys

## Security Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Client Request                        │
└───────────────────────┬─────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────┐
│              Reverse Proxy (nginx/Caddy)                │
│  - TLS Termination                                      │
│  - Rate Limiting (infrastructure level)                 │
│  - DDoS Protection                                      │
└───────────────────────┬─────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────┐
│                 Verus Gateway                           │
│  ┌──────────────────────────────────────────────────┐  │
│  │  Security Middleware Stack (in order)            │  │
│  │  1. Request ID                                    │  │
│  │  2. Real IP Detection                             │  │
│  │  3. Request Size Limits                           │  │
│  │  4. URI Length Limits                             │  │
│  │  5. Rate Limiting (application level)             │  │
│  │  6. CORS Validation                               │  │
│  │  7. Security Headers                              │  │
│  │  8. API Key Auth (optional)                       │  │
│  │  9. Audit Logging                                 │  │
│  │  10. Request Logging                              │  │
│  │  11. Panic Recovery                               │  │
│  │  12. Metrics Collection                           │  │
│  └──────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────┐  │
│  │  Input Validation                                 │  │
│  │  - TXID format validation                         │  │
│  │  - Filename sanitization                          │  │
│  │  - Chain ID validation                            │  │
│  │  - EVK format validation                          │  │
│  │  - Path traversal prevention                      │  │
│  └──────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────┐  │
│  │  Business Logic                                   │  │
│  │  - File retrieval                                 │  │
│  │  - Decryption (if EVK provided)                   │  │
│  │  - Caching                                        │  │
│  └──────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

## Input Validation

### TXID Validation

Transaction IDs must be exactly 64 hexadecimal characters:

```go
Pattern: ^[a-fA-F0-9]{64}$
Example: 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
```

**Security Benefits:**
- Prevents injection attacks
- Ensures valid blockchain transaction references
- Protects against malformed requests

### Filename Validation

Filenames are sanitized to prevent path traversal and injection:

```go
Pattern: ^[a-zA-Z0-9._\-]+$
Max Length: 255 characters
Forbidden: .., /, \, and special characters
```

**Security Benefits:**
- Prevents directory traversal attacks
- Blocks file system injection
- Safe for HTTP headers

### Chain ID Validation

Chain identifiers are validated for safety:

```go
Pattern: ^[a-zA-Z0-9_\-]+$
Max Length: 32 characters
```

### Viewing Key (EVK) Validation

Encryption viewing keys must match the expected format:

```go
Pattern: ^zxviews[a-zA-Z0-9]{90,}$
Length: 95-200 characters
```

**Security Benefits:**
- Validates key format before processing
- Prevents injection of malicious data
- Never logged or exposed in responses

## Authentication & Authorization

### API Key Authentication (Optional)

API key authentication can be enabled for production deployments:

**Configuration:**

```yaml
# config.yaml
security:
  api_keys:
    - "your-secret-api-key-here"
    - "another-key-for-backup"
```

**Usage:**

```bash
# Header-based
curl -H "X-API-Key: your-secret-api-key-here" \
  http://localhost:8080/c/vrsctest/file/{txid}

# Bearer token
curl -H "Authorization: Bearer your-secret-api-key-here" \
  http://localhost:8080/c/vrsctest/file/{txid}
```

**Security Features:**
- Constant-time comparison to prevent timing attacks
- Support for multiple keys (key rotation)
- Logged authentication attempts (audit trail)

### Public vs. Authenticated Endpoints

By default, file retrieval endpoints are public. For production:

```go
// Option 1: Require API keys for all endpoints
router.Use(apiKeyAuth.Require())

// Option 2: Optional API keys (rate limit relief)
router.Use(apiKeyAuth.Optional())

// Option 3: Selective protection
router.Group(func(r chi.Router) {
    r.Use(apiKeyAuth.Require())
    r.Delete("/admin/cache", handler.ClearCache)
})
```

## Rate Limiting

### Application-Level Rate Limiting

Built-in token bucket rate limiter:

**Default Configuration:**
```yaml
rate_limit:
  requests_per_window: 100  # requests
  window: 1m                # per minute
  cleanup_interval: 5m      # cleanup old visitors
```

**Behavior:**
- Per-IP rate limiting
- Returns HTTP 429 with `Retry-After` header
- Automatic cleanup of old visitors
- Handles X-Forwarded-For / X-Real-IP

**Example Response:**
```http
HTTP/1.1 429 Too Many Requests
Retry-After: 60
Content-Type: application/json

{
  "error": "RATE_LIMIT_EXCEEDED",
  "message": "Rate limit exceeded. Please try again later."
}
```

### Infrastructure-Level Rate Limiting

For production, also implement reverse proxy rate limiting:

**Nginx Example:**
```nginx
limit_req_zone $binary_remote_addr zone=gateway:10m rate=10r/s;

location / {
    limit_req zone=gateway burst=20 nodelay;
    proxy_pass http://gateway:8080;
}
```

## Security Headers

### Applied Headers

All responses include comprehensive security headers:

```http
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Referrer-Policy: strict-origin-when-cross-origin
Content-Security-Policy: default-src 'none'; frame-ancestors 'none'
Permissions-Policy: geolocation=(), microphone=(), camera=()
```

**HTTPS Only:**
```http
Strict-Transport-Security: max-age=31536000; includeSubDomains; preload
```

### Header Descriptions

| Header | Purpose | Value |
|--------|---------|-------|
| X-Content-Type-Options | Prevent MIME sniffing | nosniff |
| X-Frame-Options | Prevent clickjacking | DENY |
| X-XSS-Protection | Enable browser XSS filter | 1; mode=block |
| Referrer-Policy | Control referrer information | strict-origin-when-cross-origin |
| Content-Security-Policy | Prevent XSS and injection | default-src 'none' |
| Permissions-Policy | Disable dangerous features | geolocation=(), microphone=() |
| HSTS | Force HTTPS | max-age=31536000; includeSubDomains |

## CORS Configuration

### Default Configuration (Secure)

By default, CORS is restrictive:

```yaml
cors:
  allowed_origins: []  # No origins allowed (most secure)
  allowed_methods: ["GET", "HEAD", "OPTIONS"]
  allowed_headers: ["Accept", "Content-Type", "X-Request-ID"]
  exposed_headers: ["X-Request-ID", "X-Cache-Status"]
  allow_credentials: false
  max_age: 3600
```

### Production CORS

For production with specific origins:

```yaml
cors:
  allowed_origins:
    - "https://example.com"
    - "https://app.example.com"
    - "*.example.com"  # Wildcard subdomain
  allowed_methods: ["GET", "HEAD", "OPTIONS"]
  allowed_headers: ["Accept", "Content-Type", "X-API-Key"]
  allow_credentials: false
```

### Public Gateway (Open CORS)

⚠️ **Only for public CDN-style deployments:**

```yaml
cors:
  allowed_origins: ["*"]
  allowed_methods: ["GET", "HEAD", "OPTIONS"]
```

## Encryption & Data Protection

### Viewing Keys (EVK)

Viewing keys are never stored or logged:

```go
// ✅ Good - EVK only in memory
func GetFile(ctx context.Context, txid, evk string) (*File, error) {
    // EVK used for decryption only
    data := decryptor.Decrypt(txid, evk)
    // EVK discarded after use
}

// ❌ Bad - Don't log EVKs
logger.Info("Retrieving file", "txid", txid, "evk", evk) // NEVER DO THIS
```

### Cache Keys

EVKs are never stored in cache keys:

```go
func CacheKey(txid, evk string) string {
    if evk != "" {
        return txid + ":encrypted"  // Generic marker, not the actual EVK
    }
    return txid
}
```

### Sensitive Data Masking

The logger automatically masks sensitive data:

```go
logger.MaskSensitiveData()  // Masks passwords, keys, tokens in logs
```

## Audit Logging

### Security Events Logged

1. **Unauthorized Access**
   - Failed authentication attempts
   - Invalid API keys
   - Missing authentication

2. **Rate Limiting**
   - Rate limit exceeded events
   - Client IP addresses
   - Timestamp

3. **Server Errors**
   - 5xx errors
   - Panic recovery
   - Stack traces

4. **Administrative Actions**
   - Cache clears
   - Configuration changes

### Audit Log Format

```json
{
  "level": "warn",
  "event": "unauthorized_access",
  "path": "/c/vrsctest/file/abc123...",
  "remote_addr": "203.0.113.1",
  "user_agent": "Mozilla/5.0...",
  "timestamp": "2025-10-11T00:00:00Z",
  "message": "Unauthorized access attempt"
}
```

### Log Retention

Recommended retention periods:

- **Security logs**: 90 days minimum
- **Audit logs**: 1 year minimum
- **Access logs**: 30 days minimum

## Deployment Security

### Docker Security

```dockerfile
# Run as non-root user
USER verusgateway

# Read-only filesystem
RUN mkdir -p /app/cache && chown verusgateway:verusgateway /app

# Health checks
HEALTHCHECK --interval=30s --timeout=10s --retries=3 \
  CMD wget --quiet --tries=1 --spider http://localhost:8080/health || exit 1
```

### Systemd Security

```ini
[Service]
# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/verus-gateway

# Resource limits
LimitNOFILE=65536

# Capability restrictions
CapabilityBoundingSet=
AmbientCapabilities=

# System call filtering
SystemCallFilter=@system-service
```

### Network Security

```bash
# Firewall rules
sudo ufw allow 443/tcp   # HTTPS only
sudo ufw deny 8080/tcp   # Block direct access to gateway
sudo ufw enable

# Bind to localhost only
server:
  host: "127.0.0.1"
  port: 8080
```

### Reverse Proxy Configuration

**Always use a reverse proxy with:**
- TLS termination
- Rate limiting
- DDoS protection
- Request size limits
- IP filtering (if needed)

**Nginx Example:**
```nginx
server {
    listen 443 ssl http2;
    server_name gateway.example.com;

    # TLS Configuration
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    # Security headers
    add_header Strict-Transport-Security "max-age=31536000" always;

    # Rate limiting
    limit_req zone=gateway burst=20 nodelay;

    # Request size limit
    client_max_body_size 10M;

    # Proxy settings
    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## Security Best Practices

### Development

- [ ] Never commit secrets to git
- [ ] Use environment variables for credentials
- [ ] Enable all linters and security scanners
- [ ] Review dependencies regularly
- [ ] Keep dependencies up to date

### Configuration

- [ ] Use strong API keys (32+ random characters)
- [ ] Enable rate limiting
- [ ] Configure CORS restrictively
- [ ] Use HTTPS in production
- [ ] Bind to localhost, use reverse proxy

### Operations

- [ ] Monitor security logs
- [ ] Set up alerting for security events
- [ ] Rotate API keys regularly
- [ ] Backup configuration securely
- [ ] Test disaster recovery procedures

### Monitoring

```bash
# Monitor failed auth attempts
journalctl -u verus-gateway | grep "unauthorized_access"

# Monitor rate limits
curl http://localhost:8080/metrics | grep rate_limit

# Check security headers
curl -I https://gateway.example.com/health
```

## Incident Response

### Security Incident Procedure

1. **Detection**
   - Monitor logs for suspicious activity
   - Set up alerts for:
     - High rate of 401/403 errors
     - Unusual traffic patterns
     - Multiple failed auth attempts

2. **Containment**
   - Rotate API keys immediately
   - Block malicious IPs at firewall/reverse proxy
   - Scale down if under DDoS

3. **Investigation**
   - Review audit logs
   - Identify attack vector
   - Assess damage

4. **Recovery**
   - Apply patches/fixes
   - Update security rules
   - Restore from backup if needed

5. **Post-Incident**
   - Document incident
   - Update procedures
   - Improve monitoring

### Emergency Contacts

Maintain a security contact list:
- DevOps team
- Security team
- Verus node operators
- Cloud provider support

## Security Checklist

### Before Production Deployment

- [ ] Enable HTTPS with valid certificate
- [ ] Configure rate limiting (infrastructure + application)
- [ ] Set up API key authentication
- [ ] Configure CORS for specific origins only
- [ ] Enable audit logging
- [ ] Set up log retention
- [ ] Configure firewall rules
- [ ] Use reverse proxy (nginx/Caddy)
- [ ] Enable security headers
- [ ] Test authentication
- [ ] Test rate limiting
- [ ] Run security scanner
- [ ] Review all exposed endpoints
- [ ] Document security architecture
- [ ] Set up monitoring and alerting
- [ ] Test incident response procedures

### Regular Security Audits

**Monthly:**
- [ ] Review access logs
- [ ] Check for failed auth attempts
- [ ] Update dependencies
- [ ] Rotate API keys

**Quarterly:**
- [ ] Security audit
- [ ] Penetration testing
- [ ] Review security policies
- [ ] Update documentation

**Annually:**
- [ ] Comprehensive security review
- [ ] Third-party security audit
- [ ] Disaster recovery drill

## Security Reporting

To report security vulnerabilities:

1. **DO NOT** create a public GitHub issue
2. Create a private security advisory:
   - Go to https://github.com/devdudeio/verus-gateway/security/advisories
   - Click "Report a vulnerability"
3. Include:
   - Description of vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if known)

We aim to respond to security reports within 48 hours.

---

**Security is a continuous process, not a one-time setup. Stay vigilant!**
