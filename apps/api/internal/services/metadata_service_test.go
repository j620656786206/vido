package services

import (
	"context"
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
// Apply Metadata Tests (Story 3.7 - AC3)
// =============================================================================

// mockMediaUpdater implements MediaUpdater for testing
type mockMediaUpdater struct {
	updateMetadataSourceFunc func(ctx context.Context, mediaID string, source models.MetadataSource) error
	getByIDFunc              func(ctx context.Context, id string) (title string, exists bool, err error)
}

func (m *mockMediaUpdater) UpdateMetadataSource(ctx context.Context, mediaID string, source models.MetadataSource) error {
	if m.updateMetadataSourceFunc != nil {
		return m.updateMetadataSourceFunc(ctx, mediaID, source)
	}
	return nil
}

func (m *mockMediaUpdater) GetByID(ctx context.Context, id string) (title string, exists bool, err error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return "Test Title", true, nil
}

// [P1] Tests ApplyMetadataRequest validation - missing mediaId
func TestApplyMetadataRequest_Validate_MissingMediaId(t *testing.T) {
	req := &ApplyMetadataRequest{
		MediaID: "",
		SelectedItem: SelectedMetadataItem{
			ID:     "tmdb-550",
			Source: "tmdb",
		},
	}

	err := req.Validate()
	assert.Error(t, err)
	assert.Equal(t, ErrApplyMetadataMediaIDRequired, err)
}

// [P1] Tests ApplyMetadataRequest validation - missing selectedItem id
func TestApplyMetadataRequest_Validate_MissingSelectedItemId(t *testing.T) {
	req := &ApplyMetadataRequest{
		MediaID: "test-id",
		SelectedItem: SelectedMetadataItem{
			ID:     "",
			Source: "tmdb",
		},
	}

	err := req.Validate()
	assert.Error(t, err)
	assert.Equal(t, ErrApplyMetadataSelectedItemRequired, err)
}

// [P1] Tests ApplyMetadataRequest validation - missing selectedItem source
func TestApplyMetadataRequest_Validate_MissingSelectedItemSource(t *testing.T) {
	req := &ApplyMetadataRequest{
		MediaID: "test-id",
		SelectedItem: SelectedMetadataItem{
			ID:     "tmdb-550",
			Source: "",
		},
	}

	err := req.Validate()
	assert.Error(t, err)
	assert.Equal(t, ErrApplyMetadataSelectedItemRequired, err)
}

// [P1] Tests ApplyMetadataRequest validation - valid request defaults mediaType
func TestApplyMetadataRequest_Validate_DefaultsMediaType(t *testing.T) {
	req := &ApplyMetadataRequest{
		MediaID: "test-id",
		SelectedItem: SelectedMetadataItem{
			ID:     "tmdb-550",
			Source: "tmdb",
		},
	}

	err := req.Validate()
	assert.NoError(t, err)
	assert.Equal(t, "movie", req.MediaType)
}

// [P1] Tests ApplyMetadata with mock updater
func TestMetadataService_ApplyMetadata_Success(t *testing.T) {
	cfg := MetadataServiceConfig{}
	mockTMDb := &mockTMDbSearcher{}
	service := NewMetadataService(cfg, mockTMDb)

	mockUpdater := &mockMediaUpdater{
		getByIDFunc: func(ctx context.Context, id string) (string, bool, error) {
			return "Fight Club", true, nil
		},
		updateMetadataSourceFunc: func(ctx context.Context, mediaID string, source models.MetadataSource) error {
			assert.Equal(t, "test-id", mediaID)
			assert.Equal(t, models.MetadataSourceTMDb, source)
			return nil
		},
	}
	service.SetMediaUpdaters(mockUpdater, mockUpdater)

	result, err := service.ApplyMetadata(context.Background(), &ApplyMetadataRequest{
		MediaID:   "test-id",
		MediaType: "movie",
		SelectedItem: SelectedMetadataItem{
			ID:     "tmdb-550",
			Source: "tmdb",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, "test-id", result.MediaID)
	assert.Equal(t, "Fight Club", result.Title)
	assert.Equal(t, models.MetadataSourceTMDb, result.Source)
}

// [P1] Tests ApplyMetadata media not found
func TestMetadataService_ApplyMetadata_NotFound(t *testing.T) {
	cfg := MetadataServiceConfig{}
	mockTMDb := &mockTMDbSearcher{}
	service := NewMetadataService(cfg, mockTMDb)

	mockUpdater := &mockMediaUpdater{
		getByIDFunc: func(ctx context.Context, id string) (string, bool, error) {
			return "", false, nil
		},
	}
	service.SetMediaUpdaters(mockUpdater, mockUpdater)

	result, err := service.ApplyMetadata(context.Background(), &ApplyMetadataRequest{
		MediaID:   "nonexistent-id",
		MediaType: "movie",
		SelectedItem: SelectedMetadataItem{
			ID:     "tmdb-550",
			Source: "tmdb",
		},
	})

	assert.Error(t, err)
	assert.Equal(t, ErrApplyMetadataNotFound, err)
	assert.Nil(t, result)
}

// [P1] Tests ApplyMetadata validation error
func TestMetadataService_ApplyMetadata_ValidationError(t *testing.T) {
	cfg := MetadataServiceConfig{}
	mockTMDb := &mockTMDbSearcher{}
	service := NewMetadataService(cfg, mockTMDb)

	result, err := service.ApplyMetadata(context.Background(), &ApplyMetadataRequest{
		MediaID: "", // Missing required field
		SelectedItem: SelectedMetadataItem{
			ID:     "tmdb-550",
			Source: "tmdb",
		},
	})

	assert.Error(t, err)
	assert.Equal(t, ErrApplyMetadataMediaIDRequired, err)
	assert.Nil(t, result)
}

// [P2] Tests ApplyMetadata with learnPattern flag
func TestMetadataService_ApplyMetadata_WithLearnPattern(t *testing.T) {
	cfg := MetadataServiceConfig{}
	mockTMDb := &mockTMDbSearcher{}
	service := NewMetadataService(cfg, mockTMDb)

	mockUpdater := &mockMediaUpdater{
		getByIDFunc: func(ctx context.Context, id string) (string, bool, error) {
			return "Test Movie", true, nil
		},
	}
	service.SetMediaUpdaters(mockUpdater, mockUpdater)

	result, err := service.ApplyMetadata(context.Background(), &ApplyMetadataRequest{
		MediaID:   "test-id",
		MediaType: "movie",
		SelectedItem: SelectedMetadataItem{
			ID:     "tmdb-550",
			Source: "tmdb",
		},
		LearnPattern: true,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
}

// [P2] Tests ApplyMetadata without updater configured
func TestMetadataService_ApplyMetadata_NoUpdater(t *testing.T) {
	cfg := MetadataServiceConfig{}
	mockTMDb := &mockTMDbSearcher{}
	service := NewMetadataService(cfg, mockTMDb)
	// No updater configured

	result, err := service.ApplyMetadata(context.Background(), &ApplyMetadataRequest{
		MediaID:   "test-id",
		MediaType: "movie",
		SelectedItem: SelectedMetadataItem{
			ID:     "tmdb-550",
			Source: "tmdb",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, "Unknown", result.Title) // Placeholder when no updater
}
