package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/ai"
)

// ─── disc-2026-10-generate-keeps-user-en-srt-b2: the English sidecar quote ──

type stubSidecars struct {
	answer SidecarAlignment
	peeked []string
}

func (s *stubSidecars) Peek(_ context.Context, mediaPath string) SidecarAlignment {
	s.peeked = append(s.peeked, mediaPath)
	return s.answer
}

func sidecarEstimator(plan *stubPlanSource, route RoutePrediction, sc *stubSidecars) *TranscriptionEstimateService {
	svc := routedEstimator(plan, &stubDurationPredictor{route: route})
	svc.SetSidecarSource(sc)
	return svc
}

func TestTranscriptionEstimate_AlignedSidecarIsPricedAsTranslation(t *testing.T) {
	sc := &stubSidecars{answer: SidecarAlignment{Status: SidecarStatusAligned, FileName: "Movie.en.srt"}}
	est := sidecarEstimator(&stubPlanSource{available: true, translate: true}, RouteASR, sc).
		Estimate(context.Background(), sixtyMinuteMovie())

	assert.Equal(t, TranscriptionPlanSidecar, est.Plan)
	assert.Equal(t, RouteSidecarPrediction, est.Route)
	require.NotNil(t, est.SidecarCheck)
	assert.Equal(t, SidecarStatusAligned, est.SidecarCheck.Status)
	assert.Equal(t, "Movie.en.srt", est.SidecarCheck.FileName)
	want := estimateUSD(RouteExtract, 60, ai.EstimatedASRPerMinute(false), "claude-sonnet-5")
	assert.True(t, usdOf(t, est).Equal(want), "the embedded-English lane's price: got %v want %v", est.EstimatedUSD, want)
	assert.Less(t, est.EstimatedUSD, 1.04, "cheaper than speech recognition + translation")
}

func TestTranscriptionEstimate_SidecarNotYetAlignedKeepsTheSpeechRecognitionQuote(t *testing.T) {
	for _, status := range []string{SidecarStatusChecking, SidecarStatusOffset, SidecarStatusMismatch,
		SidecarStatusUnusable, SidecarStatusUnknown, SidecarStatusNone} {
		sc := &stubSidecars{answer: SidecarAlignment{Status: status, OffsetSeconds: 3.1}}
		est := sidecarEstimator(&stubPlanSource{available: true, translate: true}, RouteASR, sc).
			Estimate(context.Background(), sixtyMinuteMovie())

		assert.Equal(t, TranscriptionPlanFull, est.Plan, status)
		assert.Equal(t, "asr", est.Route, status)
		assert.Equal(t, 1.04, est.EstimatedUSD, "%s: never quote less than the run may cost", status)
		require.NotNil(t, est.SidecarCheck, status)
		assert.Equal(t, status, est.SidecarCheck.Status)
	}
}

func TestTranscriptionEstimate_SkipRouteAlsoAsksTheSidecar(t *testing.T) {
	sc := &stubSidecars{answer: SidecarAlignment{Status: SidecarStatusAligned}}
	est := sidecarEstimator(&stubPlanSource{available: true, translate: true}, RouteSkipped, sc).
		Estimate(context.Background(), sixtyMinuteMovie())
	assert.Equal(t, TranscriptionPlanSidecar, est.Plan, "an und-only file is rescued at run time too")
}

func TestTranscriptionEstimate_EmbeddedTrackNeverAsksTheSidecar(t *testing.T) {
	sc := &stubSidecars{answer: SidecarAlignment{Status: SidecarStatusAligned}}
	target := sixtyMinuteMovie()
	target.SubtitleTracksJSON = engTrackJSON
	est := sidecarEstimator(&stubPlanSource{available: true, translate: true}, RouteExtract, sc).
		Estimate(context.Background(), target)

	assert.Equal(t, TranscriptionPlanExtract, est.Plan)
	assert.Nil(t, est.SidecarCheck)
	assert.Empty(t, sc.peeked)
}

func TestTranscriptionEstimate_AlignedSidecarWinsOverAResume(t *testing.T) {
	// The run routes to the sidecar before the speech-recognition fallback
	// that would resume, so the quote must too.
	sc := &stubSidecars{answer: SidecarAlignment{Status: SidecarStatusAligned}}
	est := sidecarEstimator(&stubPlanSource{available: true, translate: true, resumable: true}, RouteASR, sc).
		Estimate(context.Background(), sixtyMinuteMovie())
	assert.Equal(t, TranscriptionPlanSidecar, est.Plan)
}

func TestTranscriptionEstimate_AlignedSidecarWithoutTranslationKeyCostsNothing(t *testing.T) {
	sc := &stubSidecars{answer: SidecarAlignment{Status: SidecarStatusAligned}}
	est := sidecarEstimator(&stubPlanSource{available: true, translate: false}, RouteASR, sc).
		Estimate(context.Background(), sixtyMinuteMovie())
	assert.Equal(t, TranscriptionPlanSidecar, est.Plan)
	assert.Zero(t, est.EstimatedUSD)
}

func TestTranscriptionEstimate_LegacyModeNeverAsksTheSidecar(t *testing.T) {
	sc := &stubSidecars{answer: SidecarAlignment{Status: SidecarStatusAligned}}
	svc := newTestEstimator(&stubPlanSource{available: true, translate: true}, false)
	svc.SetSidecarSource(sc)

	est := svc.Estimate(context.Background(), sixtyMinuteMovie())

	assert.Equal(t, TranscriptionPlanFull, est.Plan, "legacy mode never routes through the pipeline")
	assert.Nil(t, est.SidecarCheck)
	assert.Empty(t, sc.peeked)
}
