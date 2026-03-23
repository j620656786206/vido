# Epic 18: Service Health Monitoring
**Phase:** Phase 4 — Polish & Ecosystem

Users see the connection status of all configured external services (Sonarr, Radarr, qBittorrent, Plex, Jellyfin, Prowlarr, NZBGet) in a unified health panel, receive disk space warnings when storage is running low, and can browse a searchable/filterable activity log showing all system operations. This provides operational confidence that everything is working correctly.

**v4 Feature IDs covered:** P4-020, P4-021, P4-022

**Dependencies on Completed Work:**
- Epic 4: qBittorrent health monitoring (Story 4-6)
- Epic 6: Service connection status dashboard (Story 6-4), system logs viewer (Story 6-3)

**Stories (to be created):**
- L-1: Unified service health panel — connection status for all configured services with last-checked timestamps and error details
- L-2: Disk space monitor — monitor configured media paths, warn at configurable threshold (default 10%), critical alert at 5%
- L-3: Activity log — searchable, filterable log of all system operations (scans, downloads, subtitle searches, requests) with severity levels

**Success Criteria:**
- Health status updates within 30s of a service state change
- Disk warning triggers at configurable threshold with clear actionable message
- Activity log supports full-text search across 100,000+ entries in <500ms
