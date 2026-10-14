package services

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/vido/api/internal/ai"
)

// gemini-translation-dispatch: subtitle translation can run on Gemini.
//
// Why: on the golden-v2 exam (eval-gemini-free-tier-2026-10.md) the free
// gemini-3.5-flash-lite scored A (2.0% ruined, 89.5% natural) against
// Sonnet's 1.5% / 90.0%, and the FB testers who do not want to pay for Claude
// asked for it. Alexyu's ruling (2026-10-14): Gemini is a CHOICE, the default
// stays Claude, and the words are "your own free quota", never "free".

// DefaultGeminiTranslationModel is the Gemini model an install with ONLY a
// Gemini key translates with — the one the exam passed.
const DefaultGeminiTranslationModel = "gemini-3.5-flash-lite"

// DefaultGeminiMinInterval spaces Gemini calls for the free tier's
// requests-per-minute limit. Was 13 s (a first measurement that hit 429 at
// 5/min — most likely the daily quota of a different model, not the RPM);
// re-measured 2026-10-14 evening (disc-2026-10-gemini-free-tier-rpm-recheck):
// gemini-3.5-flash-lite took 15 requests at 4 s apart, 12 at 2 s apart and 20
// real 2,300-token translation prompts at 4 s apart without a single 429.
// 5 s = 12/min keeps headroom under the documented 15/min; an episode of 84
// batches now waits ~7 minutes instead of ~18. A paid-tier operator sets
// VIDO_GEMINI_MIN_INTERVAL=0.
const DefaultGeminiMinInterval = 5 * time.Second

// GeminiProviderHolder builds Gemini translation clients the way
// ClaudeProviderHolder builds Claude ones: the key is resolved per call, so
// a key saved on the settings page reaches the next batch without a restart,
// and clients are cached by key|model.
//
// The client is the OpenAI-compatible provider pointed at Google's endpoint
// (ai.GeminiOpenAIBaseURL): the same client the exam ran on, with billing on
// (catalog price) and the free-tier pacing.
type GeminiProviderHolder struct {
	resolver    KeyResolver
	minInterval time.Duration
	governor    *ai.Governor
	baseURL     string
	logger      *slog.Logger

	mu      sync.Mutex
	clients []geminiHolderClient
}

type geminiHolderClient struct {
	fingerprint string
	provider    *ai.OpenAICompatProvider
}

// NewGeminiProviderHolder builds the holder. minInterval <= 0 means no pacing.
func NewGeminiProviderHolder(resolver KeyResolver, minInterval time.Duration, governor *ai.Governor, logger *slog.Logger) *GeminiProviderHolder {
	if logger == nil {
		logger = slog.Default()
	}
	return &GeminiProviderHolder{
		resolver:    resolver,
		minInterval: minInterval,
		governor:    governor,
		baseURL:     ai.GeminiOpenAIBaseURL,
		logger:      logger.With("component", "gemini_provider_holder"),
	}
}

// SetBaseURL points the holder elsewhere (tests).
func (h *GeminiProviderHolder) SetBaseURL(u string) { h.baseURL = u }

// IsConfigured reports whether a Gemini key resolves.
func (h *GeminiProviderHolder) IsConfigured(ctx context.Context) bool {
	return h.resolver.Has(ctx, KeyGemini)
}

// MinInterval is the pacing gap every client this holder builds keeps.
func (h *GeminiProviderHolder) MinInterval() time.Duration { return h.minInterval }

// GetFor returns a client for a Gemini model. A model that is not Gemini's,
// or an empty one, is refused here — the dispatcher owns the default.
func (h *GeminiProviderHolder) GetFor(ctx context.Context, model string) (ai.TextCompleter, error) {
	if model == "" {
		model = DefaultGeminiTranslationModel
	}
	if !ai.IsSelectableModel(model) || ai.ProviderOf(model) != ai.ProviderNameGemini {
		return nil, fmt.Errorf("%w: model %q is not a Gemini model this build can run", ai.ErrAIModelNotFound, model)
	}
	key, source, err := h.resolver.Get(ctx, KeyGemini)
	if err != nil {
		return nil, fmt.Errorf("resolve gemini key: %w", err)
	}
	if key == "" {
		return nil, fmt.Errorf("%w: no Gemini API key configured", ai.ErrAINotConfigured)
	}
	fingerprint := key + "|" + model

	h.mu.Lock()
	defer h.mu.Unlock()
	for i, c := range h.clients {
		if c.fingerprint != fingerprint {
			continue
		}
		h.clients = append(h.clients[:i], h.clients[i+1:]...)
		h.clients = append([]geminiHolderClient{c}, h.clients...)
		return c.provider, nil
	}
	provider := ai.NewOpenAICompatProvider(h.baseURL, model,
		ai.WithOpenAICompatAPIKey(key),
		ai.WithOpenAICompatTimeout(ai.MaxRequestTimeout),
		ai.WithOpenAICompatGovernor(h.governor),
		ai.WithOpenAICompatBilling(true),
		ai.WithOpenAICompatMinInterval(h.minInterval))
	h.clients = append([]geminiHolderClient{{fingerprint: fingerprint, provider: provider}}, h.clients...)
	if len(h.clients) > maxCachedClients {
		h.clients = h.clients[:maxCachedClients]
	}
	// Never the key (NFR-S1).
	h.logger.Info("gemini provider (re)built",
		"key_source", source, "model", model, "min_interval", h.minInterval.String(), "cached_clients", len(h.clients))
	return provider, nil
}

// TranslationDispatcher is the one TextCompleter the translation stack holds.
// It routes each call to the holder that serves the model on the ctx
// (ai.WithModelID): Claude ids to the Claude holder, Gemini ids to the Gemini
// holder. With no model on the ctx it uses EffectiveModel.
//
// Rules (asr-scribe-b's twin, for translation):
//   - the default is Claude's default whenever a Claude key resolves;
//   - an install with ONLY a Gemini key defaults to DefaultGeminiTranslationModel
//     — "one key is enough to use it" holds for the Gemini-only tester too;
//   - a model whose provider has no key is refused BEFORE any paid call
//     (the catalog does not list it either, so the UI never offers it).
type TranslationDispatcher struct {
	claude *ClaudeProviderHolder
	gemini *GeminiProviderHolder
}

// Compile-time proof the dispatcher serves every seam the translation stack
// and the preview service read.
var (
	_ ai.TextCompleter    = (*TranslationDispatcher)(nil)
	_ ai.CachingCompleter = (*TranslationDispatcher)(nil)
)

// NewTranslationDispatcher wires the two holders. gemini may be nil (a build
// without Gemini support): every Gemini id is then refused.
func NewTranslationDispatcher(claude *ClaudeProviderHolder, gemini *GeminiProviderHolder) *TranslationDispatcher {
	return &TranslationDispatcher{claude: claude, gemini: gemini}
}

// EffectiveModel is the model an omitted model_id resolves to.
func (d *TranslationDispatcher) EffectiveModel() string {
	return d.effectiveModel(context.Background())
}

func (d *TranslationDispatcher) effectiveModel(ctx context.Context) string {
	if d.claude != nil && d.claude.IsConfigured(ctx) {
		return d.claude.EffectiveModel()
	}
	if d.gemini != nil && d.gemini.IsConfigured(ctx) {
		return DefaultGeminiTranslationModel
	}
	if d.claude != nil {
		return d.claude.EffectiveModel()
	}
	return ai.DefaultClaudeModel
}

// IsConfigured reports whether ANY translation provider has a key — the
// capability gate the subtitle pipeline reads.
func (d *TranslationDispatcher) IsConfigured(ctx context.Context) bool {
	return (d.claude != nil && d.claude.IsConfigured(ctx)) || (d.gemini != nil && d.gemini.IsConfigured(ctx))
}

// GetFor returns the completer for model ("" = EffectiveModel).
func (d *TranslationDispatcher) GetFor(ctx context.Context, model string) (ai.TextCompleter, error) {
	if model == "" {
		model = d.effectiveModel(ctx)
	}
	switch ai.ProviderOf(model) {
	case ai.ProviderNameGemini:
		if d.gemini == nil {
			return nil, fmt.Errorf("%w: model %q is served by gemini, which this build cannot dispatch to", ai.ErrAIModelNotFound, model)
		}
		return d.gemini.GetFor(ctx, model)
	default:
		if d.claude == nil {
			return nil, fmt.Errorf("%w: no translation provider wired", ai.ErrAINotConfigured)
		}
		return d.claude.GetFor(ctx, model)
	}
}

// Get returns the completer for the model on the ctx.
func (d *TranslationDispatcher) Get(ctx context.Context) (ai.TextCompleter, error) {
	return d.GetFor(ctx, ai.ModelIDFromContext(ctx))
}

// CompleteText implements ai.TextCompleter by dispatching per call.
func (d *TranslationDispatcher) CompleteText(ctx context.Context, systemPrompt, userPrompt string, maxTokens int) (string, error) {
	p, err := d.Get(ctx)
	if err != nil {
		return "", err
	}
	return p.CompleteText(ctx, systemPrompt, userPrompt, maxTokens)
}

// CompleteTextWithUsage implements ai.CachingCompleter by dispatching per
// call; a provider without the seam degrades to CompleteText with no usage,
// the shape TranslationService already handles.
func (d *TranslationDispatcher) CompleteTextWithUsage(ctx context.Context, req ai.CompletionRequest) (ai.CompletionResult, error) {
	p, err := d.Get(ctx)
	if err != nil {
		return ai.CompletionResult{}, err
	}
	if caching, ok := p.(ai.CachingCompleter); ok {
		return caching.CompleteTextWithUsage(ctx, req)
	}
	text, err := p.CompleteText(ctx, flattenSystemBlocks(req.System), req.UserPrompt, req.MaxTokens)
	if err != nil {
		return ai.CompletionResult{}, err
	}
	return ai.CompletionResult{Text: text}, nil
}

// TestKey forwards the settings page's Claude probe (the only probe today).
func (d *TranslationDispatcher) TestKey(ctx context.Context, candidate string) error {
	if d.claude == nil {
		return ai.ErrAINotConfigured
	}
	return d.claude.TestKey(ctx, candidate)
}

// MinIntervalFor reports the pacing gap a run on model will keep between
// calls — 0 for Claude, the Gemini holder's gap for Gemini — so an estimate
// can add it to the wall-clock quote.
func (d *TranslationDispatcher) MinIntervalFor(model string) time.Duration {
	if ai.ProviderOf(model) == ai.ProviderNameGemini && d.gemini != nil {
		return d.gemini.MinInterval()
	}
	return 0
}
