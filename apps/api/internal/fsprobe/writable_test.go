package fsprobe

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProbeWritable_WritableDir(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, ProbeWritable(dir))

	// No debris left behind.
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Empty(t, entries, "the probe file must be removed")
}

func TestProbeWritable_ReadOnlyDir(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores mode bits")
	}
	dir := t.TempDir()
	require.NoError(t, os.Chmod(dir, 0o555))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	err := ProbeWritable(dir)
	assert.ErrorIs(t, err, ErrNotWritable)
}

func TestProbeWritable_MissingDir(t *testing.T) {
	err := ProbeWritable(filepath.Join(t.TempDir(), "nope"))
	assert.ErrorIs(t, err, ErrNotWritable)
}

func TestProbeWritable_FileNotDir(t *testing.T) {
	f := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(f, []byte("x"), 0o644))
	assert.ErrorIs(t, ProbeWritable(f), ErrNotWritable)
}

func TestProbeWritable_Empty(t *testing.T) {
	assert.ErrorIs(t, ProbeWritable(""), ErrNotWritable)
}

func TestProbeWritableContext_HonoursDeadline(t *testing.T) {
	// NOT t.TempDir(): when ctx is already expired the function returns on the
	// ctx branch while its probe goroutine is still creating/removing the probe
	// file. t.TempDir()'s RemoveAll then races that goroutine and, roughly one
	// run in a few hundred on a loaded CI runner, dies with "directory not
	// empty" (run 34034191552, the first Go Tests red after 10 green runs).
	// The leaked goroutine is BY DESIGN (a hung mount must not hang the
	// caller), so the test waits for it to finish before tearing the dir down.
	dir, err := os.MkdirTemp("", "vido-probe-deadline-*")
	require.NoError(t, err)
	t.Cleanup(func() {
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			if leftovers, _ := filepath.Glob(filepath.Join(dir, probePattern)); len(leftovers) == 0 {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		_ = os.RemoveAll(dir)
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already expired: the probe must not be allowed to "win"
	err = ProbeWritableContext(ctx, dir)
	// Either the fast probe completed before select observed ctx (nil) or the
	// deadline path fired — both are legal; what must never happen is a hang.
	if err != nil {
		assert.ErrorIs(t, err, ErrNotWritable)
	}
}

func TestProbeWritableContext_LiveCtxPassesThrough(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	assert.NoError(t, ProbeWritableContext(ctx, t.TempDir()))
}
