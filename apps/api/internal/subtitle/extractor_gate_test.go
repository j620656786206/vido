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
	assert.Equal(t, []int{1, 0}, waits,
		"exactly one of the two queued (behind one holder) and was told when it got in")
}

// ─── AC #4c: a ctx that ends while queued returns at once, ffmpeg untouched ─

func TestExtractor_CancelledWhileQueuedIsAWaitAbortNotAnFFmpegFailure(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "ffmpeg.log")
	installFakeFFmpeg(t, logPath, 10*time.Millisecond)

	gate := NewExtractGate()
	hold, _, err := gate.Acquire(context.Background(), nil)
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
	assert.NotContains(t, err.Error(), "queued behind 0", "the depth is the one the FIRST notice reported (CR L7)")
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

// ─── CR H1: a budget spent queueing is not the file's fault ────────────────

func TestExtractor_CallerDeadlineAfterQueueingIsAWaitAbort(t *testing.T) {
	// The item's own deadline kills ffmpeg — but only because most of its
	// budget went to waiting for the slot. Our size-aware bound would have
	// fired FIRST on a whole budget, so this run says nothing about the file
	// and must not count toward the free lane's three-strikes parking.
	logPath := filepath.Join(t.TempDir(), "ffmpeg.log")
	installFakeFFmpeg(t, logPath, 500*time.Millisecond)
	media := filepath.Join(t.TempDir(), "m.mkv")
	require.NoError(t, os.WriteFile(media, []byte("x"), 0o600))

	gate := NewExtractGate()
	hold, _, err := gate.Acquire(context.Background(), nil)
	require.NoError(t, err)
	go func() {
		time.Sleep(60 * time.Millisecond)
		hold()
	}()

	e := NewExtractor(time.Minute, nil, WithExtractGate(gate))
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Millisecond)
	defer cancel()

	_, err = e.Extract(ctx, media, t.TempDir(), []int{1})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSubtitleExtractWaitAborted,
		"a starved budget is cancelled-class, not a file failure (CR H1)")
	assert.ErrorIs(t, err, ErrSubtitleExtractFailed, "the extraction did fail, and the chain says both")
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Contains(t, err.Error(), "went to waiting for the extraction slot")
}

func TestExtractor_CallerDeadlineWithoutQueueingStaysAFileFailure(t *testing.T) {
	// No wait: the item budget was simply too small for this file, which IS
	// something the operator should see as a failure of this item.
	logPath := filepath.Join(t.TempDir(), "ffmpeg.log")
	installFakeFFmpeg(t, logPath, 500*time.Millisecond)
	media := filepath.Join(t.TempDir(), "m.mkv")
	require.NoError(t, os.WriteFile(media, []byte("x"), 0o600))

	e := NewExtractor(time.Minute, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()

	_, err := e.Extract(ctx, media, t.TempDir(), []int{1})

	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrSubtitleExtractWaitAborted)
	assert.ErrorIs(t, err, ErrSubtitleExtractFailed)
}

// ─── CR M6: the timeout message names the knob that would actually help ────

func TestExtractor_TimeoutMessageNamesThePerGBKnobForBigFiles(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "ffmpeg.log")
	installFakeFFmpeg(t, logPath, 500*time.Millisecond)
	media := filepath.Join(t.TempDir(), "m.mkv")
	require.NoError(t, os.WriteFile(media, []byte("x"), 0o600))

	// A "93 GB" file whose per-GB term (93 × 1 ms) beats the 40 ms floor:
	// raising the floor to 41 ms would change nothing, so the message must
	// point at the per-GB knob instead.
	e := NewExtractor(40*time.Millisecond, nil,
		withFileSize(func(string) (int64, error) { return 93 * bytesPerGB, nil }),
		WithPerGBTimeout(time.Millisecond))

	_, err := e.Extract(context.Background(), media, t.TempDir(), []int{1})

	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Contains(t, err.Error(), extractPerGBEnv)
	assert.NotContains(t, err.Error(), extractFloorEnv)
	assert.Contains(t, err.Error(), "93.0 GB")
}

func TestExtractor_EffectiveTimeoutReportsTheDecidingKnob(t *testing.T) {
	e := NewExtractor(10*time.Minute, nil, withFileSize(func(path string) (int64, error) {
		switch path {
		case "/big.mkv":
			return 93 * bytesPerGB, nil
		case "/small.mkv":
			return 4 * bytesPerGB, nil
		}
		return 0, os.ErrNotExist
	}))

	_, _, knob := e.effectiveTimeout("/big.mkv")
	assert.Equal(t, extractPerGBEnv, knob, "past ~20 GB the size term decides")
	_, _, knob = e.effectiveTimeout("/small.mkv")
	assert.Equal(t, extractFloorEnv, knob)
	_, _, knob = e.effectiveTimeout("/missing.mkv")
	assert.Equal(t, extractFloorEnv, knob, "an unknown size falls back to the floor, and says so")
}

// ─── CR M4: a killed ffmpeg cannot strand the single slot ──────────────────

func TestExtractor_SetsWaitDelaySoAKilledFFmpegCannotHoldTheSlot(t *testing.T) {
	// A grandchild holding the stderr pipe makes Run() block forever without
	// WaitDelay. With serialization that would freeze the ONE slot for the
	// life of the process, silently stopping every future extraction.
	assert.Equal(t, 10*time.Second, extractWaitDelay)

	dir := t.TempDir()
	logPath := filepath.Join(dir, "ffmpeg.log")
	installFakeFFmpeg(t, logPath, 10*time.Millisecond)
	media := filepath.Join(dir, "m.mkv")
	require.NoError(t, os.WriteFile(media, []byte("x"), 0o600))

	e := NewExtractor(time.Minute, nil)
	_, err := e.Extract(context.Background(), media, t.TempDir(), []int{1})
	require.NoError(t, err, "the happy path is unaffected by the delay")
}
