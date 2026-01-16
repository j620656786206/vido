package tmdb

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockClient is a mock implementation of ClientInterface for testing
type MockClient struct {
	SearchMoviesResponses      map[string]*SearchResultMovies
	SearchMoviesErrors         map[string]error
	SearchTVShowsResponses     map[string]*SearchResultTVShows
	SearchTVShowsErrors        map[string]error
	GetMovieDetailsResponses   map[string]*MovieDetails
	GetMovieDetailsErrors      map[string]error
	GetTVShowDetailsResponses  map[string]*TVShowDetails
	GetTVShowDetailsErrors     map[string]error
}

func (m *MockClient) SearchMovies(ctx context.Context, query string, page int) (*SearchResultMovies, error) {
	return m.SearchMoviesWithLanguage(ctx, query, "zh-TW", page)
}

func (m *MockClient) SearchMoviesWithLanguage(ctx context.Context, query string, language string, page int) (*SearchResultMovies, error) {
	if err, ok := m.SearchMoviesErrors[language]; ok {
		return nil, err
	}
	if resp, ok := m.SearchMoviesResponses[language]; ok {
		return resp, nil
	}
	return &SearchResultMovies{}, nil
}

func (m *MockClient) GetMovieDetails(ctx context.Context, movieID int) (*MovieDetails, error) {
	return m.GetMovieDetailsWithLanguage(ctx, movieID, "zh-TW")
}

func (m *MockClient) GetMovieDetailsWithLanguage(ctx context.Context, movieID int, language string) (*MovieDetails, error) {
	if err, ok := m.GetMovieDetailsErrors[language]; ok {
		return nil, err
	}
	if resp, ok := m.GetMovieDetailsResponses[language]; ok {
		return resp, nil
	}
	return &MovieDetails{}, nil
}

func (m *MockClient) SearchTVShows(ctx context.Context, query string, page int) (*SearchResultTVShows, error) {
	return m.SearchTVShowsWithLanguage(ctx, query, "zh-TW", page)
}

func (m *MockClient) SearchTVShowsWithLanguage(ctx context.Context, query string, language string, page int) (*SearchResultTVShows, error) {
	if err, ok := m.SearchTVShowsErrors[language]; ok {
		return nil, err
	}
	if resp, ok := m.SearchTVShowsResponses[language]; ok {
		return resp, nil
	}
	return &SearchResultTVShows{}, nil
}

func (m *MockClient) GetTVShowDetails(ctx context.Context, tvID int) (*TVShowDetails, error) {
	return m.GetTVShowDetailsWithLanguage(ctx, tvID, "zh-TW")
}

func (m *MockClient) GetTVShowDetailsWithLanguage(ctx context.Context, tvID int, language string) (*TVShowDetails, error) {
	if err, ok := m.GetTVShowDetailsErrors[language]; ok {
		return nil, err
	}
	if resp, ok := m.GetTVShowDetailsResponses[language]; ok {
		return resp, nil
	}
	return &TVShowDetails{}, nil
}

func TestNewLanguageFallbackClient(t *testing.T) {
	tests := []struct {
		name      string
		languages []string
		wantLangs []string
	}{
		{
			name:      "with custom languages",
			languages: []string{"ja", "en"},
			wantLangs: []string{"ja", "en"},
		},
		{
			name:      "with nil languages uses defaults",
			languages: nil,
			wantLangs: DefaultFallbackLanguages,
		},
		{
			name:      "with empty languages uses defaults",
			languages: []string{},
			wantLangs: DefaultFallbackLanguages,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockClient{}
			client := NewLanguageFallbackClient(mock, tt.languages)

			assert.NotNil(t, client)
			assert.Equal(t, tt.wantLangs, client.languages)
		})
	}
}

func TestLanguageFallbackClient_SearchMoviesWithFallback(t *testing.T) {
	tests := []struct {
		name          string
		responses     map[string]*SearchResultMovies
		errors        map[string]error
		wantLang      string
		wantResults   int
		wantErr       bool
	}{
		{
			name: "finds results in first language (zh-TW)",
			responses: map[string]*SearchResultMovies{
				"zh-TW": {
					Page:         1,
					Results:      []Movie{{ID: 1, Title: "鬼滅之刃", Overview: "內容"}},
					TotalResults: 1,
				},
			},
			wantLang:    "zh-TW",
			wantResults: 1,
		},
		{
			name: "falls back to second language (zh-CN)",
			responses: map[string]*SearchResultMovies{
				"zh-TW": {
					Page:         1,
					Results:      []Movie{}, // Empty results
					TotalResults: 0,
				},
				"zh-CN": {
					Page:         1,
					Results:      []Movie{{ID: 1, Title: "鬼灭之刃", Overview: "内容"}},
					TotalResults: 1,
				},
			},
			wantLang:    "zh-CN",
			wantResults: 1,
		},
		{
			name: "falls back to third language (en)",
			responses: map[string]*SearchResultMovies{
				"zh-TW": {
					Page:    1,
					Results: []Movie{}, // Empty
				},
				"zh-CN": {
					Page:    1,
					Results: []Movie{}, // Empty
				},
				"en": {
					Page:         1,
					Results:      []Movie{{ID: 1, Title: "Demon Slayer", Overview: "Content"}},
					TotalResults: 1,
				},
			},
			wantLang:    "en",
			wantResults: 1,
		},
		{
			name: "skips language with error and tries next",
			responses: map[string]*SearchResultMovies{
				"zh-CN": {
					Page:         1,
					Results:      []Movie{{ID: 1, Title: "鬼灭之刃", Overview: "内容"}},
					TotalResults: 1,
				},
			},
			errors: map[string]error{
				"zh-TW": errors.New("API error"),
			},
			wantLang:    "zh-CN",
			wantResults: 1,
		},
		{
			name: "returns error when all languages fail",
			errors: map[string]error{
				"zh-TW": errors.New("API error"),
				"zh-CN": errors.New("API error"),
				"en":    errors.New("API error"),
			},
			wantErr: true,
		},
		{
			name: "returns results without overview if only option",
			responses: map[string]*SearchResultMovies{
				"zh-TW": {
					Page:    1,
					Results: []Movie{{ID: 1, Title: "Title Only"}}, // No overview
				},
				"zh-CN": {
					Page:    1,
					Results: []Movie{{ID: 1, Title: "标题"}}, // No overview
				},
				"en": {
					Page:    1,
					Results: []Movie{{ID: 1, Title: "Title"}}, // No overview
				},
			},
			wantLang:    "en", // Last language tried
			wantResults: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockClient{
				SearchMoviesResponses: tt.responses,
				SearchMoviesErrors:    tt.errors,
			}

			client := NewLanguageFallbackClient(mock, nil)
			result, lang, err := client.SearchMoviesWithFallback(context.Background(), "test", 1)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantLang, lang)
			assert.Len(t, result.Results, tt.wantResults)
		})
	}
}

func TestLanguageFallbackClient_SearchTVShowsWithFallback(t *testing.T) {
	tests := []struct {
		name          string
		responses     map[string]*SearchResultTVShows
		errors        map[string]error
		wantLang      string
		wantResults   int
		wantErr       bool
	}{
		{
			name: "finds results in first language (zh-TW)",
			responses: map[string]*SearchResultTVShows{
				"zh-TW": {
					Page:         1,
					Results:      []TVShow{{ID: 1, Name: "絕命毒師", Overview: "內容"}},
					TotalResults: 1,
				},
			},
			wantLang:    "zh-TW",
			wantResults: 1,
		},
		{
			name: "falls back to second language (zh-CN)",
			responses: map[string]*SearchResultTVShows{
				"zh-TW": {
					Page:    1,
					Results: []TVShow{},
				},
				"zh-CN": {
					Page:         1,
					Results:      []TVShow{{ID: 1, Name: "绝命毒师", Overview: "内容"}},
					TotalResults: 1,
				},
			},
			wantLang:    "zh-CN",
			wantResults: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockClient{
				SearchTVShowsResponses: tt.responses,
				SearchTVShowsErrors:    tt.errors,
			}

			client := NewLanguageFallbackClient(mock, nil)
			result, lang, err := client.SearchTVShowsWithFallback(context.Background(), "test", 1)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantLang, lang)
			assert.Len(t, result.Results, tt.wantResults)
		})
	}
}

func TestLanguageFallbackClient_GetMovieDetailsWithFallback(t *testing.T) {
	tests := []struct {
		name      string
		responses map[string]*MovieDetails
		errors    map[string]error
		wantLang  string
		wantTitle string
		wantErr   bool
	}{
		{
			name: "finds details in first language",
			responses: map[string]*MovieDetails{
				"zh-TW": {
					Movie: Movie{
						ID:       550,
						Title:    "鬥陣俱樂部",
						Overview: "內容",
					},
				},
			},
			wantLang:  "zh-TW",
			wantTitle: "鬥陣俱樂部",
		},
		{
			name: "falls back when no overview",
			responses: map[string]*MovieDetails{
				"zh-TW": {
					Movie: Movie{
						ID:    550,
						Title: "Title", // No overview
					},
				},
				"zh-CN": {
					Movie: Movie{
						ID:    550,
						Title: "标题", // No overview
					},
				},
				"en": {
					Movie: Movie{
						ID:       550,
						Title:    "Fight Club",
						Overview: "Content",
					},
				},
			},
			wantLang:  "en",
			wantTitle: "Fight Club",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockClient{
				GetMovieDetailsResponses: tt.responses,
				GetMovieDetailsErrors:    tt.errors,
			}

			client := NewLanguageFallbackClient(mock, nil)
			result, lang, err := client.GetMovieDetailsWithFallback(context.Background(), 550)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantLang, lang)
			assert.Equal(t, tt.wantTitle, result.Title)
		})
	}
}

func TestLanguageFallbackClient_GetTVShowDetailsWithFallback(t *testing.T) {
	tests := []struct {
		name     string
		responses map[string]*TVShowDetails
		errors    map[string]error
		wantLang string
		wantName string
		wantErr  bool
	}{
		{
			name: "finds details in first language",
			responses: map[string]*TVShowDetails{
				"zh-TW": {
					TVShow: TVShow{
						ID:       1396,
						Name:     "絕命毒師",
						Overview: "內容",
					},
				},
			},
			wantLang: "zh-TW",
			wantName: "絕命毒師",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockClient{
				GetTVShowDetailsResponses: tt.responses,
				GetTVShowDetailsErrors:    tt.errors,
			}

			client := NewLanguageFallbackClient(mock, nil)
			result, lang, err := client.GetTVShowDetailsWithFallback(context.Background(), 1396)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantLang, lang)
			assert.Equal(t, tt.wantName, result.Name)
		})
	}
}

func TestHasLocalizedMovieContent(t *testing.T) {
	tests := []struct {
		name   string
		movies []Movie
		want   bool
	}{
		{
			name:   "empty slice",
			movies: []Movie{},
			want:   false,
		},
		{
			name: "movie with title and overview",
			movies: []Movie{
				{Title: "Title", Overview: "Overview"},
			},
			want: true,
		},
		{
			name: "movie with only title",
			movies: []Movie{
				{Title: "Title"},
			},
			want: false,
		},
		{
			name: "movie with only overview",
			movies: []Movie{
				{Overview: "Overview"},
			},
			want: false,
		},
		{
			name: "multiple movies, one with content",
			movies: []Movie{
				{Title: "Title1"},
				{Title: "Title2", Overview: "Overview"},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasLocalizedMovieContent(tt.movies)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHasLocalizedTVShowContent(t *testing.T) {
	tests := []struct {
		name  string
		shows []TVShow
		want  bool
	}{
		{
			name:  "empty slice",
			shows: []TVShow{},
			want:  false,
		},
		{
			name: "show with name and overview",
			shows: []TVShow{
				{Name: "Name", Overview: "Overview"},
			},
			want: true,
		},
		{
			name: "show with only name",
			shows: []TVShow{
				{Name: "Name"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasLocalizedTVShowContent(tt.shows)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLanguageFallbackClient_InterfaceCompliance(t *testing.T) {
	var _ LanguageFallbackClientInterface = (*LanguageFallbackClient)(nil)
}
