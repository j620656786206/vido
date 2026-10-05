package subtitle

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/services"
)

const (
	invTraditional = "1\n00:00:01,000 --> 00:00:03,000\n艾肯尼，這是我們的家園，我們團結一致\n"
	invSimplified  = "1\n00:00:01,000 --> 00:00:03,000\n这是我们的家园，我们团结一致，一起战斗\n"
	invEnglish     = "1\n00:00:01,000 --> 00:00:03,000\nThis is our home\n"
)

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
}

func TestListSidecars_LanguageFromContentNotFilename(t *testing.T) {
	dir := t.TempDir()
	media := filepath.Join(dir, "See.S01E02.mkv")
	writeFile(t, dir, "See.S01E02.mkv", "video")
	// Tagged zh-TW but the text is Simplified — content wins.
	writeFile(t, dir, "See.S01E02.zh-TW.srt", invSimplified)
	writeFile(t, dir, "See.S01E02.chs.ass", invTraditional)
	writeFile(t, dir, "See.S01E02.en.srt", invEnglish)
	writeFile(t, dir, "See.S01E02.srt", invEnglish)
	// Noise that must be skipped.
	writeFile(t, dir, "See.S01E02.zh-Hant.srt.bak", invTraditional)
	writeFile(t, dir, "See.S01E02.zh.tmp.srt", invTraditional)
	writeFile(t, dir, "See.S01E03.zh-TW.srt", invTraditional)
	writeFile(t, dir, "See.S01E02.nfo", "<xml/>")

	files, err := ListSidecars(media, "")
	require.NoError(t, err)

	got := map[string]string{}
	for _, f := range files {
		got[f.FileName] = f.Language
	}
	assert.Equal(t, map[string]string{
		"See.S01E02.zh-TW.srt": LangSimplified,
		"See.S01E02.chs.ass":   LangTraditional,
		"See.S01E02.en.srt":    "en",
		"See.S01E02.srt":       LangUndetermined,
	}, got)
}

func TestListSidecars_MixedChineseIsUnknownVariant(t *testing.T) {
	dir := t.TempDir()
	media := filepath.Join(dir, "M.mkv")
	writeFile(t, dir, "M.zh.srt", invTraditional+"\n2\n00:00:04,000 --> 00:00:05,000\n"+"这是我们的家园，我们团结一致，一起战斗\n")

	files, err := ListSidecars(media, "")
	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.Equal(t, LangChineseUnknown, files[0].Language)
}

func TestListSidecars_ChineseTagWithoutChineseTextIsUndetermined(t *testing.T) {
	dir := t.TempDir()
	media := filepath.Join(dir, "M.mkv")
	writeFile(t, dir, "M.zh-TW.srt", invEnglish)

	files, err := ListSidecars(media, "")
	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.Equal(t, LangUndetermined, files[0].Language, "a Chinese filename tag on English text is not trusted")
}

func TestListSidecars_MarksVidosOwnSubtitle(t *testing.T) {
	dir := t.TempDir()
	media := filepath.Join(dir, "M.mkv")
	writeFile(t, dir, "M.zh-Hant.srt", invTraditional)
	writeFile(t, dir, "M.zh-TW.srt", invTraditional)

	files, err := ListSidecars(media, filepath.Join(dir, ".", "M.zh-Hant.srt"))
	require.NoError(t, err)
	own := map[string]bool{}
	for _, f := range files {
		own[f.FileName] = f.IsVidoOutput
	}
	assert.Equal(t, map[string]bool{"M.zh-Hant.srt": true, "M.zh-TW.srt": false}, own)
}

func TestListSidecars_UnreadableDirectory(t *testing.T) {
	_, err := ListSidecars(filepath.Join(t.TempDir(), "missing", "M.mkv"), "")
	assert.Error(t, err)
}

func TestEmbeddedLanguage(t *testing.T) {
	cases := []struct{ lang, title, want string }{
		{"chi", "繁體中文", LangTraditional},
		{"chi", "Chinese (Traditional)", LangTraditional},
		{"zho", "zh-TW", LangTraditional},
		{"chi", "Chinese (Hong Kong)", LangTraditional},
		{"chi", "简体", LangSimplified},
		{"chi", "Chinese (Simplified)", LangSimplified},
		{"zh-cn", "", LangSimplified},
		{"zh-tw", "", LangTraditional},
		{"chi", "粵語", "yue"},
		{"chi", "Cantonese", "yue"},
		{"yue", "", "yue"},
		{"chi", "", LangChineseUnknown},
		{"eng", "English [SDH]", "en"},
		{"en", "", "en"},
		{"jpn", "日本語", "jpn"},
		{"", "", LangUndetermined},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, embeddedLanguage(tc.lang, tc.title), "%q / %q", tc.lang, tc.title)
	}
}

type inventoryProber struct {
	available bool
	info      *services.MediaTechInfo
	err       error
	calls     int
}

func (f *inventoryProber) IsAvailable() bool { return f.available }
func (f *inventoryProber) Probe(context.Context, string) (*services.MediaTechInfo, error) {
	f.calls++
	return f.info, f.err
}

func TestBuildInventory(t *testing.T) {
	dir := t.TempDir()
	media := filepath.Join(dir, "M.mkv")
	writeFile(t, dir, "M.zh-TW.srt", invTraditional)

	prober := &inventoryProber{available: true, info: &services.MediaTechInfo{SubtitleTracks: []services.SubtitleTrack{
		{Language: "eng", Format: "subrip", StreamIndex: 8, Title: "English"},
		{Language: "chi", Format: "subrip", StreamIndex: 6, Title: "繁體"},
		{Language: "chi", Format: "hdmv_pgs_subtitle", StreamIndex: 7},
	}}}
	inv := BuildInventory(context.Background(), media, "", prober)

	assert.Equal(t, InventoryOK, inv.Sidecars.Status)
	require.Len(t, inv.Sidecars.Files, 1)
	assert.Equal(t, InventoryOK, inv.Embedded.Status)
	require.Len(t, inv.Embedded.Tracks, 3)
	assert.Equal(t, EmbeddedTrack{StreamIndex: 8, Language: "en", Title: "English", Format: "subrip", Text: true}, inv.Embedded.Tracks[0])
	assert.Equal(t, LangTraditional, inv.Embedded.Tracks[1].Language)
	assert.False(t, inv.Embedded.Tracks[2].Text, "a PGS track is an image track")
}

func TestBuildInventory_ProbeFailuresStayLocal(t *testing.T) {
	dir := t.TempDir()
	media := filepath.Join(dir, "M.mkv")
	writeFile(t, dir, "M.en.srt", invEnglish)

	unavailable := &inventoryProber{available: false}
	inv := BuildInventory(context.Background(), media, "", unavailable)
	assert.Equal(t, InventoryUnavailable, inv.Embedded.Status)
	assert.Equal(t, 0, unavailable.calls)
	assert.Len(t, inv.Sidecars.Files, 1, "the cheap half still answers")

	failing := &inventoryProber{available: true, err: errors.New("timeout")}
	inv = BuildInventory(context.Background(), media, "", failing)
	assert.Equal(t, InventoryFailed, inv.Embedded.Status)
	assert.Equal(t, InventoryOK, inv.Sidecars.Status)

	inv = BuildInventory(context.Background(), filepath.Join(dir, "gone", "M.mkv"), "", nil)
	assert.Equal(t, InventoryFailed, inv.Sidecars.Status)
	assert.Equal(t, InventoryUnavailable, inv.Embedded.Status)
	assert.NotNil(t, inv.Sidecars.Files, "empty lists serialise as [] not null")
	assert.NotNil(t, inv.Embedded.Tracks)
}
