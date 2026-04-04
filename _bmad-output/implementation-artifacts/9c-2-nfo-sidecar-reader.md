# Story 9c-2: NFO Sidecar Reader

Status: done

## Story

As a **NAS user migrating from Kodi or Jellyfin**,
I want **Vido to detect and read my existing .nfo sidecar files during scan**,
So that **my media gets precise TMDB matching using NFO uniqueid and I don't need to re-tag everything**.

## Acceptance Criteria

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

## Tasks / Subtasks

- [x] Task 1: Create NFO reader service (AC: #2, #3, #4, #5, #6)
  - [x] 1.1 Create `apps/api/internal/services/nfo_reader_service.go`
  - [x] 1.2 Define structs: `NFOReaderService`, `NFOData`, `NFOUniqueID`, `NFOStreamDetails`, `NFOSubtitle`
  - [x] 1.3 Implement `Parse(nfoPath string) (*NFOData, error)` — format detection (XML prefix / URL)
  - [x] 1.4 Implement `parseXML(content []byte) (*NFOData, error)` — Go `encoding/xml` for `<movie>`, `<tvshow>`, `<episodedetails>`
  - [x] 1.5 Implement `findNFOSidecar(videoPath string) string` — same-name .nfo path check

- [x] Task 2: Implement URL format extractors (AC: #4)
  - [x] 2.1 Implement `extractTMDbID(line string) (string, bool)` — parse TMDB URLs
  - [x] 2.2 Implement `extractIMDbID(line string) (string, bool)` — parse IMDB URLs/IDs

- [x] Task 3: Integrate into enrichment pipeline (AC: #1, #7, #8, #9)
  - [x] 3.1 Add `NFOReaderService` dependency to `EnrichmentService` struct and constructor
  - [x] 3.2 Modify `enrichMovie()` — insert NFO detection BEFORE Step 1 (Parse filename)
  - [x] 3.3 Call `ShouldOverwrite()` gate before applying NFO data
  - [x] 3.4 If NFO found and accepted: set `metadata_source = "nfo"`, skip AI parse
  - [x] 3.5 If NFO not found or rejected: continue existing AI parse flow unchanged

- [x] Task 4: Enhance TMDB enrichment with direct lookup (AC: #2, #3)
  - [x] 4.1 Modify TMDB enrichment to accept optional TMDB ID for direct `GetMovieDetails(id)` lookup
  - [x] 4.2 Modify TMDB enrichment to accept optional IMDB ID for `FindByExternalID(imdbID)` lookup
  - [x] 4.3 When NFO provides uniqueid, bypass search and use direct lookup

- [x] Task 5: Wire in main.go (AC: all)
  - [x] 5.1 Create `NFOReaderService` instance in `main.go`
  - [x] 5.2 Pass to `NewEnrichmentService()` constructor

- [x] Task 6: Write tests (AC: #1-9)
  - [x] 6.1 NFO parser unit tests: XML movie format, XML tvshow format, URL format (TMDB, IMDB)
  - [x] 6.2 NFO parser unit tests: malformed XML, empty file, unrecognized format
  - [x] 6.3 NFO parser unit tests: streamdetails extraction (video codec, resolution, audio)
  - [x] 6.4 `findNFOSidecar()` tests: exists, doesn't exist, various extensions
  - [x] 6.5 Enrichment integration tests: NFO priority over AI, manual priority over NFO
  - [x] 6.6 Enrichment integration tests: fallback on parse failure, no NFO file

## Dev Notes

### Architecture Compliance

- **Rule 4**: Handler → Service → Repository — NFO reader is a service, called from EnrichmentService
- **Rule 6**: File naming — `nfo_reader_service.go` (snake_case)
- **Rule 11**: No new interfaces needed — NFOReaderService is internal to enrichment, not exposed to handlers
- **Rule 13**: Error handling — NFO parse failure MUST log warning + fallback, never crash
- **Rule 2**: Logging — use `slog.Warn()` for NFO parse failures, `slog.Debug()` for NFO detection

### Project Structure Notes

- New file: `apps/api/internal/services/nfo_reader_service.go`
- Modified: `apps/api/internal/services/enrichment_service.go` — add NFO stage at TOP of `enrichMovie()`
- Modified: `apps/api/cmd/api/main.go` — wire NFOReaderService (line ~340, near enrichmentService creation)
- Existing: `apps/api/internal/services/enrichment_service.go` — `enrichMovie()` at line 204

### Critical Implementation Details

- **NFO detection inserts at TOP of `enrichMovie()`** — before Step 1 (Parse filename) at line 208
- **If developing in parallel with 9c-3**: 9c-2 modifies top of `enrichMovie()`, 9c-3 appends at bottom — minimal merge conflict
- **XML root elements**: `<movie>`, `<tvshow>`, `<episodedetails>` — each has different structure but same `uniqueid`/`fileinfo` children
- **Go `encoding/xml`**: Use permissive parsing — unknown fields silently ignored, missing fields left zero-value
- **URL pattern matching**: `themoviedb.org/movie/(\d+)`, `imdb.com/title/(tt\d+)`
- **ShouldOverwrite** from Story 9c-1 must be available — verify import path
- **No artwork reading**: Per ADR, skip `poster.jpg`/`fanart.jpg` references in NFO

### Kodi XML Structure Reference

```xml
<movie>
  <title>Movie Title</title>
  <originaltitle>Original Title</originaltitle>
  <year>2024</year>
  <plot>Plot description...</plot>
  <uniqueid type="tmdb" default="true">12345</uniqueid>
  <uniqueid type="imdb">tt1234567</uniqueid>
  <fileinfo>
    <streamdetails>
      <video><codec>h265</codec><width>3840</width><height>2160</height></video>
      <audio><codec>dts</codec><channels>6</channels></audio>
      <subtitle><language>chi</language></subtitle>
    </streamdetails>
  </fileinfo>
</movie>
```

### References

- [Source: architecture/adr-media-info-nfo-pipeline.md#Decision 4: NFO Parser]
- [Source: architecture/adr-media-info-nfo-pipeline.md#Decision 1: Scan Pipeline Extension]
- [Source: architecture/adr-media-info-nfo-pipeline.md#Decision 5: Data Source Priority]
- [Source: services/enrichment_service.go#enrichMovie() line 204]
- [Source: project-context.md#Rule 13: Error Handling Completeness]

## Dev Agent Record

### Agent Model Used
Claude Opus 4.6 (1M context)

### Debug Log References
- All 35 NFO-related tests pass (unit + integration)
- Full build clean (`go build ./...` — no errors)
- TMDB package tests pass, handler tests pass
- Pre-existing failures in setup_service_test.go and download_handler_test.go unrelated to this story

### Completion Notes List
- Created NFOReaderService with Parse(), parseXML(), parseURL(), FindNFOSidecar()
- Supports 3 XML root elements: `<movie>`, `<tvshow>`, `<episodedetails>`
- URL format supports both TMDB and IMDB URLs
- Streamdetails extraction: video codec, resolution (4K/1440p/1080p/720p/480p), audio codec, channels, subtitle tracks
- Integrated NFO detection at TOP of enrichMovie() — before AI parse (Step 0)
- ShouldOverwrite() gate: manual > nfo > tmdb > ai
- Added FindByExternalID to TMDB client + service for IMDB→TMDB lookup
- Direct TMDB ID lookup when NFO provides `<uniqueid type="tmdb">`
- IMDB ID → `/find/{external_id}` → GetMovieDetails chain for `<uniqueid type="imdb">`
- Malformed NFO logs warning and falls back to AI parse (no crash)
- 🎨 UX Verification: SKIPPED — no UI changes in this story

### File List
- `apps/api/internal/services/nfo_reader_service.go` — NEW: NFO reader service
- `apps/api/internal/services/nfo_reader_service_test.go` — NEW: 28 unit tests
- `apps/api/internal/services/enrichment_nfo_test.go` — NEW: 7 integration tests
- `apps/api/internal/services/enrichment_service.go` — MODIFIED: NFO stage in enrichMovie(), tryNFOEnrichment(), applyTMDbMovieDetails(), enrichFromIMDbID()
- `apps/api/internal/services/tmdb_service.go` — MODIFIED: FindByExternalID method + interface
- `apps/api/internal/tmdb/client.go` — MODIFIED: FindByExternalID in ClientInterface
- `apps/api/internal/tmdb/movies.go` — MODIFIED: FindByExternalID implementation
- `apps/api/internal/tmdb/types.go` — MODIFIED: FindByExternalIDResponse type
- `apps/api/internal/tmdb/fallback_test.go` — MODIFIED: MockClient FindByExternalID
- `apps/api/internal/handlers/tmdb_handler_test.go` — MODIFIED: MockTMDbService FindByExternalID
- `apps/api/cmd/api/main.go` — MODIFIED: wire NFOReaderService + tmdbService into enrichment
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — MODIFIED: 9c-2 status

## Senior Developer Review (AI)

**Date:** 2026-04-04
**Reviewer:** Amelia (Dev Agent CR workflow)
**Outcome:** Changes Requested → All Fixed

### Action Items (7 total, 7 resolved)

- [x] [H1] `resolveResolution()` OR→AND logic fix — misclassified non-standard aspect ratios
- [x] [H2] `FindByExternalID` nil-client guard — prevented panic via `NewTMDbServiceWithCacheService`
- [x] [H3] NFO file size limit (1MB cap) — prevented OOM on malicious/corrupted files
- [x] [M1] `applyTMDbMovieDetails` TMDbID — use `models.NewNullInt64()` for consistency
- [x] [M2] NFO subtitle tracks persisted to `movie.SubtitleTracks` — was dead data
- [x] [M3] IMDB ID from NFO persisted before TMDB lookup — prevented losing known ID
- [x] [L1] `resolveResolution` returns `""` for 0×0 — defensive guard

## Change Log
- 2026-04-04: Story 9c-2 implemented — NFO sidecar reader service with XML/URL parsing, TMDB direct lookup via uniqueid, streamdetails tech info extraction, ShouldOverwrite priority gate, integrated into enrichment pipeline. 35 tests added.
- 2026-04-04: CR fixes — 3 High + 3 Medium + 1 Low issues resolved. resolveResolution AND logic, nil-client guard, 1MB file limit, subtitle track persistence, IMDB ID preservation.
