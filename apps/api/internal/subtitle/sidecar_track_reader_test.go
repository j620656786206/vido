package subtitle

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/services"
)

var _ services.SidecarTrackReader = SidecarTrackReader{}

// disc-2026-10-episode-list-subtitle-badge-a AC #3 / #7.
func TestSidecarTrackReader_SeasonFolder(t *testing.T) {
	dir := t.TempDir()
	e1 := filepath.Join(dir, "See.S01E01.mkv")
	e2 := filepath.Join(dir, "See.S01E02.mkv")
	e3 := filepath.Join(dir, "See.S01E03.mkv")
	for _, n := range []string{"See.S01E01.mkv", "See.S01E02.mkv", "See.S01E03.mkv"} {
		writeFile(t, dir, n, "video")
	}
	writeFile(t, dir, "See.S01E02.zh-TW.srt", invTraditional)
	writeFile(t, dir, "See.S01E02.zh-Hant.srt", invTraditional) // Vido's own
	writeFile(t, dir, "See.S01E03.zh-TW.srt", invSimplified)    // name lies, content wins
	writeFile(t, dir, "See.S01E03.en.srt", invEnglish)

	missing := filepath.Join(dir, "gone", "x.mkv")
	got := SidecarTrackReader{}.ReadSidecarTracks([]string{e1, e2, e3, missing}, nil)

	require.NoError(t, got[e1].Err)
	assert.Empty(t, got[e1].Tracks)

	require.NoError(t, got[e2].Err)
	require.Len(t, got[e2].Tracks, 2)
	for i, name := range []string{"See.S01E02.zh-Hant.srt", "See.S01E02.zh-TW.srt"} {
		tr := got[e2].Tracks[i]
		assert.Equal(t, name, tr.FileName)
		assert.Equal(t, "zh-Hant", tr.Language)
		assert.True(t, tr.External)
		assert.NotEmpty(t, tr.FileSig)
	}

	require.NoError(t, got[e3].Err)
	langs := map[string]string{}
	for _, tr := range got[e3].Tracks {
		langs[tr.FileName] = tr.Language
		assert.Empty(t, tr.Title, "the file name must never reach Title — ChineseTitleScript would read it")
	}
	assert.Equal(t, map[string]string{"See.S01E03.zh-TW.srt": "zh-Hans", "See.S01E03.en.srt": "en"}, langs)

	assert.Error(t, got[missing].Err, "an unreadable folder is reported per file, not dropped")
}

// The language strings the reader emits must classify the way the verdict
// expects: a mixed-script file is still Chinese (script untold).
func TestSidecarTrackReader_LanguagesClassify(t *testing.T) {
	assert.Equal(t, models.ChineseSubtitleZhHant, models.ChineseSubtitleOfTrack(LangTraditional, "", ""))
	assert.Equal(t, models.ChineseSubtitleZhHans, models.ChineseSubtitleOfTrack(LangSimplified, "", ""))
	assert.Equal(t, models.ChineseSubtitleZh, models.ChineseSubtitleOfTrack(LangChineseUnknown, "", ""))
	assert.Equal(t, models.ChineseSubtitle(""), models.ChineseSubtitleOfTrack(LangUndetermined, "", ""))
	assert.Equal(t, models.ChineseSubtitle(""), models.ChineseSubtitleOfTrack("en", "", ""))
}

func TestSidecarTrackReader_EmptyInput(t *testing.T) {
	assert.Empty(t, SidecarTrackReader{}.ReadSidecarTracks(nil, nil))
}

// CR M1: on a NAS mount, reopening every sidecar on every season open is the
// slow part. An unchanged file (same name and size:mtime) keeps its stored
// language; a changed one is read again.
func TestSidecarTrackReader_ReusesStoredLanguageOfUnchangedFiles(t *testing.T) {
	dir := t.TempDir()
	media := filepath.Join(dir, "X.S01E01.mkv")
	writeFile(t, dir, "X.S01E01.mkv", "video")
	writeFile(t, dir, "X.S01E01.zh-TW.srt", invTraditional)

	first := SidecarTrackReader{}.ReadSidecarTracks([]string{media}, nil)[media]
	require.Len(t, first.Tracks, 1)
	require.NotEmpty(t, first.Tracks[0].FileSig)

	// Pretend the stored answer was Simplified: if the reader reuses it, the
	// file was not reopened.
	stored := first.Tracks[0]
	stored.Language = LangSimplified
	again := SidecarTrackReader{}.ReadSidecarTracks([]string{media}, map[string][]services.SubtitleTrack{media: {stored}})[media]
	assert.Equal(t, LangSimplified, again.Tracks[0].Language)

	// Same name, different signature → read again.
	stored.FileSig = "0:0"
	fresh := SidecarTrackReader{}.ReadSidecarTracks([]string{media}, map[string][]services.SubtitleTrack{media: {stored}})[media]
	assert.Equal(t, LangTraditional, fresh.Tracks[0].Language)
}
