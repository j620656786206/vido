# Story 3.1: AI Provider Abstraction Layer

Status: ready-for-dev

## Story

As a **developer**,
I want an **abstraction layer for AI providers**,
So that **we can switch between Gemini and Claude without code changes**.

## Acceptance Criteria

1. **AC1: Provider Selection via Environment Variable**
   - Given the system needs AI parsing capabilities
   - When configuring the AI provider
   - Then users can select Gemini or Claude via environment variable `AI_PROVIDER`
   - And the same interface is used regardless of provider

2. **AC2: Normalized API Responses**
   - Given an AI provider is configured
   - When making API calls
   - Then the system uses the appropriate API format for that provider
   - And responses are normalized to a common format

3. **AC3: 30-Day Result Caching**
   - Given AI parsing results are returned
   - When caching the results
   - Then results are cached for 30 days (NFR-I10)
   - And cache key is based on filename hash

4. **AC4: Timeout with Fallback**
   - Given AI API calls are made
   - When the call exceeds 15 seconds
   - Then it times out and falls back to next option (NFR-I12)

## Tasks / Subtasks

- [ ] Task 1: Create AI types and interfaces (AC: 1, 2)
  - [ ] 1.1: Create `/apps/api/internal/ai/types.go` with common response types
  - [ ] 1.2: Create `/apps/api/internal/ai/provider.go` with `AIProvider` interface
  - [ ] 1.3: Define `ParseRequest` and `ParseResponse` structs

- [ ] Task 2: Implement Gemini provider (AC: 1, 2, 4)
  - [ ] 2.1: Create `/apps/api/internal/ai/gemini.go` with `GeminiProvider` struct
  - [ ] 2.2: Implement API call with proper authentication
  - [ ] 2.3: Implement response normalization to common format
  - [ ] 2.4: Add 15-second timeout with context cancellation
  - [ ] 2.5: Write unit tests in `gemini_test.go` (mock HTTP responses)

- [ ] Task 3: Implement Claude provider (AC: 1, 2, 4)
  - [ ] 3.1: Create `/apps/api/internal/ai/claude.go` with `ClaudeProvider` struct
  - [ ] 3.2: Implement API call with proper authentication
  - [ ] 3.3: Implement response normalization to common format
  - [ ] 3.4: Add 15-second timeout with context cancellation
  - [ ] 3.5: Write unit tests in `claude_test.go` (mock HTTP responses)

- [ ] Task 4: Create provider factory (AC: 1)
  - [ ] 4.1: Create `/apps/api/internal/ai/factory.go` with `NewProvider()` function
  - [ ] 4.2: Read `AI_PROVIDER` env var to select provider
  - [ ] 4.3: Return appropriate provider based on configuration
  - [ ] 4.4: Write factory tests

- [ ] Task 5: Update configuration (AC: 1)
  - [ ] 5.1: Add `AIProvider` field to config struct (gemini|claude)
  - [ ] 5.2: Add `ClaudeAPIKey` field to config struct
  - [ ] 5.3: Update `HasAIProvider()` to check both keys
  - [ ] 5.4: Add `GetAIProvider()` method
  - [ ] 5.5: Update config tests

- [ ] Task 6: Implement caching layer (AC: 3)
  - [ ] 6.1: Create `ai_cache` table in database (migration)
  - [ ] 6.2: Create `/apps/api/internal/ai/cache.go` with cache logic
  - [ ] 6.3: Implement SHA-256 hash for filename-based cache key
  - [ ] 6.4: Implement 30-day TTL with expiry cleanup
  - [ ] 6.5: Write cache tests

- [ ] Task 7: Create AI Service (AC: 1, 2, 3, 4)
  - [ ] 7.1: Create `/apps/api/internal/services/ai_service.go`
  - [ ] 7.2: Define `AIServiceInterface` in services package
  - [ ] 7.3: Implement `ParseFilename()` method with cache-first strategy
  - [ ] 7.4: Wire up in `main.go`
  - [ ] 7.5: Write service tests with mocked provider

- [ ] Task 8: Integration with existing parser (AC: 1, 2)
  - [ ] 8.1: Update `ParserService` to optionally use `AIService`
  - [ ] 8.2: When regex fails and `ParseStatusNeedsAI`, delegate to AI service
  - [ ] 8.3: Write integration tests

## Dev Notes

### Architecture Requirements

**ARCH-3: AI Provider Abstraction Layer**
- Support Gemini/Claude switching via environment variable
- 30-day caching for AI parsing results
- Strategy pattern for provider switching

**NFRs Covered:**
- NFR-I9: AI provider abstraction (Gemini/Claude)
- NFR-I10: AI parsing cache 30 days
- NFR-I11: Per-user AI API usage tracking (future story)
- NFR-I12: AI API 15s timeout with fallback

### Current Codebase State

**What Already Exists:**
- `config.GeminiAPIKey` - Gemini API key loading from env
- `config.HasAIProvider()` - Returns true if any AI provider configured
- `parser.ParseStatusNeedsAI` - Parser already flags files needing AI
- Well-established service/repository layering pattern

**What Needs to Be Created:**
- `/apps/api/internal/ai/` directory (new package)
- AI provider interface and implementations
- Claude API key configuration
- AI service orchestration
- 30-day caching table and logic

### Project Structure Notes

**New Directory to Create:**
```
/apps/api/internal/ai/
├── types.go              # Common types (ParseRequest, ParseResponse)
├── provider.go           # AIProvider interface
├── factory.go            # NewProvider() factory function
├── cache.go              # Cache logic with 30-day TTL
├── gemini.go             # Gemini API client
├── gemini_test.go
├── claude.go             # Claude API client
└── claude_test.go
```

**Files to Modify:**
- `/apps/api/internal/config/config.go` - Add Claude key, AI provider selection
- `/apps/api/internal/services/parser_service.go` - Optional AI integration
- `/apps/api/cmd/api/main.go` - Wire up AI service

### API Integration Details

**Gemini API (generativelanguage.googleapis.com):**
```
Endpoint: POST https://generativelanguage.googleapis.com/v1/models/gemini-pro:generateContent
Auth: API key in URL param or header
Timeout: 15 seconds
```

**Claude API (api.anthropic.com):**
```
Endpoint: POST https://api.anthropic.com/v1/messages
Auth: x-api-key header
Timeout: 15 seconds
```

### Database Migration

**Migration file: `XXX_create_ai_cache_table.sql`**
```sql
CREATE TABLE IF NOT EXISTS ai_cache (
    id TEXT PRIMARY KEY,
    filename_hash TEXT UNIQUE NOT NULL,
    provider TEXT NOT NULL,
    request_prompt TEXT NOT NULL,
    response_json TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL
);

CREATE INDEX idx_ai_cache_filename_hash ON ai_cache(filename_hash);
CREATE INDEX idx_ai_cache_expires_at ON ai_cache(expires_at);
```

### Interface Design

```go
// AIProvider interface (Strategy pattern)
type AIProvider interface {
    Parse(ctx context.Context, req *ParseRequest) (*ParseResponse, error)
    Name() string
}

// ParseRequest for AI parsing
type ParseRequest struct {
    Filename string
    Prompt   string  // Optional custom prompt
}

// ParseResponse normalized from all providers
type ParseResponse struct {
    Title       string   `json:"title"`
    Year        int      `json:"year,omitempty"`
    Season      int      `json:"season,omitempty"`
    Episode     int      `json:"episode,omitempty"`
    MediaType   string   `json:"media_type"`  // "movie" or "tv"
    Quality     string   `json:"quality,omitempty"`
    FansubGroup string   `json:"fansub_group,omitempty"`
    Confidence  float64  `json:"confidence"`
    RawResponse string   `json:"raw_response"`
}
```

### Testing Strategy

1. **Unit Tests:** Mock HTTP client for API responses
2. **Cache Tests:** Test TTL expiry, hash generation
3. **Factory Tests:** Test provider selection logic
4. **Integration Tests:** Test full flow with mocked provider

**Coverage Targets:**
- Services: ≥80%
- AI package: ≥80%

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-3.1]
- [Source: _bmad-output/planning-artifacts/architecture.md#AI-Provider-Abstraction]
- [Source: project-context.md#Rule-4-Layered-Architecture]
- [Source: project-context.md#Rule-11-Interface-Location]

### Previous Story Intelligence

**From Epic 2 Implementation:**
- Repository pattern well-established in `/apps/api/internal/repository/`
- Services use dependency injection via constructor
- Config loaded at startup in `main.go`
- TMDb client pattern in `/apps/api/internal/tmdb/` can serve as reference for external API integration

**From Recent Commits:**
- `8ba7f8c feat(api): implement regex-based filename parser (Story 2.5)` - Parser already exists, marks `ParseStatusNeedsAI`
- Test coverage patterns established across packages

### Environment Variables Required

```bash
# Provider selection (required if using AI)
AI_PROVIDER=gemini  # or "claude"

# API Keys (one required based on provider)
GEMINI_API_KEY=your-gemini-key
CLAUDE_API_KEY=your-claude-key  # New - needs to be added
```

### Error Codes to Implement

Following project-context.md Rule 7:
- `AI_TIMEOUT` - AI parsing timeout (>15s)
- `AI_QUOTA_EXCEEDED` - User's API quota exhausted
- `AI_INVALID_RESPONSE` - Unparseable AI response
- `AI_PROVIDER_ERROR` - Generic provider error
- `AI_NOT_CONFIGURED` - No AI provider configured

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
