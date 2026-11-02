package services

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/models"
)

// disc-2026-11-translate-estimate-ignores-cue-density: the translation of an
// English track is priced by how many cues it has when the container says so,
// and by runtime only when it does not.

// stubCuePredictor implements RoutePredictor and both optional halves.
type stubCuePredictor struct {
	route      RoutePrediction
	seconds    float64
	cues       int // what a probe reports
	trackCues  int // what CuesFromTracks reports
	err        error
	probedCues []string
}

func (p *stubCuePredictor) FromTracks([]SubtitleTrack) RoutePrediction { return p.route }
func (p *stubCuePredictor) Probe(context.Context, string) (RoutePrediction, error) {
	return p.route, p.err
}
func (p *stubCuePredictor) ProbeWithDuration(context.Context, string) (RoutePrediction, float64, error) {
	return p.route, p.seconds, p.err
}
func (p *stubCuePredictor) CuesFromTracks([]SubtitleTrack) int { return p.trackCues }
func (p *stubCuePredictor) ProbeWithCues(_ context.Context, path string) (RoutePrediction, float64, int, error) {
	p.probedCues = append(p.probedCues, path)
	return p.route, p.seconds, p.cues, p.err
}

// ─── AC #3: the rate tables ─────────────────────────────────────────────────

func TestTranslationRatePer1kCues(t *testing.T) {
	// Sonnet: production's 75th percentile (Bob 2026-11-02).
	assert.Equal(t, "0.45", translationRatePer1kCues("claude-sonnet-5").String())
	// Everything else moves by the factor Sonnet's measured per-cue figure
	// (See S01E04–06) moved to reach $0.45, rounded up.
	lift := 0.45 / ((0.159795 + 0.132408 + 0.193857) / 1461 * 1000)
	haiku := translationRatePer1kCues("claude-haiku-4-5").InexactFloat64()
	assert.InDelta(t, 2.229/10304*1000*lift, haiku, 0.001, "eval-1's Haiku per-cue cost, lifted")
	assert.Greater(t, haiku, 0.45/3, "never the bare price ratio, which sits below Haiku's own measurement")
	anchor := translationAnchorUSDPer1kCues.InexactFloat64()
	assert.InDelta(t, 5.951/10304*1000*lift, anchor, 0.001, "Sonnet at 10 cues per request (eval-1), lifted")

	// Unmeasured models: the anchor × the blended price ratio.
	assert.InDelta(t, anchor*(5.0+25.0)/(3.0+15.0), translationRatePer1kCues("claude-opus-4-8").InexactFloat64(), 1e-9)
	assert.Equal(t, anchor, translationRatePer1kCues("claude-sonnet-4-6").InexactFloat64(),
		"same price as Sonnet 5 but a 10-cue batch — never quoted at Sonnet 5's 50-cue rate")
	assert.Equal(t, anchor, translationRatePer1kCues("some-future-model").InexactFloat64())
	for _, m := range []string{"claude-opus-4-8", "claude-sonnet-4-6", "some-future-model"} {
		assert.True(t, translationRatePer1kCues(m).GreaterThan(translationRatePer1kCues("claude-sonnet-5")), m)
	}
}

func TestExtractTranslationUSD_CuesWhenKnownRuntimeOtherwise(t *testing.T) {
	assert.True(t, extractTranslationUSD(41, 1609, "claude-sonnet-5").Equal(decimal.RequireFromString("0.72405")))
	assert.True(t, extractTranslationUSD(41, 0, "claude-sonnet-5").Equal(decimal.RequireFromString("0.3485")),
		"no stated count: 41 min × $0.0085")
	// The ASR leg never takes the cue count — its cues are not known yet.
	assert.True(t, estimateUSDByCues(RouteASR, 60, 5000, decimal.Zero, "claude-sonnet-5").
		Equal(estimateUSD(RouteASR, 60, decimal.Zero, "claude-sonnet-5")))
	assert.True(t, estimateUSDByCues(RouteSkipped, 60, 5000, decimal.Zero, "claude-sonnet-5").IsZero())
}

// ─── AC #4: Scorpion and the 30 production runs ─────────────────────────────

func TestCueEstimate_ScorpionS01E02(t *testing.T) {
	// The English track's NUMBER_OF_FRAMES says 1,609; the run translated
	// 1,502 cues and billed $0.4859. The old runtime quote was $0.19.
	got := estimateUSDByCues(RouteExtract, 2464.0/60, 1609, decimal.Zero, "claude-sonnet-5")
	assert.Equal(t, "0.72", got.String())
	assert.True(t, got.GreaterThanOrEqual(decimal.RequireFromString("0.49")), "never below the bill")
	assert.Equal(t, "0.19", roundUSD(decimal.NewFromFloat(2464.0/60).Mul(decimal.RequireFromString("0.00462"))).String(),
		"the quote this story replaces")
}

// productionRun is one completed translate run on Sonnet 5 with no cache hits
// and more than 100 cues (production DB, 2026-11-02): cues translated, the
// bill, and the container duration in seconds (0 = not recorded).
type productionRun struct {
	title   string
	cues    int
	billed  float64
	seconds int
}

var productionRuns2026_11_02 = []productionRun{
	{"See S1E2", 568, 0.3433, 0}, {"See S1E3", 439, 0.3122, 0}, {"See S1E4", 474, 0.1598, 0},
	{"See S1E5", 464, 0.1324, 0}, {"See S1E6", 573, 0.1939, 0}, {"See S1E1", 487, 0.1647, 0},
	{"See S1E2 (rerun)", 568, 0.1974, 0},
	{"Lioness S2E2", 809, 0.2775, 2792}, {"Lioness S2E3", 869, 0.2927, 3077}, {"Lioness S2E4", 825, 0.3013, 2828},
	{"Lioness S2E5", 829, 0.282, 2746}, {"Lioness S2E6", 895, 0.3244, 2953}, {"Lioness S2E7", 1016, 0.354, 3369},
	{"Lioness S2E8", 760, 0.2528, 3056},
	{"Peacemaker S1E1", 814, 0.3544, 2806}, {"Peacemaker S1E2", 728, 0.3464, 2439}, {"Peacemaker S1E3", 637, 0.2917, 2373},
	{"Peacemaker S1E4", 637, 0.2708, 2808}, {"Peacemaker S1E5", 715, 0.3584, 2546}, {"Peacemaker S1E6", 707, 0.3189, 2813},
	{"Peacemaker S1E7", 707, 0.317, 2801}, {"Peacemaker S1E8", 597, 0.2606, 2678},
	{"Landman S2E2", 940, 0.3632, 3329}, {"Landman S2E3", 1043, 0.4021, 3089}, {"Landman S2E4", 941, 0.3388, 3798},
	{"Landman S2E5", 1196, 0.448, 3666}, {"Landman S2E6", 1018, 0.3703, 3237},
	{"Scorpion S1E1", 1504, 0.4401, 2488}, {"Scorpion S1E2", 1502, 0.4859, 2464}, {"Scorpion S1E3", 1549, 0.4388, 2516},
}

// Priced by the cues actually translated (a container's NUMBER_OF_FRAMES is
// usually a little higher, so the real quote errs higher still): 5 of 30
// under-quote — See S1E2/E3 (their first runs, before thinking was turned
// off) and Peacemaker S1E2/E3/E5 — by at most $0.11. The old runtime rate
// under-quoted all 23 runs it could price; the new fallback rate 3 of 23, the
// three Scorpion episodes the cue rate covers.
func TestCueEstimate_ProductionRuns(t *testing.T) {
	require.Len(t, productionRuns2026_11_02, 30)
	var underByCues, withDuration, underOld, underFallback []string
	for _, r := range productionRuns2026_11_02 {
		billed := decimal.NewFromFloat(r.billed)
		byCues := estimateUSDByCues(RouteExtract, 0, r.cues, decimal.Zero, "claude-sonnet-5")
		if byCues.LessThan(billed) {
			underByCues = append(underByCues, r.title)
			assert.True(t, billed.Sub(byCues).LessThanOrEqual(decimal.RequireFromString("0.12")), r.title)
		}
		if r.seconds == 0 {
			continue
		}
		withDuration = append(withDuration, r.title)
		mins := decimal.NewFromInt(int64(r.seconds)).Div(decimal.NewFromInt(60))
		if roundUSD(mins.Mul(decimal.RequireFromString("0.00462"))).LessThan(billed) {
			underOld = append(underOld, r.title)
		}
		if estimateUSD(RouteExtract, mins.InexactFloat64(), decimal.Zero, "claude-sonnet-5").LessThan(billed) {
			underFallback = append(underFallback, r.title)
		}
	}
	assert.Equal(t, []string{"See S1E2", "See S1E3", "Peacemaker S1E2", "Peacemaker S1E3", "Peacemaker S1E5"}, underByCues)
	assert.Len(t, withDuration, 23)
	assert.Len(t, underOld, 23, "the old runtime rate under-quoted every run it could price")
	assert.Equal(t, []string{"Scorpion S1E1", "Scorpion S1E2", "Scorpion S1E3"}, underFallback)
}

// ─── AC #2: the batch sweep (GenerationCandidateService) ────────────────────

func TestAnalyze_PricesExtractByCues(t *testing.T) {
	stored := episodeRow("e-stored", 1, 1, "/tv/s01e01.mkv", 41)
	stored.SubtitleTracks = models.NewNullString(`[{"language":"eng","format":"subrip","stream_index":3,"cue_count":1609}]`)
	probed := episodeRow("e-probed", 1, 2, "/tv/s01e02.mkv", 41)
	pred := &stubCuePredictor{route: RouteExtract, trackCues: 1609, cues: 1000}

	svc := NewGenerationCandidateService(nil, &stubEpisodeFinder{episodes: []models.Episode{stored, probed}}, nil, pred, false, 0, nil)
	res, err := svc.Analyze(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, res.Candidates, 2)
	byID := map[string]GenerationCandidate{}
	for _, c := range res.Candidates {
		byID[c.MediaID] = c
	}
	assert.Equal(t, 0.72, byID["e-stored"].EstimatedUSD, "stored tracks: 1,609 × $0.45/1k")
	assert.Equal(t, 0.45, byID["e-probed"].EstimatedUSD, "probe: 1,000 × $0.45/1k")
	assert.Equal(t, []string{"/tv/s01e02.mkv"}, pred.probedCues, "the stored row is not probed for its count")

	// Every model's total uses the same counts.
	sonnetTotal := res.EstimatesByModel["claude-sonnet-5"].TotalUSD
	assert.Equal(t, 1.17, sonnetTotal)
}

func TestAnalyze_NoStatedCountFallsBackToRuntime(t *testing.T) {
	e := episodeRow("e1", 1, 1, "/tv/s01e01.mkv", 60)
	pred := &stubCuePredictor{route: RouteExtract} // no count anywhere
	svc := NewGenerationCandidateService(nil, &stubEpisodeFinder{episodes: []models.Episode{e}}, nil, pred, false, 0, nil)
	res, err := svc.Analyze(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, res.Candidates, 1)
	assert.Equal(t, 0.51, res.Candidates[0].EstimatedUSD, "60 min × $0.0085")
}

// The cue count survives a route-cache hit: it is stored with the verdict.
func TestAnalyze_RouteCacheKeepsTheCueCount(t *testing.T) {
	e := episodeRow("e1", 1, 1, "/tv/s01e01.mkv", 41)
	cache := newFakeRouteCache()
	stat := &stubFileIdentity{sizes: map[string]int64{"/tv/s01e01.mkv": 1000}, mtimes: map[string]int64{"/tv/s01e01.mkv": 1700000000}}
	pred := &stubCuePredictor{route: RouteExtract, cues: 1609}
	svc := NewGenerationCandidateService(nil, &stubEpisodeFinder{episodes: []models.Episode{e}}, nil, pred, false, 0, nil)
	svc.SetRouteCache(cache)
	svc.fileIdentity = stat.identity

	first, err := svc.Analyze(context.Background(), nil)
	require.NoError(t, err)
	key := routeCacheKey(routeVersion, "e1", 1000, 1700000000)
	assert.Equal(t, "extract|1609", cache.sets[key])

	second, err := svc.Analyze(context.Background(), nil)
	require.NoError(t, err)
	assert.Len(t, pred.probedCues, 1, "the second sweep is served by the cache")
	assert.Equal(t, first.Candidates[0].EstimatedUSD, second.Candidates[0].EstimatedUSD)
	assert.Equal(t, 0.72, second.Candidates[0].EstimatedUSD)
}

func TestRouteVerdictEncoding(t *testing.T) {
	assert.Equal(t, "extract|1609", encodeRouteVerdict(routeVerdict{route: RouteExtract, cues: 1609}))
	assert.Equal(t, "asr", encodeRouteVerdict(routeVerdict{route: RouteASR}))
	for value, want := range map[string]routeVerdict{
		"extract|1609": {route: RouteExtract, cues: 1609},
		"extract":      {route: RouteExtract},
		"skip":         {route: RouteSkipped},
	} {
		got, ok := decodeRouteVerdict(value)
		assert.True(t, ok, value)
		assert.Equal(t, want, got, value)
	}
	for _, bad := range []string{"", "teleport", "teleport|5", "extract|", "extract|-3", "extract|lots"} {
		_, ok := decodeRouteVerdict(bad)
		assert.False(t, ok, bad)
	}
	assert.Equal(t, 2, routeVersion, "the stored value's shape changed — v1 entries are stranded")
}

// ─── AC #2: the single-item quote (TranscriptionEstimateService) ────────────

func cueEstimator(p *stubCuePredictor) *TranscriptionEstimateService {
	svc := newTestEstimator(&stubPlanSource{available: true, translate: true}, false)
	svc.SetDurationProber(p)
	svc.SetRoutePredictor(p)
	return svc
}

func TestTranscriptionEstimate_ExtractPricedByCues(t *testing.T) {
	t.Run("from the movie's stored tracks", func(t *testing.T) {
		p := &stubCuePredictor{route: RouteExtract, trackCues: 1609}
		target := sixtyMinuteMovie()
		target.SubtitleTracksJSON = engTrackJSON
		est := cueEstimator(p).Estimate(context.Background(), target)
		assert.Equal(t, TranscriptionPlanExtract, est.Plan)
		assert.Equal(t, 0.72, est.EstimatedUSD)
		assert.Empty(t, p.probedCues)
	})
	t.Run("from the ladder's probe (no stored duration)", func(t *testing.T) {
		p := &stubCuePredictor{route: RouteExtract, seconds: 2464, cues: 1609}
		target := TranscriptionEstimateTarget{MediaID: "e1", MediaType: models.SubtitleRunMediaEpisode, FilePath: "/tv/e1.mkv"}
		est := cueEstimator(p).Estimate(context.Background(), target)
		assert.Equal(t, 0.72, est.EstimatedUSD)
		assert.Len(t, p.probedCues, 1, "one probe gives route, length and count")
	})
	t.Run("from the route probe (duration stored, no tracks)", func(t *testing.T) {
		p := &stubCuePredictor{route: RouteExtract, cues: 1000}
		est := cueEstimator(p).Estimate(context.Background(), sixtyMinuteMovie())
		assert.Equal(t, 0.45, est.EstimatedUSD)
		assert.Len(t, p.probedCues, 1)
	})
	t.Run("no stated count: runtime", func(t *testing.T) {
		p := &stubCuePredictor{route: RouteExtract}
		est := cueEstimator(p).Estimate(context.Background(), sixtyMinuteMovie())
		assert.Equal(t, 0.51, est.EstimatedUSD, "60 min × $0.0085")
	})
	t.Run("speech recognition ignores the count", func(t *testing.T) {
		p := &stubCuePredictor{route: RouteASR, cues: 1609}
		est := cueEstimator(p).Estimate(context.Background(), sixtyMinuteMovie())
		want := estimateUSD(RouteASR, 60, ai.EstimatedASRPerMinute(false), "claude-sonnet-5")
		assert.True(t, usdOf(t, est).Equal(want))
	})
}

func TestTranscriptionEstimate_AlignedSidecarPricedByItsCues(t *testing.T) {
	p := &stubCuePredictor{route: RouteASR}
	svc := cueEstimator(p)
	svc.SetSidecarSource(&stubSidecars{answer: SidecarAlignment{Status: SidecarStatusAligned, FileName: "movie.en.srt"}})
	var read []string
	srt := ""
	for i := 1; i <= 1609; i++ {
		srt += "1\n00:00:01,000 --> 00:00:02,000\nLine\n\n"
	}
	svc.readFile = func(path string) ([]byte, error) {
		read = append(read, path)
		return []byte(srt), nil
	}
	est := svc.Estimate(context.Background(), sixtyMinuteMovie())
	assert.Equal(t, TranscriptionPlanSidecar, est.Plan)
	assert.Equal(t, 0.72, est.EstimatedUSD)
	assert.Equal(t, []string{"/media/movie.en.srt"}, read, "the file beside the video")

	svc.readFile = func(string) ([]byte, error) { return nil, errors.New("gone") }
	est = svc.Estimate(context.Background(), sixtyMinuteMovie())
	assert.Equal(t, 0.51, est.EstimatedUSD, "unreadable sidecar: runtime")
}

func TestReadSidecarCapped(t *testing.T) {
	_, err := readSidecarCapped("/definitely/not/here.srt")
	assert.Error(t, err)
}

// ─── AC #1: ffprobe states the count; old rows stay readable ────────────────

func TestStreamCueCount(t *testing.T) {
	for name, tc := range map[string]struct {
		stream ffprobeStream
		want   int
	}{
		"mkvmerge statistics tag": {ffprobeStream{Tags: map[string]string{"NUMBER_OF_FRAMES": "1609"}}, 1609},
		"language-suffixed tag":   {ffprobeStream{Tags: map[string]string{"NUMBER_OF_FRAMES-eng": "1502"}}, 1502},
		"MP4 nb_frames":           {ffprobeStream{NbFrames: "812"}, 812},
		"tag wins over nb_frames": {ffprobeStream{Tags: map[string]string{"NUMBER_OF_FRAMES": "10"}, NbFrames: "20"}, 10},
		// Read, nothing stated: -1, so the backfill never reads it again.
		"nothing stated":               {ffprobeStream{Tags: map[string]string{"title": "English"}}, CueCountUnstated},
		"garbage":                      {ffprobeStream{Tags: map[string]string{"NUMBER_OF_FRAMES": "many"}, NbFrames: "N/A"}, CueCountUnstated},
		"zero is not a count":          {ffprobeStream{NbFrames: "0"}, CueCountUnstated},
		"lower-case key (other muxer)": {ffprobeStream{Tags: map[string]string{"number_of_frames": "77"}}, 77},
	} {
		t.Run(name, func(t *testing.T) { assert.Equal(t, tc.want, streamCueCount(tc.stream)) })
	}
}

func TestParseFfprobeJSON_SubtitleCueCount(t *testing.T) {
	out := []byte(`{"streams":[
		{"index":3,"codec_type":"subtitle","codec_name":"subrip","tags":{"language":"eng","NUMBER_OF_FRAMES-eng":"1609"}},
		{"index":4,"codec_type":"subtitle","codec_name":"mov_text","nb_frames":"812","tags":{"language":"eng"}},
		{"index":5,"codec_type":"subtitle","codec_name":"hdmv_pgs_subtitle","tags":{"language":"chi"}}
	],"format":{"duration":"2464.0"}}`)
	info, err := parseFfprobeJSON(out)
	require.NoError(t, err)
	require.Len(t, info.SubtitleTracks, 3)
	assert.Equal(t, 1609, info.SubtitleTracks[0].CueCount)
	assert.Equal(t, 812, info.SubtitleTracks[1].CueCount)
	assert.Equal(t, CueCountUnstated, info.SubtitleTracks[2].CueCount, "read, nothing stated")
}

func TestSubtitleTrackJSON_CueCountIsAdditive(t *testing.T) {
	var old []SubtitleTrack
	require.NoError(t, json.Unmarshal([]byte(`[{"language":"eng","format":"subrip","external":false,"stream_index":2}]`), &old))
	assert.Zero(t, old[0].CueCount, "a row stored before the field reads as 'not stated'")
	raw, err := json.Marshal(old)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "cue_count", "an unstated count adds nothing to the stored JSON")
	raw, err = json.Marshal([]SubtitleTrack{{Language: "eng", Format: "subrip", CueCount: 1609}})
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"cue_count":1609`)
}
