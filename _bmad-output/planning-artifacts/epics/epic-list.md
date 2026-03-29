# Epic List

> **v4 PRD Structure** — Epics 1-6 (completed under v3) are preserved for reference.
> New work follows the A-L epic structure organized by delivery phase.

---

## Completed Epics (v3 PRD)

| Epic | Name | Status |
|------|------|--------|
| [Epic 1](./epic-1-project-foundation-docker-deployment.md) | Project Foundation & Docker Deployment | COMPLETED |
| [Epic 2](./epic-2-media-search-traditional-chinese-metadata.md) | Media Search & Traditional Chinese Metadata | COMPLETED |
| [Epic 3](./epic-3-ai-powered-fansub-parsing-multi-source-fallback.md) | AI-Powered Fansub Parsing & Multi-Source Fallback | COMPLETED |
| [Epic 4](./epic-4-qbittorrent-download-monitoring.md) | qBittorrent Download Monitoring | COMPLETED |
| [Epic 5](./epic-5-media-library-management.md) | Media Library Management | COMPLETED |
| [Epic 6](./epic-6-system-configuration-backup.md) | System Configuration & Backup | COMPLETED |

See [Completed Work Registry](./completed-work-registry.md) for the mapping of 50 completed stories to v4 feature IDs.

---

## Phase 1: Core Media Pipeline

### Epic 7: Media Library Scanner
**File:** [epic-7-media-library-scanner.md](./epic-7-media-library-scanner.md)

Users can configure media library paths, trigger recursive scanning, have filenames parsed (regex + AI), auto-match TMDB, get zh-TW metadata fallback, and browse their library in grid/list views.

**v4 Feature IDs:** P1-001, P1-002, P1-003, P1-004, P1-005, P1-006, P1-007

---

### Epic 7b: Multi-Library Media Management
**File:** [epic-7b-multi-library-media-management.md](./epic-7b-multi-library-media-management.md)

Users can create multiple media libraries with per-folder content type assignment (movie/series), manage libraries from Setup Wizard and Settings page, and scanner uses DB-based configuration instead of environment variables.

**v4 Feature IDs:** P1-001-A, P1-001-B, P1-001-C, P1-001-D, P1-001-E

---

### Epic 8: Subtitle Engine
**File:** [epic-8-subtitle-engine.md](./epic-8-subtitle-engine.md)

Users can search for Traditional Chinese subtitles across multiple sources (Assrt, Zimuku, OpenSubtitles), with correct 簡繁 identification, automatic OpenCC conversion, subtitle scoring/ranking, auto-download, manual search UI, and batch processing.

**v4 Feature IDs:** P1-010, P1-011, P1-012, P1-013, P1-014, P1-015, P1-016, P1-017, P1-018, P1-019

---

### Epic 9: AI Subtitle Enhancement
**File:** [epic-9-ai-subtitle-enhancement.md](./epic-9-ai-subtitle-enhancement.md)

Optional AI-powered subtitle enhancements: terminology correction (簡→繁 word-level fixes using Claude API) and MKV English audio track translation (Whisper transcription → translation).

**v4 Feature IDs:** P1-020, P1-021

---

### Epic 9-T: TestSprite Journey Test Integration
**File:** [epic-9t-testsprite-journey-integration.md](./epic-9t-testsprite-journey-integration.md)

Integrate TestSprite journey-level tests against the deployed NAS app to fill the gap between unit tests and Playwright E2E tests. Covers config setup, test plan regeneration, baseline establishment, and manual trigger workflow documentation.

**Status:** IN PROGRESS (parallel to bugfix sprint)

---

## Phase 2: Discovery & Browse Experience

### Epic 10: Homepage TV Wall
**File:** [epic-10-homepage-tv-wall.md](./epic-10-homepage-tv-wall.md)

Users see a Hero Banner with trending content, customizable explore blocks, smart trending with server-side language/region filtering, auto-hiding of low-quality content, and "已有/已請求" badges.

**v4 Feature IDs:** P2-001, P2-002, P2-003, P2-004, P2-005, P2-006

---

### Epic 11: Advanced Search & Filter
**File:** [epic-11-advanced-search-filter.md](./epic-11-advanced-search-filter.md)

Users can filter content by multiple dimensions simultaneously using persistent chip UI, complex sorting, instant search with debounced suggestions, zh-TW search priority, and saved filter presets.

**v4 Feature IDs:** P2-010, P2-011, P2-012, P2-013, P2-014, P2-015

---

### Epic 12: Rich Media Detail Page
**File:** [epic-12-rich-media-detail-page.md](./epic-12-rich-media-detail-page.md)

Enhanced media detail page with TMDB + Douban dual ratings, TV show season/episode lists with subtitle status, related recommendations, streaming platform availability, trailer embeds, and Douban links.

**v4 Feature IDs:** P2-020, P2-021, P2-022, P2-023, P2-024, P2-025

---

## Phase 3: Automation & Integration

### Epic 13: Request System
**File:** [epic-13-request-system.md](./epic-13-request-system.md)

Users can one-click request movies/shows, request specific seasons/episodes, track request status, optionally route through Sonarr/Radarr, and auto-trigger subtitle search on completion.

**v4 Feature IDs:** P3-001, P3-002, P3-003, P3-004, P3-005

---

### Epic 14: Download Management v2
**File:** [epic-14-download-management-v2.md](./epic-14-download-management-v2.md)

Enhanced download management with qBittorrent control (DONE), optional NZBGet support, SSE real-time progress push, download completion notifications, and future internal BT engine.

**v4 Feature IDs:** P3-010, P3-011, P3-012, P3-013, P3-014

---

### Epic 15: Indexer Integration
**File:** [epic-15-indexer-integration.md](./epic-15-indexer-integration.md)

Users can connect to Prowlarr for indexer management, or use Vido's built-in basic public tracker search when Prowlarr is not configured.

**v4 Feature IDs:** P3-020, P3-021

---

## Phase 4: Polish & Ecosystem

### Epic 16: Media Statistics Dashboard
**File:** [epic-16-media-statistics-dashboard.md](./epic-16-media-statistics-dashboard.md)

Users see a dashboard with library overview (counts, disk usage, resolution distribution), subtitle coverage visualization, genre/region/year distribution charts, and recently added media.

**v4 Feature IDs:** P4-001, P4-002, P4-003, P4-004

---

### Epic 17: Media Server Integration
**File:** [epic-17-media-server-integration.md](./epic-17-media-server-integration.md)

Users can connect Plex or Jellyfin to sync watch history, see "繼續觀看" on homepage, and synchronize library inventory to mark owned content in explore views.

**v4 Feature IDs:** P4-010, P4-011, P4-012

---

### Epic 18: Service Health Monitoring
**File:** [epic-18-service-health-monitoring.md](./epic-18-service-health-monitoring.md)

Users see connection status of all configured external services, disk space warnings, and a searchable/filterable activity log.

**v4 Feature IDs:** P4-020, P4-021, P4-022

---

## Archived Epics (v3 PRD — superseded)

Old Epics 7-14 have been archived to `archive/`. Their scope has been redistributed across Epics A-L under the v4 feature ID system.
