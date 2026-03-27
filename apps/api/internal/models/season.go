package models

import (
	"time"
)

// Season represents a TV series season entity in the database
type Season struct {
	// Core fields
	ID           string        `db:"id" json:"id"`
	SeriesID     string        `db:"series_id" json:"series_id"`
	TMDbID       NullInt64 `db:"tmdb_id" json:"tmdb_id,omitempty"`
	SeasonNumber int           `db:"season_number" json:"season_number"`

	// Content fields
	Name         NullString  `db:"name" json:"name,omitempty"`
	Overview     NullString  `db:"overview" json:"overview,omitempty"`
	PosterPath   NullString  `db:"poster_path" json:"poster_path,omitempty"`
	AirDate      NullString  `db:"air_date" json:"air_date,omitempty"`
	EpisodeCount NullInt64   `db:"episode_count" json:"episode_count,omitempty"`
	VoteAverage  NullFloat64 `db:"vote_average" json:"vote_average,omitempty"`

	// Timestamps
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// Validate validates the season fields
func (s *Season) Validate() error {
	if s.ID == "" {
		return ErrSeasonIDRequired
	}
	if s.SeriesID == "" {
		return ErrSeasonSeriesIDRequired
	}
	if s.SeasonNumber < 0 {
		return ErrSeasonNumberInvalid
	}
	return nil
}

// Season validation errors
var (
	ErrSeasonIDRequired       = &ValidationError{Field: "id", Message: "season ID is required"}
	ErrSeasonSeriesIDRequired = &ValidationError{Field: "seriesId", Message: "season series ID is required"}
	ErrSeasonNumberInvalid    = &ValidationError{Field: "seasonNumber", Message: "season number must be non-negative"}
)
