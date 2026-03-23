// Package providers defines the SubtitleProvider interface and shared types
// for subtitle source implementations (Assrt, Zimuku, OpenSubtitles).
package providers

import (
	"context"
	"time"
)

// SubtitleProvider is the contract all subtitle sources must implement.
// Each provider searches a specific subtitle database and downloads subtitle files.
type SubtitleProvider interface {
	// Name returns the provider identifier (e.g., "assrt", "zimuku", "opensubtitles").
	Name() string

	// Search queries the subtitle source for results matching the given query.
	// Returns an empty slice (not an error) when no results are found or the provider is disabled.
	Search(ctx context.Context, query SubtitleQuery) ([]SubtitleResult, error)

	// Download fetches the raw subtitle file content by provider-specific ID.
	Download(ctx context.Context, id string) ([]byte, error)
}

// SubtitleQuery contains the search parameters for finding subtitles.
type SubtitleQuery struct {
	// Title is the media title to search for (required).
	Title string

	// Year is the release year for narrowing results (optional, 0 = unset).
	Year int

	// ImdbID is the IMDB identifier for precise matching (optional).
	ImdbID string

	// Season number for TV series (optional, 0 = unset).
	Season int

	// Episode number for TV series (optional, 0 = unset).
	Episode int

	// FileHash is a provider-specific file hash for hash-based matching (optional).
	FileHash string

	// Languages is the list of preferred language codes (e.g., "zh-Hant", "zh-Hans").
	Languages []string
}

// SubtitleResult represents a single subtitle search result from any provider.
type SubtitleResult struct {
	// ID is the provider-specific identifier used for downloading.
	ID string

	// Source is the provider name (e.g., "assrt", "zimuku", "opensubtitles").
	Source string

	// Filename is the original subtitle filename.
	Filename string

	// Language is the detected or declared language tag (e.g., "zh-Hant", "zh-Hans", "en").
	Language string

	// DownloadURL is the direct download link (if available from search results).
	DownloadURL string

	// FileSize is the subtitle file size in bytes (0 if unknown).
	FileSize int64

	// UploadDate is when the subtitle was uploaded to the source.
	UploadDate time.Time

	// Downloads is the download count from the source (used for scoring).
	Downloads int

	// Group is the fansub group or uploader name (used for scoring).
	Group string

	// Resolution is the tagged video resolution (e.g., "1080p", "720p") if available.
	Resolution string

	// Format is the subtitle file format (e.g., "srt", "ass", "ssa").
	Format string
}
