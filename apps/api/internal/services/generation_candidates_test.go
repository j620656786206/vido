package services

// Story sub-4-1 AC #2/#4/#5/#6 — the cost preview.
//
// Every assertion here is about a number the user will read before deciding to
// spend money, so the bar is "would this quote be honest?", not just "does it
// compile".

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/ai"
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

	svc := NewGenerationCandidateService(
		movies, episodes, nil, pred, false, 0, nil)
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
	svc := NewGenerationCandidateService(
		movies, nil, nil, &stubPredictor{probeRoute: RouteASR}, false, 0, nil)

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

	svc := NewGenerationCandidateService(
		movies, nil, nil, pred, false, 0, nil)
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

			svc := NewGenerationCandidateService(
				movies, nil, nil, pred, false, 0, nil)
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

	svc := NewGenerationCandidateService(
		movies, nil, nil, pred, false, 0, nil)
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
	asrRate := decimal.RequireFromString("0.006")
	asr := estimateUSD(RouteASR, 100, asrRate, ai.DefaultClaudeModel)
	extract := estimateUSD(RouteExtract, 100, asrRate, ai.DefaultClaudeModel)

	assert.True(t, asr.GreaterThan(extract),
		"speech recognition is the paid class — if this ever inverts the screen is warning about the wrong thing")
	assert.True(t, extract.IsPositive(),
		"an extract item still pays for LLM translation; calling it exactly free would be a lie the invoice contradicts")
}

// The spike ran faster-whisper locally at zero marginal cost. Quoting the
// hosted API's rate for that would invent a bill the user will never receive.
func TestAnalyze_SelfHostedASRIsNotBilledAtTheHostedRate(t *testing.T) {
	movies := &stubMovieFinder{movies: []models.Movie{movieRow("m1", "T", "/m/a.mkv", 100, "")}}

	hosted := NewGenerationCandidateService(
		movies, nil, nil, &stubPredictor{probeRoute: RouteASR}, false, 0, nil)
	selfHosted := NewGenerationCandidateService(
		movies, nil, nil, &stubPredictor{probeRoute: RouteASR}, true, 0, nil)

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

	svc := NewGenerationCandidateService(
		movies, nil, nil, pred, false, 0, nil)
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

	svc := NewGenerationCandidateService(
		movies, nil, nil, pred, false, 0, nil)
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

	svc := NewGenerationCandidateService(
		movies, nil, nil, pred, false, 0, nil)
	res, err := svc.Analyze(context.Background(), nil)
	require.NoError(t, err, "a per-file failure must not fail the whole sweep")
	assert.Empty(t, res.Candidates, "…but an unreadable file must not be priced either")
}

func TestAnalyze_EnumerationFailurePropagates(t *testing.T) {
	movies := &stubMovieFinder{err: errors.New("no such column")}
	svc := NewGenerationCandidateService(
		movies, nil, nil, &stubPredictor{}, false, 0, nil)

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
	svc := NewGenerationCandidateService(
		movies, nil, nil, pred, false, 0, nil)
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

	svc := NewGenerationCandidateService(
		movies, nil, nil, &stubPredictor{probeRoute: RouteASR}, false, 0, nil)
	_, err := svc.Analyze(ctx, nil)
	require.ErrorIs(t, err, context.Canceled)
}

// ─── AC #8: the sweep is observable and interruptible ─────────────────────

// blockingPredictor parks inside Probe so a test can observe the RUNNING state
// rather than infer it from a race.
type blockingPredictor struct {
	entered chan struct{}
	release chan struct{}
}

func (p *blockingPredictor) FromTracks([]SubtitleTrack) RoutePrediction { return RouteExtract }

func (p *blockingPredictor) Probe(ctx context.Context, _ string) (RoutePrediction, error) {
	select {
	case p.entered <- struct{}{}:
	default:
	}
	select {
	case <-p.release:
		return RouteASR, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func analysisService(t *testing.T, pred RoutePredictor, n int) *GenerationCandidateService {
	t.Helper()
	movies := make([]models.Movie, 0, n)
	for i := 0; i < n; i++ {
		id := string(rune('a' + i))
		movies = append(movies, movieRow(id, "T"+id, "/m/"+id+".mkv", 100, ""))
	}
	return NewGenerationCandidateService(
		&stubMovieFinder{movies: movies}, nil, nil, pred, false, 0, nil)
}

func TestStartAnalysis_IsSingleFlight(t *testing.T) {
	pred := &blockingPredictor{entered: make(chan struct{}, 1), release: make(chan struct{})}
	svc := analysisService(t, pred, 2)

	require.NoError(t, svc.StartAnalysis())
	<-pred.entered // the sweep is genuinely in flight, not merely scheduled

	assert.ErrorIs(t, svc.StartAnalysis(), ErrAnalysisRunning,
		"a second sweep would double the ffprobe load for an answer the first one is already producing")
	assert.Equal(t, AnalysisRunning, svc.Snapshot().Status)

	close(pred.release)
	require.Eventually(t, func() bool { return svc.Snapshot().Status == AnalysisReady },
		2*time.Second, 5*time.Millisecond)
}

func TestStartAnalysis_ReadySnapshotCarriesTheResultAndATimestamp(t *testing.T) {
	svc := analysisService(t, &stubPredictor{probeRoute: RouteASR}, 2)
	require.NoError(t, svc.StartAnalysis())

	require.Eventually(t, func() bool { return svc.Snapshot().Status == AnalysisReady },
		2*time.Second, 5*time.Millisecond)

	snap := svc.Snapshot()
	require.NotNil(t, snap.Result)
	assert.Len(t, snap.Result.Candidates, 2)
	assert.Equal(t, 2, snap.Analyzed)
	assert.Equal(t, 2, snap.Total)
	require.NotNil(t, snap.AnalyzedAt, "the client has to be able to say how stale the numbers are")
}

// A partial classification is not a quote: pricing a fraction of the library as
// if it were all of it is exactly the misleading number this feature exists to
// remove.
func TestCancelAnalysis_DiscardsThePartialResult(t *testing.T) {
	pred := &blockingPredictor{entered: make(chan struct{}, 1), release: make(chan struct{})}
	svc := analysisService(t, pred, 3)

	require.NoError(t, svc.StartAnalysis())
	<-pred.entered
	svc.CancelAnalysis()

	require.Eventually(t, func() bool { return svc.Snapshot().Status == AnalysisCancelled },
		2*time.Second, 5*time.Millisecond)
	assert.Nil(t, svc.Snapshot().Result)
}

func TestStartAnalysis_EnumerationFailureIsReportedNotSilentlyEmpty(t *testing.T) {
	svc := NewGenerationCandidateService(
		&stubMovieFinder{err: errors.New("no such column")}, nil, nil, &stubPredictor{}, false, 0, nil)
	require.NoError(t, svc.StartAnalysis())

	require.Eventually(t, func() bool { return svc.Snapshot().Status == AnalysisFailed },
		2*time.Second, 5*time.Millisecond)

	snap := svc.Snapshot()
	assert.Nil(t, snap.Result, "an empty list would read as 'nothing to generate', which is the opposite of the truth")
	assert.Contains(t, snap.Error, "no such column")
}

func TestSnapshot_StartsIdle(t *testing.T) {
	svc := analysisService(t, &stubPredictor{probeRoute: RouteASR}, 1)
	snap := svc.Snapshot()
	assert.Equal(t, AnalysisIdle, snap.Status)
	assert.Nil(t, snap.Result)
}

// A nil hub must not panic — the service is constructed before the hub is wired
// in main.go, and tests never wire one.
func TestBroadcast_IsNilHubSafe(t *testing.T) {
	svc := analysisService(t, &stubPredictor{probeRoute: RouteASR}, 1)
	require.NoError(t, svc.StartAnalysis())
	require.Eventually(t, func() bool { return svc.Snapshot().Status == AnalysisReady },
		2*time.Second, 5*time.Millisecond)
}

// ─── CR sub-4-1 H1: cancel is immediate and stale runs cannot clobber ──────

// Pressing 取消 must flip the state NOW, not after the in-flight ffprobe's
// timeout — a cancel button with up-to-10s latency reads as broken.
func TestCancelAnalysis_IsImmediate(t *testing.T) {
	pred := &blockingPredictor{entered: make(chan struct{}, 1), release: make(chan struct{})}
	svc := analysisService(t, pred, 3)

	require.NoError(t, svc.StartAnalysis())
	<-pred.entered // the probe is genuinely parked
	svc.CancelAnalysis()

	// No Eventually: the flip is synchronous even though the goroutine is
	// still parked inside Probe.
	snap := svc.Snapshot()
	assert.Equal(t, AnalysisCancelled, snap.Status)
	assert.Nil(t, snap.Result)

	close(pred.release) // let the zombie exit; it must not resurrect anything
	assert.Never(t, func() bool { return svc.Snapshot().Status != AnalysisCancelled },
		300*time.Millisecond, 20*time.Millisecond,
		"the superseded goroutine's wind-down must be a no-op")
}

// Cancel-then-restart: the superseded goroutine's wind-down must not mislabel
// the NEW run as cancelled, nil its cancel func, or fight its counters.
func TestCancelThenRestart_StaleRunCannotClobberTheNewOne(t *testing.T) {
	pred := &blockingPredictor{entered: make(chan struct{}, 2), release: make(chan struct{})}
	svc := analysisService(t, pred, 2)

	require.NoError(t, svc.StartAnalysis())
	<-pred.entered
	svc.CancelAnalysis()

	// Restart while job 1's goroutine is still parked.
	require.NoError(t, svc.StartAnalysis())
	<-pred.entered
	assert.Equal(t, AnalysisRunning, svc.Snapshot().Status)

	// Job 1's zombie wakes up and winds down; job 2 completes normally.
	close(pred.release)
	require.Eventually(t, func() bool { return svc.Snapshot().Status == AnalysisReady },
		2*time.Second, 5*time.Millisecond,
		"the new run must reach ready — a stale wind-down marking it cancelled is the CR H1 clobber")
	require.NotNil(t, svc.Snapshot().Result)
}

// And the new run must remain CANCELLABLE after a stale run existed — the old
// wind-down used to nil the shared cancel func.
func TestCancelThenRestart_TheNewRunIsStillCancellable(t *testing.T) {
	pred := &blockingPredictor{entered: make(chan struct{}, 2), release: make(chan struct{})}
	svc := analysisService(t, pred, 2)

	require.NoError(t, svc.StartAnalysis())
	<-pred.entered
	svc.CancelAnalysis()

	require.NoError(t, svc.StartAnalysis())
	<-pred.entered
	svc.CancelAnalysis()
	assert.Equal(t, AnalysisCancelled, svc.Snapshot().Status,
		"the second cancel must work — a nil-ed cancel func would make the new run unstoppable")

	close(pred.release)
}

// sub-5-1 AC #5: the envelope carries the operator's configured default budget
// on EVERY snapshot state — the F15 prefill reads it from the same GET the
// consent flow already makes, so no new endpoint exists for one float.
func TestSnapshot_CarriesTheConfiguredDefaultBudget(t *testing.T) {
	svc := NewGenerationCandidateService(
		&stubMovieFinder{}, nil, nil, &stubPredictor{}, false, 7.5, nil)

	snap := svc.Snapshot()
	assert.Equal(t, AnalysisIdle, snap.Status)
	assert.InDelta(t, 7.5, snap.DefaultBudgetUSD, 1e-9,
		"idle snapshots carry it too — the prefill must not wait for an analysis")
}

func TestSnapshot_DefaultBudgetSerializesAsSnakeCase(t *testing.T) {
	svc := NewGenerationCandidateService(
		&stubMovieFinder{}, nil, nil, &stubPredictor{}, false, 5, nil)
	raw, err := json.Marshal(svc.Snapshot())
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"default_budget_usd":5`,
		"Rule 6 wire shape — the FE reads default_budget_usd")
}

// ─── sub-5-3 AC #1: candidates carry series identity (additive, no bump) ────

// stubSeriesResolver counts lookups so the per-sweep memo is falsifiable.
type stubSeriesResolver struct {
	titles map[string]string
	err    error
	calls  int
}

func (f *stubSeriesResolver) FindByID(_ context.Context, id string) (*models.Series, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	title, ok := f.titles[id]
	if !ok {
		return nil, errors.New("series not found")
	}
	return &models.Series{ID: id, Title: title}, nil
}

func seriesEpisodeRow(id, seriesID string, season, ep int, path string) models.Episode {
	e := episodeRow(id, season, ep, path, 45)
	e.SeriesID = seriesID
	e.EpisodeNumber = ep
	return e
}

func TestAnalyze_EpisodeCandidatesCarrySeriesIdentity(t *testing.T) {
	episodes := &stubEpisodeFinder{episodes: []models.Episode{
		seriesEpisodeRow("e1", "srs-1", 1, 1, "/tv/a/s01e01.mkv"),
		// S00 specials: season number ZERO is a legal value and must survive
		// the wire (the field is deliberately NOT omitempty).
		seriesEpisodeRow("e2", "srs-1", 0, 1, "/tv/a/s00e01.mkv"),
	}}
	series := &stubSeriesResolver{titles: map[string]string{"srs-1": "怪奇物語"}}
	svc := NewGenerationCandidateService(
		nil, episodes, series, &stubPredictor{probeRoute: RouteASR}, false, 0, nil)

	result, err := svc.Analyze(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, result.Candidates, 2)

	for _, c := range result.Candidates {
		assert.Equal(t, "srs-1", c.SeriesID)
		assert.Equal(t, "怪奇物語", c.SeriesTitle)
	}
	seasons := []int{result.Candidates[0].SeasonNumber, result.Candidates[1].SeasonNumber}
	assert.ElementsMatch(t, []int{1, 0}, seasons, "S00 must arrive as 0, not be dropped")
}

func TestAnalyze_MovieCandidatesCarryNoSeriesIdentity(t *testing.T) {
	movies := &stubMovieFinder{movies: []models.Movie{movieRow("m1", "電影甲", "/movies/a.mkv", 100, "")}}
	series := &stubSeriesResolver{}
	svc := NewGenerationCandidateService(
		movies, nil, series, &stubPredictor{probeRoute: RouteExtract}, false, 0, nil)

	result, err := svc.Analyze(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, result.Candidates, 1)
	assert.Empty(t, result.Candidates[0].SeriesID)
	assert.Empty(t, result.Candidates[0].SeriesTitle)
	assert.Zero(t, series.calls, "movies must not trigger series lookups")
}

func TestAnalyze_SeriesTitleLookupIsMemoizedPerSweep(t *testing.T) {
	// 24 episodes across TWO series — the title lookup must run once per
	// series, not once per episode (a 20-season show must not mean 500 reads).
	var eps []models.Episode
	for i := 1; i <= 12; i++ {
		eps = append(eps, seriesEpisodeRow(fmt.Sprintf("a%d", i), "srs-a", 1, i, fmt.Sprintf("/tv/a/e%d.mkv", i)))
		eps = append(eps, seriesEpisodeRow(fmt.Sprintf("b%d", i), "srs-b", 1, i, fmt.Sprintf("/tv/b/e%d.mkv", i)))
	}
	series := &stubSeriesResolver{titles: map[string]string{"srs-a": "劇甲", "srs-b": "劇乙"}}
	svc := NewGenerationCandidateService(
		nil, &stubEpisodeFinder{episodes: eps}, series, &stubPredictor{probeRoute: RouteASR}, false, 0, nil)

	_, err := svc.Analyze(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, 2, series.calls, "one lookup per distinct series per sweep")
}

// Rule 13 case 3: a missing/failed series row degrades THAT ROW's title to
// empty — it must never fail the sweep or damage other rows.
func TestAnalyze_SeriesTitleFailureDegradesRowNotSweep(t *testing.T) {
	episodes := &stubEpisodeFinder{episodes: []models.Episode{
		seriesEpisodeRow("e1", "srs-gone", 1, 1, "/tv/x/e1.mkv"),
	}}
	series := &stubSeriesResolver{err: errors.New("db closed")}
	svc := NewGenerationCandidateService(
		nil, episodes, series, &stubPredictor{probeRoute: RouteASR}, false, 0, nil)

	result, err := svc.Analyze(context.Background(), nil)
	require.NoError(t, err, "a title lookup failure must not fail the sweep")
	require.Len(t, result.Candidates, 1)
	assert.Equal(t, "srs-gone", result.Candidates[0].SeriesID, "the GROUPING key survives — only the label degrades")
	assert.Empty(t, result.Candidates[0].SeriesTitle)
}

// A nil resolver (legacy wiring, or a test that does not care) degrades to
// empty identity — never panics.
func TestAnalyze_NilSeriesResolverDegradesToFlat(t *testing.T) {
	episodes := &stubEpisodeFinder{episodes: []models.Episode{
		seriesEpisodeRow("e1", "srs-1", 1, 1, "/tv/a/e1.mkv"),
	}}
	svc := NewGenerationCandidateService(
		nil, episodes, nil, &stubPredictor{probeRoute: RouteASR}, false, 0, nil)

	result, err := svc.Analyze(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, result.Candidates, 1)
	assert.Equal(t, "srs-1", result.Candidates[0].SeriesID)
	assert.Empty(t, result.Candidates[0].SeriesTitle)
}

func TestGenerationCandidate_SeriesFieldsSerializeAsSnakeCase(t *testing.T) {
	c := GenerationCandidate{MediaID: "e1", SeriesID: "srs-1", SeriesTitle: "劇甲", SeasonNumber: 0}
	raw, err := json.Marshal(c)
	require.NoError(t, err)
	s := string(raw)
	assert.Contains(t, s, `"series_id":"srs-1"`)
	assert.Contains(t, s, `"series_title":"劇甲"`)
	assert.Contains(t, s, `"season_number":0`, "S00 must serialize — season_number is NOT omitempty")
	assert.Contains(t, s, `"episode_number":0`, "episode_number ships unconditionally too — same zero-value rationale")
}

func TestGenerationCandidate_MovieWireShapeOmitsSeriesKeys(t *testing.T) {
	c := GenerationCandidate{MediaID: "m1", MediaType: models.SubtitleRunMediaMovie}
	raw, err := json.Marshal(c)
	require.NoError(t, err)
	s := string(raw)
	assert.NotContains(t, s, "series_id", "movie rows keep the pre-sub-5-3 shape minus nothing — series keys are omitempty")
	assert.NotContains(t, s, "series_title")
}

// ─── sub-6-1 AC #4: writable probe on the consent list ─────────────────────

// TestMain: fixture paths in this file do not exist on disk, so the real
// fsprobe would mark every candidate unwritable. The sub-6-1 tests below
// inject their own probe per service instance.
func TestMain(m *testing.M) {
	defaultWritableProbe = func(context.Context, string) error { return nil }
	os.Exit(m.Run())
}

func TestAnalyze_UnwritableTargetIsFlaggedAndExcludedFromTotal(t *testing.T) {
	movies := &stubMovieFinder{movies: []models.Movie{
		movieRow("m1", "Dune", "/ro/dune.mkv", 166, ""),
		movieRow("m2", "Heat", "/rw/heat.mkv", 170, ""),
	}}
	pred := &stubPredictor{probeRoute: RouteExtract}
	svc := NewGenerationCandidateService(movies, nil, nil, pred, false, 0, nil)
	svc.probeWritable = func(_ context.Context, dir string) error {
		if dir == "/ro" {
			return errors.New("permission denied")
		}
		return nil
	}

	res, err := svc.Analyze(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, res.Candidates, 2)

	byID := map[string]GenerationCandidate{}
	for _, c := range res.Candidates {
		byID[c.MediaID] = c
	}
	assert.False(t, byID["m1"].Writable)
	assert.Equal(t, "SUBTITLE_TARGET_NOT_WRITABLE", byID["m1"].Blocker, "a code, not prose — the FE composes the sentence")
	assert.Equal(t, "ro", byID["m1"].BlockerDir, "base name only: an absolute NAS path never leaves the server")
	assert.True(t, byID["m2"].Writable)
	assert.Empty(t, byID["m2"].Blocker)
	assert.Equal(t, 1, res.Summary.UnwritableCount)
	assert.InDelta(t, byID["m2"].EstimatedUSD, res.Summary.EstimatedTotalUSD, 0.0001,
		"an unwritable candidate's estimate must not be in the total the user consents to")
}

func TestAnalyze_ProbesEachDirectoryOnce(t *testing.T) {
	episodes := &stubEpisodeFinder{episodes: []models.Episode{
		episodeRow("e1", 1, 1, "/tv/show/s01/e1.mkv", 45),
		episodeRow("e2", 1, 2, "/tv/show/s01/e2.mkv", 45),
		episodeRow("e3", 1, 3, "/tv/show/s01/e3.mkv", 45),
		episodeRow("e4", 2, 1, "/tv/show/s02/e1.mkv", 45),
	}}
	pred := &stubPredictor{probeRoute: RouteExtract}
	svc := NewGenerationCandidateService(nil, episodes, nil, pred, false, 0, nil)
	probes := map[string]int{}
	svc.probeWritable = func(_ context.Context, dir string) error { probes[dir]++; return nil }

	_, err := svc.Analyze(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, map[string]int{"/tv/show/s01": 1, "/tv/show/s02": 1}, probes,
		"a season shares one directory — one probe, not one per episode")
}

// ─── sub-6-8a AC #3: the quote is per model ────────────────────────────────

type stubCandidateCatalog struct {
	models  []ai.ModelInfo
	byDeflt string
}

func (s *stubCandidateCatalog) Available(context.Context) []ai.ModelInfo { return s.models }
func (s *stubCandidateCatalog) DefaultModel(context.Context) string      { return s.byDeflt }

func twoModelCatalog() *stubCandidateCatalog {
	return &stubCandidateCatalog{
		models: []ai.ModelInfo{
			{ID: "claude-haiku-4-5", Provider: ai.ProviderNameClaude},
			{ID: "claude-sonnet-5", Provider: ai.ProviderNameClaude},
		},
		byDeflt: "claude-sonnet-5",
	}
}

func TestAnalyze_PricesEveryReachableModelAndKeepsTheRowsOnTheDefault(t *testing.T) {
	movies := &stubMovieFinder{movies: []models.Movie{
		movieRow("m1", "A", "/m/a.mkv", 120, ""),
		movieRow("m2", "B", "/m/b.mkv", 60, ""),
	}}
	svc := NewGenerationCandidateService(movies, nil, nil, &stubPredictor{probeRoute: RouteExtract}, false, 0, nil)
	svc.SetModelCatalog(twoModelCatalog())

	res, err := svc.Analyze(context.Background(), nil)
	require.NoError(t, err)

	require.Contains(t, res.EstimatesByModel, "claude-sonnet-5")
	require.Contains(t, res.EstimatesByModel, "claude-haiku-4-5")

	sonnet := res.EstimatesByModel["claude-sonnet-5"]
	haiku := res.EstimatesByModel["claude-haiku-4-5"]

	assert.InDelta(t, sonnet.TotalUSD, res.Summary.EstimatedTotalUSD, 0.005,
		"the headline total must be the DEFAULT model's price — an old client that never learned about model choice still reads a true number")

	// eval-1 measured $0.48/hr on Sonnet against $0.18/hr on Haiku.
	assert.InDelta(t, 2.67, sonnet.TotalUSD/haiku.TotalUSD, 0.15,
		"the price gap the consent screen has to show is ~2.7×, not the 3× the rate card implies")

	// Per-row prices let the client re-price without re-running the sweep.
	require.Len(t, sonnet.PerCandidate, 2)
	var sum float64
	for _, usd := range haiku.PerCandidate {
		sum += usd
	}
	assert.InDelta(t, sum, haiku.TotalUSD, 0.005, "the rows must add up to the footer, per model")

	// A 120-minute film costs twice a 60-minute one on the same model.
	assert.InDelta(t, 2.0, sonnet.PerCandidate["m1"]/sonnet.PerCandidate["m2"], 0.05)
}

func TestAnalyze_OnlyTheTranslationHalfMovesWithTheModel(t *testing.T) {
	movies := &stubMovieFinder{movies: []models.Movie{movieRow("m1", "A", "/m/a.mkv", 100, "")}}
	svc := NewGenerationCandidateService(movies, nil, nil, &stubPredictor{probeRoute: RouteASR}, false, 0, nil)
	svc.SetModelCatalog(twoModelCatalog())

	res, err := svc.Analyze(context.Background(), nil)
	require.NoError(t, err)

	asrOnly := 100 * ai.HostedASRPerMinuteUSD()
	sonnet := res.EstimatesByModel["claude-sonnet-5"].TotalUSD
	haiku := res.EstimatesByModel["claude-haiku-4-5"].TotalUSD

	assert.Greater(t, sonnet, haiku)
	rateGap := translationRatePerMinute("claude-sonnet-5").
		Sub(translationRatePerMinute("claude-haiku-4-5")).
		Mul(decimal.NewFromInt(100)).InexactFloat64()
	assert.InDelta(t, sonnet-haiku, rateGap, 0.01,
		"speech recognition is billed by a different provider per audio minute — switching translation model must not move it")
	assert.Greater(t, haiku, asrOnly, "the ASR row still pays for translation on top")
}

func TestAnalyze_QuotesProcessingTimePerModel(t *testing.T) {
	movies := &stubMovieFinder{movies: []models.Movie{movieRow("m1", "A", "/m/a.mkv", 100, "")}}
	svc := NewGenerationCandidateService(movies, nil, nil, &stubPredictor{probeRoute: RouteExtract}, false, 0, nil)
	svc.SetModelCatalog(twoModelCatalog())

	res, err := svc.Analyze(context.Background(), nil)
	require.NoError(t, err)

	assert.EqualValues(t, 17, res.EstimatedMinutesByModel["claude-sonnet-5"], "eval-1: Sonnet takes ~17% of runtime")
	assert.EqualValues(t, 11, res.EstimatedMinutesByModel["claude-haiku-4-5"], "…Haiku ~11%")
}

func TestAnalyze_WithoutACatalogStillQuotesTheDefault(t *testing.T) {
	movies := &stubMovieFinder{movies: []models.Movie{movieRow("m1", "A", "/m/a.mkv", 90, "")}}
	svc := NewGenerationCandidateService(movies, nil, nil, &stubPredictor{probeRoute: RouteExtract}, false, 0, nil)

	res, err := svc.Analyze(context.Background(), nil)
	require.NoError(t, err)

	require.Len(t, res.EstimatesByModel, 1, "an unwired catalog must never leave the screen with no price at all")
	assert.Contains(t, res.EstimatesByModel, ai.DefaultClaudeModel)
}

func TestTranslationRate_IsMeasuredForEvaluatedModelsAndScaledOtherwise(t *testing.T) {
	// Rates are decimals now; compare as float64 at the assertion boundary so
	// the eval-1 arithmetic in these comments stays readable.
	haiku := translationRatePerMinute("claude-haiku-4-5").InexactFloat64()
	sonnet := translationRatePerMinute("claude-sonnet-5").InexactFloat64()

	// eval-1 billed $2.229 (Haiku) and $5.951 (Sonnet) for 12h20m = 740 min.
	assert.InDelta(t, 2.229/740, haiku, 0.0001)
	assert.InDelta(t, 5.951/740, sonnet, 0.0001)

	// The old flat 0.0004 under-quoted a 90-minute film by ~7×; that is the
	// bug this recalibration exists to remove.
	assert.Greater(t, haiku*90, 0.20, "a 90-minute film on the cheap model really does cost more than a quarter")

	// An unmeasured model is scaled from the SLOWER, dearer anchor.
	opus := translationRatePerMinute("claude-opus-4-8").InexactFloat64()
	assert.Greater(t, opus, sonnet, "Opus is priced above Sonnet, so its quote must be too")
	assert.InDelta(t, sonnet*(5.0+25.0)/(3.0+15.0), opus, 0.0001)

	unknown := translationRatePerMinute("some-future-model").InexactFloat64()
	assert.Equal(t, sonnet, unknown, "an unknown model quotes at the anchor rather than under-promising")
	assert.Equal(t, 0.17, translationTimeShare("some-future-model"))
}
