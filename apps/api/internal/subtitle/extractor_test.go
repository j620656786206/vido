package subtitle

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/services"
)

// ─── Candidate filter (AC #3.1) ────────────────────────────────────────────

func TestSelectCandidates(t *testing.T) {
	tests := []struct {
		name    string
		tracks  []services.SubtitleTrack
		wantIdx []int
	}{
		{
			name:    "no tracks",
			tracks:  nil,
			wantIdx: nil,
		},
		{
			name: "eng subrip is a candidate",
			tracks: []services.SubtitleTrack{
				{Language: "eng", Format: "subrip", StreamIndex: 2},
			},
			wantIdx: []int{2},
		},
		{
			name: "en (2-letter) is a candidate",
			tracks: []services.SubtitleTrack{
				{Language: "en", Format: "ass", StreamIndex: 3},
			},
			wantIdx: []int{3},
		},
		{
			name: "language tag matching is case-insensitive",
			tracks: []services.SubtitleTrack{
				{Language: "ENG", Format: "SubRip", StreamIndex: 4},
			},
			wantIdx: []int{4},
		},
		{
			name: "every text codec in the set is admitted",
			tracks: []services.SubtitleTrack{
				{Language: "eng", Format: "subrip", StreamIndex: 1},
				{Language: "eng", Format: "srt", StreamIndex: 2},
				{Language: "eng", Format: "ass", StreamIndex: 3},
				{Language: "eng", Format: "ssa", StreamIndex: 4},
				{Language: "eng", Format: "mov_text", StreamIndex: 5},
				{Language: "eng", Format: "webvtt", StreamIndex: 6},
			},
			wantIdx: []int{1, 2, 3, 4, 5, 6},
		},
		{
			name: "image codecs are never candidates even when eng-tagged",
			tracks: []services.SubtitleTrack{
				{Language: "eng", Format: "hdmv_pgs_subtitle", StreamIndex: 1},
				{Language: "eng", Format: "dvd_subtitle", StreamIndex: 2},
				{Language: "eng", Format: "dvb_subtitle", StreamIndex: 3},
				{Language: "eng", Format: "xsub", StreamIndex: 4},
			},
			wantIdx: nil,
		},
		{
			name: "und is NEVER English (P0)",
			tracks: []services.SubtitleTrack{
				{Language: "und", Format: "subrip", StreamIndex: 2},
			},
			wantIdx: nil,
		},
		{
			name: "non-English tags are excluded",
			tracks: []services.SubtitleTrack{
				{Language: "jpn", Format: "subrip", StreamIndex: 1},
				{Language: "zh", Format: "ass", StreamIndex: 2},
				{Language: "chi", Format: "subrip", StreamIndex: 3},
			},
			wantIdx: nil,
		},
		{
			name: "external sidecars are excluded (embedded-only pipeline)",
			tracks: []services.SubtitleTrack{
				{Language: "eng", Format: "srt", External: true, StreamIndex: 0},
			},
			wantIdx: nil,
		},
		{
			name: "mixed container keeps only the eng text tracks, in probe order",
			tracks: []services.SubtitleTrack{
				{Language: "chi", Format: "subrip", StreamIndex: 2},
				{Language: "eng", Format: "hdmv_pgs_subtitle", StreamIndex: 3},
				{Language: "eng", Format: "subrip", StreamIndex: 4},
				{Language: "en", Format: "mov_text", StreamIndex: 6},
				{Language: "eng", Format: "srt", External: true},
			},
			wantIdx: []int{4, 6},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SelectCandidates(tt.tracks)

			gotIdx := make([]int, 0, len(got))
			for _, c := range got {
				gotIdx = append(gotIdx, c.StreamIndex)
			}
			if tt.wantIdx == nil {
				assert.Empty(t, gotIdx)
				return
			}
			assert.Equal(t, tt.wantIdx, gotIdx)
		})
	}
}

// ─── Argument assembly (AC #3.2, AC #3.3, Finding 2) ───────────────────────

func TestBuildExtractArgs(t *testing.T) {
	tmpDir := "/tmp/vido-sub-123"
	media := "/media/movies/Some Movie (2024).mkv"

	tests := []struct {
		name    string
		indexes []int
		want    []string
	}{
		{
			name:    "single track",
			indexes: []int{2},
			want: []string{
				"-nostdin", "-y", "-i", media,
				"-map", "0:2", "-c:s", "srt", filepath.Join(tmpDir, "track_2.srt"),
			},
		},
		{
			name:    "three tracks share ONE invocation (FR3 — one read pass)",
			indexes: []int{2, 5, 7},
			want: []string{
				"-nostdin", "-y", "-i", media,
				"-map", "0:2", "-c:s", "srt", filepath.Join(tmpDir, "track_2.srt"),
				"-map", "0:5", "-c:s", "srt", filepath.Join(tmpDir, "track_5.srt"),
				"-map", "0:7", "-c:s", "srt", filepath.Join(tmpDir, "track_7.srt"),
			},
		},
		{
			name:    "stream index 0 is legal",
			indexes: []int{0},
			want: []string{
				"-nostdin", "-y", "-i", media,
				"-map", "0:0", "-c:s", "srt", filepath.Join(tmpDir, "track_0.srt"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, buildExtractArgs(media, tmpDir, tt.indexes))
		})
	}
}

func TestBuildExtractArgs_NeverWritesBesideTheMedia(t *testing.T) {
	// D3 — placer.go is the sole writer into the media folder, and a stray
	// .eng.srt sidecar would be auto-detected by Plex/Jellyfin/Video Station.
	media := "/media/movies/Some Movie (2024).mkv"
	tmpDir := t.TempDir()

	args := buildExtractArgs(media, tmpDir, []int{2, 3})

	mediaDir := filepath.Dir(media)
	for _, a := range args {
		if a == media {
			continue // the -i input itself
		}
		assert.NotEqual(t, mediaDir, filepath.Dir(a), "output path must not live beside the media: %s", a)
	}
	assert.Equal(t, tmpDir, filepath.Dir(trackOutputPath(tmpDir, 2)))
}

func TestBuildExtractArgs_UniformSrtCodecForEveryCandidate(t *testing.T) {
	// Finding 2 — `-c:s copy` into an .srt muxer only works when the source is
	// already subrip; ass/ssa/mov_text need the `srt` subtitle-codec transform.
	args := buildExtractArgs("/m.mkv", "/tmp/x", []int{1, 2, 3})

	assert.Equal(t, 3, countPairs(args, "-c:s", "srt"))
	assert.NotContains(t, args, "copy")
	assert.NotContains(t, args, "-c")
}

func countPairs(args []string, flag, value string) int {
	n := 0
	for i := 0; i+1 < len(args); i++ {
		if args[i] == flag && args[i+1] == value {
			n++
		}
	}
	return n
}

func TestTrackOutputPath(t *testing.T) {
	assert.Equal(t, filepath.Join("/tmp/x", "track_0.srt"), trackOutputPath("/tmp/x", 0))
	assert.Equal(t, filepath.Join("/tmp/x", "track_12.srt"), trackOutputPath("/tmp/x", 12))
}

// ─── Construction (AC #3.4) ────────────────────────────────────────────────

func TestNewExtractor_Defaults(t *testing.T) {
	e := NewExtractor(0, nil)
	assert.Equal(t, defaultExtractTimeout, e.timeout)
	assert.NotNil(t, e.logger)

	custom := NewExtractor(90*time.Second, nil)
	assert.Equal(t, 90*time.Second, custom.timeout)
}

func TestExtractor_NotAvailable(t *testing.T) {
	e := NewExtractor(time.Second, nil)
	e.available = false

	got, err := e.Extract(context.Background(), "/m.mkv", t.TempDir(), []int{1})
	assert.Nil(t, got)
	assert.ErrorIs(t, err, services.ErrFFmpegNotAvailable)
}

func TestExtractor_NoIndexes(t *testing.T) {
	e := NewExtractor(time.Second, nil)
	e.available = true

	got, err := e.Extract(context.Background(), "/m.mkv", t.TempDir(), nil)
	assert.Nil(t, got)
	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrSubtitleExtractFailed, "an empty call is a caller bug, not an ffmpeg failure")
}

// ─── Error classification (AC #3.5, Rule 13) ───────────────────────────────

func TestExtractFailure_WrapsSentinelWithStderrTail(t *testing.T) {
	err := extractFailure("run ffmpeg", errors.New("exit status 1"),
		"Stream map '0:9' matches no streams.\n")

	assert.ErrorIs(t, err, ErrSubtitleExtractFailed)
	assert.Contains(t, err.Error(), "run ffmpeg")
	assert.Contains(t, err.Error(), "exit status 1")
	assert.Contains(t, err.Error(), "matches no streams")
}

func TestExtractFailure_NilCauseStillClassifies(t *testing.T) {
	// The missing-output-file branch has no exec error to wrap.
	err := extractFailure("output missing for stream 3", nil, "")

	assert.ErrorIs(t, err, ErrSubtitleExtractFailed)
	assert.Contains(t, err.Error(), "output missing for stream 3")
}

func TestExtractFailure_CauseStaysInTheChain(t *testing.T) {
	// CR sub-1-4 M1 — the cause must be chained with %w, not flattened by %v:
	// the orchestrator (1.5b) tells a shutdown/cancel apart from a genuine
	// ffmpeg failure via errors.Is(err, context.Canceled).
	err := extractFailure("run ffmpeg", context.Canceled, "")

	assert.ErrorIs(t, err, ErrSubtitleExtractFailed)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestExtract_ParentCancellationIsNotReportedAsTimeout(t *testing.T) {
	// CR sub-1-4 M2 — extractCtx.Err() is also non-nil when the PARENT ctx was
	// cancelled; that must classify as a cancellation, never as "timed out".
	e := NewExtractor(time.Minute, nil)
	e.available = true

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := e.Extract(ctx, "/m.mkv", t.TempDir(), []int{1})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSubtitleExtractFailed)
	assert.ErrorIs(t, err, context.Canceled)
	assert.NotContains(t, err.Error(), "timed out", "a cancel is not a timeout")
	assert.Contains(t, err.Error(), "cancelled")
}

func TestStderrTail(t *testing.T) {
	assert.Equal(t, "", stderrTail("   \n\n  "))
	assert.Equal(t, "boom", stderrTail("boom\n"))

	long := strings.Repeat("x", stderrTailBytes+200)
	tail := stderrTail(long)
	assert.Len(t, tail, stderrTailBytes)
}

func TestStderrTail_RespectsRuneBoundaries(t *testing.T) {
	// CR sub-1-4 L2 — a non-ASCII filename in ffmpeg stderr must not leave
	// invalid UTF-8 in the error message when the tail cut lands mid-rune.
	long := strings.Repeat("檔", stderrTailBytes) // 3-byte runes ≫ the cap
	tail := stderrTail(long)

	assert.True(t, utf8.ValidString(tail), "tail must remain valid UTF-8")
	assert.LessOrEqual(t, len(tail), stderrTailBytes)
	assert.NotEmpty(t, tail)
}

// ─── Real-ffmpeg integration (availability-gated, ffprobe_service_test.go precedent) ───

func TestExtractor_Integration_RealFFmpeg(t *testing.T) {
	e := NewExtractor(2*time.Minute, nil)
	if !e.IsAvailable() {
		t.Skip("ffmpeg not installed — skipping real-extraction integration case")
	}

	dir := t.TempDir()
	mediaPath := filepath.Join(dir, "fixture.mkv")
	srtPath := filepath.Join(dir, "source.srt")

	const srcSRT = "1\n00:00:00,500 --> 00:00:01,500\nHello there\n\n2\n00:00:02,000 --> 00:00:03,000\nGeneral Kenobi\n"
	require.NoError(t, os.WriteFile(srtPath, []byte(srcSRT), 0o600))

	// Mux a 2s black video + the srt track, tagged eng, into an mkv.
	//nolint:gosec // fixture paths are test-owned temp files
	build := exec.Command("ffmpeg", "-nostdin", "-y",
		"-f", "lavfi", "-i", "color=c=black:s=64x64:d=2",
		"-i", srtPath,
		"-map", "0:v", "-map", "1:s",
		"-c:v", "libx264", "-preset", "ultrafast",
		"-c:s", "srt",
		"-metadata:s:s:0", "language=eng",
		mediaPath,
	)
	if out, err := build.CombinedOutput(); err != nil {
		t.Skipf("ffmpeg build lacks the encoders needed for the fixture: %v\n%s", err, stderrTail(string(out)))
	}

	outputs, err := e.Extract(context.Background(), mediaPath, dir, []int{1})
	require.NoError(t, err)
	require.Len(t, outputs, 1)

	raw, err := os.ReadFile(outputs[1])
	require.NoError(t, err)

	blocks, err := ParseSRT(string(raw))
	require.NoError(t, err)
	require.Len(t, blocks, 2)
	assert.Equal(t, "Hello there", blocks[0].Text)
	assert.Equal(t, "00:00:00,500", blocks[0].Start)
	assert.Equal(t, "General Kenobi", blocks[1].Text)
}

func TestExtractor_Integration_MissingStreamClassifies(t *testing.T) {
	e := NewExtractor(30*time.Second, nil)
	if !e.IsAvailable() {
		t.Skip("ffmpeg not installed — skipping sentinel-classification integration case")
	}

	dir := t.TempDir()
	_, err := e.Extract(context.Background(), filepath.Join(dir, "does-not-exist.mkv"), dir, []int{1})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSubtitleExtractFailed,
		fmt.Sprintf("ffmpeg failure must classify as the sub-1-3 sentinel, got: %v", err))
}
