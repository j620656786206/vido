package services

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/ai/prompts"
	"github.com/vido/api/internal/models"
)

// ─── Story 9R-16 additions: sync entry, budget threading, AC 12 writeback ───
//
// Media-id fixture convention (9R-18 AC 7): media ids are UUID STRINGS —
// mirror the prod creation path (uuid.New().String()); do NOT invent numeric
// ids (fixtures reuse the uuid* consts from generation_batch_test.go).

type subtitleWriteCall struct {
	ID       string
	Status   models.SubtitleStatus
	Path     string
	Language string
	Score    float64
}

type fakeSubtitleWriter struct {
	mu    sync.Mutex // guards calls — the async StartTranscription path writes from a goroutine
	calls []subtitleWriteCall
	err   error
}

// snapshot returns a copy of calls for cross-goroutine assertions.
func (f *fakeSubtitleWriter) snapshot() []subtitleWriteCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]subtitleWriteCall(nil), f.calls...)
}

func (f *fakeSubtitleWriter) UpdateSubtitleGenerationStatus(ctx context.Context, id string, status models.SubtitleStatus, path, language string) error {
	return nil
}

func (f *fakeSubtitleWriter) UpdateSubtitleStatus(_ context.Context, id string, status models.SubtitleStatus, path, language string, score float64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
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
	err := svc.RunTranscription(context.Background(), uuidA, "/test.mkv", "/media")
	assert.ErrorIs(t, err, ErrTranscriptionDisabled)
}

func TestRunTranscription_SharesSingleFlightMapWithAsyncPath(t *testing.T) {
	extractor := &AudioExtractorService{available: true, semaphore: make(chan struct{}, 1)}
	whisperClient := ai.NewWhisperClient("test-key")
	svc := NewTranscriptionService(extractor, whisperClient, nil, nil)

	// Simulate an async-path registration (same map, single-flight consistency).
	svc.mu.Lock()
	svc.inProgress[uuidB] = "async-job"
	svc.mu.Unlock()

	err := svc.RunTranscription(context.Background(), uuidB, "/test.mkv", "/media")
	assert.ErrorIs(t, err, ErrTranscriptionInProgress)

	// And the reverse: a sync registration blocks StartTranscription too.
	_, err = svc.StartTranscription(context.Background(), uuidB, "/test.mkv", "/media")
	assert.ErrorIs(t, err, ErrTranscriptionInProgress)
}

func TestRunTranscription_ReturnsPipelineErrorAndReleasesSlot(t *testing.T) {
	extractor := &AudioExtractorService{available: true, semaphore: make(chan struct{}, 1)}
	whisperClient := ai.NewWhisperClient("test-key")
	svc := NewTranscriptionService(extractor, whisperClient, nil, nil)

	// Nonexistent media file — the pipeline fails at the extract phase and the
	// sync entry must RETURN the error (the async path only broadcasts SSE).
	err := svc.RunTranscription(context.Background(), uuidSeven, filepath.Join(t.TempDir(), "missing.mkv"), t.TempDir())
	require.Error(t, err)
	assert.False(t, svc.IsInProgress(uuidSeven), "single-flight slot must be released after failure")
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
	err := svc.RunTranscription(ctx, uuidEight, filepath.Join(t.TempDir(), "missing.mkv"), t.TempDir())
	require.Error(t, err)
	assert.Less(t, time.Since(start), 2*time.Second, "cancelled caller ctx must fail fast")
	assert.False(t, svc.IsInProgress(uuidEight))
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

	zhPath, _, err := svc.translateAndPersist(context.Background(), "job-1", models.SubtitleRunMediaMovie, uuidB, genTestSRT,
		filepath.Join(tmpDir, "Movie.2024.en.srt"), filePath, tmpDir, true)
	require.NoError(t, err)
	require.NotEmpty(t, zhPath)

	require.Len(t, writer.calls, 1)
	call := writer.calls[0]
	assert.Equal(t, uuidB, call.ID, "the string media id IS the row id — no conversion (9R-18)")
	assert.Equal(t, models.SubtitleStatusFound, call.Status)
	assert.Equal(t, zhPath, call.Path)
	assert.Equal(t, "zh-Hant", call.Language)
	assert.Equal(t, 0.0, call.Score, "generation has no search score — stored NULL")
}

// AC 12: en-only runs (translate absent) do NOT write zh-Hant fields.
func TestTranslateAndPersist_EnOnlyNoWrite(t *testing.T) {
	writer := &fakeSubtitleWriter{}
	svc := newWriterWiredService(t, &translationIntegrationMock{response: "[1] 你好"}, writer)

	zhPath, _, err := svc.translateAndPersist(context.Background(), "job-1", models.SubtitleRunMediaMovie, uuidB, genTestSRT,
		filepath.Join(t.TempDir(), "m.en.srt"),
		filepath.Join(t.TempDir(), "m.mkv"), t.TempDir(), false /* en-only */)
	require.NoError(t, err)
	assert.Empty(t, zhPath)
	// sub-2-2a keeps this exact: translate ABSENT means no translation was
	// EXPECTED, so `untranslated` would be a lie — nothing is written.
	assert.Empty(t, writer.calls, "en-only run must not write anything")
}

// 9R-16 AC 12 [@contract-v2] (sub-2-2a drift): a NON-FATAL translate failure
// now records `untranslated` — the English SRT is on disk and the row must say
// so, or the badge lies 缺字幕 and a re-click re-runs the whole ASR.
func TestTranslateAndPersist_TranslateFailureWritesUntranslated(t *testing.T) {
	writer := &fakeSubtitleWriter{}
	// Empty SRT → translateSRT errors with "no subtitle blocks" (a non-budget failure).
	svc := newWriterWiredService(t, &translationIntegrationMock{response: "[1] 你好"}, writer)

	enPath := filepath.Join(t.TempDir(), "m.en.srt")
	zhPath, _, err := svc.translateAndPersist(context.Background(), "job-1", models.SubtitleRunMediaMovie, uuidB, "", enPath,
		filepath.Join(t.TempDir(), "m.mkv"), t.TempDir(), true)
	require.NoError(t, err, "ordinary translate failures stay non-fatal (AC 6c)")
	assert.Empty(t, zhPath)
	require.Len(t, writer.calls, 1)
	call := writer.calls[0]
	assert.Equal(t, models.SubtitleStatusUntranslated, call.Status)
	assert.Equal(t, enPath, call.Path, "the row records the EN SRT so resume can find it")
	assert.Equal(t, "en", call.Language)
}

// AC 6c: ErrBudgetExceeded MUST propagate out of the translate phase.
func TestTranslateAndPersist_BudgetExceededPropagates(t *testing.T) {
	writer := &fakeSubtitleWriter{}
	svc := newWriterWiredService(t, &budgetExceededCompleter{}, writer)

	enPath := filepath.Join(t.TempDir(), "m.en.srt")
	_, _, err := svc.translateAndPersist(context.Background(), "job-1", models.SubtitleRunMediaMovie, uuidB, genTestSRT, enPath,
		filepath.Join(t.TempDir(), "m.mkv"), t.TempDir(), true)
	require.Error(t, err)
	assert.ErrorIs(t, err, ai.ErrBudgetExceeded,
		"the budget sentinel must survive wrapping so the batch can pause mid-item")
	// CR sub-2-2a M1: the on-disk truth is recorded BEFORE the pause — the EN
	// SRT exists and translation is missing, so `untranslated` is factually
	// true, and the post-pause resume becomes translate-only instead of
	// re-paying the whole ASR. The sentinel still propagates (above).
	require.Len(t, writer.calls, 1)
	assert.Equal(t, models.SubtitleStatusUntranslated, writer.calls[0].Status)
	assert.Equal(t, enPath, writer.calls[0].Path)
}

// Rule 13: a writeback failure propagates — the run must not report success
// while the library row still says 缺字幕.
func TestTranslateAndPersist_WritebackFailurePropagates(t *testing.T) {
	writer := &fakeSubtitleWriter{err: errors.New("db locked")}
	svc := newWriterWiredService(t, &translationIntegrationMock{response: "[1] 你好"}, writer)

	_, _, err := svc.translateAndPersist(context.Background(), "job-1", models.SubtitleRunMediaMovie, uuidB, genTestSRT,
		filepath.Join(t.TempDir(), "m.en.srt"),
		filepath.Join(t.TempDir(), "m.mkv"), t.TempDir(), true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "update subtitle status")
}

// Nil writer keeps the pre-9R-16 behavior (place file, persist nothing).
func TestTranslateAndPersist_NilWriterIsNoop(t *testing.T) {
	svc := newWriterWiredService(t, &translationIntegrationMock{response: "[1] 你好"}, nil)

	zhPath, _, err := svc.translateAndPersist(context.Background(), "job-1", models.SubtitleRunMediaMovie, uuidB, genTestSRT,
		filepath.Join(t.TempDir(), "m.en.srt"),
		filepath.Join(t.TempDir(), "m.mkv"), t.TempDir(), true)
	require.NoError(t, err)
	assert.NotEmpty(t, zhPath)
}

// ─── Story sub-2-2a: untranslated writeback + translate-only resume ─────────

// fakeStateReader satisfies SubtitleStateReader (sub-2-2a AC #3).
type fakeStateReader struct {
	movie *models.Movie
	err   error
}

func (f *fakeStateReader) FindByID(_ context.Context, _ string) (*models.Movie, error) {
	return f.movie, f.err
}

// AC #2: translation EXPECTED but the key is unconfigured (no translation
// service) → the silent skip must record `untranslated` + the EN path, or the
// badge lies 缺字幕 over a paid-for subtitle.
func TestTranslateAndPersist_KeyUnconfiguredWritesUntranslated(t *testing.T) {
	writer := &fakeSubtitleWriter{}
	svc := newWriterWiredService(t, nil /* no translation service — key unconfigured */, writer)

	enPath := filepath.Join(t.TempDir(), "m.en.srt")
	zhPath, _, err := svc.translateAndPersist(context.Background(), "job-1", models.SubtitleRunMediaMovie, uuidB, genTestSRT, enPath,
		filepath.Join(t.TempDir(), "m.mkv"), t.TempDir(), true)
	require.NoError(t, err)
	assert.Empty(t, zhPath)
	require.Len(t, writer.calls, 1)
	call := writer.calls[0]
	assert.Equal(t, uuidB, call.ID)
	assert.Equal(t, models.SubtitleStatusUntranslated, call.Status)
	assert.Equal(t, enPath, call.Path)
	assert.Equal(t, "en", call.Language)
	assert.Equal(t, 0.0, call.Score)
}

// Rule 13: failing to record `untranslated` propagates exactly like the found
// writeback — success must not be reported while the row still lies.
func TestTranslateAndPersist_UntranslatedWritebackFailurePropagates(t *testing.T) {
	writer := &fakeSubtitleWriter{err: errors.New("db locked")}
	svc := newWriterWiredService(t, nil, writer)

	_, _, err := svc.translateAndPersist(context.Background(), "job-1", models.SubtitleRunMediaMovie, uuidB, genTestSRT,
		filepath.Join(t.TempDir(), "m.en.srt"),
		filepath.Join(t.TempDir(), "m.mkv"), t.TempDir(), true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "update subtitle status")
}

// resumeService builds a service whose extract/ASR phases FAIL LOUDLY if
// invoked: the extractor probes a nonexistent media file (ffprobe errors) and
// the whisper key is bogus. A completed run therefore PROVES the resume path
// skipped them — the non-invocation proxy for the idempotency mandate.
func resumeService(t *testing.T, completer ai.TextCompleter, writer SubtitleStatusWriter, reader SubtitleStateReader) *TranscriptionService {
	t.Helper()
	extractor := &AudioExtractorService{available: true, semaphore: make(chan struct{}, 1)}
	svc := NewTranscriptionService(extractor, ai.NewWhisperClient("bogus-test-key"), nil, nil)
	if completer != nil {
		svc.SetTranslationService(NewTranslationService(completer, nil))
	}
	if writer != nil {
		svc.SetSubtitleStatusWriter(writer)
	}
	if reader != nil {
		svc.SetSubtitleStateReader(reader)
	}
	return svc
}

func untranslatedMovie(id, enPath string) *models.Movie {
	m := &models.Movie{ID: id, Title: "Resume Fixture"}
	m.SubtitleStatus = models.SubtitleStatusUntranslated
	m.SubtitlePath = models.NewNullString(enPath)
	m.SubtitleLanguage = models.NewNullString("en")
	return m
}

// AC #3 + Murat's idempotency mandate: a second run on an `untranslated` row
// with the SRT present runs translate-ONLY — extract+ASR are never invoked
// (they would error: the media file does not exist), and the row flips found.
func TestRunTranscription_ResumeSkipsASRAndTranslatesOnly(t *testing.T) {
	tmp := t.TempDir()
	enPath := filepath.Join(tmp, "Movie.en.srt")
	require.NoError(t, os.WriteFile(enPath, []byte(genTestSRT), 0644))

	writer := &fakeSubtitleWriter{}
	reader := &fakeStateReader{movie: untranslatedMovie(uuidD, enPath)}
	svc := resumeService(t, &translationIntegrationMock{response: "[1] 你好世界"}, writer, reader)

	err := svc.RunTranscription(context.Background(), uuidD,
		filepath.Join(tmp, "Movie.mkv") /* does NOT exist — extract would fail */, tmp,
		WithTranslation())
	require.NoError(t, err, "resume must complete without touching extract/ASR")

	require.NotEmpty(t, writer.calls)
	last := writer.calls[len(writer.calls)-1]
	assert.Equal(t, models.SubtitleStatusFound, last.Status)
	assert.Equal(t, "zh-Hant", last.Language)
}

// AC #3 Winston guard: the resume condition is BOUND to the status — a stray
// on-disk .srt with a non-untranslated row must NOT trigger a resume. The full
// pipeline is attempted instead (and fails here on the missing media file,
// which is the proof the full path was taken).
func TestRunTranscription_NoResumeWhenStatusNotUntranslated(t *testing.T) {
	tmp := t.TempDir()
	enPath := filepath.Join(tmp, "Movie.en.srt")
	require.NoError(t, os.WriteFile(enPath, []byte(genTestSRT), 0644))

	movie := &models.Movie{ID: uuidD, Title: "Stray SRT Fixture"}
	movie.SubtitleStatus = models.SubtitleStatusNotSearched
	movie.SubtitlePath = models.NewNullString(enPath)

	writer := &fakeSubtitleWriter{}
	reader := &fakeStateReader{movie: movie}
	svc := resumeService(t, &translationIntegrationMock{response: "[1] 你好"}, writer, reader)

	err := svc.RunTranscription(context.Background(), uuidD,
		filepath.Join(tmp, "Movie.mkv"), tmp, WithTranslation())
	require.Error(t, err, "full pipeline must be attempted (and fail on the missing media file)")
	for _, c := range writer.calls {
		assert.NotEqual(t, models.SubtitleStatusFound, c.Status,
			"a stray .srt must never be laundered into a found verdict")
	}
}

// AC #3 fail-soft: `untranslated` row whose SRT has been deleted → fall back
// to a full run (which fails here on the missing media file — the proof the
// fallback was attempted rather than erroring on the resume itself).
func TestRunTranscription_ResumeFallsBackWhenSRTMissing(t *testing.T) {
	tmp := t.TempDir()
	deleted := filepath.Join(tmp, "gone.en.srt") // never created

	writer := &fakeSubtitleWriter{}
	reader := &fakeStateReader{movie: untranslatedMovie(uuidD, deleted)}
	svc := resumeService(t, &translationIntegrationMock{response: "[1] 你好"}, writer, reader)

	err := svc.RunTranscription(context.Background(), uuidD,
		filepath.Join(tmp, "Movie.mkv"), tmp, WithTranslation())
	require.Error(t, err, "fallback full run must be attempted (and fail on the missing media file)")
}

// AC #3 + 9R-16 AC 6c: the budget sentinel propagates from the RESUME path
// too, and the row does not flip found.
func TestRunTranscription_ResumeBudgetExceededPropagates(t *testing.T) {
	tmp := t.TempDir()
	enPath := filepath.Join(tmp, "Movie.en.srt")
	require.NoError(t, os.WriteFile(enPath, []byte(genTestSRT), 0644))

	writer := &fakeSubtitleWriter{}
	reader := &fakeStateReader{movie: untranslatedMovie(uuidD, enPath)}
	svc := resumeService(t, &budgetExceededCompleter{}, writer, reader)

	err := svc.RunTranscription(context.Background(), uuidD,
		filepath.Join(tmp, "Movie.mkv"), tmp, WithTranslation())
	require.Error(t, err)
	assert.ErrorIs(t, err, ai.ErrBudgetExceeded)
	for _, c := range writer.calls {
		assert.NotEqual(t, models.SubtitleStatusFound, c.Status)
	}
}

// A nil reader (not wired) must never panic — the resume check is optional.
func TestRunTranscription_NilReaderSkipsResumeCheck(t *testing.T) {
	tmp := t.TempDir()
	writer := &fakeSubtitleWriter{}
	svc := resumeService(t, &translationIntegrationMock{response: "[1] 你好"}, writer, nil)

	err := svc.RunTranscription(context.Background(), uuidD,
		filepath.Join(tmp, "Movie.mkv"), tmp, WithTranslation())
	require.Error(t, err, "no reader → full pipeline attempted → fails on missing media file")
}

// CR sub-2-2a M2: the ASR-availability gate must not block a translate-only
// resume — an operator who removed the ASR key AFTER generating (but has a
// translation key) can still finish. Service here has NO extractor and NO ASR.
func TestRunTranscription_ResumeAllowedWithoutASR(t *testing.T) {
	tmp := t.TempDir()
	enPath := filepath.Join(tmp, "Movie.en.srt")
	require.NoError(t, os.WriteFile(enPath, []byte(genTestSRT), 0644))

	writer := &fakeSubtitleWriter{}
	svc := NewTranscriptionService(nil /* no extractor */, nil /* no ASR */, nil, nil)
	svc.SetTranslationService(NewTranslationService(&translationIntegrationMock{response: "[1] 你好世界"}, nil))
	svc.SetSubtitleStatusWriter(writer)
	svc.SetSubtitleStateReader(&fakeStateReader{movie: untranslatedMovie(uuidD, enPath)})
	require.False(t, svc.IsAvailable(), "precondition: ASR capability absent")

	err := svc.RunTranscription(context.Background(), uuidD,
		filepath.Join(tmp, "Movie.mkv"), tmp, WithTranslation())
	require.NoError(t, err)

	calls := writer.snapshot()
	require.NotEmpty(t, calls)
	assert.Equal(t, models.SubtitleStatusFound, calls[len(calls)-1].Status)
}

// CR sub-2-2a M2 counterpart: without resume eligibility the relaxed gate
// still refuses an ASR-less service — and does so at the ENTRY, not by
// panicking on a nil extractor deeper in.
func TestRunTranscription_StillDisabledWithoutASRWhenNotResumable(t *testing.T) {
	svc := NewTranscriptionService(nil, nil, nil, nil)
	svc.SetSubtitleStateReader(&fakeStateReader{movie: nil, err: errors.New("not found")})

	err := svc.RunTranscription(context.Background(), uuidD, "/x.mkv", "/media", WithTranslation())
	assert.ErrorIs(t, err, ErrTranscriptionDisabled)
}

// CR sub-2-2a M1 (resume-path variant): a budget hit DURING a resumed
// translate re-records `untranslated` (idempotent) and never flips found.
func TestRunTranscription_ResumeBudgetHitKeepsUntranslated(t *testing.T) {
	tmp := t.TempDir()
	enPath := filepath.Join(tmp, "Movie.en.srt")
	require.NoError(t, os.WriteFile(enPath, []byte(genTestSRT), 0644))

	writer := &fakeSubtitleWriter{}
	reader := &fakeStateReader{movie: untranslatedMovie(uuidD, enPath)}
	svc := resumeService(t, &budgetExceededCompleter{}, writer, reader)

	err := svc.RunTranscription(context.Background(), uuidD,
		filepath.Join(tmp, "Movie.mkv"), tmp, WithTranslation())
	require.Error(t, err)
	assert.ErrorIs(t, err, ai.ErrBudgetExceeded)
	calls := writer.snapshot()
	require.NotEmpty(t, calls, "the pre-pause untranslated record must be written")
	for _, c := range calls {
		assert.Equal(t, models.SubtitleStatusUntranslated, c.Status)
	}
}

// CR sub-2-2a L2: the ASYNC entry (StartTranscription) shares runPipeline, but
// its resume path deserves one direct smoke test — the goroutine must complete
// translate-only and flip the row found.
func TestStartTranscription_ResumeSmoke(t *testing.T) {
	tmp := t.TempDir()
	enPath := filepath.Join(tmp, "Movie.en.srt")
	require.NoError(t, os.WriteFile(enPath, []byte(genTestSRT), 0644))

	writer := &fakeSubtitleWriter{}
	reader := &fakeStateReader{movie: untranslatedMovie(uuidD, enPath)}
	svc := resumeService(t, &translationIntegrationMock{response: "[1] 你好世界"}, writer, reader)

	jobID, err := svc.StartTranscription(context.Background(), uuidD,
		filepath.Join(tmp, "Movie.mkv"), tmp, WithTranslation())
	require.NoError(t, err)
	require.NotEmpty(t, jobID)

	require.Eventually(t, func() bool {
		calls := writer.snapshot()
		return len(calls) > 0 && calls[len(calls)-1].Status == models.SubtitleStatusFound
	}, 3*time.Second, 10*time.Millisecond, "async resume must flip the row found")
}

// AC 6c regression at the TranslationService seam: the sentinel escapes the
// keep-English batch tolerance while ordinary errors do not.
func TestTranslateWithGlossary_BudgetSentinelEscapesTolerance(t *testing.T) {
	ts := NewTranslationService(&budgetExceededCompleter{}, nil)
	blocks := []TranslationBlock{{Index: 1, Start: "00:00:01,000", End: "00:00:02,000", Text: "Hi"}}

	_, _, err := ts.TranslateWithGlossary(context.Background(), blocks, nil, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, ai.ErrBudgetExceeded)
}

// ─── bugfix-j: partial translation must not persist `found` ─────────────────

// twoCueSRT pairs with a mock whose response covers only cue 1 — cue 2 keeps
// English, making the run PARTIAL (bugfix-j).
const twoCueSRT = "1\n00:00:01,000 --> 00:00:04,000\nHello world\n\n2\n00:00:05,000 --> 00:00:08,000\nStill English\n"

// bugfix-j AC #2 (9R-16 AC 12 [@contract-v2→v3]): a MATERIALLY partial
// translation (here 1/2 = 50%, over the H3 ruling's 5% bar) still places the
// mixed file (tonight-value) but the verdict demotes to untranslated + the EN
// path — the badge must never claim 繁中已就緒 over a file carrying English
// runs, and resume stays translate-only from the EN SRT.
func TestTranslateAndPersist_PartialTranslationWritesUntranslated(t *testing.T) {
	writer := &fakeSubtitleWriter{}
	svc := newWriterWiredService(t, &translationIntegrationMock{response: "[1] 你好世界"}, writer)

	tmpDir := t.TempDir()
	enPath := filepath.Join(tmpDir, "m.en.srt")
	zhPath, outcome, err := svc.translateAndPersist(context.Background(), "job-1", models.SubtitleRunMediaMovie, uuidB, twoCueSRT,
		enPath, filepath.Join(tmpDir, "m.mkv"), tmpDir, true)
	require.NoError(t, err)

	assert.NotEmpty(t, zhPath, "the mixed file is still placed — partial value is not discarded")
	assert.True(t, outcome.Partial())
	assert.Equal(t, 1, outcome.EnglishKeptBlocks)

	require.Len(t, writer.calls, 1)
	call := writer.calls[0]
	assert.Equal(t, models.SubtitleStatusUntranslated, call.Status,
		"partial demotes the verdict — found would be a lying badge")
	assert.Equal(t, enPath, call.Path, "the row records the EN SRT so resume can re-translate from source")
	assert.Equal(t, "en", call.Language)
}

// bugfix-j / Rule 13: the partial-branch verdict write is a contract write,
// not a best-effort optimization — its failure propagates exactly like the
// found/untranslated branches.
func TestTranslateAndPersist_PartialWritebackFailurePropagates(t *testing.T) {
	writer := &fakeSubtitleWriter{err: errors.New("db locked")}
	svc := newWriterWiredService(t, &translationIntegrationMock{response: "[1] 你好世界"}, writer)

	tmpDir := t.TempDir()
	_, _, err := svc.translateAndPersist(context.Background(), "job-1", models.SubtitleRunMediaMovie, uuidB, twoCueSRT,
		filepath.Join(tmpDir, "m.en.srt"), filepath.Join(tmpDir, "m.mkv"), tmpDir, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "update subtitle status")
}

// bugfix-j regression pin: a FULL success still records found/zh-Hant — the
// v3 bump narrows `found` to full translations without touching its shape
// (TestTranslateAndPersist_SuccessWritesBack pins the write itself; this pins
// the outcome signal that guards it).
func TestTranslateAndPersist_FullSuccessOutcomeNotPartial(t *testing.T) {
	writer := &fakeSubtitleWriter{}
	svc := newWriterWiredService(t, &translationIntegrationMock{response: "[1] 你好世界"}, writer)

	tmpDir := t.TempDir()
	zhPath, outcome, err := svc.translateAndPersist(context.Background(), "job-1", models.SubtitleRunMediaMovie, uuidB, genTestSRT,
		filepath.Join(tmpDir, "m.en.srt"), filepath.Join(tmpDir, "m.mkv"), tmpDir, true)
	require.NoError(t, err)
	require.NotEmpty(t, zhPath)

	assert.False(t, outcome.Partial())
	require.Len(t, writer.calls, 1)
	assert.Equal(t, models.SubtitleStatusFound, writer.calls[0].Status)
}

// bugfix-j CR M2: the complete-payload contract is unit-tested via the
// extracted pure builder — partial terminals carry the zh-TW message + the
// additive partial/english_kept_blocks keys; full successes carry NEITHER.
func TestBuildCompleteData_PartialTerminal(t *testing.T) {
	data := buildCompleteData("job-1", uuidB, "/m/a.en.srt", "/m/a.zh-Hant.srt", "3s", false,
		TranslationOutcome{EnglishKeptBlocks: 5, TotalBlocks: 100})

	msg, _ := data["message"].(string)
	assert.Contains(t, msg, "部分翻譯失敗", "AC #3: the terminal message says so")
	assert.Contains(t, msg, "5 句保留英文")
	assert.NotContains(t, msg, "complete", "CR L1: no mixed-script message")
	assert.Equal(t, true, data["partial"], "CR H2: FE verdict line keys off this, not zh_srt_path")
	assert.Equal(t, 5, data["english_kept_blocks"])
	assert.Equal(t, "/m/a.zh-Hant.srt", data["zh_srt_path"], "the mixed file is still announced — it exists")
}

func TestBuildCompleteData_FullSuccessOmitsPartialKeys(t *testing.T) {
	data := buildCompleteData("job-1", uuidB, "/m/a.en.srt", "/m/a.zh-Hant.srt", "3s", false,
		TranslationOutcome{EnglishKeptBlocks: 0, TotalBlocks: 100})

	_, hasPartial := data["partial"]
	_, hasKept := data["english_kept_blocks"]
	assert.False(t, hasPartial, "absent = full success (consumers must not read absence as 0-vs-missing)")
	assert.False(t, hasKept)
	msg, _ := data["message"].(string)
	assert.Equal(t, "轉錄完成", msg)
}

func TestBuildCompleteData_ResumedPartial(t *testing.T) {
	data := buildCompleteData("job-1", uuidB, "/m/a.en.srt", "/m/a.zh-Hant.srt", "3s", true,
		TranslationOutcome{EnglishKeptBlocks: 2, TotalBlocks: 40})
	msg, _ := data["message"].(string)
	assert.Contains(t, msg, "翻譯完成（自既有英文字幕續跑）", "CR sub-2-2a L1 held on the resumed path")
	assert.Contains(t, msg, "部分翻譯失敗，2 句保留英文")
}

// bugfix-j CR L2: the partial verdict arm must hit the EPISODE writeback
// dispatch too — writing an episode verdict into the movies table is the
// silent-corruption class 9R-10a red line ① exists for.
func TestTranslateAndPersist_PartialTranslation_EpisodeWritesEpisodeWriter(t *testing.T) {
	movieWriter := &fakeSubtitleWriter{}
	episodeWriter := &fakeEpisodeWriter{}
	svc := newWriterWiredService(t, &translationIntegrationMock{response: "[1] 你好世界"}, movieWriter)
	svc.SetEpisodeSubtitleStatusWriter(episodeWriter)

	tmpDir := t.TempDir()
	enPath := filepath.Join(tmpDir, "e.en.srt")
	zhPath, outcome, err := svc.translateAndPersist(context.Background(), "job-1", models.SubtitleRunMediaEpisode, uuidC, twoCueSRT,
		enPath, filepath.Join(tmpDir, "e.mkv"), tmpDir, true)
	require.NoError(t, err)
	require.NotEmpty(t, zhPath)
	require.True(t, outcome.Partial())

	assert.Empty(t, movieWriter.calls, "episode verdict must NOT touch the movie writer (9R-10a red line ①)")
	require.Len(t, episodeWriter.calls, 1)
	call := episodeWriter.calls[0]
	assert.Equal(t, uuidC, call.ID)
	assert.Equal(t, models.SubtitleStatusUntranslated, call.Status)
	assert.Equal(t, enPath, call.Path)
	assert.Equal(t, "en", call.Language)
}

// translationQueueMock returns a DIFFERENT response per CompleteText call —
// multi-batch partial cases need per-batch responses, which the single-shot
// translationIntegrationMock cannot express. Calls beyond the queue fail loud.
type translationQueueMock struct {
	responses []string
	callCount int
}

func (m *translationQueueMock) CompleteText(ctx context.Context, systemPrompt, userPrompt string, maxTokens int) (string, error) {
	m.callCount++
	if m.callCount > len(m.responses) {
		return "", fmt.Errorf("translationQueueMock: unexpected call %d (queued %d)", m.callCount, len(m.responses))
	}
	return m.responses[m.callCount-1], nil
}

// bugfix-j H3 ruling (option B「門檻制」): an IMMATERIAL residue — under one
// batch's worth AND under 5% of this item's own cue count — keeps the verdict
// `found` (zh path, zh-Hant) so a 1-cue miss doesn't force a full re-run. The
// residue is still disclosed: outcome.Partial() stays true and rides the SSE
// complete payload. 25 cues / 1 miss = 4% < 5%, 1 < batch size 10.
func TestTranslateAndPersist_BelowThresholdPartialStaysFound(t *testing.T) {
	require.Equal(t, 10, prompts.SubtitleTranslatorBatchSize,
		"the 3-batch layout below assumes batch size 10")

	// 25-cue SRT → batches [1-10], [11-20], [21-25].
	var srt strings.Builder
	for i := 1; i <= 25; i++ {
		fmt.Fprintf(&srt, "%d\n00:00:%02d,000 --> 00:00:%02d,500\nEnglish line %d\n\n", i, i, i, i)
	}
	batchResp := func(from, to, skip int) string {
		var b strings.Builder
		for i := from; i <= to; i++ {
			if i == skip {
				continue
			}
			fmt.Fprintf(&b, "[%d] 中文第%d句\n", i, i)
		}
		return b.String()
	}
	mock := &translationQueueMock{responses: []string{
		batchResp(1, 10, 0),
		batchResp(11, 20, 0),
		batchResp(21, 25, 23), // cue 23 missing → the only English residue
	}}

	writer := &fakeSubtitleWriter{}
	svc := newWriterWiredService(t, mock, writer)

	tmpDir := t.TempDir()
	zhPath, outcome, err := svc.translateAndPersist(context.Background(), "job-1", models.SubtitleRunMediaMovie, uuidB, srt.String(),
		filepath.Join(tmpDir, "m.en.srt"), filepath.Join(tmpDir, "m.mkv"), tmpDir, true)
	require.NoError(t, err)
	require.NotEmpty(t, zhPath)
	assert.Equal(t, 3, mock.callCount, "25 cues at batch size 10 = 3 batches")

	assert.True(t, outcome.Partial(), "disclosure is unconditional — the 1-cue miss is still reported")
	assert.Equal(t, 1, outcome.EnglishKeptBlocks)
	assert.Equal(t, 25, outcome.TotalBlocks)
	assert.False(t, outcome.DemotesVerdict(), "1/25 = 4% < 5% and 1 < 10 — immaterial")

	require.Len(t, writer.calls, 1)
	call := writer.calls[0]
	assert.Equal(t, models.SubtitleStatusFound, call.Status,
		"below-threshold residue keeps found — a full re-run for 1 cue is worse than the miss")
	assert.Equal(t, zhPath, call.Path)
	assert.Equal(t, "zh-Hant", call.Language)
}
