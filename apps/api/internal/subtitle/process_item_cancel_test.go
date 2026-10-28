package subtitle

// disc-2026-10-solo-generation-cancel-api: what a CALLER cancellation does to
// the media row under ProcessItemOptions.RestoreMediaOnCancel (AC #5), and
// that a cancellation arriving after the sidecar is placed no longer turns a
// finished run into 「中途停止」 (AC #3e).

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/ai/prompts"
	"github.com/vido/api/internal/models"
)

// livingMediaStore is a MediaStore whose Load reflects every status write —
// the real row's behaviour, which settleCancelledMedia's re-read depends on.
// (fakeMediaStore's Load always returns the original item.)
type livingMediaStore struct {
	mu     sync.Mutex
	item   MediaItem
	writes []statusWrite
}

func (s *livingMediaStore) Load(_ context.Context, _ MediaRef) (*MediaItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := s.item
	return &cp, nil
}

func (s *livingMediaStore) SetSubtitleStatus(_ context.Context, _ MediaRef, status models.SubtitleStatus, path, language string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.item.SubtitleStatus, s.item.SubtitlePath, s.item.SubtitleLanguage = status, path, language
	s.writes = append(s.writes, statusWrite{status: status, path: path, language: language})
	return nil
}

func (s *livingMediaStore) row() statusWrite {
	s.mu.Lock()
	defer s.mu.Unlock()
	return statusWrite{status: s.item.SubtitleStatus, path: s.item.SubtitlePath, language: s.item.SubtitleLanguage}
}

// newLivingStore seeds the row as a previous successful generation left it.
func newLivingStore(h *itemHarness, status models.SubtitleStatus, path, language string) *livingMediaStore {
	item := *h.media.item
	item.SubtitleStatus, item.SubtitlePath, item.SubtitleLanguage = status, path, language
	return &livingMediaStore{item: item}
}

// ctxRunStore refuses writes on a cancelled ctx, as database/sql does.
type ctxRunStore struct{ *fakeRunStore }

func (s ctxRunStore) Update(ctx context.Context, run *models.SubtitleRun) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.fakeRunStore.Update(ctx, run)
}

// writingTranscriber stands in for the ASR leg stopped mid-translation: it
// writes the `untranslated` + English SRT verdict itself (as
// TranscriptionService.translateAndPersist does on a dead ctx), then returns
// the cancellation.
type writingTranscriber struct {
	store  *livingMediaStore
	cancel context.CancelFunc
}

func (f *writingTranscriber) Available() bool { return true }

func (f *writingTranscriber) Transcribe(ctx context.Context, ref MediaRef, _, _ string) error {
	f.cancel()
	_ = f.store.SetSubtitleStatus(ctx, ref, models.SubtitleStatusUntranslated, "/media/show/ep1.en.srt", "en")
	return context.Canceled
}

func cancellingTranslator(cancel context.CancelFunc) func(int, []prompts.SubtitleTranslatorBlock) (map[int]string, ai.CompletionUsage, error) {
	return func(int, []prompts.SubtitleTranslatorBlock) (map[int]string, ai.CompletionUsage, error) {
		cancel()
		return nil, ai.CompletionUsage{}, context.Canceled
	}
}

// AC #5a: a cancelled RE-generation keeps pointing at the old subtitle.
func TestProcessItem_CancelledRegenerationRestoresTheRow(t *testing.T) {
	h := newItemHarness(t, translateDecision("Good morning."))
	store := newLivingStore(h, models.SubtitleStatusFound, "/media/show/ep1.zh-Hant.srt", "zh-Hant")
	h.pipeline.media = store
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h.trans.fn = cancellingTranslator(cancel)

	_, err := h.pipeline.ProcessItem(ctx, h.ref, ProcessItemOptions{Regenerate: true, RestoreMediaOnCancel: true})
	require.ErrorIs(t, err, context.Canceled)

	last := h.runs.lastUpdate(t)
	assert.Equal(t, models.SubtitleRunFailed, last.Status)
	assert.True(t, strings.HasPrefix(last.ErrorMessage, CancelledRunPrefix), last.ErrorMessage)
	assert.Equal(t, models.SubtitleRunFailureCancelled, models.SubtitleRunFailureReason(last.ErrorMessage))
	assert.Equal(t, statusWrite{models.SubtitleStatusFound, "/media/show/ep1.zh-Hant.srt", "zh-Hant"}, store.row(),
		"the old subtitle file is still on disk — the row must still say so")
	for _, w := range store.writes {
		assert.NotEqual(t, models.SubtitleStatusNotSearched, w.status, "the row never passes through not_searched")
	}
}

// AC #5b: the ASR leg's resume key survives the cancellation.
func TestProcessItem_CancelledASRLegKeepsTheUntranslatedVerdict(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h := newItemHarness(t, RouteDecision{Kind: RouteNoTextSource, Reason: "no text track"})
	store := newLivingStore(h, models.SubtitleStatusNotSearched, "", "")
	h.pipeline.media = store
	h.pipeline.asr = &writingTranscriber{store: store, cancel: cancel}

	_, err := h.pipeline.ProcessItem(ctx, h.ref, ProcessItemOptions{Regenerate: true, RestoreMediaOnCancel: true})
	require.ErrorIs(t, err, context.Canceled)

	assert.True(t, strings.HasPrefix(h.runs.lastUpdate(t).ErrorMessage, CancelledRunPrefix))
	assert.Equal(t, statusWrite{models.SubtitleStatusUntranslated, "/media/show/ep1.en.srt", "en"}, store.row(),
		"untranslated + the English SRT path is what lets the next click skip the paid speech recognition")
}

// AC #5c: a Load-time status that is itself in flight falls back to the floor.
func TestProcessItem_CancelledRunWithAStrandedLoadStatusFallsBackToNotSearched(t *testing.T) {
	h := newItemHarness(t, translateDecision("Good morning."))
	store := newLivingStore(h, models.SubtitleStatusTranslating, "", "")
	h.pipeline.media = store
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h.trans.fn = cancellingTranslator(cancel)

	_, err := h.pipeline.ProcessItem(ctx, h.ref, ProcessItemOptions{Regenerate: true, RestoreMediaOnCancel: true})
	require.Error(t, err)
	assert.Equal(t, statusWrite{models.SubtitleStatusNotSearched, "", ""}, store.row())
}

// AC #5d: without the option (batch, pool, free lane) nothing changes.
func TestProcessItem_CancelWithoutTheOptionStillParksAtNotSearched(t *testing.T) {
	h := newItemHarness(t, translateDecision("Good morning."))
	store := newLivingStore(h, models.SubtitleStatusFound, "/media/show/ep1.zh-Hant.srt", "zh-Hant")
	h.pipeline.media = store
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h.trans.fn = cancellingTranslator(cancel)

	_, err := h.pipeline.ProcessItem(ctx, h.ref, ProcessItemOptions{Regenerate: true})
	require.Error(t, err)
	assert.Equal(t, statusWrite{models.SubtitleStatusNotSearched, "", ""}, store.row())
}

// A non-cancellation failure keeps failItem's not_searched even with the
// option: the option is about cancels only.
func TestProcessItem_OrdinaryFailureWithTheOptionStillParksAtNotSearched(t *testing.T) {
	h := newItemHarness(t, translateDecision("Good morning."))
	store := newLivingStore(h, models.SubtitleStatusFound, "/media/show/ep1.zh-Hant.srt", "zh-Hant")
	h.pipeline.media = store
	h.placer.err = assert.AnError

	_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{Regenerate: true, RestoreMediaOnCancel: true})
	require.Error(t, err)
	assert.Equal(t, statusWrite{models.SubtitleStatusNotSearched, "", ""}, store.row())
}

// cancelOnPlacePlacer places successfully and THEN the caller cancels — the
// click on 停止 that lands while the file is being written.
type cancelOnPlacePlacer struct {
	*recordingPlacer
	cancel context.CancelFunc
}

func (p cancelOnPlacePlacer) Place(req PlaceRequest) (*PlaceResult, error) {
	res, err := p.recordingPlacer.Place(req)
	p.cancel()
	return res, err
}

// AC #3e: once the sidecar is on disk the run is complete.
func TestProcessItem_CancelAfterThePlaceStillCompletes(t *testing.T) {
	h := newItemHarness(t, translateDecision("Good morning."))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h.pipeline.placer = cancelOnPlacePlacer{recordingPlacer: h.placer, cancel: cancel}
	h.pipeline.runs = ctxRunStore{h.runs}

	out, err := h.pipeline.ProcessItem(ctx, h.ref, ProcessItemOptions{Regenerate: true, RestoreMediaOnCancel: true})
	require.NoError(t, err)
	require.NotNil(t, out.Run)
	assert.Equal(t, models.SubtitleRunCompleted, h.runs.lastUpdate(t).Status)
	assert.Equal(t, models.SubtitleStatusFound, h.media.writes[len(h.media.writes)-1].status)
}

// AC #3e, ASR lane: the leg returned success, then the cancel arrived.
func TestProcessItem_CancelAfterTheASRLegSucceededStillCompletes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h := newItemHarness(t, RouteDecision{Kind: RouteNoTextSource, Reason: "no text track"})
	h.pipeline.runs = ctxRunStore{h.runs}
	h.pipeline.asr = &ctxTranscriber{fn: func(context.Context, MediaRef) { cancel() }}

	_, err := h.pipeline.ProcessItem(ctx, h.ref, ProcessItemOptions{Regenerate: true, RestoreMediaOnCancel: true})
	require.NoError(t, err)
	assert.Equal(t, models.SubtitleRunCompleted, h.runs.lastUpdate(t).Status)
}
