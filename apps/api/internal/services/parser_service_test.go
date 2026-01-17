package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/parser"
)

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
