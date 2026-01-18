# Story 3.2: AI Fansub Filename Parsing

Status: ready-for-dev

## Story

As a **media collector with fansub content**,
I want the **system to parse complex fansub naming using AI**,
So that **files like `[Leopard-Raws] Show - 01 (BD 1080p).mkv` are correctly identified**.

## Acceptance Criteria

1. **AC1: Parse Japanese Fansub Naming**
   - Given a fansub filename like `[Leopard-Raws] Kimetsu no Yaiba - 26 (BD 1920x1080 x264 FLAC).mkv`
   - When AI parsing is triggered
   - Then it extracts:
     - Fansub group: Leopard-Raws (ignored for search)
     - Title: Kimetsu no Yaiba / 鬼滅之刃
     - Episode: 26
     - Quality: 1080p
     - Source: BD (Blu-ray)

2. **AC2: Parse Chinese Fansub Naming**
   - Given a Chinese fansub filename like `【幻櫻字幕組】【4月新番】我的英雄學院 第01話 1080P【繁體】.mp4`
   - When AI parsing is triggered
   - Then it extracts:
     - Title: 我的英雄學院
     - Episode: 1
     - Quality: 1080P
     - Language: Traditional Chinese

3. **AC3: Progress Indicator During Parsing**
   - Given AI parsing is in progress
   - When the user views the status
   - Then they see a progress indicator showing current step (UX-3)
   - And parsing completes within 10 seconds (NFR-P14)

4. **AC4: Integration with Existing Parser**
   - Given regex parsing fails with `ParseStatusNeedsAI`
   - When AI parsing is available
   - Then the system automatically delegates to AI service
   - And updates the parse status accordingly

## Tasks / Subtasks

- [ ] Task 1: Design AI Prompt for Fansub Parsing (AC: 1, 2)
  - [ ] 1.1: Create prompt template for extracting structured data from fansub filenames
  - [ ] 1.2: Include examples of common fansub naming patterns (Japanese, Chinese, English)
  - [ ] 1.3: Define JSON output schema for AI response
  - [ ] 1.4: Test prompt with various fansub naming conventions
  - [ ] 1.5: Store prompt in `/apps/api/internal/ai/prompts/fansub_parser.go`

- [ ] Task 2: Extend AI Service with Fansub Parsing (AC: 1, 2, 4)
  - [ ] 2.1: Add `ParseFansubFilename()` method to AIService interface
  - [ ] 2.2: Implement fansub-specific parsing logic using AI provider
  - [ ] 2.3: Map AI response to existing `parser.ParseResult` struct
  - [ ] 2.4: Handle edge cases (partial extraction, low confidence)
  - [ ] 2.5: Write unit tests with mocked AI responses

- [ ] Task 3: Create Fansub Pattern Recognition (AC: 1, 2)
  - [ ] 3.1: Create `/apps/api/internal/ai/fansub_detector.go`
  - [ ] 3.2: Implement `IsFansubFilename()` to detect fansub patterns
  - [ ] 3.3: Recognize common fansub group brackets: `[]`, `【】`, `「」`
  - [ ] 3.4: Write detection tests

- [ ] Task 4: Integrate with Parser Service (AC: 4)
  - [ ] 4.1: Update `ParserService` to accept optional `AIService` dependency
  - [ ] 4.2: When `ParseStatusNeedsAI`, call AI service if available
  - [ ] 4.3: Update parse status to `success` or `failed` based on AI result
  - [ ] 4.4: Store metadata source as "ai" when AI parsing succeeds
  - [ ] 4.5: Write integration tests

- [ ] Task 5: Implement Performance Optimization (AC: 3)
  - [ ] 5.1: Add context timeout of 10 seconds (NFR-P14)
  - [ ] 5.2: Implement early termination if AI takes too long
  - [ ] 5.3: Return partial results if available before timeout
  - [ ] 5.4: Add performance logging with slog

- [ ] Task 6: Add Parse Status Tracking (AC: 3)
  - [ ] 6.1: Add `ParseStatusParsing` status for in-progress parsing
  - [ ] 6.2: Emit parsing progress events (for future WebSocket/SSE)
  - [ ] 6.3: Store parsing attempt metadata (start time, provider used)

- [ ] Task 7: Handle Fansub-Specific Metadata (AC: 1, 2)
  - [ ] 7.1: Extract fansub group name (store but don't use for search)
  - [ ] 7.2: Extract language/subtitle indicators (繁體, 簡體, etc.)
  - [ ] 7.3: Extract video source (BD, WEB, TV, etc.)
  - [ ] 7.4: Extract codec information (x264, x265, HEVC, etc.)
  - [ ] 7.5: Map extracted quality to standard values (1080p, 720p, 4K)

## Dev Notes

### Dependencies

**CRITICAL: This story depends on Story 3.1 (AI Provider Abstraction Layer)**
- Must have `AIProvider` interface implemented
- Must have `AIService` with `ParseFilename()` method
- Must have 30-day caching for AI results

### Architecture Requirements

**FR15: Parse fansub naming conventions using AI (Gemini/Claude)**
- AI prompt engineered for fansub pattern recognition
- Uses Provider Abstraction from Story 3.1

**NFR-P14: AI fansub parsing <10 seconds per file**
- Context timeout must be enforced
- Early termination for slow responses

### Existing Codebase Integration

**Current ParseStatus Values (`/apps/api/internal/models/movie.go`):**
```go
const (
    ParseStatusPending ParseStatus = "pending"
    ParseStatusSuccess ParseStatus = "success"
    ParseStatusNeedsAI ParseStatus = "needs_ai"  // ← Trigger for AI parsing
    ParseStatusFailed  ParseStatus = "failed"
)
```

**Current ParseResult Struct (`/apps/api/internal/parser/types.go`):**
```go
type ParseResult struct {
    Status           ParseStatus
    MediaType        MediaType
    Title            string
    Year             int
    Season           int
    Episode          int
    Quality          string
    Source           string
    Codec            string
    ReleaseGroup     string
    OriginalFilename string
    Confidence       float64
    ErrorMessage     string
}
```

### AI Prompt Design

**Recommended Prompt Structure:**
```
You are a media filename parser specialized in fansub releases. Extract structured metadata from the following filename.

Filename: {{filename}}

Common fansub patterns:
- [GroupName] Title - Episode (Quality Info).ext
- 【字幕組】【標籤】標題 第XX話 解析度【語言】.ext
- [Group] Title S01E01 (1080p BD HEVC).ext

Extract the following fields in JSON format:
{
  "title": "extracted title (in original language)",
  "title_romanized": "romanized title if applicable",
  "episode": number or null,
  "season": number or null,
  "year": number or null,
  "quality": "1080p/720p/4K/etc or null",
  "source": "BD/WEB/TV/etc or null",
  "codec": "x264/x265/HEVC/etc or null",
  "fansub_group": "group name or null",
  "language": "language indicator or null",
  "media_type": "movie" or "tv",
  "confidence": 0.0-1.0
}

If you cannot parse the filename, return confidence: 0.0
```

### Common Fansub Naming Patterns

**Japanese Fansub Groups:**
- `[Leopard-Raws]` - Raw releases (no subtitles)
- `[SubsPlease]` - English subtitles
- `[Erai-raws]` - Multi-language releases
- `[Commie]`, `[HorribleSubs]` - Various fangroups

**Chinese Fansub Groups:**
- `【幻櫻字幕組】` - Traditional Chinese
- `【极影字幕社】` - Simplified Chinese
- `【動漫國字幕組】` - Taiwan-based
- `[ANK-Raws]` - Raw releases

**Common Patterns to Parse:**
```
[Leopard-Raws] Kimetsu no Yaiba - 26 (BD 1920x1080 x264 FLAC).mkv
【幻櫻字幕組】【4月新番】我的英雄學院 第01話 1080P【繁體】.mp4
[SubsPlease] Demon Slayer - 01 (1080p) [ABCD1234].mkv
[Commie] Steins;Gate 0 - 01 [BD 1080p AAC] [12345678].mkv
【极影字幕社】★ 进击的巨人 第01话 HDTV 720P 【简体】.mp4
```

### Project Structure Notes

**Files to Create:**
```
/apps/api/internal/ai/
├── prompts/
│   └── fansub_parser.go    # Prompt templates
├── fansub_detector.go      # Detect fansub patterns
└── fansub_detector_test.go
```

**Files to Modify:**
- `/apps/api/internal/services/ai_service.go` - Add `ParseFansubFilename()`
- `/apps/api/internal/services/parser_service.go` - Integrate AI fallback
- `/apps/api/internal/models/movie.go` - Add `ParseStatusParsing` if needed

### Testing Strategy

**Unit Tests:**
1. Prompt generation tests
2. Fansub detection tests (various bracket patterns)
3. AI response parsing tests (mock responses)
4. Timeout handling tests

**Integration Tests:**
1. Full flow: filename → AI parsing → ParseResult
2. Cache hit scenario (30-day cache from Story 3.1)
3. Fallback scenario when AI fails

**Test Data (Fansub Filenames):**
```go
var fansubTestCases = []struct {
    filename string
    expected ParseResult
}{
    {
        filename: "[Leopard-Raws] Kimetsu no Yaiba - 26 (BD 1920x1080 x264 FLAC).mkv",
        expected: ParseResult{
            Title:        "Kimetsu no Yaiba",
            Episode:      26,
            Quality:      "1080p",
            Source:       "BD",
            ReleaseGroup: "Leopard-Raws",
            MediaType:    MediaTypeTV,
        },
    },
    {
        filename: "【幻櫻字幕組】【4月新番】我的英雄學院 第01話 1080P【繁體】.mp4",
        expected: ParseResult{
            Title:        "我的英雄學院",
            Episode:      1,
            Quality:      "1080p",
            ReleaseGroup: "幻櫻字幕組",
            MediaType:    MediaTypeTV,
        },
    },
}
```

**Coverage Targets:**
- AI package: ≥80%
- Parser service: ≥80%

### Error Handling

**AI Parsing Errors:**
- `AI_TIMEOUT` - 10-second timeout exceeded
- `AI_PARSE_FAILED` - AI couldn't extract metadata
- `AI_LOW_CONFIDENCE` - Confidence < 0.5 (trigger manual review)

**Fallback Behavior:**
1. If AI fails → Set `ParseStatusFailed` with error message
2. If AI returns low confidence → Set `ParseStatusNeedsAI` for manual review
3. If AI succeeds → Set `ParseStatusSuccess`, store metadata

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-3.2]
- [Source: _bmad-output/implementation-artifacts/3-1-ai-provider-abstraction-layer.md]
- [Source: project-context.md#Rule-4-Layered-Architecture]
- [Source: apps/api/internal/parser/types.go#ParseResult]
- [Source: apps/api/internal/models/movie.go#ParseStatus]

### Previous Story Intelligence

**From Story 3.1 (AI Provider Abstraction):**
- `AIProvider` interface with `Parse()` method
- `ParseRequest` and `ParseResponse` types defined
- 30-day caching with filename hash key
- 15-second timeout per API call
- Error codes: `AI_TIMEOUT`, `AI_QUOTA_EXCEEDED`, etc.

**From Story 2.5 (Regex Parser):**
- Regex parser marks files as `ParseStatusNeedsAI` when it can't parse
- `ParseResult` struct established with all required fields
- Parser service pattern ready for AI integration

### Performance Requirements

| Metric | Target | Measurement |
|--------|--------|-------------|
| AI parsing time | <10 seconds | Context timeout |
| Cache hit rate | >80% | Prometheus metrics |
| Success rate | >90% | Logging analysis |

### Environment Variables

```bash
# From Story 3.1
AI_PROVIDER=gemini
GEMINI_API_KEY=your-key
# or
CLAUDE_API_KEY=your-key
```

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
