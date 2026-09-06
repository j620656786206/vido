---
name: Multi-Library Media Management (Route 2)
description: Multi-library (per-folder content type) is DONE as Epic 7b — API /api/v1/libraries, wizard step, and the settings manager which lives on /settings/scanner (there is no /settings/libraries route)
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

**STATUS UPDATE 2026-09-06 (verified in code, this note supersedes the pipeline list above):**
Epic 7b shipped all of Phase 1 — sprint-status `7b-2-library-crud-api` (REST `/api/v1/libraries`, `mediaLibrariesHandler.RegisterRoutes`), `7b-3-setup-wizard-multi-library`, `7b-4-settings-media-library-manager` (`LibraryCard` / `LibraryEditModal` / `MediaLibraryManager` under `apps/web/src/components/settings/`), `7b-5-scanner-library-integration` are all `done`. Migration `020_create_media_libraries.go` exists.
**The manager is mounted inside `ScannerSettings.tsx` → route `/settings/scanner` (nav label 媒體庫掃描).** There is NO `/settings/libraries` route and never was; two empty-library CTAs linked to it for months (fixed 2026-09-06 in PR #392 then retargeted to `/settings/scanner`). If a link to "library settings" is needed, use `/settings/scanner`.
