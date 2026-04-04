# Story 9c-3: FFprobe Integration

Status: review

## Story

As a **NAS user**,
I want **Vido to extract technical details (codec, resolution, audio format, HDR) from my video files**,
So that **I can see what quality my media files are in and make informed decisions about my collection**.

## Acceptance Criteria

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

## Tasks / Subtasks

- [x] Task 1: Update Dockerfile (AC: #1)
  - [x] 1.1 Add `RUN apk add --no-cache ffmpeg` to runtime stage in `Dockerfile` (root, the unified image)
  - [x] 1.2 Also update `apps/api/Dockerfile` if still used for dev

- [x] Task 2: Create FFprobe service (AC: #2, #3, #4, #5, #6)
  - [x] 2.1 Create `apps/api/internal/services/ffprobe_service.go`
  - [x] 2.2 Define `FFprobeService` struct with semaphore channel, timeout, available bool
  - [x] 2.3 Implement `NewFFprobeService(maxConcurrent int, timeout time.Duration)` — `exec.LookPath` check
  - [x] 2.4 Implement `Probe(ctx context.Context, filePath string) (*MediaTechInfo, error)` — semaphore + exec + JSON parse
  - [x] 2.5 Define `MediaTechInfo` struct: VideoCodec, VideoResolution, AudioCodec, AudioChannels, HDRFormat, SubtitleTracks
  - [x] 2.6 Define `ErrFFprobeNotAvailable` sentinel error

- [x] Task 3: Implement FFprobe JSON parser (AC: #4)
  - [x] 3.1 Implement `parseFfprobeJSON(output []byte) (*MediaTechInfo, error)`
  - [x] 3.2 Extract video stream: codec_name → normalized (h264→H.264, hevc→H.265, av1→AV1)
  - [x] 3.3 Extract video stream: width × height → "3840x2160"
  - [x] 3.4 Extract audio stream: codec_name → normalized (dts→DTS, aac→AAC, truehd→TrueHD)
  - [x] 3.5 Extract audio stream: channels count
  - [x] 3.6 Implement HDR detection: color_transfer (smpte2084=HDR10, arib-std-b67=HLG) + side_data (DolbyVision)

- [x] Task 4: Implement external subtitle detection (AC: #9)
  - [x] 4.1 Scan filesystem for sidecar files: `.srt`, `.ass`, `.ssa`, `.sub` with language tags
  - [x] 4.2 Merge embedded tracks (from FFprobe) + external tracks into `subtitle_tracks` JSON array

- [x] Task 5: Integrate into enrichment pipeline (AC: #7, #10)
  - [x] 5.1 Add `FFprobeService` dependency to `EnrichmentService`
  - [x] 5.2 Add FFprobe stage AFTER TMDB enrichment in `enrichMovie()` (append at bottom)
  - [x] 5.3 Skip if `movie.VideoCodec` already set (NFO streamdetails from 9c-2)
  - [x] 5.4 Apply `MediaTechInfo` to movie model fields

- [x] Task 6: Implement series file_size aggregation (AC: #8)
  - [x] 6.1 In scanner service, after scanning all files for a series, sum file sizes
  - [x] 6.2 Update `series.file_size` with total bytes
  - [x] 6.3 Recalculate on each scan (not additive)

- [x] Task 7: Wire in main.go (AC: all)
  - [x] 7.1 Create `FFprobeService` instance with maxConcurrent=3, timeout=10s
  - [x] 7.2 Pass to `NewEnrichmentService()` constructor

- [x] Task 8: Write tests (AC: #1-10)
  - [x] 8.1 FFprobe JSON parsing tests: H.265/4K/DTS sample, H.264/1080p/AAC sample
  - [x] 8.2 HDR detection tests: HDR10, DolbyVision, SDR (no HDR)
  - [x] 8.3 Semaphore tests: verify concurrency limiting
  - [x] 8.4 Graceful degradation test: ffprobe not found → ErrFFprobeNotAvailable
  - [x] 8.5 Timeout test: context cancellation after deadline
  - [x] 8.6 Enrichment integration: skip when VideoCodec already set
  - [x] 8.7 Series file_size calculation tests
  - [x] 8.8 External subtitle detection tests

## Dev Notes

### Architecture Compliance

- **Rule 4**: Handler → Service → Repository — FFprobeService is a service, called from EnrichmentService
- **Rule 6**: File naming — `ffprobe_service.go`
- **Rule 13**: Error handling — FFprobe failure MUST NOT crash scan, log + skip
- **Rule 14**: Resource lifecycle — semaphore prevents CPU/IO saturation; context honored for cancellation
- **Rule 15**: Wire FFprobeService in `main.go`

### Project Structure Notes

- New file: `apps/api/internal/services/ffprobe_service.go`
- Modified: `apps/api/internal/services/enrichment_service.go` — add FFprobe stage at BOTTOM of `enrichMovie()`
- Modified: `apps/api/cmd/api/main.go` — wire FFprobeService (near enrichmentService, line ~340)
- Modified: `Dockerfile` (root) — add `apk add --no-cache ffmpeg` to runtime stage (line ~66, `FROM alpine:3.21`)
- Modified: Scanner service — series file_size aggregation

### Critical Implementation Details

- **FFprobe appends at BOTTOM of `enrichMovie()`** — after Step 5 (Update DB) or as new Step 6
- **If developing in parallel with 9c-2**: 9c-3 modifies bottom, 9c-2 modifies top — minimal conflict
- **`os/exec` only** — no third-party FFprobe Go library (per ADR)
- **FFprobe command**: `ffprobe -v quiet -print_format json -show_streams -show_format {filePath}`
- **Codec normalization**: `hevc`→`H.265`, `h264`→`H.264`, `av1`→`AV1`, `dts`→`DTS`, `aac`→`AAC`, `truehd`→`TrueHD Atmos`
- **HDR detection**: Check `color_transfer` field: `smpte2084` = HDR10, `arib-std-b67` = HLG. Check `side_data_list` for DolbyVision RPU
- **subtitle_tracks JSON**: `[{"language":"zh-Hant","format":"srt","external":true}]` — IETF BCP 47 language tags
- **Series file_size**: Sum ALL video file sizes matching the series (by library_id + series record). Use `os.Stat().Size()` during scan
- **Dockerfile**: Root `Dockerfile` has runtime at line 66 `FROM alpine:3.21` — add ffmpeg there. `apps/api/Dockerfile` also has `FROM alpine:3.21` at line 42

### FFprobe JSON Output Structure Reference

```json
{
  "streams": [
    {"codec_type": "video", "codec_name": "hevc", "width": 3840, "height": 2160, "color_transfer": "smpte2084"},
    {"codec_type": "audio", "codec_name": "dts", "channels": 6},
    {"codec_type": "subtitle", "codec_name": "subrip", "tags": {"language": "eng"}}
  ],
  "format": {"filename": "...", "size": "12345678"}
}
```

### References

- [Source: architecture/adr-media-info-nfo-pipeline.md#Decision 3: FFprobe Integration]
- [Source: architecture/adr-media-info-nfo-pipeline.md#Decision 1: Scan Pipeline Extension]
- [Source: project-context.md#Rule 14: Resource Lifecycle Management]
- [Source: Dockerfile — runtime stage at line 66]
- [Source: services/enrichment_service.go#enrichMovie() line 204]

## Dev Agent Record

### Agent Model Used
Claude Opus 4.6 (1M context)

### Debug Log References
- 63 tests passing (FFprobe + NFO + enrichment)
- Full build clean (`go build ./...`)
- TMDB + handler tests pass, no regressions

### Completion Notes List
- FFprobeService: semaphore concurrency control, per-call timeout, exec.LookPath availability check
- JSON parser: codec normalization (H.264/H.265/AV1/DTS/AAC/TrueHD etc.), HDR detection (HDR10/HLG/Dolby Vision)
- External subtitle detection: scans for .srt/.ass/.ssa/.sub sidecar files, extracts language tags
- Enrichment pipeline: FFprobe stage appended after TMDB enrichment, skips if VideoCodec already set (NFO)
- Series file_size aggregation: post-scan step sums episode file sizes per series
- Dockerfile: ffmpeg added to both root and apps/api runtime stages
- 🎨 UX Verification: SKIPPED — no UI changes in this story

### File List
- `apps/api/internal/services/ffprobe_service.go` — NEW: FFprobe service + JSON parser + external subtitle detection
- `apps/api/internal/services/ffprobe_service_test.go` — NEW: 22 unit tests
- `apps/api/internal/services/enrichment_service.go` — MODIFIED: FFprobeService dependency, tryFFprobeEnrichment()
- `apps/api/internal/services/enrichment_nfo_test.go` — MODIFIED: updated NewEnrichmentService calls for new parameter
- `apps/api/internal/services/scanner_service.go` — MODIFIED: episodeRepo, aggregateSeriesFileSizes()
- `apps/api/cmd/api/main.go` — MODIFIED: wire FFprobeService + episodeRepo
- `Dockerfile` — MODIFIED: add ffmpeg to runtime stage
- `apps/api/Dockerfile` — MODIFIED: add ffmpeg to runtime stage
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — MODIFIED: 9c-3 status

## Change Log
- 2026-04-04: Story 9c-3 implemented — FFprobe service with semaphore/timeout, JSON parser with codec normalization + HDR detection, external subtitle detection, enrichment pipeline integration, series file_size aggregation, Docker ffmpeg. 22 tests added.
