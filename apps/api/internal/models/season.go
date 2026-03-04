package models

import (
	"database/sql"
	"time"
)

// Season represents a TV series season entity in the database
type Season struct {
	// Core fields
	ID           string        `db:"id" json:"id"`
	SeriesID     string        `db:"series_id" json:"seriesId"`
	TMDbID       sql.NullInt64 `db:"tmdb_id" json:"tmdbId,omitempty"`
	SeasonNumber int           `db:"season_number" json:"seasonNumber"`

	// Content fields
	Name         sql.NullString  `db:"name" json:"name,omitempty"`
	Overview     sql.NullString  `db:"overview" json:"overview,omitempty"`
	PosterPath   sql.NullString  `db:"poster_path" json:"posterPath,omitempty"`
	AirDate      sql.NullString  `db:"air_date" json:"airDate,omitempty"`
	EpisodeCount sql.NullInt64   `db:"episode_count" json:"episodeCount,omitempty"`
	VoteAverage  sql.NullFloat64 `db:"vote_average" json:"voteAverage,omitempty"`

	// Timestamps
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
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
