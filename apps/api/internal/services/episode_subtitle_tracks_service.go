// Package services — episode_subtitle_tracks_service.go
//
// disc-2026-10-episode-list-subtitle-badge-a: remember which subtitles each
// episode has, so the season list can say "has Chinese / missing Chinese"
// without running ffprobe while the user waits.
//
// Episodes never had a tech-info pass (enrichment is movies-only), so their
// embedded subtitle tracks were read only by the candidate sweep's route probe
// — and thrown away. This service is that pass for subtitles: after every
// library scan, and once a little after boot, it walks the episodes whose file
// it has not read yet (or whose file changed since), probes each one, lists
// the sidecar files beside it, and stores both halves in
// episodes.subtitle_tracks.
//
// Progress lives in the database (subtitle_tracks_file_sig), so a pass that is
// cut short by a restart simply resumes from the episodes it had not reached.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/vido/api/internal/models"
)

// Implements: disc-2026-10-episode-list-subtitle-badge-a AC #5

// EpisodeSubtitleTracksRepo is the slice of the episode repository the sweep
// needs. *repository.EpisodeRepository satisfies it.
type EpisodeSubtitleTracksRepo interface {
	FindWithFiles(ctx context.Context) ([]models.Episode, error)
	UpdateSubtitleTracks(ctx context.Context, episodeID, tracksJSON, fileSig string) error
	UpdateDurationSeconds(ctx context.Context, episodeID string, seconds int64) error
}

// EpisodeTrackProber is the expensive half: *FFprobeService.
type EpisodeTrackProber interface {
	IsAvailable() bool
	Probe(ctx context.Context, filePath string) (*MediaTechInfo, error)
}

// TrackScriptPeeker reads a sample of an embedded Chinese text track to tell
// Traditional from Simplified when its tag and title do not say. Implemented
// by subtitle.ScriptPeeker (disc-2026-10-episode-list-subtitle-badge-c).
type TrackScriptPeeker interface {
	IsAvailable() bool
	Peekable(t SubtitleTrack) bool
	PeekScript(ctx context.Context, mediaPath string, streamIndex int) (string, error)
}

// episodeTracksProgressEvery is how often (in probed episodes) a pass logs.
const episodeTracksProgressEvery = 25

// EpisodeSubtitleTracksService is the background subtitle-track sweep.
type EpisodeSubtitleTracksService struct {
	repo     EpisodeSubtitleTracksRepo
	prober   EpisodeTrackProber
	sidecars SidecarTrackReader
	peeker   TrackScriptPeeker
	logger   *slog.Logger
	stat     func(string) (os.FileInfo, error)

	mu      sync.Mutex
	wg      sync.WaitGroup
	running bool
	rerun   bool
	// failed remembers episodes whose probe failed, keyed by id, valued by the
	// file signature that failed: the same unreadable file is not retried in
	// this process, but a replaced one is.
	failed map[string]string
}

// NewEpisodeSubtitleTracksService wires the sweep. A nil repo or prober makes
// every pass a no-op; a nil sidecar reader stores the embedded half only.
func NewEpisodeSubtitleTracksService(repo EpisodeSubtitleTracksRepo, prober EpisodeTrackProber, sidecars SidecarTrackReader, logger *slog.Logger) *EpisodeSubtitleTracksService {
	if logger == nil {
		logger = slog.Default()
	}
	return &EpisodeSubtitleTracksService{
		repo:     repo,
		prober:   prober,
		sidecars: sidecars,
		logger:   logger.With("service", "episode_subtitle_tracks"),
		stat:     os.Stat,
		failed:   map[string]string{},
	}
}

// SetScriptPeeker turns on reading a sample of untold-script Chinese tracks
// (disc-2026-10-episode-list-subtitle-badge-c). nil or unavailable = off;
// those tracks then stay "Chinese, script untold".
func (s *EpisodeSubtitleTracksService) SetScriptPeeker(p TrackScriptPeeker) {
	s.peeker = p
}

// EpisodeTracksSweepResult is what one pass did, for the log line and tests.
type EpisodeTracksSweepResult struct {
	Probed  int // episodes whose tracks were read and stored
	Failed  int // probe or write errors (not retried this process)
	Skipped int // already read from the same file (or failed before, unchanged)
	Missing int // file not there — share not mounted yet, or deleted
	Total   int // episodes with a file path that the pass looked at
	// Peeked / PeekFailed count untold-script Chinese tracks whose text was
	// sampled, and samples ffmpeg could not take (the track is stored
	// without a detected script).
	Peeked     int
	PeekFailed int
}

// FileSignature is the "<size>:<mtime unix nanos>" an episode's stored tracks
// were read from; a file whose signature differs has been replaced.
func FileSignature(info os.FileInfo) string {
	return fmt.Sprintf("%d:%d", info.Size(), info.ModTime().UnixNano())
}

// Run does one pass. Fail-soft per episode; the pass never returns an error.
func (s *EpisodeSubtitleTracksService) Run(ctx context.Context) EpisodeTracksSweepResult {
	var res EpisodeTracksSweepResult
	if s.repo == nil || s.prober == nil || !s.prober.IsAvailable() {
		return res
	}

	episodes, err := s.repo.FindWithFiles(ctx)
	if err != nil {
		s.logger.Warn("episode subtitle tracks: listing episodes failed", "error", err)
		return res
	}

	res.Total = len(episodes)
	var work []episodeTracksTodo
	for _, ep := range episodes {
		if ctx.Err() != nil {
			return res
		}
		info, err := s.stat(ep.FilePath.String)
		if err != nil {
			res.Missing++ // unmounted share or deleted file — nothing to read
			continue
		}
		sig := FileSignature(info)
		if ep.SubtitleTracks.Valid && ep.SubtitleTracksFileSig.String == sig && !s.needsPeek(ep) {
			res.Skipped++
			continue
		}
		s.mu.Lock()
		failedSig, failedBefore := s.failed[ep.ID]
		s.mu.Unlock()
		if failedBefore && failedSig == sig {
			res.Skipped++
			continue
		}
		work = append(work, episodeTracksTodo{ep: ep, sig: sig})
	}
	if len(work) == 0 {
		if res.Missing > 0 {
			s.logger.Info("episode subtitle tracks: nothing to read", "missing_files", res.Missing, "already_read", res.Skipped)
		}
		return res
	}

	started := time.Now()
	s.logger.Info("episode subtitle tracks: pass started", "pending", len(work), "already_read", res.Skipped, "missing_files", res.Missing)

	// Episodes arrive in broadcast order, so a season's files are adjacent:
	// read each folder's sidecars once for the run of episodes inside it.
	var dirSidecars map[string]SidecarTracks
	currentDir := ""
	for i, w := range work {
		if ctx.Err() != nil {
			s.logger.Info("episode subtitle tracks: pass interrupted — resumes next pass",
				"probed", res.Probed, "remaining", len(work)-i)
			return res
		}
		path := w.ep.FilePath.String
		if dir := filepath.Dir(path); dir != currentDir || dirSidecars == nil {
			currentDir = dir
			dirSidecars = s.readSidecars(work[i:], dir)
		}

		info, err := s.prober.Probe(ctx, path)
		if err != nil || info == nil {
			if ctx.Err() != nil {
				continue // reported at the top of the next iteration
			}
			res.Failed++
			s.rememberFailure(w.ep.ID, w.sig)
			s.logger.Warn("episode subtitle tracks: probe failed", "episode_id", w.ep.ID, "file", path, "error", err)
			continue
		}

		tracks := make([]SubtitleTrack, 0, len(info.SubtitleTracks))
		for _, t := range info.SubtitleTracks {
			if !t.External {
				tracks = append(tracks, t)
			}
		}
		peekFailedBefore := res.PeekFailed
		if interrupted := s.peekScripts(ctx, path, tracks, &res); interrupted {
			continue // a cut-short sample must not store a half-read episode
		}
		if side, ok := dirSidecars[path]; ok && side.Err == nil {
			tracks = append(tracks, side.Tracks...)
		}
		raw, err := json.Marshal(tracks)
		if err != nil {
			res.Failed++
			s.rememberFailure(w.ep.ID, w.sig)
			continue
		}
		if err := s.repo.UpdateSubtitleTracks(ctx, w.ep.ID, string(raw), w.sig); err != nil {
			res.Failed++
			s.logger.Warn("episode subtitle tracks: write failed", "episode_id", w.ep.ID, "error", err)
			continue
		}
		if res.PeekFailed > peekFailedBefore {
			// Stored without that track's script; needsPeek would queue the
			// episode again every pass — try once per process, like a probe.
			s.rememberFailure(w.ep.ID, w.sig)
		}
		// The probe measured the length too; an episode with no stored length
		// is priced at the 45-minute assumption, so keep it
		// (backlog-episode-duration-warm-cache-gap option a).
		if !w.ep.DurationSeconds.Valid && info.DurationSeconds > 0 {
			if err := s.repo.UpdateDurationSeconds(ctx, w.ep.ID, int64(info.DurationSeconds)); err != nil {
				s.logger.Debug("episode subtitle tracks: duration write failed", "episode_id", w.ep.ID, "error", err)
			}
		}
		res.Probed++
		if res.Probed%episodeTracksProgressEvery == 0 {
			s.logger.Info("episode subtitle tracks: progress", "probed", res.Probed, "pending", len(work))
		}
	}

	s.logger.Info("episode subtitle tracks: pass finished",
		"probed", res.Probed, "failed", res.Failed, "skipped", res.Skipped, "missing_files", res.Missing,
		"scripts_peeked", res.Peeked, "peeks_failed", res.PeekFailed, "took", time.Since(started).Round(time.Second).String())
	return res
}

// peekScripts fills DetectedLanguage on every embedded Chinese text track
// whose tag and title do not tell the script. Fail-soft: a failed sample
// leaves that track as it was. interrupted = ctx ended during a sample.
func (s *EpisodeSubtitleTracksService) peekScripts(ctx context.Context, path string, tracks []SubtitleTrack, res *EpisodeTracksSweepResult) (interrupted bool) {
	if s.peeker == nil || !s.peeker.IsAvailable() {
		return false
	}
	for i := range tracks {
		if !s.peeker.Peekable(tracks[i]) {
			continue
		}
		script, err := s.peeker.PeekScript(ctx, path, tracks[i].StreamIndex)
		if ctx.Err() != nil {
			return true
		}
		if err != nil {
			res.PeekFailed++
			s.logger.Warn("episode subtitle tracks: script sample failed", "file", path, "stream_index", tracks[i].StreamIndex, "error", err)
			continue
		}
		res.Peeked++
		tracks[i].DetectedLanguage = script
	}
	return false
}

// needsPeek reports whether an episode read before the script sampler
// existed still has an untold-script Chinese track never sampled — so a
// library swept by -a gets its samples without waiting for a file to change.
// A sample that did not tell is stored as "zh"/"und", so this is true once.
func (s *EpisodeSubtitleTracksService) needsPeek(ep models.Episode) bool {
	if s.peeker == nil || !s.peeker.IsAvailable() {
		return false
	}
	var stored []SubtitleTrack
	if json.Unmarshal([]byte(ep.SubtitleTracks.String), &stored) != nil {
		return false
	}
	for _, t := range stored {
		if t.DetectedLanguage == "" && s.peeker.Peekable(t) {
			return true
		}
	}
	return false
}

// episodeTracksTodo is one episode a pass will probe, with the signature of
// the file it found.
type episodeTracksTodo struct {
	ep  models.Episode
	sig string
}

// readSidecars reads the sidecars of the leading run of upcoming episodes that
// live in dir, in one directory read.
func (s *EpisodeSubtitleTracksService) readSidecars(upcoming []episodeTracksTodo, dir string) map[string]SidecarTracks {
	if s.sidecars == nil {
		return map[string]SidecarTracks{}
	}
	var paths []string
	known := map[string][]SubtitleTrack{}
	for _, u := range upcoming {
		if filepath.Dir(u.ep.FilePath.String) != dir {
			break
		}
		paths = append(paths, u.ep.FilePath.String)
		known[u.ep.FilePath.String] = storedSidecars(u.ep)
	}
	return s.sidecars.ReadSidecarTracks(paths, known)
}

func (s *EpisodeSubtitleTracksService) rememberFailure(id, sig string) {
	s.mu.Lock()
	s.failed[id] = sig
	s.mu.Unlock()
}

// Trigger starts a pass in the background. A trigger that arrives while a
// pass is running queues exactly one more pass after it — so files a scan
// added mid-pass are still read — and never runs two at once.
func (s *EpisodeSubtitleTracksService) Trigger(ctx context.Context) {
	s.mu.Lock()
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

// Wait blocks until a running pass has returned. main calls it after
// cancelling the pass's ctx and before closing the database.
func (s *EpisodeSubtitleTracksService) Wait() {
	s.wg.Wait()
}

// unmountedRetryDelay / unmountedRetries: when the boot pass finds not one
// episode file (the NAS share is not mounted yet — common right after an
// Unraid boot), try again later instead of waiting for the next scan.
var (
	unmountedRetryDelay = 10 * time.Minute
	unmountedRetries    = 6
)

// RunAfter runs one pass once the app has settled (the scanner and
// enrichment get the first minute). Called from main at boot; the pass is
// idempotent, so every boot may run one — it only reads what it has not.
func (s *EpisodeSubtitleTracksService) RunAfter(ctx context.Context, delay time.Duration) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		wait := delay
		for attempt := 0; attempt <= unmountedRetries; attempt++ {
			select {
			case <-ctx.Done():
				return
			case <-time.After(wait):
			}
			res := s.runExclusive(ctx)
			if res.Total == 0 || res.Missing < res.Total {
				return
			}
			s.logger.Info("episode subtitle tracks: no episode file reachable — will retry", "missing_files", res.Missing, "retry_in", unmountedRetryDelay.String())
			wait = unmountedRetryDelay
		}
	}()
}

// runExclusive runs one pass now unless one is already running (then it
// queues a re-run and reports nothing missing, so RunAfter stops retrying).
func (s *EpisodeSubtitleTracksService) runExclusive(ctx context.Context) EpisodeTracksSweepResult {
	s.mu.Lock()
	if s.running {
		s.rerun = true
		s.mu.Unlock()
		return EpisodeTracksSweepResult{}
	}
	s.running = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		rerun := s.rerun && ctx.Err() == nil
		s.running = false
		s.rerun = false
		s.mu.Unlock()
		if rerun {
			s.Trigger(ctx)
		}
	}()
	return s.Run(ctx)
}
