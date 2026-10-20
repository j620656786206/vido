package services

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/sse"
)

// ─── disc-2026-10-generate-keeps-user-en-srt-a ──────────────────────────────
//
// Generating subtitles for Tears of Steel replaced the official 4,764-byte
// .en.srt beside it with the speech-recognition transcript. The transcript now
// goes to <base>.vido.en.srt whenever <base>.en.srt is already there.

// userEnglishSRT stands in for the subtitle the user placed beside the film.
const userEnglishSRT = "1\n00:00:01,000 --> 00:00:03,000\nOfficial line from the release\n"

func TestSaveEnglishTranscript_WritesPlainNameWhenNothingIsThere(t *testing.T) {
	dir := t.TempDir()

	path, err := saveEnglishTranscript(dir, "Movie", genTestSRT)
	require.NoError(t, err)

	assert.Equal(t, filepath.Join(dir, "Movie.en.srt"), path)
	assertFileContent(t, path, genTestSRT)
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0644), info.Mode().Perm(), "the media server must be able to read it")
	assertOnlyFiles(t, dir, "Movie.en.srt")
}

func TestSaveEnglishTranscript_LeavesTheUsersSubtitleAlone(t *testing.T) {
	dir := t.TempDir()
	userPath := filepath.Join(dir, "Movie.en.srt")
	require.NoError(t, os.WriteFile(userPath, []byte(userEnglishSRT), 0644))

	path, err := saveEnglishTranscript(dir, "Movie", genTestSRT)
	require.NoError(t, err)

	assert.Equal(t, filepath.Join(dir, "Movie.vido.en.srt"), path)
	assertFileContent(t, userPath, userEnglishSRT)
	assertFileContent(t, path, genTestSRT)
	assertOnlyFiles(t, dir, "Movie.en.srt", "Movie.vido.en.srt")
}

// A second generation replaces Vido's own transcript — it is ours to rewrite.
func TestSaveEnglishTranscript_RewritesItsOwnTranscript(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Movie.en.srt"), []byte(userEnglishSRT), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Movie.vido.en.srt"), []byte("stale"), 0644))

	path, err := saveEnglishTranscript(dir, "Movie", genTestSRT)
	require.NoError(t, err)

	assertFileContent(t, path, genTestSRT)
	assertFileContent(t, filepath.Join(dir, "Movie.en.srt"), userEnglishSRT)
}

// Whatever sits at the name counts as taken — even a directory.
func TestSaveEnglishTranscript_AnythingAtThePlainNameCountsAsTaken(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "Movie.en.srt"), 0755))

	path, err := saveEnglishTranscript(dir, "Movie", genTestSRT)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "Movie.vido.en.srt"), path)
}

func TestSaveEnglishTranscript_FailureLeavesNoTempFile(t *testing.T) {
	dir := t.TempDir()
	// The target is a non-empty directory, so the rename fails.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Movie.en.srt"), []byte(userEnglishSRT), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "Movie.vido.en.srt", "x"), 0755))

	_, err := saveEnglishTranscript(dir, "Movie", genTestSRT)
	require.Error(t, err)
	assertOnlyFiles(t, dir, "Movie.en.srt", "Movie.vido.en.srt")
	assertFileContent(t, filepath.Join(dir, "Movie.en.srt"), userEnglishSRT)
}

// The resume reads the path the last run recorded, so a run stopped after
// writing <base>.vido.en.srt resumes from it — and the user's .en.srt next to
// it is neither read nor touched.
func TestRunTranscription_ResumesFromTheVidoTranscript(t *testing.T) {
	tmp := t.TempDir()
	userPath := filepath.Join(tmp, "Movie.en.srt")
	require.NoError(t, os.WriteFile(userPath, []byte(userEnglishSRT), 0644))
	vidoPath := filepath.Join(tmp, "Movie.vido.en.srt")
	require.NoError(t, os.WriteFile(vidoPath, []byte(genTestSRT), 0644))

	writer := &fakeSubtitleWriter{}
	reader := &fakeStateReader{movie: untranslatedMovie(uuidD, vidoPath)}
	completer := &promptRecordingCompleter{response: "[1] 你好世界"}
	svc := resumeService(t, completer, writer, reader)

	err := svc.RunTranscription(context.Background(), uuidD,
		filepath.Join(tmp, "Movie.mkv") /* does NOT exist — extract would fail */, tmp,
		WithTranslation())
	require.NoError(t, err, "resume must complete without touching extract/ASR")

	last := writer.calls[len(writer.calls)-1]
	assert.Equal(t, models.SubtitleStatusFound, last.Status)
	assert.Contains(t, completer.lastPrompt, "Hello world", "the transcript is what got translated")
	assert.NotContains(t, completer.lastPrompt, "Official line", "the user's file is not a resume point")
	assertFileContent(t, userPath, userEnglishSRT)
}

// promptRecordingCompleter keeps every user prompt, so a test can see which
// English text was sent for translation.
type promptRecordingCompleter struct {
	response   string
	lastPrompt string
}

func (c *promptRecordingCompleter) CompleteText(_ context.Context, _, userPrompt string, _ int) (string, error) {
	c.lastPrompt += userPrompt
	return c.response, nil
}

// ─── The whole run, on a real file (needs ffmpeg) ───────────────────────────

func TestRunTranscription_MovieKeepsTheUsersEnglishSubtitle(t *testing.T) {
	tmp := t.TempDir()
	media := tinyEnglishMKV(t, tmp, "Tears of Steel (2012).mkv")
	userPath := filepath.Join(tmp, "Tears of Steel (2012).en.srt")
	require.NoError(t, os.WriteFile(userPath, []byte(userEnglishSRT), 0644))

	writer := &fakeSubtitleWriter{}
	svc, client := fullRunService(t)
	svc.SetSubtitleStatusWriter(writer)

	complete := runAndCollect(t, svc, client, uuidD, media, tmp)

	assertFileContent(t, userPath, userEnglishSRT)
	vidoPath := filepath.Join(tmp, "Tears of Steel (2012).vido.en.srt")
	assertFileHas(t, vidoPath, "Hello world")
	zhPath := filepath.Join(tmp, "Tears of Steel (2012).zh-Hant.srt")
	assertFileHas(t, zhPath, "你好世界")
	assert.Equal(t, vidoPath, complete["srt_path"], "the event names the file actually written")

	last := writer.calls[len(writer.calls)-1]
	assert.Equal(t, models.SubtitleStatusFound, last.Status)
	assert.Equal(t, zhPath, last.Path)
}

func TestRunTranscription_EpisodeKeepsTheUsersEnglishSubtitle(t *testing.T) {
	tmp := t.TempDir()
	media := tinyEnglishMKV(t, tmp, "Show.S01E01.mkv")
	userPath := filepath.Join(tmp, "Show.S01E01.en.srt")
	require.NoError(t, os.WriteFile(userPath, []byte(userEnglishSRT), 0644))

	episodeWriter := &fakeEpisodeWriter{}
	svc, client := fullRunService(t)
	svc.SetEpisodeSubtitleStatusWriter(episodeWriter)

	complete := runAndCollect(t, svc, client, uuidD, media, tmp, WithMediaType(models.SubtitleRunMediaEpisode))

	assertFileContent(t, userPath, userEnglishSRT)
	vidoPath := filepath.Join(tmp, "Show.S01E01.vido.en.srt")
	assertFileHas(t, vidoPath, "Hello world")
	assertFileHas(t, filepath.Join(tmp, "Show.S01E01.zh-Hant.srt"), "你好世界")
	assert.Equal(t, vidoPath, complete["srt_path"])
	require.NotEmpty(t, episodeWriter.calls)
	assert.Equal(t, models.SubtitleStatusFound, episodeWriter.calls[len(episodeWriter.calls)-1].Status)
}

// Nothing beside the film: the transcript keeps its usual name.
func TestRunTranscription_NoSubtitleBesideTheFileKeepsThePlainName(t *testing.T) {
	tmp := t.TempDir()
	media := tinyEnglishMKV(t, tmp, "Movie.mkv")

	svc, client := fullRunService(t)
	svc.SetSubtitleStatusWriter(&fakeSubtitleWriter{})

	complete := runAndCollect(t, svc, client, uuidD, media, tmp)

	plain := filepath.Join(tmp, "Movie.en.srt")
	assertFileHas(t, plain, "Hello world")
	assert.Equal(t, plain, complete["srt_path"])
	_, err := os.Stat(filepath.Join(tmp, "Movie.vido.en.srt"))
	assert.True(t, os.IsNotExist(err), "no second English file when the plain name was free")
}

// fullRunService wires a real ffmpeg extractor, a speech recognizer that
// answers genTestSRT, a translator and an SSE hub to read the terminal from.
func fullRunService(t *testing.T) (*TranscriptionService, *sse.Client) {
	t.Helper()
	hub := sse.NewHub()
	t.Cleanup(func() { hub.Close() })
	client := hub.Register()
	require.Eventually(t, func() bool { return hub.ClientCount() == 1 }, 2*time.Second, time.Millisecond)
	svc := NewTranscriptionService(NewAudioExtractorService(1, time.Minute, nil), &plainASR{srt: genTestSRT}, hub, nil)
	svc.SetTranslationService(NewTranslationService(&translationIntegrationMock{response: "[1] 你好世界"}, nil))
	return svc, client
}

// runAndCollect runs the synchronous pipeline and returns the complete event.
func runAndCollect(t *testing.T, svc *TranscriptionService, client *sse.Client, mediaID, media, dir string, opts ...TranscriptionOption) map[string]interface{} {
	t.Helper()
	done := make(chan error, 1)
	go func() {
		done <- svc.RunTranscription(context.Background(), mediaID, media, dir, append(opts, WithTranslation())...)
	}()
	events := collectUntilTerminal(t, client, mediaID)
	require.NoError(t, <-done)
	last := events[len(events)-1]
	require.Equal(t, EventTranscriptionComplete, last.Type, "run failed: %v", last.Data)
	return last.Data.(map[string]interface{})
}

// tinyEnglishMKV makes a two-second film with an English audio track.
func tinyEnglishMKV(t *testing.T, dir, name string) string {
	t.Helper()
	for _, bin := range []string{"ffmpeg", "ffprobe"} {
		if _, err := exec.LookPath(bin); err != nil {
			t.Skipf("%s not installed — the full run cannot be driven here", bin)
		}
	}
	path := filepath.Join(dir, name)
	out, err := exec.Command("ffmpeg", "-loglevel", "error", "-f", "lavfi", "-i", "sine=frequency=440:duration=2",
		"-c:a", "aac", "-metadata:s:a:0", "language=eng", path).CombinedOutput()
	require.NoError(t, err, string(out))
	return path
}

func assertFileContent(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, want, string(got), path)
}

func assertFileHas(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(got), want, path)
}

// assertOnlyFiles fails on anything else in dir — a leftover temp file most of all.
func assertOnlyFiles(t *testing.T, dir string, want ...string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	var got []string
	for _, e := range entries {
		got = append(got, e.Name())
	}
	assert.ElementsMatch(t, want, got, strings.Join(got, ", "))
}
