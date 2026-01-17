# Story 2.5: Standard Filename Parser (Regex-based)

Status: done

## Story

As a **media collector**,
I want the **system to parse standard naming convention filenames**,
So that **most of my files are automatically identified without AI**.

## Acceptance Criteria

1. **Given** a file with standard naming like `Movie.Name.2024.1080p.BluRay.mkv`
   **When** the parser processes the filename
   **Then** it extracts:
   - Title: "Movie Name"
   - Year: 2024
   - Quality: 1080p
   - Source: BluRay

2. **Given** a TV show file like `Show.Name.S01E05.720p.WEB-DL.mkv`
   **When** the parser processes the filename
   **Then** it extracts:
   - Title: "Show Name"
   - Season: 1
   - Episode: 5
   - Quality: 720p
   - Source: WEB-DL

3. **Given** parsing completes
   **When** measuring performance
   **Then** standard regex parsing completes within 100ms per file (NFR-P13)

4. **Given** the filename cannot be parsed by regex
   **When** parsing fails
   **Then** the file is flagged for AI parsing (Epic 3)
   **And** a clear status indicator shows "Pending AI parsing"

5. **Given** various filename formats
   **When** processing files
   **Then** the parser handles:
   - Dot-separated names (`Movie.Name.2024`)
   - Space-separated names (`Movie Name 2024`)
   - Underscore-separated names (`Movie_Name_2024`)
   - Mixed separators

## Tasks / Subtasks

### Task 1: Create Parser Package Structure (AC: #1, #2)
- [x] 1.1 Create `apps/api/internal/parser/` package
- [x] 1.2 Define `ParseResult` struct with all metadata fields
- [x] 1.3 Define `ParseStatus` enum (Success, NeedsAI, Failed)
- [x] 1.4 Create `ParserInterface` for future AI parser integration

### Task 2: Implement Movie Filename Parser (AC: #1, #5)
- [x] 2.1 Create regex pattern for standard movie naming
- [x] 2.2 Extract title (handle dots, spaces, underscores)
- [x] 2.3 Extract year (4-digit number, typically 1900-2099)
- [x] 2.4 Extract quality (480p, 720p, 1080p, 2160p/4K)
- [x] 2.5 Extract source (BluRay, WEB-DL, HDTV, DVDRip, etc.)
- [x] 2.6 Extract codec (x264, x265/HEVC, AV1, etc.)
- [x] 2.7 Handle release group tags (e.g., `-SPARKS`, `-YTS`)

### Task 3: Implement TV Show Filename Parser (AC: #2, #5)
- [x] 3.1 Create regex pattern for TV show naming (S01E05 format)
- [x] 3.2 Support alternative formats (1x05, Season 1 Episode 5)
- [x] 3.3 Extract season number
- [x] 3.4 Extract episode number (single or range: E01-E03)
- [x] 3.5 Handle daily shows (2024.01.15 format)
- [x] 3.6 Handle anime episode numbering (Episode 01, Ep01, 01)

### Task 4: Implement Quality/Source Detection (AC: #1, #2)
- [x] 4.1 Create quality detector with all common resolutions
- [x] 4.2 Create source detector (BluRay, WEB-DL, HDTV, etc.)
- [x] 4.3 Create codec detector (x264, x265, HEVC, AV1)
- [x] 4.4 Create audio codec detector (AAC, DTS, Atmos, etc.)
- [x] 4.5 Normalize quality values to standard format

### Task 5: Implement Title Cleaner (AC: #1, #2, #5)
- [x] 5.1 Remove release group tags
- [x] 5.2 Replace dots/underscores with spaces
- [x] 5.3 Remove quality/source/codec from title
- [x] 5.4 Handle edge cases (titles with years, e.g., "2001 A Space Odyssey")
- [x] 5.5 Trim and normalize whitespace

### Task 6: Create Parser Service Layer (AC: #3, #4)
- [x] 6.1 Create `apps/api/internal/services/parser_service.go`
- [x] 6.2 Implement `ParseFilename(filename string) (*ParseResult, error)`
- [x] 6.3 Try movie pattern first, then TV show pattern
- [x] 6.4 Return appropriate status (Success, NeedsAI, Failed)
- [x] 6.5 Log parsing results with slog

### Task 7: Create Parser Handler (AC: #1, #2, #4)
- [x] 7.1 Create `apps/api/internal/handlers/parser_handler.go`
- [x] 7.2 Implement `POST /api/v1/parser/parse` endpoint
- [x] 7.3 Implement `POST /api/v1/parser/parse-batch` for multiple files
- [x] 7.4 Return structured response with parse status

### Task 8: Write Comprehensive Tests (AC: #1, #2, #3, #5)
- [x] 8.1 Create test fixtures with 50+ real-world filename examples
- [x] 8.2 Test movie parsing with various formats
- [x] 8.3 Test TV show parsing with various formats
- [x] 8.4 Test edge cases (unusual characters, long names)
- [x] 8.5 Benchmark test to verify <100ms performance
- [x] 8.6 Test title cleaning edge cases

## Dev Notes

### CRITICAL: Backend Location

**ALL CODE MUST GO TO `/apps/api/internal/parser/`**

### File Locations

| Component | Path |
|-----------|------|
| Parser Package | `apps/api/internal/parser/` |
| Parse Result Types | `apps/api/internal/parser/types.go` |
| Movie Parser | `apps/api/internal/parser/movie_parser.go` |
| TV Parser | `apps/api/internal/parser/tv_parser.go` |
| Quality Detector | `apps/api/internal/parser/quality.go` |
| Title Cleaner | `apps/api/internal/parser/cleaner.go` |
| Parser Service | `apps/api/internal/services/parser_service.go` |
| Parser Handler | `apps/api/internal/handlers/parser_handler.go` |

### ParseResult Struct

```go
// parser/types.go

package parser

type ParseStatus string

const (
    ParseStatusSuccess ParseStatus = "success"
    ParseStatusNeedsAI ParseStatus = "needs_ai"
    ParseStatusFailed  ParseStatus = "failed"
)

type MediaType string

const (
    MediaTypeMovie   MediaType = "movie"
    MediaTypeTVShow  MediaType = "tv"
    MediaTypeUnknown MediaType = "unknown"
)

type ParseResult struct {
    // Original filename
    OriginalFilename string `json:"original_filename"`

    // Parse status
    Status    ParseStatus `json:"status"`
    MediaType MediaType   `json:"media_type"`

    // Extracted metadata
    Title         string `json:"title"`
    CleanedTitle  string `json:"cleaned_title"`  // For TMDb search
    Year          int    `json:"year,omitempty"`

    // TV Show specific
    Season        int    `json:"season,omitempty"`
    Episode       int    `json:"episode,omitempty"`
    EpisodeEnd    int    `json:"episode_end,omitempty"` // For ranges

    // Quality info
    Quality       string `json:"quality,omitempty"`      // 1080p, 720p, etc.
    Source        string `json:"source,omitempty"`       // BluRay, WEB-DL, etc.
    VideoCodec    string `json:"video_codec,omitempty"`  // x264, x265, etc.
    AudioCodec    string `json:"audio_codec,omitempty"`  // AAC, DTS, etc.

    // Release info
    ReleaseGroup  string `json:"release_group,omitempty"`

    // Confidence score (0-100)
    Confidence    int    `json:"confidence"`

    // Error message if failed
    ErrorMessage  string `json:"error_message,omitempty"`
}
```

### Regex Patterns

```go
// parser/patterns.go

package parser

import "regexp"

// Movie patterns
var (
    // Standard: Movie.Name.2024.1080p.BluRay.x264-GROUP.mkv
    MoviePattern = regexp.MustCompile(
        `^(?P<title>.+?)` +                    // Title (non-greedy)
        `[.\s_-]+` +                           // Separator
        `(?P<year>(?:19|20)\d{2})` +           // Year
        `(?:[.\s_-]+(?P<quality>\d{3,4}p))?` + // Quality (optional)
        `(?:[.\s_-]+(?P<source>[A-Za-z-]+))?`+ // Source (optional)
        `(?:[.\s_-]+(?P<codec>x26[45]|HEVC|AV1))?` + // Codec (optional)
        `(?:[.\s_-]+(?P<group>[A-Za-z0-9]+))?` + // Group (optional)
        `(?:\.[a-z]{2,4})?$`,                  // Extension
    )

    // Year in parentheses: Movie Name (2024) 1080p.mkv
    MoviePatternAlt = regexp.MustCompile(
        `^(?P<title>.+?)` +
        `\s*\((?P<year>(?:19|20)\d{2})\)` +
        `(?:[.\s_-]+(?P<quality>\d{3,4}p))?` +
        `.*$`,
    )
)

// TV Show patterns
var (
    // Standard: Show.Name.S01E05.720p.WEB-DL.mkv
    TVPattern = regexp.MustCompile(
        `^(?P<title>.+?)` +
        `[.\s_-]+` +
        `[Ss](?P<season>\d{1,2})[Ee](?P<episode>\d{1,3})` +
        `(?:-[Ee]?(?P<episode_end>\d{1,3}))?` +  // Episode range
        `(?:[.\s_-]+(?P<quality>\d{3,4}p))?` +
        `(?:[.\s_-]+(?P<source>[A-Za-z-]+))?` +
        `.*$`,
    )

    // Alternative: Show.Name.1x05.mkv
    TVPatternAlt = regexp.MustCompile(
        `^(?P<title>.+?)` +
        `[.\s_-]+` +
        `(?P<season>\d{1,2})x(?P<episode>\d{1,3})` +
        `.*$`,
    )

    // Daily show: Show.Name.2024.01.15.mkv
    TVPatternDaily = regexp.MustCompile(
        `^(?P<title>.+?)` +
        `[.\s_-]+` +
        `(?P<year>\d{4})[.\s_-](?P<month>\d{2})[.\s_-](?P<day>\d{2})` +
        `.*$`,
    )

    // Anime: [Group] Show Name - 01 [1080p].mkv
    AnimePattern = regexp.MustCompile(
        `^\[(?P<group>[^\]]+)\]\s*` +
        `(?P<title>.+?)` +
        `\s*-\s*` +
        `(?P<episode>\d{1,3})` +
        `.*$`,
    )
)
```

### Quality Detection

```go
// parser/quality.go

package parser

import (
    "regexp"
    "strings"
)

var qualityMap = map[string]string{
    "2160p": "2160p",
    "4k":    "2160p",
    "uhd":   "2160p",
    "1080p": "1080p",
    "1080i": "1080p",
    "720p":  "720p",
    "576p":  "576p",
    "480p":  "480p",
    "sd":    "480p",
}

var sourceMap = map[string]string{
    "bluray":    "BluRay",
    "blu-ray":   "BluRay",
    "bdrip":     "BluRay",
    "brrip":     "BluRay",
    "web-dl":    "WEB-DL",
    "webdl":     "WEB-DL",
    "webrip":    "WEBRip",
    "web":       "WEB",
    "hdtv":      "HDTV",
    "pdtv":      "PDTV",
    "dsr":       "DSR",
    "dvdrip":    "DVDRip",
    "dvd":       "DVD",
    "hdcam":     "HDCAM",
    "cam":       "CAM",
    "ts":        "TS",
    "telesync":  "TS",
    "screener":  "SCR",
    "r5":        "R5",
}

var codecMap = map[string]string{
    "x264":  "x264",
    "h264":  "x264",
    "avc":   "x264",
    "x265":  "x265",
    "h265":  "x265",
    "hevc":  "x265",
    "av1":   "AV1",
    "xvid":  "XviD",
    "divx":  "DivX",
}

func DetectQuality(filename string) string {
    lower := strings.ToLower(filename)
    for pattern, quality := range qualityMap {
        if strings.Contains(lower, pattern) {
            return quality
        }
    }
    return ""
}

func DetectSource(filename string) string {
    lower := strings.ToLower(filename)
    for pattern, source := range sourceMap {
        if strings.Contains(lower, pattern) {
            return source
        }
    }
    return ""
}

func DetectCodec(filename string) string {
    lower := strings.ToLower(filename)
    for pattern, codec := range codecMap {
        if strings.Contains(lower, pattern) {
            return codec
        }
    }
    return ""
}
```

### Title Cleaner

```go
// parser/cleaner.go

package parser

import (
    "regexp"
    "strings"
)

var (
    // Patterns to remove from title
    yearPattern    = regexp.MustCompile(`\(?\d{4}\)?`)
    qualityPattern = regexp.MustCompile(`(?i)\d{3,4}p`)
    sourcePattern  = regexp.MustCompile(`(?i)(bluray|blu-ray|web-dl|webdl|webrip|hdtv|dvdrip)`)
    codecPattern   = regexp.MustCompile(`(?i)(x26[45]|hevc|avc|av1)`)
    groupPattern   = regexp.MustCompile(`-[A-Za-z0-9]+$`)
    bracketPattern = regexp.MustCompile(`\[[^\]]*\]|\([^)]*\)`)

    // Separators to normalize
    separatorPattern = regexp.MustCompile(`[._]+`)
    multiSpacePattern = regexp.MustCompile(`\s{2,}`)
)

func CleanTitle(title string) string {
    cleaned := title

    // Remove bracketed content (often release info)
    cleaned = bracketPattern.ReplaceAllString(cleaned, " ")

    // Remove quality/source/codec indicators
    cleaned = qualityPattern.ReplaceAllString(cleaned, " ")
    cleaned = sourcePattern.ReplaceAllString(cleaned, " ")
    cleaned = codecPattern.ReplaceAllString(cleaned, " ")

    // Remove release group
    cleaned = groupPattern.ReplaceAllString(cleaned, "")

    // Replace separators with spaces
    cleaned = separatorPattern.ReplaceAllString(cleaned, " ")

    // Normalize whitespace
    cleaned = multiSpacePattern.ReplaceAllString(cleaned, " ")
    cleaned = strings.TrimSpace(cleaned)

    return cleaned
}

// CleanTitleForSearch prepares title for TMDb search
func CleanTitleForSearch(title string) string {
    cleaned := CleanTitle(title)

    // Remove year from search query (TMDb handles it separately)
    cleaned = yearPattern.ReplaceAllString(cleaned, "")
    cleaned = strings.TrimSpace(cleaned)

    return cleaned
}
```

### Parser Service

```go
// services/parser_service.go

package services

import (
    "log/slog"
    "time"

    "github.com/alexyu/vido/apps/api/internal/parser"
)

type ParserServiceInterface interface {
    ParseFilename(filename string) *parser.ParseResult
    ParseBatch(filenames []string) []*parser.ParseResult
}

type ParserService struct {
    movieParser parser.MovieParser
    tvParser    parser.TVParser
}

func NewParserService() *ParserService {
    return &ParserService{
        movieParser: parser.NewMovieParser(),
        tvParser:    parser.NewTVParser(),
    }
}

func (s *ParserService) ParseFilename(filename string) *parser.ParseResult {
    start := time.Now()

    // Try TV show pattern first (more specific)
    result := s.tvParser.Parse(filename)
    if result.Status == parser.ParseStatusSuccess {
        result.Confidence = 85
        slog.Debug("Parsed as TV show",
            "filename", filename,
            "title", result.Title,
            "season", result.Season,
            "episode", result.Episode,
            "duration_ms", time.Since(start).Milliseconds(),
        )
        return result
    }

    // Try movie pattern
    result = s.movieParser.Parse(filename)
    if result.Status == parser.ParseStatusSuccess {
        result.Confidence = 80
        slog.Debug("Parsed as movie",
            "filename", filename,
            "title", result.Title,
            "year", result.Year,
            "duration_ms", time.Since(start).Milliseconds(),
        )
        return result
    }

    // Failed to parse - needs AI
    slog.Info("Regex parsing failed, flagging for AI",
        "filename", filename,
        "duration_ms", time.Since(start).Milliseconds(),
    )

    return &parser.ParseResult{
        OriginalFilename: filename,
        Status:          parser.ParseStatusNeedsAI,
        MediaType:       parser.MediaTypeUnknown,
        Confidence:      0,
        ErrorMessage:    "Could not parse with standard patterns",
    }
}

func (s *ParserService) ParseBatch(filenames []string) []*parser.ParseResult {
    results := make([]*parser.ParseResult, len(filenames))
    for i, filename := range filenames {
        results[i] = s.ParseFilename(filename)
    }
    return results
}
```

### API Response Format

```json
// POST /api/v1/parser/parse
// Request:
{
  "filename": "Breaking.Bad.S01E05.720p.BluRay.x264-DEMAND.mkv"
}

// Response (success):
{
  "success": true,
  "data": {
    "original_filename": "Breaking.Bad.S01E05.720p.BluRay.x264-DEMAND.mkv",
    "status": "success",
    "media_type": "tv",
    "title": "Breaking Bad",
    "cleaned_title": "Breaking Bad",
    "season": 1,
    "episode": 5,
    "quality": "720p",
    "source": "BluRay",
    "video_codec": "x264",
    "release_group": "DEMAND",
    "confidence": 85
  }
}

// Response (needs AI):
{
  "success": true,
  "data": {
    "original_filename": "[Leopard-Raws] Kimetsu no Yaiba - 26 (BD 1920x1080).mkv",
    "status": "needs_ai",
    "media_type": "unknown",
    "confidence": 0,
    "error_message": "Could not parse with standard patterns"
  }
}
```

### Test Fixtures

```go
// parser/parser_test.go

var movieTestCases = []struct {
    filename string
    expected ParseResult
}{
    // Standard formats
    {"The.Matrix.1999.1080p.BluRay.x264-GROUP.mkv", ParseResult{Title: "The Matrix", Year: 1999, Quality: "1080p", Source: "BluRay"}},
    {"Inception.2010.2160p.UHD.BluRay.mkv", ParseResult{Title: "Inception", Year: 2010, Quality: "2160p", Source: "BluRay"}},
    {"Parasite.2019.720p.WEB-DL.mkv", ParseResult{Title: "Parasite", Year: 2019, Quality: "720p", Source: "WEB-DL"}},

    // Year in parentheses
    {"The Dark Knight (2008) 1080p.mkv", ParseResult{Title: "The Dark Knight", Year: 2008, Quality: "1080p"}},

    // Titles with years
    {"2001.A.Space.Odyssey.1968.1080p.mkv", ParseResult{Title: "2001 A Space Odyssey", Year: 1968, Quality: "1080p"}},
    {"Blade.Runner.2049.2017.1080p.mkv", ParseResult{Title: "Blade Runner 2049", Year: 2017, Quality: "1080p"}},

    // Various separators
    {"Avengers_Endgame_2019_1080p.mkv", ParseResult{Title: "Avengers Endgame", Year: 2019, Quality: "1080p"}},
    {"Avengers Endgame 2019 1080p.mkv", ParseResult{Title: "Avengers Endgame", Year: 2019, Quality: "1080p"}},
}

var tvTestCases = []struct {
    filename string
    expected ParseResult
}{
    // Standard S01E05 format
    {"Breaking.Bad.S01E05.720p.BluRay.mkv", ParseResult{Title: "Breaking Bad", Season: 1, Episode: 5, Quality: "720p"}},
    {"Game.of.Thrones.S08E06.1080p.WEB-DL.mkv", ParseResult{Title: "Game of Thrones", Season: 8, Episode: 6, Quality: "1080p"}},

    // Episode ranges
    {"Friends.S01E01-E02.DVDRip.mkv", ParseResult{Title: "Friends", Season: 1, Episode: 1, EpisodeEnd: 2}},

    // Alternative 1x05 format
    {"House.1x13.720p.mkv", ParseResult{Title: "House", Season: 1, Episode: 13, Quality: "720p"}},

    // Daily shows
    {"The.Daily.Show.2024.01.15.720p.mkv", ParseResult{Title: "The Daily Show", Year: 2024}},
}

var animeTestCases = []struct {
    filename string
    expected ParseResult
}{
    // Fansub format - should return NeedsAI
    {"[Leopard-Raws] Kimetsu no Yaiba - 26 (BD 1920x1080).mkv", ParseResult{Status: ParseStatusNeedsAI}},
    {"【幻櫻字幕組】【4月新番】我的英雄學院 第01話 1080P.mp4", ParseResult{Status: ParseStatusNeedsAI}},
}
```

### Performance Requirements

From NFR-P13: Standard regex parsing must complete within **100ms per file**.

```go
// parser/parser_benchmark_test.go

func BenchmarkParseFilename(b *testing.B) {
    service := NewParserService()
    filename := "Breaking.Bad.S01E05.720p.BluRay.x264-DEMAND.mkv"

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        service.ParseFilename(filename)
    }
}

// Expected result: < 1ms per operation (well under 100ms target)
```

### Project Structure After This Story

```
apps/api/internal/
├── parser/                      # NEW: Parser package
│   ├── types.go                 # ParseResult, ParseStatus, MediaType
│   ├── patterns.go              # Regex patterns
│   ├── movie_parser.go          # Movie filename parser
│   ├── movie_parser_test.go
│   ├── tv_parser.go             # TV show filename parser
│   ├── tv_parser_test.go
│   ├── quality.go               # Quality/source/codec detection
│   ├── quality_test.go
│   ├── cleaner.go               # Title cleaner
│   ├── cleaner_test.go
│   └── parser_benchmark_test.go
├── services/
│   ├── parser_service.go        # NEW: Parser service
│   ├── parser_service_test.go
│   └── ...
├── handlers/
│   ├── parser_handler.go        # NEW: Parser HTTP handler
│   ├── parser_handler_test.go
│   └── ...
```

### Integration with TMDb (Future)

The parser output's `cleaned_title` field is designed for TMDb search:

```go
// Future usage (Story 2.6 and beyond):
result := parserService.ParseFilename(filename)
if result.Status == parser.ParseStatusSuccess {
    // Use cleaned_title for TMDb search
    tmdbResults, err := tmdbService.SearchMovies(result.CleanedTitle)

    // Use year to filter results
    for _, movie := range tmdbResults {
        if movie.Year == result.Year {
            // Found match
        }
    }
}
```

### Logging Standard

**MUST use `log/slog`**:

```go
// ✅ CORRECT
slog.Debug("Parsing filename", "filename", filename)
slog.Info("Regex parsing failed", "filename", filename, "reason", "no pattern match")
slog.Error("Parser error", "error", err, "filename", filename)

// ❌ WRONG
log.Printf("Parsing: %s", filename)
fmt.Println("Failed:", filename)
```

### References

- [Source: project-context.md#Rule 2: Logging with slog ONLY]
- [Source: project-context.md#Rule 4: Layered Architecture]
- [Source: architecture.md#Filename Parsing]
- [Source: architecture.md#NFR-P13: Parsing <100ms]
- [Source: epics.md#Story 2.5: Standard Filename Parser]

## Dev Agent Record

### Agent Model Used

Claude Opus 4.5 (claude-opus-4-5-20251101)

### Debug Log References

### Completion Notes List

1. Implemented complete regex-based filename parser with TDD approach (Red-Green-Refactor cycle)
2. Parser correctly handles standard movie formats (Title.Year.Quality.Source.mkv), TV show formats (Title.S01E05.Quality.mkv), and alternative formats (1x05, daily shows, anime Episode prefix)
3. Files that cannot be parsed (fansub brackets, Chinese/Japanese characters) return `needs_ai` status for future AI parsing (Epic 3)
4. Performance benchmark verified: parsing 1000 files completes in ~42ms (avg 42µs/file), well under 100ms NFR-P13 requirement
5. Title cleaning replaces dots/underscores/hyphens with spaces (e.g., "Spider-Man" becomes "Spider Man", "9-1-1" becomes "9 1 1") - this is expected behavior for search optimization
6. Special handling for movies with years in title (2001 A Space Odyssey, Blade Runner 2049, 1917) using multiple year detection
7. Source pattern uses word boundaries to prevent partial matches (e.g., "Scrubs" no longer matches "SCR")
8. Comprehensive test suite includes 50+ real-world filename examples covering movies, TV shows, edge cases, and NeedsAI scenarios

### File List

| File | Description |
|------|-------------|
| `apps/api/internal/parser/types.go` | ParseResult, ParseStatus, MediaType definitions |
| `apps/api/internal/parser/types_test.go` | Type tests |
| `apps/api/internal/parser/patterns.go` | Regex patterns for movie and TV parsing |
| `apps/api/internal/parser/movie_parser.go` | Movie filename parser implementation |
| `apps/api/internal/parser/movie_parser_test.go` | Movie parser tests |
| `apps/api/internal/parser/tv_parser.go` | TV show filename parser implementation |
| `apps/api/internal/parser/tv_parser_test.go` | TV parser tests |
| `apps/api/internal/parser/quality.go` | Quality, source, codec detection |
| `apps/api/internal/parser/quality_test.go` | Quality detection tests |
| `apps/api/internal/parser/cleaner.go` | Title cleaning utilities |
| `apps/api/internal/parser/cleaner_test.go` | Title cleaner tests |
| `apps/api/internal/parser/parser_benchmark_test.go` | Performance benchmarks |
| `apps/api/internal/parser/parser_comprehensive_test.go` | 50+ real-world test cases |
| `apps/api/internal/services/parser_service.go` | Parser service layer |
| `apps/api/internal/services/parser_service_test.go` | Parser service tests |
| `apps/api/internal/handlers/parser_handler.go` | HTTP handler for parser endpoints |
| `apps/api/internal/handlers/parser_handler_test.go` | Parser handler tests |
| `apps/api/cmd/api/main.go` | Route registration for parser endpoints (CR fix) |

### Senior Developer Review (AI)

**Review Date:** 2026-01-17
**Reviewer:** Claude Opus 4.5 (code-review workflow)
**Outcome:** APPROVED (after fixes)

#### Issues Found and Fixed

| Severity | Issue | Resolution |
|----------|-------|------------|
| 🔴 HIGH | Parser API endpoints not registered in main.go | Fixed: Added parserService, parserHandler initialization and route registration |
| 🟡 MEDIUM | Unused `templates` variable in benchmark test | Fixed: Removed dead code |
| 🟢 LOW | Title hyphen→space conversion documented | Acknowledged as expected behavior for TMDb search |
| 🟢 LOW | Swagger docs not verified | Deferred to documentation sprint |

#### Verification

- ✅ All 50+ unit tests passing
- ✅ Benchmark: ~42µs/file (well under 100ms NFR-P13)
- ✅ Build compiles successfully
- ✅ API endpoints now registered at `/api/v1/parser/parse` and `/api/v1/parser/parse-batch`

