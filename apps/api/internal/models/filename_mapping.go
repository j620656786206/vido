package models

import "time"

// FilenameMapping represents a learned filename pattern mapping
type FilenameMapping struct {
	ID           string     `json:"id"`
	Pattern      string     `json:"pattern"`
	PatternType  string     `json:"patternType"` // "exact", "regex", "fuzzy", "fansub", "standard"
	PatternRegex string     `json:"patternRegex,omitempty"`
	FansubGroup  string     `json:"fansubGroup,omitempty"`
	TitlePattern string     `json:"titlePattern,omitempty"`
	MetadataType string     `json:"metadataType"` // "movie" or "series"
	MetadataID   string     `json:"metadataId"`
	TmdbID       int        `json:"tmdbId,omitempty"`
	Confidence   float64    `json:"confidence"`
	UseCount     int        `json:"useCount"`
	CreatedAt    time.Time  `json:"createdAt"`
	LastUsedAt   *time.Time `json:"lastUsedAt,omitempty"`
}
