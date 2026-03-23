# Implementation Gap Analysis (v4)

## Overview

This gap analysis maps the current codebase state against PRD v4 feature IDs. Epics 1-6 from the previous iteration have been partially implemented; this document identifies what exists and what remains.

---

## Already Implemented (Epics 1-6)

| Component | Status | Evidence |
|-----------|--------|----------|
| Nx monorepo structure | ✅ Complete | `nx.json`, `apps/api/`, `apps/web/` |
| Go backend with Gin | ✅ Complete | `apps/api/main.go`, handlers, services |
| React frontend with TanStack Router | ✅ Complete | `apps/web/src/router.tsx`, routes |
| SQLite database with WAL mode | ✅ Complete | `movies`, `series`, `settings` tables |
| TMDB metadata provider (zh-TW) | ✅ Complete | `internal/tmdb/` client with zh-TW priority |
| Basic filename parser (regex) | ✅ Complete | `internal/parser/` with regex patterns |
| AI-powered filename parsing | ✅ Complete | `internal/ai/` with Gemini/Claude providers |
| Multi-source metadata fallback | ✅ Complete | TMDb → Douban → Wikipedia chain |
| qBittorrent download integration | ✅ Complete | `internal/qbittorrent/` client |
| Media search UI | ✅ Complete | Search components in `apps/web/` |
| Tailwind CSS styling | ✅ Complete | `tailwind.config.js` configured |
| Testing infrastructure | ✅ Complete | Go testify + Vitest + Playwright |
| Docker deployment | ✅ Complete | `docker-compose.yml` |
| Settings management | ✅ Partial | `settings` table, basic endpoints |
| Metadata export (JSON/YAML/NFO) | ✅ Complete | Story 6-9 implementation |

---

## Gap Category 1: Missing v4 Features

### Subtitle Engine (P1-010~P1-019) — ❌ Completely Missing

| Feature ID | Description | Status |
|-----------|-------------|--------|
| P1-010 | Subtitle provider interface | ❌ Missing |
| P1-011 | Assrt API integration | ❌ Missing |
| P1-012 | Zimuku scraper integration | ❌ Missing |
| P1-013 | OpenSubtitles API integration | ❌ Missing |
| P1-014 | Multi-factor subtitle scoring | ❌ Missing |
| P1-015 | Content-based language detection | ❌ Missing |
| P1-016 | OpenCC 簡繁轉換 integration | ❌ Missing |
| P1-017 | Subtitle file placement | ❌ Missing |
| P1-018 | Subtitle search cache (SQLite) | ❌ Missing |
| P1-019 | Batch subtitle processing | ❌ Missing |

**Blocking Issues:**
- No subtitle provider implementations
- No OpenCC binding or subprocess integration
- No content-based language detection logic
- No subtitle scoring algorithm

---

### Media Scanner (P1-001, P1-005, P1-006) — ❌ Missing Scheduled/Manual Scan

| Feature ID | Description | Status |
|-----------|-------------|--------|
| P1-001 | Recursive media file scanner | ❌ Missing |
| P1-005 | Scheduled scanning (configurable interval) | ❌ Missing |
| P1-006 | Manual scan trigger via API | ❌ Missing |

**What Exists:**
- TMDB matching logic exists (from metadata fallback)
- File parsing logic exists (from filename parser)

**What's Missing:**
- `/internal/scanner/` package
- File system watcher for scheduled scans
- API endpoint for manual scan trigger
- Background task integration for scan progress

---

### Plugin Architecture — ❌ Not Implemented

| Component | Status |
|-----------|--------|
| Plugin interfaces (MediaServerPlugin, DownloaderPlugin, DVRPlugin) | ❌ Missing |
| Plugin manager (registration, health checks) | ❌ Missing |
| Plex MediaServerPlugin | ❌ Missing |
| Jellyfin MediaServerPlugin | ❌ Missing |
| Sonarr DVRPlugin | ❌ Missing |
| Radarr DVRPlugin | ❌ Missing |
| Prowlarr indexer integration | ❌ Missing |
| Per-plugin config in SQLite | ❌ Missing |

**Note:** qBittorrent client exists but is not yet refactored into the DownloaderPlugin interface.

---

### SSE Hub — ❌ Not Implemented

| Component | Status |
|-----------|--------|
| SSE hub goroutine | ❌ Missing |
| Client registration/deregistration | ❌ Missing |
| Event types (download_progress, scan_status, etc.) | ❌ Missing |
| HTTP handler for `/api/v1/events` | ❌ Missing |
| Last-Event-ID reconnection support | ❌ Missing |

**Current State:** Frontend polls for download status (5-second interval). SSE would replace this with push-based updates.

---

### Server-side TMDB Filtering — ❌ Not Implemented

| Component | Status |
|-----------|--------|
| In-memory TMDB trending/discover cache | ❌ Missing |
| 1-hour TTL for trending results | ❌ Missing |
| Explore/browse endpoints | ❌ Missing |

**Note:** TMDB client exists for search/detail lookups. Filtering cache needed for Phase 2 explore features.

---

### Request System — ❌ Not Implemented

| Component | Status |
|-----------|--------|
| Media request submission | ❌ Missing |
| Request status tracking | ❌ Missing |
| DVR plugin integration for auto-add | ❌ Missing |

---

### NAS Dashboard — ❌ Not Implemented

| Component | Status |
|-----------|--------|
| Storage usage overview | ❌ Missing |
| Recent activity feed | ❌ Missing |
| Plugin health status | ❌ Missing |
| Quick actions | ❌ Missing |

---

## Gap Category 2: Architectural Decisions vs Implementation

### Decision #3: Authentication — REMOVED ✅

Authentication was removed in v4 (single-user). Any existing auth references (if any) should be cleaned up.

### Decision #4: Caching Strategy — ⚠️ Partially Implemented

- ✅ Basic in-memory caching exists for TMDB responses
- ❌ No `CacheManager` with tiered strategy
- ❌ No `cache_entries` SQLite table
- ❌ No server-side TMDB filtering cache

### Decision #5: Background Tasks — ⚠️ Partially Implemented

- ✅ Some async operations exist
- ❌ No formal worker pool implementation
- ❌ No task types for media scan, subtitle batch, plugin health

### Decision #7: Plugin Architecture — ❌ Not Implemented

### Decision #8: SSE Hub — ❌ Not Implemented

### Decision #9: Subtitle Engine — ❌ Not Implemented

---

## Gap Summary by Priority

**🔴 Critical Gaps (Block v4 Core Features):**

1. **Subtitle Engine Pipeline** — Core differentiator, completely missing
2. **Media Library Scanner** — Essential for automated library management
3. **Plugin Architecture** — Foundation for all external service integrations
4. **SSE Hub** — Required for real-time progress updates

**🟡 Important Gaps (Affect Quality/Experience):**

5. **Server-side TMDB Filtering** — Needed for Phase 2 explore features
6. **Formal Caching System** — CacheManager with tiered strategy
7. **Worker Pool** — Formal background task system with all task types
8. **NAS Dashboard** — Overview and monitoring UI

**🟢 Deferred Gaps (Phase 3+):**

9. **Request System** — DVR integration for media requests
10. **Multi-user Authentication** — Deferred to v5.0

---
