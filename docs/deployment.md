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

| Variable       | Default | Description                                                                                                                                |
| -------------- | ------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| `VIDO_PORT`    | `8088`  | NAS host port mapped to the container's port 8080                                                                                          |
| `MEDIA_PATH`   | —       | Path to your media library (set in Compose)                                                                                                |
| `TMDB_API_KEY` | (none)  | Your own TMDb API key. **Optional since the release images ship with a bundled key** (see below) — set it only if you want your own quota. |

##### The bundled TMDb key

Official release images (`ghcr.io/j620656786206/vido`, GHCR only — there is no Docker Hub image) are built with a TMDb
API key compiled into the binary, so metadata, posters and zh-TW titles work
on a fresh install without applying for a TMDb developer account. Keys are
resolved in this order: **a key saved in Settings → API Keys → `TMDB_API_KEY`
→ the bundled key**; anything you set yourself is used instead of the bundled
one, and the settings page tells you which one is active (「內建」= bundled).

- **Licence.** The bundled key is registered to the Vido project under TMDb's
  free, **non-commercial** terms. Vido is a personal, self-hosted tool; if you
  run it in a commercial setting you must bring your own key. TMDb's required
  attribution (logo + "This product uses the TMDB API but is not endorsed or
  certified by TMDB") is already built into the UI.
- **Rate limits.** TMDb counts requests per key **and** IP, so self-hosters on
  different networks do not compete with each other. If you ever see the TMDb
  status turn to 速率限制 (rate limited), paste your own key in
  Settings → API Keys — it takes effect immediately, no restart.
- **Source and local builds have no bundled key.** `go run`, `docker compose
build` on your machine and CI test binaries all contain an empty key and
  behave as before: the setup wizard asks for one.

#### Database Variables

| Variable                  | Default              | Description                                                                          |
| ------------------------- | -------------------- | ------------------------------------------------------------------------------------ |
| `DB_PATH`                 | `/vido-data/vido.db` | Database file path (inside container)                                                |
| `DB_WAL_ENABLED`          | `true`               | Enable WAL mode                                                                      |
| `VIDO_LOG_RETENTION_DAYS` | `14`                 | Days to keep `system_logs` rows; pruned at startup and periodically (`<=0` disables) |

#### Subtitle Generation Variables

Vido can generate a Traditional-Chinese (`zh-Hant`) subtitle for media that has
no matching subtitle online, by extracting the embedded track and translating
it. This is **on by default** since 0.2 (`VIDO_SUBTITLE_PIPELINE_MODE=legacy` turns it off):

| Variable                                 | Default           | Description                                                                                                                                                                                                                                                                                                                                                                                               |
| ---------------------------------------- | ----------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `VIDO_SUBTITLE_PIPELINE_MODE`            | `pipeline`        | `pipeline` (default since 0.2) = read the embedded English track first, translate it, and only fall back to speech recognition when there is no text source. `legacy` = online subtitle search only (the pre-0.2 behaviour); the per-title 生成字幕 button then only ever listens to the audio. The 0.1 default was `legacy`, and a fresh install that followed any guide never read its embedded tracks. |
| `CLAUDE_API_KEY`                         | (none)            | Translation provider key. One of `CLAUDE_API_KEY` / `GEMINI_API_KEY` is needed for generation.                                                                                                                                                                                                                                                                                                            |
| `CLAUDE_MODEL`                           | `claude-sonnet-5` | Deployment-wide default translation model. Users can pick a different one per run; this sets what they start from.                                                                                                                                                                                                                                                                                        |
| `GEMINI_API_KEY`                         | (none)            | Lets the user pick a Gemini model for translation (gemini-translation-dispatch). On an install with only a Gemini key the default becomes `gemini-3.5-flash-lite`; with a Claude key the default stays Claude.                                                                                                                                                                                            |
| `VIDO_GEMINI_MIN_INTERVAL`               | `13`              | Seconds between Gemini translation calls. The free tier allows 5 requests a minute; 13 keeps headroom and the estimate shown before a run includes it. Set `0` on a paid tier.                                                                                                                                                                                                                            |
| `SUBTITLE_EXTRACT_TIMEOUT_SECONDS`       | `600`             | Floor of one subtitle extraction (ffmpeg) in seconds. Small files use this.                                                                                                                                                                                                                                                                                                                               |
| `SUBTITLE_EXTRACT_PER_GB_SECONDS`        | `30`              | Seconds allowed per GB of media. Past ~20 GB this is the one in force — raise it for large remuxes on slow disks.                                                                                                                                                                                                                                                                                         |
| `TRANSCRIPTION_RUN_TIMEOUT_SECONDS`      | `600`             | Floor of the part of a speech-recognition run that follows audio extraction (chunk split, recognition, translation, writeback).                                                                                                                                                                                                                                                                           |
| `TRANSCRIPTION_SECONDS_PER_MEDIA_MINUTE` | `30`              | Seconds allowed per minute of media for that same part. Past 20 minutes of media this is the one in force — a 157-minute film gets ~78 minutes.                                                                                                                                                                                                                                                           |

Notes:

- **Scanning never generates subtitles.** Completing a library scan updates
  files and metadata and nothing else — it does not queue any paid work. This
  changed in story sub-4-1: the first production run of `pipeline` mode wired a
  library-wide sweep into the scan-complete callback, so one press of "掃描媒體庫"
  queued ~1000 items, most of them onto paid speech recognition, with no cost
  shown. Generation is now chosen explicitly, from a screen that shows the
  estimate first. Per-item manual generation is unaffected.
- **Unknown values fail startup on purpose.** A typo such as `legcay` stops
  the server rather than quietly staying on `pipeline` — otherwise "I switched
  it off and nothing changed" is indistinguishable from a broken opt-out.
- **A translation key is still needed.** `pipeline` without a Claude or Gemini
  key logs one line at startup and keeps the search-only behaviour; the manual
  trigger endpoint answers `409 AI_NOT_CONFIGURED`.
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
- **API keys hot-reload (sub-5-2, TMDb since sub-7-7a, ElevenLabs since
  asr-scribe-b, Gemini since gemini-translation-dispatch).** `CLAUDE_API_KEY`,
  `GEMINI_API_KEY`, `ELEVENLABS_API_KEY`, `OPENAI_API_KEY` and
  `TMDB_API_KEY` can be set — or replaced — from
  Settings → API Keys while the server is running: they are resolved per call
  (an encrypted stored key wins over the environment variable, which wins over
  the bundled TMDb key), so saving one takes effect immediately and no restart
  is needed.
- **Show context in the translation prompt now includes the cast (sub-7-2a).**
  Since sub-7-3 Vido stores each matched title's TMDb cast; the translator
  now sees it (up to ten「Name（Character）」entries) next to the title, year,
  genres and countries, and an episode run also gets its own
  「S01E03 · Title」line. The cast is part of the per-show cache key, so a
  subtitle translated before this change is not served back as if it had
  been made with the cast: a **forced** re-generation re-translates (and is
  billed) with the fuller context. Nothing re-runs by itself — a finished
  sidecar next to the media still satisfies the pre-flight check, exactly as
  before.
- **Which cloud engine transcribes (asr-scribe-b).** With an ElevenLabs key
  the ASR leg runs on Scribe v2 (on the 2026-10 exam, `eval/asr-exam-2026-10`,
  it passed 3/3 where whisper-1 passed 0/3; $0.22/h base, $0.26/h with the
  character-name vocabulary Vido sends by default — see `VIDO_ASR_KEYTERMS`
  in docs/development.md — against Whisper's $0.36/h). An OpenAI
  key alone still means Whisper. Both set: `VIDO_ASR_PROVIDER` (`auto` /
  `elevenlabs` / `openai`), overridable from Settings → API Keys. A pinned
  engine without its key is _unconfigured_, never a silent fallback to the
  other company's bill. Estimates and the recorded spend both use the chosen
  engine's rate.
- **Self-hosted speech recognition needs no key (sub-5-2).** With `ASR_BASE_URL`
  pointed at an OpenAI-compatible engine (Speaches, WhisperLive, Subgen), leave
  `OPENAI_API_KEY` unset — the request carries no `Authorization` header at all,
  and spend is recorded as `$0`. Set the key only if your engine sits behind an
  authenticating proxy. `OPENAI_API_KEY` is required only for the paid hosted API.
- **Which model translates, and what it costs (sub-6-8a).** The default is
  `claude-sonnet-5`, chosen on measurement rather than taste: blind-scoring
  10,304 real cues from this library gave Sonnet 5 a 1.3% unusable rate and
  89.6% good, against Haiku 4.5's 3.6% and 71.8%. Sonnet costs about $0.48 per
  hour of runtime, Haiku about $0.18, when the file has an embedded text track
  to translate. When Vido has to listen to the audio instead, translation costs
  more — about $1.30 per hour on Sonnet and $0.43 on Haiku, plus speech
  recognition — because that path re-sends the title's context with every
  batch. Either way the cheaper model is offered as a
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
- **A speech-recognition run is bounded per phase, not as a whole**
  (disc-2026-09-transcription-run-5min-hard-timeout). Pulling the audio out of
  the file follows the two `SUBTITLE_EXTRACT_*` bounds above (same disk, same
  kind of full-file ffmpeg read — a 66.8 GB remux needed 4:55 for this alone).
  Everything after it — chunk split, recognition, translation, writeback — gets
  `max(TRANSCRIPTION_RUN_TIMEOUT_SECONDS, TRANSCRIPTION_SECONDS_PER_MEDIA_MINUTE × media minutes)`,
  the media length read from the audio just extracted. There is no whole-run
  deadline any more: the old fixed 5 minutes killed a 157-minute film before
  recognition even started. When one of these bounds fires, the failure is a
  single line that names the phase, the media length, the budget it hit and the
  variable to raise; a run stopped by the caller (a cancelled batch, shutdown)
  says so instead of blaming a setting.
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

The media folder must be **writable**: generated subtitles are saved next to
the video file (see [File Permissions](#file-permissions-subtitle-generation-writes-next-to-your-media)).
Vido never modifies or deletes video files.

## Deployment Scenarios

### Scenario 1: Build from source (developers)

```bash
docker compose up -d
```

Builds the image from the checkout and serves it at `http://localhost:8080`.
A local build has **no bundled TMDb key** — the setup wizard asks for one.
NAS users should use the published image (Scenario 2 / 4) instead.

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
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

### Scenario 4: NAS Deployment (Synology/QNAP)

Use the platform-specific guides:

- [Synology DSM／Container Manager 安裝指南](synology-installation-guide.zh-TW.md)
- [QNAP QTS／Container Station 安裝指南](qnap-installation-guide.zh-TW.md)

Both guides (and the [Unraid guide](unraid-installation-guide.zh-TW.md)) use
the same public image and the same three mounts: application data, backups,
and a **writable** media library.

## Volume Management

### Data Persistence

Three paths hold everything Vido owns:

| Container path  | Purpose                                                           |
| --------------- | ----------------------------------------------------------------- |
| `/vido-data`    | SQLite database, cache, encrypted keys                            |
| `/vido-backups` | Database backups and metadata exports                             |
| `/media`        | Your library. Vido only **adds** subtitle files beside the videos |

The NAS Compose file and the Unraid template map the first two to host
folders you choose; the source-build `docker-compose.yml` uses named volumes
(`vido-data`, `vido-backups`).

### Backup

Stop the container, then copy the data and backups folders somewhere else —
do not back up only the container.

```bash
docker compose -f docker-compose.nas.yml stop
cp -a /path/to/vido/data /path/to/vido/backups /somewhere/safe/
docker compose -f docker-compose.nas.yml start
```

For the named volumes of a source build:

```bash
docker run --rm -v vido-data:/data -v $(pwd):/backup alpine \
  tar czf /backup/vido-backup-$(date +%Y%m%d).tar.gz -C /data .
```

### Restore

Stop the container, put the folder (or the tar contents) back in place, start
again. The database is a single file, `vido.db`, plus its `-wal`/`-shm`
companions — restore all of them together.

## Health Checks

```bash
# Container status
docker compose -f docker-compose.nas.yml ps

# The one health endpoint (the Docker health check calls the same URL)
curl http://localhost:8088/health
```

`/health` answers `200` with `"status": "healthy"` and a `database` block
(WAL mode, connection counts); it answers `503` when the database is
unhealthy. There is no separate web container, no nginx and no
`/api-health` — Vido serves the UI and the API from one process on one port.

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

### Container won't start

```bash
# Logs (the container is named `vido`; `Vido` on Unraid)
docker logs vido --tail 100

# Is the host port already taken?
lsof -i :8088
```

A typo in `VIDO_SUBTITLE_PIPELINE_MODE` or `VIDO_ASR_PROVIDER` stops the
server on purpose — the first log lines name the variable.

### Database issues

```bash
curl http://localhost:8088/health | jq '.database'
```

If the database is corrupted, stop the container, restore `vido.db` (and its
`-wal`/`-shm` files) from `/vido-backups` or your own copy, and start again.

### Permission issues

```bash
# What user does the container run as?
docker exec vido id

# Can it write where subtitles go?
docker exec vido touch "/media/<a show folder>/.probe" && docker exec vido rm "/media/<a show folder>/.probe"
```

`PUID`/`PGID` must be able to read and write `/vido-data`, `/vido-backups`
**and** the media folder. See
[File Permissions](#file-permissions-subtitle-generation-writes-next-to-your-media).

### The settings page cannot save keys

`ENCRYPTION_KEY` is missing or is not exactly 32 characters. The page shows
`writable: false` in that case and keys from the environment still work.

## Updating

### Published image (NAS)

```bash
docker compose -f docker-compose.nas.yml pull
docker compose -f docker-compose.nas.yml up -d
```

`latest` follows the main branch. To stay on a release, put its tag in the
Compose file instead: the published tags are `0.1.1` (exact), `0.1` (follows
patch releases), `0` and `latest` — there is **no `v` prefix** on image tags
(the git tags are `v0.1.1`).

### Rollback

Change the `image:` tag to the previous release (for example
`ghcr.io/j620656786206/vido:0.1.1`) and run `docker compose … up -d` again.
Keep a copy of `/vido-data` from before the upgrade in case a newer version
migrated the database.

### Source build

```bash
git pull
docker compose build --no-cache
docker compose up -d
```

## Production Checklist

Before leaving an install running unattended:

- [ ] Media folder mounted **writable**, and `PUID`/`PGID` can write to it
- [ ] One translation key set (`CLAUDE_API_KEY` or `GEMINI_API_KEY`), either in the environment or saved from Settings
- [ ] `ENCRYPTION_KEY` set (exactly 32 characters) if you want to manage keys from the web UI
- [ ] `VIDO_AUTH_PASSWORD` set before the port is reachable from outside your LAN; reverse proxy with HTTPS for remote access
- [ ] `AI_RUN_BUDGET_USD` at a ceiling you are comfortable with
- [ ] A scheduled copy of `/vido-data` and `/vido-backups`
- [ ] `ENV=production` (the published image already sets it)

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
