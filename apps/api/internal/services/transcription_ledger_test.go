package services

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
)

// sub-7-6a — the transcription engine records its OWN runs. Before this a solo
//「生成字幕」click left no subtitle_runs row, so the monthly ASR spend could
// only ever see pipeline-mode fallbacks.

type fakeRunLedger struct {
	created []models.SubtitleRun
	updated []models.SubtitleRun
}

func (l *fakeRunLedger) Create(_ context.Context, run *models.SubtitleRun) error {
	if run.ID == "" {
		run.ID = "ledger-1"
	}
	l.created = append(l.created, *run)
	return nil
}

func (l *fakeRunLedger) Update(_ context.Context, run *models.SubtitleRun) error {
	l.updated = append(l.updated, *run)
	return nil
}

// The translate-only resume path completes without ASR (the media file does
// not exist — extract would fail), which is exactly what lets this test prove
// the ledger lifecycle end to end.
func TestRunTranscription_SoloRunWritesItsOwnLedgerRow(t *testing.T) {
	tmp := t.TempDir()
	enPath := filepath.Join(tmp, "Movie.en.srt")
	require.NoError(t, os.WriteFile(enPath, []byte(genTestSRT), 0644))

	ledger := &fakeRunLedger{}
	svc := resumeService(t, &translationIntegrationMock{response: "[1] 你好世界"}, &fakeSubtitleWriter{},
		&fakeStateReader{movie: untranslatedMovie(uuidD, enPath)})
	svc.SetRunLedger(ledger)

	err := svc.RunTranscription(context.Background(), uuidD, filepath.Join(tmp, "Movie.mkv"), tmp, WithTranslation())
	require.NoError(t, err)

	require.Len(t, ledger.created, 1, "one row opened when the run started")
	opened := ledger.created[0]
	assert.Equal(t, models.SubtitleRunRunning, opened.Status)
	assert.Equal(t, models.SubtitleRunRouteASR, opened.Route)
	assert.Equal(t, uuidD, opened.MediaID)
	assert.Equal(t, models.SubtitleRunMediaMovie, opened.MediaType)
	assert.NotEmpty(t, opened.ModelID, "the model the engine will bill is known at open time")

	require.NotEmpty(t, ledger.updated, "the terminal write closes the row")
	closed := ledger.updated[len(ledger.updated)-1]
	assert.Equal(t, opened.ID, closed.ID)
	assert.Equal(t, models.SubtitleRunCompleted, closed.Status)
	assert.Equal(t, 1, closed.CueCount, "the routed cue count comes from translateSRT")
	assert.NotEmpty(t, closed.MetadataHash, "the version tuple makes the row comparable with pipeline rows")
	assert.NotEmpty(t, closed.PromptVersion)
	assert.NotEmpty(t, closed.OutputPath, "the delivered sidecar path")
	require.NotNil(t, closed.CompletedAt)
	require.NotNil(t, closed.SpentUSD, "a Budget was on the ctx (resolveBudget), so the delta is recorded")
	require.NotNil(t, closed.BudgetUSD)
	assert.Nil(t, closed.CacheHitCues, "no segment store wired → the split was never measured (absent, not 0)")
	assert.Equal(t, "", closed.BatchID)
}

func TestRunTranscription_CallerOwnedRunOpensNoLedgerRow(t *testing.T) {
	tmp := t.TempDir()
	enPath := filepath.Join(tmp, "Movie.en.srt")
	require.NoError(t, os.WriteFile(enPath, []byte(genTestSRT), 0644))

	ledger := &fakeRunLedger{}
	svc := resumeService(t, &translationIntegrationMock{response: "[1] 你好世界"}, &fakeSubtitleWriter{},
		&fakeStateReader{movie: untranslatedMovie(uuidD, enPath)})
	svc.SetRunLedger(ledger)

	// The pipeline's ASR fallback: that item already has the pipeline's row.
	err := svc.RunTranscription(context.Background(), uuidD, filepath.Join(tmp, "Movie.mkv"), tmp,
		WithTranslation(), WithRunRecordedByCaller())
	require.NoError(t, err)

	assert.Empty(t, ledger.created, "a second row for the same work would double-count it on the spend page")
	assert.Empty(t, ledger.updated)
}

func TestRunTranscription_FailedRunClosesTheLedgerRowAsFailed(t *testing.T) {
	tmp := t.TempDir()
	ledger := &fakeRunLedger{}
	// No resume possible (no reader) → the full run is attempted and fails on
	// the missing media file before any paid call.
	svc := resumeService(t, &translationIntegrationMock{response: "[1] 你好"}, &fakeSubtitleWriter{}, nil)
	svc.SetRunLedger(ledger)

	err := svc.RunTranscription(context.Background(), uuidD, filepath.Join(tmp, "missing.mkv"), tmp, WithTranslation())
	require.Error(t, err)

	require.Len(t, ledger.created, 1)
	require.NotEmpty(t, ledger.updated)
	closed := ledger.updated[len(ledger.updated)-1]
	assert.Equal(t, models.SubtitleRunFailed, closed.Status)
	assert.NotEmpty(t, closed.ErrorMessage)
	assert.Equal(t, models.SubtitleRunRouteASR, closed.Route)
	require.NotNil(t, closed.CompletedAt)
}

func TestRunTranscription_LegacyBatchRunCarriesTheBatchID(t *testing.T) {
	tmp := t.TempDir()
	enPath := filepath.Join(tmp, "Movie.en.srt")
	require.NoError(t, os.WriteFile(enPath, []byte(genTestSRT), 0644))

	ledger := &fakeRunLedger{}
	svc := resumeService(t, &translationIntegrationMock{response: "[1] 你好世界"}, &fakeSubtitleWriter{},
		&fakeStateReader{movie: untranslatedMovie(uuidD, enPath)})
	svc.SetRunLedger(ledger)

	ctx := WithGenerationBatchID(context.Background(), "batch-legacy-3")
	require.NoError(t, svc.RunTranscription(ctx, uuidD, filepath.Join(tmp, "Movie.mkv"), tmp, WithTranslation()))

	require.Len(t, ledger.created, 1)
	assert.Equal(t, "batch-legacy-3", ledger.created[0].BatchID,
		"a legacy-mode batch item names its batch — the receipt can be summed from the ledger")
}

// ─── disc-2026-11-spend-history-a-ledger ────────────────────────────────────

// fakeRunEstimator records what it was asked and answers usd / err.
type fakeRunEstimator struct {
	usd   float64
	err   error
	calls []string
}

func (f *fakeRunEstimator) EstimateRunUSD(_ context.Context, mediaType, mediaID string) (float64, error) {
	f.calls = append(f.calls, mediaType+":"+mediaID)
	return f.usd, f.err
}

// identityASR is slowASR that can name itself, like *ASRProviderHolder.
type identityASR struct {
	slowASR
	provider, model string
}

func (a *identityASR) Identity(context.Context) (string, string) { return a.provider, a.model }

// AC #2/#3/#4 on the Route C leg: a run where speech recognition really runs
// records the dialog's quote, source=audio, and the engine that was billed.
func TestRunTranscription_LedgerRowRecordsEstimateSourceAndEngine(t *testing.T) {
	asr := &identityASR{slowASR: slowASR{srt: genTestSRT}, provider: ASRProviderElevenLabs, model: "scribe_v2"}
	svc, _ := runBudgetService(t, 30, asr, time.Minute, time.Minute)
	ledger := &fakeRunLedger{}
	svc.SetRunLedger(ledger)
	est := &fakeRunEstimator{usd: 0.42}
	svc.SetRunEstimator(est)
	media := filepath.Join(t.TempDir(), "m.mkv")
	require.NoError(t, os.WriteFile(media, []byte("x"), 0o600))

	require.NoError(t, svc.RunTranscription(context.Background(), uuidEight, media, filepath.Dir(media)))

	assert.Equal(t, []string{"movie:" + uuidEight}, est.calls, "one estimate, for this item, at run start")
	require.Len(t, ledger.created, 1)
	opened := ledger.created[0]
	require.NotNil(t, opened.EstimateUSD, "the quote is on the row from the moment it opens")
	assert.InDelta(t, 0.42, *opened.EstimateUSD, 1e-9)
	assert.Equal(t, models.SubtitleRunSourceAudio, opened.Source)
	assert.Equal(t, "", opened.ASRProvider, "the engine is stamped when ASR runs, not guessed at open time")

	closed := ledger.updated[len(ledger.updated)-1]
	assert.Equal(t, models.SubtitleRunCompleted, closed.Status)
	assert.Equal(t, ASRProviderElevenLabs, closed.ASRProvider)
	assert.Equal(t, "scribe_v2", closed.ASRModel)
	assert.Equal(t, models.SubtitleRunSourceAudio, closed.Source)
	require.NotNil(t, closed.EstimateUSD)
	assert.Equal(t, models.SubtitleRunRouteASR, closed.Route, "route is unchanged — the monthly report groups by it")
}

// AC #2: a failing estimate never fails the run — the column stays NULL.
func TestRunTranscription_EstimateFailureLeavesNullAndRunCompletes(t *testing.T) {
	tmp := t.TempDir()
	enPath := filepath.Join(tmp, "Movie.en.srt")
	require.NoError(t, os.WriteFile(enPath, []byte(genTestSRT), 0644))

	ledger := &fakeRunLedger{}
	svc := resumeService(t, &translationIntegrationMock{response: "[1] 你好世界"}, &fakeSubtitleWriter{},
		&fakeStateReader{movie: untranslatedMovie(uuidD, enPath)})
	svc.SetRunLedger(ledger)
	svc.SetRunEstimator(&fakeRunEstimator{err: errors.New("ffprobe timed out")})

	require.NoError(t, svc.RunTranscription(context.Background(), uuidD, filepath.Join(tmp, "Movie.mkv"), tmp, WithTranslation()))

	closed := ledger.updated[len(ledger.updated)-1]
	assert.Equal(t, models.SubtitleRunCompleted, closed.Status)
	assert.Nil(t, closed.EstimateUSD, "no figure is NULL, never a $0 quote")
	// A translate-only resume: the subtitle still comes from audio (an earlier
	// run's English SRT), but no speech recognition ran THIS time.
	assert.Equal(t, models.SubtitleRunSourceAudio, closed.Source)
	assert.Equal(t, "", closed.ASRProvider, "no ASR ran on this run")
	assert.Equal(t, "", closed.ASRModel)
}

// Unwired estimator (legacy tests, or a boot that never set it): NULL, run fine.
func TestRunTranscription_NoEstimatorLeavesNull(t *testing.T) {
	tmp := t.TempDir()
	enPath := filepath.Join(tmp, "Movie.en.srt")
	require.NoError(t, os.WriteFile(enPath, []byte(genTestSRT), 0644))

	ledger := &fakeRunLedger{}
	svc := resumeService(t, &translationIntegrationMock{response: "[1] 你好世界"}, &fakeSubtitleWriter{},
		&fakeStateReader{movie: untranslatedMovie(uuidD, enPath)})
	svc.SetRunLedger(ledger)

	require.NoError(t, svc.RunTranscription(context.Background(), uuidD, filepath.Join(tmp, "Movie.mkv"), tmp, WithTranslation()))
	assert.Nil(t, ledger.created[0].EstimateUSD)
}

// AC #4 on the pipeline's ASR fallback: the CALLER owns the row, so the engine
// is stamped on the row the caller handed down — and no second row is opened.
func TestRunTranscription_CallerOwnedRowGetsTheEngineStamped(t *testing.T) {
	asr := &identityASR{slowASR: slowASR{srt: genTestSRT}, provider: ASRProviderOpenAI, model: ""}
	svc, _ := runBudgetService(t, 30, asr, time.Minute, time.Minute)
	ledger := &fakeRunLedger{}
	svc.SetRunLedger(ledger)
	svc.SetRunEstimator(&fakeRunEstimator{usd: 1})
	media := filepath.Join(t.TempDir(), "m.mkv")
	require.NoError(t, os.WriteFile(media, []byte("x"), 0o600))

	callerRun := &models.SubtitleRun{ID: "pipeline-row", MediaID: uuidEight, MediaType: models.SubtitleRunMediaMovie}
	ctx := WithASRIdentityTarget(context.Background(), callerRun)
	require.NoError(t, svc.RunTranscription(ctx, uuidEight, media, filepath.Dir(media), WithRunRecordedByCaller()))

	assert.Empty(t, ledger.created, "the caller's row is the only row")
	assert.Equal(t, ASRProviderOpenAI, callerRun.ASRProvider)
	assert.Equal(t, "", callerRun.ASRModel, "Whisper without ASR_MODEL: model not knowable, stays NULL")
	assert.Nil(t, callerRun.EstimateUSD, "the caller computes its own estimate; the engine adds none")
}
