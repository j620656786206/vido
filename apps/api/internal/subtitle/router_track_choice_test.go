package subtitle

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/services"
)

// ─── disc-2026-10-english-track-selection ──────────────────────────────────
//
// See S01E02 (2026-10-05 eval) ships three English text tracks: the full one,
// an SDH one and a one-cue forced track whose only line is on-screen text
// ("We are not alone"). Selection used to be "most cues after SDH filtering",
// which ignores what the muxer says a track IS and always throws the forced
// track away.

// timedCue is one cue of a hand-built SRT, in seconds.
type timedCue struct {
	start, end float64
	text       string
}

// srtTimed builds an SRT document from explicit timings.
func srtTimed(cues ...timedCue) string {
	var sb strings.Builder
	for i, c := range cues {
		fmt.Fprintf(&sb, "%d\n%s --> %s\n%s\n\n", i+1, srtStamp(c.start), srtStamp(c.end), c.text)
	}
	return sb.String()
}

func srtStamp(sec float64) string {
	ms := int(sec*1000 + 0.5)
	return fmt.Sprintf("%02d:%02d:%02d,%03d", ms/3600000, ms/60000%60, ms/1000%60, ms%1000)
}

// nLines returns n distinct dialogue lines, one per second from startSec.
func nLines(prefix string, n int, startSec float64) []timedCue {
	out := make([]timedCue, n)
	for i := range out {
		s := startSec + float64(i)
		out[i] = timedCue{start: s, end: s + 0.8, text: fmt.Sprintf("%s line %d", prefix, i+1)}
	}
	return out
}

func track(idx int, lang, title string) services.SubtitleTrack {
	return services.SubtitleTrack{StreamIndex: idx, Language: lang, Format: "subrip", Title: title}
}

func newLoggingRouter(t *testing.T, prober *fakeProber, ex *fakeExtractor) (*Router, string, *bytes.Buffer) {
	t.Helper()
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	return NewRouter(prober, ex, logger), t.TempDir(), &buf
}

// ─── AC #2 — classification ────────────────────────────────────────────────

func TestClassifyTrack(t *testing.T) {
	cases := []struct {
		name  string
		track services.SubtitleTrack
		want  trackKind
	}{
		{"no flags no title", track(2, "eng", ""), trackRegular},
		{"plain title", track(2, "eng", "English"), trackRegular},
		{"forced flag only", services.SubtitleTrack{Language: "eng", Forced: true}, trackForced},
		{"hearing-impaired flag only", services.SubtitleTrack{Language: "eng", HearingImpaired: true}, trackSDH},
		{"title [SDH]", track(2, "eng", "English [SDH]"), trackSDH},
		{"title (Forced)", track(2, "eng", "English (Forced)"), trackForced},
		{"title CC", track(2, "eng", "English CC"), trackSDH},
		{"title hearing impaired", track(2, "eng", "English (Hearing Impaired)"), trackSDH},
		{"title 強制", track(2, "chi", "繁體中文 (強制)"), trackForced},
		{"title 聽障", track(2, "chi", "繁體中文（聽障）"), trackSDH},
		{"both flags → forced wins", services.SubtitleTrack{Language: "eng", Forced: true, HearingImpaired: true}, trackForced},
		{"forced title + SDH flag → forced wins", services.SubtitleTrack{Language: "eng", Title: "Forced", HearingImpaired: true}, trackForced},
		{"cc inside a word is not SDH", track(2, "eng", "Accent Commentary"), trackRegular},
		{"sdh inside a word is not SDH", track(2, "eng", "Gsdhx"), trackRegular},
		{"Non-Forced is regular", track(2, "eng", "English (Non-Forced)"), trackRegular},
		{"Not Forced is regular", track(2, "eng", "English (Not Forced)"), trackRegular},
		{"underscore SDH", track(2, "eng", "English_SDH"), trackSDH},
		{"underscore Forced", track(2, "eng", "English_Forced"), trackForced},
		{"[HI]", track(2, "eng", "English [HI]"), trackSDH},
		{"hi inside a word is not SDH", track(2, "eng", "Hindi"), trackRegular},
		{"簡體 强制", track(2, "chi", "简体中文（强制）"), trackForced},
		{"簡體 听障", track(2, "chi", "简体（听障）"), trackSDH},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, classifyTrack(tc.track))
		})
	}
}

// ─── AC #3 — a regular track beats an SDH one with more cues ───────────────

func TestSelectAndRoute_RegularBeatsSDHWithMoreCues(t *testing.T) {
	prober := &fakeProber{info: &services.MediaTechInfo{SubtitleTracks: []services.SubtitleTrack{
		track(3, "eng", "English [SDH]"),
		track(4, "eng", "English"),
	}}}
	ex := &fakeExtractor{contents: map[int]string{
		3: srtTimed(nLines("sdh", 12, 0)...),
		4: srtTimed(nLines("full", 10, 0)...),
	}}
	r, tmp := newTestRouter(t, prober, ex)

	got, err := r.SelectAndRoute(context.Background(), "/media/m.mkv", tmp)

	require.NoError(t, err)
	require.NotNil(t, got.Track)
	assert.Equal(t, 4, got.Track.StreamIndex)
	assert.Len(t, got.Track.Blocks, 10)
}

// ─── AC #4 — a forced track never becomes the main track ───────────────────

func TestSelectAndRoute_ForcedTrackNeverMainEvenWithMostCues(t *testing.T) {
	prober := &fakeProber{info: &services.MediaTechInfo{SubtitleTracks: []services.SubtitleTrack{
		{StreamIndex: 2, Language: "eng", Format: "subrip", Forced: true},
		track(5, "eng", ""),
	}}}
	ex := &fakeExtractor{contents: map[int]string{
		2: srtTimed(nLines("forced", 9, 0)...),
		5: srtTimed(nLines("full", 8, 0)...),
	}}
	r, tmp := newTestRouter(t, prober, ex)

	got, err := r.SelectAndRoute(context.Background(), "/media/m.mkv", tmp)

	require.NoError(t, err)
	require.NotNil(t, got.Track)
	assert.Equal(t, 5, got.Track.StreamIndex)
}

func TestSelectAndRoute_OnlyForcedTrackFallsBackToLegacyPickAndWarns(t *testing.T) {
	prober := &fakeProber{info: &services.MediaTechInfo{SubtitleTracks: []services.SubtitleTrack{
		{StreamIndex: 2, Language: "eng", Format: "subrip", Forced: true},
	}}}
	ex := &fakeExtractor{contents: map[int]string{
		2: srtTimed(nLines("forced", 3, 0)...),
	}}
	r, tmp, logs := newLoggingRouter(t, prober, ex)

	got, err := r.SelectAndRoute(context.Background(), "/media/m.mkv", tmp)

	require.NoError(t, err)
	require.NotNil(t, got.Track)
	assert.Equal(t, 2, got.Track.StreamIndex, "the old pick, not a worse verdict")
	assert.Equal(t, RouteTranslate, got.Kind)
	assert.Len(t, got.Track.Blocks, 3, "a fallback main track never merges itself")
	assert.Contains(t, logs.String(), "level=WARN")
	assert.Contains(t, logs.String(), "no subtitle track qualifies as the main track")
	assert.Contains(t, logs.String(), "stream 2: forced", "the warning names why each track was rejected")
}

func TestSelectAndRoute_FallbackMainStillMergesForcedTrack(t *testing.T) {
	// ffprobe reported a bogus duration, so the regular track fails coverage and
	// the fallback pick is used — the on-screen line must still be merged.
	prober := &fakeProber{info: &services.MediaTechInfo{
		DurationSeconds: 99999,
		SubtitleTracks: []services.SubtitleTrack{
			track(4, "eng", "English"),
			{StreamIndex: 6, Language: "eng", Format: "subrip", Forced: true},
		},
	}}
	ex := &fakeExtractor{contents: map[int]string{
		4: srtTimed(nLines("full", 6, 0)...),
		6: srtTimed(timedCue{20, 21, "We are not alone"}),
	}}
	r, tmp := newTestRouter(t, prober, ex)

	got, err := r.SelectAndRoute(context.Background(), "/media/m.mkv", tmp)

	require.NoError(t, err)
	assert.Equal(t, 4, got.Track.StreamIndex)
	assert.Len(t, got.Track.Blocks, 7)
	assert.Contains(t, cueText(got.Track.Blocks), "We are not alone")
}

// ─── AC #5 — the 50% thresholds ────────────────────────────────────────────

func TestSelectAndRoute_TinyRegularTrackLosesToFullSDH(t *testing.T) {
	// An unflagged 3-cue track is a forced track nobody labelled.
	prober := &fakeProber{info: &services.MediaTechInfo{SubtitleTracks: []services.SubtitleTrack{
		track(2, "eng", ""),
		track(3, "eng", "English SDH"),
	}}}
	ex := &fakeExtractor{contents: map[int]string{
		2: srtTimed(nLines("tiny", 3, 0)...),
		3: srtTimed(nLines("sdh", 20, 0)...),
	}}
	r, tmp := newTestRouter(t, prober, ex)

	got, err := r.SelectAndRoute(context.Background(), "/media/m.mkv", tmp)

	require.NoError(t, err)
	assert.Equal(t, 3, got.Track.StreamIndex)
}

func TestSelectAndRoute_TrackEndingBeforeHalfTheFileLosesToFullSDH(t *testing.T) {
	half := append(nLines("half", 20, 0), timedCue{start: 899, end: 900, text: "last line at 15 min"})
	full := append(nLines("sdh", 20, 0), timedCue{start: 2900, end: 2902, text: "last line near the end"})
	prober := &fakeProber{info: &services.MediaTechInfo{
		DurationSeconds: 3000,
		SubtitleTracks: []services.SubtitleTrack{
			track(2, "eng", ""),
			track(3, "eng", "English [SDH]"),
		},
	}}
	ex := &fakeExtractor{contents: map[int]string{
		2: srtTimed(half...),
		3: srtTimed(full...),
	}}
	r, tmp := newTestRouter(t, prober, ex)

	got, err := r.SelectAndRoute(context.Background(), "/media/m.mkv", tmp)

	require.NoError(t, err)
	assert.Equal(t, 3, got.Track.StreamIndex)
}

func TestSelectAndRoute_UnknownDurationSkipsTheCoverageRule(t *testing.T) {
	half := append(nLines("half", 20, 0), timedCue{start: 899, end: 900, text: "last line at 15 min"})
	full := append(nLines("sdh", 20, 0), timedCue{start: 2900, end: 2902, text: "last line near the end"})
	prober := &fakeProber{info: &services.MediaTechInfo{
		DurationSeconds: 0, // unknown
		SubtitleTracks: []services.SubtitleTrack{
			track(2, "eng", ""),
			track(3, "eng", "English [SDH]"),
		},
	}}
	ex := &fakeExtractor{contents: map[int]string{
		2: srtTimed(half...),
		3: srtTimed(full...),
	}}
	r, tmp := newTestRouter(t, prober, ex)

	got, err := r.SelectAndRoute(context.Background(), "/media/m.mkv", tmp)

	require.NoError(t, err)
	assert.Equal(t, 2, got.Track.StreamIndex, "regular beats SDH when coverage cannot be judged")
}

// ─── AC #6 — merging the forced track on the English translate route ───────

func TestSelectAndRoute_MergesForcedCuesIntoTheTranslateTrack(t *testing.T) {
	main := []timedCue{
		{1, 2, "One"},
		{3, 4, "Two"},
		{5, 6, "We are leaving."},
		{9, 10, "Four"},
		{11, 12, "Five"},
	}
	forced := []timedCue{
		{5.2, 5.8, "We are leaving!"}, // same words as main cue 3, overlapping → dropped
		{7, 8, "We are not alone"},    // on-screen text → merged between Two/Four
	}
	prober := &fakeProber{info: &services.MediaTechInfo{SubtitleTracks: []services.SubtitleTrack{
		track(4, "eng", "English"),
		{StreamIndex: 6, Language: "eng", Format: "subrip", Forced: true},
	}}}
	ex := &fakeExtractor{contents: map[int]string{
		4: srtTimed(main...),
		6: srtTimed(forced...),
	}}
	r, tmp, logs := newLoggingRouter(t, prober, ex)

	got, err := r.SelectAndRoute(context.Background(), "/media/m.mkv", tmp)

	require.NoError(t, err)
	require.Equal(t, RouteTranslate, got.Kind)
	require.NotNil(t, got.Track)
	assert.Equal(t, 4, got.Track.StreamIndex)

	var texts []string
	seen := map[int]bool{}
	for _, b := range got.Track.Blocks {
		texts = append(texts, b.Text)
		assert.False(t, seen[b.Index], "Index %d used twice", b.Index)
		seen[b.Index] = true
	}
	assert.Equal(t, []string{"One", "Two", "We are leaving.", "We are not alone", "Four", "Five"}, texts)

	// Main cues keep their original numbering (P7); the merged cue is numbered
	// above the main track's highest Index.
	assert.Equal(t, []int{1, 2, 3, 6, 4, 5}, indexesOf(got.Track.Blocks))
	assert.Contains(t, logs.String(), "forced_merged=1")
	assert.Contains(t, logs.String(), "forced_duplicates_dropped=1")
}

func TestSelectAndRoute_MergedIndexesStartAboveTheHighestMainIndex(t *testing.T) {
	// SDH filtering leaves gaps: main cues 1, 3 and 4 survive (2 was "[DOOR
	// SLAMS]"). The merged cue must not take Index 4 just because three cues
	// survived — that Index is already "See you".
	prober := &fakeProber{info: &services.MediaTechInfo{SubtitleTracks: []services.SubtitleTrack{
		track(4, "eng", "English"),
		{StreamIndex: 6, Language: "eng", Format: "subrip", Forced: true},
	}}}
	ex := &fakeExtractor{contents: map[int]string{
		4: srtTimed(timedCue{1, 2, "Hello"}, timedCue{3, 4, "[DOOR SLAMS]"}, timedCue{5, 6, "Goodbye"}, timedCue{9, 10, "See you"}),
		6: srtTimed(timedCue{7, 8, "THREE YEARS LATER"}),
	}}
	r, tmp := newTestRouter(t, prober, ex)

	got, err := r.SelectAndRoute(context.Background(), "/media/m.mkv", tmp)

	require.NoError(t, err)
	assert.Equal(t, []int{1, 3, 5, 4}, indexesOf(got.Track.Blocks))
}

func TestSelectAndRoute_FullTrackFlaggedForcedIsNotMerged(t *testing.T) {
	// Many releases set forced=1 on the FULL track so players always show it.
	// Merging it would print every line twice and double the translation bill.
	prober := &fakeProber{info: &services.MediaTechInfo{SubtitleTracks: []services.SubtitleTrack{
		{StreamIndex: 3, Language: "eng", Format: "subrip", Forced: true},
		track(4, "eng", "English SDH"),
	}}}
	ex := &fakeExtractor{contents: map[int]string{
		3: srtTimed(nLines("full", 40, 0)...),
		4: srtTimed(nLines("sdh", 42, 0)...),
	}}
	r, tmp, logs := newLoggingRouter(t, prober, ex)

	got, err := r.SelectAndRoute(context.Background(), "/media/m.mkv", tmp)

	require.NoError(t, err)
	assert.Equal(t, 4, got.Track.StreamIndex)
	assert.Len(t, got.Track.Blocks, 42)
	assert.Contains(t, logs.String(), "not merged")
}

func TestSelectAndRoute_TwoIdenticalForcedTracksMergeOnce(t *testing.T) {
	prober := &fakeProber{info: &services.MediaTechInfo{SubtitleTracks: []services.SubtitleTrack{
		track(4, "eng", "English"),
		{StreamIndex: 6, Language: "eng", Format: "subrip", Forced: true},
		{StreamIndex: 7, Language: "eng", Format: "mov_text", Forced: true},
	}}}
	ex := &fakeExtractor{contents: map[int]string{
		4: srtTimed(nLines("full", 6, 0)...),
		6: srtTimed(timedCue{20, 21, "We are not alone"}),
		7: srtTimed(timedCue{20, 21, "We are not alone"}),
	}}
	r, tmp := newTestRouter(t, prober, ex)

	got, err := r.SelectAndRoute(context.Background(), "/media/m.mkv", tmp)

	require.NoError(t, err)
	assert.Len(t, got.Track.Blocks, 7)
	assert.Equal(t, 1, strings.Count(cueText(got.Track.Blocks), "We are not alone"))
}

func TestSelectAndRoute_TraditionalSDHStillBeatsSimplifiedRegular(t *testing.T) {
	// Regular-over-SDH is an English-tier rule; between Chinese tracks the
	// script decides first (2026-07-31 Apple TV+ ruling).
	prober := &fakeProber{info: &services.MediaTechInfo{SubtitleTracks: []services.SubtitleTrack{
		track(4, "chi", "简体中文"),
		{StreamIndex: 5, Language: "chi", Format: "subrip", Title: "繁體中文", HearingImpaired: true},
	}}}
	ex := &fakeExtractor{contents: map[int]string{
		4: srtOf("我们一直以来都被骗", "所有人都必须看看这个"),
		5: srtOf("我們一直以來都被騙", "所有人都必須看看這個"),
	}}
	r, tmp := newTestRouter(t, prober, ex)

	got, err := r.SelectAndRoute(context.Background(), "/media/m.mkv", tmp)

	require.NoError(t, err)
	assert.Equal(t, 5, got.Track.StreamIndex)
	assert.Equal(t, RouteDeliverDirect, got.Kind)
}

func TestSelectAndRoute_ChineseRouteNeverMergesForcedTrack(t *testing.T) {
	prober := &fakeProber{info: &services.MediaTechInfo{SubtitleTracks: []services.SubtitleTrack{
		track(4, "chi", "繁體中文"),
		{StreamIndex: 6, Language: "chi", Format: "subrip", Title: "繁體中文 (強制)", Forced: true},
	}}}
	ex := &fakeExtractor{contents: map[int]string{
		4: srtTimed(timedCue{1, 2, "答案都在箱子裡。"}, timedCue{3, 4, "我原本希望她能親口告訴你們。"}),
		6: srtTimed(timedCue{7, 8, "（三年後）"}),
	}}
	r, tmp := newTestRouter(t, prober, ex)

	got, err := r.SelectAndRoute(context.Background(), "/media/m.mkv", tmp)

	require.NoError(t, err)
	assert.Equal(t, RouteDeliverDirect, got.Kind)
	assert.Equal(t, 4, got.Track.StreamIndex)
	assert.Len(t, got.Track.Blocks, 2)
}

// ─── AC #8 — the See S01E02 shape ───────────────────────────────────────────

func TestSelectAndRoute_SeeS01E02Shape(t *testing.T) {
	// Stream 8 full (568 cues, no flags), 9 SDH (more cues: SDH splits lines),
	// 10 forced (one on-screen line). Expect: stream 8 + the forced line = 569.
	full := nLines("full", 568, 60)
	sdh := nLines("sdh", 600, 60)
	forced := []timedCue{{start: 1185, end: 1188, text: "We are not alone"}}
	prober := &fakeProber{info: &services.MediaTechInfo{
		DurationSeconds: 3417,
		SubtitleTracks: []services.SubtitleTrack{
			track(8, "eng", ""),
			track(9, "eng", "English [SDH]"),
			{StreamIndex: 10, Language: "eng", Format: "subrip", Forced: true},
		},
	}}
	// nLines spans 568 s from 60 s → ends at ~628 s, short of half of 3417 s.
	// Stretch both full tracks across the file so coverage is not what decides.
	full = append(full, timedCue{start: 3300, end: 3302, text: "full closing line"})
	sdh = append(sdh, timedCue{start: 3300, end: 3302, text: "sdh closing line"})
	ex := &fakeExtractor{contents: map[int]string{
		8:  srtTimed(full...),
		9:  srtTimed(sdh...),
		10: srtTimed(forced...),
	}}
	r, tmp := newTestRouter(t, prober, ex)

	got, err := r.SelectAndRoute(context.Background(), "/media/see.mkv", tmp)

	require.NoError(t, err)
	require.Equal(t, RouteTranslate, got.Kind)
	assert.Equal(t, 8, got.Track.StreamIndex)
	assert.Len(t, got.Track.Blocks, 569+1, "568 + closing line + the forced line")
	assert.Contains(t, cueText(got.Track.Blocks), "We are not alone")
}

func indexesOf(blocks []SubtitleBlock) []int {
	out := make([]int, len(blocks))
	for i, b := range blocks {
		out[i] = b.Index
	}
	return out
}
