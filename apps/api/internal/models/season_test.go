package models

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSeason_Validate(t *testing.T) {
	tests := []struct {
		name    string
		season  *Season
		wantErr bool
		errType error
	}{
		{
			name: "valid season",
			season: &Season{
				ID:           "season-123",
				SeriesID:     "series-123",
				SeasonNumber: 1,
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			season: &Season{
				SeriesID:     "series-123",
				SeasonNumber: 1,
			},
			wantErr: true,
			errType: ErrSeasonIDRequired,
		},
		{
			name: "missing series ID",
			season: &Season{
				ID:           "season-123",
				SeasonNumber: 1,
			},
			wantErr: true,
			errType: ErrSeasonSeriesIDRequired,
		},
		{
			name: "negative season number",
			season: &Season{
				ID:           "season-123",
				SeriesID:     "series-123",
				SeasonNumber: -1,
			},
			wantErr: true,
			errType: ErrSeasonNumberInvalid,
		},
		{
			name: "season 0 (specials) is valid",
			season: &Season{
				ID:           "season-123",
				SeriesID:     "series-123",
				SeasonNumber: 0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.season.Validate()

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

func TestSeason_FullModel(t *testing.T) {
	now := time.Now()
	season := &Season{
		ID:           "season-abc123",
		SeriesID:     "series-xyz789",
		TMDbID:       sql.NullInt64{Int64: 54321, Valid: true},
		SeasonNumber: 2,
		Name:         sql.NullString{String: "Season 2", Valid: true},
		Overview:     sql.NullString{String: "The second season continues the story", Valid: true},
		PosterPath:   sql.NullString{String: "/posters/season-2.jpg", Valid: true},
		AirDate:      sql.NullString{String: "2024-06-15", Valid: true},
		EpisodeCount: sql.NullInt64{Int64: 10, Valid: true},
		VoteAverage:  sql.NullFloat64{Float64: 8.2, Valid: true},
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	assert.Equal(t, "season-abc123", season.ID)
	assert.Equal(t, "series-xyz789", season.SeriesID)
	assert.Equal(t, int64(54321), season.TMDbID.Int64)
	assert.True(t, season.TMDbID.Valid)
	assert.Equal(t, 2, season.SeasonNumber)
	assert.Equal(t, "Season 2", season.Name.String)
	assert.Equal(t, "The second season continues the story", season.Overview.String)
	assert.Equal(t, "/posters/season-2.jpg", season.PosterPath.String)
	assert.Equal(t, "2024-06-15", season.AirDate.String)
	assert.Equal(t, int64(10), season.EpisodeCount.Int64)
	assert.Equal(t, 8.2, season.VoteAverage.Float64)
	assert.NoError(t, season.Validate())
}

func TestSeason_NullFields(t *testing.T) {
	season := &Season{
		ID:           "season-null",
		SeriesID:     "series-null",
		SeasonNumber: 1,
	}

	assert.False(t, season.TMDbID.Valid)
	assert.False(t, season.Name.Valid)
	assert.False(t, season.Overview.Valid)
	assert.False(t, season.PosterPath.Valid)
	assert.False(t, season.AirDate.Valid)
	assert.False(t, season.EpisodeCount.Valid)
	assert.False(t, season.VoteAverage.Valid)
	assert.NoError(t, season.Validate())
}

func TestSeasonValidationErrors(t *testing.T) {
	t.Run("ErrSeasonIDRequired", func(t *testing.T) {
		assert.Equal(t, "season ID is required", ErrSeasonIDRequired.Error())
		assert.Equal(t, "id", ErrSeasonIDRequired.Field)
	})

	t.Run("ErrSeasonSeriesIDRequired", func(t *testing.T) {
		assert.Equal(t, "season series ID is required", ErrSeasonSeriesIDRequired.Error())
		assert.Equal(t, "seriesId", ErrSeasonSeriesIDRequired.Field)
	})

	t.Run("ErrSeasonNumberInvalid", func(t *testing.T) {
		assert.Equal(t, "season number must be non-negative", ErrSeasonNumberInvalid.Error())
		assert.Equal(t, "seasonNumber", ErrSeasonNumberInvalid.Field)
	})
}
