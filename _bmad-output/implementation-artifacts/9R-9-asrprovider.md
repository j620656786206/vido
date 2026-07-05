# Story 9R-9 — `ASRProvider` interface + configurable engine/base-URL

Status: review

**Epic:** epic-9R-subtitle-route-c (Track 3 — Architecture, anti-lock-in) · **Owner:** dev (Amelia)
**Date:** 2026-07-05 · **Priority:** P1 · **Effort:** M · **Feasibility:** PROVEN

## Why

Decouple transcription from the OpenAI Whisper API so any OpenAI-compatible engine can be swapped
in via base-URL config — the anti-lock-in pillar (ADR Decision 2). The Whisper *model* is
MIT-open; only the OpenAI *API* carries lock-in/cost. vido already targets the OpenAI-standard
`/v1/audio/transcriptions` and had `WithWhisperBaseURL`, so this is a small, high-leverage
refactor.

## What shipped

- **`ASRProvider` interface** (`internal/ai/asr.go`): `Transcribe` + `TranscribeWithLanguage`.
  `*WhisperClient` is one implementation (compile-time `var _ ASRProvider = (*WhisperClient)(nil)`).
- **Configurable model** — `WithWhisperModel` + a `model` field defaulting to `WhisperModel`
  (`whisper-1`); the multipart now sends `c.model` so a self-hosted engine's id (e.g. Speaches
  `Systran/faster-whisper-small`) flows through.
- **Pipeline decoupled** — `TranscriptionService` holds an `ai.ASRProvider` (not `*ai.WhisperClient`);
  all transcribe calls go through the interface. No flow change; 9R-11 governor/budget metering
  and 9R-2/3 language/chunking are unaffected (they live in the Whisper impl).
- **Config** (`ASR_BASE_URL` / `ASR_MODEL`, mirroring `AI_PROVIDER`): empty base URL = OpenAI
  Whisper default; set them in `main.go` via `WithWhisperBaseURL` / `WithWhisperModel` to point at
  a self-hosted server.

## Verified OpenAI-compatible engines (doc note, AC #3)

All MIT / commercial-OK (LICENSE-verified 2026-06-16, ADR Decision 2), documented in `asr.go`:
- **Speaches** (faster-whisper-server) — CPU, `WHISPER__COMPUTE_TYPE=int8`.
- **WhisperLive** (Collabora) — OpenVINO backend for Intel CPU/iGPU (NAS-relevant; the 9R-S2 eval).
- **Subgen** — also exposes OpenAI-compatible `/v1/audio/{transcriptions,translations}`.

Swap = point `ASR_BASE_URL` at the server + set `ASR_MODEL` to its model id. No code change.

## Acceptance Criteria

1. ✅ `ASRProvider` interface (`Transcribe(audio) → SRT`); `whisper.go` is one impl (compile-time
   assertion + the pipeline consumes the interface).
2. ✅ Provider + base URL configurable (OpenAI / self-hosted OpenAI-compatible) via `ASR_BASE_URL`
   / `ASR_MODEL`, mirroring `AI_PROVIDER`.
3. ✅ Doc note on verified MIT engines (Speaches, WhisperLive-OpenVINO, Subgen) — in `asr.go` + here.
4. ✅ Smoke test against a configurable base URL (mock OpenAI-compatible server) — asserts the
   custom model id + endpoint path are sent and SRT returns, driven through the `ASRProvider`
   interface. Plus a default-model test + interface-satisfaction test.

## Dev Notes

- The **default engine (cloud OpenAI vs local)** decision is 9R-S2 (NAS benchmark, pending Alexyu's
  run) — 9R-9 provides the pluggability; S2 decides the default. `WhisperLive-OpenVINO` on the
  NAS iGPU is exactly what S2 measures.
- 9R-10's transcribe stage already calls through the interface now — a future non-Whisper impl
  drops in with zero pipeline change.

### Discovery Triage

- **N/A — no out-of-scope work discovered.** The interface is the clean seam 9R-10 + S2 need.

### References

- [Source: subtitle-route-c-stories-2026-06.md#9R-9] — ACs.
- [Source: architecture/adr-subtitle-route-c-generation.md#Decision-2] — pluggable ASR, MIT engines.
- [Source: 9R-S2-nas-whisper-benchmark-spike.md] — the default-engine decision this unblocks.

## Dev Agent Record

### Agent Model Used

claude-fable-5 (dev)

### Completion Notes List

- ASRProvider interface + configurable model landed; TranscriptionService decoupled from the
  concrete Whisper client. Base-URL + model swap via config enables self-hosted OpenAI-compatible
  engines (Speaches/WhisperLive/Subgen) with no code change. Full suite + staticcheck green.

### File List

- `apps/api/internal/ai/asr.go` (interface + engine doc note)
- `apps/api/internal/ai/whisper.go` (+ test) — model field + WithWhisperModel
- `apps/api/internal/services/transcription_service.go` — ASRProvider field/param
- `apps/api/internal/config/config.go` — ASR_BASE_URL / ASR_MODEL
- `apps/api/cmd/api/main.go` — wire base-URL + model
- `_bmad-output/implementation-artifacts/sprint-status.yaml`

## Change Log

| Date | Change |
|---|---|
| 2026-07-05 | 9R-9 implemented (dev): ASRProvider interface (*WhisperClient is one impl) + configurable model (WithWhisperModel, default whisper-1) + ASR_BASE_URL/ASR_MODEL config; TranscriptionService now holds ai.ASRProvider. Smoke test against a mock OpenAI-compatible server (custom model id + path sent). Verified MIT engines documented. Full suite + staticcheck green. Status → review. |
