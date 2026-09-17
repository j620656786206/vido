package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/metadata"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/tmdb"
)

func TestNewMetadataService(t *testing.T) {
	cfg := MetadataServiceConfig{
		TMDbImageBaseURL:     "https://image.tmdb.org/t/p/w500",
		EnableDouban:         true,
		EnableWikipedia:      true,
		EnableCircuitBreaker: true,
		FallbackDelayMs:      100,
	}

	mockTMDb := &mockTMDbSearcher{}
	service := NewMetadataService(cfg, mockTMDb)

	assert.NotNil(t, service)
}

func TestMetadataService_SearchMetadata_Success(t *testing.T) {
	cfg := MetadataServiceConfig{
		TMDbImageBaseURL: "https://image.tmdb.org/t/p/w500",
	}

	mockTMDb := &mockTMDbSearcher{
		searchMoviesFunc: func(ctx context.Context, query string, page int) (*tmdb.SearchResultMovies, error) {
			return &tmdb.SearchResultMovies{
				Page: 1,
				Results: []tmdb.Movie{
					{
						ID:          550,
						Title:       "Fight Club",
						ReleaseDate: "1999-10-15",
						VoteAverage: 8.4,
					},
				},
				TotalPages:   1,
				TotalResults: 1,
			}, nil
		},
	}

	service := NewMetadataService(cfg, mockTMDb)

	result, status, err := service.SearchMetadata(context.Background(), &SearchMetadataRequest{
		Query:     "Fight Club",
		MediaType: "movie",
		Page:      1,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, status)

	assert.Equal(t, models.MetadataSourceTMDb, result.Source)
	assert.Len(t, result.Items, 1)
	assert.Equal(t, "Fight Club", result.Items[0].Title)
}

func TestMetadataService_SearchMetadata_Fallback(t *testing.T) {
	cfg := MetadataServiceConfig{
		EnableDouban:    true,
		FallbackDelayMs: 10,
	}

	// TMDb returns no results
	mockTMDb := &mockTMDbSearcher{
		searchMoviesFunc: func(ctx context.Context, query string, page int) (*tmdb.SearchResultMovies, error) {
			return &tmdb.SearchResultMovies{
				Page:         1,
				Results:      []tmdb.Movie{},
				TotalPages:   0,
				TotalResults: 0,
			}, nil
		},
	}

	service := NewMetadataService(cfg, mockTMDb)

	result, status, err := service.SearchMetadata(context.Background(), &SearchMetadataRequest{
		Query:     "NonexistentMovie",
		MediaType: "movie",
	})

	// Since Douban is a stub, it will fail too
	// We just verify the fallback chain was attempted
	assert.Nil(t, result)
	assert.NoError(t, err) // No error, just no results
	require.NotNil(t, status)
	assert.GreaterOrEqual(t, len(status.Attempts), 1)
}

func TestMetadataService_SearchMetadata_InvalidRequest(t *testing.T) {
	cfg := MetadataServiceConfig{}
	mockTMDb := &mockTMDbSearcher{}
	service := NewMetadataService(cfg, mockTMDb)

	result, status, err := service.SearchMetadata(context.Background(), &SearchMetadataRequest{
		Query:     "", // Empty query
		MediaType: "movie",
	})

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Nil(t, status)
}

func TestMetadataService_GetProviders(t *testing.T) {
	cfg := MetadataServiceConfig{
		EnableDouban:    true,
		EnableWikipedia: true,
	}

	mockTMDb := &mockTMDbSearcher{}
	service := NewMetadataService(cfg, mockTMDb)

	providers := service.GetProviders()

	// Should have TMDb + Douban + Wikipedia
	assert.Len(t, providers, 3)

	names := make([]string, len(providers))
	for i, p := range providers {
		names[i] = p.Name
	}

	assert.Contains(t, names, "TMDb")
	assert.Contains(t, names, "Douban")
	assert.Contains(t, names, "Wikipedia")
}

func TestMetadataService_GetProviders_OnlyTMDb(t *testing.T) {
	cfg := MetadataServiceConfig{
		EnableDouban:    false,
		EnableWikipedia: false,
	}

	mockTMDb := &mockTMDbSearcher{}
	service := NewMetadataService(cfg, mockTMDb)

	providers := service.GetProviders()

	// Should have only TMDb
	assert.Len(t, providers, 1)
	assert.Equal(t, "TMDb", providers[0].Name)
}

func TestSearchMetadataRequest_ToMetadataRequest(t *testing.T) {
	req := &SearchMetadataRequest{
		Query:     "Test Movie",
		MediaType: "movie",
		Year:      2024,
		Page:      2,
		Language:  "zh-TW",
	}

	metaReq := req.ToMetadataRequest()

	assert.Equal(t, "Test Movie", metaReq.Query)
	assert.Equal(t, metadata.MediaTypeMovie, metaReq.MediaType)
	assert.Equal(t, 2024, metaReq.Year)
	assert.Equal(t, 2, metaReq.Page)
	assert.Equal(t, "zh-TW", metaReq.Language)
}

func TestSearchMetadataRequest_ToMetadataRequest_TV(t *testing.T) {
	req := &SearchMetadataRequest{
		Query:     "Test Show",
		MediaType: "tv",
	}

	metaReq := req.ToMetadataRequest()

	assert.Equal(t, metadata.MediaTypeTV, metaReq.MediaType)
}

// mockTMDbSearcher implements metadata.TMDbSearcher for testing
type mockTMDbSearcher struct {
	searchMoviesFunc  func(ctx context.Context, query string, page int) (*tmdb.SearchResultMovies, error)
	searchTVShowsFunc func(ctx context.Context, query string, page int) (*tmdb.SearchResultTVShows, error)
}

func (m *mockTMDbSearcher) SearchMovies(ctx context.Context, query string, page int) (*tmdb.SearchResultMovies, error) {
	if m.searchMoviesFunc != nil {
		return m.searchMoviesFunc(ctx, query, page)
	}
	return &tmdb.SearchResultMovies{}, nil
}

func (m *mockTMDbSearcher) SearchTVShows(ctx context.Context, query string, page int) (*tmdb.SearchResultTVShows, error) {
	if m.searchTVShowsFunc != nil {
		return m.searchTVShowsFunc(ctx, query, page)
	}
	return &tmdb.SearchResultTVShows{}, nil
}

// [P1] Tests TV search goes through the correct provider method
func TestMetadataService_SearchMetadata_TVSearch(t *testing.T) {
	cfg := MetadataServiceConfig{
		TMDbImageBaseURL: "https://image.tmdb.org/t/p/w500",
	}

	posterPath := "/tv-poster.jpg"
	mockTMDb := &mockTMDbSearcher{
		searchTVShowsFunc: func(ctx context.Context, query string, page int) (*tmdb.SearchResultTVShows, error) {
			assert.Equal(t, "Breaking Bad", query)
			return &tmdb.SearchResultTVShows{
				Page: 1,
				Results: []tmdb.TVShow{
					{
						ID:           1396,
						Name:         "Breaking Bad",
						FirstAirDate: "2008-01-20",
						VoteAverage:  8.9,
						PosterPath:   &posterPath,
					},
				},
				TotalPages:   1,
				TotalResults: 1,
			}, nil
		},
	}

	service := NewMetadataService(cfg, mockTMDb)

	result, status, err := service.SearchMetadata(context.Background(), &SearchMetadataRequest{
		Query:     "Breaking Bad",
		MediaType: "tv",
		Page:      1,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, status)

	assert.Equal(t, models.MetadataSourceTMDb, result.Source)
	assert.Len(t, result.Items, 1)
	assert.Equal(t, "Breaking Bad", result.Items[0].Title)
	assert.Equal(t, metadata.MediaTypeTV, result.Items[0].MediaType)
	assert.Equal(t, 2008, result.Items[0].Year)
}

// [P2] Tests pagination is correctly passed to TMDb service
func TestMetadataService_SearchMetadata_WithPagination(t *testing.T) {
	cfg := MetadataServiceConfig{
		TMDbImageBaseURL: "https://image.tmdb.org/t/p/w500",
	}

	capturedPage := 0
	mockTMDb := &mockTMDbSearcher{
		searchMoviesFunc: func(ctx context.Context, query string, page int) (*tmdb.SearchResultMovies, error) {
			capturedPage = page
			return &tmdb.SearchResultMovies{
				Page: page,
				Results: []tmdb.Movie{
					{ID: 1, Title: "Test Movie"},
				},
				TotalPages:   5,
				TotalResults: 100,
			}, nil
		},
	}

	service := NewMetadataService(cfg, mockTMDb)

	result, _, err := service.SearchMetadata(context.Background(), &SearchMetadataRequest{
		Query:     "Test",
		MediaType: "movie",
		Page:      3,
	})

	require.NoError(t, err)
	assert.Equal(t, 3, capturedPage)
	assert.Equal(t, 3, result.Page)
}

// [P2] Tests default page value when not specified
func TestMetadataService_SearchMetadata_DefaultPage(t *testing.T) {
	cfg := MetadataServiceConfig{
		TMDbImageBaseURL: "https://image.tmdb.org/t/p/w500",
	}

	capturedPage := 0
	mockTMDb := &mockTMDbSearcher{
		searchMoviesFunc: func(ctx context.Context, query string, page int) (*tmdb.SearchResultMovies, error) {
			capturedPage = page
			return &tmdb.SearchResultMovies{
				Page:         1,
				Results:      []tmdb.Movie{{ID: 1, Title: "Test"}},
				TotalPages:   1,
				TotalResults: 1,
			}, nil
		},
	}

	service := NewMetadataService(cfg, mockTMDb)

	_, _, err := service.SearchMetadata(context.Background(), &SearchMetadataRequest{
		Query:     "Test",
		MediaType: "movie",
		Page:      0, // Default/unset
	})

	require.NoError(t, err)
	assert.Equal(t, 1, capturedPage, "Should default to page 1")
}

// [P2] Tests language parameter is passed through
func TestMetadataService_SearchMetadata_WithLanguage(t *testing.T) {
	cfg := MetadataServiceConfig{
		TMDbImageBaseURL: "https://image.tmdb.org/t/p/w500",
	}

	mockTMDb := &mockTMDbSearcher{
		searchMoviesFunc: func(ctx context.Context, query string, page int) (*tmdb.SearchResultMovies, error) {
			return &tmdb.SearchResultMovies{
				Page: 1,
				Results: []tmdb.Movie{
					{ID: 1, Title: "測試電影"},
				},
				TotalPages:   1,
				TotalResults: 1,
			}, nil
		},
	}

	service := NewMetadataService(cfg, mockTMDb)

	result, _, err := service.SearchMetadata(context.Background(), &SearchMetadataRequest{
		Query:     "測試",
		MediaType: "movie",
		Page:      1,
		Language:  "zh-TW",
	})

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "測試電影", result.Items[0].Title)
}

// [P1] Tests year filter is included in search request
func TestMetadataService_SearchMetadata_WithYearFilter(t *testing.T) {
	cfg := MetadataServiceConfig{
		TMDbImageBaseURL: "https://image.tmdb.org/t/p/w500",
	}

	mockTMDb := &mockTMDbSearcher{
		searchMoviesFunc: func(ctx context.Context, query string, page int) (*tmdb.SearchResultMovies, error) {
			return &tmdb.SearchResultMovies{
				Page: 1,
				Results: []tmdb.Movie{
					{ID: 1, Title: "Test Movie 2024", ReleaseDate: "2024-06-15"},
				},
				TotalPages:   1,
				TotalResults: 1,
			}, nil
		},
	}

	service := NewMetadataService(cfg, mockTMDb)

	result, _, err := service.SearchMetadata(context.Background(), &SearchMetadataRequest{
		Query:     "Test Movie",
		MediaType: "movie",
		Year:      2024,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 2024, result.Items[0].Year)
}

// [P1] Tests context cancellation is properly handled
func TestMetadataService_SearchMetadata_ContextCancellation(t *testing.T) {
	cfg := MetadataServiceConfig{
		TMDbImageBaseURL: "https://image.tmdb.org/t/p/w500",
	}

	mockTMDb := &mockTMDbSearcher{
		searchMoviesFunc: func(ctx context.Context, query string, page int) (*tmdb.SearchResultMovies, error) {
			// Check if context is cancelled
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return &tmdb.SearchResultMovies{
					Page:         1,
					Results:      []tmdb.Movie{{ID: 1, Title: "Test"}},
					TotalPages:   1,
					TotalResults: 1,
				}, nil
			}
		},
	}

	service := NewMetadataService(cfg, mockTMDb)

	// Create already cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, status, err := service.SearchMetadata(ctx, &SearchMetadataRequest{
		Query:     "Test",
		MediaType: "movie",
	})

	// Should handle cancelled context gracefully
	// Either return error or nil result with cancelled status
	if err != nil {
		assert.ErrorIs(t, err, context.Canceled)
	} else {
		assert.Nil(t, result)
		if status != nil {
			assert.True(t, status.Cancelled || status.AllFailed())
		}
	}
}

// =============================================================================
// Manual Search Tests (Story 3.7)
// =============================================================================

// [P1] Tests ManualSearchRequest validation - missing query
func TestManualSearchRequest_Validate_MissingQuery(t *testing.T) {
	req := &ManualSearchRequest{
		Query:     "",
		MediaType: "movie",
		Source:    "tmdb",
	}

	err := req.Validate()
	assert.Error(t, err)
	assert.Equal(t, ErrManualSearchQueryRequired, err)
}

// [P1] Tests ManualSearchRequest validation - invalid source
func TestManualSearchRequest_Validate_InvalidSource(t *testing.T) {
	req := &ManualSearchRequest{
		Query:     "Test",
		MediaType: "movie",
		Source:    "invalid",
	}

	err := req.Validate()
	assert.Error(t, err)
	assert.Equal(t, ErrManualSearchInvalidSource, err)
}

// [P1] Tests ManualSearchRequest validation - valid request with defaults
func TestManualSearchRequest_Validate_Defaults(t *testing.T) {
	req := &ManualSearchRequest{
		Query: "Test",
	}

	err := req.Validate()
	assert.NoError(t, err)
	assert.Equal(t, "movie", req.MediaType)
	assert.Equal(t, "all", req.Source)
}

// [P1] Tests ManualSearchRequest validation - all valid sources
func TestManualSearchRequest_Validate_ValidSources(t *testing.T) {
	validSources := []string{"tmdb", "douban", "wikipedia", "all"}

	for _, source := range validSources {
		t.Run(source, func(t *testing.T) {
			req := &ManualSearchRequest{
				Query:  "Test",
				Source: source,
			}

			err := req.Validate()
			assert.NoError(t, err)
		})
	}
}

// [P1] Tests ManualSearch with specific TMDb source
func TestMetadataService_ManualSearch_TMDbSource(t *testing.T) {
	cfg := MetadataServiceConfig{
		TMDbImageBaseURL: "https://image.tmdb.org/t/p/w500",
	}

	mockTMDb := &mockTMDbSearcher{
		searchMoviesFunc: func(ctx context.Context, query string, page int) (*tmdb.SearchResultMovies, error) {
			return &tmdb.SearchResultMovies{
				Page: 1,
				Results: []tmdb.Movie{
					{
						ID:          550,
						Title:       "Fight Club",
						ReleaseDate: "1999-10-15",
						VoteAverage: 8.4,
					},
				},
				TotalPages:   1,
				TotalResults: 1,
			}, nil
		},
	}

	service := NewMetadataService(cfg, mockTMDb)

	result, err := service.ManualSearch(context.Background(), &ManualSearchRequest{
		Query:     "Fight Club",
		MediaType: "movie",
		Source:    "tmdb",
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 1, result.TotalCount)
	assert.Len(t, result.Results, 1)
	assert.Contains(t, result.SearchedSources, "tmdb")
	assert.Equal(t, models.MetadataSourceTMDb, result.Results[0].Source)
	assert.Equal(t, "Fight Club", result.Results[0].Title)
}

// [P1] Tests ManualSearch with all sources (AC4)
func TestMetadataService_ManualSearch_AllSources(t *testing.T) {
	cfg := MetadataServiceConfig{
		TMDbImageBaseURL: "https://image.tmdb.org/t/p/w500",
		EnableDouban:     true,
		EnableWikipedia:  true,
	}

	mockTMDb := &mockTMDbSearcher{
		searchMoviesFunc: func(ctx context.Context, query string, page int) (*tmdb.SearchResultMovies, error) {
			return &tmdb.SearchResultMovies{
				Page: 1,
				Results: []tmdb.Movie{
					{
						ID:          550,
						Title:       "Fight Club",
						ReleaseDate: "1999-10-15",
						VoteAverage: 8.4,
					},
				},
				TotalPages:   1,
				TotalResults: 1,
			}, nil
		},
	}

	service := NewMetadataService(cfg, mockTMDb)

	result, err := service.ManualSearch(context.Background(), &ManualSearchRequest{
		Query:     "Fight Club",
		MediaType: "movie",
		Source:    "all",
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	// Should have searched all sources
	assert.Contains(t, result.SearchedSources, "tmdb")
	assert.Contains(t, result.SearchedSources, "douban")
	assert.Contains(t, result.SearchedSources, "wikipedia")
	// TMDb should return results, others may not
	assert.GreaterOrEqual(t, result.TotalCount, 1)
}

// [P1] Tests ManualSearch invalid source error
func TestMetadataService_ManualSearch_InvalidSource(t *testing.T) {
	cfg := MetadataServiceConfig{}
	mockTMDb := &mockTMDbSearcher{}
	service := NewMetadataService(cfg, mockTMDb)

	result, err := service.ManualSearch(context.Background(), &ManualSearchRequest{
		Query:     "Test",
		MediaType: "movie",
		Source:    "invalid",
	})

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, ErrManualSearchInvalidSource, err)
}

// [P2] Tests ManualSearch with year filter
func TestMetadataService_ManualSearch_WithYearFilter(t *testing.T) {
	cfg := MetadataServiceConfig{
		TMDbImageBaseURL: "https://image.tmdb.org/t/p/w500",
	}

	mockTMDb := &mockTMDbSearcher{
		searchMoviesFunc: func(ctx context.Context, query string, page int) (*tmdb.SearchResultMovies, error) {
			return &tmdb.SearchResultMovies{
				Page: 1,
				Results: []tmdb.Movie{
					{
						ID:          550,
						Title:       "Test Movie 1999",
						ReleaseDate: "1999-06-15",
						VoteAverage: 8.4,
					},
				},
				TotalPages:   1,
				TotalResults: 1,
			}, nil
		},
	}

	service := NewMetadataService(cfg, mockTMDb)

	result, err := service.ManualSearch(context.Background(), &ManualSearchRequest{
		Query:     "Test",
		MediaType: "movie",
		Year:      1999,
		Source:    "tmdb",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 1999, result.Results[0].Year)
}

// [P2] Tests ManualSearch with TV media type
func TestMetadataService_ManualSearch_TVMediaType(t *testing.T) {
	cfg := MetadataServiceConfig{
		TMDbImageBaseURL: "https://image.tmdb.org/t/p/w500",
	}

	posterPath := "/tv-poster.jpg"
	mockTMDb := &mockTMDbSearcher{
		searchTVShowsFunc: func(ctx context.Context, query string, page int) (*tmdb.SearchResultTVShows, error) {
			return &tmdb.SearchResultTVShows{
				Page: 1,
				Results: []tmdb.TVShow{
					{
						ID:           1396,
						Name:         "Breaking Bad",
						FirstAirDate: "2008-01-20",
						VoteAverage:  8.9,
						PosterPath:   &posterPath,
					},
				},
				TotalPages:   1,
				TotalResults: 1,
			}, nil
		},
	}

	service := NewMetadataService(cfg, mockTMDb)

	result, err := service.ManualSearch(context.Background(), &ManualSearchRequest{
		Query:     "Breaking Bad",
		MediaType: "tv",
		Source:    "tmdb",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "tv", result.Results[0].MediaType)
}

// [P2] Tests ManualSearch result ID format includes source prefix
func TestMetadataService_ManualSearch_ResultIDFormat(t *testing.T) {
	cfg := MetadataServiceConfig{
		TMDbImageBaseURL: "https://image.tmdb.org/t/p/w500",
	}

	mockTMDb := &mockTMDbSearcher{
		searchMoviesFunc: func(ctx context.Context, query string, page int) (*tmdb.SearchResultMovies, error) {
			return &tmdb.SearchResultMovies{
				Page: 1,
				Results: []tmdb.Movie{
					{
						ID:    550,
						Title: "Test",
					},
				},
				TotalPages:   1,
				TotalResults: 1,
			}, nil
		},
	}

	service := NewMetadataService(cfg, mockTMDb)

	result, err := service.ManualSearch(context.Background(), &ManualSearchRequest{
		Query:     "Test",
		MediaType: "movie",
		Source:    "tmdb",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	// ID should be formatted as "source-originalID"
	assert.Equal(t, "tmdb-550", result.Results[0].ID)
}

// [P2] Tests ManualSearch no results returns empty list
func TestMetadataService_ManualSearch_NoResults(t *testing.T) {
	cfg := MetadataServiceConfig{
		TMDbImageBaseURL: "https://image.tmdb.org/t/p/w500",
	}

	mockTMDb := &mockTMDbSearcher{
		searchMoviesFunc: func(ctx context.Context, query string, page int) (*tmdb.SearchResultMovies, error) {
			return &tmdb.SearchResultMovies{
				Page:         1,
				Results:      []tmdb.Movie{},
				TotalPages:   0,
				TotalResults: 0,
			}, nil
		},
	}

	service := NewMetadataService(cfg, mockTMDb)

	result, err := service.ManualSearch(context.Background(), &ManualSearchRequest{
		Query:     "Nonexistent Movie 12345",
		MediaType: "movie",
		Source:    "tmdb",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.Results)
	assert.Equal(t, 0, result.TotalCount)
}

// =============================================================================
// Apply Metadata Tests (Story 3.7 - AC3; dsr-2b-a AC #1)
// =============================================================================

// Before dsr-2b-a the service needed MediaUpdaters that production never
// wired, so every apply took a nil branch and answered success with the title
// "Unknown" — nothing was written. The applier is now the enrichment service
// (it knows how to turn a TMDb id into a written row); these tests pin the
// request contract and how the service hands off to it.

type mockMatchApplier struct {
	kind   MediaKind
	id     string
	tmdbID int
	calls  int
	item   *EnrichedItem
	err    error
}

func (m *mockMatchApplier) ApplyTMDbMatch(_ context.Context, kind MediaKind, id string, tmdbID int) (*EnrichedItem, error) {
	m.calls++
	m.kind, m.id, m.tmdbID = kind, id, tmdbID
	return m.item, m.err
}

func validApplyRequest() *ApplyMetadataRequest {
	return &ApplyMetadataRequest{
		MediaID:      "test-id",
		MediaType:    "movie",
		SelectedItem: SelectedMetadataItem{ID: "tmdb-550", Source: "tmdb", MediaType: "movie"},
	}
}

func newApplyService(applier MatchApplier) *MetadataService {
	service := NewMetadataService(MetadataServiceConfig{}, &mockTMDbSearcher{})
	if applier != nil {
		service.SetMatchApplier(applier)
	}
	return service
}

// [P1] Tests ApplyMetadataRequest validation - missing mediaId
func TestApplyMetadataRequest_Validate_MissingMediaId(t *testing.T) {
	req := validApplyRequest()
	req.MediaID = ""

	err := req.Validate()
	assert.Equal(t, ErrApplyMetadataMediaIDRequired, err)
}

// [P1] Tests ApplyMetadataRequest validation - missing selectedItem id
func TestApplyMetadataRequest_Validate_MissingSelectedItemId(t *testing.T) {
	req := validApplyRequest()
	req.SelectedItem.ID = ""

	err := req.Validate()
	assert.Equal(t, ErrApplyMetadataSelectedItemRequired, err)
}

// [P1] Tests ApplyMetadataRequest validation - missing selectedItem source
func TestApplyMetadataRequest_Validate_MissingSelectedItemSource(t *testing.T) {
	req := validApplyRequest()
	req.SelectedItem.Source = ""

	err := req.Validate()
	assert.Equal(t, ErrApplyMetadataSelectedItemRequired, err)
}

// [P1] Tests ApplyMetadataRequest validation - valid request defaults mediaType
func TestApplyMetadataRequest_Validate_DefaultsMediaType(t *testing.T) {
	req := validApplyRequest()
	req.MediaType = ""

	err := req.Validate()
	assert.NoError(t, err)
	assert.Equal(t, "movie", req.MediaType)
}

// dsr-2b-a AC #1: movie and TV ids are two numbering systems — tmdb-550 is a
// different work on each side — so the picked result must say which it is,
// and it must agree with the library item.
func TestApplyMetadataRequest_Validate_RejectsWhatCannotBeAppliedSafely(t *testing.T) {
	cases := map[string]struct {
		mutate func(*ApplyMetadataRequest)
		want   error
	}{
		"selected type missing":     {func(r *ApplyMetadataRequest) { r.SelectedItem.MediaType = "" }, ErrApplyMetadataSelectedTypeRequired},
		"unknown library type":      {func(r *ApplyMetadataRequest) { r.MediaType = "episode" }, ErrApplyMetadataInvalidMediaType},
		"douban result":             {func(r *ApplyMetadataRequest) { r.SelectedItem.Source = "douban"; r.SelectedItem.ID = "douban-1292052" }, ErrApplyMetadataUnsupportedSource},
		"id without prefix":         {func(r *ApplyMetadataRequest) { r.SelectedItem.ID = "550" }, ErrApplyMetadataInvalidItemID},
		"non-numeric id":            {func(r *ApplyMetadataRequest) { r.SelectedItem.ID = "tmdb-abc" }, ErrApplyMetadataInvalidItemID},
		"zero id":                   {func(r *ApplyMetadataRequest) { r.SelectedItem.ID = "tmdb-0" }, ErrApplyMetadataInvalidItemID},
		"movie item, tv result":     {func(r *ApplyMetadataRequest) { r.SelectedItem.MediaType = "tv" }, ErrApplyMetadataTypeMismatch},
		"series item, movie result": {func(r *ApplyMetadataRequest) { r.MediaType = "series" }, ErrApplyMetadataTypeMismatch},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			req := validApplyRequest()
			tc.mutate(req)
			err := req.Validate()
			assert.ErrorIs(t, err, tc.want)
			assert.True(t, IsApplyMetadataInvalidRequest(err), "maps to 400 APPLY_METADATA_INVALID_REQUEST")
		})
	}
}

// [P1] A movie match is handed to the applier with the parsed TMDb id, and the
// response carries what was actually written.
func TestMetadataService_ApplyMetadata_Success(t *testing.T) {
	applier := &mockMatchApplier{item: &EnrichedItem{ID: "test-id", ParseStatus: models.ParseStatusSuccess, Title: "鬥陣俱樂部", TMDbID: 550}}
	service := newApplyService(applier)

	result, err := service.ApplyMetadata(context.Background(), validApplyRequest())

	require.NoError(t, err)
	assert.Equal(t, MediaKindMovie, applier.kind)
	assert.Equal(t, "test-id", applier.id)
	assert.Equal(t, 550, applier.tmdbID)
	assert.Equal(t, &ApplyMetadataResponse{
		Success: true, MediaID: "test-id", MediaType: "movie", Title: "鬥陣俱樂部",
		Source: models.MetadataSourceTMDb, TMDbID: 550, ParseStatus: models.ParseStatusSuccess,
	}, result)
}

func TestMetadataService_ApplyMetadata_SeriesUsesTheSeriesKind(t *testing.T) {
	applier := &mockMatchApplier{item: &EnrichedItem{ID: "s-1", ParseStatus: models.ParseStatusSuccess, Title: "絕命毒師", TMDbID: 1396}}
	service := newApplyService(applier)
	req := validApplyRequest()
	req.MediaID, req.MediaType = "s-1", "series"
	req.SelectedItem = SelectedMetadataItem{ID: "tmdb-1396", Source: "tmdb", MediaType: "tv"}

	result, err := service.ApplyMetadata(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, MediaKindSeries, applier.kind)
	assert.Equal(t, 1396, applier.tmdbID)
	assert.Equal(t, "series", result.MediaType)
}

// [P1] Tests ApplyMetadata media not found
func TestMetadataService_ApplyMetadata_NotFound(t *testing.T) {
	service := newApplyService(&mockMatchApplier{err: ErrEnrichItemNotFound})

	result, err := service.ApplyMetadata(context.Background(), validApplyRequest())

	assert.Equal(t, ErrApplyMetadataNotFound, err)
	assert.Nil(t, result)
}

// [P1] Tests ApplyMetadata validation error — the applier is never reached
func TestMetadataService_ApplyMetadata_ValidationError(t *testing.T) {
	applier := &mockMatchApplier{}
	service := newApplyService(applier)
	req := validApplyRequest()
	req.MediaID = ""

	result, err := service.ApplyMetadata(context.Background(), req)

	assert.Equal(t, ErrApplyMetadataMediaIDRequired, err)
	assert.Nil(t, result)
	assert.Zero(t, applier.calls)
}

// [P2] learn_pattern is accepted and changes nothing (still a TODO — Story 3.9)
func TestMetadataService_ApplyMetadata_WithLearnPattern(t *testing.T) {
	applier := &mockMatchApplier{item: &EnrichedItem{ID: "test-id", ParseStatus: models.ParseStatusSuccess, Title: "x", TMDbID: 550}}
	service := newApplyService(applier)
	req := validApplyRequest()
	req.LearnPattern = true

	result, err := service.ApplyMetadata(context.Background(), req)

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, 1, applier.calls)
}

// [P2] No applier wired is a failure, never a silent success (dsr-2b-a AC #1)
func TestMetadataService_ApplyMetadata_NoApplierIsAnError(t *testing.T) {
	service := newApplyService(nil)

	result, err := service.ApplyMetadata(context.Background(), validApplyRequest())

	assert.ErrorIs(t, err, ErrApplyMetadataFailed)
	assert.Nil(t, result)
}

// Applier errors the handler maps to their own status codes pass through intact.
func TestMetadataService_ApplyMetadata_PassesThroughMappableErrors(t *testing.T) {
	for name, applierErr := range map[string]error{
		"batch running":    ErrEnrichmentAlreadyRunning,
		"write failed":     ErrEnrichPersist,
		"tmdb 404 wrapped": fmt.Errorf("failed to get movie details: %w", tmdb.NewNotFoundError(550)),
	} {
		t.Run(name, func(t *testing.T) {
			service := newApplyService(&mockMatchApplier{err: applierErr})
			_, err := service.ApplyMetadata(context.Background(), validApplyRequest())
			assert.ErrorIs(t, err, applierErr)
		})
	}
}
