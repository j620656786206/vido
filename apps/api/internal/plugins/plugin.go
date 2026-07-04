// Package plugins implements the §7 embedded plugin architecture for external
// service integration (Story 13-4a, G-4/P3-004). The DVRPlugin interface shape
// carries [@contract-v1] (13-4a AC #1) — consumers: 13-4b (Sonarr), 13-3a
// (status pipeline), 13-2a (season/episode selection). Changing it requires a
// Rule 20 bump + downstream stale-mark.
package plugins

import "context"

// PluginConfig holds per-plugin connection configuration. The API key is
// never serialized or logged — json:"-" is the guard (the slog masking
// handler exists but is NOT wired; do not rely on it).
type PluginConfig struct {
	URL    string `json:"url"`
	APIKey string `json:"-"`
}

// QueueItem is a download-queue entry normalized across Radarr/Sonarr so the
// 13-3a status pipeline can consume both uniformly.
type QueueItem struct {
	ExternalID int64 // *arr's own movie/series id
	Title      string
	Status     string // *arr raw status string
	Size       int64
	SizeLeft   int64
	DownloadID string // torrent hash — 13-3a joins this to the qBT monitor
}

// AddOptions carries the per-add parameters resolved from plugin config.
type AddOptions struct {
	QualityProfileID int64
	RootFolderPath   string
	SearchNow        bool
}

// DVRPlugin is the §7 interface for *arr-style DVR integrations. A movie-only
// plugin (Radarr) returns a typed DVR_NOT_SUPPORTED error from AddSeries;
// a series-only plugin (Sonarr, 13-4b) mirrors that from AddMovie.
type DVRPlugin interface {
	Name() string
	TestConnection(ctx context.Context, config PluginConfig) error
	AddMovie(ctx context.Context, tmdbID int64, opts AddOptions) (externalID int64, err error)
	AddSeries(ctx context.Context, tmdbID int64, opts AddOptions) (externalID int64, err error)
	GetQueue(ctx context.Context) ([]QueueItem, error)
}

// QualityProfile is a DVR quality profile, normalized across plugins for the
// settings passthrough endpoints (AC #4; consumed by config validation + 13-6).
type QualityProfile struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// RootFolder is a DVR root folder — see QualityProfile.
type RootFolder struct {
	ID   int64  `json:"id"`
	Path string `json:"path"`
}

// ProfileLister is the optional client-level capability behind the
// quality-profiles / root-folders passthrough. Deliberately NOT part of
// DVRPlugin ([@contract-v1] AC #1 shape stays exact) — clients that support
// it (radarr now, sonarr in 13-4b) implement it additionally.
type ProfileLister interface {
	GetQualityProfiles(ctx context.Context) ([]QualityProfile, error)
	GetRootFolders(ctx context.Context) ([]RootFolder, error)
}
