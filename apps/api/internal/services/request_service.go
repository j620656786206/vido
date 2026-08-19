// Package services RequestService — Story 13-1a (G-1/P3-001, Epic 13).
//
// Records a user's intent to acquire a title as a durable pending request.
// This story is intent-only: NO fulfilment (13-4), NO status transitions or
// SSE (13-3a), NO season/episode selection (13-2a). The create/list resource
// shape carries [@contract-v1] (13-1a AC #2/#3).
package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

// ErrRequestAlreadyInLibrary is returned when the requested title already
// exists in the local library (AC #5) — the FE shows 已入庫 with no action,
// but the API never trusts the FE.
var ErrRequestAlreadyInLibrary = errors.New("requested media already in library")

// RequestServiceInterface defines the contract for media request operations.
type RequestServiceInterface interface {
	CreateRequest(ctx context.Context, req CreateMediaRequestRequest) (*models.Request, error)
	ListRequests(ctx context.Context) ([]models.Request, error)
	// TVCoverage answers the 13-2b tree's owned/requested reflection for one
	// TV show (13-2a AC #5 [@contract-v1]).
	TVCoverage(ctx context.Context, tmdbID int64) (*RequestCoverage, error)
}

// EpisodeOwnershipReader is the narrow port behind the episode-level owned
// guard and the coverage endpoint (13-2a AC #3/#5) — the one episode-repo
// method this service needs. *repository.EpisodeRepository satisfies it.
type EpisodeOwnershipReader interface {
	FindBySeriesID(ctx context.Context, seriesID string) ([]models.Episode, error)
}

// CreateMediaRequestRequest is the POST /api/v1/requests body (snake_case per
// Rule 6). Named to avoid the double-"Request" collision the house
// CreateXRequest DTO convention would otherwise produce.
//
// Seasons/Episodes are the OPTIONAL 13-2a partial selection ([@contract-v1]
// AC #1): tv-only; both absent/empty = whole-title request, byte-identical to
// the 13-1a behavior.
type CreateMediaRequestRequest struct {
	TMDbID    int64            `json:"tmdb_id"`
	MediaType string           `json:"media_type"`
	Seasons   []int            `json:"seasons"`
	Episodes  map[string][]int `json:"episodes"`
}

// RequestService implements RequestServiceInterface.
type RequestService struct {
	repo       repository.RequestRepositoryInterface
	tmdb       TMDbServiceInterface
	movieRepo  repository.MovieRepositoryInterface
	seriesRepo repository.SeriesRepositoryInterface
	// episodeRepo backs the episode-level owned guard + coverage (13-2a).
	episodeRepo EpisodeOwnershipReader
	// fulfilment is an OPTIONAL nil-safe dependency (Story 13-4a AC #6) —
	// when absent/unconfigured the 13-1a create behavior is preserved
	// exactly (rows born pending, no transition).
	fulfilment FulfilmentServiceInterface
}

// Compile-time verification.
var _ RequestServiceInterface = (*RequestService)(nil)

// NewRequestService builds a new RequestService. The TMDb service is the
// existing Epic-2 singleton (zh-TW fallback chain + cache + limiter — Rule 27
// by reuse); movie/series repos back the already-owned guard; the episode
// reader backs the 13-2a episode-level guard + coverage.
func NewRequestService(
	repo repository.RequestRepositoryInterface,
	tmdbService TMDbServiceInterface,
	movieRepo repository.MovieRepositoryInterface,
	seriesRepo repository.SeriesRepositoryInterface,
	episodeRepo EpisodeOwnershipReader,
) *RequestService {
	return &RequestService{
		repo: repo, tmdb: tmdbService,
		movieRepo: movieRepo, seriesRepo: seriesRepo, episodeRepo: episodeRepo,
	}
}

func (s *RequestService) CreateRequest(ctx context.Context, req CreateMediaRequestRequest) (*models.Request, error) {
	request := &models.Request{
		TMDbID:    req.TMDbID,
		MediaType: strings.TrimSpace(req.MediaType),
		Status:    models.RequestStatusPending,
	}
	if err := request.Validate(); err != nil {
		return nil, fmt.Errorf("validation: %w", err)
	}

	// 13-2a AC #1 — a partial selection takes the selection-aware path; an
	// empty one falls through to the 13-1a whole-title flow byte-identically.
	sel, err := canonicalizeSelection(request.MediaType, req.Seasons, req.Episodes)
	if err != nil {
		return nil, err
	}
	if sel != nil {
		return s.createPartialRequest(ctx, request, sel)
	}

	// AC #5 — already-in-library guard. Bulk helper instead of FindByTMDbID:
	// its not-found is untyped, and FindOwnedTMDbIDs answers ownership without
	// error-string matching.
	owned, err := s.ownedTMDbIDs(ctx, request.MediaType, request.TMDbID)
	if err != nil {
		return nil, fmt.Errorf("owned check: %w", err)
	}
	if len(owned) > 0 {
		return nil, fmt.Errorf("tmdb_id %d (%s): %w", request.TMDbID, request.MediaType, ErrRequestAlreadyInLibrary)
	}

	// AC #4 — active-duplicate guard (clean error path; the partial unique
	// index in migration 027 backs this against races at Create below).
	if _, err := s.repo.FindActiveByTMDbID(ctx, request.TMDbID, request.MediaType); err == nil {
		return nil, fmt.Errorf("tmdb_id %d (%s): %w", request.TMDbID, request.MediaType, repository.ErrRequestDuplicate)
	} else if !errors.Is(err, repository.ErrRequestNotFound) {
		return nil, fmt.Errorf("duplicate check: %w", err)
	}

	// AC #2 — resolve the zh-TW title server-side (never trust a client title).
	// A bad tmdb_id surfaces here as the client's typed TMDB_NOT_FOUND error.
	title, err := s.resolveTitle(ctx, request.MediaType, request.TMDbID)
	if err != nil {
		return nil, fmt.Errorf("resolve tmdb target: %w", err)
	}
	request.Title = title

	if err := s.repo.Create(ctx, request); err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	slog.Info("Media request created",
		"id", request.ID, "tmdb_id", request.TMDbID, "media_type", request.MediaType, "title", request.Title)

	// Story 13-4a AC #6 — synchronous best-effort fulfilment. Never fails
	// the create (graceful degradation lives inside FulfilRequest); the
	// mutated row state rides back on the 201 response.
	if s.fulfilment != nil {
		s.fulfilment.FulfilRequest(ctx, request)
	}
	return request, nil
}

// createPartialRequest is the 13-2a selection-aware create path (tv only —
// canonicalizeSelection already rejected movies). Guard order: cheap active-
// duplicate first (⚖️ A ruling: partial does NOT bypass the one-active-per-
// title guard), then ONE TMDB details fetch shared by validation, ownership
// and the title, then the episode-level owned guard.
func (s *RequestService) createPartialRequest(ctx context.Context, request *models.Request, sel *RequestSelection) (*models.Request, error) {
	if _, err := s.repo.FindActiveByTMDbID(ctx, request.TMDbID, request.MediaType); err == nil {
		return nil, fmt.Errorf("tmdb_id %d (%s): %w", request.TMDbID, request.MediaType, repository.ErrRequestDuplicate)
	} else if !errors.Is(err, repository.ErrRequestNotFound) {
		return nil, fmt.Errorf("duplicate check: %w", err)
	}

	details, err := s.tmdb.GetTVShowDetails(ctx, int(request.TMDbID))
	if err != nil {
		return nil, fmt.Errorf("resolve tmdb target: %w", err)
	}
	if err := s.validateAgainstTMDB(ctx, request.TMDbID, sel, details); err != nil {
		return nil, err
	}

	// Episode-level owned guard (AC #3): only consulted when the series is
	// locally present at all — FindOwnedTMDbIDs is the typed-absence probe
	// (the 13-1a rationale: FindByTMDbID's not-found is untyped).
	ownedIDs, err := s.seriesRepo.FindOwnedTMDbIDs(ctx, []int64{request.TMDbID})
	if err != nil {
		return nil, fmt.Errorf("owned check: %w", err)
	}
	if len(ownedIDs) > 0 {
		ownedMap, oerr := s.ownedEpisodeNumbers(ctx, request.TMDbID)
		if oerr != nil {
			return nil, oerr
		}
		if gerr := checkSelectionOwnership(sel, ownedMap, details); gerr != nil {
			return nil, gerr
		}
	}

	request.Title = pickTitle(details.Name, details.OriginalName, request.TMDbID)
	request.Seasons, request.Episodes, err = selectionColumns(sel)
	if err != nil {
		return nil, fmt.Errorf("serialize selection: %w", err)
	}

	if err := s.repo.Create(ctx, request); err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	slog.Info("Media request created",
		"id", request.ID, "tmdb_id", request.TMDbID, "media_type", request.MediaType,
		"title", request.Title, "selection", selectionLogValue(sel))

	if s.fulfilment != nil {
		s.fulfilment.FulfilRequest(ctx, request)
	}
	return request, nil
}

// TVCoverage answers the 13-2b tree's reflection data for one TV show
// (13-2a AC #5 [@contract-v1]): locally-owned episode numbers per season plus
// the ACTIVE request's selection. A show with no local series and no active
// request returns the empty-but-valid shape (200 — the tree must open for a
// brand-new show too).
func (s *RequestService) TVCoverage(ctx context.Context, tmdbID int64) (*RequestCoverage, error) {
	cov := &RequestCoverage{
		Owned:             map[string][]int{},
		RequestedSeasons:  []int{},
		RequestedEpisodes: map[string][]int{},
	}

	ownedIDs, err := s.seriesRepo.FindOwnedTMDbIDs(ctx, []int64{tmdbID})
	if err != nil {
		return nil, fmt.Errorf("owned check: %w", err)
	}
	if len(ownedIDs) > 0 {
		ownedMap, oerr := s.ownedEpisodeNumbers(ctx, tmdbID)
		if oerr != nil {
			return nil, oerr
		}
		for season, eps := range ownedMap {
			cov.Owned[strconv.Itoa(season)] = eps
		}
	}

	active, err := s.repo.FindActiveByTMDbID(ctx, tmdbID, models.RequestMediaTypeTV)
	switch {
	case errors.Is(err, repository.ErrRequestNotFound):
		return cov, nil
	case err != nil:
		return nil, fmt.Errorf("active request lookup: %w", err)
	}

	cov.ActiveRequest = true
	sel, err := parseSelectionColumns(active.Seasons, active.Episodes)
	if err != nil {
		return nil, err
	}
	if sel == nil {
		cov.WholeSeriesRequested = true
		return cov, nil
	}
	if sel.Seasons != nil {
		cov.RequestedSeasons = sel.Seasons
	}
	for season, eps := range sel.Episodes {
		cov.RequestedEpisodes[strconv.Itoa(season)] = eps
	}
	return cov, nil
}

// selectionLogValue renders a selection for structured logs (AC #6).
func selectionLogValue(sel *RequestSelection) string {
	if sel == nil {
		return "whole"
	}
	return fmt.Sprintf("seasons=%v episodes=%v", sel.Seasons, sel.Episodes)
}

// SetFulfilmentService wires the optional fulfilment dependency (13-4a).
func (s *RequestService) SetFulfilmentService(fulfilment FulfilmentServiceInterface) {
	s.fulfilment = fulfilment
}

func (s *RequestService) ListRequests(ctx context.Context) ([]models.Request, error) {
	requests, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list requests: %w", err)
	}
	return requests, nil
}

// ownedTMDbIDs routes the ownership check by media_type: movie→movies,
// tv→series (the TMDB/FE 'tv' maps onto the local series table).
func (s *RequestService) ownedTMDbIDs(ctx context.Context, mediaType string, tmdbID int64) ([]int64, error) {
	if mediaType == models.RequestMediaTypeMovie {
		return s.movieRepo.FindOwnedTMDbIDs(ctx, []int64{tmdbID})
	}
	return s.seriesRepo.FindOwnedTMDbIDs(ctx, []int64{tmdbID})
}

// resolveTitle fetches the zh-TW-preferred title via the Epic-2 TMDb chain.
// Movies carry Title/OriginalTitle; TV carries Name/OriginalName.
func (s *RequestService) resolveTitle(ctx context.Context, mediaType string, tmdbID int64) (string, error) {
	var title, original string
	if mediaType == models.RequestMediaTypeMovie {
		details, err := s.tmdb.GetMovieDetails(ctx, int(tmdbID))
		if err != nil {
			return "", err
		}
		title, original = details.Title, details.OriginalTitle
	} else {
		details, err := s.tmdb.GetTVShowDetails(ctx, int(tmdbID))
		if err != nil {
			return "", err
		}
		title, original = details.Name, details.OriginalName
	}
	return pickTitle(title, original, tmdbID), nil
}

// pickTitle applies the zh-TW-preferred → original → placeholder fallback
// (extracted from resolveTitle so the 13-2a selection path reuses it on the
// details object it already fetched).
func pickTitle(title, original string, tmdbID int64) string {
	if t := strings.TrimSpace(title); t != "" {
		return t
	}
	if o := strings.TrimSpace(original); o != "" {
		return o
	}
	// Pathological edge: TMDb entry with no usable title in any language —
	// store a deterministic placeholder, never an empty NOT-NULL title (CR L1).
	return fmt.Sprintf("TMDB-%d", tmdbID)
}
