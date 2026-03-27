package models

import (
	"fmt"
	"time"
)

// Episode represents a TV series episode entity in the database
type Episode struct {
	// Core fields
	ID            string         `db:"id" json:"id"`
	SeriesID      string         `db:"series_id" json:"series_id"`
	SeasonID      NullString `db:"season_id" json:"season_id,omitempty"`
	TMDbID        NullInt64  `db:"tmdb_id" json:"tmdb_id,omitempty"`
	SeasonNumber  int            `db:"season_number" json:"season_number"`
	EpisodeNumber int            `db:"episode_number" json:"episode_number"`

	// Content fields
	Title       NullString  `db:"title" json:"title,omitempty"`
	Overview    NullString  `db:"overview" json:"overview,omitempty"`
	AirDate     NullString  `db:"air_date" json:"air_date,omitempty"`
	Runtime     NullInt64   `db:"runtime" json:"runtime,omitempty"`
	StillPath   NullString  `db:"still_path" json:"still_path,omitempty"`
	VoteAverage NullFloat64 `db:"vote_average" json:"vote_average,omitempty"`

	// File tracking
	FilePath NullString `db:"file_path" json:"file_path,omitempty"`

	// Timestamps
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// Validate validates the episode fields
func (e *Episode) Validate() error {
	if e.ID == "" {
		return ErrEpisodeIDRequired
	}
	if e.SeriesID == "" {
		return ErrEpisodeSeriesIDRequired
	}
	if e.SeasonNumber < 0 {
		return ErrEpisodeSeasonNumberInvalid
	}
	if e.EpisodeNumber < 0 {
		return ErrEpisodeNumberInvalid
	}
	return nil
}

// GetSeasonEpisodeCode returns a formatted season/episode code (e.g., "S01E05")
func (e *Episode) GetSeasonEpisodeCode() string {
	return formatSeasonEpisode(e.SeasonNumber, e.EpisodeNumber)
}

// formatSeasonEpisode formats season and episode numbers into SxxExx format
func formatSeasonEpisode(season, episode int) string {
	return fmt.Sprintf("S%02dE%02d", season, episode)
}

// Episode validation errors
var (
	ErrEpisodeIDRequired         = &ValidationError{Field: "id", Message: "episode ID is required"}
	ErrEpisodeSeriesIDRequired   = &ValidationError{Field: "seriesId", Message: "episode series ID is required"}
	ErrEpisodeSeasonNumberInvalid = &ValidationError{Field: "seasonNumber", Message: "episode season number must be non-negative"}
	ErrEpisodeNumberInvalid      = &ValidationError{Field: "episodeNumber", Message: "episode number must be non-negative"}
)
