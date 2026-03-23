# Epic 7: Media Library Scanner
**Phase:** Phase 1 — Core Media Pipeline

Users can configure media library paths, trigger recursive scanning of their NAS folders, have filenames parsed automatically (regex + AI), auto-match against TMDB with zh-TW metadata fallback, and browse their scanned library in grid/list views. This epic combines the completed parsing and metadata infrastructure from v3 with new folder-level scanning capabilities to deliver the foundational "point at your folders and go" experience.

**v4 Feature IDs covered:** P1-001, P1-002, P1-003, P1-004, P1-005, P1-006, P1-007

**Dependencies on Completed Work:**
- Epic 1: Repository pattern, Docker deployment, config system, media folder configuration
- Epic 2: TMDB API integration, standard filename parser, media entity/DB storage, search UI, grid view
- Epic 3: AI fansub parsing, multi-source metadata fallback chain
- Epic 5: Library grid/list views, search, sort, filter, detail pages

**Stories (to be created):**
- A-1: Recursive folder scanner — walk configured media paths, detect video files, create scan records
- A-2: Scheduled scan service — configurable cron-based automatic re-scanning (new/changed/removed files)
- A-3: Manual scan trigger UI — button in settings/library to trigger full or incremental scan with progress tracking
- A-4: Scan progress tracking — SSE-based real-time progress (files found, parsed, matched, errors)

**Success Criteria:**
- Scan 1,000 files in <5 minutes end-to-end (scan + parse + match)
- >99% parse rate for standard naming convention files (e.g., `Movie.Name.2024.1080p.BluRay.mkv`)
- Incremental scan detects new/removed files without re-processing unchanged files
