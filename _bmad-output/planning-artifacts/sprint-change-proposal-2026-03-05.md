# Sprint Change Proposal — Media Type Filter Taxonomy Correction

**Date:** 2026-03-05
**Triggered by:** Story 5-5 (Library Filtering) — discovered during Pencil design review
**Change Scope:** Minor
**Status:** Approved

---

## Section 1: Issue Summary

### Problem Statement

The media type filter in Epic 5's library filtering feature incorrectly includes "Anime" as a distinct media type alongside "Movie" and "TV Show". In the backend data model, only two media types exist (`Movie` and `Series`). "Anime/Animation" is a **genre** that spans both types (e.g., "Your Name" is a Movie with Animation genre; "Demon Slayer" is a TV Series with Animation genre).

### Discovery Context

Identified during Pencil UI design work for Epic 5 Media Library. The Architect agent confirmed the taxonomy mismatch through:

1. **Data model analysis:** `Movie` and `Series` structs have no "anime" type — Animation is stored in the `genres` JSON array field
2. **Sample data contradiction:** Design Brief sample data shows anime titles split across Movie and TV types
3. **Industry benchmark:** Netflix, Disney+, Plex, Jellyfin all use Movie/TV as top-level type filters, with Animation as a genre

### Secondary Issue

The Design Brief's genre filter mockup shows a fixed list of genres (Drama, Action, Animation, Comedy, Sci-Fi), which could be misinterpreted as a hardcoded set. The implementation spec (Story 5-5) correctly defines a dynamic `GET /api/v1/library/genres` endpoint, but the Design Brief lacks explicit annotation. Additionally, no NFR exists for handling large genre lists (20+ genres) in the filter UI.

---

## Section 2: Impact Analysis

### Epic Impact

| Epic | Impact | Details |
|------|--------|---------|
| Epic 5 (Media Library Management) | **Direct** | Story 5.5 AC1 media type options need correction |
| All other Epics (1-4, 6-14) | None | No filter taxonomy references |

### Story Impact

| Story | Impact | Change Needed |
|-------|--------|---------------|
| 5-5 (Library Filtering) | **Direct** | Remove Anime from media type filter options |
| All other Stories | None | No dependency on media type filter taxonomy |

### Artifact Conflicts

| Artifact | Impact | Change Needed |
|----------|--------|---------------|
| PRD (functional-requirements.md) | None | FR7 says "filter by media type" without specifying options |
| Architecture docs | None | No filter taxonomy definitions |
| epic-5-media-library-management.md | **Direct** | Line 157: remove Anime from media type list |
| 5-5-library-filtering.md | **Direct** | Line 19: remove Anime, align control type |
| epic5-media-library-design-brief.md | **Direct** | Lines 484-488, 595-596, 473-479, 589-591, 627-630: multiple fixes |
| ux-design-specification.md | None | "Animation" mentions are user persona context only |

### Technical Impact

- **Zero code impact** — Story 5-5 is in `ready-for-dev` status, no implementation exists yet
- Backend `GET /api/v1/library/genres` endpoint design is already correct (dynamic)
- Frontend Task 2.4 already specifies `Movie / TV Show` (no Anime) — only AC text was wrong

---

## Section 3: Recommended Approach

### Selected Path: Direct Adjustment

**Rationale:**
- Smallest possible change scope (3 files, ~15 lines total)
- No code rollback needed — Story is pre-implementation
- No MVP scope change — filtering feature remains fully intact
- No architectural change — data model already correct
- Zero risk to timeline

**Effort:** Low (document edits only)
**Risk:** Low (no code, no dependencies)
**Timeline Impact:** None

---

## Section 4: Detailed Change Proposals

### 4.1 Epic File: epic-5-media-library-management.md

**Line 157:**
```
OLD: - Media Type (Movie, TV Show, Anime)
NEW: - Media Type (Movie, TV Show)
```

### 4.2 Story File: 5-5-library-filtering.md

**Line 19:**
```
OLD:      - Media Type (Movie, TV Show, Anime — radio/checkbox)
NEW:      - Media Type (Movie, TV Show — segmented control)
```

### 4.3 Design Brief: epic5-media-library-design-brief.md

**Change A — Desktop Filter Panel Type (Lines 484-488):**
Remove `| ( ) Anime |` line from the Type radio button group.

**Change B — Mobile Bottom Sheet Type (Lines 595-596):**
```
OLD: | [All] [Movie] [TV] [Anime]|
NEW: | [All] [Movie] [TV]        |
```

**Change C — Genre Dynamic Annotation (Lines 473, 589):**
Change `Genre` headers to `Genre (dynamic from API)` in both desktop and mobile mockups.

**Change D — New Design Note (after line 604):**
Add:
```
- Genre list is dynamically populated from `GET /api/v1/library/genres` — mockup values are illustrative only
- NFR: Genre filter must handle 20+ genres gracefully without excessive panel height. UX Designer to determine specific pattern (collapsible sections, search-select, multi-column grid, etc.)
```

**Change E — Translation Table (Lines 627-630):**
Remove `| Anime | Animation |` row from the translation table.

---

## Section 5: Implementation Handoff

### Change Scope Classification: Minor

All changes are document-only edits to planning and implementation artifacts. No code changes required.

### Handoff Plan

| Role | Responsibility | Files |
|------|---------------|-------|
| **Architect (current session)** | Apply all approved edits to artifacts | All 3 files listed above |
| **UX Designer (follow-up)** | Design genre filter pattern for 20+ genres based on updated Design Brief NFR | epic5-media-library-design-brief.md, Pencil design |
| **Dev Team** | No action — Story 5-5 spec will be correct before development begins | N/A |

### Success Criteria

1. All 3 affected files updated with approved changes
2. No remaining references to "Anime" as a media type filter option in any spec document
3. Genre filter sections clearly annotated as dynamically populated
4. NFR for large genre list usability documented for UX Designer follow-up
