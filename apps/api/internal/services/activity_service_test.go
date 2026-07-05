package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/qbittorrent"
)

// --- fakes for the two activity-only sources (fakeScan/fakeDownloads reused from
// status_summary_service_test.go — same package) ---

type fakeBatch struct {
	active      bool
	percentDone int
	current     int
	total       int
	item        string
}

func (f fakeBatch) ActivityProgress() (bool, int, int, int, string) {
	return f.active, f.percentDone, f.current, f.total, f.item
}

type fakeParse struct {
	pending    []*models.ParseJob
	pendingErr error
	all        []*models.ParseJob
	allErr     error
}

func (f fakeParse) GetPending(ctx context.Context, limit int) ([]*models.ParseJob, error) {
	return f.pending, f.pendingErr
}
func (f fakeParse) ListAll(ctx context.Context, limit int) ([]*models.ParseJob, error) {
	return f.all, f.allErr
}

func pj(status models.ParseJobStatus, name string) *models.ParseJob {
	return &models.ParseJob{Status: status, FileName: name, UpdatedAt: time.Now()}
}

func TestActivity_AllOK(t *testing.T) {
	svc := NewActivityService(
		fakeScan{active: true, progress: ScanProgress{PercentDone: 62, CurrentFile: "movie.mkv", FilesFound: 1234}},
		fakeBatch{active: true, percentDone: 40, current: 12, total: 30, item: "ep.mkv"},
		fakeBatch{active: true, percentDone: 25, current: 3, total: 12, item: "gen.mkv"},
		fakeDownloads{counts: &qbittorrent.DownloadCounts{All: 8, Downloading: 3}},
		fakeParse{
			pending: []*models.ParseJob{pj(models.ParseJobPending, "a"), pj(models.ParseJobPending, "b")},
			all:     []*models.ParseJob{pj(models.ParseJobCompleted, "done.mkv"), pj(models.ParseJobFailed, "bad.mkv"), pj(models.ParseJobProcessing, "wip.mkv")},
		},
	)

	a := svc.GetActivity(context.Background())

	if a.ActiveJobs.Status != sectionOK || len(a.ActiveJobs.Jobs) != 3 {
		t.Fatalf("active = %+v, want ok with 3 jobs", a.ActiveJobs)
	}
	if a.ActiveJobs.Jobs[0].Kind != "scan" || a.ActiveJobs.Jobs[0].PercentDone != 62 || a.ActiveJobs.Jobs[0].Current != 1234 {
		t.Errorf("scan job = %+v", a.ActiveJobs.Jobs[0])
	}
	if a.ActiveJobs.Jobs[1].Kind != "subtitle_batch" || a.ActiveJobs.Jobs[1].Total != 30 || a.ActiveJobs.Jobs[1].PercentDone != 40 {
		t.Errorf("subtitle job = %+v", a.ActiveJobs.Jobs[1])
	}
	// 9R-16 AC 10: generation batch surfaces as its own active-job kind.
	if a.ActiveJobs.Jobs[2].Kind != "generation_batch" || a.ActiveJobs.Jobs[2].Total != 12 ||
		a.ActiveJobs.Jobs[2].PercentDone != 25 || a.ActiveJobs.Jobs[2].Detail != "gen.mkv" {
		t.Errorf("generation job = %+v", a.ActiveJobs.Jobs[2])
	}
	if a.Pending.Status != sectionOK || a.Pending.ParseCount != 2 {
		t.Errorf("pending = %+v, want ok/2", a.Pending)
	}
	// Queued = All(8) - Downloading(3) - Completed(0) - Seeding(0) = 5.
	if a.Downloads.Status != sectionOK || a.Downloads.Downloading != 3 || a.Downloads.Total != 8 || a.Downloads.Queued != 5 {
		t.Errorf("downloads = %+v, want ok/3/queued5/total8", a.Downloads)
	}
	if a.Recent.Status != sectionOK || len(a.Recent.Events) != 2 {
		t.Fatalf("recent = %+v, want ok with 2 terminal events", a.Recent)
	}
	if a.Recent.Events[0].Result != "completed" || a.Recent.Events[1].Result != "failed" {
		t.Errorf("recent results = %+v", a.Recent.Events)
	}
}

func TestActivity_NoActiveJobsIsOKEmpty(t *testing.T) {
	svc := NewActivityService(
		fakeScan{active: false},
		fakeBatch{active: false},
		fakeBatch{active: false},
		fakeDownloads{counts: &qbittorrent.DownloadCounts{}},
		fakeParse{},
	)
	a := svc.GetActivity(context.Background())
	if a.ActiveJobs.Status != sectionOK || len(a.ActiveJobs.Jobs) != 0 {
		t.Errorf("active = %+v, want ok with 0 jobs", a.ActiveJobs)
	}
	// Empty slices, not nil — serialize as [] for a safe client render.
	if a.ActiveJobs.Jobs == nil || a.Recent.Events == nil {
		t.Error("jobs/events should be non-nil empty slices")
	}
}

func TestActivity_PendingFailsSoft(t *testing.T) {
	svc := NewActivityService(
		fakeScan{}, fakeBatch{}, fakeBatch{}, fakeDownloads{counts: &qbittorrent.DownloadCounts{}},
		fakeParse{pendingErr: errors.New("db down"), all: []*models.ParseJob{}},
	)
	a := svc.GetActivity(context.Background())
	if a.Pending.Status != sectionUnavailable || a.Pending.Error == "" {
		t.Errorf("pending = %+v, want unavailable+error", a.Pending)
	}
	// Failure is isolated — other sections stay OK.
	if a.Downloads.Status != sectionOK || a.ActiveJobs.Status != sectionOK || a.Recent.Status != sectionOK {
		t.Errorf("a non-pending section regressed: dl=%s active=%s recent=%s",
			a.Downloads.Status, a.ActiveJobs.Status, a.Recent.Status)
	}
}

func TestActivity_DownloadsFailSoft(t *testing.T) {
	svc := NewActivityService(
		fakeScan{}, fakeBatch{}, fakeBatch{}, fakeDownloads{err: errors.New("qb unreachable")},
		fakeParse{},
	)
	a := svc.GetActivity(context.Background())
	if a.Downloads.Status != sectionUnavailable || a.Downloads.Error == "" {
		t.Errorf("downloads = %+v, want unavailable+error", a.Downloads)
	}
}

func TestActivity_RecentCapsAndFiltersTerminal(t *testing.T) {
	jobs := []*models.ParseJob{}
	for i := 0; i < 12; i++ {
		jobs = append(jobs, pj(models.ParseJobCompleted, "f"))
	}
	// Non-terminal jobs must be filtered out of the recent feed.
	jobs = append(jobs, pj(models.ParseJobPending, "skip"), pj(models.ParseJobProcessing, "skip"))
	svc := NewActivityService(fakeScan{}, fakeBatch{}, fakeBatch{}, fakeDownloads{counts: &qbittorrent.DownloadCounts{}}, fakeParse{all: jobs})

	a := svc.GetActivity(context.Background())
	if len(a.Recent.Events) != recentEventsMax {
		t.Errorf("recent events = %d, want capped at %d", len(a.Recent.Events), recentEventsMax)
	}
	for _, e := range a.Recent.Events {
		if e.Result != "completed" && e.Result != "failed" {
			t.Errorf("non-terminal event leaked: %+v", e)
		}
	}
}

func TestActivity_RecentPrefersCompletedAt(t *testing.T) {
	completed := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	job := &models.ParseJob{Status: models.ParseJobCompleted, FileName: "x.mkv", UpdatedAt: time.Now(), CompletedAt: &completed}
	svc := NewActivityService(fakeScan{}, fakeBatch{}, fakeBatch{}, fakeDownloads{counts: &qbittorrent.DownloadCounts{}}, fakeParse{all: []*models.ParseJob{job}})

	a := svc.GetActivity(context.Background())
	if len(a.Recent.Events) != 1 || !a.Recent.Events[0].At.Equal(completed) {
		t.Errorf("recent At = %+v, want completedAt %v", a.Recent.Events, completed)
	}
}

func TestActivity_NilSourcesDegradeGracefully(t *testing.T) {
	svc := NewActivityService(nil, nil, nil, nil, nil)
	a := svc.GetActivity(context.Background()) // must not panic
	if a.Pending.Status != sectionUnavailable ||
		a.Downloads.Status != sectionUnavailable ||
		a.Recent.Status != sectionUnavailable {
		t.Errorf("nil single-source sections should be unavailable, got %+v", a)
	}
	// Active-jobs stays OK-empty even with nil sources — "no job running" is a valid
	// state, not an error.
	if a.ActiveJobs.Status != sectionOK || len(a.ActiveJobs.Jobs) != 0 {
		t.Errorf("active = %+v, want ok-empty", a.ActiveJobs)
	}
}
