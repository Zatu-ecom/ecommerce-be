# ============================================
# Build Layer
# ============================================
FROM golang:1.25.0-alpine AS builder

# Install build essentials
RUN apk add --no-cache git tzdata

WORKDIR /build

# 1. Copy go mod files FIRST to cache dependencies
COPY go.mod go.sum ./

# 2. Download dependencies (this layer is cached if go.mod doesn't change)
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# 3. Copy EVERYTHING else (Source code, config, all folders)
# This replaces all the manual COPY user/ user/ lines
COPY . .

# 4. Build
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

# Verify
RUN chmod +x /build/app

# ============================================
# Runtime Layer
# ============================================
FROM alpine:3.20 AS runtime

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    postgresql-client \
    wget \
    bash \
    && addgroup -g 1000 appuser \
    && adduser -D -u 1000 -G appuser appuser

ENV TZ=UTC
WORKDIR /app

# Copy necessary files from builder
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/app .

# Copy migrations (Assumes migrations folder is in root)
COPY --chown=appuser:appuser migrations/ ./migrations/
RUN chmod +x ./migrations/run_migrations.sh

USER appuser
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:${PORT:-8080}/ || exit 1

ENV GIN_MODE=release \
    PORT=8080

ENTRYPOINT ["/app/app"]
