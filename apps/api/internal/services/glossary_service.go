package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

// GlossaryServiceInterface is the glossary management contract (Story 9R-15) —
// the REST surface over the per-show glossary (9R-6) for the F6 review UI.
type GlossaryServiceInterface interface {
	List(ctx context.Context, mediaID string) ([]models.GlossaryTerm, error)
	Add(ctx context.Context, term *models.GlossaryTerm) error
	Edit(ctx context.Context, mediaID, id, termZh string, confirmed bool) error
	Confirm(ctx context.Context, mediaID, id string) error
	ConfirmAll(ctx context.Context, mediaID string) (int64, error)
	Delete(ctx context.Context, mediaID, id string) error
}

// GlossaryService wraps GlossaryRepository with validation (Rule 4 layering).
// The route still speaks local media ids (AC #4: HTTP/web untouched); this
// service is where they become scopes.
type GlossaryService struct {
	repo   repository.GlossaryRepositoryInterface
	scopes GlossaryScopeResolverInterface
}

// NewGlossaryService builds a GlossaryService. A nil resolver keys everything
// under `local:<id>` — the pre-sub-7-1 behaviour, for partial wiring and tests.
func NewGlossaryService(repo repository.GlossaryRepositoryInterface, scopes GlossaryScopeResolverInterface) *GlossaryService {
	return &GlossaryService{repo: repo, scopes: scopes}
}

func (s *GlossaryService) scopeFor(ctx context.Context, mediaID string) (string, error) {
	if s.scopes == nil {
		return localScopeFallback(mediaID), nil
	}
	return s.scopes.Resolve(ctx, mediaID)
}

// Compile-time interface verification.
var _ GlossaryServiceInterface = (*GlossaryService)(nil)

func (s *GlossaryService) List(ctx context.Context, mediaID string) ([]models.GlossaryTerm, error) {
	if strings.TrimSpace(mediaID) == "" {
		return nil, &models.ValidationError{Field: "media_id", Message: "media_id is required"}
	}
	scope, err := s.scopeFor(ctx, mediaID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListByScope(ctx, scope)
}

// Add creates (or upserts) a term. The mediaID from the route is authoritative
// — it is stamped onto the term so a client can't cross-write another show's
// glossary via the body — and it is resolved to the scope here, so a body
// `scope` can never aim a write at a drawer the route does not own either.
func (s *GlossaryService) Add(ctx context.Context, term *models.GlossaryTerm) error {
	if term == nil {
		return fmt.Errorf("term cannot be nil")
	}
	if strings.TrimSpace(term.MediaID) == "" {
		return &models.ValidationError{Field: "media_id", Message: "media_id is required"}
	}
	scope, err := s.scopeFor(ctx, term.MediaID)
	if err != nil {
		return err
	}
	term.Scope = scope
	if term.Source == "" {
		term.Source = models.GlossarySourceManual
	}
	return s.repo.Upsert(ctx, term)
}

func (s *GlossaryService) Edit(ctx context.Context, mediaID, id, termZh string, confirmed bool) error {
	if strings.TrimSpace(id) == "" {
		return &models.ValidationError{Field: "id", Message: "id is required"}
	}
	_, err := s.repo.Update(ctx, id, termZh, confirmed)
	return err
}

func (s *GlossaryService) Confirm(ctx context.Context, mediaID, id string) error {
	if strings.TrimSpace(id) == "" {
		return &models.ValidationError{Field: "id", Message: "id is required"}
	}
	_, err := s.repo.Confirm(ctx, id)
	return err
}

func (s *GlossaryService) ConfirmAll(ctx context.Context, mediaID string) (int64, error) {
	if strings.TrimSpace(mediaID) == "" {
		return 0, &models.ValidationError{Field: "media_id", Message: "media_id is required"}
	}
	scope, err := s.scopeFor(ctx, mediaID)
	if err != nil {
		return 0, err
	}
	return s.repo.ConfirmAllByScope(ctx, scope)
}

func (s *GlossaryService) Delete(ctx context.Context, mediaID, id string) error {
	if strings.TrimSpace(id) == "" {
		return &models.ValidationError{Field: "id", Message: "id is required"}
	}
	return s.repo.Delete(ctx, id)
}
