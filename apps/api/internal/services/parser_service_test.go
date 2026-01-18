package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/parser"
)

// mockAIService implements AIServiceInterface for testing.
type mockAIService struct {
	parseFunc    func(ctx context.Context, filename string) (*ai.ParseResponse, error)
	isConfigured bool
	parseCalled  bool
}

func (m *mockAIService) ParseFilename(ctx context.Context, filename string) (*ai.ParseResponse, error) {
	m.parseCalled = true
	if m.parseFunc != nil {
		return m.parseFunc(ctx, filename)
	}
	return &ai.ParseResponse{
		Title:     "Default AI Title",
		MediaType: "movie",
	}, nil
}

func (m *mockAIService) ClearCache(ctx context.Context) (int64, error)        { return 0, nil }
func (m *mockAIService) ClearExpiredCache(ctx context.Context) (int64, error) { return 0, nil }
func (m *mockAIService) GetCacheStats(ctx context.Context) (*ai.CacheStats, error) {
	return &ai.CacheStats{}, nil
}
func (m *mockAIService) IsConfigured() bool { return m.isConfigured }

func TestParserService_ParseFilename_Movie(t *testing.T) {
	service := NewParserService()

	tests := []struct {
		name     string
		filename string
		want     struct {
			status    parser.ParseStatus
			mediaType parser.MediaType
			title     string
			year      int
		}
	}{
		{
			name:     "standard movie",
			filename: "The.Matrix.1999.1080p.BluRay.mkv",
			want: struct {
				status    parser.ParseStatus
				mediaType parser.MediaType
				title     string
				year      int
			}{parser.ParseStatusSuccess, parser.MediaTypeMovie, "The Matrix", 1999},
		},
		{
			name:     "movie with parentheses year",
			filename: "Inception (2010) 1080p.mkv",
			want: struct {
				status    parser.ParseStatus
				mediaType parser.MediaType
				title     string
				year      int
			}{parser.ParseStatusSuccess, parser.MediaTypeMovie, "Inception", 2010},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.ParseFilename(tt.filename)

			require.NotNil(t, result)
			assert.Equal(t, tt.want.status, result.Status)
			assert.Equal(t, tt.want.mediaType, result.MediaType)
			assert.Equal(t, tt.want.title, result.Title)
			assert.Equal(t, tt.want.year, result.Year)
		})
	}
}

func TestParserService_ParseFilename_TVShow(t *testing.T) {
	service := NewParserService()

	tests := []struct {
		name     string
		filename string
		want     struct {
			status    parser.ParseStatus
			mediaType parser.MediaType
			title     string
			season    int
			episode   int
		}
	}{
		{
			name:     "standard TV show",
			filename: "Breaking.Bad.S01E05.720p.BluRay.mkv",
			want: struct {
				status    parser.ParseStatus
				mediaType parser.MediaType
				title     string
				season    int
				episode   int
			}{parser.ParseStatusSuccess, parser.MediaTypeTVShow, "Breaking Bad", 1, 5},
		},
		{
			name:     "TV show with 1x05 format",
			filename: "House.1x13.720p.mkv",
			want: struct {
				status    parser.ParseStatus
				mediaType parser.MediaType
				title     string
				season    int
				episode   int
			}{parser.ParseStatusSuccess, parser.MediaTypeTVShow, "House", 1, 13},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.ParseFilename(tt.filename)

			require.NotNil(t, result)
			assert.Equal(t, tt.want.status, result.Status)
			assert.Equal(t, tt.want.mediaType, result.MediaType)
			assert.Equal(t, tt.want.title, result.Title)
			assert.Equal(t, tt.want.season, result.Season)
			assert.Equal(t, tt.want.episode, result.Episode)
		})
	}
}

func TestParserService_ParseFilename_NeedsAI(t *testing.T) {
	service := NewParserService()

	tests := []struct {
		name     string
		filename string
	}{
		{"anime fansub", "[Leopard-Raws] Kimetsu no Yaiba - 26 (BD 1920x1080).mkv"},
		{"Chinese fansub", "【幻櫻字幕組】【4月新番】我的英雄學院 第01話 1080P.mp4"},
		{"no pattern match", "random_video_file.mkv"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.ParseFilename(tt.filename)

			require.NotNil(t, result)
			assert.Equal(t, parser.ParseStatusNeedsAI, result.Status)
			assert.Equal(t, parser.MediaTypeUnknown, result.MediaType)
		})
	}
}

func TestParserService_ParseBatch(t *testing.T) {
	service := NewParserService()

	filenames := []string{
		"The.Matrix.1999.1080p.BluRay.mkv",
		"Breaking.Bad.S01E05.720p.BluRay.mkv",
		"[Leopard-Raws] Kimetsu no Yaiba - 26.mkv",
	}

	results := service.ParseBatch(filenames)

	require.Len(t, results, 3)
	assert.Equal(t, parser.ParseStatusSuccess, results[0].Status)
	assert.Equal(t, parser.MediaTypeMovie, results[0].MediaType)
	assert.Equal(t, parser.ParseStatusSuccess, results[1].Status)
	assert.Equal(t, parser.MediaTypeTVShow, results[1].MediaType)
	assert.Equal(t, parser.ParseStatusNeedsAI, results[2].Status)
}

func TestParserService_ImplementsInterface(t *testing.T) {
	var _ ParserServiceInterface = (*ParserService)(nil)
}

func TestParserServiceWithAI_DelegatesToAI_WhenRegexFails(t *testing.T) {
	aiService := &mockAIService{
		isConfigured: true,
		parseFunc: func(ctx context.Context, filename string) (*ai.ParseResponse, error) {
			return &ai.ParseResponse{
				Title:       "Kimetsu no Yaiba",
				Season:      1,
				Episode:     26,
				MediaType:   "tv",
				Quality:     "1080p",
				FansubGroup: "Leopard-Raws",
				Confidence:  0.92,
			}, nil
		},
	}
	service := NewParserServiceWithAI(aiService)

	// This filename can't be parsed by regex
	filename := "[Leopard-Raws] Kimetsu no Yaiba - 26 (BD 1920x1080).mkv"
	result := service.ParseFilename(filename)

	require.NotNil(t, result)
	assert.True(t, aiService.parseCalled)
	assert.Equal(t, parser.ParseStatusSuccess, result.Status)
	assert.Equal(t, parser.MediaTypeTVShow, result.MediaType)
	assert.Equal(t, "Kimetsu no Yaiba", result.Title)
	assert.Equal(t, 1, result.Season)
	assert.Equal(t, 26, result.Episode)
	assert.Equal(t, 92, result.Confidence)
}

func TestParserServiceWithAI_SkipsAI_WhenRegexSucceeds(t *testing.T) {
	aiService := &mockAIService{isConfigured: true}
	service := NewParserServiceWithAI(aiService)

	// This filename CAN be parsed by regex
	filename := "Breaking.Bad.S01E05.720p.BluRay.mkv"
	result := service.ParseFilename(filename)

	require.NotNil(t, result)
	assert.False(t, aiService.parseCalled) // AI should NOT be called
	assert.Equal(t, parser.ParseStatusSuccess, result.Status)
	assert.Equal(t, parser.MediaTypeTVShow, result.MediaType)
	assert.Equal(t, "Breaking Bad", result.Title)
}

func TestParserServiceWithAI_ReturnsNeedsAI_WhenAIFails(t *testing.T) {
	aiService := &mockAIService{
		isConfigured: true,
		parseFunc: func(ctx context.Context, filename string) (*ai.ParseResponse, error) {
			return nil, errors.New("AI provider error")
		},
	}
	service := NewParserServiceWithAI(aiService)

	filename := "[Leopard-Raws] Kimetsu no Yaiba - 26.mkv"
	result := service.ParseFilename(filename)

	require.NotNil(t, result)
	assert.True(t, aiService.parseCalled)
	assert.Equal(t, parser.ParseStatusNeedsAI, result.Status)
	assert.Contains(t, result.ErrorMessage, "AI parsing failed")
}

func TestParserServiceWithAI_ReturnsNeedsAI_WhenAINotConfigured(t *testing.T) {
	aiService := &mockAIService{isConfigured: false}
	service := NewParserServiceWithAI(aiService)

	filename := "[Leopard-Raws] Kimetsu no Yaiba - 26.mkv"
	result := service.ParseFilename(filename)

	require.NotNil(t, result)
	assert.False(t, aiService.parseCalled) // AI should NOT be called when not configured
	assert.Equal(t, parser.ParseStatusNeedsAI, result.Status)
}

func TestParserServiceWithAI_HandlesMovieFromAI(t *testing.T) {
	aiService := &mockAIService{
		isConfigured: true,
		parseFunc: func(ctx context.Context, filename string) (*ai.ParseResponse, error) {
			return &ai.ParseResponse{
				Title:      "My Movie Title",
				Year:       2023,
				MediaType:  "movie",
				Quality:    "1080p",
				Confidence: 0.88,
			}, nil
		},
	}
	service := NewParserServiceWithAI(aiService)

	filename := "some weird movie filename.mkv"
	result := service.ParseFilename(filename)

	require.NotNil(t, result)
	assert.Equal(t, parser.ParseStatusSuccess, result.Status)
	assert.Equal(t, parser.MediaTypeMovie, result.MediaType)
	assert.Equal(t, "My Movie Title", result.Title)
	assert.Equal(t, 2023, result.Year)
	assert.Equal(t, 88, result.Confidence)
}

func TestParserServiceWithAI_WithContext(t *testing.T) {
	aiService := &mockAIService{
		isConfigured: true,
		parseFunc: func(ctx context.Context, filename string) (*ai.ParseResponse, error) {
			// Verify context is passed through
			if ctx == nil {
				t.Error("context should not be nil")
			}
			return &ai.ParseResponse{
				Title:     "Context Test",
				MediaType: "movie",
			}, nil
		},
	}
	service := NewParserServiceWithAI(aiService)

	ctx := context.Background()
	filename := "unparseable.mkv"
	result := service.ParseFilenameWithContext(ctx, filename)

	require.NotNil(t, result)
	assert.Equal(t, parser.ParseStatusSuccess, result.Status)
}
