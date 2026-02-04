package learning

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPatternExtractor_Extract(t *testing.T) {
	extractor := NewPatternExtractor()

	tests := []struct {
		name           string
		filename       string
		wantFansub     string
		wantTitle      string
		wantPatternType string
		wantRegex      bool
	}{
		{
			name:           "fansub with square brackets",
			filename:       "[Leopard-Raws] Kimetsu no Yaiba - 26 (BD 1920x1080 x264 FLAC).mkv",
			wantFansub:     "Leopard-Raws",
			wantTitle:      "Kimetsu no Yaiba",
			wantPatternType: "fansub",
			wantRegex:      true,
		},
		{
			name:           "fansub with Chinese brackets",
			filename:       "【幻櫻字幕組】鬼滅之刃 - 01 [1080p].mkv",
			wantFansub:     "幻櫻字幕組",
			wantTitle:      "鬼滅之刃",
			wantPatternType: "fansub",
			wantRegex:      true,
		},
		{
			name:           "fansub with multiple brackets",
			filename:       "[SubsPlease] Frieren - Beyond Journey's End - 01 [1080p].mkv",
			wantFansub:     "SubsPlease",
			wantTitle:      "Frieren - Beyond Journey's End",
			wantPatternType: "fansub",
			wantRegex:      true,
		},
		{
			name:           "standard TV show format",
			filename:       "Breaking.Bad.S01E01.720p.BluRay.x264-DEMAND.mkv",
			wantFansub:     "",
			wantTitle:      "Breaking Bad",
			wantPatternType: "standard",
			wantRegex:      true,
		},
		{
			name:           "movie with year",
			filename:       "Inception.2010.1080p.BluRay.x264.mkv",
			wantFansub:     "",
			wantTitle:      "Inception",
			wantPatternType: "standard",
			wantRegex:      true,
		},
		{
			name:           "anime with episode number",
			filename:       "[Nekomoe kissaten] Bocchi the Rock! - 12 [BDRip 1920x1080 HEVC-yuv420p10 FLAC].mkv",
			wantFansub:     "Nekomoe kissaten",
			wantTitle:      "Bocchi the Rock!",
			wantPatternType: "fansub",
			wantRegex:      true,
		},
		{
			name:           "simple filename",
			filename:       "My.Movie.mkv",
			wantFansub:     "",
			wantTitle:      "My Movie",
			wantPatternType: "exact",
			wantRegex:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractor.Extract(tt.filename)
			require.NoError(t, err)
			require.NotNil(t, result)

			assert.Equal(t, tt.filename, result.OriginalFilename)
			assert.Equal(t, tt.wantFansub, result.FansubGroup)
			assert.Equal(t, tt.wantTitle, result.TitlePattern)
			assert.Equal(t, tt.wantPatternType, result.PatternType)

			if tt.wantRegex {
				assert.NotEmpty(t, result.Regex, "expected regex to be generated")
			}
		})
	}
}

func TestPatternExtractor_ExtractFansubGroup(t *testing.T) {
	extractor := NewPatternExtractor()

	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{
			name:     "square brackets",
			filename: "[Leopard-Raws] Title - 01.mkv",
			want:     "Leopard-Raws",
		},
		{
			name:     "Chinese brackets",
			filename: "【字幕組】標題 - 01.mkv",
			want:     "字幕組",
		},
		{
			name:     "nested brackets - take first",
			filename: "[Group1][Group2] Title - 01.mkv",
			want:     "Group1",
		},
		{
			name:     "no brackets",
			filename: "Movie.Name.2024.mkv",
			want:     "",
		},
		{
			name:     "quality in brackets (not fansub)",
			filename: "Movie.Name.2024.[1080p].mkv",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _ := extractor.Extract(tt.filename)
			assert.Equal(t, tt.want, result.FansubGroup)
		})
	}
}

func TestPatternExtractor_ExtractTitlePattern(t *testing.T) {
	extractor := NewPatternExtractor()

	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{
			name:     "fansub format - remove episode",
			filename: "[Group] Anime Title - 01 [1080p].mkv",
			want:     "Anime Title",
		},
		{
			name:     "standard TV format - remove season/episode",
			filename: "Show.Name.S01E05.720p.mkv",
			want:     "Show Name",
		},
		{
			name:     "movie format - remove year and quality",
			filename: "Movie.Name.2024.1080p.BluRay.mkv",
			want:     "Movie Name",
		},
		{
			name:     "Chinese title",
			filename: "[字幕組] 鬼滅之刃 - 01 [1080p].mkv",
			want:     "鬼滅之刃",
		},
		{
			name:     "title with dash",
			filename: "[SubsPlease] Frieren - Beyond Journey's End - 01.mkv",
			want:     "Frieren - Beyond Journey's End",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _ := extractor.Extract(tt.filename)
			assert.Equal(t, tt.want, result.TitlePattern)
		})
	}
}

func TestPatternExtractor_GenerateRegex(t *testing.T) {
	extractor := NewPatternExtractor()

	tests := []struct {
		name          string
		filename      string
		shouldMatch   []string
		shouldNotMatch []string
	}{
		{
			name:     "fansub pattern matches similar files",
			filename: "[Leopard-Raws] Kimetsu no Yaiba - 26 (BD 1920x1080 x264 FLAC).mkv",
			shouldMatch: []string{
				"[Leopard-Raws] Kimetsu no Yaiba - 01 (BD 1920x1080 x264 FLAC).mkv",
				"[Leopard-Raws] Kimetsu no Yaiba - 27 [1080p].mkv",
				"[Leopard-Raws] Kimetsu no Yaiba - 100.mkv",
			},
			shouldNotMatch: []string{
				"[Other-Group] Kimetsu no Yaiba - 01.mkv",
				"[Leopard-Raws] Different Anime - 01.mkv",
			},
		},
		{
			name:     "standard TV pattern",
			filename: "Breaking.Bad.S01E01.720p.BluRay.x264-DEMAND.mkv",
			shouldMatch: []string{
				"Breaking.Bad.S01E02.720p.mkv",
				"Breaking.Bad.S02E01.1080p.WEB-DL.mkv",
				"Breaking.Bad.S05E16.mkv",
			},
			shouldNotMatch: []string{
				"Better.Call.Saul.S01E01.mkv",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractor.Extract(tt.filename)
			require.NoError(t, err)
			require.NotEmpty(t, result.Regex)

			// The generated regex should match similar files
			for _, match := range tt.shouldMatch {
				assert.True(t, result.MatchesFilename(match),
					"regex should match %q", match)
			}

			// But not completely different files
			for _, noMatch := range tt.shouldNotMatch {
				assert.False(t, result.MatchesFilename(noMatch),
					"regex should NOT match %q", noMatch)
			}
		})
	}
}

func TestExtractedPattern_MatchesFilename(t *testing.T) {
	pattern := &ExtractedPattern{
		FansubGroup:  "Leopard-Raws",
		TitlePattern: "Kimetsu no Yaiba",
		Regex:        `(?i)[\[【]Leopard-Raws[\]】]\s*Kimetsu no Yaiba.*`,
	}

	assert.True(t, pattern.MatchesFilename("[Leopard-Raws] Kimetsu no Yaiba - 01.mkv"))
	assert.True(t, pattern.MatchesFilename("[Leopard-Raws] Kimetsu no Yaiba - 100 [1080p].mkv"))
	assert.False(t, pattern.MatchesFilename("[Other] Kimetsu no Yaiba - 01.mkv"))
}

func TestExtractedPattern_ToFilenameMapping(t *testing.T) {
	pattern := &ExtractedPattern{
		OriginalFilename: "[Group] Anime - 01.mkv",
		FansubGroup:      "Group",
		TitlePattern:     "Anime",
		Regex:            `(?i)[\[【]Group[\]】]\s*Anime.*`,
		PatternType:      "fansub",
	}

	mapping := pattern.ToFilenameMapping("series-123", "series", 85937)

	assert.NotEmpty(t, mapping.ID)
	assert.Equal(t, "[Group] Anime", mapping.Pattern)
	assert.Equal(t, "fansub", mapping.PatternType)
	assert.Equal(t, pattern.Regex, mapping.PatternRegex)
	assert.Equal(t, "Group", mapping.FansubGroup)
	assert.Equal(t, "Anime", mapping.TitlePattern)
	assert.Equal(t, "series", mapping.MetadataType)
	assert.Equal(t, "series-123", mapping.MetadataID)
	assert.Equal(t, 85937, mapping.TmdbID)
	assert.Equal(t, 1.0, mapping.Confidence)
	assert.Equal(t, 0, mapping.UseCount)
}
