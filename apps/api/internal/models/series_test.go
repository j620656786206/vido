package models

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeries_ScanGenres(t *testing.T) {
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
			name:     "valid JSON string",
			input:    `["Drama", "Thriller", "Mystery"]`,
			expected: []string{"Drama", "Thriller", "Mystery"},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			series := &Series{}
			err := series.ScanGenres(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, series.Genres)
			}
		})
	}
}

func TestSeries_Credits(t *testing.T) {
	t.Run("GetCredits with empty JSON", func(t *testing.T) {
		series := &Series{}
		credits, err := series.GetCredits()
		require.NoError(t, err)
		assert.NotNil(t, credits)
		assert.Empty(t, credits.Cast)
	})

	t.Run("SetCredits and GetCredits", func(t *testing.T) {
		series := &Series{}
		inputCredits := &Credits{
			Cast: []CastMember{
				{ID: 1, Name: "Actor One", Character: "Main"},
			},
		}

		err := series.SetCredits(inputCredits)
		require.NoError(t, err)

		retrieved, err := series.GetCredits()
		require.NoError(t, err)
		assert.Equal(t, 1, len(retrieved.Cast))
		assert.Equal(t, "Actor One", retrieved.Cast[0].Name)
	})
}

func TestSeries_Seasons(t *testing.T) {
	t.Run("GetSeasons with empty", func(t *testing.T) {
		series := &Series{}
		seasons, err := series.GetSeasons()
		require.NoError(t, err)
		assert.Empty(t, seasons)
	})

	t.Run("SetSeasons and GetSeasons", func(t *testing.T) {
		series := &Series{}
		input := []SeasonSummary{
			{ID: 1, SeasonNumber: 1, Name: "Season 1", EpisodeCount: 10},
			{ID: 2, SeasonNumber: 2, Name: "Season 2", EpisodeCount: 12},
		}

		err := series.SetSeasons(input)
		require.NoError(t, err)

		retrieved, err := series.GetSeasons()
		require.NoError(t, err)
		assert.Equal(t, 2, len(retrieved))
		assert.Equal(t, "Season 1", retrieved[0].Name)
		assert.Equal(t, 10, retrieved[0].EpisodeCount)
	})

	t.Run("SetSeasons with nil", func(t *testing.T) {
		series := &Series{}
		err := series.SetSeasons(nil)
		require.NoError(t, err)
		assert.False(t, series.SeasonsJSON.Valid)
	})

	t.Run("GetSeasons with invalid JSON", func(t *testing.T) {
		series := &Series{
			SeasonsJSON: sql.NullString{String: "invalid", Valid: true},
		}
		_, err := series.GetSeasons()
		assert.Error(t, err)
	})
}

func TestSeries_Networks(t *testing.T) {
	t.Run("GetNetworks with empty", func(t *testing.T) {
		series := &Series{}
		networks, err := series.GetNetworks()
		require.NoError(t, err)
		assert.Empty(t, networks)
	})

	t.Run("SetNetworks and GetNetworks", func(t *testing.T) {
		series := &Series{}
		input := []Network{
			{ID: 1, Name: "HBO", OriginCountry: "US"},
			{ID: 2, Name: "Netflix", OriginCountry: "US"},
		}

		err := series.SetNetworks(input)
		require.NoError(t, err)

		retrieved, err := series.GetNetworks()
		require.NoError(t, err)
		assert.Equal(t, 2, len(retrieved))
		assert.Equal(t, "HBO", retrieved[0].Name)
		assert.Equal(t, "Netflix", retrieved[1].Name)
	})
}

func TestSeries_Validate(t *testing.T) {
	tests := []struct {
		name    string
		series  *Series
		wantErr bool
		errType error
	}{
		{
			name: "valid series",
			series: &Series{
				ID:    "series-123",
				Title: "Test Series",
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			series: &Series{
				Title: "Test Series",
			},
			wantErr: true,
			errType: ErrSeriesIDRequired,
		},
		{
			name: "missing title",
			series: &Series{
				ID: "series-123",
			},
			wantErr: true,
			errType: ErrSeriesTitleRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.series.Validate()

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

func TestSeasonSummary(t *testing.T) {
	season := SeasonSummary{
		ID:           1,
		SeasonNumber: 1,
		Name:         "Season 1",
		Overview:     "The first season",
		PosterPath:   "/poster.jpg",
		AirDate:      "2024-01-15",
		EpisodeCount: 10,
	}

	assert.Equal(t, 1, season.ID)
	assert.Equal(t, 1, season.SeasonNumber)
	assert.Equal(t, "Season 1", season.Name)
	assert.Equal(t, 10, season.EpisodeCount)
}

func TestNetwork(t *testing.T) {
	network := Network{
		ID:            1,
		Name:          "HBO",
		LogoPath:      "/logo.png",
		OriginCountry: "US",
	}

	assert.Equal(t, 1, network.ID)
	assert.Equal(t, "HBO", network.Name)
	assert.Equal(t, "US", network.OriginCountry)
}
