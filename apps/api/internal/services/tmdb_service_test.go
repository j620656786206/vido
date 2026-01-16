package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/tmdb"
)

// MockCacheService is a mock implementation of tmdb.CacheServiceInterface
type MockCacheService struct {
	SearchMoviesResponse      *tmdb.SearchResultMovies
	SearchMoviesError         error
	SearchTVShowsResponse     *tmdb.SearchResultTVShows
	SearchTVShowsError        error
	GetMovieDetailsResponse   *tmdb.MovieDetails
	GetMovieDetailsError      error
	GetTVShowDetailsResponse  *tmdb.TVShowDetails
	GetTVShowDetailsError     error
}

func (m *MockCacheService) SearchMovies(ctx context.Context, query string, page int) (*tmdb.SearchResultMovies, error) {
	if m.SearchMoviesError != nil {
		return nil, m.SearchMoviesError
	}
	return m.SearchMoviesResponse, nil
}

func (m *MockCacheService) SearchTVShows(ctx context.Context, query string, page int) (*tmdb.SearchResultTVShows, error) {
	if m.SearchTVShowsError != nil {
		return nil, m.SearchTVShowsError
	}
	return m.SearchTVShowsResponse, nil
}

func (m *MockCacheService) GetMovieDetails(ctx context.Context, movieID int) (*tmdb.MovieDetails, error) {
	if m.GetMovieDetailsError != nil {
		return nil, m.GetMovieDetailsError
	}
	return m.GetMovieDetailsResponse, nil
}

func (m *MockCacheService) GetTVShowDetails(ctx context.Context, tvID int) (*tmdb.TVShowDetails, error) {
	if m.GetTVShowDetailsError != nil {
		return nil, m.GetTVShowDetailsError
	}
	return m.GetTVShowDetailsResponse, nil
}

func TestTMDbService_SearchMovies(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		page        int
		mockResp    *tmdb.SearchResultMovies
		mockErr     error
		wantErr     bool
		wantErrCode string
	}{
		{
			name:  "successful search",
			query: "鬼滅之刃",
			page:  1,
			mockResp: &tmdb.SearchResultMovies{
				Page: 1,
				Results: []tmdb.Movie{
					{ID: 1, Title: "鬼滅之刃"},
				},
				TotalResults: 1,
			},
		},
		{
			name:        "empty query",
			query:       "",
			page:        1,
			wantErr:     true,
			wantErrCode: tmdb.ErrCodeBadRequest,
		},
		{
			name:  "negative page defaults to 1",
			query: "test",
			page:  -1,
			mockResp: &tmdb.SearchResultMovies{
				Page:    1,
				Results: []tmdb.Movie{},
			},
		},
		{
			name:    "API error",
			query:   "test",
			page:    1,
			mockErr: errors.New("API error"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockCacheService{
				SearchMoviesResponse: tt.mockResp,
				SearchMoviesError:    tt.mockErr,
			}

			service := NewTMDbServiceWithCacheService(mock)
			result, err := service.SearchMovies(context.Background(), tt.query, tt.page)

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrCode != "" {
					tmdbErr, ok := err.(*tmdb.TMDbError)
					require.True(t, ok)
					assert.Equal(t, tt.wantErrCode, tmdbErr.Code)
				}
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
		})
	}
}

func TestTMDbService_SearchTVShows(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		page        int
		mockResp    *tmdb.SearchResultTVShows
		mockErr     error
		wantErr     bool
		wantErrCode string
	}{
		{
			name:  "successful search",
			query: "Breaking Bad",
			page:  1,
			mockResp: &tmdb.SearchResultTVShows{
				Page: 1,
				Results: []tmdb.TVShow{
					{ID: 1396, Name: "Breaking Bad"},
				},
				TotalResults: 1,
			},
		},
		{
			name:        "empty query",
			query:       "",
			page:        1,
			wantErr:     true,
			wantErrCode: tmdb.ErrCodeBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockCacheService{
				SearchTVShowsResponse: tt.mockResp,
				SearchTVShowsError:    tt.mockErr,
			}

			service := NewTMDbServiceWithCacheService(mock)
			result, err := service.SearchTVShows(context.Background(), tt.query, tt.page)

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrCode != "" {
					tmdbErr, ok := err.(*tmdb.TMDbError)
					require.True(t, ok)
					assert.Equal(t, tt.wantErrCode, tmdbErr.Code)
				}
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
		})
	}
}

func TestTMDbService_GetMovieDetails(t *testing.T) {
	tests := []struct {
		name        string
		movieID     int
		mockResp    *tmdb.MovieDetails
		mockErr     error
		wantErr     bool
		wantErrCode string
	}{
		{
			name:    "successful get",
			movieID: 550,
			mockResp: &tmdb.MovieDetails{
				Movie: tmdb.Movie{
					ID:    550,
					Title: "Fight Club",
				},
			},
		},
		{
			name:        "invalid ID - zero",
			movieID:     0,
			wantErr:     true,
			wantErrCode: tmdb.ErrCodeBadRequest,
		},
		{
			name:        "invalid ID - negative",
			movieID:     -1,
			wantErr:     true,
			wantErrCode: tmdb.ErrCodeBadRequest,
		},
		{
			name:    "API error",
			movieID: 550,
			mockErr: tmdb.NewNotFoundError(550),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockCacheService{
				GetMovieDetailsResponse: tt.mockResp,
				GetMovieDetailsError:    tt.mockErr,
			}

			service := NewTMDbServiceWithCacheService(mock)
			result, err := service.GetMovieDetails(context.Background(), tt.movieID)

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrCode != "" {
					tmdbErr, ok := err.(*tmdb.TMDbError)
					require.True(t, ok)
					assert.Equal(t, tt.wantErrCode, tmdbErr.Code)
				}
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.mockResp.Title, result.Title)
		})
	}
}

func TestTMDbService_GetTVShowDetails(t *testing.T) {
	tests := []struct {
		name        string
		tvID        int
		mockResp    *tmdb.TVShowDetails
		mockErr     error
		wantErr     bool
		wantErrCode string
	}{
		{
			name: "successful get",
			tvID: 1396,
			mockResp: &tmdb.TVShowDetails{
				TVShow: tmdb.TVShow{
					ID:   1396,
					Name: "Breaking Bad",
				},
			},
		},
		{
			name:        "invalid ID - zero",
			tvID:        0,
			wantErr:     true,
			wantErrCode: tmdb.ErrCodeBadRequest,
		},
		{
			name:        "invalid ID - negative",
			tvID:        -1,
			wantErr:     true,
			wantErrCode: tmdb.ErrCodeBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockCacheService{
				GetTVShowDetailsResponse: tt.mockResp,
				GetTVShowDetailsError:    tt.mockErr,
			}

			service := NewTMDbServiceWithCacheService(mock)
			result, err := service.GetTVShowDetails(context.Background(), tt.tvID)

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrCode != "" {
					tmdbErr, ok := err.(*tmdb.TMDbError)
					require.True(t, ok)
					assert.Equal(t, tt.wantErrCode, tmdbErr.Code)
				}
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.mockResp.Name, result.Name)
		})
	}
}

func TestTMDbService_InterfaceCompliance(t *testing.T) {
	var _ TMDbServiceInterface = (*TMDbService)(nil)
}
