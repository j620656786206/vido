# Epic 9c: Media Technical Info & NFO Integration

**Phase:** Phase 1 — Core Media Pipeline (Amendment)

Users can see technical information badges (codec, resolution, audio format) on media detail pages, leverage existing NFO sidecar files for precise TMDB matching when migrating from Kodi/Jellyfin, and filter unmatched media in the library view. Automated re-scans respect a clear data source priority chain, never overwriting user corrections.

**v4 Feature IDs covered:** P1-030, P1-031, P1-032, P1-033, P2-030

**Dependencies on Completed Work:**

- Epic 1: Repository pattern, Docker deployment, config system
- Epic 2: Media entity storage, TMDB metadata client
- Epic 3: AI parsing, multi-source fallback, enrichment pipeline
- Epic 5: Library UI (grid/list views, detail pages)
- Epic 7b: Multi-library management, scanner service

**Related Documents:**

- PRD: `prd/functional-requirements.md` (Section 3.4 媒體 Metadata 強化)
- ADR: `architecture/adr-media-info-nfo-pipeline.md`
- UX Specification: `ux-design-specification.md`

**Stories:**

- 9c-1: DB Schema Migration — Migration #021, tech info columns, series file_size, MetadataSource constants, ShouldOverwrite()
- 9c-2: NFO Sidecar Reader — NFO detection in enrichment pipeline, XML/URL parser, uniqueid TMDB matching, ShouldOverwrite gate
- 9c-3: FFprobe Integration — Docker ffprobe, Go service with semaphore/timeout, tech info extraction, series file_size aggregation
- 9c-4: Technical Info Badges UI + Unmatched Filter — TechBadge components, detail page integration, unmatched filter API + UI

**Dependencies:**

- Story 9c-1: Independent (no prerequisites within epic)
- Story 9c-2: Depends on 9c-1 (needs schema + MetadataSource constants)
- Story 9c-3: Depends on 9c-1 (needs schema); parallel with 9c-2
- Story 9c-4: Depends on 9c-2 + 9c-3 (needs API data from both)

**Implementation Decisions:**

- Migration #021: Additive (ALTER TABLE ADD COLUMN), zero-downtime, all new columns nullable
- NFO before AI Parse: NFO provides exact IDs, saves API costs, more accurate
- FFprobe last: Independent of metadata resolution, skip if NFO had streamdetails
- Data source priority: manual(100) > nfo(80) > tmdb(60) > douban(50) > wikipedia(40) > ai(20)
- FFprobe graceful degradation: `exec.LookPath` at startup, disabled if not found
- Docker image +30MB (alpine ffmpeg package)
- subtitle_tracks: JSON array `[{language, format, external}]`
- Series tech info: Representative (from first file), not per-episode
- Unmatched filter: `WHERE tmdb_id IS NULL OR tmdb_id = 0`

**Success Criteria:**

- Migration #021 runs without data loss on existing databases
- NFO files with TMDB/IMDB uniqueid achieve 100% match rate (vs ~85% for filename parsing)
- FFprobe extracts tech info for all supported formats (MKV, MP4, AVI)
- Tech badges render correctly on detail pages with proper null handling
- Unmatched filter accurately counts and displays unmatched media
- All new endpoints respond <300ms for 1,000 items
- Backend test coverage >80% for new services
