# Summary

**Architecture Overview (v4):**

Vido v4 is a single-user NAS media management platform with:
- **Plugin Architecture:** Go interface-based plugin system for media servers (Plex, Jellyfin), downloaders (qBittorrent, NZBGet), and DVR (Sonarr, Radarr)
- **SSE Hub:** Server-Sent Events for real-time download/scan/subtitle progress (replaces polling)
- **Subtitle Engine:** Multi-source subtitle pipeline with content-based language detection and OpenCC 簡繁轉換
- **No Authentication:** Single-user deployment, auth deferred to v5.0

**Pattern Enforcement Status:**

| Category | Patterns Defined | Current Compliance | Migration Required |
|----------|------------------|-------------------|-------------------|
| Naming | 15 patterns | ⚠️ Partial | Phase 1 (slog migration) |
| Structure | 12 patterns | ❌ Low | Phase 1 (backend consolidation) |
| Format | 8 patterns | ✅ High | Phase 2-4 (implementation) |
| Communication | 6 patterns | ⚠️ Partial | Phase 3-4 (frontend setup) |
| Process | 6 patterns | ❌ Low | Phase 2 (error handling, caching) |

**Total Patterns:** 47 consistency rules defined

**Architectural Decisions (8 active):**

1. Frontend Styling: Tailwind CSS
2. Testing Infrastructure: Go testing + testify / Vitest + RTL
3. ~~Authentication~~ — REMOVED in v4 (single-user, no auth)
4. Caching Strategy: Tiered In-Memory + SQLite (+ server-side TMDB filtering cache)
5. Background Task Processing: Lightweight Worker Pool
6. Error Handling & Logging: slog + Unified AppError
7. Plugin Architecture: Go Interfaces (embedded, no hot-reloading)
8. SSE Hub: Native Go http.Flusher + buffered channels
9. Subtitle Engine Pipeline: Provider interface with multi-source scoring

(8 active decisions: #3 removed, #7/#8/#9 added)

**Critical Refactoring Needed:**
1. Consolidate dual backend into `/apps/api`
2. Migrate from `zerolog` to `slog`
3. Implement unified `AppError` types
4. Establish TanStack Query patterns in frontend
5. Enforce test co-location from start
6. Implement plugin interfaces and manager
7. Build SSE hub infrastructure
8. Build subtitle engine pipeline

**Ready for Implementation:** All patterns documented and ready for execution.

---

**Next Action:** These patterns will guide all code implementation during the consolidation and feature development plan.

---
