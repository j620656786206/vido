package metadata

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/tmdb"
)

// mockTMDbSearcher implements TMDbSearcher for testing
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

func TestNewTMDbProvider(t *testing.T) {
	service := &mockTMDbSearcher{}
	provider := NewTMDbProvider(service, TMDbProviderConfig{
		ImageBaseURL: "https://image.tmdb.org/t/p/",
	})

	assert.NotNil(t, provider)
	assert.Equal(t, "TMDb", provider.Name())
	assert.Equal(t, models.MetadataSourceTMDb, provider.Source())
	assert.True(t, provider.IsAvailable())
	assert.Equal(t, ProviderStatusAvailable, provider.Status())
}

func TestTMDbProvider_Search_Movies(t *testing.T) {
	posterPath := "/abc123.jpg"
	backdropPath := "/xyz789.jpg"

	service := &mockTMDbSearcher{
		searchMoviesFunc: func(ctx context.Context, query string, page int) (*tmdb.SearchResultMovies, error) {
			return &tmdb.SearchResultMovies{
				Page: 1,
				Results: []tmdb.Movie{
					{
						ID:               550,
						Title:            "Fight Club",
						OriginalTitle:    "Fight Club",
						Overview:         "A ticking-time-bomb insomniac...",
						ReleaseDate:      "1999-10-15",
						PosterPath:       &posterPath,
						BackdropPath:     &backdropPath,
						VoteAverage:      8.4,
						VoteCount:        26280,
						Popularity:       61.416,
						GenreIDs:         []int{18, 53},
						OriginalLanguage: "en",
					},
				},
				TotalPages:   1,
				TotalResults: 1,
			}, nil
		},
	}

	provider := NewTMDbProvider(service, TMDbProviderConfig{
		ImageBaseURL: "https://image.tmdb.org/t/p/w500",
	})

	result, err := provider.Search(context.Background(), &SearchRequest{
		Query:     "Fight Club",
		MediaType: MediaTypeMovie,
		Page:      1,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, models.MetadataSourceTMDb, result.Source)
	assert.Equal(t, 1, result.TotalCount)
	assert.Len(t, result.Items, 1)

	item := result.Items[0]
	assert.Equal(t, "550", item.ID)
	assert.Equal(t, "Fight Club", item.Title)
	assert.Equal(t, 1999, item.Year)
	assert.Equal(t, "1999-10-15", item.ReleaseDate)
	assert.Equal(t, 8.4, item.Rating)
	assert.Equal(t, MediaTypeMovie, item.MediaType)
	assert.Equal(t, "https://image.tmdb.org/t/p/w500/abc123.jpg", item.PosterURL)
	assert.Equal(t, "https://image.tmdb.org/t/p/w500/xyz789.jpg", item.BackdropURL)
}

func TestTMDbProvider_Search_TVShows(t *testing.T) {
	posterPath := "/poster.jpg"

	service := &mockTMDbSearcher{
		searchTVShowsFunc: func(ctx context.Context, query string, page int) (*tmdb.SearchResultTVShows, error) {
			return &tmdb.SearchResultTVShows{
				Page: 1,
				Results: []tmdb.TVShow{
					{
						ID:               1396,
						Name:             "Breaking Bad",
						OriginalName:     "Breaking Bad",
						Overview:         "When Walter White...",
						FirstAirDate:     "2008-01-20",
						PosterPath:       &posterPath,
						VoteAverage:      8.9,
						VoteCount:        12345,
						Popularity:       369.594,
						GenreIDs:         []int{18, 80},
						OriginalLanguage: "en",
					},
				},
				TotalPages:   1,
				TotalResults: 1,
			}, nil
		},
	}

	provider := NewTMDbProvider(service, TMDbProviderConfig{
		ImageBaseURL: "https://image.tmdb.org/t/p/w500",
	})

	result, err := provider.Search(context.Background(), &SearchRequest{
		Query:     "Breaking Bad",
		MediaType: MediaTypeTV,
		Page:      1,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, models.MetadataSourceTMDb, result.Source)
	assert.Equal(t, 1, result.TotalCount)
	assert.Len(t, result.Items, 1)

	item := result.Items[0]
	assert.Equal(t, "1396", item.ID)
	assert.Equal(t, "Breaking Bad", item.Title)
	assert.Equal(t, 2008, item.Year)
	assert.Equal(t, MediaTypeTV, item.MediaType)
}

func TestTMDbProvider_Search_Error(t *testing.T) {
	service := &mockTMDbSearcher{
		searchMoviesFunc: func(ctx context.Context, query string, page int) (*tmdb.SearchResultMovies, error) {
			return nil, errors.New("API error")
		},
	}

	provider := NewTMDbProvider(service, TMDbProviderConfig{})

	result, err := provider.Search(context.Background(), &SearchRequest{
		Query:     "Test",
		MediaType: MediaTypeMovie,
	})

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestTMDbProvider_Search_EmptyResults(t *testing.T) {
	service := &mockTMDbSearcher{
		searchMoviesFunc: func(ctx context.Context, query string, page int) (*tmdb.SearchResultMovies, error) {
			return &tmdb.SearchResultMovies{
				Page:         1,
				Results:      []tmdb.Movie{},
				TotalPages:   0,
				TotalResults: 0,
			}, nil
		},
	}

	provider := NewTMDbProvider(service, TMDbProviderConfig{})

	result, err := provider.Search(context.Background(), &SearchRequest{
		Query:     "NonexistentMovie12345",
		MediaType: MediaTypeMovie,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 0, result.TotalCount)
	assert.Empty(t, result.Items)
}

func TestTMDbProvider_Search_InvalidRequest(t *testing.T) {
	service := &mockTMDbSearcher{}
	provider := NewTMDbProvider(service, TMDbProviderConfig{})

	result, err := provider.Search(context.Background(), &SearchRequest{
		Query:     "", // Empty query
		MediaType: MediaTypeMovie,
	})

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestTMDbProvider_SetAvailable(t *testing.T) {
	service := &mockTMDbSearcher{}
	provider := NewTMDbProvider(service, TMDbProviderConfig{})

	assert.True(t, provider.IsAvailable())
	assert.Equal(t, ProviderStatusAvailable, provider.Status())

	provider.SetAvailable(false)
	assert.False(t, provider.IsAvailable())
	assert.Equal(t, ProviderStatusUnavailable, provider.Status())

	provider.SetAvailable(true)
	assert.True(t, provider.IsAvailable())
	assert.Equal(t, ProviderStatusAvailable, provider.Status())
}

func TestTMDbProvider_ImplementsInterface(t *testing.T) {
	service := &mockTMDbSearcher{}
	provider := NewTMDbProvider(service, TMDbProviderConfig{})

	// Compile-time interface verification
	var _ MetadataProvider = provider
}

func TestExtractYear(t *testing.T) {
	tests := []struct {
		date     string
		expected int
	}{
		{"1999-10-15", 1999},
		{"2008-01-20", 2008},
		{"2024-12-31", 2024},
		{"invalid", 0},
		{"", 0},
		{"1999", 0}, // Not in YYYY-MM-DD format
	}

	for _, tt := range tests {
		t.Run(tt.date, func(t *testing.T) {
			assert.Equal(t, tt.expected, extractYear(tt.date))
		})
	}
}

func TestBuildImageURL(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		path     *string
		expected string
	}{
		{
			name:     "valid path",
			baseURL:  "https://image.tmdb.org/t/p/w500",
			path:     strPtr("/abc123.jpg"),
			expected: "https://image.tmdb.org/t/p/w500/abc123.jpg",
		},
		{
			name:     "nil path",
			baseURL:  "https://image.tmdb.org/t/p/w500",
			path:     nil,
			expected: "",
		},
		{
			name:     "empty path",
			baseURL:  "https://image.tmdb.org/t/p/w500",
			path:     strPtr(""),
			expected: "",
		},
		{
			name:     "no base URL",
			baseURL:  "",
			path:     strPtr("/abc123.jpg"),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, buildImageURL(tt.baseURL, tt.path))
		})
	}
}

func strPtr(s string) *string {
	return &s
}
