package subtitle

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── sub-6-3 AC #1 / #4a: size-aware deadline ──────────────────────────────

func TestExtractor_EffectiveTimeout(t *testing.T) {
	const gb = int64(bytesPerGB)
	sizes := map[string]int64{
		"/small.mkv":      4 * gb,
		"/twenty.mkv":     20 * gb,
		"/goodfellas.mkv": 93 * gb,
	}
	e := NewExtractor(10*time.Minute, nil, withFileSize(func(path string) (int64, error) {
		if n, ok := sizes[path]; ok {
			return n, nil
		}
		return 0, os.ErrNotExist
	}))

	cases := []struct {
		path     string
		want     time.Duration
		wantSize float64
	}{
		{"/small.mkv", 10 * time.Minute, 4},                      // 4 × 30 s = 2 min < floor
		{"/twenty.mkv", 10 * time.Minute, 20},                    // 20 × 30 s = 10 min == floor
		{"/goodfellas.mkv", 46*time.Minute + 30*time.Second, 93}, // 93 × 30 s
		{"/missing.mkv", 10 * time.Minute, 0},                    // stat failed → the floor, never less
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			got, size := e.EffectiveTimeout(tc.path)
			assert.Equal(t, tc.want, got)
			assert.InDelta(t, tc.wantSize, size, 0.001)
		})
	}

	fast := NewExtractor(time.Minute, nil,
		withFileSize(func(string) (int64, error) { return 93 * gb, nil }),
		WithPerGBTimeout(5*time.Second))
	got, _ := fast.EffectiveTimeout("/x.mkv")
	assert.Equal(t, 93*5*time.Second, got, "the per-GB allowance is a knob")

	zero := NewExtractor(time.Minute, nil, WithPerGBTimeout(0))
	assert.Equal(t, defaultPerGBTimeout, zero.perGBTimeout, "non-positive per-GB values are ignored")
}

// ─── AC #2 / #4b: two concurrent extractions never overlap ─────────────────

// installFakeFFmpeg puts a shell `ffmpeg` on PATH that logs a start/end line to
// logPath around a sleep, then writes the output file its last argument
// names — enough to drive Extract end-to-end without a real demux.
func installFakeFFmpeg(t *testing.T, logPath string, sleep time.Duration) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell shim not portable to windows")
	}
	bin := t.TempDir()
	script := fmt.Sprintf(`#!/bin/sh
out=""
for a in "$@"; do out="$a"; done
echo "start $(date +%%s%%N)" >> %q
sleep %s
echo "end $(date +%%s%%N)" >> %q
: > "$out"
`, logPath, fmt.Sprintf("%.3f", sleep.Seconds()), logPath)
	require.NoError(t, os.WriteFile(filepath.Join(bin, "ffmpeg"), []byte(script), 0o755))
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestExtractor_ConcurrentExtractsAreSerialized(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "ffmpeg.log")
	installFakeFFmpeg(t, logPath, 150*time.Millisecond)

	gate := NewExtractGate()
	a := NewExtractor(time.Minute, nil, WithExtractGate(gate))
	b := NewExtractor(time.Minute, nil, WithExtractGate(gate))
	require.True(t, a.IsAvailable(), "the shim must be found on PATH")

	media := filepath.Join(t.TempDir(), "m.mkv")
	require.NoError(t, os.WriteFile(media, []byte("x"), 0o600))

	var wg sync.WaitGroup
	var waits []int
	var mu sync.Mutex
	for _, e := range []*Extractor{a, b} {
		wg.Add(1)
		go func(e *Extractor) {
			defer wg.Done()
			ctx := WithExtractWaitNotifier(context.Background(), func(ahead int) {
				mu.Lock()
				waits = append(waits, ahead)
				mu.Unlock()
			})
			out, err := e.Extract(ctx, media, t.TempDir(), []int{2})
			assert.NoError(t, err)
			assert.Len(t, out, 1)
		}(e)
	}
	wg.Wait()

	raw, err := os.ReadFile(logPath)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	require.Len(t, lines, 4, "two runs, each start+end")
	// Serialized ⇔ the second start comes after the first end.
	assert.True(t, strings.HasPrefix(lines[0], "start "))
	assert.True(t, strings.HasPrefix(lines[1], "end "), "the first ffmpeg must END before the second STARTS: %v", lines)
	assert.True(t, strings.HasPrefix(lines[2], "start "))
	assert.True(t, strings.HasPrefix(lines[3], "end "))
	assert.Equal(t, []int{1}, waits, "exactly one of the two queued, behind one holder")
}

// ─── AC #4c: a ctx that ends while queued returns at once, ffmpeg untouched ─

func TestExtractor_CancelledWhileQueuedIsAWaitAbortNotAnFFmpegFailure(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "ffmpeg.log")
	installFakeFFmpeg(t, logPath, 10*time.Millisecond)

	gate := NewExtractGate()
	hold, err := gate.Acquire(context.Background(), nil)
	require.NoError(t, err)
	defer hold()

	e := NewExtractor(time.Minute, nil, WithExtractGate(gate))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	_, err = e.Extract(ctx, "/m.mkv", t.TempDir(), []int{1})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSubtitleExtractWaitAborted)
	assert.ErrorIs(t, err, context.DeadlineExceeded, "the ctx cause stays in the chain (Rule 13)")
	assert.NotErrorIs(t, err, ErrSubtitleExtractFailed, "nothing was extracted, so it is not an extraction failure")
	assert.Contains(t, err.Error(), "queued behind 1")
	_, statErr := os.Stat(logPath)
	assert.True(t, errors.Is(statErr, os.ErrNotExist), "ffmpeg must never have started")
}

// ─── AC #3: the timeout message names the knob only when OUR bound fired ───

func TestExtractor_TimeoutMessageNamesTheKnob(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "ffmpeg.log")
	installFakeFFmpeg(t, logPath, 500*time.Millisecond)
	media := filepath.Join(t.TempDir(), "m.mkv")
	require.NoError(t, os.WriteFile(media, []byte("x"), 0o600))

	t.Run("our size-aware bound", func(t *testing.T) {
		e := NewExtractor(40*time.Millisecond, nil)
		_, err := e.Extract(context.Background(), media, t.TempDir(), []int{1})

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrSubtitleExtractFailed)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
		assert.Contains(t, err.Error(), "SUBTITLE_EXTRACT_TIMEOUT_SECONDS")
		assert.Contains(t, err.Error(), "0.0 GB")
	})

	t.Run("the caller's deadline", func(t *testing.T) {
		e := NewExtractor(time.Minute, nil)
		ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
		defer cancel()
		_, err := e.Extract(ctx, media, t.TempDir(), []int{1})

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrSubtitleExtractFailed)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
		assert.NotContains(t, err.Error(), "SUBTITLE_EXTRACT_TIMEOUT_SECONDS", "our knob did not fire — do not blame it")
		assert.Contains(t, err.Error(), "caller's deadline")
	})
}

func TestNewExtractor_HasItsOwnGateByDefault(t *testing.T) {
	e := NewExtractor(0, nil)
	require.NotNil(t, e.gate, "an un-injected Extractor still serializes against itself")

	shared := NewExtractGate()
	assert.Same(t, shared, NewExtractor(0, nil, WithExtractGate(shared)).gate)
	assert.NotNil(t, NewExtractor(0, nil, WithExtractGate(nil)).gate, "a nil injection keeps the default")
}
