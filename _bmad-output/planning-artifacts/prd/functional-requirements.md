# Functional Requirements

## Media Search & Discovery

**MVP:**
- FR1: Users can search for movies and TV shows by title (Traditional Chinese or English)
- FR2: Users can view search results with Traditional Chinese metadata (title, description, release year, poster, genre, director, cast)
- FR3: Users can browse search results in grid view
- FR4: Users can view media item detail pages (read-only)

**1.0:**
- FR5: Users can search within their saved media library
- FR6: Users can sort media library by date added, title, year, rating
- FR7: Users can filter media library by genre, year, media type
- FR8: Users can toggle between grid view and list view

**Growth:**
- FR9: Users can receive smart recommendations based on genre, cast, director
- FR10: Users can see "similar titles" suggestions

---

## Filename Parsing & Metadata Retrieval

**MVP:**
- FR11: System can parse standard naming convention filenames (e.g., `Movie.Name.2024.1080p.BluRay.mkv`)
- FR12: System can extract title, year, season/episode from filenames
- FR13: System can retrieve Traditional Chinese priority metadata from TMDb API
- FR14: System can store metadata to local database

**1.0:**
- FR15: System can parse fansub naming conventions using AI (Gemini/Claude) (e.g., `[Leopard-Raws] Show - 01 (BD 1920x1080 x264 FLAC).mkv`)
- FR16: System can implement multi-source metadata fallback (TMDb → Douban → Wikipedia → AI → Manual)
- FR17: System can automatically switch to Douban web scraper when TMDb fails
- FR18: System can retrieve metadata from Wikipedia when TMDb and Douban fail
- FR19: System can use AI to re-parse filenames and generate alternative search keywords
- FR20: Users can manually search and select correct metadata
- FR21: Users can manually edit media item metadata
- FR22: Users can view parse status indicators (success/failure/processing)
- FR23: System can cache AI parsing results to reduce API costs
- FR24: System can learn from user manual corrections and remember filename mapping rules
- FR25: System can automatically retry when metadata sources are temporarily unavailable
- FR26: System can gracefully degrade when all sources fail and provide manual option

---

## Download Integration & Monitoring

**1.0:**
- FR27: Users can connect to qBittorrent instance (enter host, username, password)
- FR28: Users can test qBittorrent connection
- FR29: System can monitor qBittorrent download status in real-time (progress, speed, ETA, status)
- FR30: Users can view download list in unified dashboard
- FR31: Users can filter downloads by status (downloading, paused, completed, seeding)
- FR32: System can detect completed downloads and trigger parsing
- FR33: System can display qBittorrent connection health status

**Growth:**
- FR34: Users can control qBittorrent directly from Vido (pause/resume/delete torrents)
- FR35: Users can adjust download priority
- FR36: Users can manage bandwidth settings
- FR37: Users can schedule downloads

---

## Media Library Management

**1.0:**
- FR38: Users can browse complete media library collection
- FR39: Users can view media detail pages (cast info, trailers, complete metadata)
- FR40: Users can perform batch operations on media items (delete, re-parse)
- FR41: Users can view recently added media items
- FR42: System can display metadata source indicators (TMDb/Douban/Wikipedia/AI/Manual)

**Growth:**
- FR43: Users can track personal watch history
- FR44: System can display watch progress indicators
- FR45: Users can mark media as watched/unwatched
- FR46: Users can create custom collections of media items

---

## System Configuration & Management

**MVP:**
- FR47: Users can deploy Vido via Docker container
- FR48: System can provide zero-config startup (sensible defaults)
- FR49: Users can configure media folder locations
- FR50: Users can configure API keys via environment variables
- FR51: System can store sensitive data in encrypted format (AES-256)

**1.0:**
- FR52: Users can complete initial setup via setup wizard
- FR53: Users can manage cache (view cache size, clear old cache)
- FR54: Users can view system logs
- FR55: System can display service connection status (qBittorrent, TMDb, AI APIs)
- FR56: Users can receive automatic update notifications
- FR57: System can backup database and configuration
- FR58: Users can restore data from backup
- FR59: System can verify backup integrity (checksum)
- FR60: System can export metadata to JSON/YAML format
- FR61: System can import metadata from JSON/YAML
- FR62: System can export metadata as NFO files (Kodi/Plex/Jellyfin compatible)
- FR63: Users can configure backup schedule (daily/weekly)
- FR64: System can automatically cleanup old backups (retention policy)
- FR65: System can display performance metrics (query latency, cache hit rate)
- FR66: System can warn when approaching scalability limits (e.g., SQLite items >8,000)

---

## User Authentication & Access Control

**1.0:**
- FR67: Users must authenticate via password/PIN to access Vido
- FR68: System can manage user sessions with secure tokens
- FR69: API endpoints must be protected with authentication tokens
- FR70: System can implement rate limiting to prevent abuse

**Growth:**
- FR71: System can support multiple user accounts
- FR72: Administrators can manage user permissions (admin/user roles)
- FR73: Users can have personal watch history
- FR74: Users can have personal preference settings

---

## Subtitle Management (Growth - Post-1.0)

- FR75: Users can search for subtitles (OpenSubtitles and Zimuku)
- FR76: System can prioritize Traditional Chinese subtitles
- FR77: Users can download subtitle files
- FR78: Users can manually upload subtitle files
- FR79: System can automatically download subtitles (based on user preferences)
- FR80: System can display subtitle availability status

---

## Automation & Organization (Growth - Post-1.0)

- FR81: System can monitor watch folders to detect new files
- FR82: System can automatically trigger parsing when files are detected
- FR83: System can automatically rename files based on user-configured patterns
- FR84: System can automatically move files to organized directory structure
- FR85: System can execute automation tasks in background processing queue
- FR86: Users can configure automation rules (watch folders, naming patterns, target folders)

---

## External Integration & Extensibility (Growth - Post-1.0)

- FR87: System can provide RESTful API (versioned /api/v1)
- FR88: Developers can authenticate API requests with API tokens
- FR89: System can provide OpenAPI/Swagger documentation
- FR90: System can support webhook subscriptions for external automation
- FR91: Users can export metadata to Plex/Jellyfin
- FR92: System can sync watch status with Plex/Jellyfin
- FR93: Users can access Vido via mobile application (React Native/Flutter)
- FR94: Users can remotely control downloads from mobile device
