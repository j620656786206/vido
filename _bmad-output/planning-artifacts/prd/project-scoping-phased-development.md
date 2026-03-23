# Project Scoping & Phased Development

> Aligned with PRD v4.0 (2026-03-22). This document supersedes all prior phased development plans.

---

## MVP Strategy & Philosophy

**MVP Theme:** 字幕核心穩定 — Become the #1 Traditional Chinese subtitle solution for NAS users.

Vido's MVP is no longer about fansub filename parsing alone. The v4 vision expands Vido into an **all-in-one media management interface** for Traditional Chinese NAS users, but the MVP remains laser-focused: deliver a **self-contained Traditional Chinese subtitle management tool** that fixes every known Bazarr bug and requires zero external dependencies.

**Core Philosophy:**

- Single developer, single user (no authentication needed)
- Each phase ships independently — every release is a usable product
- Pluggable integration architecture: works standalone, but can connect to existing *arr stack
- Traditional Chinese quality is the moat — no competitor addresses this audience

**Why This MVP:**

- Bazarr's Traditional Chinese bugs (簡繁 misidentification, `.zt` extension incompatibility, Assrt API response key error, wrong conversion direction) have persisted 2+ years with no fix in sight
- No competing tool offers a dedicated Traditional Chinese subtitle engine
- MediaManager (the fastest-growing competitor) has no subtitle management at all
- Subtitle quality is the single highest pain point for the target user segment

**Resource Requirements:**

| Dimension | Detail |
|-----------|--------|
| Team size | Single full-stack developer |
| Frontend | React 19, TypeScript, Vite, TanStack Router/Query |
| Backend | Go, Gin framework, SQLite (WAL mode) |
| Deployment | Docker (single container) |
| Time estimate | 8-10 weeks for Phase 1 MVP |

---

## Completed Infrastructure (Epics 1-6, v3 PRD)

The following capabilities are **already implemented** and provide the foundation for Phase 1-4 development:

| Epic | Capability | Status |
|------|-----------|--------|
| Epic 1 | Nx monorepo + Go/React scaffold + Docker deployment | Done |
| Epic 2 | TMDB search with zh-TW language priority | Done |
| Epic 3 | AI-powered fansub filename parsing (Gemini/Claude) | Done |
| Epic 3 | Multi-source metadata fallback: TMDb → Douban → Wikipedia → AI | Done |
| Epic 4 | qBittorrent download monitoring (polling-based) | Done |
| Epic 5 | Media library UI — grid/list view, search, filter, sort, batch operations | Done |
| Epic 6 | Settings page, backup/restore, metadata export (JSON, YAML, Kodi NFO) | Done (9/11 stories) |

These completed epics map directly to several v4 feature IDs:

- **P1-002** (filename parsing engine) — Done via Epic 3
- **P1-003** (TMDB auto-matching) — Done via Epic 2
- **P1-007** (media library browsing) — Done via Epic 5
- **P3-010** (qBittorrent integration) — Done via Epic 4

---

## Phase 1: 字幕核心穩定 (MVP) — 8-10 Weeks

**Goal:** 可獨立運作的繁中字幕管理工具 — A self-contained Traditional Chinese subtitle management tool that replaces Bazarr for the target audience.

### 1.1 Media Library Scanner

| ID | Feature | Priority | Description | Status |
|----|---------|----------|-------------|--------|
| P1-001 | Folder scanning | P0 | Specify one or more media library paths; recursively scan for video files (mkv, mp4, avi, rmvb) | New |
| P1-002 | Filename parsing engine | P0 | Parse standard naming (`Movie.Name.2024.1080p.BluRay`) and fansub naming (`[SweetSub][動畫名][12][BIG5][1080P]`) | **Done** |
| P1-003 | TMDB auto-matching | P0 | Match parsed results to TMDB ID; fetch metadata (poster, synopsis, rating, cast) | **Done** |
| P1-004 | zh-TW metadata fallback | P1 | When TMDB zh-TW data is incomplete, supplement from Douban and Wikipedia | New |
| P1-005 | Scheduled scanning | P1 | Configurable scan frequency (hourly/daily); new files auto-ingested | New |
| P1-006 | Manual scan trigger | P0 | One-click full or folder-specific scan from Web UI | New |
| P1-007 | Media library browsing | P0 | Poster wall view for scanned movies/TV shows with search, sort, filter | **Done** |

### 1.2 Traditional Chinese Subtitle Engine

| ID | Feature | Priority | Description |
|----|---------|----------|-------------|
| P1-010 | Multi-source subtitle search | P0 | Search across Assrt (射手網), Zimuku (字幕庫), and OpenSubtitles |
| P1-011 | Assrt API fix | P0 | Use correct response key (`native_name` instead of `videoname`); fix subtitle-exists-but-skipped bug |
| P1-012 | Accurate 簡繁 identification | P0 | Determine Simplified vs Traditional via **content analysis** (not filename); prevent Simplified subtitles from blocking Traditional |
| P1-013 | Extension normalization | P0 | Standardize output to `.zh-Hant` or `.cht` extensions; ensure Plex/Jellyfin/Infuse recognition |
| P1-014 | Simplified → Traditional conversion | P0 | OpenCC conversion with correct direction (簡→繁); cross-strait terminology correction (軟件→軟體, 內存→記憶體) |
| P1-015 | Fansub naming resolution | P1 | Parse common fansub subtitle naming patterns: `[Group][Title][Episode][Language][Resolution]` → correct video mapping |
| P1-016 | Subtitle scoring & ranking | P1 | Score search results: language match > resolution match > source trust > download count |
| P1-017 | Auto-download best subtitle | P1 | Automatically select and download highest-scored Traditional Chinese subtitle to correct path |
| P1-018 | Manual search & selection | P0 | Web UI for searching subtitles, previewing results, and manually choosing downloads |
| P1-019 | Batch subtitle processing | P2 | Search and download subtitles for an entire TV season in one operation |

### 1.3 AI Enhancements (Optional)

| ID | Feature | Priority | Description |
|----|---------|----------|-------------|
| P1-020 | AI terminology correction | P2 | Use Claude API for fine-tuned cross-strait terminology review on 簡→繁 output (user-provided API key) |
| P1-021 | MKV English track translation | P3 | No-subtitle fallback: extract English audio → Whisper transcription → DeepL/Claude translation to zh-TW (user-provided API keys) |

### 1.4 Milestones

| Milestone | Content | Weeks |
|-----------|---------|-------|
| M1.1 | Go backend skeleton + SQLite + base API | 1-2 |
| M1.2 | Media library scanner + filename parsing engine | 2-3 |
| M1.3 | TMDB matching + zh-TW metadata | 1-2 |
| M1.4 | Subtitle engine (Assrt fix + 簡繁 identification + conversion) | 2-3 |
| M1.5 | React 19 frontend + media library browsing + subtitle management UI | 2-3 |
| M1.6 | Docker packaging + first public release | 1 |

### 1.5 Validation Metrics

| Metric | Target |
|--------|--------|
| Standard naming parse success rate | > 99% |
| Fansub naming parse success rate | > 95% |
| Traditional Chinese subtitle search hit rate | > 85% |
| Zero Simplified subtitle misidentification | 100% |
| Cross-strait terminology correction accuracy | > 95% |
| Docker start-to-usable time | < 10 seconds |
| GitHub Stars (3 months post-release) | > 100 |
| Docker Hub pulls (3 months post-release) | > 500 |
| Community-reported zh-TW subtitle bugs | < 5 |
| Personal dogfooding: daily use replaces Bazarr | Yes |

---

## Phase 2: 媒體探索 — 6-8 Weeks

**Goal:** 取代 Seerr 的探索功能 — Replace Seerr with a superior browsing experience tailored to Asian content.

**Prerequisite:** Phase 1 stable.

### 2.1 Homepage TV Wall

| ID | Feature | Priority | Description |
|----|---------|----------|-------------|
| P2-001 | Hero banner carousel | P0 | Top-of-page hero images showcasing featured/trending content with optional trailer autoplay |
| P2-002 | Customizable explore blocks | P0 | Users can add, remove, reorder homepage content blocks (e.g., "Recent Taiwan Theatrical", "Trending K-Drama", "Netflix TW New Releases") |
| P2-003 | Smart trending with filters | P0 | Trending content **forcibly applies** language/region filtering server-side (bypasses TMDB API endpoint limitations) |
| P2-004 | Hide far-future content | P0 | Auto-filter items with release dates > 6 months out (e.g., Avatar 5 in 2031) |
| P2-005 | Hide low-quality content | P1 | Auto-filter TMDB rating < 3 with < 50 votes |
| P2-006 | Owned/requested badges | P1 | Show "Available" badge for items in library; "Requested" for pending requests |

### 2.2 Advanced Search & Filtering

| ID | Feature | Priority | Description |
|----|---------|----------|-------------|
| P2-010 | Multi-dimensional filters | P0 | Simultaneous filtering: genre, year range, region/language, rating range, streaming platform |
| P2-011 | Always-visible filter UI | P0 | Filters displayed as pill/chip elements pinned to page top (not hidden in dropdowns) |
| P2-012 | Compound sorting | P0 | Sort by: popularity, release date, rating, date added. Results respect active filters |
| P2-013 | Instant search | P0 | Search box with debounced suggestions showing movies, TV shows, and people |
| P2-014 | zh-TW search priority | P1 | Search TMDB Chinese and original titles simultaneously; rank zh-TW results higher |
| P2-015 | Saved filter presets | P2 | Save frequently used filter combinations for quick access |

### 2.3 Media Detail Page

| ID | Feature | Priority | Description |
|----|---------|----------|-------------|
| P2-020 | Rich info display | P0 | Poster, backdrop, zh-TW synopsis, cast/director, genre, runtime, release date, dual ratings (TMDB + Douban) |
| P2-021 | TV season/episode list | P0 | Expandable season view with episode titles, synopses, and subtitle availability status |
| P2-022 | Related recommendations | P1 | Recommend similar content with region/language filtering applied |
| P2-023 | Streaming platform info | P1 | Show available streaming platforms in Taiwan (Netflix/Disney+/KKTV/...) via TMDB Watch Providers |
| P2-024 | Trailer playback | P2 | Embedded YouTube trailer (when available) |
| P2-025 | Douban link | P2 | Direct link to Douban page for Chinese reviews |

### 2.4 Milestones

| Milestone | Content | Weeks |
|-----------|---------|-------|
| M2.1 | TMDB API integration + server-side filtering layer | 2 |
| M2.2 | Homepage TV wall (Hero Banner + customizable blocks) | 2-3 |
| M2.3 | Advanced search + multi-dimensional filters | 2-3 |
| M2.4 | Media detail page (dual ratings + streaming platforms + subtitle status) | 1-2 |

### 2.5 Validation Metrics

| Metric | Target |
|--------|--------|
| GitHub Stars (6 months post-Phase 1) | > 500 |
| Docker Hub pulls (6 months post-Phase 1) | > 2,000 |
| Organic discussion on PTT/Bahamut NAS boards | Yes |
| Personal dogfooding: daily use replaces Seerr | Yes |

---

## Phase 3: 請求流程 + 下載管理 — 6-8 Weeks

**Goal:** 一鍵請求→自動下載→自動抓字幕 — One-click request → automatic download → automatic subtitle fetch, fully automated end-to-end.

**Prerequisite:** Phase 2 stable.

### 3.1 Request System

| ID | Feature | Priority | Description |
|----|---------|----------|-------------|
| P3-001 | One-click request | P0 | Request movies/TV shows directly from explore or detail pages |
| P3-002 | Partial TV request | P0 | Request specific seasons or individual episodes |
| P3-003 | Request status tracking | P0 | Request list page showing status: pending / searching / downloading / completed / failed |
| P3-004 | Sonarr/Radarr integration (optional) | P0 | When configured, route requests through Sonarr/Radarr API; otherwise use Vido's built-in flow |
| P3-005 | Auto subtitle trigger | P1 | Automatically trigger subtitle search (Phase 1 engine) upon download completion |

### 3.2 Download Management

| ID | Feature | Priority | Description | Status |
|----|---------|----------|-------------|--------|
| P3-010 | qBittorrent integration | P0 | Manage downloads via qBittorrent Web API (add/pause/delete/view progress) | **Done** (Epic 4) |
| P3-011 | NZBGet integration (optional) | P2 | Usenet download support |  |
| P3-012 | Real-time download progress | P0 | SSE push download progress to frontend — no manual refresh |  |
| P3-013 | Download completion notification | P1 | Web UI notification on completion (future: Telegram/Discord) |  |
| P3-014 | Built-in BT engine (future) | P3 | Use Go BT library (anacrolix/torrent) to eliminate qBittorrent dependency |  |

### 3.3 Indexer Management

| ID | Feature | Priority | Description |
|----|---------|----------|-------------|
| P3-020 | Prowlarr integration (optional) | P1 | When configured, search indexers through Prowlarr API |
| P3-021 | Built-in indexer search | P2 | When Prowlarr is not configured, provide basic torrent search against public trackers |

### 3.4 Milestones

| Milestone | Content | Weeks |
|-----------|---------|-------|
| M3.1 | Request system (one-click request + status tracking) | 2 |
| M3.2 | Pluggable integration layer (Sonarr/Radarr plugin) | 2-3 |
| M3.3 | qBittorrent integration + download progress SSE | 2 |
| M3.4 | End-to-end automation (request → download → subtitle) | 1-2 |

---

## Phase 4: NAS Dashboard — 4-6 Weeks

**Goal:** 一個介面掌握全局 — A single interface for complete NAS media system visibility.

**Prerequisite:** Phase 3 stable.

### 4.1 Media Library Statistics

| ID | Feature | Priority | Description |
|----|---------|----------|-------------|
| P4-001 | Library overview | P0 | Total movies/TV shows, disk usage, average file size, resolution distribution (4K/1080p/720p) |
| P4-002 | Subtitle coverage | P0 | Chart showing has-zh-TW / has-other-subtitle / no-subtitle ratios |
| P4-003 | Genre distribution | P1 | Charts for genre, region, and year distribution |
| P4-004 | Recently added | P0 | Media added in last 7/30 days |

### 4.2 Plex/Jellyfin Integration

| ID | Feature | Priority | Description |
|----|---------|----------|-------------|
| P4-010 | Watch history sync | P1 | Sync watch history from Plex/Jellyfin for personalized recommendations |
| P4-011 | Continue watching | P1 | Homepage "Continue Watching" block based on Plex/Jellyfin watch progress |
| P4-012 | Library inventory sync | P0 | Periodically scan Plex/Jellyfin libraries to mark owned content |

### 4.3 Health Monitoring

| ID | Feature | Priority | Description |
|----|---------|----------|-------------|
| P4-020 | External service status | P1 | Connection status for all integrated services (Sonarr/Radarr/qBittorrent/Plex/Jellyfin) |
| P4-021 | Disk space warning | P1 | Alert when media library disk usage exceeds threshold |
| P4-022 | Activity log | P2 | Searchable, filterable log of all automated actions (subtitle downloads, request processing, scan results) |

### 4.4 Milestones

| Milestone | Content | Weeks |
|-----------|---------|-------|
| M4.1 | Media library statistics + subtitle coverage charts | 2 |
| M4.2 | Plex/Jellyfin integration + watch history sync | 2-3 |
| M4.3 | Service health monitoring + disk space warnings | 1-2 |

### 4.5 Validation Metrics

| Metric | Target |
|--------|--------|
| GitHub Stars (12 months post-Phase 1) | > 1,000 |
| Docker Hub pulls (12 months post-Phase 1) | > 5,000 |
| External contributors | > 5 |
| Recommended as a tool in zh-TW NAS communities | Yes |

---

## Out of Scope (v4)

The following are **explicitly excluded** from all v4 phases:

| Item | Reason |
|------|--------|
| Video playback | Plex/Jellyfin/Infuse already excel at this — no need to reinvent |
| Multi-user + permissions | v4 targets single-user; multi-user (family request/approval) deferred to v5 |
| ML-based recommendation engine | Use TMDB recommendation API + watch history for simple recommendations; no model training |
| Subtitle community features | No subtitle upload, sharing, or commenting |
| Whisper local model | AI features rely on external APIs (user-provided keys); no local Whisper inference |
| Docker/container management | Not a Portainer or Unraid Docker Manager replacement |
| Music management | Video only (movies/TV/anime); no music |
| Live/IPTV | Not in scope |

---

## Risk Mitigation

### Technical Risks

| Risk | Severity | Mitigation Strategy |
|------|----------|---------------------|
| **Zimuku/Douban scraper maintenance** | High | Use TMDB + Assrt (official API) as primary sources; scrapers are fallback only. Monitor scraper health with automated checks; isolate scraper code for rapid patching when sites change layout |
| **Assrt API stability** | High | Multi-source redundancy design. When Assrt is unavailable, fallback to OpenSubtitles. Cache successful results to reduce dependency |
| **TMDB API rate limits** | Medium | Server-side caching (TTL 1 hour) + request debounce + batch queries. Free tier limits manageable for single-user deployment |
| **Fansub naming format diversity** | Medium | Configurable regex engine + community-contributed format definitions. AI parsing (already implemented) handles edge cases |
| **Go BT library maturity** | Medium | Phase 3 integrates qBittorrent first (already done). Built-in BT engine (anacrolix/torrent) is P3 priority — only pursued after qBit integration is proven stable |
| **Single developer bandwidth** | High | Strict phase ordering — each phase is independently usable and shippable. Phase 1 MVP is absolute priority. Four phases at 8-10 + 6-8 + 6-8 + 4-6 weeks = ~30 weeks total; realistic for single developer with no deadline pressure |

### Competition Risks

| Risk Scenario | Likelihood | Impact | Vido's Response |
|---------------|-----------|--------|-----------------|
| Seerr accelerates explore/filter improvements | High | Phase 2 differentiation shrinks | Accelerate Phase 2 development; focus on Asian content curation (Seerr unlikely to invest here) |
| MediaManager adds subtitle management | Medium | Phase 1 moat eroded | Fansub naming resolution for Traditional Chinese is a technical barrier; English-speaking developers are unlikely to replicate |
| Bazarr fixes Traditional Chinese bugs | Low | Phase 1 core differentiation gone | These bugs have existed 2+ years unfixed; maintainer priorities lie elsewhere |
| New all-in-one Chinese competitor appears | Medium-Low | Full competition | First-mover advantage + community reputation + open-source contributors |

### Risk Monitoring

- **Weekly:** API usage, costs, scraper health
- **Per release:** Dependency security audit, scraper compatibility verification
- **Monthly:** Performance metrics, competitor landscape check
- **Escalation:** Any HIGH risk materializes → immediate scope adjustment within current phase
