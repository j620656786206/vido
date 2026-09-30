package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"

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
