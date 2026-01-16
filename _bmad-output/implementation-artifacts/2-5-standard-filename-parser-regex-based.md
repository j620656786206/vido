# Story 2.5: Standard Filename Parser (Regex-based)

Status: ready-for-dev

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
- [ ] 1.1 Create `apps/api/internal/parser/` package
- [ ] 1.2 Define `ParseResult` struct with all metadata fields
- [ ] 1.3 Define `ParseStatus` enum (Success, NeedsAI, Failed)
- [ ] 1.4 Create `ParserInterface` for future AI parser integration

### Task 2: Implement Movie Filename Parser (AC: #1, #5)
- [ ] 2.1 Create regex pattern for standard movie naming
- [ ] 2.2 Extract title (handle dots, spaces, underscores)
- [ ] 2.3 Extract year (4-digit number, typically 1900-2099)
- [ ] 2.4 Extract quality (480p, 720p, 1080p, 2160p/4K)
- [ ] 2.5 Extract source (BluRay, WEB-DL, HDTV, DVDRip, etc.)
- [ ] 2.6 Extract codec (x264, x265/HEVC, AV1, etc.)
- [ ] 2.7 Handle release group tags (e.g., `-SPARKS`, `-YTS`)

### Task 3: Implement TV Show Filename Parser (AC: #2, #5)
- [ ] 3.1 Create regex pattern for TV show naming (S01E05 format)
- [ ] 3.2 Support alternative formats (1x05, Season 1 Episode 5)
- [ ] 3.3 Extract season number
- [ ] 3.4 Extract episode number (single or range: E01-E03)
- [ ] 3.5 Handle daily shows (2024.01.15 format)
- [ ] 3.6 Handle anime episode numbering (Episode 01, Ep01, 01)

### Task 4: Implement Quality/Source Detection (AC: #1, #2)
- [ ] 4.1 Create quality detector with all common resolutions
- [ ] 4.2 Create source detector (BluRay, WEB-DL, HDTV, etc.)
- [ ] 4.3 Create codec detector (x264, x265, HEVC, AV1)
- [ ] 4.4 Create audio codec detector (AAC, DTS, Atmos, etc.)
- [ ] 4.5 Normalize quality values to standard format

### Task 5: Implement Title Cleaner (AC: #1, #2, #5)
- [ ] 5.1 Remove release group tags
- [ ] 5.2 Replace dots/underscores with spaces
- [ ] 5.3 Remove quality/source/codec from title
- [ ] 5.4 Handle edge cases (titles with years, e.g., "2001 A Space Odyssey")
- [ ] 5.5 Trim and normalize whitespace

### Task 6: Create Parser Service Layer (AC: #3, #4)
- [ ] 6.1 Create `apps/api/internal/services/parser_service.go`
- [ ] 6.2 Implement `ParseFilename(filename string) (*ParseResult, error)`
- [ ] 6.3 Try movie pattern first, then TV show pattern
- [ ] 6.4 Return appropriate status (Success, NeedsAI, Failed)
- [ ] 6.5 Log parsing results with slog

### Task 7: Create Parser Handler (AC: #1, #2, #4)
- [ ] 7.1 Create `apps/api/internal/handlers/parser_handler.go`
- [ ] 7.2 Implement `POST /api/v1/parser/parse` endpoint
- [ ] 7.3 Implement `POST /api/v1/parser/parse-batch` for multiple files
- [ ] 7.4 Return structured response with parse status

### Task 8: Write Comprehensive Tests (AC: #1, #2, #3, #5)
- [ ] 8.1 Create test fixtures with 50+ real-world filename examples
- [ ] 8.2 Test movie parsing with various formats
- [ ] 8.3 Test TV show parsing with various formats
- [ ] 8.4 Test edge cases (unusual characters, long names)
- [ ] 8.5 Benchmark test to verify <100ms performance
- [ ] 8.6 Test title cleaning edge cases

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

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List

