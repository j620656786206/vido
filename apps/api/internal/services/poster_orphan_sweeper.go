package services

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vido/api/internal/images"
)

// posterOrphanGrace protects files written in the last few minutes: an upload
// writes <id>.jpg BEFORE the row points at it, and parks the old pair as
// .bak while it runs.
const posterOrphanGrace = 10 * time.Minute

// uploadedPosterLister is the one query the sweep needs
// (repository.PosterReferenceRepository).
type uploadedPosterLister interface {
	ListUploadedPosterPaths(ctx context.Context) ([]string, error)
}

// PosterOrphanSweeper keeps data/posters to the files some row still points at
// (bugfix-poster-orphan-sweep). Seven code paths can delete a film or series
// and none removes its poster; reconciling in ONE place covers all of them —
// and any path added later — plus .bak leftovers from a crashed upload.
//
// It errs towards keeping: a failed query deletes nothing, a name that is not
// a poster is never touched, and anything newer than the grace window stays.
// Uploaded posters have no second copy (backups hold only the database).
type PosterOrphanSweeper struct {
	dir   string
	refs  uploadedPosterLister
	now   func() time.Time
	grace time.Duration
}

// NewPosterOrphanSweeper creates a sweeper over dir.
func NewPosterOrphanSweeper(dir string, refs uploadedPosterLister) *PosterOrphanSweeper {
	return &PosterOrphanSweeper{dir: dir, refs: refs, now: time.Now, grace: posterOrphanGrace}
}

// Sweep removes unreferenced poster files and reports how many. Shaped for
// CacheSweepScheduler (SweepFunc).
func (s *PosterOrphanSweeper) Sweep(ctx context.Context) (int64, error) {
	paths, err := s.refs.ListUploadedPosterPaths(ctx)
	if err != nil {
		return 0, fmt.Errorf("poster orphan sweep: %w", err)
	}
	keep := make(map[string]bool, len(paths)*2)
	for _, p := range paths {
		// The file name follows the LAST "/posters/" — the stored form, or a
		// pasted absolute URL of this app's own poster route.
		i := strings.LastIndex(p, "/posters/")
		if i < 0 {
			continue
		}
		name := p[i+len("/posters/"):]
		if i := strings.IndexByte(name, '?'); i >= 0 {
			name = name[:i]
		}
		if images.IsPosterFileName(name) {
			keep[name] = true
			keep[strings.TrimSuffix(name, ".jpg")+"-thumb.jpg"] = true
		}
	}

	entries, err := os.ReadDir(s.dir)
	if errors.Is(err, fs.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("poster orphan sweep: read %s: %w", s.dir, err)
	}

	cutoff := s.now().Add(-s.grace)
	var removed, kept int64
	for _, e := range entries {
		// Regular files only: never a directory, never follow a symlink.
		if !e.Type().IsRegular() {
			continue
		}
		name := e.Name()
		base, parked := strings.CutSuffix(name, ".bak")
		if !images.IsPosterFileName(base) {
			continue
		}
		if !parked && keep[name] {
			kept++
			continue
		}
		info, err := e.Info()
		if err != nil || info.ModTime().After(cutoff) {
			kept++
			continue
		}
		if err := os.Remove(filepath.Join(s.dir, name)); err != nil {
			slog.Warn("Poster orphan sweep: remove failed", "file", name, "error", err)
			continue
		}
		removed++
	}
	if removed > 0 {
		slog.Info("Poster orphan sweep", "removed", removed, "kept", kept)
	}
	return removed, nil
}
