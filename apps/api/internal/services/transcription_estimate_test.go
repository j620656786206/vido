package services

// Story dsr-6a AC #3 — the price on the 管理字幕 dialog's paid buttons.
//
// The single-item button does NOT take the candidate list's routes: it always
// runs speech recognition + translation, except an `untranslated` row whose
// English SRT is still on disk, which resumes translate-only. Every test here
// pins "the quote prices what the click will actually do".

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/models"
)

type stubPlanSource struct {
	available bool
	translate bool
	resumable bool
	resumeAsk []string
}

func (s *stubPlanSource) IsAvailable() bool        { return s.available }
func (s *stubPlanSource) translationEnabled() bool { return s.translate }
func (s *stubPlanSource) canResumeTranslateOnly(_ context.Context, mediaType, mediaID string) bool {
	s.resumeAsk = append(s.resumeAsk, mediaType+":"+mediaID)
	return s.resumable
}

// sixtyMinuteMovie has a measured duration, so no probe runs.
func sixtyMinuteMovie() TranscriptionEstimateTarget {
	return TranscriptionEstimateTarget{
		MediaID:         "movie-1",
		MediaType:       models.SubtitleRunMediaMovie,
		FilePath:        "/media/movie.mkv",
		DurationSeconds: models.NewNullInt64(3600),
	}
}

func newTestEstimator(plan *stubPlanSource, selfHosted bool) *TranscriptionEstimateService {
	return NewTranscriptionEstimateService(plan, func() string { return "claude-sonnet-5" }, selfHosted, nil)
}

func usdOf(t *testing.T, e TranscriptionEstimate) decimal.Decimal {
	t.Helper()
	return decimal.NewFromFloat(e.EstimatedUSD)
}

func TestTranscriptionEstimate_FullRunPaysSpeechRecognitionAndTranslation(t *testing.T) {
	est := newTestEstimator(&stubPlanSource{available: true, translate: true}, false).
		Estimate(context.Background(), sixtyMinuteMovie())

	want := estimateUSD(RouteASR, 60, ai.EstimatedASRPerMinute(false), "claude-sonnet-5")
	assert.Equal(t, TranscriptionPlanFull, est.Plan)
	assert.True(t, usdOf(t, est).Equal(want), "got %v want %v", est.EstimatedUSD, want)
	// 60 × (0.006 + 0.00804) = 0.8424 → $0.84
	assert.Equal(t, 0.84, est.EstimatedUSD)
	assert.Equal(t, "claude-sonnet-5", est.ModelID)
	assert.True(t, est.TranslationConfigured)
	assert.True(t, est.ASRAvailable)
	assert.False(t, est.SelfHostedASR)
}

func TestTranscriptionEstimate_WithoutTranslationKeyOnlySpeechRecognitionIsCharged(t *testing.T) {
	est := newTestEstimator(&stubPlanSource{available: true, translate: false}, false).
		Estimate(context.Background(), sixtyMinuteMovie())

	// The run writes the English SRT and stops (translateAndPersist skips the
	// translate leg) — quoting translation would overcharge.
	assert.Equal(t, TranscriptionPlanFull, est.Plan)
	assert.Equal(t, 0.36, est.EstimatedUSD) // 60 × 0.006
	assert.Empty(t, est.ModelID, "no translation leg means no model is billed")
	assert.False(t, est.TranslationConfigured)
}

func TestTranscriptionEstimate_ResumableRowOnlyPaysTranslation(t *testing.T) {
	plan := &stubPlanSource{available: true, translate: true, resumable: true}
	est := newTestEstimator(plan, false).Estimate(context.Background(), sixtyMinuteMovie())

	want := estimateUSD(RouteExtract, 60, ai.EstimatedASRPerMinute(false), "claude-sonnet-5")
	assert.Equal(t, TranscriptionPlanTranslateOnly, est.Plan)
	assert.True(t, usdOf(t, est).Equal(want))
	assert.Equal(t, 0.48, est.EstimatedUSD) // 60 × 0.00804 = 0.4824
	assert.Equal(t, []string{"movie:movie-1"}, plan.resumeAsk,
		"the resume check must be asked for THIS media type — the movie variant on an episode id is always false")
}

func TestTranscriptionEstimate_ResumableRowWithoutTranslationKeyCostsNothing(t *testing.T) {
	est := newTestEstimator(&stubPlanSource{available: true, translate: false, resumable: true}, false).
		Estimate(context.Background(), sixtyMinuteMovie())

	assert.Equal(t, TranscriptionPlanTranslateOnly, est.Plan)
	assert.Equal(t, 0.0, est.EstimatedUSD)
	assert.False(t, est.TranslationConfigured)
}

func TestTranscriptionEstimate_SelfHostedSpeechRecognitionIsNotBilled(t *testing.T) {
	withTranslation := newTestEstimator(&stubPlanSource{available: true, translate: true}, true).
		Estimate(context.Background(), sixtyMinuteMovie())
	assert.Equal(t, 0.48, withTranslation.EstimatedUSD, "self-hosted ASR still pays the translation half")
	assert.True(t, withTranslation.SelfHostedASR)

	withoutTranslation := newTestEstimator(&stubPlanSource{available: true, translate: false}, true).
		Estimate(context.Background(), sixtyMinuteMovie())
	assert.Equal(t, 0.0, withoutTranslation.EstimatedUSD)
}

func TestTranscriptionEstimate_FullRunMatchesTheCandidateListASRPrice(t *testing.T) {
	// One rate table, two callers: if either side ever copies a constant, the
	// dialog and the consent list will quote the same file differently.
	for _, minutes := range []float64{1, 23.5, 45, 97.25, 180} {
		target := sixtyMinuteMovie()
		target.DurationSeconds = models.NewNullInt64(int64(minutes * 60))
		est := newTestEstimator(&stubPlanSource{available: true, translate: true}, false).
			Estimate(context.Background(), target)
		want := estimateUSD(RouteASR, minutes, ai.EstimatedASRPerMinute(false), "claude-sonnet-5")
		assert.True(t, usdOf(t, est).Equal(want), "%v min: got %v want %v", minutes, est.EstimatedUSD, want)
	}
}

func TestTranscriptionEstimate_UsesTheModelTheRunWillBill(t *testing.T) {
	// Haiku's measured rate (0.00301) differs from Sonnet's (0.00804). An id
	// outside the catalog would NOT tell the two apart — it is priced at the
	// Sonnet anchor too.
	est := NewTranscriptionEstimateService(
		&stubPlanSource{available: true, translate: true},
		func() string { return "claude-haiku-4-5" }, false, nil,
	).Estimate(context.Background(), sixtyMinuteMovie())

	assert.Equal(t, "claude-haiku-4-5", est.ModelID)
	// 60 × (0.006 + 0.00301) = 0.5406 → $0.54
	assert.Equal(t, 0.54, est.EstimatedUSD)
}

func TestTranscriptionEstimate_DurationLadder(t *testing.T) {
	cases := []struct {
		name       string
		stored     int64 // duration_seconds; 0 = NULL
		tmdb       int64 // runtime minutes; 0 = NULL
		probe      *stubDurationPredictor
		wantMin    float64
		wantKnown  bool
		wantSource string
		wantProbed bool
	}{
		{"stored measurement wins and skips the probe", 5400, 100, &stubDurationPredictor{seconds: 60}, 90, true, RuntimeSourceFFprobe, false},
		{"no stored measurement → live probe", 0, 100, &stubDurationPredictor{seconds: 2880}, 48, true, RuntimeSourceFFprobe, true},
		{"probe failure falls through to TMDb", 0, 100, &stubDurationPredictor{err: errors.New("ffprobe: timeout")}, 100, true, RuntimeSourceTMDb, true},
		{"probe without a duration header falls through to TMDb", 0, 100, &stubDurationPredictor{seconds: 0}, 100, true, RuntimeSourceTMDb, true},
		{"nothing known → stated 45-minute assumption", 0, 0, &stubDurationPredictor{err: errors.New("no file")}, unknownRuntimeMinutes, false, RuntimeSourceFallback, true},
		{"no prober wired → TMDb", 0, 100, nil, 100, true, RuntimeSourceTMDb, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			target := sixtyMinuteMovie()
			target.DurationSeconds = models.NullInt64{}
			if tc.stored > 0 {
				target.DurationSeconds = models.NewNullInt64(tc.stored)
			}
			if tc.tmdb > 0 {
				target.Runtime = models.NewNullInt64(tc.tmdb)
			}
			svc := newTestEstimator(&stubPlanSource{available: true, translate: true}, false)
			if tc.probe != nil {
				svc.SetDurationProber(tc.probe)
			}

			est := svc.Estimate(context.Background(), target)

			assert.Equal(t, tc.wantMin, est.RuntimeMinutes)
			assert.Equal(t, tc.wantKnown, est.RuntimeKnown)
			assert.Equal(t, tc.wantSource, est.RuntimeSource)
			if tc.probe != nil {
				assert.Equal(t, tc.wantProbed, len(tc.probe.probedWD) == 1)
			}
		})
	}
}

func TestTranscriptionEstimate_OnlyEpisodesRememberAProbedDuration(t *testing.T) {
	writer := &recordingDurationWriter{}

	episode := TranscriptionEstimateTarget{MediaID: "ep-1", MediaType: models.SubtitleRunMediaEpisode, FilePath: "/tv/s01e01.mkv"}
	svc := newTestEstimator(&stubPlanSource{available: true, translate: true}, false)
	svc.SetDurationProber(&stubDurationPredictor{seconds: 2880.9})
	svc.SetEpisodeDurationWriter(writer)
	est := svc.Estimate(context.Background(), episode)
	assert.Equal(t, 48.0, est.RuntimeMinutes, "seconds are truncated the way the sweep stores them")
	assert.Equal(t, map[string]int64{"ep-1": 2880}, writer.written)

	movieWriter := &recordingDurationWriter{}
	movie := sixtyMinuteMovie()
	movie.DurationSeconds = models.NullInt64{}
	svc = newTestEstimator(&stubPlanSource{available: true, translate: true}, false)
	svc.SetDurationProber(&stubDurationPredictor{seconds: 3600})
	svc.SetEpisodeDurationWriter(movieWriter)
	svc.Estimate(context.Background(), movie)
	assert.Empty(t, movieWriter.written, "a movie's duration belongs to enrichment, not to the estimate")
}

func TestTranscriptionEstimate_WriteBackFailureDoesNotChangeTheQuote(t *testing.T) {
	svc := newTestEstimator(&stubPlanSource{available: true, translate: true}, false)
	svc.SetDurationProber(&stubDurationPredictor{seconds: 2880})
	svc.SetEpisodeDurationWriter(&recordingDurationWriter{err: errors.New("database is locked")})

	est := svc.Estimate(context.Background(), TranscriptionEstimateTarget{
		MediaID: "ep-1", MediaType: models.SubtitleRunMediaEpisode, FilePath: "/tv/s01e01.mkv",
	})
	assert.Equal(t, 48.0, est.RuntimeMinutes)
	assert.True(t, est.RuntimeKnown)
}

func TestTranscriptionEstimate_ReportsAvailabilityWithoutBlockingThePrice(t *testing.T) {
	est := newTestEstimator(&stubPlanSource{available: false, translate: true}, false).
		Estimate(context.Background(), sixtyMinuteMovie())
	assert.False(t, est.ASRAvailable)
	assert.Equal(t, 0.84, est.EstimatedUSD, "the frontend disables the button; the number stays honest")
	assert.Equal(t, "movie-1", est.MediaID)
	assert.Equal(t, models.SubtitleRunMediaMovie, est.MediaType)
}

func TestTranscriptionEstimate_TheRealServiceIsAPlanSourceAndEstimatingStartsNoJob(t *testing.T) {
	transcription := NewTranscriptionService(nil, nil, nil, nil)
	var _ transcriptionPlanSource = transcription

	est := NewTranscriptionEstimateService(transcription, func() string { return "claude-sonnet-5" }, false, nil).
		Estimate(context.Background(), sixtyMinuteMovie())

	assert.False(t, transcription.IsInProgress("movie-1"), "an estimate must never take the single-flight slot")
	assert.False(t, est.ASRAvailable)
	assert.False(t, est.TranslationConfigured)
	assert.Equal(t, TranscriptionPlanFull, est.Plan)
}

type keyProbeCompleter struct{ configured bool }

func (c keyProbeCompleter) CompleteText(context.Context, string, string, int) (string, error) {
	return "", nil
}
func (c keyProbeCompleter) IsConfigured(context.Context) bool { return c.configured }

func TestTranscriptionService_TranslationEnabledIsTheRunsOwnCheck(t *testing.T) {
	svc := NewTranscriptionService(nil, nil, nil, nil)
	assert.False(t, svc.translationEnabled(), "no translation service wired")

	svc.SetTranslationService(&TranslationService{})
	require.NotNil(t, svc.translationService)
	assert.False(t, svc.translationEnabled(), "a translation service without a provider is not configured")

	svc.SetTranslationService(&TranslationService{provider: keyProbeCompleter{configured: false}})
	assert.False(t, svc.translationEnabled(), "a lazy holder whose key does not resolve is not configured")

	svc.SetTranslationService(&TranslationService{provider: keyProbeCompleter{configured: true}})
	assert.True(t, svc.translationEnabled(), "a resolving key means the translate leg runs")
}

type untranslatedEpisodeReader struct{ episode *models.Episode }

func (r untranslatedEpisodeReader) FindByID(context.Context, string) (*models.Episode, error) {
	return r.episode, nil
}

func TestTranscriptionEstimate_RealServiceResumesAnUntranslatedEpisodeTranslateOnly(t *testing.T) {
	// End-to-end through the REAL plan source: the same resume check the run's
	// tryTranslateOnlyResume uses, on the EPISODES table.
	srt := filepath.Join(t.TempDir(), "s01e01.en.srt")
	require.NoError(t, os.WriteFile(srt, []byte("1\n00:00:01,000 --> 00:00:02,000\nHi\n"), 0o644))

	transcription := NewTranscriptionService(nil, nil, nil, nil)
	transcription.SetTranslationService(&TranslationService{provider: keyProbeCompleter{configured: true}})
	transcription.SetEpisodeSubtitleStateReader(untranslatedEpisodeReader{episode: &models.Episode{
		ID:             "ep-1",
		SubtitleStatus: models.SubtitleStatusUntranslated,
		SubtitlePath:   models.NewNullString(srt),
	}})

	est := NewTranscriptionEstimateService(transcription, func() string { return "claude-sonnet-5" }, false, nil).
		Estimate(context.Background(), TranscriptionEstimateTarget{
			MediaID:         "ep-1",
			MediaType:       models.SubtitleRunMediaEpisode,
			FilePath:        "/tv/s01e01.mkv",
			DurationSeconds: models.NewNullInt64(2880),
		})

	assert.Equal(t, TranscriptionPlanTranslateOnly, est.Plan)
	assert.Equal(t, 0.39, est.EstimatedUSD) // 48 × 0.00804 = 0.38592
	assert.False(t, est.ASRAvailable, "no extractor/ASR wired — a resume still prices, the ASR gate does not apply")

	require.NoError(t, os.Remove(srt))
	gone := NewTranscriptionEstimateService(transcription, func() string { return "claude-sonnet-5" }, false, nil).
		Estimate(context.Background(), TranscriptionEstimateTarget{
			MediaID: "ep-1", MediaType: models.SubtitleRunMediaEpisode, FilePath: "/tv/s01e01.mkv",
			DurationSeconds: models.NewNullInt64(2880),
		})
	assert.Equal(t, TranscriptionPlanFull, gone.Plan, "the English SRT is gone, so the run would transcribe again")
}

func TestTranscriptionEstimate_CancelledProbeStillQuotes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	svc := newTestEstimator(&stubPlanSource{available: true, translate: true}, false)
	svc.SetDurationProber(&stubDurationPredictor{err: context.Canceled})
	target := sixtyMinuteMovie()
	target.DurationSeconds = models.NullInt64{}
	target.Runtime = models.NewNullInt64(100)

	est := svc.Estimate(ctx, target)
	assert.Equal(t, RuntimeSourceTMDb, est.RuntimeSource)
}
