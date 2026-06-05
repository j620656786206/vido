---
name: qBittorrent State Mapping
description: qBT 4.x + 5.0+ torrent state to Vido status mapping follows Sonarr/Radarr industry standard
type: project
---

qBittorrent 5.0 renamed `pausedDL`→`stoppedDL` and `pausedUP`→`stoppedUP`. Both versions must be handled.

**Why:** Sonarr/Radarr established industry standard; qBT 5.0+ states were causing 100% torrents to show as "downloading" via default fallback.

**How to apply:** When adding new qBT state handling, check both the 4.x and 5.0+ wiki pages. Key mapping:
- `stoppedUP`/`pausedUP`/`stalledUP` → completed
- `stoppedDL`/`pausedDL` → paused
- `uploading`/`forcedUP` → seeding
- `moving` (5.0+) → checking
- `allocating` → downloading

Reference: https://github.com/qbittorrent/qBittorrent/wiki/WebUI-API-(qBittorrent-5.0)
