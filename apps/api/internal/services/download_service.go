package services

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"

	"github.com/vido/api/internal/qbittorrent"
)

// DownloadServiceInterface defines the contract for download monitoring operations.
type DownloadServiceInterface interface {
	GetAllDownloads(ctx context.Context, sortField string, order string) ([]qbittorrent.Torrent, error)
	GetDownloadDetails(ctx context.Context, hash string) (*qbittorrent.TorrentDetails, error)
}

// DownloadService provides business logic for download monitoring.
type DownloadService struct {
	qbService QBittorrentServiceInterface
	logger    *slog.Logger

	mu              sync.Mutex
	cachedClient    *qbittorrent.Client
	cachedConfigKey string // "host|username|password" fingerprint
}

// NewDownloadService creates a new DownloadService.
func NewDownloadService(qbService QBittorrentServiceInterface, logger *slog.Logger) *DownloadService {
	return &DownloadService{
		qbService: qbService,
		logger:    logger,
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

// GetAllDownloads retrieves all torrents from qBittorrent with optional sorting.
// When sortField is "status", sorting is performed server-side since qBittorrent
// does not support native status sorting.
func (s *DownloadService) GetAllDownloads(ctx context.Context, sortField string, order string) ([]qbittorrent.Torrent, error) {
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

	// For "status" sort, fetch without sort and sort in Go
	var opts *qbittorrent.ListTorrentsOptions
	if sortField != "status" {
		opts = &qbittorrent.ListTorrentsOptions{
			Sort:    qbittorrent.TorrentsSort(sortField),
			Reverse: order == "desc",
		}
	}

	torrents, err := client.GetTorrents(ctx, opts)
	if err != nil {
		s.logger.Error("Failed to get torrents", "error", err)
		return nil, err
	}

	// Server-side sort by status (AC5)
	if sortField == "status" {
		sort.Slice(torrents, func(i, j int) bool {
			if order == "desc" {
				return string(torrents[i].Status) > string(torrents[j].Status)
			}
			return string(torrents[i].Status) < string(torrents[j].Status)
		})
	}

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

// Compile-time interface verification
var _ DownloadServiceInterface = (*DownloadService)(nil)
