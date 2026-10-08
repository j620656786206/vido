package services

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

// disc-2026-10-movie-subtitle-tracks-unknown-refresh.

// main wires the real repository by type assertion; a renamed method would
// silently disable the backfill, so pin it at compile time.
var _ MovieSubtitleTracksBackfillRepo = (*repository.MovieRepository)(nil)

type fakeMovieTracksRepo struct {
	mu     sync.Mutex
	movies []models.Movie
	writes map[string]string
	// taken: ids a concurrent scan filled while the backfill probed.
	taken map[string]bool
}

// FindMissingSubtitleTracks pages like the real query: by id, after afterID.
func (f *fakeMovieTracksRepo) FindMissingSubtitleTracks(_ context.Context, afterID string, limit int) ([]models.Movie, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	sorted := append([]models.Movie(nil), f.movies...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })
	var out []models.Movie
	for _, m := range sorted {
		if !m.SubtitleTracks.Valid && m.ID > afterID && len(out) < limit {
			out = append(out, m)
		}
	}
	return out, nil
}

func (f *fakeMovieTracksRepo) UpdateSubtitleTracksIfMissing(_ context.Context, id, tracks string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.taken[id] {
		return false, nil
	}
	if f.writes == nil {
		f.writes = map[string]string{}
	}
	f.writes[id] = tracks
	for i := range f.movies {
		if f.movies[i].ID == id {
			f.movies[i].SubtitleTracks = models.NewNullString(tracks)
		}
	}
	return true, nil
}

func (f *fakeMovieTracksRepo) written(id string) (string, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	v, ok := f.writes[id]
	return v, ok
}

// movieFile creates a real file (the backfill stats it) and a movie row for it.
func movieFile(t *testing.T, dir, id string) models.Movie {
	t.Helper()
	p := filepath.Join(dir, id+".mkv")
	require.NoError(t, os.WriteFile(p, []byte("video"), 0o644))
	return models.Movie{ID: id, Title: id, FilePath: models.NewNullString(p)}
}

func newMovieBackfill(repo *fakeMovieTracksRepo, prober *fakeProber, sidecars map[string][]SubtitleTrack) *MovieSubtitleTracksBackfillService {
	s := NewMovieSubtitleTracksBackfillService(repo, prober, nil)
	s.sidecars = func(p string) []SubtitleTrack { return sidecars[p] }
	return s
}

func decodeTracks(t *testing.T, raw string) []SubtitleTrack {
	t.Helper()
	var out []SubtitleTrack
	require.NoError(t, json.Unmarshal([]byte(raw), &out))
	return out
}

func TestMovieTracksBackfill_FillsUnknownMoviesWithTheScansMerge(t *testing.T) {
	dir := t.TempDir()
	withChi := movieFile(t, dir, "with-chi")
	nothing := movieFile(t, dir, "nothing")
	sidecarOnly := movieFile(t, dir, "sidecar-only")
	repo := &fakeMovieTracksRepo{movies: []models.Movie{withChi, nothing, sidecarOnly}}
	prober := &fakeProber{available: true, info: map[string]*MediaTechInfo{
		withChi.FilePath.String: {SubtitleTracks: []SubtitleTrack{{Language: "chi", Format: "subrip", Title: "繁體中文", StreamIndex: 3}}},
	}}
	zht := SubtitleTrack{Language: "zh-Hant", Format: "srt", External: true}
	s := newMovieBackfill(repo, prober, map[string][]SubtitleTrack{sidecarOnly.FilePath.String: {zht}})

	res := s.Run(context.Background())

	assert.Equal(t, MovieTracksBackfillResult{Filled: 3, Total: 3}, res)
	raw, _ := repo.written("with-chi")
	assert.Equal(t, []SubtitleTrack{{Language: "chi", Format: "subrip", Title: "繁體中文", StreamIndex: 3}}, decodeTracks(t, raw))
	raw, _ = repo.written("nothing")
	assert.Equal(t, "[]", raw, "a successful probe with no subtitles is the fact 「缺中文」, not unknown")
	raw, _ = repo.written("sidecar-only")
	assert.Equal(t, []SubtitleTrack{zht}, decodeTracks(t, raw))

	// Filled rows leave the list: the next pass probes nothing.
	again := s.Run(context.Background())
	assert.Equal(t, MovieTracksBackfillResult{}, again)
	assert.Equal(t, 3, prober.callCount())
}

func TestMovieTracksBackfill_MissingFileStaysUnknownAndIsTriedAgain(t *testing.T) {
	dir := t.TempDir()
	gone := models.Movie{ID: "gone", Title: "gone", FilePath: models.NewNullString(filepath.Join(dir, "not-mounted.mkv"))}
	repo := &fakeMovieTracksRepo{movies: []models.Movie{gone}}
	prober := &fakeProber{available: true}
	s := newMovieBackfill(repo, prober, nil)

	res := s.Run(context.Background())
	assert.Equal(t, MovieTracksBackfillResult{Missing: 1, Total: 1}, res)
	_, wrote := repo.written("gone")
	assert.False(t, wrote)
	assert.Zero(t, prober.callCount(), "no file, no probe")

	// The share comes up: the next pass reads it.
	require.NoError(t, os.WriteFile(gone.FilePath.String, []byte("video"), 0o644))
	res = s.Run(context.Background())
	assert.Equal(t, 1, res.Filled)
}

func TestMovieTracksBackfill_FailedProbeStaysUnknownUntilTheFileChanges(t *testing.T) {
	dir := t.TempDir()
	broken := movieFile(t, dir, "broken")
	repo := &fakeMovieTracksRepo{movies: []models.Movie{broken}}
	prober := &fakeProber{available: true, fail: map[string]bool{broken.FilePath.String: true}}
	s := newMovieBackfill(repo, prober, nil)

	res := s.Run(context.Background())
	assert.Equal(t, MovieTracksBackfillResult{Failed: 1, Total: 1}, res)
	_, wrote := repo.written("broken")
	assert.False(t, wrote, "a failed probe must not turn unknown into 「缺中文」")

	res = s.Run(context.Background())
	assert.Equal(t, MovieTracksBackfillResult{Skipped: 1, Total: 1}, res, "the same unreadable file is not retried this process")
	assert.Equal(t, 1, prober.callCount())

	// The user replaces the file: it is tried again.
	require.NoError(t, os.WriteFile(broken.FilePath.String, []byte("a re-ripped, longer video"), 0o644))
	prober.mu.Lock()
	prober.fail = nil
	prober.mu.Unlock()
	res = s.Run(context.Background())
	assert.Equal(t, 1, res.Filled)
}

func TestMovieTracksBackfill_NoProberLeavesEverythingUnknown(t *testing.T) {
	dir := t.TempDir()
	m := movieFile(t, dir, "m")
	repo := &fakeMovieTracksRepo{movies: []models.Movie{m}}
	s := newMovieBackfill(repo, &fakeProber{available: false},
		map[string][]SubtitleTrack{m.FilePath.String: {{Language: "eng", Format: "srt", External: true}}})

	assert.Equal(t, MovieTracksBackfillResult{}, s.Run(context.Background()))
	_, wrote := repo.written("m")
	assert.False(t, wrote, "sidecars alone cannot say an embedded Chinese track is absent")
}

func TestMovieTracksBackfill_KeepsWhatAScanStoredMeanwhile(t *testing.T) {
	dir := t.TempDir()
	m := movieFile(t, dir, "raced")
	repo := &fakeMovieTracksRepo{movies: []models.Movie{m}, taken: map[string]bool{"raced": true}}
	s := newMovieBackfill(repo, &fakeProber{available: true}, nil)

	assert.Equal(t, MovieTracksBackfillResult{Skipped: 1, Total: 1}, s.Run(context.Background()))
}

func TestMovieTracksBackfill_TriggerQueuesOneRerunAndWaitReturns(t *testing.T) {
	dir := t.TempDir()
	first := movieFile(t, dir, "first")
	repo := &fakeMovieTracksRepo{movies: []models.Movie{first}}
	prober := &fakeProber{available: true, block: make(chan struct{})}
	s := newMovieBackfill(repo, prober, nil)
	ctx := context.Background()

	s.Trigger(ctx) // pass 1 blocks inside the probe
	// A scan finishes mid-pass and adds a movie: the triggers coalesce into
	// one more pass.
	repo.mu.Lock()
	repo.movies = append(repo.movies, movieFile(t, dir, "added-mid-pass"))
	repo.mu.Unlock()
	s.Trigger(ctx)
	s.Trigger(ctx)
	close(prober.block)
	s.Wait()

	for _, id := range []string{"first", "added-mid-pass"} {
		_, wrote := repo.written(id)
		assert.True(t, wrote, id)
	}
	assert.Equal(t, 2, prober.callCount(), "two passes, each probing only what was still unknown")
}

func TestMovieTracksBackfill_RunAfterStopsOnShutdown(t *testing.T) {
	dir := t.TempDir()
	repo := &fakeMovieTracksRepo{movies: []models.Movie{movieFile(t, dir, "m")}}
	prober := &fakeProber{available: true}
	s := newMovieBackfill(repo, prober, nil)
	ctx, cancel := context.WithCancel(context.Background())

	s.RunAfter(ctx, time.Hour)
	cancel()
	s.Wait()
	assert.Zero(t, prober.callCount())
}

// Rows that stay unknown (files on an unmounted share) must not hide the rows
// behind them: a pass walks every page.
func TestMovieTracksBackfill_StuckRowsDoNotStarveTheRest(t *testing.T) {
	old := movieTracksBackfillPage
	movieTracksBackfillPage = 2
	t.Cleanup(func() { movieTracksBackfillPage = old })

	dir := t.TempDir()
	var movies []models.Movie
	for _, id := range []string{"a-gone", "b-gone", "c-gone"} {
		movies = append(movies, models.Movie{ID: id, Title: id, FilePath: models.NewNullString(filepath.Join(dir, id+".mkv"))})
	}
	movies = append(movies, movieFile(t, dir, "d-real"))
	repo := &fakeMovieTracksRepo{movies: movies}
	s := newMovieBackfill(repo, &fakeProber{available: true}, nil)

	res := s.Run(context.Background())
	assert.Equal(t, MovieTracksBackfillResult{Filled: 1, Missing: 3, Total: 4}, res)
	_, wrote := repo.written("d-real")
	assert.True(t, wrote)
}

// A probe cut short by shutdown is not a broken file: it is not remembered,
// so the next start probes it again.
func TestMovieTracksBackfill_CancelledProbeIsNotRememberedAsAFailure(t *testing.T) {
	dir := t.TempDir()
	m := movieFile(t, dir, "m")
	repo := &fakeMovieTracksRepo{movies: []models.Movie{m}}
	// The probe blocks until the deadline cuts it short — shutdown mid-probe.
	prober := &fakeProber{available: true, block: make(chan struct{})}
	s := newMovieBackfill(repo, prober, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	res := s.Run(ctx)
	assert.Equal(t, 1, res.Total, "the pass reached the probe")
	assert.Zero(t, res.Failed)
	s.mu.Lock()
	_, remembered := s.failed["m"]
	s.mu.Unlock()
	assert.False(t, remembered)
}

func TestMovieTracksBackfill_TriggerAfterShutdownDoesNothing(t *testing.T) {
	dir := t.TempDir()
	repo := &fakeMovieTracksRepo{movies: []models.Movie{movieFile(t, dir, "m")}}
	prober := &fakeProber{available: true}
	s := newMovieBackfill(repo, prober, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	s.Trigger(ctx)
	s.Wait()
	assert.Zero(t, prober.callCount())
}
