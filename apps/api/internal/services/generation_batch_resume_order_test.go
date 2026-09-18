package services

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
)

// ─── disc-2026-09-generation-resume-b-asr-chunk-store AC #6 ─────────────────
//
// The ceiling is ONE envelope per batch. A $1 batch over a $3.65 film stops
// at chunk 5 every time and, with the order left to the caller, a user who
// picks different films each time leaves a trail of half-done ones that are
// never finished. A batch therefore finishes what an earlier batch started
// before spending on anything new.

type fakeResumeFinder struct {
	mu       sync.Mutex
	progress map[string]bool
	err      error
	calls    int
}

// The port answers a plain bool: a lookup failure is "no progress" (fail-soft,
// Rule 13), which is why `err` here only ever produces false.
func (f *fakeResumeFinder) HasResumeProgress(_ context.Context, _, mediaID, _ string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.err != nil {
		return false
	}
	return f.progress[mediaID]
}

func startOrder(t *testing.T, finder *fakeResumeFinder) []string {
	t.Helper()
	movies := []string{uuidA, uuidB, uuidC}
	cf := &fakeCandidateFinder{byID: map[string]*models.Movie{}}
	for i, id := range movies {
		m := genMovie(id, "T"+string(rune('A'+i)), "/m/"+id+".mkv")
		cf.byID[id] = &m
	}
	p, _ := newTestGenerationProcessor(t, &fakeGenerationRunner{available: true}, cf, 5)
	if finder != nil {
		p.SetResumeProgressFinder(finder)
	}
	_, items, err := p.Start(context.Background(), "selected", movies, 5, "")
	require.NoError(t, err)
	t.Cleanup(func() { p.Cancel() })
	ids := make([]string, len(items))
	for i, it := range items {
		ids[i] = it.MediaID
	}
	return ids
}

func TestGenerationBatch_FilmsWithProgressRunFirst(t *testing.T) {
	assert.Equal(t, []string{uuidB, uuidA, uuidC},
		startOrder(t, &fakeResumeFinder{progress: map[string]bool{uuidB: true}}))
	assert.Equal(t, []string{uuidC, uuidA, uuidB},
		startOrder(t, &fakeResumeFinder{progress: map[string]bool{uuidC: true}}))
	// Two with progress keep their relative order (stable).
	assert.Equal(t, []string{uuidA, uuidC, uuidB},
		startOrder(t, &fakeResumeFinder{progress: map[string]bool{uuidA: true, uuidC: true}}))
}

func TestGenerationBatch_OrderIsUntouchedWithoutAFinderOrOnError(t *testing.T) {
	assert.Equal(t, []string{uuidA, uuidB, uuidC}, startOrder(t, nil))
	assert.Equal(t, []string{uuidA, uuidB, uuidC},
		startOrder(t, &fakeResumeFinder{err: errors.New("db down")}),
		"a lookup failure costs the ordering, never the batch")
}

func TestGenerationBatch_ResumeLookupIsOnePerFilm(t *testing.T) {
	f := &fakeResumeFinder{progress: map[string]bool{}}
	startOrder(t, f)
	assert.Equal(t, 3, f.calls, "one question per film — no second pass")
}
