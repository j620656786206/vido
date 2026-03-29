# Epic 7b: Multi-Library Media Management

**Phase:** Phase 1 — Core Media Pipeline (Amendment)

Users can create multiple media libraries, assign content types (movie/series) per library, add multiple folder paths to each library, and manage libraries from both the Setup Wizard and Settings page. The scanner uses DB-based library configuration instead of environment variables, enabling per-library content type classification during scanning.

**v4 Feature IDs covered:** P1-001-A, P1-001-B, P1-001-C, P1-001-D, P1-001-E

**Dependencies on Completed Work:**

- Epic 1: Repository pattern, Docker deployment, config system, media folder configuration (Story 1.5)
- Epic 5: Library grid/list views, settings page foundation
- Epic 6: Setup Wizard (Story 6-1), settings page (Story 6-0)
- Epic 7: Scanner service, scan scheduling, progress tracking (Stories 7-1 through 7-4)

**Related Documents:**

- PRD Amendment: `prd/prd-multi-library-amendment.md`
- ADR: `architecture/adr-multi-library-media-management.md`
- UX Specification: `multi-library-ux-spec.md`

**Stories:**

- 7b-1: DB migration + models + repository — Create `media_libraries` and `media_library_paths` tables, Go models, `MediaLibraryRepository` interface + SQLite implementation
- 7b-2: Library CRUD API — RESTful endpoints for library and path management with validation
- 7b-3: Setup Wizard multi-library step — Replace `MediaFolderStep` with `MediaLibrarySetupStep`, update `SetupService` to create library records
- 7b-4: Settings media library manager — Replace env var display in `ScannerSettings` with `MediaLibraryManager` card-based UI with edit/delete modals
- 7b-5: Scanner library integration — Modify `MediaService` and `ScannerService` to read from DB, assign `library_id` and use `content_type` for classification, env var fallback

**Implementation Decisions:**

- Migration #020: additive (new tables + nullable columns), zero-downtime
- `VIDO_MEDIA_DIRS` demoted to fallback, not removed
- Content types: `movie` | `series` only (no `mixed` — Jellyfin proved unreliable)
- Schema reserves `auto_detect`, `detected_type`, `override_type` for Phase 2 (not exposed in UI)
- One path belongs to exactly one library (UNIQUE constraint, no overlapping)
- Story splitting rule applied: backend-only stories (7b-1, 7b-2, 7b-5) separated from full-stack (7b-3, 7b-4)

**Success Criteria:**

- User can create ≥2 media libraries with different content types via Setup Wizard
- User can manage libraries (add/edit/delete) from Settings page
- Scanner uses DB libraries instead of env var
- Existing NAS deployment migrates automatically without data loss
- Library CRUD API response time <200ms
