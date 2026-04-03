# ADR: Media Technical Info, NFO Sidecar & Data Source Priority

> **Status:** ACCEPTED
> **Date:** 2026-04-03
> **Deciders:** Alexyu (product owner), Winston (architect)
> **Related PRD:** P1-030 (Media Technical Info), P1-031 (NFO Sidecar), P1-032 (Data Source Priority), P1-033 (Series File Size), P2-030 (Unmatched Media Filter)

---

## Context

Competitive gap analysis against Plex, Jellyfin, Emby, Infuse, and Kodi reveals that Vido lacks three standard features:

1. **Media technical information** (codec, resolution, audio, HDR) — displayed as visual badges
2. **NFO sidecar reading** — enables zero-friction migration from Kodi/Jellyfin
3. **Explicit data source priority** — prevents automated enrichment from overwriting user corrections

The existing scan pipeline (`scanner_service.go`) handles file discovery and DB insertion. Post-scan enrichment (`enrichment_service.go`) handles AI parsing → metadata search → apply. These new features extend both stages.

---

## Decision 1: Scan Pipeline Extension

### Current Pipeline

```
File Discovery → DB Insert (parse_status=pending)
    ↓ (post-scan enrichment)
AI Parse → Metadata Search (TMDB/Douban/Wikipedia) → Apply to DB
```

### Extended Pipeline

```
File Discovery → DB Insert (parse_status=pending)
    ↓ (post-scan enrichment, per file)
1. NFO Detection
   ├─ NFO found → NFO Parse (title, year, uniqueid, streamdetails)
   │              metadata_source = "nfo"
   └─ No NFO   → AI Parse (existing flow)
                   metadata_source = "ai"
    ↓
2. TMDB Enrichment
   - If NFO provided uniqueid (tmdb/imdb) → direct lookup (skip search)
   - Otherwise → search by title + year (existing flow)
    ↓
3. FFprobe Technical Info Extraction
   - Skip if NFO already provided streamdetails
   - Independent of metadata — only reads file container
```

### Design Rationale

- **NFO before AI Parse:** NFO provides exact identifiers (TMDB ID, IMDB ID), eliminating guesswork. Skipping AI parse when NFO exists saves API costs and is more accurate
- **FFprobe last:** Technical info extraction is independent of metadata resolution. Runs after enrichment to avoid blocking the metadata flow. If NFO has `streamdetails`, FFprobe is skipped entirely (saves exec overhead)
- **Serial execution:** SQLite WAL mode supports concurrent reads but single writer. Pipeline stages are serial per file to avoid write contention. Can be parallelized later if needed
- **NFO parse failure → fallback:** If NFO XML is malformed, log warning and fall back to AI parse. The file still gets processed

### Implementation Location

Extend `enrichment_service.go` `StartEnrichment()` method:

```go
func (s *EnrichmentService) enrichSingleItem(ctx context.Context, movie *Movie) error {
    nfoPath := findNFOSidecar(movie.FilePath)

    // Stage 1: NFO or AI Parse
    if nfoPath != "" {
        nfoData, err := s.nfoReader.Parse(nfoPath)
        if err != nil {
            slog.Warn("NFO parse failed, falling back to AI", "path", nfoPath, "error", err)
            // fallthrough to AI parse
        } else {
            applyNFOData(movie, nfoData)
            movie.MetadataSource = MetadataSourceNFO
        }
    }
    if movie.MetadataSource != MetadataSourceNFO {
        // Existing AI parse flow
        s.parseAndEnrich(ctx, movie)
    }

    // Stage 2: TMDB Enrichment (existing, enhanced with NFO uniqueid)
    s.tmdbEnrich(ctx, movie)

    // Stage 3: FFprobe (skip if NFO had streamdetails)
    if movie.VideoCodec == "" {
        s.extractTechInfo(ctx, movie)
    }

    return s.movieRepo.Update(ctx, movie)
}
```

---

## Decision 2: Database Schema

### Approach: Extend Existing Tables

Technical info fields added directly to `movies` and `series` tables (not a separate table).

**Rationale:**
- 1:1 relationship with media records — a JOIN table adds complexity with no benefit
- Consistent with existing pattern (`credits`, `production_countries` are JSON columns on the same table)
- SQLite `ALTER TABLE ADD COLUMN` is O(1) — no table rewrite

### Migration 021

```sql
-- Movies: technical info
ALTER TABLE movies ADD COLUMN video_codec TEXT;
ALTER TABLE movies ADD COLUMN video_resolution TEXT;
ALTER TABLE movies ADD COLUMN audio_codec TEXT;
ALTER TABLE movies ADD COLUMN audio_channels INTEGER;
ALTER TABLE movies ADD COLUMN subtitle_tracks TEXT;
ALTER TABLE movies ADD COLUMN hdr_format TEXT;

-- Series: file size + technical info (representative, from first episode)
ALTER TABLE series ADD COLUMN file_size INTEGER;
ALTER TABLE series ADD COLUMN video_codec TEXT;
ALTER TABLE series ADD COLUMN video_resolution TEXT;
ALTER TABLE series ADD COLUMN audio_codec TEXT;
ALTER TABLE series ADD COLUMN audio_channels INTEGER;
ALTER TABLE series ADD COLUMN subtitle_tracks TEXT;
ALTER TABLE series ADD COLUMN hdr_format TEXT;
```

### Field Semantics

| Field | Type | Example Values | Source |
|-------|------|---------------|--------|
| `video_codec` | TEXT | `"H.264"`, `"H.265"`, `"AV1"` | NFO streamdetails or FFprobe |
| `video_resolution` | TEXT | `"3840x2160"`, `"1920x1080"` | NFO streamdetails or FFprobe |
| `audio_codec` | TEXT | `"DTS"`, `"AAC"`, `"TrueHD Atmos"` | NFO streamdetails or FFprobe |
| `audio_channels` | INTEGER | `2`, `6`, `8` (stereo, 5.1, 7.1) | NFO streamdetails or FFprobe |
| `subtitle_tracks` | TEXT | JSON array (see below) | NFO streamdetails or FFprobe + filesystem |
| `hdr_format` | TEXT | `"HDR10"`, `"DolbyVision"`, `null` | FFprobe color metadata |
| `file_size` (series) | INTEGER | bytes | Sum of all episode files in scan |

### subtitle_tracks JSON Schema

```json
[
  { "language": "zh-Hant", "format": "srt", "external": true },
  { "language": "eng", "format": "ass", "external": false },
  { "language": "jpn", "format": "srt", "external": true }
]
```

- `external: true` — sidecar subtitle file detected on filesystem
- `external: false` — embedded track detected by FFprobe
- `language` — IETF BCP 47 tag (consistent with existing subtitle extension convention)
- `format` — `"srt"`, `"ass"`, `"ssa"`, `"pgs"`, `"vobsub"`

### Series File Size Calculation

During scan, accumulate file sizes for all video files belonging to a series (matched by `library_id` + series record). Store the sum in `series.file_size`. Recalculated on each scan.

---

## Decision 3: FFprobe Integration

### Docker Packaging

```dockerfile
# In runtime stage (alpine:3.21)
RUN apk add --no-cache ffmpeg
```

`ffmpeg` package on Alpine includes `ffprobe`. Adds ~30MB to image size. Acceptable trade-off for a feature all competing tools have.

### Go Integration

Direct `os/exec` — no third-party library.

```go
type FFprobeService struct {
    semaphore chan struct{} // concurrency limiter
    timeout   time.Duration
}

func NewFFprobeService(maxConcurrent int, timeout time.Duration) *FFprobeService {
    return &FFprobeService{
        semaphore: make(chan struct{}, maxConcurrent),
        timeout:   timeout,
    }
}

func (s *FFprobeService) Probe(ctx context.Context, filePath string) (*MediaTechInfo, error) {
    // Acquire semaphore
    select {
    case s.semaphore <- struct{}{}:
        defer func() { <-s.semaphore }()
    case <-ctx.Done():
        return nil, ctx.Err()
    }

    ctx, cancel := context.WithTimeout(ctx, s.timeout)
    defer cancel()

    cmd := exec.CommandContext(ctx, "ffprobe",
        "-v", "quiet",
        "-print_format", "json",
        "-show_streams",
        "-show_format",
        filePath,
    )

    output, err := cmd.Output()
    if err != nil {
        return nil, fmt.Errorf("ffprobe exec: %w", err)
    }

    return parseFfprobeJSON(output)
}
```

### Configuration

| Parameter | Default | Rationale |
|-----------|---------|-----------|
| `maxConcurrent` | 3 | Prevent CPU/IO saturation on NAS hardware |
| `timeout` | 10s | Network-mounted files may be slow |

### Graceful Degradation

On startup, check if `ffprobe` binary exists (`exec.LookPath("ffprobe")`). If not found:
- Log warning: `"ffprobe not found — technical info extraction disabled"`
- `FFprobeService.Probe()` returns `nil, ErrFFprobeNotAvailable`
- Pipeline skips technical info extraction (no error, no crash)

---

## Decision 4: NFO Parser

### Service Interface

```go
type NFOReaderService struct{}

type NFOData struct {
    Title         string
    OriginalTitle string
    Year          string
    Plot          string
    UniqueIDs     []NFOUniqueID
    StreamDetails *NFOStreamDetails
    SourceFormat  string // "xml" | "url"
}

type NFOUniqueID struct {
    Type  string // "tmdb" | "imdb"
    Value string // "12345" | "tt1234567"
}

type NFOStreamDetails struct {
    VideoCodec      string
    VideoResolution string // "{width}x{height}"
    AudioCodec      string
    AudioChannels   int
    Subtitles       []NFOSubtitle
}

type NFOSubtitle struct {
    Language string
}
```

### Format Detection

```go
func (s *NFOReaderService) Parse(nfoPath string) (*NFOData, error) {
    content, err := os.ReadFile(nfoPath)
    if err != nil {
        return nil, fmt.Errorf("read nfo: %w", err)
    }

    trimmed := bytes.TrimSpace(content)

    // Format 1: XML (starts with <?xml or <movie or <tvshow or <episodedetails)
    if bytes.HasPrefix(trimmed, []byte("<?xml")) ||
       bytes.HasPrefix(trimmed, []byte("<movie")) ||
       bytes.HasPrefix(trimmed, []byte("<tvshow")) ||
       bytes.HasPrefix(trimmed, []byte("<episodedetails")) {
        return s.parseXML(content)
    }

    // Format 2: Single-line URL
    line := strings.TrimSpace(string(trimmed))
    if id, ok := extractTMDbID(line); ok {
        return &NFOData{UniqueIDs: []NFOUniqueID{{Type: "tmdb", Value: id}}, SourceFormat: "url"}, nil
    }
    if id, ok := extractIMDbID(line); ok {
        return &NFOData{UniqueIDs: []NFOUniqueID{{Type: "imdb", Value: id}}, SourceFormat: "url"}, nil
    }

    return nil, fmt.Errorf("unrecognized NFO format")
}
```

### NFO File Discovery

```go
func findNFOSidecar(videoPath string) string {
    // /media/movies/Movie.2024.mkv → /media/movies/Movie.2024.nfo
    nfoPath := strings.TrimSuffix(videoPath, filepath.Ext(videoPath)) + ".nfo"
    if _, err := os.Stat(nfoPath); err == nil {
        return nfoPath
    }
    return ""
}
```

### Artwork Paths

**Not reading artwork from NFO.** Vido uses TMDB poster/backdrop URLs. Reading local `poster.jpg` / `fanart.jpg` referenced in NFO adds filesystem complexity (relative path resolution, missing files) for marginal value. Can be added later as a backward-compatible extension if requested.

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
      <video>
        <codec>h265</codec>
        <width>3840</width>
        <height>2160</height>
      </video>
      <audio>
        <codec>dts</codec>
        <channels>6</channels>
      </audio>
      <subtitle>
        <language>chi</language>
      </subtitle>
    </streamdetails>
  </fileinfo>
</movie>
```

---

## Decision 5: Data Source Priority

### Priority Chain

```
manual (100) > nfo (80) > tmdb (60) > douban (50) > wikipedia (40) > ai (20)
```

### Implementation

```go
// In models/types.go
const (
    MetadataSourceManual    MetadataSource = "manual"
    MetadataSourceNFO       MetadataSource = "nfo"
    MetadataSourceTMDb      MetadataSource = "tmdb"
    MetadataSourceDouban    MetadataSource = "douban"
    MetadataSourceWikipedia MetadataSource = "wikipedia"
    MetadataSourceAI        MetadataSource = "ai"
)

var metadataSourcePriority = map[MetadataSource]int{
    MetadataSourceManual:    100,
    MetadataSourceNFO:       80,
    MetadataSourceTMDb:      60,
    MetadataSourceDouban:    50,
    MetadataSourceWikipedia: 40,
    MetadataSourceAI:        20,
}

// ShouldOverwrite returns true if incoming source may overwrite current source.
func ShouldOverwrite(current, incoming MetadataSource) bool {
    if current == "" {
        return true // No existing source — always accept
    }
    return metadataSourcePriority[incoming] >= metadataSourcePriority[current]
}
```

### Enrichment Behavior

| Scenario | Current Source | Incoming Source | Result |
|----------|--------------|----------------|--------|
| First scan (new file) | `""` (empty) | `"ai"` | Accept — first data |
| Re-scan, NFO added | `"ai"` | `"nfo"` | Accept — NFO > AI |
| Re-scan, TMDB refresh | `"nfo"` | `"tmdb"` | **Reject** — NFO > TMDB |
| User manual edit | `"tmdb"` | `"manual"` | Accept — manual > all |
| Re-scan after manual edit | `"manual"` | `"nfo"` | **Reject** — manual > NFO |

### Integration Points

1. **EnrichmentService:** Call `ShouldOverwrite()` before applying metadata from any source
2. **MetadataEditService.UpdateMetadata():** Always set `metadata_source = "manual"` (existing behavior, unchanged)
3. **MetadataService.ApplyMetadata():** Pass source through, check priority before overwrite
4. **ScannerService:** File attribute updates (`file_size`, `is_removed`) are **not** gated by priority — they're filesystem facts, not metadata

---

## API Changes

### New Endpoint: Media Technical Info

No separate endpoint needed. Technical info fields are included in existing `GET /api/v1/movies/:id` and `GET /api/v1/series/:id` responses:

```json
{
  "id": "uuid",
  "title": "Movie Title",
  "video_codec": "H.265",
  "video_resolution": "3840x2160",
  "audio_codec": "DTS",
  "audio_channels": 6,
  "hdr_format": "HDR10",
  "subtitle_tracks": [
    { "language": "zh-Hant", "format": "srt", "external": true }
  ],
  "metadata_source": "nfo",
  ...existing fields...
}
```

### Unmatched Media Filter (P2-030)

Extend existing `GET /api/v1/movies` with query parameter:

```
GET /api/v1/movies?unmatched=true
```

Backend filter: `WHERE tmdb_id IS NULL OR tmdb_id = 0`

Count endpoint for badge:

```
GET /api/v1/movies/stats
```

Response includes `unmatched_count` field (already a natural extension of the stats pattern).

---

## New Service Files

| File | Package | Purpose |
|------|---------|---------|
| `internal/services/nfo_reader_service.go` | `services` | NFO file detection, XML/URL parsing |
| `internal/services/ffprobe_service.go` | `services` | FFprobe exec wrapper with semaphore |
| `internal/database/migrations/021_media_tech_info.go` | `migrations` | Schema migration |

No new packages — follows existing flat `services` structure per architecture pattern 2.2.

---

## Consequences

### Positive

- Users migrating from Kodi/Jellyfin can drop in their existing NFO files for instant metadata recognition
- Technical info badges match feature parity with Plex/Jellyfin/Infuse
- Data source priority prevents automated re-scans from destroying user corrections
- FFprobe graceful degradation means non-Docker users still get a functional app

### Negative

- Docker image size increases ~30MB (ffmpeg package)
- Series technical info is representative (first file) — not per-episode until episodes table exists
- NFO read-only means Vido users can't generate NFO for other tools (acceptable per PRD)

### Risks

| Risk | Severity | Mitigation |
|------|----------|------------|
| FFprobe slow on network mounts | Medium | 10s timeout + 3-concurrency semaphore |
| NFO XML format variations | Low | Minimal struct — only parse known fields, ignore unknown |
| SQLite write contention during large scan | Medium | Serial pipeline per file; batch FFprobe later if needed |

---

## Alternatives Considered

1. **Separate `media_info` table:** Rejected — 1:1 relationship doesn't justify JOIN overhead. SQLite ALTER TABLE ADD COLUMN is zero-cost
2. **Third-party FFprobe Go library (floostack/transcoder):** Rejected — adds dependency for trivial `exec.Command` + JSON unmarshal. Direct exec is more maintainable and debuggable
3. **Read NFO artwork paths:** Rejected — TMDB already provides poster/backdrop. Local artwork support can be added later without schema changes
4. **Parallel FFprobe + enrichment:** Rejected for now — SQLite single-writer constraint. Can be revisited with a write queue if scan performance becomes an issue
5. **metadata_source as enum column:** Rejected — TEXT with application-level constants is more flexible for adding new sources without migration
