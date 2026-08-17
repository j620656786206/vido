package services

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/vido/api/internal/ai"
)

// ASRProviderHolder owns the speech-recognition client and rebuilds it when the
// resolved key (or endpoint/model identity) changes.
//
// [@contract-v1] — consumed by TranscriptionService (sub-5-2 AC #2) and wired in
// main.go. It is the ASR twin of ClaudeProviderHolder (sub-2-1a AC #2), which
// fixed exactly this class for translation: the client used to be built ONCE at
// startup from an env-var, inside an `if cfg.HasOpenAIKey()` guard, so a key
// saved later in /settings/keys reached a client that did not exist — the
// settings page said 「已設定」 while every transcribe stayed 503 until the
// container was restarted.
//
// NAMING: the backlog entry called for a "WhisperProviderHolder", but what it
// holds is ai.ASRProvider — an engine-agnostic port since 9R-9, satisfied by
// self-hosted OpenAI-compatible engines just as much as by the hosted API. The
// holder is named for the port, not for one implementation of it.
//
// ⚖️ NOT A GENERIC HOLDER, DELIBERATELY (sub-5-2 AC #6). With this file there
// are TWO concrete holders (Claude, ASR) that share the fingerprint-cache /
// options-replay / per-call-resolve shape. Extracting a `providerHolder[T]` at
// the second implementation is premature generalisation (ADR
// adr-external-api-integration-standard Decision 3 — wait for the third
// consumer). TMDb is still only a backlog entry; the extraction decision is
// therefore delegated to `backlog-tmdb-runtime-key-resolution`, which will be
// the third. Do not re-litigate it here.
type ASRProviderHolder struct {
	resolver KeyResolver
	baseURL  string
	model    string
	opts     []ai.WhisperOption
	logger   *slog.Logger

	mu sync.Mutex
	// cached is the live client; fingerprint is the "key|baseURL|model" identity
	// it was built from (the ClaudeProviderHolder / download_service.go pattern).
	cached      *ai.WhisperClient
	fingerprint string
}

// Compile-time proof the holder can stand in for the client everywhere the
// transcription pipeline expects a provider.
var _ ai.ASRProvider = (*ASRProviderHolder)(nil)

// NewASRProviderHolder builds the holder. `baseURL` and `model` may be empty
// (the ai package's hosted defaults apply); logger may be nil. `opts` are
// captured ONCE and replayed on every rebuild — which is how the shared Governor
// survives a key change instead of the run budget silently resetting.
func NewASRProviderHolder(resolver KeyResolver, baseURL, model string, logger *slog.Logger, opts ...ai.WhisperOption) *ASRProviderHolder {
	if logger == nil {
		logger = slog.Default()
	}
	return &ASRProviderHolder{
		resolver: resolver,
		baseURL:  baseURL,
		model:    model,
		opts:     opts,
		logger:   logger.With("component", "asr_provider_holder"),
	}
}

// selfHosted routes through ai.IsSelfHostedASRBaseURL — the ONE self-hosted
// predicate (sub-5-1 CR M1). A second, locally-invented detector ("base URL is
// non-empty") disagrees precisely when ASR_BASE_URL is explicitly set to the
// official endpoint, which here would mean handing out a keyless client that
// 401s on every call.
func (h *ASRProviderHolder) selfHosted() bool {
	return ai.IsSelfHostedASRBaseURL(h.baseURL)
}

// Get returns the current client, rebuilding only when the resolved key or the
// endpoint/model identity differs from the cached one. Returns
// ai.ErrWhisperNotConfigured when the deployment has no usable ASR — consumers
// already classify on that sentinel.
func (h *ASRProviderHolder) Get(ctx context.Context) (ai.ASRProvider, error) {
	key, source, err := h.resolver.Get(ctx, KeyOpenAI)
	if err != nil {
		return nil, fmt.Errorf("resolve openai key: %w", err)
	}

	// sub-5-2 AC #3: a self-hosted OpenAI-compatible engine authenticates
	// nothing, so "no key" is only an unconfigured state on the PAID hosted API.
	if key == "" && !h.selfHosted() {
		return nil, ai.ErrWhisperNotConfigured
	}

	fingerprint := key + "|" + h.baseURL + "|" + h.model

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.cached != nil && h.fingerprint == fingerprint {
		return h.cached, nil
	}

	opts := append([]ai.WhisperOption(nil), h.opts...)
	if h.baseURL != "" {
		opts = append(opts, ai.WithWhisperBaseURL(h.baseURL))
	}
	if h.model != "" {
		opts = append(opts, ai.WithWhisperModel(h.model))
	}
	h.cached = ai.NewWhisperClient(key, opts...)
	h.fingerprint = fingerprint

	// Deliberately logs neither the key nor any prefix of it (NFR-S1).
	h.logger.Info("asr provider (re)built",
		"key_source", source,
		"self_hosted", h.selfHosted(),
		"base_url_override", h.baseURL != "",
		"model_override", h.model != "")
	return h.cached, nil
}

// IsConfigured reports whether speech recognition can run at all. This is what
// TranscriptionService.IsAvailable probes, replacing the startup-only
// cfg.HasOpenAIKey — and, per AC #3, a self-hosted endpoint counts as configured
// with no key.
func (h *ASRProviderHolder) IsConfigured(ctx context.Context) bool {
	return h.selfHosted() || h.resolver.Has(ctx, KeyOpenAI)
}

// Transcribe implements ai.ASRProvider by resolving per call.
func (h *ASRProviderHolder) Transcribe(ctx context.Context, audioPath string) (string, error) {
	provider, err := h.Get(ctx)
	if err != nil {
		return "", err
	}
	return provider.Transcribe(ctx, audioPath)
}

// TranscribeWithLanguage implements ai.ASRProvider by resolving per call.
func (h *ASRProviderHolder) TranscribeWithLanguage(ctx context.Context, audioPath, lang string) (string, error) {
	provider, err := h.Get(ctx)
	if err != nil {
		return "", err
	}
	return provider.TranscribeWithLanguage(ctx, audioPath, lang)
}
