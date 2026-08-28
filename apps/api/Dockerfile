# NOTE: For production, use the unified Dockerfile at project root.
# This file is kept for development and CI split builds.
# ==============================================================================
# Vido API - Multi-Stage Dockerfile
# ==============================================================================
# This Dockerfile creates a minimal, secure production image for the Vido API.
# Uses multi-stage build to minimize final image size (~20MB).
#
# Build: docker build -t vido-api ./apps/api
# Run:   docker run -p 8080:8080 -v vido-data:/vido-data vido-api
# ==============================================================================

# Stage 1: Build official C++ OpenCC helper
# ------------------------------------------------------------------------------
FROM alpine:3.21 AS opencc-builder
ARG OPENCC_VERSION=ver.1.4.2
RUN apk add --no-cache git cmake make g++ python3
WORKDIR /src
RUN git clone --depth 1 --branch "${OPENCC_VERSION}" https://github.com/BYVoid/OpenCC.git opencc
WORKDIR /src/opencc
RUN cmake -S . -B build -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX=/opt/opencc -DENABLE_GTEST=OFF \
    && cmake --build build --parallel \
    && cmake --install build

# Stage 2: Build
# ------------------------------------------------------------------------------
FROM golang:1.25-alpine AS builder

# Install build dependencies
# - git: required for go mod download of some packages
# - ca-certificates: for HTTPS requests
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
# - CGO_ENABLED=0: Use pure Go SQLite (modernc.org/sqlite)
# - -ldflags="-s -w": Strip debug info for smaller binary
# - -trimpath: Remove file paths from binary
# Buildx supplies the target architecture; do not hard-code amd64 so the
# same Dockerfile remains valid for arm64 NAS images.
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -trimpath \
    -o /api ./cmd/api

# Stage 3: Runtime
# ------------------------------------------------------------------------------
FROM alpine:3.21

# Install runtime dependencies
# - ca-certificates: for HTTPS requests to TMDb API
# - tzdata: for timezone support
RUN apk add --no-cache ca-certificates tzdata ffmpeg

# Create non-root user for security
# Using fixed UID/GID for consistent permissions across hosts
RUN addgroup -g 1000 vido && \
    adduser -u 1000 -G vido -D -h /home/vido vido

# Create required directories with correct ownership
# /vido-data: Database, cache files
# /vido-backups: Backup files
RUN mkdir -p /vido-data /vido-backups && \
    chown -R vido:vido /vido-data /vido-backups

# Copy binary from builder
COPY --from=builder /api /usr/local/bin/api
COPY --from=opencc-builder /opt/opencc/bin/opencc /usr/local/bin/opencc
COPY --from=opencc-builder /opt/opencc/lib/ /usr/local/lib/
COPY --from=opencc-builder /opt/opencc/share/opencc/ /usr/share/opencc/

# Set working directory
WORKDIR /home/vido

# Switch to non-root user
USER vido

# Expose API port
EXPOSE 8080

# Health check using wget (available in alpine)
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD wget -q --spider http://localhost:8080/health || exit 1

# Set default environment variables
ENV PORT=8080 \
    ENV=production \
    DB_PATH=/vido-data/vido.db \
    VIDO_OPENCC_BIN=/usr/local/bin/opencc \
    VIDO_OPENCC_CONFIG=/usr/share/opencc/s2twp.json

# Run the API server
CMD ["api"]
