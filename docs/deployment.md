# Vido Deployment Guide

This guide covers deploying Vido using Docker and Docker Compose.

## Prerequisites

- **Docker**: 20.10 or later
- **Docker Compose**: 2.0 or later (included with Docker Desktop)
- **Memory**: Minimum 512MB RAM available
- **Storage**: At least 1GB for application + space for your media library

### What the image already includes

You do **not** need to install anything on the host beyond Docker:

- **ffmpeg / ffprobe are bundled in the image.** The subtitle pipeline uses
  `ffprobe` to enumerate embedded subtitle tracks and `ffmpeg` to extract them.
  Because they ship inside the container, a host without ffmpeg is fine — but
  note the inverse: if you build a custom image and drop them, subtitle
  extraction degrades **silently** (every item routes to "no text source"
  instead of failing loudly).
- **The image is multi-architecture** (`linux/amd64` + `linux/arm64`), so the
  same `docker compose pull` works on an x86 NAS (Synology DS920+, QNAP,
  Unraid) and on ARM boards without picking a different tag.

## Quick Start

For a NAS installation, use the published multi-architecture image. You do not
need to clone the repository or build the image yourself:

```bash
# 1. Download the NAS Compose file
curl -LO https://raw.githubusercontent.com/j620656786206/vido/main/docs/docker-compose.nas.yml

# 2. Edit the three host paths in the file, then start the application
docker compose -f docker-compose.nas.yml up -d

# 3. Access Vido on the NAS port selected in the file
open http://localhost:8088
```

That's it! Vido should now be running at `http://localhost:8088`.

## Configuration

### Environment Variables

All configuration is done through environment variables. Copy `.env.example` to `.env` and modify as needed:

```bash
cp .env.example .env
```

#### Essential Variables

| Variable       | Default | Description                                       |
| -------------- | ------- | ------------------------------------------------- |
| `VIDO_PORT`    | `8088`  | NAS host port mapped to the container's port 8080 |
| `MEDIA_PATH`   | —       | Path to your media library (set in Compose)       |
| `TMDB_API_KEY` | (none)  | TMDb API key for metadata                         |

#### Database Variables

| Variable                  | Default              | Description                                                                          |
| ------------------------- | -------------------- | ------------------------------------------------------------------------------------ |
| `DB_PATH`                 | `/vido-data/vido.db` | Database file path (inside container)                                                |
| `DB_WAL_ENABLED`          | `true`               | Enable WAL mode                                                                      |
| `VIDO_LOG_RETENTION_DAYS` | `14`                 | Days to keep `system_logs` rows; pruned at startup and periodically (`<=0` disables) |

#### Subtitle Generation Variables

Vido can generate a Traditional-Chinese (`zh-Hant`) subtitle for media that has
no matching subtitle online, by extracting the embedded track and translating
it. This is **off by default**:

| Variable                           | Default           | Description                                                                                                        |
| ---------------------------------- | ----------------- | ------------------------------------------------------------------------------------------------------------------ |
| `VIDO_SUBTITLE_PIPELINE_MODE`      | `legacy`          | `legacy` = subtitle search only (unchanged behaviour). `pipeline` = also generate subtitles.                       |
| `CLAUDE_API_KEY`                   | (none)            | Translation provider key. Required when the mode is `pipeline`.                                                    |
| `CLAUDE_MODEL`                     | `claude-sonnet-5` | Deployment-wide default translation model. Users can pick a different one per run; this sets what they start from. |
| `SUBTITLE_EXTRACT_TIMEOUT_SECONDS` | `600`             | Floor of one subtitle extraction (ffmpeg) in seconds. Small files use this.                                        |
| `SUBTITLE_EXTRACT_PER_GB_SECONDS`  | `30`              | Seconds allowed per GB of media. Past ~20 GB this is the one in force — raise it for large remuxes on slow disks.  |

Notes:

- **Scanning never generates subtitles.** Completing a library scan updates
  files and metadata and nothing else — it does not queue any paid work. This
  changed in story sub-4-1: the first production run of `pipeline` mode wired a
  library-wide sweep into the scan-complete callback, so one press of "掃描媒體庫"
  queued ~1000 items, most of them onto paid speech recognition, with no cost
  shown. Generation is now chosen explicitly, from a screen that shows the
  estimate first. Per-item manual generation is unaffected.
- **Unknown values fail startup on purpose.** A typo such as `pipelien` stops
  the server rather than quietly staying on `legacy` — otherwise "I enabled it
  and nothing happens" is indistinguishable from a broken pipeline.
- **Both are needed together.** `pipeline` without `CLAUDE_API_KEY` logs one
  line at startup and keeps the search-only behaviour; the manual trigger
  endpoint answers `409 AI_NOT_CONFIGURED`.
- **The generation batch follows the cheap route first (sub-4-2).** In
  `pipeline` mode, a batch started from the generation screen runs each item
  through extract→translate first and only falls back to paid speech
  recognition when no text source exists — so the per-route estimate shown
  before starting matches what actually runs. In `legacy` mode the batch keeps
  the previous transcription engine. Batches accept mixed movie and episode
  selections, and an optional user-approved budget ceiling (`budget_usd`) that
  overrides `AI_RUN_BUDGET_USD` for that batch; the ceiling is a soft cap —
  checked before each paid call, reaching it pauses the remaining items and
  keeps completed ones.
- Manual generation (per item, or an explicitly confirmed batch) runs at low
  concurrency so the NAS stays responsive. Progress arrives on the existing
  SSE streams.
- **What `AI_RUN_BUDGET_USD` covers (sub-5-1).** Every subtitle-generation
  path is metered and capped: the consent batch (one shared ceiling per batch,
  or the user-approved `budget_usd`), the manual per-item trigger and the
  automatic pipeline (one ceiling per item). Self-hosted speech recognition
  (`ASR_BASE_URL` set) records `$0` per audio minute — the estimate and the
  after-the-fact spend figure use the same rate by construction. **Deliberately
  NOT covered:** filename parsing, fansub parsing and keyword generation (both
  Gemini and Claude). These are background metadata calls with no user-consent
  boundary and no "run" to attach a ceiling to; they are unmetered by design
  (tracked as `backlog-parse-path-ai-metering` should observability counters
  ever be wanted).
- **API keys hot-reload (sub-5-2).** `CLAUDE_API_KEY` and `OPENAI_API_KEY` can
  be set — or replaced — from Settings → API Keys while the server is running:
  they are resolved per call (an encrypted stored key wins over the environment
  variable), so saving one takes effect immediately and no restart is needed.
  `TMDB_API_KEY` is the exception: it can be stored from that page, but the
  running metadata clients keep their boot-time value until a restart.
- **Self-hosted speech recognition needs no key (sub-5-2).** With `ASR_BASE_URL`
  pointed at an OpenAI-compatible engine (Speaches, WhisperLive, Subgen), leave
  `OPENAI_API_KEY` unset — the request carries no `Authorization` header at all,
  and spend is recorded as `$0`. Set the key only if your engine sits behind an
  authenticating proxy. `OPENAI_API_KEY` is required only for the paid hosted API.
- **Which model translates, and what it costs (sub-6-8a).** The default is
  `claude-sonnet-5`, chosen on measurement rather than taste: blind-scoring
  10,304 real cues from this library gave Sonnet 5 a 1.3% unusable rate and
  89.6% good, against Haiku 4.5's 3.6% and 71.8%. Sonnet costs about $0.48 per
  hour of runtime, Haiku about $0.18 — so the cheaper model is offered as a
  visible, per-run choice instead of a config flip nobody sees. The consent
  screen prices every model your keys can reach and shows the estimated
  processing time (Sonnet takes roughly 17% of a title's runtime, Haiku 11%).
  `GET /api/v1/settings/models` is the list; a run may name one with
  `model_id`. `CLAUDE_MODEL` still sets the deployment-wide starting point.
- **Extraction is bounded by file size (sub-6-3).** Pulling the subtitle track
  out of a large remux is pure disk I/O. Each ffmpeg pass gets
  `max(SUBTITLE_EXTRACT_TIMEOUT_SECONDS, SUBTITLE_EXTRACT_PER_GB_SECONDS × file size in GB)`
  — a 93 GB file gets about 46 minutes, a 4 GB file the 600 s floor.
  **Which knob to raise:** past roughly 20 GB the per-GB term is the one in
  force, so raising the floor alone changes nothing for a big file. The timeout
  message names the file size, the bound it hit, and the variable that would
  actually move it.
- **Extractions take turns (sub-6-3).** Two workers demuxing two 20 GB files at
  once fought over the same spindle and both timed out, where either alone took
  3½ minutes — so only one extraction runs at a time process-wide, and the
  progress stream shows 「等待抽軌（前方 N 件）」 while an item waits. Audio
  extraction for speech recognition takes turns in the same queue (it is ffmpeg
  on the same disk). Translation is **not** serialised: two items still
  translate at once. One consequence worth knowing: the queue has no priority,
  so a batch you start by hand can wait behind a large background extraction
  that was already running.
- Everything else in this section — including `ASR_BASE_URL`, `ASR_MODEL`,
  `AI_RUN_BUDGET_USD`, the two `SUBTITLE_EXTRACT_*` variables and
  `VIDO_SUBTITLE_PIPELINE_MODE` — has no settings-page toggle yet; those are
  environment variables and a restart is required to change them.

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
# .env file (or edit the `ports` entry in the Compose file)
VIDO_PORT=9000

docker compose -f docker-compose.nas.yml up -d
```

Access at `http://localhost:9000`

### Scenario 3: Production Deployment

```bash
# Use production overrides for resource limits and security
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

### Scenario 4: NAS Deployment (Synology/QNAP)

Use the platform-specific guides:

- [Synology DSM／Container Manager 安裝指南](synology-installation-guide.zh-TW.md)
- [QNAP QTS／Container Station 安裝指南](qnap-installation-guide.zh-TW.md)

Both guides use the same public image and the same three mounts: application
data, backups, and a read-only media library.

## Volume Management

### Data Persistence

Vido uses Docker volumes for persistent storage:

| Volume         | Purpose         | Container Path  |
| -------------- | --------------- | --------------- |
| `vido-data`    | Database, cache | `/vido-data`    |
| `vido-backups` | Backup files    | `/vido-backups` |

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

## File Permissions (subtitle generation writes next to your media)

The subtitle pipeline writes the generated `.zh-Hant.srt` **beside the video file**. Before it spends any AI credit it now runs a real write probe on that folder (a temporary `.vido-write-probe-*` file is created and removed). If the probe fails the item is recorded as `failed` with `SUBTITLE_TARGET_NOT_WRITABLE`, **nothing is charged**, and the consent list flags the row as「資料夾無法寫入」.

Two configurations trip it in practice:

- **Read-only media mount.** Unraid's default template maps `/media` with `Mode="ro"`; Synology/QNAP "read-only" shares do the same. Set the mapping to **Read/Write** (Unraid: edit the container → the `/media` path → Access Mode). Vido never deletes or rewrites your video files — it only adds subtitle sidecars (and `.bak` copies of ones it replaces).
- **Container user not in the folder's group.** The image runs as `PUID`/`PGID` (default `1000`/`1000`). On Unraid, media folders are usually `nobody:users` (gid `100`), so either set `PGID=100` on the container or grant the folder's group write access. Mode bits that _look_ writable are not enough — the probe creates a real file, exactly as the placer will.

Troubleshooting a `SUBTITLE_TARGET_NOT_WRITABLE`:

1. `docker exec Vido touch "/media/<that folder>/.probe" && docker exec Vido rm "/media/<that folder>/.probe"` — reproduce the probe as the container user.
2. Compare `docker exec Vido id` with `ls -ln` on the host folder.
3. Fix the mount mode or `PGID`, restart the container, re-run the item from the activity page.

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

| Environment | CPU      | Memory | Storage |
| ----------- | -------- | ------ | ------- |
| Minimum     | 1 core   | 512MB  | 1GB     |
| Recommended | 2 cores  | 1GB    | 5GB     |
| Production  | 2+ cores | 2GB    | 10GB+   |

## Support

- **Issues**: [GitHub Issues](https://github.com/j620656786206/vido/issues)
- **Documentation**: See `docs/` folder
- **Architecture**: See `project-context.md`
