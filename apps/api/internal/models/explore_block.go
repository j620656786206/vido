package models

import (
	"strings"
	"time"
)

// ExploreBlockContentType distinguishes TMDb movie discover from TV discover.
// Uses 'tv' (not 'series') because the TMDb discover endpoint is /discover/tv.
type ExploreBlockContentType string

const (
	ExploreBlockContentMovie ExploreBlockContentType = "movie"
	ExploreBlockContentTV    ExploreBlockContentType = "tv"
)

const (
	ExploreBlockMinMaxItems     = 1
	ExploreBlockMaxMaxItems     = 40
	ExploreBlockDefaultMaxItems = 20
	ExploreBlockMaxNameLength   = 255
)

// ExploreBlock is a homepage custom discover block configured by the user.
// Story 10.3 — P2-002.
type ExploreBlock struct {
	ID          string                  `db:"id" json:"id"`
	Name        string                  `db:"name" json:"name"`
	ContentType ExploreBlockContentType `db:"content_type" json:"content_type"`
	GenreIDs    string                  `db:"genre_ids" json:"genre_ids"` // comma-separated TMDb genre IDs
	Language    string                  `db:"language" json:"language"`   // BCP 47 (e.g. "zh-TW")
	Region      string                  `db:"region" json:"region"`       // ISO 3166-1 alpha-2 (e.g. "TW")
	SortBy      string                  `db:"sort_by" json:"sort_by"`     // e.g. "popularity.desc"
	MaxItems    int                     `db:"max_items" json:"max_items"`
	SortOrder   int                     `db:"sort_order" json:"sort_order"`
	CreatedAt   time.Time               `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time               `db:"updated_at" json:"updated_at"`
}

// Validate checks required fields and normalized value ranges.
func (b *ExploreBlock) Validate() error {
	name := strings.TrimSpace(b.Name)
	if name == "" {
		return &ValidationError{Field: "name", Message: "block name is required"}
	}
	if len(name) > ExploreBlockMaxNameLength {
		return &ValidationError{Field: "name", Message: "block name must be 255 characters or fewer"}
	}
	if b.ContentType != ExploreBlockContentMovie && b.ContentType != ExploreBlockContentTV {
		return &ValidationError{Field: "content_type", Message: "content type must be 'movie' or 'tv'"}
	}
	if b.MaxItems < ExploreBlockMinMaxItems || b.MaxItems > ExploreBlockMaxMaxItems {
		return &ValidationError{Field: "max_items", Message: "max_items must be between 1 and 40"}
	}
	return nil
}
