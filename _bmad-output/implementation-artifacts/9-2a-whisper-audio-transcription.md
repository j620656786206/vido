# Story 9.2a: Whisper Audio Transcription

Status: ready-for-dev

## Story

As a Traditional Chinese NAS user with media that has no Chinese subtitles,
I want the system to extract English audio tracks from MKV files and transcribe them using Whisper,
so that I have a raw English SRT file that can later be translated to Traditional Chinese.

## Acceptance Criteria

1. Given an MKV file with an English audio track, when Whisper transcription is triggered, then the system extracts the audio track using FFmpeg and produces a timestamped SRT file
2. Given an MKV file with multiple audio tracks, when extracting audio, then the system selects the English track (by language tag) or falls back to the first audio track
3. Given the Whisper transcription completes, when the SRT is generated, then timestamps are accurate within ±2 seconds of actual dialogue
4. Given FFmpeg or Whisper is not available in the environment, when transcription is triggered, then the system returns a clear error and does not crash
5. Given a Whisper API key is configured (OpenAI), when transcription runs, then it uses the cloud Whisper API; if not configured, the feature is disabled
6. Given the transcription process, when running, then progress is reported via SSE events (extracting audio → transcribing → complete)
7. Given an audio file exceeding 25MB (Whisper API limit), when transcription is triggered, then the system splits the audio into chunks, transcribes each, and merges SRT output with correct timestamp offsets

## Tasks / Subtasks

- [ ] Task 1: Audio extraction service (AC: #1, #2, #4)
  - [ ] 1.1 Create `apps/api/internal/services/audio_extractor_service.go`
  - [ ] 1.2 Use `os/exec` to call `ffmpeg -i input.mkv -vn -acodec pcm_s16le -ar 16000 -ac 1 output.wav`
  - [ ] 1.3 Track selection: parse `ffprobe -show_streams` output, find stream with `language=eng`, fall back to first audio stream (AC: #2)
  - [ ] 1.4 Graceful degradation: check `exec.LookPath("ffmpeg")` on startup (AC: #4)
  - [ ] 1.5 Temp file management: extract to OS temp dir, cleanup after transcription

- [ ] Task 2: Whisper API client (AC: #5, #7)
  - [ ] 2.1 Create `apps/api/internal/ai/whisper.go` — OpenAI Whisper API client
  - [ ] 2.2 Endpoint: `POST https://api.openai.com/v1/audio/transcriptions`
  - [ ] 2.3 Model: `whisper-1`, response_format: `srt`
  - [ ] 2.4 Auth: `Authorization: Bearer {OPENAI_API_KEY}` from config
  - [ ] 2.5 Add `OPENAI_API_KEY` to `config/api_keys.go`: `HasOpenAIKey()`, `GetOpenAIAPIKey()`
  - [ ] 2.6 Implement audio chunking for files >25MB: split WAV by duration (10-minute chunks), transcribe each, merge SRT with timestamp offset correction (AC: #7)

- [ ] Task 3: Transcription orchestrator service (AC: #1, #6)
  - [ ] 3.1 Create `apps/api/internal/services/transcription_service.go`
  - [ ] 3.2 Pipeline: Extract audio → (optional chunk) → Whisper API → Merge SRT → Save
  - [ ] 3.3 SSE progress events: `transcription_extracting`, `transcription_progress`, `transcription_complete`, `transcription_failed` (AC: #6)
  - [ ] 3.4 Output SRT path: `{media_dir}/{filename}.en.srt` (English transcription)
  - [ ] 3.5 Timeout: 5 minutes per transcription (long audio files need time)

- [ ] Task 4: API endpoint (AC: #1)
  - [ ] 4.1 `POST /api/v1/movies/:id/transcribe` — trigger transcription for a movie
  - [ ] 4.2 Returns 202 Accepted with job ID (async operation)
  - [ ] 4.3 Validate: movie exists, has file_path, file is accessible
  - [ ] 4.4 Reject if transcription already in progress for this media

- [ ] Task 5: Unit & integration tests (AC: #1-7)
  - [ ] 5.1 Audio extractor: mock ffmpeg exec, test track selection logic
  - [ ] 5.2 Whisper client: mock HTTP server, test success/error/chunking
  - [ ] 5.3 Orchestrator: end-to-end mock test with SSE event verification
  - [ ] 5.4 API endpoint: handler tests with mock service

## Dev Notes

### Architecture Compliance

- **FFmpeg dependency:** Same approach as FFprobe in Epic 9c — `apk add --no-cache ffmpeg` in Docker. FFmpeg is already in the image if 9c is implemented first
- **Whisper API, NOT local model:** PRD explicitly states "AI features rely on external APIs (user-provided keys); no local Whisper inference". Use OpenAI cloud API only
- **Service pattern:** Follow existing `FFprobeService` pattern from ADR `adr-media-info-nfo-pipeline.md` — semaphore for concurrency, timeout, graceful degradation
- **SSE events:** Follow existing `scan_progress`, `enrich_progress` patterns in `scanner_service.go`
- **No new Go dependencies:** Use `net/http` for Whisper API, `os/exec` for FFmpeg. No SDK needed

### Project Structure Notes

- New files:
  - `apps/api/internal/services/audio_extractor_service.go`
  - `apps/api/internal/ai/whisper.go`
  - `apps/api/internal/services/transcription_service.go`
  - `apps/api/internal/handlers/transcription_handler.go`
  - Tests for each
- Modified files:
  - `apps/api/internal/config/api_keys.go` (add OpenAI key)
  - `apps/api/internal/config/config.go` (OPENAI_API_KEY env var)
  - `apps/api/cmd/main.go` (wire services and routes)

### Key Implementation Details

- **Audio format for Whisper:** WAV 16kHz mono PCM — best compatibility with Whisper API
- **Chunking strategy:** For files >25MB, split by 10-minute segments using `ffmpeg -ss {start} -t 600`. Each chunk gets independent Whisper call. Merge SRTs by adding time offset to each subsequent chunk's timestamps
- **Temp file cleanup:** Use `defer os.Remove(tempFile)` pattern. Extract to `os.TempDir()`
- **Concurrency:** Max 1 transcription at a time (Whisper API has rate limits, and each job is resource-intensive). Use mutex or semaphore(1)

### References

- [Source: apps/api/internal/services/ffprobe_service.go] — FFprobe exec pattern (if 9c done)
- [Source: apps/api/internal/ai/claude.go] — AI API client pattern
- [Source: apps/api/internal/config/api_keys.go] — API key management
- [Source: _bmad-output/planning-artifacts/prd/functional-requirements.md#3.3] — P1-021 spec
- [Source: _bmad-output/planning-artifacts/epics/epic-9-ai-subtitle-enhancement.md] — C-2a spec

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
