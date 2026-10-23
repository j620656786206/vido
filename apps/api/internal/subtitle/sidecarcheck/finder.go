package sidecarcheck

import (
	"context"
	"errors"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/vido/api/internal/services"
	"github.com/vido/api/internal/subtitle"
)

// Finder picks the English subtitle beside a video that matches its timing
// (disc-2026-10-generate-keeps-user-en-srt-b2). It is both the router's
// subtitle.SidecarFinder (the run checks, and waits) and the estimate's
// services.SidecarAlignmentSource (the quote reads stored checks and starts a
// missing one in the background — it never waits for ffmpeg).
//
// One check of a film runs at a time however many callers ask (singleflight:
// a click while the dialog's background check runs waits for that check), and
// one film is decoded at a time across the whole server (slot): opening five
// dialogs in a row must not start five remux decodes on the NAS.
type Finder struct {
	checker *Checker
	audio   AudioLister
	logger  *slog.Logger

	flights singleflight.Group
	slot    chan struct{}

	mu       sync.Mutex
	inflight map[string]bool          // media path → a background check is running
	failed   map[string]time.Time     // media path → when its check could not run
	recent   map[string]recentVerdict // media path → the last run's verdicts
}

// recentVerdict keeps a run's verdicts in memory, so a cache that stored
// nothing (no cache wired, a write that failed) cannot turn every quote into
// a new decode and the dialog's "checking" into a poll that never ends.
type recentVerdict struct {
	at       time.Time
	verdicts map[string]Verdict // candidate path → verdict
}

// AudioLister lists a file's audio streams; *services.AudioExtractorService
// satisfies it.
type AudioLister interface {
	ListAudioTracks(ctx context.Context, filePath string) ([]services.AudioTrack, error)
}

// Compile-time proof of both ports.
var (
	_ subtitle.SidecarFinder          = (*Finder)(nil)
	_ services.SidecarAlignmentSource = (*Finder)(nil)
)

const (
	// backgroundTimeout bounds one background check of all of a film's
	// sidecars (a cold 99 GB remux took 37 s for one), waiting for the slot
	// included.
	backgroundTimeout = 10 * time.Minute
	// failedFor is how long a film whose check could not run reports
	// "unknown" before the next quote tries again.
	failedFor = 10 * time.Minute
	// recentFor is how long a run's verdicts are served from memory.
	recentFor = 30 * time.Minute
)

// NewFinder wires a finder.
func NewFinder(checker *Checker, audio AudioLister, logger *slog.Logger) *Finder {
	if logger == nil {
		logger = slog.Default()
	}
	return &Finder{
		checker:  checker,
		audio:    audio,
		logger:   logger.With("component", "sidecar_finder"),
		slot:     make(chan struct{}, 1),
		inflight: map[string]bool{},
		failed:   map[string]time.Time{},
		recent:   map[string]recentVerdict{},
	}
}

// candidates are the English .srt files beside mediaPath, in the order the
// route tries them: the user's plain <base>.en.srt / .eng.srt, then their
// other English files (.en.sdh.srt, .en.forced.srt…) by name, then Vido's own
// transcript <base>.vido.en.srt — it was heard from this film, but the film
// may have been replaced since, so it is checked like any other.
func candidates(mediaPath string) []string {
	files, err := subtitle.ListSidecars(mediaPath, "")
	if err != nil {
		return nil
	}
	dir := filepath.Dir(mediaPath)
	stem := strings.TrimSuffix(filepath.Base(mediaPath), filepath.Ext(mediaPath))
	var plain, other, vido []string
	for _, f := range files { // ListSidecars sorts by name
		if f.Language != "en" || f.Format != "srt" {
			continue
		}
		p := filepath.Join(dir, f.FileName)
		tag := strings.ToLower(strings.TrimSuffix(strings.TrimPrefix(f.FileName, stem+"."), ".srt"))
		switch {
		case strings.HasPrefix(tag, services.VidoTranscriptTag+"."):
			vido = append(vido, p)
		case tag == "en" || tag == "eng":
			plain = append(plain, p)
		default:
			other = append(other, p)
		}
	}
	return append(append(plain, other...), vido...)
}

// FindAligned checks the candidates in order and returns the first that
// matches. It waits for ffmpeg: the router calls it from inside a run.
func (f *Finder) FindAligned(ctx context.Context, mediaPath string) (string, bool) {
	r := f.run(ctx, mediaPath)
	return r.path, r.ok
}

// findResult is one run over a film's candidates.
type findResult struct {
	path     string
	ok       bool
	verdicts map[string]Verdict
	err      error // the audio stream could not be chosen, or a check errored
}

// run is find, once per film at a time and one film at a time.
func (f *Finder) run(ctx context.Context, mediaPath string) findResult {
	v, _, _ := f.flights.Do(mediaPath, func() (any, error) {
		select {
		case f.slot <- struct{}{}:
			defer func() { <-f.slot }()
		case <-ctx.Done():
			r := findResult{err: ctx.Err()}
			f.remember(mediaPath, r)
			return r, nil
		}
		r := f.find(ctx, mediaPath)
		f.remember(mediaPath, r)
		return r, nil
	})
	return v.(findResult)
}

// find checks the candidates in order and stops at the first aligned one (an
// aligned one found before a failing one still wins).
func (f *Finder) find(ctx context.Context, mediaPath string) findResult {
	r := findResult{verdicts: map[string]Verdict{}}
	cands := candidates(mediaPath)
	if len(cands) == 0 {
		return r
	}
	tracks, err := f.audio.ListAudioTracks(ctx, mediaPath)
	if err != nil {
		f.logger.Warn("sidecar check skipped: audio tracks unreadable", "media", mediaPath, "error", err)
		r.err = err
		return r
	}
	track, err := services.SelectEnglishTrack(tracks)
	if err != nil {
		f.logger.Warn("sidecar check skipped: no audio track", "media", mediaPath, "error", err)
		r.err = err
		return r
	}
	var errs []error
	for _, c := range cands {
		v, err := f.checker.Check(ctx, mediaPath, track.Index, c)
		if err != nil {
			f.logger.Warn("sidecar check failed", "media", mediaPath, "sidecar", c, "error", err)
			errs = append(errs, err)
			continue
		}
		r.verdicts[c] = v
		f.logger.Info("sidecar check", "media", mediaPath, "sidecar", filepath.Base(c),
			"result", v.Result, "offset_seconds", v.OffsetSeconds, "reason", v.Reason, "cached", v.Cached)
		if v.Result == Aligned {
			r.path, r.ok = c, true
			return r
		}
	}
	r.err = errors.Join(errs...)
	return r
}

// remember records a run: its verdicts for Peek, and whether it failed.
// Expired entries are dropped here, so the maps stay as small as the set of
// films looked at in the last half hour.
func (f *Finder) remember(mediaPath string, r findResult) {
	f.mu.Lock()
	defer f.mu.Unlock()
	now := time.Now()
	for m, e := range f.recent {
		if now.Sub(e.at) > recentFor {
			delete(f.recent, m)
		}
	}
	for m, at := range f.failed {
		if now.Sub(at) > failedFor {
			delete(f.failed, m)
		}
	}
	f.recent[mediaPath] = recentVerdict{at: now, verdicts: r.verdicts}
	if r.err != nil {
		f.failed[mediaPath] = now
	} else {
		delete(f.failed, mediaPath)
	}
}

// verdict is the stored verdict for a candidate, else the last run's.
func (f *Finder) verdict(ctx context.Context, mediaPath, cand string) (Verdict, bool) {
	if v, ok := f.checker.Lookup(ctx, mediaPath, cand); ok {
		return v, true
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	e, ok := f.recent[mediaPath]
	if !ok || time.Since(e.at) > recentFor {
		return Verdict{}, false
	}
	v, ok := e.verdicts[cand]
	return v, ok
}

// Peek reports what the stored checks say, in the order the route tries the
// candidates: the first aligned one wins; a candidate not yet checked before
// it means "checking" (and starts the check); otherwise the first
// candidate's verdict. A candidate whose check could not run lately is
// passed over as "unknown" rather than restarted on every quote.
func (f *Finder) Peek(ctx context.Context, mediaPath string) services.SidecarAlignment {
	cands := candidates(mediaPath)
	if len(cands) == 0 {
		return services.SidecarAlignment{Status: services.SidecarStatusNone}
	}
	failed := f.recentlyFailed(mediaPath)
	var first *services.SidecarAlignment
	for _, c := range cands {
		v, ok := f.verdict(ctx, mediaPath, c)
		if !ok {
			if failed {
				if first == nil {
					first = &services.SidecarAlignment{Status: services.SidecarStatusUnknown, FileName: filepath.Base(c)}
				}
				continue
			}
			return f.startBackground(mediaPath, filepath.Base(c))
		}
		sa := services.SidecarAlignment{Status: string(v.Result), OffsetSeconds: v.OffsetSeconds, FileName: filepath.Base(c)}
		if v.Result == Aligned {
			return sa
		}
		if first == nil {
			first = &sa
		}
	}
	return *first
}

// MayRoute answers the click's gate: might the router translate a sidecar of
// mediaPath instead of running speech recognition? It reads the stored checks
// like Peek (and, like Peek, starts a missing one in the background) and says
// no only when that answer is final — nothing beside the file, or every
// candidate checked and rejected. A check still running or one that could not
// run says yes: the run's FindAligned joins or retries it, and a gate that
// cannot judge must not refuse (disc-2026-10-solo-gate-en-sidecar).
func (f *Finder) MayRoute(ctx context.Context, mediaPath string) bool {
	return SidecarMayRoute(f.Peek(ctx, mediaPath).Status)
}

// SidecarMayRoute is MayRoute's rule over one Peek status.
func SidecarMayRoute(status string) bool {
	switch status {
	case services.SidecarStatusNone, services.SidecarStatusOffset,
		services.SidecarStatusMismatch, services.SidecarStatusUnusable:
		return false
	default: // aligned, checking, unknown
		return true
	}
}

func (f *Finder) recentlyFailed(mediaPath string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	at, ok := f.failed[mediaPath]
	return ok && time.Since(at) < failedFor
}

// startBackground runs the check on its own context — the estimate request
// that asked returns at once. One per film at a time.
func (f *Finder) startBackground(mediaPath, fileName string) services.SidecarAlignment {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.inflight[mediaPath] {
		f.inflight[mediaPath] = true
		go func() {
			defer func() {
				if p := recover(); p != nil {
					f.logger.Error("sidecar background check panicked", "media", mediaPath, "panic", p)
					f.remember(mediaPath, findResult{err: errors.New("panic")})
				}
				f.mu.Lock()
				delete(f.inflight, mediaPath)
				f.mu.Unlock()
			}()
			ctx, cancel := context.WithTimeout(context.Background(), backgroundTimeout)
			defer cancel()
			f.run(ctx, mediaPath)
		}()
	}
	return services.SidecarAlignment{Status: services.SidecarStatusChecking, FileName: fileName}
}
