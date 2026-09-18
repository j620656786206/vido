package services

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── disc-2026-09-transcription-run-5min-hard-timeout AC #1 ─────────────────
//
// One ffmpeg pass over a media file is disk I/O proportional to file SIZE.
// The subtitle extractor already gets max(floor, size × per-GB) (sub-6-3);
// the ASR audio extractor ran on a fixed 5 minutes, which a 66.8 GB remux
// needed 4:55 of on the NAS — the next cliff once the run timeout is gone.

func TestSizedFFmpegTimeout(t *testing.T) {
	const gb = int64(BytesPerGB)
	floor, perGB := 10*time.Minute, 30*time.Second
	cases := []struct {
		size int64
		want time.Duration
	}{
		{4 * gb, 10 * time.Minute},                      // 4 × 30 s = 2 min < floor
		{20 * gb, 10 * time.Minute},                     // 20 × 30 s = 10 min == floor
		{93 * gb, 46*time.Minute + 30*time.Second},      // 93 × 30 s
		{0, 10 * time.Minute},                           // unknown size → the floor, never less
		{-1, 10 * time.Minute},                          // a stat error reported as -1 is unknown too
		{int64(66.8 * float64(gb)), 2004 * time.Second}, // the NAS remux: 33m24s
	}
	for _, tc := range cases {
		assert.Equalf(t, tc.want, SizedFFmpegTimeout(floor, perGB, tc.size), "size=%d", tc.size)
	}
	assert.Equal(t, floor, SizedFFmpegTimeout(floor, 0, 93*gb), "a non-positive per-GB allowance means the floor alone")
}

func TestAudioExtractorService_EffectiveTimeout(t *testing.T) {
	const gb = int64(BytesPerGB)
	svc := NewAudioExtractorService(1, 10*time.Minute, nil,
		WithAudioExtractPerGB(30*time.Second),
		withAudioFileSize(func(path string) (int64, error) {
			switch filepath.Base(path) {
			case "small.mkv":
				return 4 * gb, nil
			case "goodfellas.mkv":
				return 93 * gb, nil
			}
			return 0, errors.New("stat failed")
		}))

	got, size := svc.EffectiveTimeout("/small.mkv")
	assert.Equal(t, 10*time.Minute, got)
	assert.InDelta(t, 4.0, size, 0.01)

	got, size = svc.EffectiveTimeout("/goodfellas.mkv")
	assert.Equal(t, 46*time.Minute+30*time.Second, got)
	assert.InDelta(t, 93.0, size, 0.01)

	got, size = svc.EffectiveTimeout("/missing.mkv")
	assert.Equal(t, 10*time.Minute, got, "stat failed → the floor, never less")
	assert.Zero(t, size)

	zero := NewAudioExtractorService(1, time.Minute, nil, WithAudioExtractPerGB(0))
	assert.Equal(t, defaultAudioExtractPerGB, zero.perGBTimeout, "non-positive per-GB values are ignored")
}

// installFakeAudioFFmpeg puts a shell `ffmpeg` on PATH that sleeps, then writes
// the output file its last argument names — enough to drive ExtractAudio
// without a real decode (the subtitle package's installFakeFFmpeg, kept
// separate because services cannot import subtitle — Rule 19).
func installFakeAudioFFmpeg(t *testing.T, sleep time.Duration) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell shim not portable to windows")
	}
	bin := t.TempDir()
	script := fmt.Sprintf(`#!/bin/sh
out=""
for a in "$@"; do out="$a"; done
sleep %s
: > "$out"
`, fmt.Sprintf("%.3f", sleep.Seconds()))
	require.NoError(t, os.WriteFile(filepath.Join(bin, "ffmpeg"), []byte(script), 0o755))
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// The timeout message names the knob an operator would have to raise — and
// only when OUR bound fired, never when the caller's deadline did.
func TestAudioExtractorService_TimeoutMessageNamesTheKnob(t *testing.T) {
	installFakeAudioFFmpeg(t, 500*time.Millisecond)
	media := filepath.Join(t.TempDir(), "m.mkv")
	require.NoError(t, os.WriteFile(media, []byte("x"), 0o600))

	t.Run("our size-aware bound", func(t *testing.T) {
		svc := NewAudioExtractorService(1, 40*time.Millisecond, nil)
		require.True(t, svc.IsAvailable(), "the shim must be found on PATH")
		_, err := svc.ExtractAudio(context.Background(), media, 1)

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrAudioExtractionTimeout)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
		assert.Contains(t, err.Error(), "SUBTITLE_EXTRACT_TIMEOUT_SECONDS")
		assert.Contains(t, err.Error(), "0.0 GB")
		assert.NotContains(t, err.Error(), "\n", "one line — this string reaches the screen")
	})

	t.Run("past ~20 GB the per-GB knob is the one to raise", func(t *testing.T) {
		svc := NewAudioExtractorService(1, 40*time.Millisecond, nil,
			WithAudioExtractPerGB(time.Millisecond),
			withAudioFileSize(func(string) (int64, error) { return 93 * BytesPerGB, nil }))
		_, err := svc.ExtractAudio(context.Background(), media, 1)

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrAudioExtractionTimeout)
		assert.Contains(t, err.Error(), "SUBTITLE_EXTRACT_PER_GB_SECONDS")
		assert.Contains(t, err.Error(), "93.0 GB")
	})

	t.Run("the caller cancelled (a cancelled batch, a shutdown)", func(t *testing.T) {
		svc := NewAudioExtractorService(1, time.Minute, nil)
		ctx, cancel := context.WithCancel(context.Background())
		go func() { time.Sleep(60 * time.Millisecond); cancel() }()
		_, err := svc.ExtractAudio(ctx, media, 1)

		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
		assert.ErrorIs(t, err, ErrAudioExtractionTimeout)
		assert.Contains(t, err.Error(), "stopped by the caller")
		assert.NotContains(t, err.Error(), "signal: killed")
		assert.NotContains(t, err.Error(), "SUBTITLE_EXTRACT_")
	})

	t.Run("the caller's deadline", func(t *testing.T) {
		svc := NewAudioExtractorService(1, time.Minute, nil)
		ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
		defer cancel()
		_, err := svc.ExtractAudio(ctx, media, 1)

		require.Error(t, err)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
		assert.NotContains(t, err.Error(), "SUBTITLE_EXTRACT_", "our knob did not fire — do not blame it")
		assert.Contains(t, err.Error(), "stopped by the caller")
	})
}
