# Unraid Installation Guide

> **Last Updated:** 2026-10-21

## Prerequisites

- Unraid 6.x or later with Docker enabled
- A media folder (e.g. `/mnt/user/media`). **It must be writable**: generated subtitles are saved next to the video file
- For AI subtitles, one translation key: **Claude or Gemini** (Gemini runs on your own quota; on the free tier Vido paces itself). Videos with no subtitle track at all are transcribed first, which needs an ElevenLabs key (recommended) or an OpenAI key
- No TMDb key needed — one is bundled in the image. Set `TMDB_API_KEY` only if you want to use your own

Vido is **not** listed in Community Applications yet. Install it from the template in this repository (Method A) or by hand (Method B).

## Method A: Template from this repository (Recommended)

1. In the Unraid web UI, go to **Docker**, scroll to the bottom and open **Template repositories**
2. Add `https://github.com/j620656786206/vido` and click **Save**
3. Click **Add Container**, pick **Vido** from the **Template** dropdown
4. Fill in the paths and keys described in [Configuration](#configuration)
5. Click **Apply**

If the template does not show up, use Method B — the template file is [`unraid-template/vido.xml`](../unraid-template/vido.xml) and every field below matches it.

## Method B: Manual Docker Setup

1. In the Unraid web UI, go to **Docker** > **Add Container**
2. Set the following fields:

| Field            | Value                                         |
| ---------------- | --------------------------------------------- |
| Name             | `Vido`                                        |
| Repository       | `ghcr.io/j620656786206/vido:latest`           |
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

| Name          | Container Path  | Host Path                                | Mode           | Description                                                                                              |
| ------------- | --------------- | ---------------------------------------- | -------------- | -------------------------------------------------------------------------------------------------------- |
| App Data      | `/vido-data`    | `/mnt/user/appdata/vido`                 | Read/Write     | SQLite database, cache, and app data                                                                     |
| Backups       | `/vido-backups` | `/mnt/user/appdata/vido/backups`         | Read/Write     | Database backups and metadata exports                                                                    |
| Media Library | `/media`        | Your media path (e.g. `/mnt/user/media`) | **Read/Write** | Your movie and TV library. Generated subtitles are written beside the video, so this cannot be read-only |

### Environment Variables

**Keys (all optional — Vido starts without any of them):**

| Variable             | Default | Description                                                                                                                              |
| -------------------- | ------- | ---------------------------------------------------------------------------------------------------------------------------------------- |
| `CLAUDE_API_KEY`     | (empty) | Subtitle translation (default provider). One of Claude / Gemini is needed for AI subtitles                                               |
| `GEMINI_API_KEY`     | (empty) | Subtitle translation on your own Gemini quota, and AI filename parsing                                                                   |
| `ELEVENLABS_API_KEY` | (empty) | Speech recognition (ElevenLabs Scribe, ~US$0.26/h incl. the name vocabulary) for videos with no subtitle track. Recommended over Whisper |
| `OPENAI_API_KEY`     | (empty) | Speech recognition with OpenAI Whisper (~US$0.36/h); used when no ElevenLabs key is set                                                  |
| `TMDB_API_KEY`       | (empty) | Your own TMDb key. Leave empty to use the one bundled in the image                                                                       |
| `ENCRYPTION_KEY`     | (empty) | **Exactly 32 characters.** Required to save keys from the web UI (Settings → 金鑰設定); without it that page is read-only                |
| `VIDO_AUTH_PASSWORD` | (empty) | Shared login password for the whole app. Set it before sharing the instance or exposing it beyond your LAN                               |

**Behaviour:**

| Variable                      | Default      | Description                                                                                                                      |
| ----------------------------- | ------------ | -------------------------------------------------------------------------------------------------------------------------------- |
| `PUID` / `PGID`               | `99` / `100` | User the container runs as (Unraid's `nobody:users`). Must be able to write the media folder                                     |
| `TMDB_DEFAULT_LANGUAGE`       | `zh-TW`      | Default metadata language (ISO 639-1)                                                                                            |
| `VIDO_SUBTITLE_PIPELINE_MODE` | `pipeline`   | `pipeline` reads the embedded English track first, then falls back to speech recognition. `legacy` = online subtitle search only |
| `VIDO_ASR_PROVIDER`           | `auto`       | Which speech recogniser to use when both keys are set: `auto` (ElevenLabs if present), `elevenlabs`, `openai`                    |
| `ASR_BASE_URL` / `ASR_MODEL`  | (empty)      | Point at a self-hosted OpenAI-compatible speech recogniser (e.g. Speaches) instead of a cloud key                                |
| `AI_RUN_BUDGET_USD`           | `5`          | Spend ceiling per run. Work done before the ceiling is kept; the next run only does the rest                                     |
| `AI_PROVIDER`                 | `gemini`     | Which provider AI filename parsing prefers when both keys exist; if it has no key, the other one is used automatically           |
| `VIDO_LOG_LEVEL`              | `info`       | Set to `debug` for troubleshooting                                                                                               |

## Post-Installation Verification

1. Start the Vido container from the Docker tab
2. Wait for the health check to pass (green icon, usually within 30 seconds)
3. Open the WebUI at `http://[YOUR-UNRAID-IP]:8088`
4. The first visit opens a six-step setup wizard (welcome · qBittorrent · media library · API keys · anonymous usage report · done). Every step can be skipped and finished later under **Settings**
5. Go to the media library and trigger your first scan

### Health Check

The container includes a built-in health check that queries `GET /health` every 30 seconds. If the health check fails 3 times consecutively, Docker marks the container as unhealthy.

## Updating

`latest` is the newest release: it only changes when a new version is published, not on everyday code changes. To update, click **Check for Updates** on the **Docker** page, then **apply update** if one is offered. Back up `/mnt/user/appdata/vido` first.

To stay on one version, set Repository to a version tag instead (for example `ghcr.io/j620656786206/vido:0.2.0` — **no `v`**; `0.2` follows patch releases). `main` is the development build and changes with every change; don't use it.

## Troubleshooting

### Container won't start

- Check the Docker log for error messages (click the Vido container icon > **Log**)
- Verify the App Data path exists and is writable
- Ensure port 8088 is not already in use by another container (8080 is often taken by qBittorrent)

### Media files not found

- Verify the Media Library path points to the correct directory on your Unraid server
- Supported video formats: `.mkv`, `.mp4`, `.avi`, `.m4v`, `.ts`, `.wmv`, `.flv`, `.webm`, `.mov`

### Subtitle generation fails with "folder not writable"

- The media path is mounted read-only, or the `PUID`/`PGID` user cannot write to it. Subtitles are saved next to the video file, so the media folder must be writable

### The settings page cannot save keys

- `ENCRYPTION_KEY` is missing or not exactly 32 characters

### Permission errors

- The container starts as root, takes ownership of `/vido-data` and `/vido-backups`, then drops to `PUID`/`PGID` (template default `99:100`, Unraid's `nobody:users`)
- If you changed `PUID`/`PGID`, make sure that user can read and write the App Data, Backups **and** media paths

### Database errors

- The SQLite database is stored at `/vido-data/vido.db`
- WAL mode is enabled by default for better performance
- If the database is corrupted, restore from a backup in `/vido-backups`

## Security Notes

- After start-up the process runs as `PUID`/`PGID`, not root
- The root filesystem is **read-only** (`--read-only` flag); only `/tmp`, the data, backup and media paths are writable
- Vido writes **only subtitle files** into the media folder, next to the videos; it never modifies or deletes video files
- API keys are masked in the Unraid UI (not shown in plain text)
- Set `VIDO_AUTH_PASSWORD` before exposing the port beyond your home network
