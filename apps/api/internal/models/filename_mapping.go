package models

import "time"

// FilenameMapping represents a learned filename pattern mapping
type FilenameMapping struct {
	ID           string     `json:"id"`
	Pattern      string     `json:"pattern"`
	PatternType  string     `json:"pattern_type"` // "exact", "regex", "fuzzy", "fansub", "standard"
	PatternRegex string     `json:"pattern_regex,omitempty"`
	FansubGroup  string     `json:"fansub_group,omitempty"`
	TitlePattern string     `json:"title_pattern,omitempty"`
	MetadataType string     `json:"metadata_type"` // "movie" or "series"
	MetadataID   string     `json:"metadata_id"`
	TmdbID       int        `json:"tmdb_id,omitempty"`
	Confidence   float64    `json:"confidence"`
	UseCount     int        `json:"use_count"`
	CreatedAt    time.Time  `json:"created_at"`
	LastUsedAt   *time.Time `json:"last_used_at,omitempty"`
}
