package services

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/models"
)

func TestTranscriptionService_IsAvailable_NoExtractor(t *testing.T) {
	svc := NewTranscriptionService(nil, nil, nil, nil)
	assert.False(t, svc.IsAvailable())
}

func TestTranscriptionService_IsAvailable_ExtractorNotAvailable(t *testing.T) {
	extractor := &AudioExtractorService{available: false}
	svc := NewTranscriptionService(extractor, nil, nil, nil)
	assert.False(t, svc.IsAvailable())
}

func TestTranscriptionService_IsAvailable_NoWhisper(t *testing.T) {
	extractor := &AudioExtractorService{available: true}
	svc := NewTranscriptionService(extractor, nil, nil, nil)
	assert.False(t, svc.IsAvailable())
}

func TestTranscriptionService_IsInProgress_Empty(t *testing.T) {
	svc := NewTranscriptionService(nil, nil, nil, nil)
	assert.False(t, svc.IsInProgress(1))
}

func TestTranscriptionService_IsInProgress_Set(t *testing.T) {
	svc := NewTranscriptionService(nil, nil, nil, nil)
	svc.mu.Lock()
	svc.inProgress[42] = "job-123"
	svc.mu.Unlock()

	assert.True(t, svc.IsInProgress(42))
	assert.False(t, svc.IsInProgress(99))
}

func TestTranscriptionService_StartTranscription_Disabled(t *testing.T) {
	svc := NewTranscriptionService(nil, nil, nil, nil)
	_, err := svc.StartTranscription(context.Background(), 1, "/test.mkv", "/media")
	assert.ErrorIs(t, err, ErrTranscriptionDisabled)
}

func TestTranscriptionService_StartTranscription_AlreadyInProgress(t *testing.T) {
	extractor := &AudioExtractorService{
		available: true,
		semaphore: make(chan struct{}, 1),
	}
	whisperClient := ai.NewWhisperClient("test-key")

	svc := NewTranscriptionService(extractor, whisperClient, nil, nil)

	// Manually set in-progress
	svc.mu.Lock()
	svc.inProgress[42] = "existing-job"
	svc.mu.Unlock()

	_, err := svc.StartTranscription(context.Background(), 42, "/test.mkv", "/media")
	assert.ErrorIs(t, err, ErrTranscriptionInProgress)
}

func TestTranscriptionService_IsAvailable_FullyWired(t *testing.T) {
	extractor := &AudioExtractorService{available: true, semaphore: make(chan struct{}, 1)}
	whisperClient := ai.NewWhisperClient("test-key")
	svc := NewTranscriptionService(extractor, whisperClient, nil, nil)
	assert.True(t, svc.IsAvailable())
}

func TestTranscriptionService_BroadcastEvent_NilHub(t *testing.T) {
	// broadcastEvent with nil sseHub should not panic
	svc := NewTranscriptionService(nil, nil, nil, nil)
	assert.NotPanics(t, func() {
		svc.broadcastEvent(EventTranscriptionProgress, map[string]interface{}{"test": true})
	})
}

func TestTranscriptionService_FailJob_NilHub(t *testing.T) {
	// failJob with nil sseHub should not panic
	svc := NewTranscriptionService(nil, nil, nil, nil)
	assert.NotPanics(t, func() {
		svc.failJob("job-1", 1, "test error")
	})
}

func TestTranscriptionEventTypes(t *testing.T) {
	// Verify event type constants match expected SSE event names
	assert.Equal(t, "transcription_extracting", string(EventTranscriptionExtracting))
	assert.Equal(t, "transcription_progress", string(EventTranscriptionProgress))
	assert.Equal(t, "transcription_complete", string(EventTranscriptionComplete))
	assert.Equal(t, "transcription_failed", string(EventTranscriptionFailed))
}

// --- 9R-2: track-language -> ISO-639-1 mapping ---

func TestWhisperLanguageFromTrack(t *testing.T) {
	cases := map[string]string{
		"eng":   "en",
		"ENG":   "en",
		"en":    "en",
		"jpn":   "ja",
		"chi":   "zh",
		"zho":   "zh",
		"kor":   "ko",
		"fre":   "fr",
		"ger":   "de",
		"und":   "", // AC #2: auto-detect only when und
		"":      "",
		"xxx":   "", // unknown 3-letter: safer to auto-detect than mis-pin
		" eng ": "en",
	}
	for in, want := range cases {
		if got := WhisperLanguageFromTrack(in); got != want {
			t.Errorf("WhisperLanguageFromTrack(%q) = %q, want %q", in, got, want)
		}
	}
}

// --- 9R-10: Route C pipeline — glossary-aware translate → OpenCC → place ---

type fakeOpenCC struct {
	called bool
	input  []byte
}

func (f *fakeOpenCC) IsAvailable() bool { return true }
func (f *fakeOpenCC) ConvertS2TWP(content []byte) ([]byte, error) {
	f.called = true
	f.input = content
	// s2twp: 软 → 軟 (proves the safety net actually ran on the output).
	return []byte(strings.ReplaceAll(string(content), "软", "軟")), nil
}

type fakePlacer struct {
	mediaPath string
	data      []byte
	language  string
	format    string
}

func (f *fakePlacer) PlaceSubtitle(mediaFilePath string, data []byte, language, format string) (string, error) {
	f.mediaPath = mediaFilePath
	f.data = data
	f.language = language
	f.format = format
	return mediaFilePath + "." + language + "." + format, nil
}

// stubGlossaryRepo implements repository.GlossaryRepositoryInterface with only
// LookupByMedia meaningful.
type stubGlossaryRepo struct{ terms map[string]string }

func (s *stubGlossaryRepo) Upsert(ctx context.Context, t *models.GlossaryTerm) error { return nil }
func (s *stubGlossaryRepo) ListByMedia(ctx context.Context, mediaID string) ([]models.GlossaryTerm, error) {
	return nil, nil
}
func (s *stubGlossaryRepo) LookupByMedia(ctx context.Context, mediaID string, confirmedOnly bool) (map[string]string, error) {
	return s.terms, nil
}
func (s *stubGlossaryRepo) Update(ctx context.Context, id, termZh string, confirmed bool) (time.Time, error) {
	return time.Time{}, nil
}
func (s *stubGlossaryRepo) Confirm(ctx context.Context, id string) (time.Time, error) {
	return time.Time{}, nil
}
func (s *stubGlossaryRepo) Delete(ctx context.Context, id string) error { return nil }

func TestTranscriptionService_TranslateSRT_GlossaryOpenCCPlace(t *testing.T) {
	// Mock LLM returns an index-prefixed translation carrying a Simplified char
	// (软) so we can prove the OpenCC safety net ran.
	completer := &mockTranslationCompleter{response: "[1] 软體魔王獸來了"}
	translation := NewTranslationService(completer, nil)

	opencc := &fakeOpenCC{}
	placer := &fakePlacer{}

	svc := NewTranscriptionService(nil, nil, nil, nil)
	svc.SetTranslationService(translation)
	svc.SetOpenCCConverter(opencc)
	svc.SetPlacer(placer)
	svc.SetGlossaryRepository(&stubGlossaryRepo{terms: map[string]string{"Demogorgon": "魔王獸"}})

	srt := "1\n00:00:01,000 --> 00:00:03,000\nThe Demogorgon is coming\n"
	path, err := svc.translateSRT(context.Background(), "job1", 42, srt, "/media/Show.mkv", "/media")
	require.NoError(t, err)

	// Glossary-aware: the fixed rendering reached the LLM prompt.
	require.Len(t, completer.calls, 1)
	assert.Contains(t, completer.calls[0].UserPrompt, "Demogorgon → 魔王獸")

	// OpenCC safety net ran on the translated SRT.
	assert.True(t, opencc.called, "OpenCC ConvertS2TWP must be invoked")

	// Placer received the OpenCC'd content (软→軟) with zh-Hant/srt.
	assert.Equal(t, "zh-Hant", placer.language)
	assert.Equal(t, "srt", placer.format)
	assert.Contains(t, string(placer.data), "軟體魔王獸", "placed content must be the OpenCC-converted output")
	assert.NotContains(t, string(placer.data), "软")
	assert.Equal(t, "/media/Show.mkv.zh-Hant.srt", path)
}

func TestTranscriptionService_TranslateSRT_FailSoftNoDeps(t *testing.T) {
	// No glossary repo, no OpenCC, no placer → still produces a file via the
	// direct-write fallback (pipeline functions without the optional stages).
	completer := &mockTranslationCompleter{response: "[1] 你好"}
	svc := NewTranscriptionService(nil, nil, nil, nil)
	svc.SetTranslationService(NewTranslationService(completer, nil))

	dir := t.TempDir()
	srt := "1\n00:00:01,000 --> 00:00:02,000\nHi\n"
	path, err := svc.translateSRT(context.Background(), "job1", 1, srt, dir+"/Movie.mkv", dir)
	require.NoError(t, err)
	assert.FileExists(t, path)
	// No glossary → prompt has no Glossary section (unchanged behavior).
	assert.NotContains(t, completer.calls[0].UserPrompt, "Glossary")
}
