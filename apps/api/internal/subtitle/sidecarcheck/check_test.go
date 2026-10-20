package sidecarcheck

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/services"
)

type fakeProber struct{ seconds float64 }

func (p fakeProber) Probe(context.Context, string) (*services.MediaTechInfo, error) {
	return &services.MediaTechInfo{DurationSeconds: p.seconds}, nil
}

type memoryCache struct{ entries map[string]string }

func (m *memoryCache) Get(_ context.Context, key string) (*repository.CacheEntry, error) {
	v, ok := m.entries[key]
	if !ok {
		return nil, nil
	}
	return &repository.CacheEntry{Key: key, Value: v}, nil
}

func (m *memoryCache) Set(_ context.Context, key, value, _ string, _ time.Duration) error {
	m.entries[key] = value
	return nil
}

// fixtureCues is n lines at irregular times, like dialogue — evenly spaced
// lines would match the audio equally well at every multiple of the spacing.
func fixtureCues(n int) []cue {
	out := make([]cue, n)
	for i := range out {
		s := 10_000 + i*5_000 + (i*i*137)%2_500
		out[i] = cue{StartMS: s, EndMS: s + 1_200 + (i*311)%1_600}
	}
	return out
}

// englishSRT writes fixtureCues(n) as an English subtitle.
func englishSRT(t *testing.T, dir string, n int) string {
	t.Helper()
	var b strings.Builder
	for i, c := range fixtureCues(n) {
		fmt.Fprintf(&b, "%d\n%s --> %s\nWhat do you want me to say, Thom?\n\n", i+1, srtTime(c.StartMS), srtTime(c.EndMS))
	}
	p := filepath.Join(dir, "Film.en.srt")
	require.NoError(t, os.WriteFile(p, []byte(b.String()), 0o644))
	return p
}

// pcmFor synthesizes a window: a loud tone where the cue timeline is on.
func pcmFor(w window, line []bool) []byte {
	perFrame := sampleRate * frameMS / 1000
	frames := w.LenMS / frameMS
	out := make([]byte, frames*perFrame*2)
	for k := range frames {
		f := (w.StartMS + k*frameMS) / frameMS
		amp := 0.01
		if f < len(line) && line[f] {
			amp = 0.4
		}
		for s := range perFrame {
			v := amp * math.Sin(2*math.Pi*440*float64(k*perFrame+s)/sampleRate)
			binary.LittleEndian.PutUint16(out[(k*perFrame+s)*2:], uint16(int16(v*32767)))
		}
	}
	return out
}

func newTestChecker(t *testing.T, cache Cache, seconds float64) (*Checker, *int, string, string) {
	t.Helper()
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed — Check refuses to run without it")
	}
	dir := t.TempDir()
	media := filepath.Join(dir, "Film.mkv")
	require.NoError(t, os.WriteFile(media, []byte("video"), 0o644))
	sub := englishSRT(t, dir, 100)
	c := NewChecker(fakeProber{seconds}, cache, nil)
	c.hasCentre = func(context.Context, string, int) bool { return false }
	calls := 0
	c.decode = func(_ context.Context, _ string, _ int, w window, _ bool) ([]byte, error) {
		calls++
		// The audio is the subtitle as written: aligned.
		return pcmFor(w, timeline(fixtureCues(100))), nil
	}
	return c, &calls, media, sub
}

func TestCheck_AlignedAndCached(t *testing.T) {
	cache := &memoryCache{entries: map[string]string{}}
	c, calls, media, sub := newTestChecker(t, cache, 510)

	v, err := c.Check(context.Background(), media, 1, sub)
	require.NoError(t, err)
	assert.Equal(t, Aligned, v.Result)
	assert.False(t, v.Cached)
	decoded := *calls
	assert.Positive(t, decoded)

	v2, err := c.Check(context.Background(), media, 1, sub)
	require.NoError(t, err)
	assert.True(t, v2.Cached)
	assert.Equal(t, v.Result, v2.Result)
	assert.Equal(t, decoded, *calls, "the second ask never runs ffmpeg")

	// Touch the subtitle: a new key, decoded again.
	require.NoError(t, os.Chtimes(sub, time.Now().Add(time.Hour), time.Now().Add(time.Hour)))
	v3, err := c.Check(context.Background(), media, 1, sub)
	require.NoError(t, err)
	assert.False(t, v3.Cached)
	assert.Greater(t, *calls, decoded)

	// Another audio stream is another key.
	_, err = c.Check(context.Background(), media, 2, sub)
	require.NoError(t, err)
	assert.Len(t, cache.entries, 3)
}

func TestCheck_UnusableNeedsNoAudio(t *testing.T) {
	c, calls, media, _ := newTestChecker(t, nil, 510)
	dir := filepath.Dir(media)

	short := filepath.Join(dir, "short.srt")
	require.NoError(t, os.WriteFile(short, []byte("1\n00:00:01,000 --> 00:00:02,000\nHello\n"), 0o644))
	v, err := c.Check(context.Background(), media, 1, short)
	require.NoError(t, err)
	assert.Equal(t, Verdict{Result: Unusable, Reason: "too_few_cues", ElapsedMS: v.ElapsedMS}, v)

	garbage := filepath.Join(dir, "garbage.srt")
	require.NoError(t, os.WriteFile(garbage, []byte{0xff, 0xfe, 0x00}, 0o644))
	v, err = c.Check(context.Background(), media, 1, garbage)
	require.NoError(t, err)
	assert.Equal(t, Unusable, v.Result)

	assert.Zero(t, *calls, "an unusable subtitle never decodes audio")
}

func TestCheck_ErrorsAreUnknownNotAligned(t *testing.T) {
	c, _, media, sub := newTestChecker(t, nil, 0)
	_, err := c.Check(context.Background(), media, 1, sub)
	require.Error(t, err, "duration unknown")

	_, err = c.Check(context.Background(), filepath.Join(t.TempDir(), "gone.mkv"), 1, sub)
	require.Error(t, err)
	_, err = c.Check(context.Background(), media, 1, filepath.Join(t.TempDir(), "gone.srt"))
	require.Error(t, err)
}

// The real ffmpeg path: a synthetic film whose tone follows the subtitle,
// decoded window by window through decodeWindow.
func TestCheck_ThroughFFmpeg(t *testing.T) {
	for _, bin := range []string{"ffmpeg", "ffprobe"} {
		if _, err := exec.LookPath(bin); err != nil {
			t.Skipf("%s not installed", bin)
		}
	}
	dir := t.TempDir()
	sub := englishSRT(t, dir, 100)
	pcm := pcmFor(window{StartMS: 0, LenMS: 520_000}, timeline(fixtureCues(100)))
	raw := filepath.Join(dir, "audio.pcm")
	require.NoError(t, os.WriteFile(raw, pcm, 0o644))
	media := filepath.Join(dir, "Film.mkv")
	out, err := exec.Command("ffmpeg", "-loglevel", "error", "-f", "s16le", "-ar", "16000", "-ac", "1", "-i", raw,
		"-c:a", "aac", media).CombinedOutput()
	require.NoError(t, err, string(out))

	c := NewChecker(services.NewFFprobeService(1, time.Minute, nil), nil, nil)
	v, err := c.Check(context.Background(), media, 0, sub)
	require.NoError(t, err)
	assert.Equal(t, Aligned, v.Result, "%+v", v)
	assert.Len(t, v.Windows, 3, "8:40 leaves room for three 90 s windows")
}
