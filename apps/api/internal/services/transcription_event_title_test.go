package services

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/sse"
)

// ─── Story dsr-6d-c-2 AC #8: `transcription_*` events carry the media title ──
//
// The generation workspace's live log could never say WHICH film an event was
// about: no transcription event carried a title, although a solo run already
// resolves one for the Activity page (resolveActivityTitle). Every event now
// carries `title` — the resolved title for a solo run, "" for a batch/pipeline
// run (its titles reach the client through changed_item) and "" whenever the
// lookup fell back to the raw media id (a UUID is never a title).

// countingStateReader counts FindByID calls so a test can prove the batch path
// did not start querying for a title.
type countingStateReader struct {
	mu    sync.Mutex
	calls int
	movie *models.Movie
}

func (r *countingStateReader) FindByID(_ context.Context, _ string) (*models.Movie, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	return r.movie, nil
}

func (r *countingStateReader) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls
}

// titleTestService wires a real hub so the events can be read back.
func titleTestService(t *testing.T, completer ai.TextCompleter, reader SubtitleStateReader) (*TranscriptionService, *sse.Client) {
	t.Helper()
	hub := sse.NewHub()
	t.Cleanup(func() { hub.Close() })
	client := hub.Register()
	require.Eventually(t, func() bool { return hub.ClientCount() == 1 }, 2*time.Second, time.Millisecond,
		"SSE client never registered")
	extractor := &AudioExtractorService{available: true, semaphore: make(chan struct{}, 1)}
	svc := NewTranscriptionService(extractor, ai.NewWhisperClient("bogus-test-key"), hub, nil)
	if completer != nil {
		svc.SetTranslationService(NewTranslationService(completer, nil))
	}
	svc.SetSubtitleStatusWriter(&fakeSubtitleWriter{})
	if reader != nil {
		svc.SetSubtitleStateReader(reader)
	}
	return svc, client
}

// collectUntilTerminal reads transcription events for mediaID until the
// complete/failed terminal arrives.
func collectUntilTerminal(t *testing.T, client *sse.Client, mediaID string) []sse.Event {
	t.Helper()
	var got []sse.Event
	deadline := time.After(5 * time.Second)
	for {
		select {
		case ev := <-client.Events:
			data, ok := ev.Data.(map[string]interface{})
			if !ok || data["media_id"] != mediaID {
				continue
			}
			got = append(got, ev)
			if ev.Type == EventTranscriptionComplete || ev.Type == EventTranscriptionFailed {
				return got
			}
		case <-deadline:
			t.Fatalf("no terminal transcription event for %s; got %d events", mediaID, len(got))
			return nil
		}
	}
}

func titlesOf(t *testing.T, events []sse.Event) map[sse.EventType][]interface{} {
	t.Helper()
	out := map[sse.EventType][]interface{}{}
	for _, ev := range events {
		data := ev.Data.(map[string]interface{})
		title, present := data["title"]
		require.Truef(t, present, "%s payload has no `title` key: %v", ev.Type, data)
		out[ev.Type] = append(out[ev.Type], title)
	}
	return out
}

func TestEventTitle(t *testing.T) {
	svc := newActivityTestService()
	_, err := svc.acquireJob("solo-1", "沙丘：第二部", true)
	require.NoError(t, err)
	_, err = svc.acquireJob("solo-fallback", "solo-fallback", true) // resolveActivityTitle's miss
	require.NoError(t, err)
	_, err = svc.acquireJob("batch-1", "", false)
	require.NoError(t, err)
	// A non-solo job that somehow carries a title still sends none: only a solo
	// run's title is a promise that the event is not a batch item's.
	_, err = svc.acquireJob("batch-titled", "某片名", false)
	require.NoError(t, err)

	assert.Equal(t, "沙丘：第二部", svc.eventTitle("solo-1"))
	assert.Equal(t, "", svc.eventTitle("solo-fallback"), "a raw media id is never a title")
	assert.Equal(t, "", svc.eventTitle("batch-1"), "batch titles travel through changed_item")
	assert.Equal(t, "", svc.eventTitle("batch-titled"), "only a solo run's title is sent")
	assert.Equal(t, "", svc.eventTitle("not-running"))
}

// A solo translate-only resume emits translation_progress (the 0% frame and the
// per-chunk frames) and the complete terminal — every one names the film.
func TestStartTranscription_EveryEventCarriesTheResolvedTitle(t *testing.T) {
	tmp := t.TempDir()
	enPath := filepath.Join(tmp, "Movie.en.srt")
	require.NoError(t, os.WriteFile(enPath, []byte(genTestSRT), 0644))
	movie := untranslatedMovie(uuidD, enPath)
	movie.Title = "沙丘：第二部"

	svc, client := titleTestService(t, &translationIntegrationMock{response: "[1] 你好世界"},
		&fakeStateReader{movie: movie})

	_, err := svc.StartTranscription(context.Background(), uuidD,
		filepath.Join(tmp, "Movie.mkv"), tmp, WithTranslation())
	require.NoError(t, err)

	byType := titlesOf(t, collectUntilTerminal(t, client, uuidD))
	require.NotEmpty(t, byType[EventTranscriptionTranslating])
	for _, title := range byType[EventTranscriptionTranslating] {
		assert.Equal(t, "沙丘：第二部", title)
	}
	assert.Equal(t, []interface{}{"沙丘：第二部"}, byType[EventTranscriptionComplete])
}

// The full path: the extracting event and the failure that follows (the media
// file does not exist) both carry the title.
func TestStartTranscription_ExtractingAndFailedCarryTheTitle(t *testing.T) {
	tmp := t.TempDir()
	movie := &models.Movie{ID: uuidD, Title: "奧本海默"}

	svc, client := titleTestService(t, nil, &fakeStateReader{movie: movie})

	_, err := svc.StartTranscription(context.Background(), uuidD,
		filepath.Join(tmp, "missing.mkv"), tmp)
	require.NoError(t, err)

	byType := titlesOf(t, collectUntilTerminal(t, client, uuidD))
	assert.Equal(t, []interface{}{"奧本海默"}, byType[EventTranscriptionExtracting])
	assert.Equal(t, []interface{}{"奧本海默"}, byType[EventTranscriptionFailed])
}

// No title could be resolved → resolveActivityTitle fell back to the media id.
// The event must say "" — never ship a UUID as a title.
func TestStartTranscription_UnresolvedTitleIsEmptyNotTheMediaID(t *testing.T) {
	tmp := t.TempDir()

	svc, client := titleTestService(t, nil, nil)

	_, err := svc.StartTranscription(context.Background(), uuidD,
		filepath.Join(tmp, "missing.mkv"), tmp)
	require.NoError(t, err)

	for typ, titles := range titlesOf(t, collectUntilTerminal(t, client, uuidD)) {
		for _, title := range titles {
			assert.Equalf(t, "", title, "%s leaked a non-title", typ)
		}
	}
}

// The batch/pipeline path sends "" and does NOT start looking titles up. With
// translate off there is no resume check either, so the row is never read.
func TestRunTranscription_BatchPathSendsEmptyTitleWithoutExtraLookups(t *testing.T) {
	tmp := t.TempDir()
	reader := &countingStateReader{movie: &models.Movie{ID: uuidD, Title: "不該被查的片名"}}

	svc, client := titleTestService(t, nil, reader)

	err := svc.RunTranscription(context.Background(), uuidD, filepath.Join(tmp, "missing.mkv"), tmp)
	require.Error(t, err)

	for typ, titles := range titlesOf(t, collectUntilTerminal(t, client, uuidD)) {
		for _, title := range titles {
			assert.Equalf(t, "", title, "%s on the batch path must carry an empty title", typ)
		}
	}
	assert.Equal(t, 0, reader.count(), "the batch path must not look a title up")
}
