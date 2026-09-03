package subtitle

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/services"
)

// ─── Fakes ─────────────────────────────────────────────────────────────────

type fakeProber struct {
	info      *services.MediaTechInfo
	err       error
	gotPath   string
	callCount int
}

func (f *fakeProber) Probe(_ context.Context, filePath string) (*services.MediaTechInfo, error) {
	f.callCount++
	f.gotPath = filePath
	return f.info, f.err
}

// fakeExtractor writes the configured SRT content to the real temp paths the
// router will read back. An index absent from `contents` simulates a candidate
// whose output never materialised.
type fakeExtractor struct {
	contents  map[int]string
	err       error
	gotIdx    []int
	callCount int
}

func (f *fakeExtractor) Extract(_ context.Context, _, tmpDir string, streamIndexes []int) (map[int]string, error) {
	f.callCount++
	f.gotIdx = append([]int(nil), streamIndexes...)
	if f.err != nil {
		return nil, f.err
	}

	out := make(map[int]string, len(streamIndexes))
	for _, idx := range streamIndexes {
		content, ok := f.contents[idx]
		if !ok {
			continue
		}
		path := trackOutputPath(tmpDir, idx)
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			return nil, err
		}
		out[idx] = path
	}
	return out, nil
}

// srtOf builds a valid SRT document, one cue per line of text.
func srtOf(texts ...string) string {
	var sb strings.Builder
	for i, text := range texts {
		sb.WriteString(fmt.Sprintf("%d\n00:00:%02d,000 --> 00:00:%02d,000\n%s\n\n", i+1, i, i+1, text))
	}
	return sb.String()
}

func newTestRouter(t *testing.T, prober *fakeProber, ex *fakeExtractor) (*Router, string) {
	t.Helper()
	return NewRouter(prober, ex, nil), t.TempDir()
}

func embedded(idx int, lang, codec string) services.SubtitleTrack {
	return services.SubtitleTrack{StreamIndex: idx, Language: lang, Format: codec, External: false}
}

// ─── AC #5 — the nine-row verdict matrix ───────────────────────────────────

func TestSelectAndRoute_NoSubtitleStreamsAtAll(t *testing.T) {
	prober := &fakeProber{info: &services.MediaTechInfo{VideoCodec: "H.264"}}
	ex := &fakeExtractor{}
	r, tmp := newTestRouter(t, prober, ex)

	got, err := r.SelectAndRoute(context.Background(), "/media/m.mkv", tmp)

	require.NoError(t, err)
	assert.Equal(t, RouteNoTextSource, got.Kind)
	assert.Nil(t, got.Track)
	assert.NotEmpty(t, got.Reason)
	assert.Zero(t, ex.callCount, "nothing to extract — ffmpeg must not be invoked")
}

func TestSelectAndRoute_OnlyImageCodecTracks(t *testing.T) {
	prober := &fakeProber{info: &services.MediaTechInfo{SubtitleTracks: []services.SubtitleTrack{
		embedded(2, "eng", "hdmv_pgs_subtitle"),
		embedded(3, "chi", "dvd_subtitle"),
	}}}
	ex := &fakeExtractor{}
	r, tmp := newTestRouter(t, prober, ex)

	got, err := r.SelectAndRoute(context.Background(), "/media/m.mkv", tmp)

	require.NoError(t, err)
	assert.Equal(t, RouteNoTextSource, got.Kind, "FR5 — image-only is not a usable text source")
	assert.Nil(t, got.Track)
	assert.Zero(t, ex.callCount)
}

func TestSelectAndRoute_TextTracksButNoneUsable(t *testing.T) {
	// P0 — `und` is NEVER English, and a non-Chinese foreign tag is not a
	// candidate either. (zh/chi moved OUT of this list on 2026-07-31: Chinese
	// tracks are now the PREFERRED candidate — see the Chinese-first tests.)
	for _, lang := range []string{"und", "jpn", "kor", "fre"} {
		t.Run(lang, func(t *testing.T) {
			prober := &fakeProber{info: &services.MediaTechInfo{SubtitleTracks: []services.SubtitleTrack{
				embedded(2, lang, "subrip"),
			}}}
			ex := &fakeExtractor{}
			r, tmp := newTestRouter(t, prober, ex)

			got, err := r.SelectAndRoute(context.Background(), "/media/m.mkv", tmp)

			require.NoError(t, err)
			assert.Equal(t, RouteSkip, got.Kind)
			assert.Nil(t, got.Track)
			assert.Zero(t, ex.callCount)
		})
	}
}

// ─── Chinese-first routing (2026-07-31) ────────────────────────────────────

func TestSelectAndRoute_ChineseTrackWinsOverEnglish(t *testing.T) {
	// The 135/157 case measured on the owner's NAS: an official zh track and an
	// eng track in the same file. The zh track must be used — and the eng track
	// must not even be extracted, so no LLM translation is ever queued.
	prober := &fakeProber{info: &services.MediaTechInfo{SubtitleTracks: []services.SubtitleTrack{
		embedded(2, "eng", "subrip"),
		embedded(7, "chi", "subrip"),
	}}}
	ex := &fakeExtractor{contents: map[int]string{
		2: srtOf("We're being lied to.", "Everyone has to see this."),
		7: srtOf("我們一直以來都被騙", "所有人都必須看看這個"),
	}}
	r, tmp := newTestRouter(t, prober, ex)

	got, err := r.SelectAndRoute(context.Background(), "/media/m.mkv", tmp)

	require.NoError(t, err)
	assert.Equal(t, RouteDeliverDirect, got.Kind, "an official zh-Hant track ships as-is — no LLM")
	assert.Equal(t, LangTraditional, got.DetectedVariant)
	require.NotNil(t, got.Track)
	assert.Equal(t, 7, got.Track.StreamIndex)
	assert.Equal(t, []int{7}, ex.gotIdx, "the eng track must not be extracted at all")
}

func TestSelectAndRoute_SimplifiedOnlyChineseTrackIsConverted(t *testing.T) {
	// Owner's rule: a Simplified track is still the right source — extract it
	// and convert, rather than paying to translate the English one.
	prober := &fakeProber{info: &services.MediaTechInfo{SubtitleTracks: []services.SubtitleTrack{
		embedded(2, "eng", "subrip"),
		embedded(6, "chi", "subrip"),
	}}}
	ex := &fakeExtractor{contents: map[int]string{
		2: srtOf("We're being lied to."),
		6: srtOf("我们被骗了", "所有人都必须看看这个"),
	}}
	r, tmp := newTestRouter(t, prober, ex)

	got, err := r.SelectAndRoute(context.Background(), "/media/m.mkv", tmp)

	require.NoError(t, err)
	assert.Equal(t, RouteConvertThenDeliver, got.Kind)
	assert.Equal(t, LangSimplified, got.DetectedVariant)
	require.NotNil(t, got.Track)
	assert.Equal(t, 6, got.Track.StreamIndex)
}

func TestSelectAndRoute_TraditionalWinsTheTieAgainstSimplified(t *testing.T) {
	// Live shape (Apple TV+ / Silo S01E10): Simplified, Traditional and
	// Cantonese-Traditional tracks with IDENTICAL cue counts. The old
	// cue-count→stream-index pair always took the LOWEST index (Simplified) and
	// paid a needless s2twp round-trip. Traditional must win the tie.
	prober := &fakeProber{info: &services.MediaTechInfo{SubtitleTracks: []services.SubtitleTrack{
		embedded(6, "chi", "subrip"),  // Chinese (Simplified)
		embedded(7, "chi", "subrip"),  // Chinese (Traditional)
		embedded(43, "chi", "subrip"), // Cantonese (Traditional)
	}}}
	ex := &fakeExtractor{contents: map[int]string{
		6:  srtOf("我们被骗了", "所有人都必须看看这个"),
		7:  srtOf("我們一直以來都被騙", "所有人都必須看看這個"),
		43: srtOf("我們被騙了", "所有人都必須睇睇呢個"),
	}}
	r, tmp := newTestRouter(t, prober, ex)

	got, err := r.SelectAndRoute(context.Background(), "/media/m.mkv", tmp)

	require.NoError(t, err)
	assert.Equal(t, RouteDeliverDirect, got.Kind)
	require.NotNil(t, got.Track)
	assert.Equal(t, 7, got.Track.StreamIndex,
		"stream 7 is Traditional; 6 is Simplified and 43 is Cantonese — lowest index must NOT win")
}

func TestSelectAndRoute_CueCountStillBeatsVariant(t *testing.T) {
	// Guard on the tie-break's ordering: a forced-narrative track can be
	// genuinely Traditional yet carry a handful of cues. It must never beat a
	// full Simplified track that we can simply convert.
	prober := &fakeProber{info: &services.MediaTechInfo{SubtitleTracks: []services.SubtitleTrack{
		embedded(6, "chi", "subrip"), // full Simplified
		embedded(7, "chi", "subrip"), // forced-narrative Traditional
	}}}
	ex := &fakeExtractor{contents: map[int]string{
		6: srtOf("我们被骗了", "所有人都必须看看这个", "哪些人", "所有人"),
		7: srtOf("羊毛戰記"),
	}}
	r, tmp := newTestRouter(t, prober, ex)

	got, err := r.SelectAndRoute(context.Background(), "/media/m.mkv", tmp)

	require.NoError(t, err)
	require.NotNil(t, got.Track)
	assert.Equal(t, 6, got.Track.StreamIndex, "cue count is the primary key")
	assert.Equal(t, RouteConvertThenDeliver, got.Kind)
}

func TestSelectAndRoute_ChineseTaggedTrackThatIsActuallyEnglish(t *testing.T) {
	// The mislabel case in the other direction: tagged chi, content has no CJK.
	// Content routing (FR6) still decides — it queues for translation.
	prober := &fakeProber{info: &services.MediaTechInfo{SubtitleTracks: []services.SubtitleTrack{
		embedded(6, "chi", "subrip"),
	}}}
	ex := &fakeExtractor{contents: map[int]string{6: srtOf("Hello there", "General Kenobi")}}
	r, tmp := newTestRouter(t, prober, ex)

	got, err := r.SelectAndRoute(context.Background(), "/media/m.mkv", tmp)

	require.NoError(t, err)
	assert.Equal(t, RouteTranslate, got.Kind)
	assert.Equal(t, LangUndetermined, got.DetectedVariant)
}

func TestSelectAndRoute_EngTaggedContentIsTraditional(t *testing.T) {
	// FR7 — the Bazarr mislabel case: tagged eng, content is actually zh-Hant.
	prober := &fakeProber{info: &services.MediaTechInfo{SubtitleTracks: []services.SubtitleTrack{
		embedded(2, "eng", "subrip"),
	}}}
	ex := &fakeExtractor{contents: map[int]string{2: srtOf("這個時間過來", "還沒開始說話")}}
	r, tmp := newTestRouter(t, prober, ex)

	got, err := r.SelectAndRoute(context.Background(), "/media/m.mkv", tmp)

	require.NoError(t, err)
	assert.Equal(t, RouteDeliverDirect, got.Kind)
	assert.Equal(t, LangTraditional, got.DetectedVariant)
	require.NotNil(t, got.Track)
	assert.Equal(t, 2, got.Track.StreamIndex)
	assert.Equal(t, "eng", got.Track.Language)
	assert.Equal(t, "subrip", got.Track.Codec)
	assert.Len(t, got.Track.Blocks, 2)
}

func TestSelectAndRoute_EngTaggedContentIsSimplified(t *testing.T) {
	// FR8 — content is zh-Hans; the CALLER runs OpenCC, this story never does.
	prober := &fakeProber{info: &services.MediaTechInfo{SubtitleTracks: []services.SubtitleTrack{
		embedded(2, "eng", "ass"),
	}}}
	ex := &fakeExtractor{contents: map[int]string{2: srtOf("这个时间过来", "还没开始说话")}}
	r, tmp := newTestRouter(t, prober, ex)

	got, err := r.SelectAndRoute(context.Background(), "/media/m.mkv", tmp)

	require.NoError(t, err)
	assert.Equal(t, RouteConvertThenDeliver, got.Kind)
	assert.Equal(t, LangSimplified, got.DetectedVariant)
	require.NotNil(t, got.Track)
	assert.Equal(t, "ass", got.Track.Codec)
}

func TestSelectAndRoute_EngTaggedContentIsAmbiguousChinese(t *testing.T) {
	// s2twp is idempotent on already-Traditional text, so converting is the safe
	// branch for the 30–70% band (§9b co-production precedent).
	prober := &fakeProber{info: &services.MediaTechInfo{SubtitleTracks: []services.SubtitleTrack{
		embedded(2, "eng", "subrip"),
	}}}
	ex := &fakeExtractor{contents: map[int]string{2: srtOf("這个")}}
	r, tmp := newTestRouter(t, prober, ex)

	got, err := r.SelectAndRoute(context.Background(), "/media/m.mkv", tmp)

	require.NoError(t, err)
	assert.Equal(t, LangAmbiguous, got.DetectedVariant)
	assert.Equal(t, RouteConvertThenDeliver, got.Kind)
}

func TestSelectAndRoute_EngTaggedContentIsRealEnglish(t *testing.T) {
	// FR10 — no CJK at all means the tag told the truth: queue for translation.
	prober := &fakeProber{info: &services.MediaTechInfo{SubtitleTracks: []services.SubtitleTrack{
		embedded(4, "en", "mov_text"),
	}}}
	ex := &fakeExtractor{contents: map[int]string{4: srtOf("Hello there", "General Kenobi")}}
	r, tmp := newTestRouter(t, prober, ex)

	got, err := r.SelectAndRoute(context.Background(), "/media/m.mkv", tmp)

	require.NoError(t, err)
	assert.Equal(t, RouteTranslate, got.Kind)
	assert.Equal(t, LangUndetermined, got.DetectedVariant)
	require.NotNil(t, got.Track)
	assert.Equal(t, 4, got.Track.StreamIndex)
	assert.Equal(t, "en", got.Track.Language)
	assert.Equal(t, trackOutputPath(tmp, 4), got.Track.Path)
}

func TestSelectAndRoute_ZeroCuesSurviveSDHFilter(t *testing.T) {
	// FR5's word is *usable* — a pure-SDH track is not a usable text source.
	prober := &fakeProber{info: &services.MediaTechInfo{SubtitleTracks: []services.SubtitleTrack{
		embedded(2, "eng", "subrip"),
	}}}
	ex := &fakeExtractor{contents: map[int]string{2: srtOf("[door slams]", "♪ dramatic music ♪", "(sighs)")}}
	r, tmp := newTestRouter(t, prober, ex)

	got, err := r.SelectAndRoute(context.Background(), "/media/m.mkv", tmp)

	require.NoError(t, err)
	assert.Equal(t, RouteNoTextSource, got.Kind)
	assert.Nil(t, got.Track)
}

func TestSelectAndRoute_FirstCandidateUnparseableFallsBack(t *testing.T) {
	prober := &fakeProber{info: &services.MediaTechInfo{SubtitleTracks: []services.SubtitleTrack{
		embedded(2, "eng", "subrip"),
		embedded(5, "eng", "subrip"),
	}}}
	ex := &fakeExtractor{contents: map[int]string{
		2: "not an srt document at all",
		5: srtOf("Hello there", "General Kenobi"),
	}}
	r, tmp := newTestRouter(t, prober, ex)

	got, err := r.SelectAndRoute(context.Background(), "/media/m.mkv", tmp)

	require.NoError(t, err)
	assert.Equal(t, RouteTranslate, got.Kind)
	require.NotNil(t, got.Track)
	assert.Equal(t, 5, got.Track.StreamIndex)
	assert.Equal(t, 1, ex.callCount, "FR3 — both candidates extracted in ONE pass")
	assert.Equal(t, []int{2, 5}, ex.gotIdx)
}

func TestSelectAndRoute_AllCandidatesUnparseable(t *testing.T) {
	prober := &fakeProber{info: &services.MediaTechInfo{SubtitleTracks: []services.SubtitleTrack{
		embedded(2, "eng", "subrip"),
		embedded(5, "eng", "ass"),
	}}}
	ex := &fakeExtractor{contents: map[int]string{2: "garbage", 5: "   "}}
	r, tmp := newTestRouter(t, prober, ex)

	_, err := r.SelectAndRoute(context.Background(), "/media/m.mkv", tmp)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSubtitleExtractFailed)
}

// ─── AC #5 — multi-candidate selection heuristic ───────────────────────────

func TestSelectAndRoute_PicksHighestPostFilterCueCount(t *testing.T) {
	// A forced-narrative track has few cues; the full track wins even when it
	// sits at a higher stream index.
	prober := &fakeProber{info: &services.MediaTechInfo{SubtitleTracks: []services.SubtitleTrack{
		embedded(2, "eng", "subrip"),
		embedded(7, "eng", "subrip"),
	}}}
	ex := &fakeExtractor{contents: map[int]string{
		2: srtOf("Forced line"),
		7: srtOf("One", "Two", "Three", "Four"),
	}}
	r, tmp := newTestRouter(t, prober, ex)

	got, err := r.SelectAndRoute(context.Background(), "/media/m.mkv", tmp)

	require.NoError(t, err)
	require.NotNil(t, got.Track)
	assert.Equal(t, 7, got.Track.StreamIndex)
	assert.Len(t, got.Track.Blocks, 4)
}

func TestSelectAndRoute_CueCountIsCountedAfterSDHFilter(t *testing.T) {
	// Track 2 has more RAW cues but they are nearly all SDH annotations; track 7
	// wins on surviving dialogue. Counting before the filter would pick wrong.
	prober := &fakeProber{info: &services.MediaTechInfo{SubtitleTracks: []services.SubtitleTrack{
		embedded(2, "eng", "subrip"),
		embedded(7, "eng", "subrip"),
	}}}
	ex := &fakeExtractor{contents: map[int]string{
		2: srtOf("[door slams]", "(sighs)", "♪ theme ♪", "[gunshot]", "Only real line"),
		7: srtOf("One", "Two", "Three"),
	}}
	r, tmp := newTestRouter(t, prober, ex)

	got, err := r.SelectAndRoute(context.Background(), "/media/m.mkv", tmp)

	require.NoError(t, err)
	require.NotNil(t, got.Track)
	assert.Equal(t, 7, got.Track.StreamIndex)
	assert.Len(t, got.Track.Blocks, 3)
}

// sub-6-4: bare-♪ cues no longer count as dialogue, so an SDH track padded
// with music-only cues must NOT out-rank a leaner clean track on raw length.
// Before the fix track 2 (6 raw cues) beat track 7 (3 real lines).
func TestSelectAndRoute_MusicOnlyCuesDoNotInflateCandidateRank(t *testing.T) {
	prober := &fakeProber{info: &services.MediaTechInfo{SubtitleTracks: []services.SubtitleTrack{
		embedded(2, "eng", "subrip"),
		embedded(7, "eng", "subrip"),
	}}}
	ex := &fakeExtractor{contents: map[int]string{
		2: srtOf("♪", "♪♪", "♫", "Only real line", "♪", "#"),
		7: srtOf("One", "Two", "Three"),
	}}
	r, tmp := newTestRouter(t, prober, ex)

	got, err := r.SelectAndRoute(context.Background(), "/media/m.mkv", tmp)

	require.NoError(t, err)
	require.NotNil(t, got.Track)
	assert.Equal(t, 7, got.Track.StreamIndex)
	assert.Len(t, got.Track.Blocks, 3)
}

func TestSelectAndRoute_TieBreaksOnLowestStreamIndex(t *testing.T) {
	prober := &fakeProber{info: &services.MediaTechInfo{SubtitleTracks: []services.SubtitleTrack{
		embedded(7, "eng", "subrip"),
		embedded(3, "eng", "subrip"),
	}}}
	ex := &fakeExtractor{contents: map[int]string{
		7: srtOf("One", "Two"),
		3: srtOf("Alpha", "Beta"),
	}}
	r, tmp := newTestRouter(t, prober, ex)

	got, err := r.SelectAndRoute(context.Background(), "/media/m.mkv", tmp)

	require.NoError(t, err)
	require.NotNil(t, got.Track)
	assert.Equal(t, 3, got.Track.StreamIndex, "deterministic tie-break is lowest StreamIndex")
}

func TestSelectAndRoute_CandidateWithNoExtractedOutputIsSkipped(t *testing.T) {
	prober := &fakeProber{info: &services.MediaTechInfo{SubtitleTracks: []services.SubtitleTrack{
		embedded(2, "eng", "subrip"),
		embedded(5, "eng", "subrip"),
	}}}
	ex := &fakeExtractor{contents: map[int]string{5: srtOf("Hello there")}} // 2 produced nothing
	r, tmp := newTestRouter(t, prober, ex)

	got, err := r.SelectAndRoute(context.Background(), "/media/m.mkv", tmp)

	require.NoError(t, err)
	require.NotNil(t, got.Track)
	assert.Equal(t, 5, got.Track.StreamIndex)
}

// ─── Blocks are the SDH-filtered cues with original numbering (P7) ─────────

func TestSelectAndRoute_BlocksAreFilteredAndKeepOriginalNumbering(t *testing.T) {
	prober := &fakeProber{info: &services.MediaTechInfo{SubtitleTracks: []services.SubtitleTrack{
		embedded(2, "eng", "subrip"),
	}}}
	ex := &fakeExtractor{contents: map[int]string{
		2: srtOf("[door slams]", "JOHN: Hello there", "(sighs)", "General Kenobi"),
	}}
	r, tmp := newTestRouter(t, prober, ex)

	got, err := r.SelectAndRoute(context.Background(), "/media/m.mkv", tmp)

	require.NoError(t, err)
	require.NotNil(t, got.Track)
	require.Len(t, got.Track.Blocks, 2)

	assert.Equal(t, 2, got.Track.Blocks[0].Index, "P7 — survivors keep original numbering")
	assert.Equal(t, "Hello there", got.Track.Blocks[0].Text, "SDH filtering happens HERE, before translation (P6)")
	assert.Equal(t, 4, got.Track.Blocks[1].Index)
	assert.Equal(t, "General Kenobi", got.Track.Blocks[1].Text)
}

// ─── Probe failures + fresh-probe contract (AC #2) ─────────────────────────

func TestSelectAndRoute_ProbesFreshAtRunTime(t *testing.T) {
	prober := &fakeProber{info: &services.MediaTechInfo{SubtitleTracks: []services.SubtitleTrack{
		embedded(2, "eng", "subrip"),
	}}}
	ex := &fakeExtractor{contents: map[int]string{2: srtOf("Hello there")}}
	r, tmp := newTestRouter(t, prober, ex)

	_, err := r.SelectAndRoute(context.Background(), "/media/m.mkv", tmp)

	require.NoError(t, err)
	assert.Equal(t, 1, prober.callCount, "files move/change after scan — the run-time probe is the truth")
	assert.Equal(t, "/media/m.mkv", prober.gotPath)
}

func TestSelectAndRoute_ProbeErrorPropagates(t *testing.T) {
	probeErr := errors.New("ffprobe exec: exit status 1")
	prober := &fakeProber{err: probeErr}
	ex := &fakeExtractor{}
	r, tmp := newTestRouter(t, prober, ex)

	_, err := r.SelectAndRoute(context.Background(), "/media/m.mkv", tmp)

	require.Error(t, err)
	assert.ErrorIs(t, err, probeErr, "Rule 13 — wrapped with %w, never swallowed")
	assert.Zero(t, ex.callCount)
}

func TestSelectAndRoute_ProbeReturnsNilInfo(t *testing.T) {
	prober := &fakeProber{info: nil}
	ex := &fakeExtractor{}
	r, tmp := newTestRouter(t, prober, ex)

	got, err := r.SelectAndRoute(context.Background(), "/media/m.mkv", tmp)

	require.NoError(t, err)
	assert.Equal(t, RouteNoTextSource, got.Kind)
}

func TestSelectAndRoute_ExtractErrorPropagates(t *testing.T) {
	prober := &fakeProber{info: &services.MediaTechInfo{SubtitleTracks: []services.SubtitleTrack{
		embedded(2, "eng", "subrip"),
	}}}
	ex := &fakeExtractor{err: fmt.Errorf("%w: ffmpeg exec: exit status 1", ErrSubtitleExtractFailed)}
	r, tmp := newTestRouter(t, prober, ex)

	_, err := r.SelectAndRoute(context.Background(), "/media/m.mkv", tmp)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSubtitleExtractFailed)
}

// ─── Scope fence (AC #8) ───────────────────────────────────────────────────

func TestSelectAndRoute_ExternalSidecarsAreIgnored(t *testing.T) {
	// M1's pipeline is embedded-only; sidecars remain the search engine's domain.
	prober := &fakeProber{info: &services.MediaTechInfo{SubtitleTracks: []services.SubtitleTrack{
		{Language: "eng", Format: "srt", External: true},
	}}}
	ex := &fakeExtractor{}
	r, tmp := newTestRouter(t, prober, ex)

	got, err := r.SelectAndRoute(context.Background(), "/media/m.mkv", tmp)

	require.NoError(t, err)
	assert.Equal(t, RouteNoTextSource, got.Kind)
	assert.Zero(t, ex.callCount)
}

func TestRouteForVariant_UnknownVariantFailsClosed(t *testing.T) {
	// CR sub-1-4 L5 — Detect only returns the four Lang* constants today, but if
	// it ever grows a new value, an unknown variant must NOT silently reach the
	// LLM (P0: fail closed rather than mistranslate).
	kind, reason := routeForVariant("zh-Hant-HK-new-value", 2, "eng")
	assert.Equal(t, RouteSkip, kind)
	assert.Contains(t, reason, "failing closed")

	// The known four still route exactly as the AC #5 matrix rules.
	kind, _ = routeForVariant(LangUndetermined, 2, "eng")
	assert.Equal(t, RouteTranslate, kind)
	kind, _ = routeForVariant(LangTraditional, 2, "eng")
	assert.Equal(t, RouteDeliverDirect, kind)
	kind, _ = routeForVariant(LangSimplified, 2, "eng")
	assert.Equal(t, RouteConvertThenDeliver, kind)
	kind, _ = routeForVariant(LangAmbiguous, 2, "eng")
	assert.Equal(t, RouteConvertThenDeliver, kind)
}

func TestRouteKindWireValues(t *testing.T) {
	// [@contract-v1] — consumed by sub-1-5a / sub-1-5b / sub-1-6. Changing a
	// literal here is a Rule 20 bump, not a rename.
	assert.Equal(t, RouteKind("deliver_direct"), RouteDeliverDirect)
	assert.Equal(t, RouteKind("convert_then_deliver"), RouteConvertThenDeliver)
	assert.Equal(t, RouteKind("translate"), RouteTranslate)
	assert.Equal(t, RouteKind("skip"), RouteSkip)
	assert.Equal(t, RouteKind("no_text_source"), RouteNoTextSource)
}

func TestExtractorSatisfiesTrackExtractor(t *testing.T) {
	// The production Extractor must remain drop-in for the router's narrow port.
	var _ TrackExtractor = NewExtractor(0, nil)
	var _ TechProber = services.NewFFprobeService(1, 0, nil)
}
