# Story 9.2b: AI Subtitle Translation

Status: review

## Story

As a Traditional Chinese NAS user,
I want Whisper-generated English SRT files to be translated to Traditional Chinese using Claude API,
so that I can watch content with no existing Chinese subtitles in my native language.

## Acceptance Criteria

1. Given a Whisper-generated English SRT file, when translation is triggered, then Claude API translates each subtitle block to Traditional Chinese and outputs a `.zh-Hant.srt` file
2. Given translation is running, when processing subtitle blocks, then Claude maintains context across adjacent blocks for consistent terminology and natural flow
3. Given a completed translation, when the `.zh-Hant.srt` is saved, then timestamps from the original English SRT are preserved exactly
4. Given no Claude API key configured, when translation is triggered, then the feature is disabled and only the English SRT remains
5. Given Claude API fails mid-translation, when an error occurs, then partially translated content is saved with a warning, and untranslated blocks retain English text
6. Given the translation feature, when processing, then progress is reported via SSE events per batch of subtitle blocks

## Tasks / Subtasks

- [x] Task 1: Translation prompt (AC: #2)
  - [x] 1.1 Create `apps/api/internal/ai/prompts/subtitle_translator.go`
  - [x] 1.2 System prompt: translate English dialogue to natural Traditional Chinese (Taiwan usage), preserve speaker tone, keep proper nouns in original
  - [x] 1.3 Context window: send 5 previous blocks as context for each batch to maintain consistency (AC: #2)
  - [x] 1.4 Unit tests with sample SRT dialogue

- [x] Task 2: SRT parser utility (AC: #3)
  - [x] 2.1 Create `apps/api/internal/subtitle/srt_parser.go` — parse SRT into `[]SubtitleBlock{Index, Start, End, Text}`
  - [x] 2.2 Serialize back to SRT format preserving exact timestamps (AC: #3)
  - [x] 2.3 Unit tests with edge cases: multi-line blocks, HTML tags, empty lines

- [x] Task 3: Translation service (AC: #1, #4, #5, #6)
  - [x] 3.1 Create `apps/api/internal/services/translation_service.go`
  - [x] 3.2 Batch processing: send 10 subtitle blocks per Claude request (balance cost vs context)
  - [x] 3.3 Use existing `ai.Provider` — reuse factory, do NOT create new client
  - [x] 3.4 Graceful degradation: on error, keep English text for failed blocks (AC: #5)
  - [x] 3.5 SSE progress: `translation_progress` with percentage (AC: #6)
  - [x] 3.6 Output: `{media_dir}/{filename}.zh-Hant.srt`
  - [x] 3.7 Skip if `!config.HasClaudeKey()` (AC: #4)

- [x] Task 4: Wire into transcription pipeline (AC: #1)
  - [x] 4.1 Extend `POST /api/v1/movies/:id/transcribe` — add optional `translate=true` query param
  - [x] 4.2 Pipeline: Extract audio → Whisper → English SRT → (if translate) Claude → zh-Hant SRT
  - [x] 4.3 Both English and zh-Hant SRT are kept (user may want both)

- [x] Task 5: Tests (AC: #1-6)
  - [x] 5.1 Translation service: mock Claude, verify batch processing and context passing
  - [x] 5.2 SRT parser round-trip: parse → serialize → compare
  - [x] 5.3 Partial failure: verify English fallback for failed blocks
  - [x] 5.4 API integration: test translate=true parameter

## Dev Notes

### Architecture Compliance

- **AI Provider:** MUST reuse `ai.Provider` — same as Story 9-1. No new HTTP clients
- **Prompt pattern:** Follow `apps/api/internal/ai/prompts/` conventions
- **SRT parser:** Keep in `subtitle/` package — it's subtitle domain logic
- **Depends on Story 9-2a:** Needs the Whisper transcription output (English SRT) as input

### Project Structure Notes

- New files:
  - `apps/api/internal/ai/prompts/subtitle_translator.go`
  - `apps/api/internal/subtitle/srt_parser.go`
  - `apps/api/internal/services/translation_service.go`
  - Tests for each
- Modified files:
  - `apps/api/internal/services/transcription_service.go` (add translation step)
  - `apps/api/internal/handlers/transcription_handler.go` (add translate param)

### Key Implementation Details

- **Batching:** 10 blocks per Claude request. Each request includes 5 previous blocks as context (read-only, not re-translated). This balances API cost with translation quality
- **Claude model:** Use `claude-3-5-haiku-latest` — fast, cheap, sufficient for translation
- **Token budget:** ~4000 tokens per request (10 blocks × ~100 chars + context + prompt)
- **Output parsing:** Claude returns translated text in same block order. Parse by matching block indices
- **File naming:** Follow existing convention from `subtitle/placer.go` — `.zh-Hant.srt` extension for Plex/Jellyfin recognition

### References

- [Source: apps/api/internal/ai/prompts/fansub_parser.go] — Prompt pattern
- [Source: apps/api/internal/subtitle/placer.go] — File placement and naming conventions
- [Source: apps/api/internal/subtitle/converter.go] — Subtitle processing pipeline pattern
- [Source: _bmad-output/planning-artifacts/epics/epic-9-ai-subtitle-enhancement.md] — C-2b spec

## Dev Agent Record

### Agent Model Used
Claude Opus 4.6 (1M context)

### Debug Log References

### Completion Notes List
- Task 1: Created subtitle_translator.go with system prompt for EN→zh-Hant translation. Constants: ContextWindow=5, BatchSize=10. Prompt uses block index format [N] for output parsing. 6 unit tests passing.
- Task 2: Created srt_parser.go with ParseSRT/SerializeSRT. Handles BOM, Windows line endings, multi-line blocks, HTML tags, extra blank lines. Round-trip preserves timestamps exactly (AC #3). 12 unit tests passing.
- Task 3: Created translation_service.go with batch processing (10 blocks/batch), context window (5 blocks), partial failure fallback (AC #5), progress callback (AC #6), cancellation support. Uses TranslationBlock type (avoids circular import with subtitle pkg). 10 unit tests passing.
- Task 4: Wired translation into transcription pipeline. Added `translate=true` query param to handler, TranscriptionOption pattern for backward compat, SetTranslationService setter, translateSRT method with inline SRT parsing (avoids circular dep), SSE progress events. Shared ClaudeProvider between terminology+translation services. All 9 existing handler tests pass.
- Task 5: Added handler tests for translate=true param (2 tests). Full regression: 40 story-related tests pass. Pre-existing failures filed: setup_service_test.go (panic), download_handler_test.go (4 tests).
- 🎨 UX Verification: SKIPPED — no UI changes in this story

### File List
- apps/api/internal/ai/prompts/subtitle_translator.go (new)
- apps/api/internal/ai/prompts/subtitle_translator_test.go (new)
- apps/api/internal/subtitle/srt_parser.go (new)
- apps/api/internal/subtitle/srt_parser_test.go (new)
- apps/api/internal/services/translation_service.go (new)
- apps/api/internal/services/translation_service_test.go (new)
- apps/api/internal/services/transcription_service.go (modified)
- apps/api/internal/handlers/transcription_handler.go (modified)
- apps/api/internal/handlers/transcription_handler_test.go (modified)
- apps/api/cmd/api/main.go (modified)
- _bmad-output/implementation-artifacts/sprint-status.yaml (modified)

### Change Log
- 2026-04-08: Task 1 — Translation prompt with system prompt, context window, batch constants, and 6 unit tests
- 2026-04-08: Task 2 — SRT parser utility with parse/serialize, edge case handling, 12 unit tests
- 2026-04-08: Task 3 — Translation service with batch processing, context passing, partial failure fallback, 10 unit tests
- 2026-04-08: Task 4 — Pipeline integration: translate=true query param, TranscriptionOption, SetTranslationService, main.go wiring
- 2026-04-08: Task 5 — Handler integration tests for translate param + regression verification. Pre-existing failures tracked.
