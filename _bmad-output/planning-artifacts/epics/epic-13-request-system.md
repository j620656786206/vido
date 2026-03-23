# Epic 13: Request System
**Phase:** Phase 3 — Automation & Integration

Users can one-click request movies or TV shows from explore or detail pages, request specific seasons or individual episodes, track request status through a clear pipeline (pending → searching → downloading → completed → failed), optionally route requests through Sonarr/Radarr APIs for automated downloading, and auto-trigger subtitle search upon download completion. This replaces the need for a separate Overseerr/Jellyseerr instance.

**v4 Feature IDs covered:** P3-001, P3-002, P3-003, P3-004, P3-005

**Dependencies on Completed Work:**
- Epic 2: TMDB media identification (for request targets)
- Epic 4: qBittorrent integration pattern (reusable for download pipeline)
- Epic 8: Subtitle engine (for auto-trigger on completion)

**Stories (to be created):**
- G-1: Request UI — one-click request button on explore/detail pages with movie/TV differentiation
- G-2: Partial request — request specific seasons or episodes for TV shows
- G-3: Request status tracking — status pipeline with real-time updates (pending/searching/downloading/completed/failed)
- G-4: Sonarr/Radarr DVR plugin — optional integration to route requests through *arr stack APIs
- G-5: Auto-subtitle trigger — automatically initiate subtitle search when a requested download completes

**Success Criteria:**
- Request-to-download pipeline initiates in <30s when Sonarr/Radarr is configured
- Request status accurately reflects real download state
- Auto-subtitle trigger fires within 60s of download completion
