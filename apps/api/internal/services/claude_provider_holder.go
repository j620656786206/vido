package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/vido/api/internal/ai"
)

// ClaudeProviderHolder owns the Claude client and rebuilds it when the resolved
// key or model changes.
//
// [@contract-v1] — consumed by sub-2-1b (through the settings API's effect) and
// by every service that used to take a provider at construction time.
//
// It exists because of Break 2: the provider used to be built ONCE at startup
// from an env-var, inside an `if cfg.HasClaudeKey()` guard. A key supplied at
// runtime reached a client that no longer existed to receive it, and if the key
// was empty at boot the dependent services were never constructed at all. The
// holder makes "which key is in force" a per-call question while keeping the
// client itself Rule-14 cached.
//
// It implements ai.TextCompleter AND ai.CachingCompleter by delegation, so it
// can be handed to TranslationService / TerminologyCorrectionService directly.
// The CachingCompleter half is NOT optional: TranslationService type-asserts it
// (translation_service.go:323) and, when the assertion fails, logs one line and
// continues without prompt caching or usage reporting — silently voiding
// sub-1-5b's caching design and paying full price on every cue.
type ClaudeProviderHolder struct {
	resolver KeyResolver
	model    string
	opts     []ai.ClaudeProviderOption
	logger   *slog.Logger

	mu sync.Mutex
	// clients is a bounded LRU of live clients keyed by their "key|model"
	// fingerprint (the download_service.go:24-60 pattern, widened by sub-6-8a).
	//
	// It is a CACHE, not a registry: every entry is built from the same
	// captured opts, so they all share one Governor — a new client must never
	// mean a new budget pool (claude.go's note on WithClaudeGovernor). The
	// bound exists because the fingerprint now contains a user-supplied model:
	// without it, a caller looping over model ids would grow this map without
	// limit (Rule 14). maxCachedClients is small on purpose — a deployment
	// uses one or two models in practice, and evicting a client costs only the
	// next call's construction.
	clients []holderClient
}

// holderClient is one cached provider plus the fingerprint it was built from.
type holderClient struct {
	fingerprint string
	provider    *ai.ClaudeProvider
}

// maxCachedClients bounds the LRU. Four covers "default + the model the user
// picked + two they are comparing" with room to spare. A var, not a const, so
// the package tests can shrink it — the catalog carries only four Claude
// models, too few to demonstrate eviction at the production bound (the
// retryBaseDelay precedent).
var maxCachedClients = 4

// Compile-time proof of both halves — the CachingCompleter assertion above is
// the one that silently degrades if it ever stops holding.
var (
	_ ai.TextCompleter    = (*ClaudeProviderHolder)(nil)
	_ ai.CachingCompleter = (*ClaudeProviderHolder)(nil)
)

// NewClaudeProviderHolder builds the holder. `model` may be empty (the ai
// package's default applies); logger may be nil. `opts` are captured ONCE and
// replayed on every rebuild — which is how the shared Governor survives a key
// change instead of the run budget silently resetting.
func NewClaudeProviderHolder(resolver KeyResolver, model string, logger *slog.Logger, opts ...ai.ClaudeProviderOption) *ClaudeProviderHolder {
	if logger == nil {
		logger = slog.Default()
	}
	return &ClaudeProviderHolder{
		resolver: resolver,
		model:    model,
		opts:     opts,
		logger:   logger.With("component", "claude_provider_holder"),
	}
}

// Get returns the client for THIS call: the ctx's per-run model when the
// caller pinned one (ai.WithModelID, sub-6-8a), otherwise the holder's
// effective default. Rebuilds only when the resolved key or model differs from
// what is cached. Returns ErrAINotConfigured when no key resolves — consumers
// already switch on that sentinel and degrade fail-soft (NFR-R1).
func (h *ClaudeProviderHolder) Get(ctx context.Context) (ai.TextCompleter, error) {
	return h.GetFor(ctx, ai.ModelIDFromContext(ctx))
}

// GetFor is Get with the model named explicitly. An empty model means the
// holder's effective default; an unknown one is rejected here rather than at
// the provider, so a typo cannot reach the API as a 404 the user paid to
// discover (sub-6-8a AC #5).
func (h *ClaudeProviderHolder) GetFor(ctx context.Context, model string) (ai.TextCompleter, error) {
	if model == "" {
		model = h.EffectiveModel()
	} else if !ai.IsSelectableModel(model) {
		return nil, fmt.Errorf("%w: unsupported model %q", ai.ErrAIModelNotFound, model)
	} else if provider := ai.ProviderOf(model); provider != ai.ProviderNameClaude {
		// A CLAUDE holder can only ever send to Anthropic. Without this the
		// model id would be handed to WithClaudeModel verbatim and posted to
		// api.anthropic.com, which answers 404 — a paid-looking failure the
		// user only discovers after consenting (CR H2). The request boundary
		// rejects these too; this is the invariant behind that check, so a
		// future caller that skips validation still cannot get it wrong.
		return nil, fmt.Errorf("%w: model %q is served by %s, not by this Claude provider",
			ai.ErrAIModelNotFound, model, provider)
	}

	key, source, err := h.resolver.Get(ctx, KeyClaude)
	if err != nil {
		return nil, fmt.Errorf("resolve claude key: %w", err)
	}
	if key == "" {
		return nil, fmt.Errorf("%w: no Claude API key configured", ai.ErrAINotConfigured)
	}

	fingerprint := key + "|" + model

	h.mu.Lock()
	defer h.mu.Unlock()

	for i, c := range h.clients {
		if c.fingerprint != fingerprint {
			continue
		}
		// Most-recently-used moves to the front, so the eviction below always
		// drops the client nobody has asked for in the longest time.
		h.clients = append(h.clients[:i], h.clients[i+1:]...)
		h.clients = append([]holderClient{c}, h.clients...)
		return c.provider, nil
	}

	// The model option is appended LAST and ALWAYS, so the client sends exactly
	// the model this fingerprint names — a WithClaudeModel smuggled in through
	// the captured opts can no longer disagree with what the run rows record
	// (sub-6-5 CR H1). The captured opts also carry the shared Governor, so
	// every client in this cache draws on ONE budget and rate pool.
	opts := append(append([]ai.ClaudeProviderOption(nil), h.opts...), ai.WithClaudeModel(model))
	provider := ai.NewClaudeProvider(key, opts...)
	h.clients = append([]holderClient{{fingerprint: fingerprint, provider: provider}}, h.clients...)
	if len(h.clients) > maxCachedClients {
		h.clients = h.clients[:maxCachedClients]
	}

	// Deliberately does NOT log the key or any prefix of it (NFR-S1).
	h.logger.Info("claude provider (re)built",
		"key_source", source, "model", model, "model_override", h.model != "", "cached_clients", len(h.clients))
	return provider, nil
}

// EffectiveModel returns the model id every client this holder builds sends:
// the override this holder was built with, else the ai package default. Get
// and TestKey both append exactly this value as the LAST provider option, so
// there is one owner of "which model" (sub-6-5 CR H1). It needs no key and no
// ctx — the model is a property of the holder, not of the resolved key — so
// the pipeline can call it while assembling every RunVersion (sub-6-5 AC
// #1/#2) instead of snapshotting a possibly-empty string at boot.
//
// h.model stays write-once (constructor), so this read needs no mutex —
// sub-6-8a made the model per-RUN without making it mutable state: a per-run
// choice rides the ctx (ai.WithModelID) and is resolved in GetFor, never by
// writing here. Do not turn h.model into something a request can set.
func (h *ClaudeProviderHolder) EffectiveModel() string {
	if h.model != "" {
		return h.model
	}
	return ai.DefaultClaudeModel
}

// IsConfigured reports whether a key resolves. This is what the subtitle
// pipeline's FR23 capability gate reads (sub-1-6 AC #5), replacing the
// startup-only cfg.HasClaudeKey.
func (h *ClaudeProviderHolder) IsConfigured(ctx context.Context) bool {
	return h.resolver.Has(ctx, KeyClaude)
}

// CompleteText implements ai.TextCompleter by resolving per call.
func (h *ClaudeProviderHolder) CompleteText(ctx context.Context, systemPrompt, userPrompt string, maxTokens int) (string, error) {
	provider, err := h.Get(ctx)
	if err != nil {
		return "", err
	}
	return provider.CompleteText(ctx, systemPrompt, userPrompt, maxTokens)
}

// CompleteTextWithUsage implements ai.CachingCompleter by resolving per call.
// See the type doc for why omitting this would be a silent, expensive bug.
func (h *ClaudeProviderHolder) CompleteTextWithUsage(ctx context.Context, req ai.CompletionRequest) (ai.CompletionResult, error) {
	provider, err := h.Get(ctx)
	if err != nil {
		return ai.CompletionResult{}, err
	}
	caching, ok := provider.(ai.CachingCompleter)
	if !ok {
		// Unreachable today (Get always yields *ai.ClaudeProvider, which
		// implements it) but fails loudly rather than degrading quietly if the
		// concrete type ever changes.
		return ai.CompletionResult{}, fmt.Errorf("claude provider does not implement ai.CachingCompleter")
	}
	return caching.CompleteTextWithUsage(ctx, req)
}

// TestKey validates a key with the provider's Ping — the smallest real call
// the API allows, judged on TRANSPORT alone (sub-6-6): 401/403 → unauthorized,
// 404 → model not found, timeouts and 5xx → their sentinels, and a 2xx with an
// empty reply is a PASS. It used to go through CompleteText, whose
// empty-text = ErrAIInvalidResponse rule made a valid Sonnet 5 key fail as
// "Cannot parse AI response" (eval-1 product problem 6). `candidate` lets the
// settings page test a key BEFORE saving it; empty means "test whatever
// currently resolves".
//
// The throwaway provider reuses this holder's options — so the shared Governor
// still rate-limits the probe — but is never cached: a key being validated is
// not yet the key in force.
func (h *ClaudeProviderHolder) TestKey(ctx context.Context, candidate string) error {
	if strings.TrimSpace(candidate) == "" {
		completer, err := h.Get(ctx)
		if err != nil {
			return err
		}
		pinger, ok := completer.(ai.Pinger)
		if !ok {
			// Unreachable today (Get always yields *ai.ClaudeProvider, which
			// implements Pinger — see the compile-time proof in claude.go) and
			// deliberately LOUD if that ever changes: falling back to
			// CompleteText would silently re-introduce the empty-reply failure
			// this method exists to remove (sub-6-6 CR H1).
			return fmt.Errorf("claude provider does not implement ai.Pinger")
		}
		return pinger.Ping(ctx)
	}

	opts := append(append([]ai.ClaudeProviderOption(nil), h.opts...), ai.WithClaudeModel(h.EffectiveModel()))
	return ai.NewClaudeProvider(candidate, opts...).Ping(ctx)
}
