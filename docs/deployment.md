# Vido Deployment Guide

This guide covers deploying Vido using Docker and Docker Compose.

## Prerequisites

- **Docker**: 20.10 or later
- **Docker Compose**: 2.0 or later (included with Docker Desktop)
- **Memory**: Minimum 512MB RAM available
- **Storage**: At least 1GB for application + space for your media library

## Quick Start

Deploy Vido in under 5 minutes:

```bash
# 1. Clone the repository (if not already done)
git clone https://github.com/your-org/vido.git
cd vido

# 2. Create environment file (optional - sensible defaults are provided)
cp .env.example .env

# 3. Start the application
docker-compose up -d

# 4. Access Vido
open http://localhost:8080
```

That's it! Vido should now be running at `http://localhost:8080`.

## Configuration

### Environment Variables

All configuration is done through environment variables. Copy `.env.example` to `.env` and modify as needed:

```bash
cp .env.example .env
```

#### Essential Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `WEB_PORT` | `8080` | Port for web interface |
| `MEDIA_PATH` | `./media` | Path to your media library |
| `TMDB_API_KEY` | (none) | TMDb API key for metadata |

#### Database Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_PATH` | `/vido-data/vido.db` | Database file path (inside container) |
| `DB_WAL_ENABLED` | `true` | Enable WAL mode |

### Media Library Setup

Mount your media library by setting `MEDIA_PATH`:

```bash
# Linux
MEDIA_PATH=/mnt/media

# Synology NAS
MEDIA_PATH=/volume1/media

# macOS
MEDIA_PATH=/Users/yourname/Movies
```

The media folder is mounted **read-only** for security.

## Deployment Scenarios

### Scenario 1: Quick Test (Default)

```bash
docker-compose up -d
```

Access at `http://localhost:8080`

### Scenario 2: Custom Ports

```bash
# .env file
WEB_PORT=9000
API_PORT=9001

docker-compose up -d
```

Access at `http://localhost:9000`

### Scenario 3: Production Deployment

```bash
# Use production overrides for resource limits and security
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

### Scenario 4: NAS Deployment (Synology/QNAP)

1. Copy the `docker-compose.yml` to your NAS
2. Create `.env` file:

```bash
WEB_PORT=8080
MEDIA_PATH=/volume1/video
TMDB_API_KEY=your_key_here
```

3. Run:

```bash
docker-compose up -d
```

## Volume Management

### Data Persistence

Vido uses Docker volumes for persistent storage:

| Volume | Purpose | Container Path |
|--------|---------|---------------|
| `vido-data` | Database, cache | `/vido-data` |
| `vido-backups` | Backup files | `/vido-backups` |

### Backup Data

```bash
# Create a backup of the database
docker run --rm -v vido-data:/data -v $(pwd):/backup alpine \
  tar czf /backup/vido-backup-$(date +%Y%m%d).tar.gz -C /data .
```

### Restore Data

```bash
# Restore from backup
docker-compose down
docker run --rm -v vido-data:/data -v $(pwd):/backup alpine \
  tar xzf /backup/vido-backup-20240115.tar.gz -C /data
docker-compose up -d
```

## Health Checks

### Check Service Status

```bash
# View container status
docker-compose ps

# Check API health
curl http://localhost:8080/api-health

# Check nginx health
curl http://localhost:8080/nginx-health
```

### Expected Health Response

```json
{
  "status": "healthy",
  "service": "vido-api",
  "database": {
    "status": "healthy",
    "latency": 1,
    "walEnabled": true,
    "walMode": "wal",
    "syncMode": "1",
    "openConnections": 1,
    "idleConnections": 0
  }
}
```

## Troubleshooting

### Container Won't Start

```bash
# Check logs
docker-compose logs vido-api
docker-compose logs vido-web

# Check if ports are in use
lsof -i :8080
```

### Database Issues

```bash
# Check database health
curl http://localhost:8080/api-health | jq '.database'

# If database is corrupted, restore from backup
docker-compose down
docker volume rm vido_vido-data
# Restore from backup as shown above
```

### Permission Issues

```bash
# On Linux, ensure the media path is readable
chmod -R a+rX /path/to/media

# Check volume permissions
docker run --rm -v vido-data:/data alpine ls -la /data
```

### Network Issues

```bash
# Verify internal network
docker network inspect vido_vido-network

# Test API from web container
docker-compose exec vido-web wget -q -O - http://vido-api:8080/health
```

## Updating

### Update to Latest Version

```bash
# Pull latest changes
git pull

# Rebuild and restart
docker-compose down
docker-compose build --no-cache
docker-compose up -d
```

### Rollback

```bash
# Checkout previous version
git checkout v1.0.0

# Rebuild
docker-compose down
docker-compose build
docker-compose up -d
```

## Production Checklist

Before deploying to production:

- [ ] Set `ENV=production` in `.env`
- [ ] Configure `TMDB_API_KEY` for metadata fetching
- [ ] Set appropriate `MEDIA_PATH` pointing to your media library
- [ ] Use production compose file: `-f docker-compose.prod.yml`
- [ ] Set up regular backups of `vido-data` volume
- [ ] Configure reverse proxy (nginx/traefik) for HTTPS
- [ ] Review and adjust resource limits in `docker-compose.prod.yml`

## Resource Requirements

| Environment | CPU | Memory | Storage |
|-------------|-----|--------|---------|
| Minimum | 1 core | 512MB | 1GB |
| Recommended | 2 cores | 1GB | 5GB |
| Production | 2+ cores | 2GB | 10GB+ |

## Support

- **Issues**: [GitHub Issues](https://github.com/your-org/vido/issues)
- **Documentation**: See `docs/` folder
- **Architecture**: See `project-context.md`
