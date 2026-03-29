package models

import "time"

// MediaLibraryContentType represents the content type of a media library.
type MediaLibraryContentType string

const (
	ContentTypeMovie  MediaLibraryContentType = "movie"
	ContentTypeSeries MediaLibraryContentType = "series"
)

// MediaLibraryPathStatus represents the accessibility status of a library path.
type MediaLibraryPathStatus string

const (
	PathStatusUnknown      MediaLibraryPathStatus = "unknown"
	PathStatusAccessible   MediaLibraryPathStatus = "accessible"
	PathStatusNotFound     MediaLibraryPathStatus = "not_found"
	PathStatusNotReadable  MediaLibraryPathStatus = "not_readable"
	PathStatusNotDirectory MediaLibraryPathStatus = "not_directory"
)

// MediaLibrary represents a named media library with a content type.
type MediaLibrary struct {
	ID          string                  `db:"id" json:"id"`
	Name        string                  `db:"name" json:"name"`
	ContentType MediaLibraryContentType `db:"content_type" json:"content_type"`
	AutoDetect  bool                    `db:"auto_detect" json:"auto_detect"`
	SortOrder   int                     `db:"sort_order" json:"sort_order"`
	CreatedAt   time.Time               `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time               `db:"updated_at" json:"updated_at"`
}

// MediaLibraryPath represents a filesystem path belonging to a library.
type MediaLibraryPath struct {
	ID            string                 `db:"id" json:"id"`
	LibraryID     string                 `db:"library_id" json:"library_id"`
	Path          string                 `db:"path" json:"path"`
	Status        MediaLibraryPathStatus `db:"status" json:"status"`
	LastCheckedAt *time.Time             `db:"last_checked_at" json:"last_checked_at,omitempty"`
	CreatedAt     time.Time              `db:"created_at" json:"created_at"`
}

// MediaLibraryWithPaths is a joined view of a library with its paths and media count.
type MediaLibraryWithPaths struct {
	MediaLibrary
	Paths      []MediaLibraryPath `json:"paths"`
	MediaCount int                `json:"media_count"`
}

// Validate validates the media library fields.
func (ml *MediaLibrary) Validate() error {
	if ml.Name == "" {
		return &ValidationError{Field: "name", Message: "library name is required"}
	}
	if len(ml.Name) > 255 {
		return &ValidationError{Field: "name", Message: "library name must be 255 characters or fewer"}
	}
	if ml.ContentType != ContentTypeMovie && ml.ContentType != ContentTypeSeries {
		return &ValidationError{Field: "content_type", Message: "content type must be 'movie' or 'series'"}
	}
	return nil
}

// Validate validates the media library path fields.
func (p *MediaLibraryPath) Validate() error {
	if p.LibraryID == "" {
		return &ValidationError{Field: "library_id", Message: "library ID is required"}
	}
	if p.Path == "" {
		return &ValidationError{Field: "path", Message: "path is required"}
	}
	return nil
}
