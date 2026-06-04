package models

import (
	"encoding/json"
	"strings"
	"time"
)

const (
	// FilterPresetMaxNameLength caps the preset name (matches the frontend
	// Task 2.3 max-30 validation so client and server agree).
	FilterPresetMaxNameLength = 30
	// FilterPresetMaxCount bounds total stored presets to prevent unbounded
	// growth (Story 11-4 Task 1.4).
	FilterPresetMaxCount = 20
)

// FilterPreset is a named, saved combination of discover filters (Story 11-4,
// P2-015). Filters is stored as an opaque JSON string matching the URL search
// param format (e.g. {"genre":"28","year_gte":"2024","region":"KR"}); the
// frontend owns its serialization so the API never key-transforms its contents.
type FilterPreset struct {
	ID        string    `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	Filters   string    `db:"filters" json:"filters"` // raw JSON string (URL-param shape)
	SortOrder int       `db:"sort_order" json:"sort_order"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// Validate checks required fields, the name length cap, and that Filters is
// well-formed JSON (defends against a malformed client payload landing in DB).
func (p *FilterPreset) Validate() error {
	name := strings.TrimSpace(p.Name)
	if name == "" {
		return &ValidationError{Field: "name", Message: "preset name is required"}
	}
	if len([]rune(name)) > FilterPresetMaxNameLength {
		return &ValidationError{Field: "name", Message: "preset name must be 30 characters or fewer"}
	}
	if strings.TrimSpace(p.Filters) == "" {
		return &ValidationError{Field: "filters", Message: "filters are required"}
	}
	if !json.Valid([]byte(p.Filters)) {
		return &ValidationError{Field: "filters", Message: "filters must be valid JSON"}
	}
	return nil
}
