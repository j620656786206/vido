# UX Design Gap Analysis — PRD v4

**Date:** 2026-03-23
**Purpose:** Identify new pages/flows required for v4 that are not covered by the current `ux-design.pen` design file.

---

## Current UX Coverage (Flows A-F)

The existing design covers media library browsing and settings — approximately **30% of v4's UI surface area**.

| Flow | Screens | v4 Coverage |
|------|---------|-------------|
| A: Browse Desktop | Empty → Loading → Grid → List | P1-007 (DONE) |
| B: Hover/Detail Desktop | Hover → Context Menu → Detail → Detail Context | P1-007 (DONE) |
| C: Search/Filter/Settings Desktop | Search+Filter → Batch Ops → Settings | P1-007, Settings (DONE) |
| D: Browse Mobile | Empty → Loading → Grid → Sort → Filter | P1-007 mobile (DONE) |
| E: Interaction Mobile | Context Menu → Detail → Detail Context | P1-007 mobile (DONE) |
| F: Batch/Settings Mobile | Batch Ops → Settings | Settings mobile (DONE) |

---

## New Flows Required for v4

### Phase 1 — New Flows

**Flow G: Subtitle Management**
- Subtitle search results panel (multi-source results with scoring)
- Subtitle download progress indicator
- Batch subtitle processing view (whole season)
- Subtitle status indicators on media cards/detail pages
- Manual subtitle search UI with source selection
- **Priority:** HIGH — Core v4 MVP differentiator

**Flow H: Media Scanner**
- Scan progress view (file count, current file, ETA)
- Scan results summary (matched / unmatched / errors)
- Unmatched items list with manual match UI
- Scan schedule configuration in settings
- **Priority:** HIGH — Phase 1 required

### Phase 2 — New Flows

**Flow I: Homepage TV Wall**
- Hero Banner with auto-rotating featured content
- Customizable explore blocks (add/remove/reorder)
- Trending section with server-side filtered results
- "繼續觀看" section (when Plex/Jellyfin connected)
- "最近新增" section
- "字幕待處理" section
- **Priority:** HIGH — Phase 2 hero feature, requires new landing page design

**Flow J: Advanced Search**
- Multi-filter chip UI (persistent, not in dropdown)
- Filter categories: genre, year range, region/language, rating range, streaming platform
- Complex sort controls (multi-field)
- Instant search with debounced suggestions (movies, shows, people)
- Saved filter presets management
- **Priority:** HIGH — Phase 2 core interaction

**Flow K: Rich Detail Page v2**
- TMDB + Douban dual rating display
- TV show season/episode expandable list with subtitle status per episode
- Streaming platform availability badges (Netflix/Disney+/KKTV etc.)
- Related recommendations section
- Trailer embed (YouTube)
- Douban link
- Request button integration
- **Priority:** MEDIUM — Extends existing detail page

### Phase 3 — New Flows

**Flow L: Request System**
- Request button on explore/detail pages
- Partial request UI (select specific seasons/episodes)
- Request status tracker page (pending/searching/downloading/completed/failed)
- Sonarr/Radarr status indicators
- **Priority:** MEDIUM — Phase 3

**Flow M: Download Dashboard v2**
- SSE-powered real-time download progress (replaces polling)
- Download completion notifications (toast/banner)
- NZBGet integration view (when configured)
- Download history with search/filter
- **Priority:** MEDIUM — Extends existing download view

### Phase 4 — New Flows

**Flow N: NAS Dashboard**
- Media library statistics overview (total counts, disk usage, resolution pie chart)
- Subtitle coverage rate visualization (bar/donut chart)
- Genre/region/year distribution charts
- Recently added media list (7/30 day)
- Service health panel (connection status for all plugins)
- Disk space usage with warning thresholds
- Activity log (searchable, filterable)
- **Priority:** LOW — Phase 4

### Cross-Phase

**Flow O: Plugin Settings**
- Plugin list with connection status indicators
- Per-plugin configuration form (URL, API key, test connection button)
- Plugin enable/disable toggle
- Plugin health check results
- **Priority:** MEDIUM — Needed when Phase 3 plugins are implemented

---

## Navigation Structure Impact

Current navigation is **Library-centric** (library is the main view). v4 shifts to **Explore-centric**:

| Current Nav | v4 Nav |
|-------------|--------|
| Library (main) | Homepage/Explore (main) |
| Downloads | Library |
| Settings | Downloads |
| — | Requests |
| — | Dashboard |
| — | Settings |

This requires redesigning the navigation shell (currently Story 5-0).

---

## Design Priority Order

1. **Flow G + H** (Phase 1) — Subtitle management + Scanner — needed before Phase 1 development
2. **Flow I** (Phase 2) — Homepage TV Wall — hero feature, needs early design
3. **Flow J + K** (Phase 2) — Advanced search + Rich detail — extends existing patterns
4. **Flow L + M** (Phase 3) — Request + Download v2
5. **Flow N + O** (Phase 4) — Dashboard + Plugin settings

---

## Next Steps

- [ ] Create design brief for Subtitle Engine UI (Flow G) — needed for Epic 8
- [ ] Create design brief for Homepage TV Wall (Flow I) — needed for Epic 10
- [ ] Update `ux-design.pen` with new flows (when starting each Phase)
- [ ] Update `scripts/export-pen-screenshots.py` SCREENS dict for new flows
