package services

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/vido/api/internal/qbittorrent"
)

// DownloadServiceInterface defines the contract for download monitoring operations.
type DownloadServiceInterface interface {
	GetAllDownloads(ctx context.Context, filter string, sortField string, order string) ([]qbittorrent.Torrent, error)
	GetDownloadDetails(ctx context.Context, hash string) (*qbittorrent.TorrentDetails, error)
	GetDownloadCounts(ctx context.Context) (*qbittorrent.DownloadCounts, error)
	PauseDownload(ctx context.Context, hash string) error
	ResumeDownload(ctx context.Context, hash string) error
	RemoveDownload(ctx context.Context, hash string, deleteFiles bool) error
}

// defaultTorrentSnapshotTTL is how long one full torrent fetch is shared by
// every read path (bugfix-f-downloads-activity-perf). It matches the SSE
// broadcaster's 2s poll cadence: within one UI refresh window the list, the
// counts and the activity aggregate all describe the same moment.
const defaultTorrentSnapshotTTL = 2 * time.Second

// torrentSnapshot is one cached full (unfiltered) torrent list.
type torrentSnapshot struct {
	torrents  []qbittorrent.Torrent
	fetchedAt time.Time
	configKey string // invalidates the snapshot when the qBT config changes
}

// DownloadService provides business logic for download monitoring.
type DownloadService struct {
	qbService QBittorrentServiceInterface
	logger    *slog.Logger

	mu              sync.Mutex
	cachedClient    *qbittorrent.Client
	cachedConfigKey string // "host|username|password" fingerprint

	// Torrent snapshot cache (bugfix-f-downloads-activity-perf). On a NAS with
	// thousands of torrents one qBittorrent fetch costs >1s; GET /downloads,
	// GET /downloads/counts and GET /activity used to pay it independently on
	// every request. All read paths now share one snapshot within snapTTL, and
	// filtering/sorting happens locally. Errors are never cached.
	snapMu  sync.Mutex
	snap    *torrentSnapshot
	sf      singleflight.Group
	snapTTL time.Duration
}

// NewDownloadService creates a new DownloadService.
func NewDownloadService(qbService QBittorrentServiceInterface, logger *slog.Logger) *DownloadService {
	return &DownloadService{
		qbService: qbService,
		logger:    logger,
		snapTTL:   defaultTorrentSnapshotTTL,
	}
}

// configFingerprint returns a string key representing the config identity.
func configFingerprint(cfg *qbittorrent.Config) string {
	return cfg.Host + "|" + cfg.Username + "|" + cfg.Password + "|" + cfg.BasePath
}

// getClient returns a cached qBittorrent client, creating a new one only when config changes.
func (s *DownloadService) getClient(config *qbittorrent.Config) *qbittorrent.Client {
	key := configFingerprint(config)

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cachedClient != nil && s.cachedConfigKey == key {
		return s.cachedClient
	}

	s.cachedClient = qbittorrent.NewClient(config)
	s.cachedConfigKey = key
	return s.cachedClient
}

// validFilters defines the set of accepted filter values.
var validFilters = map[string]bool{
	"all": true, "downloading": true, "paused": true,
	"completed": true, "seeding": true, "error": true,
}

// getTorrentSnapshot returns the full (unfiltered) torrent list, serving it
// from the snapshot cache when it is fresh. Concurrent cache misses share ONE
// upstream fetch via singleflight — the three Activity-hub requests that fire
// together on page load cost one qBittorrent round-trip, not three. Errors are
// returned to every waiter and never cached.
func (s *DownloadService) getTorrentSnapshot(ctx context.Context) ([]qbittorrent.Torrent, error) {
	config, err := s.qbService.GetConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("get qBittorrent config: %w", err)
	}

	if config.Host == "" {
		return nil, &qbittorrent.ConnectionError{
			Code:    qbittorrent.ErrCodeNotConfigured,
			Message: "qBittorrent not configured",
		}
	}

	key := configFingerprint(config)

	s.snapMu.Lock()
	if s.snap != nil && s.snap.configKey == key && time.Since(s.snap.fetchedAt) < s.snapTTL {
		torrents := s.snap.torrents
		s.snapMu.Unlock()
		return torrents, nil
	}
	s.snapMu.Unlock()

	// NOTE: the singleflight leader's ctx governs the shared fetch; if the
	// leader is cancelled every waiter sees the error and the next request
	// simply fetches again — acceptable at a 2s TTL.
	v, err, _ := s.sf.Do(key, func() (interface{}, error) {
		client := s.getClient(config)
		torrents, err := client.GetTorrents(ctx, &qbittorrent.ListTorrentsOptions{
			Filter: qbittorrent.FilterAll,
		})
		if err != nil {
			s.logger.Error("Failed to get torrents", "error", err)
			return nil, err
		}
		s.snapMu.Lock()
		s.snap = &torrentSnapshot{torrents: torrents, fetchedAt: time.Now(), configKey: key}
		s.snapMu.Unlock()
		return torrents, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]qbittorrent.Torrent), nil
}

// invalidateSnapshot drops the cached torrent list. Called after any mutating
// action (pause/resume/remove) so the next read reflects it immediately.
func (s *DownloadService) invalidateSnapshot() {
	s.snapMu.Lock()
	s.snap = nil
	s.snapMu.Unlock()
}

// statusMatchesFilter implements the list filters locally against the client's
// normalized TorrentStatus. The "downloading" view keeps qBittorrent's broad
// membership (actively downloading + stalled + queued + checking) so items do
// not vanish from the view they appeared in before the snapshot rework.
func statusMatchesFilter(status qbittorrent.TorrentStatus, filter string) bool {
	switch filter {
	case "downloading":
		return status == qbittorrent.StatusDownloading ||
			status == qbittorrent.StatusStalled ||
			status == qbittorrent.StatusQueued ||
			status == qbittorrent.StatusChecking
	case "paused":
		return status == qbittorrent.StatusPaused
	case "completed":
		return status == qbittorrent.StatusCompleted
	case "seeding":
		return status == qbittorrent.StatusSeeding
	case "error":
		return status == qbittorrent.StatusError
	default: // "all"
		return true
	}
}

// sortTorrents orders the list locally. Supported fields mirror what the
// downloads UI offers (name/status/progress + the added_on default); unknown
// fields fall back to added_on. The sort is stable with a hash tiebreak so
// pagination never sees an item twice across pages.
func sortTorrents(torrents []qbittorrent.Torrent, sortField, order string) {
	desc := order == "desc"
	less := func(i, j qbittorrent.Torrent) bool {
		switch sortField {
		case "name":
			return strings.ToLower(i.Name) < strings.ToLower(j.Name)
		case "status":
			return string(i.Status) < string(j.Status)
		case "progress":
			return i.Progress < j.Progress
		case "size":
			return i.Size < j.Size
		case "eta":
			return i.ETA < j.ETA
		default: // "added_on" and anything unknown
			return i.AddedOn.Before(j.AddedOn)
		}
	}
	sort.SliceStable(torrents, func(a, b int) bool {
		ta, tb := torrents[a], torrents[b]
		la, lb := less(ta, tb), less(tb, ta)
		if la == lb { // equal on the sort key
			return ta.Hash < tb.Hash // deterministic tiebreak
		}
		if desc {
			return lb
		}
		return la
	})
}

// GetAllDownloads retrieves all torrents with optional filtering and sorting.
// The torrent list comes from the shared snapshot (one qBittorrent fetch per
// TTL window); filtering and sorting happen locally, which also makes the list
// views consistent with GetDownloadCounts — both read the same snapshot.
func (s *DownloadService) GetAllDownloads(ctx context.Context, filter string, sortField string, order string) ([]qbittorrent.Torrent, error) {
	all, err := s.getTorrentSnapshot(ctx)
	if err != nil {
		return nil, err
	}

	if !validFilters[filter] {
		filter = "all"
	}

	// Copy before filter/sort — the snapshot slice is shared across callers.
	torrents := make([]qbittorrent.Torrent, 0, len(all))
	for _, t := range all {
		if statusMatchesFilter(t.Status, filter) {
			torrents = append(torrents, t)
		}
	}

	sortTorrents(torrents, sortField, order)
	return torrents, nil
}

// GetDownloadDetails retrieves detailed information for a specific torrent.
func (s *DownloadService) GetDownloadDetails(ctx context.Context, hash string) (*qbittorrent.TorrentDetails, error) {
	config, err := s.qbService.GetConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("get qBittorrent config: %w", err)
	}

	if config.Host == "" {
		return nil, &qbittorrent.ConnectionError{
			Code:    qbittorrent.ErrCodeNotConfigured,
			Message: "qBittorrent not configured",
		}
	}

	client := s.getClient(config)

	details, err := client.GetTorrentDetails(ctx, hash)
	if err != nil {
		s.logger.Error("Failed to get torrent details", "error", err, "hash", hash)
		return nil, err
	}

	return details, nil
}

// GetDownloadCounts retrieves the count of torrents grouped by status. It
// counts the shared snapshot directly — no copy, no sort, and (within the TTL
// window) no extra qBittorrent round-trip beyond the one the list view paid.
func (s *DownloadService) GetDownloadCounts(ctx context.Context) (*qbittorrent.DownloadCounts, error) {
	torrents, err := s.getTorrentSnapshot(ctx)
	if err != nil {
		return nil, err
	}

	counts := &qbittorrent.DownloadCounts{
		All: len(torrents),
	}

	for _, t := range torrents {
		switch t.Status {
		case qbittorrent.StatusDownloading:
			counts.Downloading++
		case qbittorrent.StatusPaused:
			counts.Paused++
		case qbittorrent.StatusCompleted:
			counts.Completed++
		case qbittorrent.StatusSeeding:
			counts.Seeding++
		case qbittorrent.StatusError:
			counts.Error++
		case qbittorrent.StatusStalled:
			counts.Downloading++
		case qbittorrent.StatusQueued:
			counts.Queued++
		case qbittorrent.StatusChecking:
			counts.Downloading++
		}
	}

	return counts, nil
}

// clientForAction resolves the qBittorrent config and returns a client, mirroring
// the not-configured guard the GET methods use. Shared by the pause/resume/remove
// action methods.
func (s *DownloadService) clientForAction(ctx context.Context) (*qbittorrent.Client, error) {
	config, err := s.qbService.GetConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("get qBittorrent config: %w", err)
	}
	if config.Host == "" {
		return nil, &qbittorrent.ConnectionError{
			Code:    qbittorrent.ErrCodeNotConfigured,
			Message: "qBittorrent not configured",
		}
	}
	return s.getClient(config), nil
}

// PauseDownload pauses the torrent with the given hash.
func (s *DownloadService) PauseDownload(ctx context.Context, hash string) error {
	client, err := s.clientForAction(ctx)
	if err != nil {
		return err
	}
	if err := client.PauseTorrents(ctx, []string{hash}); err != nil {
		s.logger.Error("Failed to pause download", "error", err, "hash", hash)
		return err
	}
	s.invalidateSnapshot()
	return nil
}

// ResumeDownload resumes the torrent with the given hash.
func (s *DownloadService) ResumeDownload(ctx context.Context, hash string) error {
	client, err := s.clientForAction(ctx)
	if err != nil {
		return err
	}
	if err := client.ResumeTorrents(ctx, []string{hash}); err != nil {
		s.logger.Error("Failed to resume download", "error", err, "hash", hash)
		return err
	}
	s.invalidateSnapshot()
	return nil
}

// RemoveDownload removes the torrent with the given hash from qBittorrent.
// When deleteFiles is true the downloaded data is also deleted from disk.
func (s *DownloadService) RemoveDownload(ctx context.Context, hash string, deleteFiles bool) error {
	client, err := s.clientForAction(ctx)
	if err != nil {
		return err
	}
	if err := client.DeleteTorrents(ctx, []string{hash}, deleteFiles); err != nil {
		s.logger.Error("Failed to remove download", "error", err, "hash", hash, "delete_files", deleteFiles)
		return err
	}
	s.invalidateSnapshot()
	return nil
}

// Compile-time interface verification
var _ DownloadServiceInterface = (*DownloadService)(nil)
