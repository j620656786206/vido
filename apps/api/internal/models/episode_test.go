package models

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEpisode_Validate(t *testing.T) {
	tests := []struct {
		name    string
		episode *Episode
		wantErr bool
		errType error
	}{
		{
			name: "valid episode",
			episode: &Episode{
				ID:            "ep-123",
				SeriesID:      "series-123",
				SeasonNumber:  1,
				EpisodeNumber: 1,
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			episode: &Episode{
				SeriesID:      "series-123",
				SeasonNumber:  1,
				EpisodeNumber: 1,
			},
			wantErr: true,
			errType: ErrEpisodeIDRequired,
		},
		{
			name: "missing series ID",
			episode: &Episode{
				ID:            "ep-123",
				SeasonNumber:  1,
				EpisodeNumber: 1,
			},
			wantErr: true,
			errType: ErrEpisodeSeriesIDRequired,
		},
		{
			name: "negative season number",
			episode: &Episode{
				ID:            "ep-123",
				SeriesID:      "series-123",
				SeasonNumber:  -1,
				EpisodeNumber: 1,
			},
			wantErr: true,
			errType: ErrEpisodeSeasonNumberInvalid,
		},
		{
			name: "negative episode number",
			episode: &Episode{
				ID:            "ep-123",
				SeriesID:      "series-123",
				SeasonNumber:  1,
				EpisodeNumber: -1,
			},
			wantErr: true,
			errType: ErrEpisodeNumberInvalid,
		},
		{
			name: "season 0 (specials) is valid",
			episode: &Episode{
				ID:            "ep-123",
				SeriesID:      "series-123",
				SeasonNumber:  0,
				EpisodeNumber: 1,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.episode.Validate()

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

func TestEpisode_GetSeasonEpisodeCode(t *testing.T) {
	tests := []struct {
		name     string
		season   int
		episode  int
		expected string
	}{
		{
			name:     "single digit season and episode",
			season:   1,
			episode:  5,
			expected: "S01E05",
		},
		{
			name:     "double digit season and episode",
			season:   12,
			episode:  24,
			expected: "S12E24",
		},
		{
			name:     "season 0 (specials)",
			season:   0,
			episode:  1,
			expected: "S00E01",
		},
		{
			name:     "episode 0",
			season:   1,
			episode:  0,
			expected: "S01E00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			episode := &Episode{
				SeasonNumber:  tt.season,
				EpisodeNumber: tt.episode,
			}
			result := episode.GetSeasonEpisodeCode()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEpisode_FullModel(t *testing.T) {
	now := time.Now()
	episode := &Episode{
		ID:            "ep-abc123",
		SeriesID:      "series-xyz789",
		TMDbID:        sql.NullInt64{Int64: 12345, Valid: true},
		SeasonNumber:  2,
		EpisodeNumber: 8,
		Title:         sql.NullString{String: "The One With All the Episodes", Valid: true},
		Overview:      sql.NullString{String: "An exciting episode happens", Valid: true},
		AirDate:       sql.NullString{String: "2024-03-15", Valid: true},
		Runtime:       sql.NullInt64{Int64: 45, Valid: true},
		StillPath:     sql.NullString{String: "/stills/ep-abc123.jpg", Valid: true},
		VoteAverage:   sql.NullFloat64{Float64: 8.5, Valid: true},
		FilePath:      sql.NullString{String: "/media/series/S02E08.mkv", Valid: true},
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	assert.Equal(t, "ep-abc123", episode.ID)
	assert.Equal(t, "series-xyz789", episode.SeriesID)
	assert.Equal(t, int64(12345), episode.TMDbID.Int64)
	assert.Equal(t, 2, episode.SeasonNumber)
	assert.Equal(t, 8, episode.EpisodeNumber)
	assert.Equal(t, "The One With All the Episodes", episode.Title.String)
	assert.Equal(t, "An exciting episode happens", episode.Overview.String)
	assert.Equal(t, "2024-03-15", episode.AirDate.String)
	assert.Equal(t, int64(45), episode.Runtime.Int64)
	assert.Equal(t, "/stills/ep-abc123.jpg", episode.StillPath.String)
	assert.Equal(t, 8.5, episode.VoteAverage.Float64)
	assert.Equal(t, "/media/series/S02E08.mkv", episode.FilePath.String)
	assert.Equal(t, "S02E08", episode.GetSeasonEpisodeCode())
	assert.NoError(t, episode.Validate())
}

func TestEpisodeValidationErrors(t *testing.T) {
	t.Run("ErrEpisodeIDRequired", func(t *testing.T) {
		assert.Equal(t, "episode ID is required", ErrEpisodeIDRequired.Error())
		assert.Equal(t, "id", ErrEpisodeIDRequired.Field)
	})

	t.Run("ErrEpisodeSeriesIDRequired", func(t *testing.T) {
		assert.Equal(t, "episode series ID is required", ErrEpisodeSeriesIDRequired.Error())
		assert.Equal(t, "seriesId", ErrEpisodeSeriesIDRequired.Field)
	})

	t.Run("ErrEpisodeSeasonNumberInvalid", func(t *testing.T) {
		assert.Equal(t, "episode season number must be non-negative", ErrEpisodeSeasonNumberInvalid.Error())
		assert.Equal(t, "seasonNumber", ErrEpisodeSeasonNumberInvalid.Field)
	})

	t.Run("ErrEpisodeNumberInvalid", func(t *testing.T) {
		assert.Equal(t, "episode number must be non-negative", ErrEpisodeNumberInvalid.Error())
		assert.Equal(t, "episodeNumber", ErrEpisodeNumberInvalid.Field)
	})
}
