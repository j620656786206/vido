# PRD v4 Migration Changelog

**Migration Date:** 2026-03-23
**Migrated By:** Alex Yu + Claude Code
**Source Document:** `prd/prd-v4-source.md` (copied from Claude Desktop discussion)

---

## Summary

Vido's product positioning shifted from **"繁中字幕管理工具"** (v3) to **"All-in-one 媒體管理介面"** (v4). All planning documents were rewritten to align with the new 4-phase structure.

## Phase Structure Change

| v3 | v4 |
|----|-----|
| Phase 1: MVP (search + metadata + basic parsing) | Phase 1: 字幕核心穩定 (scanner + subtitle engine) |
| Phase 2: 1.0 Core Platform (AI parsing + qBit + library + settings + auth) | Phase 2: 媒體探索 (Hero Banner + search + detail) |
| Phase 3: Growth (subtitles + automation + recommendations + multi-user + mobile) | Phase 3: 請求流程 + 下載管理 (requests + Sonarr/Radarr + SSE) |
| — | Phase 4: NAS Dashboard (stats + Plex/Jellyfin + health) |

## Epic Structure Change

| v3 (14 Epics) | v4 (12 Epics) | Status |
|---------------|---------------|--------|
| Epic 1: Foundation | Completed, infrastructure reused | ARCHIVED |
| Epic 2: TMDB Search | Completed, maps to P1-003/P2-013 | ARCHIVED |
| Epic 3: AI Fansub Parsing | Completed, maps to P1-002/P1-004 | ARCHIVED |
| Epic 4: qBittorrent Monitor | Completed, maps to P3-010 | ARCHIVED |
| Epic 5: Media Library UI | Completed, maps to P1-007 | ARCHIVED |
| Epic 6: Settings/Backup | 9/11 done, maps to Settings | ARCHIVED |
| Epic 7: Auth | **DELETED** — v4 is single-user | DELETED |
| Epics 8-14 | Replaced by Epics A-L | ARCHIVED |
| — | Epic 7: Media Library Scanner (P1) | NEW |
| — | Epic 8: Subtitle Engine (P1) | NEW |
| — | Epic 9: AI Subtitle Enhancement (P1) | NEW |
| — | Epic 10: Homepage TV Wall (P2) | NEW |
| — | Epic 11: Advanced Search (P2) | NEW |
| — | Epic 12: Rich Media Detail Page (P2) | NEW |
| — | Epic 13: Request System (P3) | NEW |
| — | Epic 14: Download Management v2 (P3) | NEW |
| — | Epic 15: Indexer Integration (P3) | NEW |
| — | Epic 16: Media Stats Dashboard (P4) | NEW |
| — | Epic 17: Media Server Integration (P4) | NEW |
| — | Epic 18: Service Health Monitoring (P4) | NEW |

## Feature ID System Change

- **Old:** FR1-FR94 (94 functional requirements)
- **New:** P{phase}-{sequence} (P1-001 through P4-022, ~60 feature IDs)
- **Mapping:** See `prd/functional-requirements.md` Legacy FR Mapping table

## Deleted Features (Not in v4 Scope)

| Old FR | Feature | Reason |
|--------|---------|--------|
| FR67-70 | Password/PIN auth, JWT sessions | Single-user, no auth needed |
| FR71-74 | Multi-user, roles, permissions | Deferred to v5.0 |
| FR83-84 | Auto file rename/move | Not in v4 scope |
| FR87-90 | External API, webhooks, OpenAPI | Not in v4 scope |
| FR93-94 | Mobile app, remote control | Not in v4 scope |

## New Architectural Decisions

| Decision | Description |
|----------|-------------|
| #7 Plugin Architecture | Go interfaces for MediaServer/Downloader/DVR plugins |
| #8 SSE Hub | Server-Sent Events replacing polling for real-time updates |
| #9 Subtitle Engine | Multi-source search + OpenCC conversion pipeline |
| #3 Auth REMOVED | JWT authentication removed for single-user v4 |

## Files Modified

### PRD (9 files)
- `prd/index.md` — Updated TOC
- `prd/project-scoping-phased-development.md` — Complete rewrite (4 phases)
- `prd/functional-requirements.md` — Complete rewrite (v4 feature IDs)
- `prd/success-criteria.md` — Complete rewrite (v4 metrics)
- `prd/user-journeys.md` — Complete rewrite (3 journeys)
- `prd/technical-considerations.md` — Updated (auth→plugins/SSE/subtitle)
- `prd/non-functional-requirements.md` — Updated (remove auth NFRs, add new)
- `prd/innovation-novel-patterns.md` — Updated (add 3 new innovations)
- `prd/web-application-specific-requirements.md` — Updated (SSE, context menus)

### Epics (20+ files)
- 12 new epic files (A-L)
- 6 completed epic files updated with header
- 8 old epic files moved to archive/
- epic-list.md, overview.md, index.md, requirements-inventory.md rewritten
- completed-work-registry.md created

### Architecture (8+ files)
- core-architectural-decisions.md — Major update
- project-structure-boundaries.md — New packages
- implementation-gap-analysis.md — Rewritten
- summary.md, next-steps.md — Updated
- Pattern files (2, 4, 5) — New patterns added
- enforcement-guidelines.md — Auth→Plugin rules

### Implementation Artifacts
- sprint-status.yaml — Complete rewrite
- 19 story files moved to archive/

### Project Context
- project-context.md — Major update (auth removal, 3 new decisions)

### UX Design
- ux-design-gap-analysis-v4.md — New (gap analysis for Flows G-O)
