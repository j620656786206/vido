# User Journeys

## Journey 1: Alex — 字幕自動化之旅 (Subtitle Automation)

**Setting:** Alex downloads a new anime series via qBittorrent. He wants subtitles ready by the time he opens Plex.

### 🎬 Act 1 — Scanner Detects New Media

1. qBittorrent finishes downloading a batch of anime episodes
2. Vido's media scanner detects new files in the configured media directory
3. The filename parser engine processes each file:
   - Standard names (e.g., `Demon.Slayer.S01E26.1080p.BluRay.mkv`) are parsed instantly
   - Fansub names (e.g., `[Leopard-Raws] Kimetsu no Yaiba - 26 (END) (BD 1920x1080 x264 FLAC).mkv`) trigger AI-powered parsing
4. Each parsed result is matched against TMDB for metadata, with Traditional Chinese metadata prioritized
5. **Key features exercised:** P1-001, P1-002, P1-003, P1-004, P1-005

### 🔍 Act 2 — Subtitle Engine Searches

6. For each matched media item, the subtitle engine kicks off an automated search
7. Multiple sources are queried in parallel: Assrt, Zimuku, OpenSubtitles
8. Results are filtered with strict 簡/繁 identification — no Simplified Chinese false positives
9. Best match is selected based on format compatibility, source reliability, and language accuracy
10. **Key features exercised:** P1-010, P1-011, P1-012, P1-013

### ✨ Act 3 — Conversion and Placement

11. If the best available subtitle is in Simplified Chinese, OpenCC converts it to Traditional Chinese
12. Cross-strait terminology correction is applied (e.g., 程式 vs 程序, 記憶體 vs 內存)
13. The subtitle file is renamed to match the media file and placed in the correct directory
14. The `.srt` / `.ass` file uses proper naming so Plex/Jellyfin auto-detects it
15. **Key features exercised:** P1-014, P1-015, P1-016, P1-017

### 📺 Act 4 — Ready to Watch

16. Alex opens Plex (or Jellyfin) — the subtitle is already attached and ready
17. Traditional Chinese metadata (title, plot summary, poster) displays correctly
18. If no subtitle was found automatically, Alex opens Vido's manual search UI
19. He searches by show name, browses available subtitles, and downloads the best match manually
20. The manual selection is remembered for future episodes of the same series
21. **Key features exercised:** P1-006, P1-007, P1-018, P1-019

---

## Journey 2: Alex — 媒體探索之旅 (Media Discovery)

**Setting:** Alex wants to find new shows to watch this weekend. He opens Vido to browse and discover content.

### 🏠 Act 1 — Homepage Experience

1. Alex opens Vido's homepage in his browser
2. A Hero Banner showcases trending content, curated for the zh-TW audience
3. Below the banner, custom explore blocks display curated categories:
   - "熱門日韓劇" (Popular J/K-Drama)
   - "本季新番" (This Season's Anime)
   - "高分電影" (Top-Rated Movies)
4. All titles, descriptions, and metadata are in Traditional Chinese
5. **Key features exercised:** P2-001, P2-002, P2-003, P2-004, P2-005

### 🔎 Act 2 — Browse and Filter

6. Alex taps into the "熱門日韓劇" block to browse more
7. He uses the multi-filter chip UI to narrow results:
   - Genre: 懸疑 (Thriller)
   - Year: 2025-2026
   - Region: 韓國 (South Korea)
8. Results update instantly with server-side filtering
9. He scrolls through the grid at a smooth 60 FPS
10. **Key features exercised:** P2-006, P2-007, P2-008, P2-009, P2-010

### 📋 Act 3 — Detail Page Deep Dive

11. Alex taps on a show that looks interesting
12. The detail page displays:
    - TMDB rating alongside 豆瓣 (Douban) rating for a dual perspective
    - Available streaming platforms (Netflix, Disney+, etc.)
    - Trailer embed
    - Cast and crew with Traditional Chinese names
    - Subtitle availability status
13. He reads the zh-TW plot summary and checks the Douban score
14. **Key features exercised:** P2-015, P2-016, P2-017, P2-018, P2-019, P2-020

### 🚀 Act 4 — One-Click Request

15. Alex decides he wants to watch this show
16. He clicks the "Request" button — a single click
17. The request is routed to Sonarr (or Radarr for movies) via the pluggable integration layer
18. Sonarr picks up the request, searches indexers via Prowlarr, and sends the torrent to qBittorrent
19. **Key features exercised:** P2-021, P2-022, P2-023, P3-001, P3-002, P3-003

### 🎉 Act 5 — Full Automation Loop

20. qBittorrent downloads the series; progress is visible in Vido's download manager
21. Download completes — Vido's scanner auto-detects the new files
22. Filename parsing, TMDB matching, and subtitle search all trigger automatically
23. By the time Alex sits down to watch, the show is in his library with zh-TW subtitles attached
24. **Key features exercised:** P3-004, P3-005, P2-024, P2-025

---

## Journey 3: Alex — 系統管理之旅 (System Administration)

**Setting:** Alex periodically checks his NAS media system health through Vido's dashboard.

### 📊 Act 1 — Dashboard Overview

1. Alex opens Vido's Dashboard page
2. Media library statistics are displayed at a glance:
   - Total movies and TV series count
   - Resolution distribution chart (4K / 1080p / 720p / SD)
   - Storage usage breakdown by media type
   - Average file size metrics
3. Subtitle coverage rate is shown as a visual chart:
   - Has zh-TW subtitle / Has other subtitle / No subtitle
4. Recently added media from the last 7 and 30 days is listed
5. **Key features exercised:** P4-001, P4-002, P4-003, P4-004

### 🔗 Act 2 — Service Health Check

6. Alex checks the service connectivity panel
7. Integration status is shown for each connected service:
   - qBittorrent: Connected, 2 active downloads, 5 seeding
   - Plex: Synced, last sync 3 minutes ago
   - Sonarr/Radarr: Connected, 0 pending requests
8. Disk space indicator shows 72% used with a projected "days until full" estimate
9. If any service is unreachable, a warning badge appears with last-known-good timestamp
10. **Key features exercised:** P4-005, P4-006, P4-007, P4-008

### 📝 Act 3 — Activity Log Review

11. Alex opens the activity log
12. Today's activity summary:
    - 3 subtitles downloaded (2 auto, 1 manual)
    - 1 media scan completed (12 new files detected)
    - 1 failed subtitle search (queued for retry)
13. He can filter logs by type: scan, subtitle, download, system
14. Each log entry links to the relevant media item for quick access
15. **Key features exercised:** P4-009, P4-010, P4-011, P4-012

### ⚙️ Act 4 — Settings and Configuration

16. Alex navigates to Settings
17. He adjusts the media scan schedule from every 6 hours to every 4 hours
18. He configures a new Prowlarr indexer that a friend recommended
19. He reviews subtitle source priority order: Assrt > Zimuku > OpenSubtitles
20. He checks cache usage and clears metadata cache older than 30 days
21. All changes take effect immediately — no restart required
22. **Key features exercised:** P4-013, P4-014, P4-015, P4-016, P4-017, P4-018, P4-019, P4-020, P4-021, P4-022, Settings
