package sidecarcheck

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/services"
)

// ─── disc-2026-10-generate-keeps-user-en-srt-b2 ─────────────────────────────

type fakeAudio struct {
	err   error
	calls atomic.Int32
}

func (a *fakeAudio) ListAudioTracks(context.Context, string) ([]services.AudioTrack, error) {
	a.calls.Add(1)
	if a.err != nil {
		return nil, a.err
	}
	return []services.AudioTrack{{Index: 3, Language: "jpn"}, {Index: 1, Language: "eng"}}, nil
}

// finderFixture is a film with whatever sidecars the test writes, and a
// checker whose "audio" is fixtureCues(100): a subtitle with those cues is
// aligned, the same shifted by 40 s is not.
type finderFixture struct {
	dir, media string
	finder     *Finder
	audio      *fakeAudio
	decodes    *atomic.Int32
	streams    chan int
}

func newFinderFixture(t *testing.T) *finderFixture {
	return newFinderFixtureWith(t, &memoryCache{entries: map[string]string{}})
}

func newFinderFixtureWith(t *testing.T, cache Cache) *finderFixture {
	t.Helper()
	dir := t.TempDir()
	media := filepath.Join(dir, "Film.mkv")
	require.NoError(t, os.WriteFile(media, []byte("video"), 0o644))
	c := NewChecker(fakeProber{510}, cache, nil)
	c.hasFFmpeg = func() bool { return true }
	c.hasCentre = func(context.Context, string, int) bool { return false }
	fx := &finderFixture{dir: dir, media: media, audio: &fakeAudio{}, decodes: &atomic.Int32{}, streams: make(chan int, 64)}
	c.decode = func(_ context.Context, _ string, stream int, w window, _ bool) ([]byte, error) {
		fx.decodes.Add(1)
		fx.streams <- stream
		return pcmFor(w, timeline(fixtureCues(100))), nil
	}
	fx.finder = NewFinder(c, fx.audio, nil)
	return fx
}

// write puts an English subtitle beside the film, shifted by shiftMS.
func (fx *finderFixture) write(t *testing.T, name string, shiftMS int) string {
	t.Helper()
	cues := fixtureCues(100)
	var b []byte
	for i, c := range cues {
		b = append(b, []byte(
			srtBlock(i+1, c.StartMS+shiftMS, c.EndMS+shiftMS, "What do you want me to say, Thom?"))...)
	}
	p := filepath.Join(fx.dir, name)
	require.NoError(t, os.WriteFile(p, b, 0o644))
	return p
}

func srtBlock(i, s, e int, text string) string {
	return fmt.Sprintf("%d\n%s --> %s\n%s\n\n", i, srtTime(s), srtTime(e), text)
}

func TestCandidates_UserFilesBeforeVidosTranscript(t *testing.T) {
	fx := newFinderFixture(t)
	vido := fx.write(t, "Film.vido.en.srt", 0)
	user := fx.write(t, "Film.en.srt", 0)
	sdh := fx.write(t, "Film.en.sdh.srt", 0)
	fx.write(t, "Film.zh-TW.srt", 0) // Chinese tag, English text: undetermined, not a candidate
	require.NoError(t, os.WriteFile(filepath.Join(fx.dir, "Film.en.ass"), []byte("[Script Info]"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(fx.dir, "Other.en.srt"), []byte("x"), 0o644))

	assert.Equal(t, []string{user, sdh, vido}, candidates(fx.media),
		"the user's .srt files by name, Vido's own transcript last; .ass and other films' files are not read")
}

func TestFindAligned_PicksTheFirstMatchOnTheEnglishAudio(t *testing.T) {
	fx := newFinderFixture(t)
	fx.write(t, "Film.en.srt", 40_000) // the user's file is for another cut
	vido := fx.write(t, "Film.vido.en.srt", 0)

	path, ok := fx.finder.FindAligned(context.Background(), fx.media)
	require.True(t, ok)
	assert.Equal(t, vido, path, "the user's file does not match, Vido's own transcript does")
	assert.Equal(t, 1, <-fx.streams, "checked on the English audio stream, not the first one")
}

func TestFindAligned_NothingBesideTheFile(t *testing.T) {
	fx := newFinderFixture(t)
	_, ok := fx.finder.FindAligned(context.Background(), fx.media)
	assert.False(t, ok)
	assert.Zero(t, fx.audio.calls.Load(), "no sidecar: the audio is never even listed")
}

func TestPeek_StatesAndBackgroundCheck(t *testing.T) {
	fx := newFinderFixture(t)
	ctx := context.Background()

	assert.Equal(t, services.SidecarAlignment{Status: services.SidecarStatusNone}, fx.finder.Peek(ctx, fx.media))

	fx.write(t, "Film.en.srt", 0)
	got := fx.finder.Peek(ctx, fx.media)
	assert.Equal(t, services.SidecarStatusChecking, got.Status, "nothing stored: checking, and the caller is not made to wait")
	assert.Equal(t, "Film.en.srt", got.FileName)
	// A second quote while it runs does not start a second check.
	fx.finder.Peek(ctx, fx.media)

	require.Eventually(t, func() bool {
		return fx.finder.Peek(ctx, fx.media).Status == services.SidecarStatusAligned
	}, 5*time.Second, 10*time.Millisecond)
	assert.LessOrEqual(t, fx.audio.calls.Load(), int32(1), "one background check per film at a time")

	decodes := fx.decodes.Load()
	fx.finder.Peek(ctx, fx.media)
	assert.Equal(t, decodes, fx.decodes.Load(), "a stored answer never decodes again")
}

func TestPeek_ReportsTheFirstVerdictWhenNoneMatches(t *testing.T) {
	fx := newFinderFixture(t)
	ctx := context.Background()
	fx.write(t, "Film.en.srt", 40_000)

	_, ok := fx.finder.FindAligned(ctx, fx.media) // stores the verdict
	require.False(t, ok)
	got := fx.finder.Peek(ctx, fx.media)
	assert.Equal(t, services.SidecarStatusMismatch, got.Status)
	assert.Equal(t, "Film.en.srt", got.FileName)
}

func TestPeek_ACheckThatCannotRunIsUnknownNotRetriedEveryQuote(t *testing.T) {
	fx := newFinderFixture(t)
	ctx := context.Background()
	fx.write(t, "Film.en.srt", 0)
	fx.audio.err = errors.New("ffprobe: no such file")

	assert.Equal(t, services.SidecarStatusChecking, fx.finder.Peek(ctx, fx.media).Status)
	require.Eventually(t, func() bool {
		return fx.finder.Peek(ctx, fx.media).Status == services.SidecarStatusUnknown
	}, 5*time.Second, 10*time.Millisecond)
	calls := fx.audio.calls.Load()
	fx.finder.Peek(ctx, fx.media)
	assert.Equal(t, calls, fx.audio.calls.Load(), "within failedFor, a quote does not start the check again")
}

// CR: a cache that stores nothing (none wired, a failed write) must not turn
// every quote into a new decode — the dialog would poll "checking" forever.
func TestPeek_NoStoredVerdictStillAnswersFromTheLastRun(t *testing.T) {
	fx := newFinderFixtureWith(t, nil)
	ctx := context.Background()
	fx.write(t, "Film.en.srt", 0)

	require.Equal(t, services.SidecarStatusChecking, fx.finder.Peek(ctx, fx.media).Status)
	require.Eventually(t, func() bool {
		return fx.finder.Peek(ctx, fx.media).Status == services.SidecarStatusAligned
	}, 5*time.Second, 10*time.Millisecond)
	decodes := fx.decodes.Load()
	for range 5 {
		fx.finder.Peek(ctx, fx.media)
	}
	assert.Equal(t, decodes, fx.decodes.Load())
}

// CR: a click while the dialog's background check runs waits for that check
// instead of decoding the film a second time; many quotes start one check.
func TestFinder_ConcurrentCallersShareOneCheck(t *testing.T) {
	fx := newFinderFixtureWith(t, nil)
	ctx := context.Background()
	fx.write(t, "Film.en.srt", 0)

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(2)
		go func() { defer wg.Done(); fx.finder.Peek(ctx, fx.media) }()
		go func() { defer wg.Done(); fx.finder.FindAligned(ctx, fx.media) }()
	}
	wg.Wait()
	require.Eventually(t, func() bool {
		return fx.finder.Peek(ctx, fx.media).Status == services.SidecarStatusAligned
	}, 5*time.Second, 10*time.Millisecond)

	windows := int32(len(planWindows(510_000, windowCount, int(windowLen.Milliseconds()), minWindows)))
	assert.LessOrEqual(t, fx.decodes.Load(), 2*windows,
		"at most the run and one background check — not one per caller (got %d decodes)", fx.decodes.Load())
}
