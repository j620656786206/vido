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

// Implements: disc-2026-10-movie-subtitle-tracks-unknown-refresh AC #1–#7

// MovieSubtitleTracksBackfillRepo is the slice of the movie repository the
// backfill needs. *repository.MovieRepository satisfies it.
type MovieSubtitleTracksBackfillRepo interface {
	FindMissingSubtitleTracks(ctx context.Context, afterID string, limit int) ([]models.Movie, error)
	UpdateSubtitleTracksIfMissing(ctx context.Context, id, tracksJSON string) (bool, error)
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

// MovieTracksBackfillResult is what one pass did, for the log line and tests.
type MovieTracksBackfillResult struct {
	Filled  int // movies that now have subtitle tracks (possibly `[]`)
	Failed  int // probe or write errors
	Skipped int // probe failed before on the same file, or a scan filled it first
	Missing int // file not there — share not mounted yet, or deleted
	Total   int // movies listed as unknown
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
	if res.Total == 0 {
		return res
	}
	s.logger.Info("movie subtitle tracks backfill: pass finished",
		"filled", res.Filled, "failed", res.Failed, "skipped", res.Skipped, "missing_files", res.Missing,
		"took", time.Since(started).Round(time.Second).String())
	return res
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
