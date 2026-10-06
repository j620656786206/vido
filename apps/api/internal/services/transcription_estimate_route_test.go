package services

// disc-2026-10-single-generate-ignores-embedded-english-a — in pipeline mode
// the click routes like the batch, so the quote must say which lane the file
// will take and price that lane (dsr-6a AC #2 [@contract-v2]).

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/models"
)

const engTrackJSON = `[{"language":"eng","format":"subrip","external":false,"stream_index":2}]`

func routedEstimator(plan *stubPlanSource, p *stubDurationPredictor) *TranscriptionEstimateService {
	svc := newTestEstimator(plan, false)
	svc.SetDurationProber(p)
	svc.SetRoutePredictor(p)
	return svc
}

func TestTranscriptionEstimate_LegacyModeHasNoRoute(t *testing.T) {
	// No route predictor wired = legacy mode: the quote is exactly what it was.
	svc := newTestEstimator(&stubPlanSource{available: true, translate: true}, false)
	svc.SetDurationProber(&stubDurationPredictor{route: RouteExtract, seconds: 3600})
	target := sixtyMinuteMovie()
	target.SubtitleTracksJSON = engTrackJSON

	est := svc.Estimate(context.Background(), target)

	assert.Empty(t, est.Route)
	assert.Equal(t, TranscriptionPlanFull, est.Plan)
	assert.Equal(t, 1.66, est.EstimatedUSD, "the ASR + translation quote, as before")
}

func TestTranscriptionEstimate_PersistedTracksDecideTheRouteWithoutProbing(t *testing.T) {
	p := &stubDurationPredictor{route: RouteExtract}
	target := sixtyMinuteMovie() // stored duration → the ladder itself never probes
	target.SubtitleTracksJSON = engTrackJSON

	est := routedEstimator(&stubPlanSource{available: true, translate: true}, p).Estimate(context.Background(), target)

	assert.Equal(t, "extract", est.Route)
	assert.Equal(t, TranscriptionPlanExtract, est.Plan)
	assert.Empty(t, p.probed, "a persisted track list is enough — no ffprobe")
	assert.Empty(t, p.probedWD)
	want := estimateUSD(RouteExtract, 60, ai.EstimatedASRPerMinute(false), "claude-sonnet-5")
	assert.True(t, usdOf(t, est).Equal(want), "extract is priced by the candidate list's own function: got %v want %v", est.EstimatedUSD, want)
	assert.Less(t, est.EstimatedUSD, 1.66, "no speech recognition is billed on the extract lane")
}

func TestTranscriptionEstimate_NoTracksAndStoredDurationProbesOnceForTheRoute(t *testing.T) {
	p := &stubDurationPredictor{route: RouteASR}
	target := sixtyMinuteMovie() // stored duration, no tracks JSON

	est := routedEstimator(&stubPlanSource{available: true, translate: true}, p).Estimate(context.Background(), target)

	assert.Equal(t, "asr", est.Route)
	assert.Equal(t, TranscriptionPlanFull, est.Plan)
	assert.Equal(t, 1.66, est.EstimatedUSD)
	assert.Len(t, p.probed, 1, "one probe, for the route only")
	assert.Empty(t, p.probedWD, "the duration was stored — the ladder did not probe")
}

func TestTranscriptionEstimate_UnstoredDurationProbeAlsoYieldsTheRoute(t *testing.T) {
	p := &stubDurationPredictor{route: RouteExtract, seconds: 2880}
	target := TranscriptionEstimateTarget{MediaID: "ep-1", MediaType: models.SubtitleRunMediaEpisode, FilePath: "/tv/s01e02.mkv"}

	est := routedEstimator(&stubPlanSource{available: true, translate: true}, p).Estimate(context.Background(), target)

	assert.Equal(t, "extract", est.Route)
	assert.Equal(t, TranscriptionPlanExtract, est.Plan)
	assert.Equal(t, 48.0, est.RuntimeMinutes)
	assert.Len(t, p.probedWD, 1, "the ladder's probe is reused for the route")
	assert.Empty(t, p.probed, "no second probe")
}

func TestTranscriptionEstimate_SkipRouteIsPricedLikeSpeechRecognition(t *testing.T) {
	// `und`-tagged text track: the click sends it down the ASR leg
	// (TranscribeWhenSkipped), so that is what it costs.
	p := &stubDurationPredictor{route: RouteSkipped}

	est := routedEstimator(&stubPlanSource{available: true, translate: true}, p).Estimate(context.Background(), sixtyMinuteMovie())

	assert.Equal(t, "skip", est.Route)
	assert.Equal(t, TranscriptionPlanFull, est.Plan)
	assert.Equal(t, 1.66, est.EstimatedUSD)
}

func TestTranscriptionEstimate_ResumeOnlyAppliesOnTheSpeechRecognitionLane(t *testing.T) {
	// An `untranslated` row with its .en.srt resumes translate-only ONLY when
	// the click would reach the ASR leg; a file with an embedded English
	// track is translated from that track instead.
	asr := routedEstimator(&stubPlanSource{available: true, translate: true, resumable: true},
		&stubDurationPredictor{route: RouteASR}).Estimate(context.Background(), sixtyMinuteMovie())
	assert.Equal(t, TranscriptionPlanTranslateOnly, asr.Plan)
	assert.Equal(t, "asr", asr.Route)

	target := sixtyMinuteMovie()
	target.SubtitleTracksJSON = engTrackJSON
	extract := routedEstimator(&stubPlanSource{available: true, translate: true, resumable: true},
		&stubDurationPredictor{route: RouteExtract}).Estimate(context.Background(), target)
	assert.Equal(t, TranscriptionPlanExtract, extract.Plan)
	assert.Equal(t, "extract", extract.Route)
}

func TestTranscriptionEstimate_ExtractWithoutTranslationKeyCostsNothing(t *testing.T) {
	// Deliver / convert are free; a translate verdict would fail before any
	// call. Either way nothing is billed — the dialog shows the degraded line.
	target := sixtyMinuteMovie()
	target.SubtitleTracksJSON = engTrackJSON
	est := routedEstimator(&stubPlanSource{available: true, translate: false},
		&stubDurationPredictor{route: RouteExtract}).Estimate(context.Background(), target)

	assert.Equal(t, TranscriptionPlanExtract, est.Plan)
	assert.False(t, est.TranslationConfigured)
	assert.Equal(t, 0.0, est.EstimatedUSD)
}

func TestTranscriptionEstimate_RouteProbeFailureLeavesRouteUnknown(t *testing.T) {
	p := &stubDurationPredictor{err: assert.AnError}
	est := routedEstimator(&stubPlanSource{available: true, translate: true}, p).Estimate(context.Background(), sixtyMinuteMovie())

	assert.Empty(t, est.Route, "unknown is reported as absent, never guessed")
	assert.Equal(t, TranscriptionPlanFull, est.Plan, "and the quote stays the safe (higher) one")
	assert.Equal(t, 1.66, est.EstimatedUSD)
}

func TestTranscriptionEstimate_MalformedTracksJSONFallsBackToTheProbe(t *testing.T) {
	p := &stubDurationPredictor{route: RouteASR}
	target := sixtyMinuteMovie()
	target.SubtitleTracksJSON = "{not json"

	est := routedEstimator(&stubPlanSource{available: true, translate: true}, p).Estimate(context.Background(), target)

	require.Len(t, p.probed, 1)
	assert.Equal(t, "asr", est.Route)
}

func TestTranscriptionEstimate_AFailedLadderProbeIsNotRepeatedForTheRoute(t *testing.T) {
	// No stored duration, disk asleep: the ladder's probe times out. The
	// route probe must not make the dialog wait out a second timeout.
	p := &stubDurationPredictor{err: assert.AnError}
	target := TranscriptionEstimateTarget{MediaID: "ep-1", MediaType: models.SubtitleRunMediaEpisode, FilePath: "/tv/s01e02.mkv", Runtime: models.NewNullInt64(50)}

	est := routedEstimator(&stubPlanSource{available: true, translate: true}, p).Estimate(context.Background(), target)

	assert.Len(t, p.probedWD, 1)
	assert.Empty(t, p.probed, "one touch of the file per quote, success or failure")
	assert.Empty(t, est.Route)
	assert.Equal(t, RuntimeSourceTMDb, est.RuntimeSource)
}
