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
}

// CreateMediaRequestRequest is the POST /api/v1/requests body (snake_case per
// Rule 6). Named to avoid the double-"Request" collision the house
// CreateXRequest DTO convention would otherwise produce.
type CreateMediaRequestRequest struct {
	TMDbID    int64  `json:"tmdb_id"`
	MediaType string `json:"media_type"`
}

// RequestService implements RequestServiceInterface.
type RequestService struct {
	repo       repository.RequestRepositoryInterface
	tmdb       TMDbServiceInterface
	movieRepo  repository.MovieRepositoryInterface
	seriesRepo repository.SeriesRepositoryInterface
}

// Compile-time verification.
var _ RequestServiceInterface = (*RequestService)(nil)

// NewRequestService builds a new RequestService. The TMDb service is the
// existing Epic-2 singleton (zh-TW fallback chain + cache + limiter — Rule 27
// by reuse); movie/series repos back the already-owned guard.
func NewRequestService(
	repo repository.RequestRepositoryInterface,
	tmdbService TMDbServiceInterface,
	movieRepo repository.MovieRepositoryInterface,
	seriesRepo repository.SeriesRepositoryInterface,
) *RequestService {
	return &RequestService{repo: repo, tmdb: tmdbService, movieRepo: movieRepo, seriesRepo: seriesRepo}
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
	return request, nil
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
	if t := strings.TrimSpace(title); t != "" {
		return t, nil
	}
	if o := strings.TrimSpace(original); o != "" {
		return o, nil
	}
	// Pathological edge: TMDb entry with no usable title in any language —
	// store a deterministic placeholder, never an empty NOT-NULL title (CR L1).
	return fmt.Sprintf("TMDB-%d", tmdbID), nil
}
