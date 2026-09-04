package services

import (
	"context"

	"github.com/vido/api/internal/ai"
)

// ModelCatalogService answers "which translation models can THIS deployment
// actually run right now?" — the priced, non-retired catalog (ai.Catalog())
// narrowed to the providers whose key resolves.
//
// The filter is not cosmetic: offering a Gemini model to a Claude-only
// deployment would let a user consent to a price for a run that fails on its
// first call, after the consent screen already told them what it would cost.
type ModelCatalogService struct {
	resolver KeyResolver
	// geminiConfigured reports whether a Gemini key is available.
	//
	// It is a func rather than another KeyResolver lookup because Gemini is
	// NOT part of the resolver's closed key set (KeyClaude/KeyTMDb/KeyOpenAI):
	// it has no secret row and no settings-page field, so its key can only
	// come from GEMINI_API_KEY. main.go supplies that read. When Gemini gains
	// a stored key, this becomes resolver.Has(ctx, KeyGemini) and the func
	// goes away — tracked as backlog-gemini-key-in-resolver.
	geminiConfigured func() bool
}

// NewModelCatalogService builds the catalog reader. geminiConfigured may be
// nil, which reads as "no Gemini key" — the safe answer, since an unavailable
// model that is hidden costs nothing while one that is offered costs a failed
// run.
func NewModelCatalogService(resolver KeyResolver, geminiConfigured func() bool) *ModelCatalogService {
	return &ModelCatalogService{resolver: resolver, geminiConfigured: geminiConfigured}
}

// Available returns the models this deployment can run, in ai.Catalog order.
func (s *ModelCatalogService) Available(ctx context.Context) []ai.ModelInfo {
	claude := s.resolver != nil && s.resolver.Has(ctx, KeyClaude)
	gemini := s.geminiConfigured != nil && s.geminiConfigured()

	out := make([]ai.ModelInfo, 0, 8)
	for _, m := range ai.Catalog() {
		switch m.Provider {
		case ai.ProviderNameClaude:
			if claude {
				out = append(out, m)
			}
		case ai.ProviderNameGemini:
			if gemini {
				out = append(out, m)
			}
		}
	}
	return out
}

// Supports reports whether a model id may be requested for a run here. An
// empty id is always allowed and means "the deployment default" — the API
// treats absent and "" identically so a client can omit the field.
//
// This is the validation behind the 400 (sub-6-8a AC #4): a model the user
// cannot reach must be refused at the boundary, before a batch is started and
// a ceiling is committed.
func (s *ModelCatalogService) Supports(ctx context.Context, model string) bool {
	if model == "" {
		return true
	}
	for _, m := range s.Available(ctx) {
		if m.ID == model {
			return true
		}
	}
	return false
}

// DefaultModel returns the id the UI should pre-select: the deployment's
// effective default when it is actually available, otherwise the first
// available model, otherwise "". A Claude-less deployment must not pre-select
// a Claude model.
func (s *ModelCatalogService) DefaultModel(ctx context.Context) string {
	available := s.Available(ctx)
	for _, m := range available {
		if m.IsDefault {
			return m.ID
		}
	}
	if len(available) > 0 {
		return available[0].ID
	}
	return ""
}
