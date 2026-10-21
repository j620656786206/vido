package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/vido/api/internal/ai"
)

// ParseProviderFactory builds one filename-parse client for a resolved key.
// Production uses defaultParseProviderFactory (the native Gemini / Claude
// providers); tests swap in a fake so no test can spend money.
type ParseProviderFactory func(name ai.ProviderName, key, model string, governor *ai.Governor) ai.Provider

// defaultParseProviderFactory builds the NATIVE providers — Gemini parsing
// goes through ai.GeminiProvider, never the OpenAI-compatible translation
// client. The model is always passed explicitly (the holder already applied
// GEMINI_MODEL / CLAUDE_MODEL or the package default), and the shared
// Governor rides every client so a key change never means a new rate pool.
func defaultParseProviderFactory(name ai.ProviderName, key, model string, governor *ai.Governor) ai.Provider {
	if name == ai.ProviderClaude {
		return ai.NewClaudeProvider(key, ai.WithClaudeModel(model), ai.WithClaudeGovernor(governor))
	}
	return ai.NewGeminiProvider(key, ai.WithGeminiModel(model), ai.WithGeminiGovernor(governor))
}

// ResolvedParseProvider is the client chosen for ONE call, plus what the
// side-spend row needs to say about it.
type ResolvedParseProvider struct {
	Provider ai.Provider
	Name     ai.ProviderName
	Model    string
}

// ParseProviderHolder resolves the filename-parse provider per call
// (backlog-ai-parse-reads-env-keys-only-a). It is the parse-path twin of
// ClaudeProviderHolder: that one hands out an ai.TextCompleter, which has no
// Parse.
//
// It exists because AIService used to read the env-var keys ONCE at boot: a
// key saved in the setup wizard or the settings page (the encrypted store)
// never reached filename parsing — production itself ran with AI parsing off.
// Now every call asks the KeyResolver (secrets store → env) which keys exist,
// picks the provider, and reuses the cached client unless the
// provider + key + model fingerprint changed.
type ParseProviderHolder struct {
	resolver    KeyResolver
	preferred   ai.ProviderName
	claudeModel string
	geminiModel string
	governor    *ai.Governor
	factory     ParseProviderFactory
	logger      *slog.Logger

	mu sync.Mutex
	// clients holds at most one live client per provider, keyed by name; each
	// remembers the fingerprint it was built from. Two entries is the bound —
	// there are only two parse providers.
	clients map[ai.ProviderName]parseHolderClient
}

type parseHolderClient struct {
	fingerprint string
	provider    ai.Provider
}

// NewParseProviderHolder builds the holder.
//
//   - preferred is AI_PROVIDER: which provider to use when BOTH have a key.
//     When only one has a key, that one is used whatever AI_PROVIDER says — so
//     a Claude-only deployment no longer has to set AI_PROVIDER=claude (and no
//     longer fails to boot when it doesn't). An unknown value is treated as
//     the default (gemini) with a warning, instead of the old boot failure.
//   - claudeModel / geminiModel are CLAUDE_MODEL / GEMINI_MODEL; empty means
//     the ai package default.
//   - governor is main.go's single process-wide throttle (governor.go: never
//     build one per request). nil = unthrottled.
//   - factory nil = the native providers.
func NewParseProviderHolder(resolver KeyResolver, preferred, claudeModel, geminiModel string, governor *ai.Governor, factory ParseProviderFactory, logger *slog.Logger) *ParseProviderHolder {
	if logger == nil {
		logger = slog.Default()
	}
	if factory == nil {
		factory = defaultParseProviderFactory
	}
	logger = logger.With("component", "parse_provider_holder")

	pref := ai.ProviderName(strings.ToLower(strings.TrimSpace(preferred)))
	switch {
	case pref == "":
		pref = ai.ProviderGemini
	case !pref.IsValid():
		logger.Warn("AI_PROVIDER is not gemini or claude — treating it as gemini",
			"value", preferred)
		pref = ai.ProviderGemini
	}

	return &ParseProviderHolder{
		resolver:    resolver,
		preferred:   pref,
		claudeModel: claudeModel,
		geminiModel: geminiModel,
		governor:    governor,
		factory:     factory,
		logger:      logger,
		clients:     map[ai.ProviderName]parseHolderClient{},
	}
}

// keyNameFor maps a parse provider onto its resolver key.
func keyNameFor(name ai.ProviderName) KeyName {
	if name == ai.ProviderClaude {
		return KeyClaude
	}
	return KeyGemini
}

// otherProvider is the fallback when the preferred one has no key.
func otherProvider(name ai.ProviderName) ai.ProviderName {
	if name == ai.ProviderClaude {
		return ai.ProviderGemini
	}
	return ai.ProviderClaude
}

// modelFor returns the model a client for this provider sends.
func (h *ParseProviderHolder) modelFor(name ai.ProviderName) string {
	if name == ai.ProviderClaude {
		if h.claudeModel != "" {
			return h.claudeModel
		}
		return ai.DefaultClaudeModel
	}
	if h.geminiModel != "" {
		return h.geminiModel
	}
	return ai.DefaultGeminiModel
}

// choose picks the provider for this call: the preferred one when its key
// resolves, otherwise the other one, otherwise none ("", "").
func (h *ParseProviderHolder) choose(ctx context.Context) (ai.ProviderName, string, KeySource) {
	for _, name := range []ai.ProviderName{h.preferred, otherProvider(h.preferred)} {
		key, source, err := h.resolver.Get(ctx, keyNameFor(name))
		if err != nil {
			// Unreachable for the two known key names; logged, then treated as
			// "no key" so a resolver fault degrades to the no-AI path.
			h.logger.Warn("resolve parse key failed", "provider", name, "error", err)
			continue
		}
		if strings.TrimSpace(key) != "" {
			return name, key, source
		}
	}
	return "", "", KeySourceNone
}

// ProviderName reports which provider the next call would use, "" when no key
// resolves. Builds nothing.
func (h *ParseProviderHolder) ProviderName(ctx context.Context) string {
	name, _, _ := h.choose(ctx)
	return string(name)
}

// IsConfigured reports whether either parse key resolves right now.
func (h *ParseProviderHolder) IsConfigured(ctx context.Context) bool {
	return h.ProviderName(ctx) != ""
}

// Resolve returns the client for THIS call, rebuilding it only when the
// provider + key + model fingerprint differs from the cached one. Returns
// ai.ErrAINotConfigured when no key resolves — AIService's callers already
// degrade on that sentinel.
func (h *ParseProviderHolder) Resolve(ctx context.Context) (ResolvedParseProvider, error) {
	name, key, source := h.choose(ctx)
	if name == "" {
		// Both keys were deleted: drop the cached clients so a removed key
		// does not stay live in memory until the next one is saved.
		h.mu.Lock()
		clear(h.clients)
		h.mu.Unlock()
		return ResolvedParseProvider{}, fmt.Errorf("%w: no Gemini or Claude API key configured", ai.ErrAINotConfigured)
	}
	model := h.modelFor(name)
	fingerprint := string(name) + "|" + key + "|" + model

	h.mu.Lock()
	defer h.mu.Unlock()
	if c, ok := h.clients[name]; ok && c.fingerprint == fingerprint {
		return ResolvedParseProvider{Provider: c.provider, Name: name, Model: model}, nil
	}
	provider := h.factory(name, key, model, h.governor)
	h.clients[name] = parseHolderClient{fingerprint: fingerprint, provider: provider}

	// Deliberately does NOT log the key or any prefix of it (NFR-S1).
	h.logger.Info("parse provider (re)built", "provider", name, "model", model, "key_source", source)
	return ResolvedParseProvider{Provider: provider, Name: name, Model: model}, nil
}
