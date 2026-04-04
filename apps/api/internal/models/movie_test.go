package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMovie_ScanGenres(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected []string
		wantErr  bool
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: []string{},
			wantErr:  false,
		},
		{
			name:     "empty string",
			input:    "[]",
			expected: []string{},
			wantErr:  false,
		},
		{
			name:     "valid JSON string",
			input:    `["Action", "Adventure", "Sci-Fi"]`,
			expected: []string{"Action", "Adventure", "Sci-Fi"},
			wantErr:  false,
		},
		{
			name:     "valid JSON bytes",
			input:    []byte(`["Drama", "Comedy"]`),
			expected: []string{"Drama", "Comedy"},
			wantErr:  false,
		},
		{
			name:     "invalid JSON",
			input:    "not json",
			expected: []string{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			movie := &Movie{}
			err := movie.ScanGenres(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, movie.Genres)
			}
		})
	}
}

func TestMovie_GenresJSON(t *testing.T) {
	tests := []struct {
		name     string
		genres   []string
		expected string
	}{
		{
			name:     "nil genres",
			genres:   nil,
			expected: "[]",
		},
		{
			name:     "empty genres",
			genres:   []string{},
			expected: "[]",
		},
		{
			name:     "with genres",
			genres:   []string{"Action", "Drama"},
			expected: `["Action","Drama"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			movie := &Movie{Genres: tt.genres}
			result, err := movie.GenresJSON()

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMovie_Credits(t *testing.T) {
	t.Run("GetCredits with empty JSON", func(t *testing.T) {
		movie := &Movie{}
		credits, err := movie.GetCredits()
		require.NoError(t, err)
		assert.NotNil(t, credits)
		assert.Empty(t, credits.Cast)
		assert.Empty(t, credits.Crew)
	})

	t.Run("SetCredits and GetCredits", func(t *testing.T) {
		movie := &Movie{}
		inputCredits := &Credits{
			Cast: []CastMember{
				{ID: 1, Name: "Actor One", Character: "Hero", Order: 0},
				{ID: 2, Name: "Actor Two", Character: "Villain", Order: 1},
			},
			Crew: []CrewMember{
				{ID: 10, Name: "Director", Job: "Director", Department: "Directing"},
			},
		}

		err := movie.SetCredits(inputCredits)
		require.NoError(t, err)
		assert.True(t, movie.CreditsJSON.Valid)

		retrieved, err := movie.GetCredits()
		require.NoError(t, err)
		assert.Equal(t, 2, len(retrieved.Cast))
		assert.Equal(t, "Actor One", retrieved.Cast[0].Name)
		assert.Equal(t, "Hero", retrieved.Cast[0].Character)
		assert.Equal(t, 1, len(retrieved.Crew))
		assert.Equal(t, "Director", retrieved.Crew[0].Job)
	})

	t.Run("SetCredits with nil", func(t *testing.T) {
		movie := &Movie{}
		err := movie.SetCredits(nil)
		require.NoError(t, err)
		assert.False(t, movie.CreditsJSON.Valid)
	})

	t.Run("GetCredits with invalid JSON", func(t *testing.T) {
		movie := &Movie{
			CreditsJSON: NewNullString("invalid json"),
		}
		_, err := movie.GetCredits()
		assert.Error(t, err)
	})
}

func TestMovie_ProductionCountries(t *testing.T) {
	t.Run("GetProductionCountries with empty", func(t *testing.T) {
		movie := &Movie{}
		countries, err := movie.GetProductionCountries()
		require.NoError(t, err)
		assert.Empty(t, countries)
	})

	t.Run("SetProductionCountries and Get", func(t *testing.T) {
		movie := &Movie{}
		input := []ProductionCountry{
			{ISO3166_1: "US", Name: "United States of America"},
			{ISO3166_1: "TW", Name: "Taiwan"},
		}

		err := movie.SetProductionCountries(input)
		require.NoError(t, err)

		retrieved, err := movie.GetProductionCountries()
		require.NoError(t, err)
		assert.Equal(t, 2, len(retrieved))
		assert.Equal(t, "US", retrieved[0].ISO3166_1)
		assert.Equal(t, "Taiwan", retrieved[1].Name)
	})

	t.Run("SetProductionCountries with nil", func(t *testing.T) {
		movie := &Movie{}
		err := movie.SetProductionCountries(nil)
		require.NoError(t, err)
		assert.False(t, movie.ProductionCountriesJSON.Valid)
	})
}

func TestMovie_SpokenLanguages(t *testing.T) {
	t.Run("GetSpokenLanguages with empty", func(t *testing.T) {
		movie := &Movie{}
		languages, err := movie.GetSpokenLanguages()
		require.NoError(t, err)
		assert.Empty(t, languages)
	})

	t.Run("SetSpokenLanguages and Get", func(t *testing.T) {
		movie := &Movie{}
		input := []SpokenLanguage{
			{ISO639_1: "en", Name: "English"},
			{ISO639_1: "zh", Name: "中文"},
		}

		err := movie.SetSpokenLanguages(input)
		require.NoError(t, err)

		retrieved, err := movie.GetSpokenLanguages()
		require.NoError(t, err)
		assert.Equal(t, 2, len(retrieved))
		assert.Equal(t, "en", retrieved[0].ISO639_1)
		assert.Equal(t, "中文", retrieved[1].Name)
	})
}

func TestMovie_Validate(t *testing.T) {
	tests := []struct {
		name    string
		movie   *Movie
		wantErr bool
		errType error
	}{
		{
			name: "valid movie",
			movie: &Movie{
				ID:    "movie-123",
				Title: "Test Movie",
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			movie: &Movie{
				Title: "Test Movie",
			},
			wantErr: true,
			errType: ErrMovieIDRequired,
		},
		{
			name: "missing title",
			movie: &Movie{
				ID: "movie-123",
			},
			wantErr: true,
			errType: ErrMovieTitleRequired,
		},
		{
			name:    "empty movie",
			movie:   &Movie{},
			wantErr: true,
			errType: ErrMovieIDRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.movie.Validate()

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.Equal(t, tt.errType, err)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	err := &ValidationError{
		Field:   "title",
		Message: "title is required",
	}

	assert.Equal(t, "title is required", err.Error())
}

func TestParseStatus_Constants(t *testing.T) {
	assert.Equal(t, ParseStatus("pending"), ParseStatusPending)
	assert.Equal(t, ParseStatus("success"), ParseStatusSuccess)
	assert.Equal(t, ParseStatus("needs_ai"), ParseStatusNeedsAI)
	assert.Equal(t, ParseStatus("failed"), ParseStatusFailed)
}

func TestMetadataSource_Constants(t *testing.T) {
	assert.Equal(t, MetadataSource("tmdb"), MetadataSourceTMDb)
	assert.Equal(t, MetadataSource("douban"), MetadataSourceDouban)
	assert.Equal(t, MetadataSource("wikipedia"), MetadataSourceWikipedia)
	assert.Equal(t, MetadataSource("manual"), MetadataSourceManual)
	assert.Equal(t, MetadataSource("nfo"), MetadataSourceNFO)
	assert.Equal(t, MetadataSource("ai"), MetadataSourceAI)
}

func TestShouldOverwrite(t *testing.T) {
	tests := []struct {
		name     string
		current  MetadataSource
		incoming MetadataSource
		want     bool
	}{
		// Empty current — always accept
		{"empty current accepts ai", "", MetadataSourceAI, true},
		{"empty current accepts nfo", "", MetadataSourceNFO, true},
		{"empty current accepts manual", "", MetadataSourceManual, true},

		// Same source — accept (idempotent)
		{"nfo overwrites nfo", MetadataSourceNFO, MetadataSourceNFO, true},
		{"tmdb overwrites tmdb", MetadataSourceTMDb, MetadataSourceTMDb, true},
		{"ai overwrites ai", MetadataSourceAI, MetadataSourceAI, true},

		// Higher priority overwrites lower
		{"nfo overwrites ai", MetadataSourceAI, MetadataSourceNFO, true},
		{"nfo overwrites tmdb", MetadataSourceTMDb, MetadataSourceNFO, true},
		{"tmdb overwrites ai", MetadataSourceAI, MetadataSourceTMDb, true},
		{"manual overwrites nfo", MetadataSourceNFO, MetadataSourceManual, true},
		{"manual overwrites tmdb", MetadataSourceTMDb, MetadataSourceManual, true},

		// Lower priority does NOT overwrite higher
		{"ai cannot overwrite nfo", MetadataSourceNFO, MetadataSourceAI, false},
		{"ai cannot overwrite tmdb", MetadataSourceTMDb, MetadataSourceAI, false},
		{"tmdb cannot overwrite nfo", MetadataSourceNFO, MetadataSourceTMDb, false},
		{"nfo cannot overwrite manual", MetadataSourceManual, MetadataSourceNFO, false},
		{"tmdb cannot overwrite manual", MetadataSourceManual, MetadataSourceTMDb, false},

		// Full priority chain verification
		{"douban overwrites wikipedia", MetadataSourceWikipedia, MetadataSourceDouban, true},
		{"wikipedia cannot overwrite douban", MetadataSourceDouban, MetadataSourceWikipedia, false},
		{"douban overwrites ai", MetadataSourceAI, MetadataSourceDouban, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldOverwrite(tt.current, tt.incoming)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMovie_TechInfoJSON_SnakeCase(t *testing.T) {
	movie := Movie{
		ID:    "test-1",
		Title: "Test Movie",
		VideoCodec:      NewNullString("H.265"),
		VideoResolution: NewNullString("3840x2160"),
		AudioCodec:      NewNullString("DTS"),
		AudioChannels:   NewNullInt64(6),
		SubtitleTracks:  NewNullString(`[{"language":"zh-Hant"}]`),
		HDRFormat:       NewNullString("HDR10"),
	}

	data, err := json.Marshal(movie)
	require.NoError(t, err)

	jsonStr := string(data)
	assert.Contains(t, jsonStr, `"video_codec"`)
	assert.Contains(t, jsonStr, `"video_resolution"`)
	assert.Contains(t, jsonStr, `"audio_codec"`)
	assert.Contains(t, jsonStr, `"audio_channels"`)
	assert.Contains(t, jsonStr, `"subtitle_tracks"`)
	assert.Contains(t, jsonStr, `"hdr_format"`)

	// Verify values
	assert.Contains(t, jsonStr, `"H.265"`)
	assert.Contains(t, jsonStr, `"3840x2160"`)
	assert.Contains(t, jsonStr, `"DTS"`)
	assert.Contains(t, jsonStr, `"HDR10"`)
}

func TestMovie_TechInfoJSON_NullWhenEmpty(t *testing.T) {
	movie := Movie{
		ID:    "test-2",
		Title: "Minimal Movie",
	}

	data, err := json.Marshal(movie)
	require.NoError(t, err)

	jsonStr := string(data)
	// NullString/NullInt64 marshal to null (not omitted) — consistent with existing fields
	assert.Contains(t, jsonStr, `"video_codec":null`)
	assert.Contains(t, jsonStr, `"video_resolution":null`)
	assert.Contains(t, jsonStr, `"audio_codec":null`)
	assert.Contains(t, jsonStr, `"audio_channels":null`)
	assert.Contains(t, jsonStr, `"subtitle_tracks":null`)
	assert.Contains(t, jsonStr, `"hdr_format":null`)
}
