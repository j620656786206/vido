package services

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/sse"
)

// ─── disc-2026-09-transcription-run-5min-hard-timeout AC #2/#3 ──────────────
//
// The whole run used to share ONE fixed 5-minute deadline. Extracting the
// audio of a 66.8 GB remux took 4:55 of it on the NAS; the ffmpeg that split
// the audio was then killed at exactly 5:00 with $0 spent. Now the extraction
// is bounded by file size (AC #1) and everything after it by media LENGTH,
// read from the WAV that was just extracted.

func TestRunPhaseBudget(t *testing.T) {
	floor, perMin := 10*time.Minute, 30*time.Second
	cases := []struct {
		mediaSeconds float64
		want         time.Duration
	}{
		{0, 10 * time.Minute},  // unknown → the floor
		{-5, 10 * time.Minute}, // garbage → the floor
		{9425, 78*time.Minute + 32*time.Second + 500*time.Millisecond}, // the NAS film: 157.08 min × 30 s
		{2700, 22*time.Minute + 30*time.Second},                        // a 45-minute episode
		{1200, 10 * time.Minute},                                       // 20 min × 30 s == the floor
		{600, 10 * time.Minute},                                        // shorter than the floor allows → the floor
	}
	for _, tc := range cases {
		got, _ := runPhaseBudget(floor, perMin, tc.mediaSeconds)
		assert.Equalf(t, tc.want, got, "media=%vs", tc.mediaSeconds)
	}
	_, knob := runPhaseBudget(floor, perMin, 9425)
	assert.Equal(t, transcriptionPerMinuteEnv, knob, "past the floor the per-minute term decides")
	_, knob = runPhaseBudget(floor, perMin, 600)
	assert.Equal(t, transcriptionFloorEnv, knob)
	got, _ := runPhaseBudget(floor, 0, 9425)
	assert.Equal(t, floor, got, "a non-positive allowance means the floor alone")
}

func TestSRTSpanSeconds(t *testing.T) {
	srt := "1\n00:00:01,000 --> 00:00:04,000\nHello\n\n2\n00:05:10,500 --> 02:37:05,250\nBye\n"
	assert.InDelta(t, 2*3600+37*60+5.25, srtSpanSeconds(srt), 0.001)
	assert.Zero(t, srtSpanSeconds(""), "no cues → unknown")
	assert.Zero(t, srtSpanSeconds("1\nno timing line here\n"), "no arrow → unknown")
	assert.Zero(t, srtSpanSeconds("1\n00:00:01,000 --> garbage\n"), "unparsable end → unknown")
	// CRLF files (Windows-authored SRTs) and a cue whose TEXT contains an arrow.
	assert.InDelta(t, 4, srtSpanSeconds("1\r\n00:00:01,000 --> 00:00:04,000\r\nHello\r\n"), 0.001)
	assert.InDelta(t, 4, srtSpanSeconds("1\n00:00:01,000 --> 00:00:04,000\nNext --> please\n"), 0.001,
		"a text line with an arrow is skipped, not mistaken for the end")
}

// CR H1: a provider's bare sentinel (no ctx error wrapped) after OUR deadline
// fired is still explained as our timeout — the deadline is the cause.
func TestPhaseTimeoutError(t *testing.T) {
	parent := context.Background()
	phase, cancel := context.WithTimeout(parent, time.Nanosecond)
	defer cancel()
	<-phase.Done()
	bare := errors.New("whisper: request timed out")

	err := phaseTimeoutError(parent, phase, "transcribing", time.Now(), 9425, 4712*time.Second, transcriptionPerMinuteEnv, bare)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Contains(t, err.Error(), "timed out")
	assert.Contains(t, err.Error(), "transcribing")
	assert.Contains(t, err.Error(), transcriptionPerMinuteEnv)
	assert.Contains(t, err.Error(), "whisper: request timed out", "the original text rides along")

	live, cancelLive := context.WithCancel(parent)
	defer cancelLive()
	assert.Same(t, bare, phaseTimeoutError(parent, live, "transcribing", time.Now(), 0, 0, "", bare),
		"no deadline of ours fired → the error stands on its own")
	assert.NoError(t, phaseTimeoutError(parent, phase, "x", time.Now(), 0, 0, "", nil))
}

// writeHeaderOnlyWAV writes a WAV whose header CLAIMS durationSeconds of
// 16 kHz mono PCM without carrying the payload — parseWAVInfo seeks past the
// data chunk, so the run reads the claimed length while the file stays tiny
// (no chunking, one fake ASR call).
func writeHeaderOnlyWAV(t *testing.T, path string, durationSeconds float64) {
	t.Helper()
	const byteRate = 32000
	var buf bytes.Buffer
	buf.WriteString("RIFF")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(0))
	buf.WriteString("WAVE")
	buf.WriteString("fmt ")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(16))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(&buf, binary.LittleEndian, uint32(16000))
	_ = binary.Write(&buf, binary.LittleEndian, uint32(byteRate))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(2))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(16))
	buf.WriteString("data")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(durationSeconds*byteRate))
	require.NoError(t, os.WriteFile(path, buf.Bytes(), 0o644))
}

// installFakeMediaTools puts `ffprobe` (one English audio track) and `ffmpeg`
// (copies wavFixture to its last argument) on PATH, so runPipeline can be
// driven end to end without decoding anything.
func installFakeMediaTools(t *testing.T, wavFixture string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell shim not portable to windows")
	}
	bin := t.TempDir()
	ffprobe := `#!/bin/sh
echo '{"streams":[{"index":1,"codec_name":"aac","channels":2,"tags":{"language":"eng"}}]}'
`
	ffmpeg := fmt.Sprintf(`#!/bin/sh
out=""
for a in "$@"; do out="$a"; done
cp %q "$out"
`, wavFixture)
	require.NoError(t, os.WriteFile(filepath.Join(bin, "ffprobe"), []byte(ffprobe), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(bin, "ffmpeg"), []byte(ffmpeg), 0o755))
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// slowASR answers after `delay`, or with the ctx error if the deadline lands first.
type slowASR struct {
	delay time.Duration
	srt   string
	// onCall runs when the ASR is asked — a test's hook to cancel the CALLER
	// while the run is provably in the transcribing step.
	onCall func()
}

func (a *slowASR) Transcribe(ctx context.Context, audioPath string) (string, error) {
	return a.TranscribeWithLanguage(ctx, audioPath, "")
}

func (a *slowASR) TranscribeWithLanguage(ctx context.Context, audioPath, lang string) (string, error) {
	if a.onCall != nil {
		a.onCall()
	}
	select {
	case <-time.After(a.delay):
		return a.srt, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// runBudgetService wires a service whose extraction is the fake ffmpeg and
// whose ASR is slowASR, with a hub so the failed event can be read back.
func runBudgetService(t *testing.T, wavSeconds float64, asr ai.ASRProvider, floor, perMin time.Duration) (*TranscriptionService, *sse.Client) {
	t.Helper()
	wav := filepath.Join(t.TempDir(), "fixture.wav")
	writeHeaderOnlyWAV(t, wav, wavSeconds)
	installFakeMediaTools(t, wav)

	hub := sse.NewHub()
	t.Cleanup(func() { hub.Close() })
	client := hub.Register()
	require.Eventually(t, func() bool { return hub.ClientCount() == 1 }, 2*time.Second, time.Millisecond)

	extractor := NewAudioExtractorService(1, time.Minute, nil)
	require.True(t, extractor.IsAvailable(), "the ffmpeg shim must be on PATH")
	svc := NewTranscriptionService(extractor, asr, hub, nil)
	svc.SetRunBudget(floor, perMin)
	return svc, client
}

func failedEventError(t *testing.T, client *sse.Client) string {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for {
		select {
		case ev := <-client.Events:
			if ev.Type != EventTranscriptionFailed {
				continue
			}
			data, ok := ev.Data.(map[string]interface{})
			require.True(t, ok)
			s, _ := data["error"].(string)
			return s
		case <-deadline:
			t.Fatal("no transcription_failed event")
			return ""
		}
	}
}

// The budget comes from the WAV's length, not a constant: a floor far too
// short for the ASR call still completes because the media is long.
func TestRunTranscription_BudgetFollowsTheMediaLength(t *testing.T) {
	svc, _ := runBudgetService(t, 9425 /* the NAS film */, &slowASR{delay: 300 * time.Millisecond, srt: genTestSRT},
		50*time.Millisecond /* floor */, 10*time.Millisecond /* per media minute → ~1.57 s */)
	media := filepath.Join(t.TempDir(), "m.mkv")
	require.NoError(t, os.WriteFile(media, []byte("x"), 0o600))

	err := svc.RunTranscription(context.Background(), uuidEight, media, filepath.Dir(media))

	require.NoError(t, err, "a 157-minute film must not be held to a 50 ms floor")
	assert.FileExists(t, filepath.Join(filepath.Dir(media), "m.en.srt"))
}

// AC #3: when OUR phase budget fires, the failed event carries ONE line that
// names the phase, the elapsed time, the media length, the budget and the knob
// to raise — never thirty lines of ffmpeg output.
func TestRunTranscription_TimeoutIsOneLineNamingTheKnob(t *testing.T) {
	t.Run("the per-minute knob decided the budget", func(t *testing.T) {
		svc, client := runBudgetService(t, 9425, &slowASR{delay: 2 * time.Second, srt: genTestSRT},
			50*time.Millisecond, 1*time.Millisecond /* → ~157 ms */)
		media := filepath.Join(t.TempDir(), "m.mkv")
		require.NoError(t, os.WriteFile(media, []byte("x"), 0o600))

		err := svc.RunTranscription(context.Background(), uuidEight, media, filepath.Dir(media))

		require.Error(t, err)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
		msg := failedEventError(t, client)
		assert.Contains(t, msg, "timed out")
		assert.Contains(t, msg, "transcribing")
		assert.Contains(t, msg, "157 min")
		assert.Contains(t, msg, transcriptionPerMinuteEnv)
		assert.NotContains(t, msg, "\n", "one line — this string reaches the screen")
		assert.NotContains(t, msg, "signal: killed")
	})

	t.Run("the floor decided the budget", func(t *testing.T) {
		svc, client := runBudgetService(t, 30 /* half a minute */, &slowASR{delay: 2 * time.Second, srt: genTestSRT},
			100*time.Millisecond, 1*time.Millisecond)
		media := filepath.Join(t.TempDir(), "m.mkv")
		require.NoError(t, os.WriteFile(media, []byte("x"), 0o600))

		err := svc.RunTranscription(context.Background(), uuidEight, media, filepath.Dir(media))

		require.Error(t, err)
		msg := failedEventError(t, client)
		assert.Contains(t, msg, transcriptionFloorEnv)
		assert.NotContains(t, msg, transcriptionPerMinuteEnv)
	})

	t.Run("the caller's deadline is not our knob's fault", func(t *testing.T) {
		// The caller (a batch item's bound, a shutdown) goes away while the run
		// is provably in the transcribing step — the ASR hook pulls the plug.
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		svc, client := runBudgetService(t, 9425, &slowASR{delay: 2 * time.Second, srt: genTestSRT, onCall: cancel},
			time.Minute, time.Second)
		media := filepath.Join(t.TempDir(), "m.mkv")
		require.NoError(t, os.WriteFile(media, []byte("x"), 0o600))

		err := svc.RunTranscription(ctx, uuidEight, media, filepath.Dir(media))

		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
		msg := failedEventError(t, client)
		assert.Contains(t, msg, "stopped by the caller's deadline")
		assert.Contains(t, msg, "transcribing")
		assert.False(t, strings.Contains(msg, "TRANSCRIPTION_"), "our knob did not fire — do not blame it: %s", msg)
		assert.False(t, strings.Contains(msg, filepath.Dir(media)), "no server path on screen: %s", msg)
	})
}

// The fixed 5-minute field is gone. A service with no SetRunBudget call still
// has a sane floor (the config default), not zero.
func TestNewTranscriptionService_DefaultRunBudget(t *testing.T) {
	svc := NewTranscriptionService(nil, nil, nil, nil)
	got, _ := runPhaseBudget(svc.runFloor, svc.runPerMediaMinute, 0)
	assert.Equal(t, 10*time.Minute, got, "default floor = TRANSCRIPTION_RUN_TIMEOUT_SECONDS 600")
	assert.Equal(t, 30*time.Second, svc.runPerMediaMinute)
	svc.SetRunBudget(0, 0)
	assert.Equal(t, 10*time.Minute, svc.runFloor, "non-positive values keep the defaults")
}

// blockingCompleter never answers — the translation runs until the ctx ends.
type blockingCompleter struct{}

func (blockingCompleter) CompleteText(ctx context.Context, _, _ string, _ int) (string, error) {
	<-ctx.Done()
	return "", ctx.Err()
}

// CR M2: a phase deadline during TRANSLATION must still record `untranslated`
// + the EN path (the resume enabler), on a ctx that is not the dead one —
// otherwise the next click re-pays the whole ASR.
func TestRunTranscription_TranslationDeadlineStillWritesUntranslated(t *testing.T) {
	tmp := t.TempDir()
	enPath := filepath.Join(tmp, "Movie.en.srt")
	require.NoError(t, os.WriteFile(enPath, []byte(genTestSRT), 0644))
	writer := &fakeSubtitleWriter{}
	reader := &fakeStateReader{movie: untranslatedMovie(uuidD, enPath)}
	svc := resumeService(t, blockingCompleter{}, writer, reader)
	svc.SetRunBudget(100*time.Millisecond, time.Millisecond) // the 4-second SRT → the floor

	err := svc.RunTranscription(context.Background(), uuidD, filepath.Join(tmp, "Movie.mkv"), tmp, WithTranslation())

	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Contains(t, err.Error(), "timed out")
	assert.Contains(t, err.Error(), "translating")
	assert.Contains(t, err.Error(), transcriptionFloorEnv)
	calls := writer.snapshot()
	require.NotEmpty(t, calls, "the verdict must be written even though the phase ctx is dead")
	last := calls[len(calls)-1]
	assert.Equal(t, models.SubtitleStatusUntranslated, last.Status)
	assert.Equal(t, enPath, last.Path)
}
