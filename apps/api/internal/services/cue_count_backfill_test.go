package services

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

// disc-2026-11-backfill-subtitle-cue-counts: tracks stored before cue counts
// were read get ONE more probe when their English track is what the extract
// lane would translate; a count that is not stated is stored as -1 and never
// asked for again.

var _ MovieCueCountRepo = (*repository.MovieRepository)(nil)

// englishTextRule stands in for subtitle.CueCountRule (services cannot import
// subtitle): embedded subrip tracks, Chinese first, else English. The real
// rule is tested in the subtitle package and end to end in cmd/api.
type englishTextRule struct{}

func (englishTextRule) candidates(tracks []SubtitleTrack) []SubtitleTrack {
	var zh, en []SubtitleTrack
	for _, t := range tracks {
		if t.External || t.Format != "subrip" {
			continue
		}
		switch {
		case strings.HasPrefix(t.Language, "chi") || strings.HasPrefix(t.Language, "zh"):
			zh = append(zh, t)
		case t.Language == "eng" || t.Language == "en":
			en = append(en, t)
		}
	}
	if len(zh) > 0 {
		return zh
	}
	return en
}

func (r englishTextRule) NeedsCueCount(tracks []SubtitleTrack) bool {
	c := r.candidates(tracks)
	if len(c) == 0 || c[0].Language != "eng" && c[0].Language != "en" {
		return false
	}
	for _, t := range c {
		if t.CueCount == 0 {
			return true
		}
	}
	return false
}

func (r englishTextRule) CuesFromTracks(tracks []SubtitleTrack) int {
	n := 0
	for _, t := range r.candidates(tracks) {
		n = max(n, t.CueCount)
	}
	return n
}

// storedEpisode is an episode whose tracks were read from THIS file before
// cue counts existed.
func storedEpisode(t *testing.T, id, path, tracksJSON string) models.Episode {
	info := touchEpisodeFile(t, path)
	return models.Episode{
		ID: id, FilePath: models.NewNullString(path),
		SubtitleTracks:        models.NewNullString(tracksJSON),
		SubtitleTracksFileSig: models.NewNullString(FileSignature(info)),
		DurationSeconds:       models.NewNullInt64(2464),
	}
}

func TestEpisodeSweep_BackfillsTheCueCountOnce(t *testing.T) {
	dir := t.TempDir()
	scorpion := filepath.Join(dir, "Scorpion.S01E02.mkv")
	untagged := filepath.Join(dir, "Untagged.S01E01.mkv")
	const oldEnglish = `[{"language":"eng","format":"subrip","external":false,"stream_index":3}]`
	repo := &fakeTracksRepo{episodes: []models.Episode{
		storedEpisode(t, "scorpion", scorpion, oldEnglish),
		storedEpisode(t, "untagged", untagged, oldEnglish),
	}}
	prober := &fakeProber{available: true, info: map[string]*MediaTechInfo{
		scorpion: {SubtitleTracks: []SubtitleTrack{{Language: "eng", Format: "subrip", StreamIndex: 3, CueCount: 1609}}},
		untagged: {SubtitleTracks: []SubtitleTrack{{Language: "eng", Format: "subrip", StreamIndex: 3, CueCount: CueCountUnstated}}},
	}}
	svc := NewEpisodeSubtitleTracksService(repo, prober, nil, nil)
	svc.SetCueCountRule(englishTextRule{})

	res := svc.Run(context.Background())
	assert.Equal(t, 2, res.Probed, "same file, re-read for the count")
	assert.Equal(t, 1, res.CuesFilled)
	assert.Equal(t, 1, res.CuesUnstated)

	var stored []SubtitleTrack
	require.NoError(t, json.Unmarshal([]byte(repo.writes["scorpion"][0]), &stored))
	assert.Equal(t, 1609, stored[0].CueCount)
	assert.Contains(t, repo.writes["untagged"][0], `"cue_count":-1`, "read, unstated — kept so it is not asked again")

	// Next pass: nothing left to backfill.
	again := svc.Run(context.Background())
	assert.Zero(t, again.Probed)
	assert.Equal(t, 2, again.Skipped)
	assert.Len(t, prober.calls, 2, "each file was probed exactly once")
}

func TestEpisodeSweep_NoEnglishTrackToTranslateIsNotReread(t *testing.T) {
	dir := t.TempDir()
	cases := map[string]string{
		"chinese text track":           `[{"language":"chi","format":"subrip","external":false,"stream_index":3},{"language":"eng","format":"subrip","external":false,"stream_index":4}]`,
		"image track only":             `[{"language":"eng","format":"hdmv_pgs_subtitle","external":false,"stream_index":3}]`,
		"no tracks":                    `[]`,
		"und text track":               `[{"language":"und","format":"subrip","external":false,"stream_index":3}]`,
		"English sidecar only":         `[{"language":"en","format":"srt","external":true,"stream_index":0,"file_name":"x.en.srt"}]`,
		"already counted":              `[{"language":"eng","format":"subrip","external":false,"stream_index":3,"cue_count":900}]`,
		"already read, count unstated": `[{"language":"eng","format":"subrip","external":false,"stream_index":3,"cue_count":-1}]`,
	}
	var eps []models.Episode
	for name, tracks := range cases {
		eps = append(eps, storedEpisode(t, name, filepath.Join(dir, name+".mkv"), tracks))
	}
	prober := &fakeProber{available: true}
	svc := NewEpisodeSubtitleTracksService(&fakeTracksRepo{episodes: eps}, prober, nil, nil)
	svc.SetCueCountRule(englishTextRule{})

	res := svc.Run(context.Background())
	assert.Zero(t, res.Probed)
	assert.Equal(t, len(cases), res.Skipped)
	assert.Empty(t, prober.calls)
}

func TestEpisodeSweep_WithoutARuleNothingIsReread(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "S01E01.mkv")
	prober := &fakeProber{available: true}
	svc := NewEpisodeSubtitleTracksService(&fakeTracksRepo{episodes: []models.Episode{
		storedEpisode(t, "e1", path, `[{"language":"eng","format":"subrip","external":false,"stream_index":3}]`),
	}}, prober, nil, nil)
	assert.Zero(t, svc.Run(context.Background()).Probed)
}

func TestEpisodeSweep_FailedCueProbeIsNotRetriedThisProcess(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "S01E01.mkv")
	prober := &fakeProber{available: true, fail: map[string]bool{path: true}}
	svc := NewEpisodeSubtitleTracksService(&fakeTracksRepo{episodes: []models.Episode{
		storedEpisode(t, "e1", path, `[{"language":"eng","format":"subrip","external":false,"stream_index":3}]`),
	}}, prober, nil, nil)
	svc.SetCueCountRule(englishTextRule{})
	assert.Equal(t, 1, svc.Run(context.Background()).Failed)
	assert.Equal(t, 1, svc.Run(context.Background()).Skipped)
	assert.Len(t, prober.calls, 1)
}

// ─── Movies ─────────────────────────────────────────────────────────────────

type fakeCueMovieRepo struct {
	fakeMovieTracksRepo
	cueMovies []models.Movie
	replaced  map[string]string
	stale     map[string]bool // ReplaceSubtitleTracks finds the row changed
}

func (f *fakeCueMovieRepo) FindMoviesMissingCueCounts(_ context.Context, afterID string, limit int) ([]models.Movie, error) {
	var out []models.Movie
	for _, m := range f.cueMovies {
		if m.ID > afterID && len(out) < limit {
			out = append(out, m)
		}
	}
	return out, nil
}

func (f *fakeCueMovieRepo) ReplaceSubtitleTracks(_ context.Context, id, previous, next string) (bool, error) {
	if f.stale[id] {
		return false, nil
	}
	if f.replaced == nil {
		f.replaced = map[string]string{}
	}
	f.replaced[id] = next
	for i := range f.cueMovies {
		if f.cueMovies[i].ID == id && f.cueMovies[i].SubtitleTracks.String == previous {
			f.cueMovies[i].SubtitleTracks = models.NewNullString(next)
		}
	}
	return true, nil
}

func cueMovie(t *testing.T, dir, id, tracks string) models.Movie {
	path := filepath.Join(dir, id+".mkv")
	require.NoError(t, os.WriteFile(path, []byte("video"), 0o644))
	m := models.Movie{ID: id, Title: id}
	m.FilePath = models.NewNullString(path)
	m.SubtitleTracks = models.NewNullString(tracks)
	return m
}

func TestMovieBackfill_FillsCueCountsWithTheSameRule(t *testing.T) {
	dir := t.TempDir()
	english := cueMovie(t, dir, "a-english", `[{"language":"eng","format":"subrip","external":false,"stream_index":2}]`)
	chinese := cueMovie(t, dir, "b-chinese", `[{"language":"chi","format":"subrip","external":false,"stream_index":2}]`)
	pgs := cueMovie(t, dir, "c-pgs", `[{"language":"eng","format":"hdmv_pgs_subtitle","external":false,"stream_index":2}]`)
	untagged := cueMovie(t, dir, "d-untagged", `[{"language":"eng","format":"subrip","external":false,"stream_index":2}]`)
	repo := &fakeCueMovieRepo{cueMovies: []models.Movie{english, chinese, pgs, untagged}}
	prober := &fakeProber{available: true, info: map[string]*MediaTechInfo{
		english.FilePath.String:  {SubtitleTracks: []SubtitleTrack{{Language: "eng", Format: "subrip", StreamIndex: 2, CueCount: 1609}}},
		untagged.FilePath.String: {SubtitleTracks: []SubtitleTrack{{Language: "eng", Format: "subrip", StreamIndex: 2, CueCount: CueCountUnstated}}},
	}}
	svc := NewMovieSubtitleTracksBackfillService(repo, prober, nil)
	svc.sidecars = func(string) []SubtitleTrack { return nil }
	svc.SetCueCountRule(englishTextRule{})

	res := svc.Run(context.Background())
	assert.Equal(t, 1, res.CuesFilled)
	assert.Equal(t, 1, res.CuesUnstated)
	assert.ElementsMatch(t, []string{english.FilePath.String, untagged.FilePath.String}, prober.calls,
		"a Chinese track or an image track is never re-read")
	assert.Contains(t, repo.replaced["a-english"], `"cue_count":1609`)
	assert.Contains(t, repo.replaced["d-untagged"], `"cue_count":-1`)
	assert.NotContains(t, repo.replaced, "b-chinese")
	assert.NotContains(t, repo.replaced, "c-pgs")

	// The fake listing still returns them (the SQL half drops rows carrying a
	// cue_count); the rule now says nothing is needed, so nothing is probed.
	again := svc.Run(context.Background())
	assert.Zero(t, again.CuesFilled+again.CuesUnstated)
	assert.Len(t, prober.calls, 2)
}

func TestMovieBackfill_CueWriteNeverOverwritesANewerRead(t *testing.T) {
	dir := t.TempDir()
	m := cueMovie(t, dir, "m1", `[{"language":"eng","format":"subrip","external":false,"stream_index":2}]`)
	repo := &fakeCueMovieRepo{cueMovies: []models.Movie{m}, stale: map[string]bool{"m1": true}}
	prober := &fakeProber{available: true, info: map[string]*MediaTechInfo{
		m.FilePath.String: {SubtitleTracks: []SubtitleTrack{{Language: "eng", Format: "subrip", StreamIndex: 2, CueCount: 700}}},
	}}
	svc := NewMovieSubtitleTracksBackfillService(repo, prober, nil)
	svc.sidecars = func(string) []SubtitleTrack { return nil }
	svc.SetCueCountRule(englishTextRule{})
	res := svc.Run(context.Background())
	assert.Equal(t, 1, res.Skipped, "a scan re-read the movie meanwhile — its answer stands")
	assert.Zero(t, res.CuesFilled)
}

func TestMovieBackfill_RepoWithoutCuePhaseIsUnchanged(t *testing.T) {
	prober := &fakeProber{available: true}
	svc := NewMovieSubtitleTracksBackfillService(&fakeMovieTracksRepo{}, prober, nil)
	svc.SetCueCountRule(englishTextRule{})
	assert.Equal(t, MovieTracksBackfillResult{}, svc.Run(context.Background()))
}
