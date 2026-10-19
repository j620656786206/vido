package services

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"sort"
	"time"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/qbittorrent"
	"github.com/vido/api/internal/repository"
)

// sectionNotConfigured marks the downloads section when qBittorrent has not
// been set up: nothing failed, so the hub hides the section instead of
// offering a pointless retry (disc-activity-downloads-unconfigured-copy
// AC #1 [@contract-v1], additive to ok/unavailable).
const sectionNotConfigured = "not_configured"

// ActivityService composes the background-job activity sections that feed the v2
// Activity hub (GET /api/v1/activity — UX Redesign D4-1 / ux3-2-1). Like
// StatusSummaryService it READS existing services (Rule 4 — no duplicated subsystem
// logic) and is fail-soft per section (B1/F3): a degraded source marks ONLY its section
// "unavailable" and never fails the whole endpoint (the web client renders an empty/stale
// section, never a fail page — N1/N4).
//
// It aggregates the sources that exist today: live scan + batch-subtitle progress
// (進行中), pending-parse count (待處理), download counts (下載), and the activity
// record (活動記錄) — finished parse jobs merged with finished subtitle-generation
// runs (disc-2026-10-activity-subtitle-run-time-a). The design's AI-correction active
// jobs and a scan-finished event are GREENFIELD (Rule 24 — honour backend capability,
// don't fabricate): nothing persists them yet, so they are omitted. The web client
// renders what's present.
type ActivityService struct {
	scan          scanStateSource     // reused from status_summary_service.go
	batch         batchJobSource      // subtitle.BatchProcessor (primitives only — no import)
	generation    batchJobSource      // GenerationBatchProcessor (Story 9R-16 AC 10)
	transcription batchJobSource      // TranscriptionService, solo jobs only (disc-2026-07-transcription-active-jobs)
	downloads     downloadCountSource // reused from status_summary_service.go
	parse         parseJobSource      // repository.ParseJobRepository (existing interface)
	steps         GenerationStepReader
	runs          recentRunSource // repository.SubtitleRunRepository; nil = parse events only
}

// recentRunSource is the activity record's subtitle source
// (repository.SubtitleRunRepository.RecentTerminalRuns).
type recentRunSource interface {
	RecentTerminalRuns(ctx context.Context, limit int) ([]repository.RecentSubtitleRun, error)
}

// SetRecentRuns wires finished subtitle-generation runs into the activity
// record. nil (the default) keeps the record as it was: parse events only.
func (s *ActivityService) SetRecentRuns(src recentRunSource) { s.runs = src }

// SetStepReader wires the per-item progress the generating rows show. nil (the
// default) keeps the rows as they were: no step.
func (s *ActivityService) SetStepReader(r GenerationStepReader) { s.steps = r }

// stepOf is the step for source's single active item, or nil.
func (s *ActivityService) stepOf(source batchJobSource) *ActiveJobStep {
	if s.steps == nil {
		return nil
	}
	named, ok := source.(activeMediaSource)
	if !ok {
		return nil
	}
	mediaID, ok := named.ActiveMediaID()
	if !ok || mediaID == "" {
		return nil
	}
	phase, current, total, ok := s.steps.Step(mediaID)
	if !ok {
		return nil
	}
	return &ActiveJobStep{Phase: phase, Current: current, Total: total}
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
func NewActivityService(scan scanStateSource, batch batchJobSource, generation batchJobSource, transcription batchJobSource, downloads downloadCountSource, parse parseJobSource) *ActivityService {
	return &ActivityService{scan: scan, batch: batch, generation: generation, transcription: transcription, downloads: downloads, parse: parse}
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
	Kind        string `json:"kind"` // "scan" | "subtitle_batch" | "generation_batch" | "transcription"
	PercentDone int    `json:"percent_done"`
	Detail      string `json:"detail,omitempty"` // current file / item (raw)
	Current     int    `json:"current,omitempty"`
	Total       int    `json:"total,omitempty"`
	// Step is where the generating item is right now — "轉錄 2/6 段"
	// (disc-2026-10-activity-generation-progress-detail, an additive field).
	// transcription and generation_batch only; absent when unknown or when
	// several solo jobs run (which one would it describe?).
	Step *ActiveJobStep `json:"step,omitempty"`
}

// ActiveJobStep is one item's stage and, when the run reported one, its count:
// transcribing = chunks done of total, translating = cues done of total.
// Phase is extracting | transcribing | translating.
type ActiveJobStep struct {
	Phase   string `json:"phase"`
	Current int    `json:"current,omitempty"`
	Total   int    `json:"total,omitempty"`
}

// GenerationStepReader is the per-item progress the hub observer keeps
// (subtitle.PhaseTracker.Step). ok=false: nothing recorded for the item.
type GenerationStepReader interface {
	Step(mediaID string) (phase string, current, total int, ok bool)
}

// activeMediaSource is the optional seam a job source implements to name the
// ONE item it is generating; ok=false when idle or when several run.
type activeMediaSource interface {
	ActiveMediaID() (mediaID string, ok bool)
}

type PendingSection struct {
	Status     string `json:"status"`
	ParseCount int    `json:"parse_count"`
	Error      string `json:"error,omitempty"`
}

// DownloadsSection is the activity hub's downloads bucket split. `Error` is the
// section-level FAILURE MESSAGE (string); `Errored` is the count of errored
// torrents — deliberately distinct names so the additive count never collides
// with the pre-existing key (bugfix-e AC #1).
type DownloadsSection struct {
	Status      string `json:"status"`
	Downloading int    `json:"downloading"`
	Queued      int    `json:"queued"`
	Errored     int    `json:"errored"`
	Paused      int    `json:"paused"`
	Total       int    `json:"total"`
	Error       string `json:"error,omitempty"`
}

type RecentSection struct {
	Status string        `json:"status"`
	Events []RecentEvent `json:"events"`
	Error  string        `json:"error,omitempty"`
}

// RecentEvent is one recently-finished job: a parse job (kind "parse") or a
// subtitle-generation run (kind "subtitle"). Kind/Result drive the client's
// icon + copy; the backend stays copy-free.
//
// A parse event carries Detail (the file name) and none of the subtitle keys —
// its JSON is unchanged from v0. A subtitle event never carries Detail (that
// would be the raw Go error, with NAS paths); its keys are
// disc-2026-10-activity-subtitle-run-time-a AC #1 [@contract-v1].
type RecentEvent struct {
	Kind   string    `json:"kind"`   // "parse" | "subtitle"
	Result string    `json:"result"` // "completed" | "failed"
	Detail string    `json:"detail,omitempty"`
	At     time.Time `json:"at"`

	// Subtitle only. MediaTitle is a pointer so a subtitle event always sends
	// the key ("" = the item left the library) while a parse event sends none.
	MediaType     string  `json:"media_type,omitempty"` // "movie" | "episode" | "series"
	MediaTitle    *string `json:"media_title,omitempty"`
	SeasonNumber  *int    `json:"season_number,omitempty"`  // episode runs whose episode still exists
	EpisodeNumber *int    `json:"episode_number,omitempty"` // same
	Route         string  `json:"route,omitempty"`          // models.SubtitleRunRoute*; absent before migration 041
	CueCount      int     `json:"cue_count,omitempty"`      // completed runs with cues
	// DurationSeconds is completed_at − started_at rounded to the second —
	// the item from start of processing to the end, including any wait for an
	// extraction slot. Absent when the clock went backwards.
	DurationSeconds *int `json:"duration_seconds,omitempty"`
	// SpentUSD is the raw recorded spend, never pre-rounded (the client rounds
	// with decimal.js). 0 is sent as 0 (沒花錢); absent = not recorded.
	SpentUSD      *float64 `json:"spent_usd,omitempty"`
	FailureReason string   `json:"failure_reason,omitempty"` // models.SubtitleRunFailure*; failed only

	// id breaks ties in the merge order; never on the wire.
	id string
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
				Step:        s.stepOf(s.generation),
			})
		}
	}
	if s.transcription != nil {
		if active, pct, cur, total, item := s.transcription.ActivityProgress(); active {
			job := ActiveJob{
				Kind:        "transcription",
				PercentDone: pct,
				Detail:      item,
				Current:     cur,
				Total:       total,
				Step:        s.stepOf(s.transcription),
			}
			// A counted step is the row's progress: the bar and percent_done
			// follow the chunks / cues of the one running item.
			if job.Step != nil && job.Step.Total > 0 {
				job.PercentDone = job.Step.Current * 100 / job.Step.Total
			}
			jobs = append(jobs, job)
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
	var connErr *qbittorrent.ConnectionError
	if errors.As(err, &connErr) && connErr.Code == qbittorrent.ErrCodeNotConfigured {
		return DownloadsSection{Status: sectionNotConfigured}
	}
	if err != nil || c == nil {
		msg := "unavailable"
		if err != nil {
			msg = err.Error()
		}
		return DownloadsSection{Status: sectionUnavailable, Error: msg}
	}
	// bugfix-e: each bucket is reported from its OWN count instead of being
	// derived by subtraction. The old `All - Downloading - Completed - Seeding`
	// swept errored AND paused torrents into 排隊中 — a live NAS showed
	// queued=3102 (3068 errored + 34 paused/queued) while /downloads/counts
	// simultaneously reported error=3068, i.e. two endpoints contradicting each
	// other about the same torrents. Counts are disjoint and total (see
	// DownloadCounts), so no clamp is needed: every value is already ≥ 0.
	return DownloadsSection{
		Status:      sectionOK,
		Downloading: c.Downloading,
		Queued:      c.Queued,
		Errored:     c.Error,
		Paused:      c.Paused,
		Total:       c.All,
	}
}

// recentSection merges the two finished-job sources newest first and keeps
// the latest recentEventsMax — one cap for both; the newest win. Each source
// degrades on its own: one failing leaves the other's events (status ok, the
// failure logged). The section is unavailable when both are missing or
// failing — and also when one failed and the other has nothing to show: an
// "ok" empty record renders as nothing at all, which would hide the failure
// (正式機 has no parse jobs, so a broken subtitle source would otherwise just
// make the section vanish, with no error and no retry).
//
// The wire `error` stays what it was when only the parse source existed (its
// own message); any case involving the subtitle source sends a fixed message
// and keeps the repository error in the log.
func (s *ActivityService) recentSection(ctx context.Context) RecentSection {
	parseEvents, parseErr := s.recentParseEvents(ctx)
	runEvents, runErr := s.recentRunEvents(ctx)
	parseFailed := parseErr != nil && !errors.Is(parseErr, errSourceNotWired)
	runFailed := runErr != nil && !errors.Is(runErr, errSourceNotWired)

	if parseFailed {
		slog.Warn("activity: recent parse source failed", "error", parseErr)
	}
	if runFailed {
		slog.Warn("activity: recent subtitle-run source failed", "error", runErr)
	}

	if parseErr != nil && runErr != nil {
		if errors.Is(runErr, errSourceNotWired) {
			// Subtitle source not wired: exactly the parse-only behaviour.
			return RecentSection{Status: sectionUnavailable, Error: parseErr.Error()}
		}
		return RecentSection{Status: sectionUnavailable, Error: errSourceNotWired.Error()}
	}
	if (parseFailed || runFailed) && len(parseEvents)+len(runEvents) == 0 {
		return RecentSection{Status: sectionUnavailable, Error: errSourceNotWired.Error()}
	}

	events := append(parseEvents, runEvents...)
	sort.SliceStable(events, func(i, j int) bool { return recentEventBefore(events[i], events[j]) })
	if len(events) > recentEventsMax {
		events = events[:recentEventsMax]
	}
	if events == nil {
		events = []RecentEvent{}
	}
	return RecentSection{Status: sectionOK, Events: events}
}

// recentEventBefore is the record's fixed order: newest first; at the same
// instant a subtitle run before a parse job; then by id.
func recentEventBefore(a, b RecentEvent) bool {
	if !a.At.Equal(b.At) {
		return a.At.After(b.At)
	}
	if a.Kind != b.Kind {
		return a.Kind == recentKindSubtitle
	}
	return a.id < b.id
}

const (
	recentKindParse    = "parse"
	recentKindSubtitle = "subtitle"
)

// errSourceNotWired marks a nil source — "not there", which is not worth a
// warning, unlike a source that errored.
var errSourceNotWired = errors.New("service unavailable")

func (s *ActivityService) recentParseEvents(ctx context.Context) ([]RecentEvent, error) {
	if s.parse == nil {
		return nil, errSourceNotWired
	}
	jobs, err := s.parse.ListAll(ctx, recentScanWindow)
	if err != nil {
		return nil, err
	}
	var events []RecentEvent
	for _, j := range jobs {
		if j.Status != models.ParseJobCompleted && j.Status != models.ParseJobFailed {
			continue
		}
		at := j.UpdatedAt
		if j.CompletedAt != nil {
			at = *j.CompletedAt
		}
		events = append(events, RecentEvent{
			Kind:   recentKindParse,
			Result: string(j.Status),
			Detail: j.FileName,
			At:     at,
			id:     j.ID,
		})
		if len(events) >= recentEventsMax {
			break
		}
	}
	return events, nil
}

func (s *ActivityService) recentRunEvents(ctx context.Context) ([]RecentEvent, error) {
	if s.runs == nil {
		return nil, errSourceNotWired
	}
	runs, err := s.runs.RecentTerminalRuns(ctx, recentEventsMax)
	if err != nil {
		return nil, err
	}
	var events []RecentEvent
	for _, r := range runs {
		if ev, ok := subtitleRunEvent(r); ok {
			events = append(events, ev)
		}
	}
	return events, nil
}

// subtitleRunEvent turns one finished run into its [@contract-v1] event. ok is
// false for a row the record must not show (the repository already filters
// these; this is the second lock).
func subtitleRunEvent(r repository.RecentSubtitleRun) (RecentEvent, bool) {
	run := r.Run
	if run.CompletedAt == nil ||
		(run.Status != models.SubtitleRunCompleted && run.Status != models.SubtitleRunFailed) {
		return RecentEvent{}, false
	}
	title := r.MediaTitle
	ev := RecentEvent{
		Kind:       recentKindSubtitle,
		Result:     string(run.Status),
		At:         run.CompletedAt.UTC(),
		MediaType:  run.MediaType,
		MediaTitle: &title,
		Route:      run.Route,
		id:         run.ID,
	}
	// Numbers only travel with a title: an episode whose series row is gone
	// is "an item that left the library", not "S01E04 of nothing".
	if run.MediaType == models.SubtitleRunMediaEpisode && title != "" && r.SeasonNumber != nil && r.EpisodeNumber != nil {
		sn, en := *r.SeasonNumber, *r.EpisodeNumber
		ev.SeasonNumber, ev.EpisodeNumber = &sn, &en
	}
	if run.Status == models.SubtitleRunCompleted && run.CueCount > 0 {
		ev.CueCount = run.CueCount
	}
	if !run.StartedAt.IsZero() {
		if d := run.CompletedAt.Sub(run.StartedAt); d >= 0 {
			secs := int(math.Round(d.Seconds()))
			ev.DurationSeconds = &secs
		}
	}
	if run.SpentUSD != nil {
		spent := *run.SpentUSD
		ev.SpentUSD = &spent
	}
	if run.Status == models.SubtitleRunFailed {
		ev.FailureReason = models.SubtitleRunFailureReason(run.ErrorMessage)
	}
	return ev, true
}
