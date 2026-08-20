package services

// Story sub-3-2 — episode ASR fallback: TranscriptionService media-type
// awareness. The AC #9 service-side cases (c)(d)(e)(f) live here; the leg and
// sweep cases live in the subtitle package.
//
// Media-id fixture convention (9R-18 AC 7): UUID STRINGS, reusing the uuid*
// consts from generation_batch_test.go.

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
)

type episodeWriteCall struct {
	ID       string
	Status   models.SubtitleStatus
	Path     string
	Language string
}

// fakeEpisodeWriter satisfies EpisodeSubtitleStatusWriter (mirrors
// fakeSubtitleWriter — the async path writes from a goroutine).
type fakeEpisodeWriter struct {
	mu    sync.Mutex
	calls []episodeWriteCall
	err   error
}

func (f *fakeEpisodeWriter) UpdateEpisodeSubtitleStatus(_ context.Context, id string, status models.SubtitleStatus, path, language string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, episodeWriteCall{ID: id, Status: status, Path: path, Language: language})
	return f.err
}

// fakeEpisodeStateReader satisfies EpisodeSubtitleStateReader.
type fakeEpisodeStateReader struct {
	episode *models.Episode
	err     error
}

func (f *fakeEpisodeStateReader) FindByID(_ context.Context, _ string) (*models.Episode, error) {
	return f.episode, f.err
}

func untranslatedEpisode(id, enPath string) *models.Episode {
	e := &models.Episode{ID: id}
	e.SubtitleStatus = models.SubtitleStatusUntranslated
	e.SubtitlePath = models.NewNullString(enPath)
	e.SubtitleLanguage = models.NewNullString("en")
	return e
}

// ─── AC #9 (c): episode outcomes write through the EPISODE writer ───────────

func TestTranslateAndPersist_EpisodeSuccessWritesThroughEpisodeWriter(t *testing.T) {
	movieWriter := &fakeSubtitleWriter{}
	episodeWriter := &fakeEpisodeWriter{}
	svc := newWriterWiredService(t, &translationIntegrationMock{response: "[1] 你好世界"}, movieWriter)
	svc.SetEpisodeSubtitleStatusWriter(episodeWriter)

	tmpDir := t.TempDir()
	zhPath, _, err := svc.translateAndPersist(context.Background(), "job-1", models.SubtitleRunMediaEpisode, uuidB,
		genTestSRT, filepath.Join(tmpDir, "e.en.srt"), filepath.Join(tmpDir, "e.mkv"), tmpDir, true)
	require.NoError(t, err)
	require.NotEmpty(t, zhPath)

	require.Len(t, episodeWriter.calls, 1, "the episode outcome goes to the episode repository")
	call := episodeWriter.calls[0]
	assert.Equal(t, uuidB, call.ID)
	assert.Equal(t, models.SubtitleStatusFound, call.Status)
	assert.Equal(t, zhPath, call.Path)
	assert.Equal(t, "zh-Hant", call.Language)
	assert.Empty(t, movieWriter.snapshot(), "the movie writer must never see an episode id")
}

func TestTranslateAndPersist_EpisodeKeyUnconfiguredWritesUntranslated(t *testing.T) {
	movieWriter := &fakeSubtitleWriter{}
	episodeWriter := &fakeEpisodeWriter{}
	svc := newWriterWiredService(t, nil /* no translation service — key unconfigured */, movieWriter)
	svc.SetEpisodeSubtitleStatusWriter(episodeWriter)

	enPath := filepath.Join(t.TempDir(), "e.en.srt")
	zhPath, _, err := svc.translateAndPersist(context.Background(), "job-1", models.SubtitleRunMediaEpisode, uuidB,
		genTestSRT, enPath, filepath.Join(t.TempDir(), "e.mkv"), t.TempDir(), true)
	require.NoError(t, err)
	assert.Empty(t, zhPath)

	require.Len(t, episodeWriter.calls, 1)
	call := episodeWriter.calls[0]
	assert.Equal(t, models.SubtitleStatusUntranslated, call.Status)
	assert.Equal(t, enPath, call.Path)
	assert.Equal(t, "en", call.Language)
	assert.Empty(t, movieWriter.snapshot())
}

// ─── AC #9 (f): partial wiring fails loudly, not with the movie repo's lie ──

func TestTranslateAndPersist_EpisodeNilWriterFailsPrecisely(t *testing.T) {
	svc := newWriterWiredService(t, &translationIntegrationMock{response: "[1] 你好"}, &fakeSubtitleWriter{})
	// deliberately NO episode writer wired

	_, _, err := svc.translateAndPersist(context.Background(), "job-1", models.SubtitleRunMediaEpisode, uuidB,
		genTestSRT, filepath.Join(t.TempDir(), "e.en.srt"), filepath.Join(t.TempDir(), "e.mkv"), t.TempDir(), true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "episode subtitle writer not wired",
		"the failure must name the wiring gap, not report a phantom missing movie row")
}

// ─── AC #9 (e): default media type is MOVIE — existing callers unchanged ────

func TestTranslateAndPersist_DefaultMediaTypeHitsMovieWriter(t *testing.T) {
	movieWriter := &fakeSubtitleWriter{}
	episodeWriter := &fakeEpisodeWriter{}
	svc := newWriterWiredService(t, &translationIntegrationMock{response: "[1] 你好"}, movieWriter)
	svc.SetEpisodeSubtitleStatusWriter(episodeWriter)

	tmpDir := t.TempDir()
	// The movie media type is what every legacy call site resolves to when
	// WithMediaType is omitted (pinned end-to-end below via RunTranscription).
	zhPath, _, err := svc.translateAndPersist(context.Background(), "job-1", models.SubtitleRunMediaMovie, uuidB,
		genTestSRT, filepath.Join(tmpDir, "m.en.srt"), filepath.Join(tmpDir, "m.mkv"), tmpDir, true)
	require.NoError(t, err)
	require.NotEmpty(t, zhPath)

	require.Len(t, movieWriter.snapshot(), 1)
	assert.Empty(t, episodeWriter.calls, "a movie run must never reach the episode writer")
}

// ─── AC #9 (d): episode translate-only resume ───────────────────────────────

// Mirrors TestRunTranscription_ResumeSkipsASRAndTranslatesOnly: extract/ASR
// would fail loudly (missing media file, bogus key) — a completed run PROVES
// the resume path skipped them for the episode media type too.
func TestRunTranscription_EpisodeResumeSkipsASRAndTranslatesOnly(t *testing.T) {
	tmp := t.TempDir()
	enPath := filepath.Join(tmp, "S01E01.en.srt")
	require.NoError(t, os.WriteFile(enPath, []byte(genTestSRT), 0644))

	movieWriter := &fakeSubtitleWriter{}
	episodeWriter := &fakeEpisodeWriter{}
	episodeReader := &fakeEpisodeStateReader{episode: untranslatedEpisode(uuidD, enPath)}
	svc := resumeService(t, &translationIntegrationMock{response: "[1] 你好世界"}, movieWriter, nil)
	svc.SetEpisodeSubtitleStatusWriter(episodeWriter)
	svc.SetEpisodeSubtitleStateReader(episodeReader)

	err := svc.RunTranscription(context.Background(), uuidD,
		filepath.Join(tmp, "S01E01.mkv") /* does NOT exist — extract would fail */, tmp,
		WithTranslation(), WithMediaType(models.SubtitleRunMediaEpisode))
	require.NoError(t, err, "episode resume must complete without touching extract/ASR")

	require.NotEmpty(t, episodeWriter.calls)
	last := episodeWriter.calls[len(episodeWriter.calls)-1]
	assert.Equal(t, models.SubtitleStatusFound, last.Status)
	assert.Equal(t, "zh-Hant", last.Language)
	assert.Empty(t, movieWriter.snapshot(), "the movie reader/writer pair must stay out of an episode run")
}

// The episode resume is status-bound exactly like the movie one: a nil episode
// reader means no resume — the full pipeline is attempted (and fails on the
// missing media file, which is the proof).
func TestRunTranscription_EpisodeNilReaderSkipsResumeCheck(t *testing.T) {
	tmp := t.TempDir()
	enPath := filepath.Join(tmp, "S01E01.en.srt")
	require.NoError(t, os.WriteFile(enPath, []byte(genTestSRT), 0644))

	svc := resumeService(t, &translationIntegrationMock{response: "[1] 你好"}, &fakeSubtitleWriter{}, nil)
	// movie reader wired, episode reader NOT — an episode run must not consult
	// the movie reader and must therefore attempt the full pipeline.
	svc.SetSubtitleStateReader(&fakeStateReader{movie: untranslatedMovie(uuidD, enPath)})

	err := svc.RunTranscription(context.Background(), uuidD,
		filepath.Join(tmp, "S01E01.mkv"), tmp,
		WithTranslation(), WithMediaType(models.SubtitleRunMediaEpisode))
	require.Error(t, err, "episode run must not resume off the MOVIE reader's row")
	// CR L2: pin the FAILURE MODE too — the full pipeline was attempted (extract
	// fails on the missing media file), not rejected at the availability gate.
	assert.NotErrorIs(t, err, ErrTranscriptionDisabled,
		"the entry gate must pass (ASR is available); the error comes from the attempted full run")
}
