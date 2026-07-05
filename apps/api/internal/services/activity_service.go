package services

import (
	"context"
	"time"

	"github.com/vido/api/internal/models"
)

// ActivityService composes the background-job activity sections that feed the v2
// Activity hub (GET /api/v1/activity — UX Redesign D4-1 / ux3-2-1). Like
// StatusSummaryService it READS existing services (Rule 4 — no duplicated subsystem
// logic) and is fail-soft per section (B1/F3): a degraded source marks ONLY its section
// "unavailable" and never fails the whole endpoint (the web client renders an empty/stale
// section, never a fail page — N1/N4).
//
// v1 aggregates the sources that exist today: live scan + batch-subtitle progress
// (進行中), pending-parse count (待處理), download counts (下載), and recent terminal
// parse events (活動記錄). The design's AI-correction active jobs and a fully-persisted
// activity log are GREENFIELD (Rule 24 — honour backend capability, don't fabricate):
// AI active-jobs are omitted until job-tracking exists, and the recent feed is parse
// events only until an activity-log table lands. The web client renders what's present.
type ActivityService struct {
	scan       scanStateSource     // reused from status_summary_service.go
	batch      batchJobSource      // subtitle.BatchProcessor (primitives only — no import)
	generation batchJobSource      // GenerationBatchProcessor (Story 9R-16 AC 10)
	downloads  downloadCountSource // reused from status_summary_service.go
	parse      parseJobSource      // repository.ParseJobRepository (existing interface)
}

// batchJobSource is the active batch-subtitle job (subtitle.BatchProcessor). Primitive
// returns on purpose — internal/services must not import internal/subtitle (that package
// imports services; a shared type here would create an import cycle).
type batchJobSource interface {
	ActivityProgress() (active bool, percentDone, current, total int, currentItem string)
}

// parseJobSource reads the parse-job store (repository.ParseJobRepository). Both the
// pending count and the recent-terminal feed go through its existing interface methods —
// no new repository surface (and no mock churn).
type parseJobSource interface {
	GetPending(ctx context.Context, limit int) ([]*models.ParseJob, error)
	ListAll(ctx context.Context, limit int) ([]*models.ParseJob, error)
}

// NewActivityService wires the existing services/repos it composes. generation
// is the Route C generation-batch source (9R-16 AC 10) — same primitive
// interface as the fetch-batch source, nil-safe (fail-soft per section).
func NewActivityService(scan scanStateSource, batch batchJobSource, generation batchJobSource, downloads downloadCountSource, parse parseJobSource) *ActivityService {
	return &ActivityService{scan: scan, batch: batch, generation: generation, downloads: downloads, parse: parse}
}

const (
	// pendingCountLimit caps the pending-parse scan — a count display tolerates a cap,
	// and a healthy queue never approaches it.
	pendingCountLimit = 500
	// recentScanWindow is how many recent jobs to inspect; recentEventsMax bounds the feed.
	recentScanWindow = 50
	recentEventsMax  = 8
)

// ActivitySummary is the GET /api/v1/activity payload. snake_case tags match the codebase
// convention; the web client camelCases at the fetchApi boundary (Rule 18).
type ActivitySummary struct {
	ActiveJobs ActiveJobsSection `json:"active_jobs"`
	Pending    PendingSection    `json:"pending"`
	Downloads  DownloadsSection  `json:"downloads"`
	Recent     RecentSection     `json:"recent"`
}

type ActiveJobsSection struct {
	Status string      `json:"status"`
	Jobs   []ActiveJob `json:"jobs"`
	Error  string      `json:"error,omitempty"`
}

// ActiveJob is one in-flight background job. Kind drives the web client's icon + label
// (copy lives on the client for i18n) — the backend stays copy-free.
type ActiveJob struct {
	Kind        string `json:"kind"` // "scan" | "subtitle_batch" | "generation_batch"
	PercentDone int    `json:"percent_done"`
	Detail      string `json:"detail,omitempty"` // current file / item (raw)
	Current     int    `json:"current,omitempty"`
	Total       int    `json:"total,omitempty"`
}

type PendingSection struct {
	Status     string `json:"status"`
	ParseCount int    `json:"parse_count"`
	Error      string `json:"error,omitempty"`
}

type DownloadsSection struct {
	Status      string `json:"status"`
	Downloading int    `json:"downloading"`
	Queued      int    `json:"queued"`
	Total       int    `json:"total"`
	Error       string `json:"error,omitempty"`
}

type RecentSection struct {
	Status string        `json:"status"`
	Events []RecentEvent `json:"events"`
	Error  string        `json:"error,omitempty"`
}

// RecentEvent is one recently-finished job. v1 sources parse jobs only (scan/subtitle/AI
// completion is not persisted yet). Kind/Result drive the client's icon + copy.
type RecentEvent struct {
	Kind   string    `json:"kind"`   // "parse"
	Result string    `json:"result"` // "completed" | "failed"
	Detail string    `json:"detail,omitempty"`
	At     time.Time `json:"at"`
}

// GetActivity composes all four sections. It NEVER returns an error — each section
// degrades independently (fail-soft).
func (s *ActivityService) GetActivity(ctx context.Context) ActivitySummary {
	return ActivitySummary{
		ActiveJobs: s.activeJobsSection(),
		Pending:    s.pendingSection(ctx),
		Downloads:  s.downloadsSection(ctx),
		Recent:     s.recentSection(ctx),
	}
}

// activeJobsSection lists the in-flight jobs. An empty list is a valid OK state ("nothing
// running") — the section is only ever OK here, since the in-memory sources can't error.
func (s *ActivityService) activeJobsSection() ActiveJobsSection {
	jobs := []ActiveJob{}
	if s.scan != nil && s.scan.IsScanActive() {
		p := s.scan.GetProgress()
		jobs = append(jobs, ActiveJob{
			Kind:        "scan",
			PercentDone: p.PercentDone,
			Detail:      p.CurrentFile,
			Current:     p.FilesFound,
		})
	}
	if s.batch != nil {
		if active, pct, cur, total, item := s.batch.ActivityProgress(); active {
			jobs = append(jobs, ActiveJob{
				Kind:        "subtitle_batch",
				PercentDone: pct,
				Detail:      item,
				Current:     cur,
				Total:       total,
			})
		}
	}
	if s.generation != nil {
		if active, pct, cur, total, item := s.generation.ActivityProgress(); active {
			jobs = append(jobs, ActiveJob{
				Kind:        "generation_batch",
				PercentDone: pct,
				Detail:      item,
				Current:     cur,
				Total:       total,
			})
		}
	}
	return ActiveJobsSection{Status: sectionOK, Jobs: jobs}
}

func (s *ActivityService) pendingSection(ctx context.Context) PendingSection {
	if s.parse == nil {
		return PendingSection{Status: sectionUnavailable, Error: "service unavailable"}
	}
	jobs, err := s.parse.GetPending(ctx, pendingCountLimit)
	if err != nil {
		return PendingSection{Status: sectionUnavailable, Error: err.Error()}
	}
	return PendingSection{Status: sectionOK, ParseCount: len(jobs)}
}

func (s *ActivityService) downloadsSection(ctx context.Context) DownloadsSection {
	if s.downloads == nil {
		return DownloadsSection{Status: sectionUnavailable, Error: "service unavailable"}
	}
	c, err := s.downloads.GetDownloadCounts(ctx)
	if err != nil || c == nil {
		msg := "unavailable"
		if err != nil {
			msg = err.Error()
		}
		return DownloadsSection{Status: sectionUnavailable, Error: msg}
	}
	// Queued = anything tracked but not actively downloading and not finished
	// (paused / stalled / queued / errored). Clamped at zero.
	queued := c.All - c.Downloading - c.Completed - c.Seeding
	if queued < 0 {
		queued = 0
	}
	return DownloadsSection{Status: sectionOK, Downloading: c.Downloading, Queued: queued, Total: c.All}
}

func (s *ActivityService) recentSection(ctx context.Context) RecentSection {
	if s.parse == nil {
		return RecentSection{Status: sectionUnavailable, Error: "service unavailable"}
	}
	jobs, err := s.parse.ListAll(ctx, recentScanWindow)
	if err != nil {
		return RecentSection{Status: sectionUnavailable, Error: err.Error()}
	}
	events := []RecentEvent{}
	for _, j := range jobs {
		if j.Status != models.ParseJobCompleted && j.Status != models.ParseJobFailed {
			continue
		}
		at := j.UpdatedAt
		if j.CompletedAt != nil {
			at = *j.CompletedAt
		}
		events = append(events, RecentEvent{
			Kind:   "parse",
			Result: string(j.Status),
			Detail: j.FileName,
			At:     at,
		})
		if len(events) >= recentEventsMax {
			break
		}
	}
	return RecentSection{Status: sectionOK, Events: events}
}
