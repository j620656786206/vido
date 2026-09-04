package services

import (
	"context"

	"github.com/vido/api/internal/ai"
)

// ModelCatalogService answers "which translation models can THIS deployment
// actually run right now?" — the priced, non-retired catalog (ai.Catalog())
// narrowed to what the subtitle pipeline can really dispatch to, and stamped
// with this deployment's own default.
//
// The filter is not cosmetic: offering a model the pipeline cannot reach would
// let a user consent to a price for a run that fails on its first call, after
// the consent screen already told them what it would cost.
type ModelCatalogService struct {
	resolver KeyResolver
	// effectiveDefault reports the model an omitted model_id actually runs on.
	//
	// It is NOT ai.DefaultClaudeModel: an operator who sets CLAUDE_MODEL is
	// overriding that constant, and the holder's EffectiveModel is the single
	// truth sub-6-5 established. Reading the constant here would tell such an
	// operator that Sonnet is their default and quote every sweep at Sonnet's
	// rate while their runs billed Haiku (CR H1). nil falls back to the
	// package default.
	effectiveDefault func() string
}

// NewModelCatalogService builds the catalog reader. effectiveDefault is
// normally ClaudeProviderHolder.EffectiveModel.
func NewModelCatalogService(resolver KeyResolver, effectiveDefault func() string) *ModelCatalogService {
	return &ModelCatalogService{resolver: resolver, effectiveDefault: effectiveDefault}
}

// defaultModelID is this deployment's effective default, always non-empty.
func (s *ModelCatalogService) defaultModelID() string {
	if s.effectiveDefault != nil {
		if id := s.effectiveDefault(); id != "" {
			return id
		}
	}
	return ai.DefaultClaudeModel
}

// Available returns the models this deployment can run, in ai.Catalog order,
// with IsDefault stamped on the effective default.
//
// ⚠️ CLAUDE ONLY, deliberately (CR H2). The subtitle pipeline's translator is
// the Claude holder and nothing else — main.go builds TranslationService with
// it, and there is no per-model provider dispatch anywhere on the translation
// path. Listing Gemini models because GEMINI_API_KEY happens to be set (which
// the README still advertises for translation) would offer a choice whose only
// possible outcome is a 404 from api.anthropic.com, charged after consent.
// When translation learns to dispatch per provider, this grows a Gemini branch
// — tracked as backlog-gemini-translation-dispatch.
func (s *ModelCatalogService) Available(ctx context.Context) []ai.ModelInfo {
	if s.resolver == nil || !s.resolver.Has(ctx, KeyClaude) {
		return nil
	}

	defaultID := s.defaultModelID()
	out := make([]ai.ModelInfo, 0, 4)
	for _, m := range ai.Catalog() {
		if m.Provider != ai.ProviderNameClaude {
			continue
		}
		m.IsDefault = m.ID == defaultID
		out = append(out, m)
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

// DefaultModel returns the id the UI should pre-select: this deployment's
// effective default when it is actually available, otherwise the first
// available model, otherwise "". A deployment with no usable key pre-selects
// nothing.
func (s *ModelCatalogService) DefaultModel(ctx context.Context) string {
	available := s.Available(ctx)
	for _, m := range available {
		if m.IsDefault {
			return m.ID
		}
	}
	if len(available) > 0 {
		// CLAUDE_MODEL naming something outside the catalog (an alias, a
		// preview id) is legal for the deployment default but cannot be
		// pre-selected as a listed choice; fall back to a real entry rather
		// than pointing the picker at an id it cannot render.
		return available[0].ID
	}
	return ""
}
