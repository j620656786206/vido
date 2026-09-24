package services

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakePosterRefs struct {
	paths []string
	err   error
}

func (f fakePosterRefs) ListUploadedPosterPaths(context.Context) ([]string, error) {
	return f.paths, f.err
}

var sweepNow = time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)

// touch writes a file whose mtime is `age` before sweepNow.
func touch(t *testing.T, dir, name string, age time.Duration) {
	t.Helper()
	p := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(p, []byte("x"), 0o644))
	mt := sweepNow.Add(-age)
	require.NoError(t, os.Chtimes(p, mt, mt))
}

func remaining(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func newSweeper(dir string, refs fakePosterRefs) *PosterOrphanSweeper {
	s := NewPosterOrphanSweeper(dir, refs)
	s.now = func() time.Time { return sweepNow }
	return s
}

const old = 2 * time.Hour

func TestPosterOrphanSweeper_RemovesOnlyUnreferencedOldPosters(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{
		"kept.jpg", "kept-thumb.jpg", // referenced with ?v=
		"legacy.jpg", "legacy-thumb.jpg", // referenced without ?v=
		"gone.jpg", "gone-thumb.jpg", // a deleted film's
	} {
		touch(t, dir, n, old)
	}
	n, err := newSweeper(dir, fakePosterRefs{paths: []string{"/posters/kept.jpg?v=17", "http://nas:8080/api/v1/posters/legacy.jpg"}}).
		Sweep(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(2), n)
	assert.ElementsMatch(t, []string{"kept.jpg", "kept-thumb.jpg", "legacy.jpg", "legacy-thumb.jpg"}, remaining(t, dir))
}

func TestPosterOrphanSweeper_RecentFilesAreLeftAlone(t *testing.T) {
	// An upload writes the file BEFORE the row points at it; the grace window
	// covers that moment (and a .bak parked mid-upload).
	dir := t.TempDir()
	touch(t, dir, "uploading.jpg", 2*time.Minute)
	touch(t, dir, "uploading-thumb.jpg", 2*time.Minute)
	touch(t, dir, "parked.jpg.bak", 2*time.Minute)
	touch(t, dir, "crashed.jpg.bak", old)
	n, err := newSweeper(dir, fakePosterRefs{}).Sweep(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)
	assert.ElementsMatch(t, []string{"uploading.jpg", "uploading-thumb.jpg", "parked.jpg.bak"}, remaining(t, dir))
}

func TestPosterOrphanSweeper_AlwaysSweepsOldBakEvenWhenItsPosterIsReferenced(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "m1.jpg", old)
	touch(t, dir, "m1.jpg.bak", old)
	n, err := newSweeper(dir, fakePosterRefs{paths: []string{"/posters/m1.jpg"}}).Sweep(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)
	assert.ElementsMatch(t, []string{"m1.jpg"}, remaining(t, dir))
}

func TestPosterOrphanSweeper_NeverTouchesAnythingThatIsNotAPoster(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"notes.txt", "x.png", "a.b.jpg", "x.JPG"} {
		touch(t, dir, n, old)
	}
	require.NoError(t, os.Mkdir(filepath.Join(dir, "sub.jpg"), 0o755))
	outside := filepath.Join(t.TempDir(), "outside.jpg")
	require.NoError(t, os.WriteFile(outside, []byte("x"), 0o644))
	require.NoError(t, os.Symlink(outside, filepath.Join(dir, "link.jpg")))

	n, err := newSweeper(dir, fakePosterRefs{}).Sweep(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(0), n)
	assert.ElementsMatch(t, []string{"notes.txt", "x.png", "a.b.jpg", "x.JPG", "sub.jpg", "link.jpg"}, remaining(t, dir))
	_, err = os.Stat(outside)
	assert.NoError(t, err, "the link's target must survive")
}

func TestPosterOrphanSweeper_QueryFailureDeletesNothing(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "m1.jpg", old)
	_, err := newSweeper(dir, fakePosterRefs{err: errors.New("database is locked")}).Sweep(context.Background())
	require.Error(t, err)
	assert.ElementsMatch(t, []string{"m1.jpg"}, remaining(t, dir))
}

func TestPosterOrphanSweeper_MissingFolderIsNotAnError(t *testing.T) {
	n, err := newSweeper(filepath.Join(t.TempDir(), "posters"), fakePosterRefs{}).Sweep(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(0), n)
}
