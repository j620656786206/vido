---
name: Multi-Library Media Management (Route 2)
description: Epic-level feature to replace single media_folder_path with multi-library system supporting per-folder content type assignment
type: project
---

**Decision:** Route 2 — Progressive Enhancement (decided 2026-03-29 party mode session)

**Why:** Current system only supports single media_folder_path (Setup Wizard) or VIDO_MEDIA_DIRS env var (Scanner), with no content type assignment. User's NAS has /movies + /tv, needs to tell Vido which folder is which type. All competitors (Plex, Jellyfin, Emby) support this.

**How to apply:**
- Phase 1: Multi-folder + manual type (movie/series) = 5-6 stories, 1 Epic
- Schema reserves `auto_detect`, `detected_type`, `override_type` for Phase 2
- Phase 2 (future, if needed): Auto-detection via existing filename parser (tested 2097 NAS files, 0% misclassification)
- Phase 3 (future, unlikely): Dynamic views / tag system

**Key files produced:**
- PRD: `_bmad-output/planning-artifacts/prd/prd-multi-library-amendment.md`
- UX spec: `_bmad-output/planning-artifacts/multi-library-ux-spec.md`

**Workflow pipeline (pending):**
1. UX: Supplement Setup Wizard designs in ux-design.pen (currently NO wizard screens exist)
2. UX: Design new multi-library Setup + Settings screens
3. Architect: Update architecture docs
4. SM: Create Epic + Stories
5. Dev: Implement
6. QA: Test Architecture
7. Code Review

**Data model:** New tables `media_libraries` + `media_library_paths`, migration #020, deprecate `VIDO_MEDIA_DIRS` to fallback.
