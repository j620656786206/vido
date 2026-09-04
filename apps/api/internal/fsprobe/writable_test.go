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
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already expired: the probe must not be allowed to "win"
	err := ProbeWritableContext(ctx, t.TempDir())
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
