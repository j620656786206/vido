package services

// Story sub-4-1 AC #2/#4/#5/#6 — the cost preview.
//
// Every assertion here is about a number the user will read before deciding to
// spend money, so the bar is "would this quote be honest?", not just "does it
// compile".

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
)

// ─── Fakes ─────────────────────────────────────────────────────────────────

type stubMovieFinder struct {
	movies []models.Movie
	err    error
}

func (f *stubMovieFinder) FindMissingZhHantSubtitle(context.Context) ([]models.Movie, error) {
	return f.movies, f.err
}

type stubEpisodeFinder struct {
	episodes []models.Episode
	err      error
}

func (f *stubEpisodeFinder) FindMissingZhHantSubtitle(context.Context) ([]models.Episode, error) {
	return f.episodes, f.err
}

// stubPredictor records every probe so a test can prove the persisted-tracks
// fast path avoided disk entirely.
type stubPredictor struct {
	fromTracks RoutePrediction
	probeRoute RoutePrediction
	probeErr   error
	probed     []string
}

func (p *stubPredictor) FromTracks([]SubtitleTrack) RoutePrediction { return p.fromTracks }

func (p *stubPredictor) Probe(_ context.Context, mediaPath string) (RoutePrediction, error) {
	p.probed = append(p.probed, mediaPath)
	return p.probeRoute, p.probeErr
}

func movieRow(id, title, path string, runtime int64, tracksJSON string) models.Movie {
	m := models.Movie{ID: id, Title: title}
	m.FilePath = models.NewNullString(path)
	if runtime > 0 {
		m.Runtime = models.NewNullInt64(runtime)
	}
	if tracksJSON != "" {
		m.SubtitleTracks = models.NewNullString(tracksJSON)
	}
	return m
}

func episodeRow(id string, season, ep int, path string, runtime int64) models.Episode {
	e := models.Episode{ID: id, SeasonNumber: season, EpisodeNumber: ep}
	e.FilePath = models.NewNullString(path)
	if runtime > 0 {
		e.Runtime = models.NewNullInt64(runtime)
	}
	return e
}

// ─── AC #2: both media types are candidates (D1 ruling) ────────────────────

func TestAnalyze_EnumeratesMoviesAndEpisodes(t *testing.T) {
	movies := &stubMovieFinder{movies: []models.Movie{movieRow("m1", "Dune", "/m/dune.mkv", 166, "")}}
	episodes := &stubEpisodeFinder{episodes: []models.Episode{episodeRow("e1", 4, 7, "/tv/s04e07.mkv", 45)}}
	pred := &stubPredictor{probeRoute: RouteASR}

	svc := NewGenerationCandidateService(movies, episodes, pred, false, nil)
	res, err := svc.Analyze(context.Background(), nil)
	require.NoError(t, err)

	require.Len(t, res.Candidates, 2, "episodes are in scope since the D1 ruling")
	types := map[string]bool{}
	for _, c := range res.Candidates {
		types[c.MediaType] = true
	}
	assert.True(t, types[models.SubtitleRunMediaMovie])
	assert.True(t, types[models.SubtitleRunMediaEpisode])
}

func TestAnalyze_SkipsItemsWithNoMediaFile(t *testing.T) {
	movies := &stubMovieFinder{movies: []models.Movie{{ID: "m-nofile", Title: "Ghost"}}}
	svc := NewGenerationCandidateService(movies, nil, &stubPredictor{probeRoute: RouteASR}, false, nil)

	res, err := svc.Analyze(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, res.Candidates, "a row with no file on disk cannot be transcribed and must not be quoted")
}

// ─── AC #4: persisted tracks avoid the probe; everything else probes ───────

func TestAnalyze_PersistedTracksAvoidProbing(t *testing.T) {
	movies := &stubMovieFinder{movies: []models.Movie{
		movieRow("m-enriched", "Enriched", "/m/a.mkv", 120, `[{"language":"eng","format":"subrip","stream_index":2}]`),
		movieRow("m-bare", "Unenriched", "/m/b.mkv", 120, ""),
	}}
	pred := &stubPredictor{fromTracks: RouteExtract, probeRoute: RouteASR}

	svc := NewGenerationCandidateService(movies, nil, pred, false, nil)
	res, err := svc.Analyze(context.Background(), nil)
	require.NoError(t, err)

	assert.Equal(t, []string{"/m/b.mkv"}, pred.probed,
		"a movie whose scan-time tracks are on record must not be re-probed; one without them must be")
	require.Len(t, res.Candidates, 2)
	assert.Equal(t, 1, res.Summary.ExtractCount)
	assert.Equal(t, 1, res.Summary.ASRCount)
}

// An empty/blank/broken tracks column means "we have no scan-time answer", not
// "this file has no tracks" — concluding the latter would quote paid speech
// recognition for every movie whose enrichment never ran.
func TestAnalyze_UnusableTracksJSONFallsBackToProbing(t *testing.T) {
	for name, raw := range map[string]string{
		"empty array": `[]`,
		"malformed":   `{not json`,
	} {
		t.Run(name, func(t *testing.T) {
			movies := &stubMovieFinder{movies: []models.Movie{movieRow("m1", "T", "/m/a.mkv", 100, raw)}}
			pred := &stubPredictor{fromTracks: RouteExtract, probeRoute: RouteASR}

			svc := NewGenerationCandidateService(movies, nil, pred, false, nil)
			_, err := svc.Analyze(context.Background(), nil)
			require.NoError(t, err)
			assert.Equal(t, []string{"/m/a.mkv"}, pred.probed)
		})
	}
}

// ─── AC #5: runtime fallback is visible, not silent ───────────────────────

func TestAnalyze_UnknownRuntimeIsFlaggedAndPricedAtTheStatedDefault(t *testing.T) {
	movies := &stubMovieFinder{movies: []models.Movie{movieRow("m1", "No Runtime", "/m/a.mkv", 0, "")}}
	pred := &stubPredictor{probeRoute: RouteASR}

	svc := NewGenerationCandidateService(movies, nil, pred, false, nil)
	res, err := svc.Analyze(context.Background(), nil)
	require.NoError(t, err)

	require.Len(t, res.Candidates, 1)
	c := res.Candidates[0]
	assert.False(t, c.RuntimeKnown, "the UI has to be able to say 片長未知")
	assert.Equal(t, unknownRuntimeMinutes, c.RuntimeMinutes,
		"the assumed duration must be reported, not hidden inside the price")
}

// ─── AC #6: the money ─────────────────────────────────────────────────────

func TestAnalyze_ASRCostsMoreThanExtractForTheSameRuntime(t *testing.T) {
	asr := estimateUSD(RouteASR, 100, 0.006)
	extract := estimateUSD(RouteExtract, 100, 0.006)

	assert.Greater(t, asr, extract,
		"speech recognition is the paid class — if this ever inverts the screen is warning about the wrong thing")
	assert.Greater(t, extract, 0.0,
		"an extract item still pays for LLM translation; calling it exactly free would be a lie the invoice contradicts")
}

// The spike ran faster-whisper locally at zero marginal cost. Quoting the
// hosted API's rate for that would invent a bill the user will never receive.
func TestAnalyze_SelfHostedASRIsNotBilledAtTheHostedRate(t *testing.T) {
	movies := &stubMovieFinder{movies: []models.Movie{movieRow("m1", "T", "/m/a.mkv", 100, "")}}

	hosted := NewGenerationCandidateService(movies, nil, &stubPredictor{probeRoute: RouteASR}, false, nil)
	selfHosted := NewGenerationCandidateService(movies, nil, &stubPredictor{probeRoute: RouteASR}, true, nil)

	h, err := hosted.Analyze(context.Background(), nil)
	require.NoError(t, err)
	s, err := selfHosted.Analyze(context.Background(), nil)
	require.NoError(t, err)

	assert.Greater(t, h.Summary.EstimatedTotalUSD, s.Summary.EstimatedTotalUSD)
	assert.True(t, s.Summary.SelfHostedASR,
		"the client must be able to explain the low number instead of showing a suspicious $0.00")
}

func TestAnalyze_SummaryTotalEqualsTheSumOfItsRows(t *testing.T) {
	movies := &stubMovieFinder{movies: []models.Movie{
		movieRow("m1", "A", "/m/a.mkv", 120, ""),
		movieRow("m2", "B", "/m/b.mkv", 90, ""),
	}}
	pred := &stubPredictor{probeRoute: RouteASR}

	svc := NewGenerationCandidateService(movies, nil, pred, false, nil)
	res, err := svc.Analyze(context.Background(), nil)
	require.NoError(t, err)

	var sum float64
	for _, c := range res.Candidates {
		sum += c.EstimatedUSD
	}
	assert.InDelta(t, sum, res.Summary.EstimatedTotalUSD, 0.005,
		"the footer total must equal what the rows add up to — a mismatch is the fastest way to lose trust in the number")
}

// ─── Declined items are counted but never quoted ──────────────────────────

func TestAnalyze_SkippedItemsAreCountedNotQuoted(t *testing.T) {
	movies := &stubMovieFinder{movies: []models.Movie{movieRow("m1", "Und Tagged", "/m/a.mkv", 120, "")}}
	pred := &stubPredictor{probeRoute: RouteSkipped}

	svc := NewGenerationCandidateService(movies, nil, pred, false, nil)
	res, err := svc.Analyze(context.Background(), nil)
	require.NoError(t, err)

	assert.Empty(t, res.Candidates, "the pipeline declines these outright — offering to generate them would fail")
	assert.Equal(t, 1, res.Summary.SkippedCount, "but the count is reported so the numbers still add up")
	assert.Zero(t, res.Summary.EstimatedTotalUSD)
}

// ─── Fail-soft + progress ─────────────────────────────────────────────────

func TestAnalyze_OneUnreadableFileDoesNotDenyTheRestAQuote(t *testing.T) {
	movies := &stubMovieFinder{movies: []models.Movie{movieRow("m1", "Broken", "/m/broken.mkv", 120, "")}}
	pred := &stubPredictor{probeErr: errors.New("ffprobe exploded")}

	svc := NewGenerationCandidateService(movies, nil, pred, false, nil)
	res, err := svc.Analyze(context.Background(), nil)
	require.NoError(t, err, "a per-file failure must not fail the whole sweep")
	assert.Empty(t, res.Candidates, "…but an unreadable file must not be priced either")
}

func TestAnalyze_EnumerationFailurePropagates(t *testing.T) {
	movies := &stubMovieFinder{err: errors.New("no such column")}
	svc := NewGenerationCandidateService(movies, nil, &stubPredictor{}, false, nil)

	_, err := svc.Analyze(context.Background(), nil)
	require.Error(t, err, "a broken enumeration means the list is incomplete — showing a short list as if it were the library is worse than an error")
}

func TestAnalyze_ReportsProgressForEveryItem(t *testing.T) {
	movies := &stubMovieFinder{movies: []models.Movie{
		movieRow("m1", "A", "/m/a.mkv", 100, ""),
		movieRow("m2", "B", "/m/b.mkv", 100, ""),
	}}
	pred := &stubPredictor{probeRoute: RouteASR}

	var seen [][2]int
	svc := NewGenerationCandidateService(movies, nil, pred, false, nil)
	_, err := svc.Analyze(context.Background(), func(done, total int) {
		seen = append(seen, [2]int{done, total})
	})
	require.NoError(t, err)

	assert.Equal(t, [][2]int{{1, 2}, {2, 2}}, seen,
		"the analysis pass is real work (one ffprobe per un-enriched file) and must be observable — F14 renders this")
}

func TestAnalyze_CancellationStopsTheSweep(t *testing.T) {
	movies := &stubMovieFinder{movies: []models.Movie{
		movieRow("m1", "A", "/m/a.mkv", 100, ""),
		movieRow("m2", "B", "/m/b.mkv", 100, ""),
	}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	svc := NewGenerationCandidateService(movies, nil, &stubPredictor{probeRoute: RouteASR}, false, nil)
	_, err := svc.Analyze(ctx, nil)
	require.ErrorIs(t, err, context.Canceled)
}
