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
	// cached is the live client; fingerprint is the "key|model" identity it was
	// built from (the download_service.go:24-60 pattern).
	cached      *ai.ClaudeProvider
	fingerprint string
}

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

// Get returns the current client, rebuilding only when the resolved key or model
// differs from the cached one. Returns ErrAINotConfigured when no key resolves —
// consumers already switch on that sentinel and degrade fail-soft (NFR-R1).
func (h *ClaudeProviderHolder) Get(ctx context.Context) (ai.TextCompleter, error) {
	key, source, err := h.resolver.Get(ctx, KeyClaude)
	if err != nil {
		return nil, fmt.Errorf("resolve claude key: %w", err)
	}
	if key == "" {
		return nil, fmt.Errorf("%w: no Claude API key configured", ai.ErrAINotConfigured)
	}

	fingerprint := key + "|" + h.model

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.cached != nil && h.fingerprint == fingerprint {
		return h.cached, nil
	}

	opts := h.opts
	if h.model != "" {
		opts = append(append([]ai.ClaudeProviderOption(nil), opts...), ai.WithClaudeModel(h.model))
	}
	h.cached = ai.NewClaudeProvider(key, opts...)
	h.fingerprint = fingerprint

	// Deliberately does NOT log the key or any prefix of it (NFR-S1).
	h.logger.Info("claude provider (re)built", "key_source", source, "model_override", h.model != "")
	return h.cached, nil
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

// TestKey validates a key by making the smallest real call the API allows
// (max_tokens 1). `candidate` lets the settings page test a key BEFORE saving
// it; empty means "test whatever currently resolves".
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
		_, err = completer.CompleteText(ctx, "", "hi", 1)
		return err
	}

	opts := h.opts
	if h.model != "" {
		opts = append(append([]ai.ClaudeProviderOption(nil), opts...), ai.WithClaudeModel(h.model))
	}
	_, err := ai.NewClaudeProvider(candidate, opts...).CompleteText(ctx, "", "hi", 1)
	return err
}
