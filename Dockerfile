# ============================================
# Stage 1: Dependencies Cache Layer
# ============================================
FROM golang:1.25.0-alpine AS deps

# Install build dependencies
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata

WORKDIR /build

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./

# Download dependencies with caching
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download && \
    go mod verify

# ============================================
# Stage 2: Build Layer
# ============================================
FROM golang:1.25.0-alpine AS builder

# Install build essentials
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Copy source code (all modules and common packages)
COPY main.go ./
COPY common/ ./common/
COPY user/ ./user/
COPY product/ ./product/
COPY order/ ./order/
COPY payment/ ./payment/
COPY notification/ ./notification/

# Download dependencies with cache
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Build the application with optimizations
# - CGO_ENABLED=0: Static binary (no C dependencies)
# - -ldflags="-s -w": Strip debug info and symbol table
# - -trimpath: Remove file system paths from binary
# - -a: Force rebuilding of packages
# - GOARCH: Auto-detect architecture (amd64 or arm64)
ARG TARGETARCH
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH:-amd64} \
    go build \
    -ldflags="-s -w -X main.version=$(date +%Y%m%d-%H%M%S)" \
    -trimpath \
    -a \
    -installsuffix cgo \
    -o app \
    ./main.go

# Verify the binary is executable
RUN chmod +x /build/app

# ============================================
# Stage 3: Runtime Layer (Minimal)
# ============================================
FROM alpine:3.20 AS runtime

# Install runtime dependencies only
# - ca-certificates: For HTTPS connections
# - tzdata: Timezone data
# - postgresql-client: For running migrations (optional, can be removed if migrations run separately)
# - wget: For health checks
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    postgresql-client \
    wget \
    bash \
    && addgroup -g 1000 appuser \
    && adduser -D -u 1000 -G appuser appuser

# Set timezone (optional, can be overridden)
ENV TZ=UTC

WORKDIR /app

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy CA certificates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary from builder
COPY --from=builder /build/app .

# Copy migrations and seeds for database setup
COPY --chown=appuser:appuser migrations/ ./migrations/

# Make migration script executable
RUN chmod +x ./migrations/run_migrations.sh

# Switch to non-root user for security
USER appuser

# Expose port (default 8080, can be overridden)
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:${PORT:-8080}/ || exit 1

# Set default environment variables
ENV GIN_MODE=release \
    PORT=8080

# Run the application
ENTRYPOINT ["/app/app"]
