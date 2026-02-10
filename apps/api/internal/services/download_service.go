package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/vido/api/internal/qbittorrent"
)

// DownloadServiceInterface defines the contract for download monitoring operations.
type DownloadServiceInterface interface {
	GetAllDownloads(ctx context.Context, sort string, order string) ([]qbittorrent.Torrent, error)
	GetDownloadDetails(ctx context.Context, hash string) (*qbittorrent.TorrentDetails, error)
}

// DownloadService provides business logic for download monitoring.
type DownloadService struct {
	qbService QBittorrentServiceInterface
	logger    *slog.Logger
}

// NewDownloadService creates a new DownloadService.
func NewDownloadService(qbService QBittorrentServiceInterface, logger *slog.Logger) *DownloadService {
	return &DownloadService{
		qbService: qbService,
		logger:    logger,
	}
}

// GetAllDownloads retrieves all torrents from qBittorrent with optional sorting.
func (s *DownloadService) GetAllDownloads(ctx context.Context, sort string, order string) ([]qbittorrent.Torrent, error) {
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

	client := qbittorrent.NewClient(config)

	opts := &qbittorrent.ListTorrentsOptions{
		Sort:    qbittorrent.TorrentsSort(sort),
		Reverse: order == "desc",
	}

	torrents, err := client.GetTorrents(ctx, opts)
	if err != nil {
		s.logger.Error("Failed to get torrents", "error", err)
		return nil, err
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

	client := qbittorrent.NewClient(config)

	details, err := client.GetTorrentDetails(ctx, hash)
	if err != nil {
		s.logger.Error("Failed to get torrent details", "error", err, "hash", hash)
		return nil, err
	}

	return details, nil
}

// Compile-time interface verification
var _ DownloadServiceInterface = (*DownloadService)(nil)
