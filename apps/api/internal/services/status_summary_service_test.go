package services

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/qbittorrent"
)

// --- fakes for the four narrow sources (status_summary_service.go) ---

type fakeHealth struct {
	statuses []models.ServiceStatus
	err      error
}

func (f fakeHealth) GetAllStatuses(ctx context.Context) ([]models.ServiceStatus, error) {
	return f.statuses, f.err
}

type fakeScan struct {
	active   bool
	progress ScanProgress
}

func (f fakeScan) IsScanActive() bool        { return f.active }
func (f fakeScan) GetProgress() ScanProgress { return f.progress }

type fakeDownloads struct {
	counts *qbittorrent.DownloadCounts
	err    error
}

func (f fakeDownloads) GetDownloadCounts(ctx context.Context) (*qbittorrent.DownloadCounts, error) {
	return f.counts, f.err
}

type fakeLibraries struct {
	libs []models.MediaLibraryWithPaths
	err  error
}

func (f fakeLibraries) GetAllLibraries(ctx context.Context) ([]models.MediaLibraryWithPaths, error) {
	return f.libs, f.err
}

func libsWithPaths(paths ...string) []models.MediaLibraryWithPaths {
	mp := make([]models.MediaLibraryPath, 0, len(paths))
	for _, p := range paths {
		mp = append(mp, models.MediaLibraryPath{Path: p})
	}
	return []models.MediaLibraryWithPaths{{Paths: mp}}
}

func TestStatusSummary_AllOK(t *testing.T) {
	dir := t.TempDir()
	svc := NewStatusSummaryService(
		fakeHealth{statuses: []models.ServiceStatus{{Name: "qbittorrent", Status: "connected"}}},
		fakeScan{active: true, progress: ScanProgress{PercentDone: 42, CurrentFile: "movie.mkv"}},
		fakeDownloads{counts: &qbittorrent.DownloadCounts{All: 5, Downloading: 2}},
		fakeLibraries{libs: libsWithPaths(dir)},
	)

	s := svc.GetSummary(context.Background())

	if s.ServiceHealth.Status != sectionOK || len(s.ServiceHealth.Services) != 1 {
		t.Errorf("health = %+v, want ok with 1 service", s.ServiceHealth)
	}
	if s.ActiveScan.Status != sectionOK || !s.ActiveScan.Active || s.ActiveScan.PercentDone != 42 || s.ActiveScan.CurrentFile != "movie.mkv" {
		t.Errorf("scan = %+v, want ok/active/42/movie.mkv", s.ActiveScan)
	}
	if s.DownloadQueue.Status != sectionOK || s.DownloadQueue.Downloading != 2 || s.DownloadQueue.Total != 5 {
		t.Errorf("queue = %+v, want ok/2/5", s.DownloadQueue)
	}
	if s.DiskHeadroom.Status != sectionOK || s.DiskHeadroom.TotalBytes == 0 || s.DiskHeadroom.UsedBytes > s.DiskHeadroom.TotalBytes {
		t.Errorf("disk = %+v, want ok with sane totals", s.DiskHeadroom)
	}
}

func TestStatusSummary_HealthSourceFailsSoft(t *testing.T) {
	dir := t.TempDir()
	svc := NewStatusSummaryService(
		fakeHealth{err: errors.New("monitor down")},
		fakeScan{},
		fakeDownloads{counts: &qbittorrent.DownloadCounts{}},
		fakeLibraries{libs: libsWithPaths(dir)},
	)

	s := svc.GetSummary(context.Background())

	if s.ServiceHealth.Status != sectionUnavailable || s.ServiceHealth.Error == "" {
		t.Errorf("health = %+v, want unavailable+error", s.ServiceHealth)
	}
	// Other sections must remain OK — failure is isolated to its section.
	if s.ActiveScan.Status != sectionOK || s.DownloadQueue.Status != sectionOK || s.DiskHeadroom.Status != sectionOK {
		t.Errorf("a non-health section regressed: scan=%s queue=%s disk=%s",
			s.ActiveScan.Status, s.DownloadQueue.Status, s.DiskHeadroom.Status)
	}
	// Services is always a non-nil slice (never nil → safe to render).
	if s.ServiceHealth.Services == nil {
		t.Error("health.Services should be non-nil even when unavailable")
	}
}

func TestStatusSummary_DownloadSourceFailsSoft(t *testing.T) {
	svc := NewStatusSummaryService(
		fakeHealth{statuses: []models.ServiceStatus{}},
		fakeScan{},
		fakeDownloads{err: errors.New("qbittorrent unreachable")},
		fakeLibraries{libs: libsWithPaths(t.TempDir())},
	)

	s := svc.GetSummary(context.Background())

	if s.DownloadQueue.Status != sectionUnavailable || s.DownloadQueue.Error == "" {
		t.Errorf("queue = %+v, want unavailable+error", s.DownloadQueue)
	}
	if s.ServiceHealth.Status != sectionOK || s.ActiveScan.Status != sectionOK || s.DiskHeadroom.Status != sectionOK {
		t.Errorf("a non-queue section regressed")
	}
}

func TestStatusSummary_DiskUnreadablePath(t *testing.T) {
	svc := NewStatusSummaryService(
		fakeHealth{}, fakeScan{}, fakeDownloads{counts: &qbittorrent.DownloadCounts{}},
		fakeLibraries{libs: libsWithPaths(filepath.Join(t.TempDir(), "does", "not", "exist"))},
	)

	s := svc.GetSummary(context.Background())

	if s.DiskHeadroom.Status != sectionUnavailable {
		t.Errorf("disk = %+v, want unavailable for an unreadable path", s.DiskHeadroom)
	}
}

func TestStatusSummary_DiskDedupsVolumes(t *testing.T) {
	// Two distinct paths on the SAME volume (subdirs of one temp dir) must count once.
	root := t.TempDir()
	a := filepath.Join(root, "a")
	b := filepath.Join(root, "b")
	for _, p := range []string{a, b} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}
	svc := NewStatusSummaryService(
		fakeHealth{}, fakeScan{}, fakeDownloads{counts: &qbittorrent.DownloadCounts{}},
		fakeLibraries{libs: libsWithPaths(a, b)},
	)

	s := svc.GetSummary(context.Background())

	if s.DiskHeadroom.Status != sectionOK {
		t.Fatalf("disk = %+v, want ok", s.DiskHeadroom)
	}
	if s.DiskHeadroom.Volumes != 1 {
		t.Errorf("volumes = %d, want 1 (same-volume subdirs deduped)", s.DiskHeadroom.Volumes)
	}
}

func TestStatusSummary_NilSourcesDegradeGracefully(t *testing.T) {
	svc := NewStatusSummaryService(nil, nil, nil, nil)
	s := svc.GetSummary(context.Background()) // must not panic
	if s.ServiceHealth.Status != sectionUnavailable ||
		s.ActiveScan.Status != sectionUnavailable ||
		s.DownloadQueue.Status != sectionUnavailable ||
		s.DiskHeadroom.Status != sectionUnavailable {
		t.Errorf("nil sources should yield all-unavailable, got %+v", s)
	}
}
