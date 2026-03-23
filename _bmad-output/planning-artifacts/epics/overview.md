# Overview

Vido v4 is an **all-in-one NAS media management interface for Traditional Chinese users**, replacing the fragmented *arr stack + Seerr combination with a single Docker container. Where existing solutions require users to configure and maintain 5-7 separate applications (Sonarr, Radarr, Prowlarr, Overseerr, Bazarr, qBittorrent WebUI, Plex/Jellyfin), Vido provides a unified experience — from content discovery and requesting, through downloading and subtitle management, to library browsing — all with first-class Traditional Chinese (zh-TW) support.

## Epic Structure

The v4 PRD organizes work into **4 phases** with **12 new epics (A-L)**, building on top of **6 completed epics (1-6)** from the v3 PRD that established the foundation:

- **Completed (v3):** Epics 1-6 — Infrastructure, TMDB integration, AI parsing, qBittorrent monitoring, library UI, system config. 50 stories delivered.
- **Phase 1 — Core Media Pipeline:** Epics A-C — Folder scanning, subtitle engine (the MVP differentiator), AI subtitle enhancement
- **Phase 2 — Discovery & Browse:** Epics D-F — Homepage TV wall, advanced search/filter, rich detail pages
- **Phase 3 — Automation & Integration:** Epics G-I — Request system, download management v2, indexer integration
- **Phase 4 — Polish & Ecosystem:** Epics J-L — Statistics dashboard, media server integration, service health monitoring

## Feature ID System

All v4 features are tracked with IDs in the format `P{phase}-{number}`:
- **P1-001 through P1-021:** Phase 1 features (Epics A-C)
- **P2-001 through P2-025:** Phase 2 features (Epics D-F)
- **P3-001 through P3-021:** Phase 3 features (Epics G-I)
- **P4-001 through P4-022:** Phase 4 features (Epics J-L)

See [Requirements Inventory](./requirements-inventory.md) for the complete feature ID mapping, and [Completed Work Registry](./completed-work-registry.md) for how v3 stories map to v4 feature IDs.
