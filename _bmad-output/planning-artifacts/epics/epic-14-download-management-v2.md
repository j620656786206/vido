# Epic 14: Download Management v2
**Phase:** Phase 3 — Automation & Integration

Enhanced download management building on the existing qBittorrent monitoring with optional NZBGet support for Usenet users, SSE-based real-time progress push replacing polling, download completion notifications (in-app and optional webhook), and a future internal BitTorrent engine for users who prefer not to run a separate torrent client.

**v4 Feature IDs covered:** P3-010, P3-011, P3-012, P3-013, P3-014

**Dependencies on Completed Work:**
- Epic 4: qBittorrent connection, monitoring, download dashboard, status filtering, completion detection, health monitoring (Stories 4-1 through 4-6) — provides P3-010

**Stories (to be created):**
- H-1: SSE real-time progress hub — replace polling with server-sent events for download progress, speed, ETA
- H-2: NZBGet plugin — connection config, download monitoring, status mapping to unified model
- H-3: Download completion notifications — in-app toast notifications + optional webhook for external integrations
- H-4: Download dashboard v2 — unified view across qBittorrent + NZBGet with source indicators

**Success Criteria:**
- SSE latency <1s from download state change to UI update
- Download status accurately reflected across both qBittorrent and NZBGet sources
- Notifications delivered within 5s of download completion
