# Completed Work Registry

This document maps all 50 completed stories from v3 Epics 1-6 to their corresponding v4 feature IDs, documenting what each story provides to the v4 codebase.

---

## Epic 1: Project Foundation & Docker Deployment (5 stories)

| Story ID | Story Name | v4 Feature ID | What It Provides |
|----------|-----------|---------------|-----------------|
| 1-1 | Repository Pattern Database Abstraction Layer | Infrastructure | SQLite repositories with interface-based abstraction for future PostgreSQL migration |
| 1-2 | Docker Compose Production Configuration | Infrastructure | Multi-stage Docker build, docker-compose.yml, volume mounts, health checks |
| 1-3 | Environment Variable Configuration System | Infrastructure | Viper-based config with env var overrides, sensible defaults, validation |
| 1-4 | Secrets Management with AES-256 Encryption | Infrastructure | Encrypted storage for API keys, encryption key from env var or machine-ID fallback |
| 1-5 | Media Folder Configuration | Infrastructure | Configurable media library paths, folder validation, persistence |

---

## Epic 2: Media Search & Traditional Chinese Metadata (6 stories)

| Story ID | Story Name | v4 Feature ID | What It Provides |
|----------|-----------|---------------|-----------------|
| 2-1 | TMDb API Integration with zh-TW Priority | P1-003 | TMDB client with zh-TW language priority, rate limiting, 24h response cache |
| 2-2 | Media Search Interface | P1-007 | Search input with debounce, search results display, TMDB search API endpoint |
| 2-3 | Search Results Grid View | P1-007 | Responsive poster grid, media cards with title/year/rating, virtual scrolling |
| 2-4 | Media Detail Page | P1-007 | Detail page with poster, backdrop, cast, description, genres, ratings |
| 2-5 | Standard Filename Parser (Regex-based) | P1-002 | Regex parser for standard naming (Movie.Name.2024.1080p.BluRay.mkv), title/year/season/episode extraction |
| 2-6 | Media Entity and Database Storage | P1-003 | Media model, SQLite schema, CRUD repository, migration system |

---

## Epic 3: AI-Powered Fansub Parsing & Multi-Source Fallback (12 stories)

| Story ID | Story Name | v4 Feature ID | What It Provides |
|----------|-----------|---------------|-----------------|
| 3-1 | AI Provider Abstraction Layer | P1-002 | Provider interface for Gemini/Claude, 30-day result caching, usage tracking |
| 3-2 | AI Fansub Filename Parsing | P1-002 | AI-powered parsing for fansub naming conventions ([Leopard-Raws], 【幻櫻字幕組】) |
| 3-3 | Multi-Source Metadata Fallback Chain | P1-004 | Orchestrator: TMDB → Douban → Wikipedia → AI → Manual fallback |
| 3-4 | Douban Web Scraper | P1-004 | Douban metadata scraping with anti-detection, rate limiting, zh-TW data |
| 3-5 | Wikipedia Metadata Fallback | P1-004 | Wikipedia MediaWiki API client, infobox parsing, zh-TW article extraction |
| 3-6 | AI Search Keyword Generation | P1-002 | AI generates alternative search keywords when initial parsing fails |
| 3-7 | Manual Metadata Search and Selection | P1-007 | UI for manual TMDB/Douban search when auto-matching fails |
| 3-8 | Metadata Editor | P1-007 | Edit media metadata fields (title, year, genre, description) with save |
| 3-9 | Filename Mapping Learning System | P1-002 | Remember user corrections, auto-apply learned mappings to similar filenames |
| 3-10 | Parse Status Indicators | P1-007 | Visual indicators (success/failure/processing) in library grid |
| 3-11 | Auto-Retry Mechanism | P1-004 | Exponential backoff retry for failed metadata sources |
| 3-12 | Graceful Degradation | P1-004 | Always provides next step when sources fail, manual option always available |

---

## Epic 4: qBittorrent Download Monitoring (6 stories)

| Story ID | Story Name | v4 Feature ID | What It Provides |
|----------|-----------|---------------|-----------------|
| 4-1 | qBittorrent Connection Configuration | P3-010 | Connection config UI (host, port, username, password), connection test |
| 4-2 | Real-Time Download Status Monitoring | P3-010 | Polling-based download status (progress, speed, ETA, state) |
| 4-3 | Unified Download Dashboard | P3-010 | Download list view with progress bars, speed indicators, status badges |
| 4-4 | Download Status Filtering | P3-010 | Filter by status (downloading, paused, completed, seeding) |
| 4-5 | Completed Download Detection and Parsing Trigger | P3-010 | Auto-detect completed downloads, trigger filename parsing pipeline |
| 4-6 | Connection Health Monitoring | P3-010 | Health check with <10s detection, reconnection with 30s backoff |

---

## Epic 5: Media Library Management (11 stories)

| Story ID | Story Name | v4 Feature ID | What It Provides |
|----------|-----------|---------------|-----------------|
| 5-0 | Navigation Shell & Layout Foundation | P1-007 | App shell with sidebar navigation, responsive layout, route structure |
| 5-1 | Media Library Grid View | P1-007 | Library grid with poster cards, virtual scrolling, empty state |
| 5-2 | Library List View Toggle | P1-007 | List view with table layout, toggle between grid/list, preference persistence |
| 5-3 | Library Search | P1-007 | FTS5 full-text search, search input with debounce, highlight matches |
| 5-4 | Library Sorting | P1-007 | Sort by date added, title, year, rating with ascending/descending toggle |
| 5-5 | Library Filtering | P1-007 | Filter by genre, year range, media type with chip UI |
| 5-6 | Media Detail Page (Full Version) | P1-007 | Enhanced detail page with cast, trailers, metadata source indicators |
| 5-7 | Batch Operations | P1-007 | Multi-select with batch delete, re-parse, export actions |
| 5-8 | Recently Added Section | P1-007 | Homepage section showing recently added media |
| 5-9 | Context Menu Actions | P1-007 | Right-click context menu on media cards with quick actions |
| 5-10 | Hover Detail Preview | P1-007 | Hover card showing media details without navigation |

---

## Epic 6: System Configuration & Backup (10 stories)

| Story ID | Story Name | v4 Feature ID | What It Provides |
|----------|-----------|---------------|-----------------|
| 6-0 | Settings Page Shell & Navigation | Settings | Settings page layout with tabbed navigation |
| 6-1 | Setup Wizard | Settings | Guided initial setup (<5 steps), media path config, API key entry |
| 6-2 | Cache Management | Settings | View cache size by category, clear cache actions, auto-cleanup policies |
| 6-3 | System Logs Viewer | Settings | Log viewer with severity filtering, search, auto-scroll, export |
| 6-4 | Service Connection Status Dashboard | Settings | Connection status panel for qBittorrent, TMDB, AI APIs |
| 6-5 | Database Backup | Settings | SQLite .backup command, manual trigger, backup file management |
| 6-6 | Backup Integrity Verification | Settings | SHA-256 checksum verification for backup files |
| 6-7 | Data Restore | Settings | Restore from backup with pre-restore snapshot, integrity check |
| 6-8 | Scheduled Backups | Settings | Configurable backup schedule (daily/weekly), retention policy |
| 6-9 | Metadata Export (JSON/YAML/NFO) | Settings | Export media metadata in JSON, YAML, Kodi-compatible NFO formats |

---

## Summary

| Old Epic | Stories | Primary v4 Feature IDs |
|----------|---------|----------------------|
| Epic 1 | 5 | Infrastructure (foundation for all epics) |
| Epic 2 | 6 | P1-002, P1-003, P1-007 |
| Epic 3 | 12 | P1-002, P1-004 |
| Epic 4 | 6 | P3-010 |
| Epic 5 | 11 | P1-007 |
| Epic 6 | 10 | Settings, P4-020 (partial) |
| **Total** | **50** | |

---

## Post-Completion Fixes: Epic 4 (2026-03-30)

| Fix | Files | Description |
|-----|-------|-------------|
| qBT 5.0 state mapping | `torrent.go` | Added `stoppedUP`, `stoppedDL`, `allocating`, `moving` states for qBT 5.0+ compatibility. `stalledUP` → completed (was seeding). Follows Sonarr/Radarr standard. |
| Filter tab counts | `download_service.go` | `stalled`, `queued`, `checking` states were not counted in any tab, causing all counts to show 0. Now mapped to downloading/paused. |
| Migration 020 idempotent | `020_create_media_libraries.go` | Split multi-statement ALTER TABLE into individual calls with `columnExists()` check to prevent SQLite retry failures. |
| Settings form init | `QBittorrentForm.tsx` | Fixed React render timing bug — saved qBT config wasn't loading into settings form. Changed from if-guard to useEffect. |
| Docker arm64 build | `Dockerfile` | `NX_DAEMON=false` prevents Nx daemon OOM crash under QEMU arm64 emulation. |
| Vite dev proxy | `vite.config.mts` | Added `/api` proxy to localhost:8080 for local development. |
| Download pagination | `download_handler.go`, `downloadService.ts`, `downloads.tsx` | Backend page+pageSize API + frontend Pagination component with pageSize selector (50/100/200/500). |
| Download page design | `ux-design.pen` | Added Flow G with 4 screens: G1-G4 covering desktop/mobile download management UI. |
