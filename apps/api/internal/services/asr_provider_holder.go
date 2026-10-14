package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/shopspring/decimal"

	"github.com/vido/api/internal/ai"
)

// ASR provider ids (asr-scribe-b). These are what the API reports as
// `asr_provider` and what the operator's preference names.
const (
	// ASRProviderOpenAI is the hosted OpenAI Whisper API.
	ASRProviderOpenAI = "openai"
	// ASRProviderElevenLabs is ElevenLabs Scribe v2.
	ASRProviderElevenLabs = "elevenlabs"
	// ASRProviderSelfHosted is an OpenAI-compatible engine at ASR_BASE_URL —
	// Whisper's wire format, the operator's own hardware, $0 per minute.
	ASRProviderSelfHosted = "self_hosted"
	// ASRPreferenceAuto is the default preference: ElevenLabs when its key
	// resolves, otherwise OpenAI (or the self-hosted engine).
	ASRPreferenceAuto = "auto"
)

// ASRPreferences are the values the operator may choose from.
var ASRPreferences = []string{ASRPreferenceAuto, ASRProviderElevenLabs, ASRProviderOpenAI}

// ValidASRPreference reports whether p is one of ASRPreferences.
func ValidASRPreference(p string) bool {
	for _, v := range ASRPreferences {
		if v == p {
			return true
		}
	}
	return false
}

// ASRProviderHolder owns the speech-recognition client and rebuilds it when the
// resolved key, the operator's provider preference, or the endpoint/model
// identity changes.
//
// [@contract-v1] — consumed by TranscriptionService (sub-5-2 AC #2) and wired in
// main.go. It is the ASR twin of ClaudeProviderHolder (sub-2-1a AC #2), which
// fixed exactly this class for translation: the client used to be built ONCE at
// startup from an env-var, inside an `if cfg.HasOpenAIKey()` guard, so a key
// saved later in /settings/keys reached a client that did not exist — the
// settings page said 「已設定」 while every transcribe stayed 503 until the
// container was restarted.
//
// asr-scribe-b: the holder now chooses BETWEEN engines. On the 2026-10 exam
// (eval/asr-exam-2026-10) ElevenLabs Scribe v2 passed 3/3 and whisper-1 0/3,
// so when an ElevenLabs key resolves it is the default; an OpenAI key alone
// still means Whisper, a self-hosted ASR_BASE_URL still means that engine, and
// the operator can pin either cloud engine through the preference (settings
// page, or VIDO_ASR_PROVIDER). Nothing a deployment without an ElevenLabs key
// does has changed.
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
	scribe   []ai.ScribeOption
	// preference returns the operator's choice (one of ASRPreferences) per
	// call, so a save on the settings page applies to the next transcribe
	// without a restart. nil = auto.
	preference func(ctx context.Context) string
	logger     *slog.Logger

	mu sync.Mutex
	// cached is the live client; fingerprint is the "provider|key|baseURL|model"
	// identity it was built from (the ClaudeProviderHolder / download_service.go
	// pattern).
	cached      ai.ASRProvider
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

// SetScribeOptions captures the options every ElevenLabs client this holder
// builds is given (the Governor, a test base URL). Captured once, replayed on
// rebuild, like the Whisper options.
func (h *ASRProviderHolder) SetScribeOptions(opts ...ai.ScribeOption) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.scribe = opts
	h.fingerprint = ""
}

// SetPreferenceSource installs the operator's provider preference reader.
// A reader that returns "" or an unknown value is read as auto.
func (h *ASRProviderHolder) SetPreferenceSource(fn func(ctx context.Context) string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.preference = fn
	h.fingerprint = ""
}

// selfHosted routes through ai.IsSelfHostedASRBaseURL — the ONE self-hosted
// predicate (sub-5-1 CR M1). A second, locally-invented detector ("base URL is
// non-empty") disagrees precisely when ASR_BASE_URL is explicitly set to the
// official endpoint, which here would mean handing out a keyless client that
// 401s on every call.
func (h *ASRProviderHolder) selfHosted() bool {
	return ai.IsSelfHostedASRBaseURL(h.baseURL)
}

func (h *ASRProviderHolder) preferred(ctx context.Context) string {
	if h.preference == nil {
		return ASRPreferenceAuto
	}
	p := strings.TrimSpace(strings.ToLower(h.preference(ctx)))
	if !ValidASRPreference(p) {
		return ASRPreferenceAuto
	}
	return p
}

// choice is one resolved decision: which engine, with which key.
type asrChoice struct {
	provider string
	key      string
	source   KeySource
}

// choose applies the rule (asr-scribe-b):
//
//	self-hosted ASR_BASE_URL             → that engine, no key needed
//	preference elevenlabs                → Scribe (needs its key)
//	preference openai                    → Whisper (needs its key)
//	auto: ElevenLabs key → Scribe; else OpenAI key → Whisper; else nothing
//
// A pinned engine without its key is UNCONFIGURED, not a silent fallback to
// the other engine: an operator who pinned Whisper and then cleared its key
// must see 尚未設定, not be billed by ElevenLabs behind their back.
func (h *ASRProviderHolder) choose(ctx context.Context) (asrChoice, error) {
	if h.selfHosted() {
		key, source, err := h.resolver.Get(ctx, KeyOpenAI)
		if err != nil {
			return asrChoice{}, fmt.Errorf("resolve openai key: %w", err)
		}
		return asrChoice{provider: ASRProviderSelfHosted, key: key, source: source}, nil
	}
	eleven, elevenSource, err := h.resolver.Get(ctx, KeyElevenLabs)
	if err != nil {
		return asrChoice{}, fmt.Errorf("resolve elevenlabs key: %w", err)
	}
	openai, openaiSource, err := h.resolver.Get(ctx, KeyOpenAI)
	if err != nil {
		return asrChoice{}, fmt.Errorf("resolve openai key: %w", err)
	}
	switch h.preferred(ctx) {
	case ASRProviderElevenLabs:
		return asrChoice{provider: ASRProviderElevenLabs, key: eleven, source: elevenSource}, nil
	case ASRProviderOpenAI:
		return asrChoice{provider: ASRProviderOpenAI, key: openai, source: openaiSource}, nil
	}
	if strings.TrimSpace(eleven) != "" {
		return asrChoice{provider: ASRProviderElevenLabs, key: eleven, source: elevenSource}, nil
	}
	return asrChoice{provider: ASRProviderOpenAI, key: openai, source: openaiSource}, nil
}

// Provider reports which engine the next transcribe would use: one of the
// ASRProvider* ids, or "" when nothing is configured. This is what the
// candidate sweep and the single-item estimate stamp on their answers so the
// consent copy can name the company that will be billed.
func (h *ASRProviderHolder) Provider(ctx context.Context) string {
	c, err := h.choose(ctx)
	if err != nil {
		return ""
	}
	if c.provider != ASRProviderSelfHosted && strings.TrimSpace(c.key) == "" {
		return ""
	}
	return c.provider
}

// ASRRate returns the per-audio-minute price the next transcribe will be
// metered at, and the provider it belongs to — the ONE rate source both the
// estimators and the metering path read (sub-5-1 費率同源), now per call
// because the engine can change at runtime.
func (h *ASRProviderHolder) ASRRate(ctx context.Context) (decimal.Decimal, string) {
	provider := h.Provider(ctx)
	return ai.ASRPerMinute(provider), provider
}

// EndpointFingerprint names the engine — NEVER the key. It is key material for
// the ASR chunk store (disc-2026-09-generation-resume-b): a stored transcript
// is only reused by the same engine, while a rotated key must not invalidate
// paid-for work. The private `fingerprint` below carries the key and must
// never be used for this.
//
// Whisper (hosted or self-hosted) keeps the historical `baseURL|model` form so
// every chunk cached before asr-scribe-b is still found; Scribe names itself
// `elevenlabs|<model>`, so a Whisper transcript is never served as Scribe's.
//
// Derived from the CURRENT choice every time, never from the cached client:
// the chunk scope is built at run start, before the first Get of that run,
// and a holder that still remembered yesterday's Whisper would file today's
// Scribe chunks under Whisper's key (CR H1).
func (h *ASRProviderHolder) EndpointFingerprint() string {
	if h.Provider(context.Background()) == ASRProviderElevenLabs {
		return ai.ScribeProvider + "|" + ai.ScribeModel
	}
	model := h.model
	if model == "" {
		model = ai.WhisperModel
	}
	return h.baseURL + "|" + model
}

// Get returns the current client, rebuilding only when the resolved key, the
// preference, or the endpoint/model identity differs from the cached one.
// Returns ai.ErrWhisperNotConfigured when the deployment has no usable ASR —
// consumers already classify on that sentinel.
func (h *ASRProviderHolder) Get(ctx context.Context) (ai.ASRProvider, error) {
	c, err := h.choose(ctx)
	if err != nil {
		return nil, err
	}

	// sub-5-2 AC #3: a self-hosted OpenAI-compatible engine authenticates
	// nothing, so "no key" is only an unconfigured state on the PAID APIs.
	if strings.TrimSpace(c.key) == "" && c.provider != ASRProviderSelfHosted {
		// Still the shipped sentinel (consumers errors.Is on it), but the
		// text names the engine whose key is missing — an operator pinned to
		// Scribe must not be told to go find an OpenAI key.
		return nil, fmt.Errorf("%w (engine %s: no %s key)", ai.ErrWhisperNotConfigured, c.provider, c.provider)
	}

	fingerprint := c.provider + "|" + c.key + "|" + h.baseURL + "|" + h.model

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.cached != nil && h.fingerprint == fingerprint {
		return h.cached, nil
	}

	switch c.provider {
	case ASRProviderElevenLabs:
		h.cached = ai.NewScribeClient(c.key, append([]ai.ScribeOption(nil), h.scribe...)...)
	default:
		opts := append([]ai.WhisperOption(nil), h.opts...)
		if h.baseURL != "" {
			opts = append(opts, ai.WithWhisperBaseURL(h.baseURL))
		}
		if h.model != "" {
			opts = append(opts, ai.WithWhisperModel(h.model))
		}
		h.cached = ai.NewWhisperClient(c.key, opts...)
	}
	h.fingerprint = fingerprint

	// Deliberately logs neither the key nor any prefix of it (NFR-S1).
	h.logger.Info("asr provider (re)built",
		"provider", c.provider,
		"key_source", c.source,
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
	return h.Provider(ctx) != ""
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

// TranscribeDetailed forwards the 9R-5 hallucination-filter seam to the live
// client. Without this the holder — which is what the pipeline actually holds —
// would fail the ai.DetailedTranscriber type assertion, and the whole-file
// empty-result guard would silently never engage.
//
// A resolved provider that is NOT a DetailedTranscriber (a future alternative
// engine) degrades to the plain SRT with Filtered=false, exactly as a direct
// caller would.
func (h *ASRProviderHolder) TranscribeDetailed(ctx context.Context, audioPath, lang string) (ai.TranscriptionDetail, error) {
	provider, err := h.Get(ctx)
	if err != nil {
		return ai.TranscriptionDetail{}, err
	}
	if detailed, ok := provider.(ai.DetailedTranscriber); ok {
		return detailed.TranscribeDetailed(ctx, audioPath, lang)
	}
	srt, err := provider.TranscribeWithLanguage(ctx, audioPath, lang)
	if err != nil {
		return ai.TranscriptionDetail{}, err
	}
	return ai.TranscriptionDetail{SRT: srt, Unfiltered: srt}, nil
}

// Compile-time proof the holder carries the detailed seam through.
var _ ai.DetailedTranscriber = (*ASRProviderHolder)(nil)
