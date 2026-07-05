package services

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/models"
)

// ─── Story 9R-16 additions: sync entry, budget threading, AC 12 writeback ───

type subtitleWriteCall struct {
	ID       string
	Status   models.SubtitleStatus
	Path     string
	Language string
	Score    float64
}

type fakeSubtitleWriter struct {
	calls []subtitleWriteCall
	err   error
}

func (f *fakeSubtitleWriter) UpdateSubtitleStatus(_ context.Context, id string, status models.SubtitleStatus, path, language string, score float64) error {
	f.calls = append(f.calls, subtitleWriteCall{ID: id, Status: status, Path: path, Language: language, Score: score})
	return f.err
}

// budgetExceededCompleter simulates the governor's budget pre-check sentinel.
type budgetExceededCompleter struct{}

func (b *budgetExceededCompleter) CompleteText(_ context.Context, _, _ string, _ int) (string, error) {
	return "", ai.ErrBudgetExceeded
}

// ─── RunTranscription (AC 6a) ────────────────────────────────────────────────

func TestRunTranscription_Disabled(t *testing.T) {
	svc := NewTranscriptionService(nil, nil, nil, nil)
	err := svc.RunTranscription(context.Background(), 1, "/test.mkv", "/media")
	assert.ErrorIs(t, err, ErrTranscriptionDisabled)
}

func TestRunTranscription_SharesSingleFlightMapWithAsyncPath(t *testing.T) {
	extractor := &AudioExtractorService{available: true, semaphore: make(chan struct{}, 1)}
	whisperClient := ai.NewWhisperClient("test-key")
	svc := NewTranscriptionService(extractor, whisperClient, nil, nil)

	// Simulate an async-path registration (same map, single-flight consistency).
	svc.mu.Lock()
	svc.inProgress[42] = "async-job"
	svc.mu.Unlock()

	err := svc.RunTranscription(context.Background(), 42, "/test.mkv", "/media")
	assert.ErrorIs(t, err, ErrTranscriptionInProgress)

	// And the reverse: a sync registration blocks StartTranscription too.
	_, err = svc.StartTranscription(context.Background(), 42, "/test.mkv", "/media")
	assert.ErrorIs(t, err, ErrTranscriptionInProgress)
}

func TestRunTranscription_ReturnsPipelineErrorAndReleasesSlot(t *testing.T) {
	extractor := &AudioExtractorService{available: true, semaphore: make(chan struct{}, 1)}
	whisperClient := ai.NewWhisperClient("test-key")
	svc := NewTranscriptionService(extractor, whisperClient, nil, nil)

	// Nonexistent media file — the pipeline fails at the extract phase and the
	// sync entry must RETURN the error (the async path only broadcasts SSE).
	err := svc.RunTranscription(context.Background(), 7, filepath.Join(t.TempDir(), "missing.mkv"), t.TempDir())
	require.Error(t, err)
	assert.False(t, svc.IsInProgress(7), "single-flight slot must be released after failure")
}

func TestRunTranscription_DerivesTimeoutFromCallerCtx(t *testing.T) {
	extractor := &AudioExtractorService{available: true, semaphore: make(chan struct{}, 1)}
	whisperClient := ai.NewWhisperClient("test-key")
	svc := NewTranscriptionService(extractor, whisperClient, nil, nil)

	// A cancelled caller ctx must abort the sync pipeline — the async path's
	// context.Background() detach would ignore it (the AC 6a ctx trap).
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	err := svc.RunTranscription(ctx, 8, filepath.Join(t.TempDir(), "missing.mkv"), t.TempDir())
	require.Error(t, err)
	assert.Less(t, time.Since(start), 2*time.Second, "cancelled caller ctx must fail fast")
	assert.False(t, svc.IsInProgress(8))
}

// ─── resolveBudget (AC 6b) ───────────────────────────────────────────────────

func TestResolveBudget_ReusesCtxAttachedBudget(t *testing.T) {
	svc := NewTranscriptionService(nil, nil, nil, nil)
	svc.SetRunBudgetUSD(9.99)

	shared := ai.NewBudget(5.0)
	ctx := ai.WithBudget(context.Background(), shared)

	got, outCtx := svc.resolveBudget(ctx)
	assert.Same(t, shared, got, "a ctx-attached (batch) budget must be reused, not replaced")
	assert.Same(t, shared, ai.BudgetFromContext(outCtx))
}

func TestResolveBudget_CreatesPerRunBudgetWhenAbsent(t *testing.T) {
	svc := NewTranscriptionService(nil, nil, nil, nil)
	svc.SetRunBudgetUSD(2.5)

	got, outCtx := svc.resolveBudget(context.Background())
	require.NotNil(t, got)
	assert.Same(t, got, ai.BudgetFromContext(outCtx), "fresh budget must ride the returned ctx")
	assert.Equal(t, 2.5, got.Snapshot().BudgetUSD)
}

// ─── translateAndPersist (AC 6c + AC 12) ─────────────────────────────────────

func newWriterWiredService(t *testing.T, completer ai.TextCompleter, writer SubtitleStatusWriter) *TranscriptionService {
	t.Helper()
	svc := NewTranscriptionService(nil, nil, nil, nil)
	if completer != nil {
		svc.SetTranslationService(NewTranslationService(completer, nil))
	}
	if writer != nil {
		svc.SetSubtitleStatusWriter(writer)
	}
	return svc
}

const genTestSRT = "1\n00:00:01,000 --> 00:00:04,000\nHello world\n"

// AC 12: success writes found/path/zh-Hant via the narrow updater.
func TestTranslateAndPersist_SuccessWritesBack(t *testing.T) {
	writer := &fakeSubtitleWriter{}
	svc := newWriterWiredService(t, &translationIntegrationMock{response: "[1] 你好世界"}, writer)

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "Movie.2024.mkv")

	zhPath, err := svc.translateAndPersist(context.Background(), "job-1", 42, genTestSRT, filePath, tmpDir, true)
	require.NoError(t, err)
	require.NotEmpty(t, zhPath)

	require.Len(t, writer.calls, 1)
	call := writer.calls[0]
	assert.Equal(t, "42", call.ID, "int64 media id converts to the string row id")
	assert.Equal(t, models.SubtitleStatusFound, call.Status)
	assert.Equal(t, zhPath, call.Path)
	assert.Equal(t, "zh-Hant", call.Language)
	assert.Equal(t, 0.0, call.Score, "generation has no search score — stored NULL")
}

// AC 12: en-only runs (translate absent) do NOT write zh-Hant fields.
func TestTranslateAndPersist_EnOnlyNoWrite(t *testing.T) {
	writer := &fakeSubtitleWriter{}
	svc := newWriterWiredService(t, &translationIntegrationMock{response: "[1] 你好"}, writer)

	zhPath, err := svc.translateAndPersist(context.Background(), "job-1", 42, genTestSRT,
		filepath.Join(t.TempDir(), "m.mkv"), t.TempDir(), false /* en-only */)
	require.NoError(t, err)
	assert.Empty(t, zhPath)
	assert.Empty(t, writer.calls, "en-only run must not write zh-Hant fields")
}

// AC 12: translation failure writes nothing (English SRT preserved, non-fatal).
func TestTranslateAndPersist_TranslateFailureNoWrite(t *testing.T) {
	writer := &fakeSubtitleWriter{}
	// Empty SRT → translateSRT errors with "no subtitle blocks" (a non-budget failure).
	svc := newWriterWiredService(t, &translationIntegrationMock{response: "[1] 你好"}, writer)

	zhPath, err := svc.translateAndPersist(context.Background(), "job-1", 42, "",
		filepath.Join(t.TempDir(), "m.mkv"), t.TempDir(), true)
	require.NoError(t, err, "ordinary translate failures stay non-fatal (AC 6c)")
	assert.Empty(t, zhPath)
	assert.Empty(t, writer.calls, "failed generation must not write")
}

// AC 6c: ErrBudgetExceeded MUST propagate out of the translate phase.
func TestTranslateAndPersist_BudgetExceededPropagates(t *testing.T) {
	writer := &fakeSubtitleWriter{}
	svc := newWriterWiredService(t, &budgetExceededCompleter{}, writer)

	_, err := svc.translateAndPersist(context.Background(), "job-1", 42, genTestSRT,
		filepath.Join(t.TempDir(), "m.mkv"), t.TempDir(), true)
	require.Error(t, err)
	assert.ErrorIs(t, err, ai.ErrBudgetExceeded,
		"the budget sentinel must survive wrapping so the batch can pause mid-item")
	assert.Empty(t, writer.calls)
}

// Rule 13: a writeback failure propagates — the run must not report success
// while the library row still says 缺字幕.
func TestTranslateAndPersist_WritebackFailurePropagates(t *testing.T) {
	writer := &fakeSubtitleWriter{err: errors.New("db locked")}
	svc := newWriterWiredService(t, &translationIntegrationMock{response: "[1] 你好"}, writer)

	_, err := svc.translateAndPersist(context.Background(), "job-1", 42, genTestSRT,
		filepath.Join(t.TempDir(), "m.mkv"), t.TempDir(), true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "update subtitle status")
}

// Nil writer keeps the pre-9R-16 behavior (place file, persist nothing).
func TestTranslateAndPersist_NilWriterIsNoop(t *testing.T) {
	svc := newWriterWiredService(t, &translationIntegrationMock{response: "[1] 你好"}, nil)

	zhPath, err := svc.translateAndPersist(context.Background(), "job-1", 42, genTestSRT,
		filepath.Join(t.TempDir(), "m.mkv"), t.TempDir(), true)
	require.NoError(t, err)
	assert.NotEmpty(t, zhPath)
}

// AC 6c regression at the TranslationService seam: the sentinel escapes the
// keep-English batch tolerance while ordinary errors do not.
func TestTranslateWithGlossary_BudgetSentinelEscapesTolerance(t *testing.T) {
	ts := NewTranslationService(&budgetExceededCompleter{}, nil)
	blocks := []TranslationBlock{{Index: 1, Start: "00:00:01,000", End: "00:00:02,000", Text: "Hi"}}

	_, err := ts.TranslateWithGlossary(context.Background(), blocks, nil, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, ai.ErrBudgetExceeded)
}
