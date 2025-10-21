# Deployment Guide - Consent Management API

## Table of Contents
1. [Building for Production](#building-for-production)
2. [Configuration](#configuration)
3. [Running in Production](#running-in-production)
4. [Docker Deployment](#docker-deployment)
5. [Systemd Service](#systemd-service)
6. [Health Checks](#health-checks)

---

## Building for Production

### 1. Build Binary

Build an optimized binary for your target platform:

```bash
# For Linux (most common for production)
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o consent-api-server \
  -ldflags="-w -s" \
  ./cmd/server/main.go

# For macOS
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o consent-api-server \
  -ldflags="-w -s" \
  ./cmd/server/main.go

# For Windows
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o consent-api-server.exe \
  -ldflags="-w -s" \
  ./cmd/server/main.go
```

**Build flags explained:**
- `CGO_ENABLED=0` - Disables CGO for static binary
- `-ldflags="-w -s"` - Strips debug info to reduce binary size
- `-o` - Output binary name

### 2. Verify Binary

```bash
./consent-api-server --help
```

---

## Configuration

### 1. Production Config File

Create a production config file at `configs/config.prod.yaml`:

```yaml
server:
  host: "0.0.0.0"  # Listen on all interfaces
  port: 9446
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 60s
  shutdown_timeout: 30s

database:
  host: "your-db-host.example.com"  # Production database
  port: 3306
  user: "consent_api_user"
  password: "${DB_PASSWORD}"  # Use environment variable
  database: "consent_management"
  max_open_conns: 25
  max_idle_conns: 10
  conn_max_lifetime: 5m
  conn_max_idle_time: 5m

logging:
  level: "info"  # Use "info" or "warn" in production
  format: "json"
  output: "stdout"
```

### 2. Environment Variables

Set these environment variables in production:

```bash
export DB_PASSWORD="your-secure-password"
export GIN_MODE=release  # Important for production!
export CONFIG_PATH="/path/to/configs/config.prod.yaml"
```

---

## Running in Production

### Option 1: Direct Execution

```bash
# Set environment variables
export GIN_MODE=release
export DB_PASSWORD="your-password"

# Run the server
./consent-api-server
```

### Option 2: Using nohup (Background)

```bash
nohup ./consent-api-server > /var/log/consent-api/server.log 2>&1 &
echo $! > /var/run/consent-api.pid
```

### Option 3: Using screen/tmux

```bash
# Using screen
screen -dmS consent-api ./consent-api-server

# Using tmux
tmux new-session -d -s consent-api './consent-api-server'
```

---

## Docker Deployment

### 1. Create Dockerfile

Create `Dockerfile` in project root:

```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o consent-api-server \
    -ldflags="-w -s" \
    ./cmd/server/main.go

# Runtime stage
FROM alpine:3.19

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/consent-api-server .
COPY --from=builder /app/configs ./configs

# Set ownership
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 9446

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:9446/health || exit 1

# Run the application
CMD ["./consent-api-server"]
```

### 2. Create docker-compose.yml

```yaml
version: '3.8'

services:
  consent-api:
    build: .
    container_name: consent-api-server
    ports:
      - "9446:9446"
    environment:
      - GIN_MODE=release
      - DB_PASSWORD=${DB_PASSWORD}
      - CONFIG_PATH=/app/configs/config.yaml
    volumes:
      - ./configs:/app/configs:ro
      - ./logs:/app/logs
    restart: unless-stopped
    depends_on:
      - mysql
    networks:
      - consent-network

  mysql:
    image: mysql:8.0
    container_name: consent-mysql
    environment:
      - MYSQL_ROOT_PASSWORD=${MYSQL_ROOT_PASSWORD}
      - MYSQL_DATABASE=consent_management
      - MYSQL_USER=consent_api_user
      - MYSQL_PASSWORD=${DB_PASSWORD}
    volumes:
      - mysql-data:/var/lib/mysql
      - ./migrations:/docker-entrypoint-initdb.d
    ports:
      - "3306:3306"
    restart: unless-stopped
    networks:
      - consent-network

volumes:
  mysql-data:

networks:
  consent-network:
    driver: bridge
```

### 3. Build and Run with Docker

```bash
# Build the image
docker build -t consent-api:latest .

# Run with docker-compose
docker-compose up -d

# View logs
docker-compose logs -f consent-api

# Stop services
docker-compose down
```

---

## Systemd Service

### 1. Create Service File

Create `/etc/systemd/system/consent-api.service`:

```ini
[Unit]
Description=Consent Management API Server
After=network.target mysql.service
Wants=mysql.service

[Service]
Type=simple
User=consent-api
Group=consent-api
WorkingDirectory=/opt/consent-api
ExecStart=/opt/consent-api/consent-api-server
Restart=always
RestartSec=5
StandardOutput=append:/var/log/consent-api/server.log
StandardError=append:/var/log/consent-api/error.log

# Environment variables
Environment="GIN_MODE=release"
Environment="CONFIG_PATH=/opt/consent-api/configs/config.prod.yaml"
EnvironmentFile=/etc/consent-api/environment

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/log/consent-api

[Install]
WantedBy=multi-user.target
```

### 2. Setup Systemd Service

```bash
# Create user and directories
sudo useradd -r -s /bin/false consent-api
sudo mkdir -p /opt/consent-api/configs
sudo mkdir -p /var/log/consent-api
sudo mkdir -p /etc/consent-api

# Copy files
sudo cp consent-api-server /opt/consent-api/
sudo cp configs/config.prod.yaml /opt/consent-api/configs/
sudo chmod +x /opt/consent-api/consent-api-server

# Create environment file
echo "DB_PASSWORD=your-password" | sudo tee /etc/consent-api/environment
sudo chmod 600 /etc/consent-api/environment

# Set ownership
sudo chown -R consent-api:consent-api /opt/consent-api
sudo chown -R consent-api:consent-api /var/log/consent-api

# Enable and start service
sudo systemctl daemon-reload
sudo systemctl enable consent-api
sudo systemctl start consent-api

# Check status
sudo systemctl status consent-api

# View logs
sudo journalctl -u consent-api -f
```

### 3. Service Management Commands

```bash
# Start service
sudo systemctl start consent-api

# Stop service
sudo systemctl stop consent-api

# Restart service
sudo systemctl restart consent-api

# Reload configuration
sudo systemctl reload consent-api

# View status
sudo systemctl status consent-api

# Enable auto-start on boot
sudo systemctl enable consent-api

# Disable auto-start
sudo systemctl disable consent-api

# View logs
sudo journalctl -u consent-api -n 100 -f
```

---

## Health Checks

### 1. Health Endpoint

The API exposes a health check endpoint:

```bash
curl http://localhost:9446/health
```

Expected response:
```json
{
  "status": "healthy"
}
```

### 2. Database Health Check

```bash
# Simple check
curl -f http://localhost:9446/health || echo "Service is down"

# Detailed check with timeout
curl --max-time 3 -f http://localhost:9446/health && echo "✓ Healthy" || echo "✗ Unhealthy"
```

### 3. Monitoring Script

Create `/usr/local/bin/check-consent-api.sh`:

```bash
#!/bin/bash

HEALTH_URL="http://localhost:9446/health"
MAX_RETRIES=3
RETRY_DELAY=2

for i in $(seq 1 $MAX_RETRIES); do
    if curl -sf --max-time 3 "$HEALTH_URL" > /dev/null; then
        echo "✓ Consent API is healthy"
        exit 0
    fi
    echo "Attempt $i/$MAX_RETRIES failed, retrying in ${RETRY_DELAY}s..."
    sleep $RETRY_DELAY
done

echo "✗ Consent API is unhealthy after $MAX_RETRIES attempts"
exit 1
```

```bash
# Make executable
sudo chmod +x /usr/local/bin/check-consent-api.sh

# Add to crontab for monitoring
*/5 * * * * /usr/local/bin/check-consent-api.sh >> /var/log/consent-api/health-check.log 2>&1
```

---

## Nginx Reverse Proxy (Optional)

If you want to run behind Nginx:

```nginx
upstream consent_api {
    server localhost:9446;
    keepalive 32;
}

server {
    listen 80;
    server_name api.yourdomain.com;

    # Redirect to HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name api.yourdomain.com;

    ssl_certificate /etc/ssl/certs/your-cert.pem;
    ssl_certificate_key /etc/ssl/private/your-key.pem;

    location / {
        proxy_pass http://consent_api;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_connect_timeout 30s;
        proxy_send_timeout 30s;
        proxy_read_timeout 30s;
    }

    location /health {
        proxy_pass http://consent_api/health;
        access_log off;
    }
}
```

---

## Performance Tuning

### 1. Database Connection Pool

In your production config:

```yaml
database:
  max_open_conns: 25      # Max open connections
  max_idle_conns: 10      # Max idle connections
  conn_max_lifetime: 5m   # Connection max lifetime
  conn_max_idle_time: 5m  # Connection max idle time
```

### 2. Server Timeouts

```yaml
server:
  read_timeout: 30s       # Time to read request
  write_timeout: 30s      # Time to write response
  idle_timeout: 60s       # Keep-alive timeout
  shutdown_timeout: 30s   # Graceful shutdown timeout
```

---

## Security Checklist

- [ ] Set `GIN_MODE=release` in production
- [ ] Use strong database passwords
- [ ] Store secrets in environment variables, not config files
- [ ] Run as non-root user
- [ ] Enable firewall rules
- [ ] Use HTTPS/TLS in production
- [ ] Implement rate limiting
- [ ] Regular security updates
- [ ] Enable logging and monitoring
- [ ] Backup database regularly

---

## Troubleshooting

### Service won't start

```bash
# Check logs
sudo journalctl -u consent-api -n 50

# Check configuration
./consent-api-server --help

# Verify database connectivity
mysql -h hostname -u username -p
```

### High memory usage

```bash
# Check resource usage
docker stats consent-api-server

# View Go runtime metrics
curl http://localhost:9446/debug/pprof/heap
```

### Database connection issues

```bash
# Test database connection
mysql -h db-host -P 3306 -u consent_api_user -p

# Check connection pool settings in config
# Adjust max_open_conns and max_idle_conns
```

---

## Quick Start Commands

```bash
# 1. Build
CGO_ENABLED=0 GOOS=linux go build -o consent-api-server -ldflags="-w -s" ./cmd/server/main.go

# 2. Set environment
export GIN_MODE=release
export DB_PASSWORD="your-password"

# 3. Run
./consent-api-server

# 4. Test
curl http://localhost:9446/health
```
