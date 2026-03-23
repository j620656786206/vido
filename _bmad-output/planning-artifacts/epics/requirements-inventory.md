# Requirements Inventory

## v4 Feature ID System

All features use the format `P{phase}-{number}`. Features marked with **(DONE)** have working implementations from v3 Epics 1-6.

---

## Phase 1: Core Media Pipeline

### Epic 7: Media Library Scanner

| Feature ID | Description | Status |
|-----------|-------------|--------|
| P1-001 | Recursive folder scanning of configured media library paths | NEW |
| P1-002 | Filename parsing (regex for standard names, AI for fansub names) | DONE (Epic 2: Story 2-5, Epic 3: Stories 3-1, 3-2, 3-6, 3-9) |
| P1-003 | TMDB auto-matching with zh-TW metadata retrieval | DONE (Epic 2: Stories 2-1, 2-6) |
| P1-004 | Multi-source metadata fallback (TMDB → Douban → Wikipedia → AI → Manual) | DONE (Epic 3: Stories 3-3, 3-4, 3-5, 3-11, 3-12) |
| P1-005 | Scheduled automatic re-scanning (cron-based) | NEW |
| P1-006 | Manual scan trigger with progress tracking UI | NEW |
| P1-007 | Library browsing (grid/list views, search, sort, filter, detail pages) | DONE (Epic 2: Stories 2-2, 2-3, 2-4, Epic 3: Stories 3-7, 3-8, 3-10, Epic 5: Stories 5-0 through 5-10) |

### Epic 8: Subtitle Engine

| Feature ID | Description | Status |
|-----------|-------------|--------|
| P1-010 | Assrt API subtitle search client | NEW |
| P1-011 | Zimuku web scraper subtitle search | NEW |
| P1-012 | OpenSubtitles API subtitle search client | NEW |
| P1-013 | 簡繁 language detection for subtitle files | NEW |
| P1-014 | OpenCC simplified→traditional auto-conversion | NEW |
| P1-015 | Subtitle scoring and ranking algorithm | NEW |
| P1-016 | Auto-download best-match subtitle for new media | NEW |
| P1-017 | Manual subtitle search UI | NEW |
| P1-018 | Batch subtitle processing across library | NEW |
| P1-019 | Subtitle file management (store, rename, associate) | NEW |

### Epic 9: AI Subtitle Enhancement

| Feature ID | Description | Status |
|-----------|-------------|--------|
| P1-020 | AI terminology correction (簡→繁 word-level fixes via Claude API) | NEW (P2 priority) |
| P1-021 | MKV English audio track translation (Whisper + AI) | NEW (P3 priority) |

---

## Phase 2: Discovery & Browse Experience

### Epic 10: Homepage TV Wall

| Feature ID | Description | Status |
|-----------|-------------|--------|
| P2-001 | Hero Banner with trending content and auto-rotation | NEW |
| P2-002 | Customizable explore blocks (themed content sections) | NEW |
| P2-003 | Server-side TMDB filtering (language, region, date) | NEW |
| P2-004 | Content quality filtering (hide far-future, low-quality, adult) | NEW |
| P2-005 | "已有/已請求" availability badges on media cards | NEW |
| P2-006 | Homepage layout engine with configurable block ordering | NEW |

### Epic 11: Advanced Search & Filter

| Feature ID | Description | Status |
|-----------|-------------|--------|
| P2-010 | Multi-dimensional filter engine (genre + year + region + rating + platform) | NEW (extends Epic 5 filters) |
| P2-011 | Persistent chip UI for active filters | NEW (extends Epic 5 chip UI) |
| P2-012 | Compound multi-key sorting | NEW (extends Epic 5 sorting) |
| P2-013 | Instant search with debounced dropdown suggestions | NEW (extends Epic 5 search) |
| P2-014 | zh-TW search priority and romanization handling | NEW |
| P2-015 | Saved filter presets | NEW |

### Epic 12: Rich Media Detail Page

| Feature ID | Description | Status |
|-----------|-------------|--------|
| P2-020 | TMDB + Douban dual rating display | NEW (extends Epic 5 detail page) |
| P2-021 | TV show season/episode list with subtitle status | NEW |
| P2-022 | Related content recommendations with availability badges | NEW |
| P2-023 | Streaming platform availability (TMDB Watch Providers) | NEW |
| P2-024 | Trailer embeds (YouTube) | NEW (extends Epic 5 trailer support) |
| P2-025 | Douban integration (links, review summary) | NEW (extends Epic 3 Douban scraper) |

---

## Phase 3: Automation & Integration

### Epic 13: Request System

| Feature ID | Description | Status |
|-----------|-------------|--------|
| P3-001 | One-click request from explore/detail pages | NEW |
| P3-002 | Partial request (specific seasons/episodes) | NEW |
| P3-003 | Request status tracking pipeline | NEW |
| P3-004 | Sonarr/Radarr DVR plugin for automated downloading | NEW |
| P3-005 | Auto-trigger subtitle search on download completion | NEW |

### Epic 14: Download Management v2

| Feature ID | Description | Status |
|-----------|-------------|--------|
| P3-010 | qBittorrent monitoring and download dashboard | DONE (Epic 4: Stories 4-1 through 4-6) |
| P3-011 | NZBGet download client support | NEW |
| P3-012 | SSE real-time progress push (replace polling) | NEW |
| P3-013 | Download completion notifications (in-app + webhook) | NEW |
| P3-014 | Internal BitTorrent engine (future, P3 priority) | NEW |

### Epic 15: Indexer Integration

| Feature ID | Description | Status |
|-----------|-------------|--------|
| P3-020 | Prowlarr integration for indexer management | NEW |
| P3-021 | Built-in basic public tracker search | NEW |

---

## Phase 4: Polish & Ecosystem

### Epic 16: Media Statistics Dashboard

| Feature ID | Description | Status |
|-----------|-------------|--------|
| P4-001 | Library overview (counts, disk usage, resolution distribution) | NEW (partially from Epic 6 Story 6-11) |
| P4-002 | Subtitle coverage rate visualization | NEW |
| P4-003 | Genre/region/year distribution charts | NEW |
| P4-004 | Recently added media timeline | NEW (extends Epic 5 Story 5-8) |

### Epic 17: Media Server Integration

| Feature ID | Description | Status |
|-----------|-------------|--------|
| P4-010 | Plex integration plugin | NEW |
| P4-011 | Jellyfin integration plugin | NEW |
| P4-012 | Watch history sync and library inventory sync | NEW |

### Epic 18: Service Health Monitoring

| Feature ID | Description | Status |
|-----------|-------------|--------|
| P4-020 | Unified service health panel for all external services | NEW (extends Epic 6 Story 6-4) |
| P4-021 | Disk space monitoring with configurable warnings | NEW |
| P4-022 | Searchable/filterable activity log | NEW (extends Epic 6 Story 6-3) |

---

## Coverage Summary

| Phase | Feature IDs | Epic Coverage | NEW / DONE |
|-------|-------------|---------------|------------|
| Phase 1 | P1-001 through P1-021 | Epics A, B, C | 14 NEW, 7 DONE |
| Phase 2 | P2-001 through P2-025 | Epics D, E, F | 18 NEW, 0 DONE |
| Phase 3 | P3-001 through P3-021 | Epics G, H, I | 9 NEW, 1 DONE |
| Phase 4 | P4-001 through P4-022 | Epics J, K, L | 10 NEW, 0 DONE |
| **Total** | **67 features** | **12 epics** | **51 NEW, 8 DONE** |
