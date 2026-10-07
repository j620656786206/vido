package subtitle

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/services"
)

// disc-2026-10-episode-list-subtitle-badge-c AC #3 / #6.

// A couple of minutes of dialogue: plenty of characters that exist in only
// one script.
const (
	peekTraditional = "1\n00:04:01,000 --> 00:04:03,000\n這是我們的家園，我們團結一致。\n\n2\n00:04:05,000 --> 00:04:07,000\n說話之前先聽聽長輩們的經驗，會議時間還沒決定。\n"
	peekSimplified  = "1\n00:04:01,000 --> 00:04:03,000\n这是我们的家园，我们团结一致。\n\n2\n00:04:05,000 --> 00:04:07,000\n说话之前先听听长辈们的经验，会议时间还没决定。\n"
	// One stray Simplified character in otherwise shared text.
	peekThin = "1\n00:04:01,000 --> 00:04:03,000\n好。你好嗎？一个人\n"
)

var _ services.TrackScriptPeeker = (*ScriptPeeker)(nil)

// fakePeekFFmpeg writes, for each window start ("-ss" value), the given
// content to the output path (the last arg), and records the calls.
type fakePeekFFmpeg struct {
	byStart map[string]string
	calls   [][]string
	err     error
}

func (f *fakePeekFFmpeg) run(_ context.Context, args []string) error {
	f.calls = append(f.calls, args)
	if f.err != nil {
		return f.err
	}
	start := ""
	for i, a := range args {
		if a == "-ss" && i+1 < len(args) {
			start = args[i+1]
		}
	}
	content, ok := f.byStart[start]
	if !ok {
		return nil // ffmpeg exits 0 and writes nothing past the end of a file
	}
	return os.WriteFile(args[len(args)-1], []byte(content), 0o644)
}

func newTestPeeker(f *fakePeekFFmpeg) *ScriptPeeker {
	return &ScriptPeeker{available: true, timeout: time.Second, run: f.run}
}

func TestScriptPeeker_FirstWindowDecides(t *testing.T) {
	f := &fakePeekFFmpeg{byStart: map[string]string{"240": peekTraditional}}
	got, err := newTestPeeker(f).PeekScript(context.Background(), "/tv/See.S02E01.mkv", 3)
	require.NoError(t, err)
	assert.Equal(t, LangTraditional, got)
	require.Len(t, f.calls, 1, "a decided first window needs no second read")
	assert.Equal(t, []string{"-nostdin", "-y", "-v", "error", "-ss", "240", "-i", "/tv/See.S02E01.mkv",
		"-t", "150", "-map", "0:3", "-c:s", "srt"}, f.calls[0][:len(f.calls[0])-1],
		"-ss before -i (index seek), only the subtitle stream, short window")
}

func TestScriptPeeker_EmptyFirstWindowFallsBackToTheOpening(t *testing.T) {
	f := &fakePeekFFmpeg{byStart: map[string]string{"240": "", "0": peekSimplified}}
	got, err := newTestPeeker(f).PeekScript(context.Background(), "/tv/x.mkv", 2)
	require.NoError(t, err)
	assert.Equal(t, LangSimplified, got)
	assert.Len(t, f.calls, 2)
}

func TestScriptPeeker_UndecidedAnswers(t *testing.T) {
	got, err := newTestPeeker(&fakePeekFFmpeg{}).PeekScript(context.Background(), "/tv/x.mkv", 2)
	require.NoError(t, err)
	assert.Equal(t, LangUndetermined, got, "no Chinese characters in either window is an answer, not an error")

	mixed := "1\n00:00:01,000 --> 00:00:03,000\n這是我們的家园，我们團結一致\n"
	got, err = newTestPeeker(&fakePeekFFmpeg{byStart: map[string]string{"240": mixed}}).PeekScript(context.Background(), "/tv/x.mkv", 2)
	require.NoError(t, err)
	assert.Equal(t, LangAmbiguous, got)
}

func TestScriptPeeker_FFmpegFailureIsAnError(t *testing.T) {
	_, err := newTestPeeker(&fakePeekFFmpeg{err: errors.New("exit status 1")}).PeekScript(context.Background(), "/tv/x.mkv", 2)
	assert.Error(t, err)

	_, err = (&ScriptPeeker{}).PeekScript(context.Background(), "/tv/x.mkv", 2)
	assert.ErrorIs(t, err, services.ErrFFmpegNotAvailable)
}

func TestScriptPeeker_Peekable(t *testing.T) {
	p := &ScriptPeeker{}
	assert.True(t, p.Peekable(services.SubtitleTrack{Language: "chi", Format: "subrip"}))
	assert.True(t, p.Peekable(services.SubtitleTrack{Language: "chi", Format: "ass", Title: "Chinese"}))
	assert.False(t, p.Peekable(services.SubtitleTrack{Language: "chi", Format: "subrip", Title: "繁體中文"}), "the title already tells")
	assert.False(t, p.Peekable(services.SubtitleTrack{Language: "chi", Format: "hdmv_pgs_subtitle"}), "bitmap: no characters to read")
	assert.False(t, p.Peekable(services.SubtitleTrack{Language: "eng", Format: "subrip"}))
	assert.False(t, p.Peekable(services.SubtitleTrack{Language: "zh-Hant", Format: "srt", External: true}))
}

// CR M1/M2: a window with a handful of lines decides nothing on its own — one
// stray 个 must not label a track Simplified (which would then outrank its
// title); its characters join the next window's.
func TestScriptPeeker_ThinWindowDoesNotDecide(t *testing.T) {
	f := &fakePeekFFmpeg{byStart: map[string]string{"240": peekThin, "0": peekTraditional}}
	got, err := newTestPeeker(f).PeekScript(context.Background(), "/tv/x.mkv", 2)
	require.NoError(t, err)
	assert.Equal(t, LangTraditional, got)
	assert.Len(t, f.calls, 2)

	f = &fakePeekFFmpeg{byStart: map[string]string{"240": peekThin}}
	got, err = newTestPeeker(f).PeekScript(context.Background(), "/tv/x.mkv", 2)
	require.NoError(t, err)
	assert.Equal(t, LangAmbiguous, got, "Chinese, but too little to tell — not Simplified")
}
