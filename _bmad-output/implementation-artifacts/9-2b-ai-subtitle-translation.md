# Story 9.2b: AI Subtitle Translation

Status: ready-for-dev

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

- [ ] Task 1: Translation prompt (AC: #2)
  - [ ] 1.1 Create `apps/api/internal/ai/prompts/subtitle_translator.go`
  - [ ] 1.2 System prompt: translate English dialogue to natural Traditional Chinese (Taiwan usage), preserve speaker tone, keep proper nouns in original
  - [ ] 1.3 Context window: send 5 previous blocks as context for each batch to maintain consistency (AC: #2)
  - [ ] 1.4 Unit tests with sample SRT dialogue

- [ ] Task 2: SRT parser utility (AC: #3)
  - [ ] 2.1 Create `apps/api/internal/subtitle/srt_parser.go` — parse SRT into `[]SubtitleBlock{Index, Start, End, Text}`
  - [ ] 2.2 Serialize back to SRT format preserving exact timestamps (AC: #3)
  - [ ] 2.3 Unit tests with edge cases: multi-line blocks, HTML tags, empty lines

- [ ] Task 3: Translation service (AC: #1, #4, #5, #6)
  - [ ] 3.1 Create `apps/api/internal/services/translation_service.go`
  - [ ] 3.2 Batch processing: send 10 subtitle blocks per Claude request (balance cost vs context)
  - [ ] 3.3 Use existing `ai.Provider` — reuse factory, do NOT create new client
  - [ ] 3.4 Graceful degradation: on error, keep English text for failed blocks (AC: #5)
  - [ ] 3.5 SSE progress: `translation_progress` with percentage (AC: #6)
  - [ ] 3.6 Output: `{media_dir}/{filename}.zh-Hant.srt`
  - [ ] 3.7 Skip if `!config.HasClaudeKey()` (AC: #4)

- [ ] Task 4: Wire into transcription pipeline (AC: #1)
  - [ ] 4.1 Extend `POST /api/v1/movies/:id/transcribe` — add optional `translate=true` query param
  - [ ] 4.2 Pipeline: Extract audio → Whisper → English SRT → (if translate) Claude → zh-Hant SRT
  - [ ] 4.3 Both English and zh-Hant SRT are kept (user may want both)

- [ ] Task 5: Tests (AC: #1-6)
  - [ ] 5.1 Translation service: mock Claude, verify batch processing and context passing
  - [ ] 5.2 SRT parser round-trip: parse → serialize → compare
  - [ ] 5.3 Partial failure: verify English fallback for failed blocks
  - [ ] 5.4 API integration: test translate=true parameter

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

### Debug Log References

### Completion Notes List

### File List
