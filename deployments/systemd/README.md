# Systemd Deployment

This directory contains systemd service files for deploying Verus Gateway as a system service.

## Installation

### 1. Build the Binary

```bash
make build
```

### 2. Install Binary

```bash
sudo cp bin/verus-gateway /usr/local/bin/
sudo chmod +x /usr/local/bin/verus-gateway
```

### 3. Create System User

```bash
sudo useradd --system --no-create-home --shell /bin/false verus-gateway
```

### 4. Create Required Directories

```bash
# Config directory
sudo mkdir -p /etc/verus-gateway

# Data directory
sudo mkdir -p /var/lib/verus-gateway/cache

# Log directory
sudo mkdir -p /var/log/verus-gateway

# Set ownership
sudo chown -R verus-gateway:verus-gateway /var/lib/verus-gateway
sudo chown -R verus-gateway:verus-gateway /var/log/verus-gateway
```

### 5. Install Configuration

```bash
# Copy and customize your config
sudo cp config.yaml /etc/verus-gateway/config.yaml
sudo chown verus-gateway:verus-gateway /etc/verus-gateway/config.yaml
sudo chmod 600 /etc/verus-gateway/config.yaml  # Protect RPC credentials
```

**Important**: Edit `/etc/verus-gateway/config.yaml` and configure your Verus RPC connection.

### 6. Install Systemd Service

```bash
sudo cp deployments/systemd/verus-gateway.service /etc/systemd/system/
sudo systemctl daemon-reload
```

### 7. Enable and Start Service

```bash
# Enable service (start on boot)
sudo systemctl enable verus-gateway

# Start service
sudo systemctl start verus-gateway

# Check status
sudo systemctl status verus-gateway
```

## Management

### View Logs

```bash
# Follow logs in real-time
sudo journalctl -u verus-gateway -f

# Last 100 lines
sudo journalctl -u verus-gateway -n 100

# Logs from today
sudo journalctl -u verus-gateway --since today

# Logs from specific time
sudo journalctl -u verus-gateway --since "2025-10-10 10:00:00"
```

### Service Control

```bash
# Stop service
sudo systemctl stop verus-gateway

# Restart service
sudo systemctl restart verus-gateway

# Reload configuration (graceful restart)
sudo systemctl reload verus-gateway

# Disable service (prevent start on boot)
sudo systemctl disable verus-gateway

# Check service status
sudo systemctl status verus-gateway

# View service configuration
systemctl cat verus-gateway
```

### Configuration Changes

After modifying `/etc/verus-gateway/config.yaml`:

```bash
# Restart the service to apply changes
sudo systemctl restart verus-gateway
```

### Updating the Binary

```bash
# Stop the service
sudo systemctl stop verus-gateway

# Replace the binary
sudo cp bin/verus-gateway /usr/local/bin/

# Start the service
sudo systemctl start verus-gateway

# Check status
sudo systemctl status verus-gateway
```

## Troubleshooting

### Service Won't Start

Check the logs:
```bash
sudo journalctl -u verus-gateway -n 50 --no-pager
```

Common issues:
- **Permission denied**: Check file ownership and permissions
- **Port already in use**: Change `server.port` in config.yaml
- **Can't connect to Verus node**: Verify `rpc_url`, firewall, node status
- **Config file not found**: Ensure `/etc/verus-gateway/config.yaml` exists

### Check Service Health

```bash
# Health endpoint
curl http://localhost:8080/health

# Readiness endpoint
curl http://localhost:8080/ready

# List available chains
curl http://localhost:8080/chains

# Prometheus metrics
curl http://localhost:8080/metrics
```

### Performance Monitoring

```bash
# Resource usage
systemctl status verus-gateway

# Detailed process info
ps aux | grep verus-gateway

# Open file descriptors
sudo lsof -p $(pgrep verus-gateway)

# Network connections
sudo netstat -tlnp | grep verus-gateway
```

## Security Hardening

The service file includes comprehensive security hardening:

- **NoNewPrivileges**: Prevents privilege escalation
- **ProtectSystem**: Read-only root filesystem
- **ProtectHome**: No access to user home directories
- **PrivateTmp**: Private /tmp directory
- **SystemCallFilter**: Restricted system calls
- **File Limits**: Configurable resource limits

To further harden:

1. **Firewall**: Only expose necessary ports
   ```bash
   sudo ufw allow 8080/tcp
   sudo ufw enable
   ```

2. **AppArmor/SELinux**: Add mandatory access control (advanced)

3. **Audit Logging**: Enable audit logs for security monitoring

## Uninstallation

```bash
# Stop and disable service
sudo systemctl stop verus-gateway
sudo systemctl disable verus-gateway

# Remove service file
sudo rm /etc/systemd/system/verus-gateway.service
sudo systemctl daemon-reload

# Remove binary
sudo rm /usr/local/bin/verus-gateway

# Remove configuration (optional)
sudo rm -rf /etc/verus-gateway

# Remove data (optional - warning: deletes cache)
sudo rm -rf /var/lib/verus-gateway

# Remove user (optional)
sudo userdel verus-gateway
```

## Production Recommendations

1. **Use a reverse proxy** (nginx/Caddy) for HTTPS
2. **Enable rate limiting** at the proxy level
3. **Monitor metrics** with Prometheus/Grafana
4. **Set up log rotation** for journal logs
5. **Configure backups** for config file
6. **Test failover** scenarios

See [deployment.md](../../docs/deployment.md) for complete production deployment guide.
