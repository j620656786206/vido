// Package services — movie_subtitle_tracks_backfill_service.go
//
// disc-2026-10-movie-subtitle-tracks-unknown-refresh: give every movie an
// answer to "does it have Chinese subtitles?".
//
// A movie's subtitle_tracks is written by enrichment's ffprobe pass. Two gaps
// leave it NULL ("we don't know") for good — 12 movies on the NAS:
//
//  1. scans before disc-2026-10-subtitle-filter-disagrees-with-badges did not
//     write an empty probe result, and nothing re-reads a scanned row;
//  2. when an NFO already supplied the tech info, enrichment skips the probe
//     altogether, so a movie with no sidecar file stays NULL — this one still
//     creates new NULLs today.
//
// This service probes exactly those rows — once a little after boot and after
// every scan — and stores the same merge of embedded tracks and sidecar files
// the scan would have. It only ever fills a NULL; a stored answer, `[]`
// included, is never re-read or overwritten.
package services

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/vido/api/internal/models"
)

// Implements: disc-2026-10-movie-subtitle-tracks-unknown-refresh AC #1–#7;
// disc-2026-11-backfill-subtitle-cue-counts AC #3

// MovieSubtitleTracksBackfillRepo is the slice of the movie repository the
// backfill needs. *repository.MovieRepository satisfies it.
type MovieSubtitleTracksBackfillRepo interface {
	FindMissingSubtitleTracks(ctx context.Context, afterID string, limit int) ([]models.Movie, error)
	UpdateSubtitleTracksIfMissing(ctx context.Context, id, tracksJSON string) (bool, error)
}

// MovieCueCountRepo is the OPTIONAL slice the cue-count backfill needs
// (disc-2026-11-backfill-subtitle-cue-counts); *repository.MovieRepository
// satisfies it. A repo without it simply skips that phase.
type MovieCueCountRepo interface {
	FindMoviesMissingCueCounts(ctx context.Context, afterID string, limit int) ([]models.Movie, error)
	ReplaceSubtitleTracks(ctx context.Context, id, previous, next string) (bool, error)
}

// movieTracksBackfillPage is how many rows one listing returns; a pass walks
// every page, so rows that stay NULL (file missing, probe failed) never hide
// the ones behind them. A var so tests can page with a handful of rows.
var movieTracksBackfillPage = 500

// MovieSubtitleTracksBackfillService fills unknown movie subtitle tracks.
type MovieSubtitleTracksBackfillService struct {
	repo   MovieSubtitleTracksBackfillRepo
	prober EpisodeTrackProber
	logger *slog.Logger
	stat   func(string) (os.FileInfo, error)
	// sidecars lists the subtitle files beside a video; DetectExternalSubtitles
	// in production, the same reader the scan uses.
	sidecars func(videoPath string) []SubtitleTrack
	// cueRule turns on the cue-count phase; nil = off.
	cueRule CueCountRule

	mu      sync.Mutex
	wg      sync.WaitGroup
	running bool
	rerun   bool
	// failed remembers movies whose probe failed, valued by the file
	// signature that failed: the same unreadable file is not retried in this
	// process, but a replaced one is.
	failed map[string]string
}

// NewMovieSubtitleTracksBackfillService wires the backfill. A nil repo or
// prober makes every pass a no-op.
func NewMovieSubtitleTracksBackfillService(repo MovieSubtitleTracksBackfillRepo, prober EpisodeTrackProber, logger *slog.Logger) *MovieSubtitleTracksBackfillService {
	if logger == nil {
		logger = slog.Default()
	}
	return &MovieSubtitleTracksBackfillService{
		repo:     repo,
		prober:   prober,
		logger:   logger.With("service", "movie_subtitle_tracks_backfill"),
		stat:     os.Stat,
		sidecars: DetectExternalSubtitles,
		failed:   map[string]string{},
	}
}

// SetCueCountRule turns on the cue-count phase: a movie whose stored tracks
// were read before counts existed, and whose English track is what the
// extract lane would translate, is probed again once
// (disc-2026-11-backfill-subtitle-cue-counts). nil = off.
func (s *MovieSubtitleTracksBackfillService) SetCueCountRule(r CueCountRule) {
	s.cueRule = r
}

// MovieTracksBackfillResult is what one pass did, for the log line and tests.
type MovieTracksBackfillResult struct {
	Filled  int // movies that now have subtitle tracks (possibly `[]`)
	Failed  int // probe or write errors
	Skipped int // probe failed before on the same file, or a scan filled it first
	Missing int // file not there — share not mounted yet, or deleted
	Total   int // movies listed as unknown
	// CuesFilled / CuesUnstated: movies re-read for a cue count — it came
	// back, or the container states none (stored as -1, not read again).
	CuesFilled   int
	CuesUnstated int
}

// Run does one pass. Fail-soft per movie; the pass never returns an error.
func (s *MovieSubtitleTracksBackfillService) Run(ctx context.Context) MovieTracksBackfillResult {
	var res MovieTracksBackfillResult
	// No probe, no answer: a guess from sidecars alone would turn "unknown"
	// into "missing Chinese" for a film whose Chinese track is embedded.
	if s.repo == nil || s.prober == nil || !s.prober.IsAvailable() {
		return res
	}
	started := time.Now()
	afterID := ""
	for {
		page, err := s.repo.FindMissingSubtitleTracks(ctx, afterID, movieTracksBackfillPage)
		if err != nil {
			if ctx.Err() == nil {
				s.logger.Warn("movie subtitle tracks backfill: listing movies failed", "error", err)
			}
			break
		}
		if res.Total == 0 && len(page) > 0 {
			s.logger.Info("movie subtitle tracks backfill: pass started")
		}
		res.Total += len(page)
		for _, m := range page {
			if ctx.Err() != nil {
				s.logger.Info("movie subtitle tracks backfill: pass interrupted — resumes next pass", "filled", res.Filled)
				return res
			}
			s.fill(ctx, m, &res)
		}
		if len(page) < movieTracksBackfillPage {
			break
		}
		afterID = page[len(page)-1].ID
	}
	reread := s.backfillCues(ctx, &res)
	if res.Total == 0 && reread == 0 {
		return res
	}
	s.logger.Info("movie subtitle tracks backfill: pass finished",
		"filled", res.Filled, "failed", res.Failed, "skipped", res.Skipped, "missing_files", res.Missing,
		"cue_counts_filled", res.CuesFilled, "cue_counts_unstated", res.CuesUnstated,
		"took", time.Since(started).Round(time.Second).String())
	return res
}

// backfillCues is the cue-count phase: movies stored before counts were read
// whose English track is what the extract lane would translate get one more
// probe, written back with compare-and-set. Same per-file failure memory and
// the same embedded+sidecar merge as fill. Returns how many movies it
// probed (for the pass log).
func (s *MovieSubtitleTracksBackfillService) backfillCues(ctx context.Context, res *MovieTracksBackfillResult) int {
	repo, ok := s.repo.(MovieCueCountRepo)
	if !ok || s.cueRule == nil {
		return 0
	}
	probed := 0
	afterID := ""
	for {
		page, err := repo.FindMoviesMissingCueCounts(ctx, afterID, movieTracksBackfillPage)
		if err != nil {
			if ctx.Err() == nil {
				s.logger.Warn("movie subtitle tracks backfill: listing movies without cue counts failed", "error", err)
			}
			return probed
		}
		for _, m := range page {
			if ctx.Err() != nil {
				return probed
			}
			var stored []SubtitleTrack
			if json.Unmarshal([]byte(m.SubtitleTracks.String), &stored) != nil || !s.cueRule.NeedsCueCount(stored) {
				continue // no English track to translate (Chinese, image-only, und) — nothing to count
			}
			if s.refillCues(ctx, repo, m, res) {
				probed++
			}
		}
		if len(page) < movieTracksBackfillPage {
			return probed
		}
		afterID = page[len(page)-1].ID
	}
}

// refillCues re-probes one movie for its cue count; true when a probe ran.
func (s *MovieSubtitleTracksBackfillService) refillCues(ctx context.Context, repo MovieCueCountRepo, m models.Movie, res *MovieTracksBackfillResult) bool {
	path := m.FilePath.String
	info, err := s.stat(path)
	if err != nil {
		res.Missing++
		return false
	}
	sig := FileSignature(info)
	s.mu.Lock()
	failedSig, failedBefore := s.failed[m.ID]
	s.mu.Unlock()
	if failedBefore && failedSig == sig {
		res.Skipped++
		return false
	}
	tech, err := s.prober.Probe(ctx, path)
	if err != nil || tech == nil {
		if ctx.Err() != nil {
			return false
		}
		res.Failed++
		s.rememberFailure(m.ID, sig)
		s.logger.Warn("movie subtitle tracks backfill: cue-count probe failed", "movie_id", m.ID, "file", path, "error", err)
		return true
	}
	tracks := MergeSubtitleTracks(tech.SubtitleTracks, s.sidecars(path))
	raw, err := json.Marshal(tracks)
	if err != nil {
		res.Failed++
		s.rememberFailure(m.ID, sig)
		return true
	}
	written, err := repo.ReplaceSubtitleTracks(ctx, m.ID, m.SubtitleTracks.String, string(raw))
	if err != nil {
		res.Failed++
		s.logger.Warn("movie subtitle tracks backfill: cue-count write failed", "movie_id", m.ID, "error", err)
		return true
	}
	if !written {
		res.Skipped++ // a scan re-read the movie meanwhile — keep its answer
		return true
	}
	if s.cueRule.CuesFromTracks(tracks) > 0 {
		res.CuesFilled++
	} else {
		res.CuesUnstated++
	}
	return true
}

func (s *MovieSubtitleTracksBackfillService) fill(ctx context.Context, m models.Movie, res *MovieTracksBackfillResult) {
	path := m.FilePath.String
	info, err := s.stat(path)
	if err != nil {
		res.Missing++ // unmounted share or deleted file — try again next pass
		return
	}
	sig := FileSignature(info)
	s.mu.Lock()
	failedSig, failedBefore := s.failed[m.ID]
	s.mu.Unlock()
	if failedBefore && failedSig == sig {
		res.Skipped++
		return
	}

	tech, err := s.prober.Probe(ctx, path)
	if err != nil || tech == nil {
		if ctx.Err() != nil {
			return // reported by Run's loop
		}
		res.Failed++
		s.rememberFailure(m.ID, sig)
		s.logger.Warn("movie subtitle tracks backfill: probe failed — stays unknown", "movie_id", m.ID, "file", path, "error", err)
		return
	}

	// Same merge enrichment stores after a successful probe: the probe read
	// the file, so "no tracks" is the fact `[]`, not unknown.
	tracks := MergeSubtitleTracks(tech.SubtitleTracks, s.sidecars(path))
	raw, err := json.Marshal(tracks)
	if err != nil {
		res.Failed++
		s.rememberFailure(m.ID, sig)
		s.logger.Warn("movie subtitle tracks backfill: encoding tracks failed", "movie_id", m.ID, "error", err)
		return
	}
	written, err := s.repo.UpdateSubtitleTracksIfMissing(ctx, m.ID, string(raw))
	if err != nil {
		res.Failed++
		s.logger.Warn("movie subtitle tracks backfill: write failed", "movie_id", m.ID, "error", err)
		return
	}
	if !written {
		res.Skipped++ // a scan stored an answer while we probed — keep it
		return
	}
	res.Filled++
}

func (s *MovieSubtitleTracksBackfillService) rememberFailure(id, sig string) {
	s.mu.Lock()
	s.failed[id] = sig
	s.mu.Unlock()
}

// Trigger starts a pass in the background. A trigger that arrives while a
// pass is running queues exactly one more pass after it, and never runs two
// at once. A trigger after shutdown began (a scan finishing as the app
// stops) is dropped: the database may already be closing.
func (s *MovieSubtitleTracksBackfillService) Trigger(ctx context.Context) {
	s.mu.Lock()
	if ctx.Err() != nil {
		s.mu.Unlock()
		return
	}
	if s.running {
		s.rerun = true
		s.mu.Unlock()
		return
	}
	s.running = true
	s.wg.Add(1)
	s.mu.Unlock()

	go func() {
		defer s.wg.Done()
		for {
			s.Run(ctx)
			s.mu.Lock()
			if !s.rerun || ctx.Err() != nil {
				s.running = false
				s.rerun = false
				s.mu.Unlock()
				return
			}
			s.rerun = false
			s.mu.Unlock()
		}
	}()
}

// RunAfter triggers one pass once the app has settled. Called from main at
// boot; a pass only reads what is still unknown, so every boot may run one.
func (s *MovieSubtitleTracksBackfillService) RunAfter(ctx context.Context, delay time.Duration) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}
		s.Trigger(ctx)
	}()
}

// Wait blocks until a running pass has returned. main calls it after
// cancelling the pass's ctx and before closing the database.
func (s *MovieSubtitleTracksBackfillService) Wait() {
	s.wg.Wait()
}
