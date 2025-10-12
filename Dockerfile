# Multi-stage Dockerfile for Verus Gateway
# Optimized for small image size and security

# Stage 1: Builder
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
# CGO_ENABLED=0 for static binary
# -ldflags for smaller binary size
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a \
    -o verus-gateway \
    ./cmd/gateway

# Stage 2: Runtime
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    wget \
    && rm -rf /var/cache/apk/*

# Create non-root user
RUN addgroup -g 1000 verusgateway && \
    adduser -D -u 1000 -G verusgateway verusgateway

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/verus-gateway /app/verus-gateway

# Copy example configuration (can be overridden with volume mount)
COPY --from=builder /build/config.example.yaml /app/config.example.yaml

# Create cache directory
RUN mkdir -p /app/cache && \
    chown -R verusgateway:verusgateway /app

# Switch to non-root user
USER verusgateway

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --quiet --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
ENTRYPOINT ["/app/verus-gateway"]
CMD ["-config", "/app/config.yaml"]
