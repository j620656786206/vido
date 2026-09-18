package ai

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── disc-2026-09-transcription-run-5min-hard-timeout AC #3 ─────────────────
//
// The chunk splitter used to put ffmpeg's ENTIRE combined output into the
// error string — banner, build configuration, stream mapping — and a
// deadline kill surfaced as a bare "signal: killed". Both reached the screen.

type failingChunkCmd struct {
	output string
	err    error
}

func (c failingChunkCmd) CombinedOutput() ([]byte, error) { return []byte(c.output), c.err }

func chunkFixture(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "*.wav")
	require.NoError(t, err)
	writeWAVWithChunks(t, f, 32000, 26*1024*1024, false) // 2 chunks of 600 s
	return f.Name()
}

func TestSplitAudioChunks_ErrorKeepsOnlyTheStderrTail(t *testing.T) {
	banner := "ffmpeg version 6.1.2 Copyright (c) 2000-2024 the FFmpeg developers\n  built with gcc 14.2.0\n  configuration: --prefix=/usr --enable-gpl\n"
	for i := 0; i < 30; i++ {
		banner += "  libavutil      58. 29.100 / 58. 29.100\n"
	}
	banner += "Input #0, wav, from '/tmp/x.wav':\n[out#0] Error writing trailer: No space left on device\nConversion failed!\n"

	origExec := execCommandContext
	execCommandContext = func(ctx context.Context, name string, args ...string) command {
		return failingChunkCmd{output: banner, err: errors.New("exit status 1")}
	}
	defer func() { execCommandContext = origExec }()

	_, _, err := SplitAudioChunks(context.Background(), chunkFixture(t))

	require.Error(t, err)
	msg := err.Error()
	assert.Contains(t, msg, "No space left on device", "the line that explains the failure survives")
	assert.Contains(t, msg, "Conversion failed!")
	assert.NotContains(t, msg, "ffmpeg version", "the banner never reaches an error string")
	assert.NotContains(t, msg, "libavutil")
	assert.NotContains(t, msg, "\n", "one line")
	assert.LessOrEqual(t, len(msg), 400)
}

func TestSplitAudioChunks_DeadlineIsReportedAsTheRunDeadline(t *testing.T) {
	origExec := execCommandContext
	execCommandContext = func(ctx context.Context, name string, args ...string) command {
		return failingChunkCmd{output: "ffmpeg version 6.1.2 …\nPress [q] to stop\n", err: errors.New("signal: killed")}
	}
	defer func() { execCommandContext = origExec }()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // the run deadline has already fired when ffmpeg died
	_, _, err := SplitAudioChunks(ctx, chunkFixture(t))

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Contains(t, err.Error(), "stopped by the run deadline")
	assert.NotContains(t, err.Error(), "signal: killed")
	assert.NotContains(t, err.Error(), "ffmpeg version")
}

func TestStderrTail(t *testing.T) {
	assert.Equal(t, "", stderrTail(nil, 3, 300))
	assert.Equal(t, "c | d | e", stderrTail([]byte("a\nb\nc\nd\ne\n"), 3, 300))
	long := strings.Repeat("x", 500)
	got := stderrTail([]byte(long), 3, 100)
	assert.True(t, strings.HasPrefix(got, "…"))
	assert.LessOrEqual(t, len(got), 104)
}

func TestWAVDuration(t *testing.T) {
	d, err := WAVDuration(chunkFixture(t))
	require.NoError(t, err)
	assert.InDelta(t, float64(26*1024*1024)/32000, d, 0.01)
	_, err = WAVDuration("/definitely/missing.wav")
	assert.Error(t, err)
}
