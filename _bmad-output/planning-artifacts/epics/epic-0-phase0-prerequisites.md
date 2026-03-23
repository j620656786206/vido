# Epic 0: Phase 0 Architecture Prerequisites
**Phase:** Phase 0 — Infrastructure (pre-Epic 7)

> **STATUS: COMPLETED (2026-03-23)**

Prepare the codebase architecture for Epic 7 (Media Library Scanner) and Epic 8 (Subtitle Engine) by adding subtitle tracking fields, extending repository interfaces, and implementing the SSE hub for real-time event broadcasting.

**v4 Feature IDs covered:** Infrastructure — enables P1-001~P1-019

**Dependencies on Completed Work:**
- Epic 1: Repository pattern, database migration system
- Epic 4: qBittorrent events pattern (ChannelEmitter reference)

**Stories:**
- 0-1: Subtitle fields migration — Add subtitle_status/path/language/last_searched/search_score to movies and series tables
- 0-2: Repository extensions — Add BulkCreate, FindByParseStatus, UpdateSubtitleStatus, FindBySubtitleStatus, FindNeedingSubtitleSearch
- 0-3: SSE hub — Implement global event broadcasting for scan/subtitle progress

**Implementation Decisions (Gate 2B — 2026-03-23):**
- See `architecture/phase0-prerequisites-spec.md` for full technical specification
- Migration 018 follows existing ALTER TABLE pattern from migration 006
- Repository methods follow existing scan helper patterns
- SSE hub uses native Go http.Flusher + buffered channels (no external deps)

**Success Criteria:**
- `go build ./cmd/api/` passes with all new code ✅
- SSE hub tests pass (7/7) ✅
- Migration applies cleanly to existing database ✅
