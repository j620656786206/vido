# Epic 17: Media Server Integration
**Phase:** Phase 4 — Polish & Ecosystem

Users can connect their Plex or Jellyfin media server to sync watch history, see a "繼續觀看" (continue watching) section on the homepage, and synchronize library inventory to mark owned content in explore views with "已有" badges. This bridges Vido's management capabilities with the user's actual viewing experience.

**v4 Feature IDs covered:** P4-010, P4-011, P4-012

**Dependencies on Completed Work:**
- Epic 1: External service connection patterns, secrets management
- Epic 4: Service health monitoring pattern (reusable for Plex/Jellyfin)
- Epic 10: Homepage (for "繼續觀看" block and "已有" badges)

**Stories (to be created):**
- K-1: Plex plugin — connection config, auth token management, API client
- K-2: Jellyfin plugin — connection config, API key management, API client
- K-3: Watch history sync — pull watch history from Plex/Jellyfin, display "繼續觀看" on homepage
- K-4: Library inventory sync — sync owned media list to enable "已有" badges in explore views

**Success Criteria:**
- Library sync completes in <2 minutes for 5,000 items
- Watch history updates within 5 minutes of playback on media server
- "繼續觀看" section shows accurate resume points
