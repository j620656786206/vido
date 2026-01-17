package models

import (
	"database/sql"
	"fmt"
	"time"
)

// Episode represents a TV series episode entity in the database
type Episode struct {
	// Core fields
	ID            string        `db:"id" json:"id"`
	SeriesID      string        `db:"series_id" json:"seriesId"`
	TMDbID        sql.NullInt64 `db:"tmdb_id" json:"tmdbId,omitempty"`
	SeasonNumber  int           `db:"season_number" json:"seasonNumber"`
	EpisodeNumber int           `db:"episode_number" json:"episodeNumber"`

	// Content fields
	Title       sql.NullString  `db:"title" json:"title,omitempty"`
	Overview    sql.NullString  `db:"overview" json:"overview,omitempty"`
	AirDate     sql.NullString  `db:"air_date" json:"airDate,omitempty"`
	Runtime     sql.NullInt64   `db:"runtime" json:"runtime,omitempty"`
	StillPath   sql.NullString  `db:"still_path" json:"stillPath,omitempty"`
	VoteAverage sql.NullFloat64 `db:"vote_average" json:"voteAverage,omitempty"`

	// File tracking
	FilePath sql.NullString `db:"file_path" json:"filePath,omitempty"`

	// Timestamps
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
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
