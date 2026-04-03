---
stepsCompleted: ["step-01-validate-prerequisites", "step-02-design-epics", "step-03-create-stories", "step-04-final-validation"]
inputDocuments:
  - "_bmad-output/planning-artifacts/prd/functional-requirements.md"
  - "_bmad-output/planning-artifacts/prd/non-functional-requirements.md"
  - "_bmad-output/planning-artifacts/architecture/adr-media-info-nfo-pipeline.md"
  - "_bmad-output/planning-artifacts/architecture/core-architectural-decisions.md"
  - "_bmad-output/planning-artifacts/ux-design-specification.md"
  - "_bmad-output/planning-artifacts/epics/epic-list.md"
---

# Vido - Epic Breakdown (Epic 9c)

## Overview

This document provides the epic and story breakdown for the Media Technical Info & NFO Integration epic, decomposing requirements P1-030 ~ P1-033 and P2-030 from the PRD, along with the accepted ADR for media info/NFO pipeline.

## Requirements Inventory

### Functional Requirements

- **P1-030**: Media Technical Info — Extract video codec, resolution, audio codec, audio channels, subtitle tracks during scan. Display as visual badges on detail page (e.g., H.265 · 4K · DTS). Data source priority: NFO streamdetails > FFprobe. Supported formats: MKV, MP4, AVI.
- **P1-031**: NFO Sidecar Reading (Read-Only) — Detect same-name `.nfo` sidecar files during scan. Support Kodi-style XML and single-line TMDB URL formats. NFO metadata takes priority over AI parsing and TMDB enrichment. Use `uniqueid` fields (tmdb/imdb) for precise TMDB matching. Read-only — Vido never writes NFO files.
- **P1-032**: Data Source Priority Chain — Establish explicit metadata priority: manual > nfo > tmdb > douban > wikipedia > ai. Each media record stores `metadata_source` field. Foundational infrastructure for all metadata resolution logic.
- **P1-033**: Series File Size — Add `file_size` field to Series model. Calculate total file size per season and per series during scan.
- **P2-030**: Unmatched Media Filter — Add "Unmatched" filter to library page; batch-review all media without TMDB metadata. Display unmatched count as badge on filter option.

### Non-Functional Requirements

- **NFR-P6**: Media library listing API must respond within <300ms (p95) for up to 1,000 items — new columns must not degrade performance.
- **NFR-P13**: Standard regex-based filename parsing must complete within <100ms per file — NFO detection must not slow down scan.
- **NFR-SC1**: SQLite database must support up to 10,000 media items with <500ms query latency — new fields need proper indexing.
- **NFR-R2**: System must gracefully handle all external API failures without crashing — FFprobe must gracefully degrade.
- **NFR-M1**: Backend test coverage must be >80%.
- **NFR-M6**: Database migrations must be versioned and automated.

### Additional Requirements

- **Migration #021**: 7 new columns on `movies` table + 7 columns on `series` table (video_codec, video_resolution, audio_codec, audio_channels, subtitle_tracks, hdr_format, file_size for series).
- **FFprobe Docker packaging**: Alpine `apk add ffmpeg` (~30MB). Startup check via `exec.LookPath("ffprobe")`. Graceful degradation if not found.
- **NFO Parser**: `services/nfo_reader_service.go`. Go `encoding/xml`. Dual format detection (XML / URL).
- **FFprobe Service**: `services/ffprobe_service.go`. Semaphore (maxConcurrent=3), timeout (10s).
- **Data Source Priority**: `ShouldOverwrite()` function with priority map. Applied in EnrichmentService before metadata writes.
- **Enrichment Pipeline Extension**: NFO Detection → AI Parse (if no NFO) → TMDB Enrichment → FFprobe (serial per file).
- **subtitle_tracks JSON schema**: Array of `{language, format, external}` objects.
- **Unmatched filter API**: `GET /api/v1/movies?unmatched=true` + `GET /api/v1/movies/stats` for count badge.

### FR Coverage Map

| FR | Epic | Story | Description |
|----|------|-------|-------------|
| P1-030 | 9c | Story 3 + Story 4 | FFprobe extracts tech info → badges display on UI |
| P1-031 | 9c | Story 2 | NFO sidecar detection, XML/URL parsing, uniqueid matching |
| P1-032 | 9c | Story 1 + Story 2 | metadata_source field + ShouldOverwrite() priority logic |
| P1-033 | 9c | Story 1 | Series file_size field, scan-time calculation |
| P2-030 | 9c | Story 4 | Unmatched filter UI + API endpoint |

**Coverage: 5/5 FRs mapped (100%)**

## Epic List

### Epic 9c: Media Technical Info & NFO Integration

Users can see technical information badges (codec, resolution, audio format) on media detail pages, leverage existing NFO sidecar files for precise TMDB matching when migrating from Kodi/Jellyfin, and filter unmatched media in the library view. Automated re-scans respect a clear data source priority chain, never overwriting user corrections.

**FRs covered:** P1-030, P1-031, P1-032, P1-033, P2-030

**Stories:**
1. DB Schema Migration — New tech info columns + metadata_source + series file_size
2. NFO Sidecar Reader — Scan pipeline NFO detection, XML/URL parsing, TMDB precise match
3. FFprobe Integration — Docker ffprobe, Go service, tech info extraction, API response
4. Technical Info Badges UI + Unmatched Filter — Frontend badges, detail page integration, unmatched filter

**Dependencies:**
- Story 1: Independent (no prerequisites)
- Story 2: Depends on Story 1
- Story 3: Depends on Story 1 (parallel with Story 2)
- Story 4: Depends on Story 2 + Story 3

---

## Epic 9c: Media Technical Info & NFO Integration

Users can see technical information badges (codec, resolution, audio format) on media detail pages, leverage existing NFO sidecar files for precise TMDB matching when migrating from Kodi/Jellyfin, and filter unmatched media in the library view. Automated re-scans respect a clear data source priority chain, never overwriting user corrections.

### Story 9c-1: DB Schema Migration — Tech Info Fields & Data Source Priority

As a **developer**,
I want **the database schema extended with technical info columns, series file_size, and metadata source priority constants**,
So that **subsequent stories (NFO reader, FFprobe, badges UI) have the data foundation they need**.

**Acceptance Criteria:**

1. **Given** the application starts with an existing database
   **When** migration #021 runs
   **Then** `movies` table gains columns: `video_codec TEXT`, `video_resolution TEXT`, `audio_codec TEXT`, `audio_channels INTEGER`, `subtitle_tracks TEXT`, `hdr_format TEXT`
   **And** `series` table gains the same 6 columns plus `file_size INTEGER`

2. **Given** migration #021 completes
   **When** querying existing movie/series records
   **Then** all new columns default to NULL (no data loss, no breakage)

3. **Given** the `MetadataSource` type in `models/movie.go`
   **When** the new constants are added
   **Then** `MetadataSourceNFO = "nfo"` and `MetadataSourceAI = "ai"` are available
   **And** existing constants (tmdb, douban, wikipedia, manual) are unchanged

4. **Given** the `ShouldOverwrite(current, incoming MetadataSource) bool` function
   **When** called with various source combinations
   **Then** it returns true when incoming priority >= current priority
   **And** returns true when current is empty (first data)
   **And** priority order is: manual(100) > nfo(80) > tmdb(60) > douban(50) > wikipedia(40) > ai(20)

5. **Given** the Movie and Series Go models
   **When** the new fields are added
   **Then** JSON serialization uses snake_case (`video_codec`, `audio_channels`, etc.)
   **And** repository INSERT/UPDATE SQL includes all new fields

**Tasks:**
- 1.1: Create migration `021_media_tech_info.go` — ALTER TABLE for movies (6 cols) and series (7 cols)
- 1.2: Add `MetadataSourceNFO` and `MetadataSourceAI` constants + `metadataSourcePriority` map + `ShouldOverwrite()` function
- 1.3: Update `Movie` struct — add VideoCodec, VideoResolution, AudioCodec, AudioChannels, SubtitleTracks, HDRFormat fields
- 1.4: Update `Series` struct — add same 6 fields + FileSize
- 1.5: Update movie repository INSERT/UPDATE SQL to include new columns
- 1.6: Update series repository INSERT/UPDATE SQL to include new columns
- 1.7: Write migration test (tables altered, NULLs, no data loss)
- 1.8: Write `ShouldOverwrite()` unit tests (all priority combinations)
- 1.9: Write model serialization tests (JSON field names)

---

### Story 9c-2: NFO Sidecar Reader

As a **NAS user migrating from Kodi or Jellyfin**,
I want **Vido to detect and read my existing .nfo sidecar files during scan**,
So that **my media gets precise TMDB matching using NFO uniqueid and I don't need to re-tag everything**.

**Acceptance Criteria:**

1. **Given** a video file `/media/movies/Movie.2024.mkv` with a sidecar `/media/movies/Movie.2024.nfo`
   **When** the enrichment pipeline processes this file
   **Then** the NFO file is detected and parsed before AI parsing
   **And** `metadata_source` is set to `"nfo"`

2. **Given** an NFO file containing Kodi-style XML with `<uniqueid type="tmdb">12345</uniqueid>`
   **When** the NFO parser processes it
   **Then** the TMDB ID `12345` is extracted
   **And** TMDB enrichment uses direct lookup (skip search) for this ID

3. **Given** an NFO file containing Kodi-style XML with `<uniqueid type="imdb">tt1234567</uniqueid>`
   **When** the NFO parser processes it
   **Then** the IMDB ID is extracted and used for TMDB find-by-external-id

4. **Given** an NFO file containing a single-line URL like `https://www.themoviedb.org/movie/12345`
   **When** the NFO parser processes it
   **Then** TMDB ID `12345` is extracted from the URL
   **And** `source_format` is set to `"url"`

5. **Given** an NFO file with XML containing `<fileinfo><streamdetails>` with video codec/resolution/audio
   **When** the NFO parser processes it
   **Then** technical info fields (video_codec, video_resolution, audio_codec, audio_channels) are populated from NFO
   **And** FFprobe extraction is skipped for this file (NFO streamdetails takes priority)

6. **Given** a malformed or unrecognizable NFO file
   **When** the NFO parser processes it
   **Then** a warning is logged with the file path and error
   **And** the pipeline falls back to AI parsing (existing flow)
   **And** no crash or scan interruption occurs

7. **Given** a video file with no corresponding .nfo sidecar
   **When** the enrichment pipeline processes this file
   **Then** the existing AI parse → TMDB enrichment flow runs as before (no behavior change)

8. **Given** a media record with `metadata_source = "nfo"` from a previous scan
   **When** a re-scan occurs and the NFO file still exists
   **Then** NFO data is re-read (idempotent)
   **And** `ShouldOverwrite("nfo", "nfo")` returns true (same priority)

9. **Given** a media record with `metadata_source = "manual"` (user corrected)
   **When** a re-scan occurs with an NFO file present
   **Then** NFO metadata is NOT applied (`ShouldOverwrite("manual", "nfo")` returns false)
   **And** the user's manual correction is preserved

**Tasks:**
- 2.1: Create `services/nfo_reader_service.go` with `NFOReaderService`, `NFOData`, `NFOUniqueID`, `NFOStreamDetails` structs
- 2.2: Implement `Parse(nfoPath string) (*NFOData, error)` — format detection (XML prefix check / URL extraction)
- 2.3: Implement `parseXML(content []byte) (*NFOData, error)` — Go `encoding/xml` for `<movie>`, `<tvshow>`, `<episodedetails>` root elements
- 2.4: Implement `findNFOSidecar(videoPath string) string` — same-name .nfo path check
- 2.5: Implement URL extractors — `extractTMDbID(line)` and `extractIMDbID(line)` for single-line NFO format
- 2.6: Integrate into `enrichment_service.go` `enrichSingleItem()` — NFO detection before AI parse, `ShouldOverwrite()` gate
- 2.7: Modify TMDB enrichment to accept direct TMDB ID / IMDB find-by-external-id when NFO provides uniqueid
- 2.8: Write NFO parser unit tests — XML format, URL format, malformed, missing fields, streamdetails
- 2.9: Write enrichment integration tests — NFO priority over AI, manual priority over NFO, fallback on parse failure

---

### Story 9c-3: FFprobe Integration

As a **NAS user**,
I want **Vido to extract technical details (codec, resolution, audio format, HDR) from my video files**,
So that **I can see what quality my media files are in and make informed decisions about my collection**.

**Acceptance Criteria:**

1. **Given** the Docker image build
   **When** the runtime stage is built
   **Then** `ffprobe` binary is available at runtime (`apk add --no-cache ffmpeg`)

2. **Given** the application starts on a system with `ffprobe` installed
   **When** `FFprobeService` initializes
   **Then** `exec.LookPath("ffprobe")` succeeds
   **And** the service is marked as available

3. **Given** the application starts on a system WITHOUT `ffprobe`
   **When** `FFprobeService` initializes
   **Then** a warning is logged: "ffprobe not found — technical info extraction disabled"
   **And** `Probe()` calls return `nil, ErrFFprobeNotAvailable`
   **And** the scan pipeline skips technical info extraction without error

4. **Given** a video file `Movie.2024.mkv` (H.265, 3840x2160, DTS 5.1, HDR10)
   **When** `FFprobeService.Probe(ctx, filePath)` is called
   **Then** it returns `MediaTechInfo` with: VideoCodec="H.265", VideoResolution="3840x2160", AudioCodec="DTS", AudioChannels=6, HDRFormat="HDR10"

5. **Given** 4 concurrent FFprobe requests with `maxConcurrent=3`
   **When** all 4 are submitted
   **Then** only 3 run simultaneously
   **And** the 4th waits for a semaphore slot

6. **Given** a video file on a slow network mount
   **When** FFprobe execution exceeds 10 seconds
   **Then** the context is cancelled
   **And** an error is returned (not a hang)

7. **Given** the enrichment pipeline processes a file that already has `video_codec` set (from NFO streamdetails)
   **When** the FFprobe stage runs
   **Then** FFprobe extraction is skipped (NFO data preserved)

8. **Given** a series with 3 video files (1.2GB, 800MB, 950MB)
   **When** the scan processes these files
   **Then** `series.file_size` is set to the sum: 2,950MB (in bytes)
   **And** file_size is recalculated on each scan

9. **Given** FFprobe extracts subtitle track information from an MKV file
   **When** the results are stored
   **Then** `subtitle_tracks` contains a JSON array with `{language, format, external: false}` for embedded tracks
   **And** external sidecar subtitle files detected on filesystem are added with `{external: true}`

10. **Given** the existing `GET /api/v1/movies/:id` endpoint
    **When** a movie with tech info is requested
    **Then** the response includes `video_codec`, `video_resolution`, `audio_codec`, `audio_channels`, `hdr_format`, `subtitle_tracks` fields

**Tasks:**
- 3.1: Update `Dockerfile` runtime stage — `apk add --no-cache ffmpeg`
- 3.2: Create `services/ffprobe_service.go` — `FFprobeService` struct with semaphore, timeout, `Probe()` method
- 3.3: Implement `parseFfprobeJSON(output []byte) (*MediaTechInfo, error)` — extract streams/format info
- 3.4: Implement startup check — `exec.LookPath("ffprobe")` with graceful degradation
- 3.5: Implement HDR detection — parse color_transfer/color_primaries for HDR10/DolbyVision
- 3.6: Integrate into enrichment pipeline — FFprobe after TMDB enrichment, skip if NFO had streamdetails
- 3.7: Implement series file_size aggregation in scanner service — sum file sizes per series
- 3.8: Implement external subtitle track detection — scan filesystem for sidecar .srt/.ass files, merge with embedded tracks
- 3.9: Wire `FFprobeService` in `main.go` dependency injection
- 3.10: Write FFprobe service unit tests — JSON parsing, semaphore, timeout, graceful degradation
- 3.11: Write enrichment integration tests — FFprobe stage, NFO skip logic
- 3.12: Write series file_size calculation tests

---

### Story 9c-4: Technical Info Badges UI + Unmatched Filter

As a **NAS user**,
I want **to see visual badges showing video quality (H.265, 4K, DTS) on media detail pages and filter unmatched media in my library**,
So that **I can quickly assess media quality and identify items that need manual TMDB matching**.

**Acceptance Criteria:**

1. **Given** a movie with `video_codec="H.265"`, `video_resolution="3840x2160"`, `audio_codec="DTS"`, `audio_channels=6`, `hdr_format="HDR10"`
   **When** the user views the movie detail page
   **Then** visual badges display: "H.265", "4K", "DTS 5.1", "HDR10"
   **And** badges use appropriate color coding (e.g., 4K = distinct color, HDR = distinct color)

2. **Given** a movie with only `video_codec` and `video_resolution` (no audio info)
   **When** the user views the detail page
   **Then** only video badges are shown (no empty/null badges displayed)

3. **Given** a movie with `subtitle_tracks` containing 3 tracks
   **When** the user views the detail page
   **Then** subtitle track information is displayed (language labels)
   **And** external vs embedded subtitles are visually distinguishable

4. **Given** a movie with no technical info (all fields NULL)
   **When** the user views the detail page
   **Then** no tech info section is rendered (graceful absence, not "No data" message)

5. **Given** the library page with mixed matched/unmatched media
   **When** the user selects the "Unmatched" filter
   **Then** only media with `tmdb_id IS NULL OR tmdb_id = 0` are displayed
   **And** the filter option shows a count badge (e.g., "Unmatched (3)")

6. **Given** the `GET /api/v1/movies?unmatched=true` endpoint
   **When** called
   **Then** returns only movies where tmdb_id is NULL or 0
   **And** response time is <300ms for 1,000 items (NFR-P6)

7. **Given** the `GET /api/v1/movies/stats` endpoint
   **When** called
   **Then** response includes `unmatched_count` field
   **And** the count is accurate (matches actual unmatched records)

8. **Given** the resolution value `"3840x2160"` from the API
   **When** the badge component renders
   **Then** it displays "4K" (human-friendly label)
   **And** `"1920x1080"` → "1080p", `"1280x720"` → "720p"

**Tasks:**
- 4.1: Create `TechBadge` component — renders individual badge with color variant (codec, resolution, audio, HDR)
- 4.2: Create `TechBadgeGroup` component — renders row of badges from media tech info, handles null/missing fields
- 4.3: Create resolution label mapping utility — `3840x2160` → `4K`, `1920x1080` → `1080p` etc.
- 4.4: Integrate `TechBadgeGroup` into movie detail page
- 4.5: Integrate `TechBadgeGroup` into series detail page
- 4.6: Add subtitle tracks display section to detail pages
- 4.7: Add `?unmatched=true` query parameter support to movie repository + handler
- 4.8: Add `GET /api/v1/movies/stats` endpoint with `unmatched_count`
- 4.9: Add series equivalent: `GET /api/v1/series?unmatched=true` + `GET /api/v1/series/stats`
- 4.10: Add "Unmatched" filter option to library page filter UI with count badge
- 4.11: Wire stats API call for unmatched count badge (TanStack Query)
- 4.12: Write TechBadge/TechBadgeGroup component tests
- 4.13: Write resolution mapping utility tests
- 4.14: Write unmatched filter API handler tests
- 4.15: Write stats endpoint handler tests
