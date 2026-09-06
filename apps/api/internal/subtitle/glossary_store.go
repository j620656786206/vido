package subtitle

import (
	"context"
	"errors"
	"fmt"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

// GlossaryStore is the narrow port over the per-show glossary (sub-5-5 AC #5,
// the SegmentCache pattern): the item flow needs exactly the feed read and the
// harvest write, so tests fake two methods instead of the repository's eight.
//
// Lookup returns the term_src→term_zh map fed into the translation prompt.
// ALL terms, confirmed and auto-mined alike (confirmedOnly=false) — the 9R-10
// legacy posture, kept identical across both paths; F6 review corrects dirty
// terms, and the corrected glossary re-keys the cache via GlossaryVersion.
//
// InsertNew writes harvested terms insert-if-absent (AC #4 red line 2: an
// existing term — whatever its source or confirmed state — is NEVER touched).
// It returns how many terms were actually inserted (deduped conflicts do not
// count). Best-effort: a per-term failure skips that term and surfaces in the
// returned error while the rest still land — the caller fail-softs.
//
// Both methods still take the pipeline's glossary KEY (a local show id —
// `glossaryKeyFor`): the pipeline does not know what a scope is, and the
// harvest write needs the local id anyway for the audit column. The adapter
// below is where the key becomes a scope (sub-7-1 AC #2) — this package
// cannot import services (services imports subtitle), so the resolver comes
// in through the GlossaryScopeResolver port.
type GlossaryStore interface {
	Lookup(ctx context.Context, mediaID string) (map[string]string, error)
	InsertNew(ctx context.Context, mediaID string, terms map[string]string) (int, error)
}

// GlossaryScopeResolver is the port the store adapter resolves through —
// satisfied by services.GlossaryScopeResolver, wired in main.
type GlossaryScopeResolver interface {
	Resolve(ctx context.Context, mediaID string) (string, error)
}

// glossaryStoreRepository adapts repository.GlossaryRepositoryInterface to
// GlossaryStore (the NewSegmentCacheRepository pattern).
type glossaryStoreRepository struct {
	repo   repository.GlossaryRepositoryInterface
	scopes GlossaryScopeResolver
}

// NewGlossaryStoreRepository wires the glossary store onto the shared
// show_glossary table. A nil resolver keys everything under `local:<id>` —
// the pre-sub-7-1 behaviour, so partial wiring degrades instead of breaking.
func NewGlossaryStoreRepository(repo repository.GlossaryRepositoryInterface, scopes GlossaryScopeResolver) GlossaryStore {
	return &glossaryStoreRepository{repo: repo, scopes: scopes}
}

func (r *glossaryStoreRepository) scopeFor(ctx context.Context, mediaID string) (string, error) {
	if r.scopes == nil {
		return models.GlossaryScopeLocal(mediaID), nil
	}
	return r.scopes.Resolve(ctx, mediaID)
}

func (r *glossaryStoreRepository) Lookup(ctx context.Context, mediaID string) (map[string]string, error) {
	scope, err := r.scopeFor(ctx, mediaID)
	if err != nil {
		return nil, err
	}
	return r.repo.LookupByScope(ctx, scope, false)
}

func (r *glossaryStoreRepository) InsertNew(ctx context.Context, mediaID string, terms map[string]string) (int, error) {
	scope, err := r.scopeFor(ctx, mediaID)
	if err != nil {
		return 0, err
	}
	inserted := 0
	var errs []error
	for src, zh := range terms {
		ok, err := r.repo.InsertIfAbsent(ctx, &models.GlossaryTerm{
			MediaID: mediaID,
			Scope:   scope,
			TermSrc: src,
			TermZh:  zh,
			Source:  models.GlossarySourceSubtitle,
			// Confirmed stays false: harvested terms enter the existing F6
			// review flow — the spec's "不默默全信".
		})
		if err != nil {
			errs = append(errs, fmt.Errorf("term %q: %w", src, err))
			continue
		}
		if ok {
			inserted++
		}
	}
	return inserted, errors.Join(errs...)
}

// compile-time proof the adapter satisfies the port.
var _ GlossaryStore = (*glossaryStoreRepository)(nil)
