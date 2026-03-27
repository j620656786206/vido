# Unraid Installation Guide

> **Last Updated:** 2026-03-26

## Prerequisites

- Unraid 6.x or later with Docker enabled
- Community Applications plugin installed (recommended, for one-click install)
- A TMDb API key from <https://www.themoviedb.org/settings/api> (optional but recommended)

## Method A: Community Applications (Recommended)

1. Open the Unraid web UI and navigate to **Apps**
2. Search for **Vido**
3. Click **Install**
4. Configure the required paths (see [Configuration](#configuration) below)
5. Click **Apply**

## Method B: Manual Docker Setup

1. In the Unraid web UI, go to **Docker** > **Add Container**
2. Set the following fields:

| Field            | Value                                         |
| ---------------- | --------------------------------------------- |
| Name             | `Vido`                                        |
| Repository       | `ghcr.io/j620656786206/vido:main`             |
| Network Type     | `Bridge`                                      |
| WebUI            | `http://[IP]:[PORT:8088]`                     |
| Extra Parameters | `--read-only --tmpfs /tmp:size=64M,mode=1777` |

3. Add port, path, and variable mappings as described in the [Configuration](#configuration) section
4. Click **Apply**

## Configuration

### Port

| Name       | Container Port | Host Port             | Protocol |
| ---------- | -------------- | --------------------- | -------- |
| WebUI Port | `8080`         | `8088` (configurable) | TCP      |

Vido serves both the web UI and API on a single port.

### Paths (Volumes)

| Name          | Container Path  | Host Path                                | Mode          | Description                           |
| ------------- | --------------- | ---------------------------------------- | ------------- | ------------------------------------- |
| App Data      | `/vido-data`    | `/mnt/user/appdata/vido`                 | Read/Write    | SQLite database, cache, and app data  |
| Backups       | `/vido-backups` | `/mnt/user/appdata/vido/backups`         | Read/Write    | Database backups and metadata exports |
| Media Library | `/media`        | Your media path (e.g. `/mnt/user/media`) | **Read Only** | Your movie and TV show library        |

### Environment Variables

**Always Shown:**

| Variable                | Default | Description                                    |
| ----------------------- | ------- | ---------------------------------------------- |
| `TMDB_API_KEY`          | (empty) | TMDb API key for metadata, posters, and images |
| `TMDB_DEFAULT_LANGUAGE` | `zh-TW` | Default metadata language (ISO 639-1)          |

**Advanced (click "Show more settings"):**

| Variable         | Default   | Description                                            |
| ---------------- | --------- | ------------------------------------------------------ |
| `GIN_MODE`       | `release` | Set to `debug` for troubleshooting                     |
| `AI_PROVIDER`    | `gemini`  | AI provider for filename parsing: `gemini` or `claude` |
| `GEMINI_API_KEY` | (empty)   | Google Gemini API key for AI parsing                   |
| `CLAUDE_API_KEY` | (empty)   | Anthropic Claude API key (alternative to Gemini)       |

## Post-Installation Verification

1. Start the Vido container from the Docker tab
2. Wait for the health check to pass (green icon, usually within 30 seconds)
3. Open the WebUI at `http://[YOUR-UNRAID-IP]:8088`
4. Navigate to **Settings** and configure your TMDb API key
5. Go to the media library and trigger your first scan

### Health Check

The container includes a built-in health check that queries `GET /health` every 30 seconds. If the health check fails 3 times consecutively, Docker marks the container as unhealthy.

## Troubleshooting

### Container won't start

- Check the Docker log for error messages (click the Vido container icon > **Log**)
- Verify the App Data path exists and is writable
- Ensure port 8088 is not already in use by another container (8080 is often taken by qBittorrent)

### Media files not found

- Verify the Media Library path points to the correct directory on your Unraid server
- The media path is mounted read-only — Vido only reads, never modifies your media files
- Supported video formats: `.mkv`, `.mp4`, `.avi`, `.m4v`, `.ts`, `.wmv`, `.flv`, `.webm`, `.mov`

### Permission errors

- The container runs as user `vido` (UID 1000, GID 1000)
- Ensure the App Data and Backups directories are accessible by UID 1000
- If needed, run `chown -R 1000:1000 /mnt/user/appdata/vido` from the Unraid terminal

### Database errors

- The SQLite database is stored at `/vido-data/vido.db`
- WAL mode is enabled by default for better performance
- If the database is corrupted, restore from a backup in `/vido-backups`

## Security Notes

- The container runs as a **non-root user** (UID 1000)
- The root filesystem is **read-only** (`--read-only` flag)
- Media files are mounted **read-only** — Vido never modifies your media
- API keys are masked in the Unraid UI (not shown in plain text)
