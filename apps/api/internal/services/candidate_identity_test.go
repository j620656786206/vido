package services

// Story sub-6-10a — row identity on the consent list.
//
// The critique that opened this story was a production screenshot: 2,399 rows,
// every one a grey square titled with a release filename and priced at 「片長
// 未知，以 45 分鐘估算」. Consent without recognition is not consent, and an
// estimate that ignores a number ffprobe already measured is a guess wearing
// an estimate's clothes. These tests are about both.

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
)

// ─── Fakes ─────────────────────────────────────────────────────────────────

// stubDurationPredictor implements BOTH RoutePredictor and the optional
// RouteDurationPredictor, so the type assertion in classify takes the
// duration-bearing branch.
type stubDurationPredictor struct {
	route    RoutePrediction
	seconds  float64
	err      error
	probed   []string
	probedWD []string
}

func (p *stubDurationPredictor) FromTracks([]SubtitleTrack) RoutePrediction { return p.route }

func (p *stubDurationPredictor) Probe(_ context.Context, path string) (RoutePrediction, error) {
	p.probed = append(p.probed, path)
	return p.route, p.err
}

func (p *stubDurationPredictor) ProbeWithDuration(_ context.Context, path string) (RoutePrediction, float64, error) {
	p.probedWD = append(p.probedWD, path)
	return p.route, p.seconds, p.err
}

type recordingDurationWriter struct {
	written map[string]int64
	err     error
}

func (w *recordingDurationWriter) UpdateDurationSeconds(_ context.Context, id string, seconds int64) error {
	if w.written == nil {
		w.written = map[string]int64{}
	}
	if w.err != nil {
		return w.err
	}
	w.written[id] = seconds
	return nil
}

// ─── AC #2: the three-rung runtime ladder ──────────────────────────────────

func TestRuntimeMinutes_PrefersContainerOverTMDbOverFallback(t *testing.T) {
	cases := []struct {
		name        string
		duration    int64 // seconds; 0 = column NULL
		tmdbRuntime int64 // minutes; 0 = column NULL
		wantMinutes float64
		wantKnown   bool
		wantSource  string
	}{
		{
			// 6,720s = 112 min. TMDb says 105 — the theatrical cut. The FILE is
			// what gets translated, so the file's length is what gets priced.
			name: "container wins over TMDb", duration: 6720, tmdbRuntime: 105,
			wantMinutes: 112, wantKnown: true, wantSource: RuntimeSourceFFprobe,
		},
		{
			name: "TMDb answers when nothing measured the file", tmdbRuntime: 105,
			wantMinutes: 105, wantKnown: true, wantSource: RuntimeSourceTMDb,
		},
		{
			name:        "neither — the 45-minute assumption, marked as such",
			wantMinutes: unknownRuntimeMinutes, wantKnown: false, wantSource: RuntimeSourceFallback,
		},
		{
			// A zero duration is ffprobe saying "no duration header", not
			// "zero-length film" — it must not price as free.
			name:     "zero container duration falls through, never prices as free",
			duration: 0, tmdbRuntime: 90,
			wantMinutes: 90, wantKnown: true, wantSource: RuntimeSourceTMDb,
		},
		{
			// Sub-minute precision survives: 90s is 1.5 min, not 1 or 2.
			name: "seconds are not rounded on the way in", duration: 90,
			wantMinutes: 1.5, wantKnown: true, wantSource: RuntimeSourceFFprobe,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var row candidateRow
			if tc.duration > 0 {
				row.durationSeconds = models.NewNullInt64(tc.duration)
			}
			if tc.tmdbRuntime > 0 {
				row.runtime = models.NewNullInt64(tc.tmdbRuntime)
			}

			minutes, known, source := row.runtimeMinutes()
			assert.InDelta(t, tc.wantMinutes, minutes, 1e-9)
			assert.Equal(t, tc.wantKnown, known)
			assert.Equal(t, tc.wantSource, source)
		})
	}
}

func TestAnalyze_ReportsRuntimeSourceOnEveryRow(t *testing.T) {
	m := movieRow("m1", "Dune", "/m/dune.mkv", 0, `[{"language":"eng","format":"subrip"}]`)
	m.DurationSeconds = models.NewNullInt64(9960) // 166 min
	movies := &stubMovieFinder{movies: []models.Movie{m}}
	svc := NewGenerationCandidateService(
		movies, nil, nil, &stubPredictor{fromTracks: RouteExtract}, false, 0, nil)

	res, err := svc.Analyze(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, res.Candidates, 1)
	assert.Equal(t, RuntimeSourceFFprobe, res.Candidates[0].RuntimeSource)
	assert.InDelta(t, 166, res.Candidates[0].RuntimeMinutes, 1e-9)
	assert.True(t, res.Candidates[0].RuntimeKnown)
}

// ─── AC #1: the probe's measurement is kept, not thrown away ───────────────

func TestAnalyze_EpisodeDurationFromProbeIsPricedAndPersisted(t *testing.T) {
	episodes := &stubEpisodeFinder{episodes: []models.Episode{episodeRow("e1", 1, 1, "/tv/s01e01.mkv", 0)}}
	pred := &stubDurationPredictor{route: RouteExtract, seconds: 3300} // 55 min
	writer := &recordingDurationWriter{}

	svc := NewGenerationCandidateService(nil, episodes, nil, pred, false, 0, nil)
	svc.SetEpisodeDurationWriter(writer)

	res, err := svc.Analyze(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, res.Candidates, 1)

	// Priced from the measurement IN THIS SWEEP — not next time.
	assert.InDelta(t, 55, res.Candidates[0].RuntimeMinutes, 1e-9)
	assert.Equal(t, RuntimeSourceFFprobe, res.Candidates[0].RuntimeSource)
	assert.True(t, res.Candidates[0].RuntimeKnown)
	// …and remembered, so the next sweep needs no probe to know it.
	assert.Equal(t, map[string]int64{"e1": 3300}, writer.written)
	assert.Equal(t, []string{"/tv/s01e01.mkv"}, pred.probedWD,
		"the duration-bearing probe is the one that ran — not a second probe")
	assert.Empty(t, pred.probed, "no second probe path")
}

func TestAnalyze_StoredEpisodeDurationIsNotRewritten(t *testing.T) {
	e := episodeRow("e1", 1, 1, "/tv/s01e01.mkv", 0)
	e.DurationSeconds = models.NewNullInt64(3300)
	episodes := &stubEpisodeFinder{episodes: []models.Episode{e}}
	// The probe reports something different; the stored value stands, because
	// re-writing it every sweep would be a write per episode per analysis.
	pred := &stubDurationPredictor{route: RouteExtract, seconds: 9999}
	writer := &recordingDurationWriter{}

	svc := NewGenerationCandidateService(nil, episodes, nil, pred, false, 0, nil)
	svc.SetEpisodeDurationWriter(writer)

	res, err := svc.Analyze(context.Background(), nil)
	require.NoError(t, err)
	assert.InDelta(t, 55, res.Candidates[0].RuntimeMinutes, 1e-9)
	assert.Empty(t, writer.written)
}

func TestAnalyze_DurationWriteFailureNeverFailsTheSweep(t *testing.T) {
	episodes := &stubEpisodeFinder{episodes: []models.Episode{episodeRow("e1", 1, 1, "/tv/s01e01.mkv", 0)}}
	pred := &stubDurationPredictor{route: RouteExtract, seconds: 3300}
	writer := &recordingDurationWriter{err: errDurationWriteFailed}

	svc := NewGenerationCandidateService(nil, episodes, nil, pred, false, 0, nil)
	svc.SetEpisodeDurationWriter(writer)

	res, err := svc.Analyze(context.Background(), nil)
	require.NoError(t, err, "one failed write must not deny the library its quote")
	// The quote in front of the user still used the measurement.
	assert.InDelta(t, 55, res.Candidates[0].RuntimeMinutes, 1e-9)
}

func TestAnalyze_MovieDurationIsNeverWrittenBySweep(t *testing.T) {
	// Movies get their duration from the enrichment pass that owns the rest of
	// their tech info. Two writers for one column is how a column ends up with
	// no owner.
	movies := &stubMovieFinder{movies: []models.Movie{movieRow("m1", "Dune", "/m/dune.mkv", 0, "")}}
	pred := &stubDurationPredictor{route: RouteASR, seconds: 9960}
	writer := &recordingDurationWriter{}

	svc := NewGenerationCandidateService(movies, nil, nil, pred, false, 0, nil)
	svc.SetEpisodeDurationWriter(writer)

	res, err := svc.Analyze(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, writer.written, "the episode writer must not be handed a movie id")
	// It is still PRICED from the measurement — only the persistence differs.
	assert.InDelta(t, 166, res.Candidates[0].RuntimeMinutes, 1e-9)
}

func TestAnalyze_PredictorWithoutDurationSupportStillWorks(t *testing.T) {
	// stubPredictor implements RoutePredictor only. The optional branch must
	// degrade to exactly the pre-sub-6-10a behaviour.
	episodes := &stubEpisodeFinder{episodes: []models.Episode{episodeRow("e1", 1, 1, "/tv/s01e01.mkv", 0)}}
	pred := &stubPredictor{probeRoute: RouteExtract}
	writer := &recordingDurationWriter{}

	svc := NewGenerationCandidateService(nil, episodes, nil, pred, false, 0, nil)
	svc.SetEpisodeDurationWriter(writer)

	res, err := svc.Analyze(context.Background(), nil)
	require.NoError(t, err)
	assert.InDelta(t, unknownRuntimeMinutes, res.Candidates[0].RuntimeMinutes, 1e-9)
	assert.Equal(t, RuntimeSourceFallback, res.Candidates[0].RuntimeSource)
	assert.Empty(t, writer.written)
}

// ─── AC #3: artwork ────────────────────────────────────────────────────────

func TestAnalyze_PosterPath_MovieOwnEpisodeInheritsSeries(t *testing.T) {
	m := movieRow("m1", "Dune", "/m/dune.mkv", 166, "")
	m.PosterPath = models.NewNullString("/poster-dune.jpg")
	movies := &stubMovieFinder{movies: []models.Movie{m}}

	e := episodeRow("e1", 1, 1, "/tv/s01e01.mkv", 45)
	e.SeriesID = "s1"
	episodes := &stubEpisodeFinder{episodes: []models.Episode{e}}
	series := &stubSeriesResolver{
		titles:  map[string]string{"s1": "Severance"},
		posters: map[string]string{"s1": "/poster-severance.jpg"},
	}

	svc := NewGenerationCandidateService(
		movies, episodes, series, &stubPredictor{probeRoute: RouteASR, fromTracks: RouteExtract}, false, 0, nil)
	res, err := svc.Analyze(context.Background(), nil)
	require.NoError(t, err)

	byID := map[string]GenerationCandidate{}
	for _, c := range res.Candidates {
		byID[c.MediaID] = c
	}
	assert.Equal(t, "/poster-dune.jpg", byID["m1"].PosterPath)
	assert.Equal(t, "/poster-severance.jpg", byID["e1"].PosterPath,
		"an episode's still is a frame grab — the series poster is its identity")
}

func TestAnalyze_SeriesLookedUpOncePerSeries(t *testing.T) {
	var eps []models.Episode
	for i := 1; i <= 12; i++ {
		e := episodeRow("e"+string(rune('a'+i)), 1, i, "/tv/ep.mkv", 45)
		e.SeriesID = "s1"
		eps = append(eps, e)
	}
	series := &stubSeriesResolver{
		titles:  map[string]string{"s1": "Severance"},
		posters: map[string]string{"s1": "/p.jpg"},
	}

	svc := NewGenerationCandidateService(
		nil, &stubEpisodeFinder{episodes: eps}, series, &stubPredictor{probeRoute: RouteExtract}, false, 0, nil)
	_, err := svc.Analyze(context.Background(), nil)
	require.NoError(t, err)

	assert.Equal(t, 1, series.calls,
		"widening the memo to carry the poster must not turn one read into twelve")
}

func TestAnalyze_MissingSeriesRowDegradesToNoPosterNoTitle(t *testing.T) {
	e := episodeRow("e1", 1, 1, "/tv/s01e01.mkv", 45)
	e.SeriesID = "gone"
	series := &stubSeriesResolver{titles: map[string]string{}}

	svc := NewGenerationCandidateService(
		nil, &stubEpisodeFinder{episodes: []models.Episode{e}}, series,
		&stubPredictor{probeRoute: RouteExtract}, false, 0, nil)
	res, err := svc.Analyze(context.Background(), nil)
	require.NoError(t, err)

	require.Len(t, res.Candidates, 1, "a broken series row must not drop the episode from the quote")
	assert.Empty(t, res.Candidates[0].PosterPath)
	assert.Empty(t, res.Candidates[0].SeriesTitle)
	assert.Equal(t, "gone", res.Candidates[0].SeriesID, "grouping still works on the id")
}

// ─── AC #4: an honest title ────────────────────────────────────────────────

func TestCandidateDisplayTitle(t *testing.T) {
	cases := []struct {
		name    string
		title   string
		path    string
		matched bool
		want    string
	}{
		{
			name:  "matched row keeps the TMDb title untouched",
			title: "沙丘", path: "/m/Dune.2021.2160p.WEB-DL.mkv", matched: true,
			want: "沙丘",
		},
		{
			// The eval-1 finding-12 shape, on the UI side this time.
			name:  "unmatched release name is re-read from the filename",
			title: "[bitsearch.to] Predator.Badlands.2025.2160p.WEB-DL",
			path:  "/m/[bitsearch.to] Predator.Badlands.2025.2160p.WEB-DL.mkv",
			want:  "Predator Badlands (2025)",
		},
		{
			// sub-6-7 CR M4, applied here: an unmatched row with a clean title
			// was named by the parser or typed by the user. Re-parsing the
			// filename would replace the better answer with a worse one.
			name:  "unmatched but clean title is left alone",
			title: "My Home Movie", path: "/m/My Home Movie.mkv",
			want: "My Home Movie",
		},
		{
			name:  "unparseable filename falls back to the raw title",
			title: "wat.wat.wat.2020.mkv", path: "",
			want: "wat.wat.wat.2020.mkv",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := candidateDisplayTitle(tc.title, tc.path, tc.matched)
			if tc.name == "unmatched release name is re-read from the filename" {
				// The parser's exact spacing is its own business; what this
				// story promises is "no release tokens, and the year is kept".
				assert.NotContains(t, got, "2160p")
				assert.NotContains(t, got, "bitsearch")
				assert.Contains(t, got, "2025")
				return
			}
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestAnalyze_TMDbMatchedAndDisplayTitleReachTheWire(t *testing.T) {
	matched := movieRow("m1", "沙丘", "/m/dune.mkv", 155, "")
	matched.TMDbID = models.NewNullInt64(438631)
	unmatched := movieRow("m2", "[bitsearch.to] Predator.Badlands.2025.2160p.WEB-DL",
		"/m/[bitsearch.to] Predator.Badlands.2025.2160p.WEB-DL.mkv", 0, "")

	svc := NewGenerationCandidateService(
		&stubMovieFinder{movies: []models.Movie{matched, unmatched}}, nil, nil,
		&stubPredictor{probeRoute: RouteASR}, false, 0, nil)
	res, err := svc.Analyze(context.Background(), nil)
	require.NoError(t, err)

	byID := map[string]GenerationCandidate{}
	for _, c := range res.Candidates {
		byID[c.MediaID] = c
	}

	assert.True(t, byID["m1"].TMDbMatched)
	assert.Equal(t, "沙丘", byID["m1"].DisplayTitle)

	assert.False(t, byID["m2"].TMDbMatched, "the UI marks this row unverified")
	assert.NotContains(t, byID["m2"].DisplayTitle, "2160p")
	// Title keeps its old meaning — an older client renders exactly what it
	// rendered before this story (sub-4-1 [@contract-v1]: add keys, never
	// redefine them).
	assert.Equal(t, unmatched.Title, byID["m2"].Title)
}

// errDurationWriteFailed is the sentinel the fail-soft test injects.
var errDurationWriteFailed = errors.New("duration write failed")
