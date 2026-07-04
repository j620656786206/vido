package models

import (
	"strings"
	"time"
)

// Request statuses — the 5-value enum is the single source of truth the
// request pipeline (13-3a) and the frontend render against (Epic 13).
const (
	RequestStatusPending     = "pending"
	RequestStatusSearching   = "searching"
	RequestStatusDownloading = "downloading"
	RequestStatusCompleted   = "completed"
	RequestStatusFailed      = "failed"
)

// Request media types use the TMDB/frontend vocabulary ('movie'|'tv') —
// deliberately NOT media_libraries.content_type's 'series' (that column
// classifies local folders; requests target TMDB entities).
const (
	RequestMediaTypeMovie = "movie"
	RequestMediaTypeTV    = "tv"
)

// Fulfilment sources (migration 027 CHECK enum). 'arr' rows are claimed by
// the DVR plugin pipeline (Story 13-4a); 'builtin' is reserved for the
// Epic-15-blocked built-in download path (13-X).
const (
	RequestFulfilmentSourceArr     = "arr"
	RequestFulfilmentSourceBuiltin = "builtin"
)

// Request records a user's intent to acquire a title (Story 13-1a, G-1/P3-001).
// Rows are born pending; fulfilment (13-4) and status transitions (13-3a)
// happen downstream. The JSON shape carries [@contract-v1] (13-1a AC #2/#3).
type Request struct {
	ID               string     `db:"id" json:"id"`
	TMDbID           int64      `db:"tmdb_id" json:"tmdb_id"`
	MediaType        string     `db:"media_type" json:"media_type"`
	Title            string     `db:"title" json:"title"`
	Status           string     `db:"status" json:"status"`
	FulfilmentSource NullString `db:"fulfilment_source" json:"fulfilment_source"`
	ExternalID       NullString `db:"external_id" json:"external_id"`
	Seasons          NullString `db:"seasons" json:"seasons"`
	Episodes         NullString `db:"episodes" json:"episodes"`
	ErrorMessage     NullString `db:"error_message" json:"error_message"`
	RequestedAt      time.Time  `db:"requested_at" json:"requested_at"`
	UpdatedAt        time.Time  `db:"updated_at" json:"updated_at"`
}

// Validate checks the request target fields supplied by the client.
func (r *Request) Validate() error {
	if r.TMDbID <= 0 {
		return &ValidationError{Field: "tmdb_id", Message: "tmdb_id is required and must be a positive integer"}
	}
	mt := strings.TrimSpace(r.MediaType)
	if mt != RequestMediaTypeMovie && mt != RequestMediaTypeTV {
		return &ValidationError{Field: "media_type", Message: "media_type must be 'movie' or 'tv'"}
	}
	return nil
}
