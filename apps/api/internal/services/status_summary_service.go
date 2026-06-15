package services

import (
	"context"
	"os"
	"syscall"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/qbittorrent"
)

// StatusSummaryService composes the ambient NAS-status sections that feed the v2
// sidebar-footer status strip (UX Redesign D4-2 / ux3-0-3): disk headroom, active
// scan, download queue, and per-service health.
//
// It READS existing services (Rule 4 — no repository reach-through, no duplicated
// subsystem logic) and is fail-soft per section (ADR B1/F3): a downstream failure
// marks ONLY its section "unavailable" and never fails the whole endpoint. The
// strip renders an empty/stale section rather than a fail page (N1/N4).
type StatusSummaryService struct {
	health    serviceHealthSource
	scan      scanStateSource
	downloads downloadCountSource
	libraries libraryPathSource
}

// Narrow interfaces (testability) — satisfied by the concrete services in main.go.
type serviceHealthSource interface {
	GetAllStatuses(ctx context.Context) ([]models.ServiceStatus, error)
}
type scanStateSource interface {
	IsScanActive() bool
	GetProgress() ScanProgress
}
type downloadCountSource interface {
	GetDownloadCounts(ctx context.Context) (*qbittorrent.DownloadCounts, error)
}
type libraryPathSource interface {
	GetAllLibraries(ctx context.Context) ([]models.MediaLibraryWithPaths, error)
}

// NewStatusSummaryService wires the existing services it composes.
func NewStatusSummaryService(
	health serviceHealthSource,
	scan scanStateSource,
	downloads downloadCountSource,
	libraries libraryPathSource,
) *StatusSummaryService {
	return &StatusSummaryService{health: health, scan: scan, downloads: downloads, libraries: libraries}
}

// Section status values (B1/F3). A section is "ok" or "unavailable" — never an error page.
const (
	sectionOK          = "ok"
	sectionUnavailable = "unavailable"
)

// StatusSummary is the GET /api/v1/status/summary payload. snake_case tags match the
// codebase convention; the web client camelCases at the fetchApi boundary (Rule 18).
type StatusSummary struct {
	DiskHeadroom  DiskSection   `json:"disk_headroom"`
	ActiveScan    ScanSection   `json:"active_scan"`
	DownloadQueue QueueSection  `json:"download_queue"`
	ServiceHealth HealthSection `json:"service_health"`
}

type DiskSection struct {
	Status     string `json:"status"`
	UsedBytes  uint64 `json:"used_bytes"`
	TotalBytes uint64 `json:"total_bytes"`
	Volumes    int    `json:"volumes"`
	Error      string `json:"error,omitempty"`
}

type ScanSection struct {
	Status      string `json:"status"`
	Active      bool   `json:"active"`
	PercentDone int    `json:"percent_done"`
	CurrentFile string `json:"current_file,omitempty"`
	Error       string `json:"error,omitempty"`
}

type QueueSection struct {
	Status      string `json:"status"`
	Downloading int    `json:"downloading"`
	Total       int    `json:"total"`
	Error       string `json:"error,omitempty"`
}

type HealthSection struct {
	Status   string                 `json:"status"`
	Services []models.ServiceStatus `json:"services"`
	Error    string                 `json:"error,omitempty"`
}

// GetSummary composes all four sections. It NEVER returns an error — each section
// degrades independently (fail-soft).
func (s *StatusSummaryService) GetSummary(ctx context.Context) StatusSummary {
	return StatusSummary{
		DiskHeadroom:  s.diskSection(ctx),
		ActiveScan:    s.scanSection(),
		DownloadQueue: s.queueSection(ctx),
		ServiceHealth: s.healthSection(ctx),
	}
}

func (s *StatusSummaryService) healthSection(ctx context.Context) HealthSection {
	if s.health == nil {
		return HealthSection{Status: sectionUnavailable, Error: "service unavailable", Services: []models.ServiceStatus{}}
	}
	statuses, err := s.health.GetAllStatuses(ctx)
	if err != nil {
		return HealthSection{Status: sectionUnavailable, Error: err.Error(), Services: []models.ServiceStatus{}}
	}
	if statuses == nil {
		statuses = []models.ServiceStatus{}
	}
	return HealthSection{Status: sectionOK, Services: statuses}
}

func (s *StatusSummaryService) scanSection() ScanSection {
	if s.scan == nil {
		return ScanSection{Status: sectionUnavailable, Error: "service unavailable"}
	}
	p := s.scan.GetProgress()
	return ScanSection{
		Status:      sectionOK,
		Active:      s.scan.IsScanActive(),
		PercentDone: p.PercentDone,
		CurrentFile: p.CurrentFile,
	}
}

func (s *StatusSummaryService) queueSection(ctx context.Context) QueueSection {
	if s.downloads == nil {
		return QueueSection{Status: sectionUnavailable, Error: "service unavailable"}
	}
	counts, err := s.downloads.GetDownloadCounts(ctx)
	if err != nil || counts == nil {
		msg := "unavailable"
		if err != nil {
			msg = err.Error()
		}
		return QueueSection{Status: sectionUnavailable, Error: msg}
	}
	return QueueSection{Status: sectionOK, Downloading: counts.Downloading, Total: counts.All}
}

func (s *StatusSummaryService) diskSection(ctx context.Context) DiskSection {
	if s.libraries == nil {
		return DiskSection{Status: sectionUnavailable, Error: "service unavailable"}
	}
	libs, err := s.libraries.GetAllLibraries(ctx)
	if err != nil {
		return DiskSection{Status: sectionUnavailable, Error: err.Error()}
	}

	seen := map[uint64]bool{}
	var total, free uint64
	volumes := 0
	for _, lib := range libs {
		for _, p := range lib.Paths {
			if p.Path == "" {
				continue
			}
			var fs syscall.Statfs_t
			if err := syscall.Statfs(p.Path, &fs); err != nil {
				continue // a missing/unreadable path degrades only itself
			}
			// Dedup volumes by device id so multiple library folders on one volume
			// (Epic 7b) are not double-counted.
			if dev, derr := deviceID(p.Path); derr == nil {
				if seen[dev] {
					continue
				}
				seen[dev] = true
			}
			bsize := uint64(fs.Bsize)
			total += fs.Blocks * bsize
			free += fs.Bavail * bsize
			volumes++
		}
	}

	if volumes == 0 || total == 0 {
		return DiskSection{Status: sectionUnavailable, Error: "no readable media-library volume"}
	}
	return DiskSection{Status: sectionOK, UsedBytes: total - free, TotalBytes: total, Volumes: volumes}
}

// deviceID returns the filesystem device id for a path (for volume de-duplication).
func deviceID(path string) (uint64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, os.ErrInvalid
	}
	return uint64(st.Dev), nil
}
