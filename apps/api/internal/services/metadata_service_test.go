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
