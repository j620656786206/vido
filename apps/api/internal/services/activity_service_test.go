package services

import (
	"context"
	"encoding/json"
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
		// bugfix-e: buckets are disjoint and total (All == the sum), so the
		// fixture states every bucket instead of leaving a phantom remainder
		// for the old subtraction to sweep into 排隊中.
		fakeDownloads{counts: &qbittorrent.DownloadCounts{All: 8, Downloading: 3, Queued: 5}},
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
	// bugfix-e: Queued is reported from its OWN bucket, not derived by subtraction.
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

// ─── bugfix-e: errored/paused torrents must not be swept into 排隊中 ──────────

// The live-NAS shape that motivated the story: 3,068 errored + 34 paused/queued
// torrents were ALL reported as queued, while /downloads/counts simultaneously
// reported error=3068 — two endpoints contradicting each other about the same
// library. Table-driven over the buckets that used to vanish.
func TestActivity_DownloadsBucketsAreTruthful(t *testing.T) {
	cases := []struct {
		name                                                 string
		counts                                               qbittorrent.DownloadCounts
		wantDownloading, wantQueued, wantErrored, wantPaused int
	}{
		{
			name:   "live NAS shape — errors dominate and must not read as queued",
			counts: qbittorrent.DownloadCounts{All: 3641, Downloading: 500, Queued: 39, Paused: 34, Error: 3068},
			// Before bugfix-e this reported queued=3141 (error+paused+queued).
			wantDownloading: 500, wantQueued: 39, wantErrored: 3068, wantPaused: 34,
		},
		{
			name:            "errored only",
			counts:          qbittorrent.DownloadCounts{All: 5, Error: 5},
			wantDownloading: 0, wantQueued: 0, wantErrored: 5, wantPaused: 0,
		},
		{
			name:            "paused only — paused is its own bucket, not queued",
			counts:          qbittorrent.DownloadCounts{All: 4, Paused: 4},
			wantDownloading: 0, wantQueued: 0, wantErrored: 0, wantPaused: 4,
		},
		{
			name:            "mixed healthy library",
			counts:          qbittorrent.DownloadCounts{All: 10, Downloading: 2, Queued: 3, Paused: 1, Completed: 3, Seeding: 1},
			wantDownloading: 2, wantQueued: 3, wantErrored: 0, wantPaused: 1,
		},
		{
			name:            "all zero",
			counts:          qbittorrent.DownloadCounts{},
			wantDownloading: 0, wantQueued: 0, wantErrored: 0, wantPaused: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			counts := tc.counts
			svc := NewActivityService(fakeScan{}, fakeBatch{}, fakeBatch{}, fakeDownloads{counts: &counts}, fakeParse{})
			d := svc.GetActivity(context.Background()).Downloads

			if d.Status != sectionOK {
				t.Fatalf("status = %q, want ok", d.Status)
			}
			if d.Downloading != tc.wantDownloading || d.Queued != tc.wantQueued ||
				d.Errored != tc.wantErrored || d.Paused != tc.wantPaused {
				t.Errorf("buckets = downloading:%d queued:%d errored:%d paused:%d, want %d/%d/%d/%d",
					d.Downloading, d.Queued, d.Errored, d.Paused,
					tc.wantDownloading, tc.wantQueued, tc.wantErrored, tc.wantPaused)
			}
			if d.Total != tc.counts.All {
				t.Errorf("total = %d, want %d", d.Total, tc.counts.All)
			}
		})
	}
}

// AC #1 wire shape: the additive count keys are snake_case and, critically,
// `errored` (the count) never collides with `error` (the section failure
// message, which stays omitempty and absent on a healthy section).
func TestActivity_DownloadsSectionWireKeys(t *testing.T) {
	counts := qbittorrent.DownloadCounts{All: 9, Downloading: 1, Queued: 2, Paused: 3, Error: 3}
	svc := NewActivityService(fakeScan{}, fakeBatch{}, fakeBatch{}, fakeDownloads{counts: &counts}, fakeParse{})

	blob, err := json.Marshal(svc.GetActivity(context.Background()).Downloads)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(blob, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	for key, want := range map[string]float64{
		"downloading": 1, "queued": 2, "paused": 3, "errored": 3, "total": 9,
	} {
		v, ok := got[key]
		if !ok {
			t.Errorf("key %q missing from the wire shape", key)
			continue
		}
		if v != want {
			t.Errorf("%q = %v, want %v", key, v, want)
		}
	}
	if _, ok := got["error"]; ok {
		t.Errorf("`error` is the section failure MESSAGE and must stay absent on a healthy section; got %v", got["error"])
	}
}

// A section-level failure still carries the string `error` message — the additive
// count keys must not have displaced it.
func TestActivity_DownloadsUnavailableKeepsErrorMessage(t *testing.T) {
	svc := NewActivityService(fakeScan{}, fakeBatch{}, fakeBatch{}, fakeDownloads{err: errors.New("qbt down")}, fakeParse{})

	blob, err := json.Marshal(svc.GetActivity(context.Background()).Downloads)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(blob, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["status"] != sectionUnavailable {
		t.Errorf("status = %v, want %q", got["status"], sectionUnavailable)
	}
	if got["error"] != "qbt down" {
		t.Errorf("error message = %v, want the qBT failure string", got["error"])
	}
}
