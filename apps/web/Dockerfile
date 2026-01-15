# ==============================================================================
# Vido Web - Multi-Stage Dockerfile
# ==============================================================================
# This Dockerfile creates a minimal production image for the Vido Web frontend.
# Uses multi-stage build with Nginx for serving static files.
#
# IMPORTANT: This Dockerfile must be built from the project root to access
# the monorepo structure. Use: docker build -f apps/web/Dockerfile .
#
# Build: docker build -f apps/web/Dockerfile -t vido-web .
# Run:   docker run -p 80:80 vido-web
# ==============================================================================

# Stage 1: Build
# ------------------------------------------------------------------------------
FROM node:20-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

WORKDIR /app

# Copy package files for dependency installation
COPY package.json package-lock.json ./

# Install ALL dependencies (including devDependencies for build)
RUN npm ci

# Copy source files needed for build
COPY apps/web ./apps/web
COPY libs ./libs
COPY nx.json tsconfig.base.json ./

# Build the web application using Nx
# Output will be in dist/apps/web
RUN npx nx build web --configuration=production

# Stage 2: Runtime
# ------------------------------------------------------------------------------
FROM nginx:1.27-alpine

# Remove default nginx static files
RUN rm -rf /usr/share/nginx/html/*

# Copy built static files from builder
COPY --from=builder /app/dist/apps/web /usr/share/nginx/html

# Copy custom nginx configuration
COPY apps/web/nginx.conf /etc/nginx/nginx.conf

# Expose HTTP port
EXPOSE 80

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget -q --spider http://localhost/nginx-health || exit 1

# Run nginx in foreground
CMD ["nginx", "-g", "daemon off;"]
