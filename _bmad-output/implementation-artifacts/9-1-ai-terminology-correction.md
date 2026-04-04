# Story 9.1: AI Terminology Correction

Status: ready-for-dev

## Story

As a Traditional Chinese NAS user,
I want the system to use Claude API for fine-grained terminology correction after OpenCC conversion,
so that cross-strait terminology differences (e.g., 軟件→軟體, 內存→記憶體) are caught beyond OpenCC's dictionary.

## Acceptance Criteria

1. Given a subtitle file that has been converted by OpenCC (s2twp), when the user has a valid Claude API key configured, then the system sends the converted text to Claude for terminology review and applies corrections
2. Given no Claude API key configured, when a subtitle is downloaded, then the system skips AI terminology correction and uses OpenCC output as-is (graceful degradation)
3. Given Claude API returns corrections, when applied, then the corrected subtitle replaces the OpenCC output at the same file path
4. Given Claude API times out or returns an error, when processing a subtitle, then the system falls back to the OpenCC output, logs a warning, and does not fail the subtitle pipeline
5. Given a subtitle with mixed content (dialogue + technical terms), when processed by Claude, then proper nouns (人名, 地名) are preserved while only cross-strait vocabulary is corrected
6. Given the terminology correction feature, when enabled, then processing time per subtitle file does not exceed 30 seconds total (OpenCC + Claude combined)

## Tasks / Subtasks

- [ ] Task 1: Create terminology correction prompt (AC: #5)
  - [ ] 1.1 Create `apps/api/internal/ai/prompts/terminology_corrector.go` with system prompt and user prompt template
  - [ ] 1.2 Prompt must instruct Claude to: only fix cross-strait terms, preserve proper nouns, return corrected text in same format
  - [ ] 1.3 Add prompt unit tests with known 簡繁 edge cases (at least 10 test pairs)

- [ ] Task 2: Create TerminologyCorrectionService (AC: #1, #2, #4)
  - [ ] 2.1 Create `apps/api/internal/services/terminology_service.go`
  - [ ] 2.2 Interface: `TerminologyCorrectionServiceInterface` with `Correct(ctx, subtitleContent string) (string, error)`
  - [ ] 2.3 Use existing `ai.Provider` (via factory) — do NOT create new Claude client
  - [ ] 2.4 Implement `IsConfigured() bool` — delegates to `config.HasClaudeKey()`
  - [ ] 2.5 Add 30-second context timeout (AC: #6)
  - [ ] 2.6 On error/timeout: return original content unchanged + log warning (AC: #4)

- [ ] Task 3: Integrate into subtitle pipeline (AC: #1, #2)
  - [ ] 3.1 Modify `apps/api/internal/subtitle/converter.go` — add optional post-OpenCC AI correction step
  - [ ] 3.2 Pipeline flow: Download → OpenCC s2twp → (if AI configured) Claude correction → Place
  - [ ] 3.3 Skip AI step entirely if `!terminologyService.IsConfigured()` (AC: #2)

- [ ] Task 4: Unit tests (AC: #1-6)
  - [ ] 4.1 Test terminology service with mock AI provider (success, timeout, error cases)
  - [ ] 4.2 Test pipeline integration: verify AI step is skipped when not configured
  - [ ] 4.3 Test known edge cases: 軟件→軟體, 內存→記憶體, 數據→資料, 視頻→影片
  - [ ] 4.4 Test proper noun preservation: 人名, 電影名 should not be altered

## Dev Notes

### Architecture Compliance

- **AI Provider:** MUST reuse existing `ai.Provider` interface and factory (`apps/api/internal/ai/`). Do NOT create new HTTP clients for Claude
- **Prompt pattern:** Follow `apps/api/internal/ai/prompts/fansub_parser.go` structure — system prompt + user prompt template
- **Error handling:** Follow existing `ai.ErrAITimeout`, `ai.ErrAIQuotaExceeded` patterns
- **Caching:** Consider caching corrections by content hash (30-day TTL, same as parser cache). Subtitle content is relatively static
- **Service registration:** Wire into `main.go` dependency injection, gated by `config.HasClaudeKey()`

### Project Structure Notes

- New files:
  - `apps/api/internal/ai/prompts/terminology_corrector.go`
  - `apps/api/internal/services/terminology_service.go`
  - `apps/api/internal/services/terminology_service_test.go`
- Modified files:
  - `apps/api/internal/subtitle/converter.go` (add AI correction step)
  - `apps/api/internal/subtitle/converter_test.go`
  - `apps/api/cmd/main.go` (wire service)

### Key Implementation Details

- **Claude model:** Use `claude-3-5-haiku-latest` (already configured as default in `ai/claude.go`) — fast and cost-effective for terminology correction
- **Max tokens:** ~2000 for subtitle correction (subtitle files are typically <5KB text)
- **Chunking strategy:** If subtitle content exceeds 4000 chars, split by subtitle blocks (separated by blank lines), process in parallel, reassemble
- **CN subtitle policy:** Per project memory, 大陸影片保留簡體字幕不轉換. The terminology service should respect this — check `production_countries` before applying correction

### References

- [Source: apps/api/internal/ai/claude.go] — Claude provider implementation
- [Source: apps/api/internal/ai/prompts/fansub_parser.go] — Prompt template pattern
- [Source: apps/api/internal/subtitle/converter.go] — OpenCC conversion pipeline
- [Source: apps/api/internal/services/ai_service.go] — Service layer pattern
- [Source: _bmad-output/planning-artifacts/prd/functional-requirements.md#3.3] — P1-020 spec
- [Source: _bmad-output/planning-artifacts/epics/epic-9-ai-subtitle-enhancement.md] — Epic context

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
