package services

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

// disc-2026-10-episode-list-subtitle-badge-a AC #5.

// main wires the real repository by type assertion; a renamed method would
// silently disable the sweep and the write-back, so pin it at compile time.
var (
	_ EpisodeSubtitleTracksRepo = (*repository.EpisodeRepository)(nil)
	_ episodeTracksRefresher    = (*repository.EpisodeRepository)(nil)
	_ EpisodeTrackProber        = (*FFprobeService)(nil)
)

type fakeTracksRepo struct {
	mu        sync.Mutex
	episodes  []models.Episode
	writes    map[string][2]string // id → {tracks, sig}
	durations map[string]int64
}

func (f *fakeTracksRepo) FindWithFiles(context.Context) ([]models.Episode, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]models.Episode(nil), f.episodes...), nil
}

func (f *fakeTracksRepo) UpdateSubtitleTracks(_ context.Context, id, tracks, sig string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.writes == nil {
		f.writes = map[string][2]string{}
	}
	f.writes[id] = [2]string{tracks, sig}
	for i := range f.episodes {
		if f.episodes[i].ID == id {
			f.episodes[i].SubtitleTracks = models.NewNullString(tracks)
			f.episodes[i].SubtitleTracksFileSig = models.NewNullString(sig)
		}
	}
	return nil
}

func (f *fakeTracksRepo) UpdateDurationSeconds(_ context.Context, id string, s int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.durations == nil {
		f.durations = map[string]int64{}
	}
	f.durations[id] = s
	return nil
}

type fakeProber struct {
	mu        sync.Mutex
	available bool
	info      map[string]*MediaTechInfo
	fail      map[string]bool
	calls     []string
	block     chan struct{} // when set, Probe waits on it
}

func (p *fakeProber) IsAvailable() bool { return p.available }
func (p *fakeProber) Probe(ctx context.Context, path string) (*MediaTechInfo, error) {
	if p.block != nil {
		select {
		case <-p.block:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	p.mu.Lock()
	p.calls = append(p.calls, path)
	p.mu.Unlock()
	if p.fail[path] {
		return nil, errors.New("ffprobe exited 1")
	}
	if info, ok := p.info[path]; ok {
		return info, nil
	}
	return &MediaTechInfo{}, nil
}
func (p *fakeProber) callCount() int { p.mu.Lock(); defer p.mu.Unlock(); return len(p.calls) }

type fakeSidecars struct {
	calls [][]string
	known []map[string][]SubtitleTrack
	out   map[string]SidecarTracks
}

func (f *fakeSidecars) ReadSidecarTracks(paths []string, known map[string][]SubtitleTrack) map[string]SidecarTracks {
	f.calls = append(f.calls, paths)
	f.known = append(f.known, known)
	res := map[string]SidecarTracks{}
	for _, p := range paths {
		res[p] = f.out[p]
	}
	return res
}

func touchEpisodeFile(t *testing.T, path string) os.FileInfo {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("video"), 0o644))
	info, err := os.Stat(path)
	require.NoError(t, err)
	return info
}

func TestEpisodeSubtitleTracks_ProbesUnreadAndStoresBothHalves(t *testing.T) {
	dir := t.TempDir()
	e1 := filepath.Join(dir, "S1", "See.S01E01.mkv")
	e2 := filepath.Join(dir, "S1", "See.S01E02.mkv")
	touchEpisodeFile(t, e1)
	touchEpisodeFile(t, e2)

	repo := &fakeTracksRepo{episodes: []models.Episode{
		{ID: "e1", FilePath: models.NewNullString(e1)},
		{ID: "e2", FilePath: models.NewNullString(e2), DurationSeconds: models.NewNullInt64(3400)},
		{ID: "gone", FilePath: models.NewNullString(filepath.Join(dir, "missing.mkv"))},
	}}
	prober := &fakeProber{available: true, info: map[string]*MediaTechInfo{
		e1: {SubtitleTracks: []SubtitleTrack{{Language: "eng", Format: "subrip", StreamIndex: 8}}, DurationSeconds: 3355.6},
		e2: {SubtitleTracks: []SubtitleTrack{{Language: "eng", Format: "subrip", StreamIndex: 8}}, DurationSeconds: 3400},
	}}
	side := &fakeSidecars{out: map[string]SidecarTracks{
		e2: {Tracks: []SubtitleTrack{{Language: "zh-Hant", Format: "srt", External: true, FileName: "See.S01E02.zh-TW.srt"}}},
	}}
	svc := NewEpisodeSubtitleTracksService(repo, prober, side, nil)

	res := svc.Run(context.Background())
	assert.Equal(t, EpisodeTracksSweepResult{Probed: 2, Missing: 1, Total: 3}, res)
	require.Len(t, side.calls, 1, "both episodes share a folder — one directory read")
	assert.ElementsMatch(t, []string{e1, e2}, side.calls[0])

	var stored []SubtitleTrack
	require.NoError(t, json.Unmarshal([]byte(repo.writes["e2"][0]), &stored))
	assert.Len(t, stored, 2)
	assert.True(t, stored[1].External)
	assert.Equal(t, models.ChineseSubtitleZhHant, models.ChineseSubtitleVerdict("not_searched", "", repo.writes["e2"][0]))
	assert.Equal(t, models.ChineseSubtitleNone, models.ChineseSubtitleVerdict("not_searched", "", repo.writes["e1"][0]))

	info, _ := os.Stat(e1)
	assert.Equal(t, FileSignature(info), repo.writes["e1"][1])
	assert.Equal(t, map[string]int64{"e1": 3355}, repo.durations, "only the episode with no stored length gets one")

	// Second pass: nothing changed → no probe.
	res = svc.Run(context.Background())
	assert.Equal(t, 2, prober.callCount())
	assert.Equal(t, 2, res.Skipped)
	assert.Equal(t, 1, res.Missing)
}

func TestEpisodeSubtitleTracks_ReprobesAReplacedFile(t *testing.T) {
	dir := t.TempDir()
	e1 := filepath.Join(dir, "X.S01E01.mkv")
	touchEpisodeFile(t, e1)
	repo := &fakeTracksRepo{episodes: []models.Episode{{ID: "e1", FilePath: models.NewNullString(e1),
		SubtitleTracks: models.NewNullString("[]"), SubtitleTracksFileSig: models.NewNullString("1:1")}}}
	prober := &fakeProber{available: true}
	svc := NewEpisodeSubtitleTracksService(repo, prober, nil, nil)

	assert.Equal(t, 1, svc.Run(context.Background()).Probed, "stored signature differs from the file — re-encode, re-read")
	assert.Equal(t, "[]", repo.writes["e1"][0], "a probe that found no tracks is a fact: [] not NULL")
}

func TestEpisodeSubtitleTracks_FailedProbeIsNotRetriedUntilTheFileChanges(t *testing.T) {
	dir := t.TempDir()
	e1 := filepath.Join(dir, "X.S01E01.mkv")
	touchEpisodeFile(t, e1)
	repo := &fakeTracksRepo{episodes: []models.Episode{{ID: "e1", FilePath: models.NewNullString(e1)}}}
	prober := &fakeProber{available: true, fail: map[string]bool{e1: true}}
	svc := NewEpisodeSubtitleTracksService(repo, prober, nil, nil)

	assert.Equal(t, 1, svc.Run(context.Background()).Failed)
	assert.Equal(t, 1, svc.Run(context.Background()).Skipped)
	assert.Equal(t, 1, prober.callCount())
	assert.Empty(t, repo.writes, "a failed probe stores nothing — the episode stays 'not read'")

	// The file is replaced → a new signature → tried again.
	future := time.Now().Add(time.Hour)
	require.NoError(t, os.Chtimes(e1, future, future))
	prober.fail = nil
	assert.Equal(t, 1, svc.Run(context.Background()).Probed)
}

func TestEpisodeSubtitleTracks_NoFFprobeIsANoOp(t *testing.T) {
	dir := t.TempDir()
	e1 := filepath.Join(dir, "X.S01E01.mkv")
	touchEpisodeFile(t, e1)
	repo := &fakeTracksRepo{episodes: []models.Episode{{ID: "e1", FilePath: models.NewNullString(e1)}}}
	svc := NewEpisodeSubtitleTracksService(repo, &fakeProber{available: false}, nil, nil)
	assert.Equal(t, EpisodeTracksSweepResult{}, svc.Run(context.Background()))
	assert.Empty(t, repo.writes)
}

func TestEpisodeSubtitleTracks_CancelStopsAndTheNextPassResumes(t *testing.T) {
	dir := t.TempDir()
	var eps []models.Episode
	for _, n := range []string{"a", "b", "c"} {
		p := filepath.Join(dir, n+".mkv")
		touchEpisodeFile(t, p)
		eps = append(eps, models.Episode{ID: n, FilePath: models.NewNullString(p)})
	}
	repo := &fakeTracksRepo{episodes: eps}
	ctx, cancel := context.WithCancel(context.Background())
	prober := &cancelAfterProber{fakeProber: fakeProber{available: true}, after: 1, cancel: cancel}
	svc := NewEpisodeSubtitleTracksService(repo, prober, nil, nil)

	res := svc.Run(ctx)
	assert.Equal(t, 1, res.Probed, "stops at the first cancelled probe")

	res = svc.Run(context.Background())
	assert.Equal(t, 2, res.Probed, "resumes with the two it had not reached")
	assert.Len(t, repo.writes, 3)
}

type cancelAfterProber struct {
	fakeProber
	after  int
	cancel context.CancelFunc
}

func (p *cancelAfterProber) Probe(ctx context.Context, path string) (*MediaTechInfo, error) {
	info, err := p.fakeProber.Probe(ctx, path)
	if p.callCount() >= p.after {
		p.cancel()
	}
	return info, err
}

func TestEpisodeSubtitleTracks_TriggerCoalesces(t *testing.T) {
	dir := t.TempDir()
	e1 := filepath.Join(dir, "a.mkv")
	touchEpisodeFile(t, e1)
	repo := &fakeTracksRepo{episodes: []models.Episode{{ID: "a", FilePath: models.NewNullString(e1)}}}
	release := make(chan struct{})
	prober := &fakeProber{available: true, block: release}
	svc := NewEpisodeSubtitleTracksService(repo, prober, nil, nil)

	ctx := context.Background()
	svc.Trigger(ctx)
	// A scan finishes and adds a file while the first pass is blocked.
	e2 := filepath.Join(dir, "b.mkv")
	touchEpisodeFile(t, e2)
	repo.mu.Lock()
	repo.episodes = append(repo.episodes, models.Episode{ID: "b", FilePath: models.NewNullString(e2)})
	repo.mu.Unlock()
	svc.Trigger(ctx)
	svc.Trigger(ctx)
	close(release)

	require.Eventually(t, func() bool {
		svc.mu.Lock()
		defer svc.mu.Unlock()
		return !svc.running
	}, 2*time.Second, 10*time.Millisecond)
	assert.Equal(t, 2, prober.callCount(), "one queued re-run reads the new file; no file is probed twice")
	repo.mu.Lock()
	defer repo.mu.Unlock()
	assert.Len(t, repo.writes, 2)
}

func TestEpisodeSubtitleTracks_UnreadableFolderStoresTheEmbeddedHalf(t *testing.T) {
	dir := t.TempDir()
	e1 := filepath.Join(dir, "X.S01E01.mkv")
	touchEpisodeFile(t, e1)
	repo := &fakeTracksRepo{episodes: []models.Episode{{ID: "e1", FilePath: models.NewNullString(e1)}}}
	prober := &fakeProber{available: true, info: map[string]*MediaTechInfo{
		e1: {SubtitleTracks: []SubtitleTrack{{Language: "chi", Format: "subrip", Title: "繁體中文"}}},
	}}
	side := &fakeSidecars{out: map[string]SidecarTracks{e1: {Err: errors.New("EIO")}}}
	svc := NewEpisodeSubtitleTracksService(repo, prober, side, nil)

	assert.Equal(t, 1, svc.Run(context.Background()).Probed)
	assert.Equal(t, models.ChineseSubtitleZhHant, models.ChineseSubtitleVerdict("not_searched", "", repo.writes["e1"][0]),
		"the probe result is kept; the season list fills the sidecars in later")
}

func TestEpisodeSubtitleTracks_PassesStoredSidecarsSoUnchangedFilesAreNotReread(t *testing.T) {
	dir := t.TempDir()
	e1 := filepath.Join(dir, "X.S01E01.mkv")
	touchEpisodeFile(t, e1)
	stored := tracksJSON(t, sidecar("zh-Hant", "X.S01E01.zh-TW.srt"))
	repo := &fakeTracksRepo{episodes: []models.Episode{{ID: "e1", FilePath: models.NewNullString(e1),
		SubtitleTracks: stored, SubtitleTracksFileSig: models.NewNullString("old")}}}
	side := &fakeSidecars{}
	svc := NewEpisodeSubtitleTracksService(repo, &fakeProber{available: true}, side, nil)

	svc.Run(context.Background())
	require.Len(t, side.known, 1)
	assert.Equal(t, []SubtitleTrack{sidecar("zh-Hant", "X.S01E01.zh-TW.srt")}, side.known[0][e1])
}

func TestEpisodeSubtitleTracks_RunAfterRetriesWhenNoFileIsReachable(t *testing.T) {
	oldDelay, oldRetries := unmountedRetryDelay, unmountedRetries
	unmountedRetryDelay, unmountedRetries = 20*time.Millisecond, 3
	t.Cleanup(func() { unmountedRetryDelay, unmountedRetries = oldDelay, oldRetries })

	dir := t.TempDir()
	e1 := filepath.Join(dir, "share", "X.S01E01.mkv") // not there yet: share unmounted
	repo := &fakeTracksRepo{episodes: []models.Episode{{ID: "e1", FilePath: models.NewNullString(e1)}}}
	prober := &fakeProber{available: true}
	svc := NewEpisodeSubtitleTracksService(repo, prober, nil, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.RunAfter(ctx, time.Millisecond)
	time.Sleep(5 * time.Millisecond)
	touchEpisodeFile(t, e1) // the share comes up

	require.Eventually(t, func() bool { return prober.callCount() == 1 }, 2*time.Second, 5*time.Millisecond,
		"the boot pass must not give up just because the NAS mounted late")
	cancel()
	svc.Wait()
}
