package ai

import "context"

// ASRProvider transcribes an audio file to SRT (Story 9R-9). It decouples the
// transcription pipeline from the OpenAI Whisper API so any OpenAI-compatible
// engine can be swapped in via base-URL config — enabling self-hosted,
// anti-lock-in deployments.
//
// Verified MIT / commercial-OK OpenAI-compatible engines (2026-06-16, ADR
// adr-subtitle-route-c-generation Decision 2):
//   - Speaches (faster-whisper-server) — CPU, `WHISPER__COMPUTE_TYPE=int8`.
//   - WhisperLive (Collabora) — OpenVINO backend for Intel CPU/iGPU (NAS).
//   - Subgen — also exposes OpenAI-compatible /v1/audio/transcriptions.
//
// vido's *WhisperClient is one implementation; point ASR_BASE_URL at a
// self-hosted server and set ASR_MODEL to that engine's model id to swap.
type ASRProvider interface {
	// Transcribe uses the provider's default/pinned language hint.
	Transcribe(ctx context.Context, audioPath string) (string, error)
	// TranscribeWithLanguage passes an explicit ISO-639-1 language hint
	// ("" = auto-detect).
	TranscribeWithLanguage(ctx context.Context, audioPath, lang string) (string, error)
}

// Compile-time proof the Whisper client satisfies the interface.
var _ ASRProvider = (*WhisperClient)(nil)
