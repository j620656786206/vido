package services

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
)

// disc-2026-10-episode-list-subtitle-badge-a AC #2 / #7 — the real shapes of
// 《末日光明》(See) on the NAS.

func tracksJSON(t *testing.T, tracks ...SubtitleTrack) models.NullString {
	t.Helper()
	if tracks == nil {
		tracks = []SubtitleTrack{}
	}
	raw, err := json.Marshal(tracks)
	require.NoError(t, err)
	return models.NewNullString(string(raw))
}

func seeEpisode(n int, file string) models.Episode {
	return models.Episode{ID: "ep", SeriesID: "see", SeasonNumber: 1, EpisodeNumber: n,
		FilePath: models.NewNullString(file), SubtitleStatus: models.SubtitleStatusNotSearched}
}

var (
	engEmbedded = SubtitleTrack{Language: "eng", Format: "subrip", StreamIndex: 8}
	engSDH      = SubtitleTrack{Language: "eng", Format: "subrip", StreamIndex: 9, Title: "SDH", HearingImpaired: true}
)

func sidecar(lang, name string) SubtitleTrack {
	return SubtitleTrack{Language: lang, Format: "srt", External: true, FileName: name}
}

func TestComputeEpisodeSubtitle_SeeShapes(t *testing.T) {
	// S01E01: only English embedded tracks, nothing beside it → 缺中文.
	e1 := seeEpisode(1, "/tv/See/S1/See.S01E01.mkv")
	e1.SubtitleTracks = tracksJSON(t, engEmbedded, engSDH)
	v := computeEpisodeSubtitle(e1, &SidecarTracks{Tracks: []SubtitleTrack{}})
	assert.Equal(t, models.ChineseSubtitleNone, v.verdict)
	assert.Empty(t, v.sources)
	assert.Empty(t, v.refresh, "stored and live agree — nothing to write")

	// S01E02: official zh-TW beside it + the subtitle Vido generated.
	e2 := seeEpisode(2, "/tv/See/S1/See.S01E02.mkv")
	e2.SubtitleTracks = tracksJSON(t, engEmbedded)
	e2.SubtitleStatus = models.SubtitleStatusFound
	e2.SubtitleLanguage = models.NewNullString("zh-Hant")
	e2.SubtitlePath = models.NewNullString("/tv/See/S1/See.S01E02.zh-Hant.srt")
	live := &SidecarTracks{Tracks: []SubtitleTrack{
		sidecar("zh-Hant", "See.S01E02.zh-Hant.srt"),
		sidecar("zh-Hant", "See.S01E02.zh-TW.srt"),
	}}
	v = computeEpisodeSubtitle(e2, live)
	assert.Equal(t, models.ChineseSubtitleZhHant, v.verdict)
	assert.Equal(t, []ChineseSubtitleSource{
		{Kind: ChineseSourceVido, Language: "zh-Hant", Label: "zh-Hant"},
		{Kind: ChineseSourceSidecar, Language: "zh-Hant", Label: "zh-TW"},
	}, v.sources)
	assert.NotEmpty(t, v.refresh, "the sweep stored no sidecars; the live read must be written back")

	// S01E03–E08: the official zh-TW file only.
	e3 := seeEpisode(3, "/tv/See/S1/See.S01E03.mkv")
	e3.SubtitleTracks = tracksJSON(t, engEmbedded, sidecar("zh-Hant", "See.S01E03.zh-TW.srt"))
	v = computeEpisodeSubtitle(e3, &SidecarTracks{Tracks: []SubtitleTrack{sidecar("zh-Hant", "See.S01E03.zh-TW.srt")}})
	assert.Equal(t, models.ChineseSubtitleZhHant, v.verdict)
	assert.Equal(t, []ChineseSubtitleSource{{Kind: ChineseSourceSidecar, Language: "zh-Hant", Label: "zh-TW"}}, v.sources)
	assert.Empty(t, v.refresh)

	// S02: Chinese inside the file. Titled → Traditional; untitled `chi` →
	// Chinese with the script untold (PR #674's 分不出繁簡).
	s2 := seeEpisode(1, "/tv/See/S2/See.S02E01.mkv")
	s2.SubtitleTracks = tracksJSON(t, SubtitleTrack{Language: "chi", Format: "subrip", StreamIndex: 3, Title: "繁體中文"}, engEmbedded)
	v = computeEpisodeSubtitle(s2, &SidecarTracks{Tracks: []SubtitleTrack{}})
	assert.Equal(t, models.ChineseSubtitleZhHant, v.verdict)
	assert.Equal(t, []ChineseSubtitleSource{{Kind: ChineseSourceEmbedded, Language: "zh-Hant", Label: "繁體中文"}}, v.sources)

	s2.SubtitleTracks = tracksJSON(t, SubtitleTrack{Language: "chi", Format: "subrip", StreamIndex: 3})
	v = computeEpisodeSubtitle(s2, &SidecarTracks{Tracks: []SubtitleTrack{}})
	assert.Equal(t, models.ChineseSubtitleZh, v.verdict)
	assert.Equal(t, []ChineseSubtitleSource{{Kind: ChineseSourceEmbedded, Language: "zh-unknown", Label: ""}}, v.sources)
}

func TestComputeEpisodeSubtitle_UnprobedFileCannotBeNone(t *testing.T) {
	ep := seeEpisode(1, "/tv/x/X.S01E01.mkv") // subtitle_tracks NULL: never probed

	v := computeEpisodeSubtitle(ep, &SidecarTracks{Tracks: []SubtitleTrack{sidecar("en", "X.S01E01.en.srt")}})
	assert.Equal(t, models.ChineseSubtitleUnknown, v.verdict, "an English .srt says nothing about the tracks inside an unread file")
	assert.Empty(t, v.refresh, "nothing is stored before the embedded half is known")

	v = computeEpisodeSubtitle(ep, &SidecarTracks{Tracks: []SubtitleTrack{sidecar("zh-Hant", "X.S01E01.zh.srt")}})
	assert.Equal(t, models.ChineseSubtitleZhHant, v.verdict, "a Chinese sidecar proves it without the probe")

	v = computeEpisodeSubtitle(ep, nil)
	assert.Equal(t, models.ChineseSubtitleUnknown, v.verdict)
}

func TestComputeEpisodeSubtitle_NameSaysTraditionalTextSaysSimplified(t *testing.T) {
	ep := seeEpisode(5, "/tv/x/X.S01E05.mkv")
	ep.SubtitleTracks = tracksJSON(t, engEmbedded)
	v := computeEpisodeSubtitle(ep, &SidecarTracks{Tracks: []SubtitleTrack{sidecar("zh-Hans", "X.S01E05.zh-TW.srt")}})
	assert.Equal(t, models.ChineseSubtitleZhHans, v.verdict)
	assert.Equal(t, []ChineseSubtitleSource{{Kind: ChineseSourceSidecar, Language: "zh-Hans", Label: "zh-TW"}}, v.sources)
}

func TestComputeEpisodeSubtitle_SidecarReadFailureUsesStored(t *testing.T) {
	ep := seeEpisode(3, "/tv/x/X.S01E03.mkv")
	ep.SubtitleTracks = tracksJSON(t, engEmbedded, sidecar("zh-Hant", "X.S01E03.zh-TW.srt"))
	v := computeEpisodeSubtitle(ep, &SidecarTracks{Err: errors.New("EIO")})
	assert.Equal(t, models.ChineseSubtitleZhHant, v.verdict)
	assert.Empty(t, v.refresh, "a failed read must not erase the stored sidecars")
}

func TestComputeEpisodeSubtitle_VidoRecordWithoutFileStillCounts(t *testing.T) {
	ep := seeEpisode(4, "/tv/x/X.S01E04.mkv")
	ep.SubtitleTracks = tracksJSON(t)
	ep.SubtitleStatus = models.SubtitleStatusFound
	ep.SubtitleLanguage = models.NewNullString("zh-Hant")
	v := computeEpisodeSubtitle(ep, &SidecarTracks{Tracks: []SubtitleTrack{}})
	assert.Equal(t, models.ChineseSubtitleZhHant, v.verdict)
	assert.Equal(t, []ChineseSubtitleSource{{Kind: ChineseSourceVido, Language: "zh-Hant"}}, v.sources)
}

func TestComputeEpisodeSubtitle_BrokenStoredJSONIsUnread(t *testing.T) {
	ep := seeEpisode(1, "/tv/x/X.S01E01.mkv")
	ep.SubtitleTracks = models.NewNullString("{not json")
	v := computeEpisodeSubtitle(ep, &SidecarTracks{Tracks: []SubtitleTrack{sidecar("en", "X.S01E01.en.srt")}})
	assert.Equal(t, models.ChineseSubtitleUnknown, v.verdict)
	assert.Empty(t, v.refresh, "never overwrite a column we could not read")
	assert.Nil(t, storedSidecars(ep))
}

// PR #674's rule: an untagged sidecar whose text is not Chinese is `und`, so
// the episode stays "unknown" rather than "missing Chinese" — the reader
// cannot tell an English file from, say, a Japanese one by its name.
func TestComputeEpisodeSubtitle_UntaggedNonChineseSidecarIsNotProofOfNone(t *testing.T) {
	ep := seeEpisode(1, "/tv/x/X.S01E01.mkv")
	ep.SubtitleTracks = tracksJSON(t)
	v := computeEpisodeSubtitle(ep, &SidecarTracks{Tracks: []SubtitleTrack{sidecar("und", "X.S01E01.srt")}})
	assert.Equal(t, models.ChineseSubtitleUnknown, v.verdict)
}
