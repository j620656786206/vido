# ==============================================================================
# Vido - Unified Multi-Stage Dockerfile
# ==============================================================================
# Single image containing Go API + React SPA for NAS deployment.
# Go serves static files directly — no Nginx required.
#
# Build: docker build -t vido .
# Run:   docker run -p 8080:8080 -v vido-data:/vido-data vido
#
# For development with separate containers, see:
#   apps/api/Dockerfile  (API only)
#   apps/web/Dockerfile  (Web + Nginx)
# ==============================================================================

# Stage 1: Build frontend
# ------------------------------------------------------------------------------
FROM node:20-alpine AS web-builder

RUN corepack enable && corepack prepare pnpm@latest --activate

WORKDIR /app

# Copy package files and workspace config for dependency installation
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY libs/shared-types/package.json ./libs/shared-types/

# Install dependencies (frozen lockfile for reproducible builds)
RUN pnpm install --frozen-lockfile

# Copy source files needed for build
COPY apps/web ./apps/web
COPY libs ./libs
COPY nx.json tsconfig.base.json ./

# Build the web application using Nx
# Output: dist/apps/web
# NX_DAEMON=false prevents Nx daemon crashes under QEMU arm64 emulation
RUN NX_DAEMON=false npx nx build web --configuration=production

# Stage 2: Build backend
# ------------------------------------------------------------------------------
FROM golang:1.25-alpine AS api-builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy go mod files first for better layer caching
COPY apps/api/go.mod apps/api/go.sum ./
RUN go mod download

# Copy source code
COPY apps/api/ ./

# Build the binary
# - CGO_ENABLED=0: Pure Go SQLite (modernc.org/sqlite)
# - -ldflags="-s -w": Strip debug info for smaller binary
# - -trimpath: Remove file paths from binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -trimpath \
    -o /api ./cmd/api

# Stage 3: Runtime
# ------------------------------------------------------------------------------
FROM alpine:3.21

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user (matching existing API Dockerfile)
RUN addgroup -g 1000 vido && \
    adduser -u 1000 -G vido -D -h /home/vido vido

# Create required directories with correct ownership
RUN mkdir -p /vido-data /vido-backups /app/public && \
    chown -R vido:vido /vido-data /vido-backups /app/public

# Copy binary from api-builder
COPY --from=api-builder /api /usr/local/bin/api

# Copy built web assets from web-builder
COPY --from=web-builder --chown=vido:vido /app/dist/apps/web /app/public

WORKDIR /home/vido

USER vido

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD wget -q --spider http://localhost:8080/health || exit 1

ENV PORT=8080 \
    ENV=production \
    DB_PATH=/vido-data/vido.db \
    VIDO_PUBLIC_DIR=/app/public

CMD ["api"]
