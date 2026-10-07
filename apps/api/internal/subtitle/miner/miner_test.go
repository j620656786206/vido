package miner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/services"
	"github.com/vido/api/internal/subtitle/mine"
)

// ─── fakes ──────────────────────────────────────────────────────────────────

type fakeEpisodes struct{ bySeries map[string][]models.Episode }

func (f fakeEpisodes) FindBySeriesID(_ context.Context, id string) ([]models.Episode, error) {
	return f.bySeries[id], nil
}

type fakeSeries struct{ rows []models.Series }

func (f fakeSeries) FindByID(_ context.Context, id string) (*models.Series, error) {
	for i := range f.rows {
		if f.rows[i].ID == id {
			return &f.rows[i], nil
		}
	}
	return nil, errors.New("not found")
}
func (f fakeSeries) List(_ context.Context, p repository.ListParams) ([]models.Series, *repository.PaginationResult, error) {
	return f.rows, &repository.PaginationResult{Page: 1, TotalPages: 1}, nil
}

type fakeScopes struct{}

func (fakeScopes) Resolve(_ context.Context, id string) (string, error) { return "tmdb:tv:" + id, nil }

type fakeGlossary struct {
	existing []models.GlossaryTerm
	inserted []models.GlossaryTerm
	upserted []models.GlossaryTerm
	cleared  []string // scopes whose season drawers were cleared
	deleted  []string // ids pruned
}

func (g *fakeGlossary) DeleteSeasonDrawers(_ context.Context, scope string) (int64, error) {
	g.cleared = append(g.cleared, scope)
	return 0, nil
}
func (g *fakeGlossary) Delete(_ context.Context, id string) error {
	g.deleted = append(g.deleted, id)
	return nil
}

func (g *fakeGlossary) ListByScope(context.Context, string) ([]models.GlossaryTerm, error) {
	return g.existing, nil
}

// ReplaceUnconfirmedGuess mirrors the repository's SQL guard: insert when
// absent; overwrite only an unconfirmed harvest / earlier-mining row whose
// rendering differs; leave everything else alone.
func (g *fakeGlossary) ReplaceUnconfirmedGuess(_ context.Context, t *models.GlossaryTerm) (bool, error) {
	for i, e := range g.existing {
		if !strings.EqualFold(e.TermSrc, t.TermSrc) || (e.Scope != "" && e.Scope != t.Scope) {
			continue
		}
		guess := !e.Confirmed && (e.Source == models.GlossarySourceSubtitle || e.Source == models.GlossarySourceOfficialSubtitle)
		if guess && e.TermZh != t.TermZh {
			g.existing[i].TermZh, g.existing[i].Source = t.TermZh, t.Source
			g.upserted = append(g.upserted, *t)
			return true, nil
		}
		return false, nil
	}
	g.inserted = append(g.inserted, *t)
	return true, nil
}

func srt(lines ...[3]string) string {
	out := ""
	for i, l := range lines {
		out += fmt.Sprintf("%d\n%s --> %s\n%s\n\n", i+1, l[0], l[1], l[2])
	}
	return out
}

// writeShow lays out a sidecar-only show: n episodes with official zh and en
// sidecars, plus the given extra files.
func writeShow(t *testing.T, dir string, n int, extra map[string]string) []models.Episode {
	t.Helper()
	var eps []models.Episode
	ctx := [][2]string{
		{"Hey, Walter, we need you.", "嘿，華特，我們需要你"},
		{"Is Walter coming or not?", "華特到底來不來"},
		{"Ask Walter, he knows.", "去問華特，他知道"},
		{"Walter will fix it.", "華特會修好"},
	}
	for ep := 1; ep <= n; ep++ {
		base := fmt.Sprintf("Show.S01E%02d", ep)
		media := filepath.Join(dir, base+".mkv")
		require.NoError(t, os.WriteFile(media, nil, 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, base+".en.srt"), []byte(srt(
			[3]string{"00:00:01,000", "00:00:03,000", ctx[(ep-1)%len(ctx)][0]},
			[3]string{"00:00:04,000", "00:00:06,000", "Tell Toby to hurry."},
		)), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, base+".zh-TW.srt"), []byte(srt(
			[3]string{"00:00:01,100", "00:00:03,100", ctx[(ep-1)%len(ctx)][1]},
			[3]string{"00:00:04,000", "00:00:06,200", fmt.Sprintf("叫托比快一點（%d）", ep)},
		)), 0o644))
		eps = append(eps, models.Episode{ID: fmt.Sprintf("e%d", ep), SeriesID: "s1", SeasonNumber: 1, EpisodeNumber: ep,
			FilePath: models.NewNullString(media)})
	}
	for name, body := range extra {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644))
	}
	return eps
}

func newMiner(eps fakeEpisodes, series fakeSeries, g *fakeGlossary) *OfficialSubtitleMiner {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	m := NewOfficialSubtitleMiner(eps, series, fakeScopes{}, g, nil, nil, logger)
	t0 := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	m.now = func() time.Time { t0 = t0.Add(time.Second); return t0 }
	return m
}

// ─── AC #1: one show end to end ─────────────────────────────────────────────

func TestMineSeries_LearnsWritesAndReports(t *testing.T) {
	dir := t.TempDir()
	eps := writeShow(t, dir, 4, map[string]string{
		// A fifth episode with NO zh sidecar (the one that needs translating).
		"Show.S01E05.mkv":    "",
		"Show.S01E05.en.srt": srt([3]string{"00:00:01,000", "00:00:03,000", "Walter again."}),
		// Vido's own delivery beside E01 must not count as a zh source.
		"Show.S01E01.zh-Hant.srt": srt([3]string{"00:00:01,000", "00:00:03,000", "機器翻的"}),
	})
	eps = append(eps, models.Episode{ID: "e5", SeriesID: "s1", SeasonNumber: 1, EpisodeNumber: 5,
		FilePath: models.NewNullString(filepath.Join(dir, "Show.S01E05.mkv"))})
	g := &fakeGlossary{existing: []models.GlossaryTerm{
		// Seeded by TMDb earlier: verified, never re-inserted.
		{TermSrc: "Toby", TermZh: "托比", Source: models.GlossarySourceMetadata},
	}}
	m := newMiner(fakeEpisodes{bySeries: map[string][]models.Episode{"s1": eps}},
		fakeSeries{rows: []models.Series{{ID: "s1", Title: "Show"}}}, g)

	res, err := m.MineSeries(context.Background(), "s1")
	require.NoError(t, err)
	assert.Equal(t, "tmdb:tv:s1", res.Scope)
	assert.Equal(t, 5, res.EpisodesTotal)
	assert.Equal(t, 4, res.EpisodesUsed)
	assert.Equal(t, "no official zh subtitle", res.Episodes[4].Skipped)
	assert.Equal(t, "Show.S01E01.zh-TW.srt", res.Episodes[0].ZhSource, "the .zh-Hant.srt delivery is never the source")

	byName := map[string]string{}
	for _, tm := range res.Terms {
		byName[tm.Src] = tm.How
	}
	assert.Equal(t, "cooccurrence", byName["Walter"])
	assert.Equal(t, "known", byName["Toby"])

	require.Len(t, g.inserted, 1, "only the NEW rendering is written; the known one is verified, not duplicated")
	row := g.inserted[0]
	assert.Equal(t, "Walter", row.TermSrc)
	assert.Equal(t, "華特", row.TermZh)
	assert.Equal(t, models.GlossarySourceOfficialSubtitle, row.Source)
	assert.Equal(t, "tmdb:tv:s1", row.Scope)
	assert.Equal(t, "s1", row.MediaID)
	assert.False(t, row.Confirmed, "the user still approves it in the panel")
	assert.Equal(t, 1, res.TermsInserted)

	st := m.Status()
	assert.False(t, st.Running)
	require.Len(t, st.Results, 1)
	assert.Equal(t, "Show", st.Results[0].Title)
	assert.NotNil(t, st.LastRunAt)
}

// ─── sub-7-5a finding: fan-group files are skipped ──────────────────────────

func TestMineSeries_SkipsFanGroupFiles(t *testing.T) {
	dir := t.TempDir()
	eps := writeShow(t, dir, 3, nil)
	// Overwrite E02's zh sidecar with a fansub header.
	fansub := "1\n00:00:00,000 --> 00:00:03,000\n本字幕由人人影視字幕組翻譯\n\n" + srt(
		[3]string{"00:00:01,100", "00:00:03,100", "華特到底來不來"},
	)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Show.S01E02.zh-TW.srt"), []byte(fansub), 0o644))
	g := &fakeGlossary{}
	m := newMiner(fakeEpisodes{bySeries: map[string][]models.Episode{"s1": eps}},
		fakeSeries{rows: []models.Series{{ID: "s1", Title: "Show"}}}, g)

	res, err := m.MineSeries(context.Background(), "s1")
	require.NoError(t, err)
	assert.Equal(t, 1, res.FansubSkipped)
	assert.Equal(t, 2, res.EpisodesUsed)
	assert.Equal(t, "only fan-group zh subtitles", res.Episodes[1].Skipped)
}

// ─── AC #2: the post-scan sweep only touches partial shows ──────────────────

func TestMinePartial_OnlyPartialShows(t *testing.T) {
	root := t.TempDir()
	partialDir := filepath.Join(root, "partial")
	fullDir := filepath.Join(root, "full")
	require.NoError(t, os.MkdirAll(partialDir, 0o755))
	require.NoError(t, os.MkdirAll(fullDir, 0o755))
	partial := writeShow(t, partialDir, 3, map[string]string{"Show.S01E04.mkv": ""})
	partial = append(partial, models.Episode{ID: "p4", SeriesID: "p", SeasonNumber: 1, EpisodeNumber: 4,
		FilePath: models.NewNullString(filepath.Join(partialDir, "Show.S01E04.mkv"))})
	full := writeShow(t, fullDir, 3, nil) // every episode has zh: nothing to learn FOR
	for i := range full {
		full[i].SeriesID = "f"
	}
	g := &fakeGlossary{}
	m := newMiner(fakeEpisodes{bySeries: map[string][]models.Episode{"p": partial, "f": full}},
		fakeSeries{rows: []models.Series{{ID: "f", Title: "Full"}, {ID: "p", Title: "Partial"}}}, g)

	results, err := m.MinePartial(context.Background())
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "p", results[0].SeriesID)
	assert.Equal(t, 2, results[0].TermsInserted, "Walter and Toby, learned from three episodes")
	assert.Len(t, m.Status().Results, 1)
}

func TestMiner_BusyGuardAndErrors(t *testing.T) {
	g := &fakeGlossary{}
	m := newMiner(fakeEpisodes{bySeries: map[string][]models.Episode{}},
		fakeSeries{rows: []models.Series{{ID: "s1", Title: "Show"}}}, g)
	require.True(t, m.begin("s1"))
	_, err := m.MineSeries(context.Background(), "s1")
	assert.ErrorIs(t, err, ErrMinerBusy)
	_, err = m.MinePartial(context.Background())
	assert.ErrorIs(t, err, ErrMinerBusy)
	assert.True(t, m.Status().Running)
	assert.Equal(t, "s1", m.Status().RunningFor)
	m.finish(nil)

	res, err := m.MineSeries(context.Background(), "nope")
	require.Error(t, err)
	assert.Contains(t, res.Error, "not found")
	assert.False(t, m.Status().Running)
}

func TestStartPartial_ClaimsBeforeReturning(t *testing.T) {
	g := &fakeGlossary{}
	m := newMiner(fakeEpisodes{bySeries: map[string][]models.Episode{}}, fakeSeries{}, g)
	release := make(chan struct{})
	m.series = blockingSeries{release: release}
	require.NoError(t, m.StartPartial(context.Background()))
	assert.True(t, m.Status().Running, "running is visible the moment StartPartial returns")
	assert.Equal(t, "partial", m.Status().RunningFor)
	assert.ErrorIs(t, m.StartPartial(context.Background()), ErrMinerBusy)
	close(release)
	assert.Eventually(t, func() bool { return !m.Status().Running }, time.Second, 5*time.Millisecond)
	assert.NotNil(t, m.Status().LastRunAt)
}

type blockingSeries struct{ release chan struct{} }

func (b blockingSeries) FindByID(context.Context, string) (*models.Series, error) {
	return nil, errors.New("none")
}
func (b blockingSeries) List(context.Context, repository.ListParams) ([]models.Series, *repository.PaginationResult, error) {
	<-b.release
	return nil, &repository.PaginationResult{TotalPages: 1}, nil
}

// disc-2026-10-asr-harvest-pollutes-glossary: a rendering our own
// speech-recognition run harvested is a guess. It must not count as "known"
// (that silenced the miner on See's Jerlamarel), and the official file's
// rendering overwrites it; TMDb / manual / confirmed rows are untouched.
func TestKnownRenderings_HarvestIsNotEvidence(t *testing.T) {
	rows := []models.GlossaryTerm{
		{TermSrc: "Jerlamarel", TermZh: "傑拉瑪瑞爾", Source: models.GlossarySourceSubtitle},
		{TermSrc: "Chola Morel", TermZh: "丘拉·莫瑞爾", Source: models.GlossarySourceSubtitle},
		{TermSrc: "Paris", TermZh: "芭麗絲", Source: models.GlossarySourceSubtitle, Confirmed: true}, // the user approved it
		{TermSrc: "Baba Voss", TermZh: "巴巴佛斯", Source: models.GlossarySourceMetadata},
		{TermSrc: "Maghra", TermZh: "瑪格拉", Source: models.GlossarySourceManual},
		{TermSrc: "Kofun", TermZh: "柯方", Source: models.GlossarySourceOfficialSubtitle}, // our earlier guess, unconfirmed
	}
	known := knownRenderings(rows)
	assert.Equal(t, map[string]string{"Paris": "芭麗絲", "Baba Voss": "巴巴佛斯", "Maghra": "瑪格拉"}, known)
	guesses := harvestedGuesses(rows)
	assert.Equal(t, map[string]struct{}{"jerlamarel": {}, "chola morel": {}}, guesses)
}

func TestMineSeries_OfficialRenderingReplacesHarvestedGuess(t *testing.T) {
	dir := t.TempDir()
	// Four episodes where "Jerlamarel" is rendered 謝拉馬威 every time.
	var eps []models.Episode
	for i := 1; i <= 4; i++ {
		base := filepath.Join(dir, fmt.Sprintf("Show.S01E0%d", i))
		require.NoError(t, os.WriteFile(base+".mkv", []byte("x"), 0o644))
		require.NoError(t, os.WriteFile(base+".en.srt", []byte(srt(
			[3]string{"00:00:01,000", "00:00:03,000", "Jerlamarel told me."},
			[3]string{"00:00:05,000", "00:00:07,000", "Jerlamarel is waiting."},
			[3]string{"00:00:09,000", "00:00:11,000", "Find Jerlamarel."},
		)), 0o644))
		require.NoError(t, os.WriteFile(base+".zh-TW.srt", []byte(srt(
			[3]string{"00:00:01,000", "00:00:03,000", "謝拉馬威告訴我"},
			[3]string{"00:00:05,000", "00:00:07,000", "謝拉馬威在等"},
			[3]string{"00:00:09,000", "00:00:11,000", "去找謝拉馬威"},
		)), 0o644))
		eps = append(eps, models.Episode{ID: fmt.Sprintf("e%d", i), SeriesID: "s1", FilePath: models.NewNullString(base + ".mkv")})
	}
	g := &fakeGlossary{existing: []models.GlossaryTerm{
		{TermSrc: "Jerlamarel", TermZh: "傑拉瑪瑞爾", Source: models.GlossarySourceSubtitle}, // the garble
	}}
	m := NewOfficialSubtitleMiner(fakeEpisodes{bySeries: map[string][]models.Episode{"s1": eps}},
		fakeSeries{rows: []models.Series{{ID: "s1", Title: "See"}}}, fakeScopes{}, g, nil, nil, nil)

	res, err := m.MineSeries(context.Background(), "s1")
	require.NoError(t, err)
	require.Len(t, g.upserted, 1, "the official rendering replaces the harvested guess")
	assert.Equal(t, "Jerlamarel", g.upserted[0].TermSrc)
	assert.Equal(t, "謝拉馬威", g.upserted[0].TermZh)
	assert.Equal(t, models.GlossarySourceOfficialSubtitle, g.upserted[0].Source)
	assert.Empty(t, g.inserted)
	assert.Equal(t, 1, res.TermsReplaced)
	assert.Equal(t, 0, res.TermsInserted)
}

func TestMineSeries_ConfirmedGuessIsNotReplaced(t *testing.T) {
	dir := t.TempDir()
	var eps []models.Episode
	for i := 1; i <= 4; i++ {
		base := filepath.Join(dir, fmt.Sprintf("Show.S01E0%d", i))
		require.NoError(t, os.WriteFile(base+".mkv", []byte("x"), 0o644))
		require.NoError(t, os.WriteFile(base+".en.srt", []byte(srt(
			[3]string{"00:00:01,000", "00:00:03,000", "Jerlamarel told me."},
			[3]string{"00:00:05,000", "00:00:07,000", "Jerlamarel is waiting."},
			[3]string{"00:00:09,000", "00:00:11,000", "Find Jerlamarel."},
		)), 0o644))
		require.NoError(t, os.WriteFile(base+".zh-TW.srt", []byte(srt(
			[3]string{"00:00:01,000", "00:00:03,000", "謝拉馬威告訴我"},
			[3]string{"00:00:05,000", "00:00:07,000", "謝拉馬威在等"},
			[3]string{"00:00:09,000", "00:00:11,000", "去找謝拉馬威"},
		)), 0o644))
		eps = append(eps, models.Episode{ID: fmt.Sprintf("e%d", i), SeriesID: "s1", FilePath: models.NewNullString(base + ".mkv")})
	}
	g := &fakeGlossary{existing: []models.GlossaryTerm{
		{TermSrc: "Jerlamarel", TermZh: "傑拉瑪瑞爾", Source: models.GlossarySourceSubtitle, Confirmed: true}, // the user approved it
	}}
	m := NewOfficialSubtitleMiner(fakeEpisodes{bySeries: map[string][]models.Episode{"s1": eps}},
		fakeSeries{rows: []models.Series{{ID: "s1", Title: "See"}}}, fakeScopes{}, g, nil, nil, nil)

	res, err := m.MineSeries(context.Background(), "s1")
	require.NoError(t, err)
	assert.Empty(t, g.upserted, "a confirmed row is the user's word, whatever its source")
	assert.Empty(t, g.inserted)
	assert.Equal(t, 0, res.TermsReplaced)
	assert.Equal(t, "傑拉瑪瑞爾", g.existing[0].TermZh)
}

// --- disc-2026-10-mine-en-source-selection fakes ---

type fakeProber struct{ tracks []services.SubtitleTrack }

func (p fakeProber) Probe(context.Context, string) (*services.MediaTechInfo, error) {
	return &services.MediaTechInfo{SubtitleTracks: p.tracks}, nil
}

// fakeExtractor "extracts" by writing the SRT text registered per stream.
type fakeExtractor struct {
	byStream map[int]string
	asked    []int
}

func (e *fakeExtractor) Extract(_ context.Context, _ string, tmp string, streams []int) (map[int]string, error) {
	out := map[int]string{}
	for _, s := range streams {
		e.asked = append(e.asked, s)
		p := filepath.Join(tmp, fmt.Sprintf("s%d.srt", s))
		if err := os.WriteFile(p, []byte(e.byStream[s]), 0o644); err != nil {
			return nil, err
		}
		out[s] = p
	}
	return out, nil
}

func fourLines(zh bool, name string) string {
	if zh {
		return srt([3]string{"00:00:01,000", "00:00:03,000", name + "告訴我"}, [3]string{"00:00:05,000", "00:00:07,000", "去找" + name},
			[3]string{"00:00:09,000", "00:00:11,000", name + "在等"}, [3]string{"00:00:13,000", "00:00:15,000", "是" + name + "嗎"})
	}
	return srt([3]string{"00:00:01,000", "00:00:03,000", "Jerlamarel told me."}, [3]string{"00:00:05,000", "00:00:07,000", "Go find Jerlamarel."},
		[3]string{"00:00:09,000", "00:00:11,000", "Jerlamarel is waiting."}, [3]string{"00:00:13,000", "00:00:15,000", "Is that Jerlamarel?"})
}

// Vido's own ASR transcript sits beside the file as `.en.srt` with the name
// garbled; the release's English track is the source the Chinese was made
// from. The miner must read the embedded track, not the sidecar.
func TestMineEpisode_EmbeddedEnglishBeatsASRSidecar(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "Show.S01E01")
	require.NoError(t, os.WriteFile(base+".mkv", []byte("x"), 0o644))
	require.NoError(t, os.WriteFile(base+".zh-TW.srt", []byte(fourLines(true, "謝拉馬威")), 0o644))
	garbled := strings.ReplaceAll(fourLines(false, ""), "Jerlamarel", "Trilla Morel")
	// pad the sidecar past minUsableCues so size alone does not decide
	for i := 0; i < 20; i++ {
		garbled += fmt.Sprintf("%d\n00:01:%02d,000 --> 00:01:%02d,500\nfiller\n\n", 10+i, i, i)
	}
	require.NoError(t, os.WriteFile(base+".en.srt", []byte(garbled), 0o644))
	full := fourLines(false, "")
	for i := 0; i < 20; i++ { // a real dialogue track is long; the forced one is not
		full += fmt.Sprintf("%d\n00:02:%02d,000 --> 00:02:%02d,500\nmore dialogue\n\n", 10+i, i, i)
	}
	ext := &fakeExtractor{byStream: map[int]string{2: srt([3]string{"00:00:01,000", "00:00:02,000", "FORCED"}), 3: full}}
	prober := fakeProber{tracks: []services.SubtitleTrack{
		{Language: "eng", Format: "subrip", StreamIndex: 2, Forced: true},
		{Language: "eng", Format: "subrip", StreamIndex: 3},
	}}
	m := NewOfficialSubtitleMiner(nil, nil, fakeScopes{}, &fakeGlossary{}, prober, ext, nil)

	rep := MineEpisodeReport{}
	segs, _, _ := m.mineEpisode(context.Background(), base+".mkv", t.TempDir(), &rep)

	assert.Equal(t, "embedded stream 3", rep.EnSource, "the full embedded track, not the forced one, not the ASR sidecar")
	assert.Equal(t, []int{3}, ext.asked, "the forced track is never even extracted; one ffmpeg pass")
	assert.Equal(t, 4, len(segs))
	terms := mine.Mine(segs, mine.Options{})
	require.Len(t, terms, 1)
	assert.Equal(t, "謝拉馬威", terms[0].Zh)
}

// With no embedded English track the sidecar is still used.
func TestMineEpisode_SidecarWhenNoEmbeddedEnglish(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "Show.S01E01")
	require.NoError(t, os.WriteFile(base+".mkv", []byte("x"), 0o644))
	require.NoError(t, os.WriteFile(base+".zh-TW.srt", []byte(fourLines(true, "謝拉馬威")), 0o644))
	en := fourLines(false, "")
	for i := 0; i < 20; i++ {
		en += fmt.Sprintf("%d\n00:01:%02d,000 --> 00:01:%02d,500\nfiller\n\n", 10+i, i, i)
	}
	require.NoError(t, os.WriteFile(base+".en.srt", []byte(en), 0o644))
	m := NewOfficialSubtitleMiner(nil, nil, fakeScopes{}, &fakeGlossary{}, fakeProber{}, &fakeExtractor{}, nil)
	rep := MineEpisodeReport{}
	segs, _, _ := m.mineEpisode(context.Background(), base+".mkv", t.TempDir(), &rep)
	assert.Equal(t, "Show.S01E01.en.srt", rep.EnSource)
	assert.Equal(t, 4, len(segs))
}

type failingProber struct{}

func (failingProber) Probe(context.Context, string) (*services.MediaTechInfo, error) {
	return nil, errors.New("ffprobe: timeout")
}

// CR 2 / CR 6: a short embedded track falls back to the sidecar; a failed
// probe falls back to the sidecar AND says so in the report; no usable source
// names every reason.
func TestMineEpisode_EnglishFallbacksAreExplained(t *testing.T) {
	mk := func(t *testing.T, withSidecar bool) string {
		dir := t.TempDir()
		base := filepath.Join(dir, "Show.S01E01")
		require.NoError(t, os.WriteFile(base+".mkv", []byte("x"), 0o644))
		require.NoError(t, os.WriteFile(base+".zh-TW.srt", []byte(fourLines(true, "謝拉馬威")), 0o644))
		if withSidecar {
			require.NoError(t, os.WriteFile(base+".en.srt", []byte(fourLines(false, "")), 0o644))
		}
		return base + ".mkv"
	}
	t.Run("short embedded track → sidecar", func(t *testing.T) {
		media := mk(t, true)
		ext := &fakeExtractor{byStream: map[int]string{3: fourLines(false, "")}} // 4 cues < minUsableCues
		m := NewOfficialSubtitleMiner(nil, nil, fakeScopes{}, &fakeGlossary{}, fakeProber{tracks: []services.SubtitleTrack{{Language: "eng", Format: "subrip", StreamIndex: 3}}}, ext, nil)
		rep := MineEpisodeReport{}
		segs, _, _ := m.mineEpisode(context.Background(), media, t.TempDir(), &rep)
		assert.Equal(t, "Show.S01E01.en.srt", rep.EnSource)
		assert.Equal(t, 4, len(segs))
	})
	t.Run("probe failed → sidecar, flagged", func(t *testing.T) {
		media := mk(t, true)
		m := NewOfficialSubtitleMiner(nil, nil, fakeScopes{}, &fakeGlossary{}, failingProber{}, &fakeExtractor{}, nil)
		rep := MineEpisodeReport{}
		_, _, _ = m.mineEpisode(context.Background(), media, t.TempDir(), &rep)
		assert.Equal(t, "Show.S01E01.en.srt (probe failed; embedded tracks unknown)", rep.EnSource)
	})
	t.Run("only short tracks, no sidecar → skipped with every reason", func(t *testing.T) {
		media := mk(t, false)
		ext := &fakeExtractor{byStream: map[int]string{3: fourLines(false, ""), 4: srt([3]string{"00:00:01,000", "00:00:02,000", "x"})}}
		m := NewOfficialSubtitleMiner(nil, nil, fakeScopes{}, &fakeGlossary{}, fakeProber{tracks: []services.SubtitleTrack{
			{Language: "eng", Format: "subrip", StreamIndex: 3}, {Language: "eng", Format: "subrip", StreamIndex: 4, HearingImpaired: true}}}, ext, nil)
		rep := MineEpisodeReport{}
		_, _, _ = m.mineEpisode(context.Background(), media, t.TempDir(), &rep)
		assert.Equal(t, "en: stream 3 has only 4 cues; stream 4 has only 1 cues", rep.Skipped)
		assert.Equal(t, []int{3, 4}, ext.asked, "both candidates in one extraction call")
	})
}

// disc-2026-10-glossary-season-scope-a: the two seasons' official files spell
// Jerlamarel differently; Haniwa the same. The show-wide row follows the
// season with more EPISODES; each season gets its own drawer; the consistent
// name gets no drawer.
func TestMineSeries_SeasonDrawers(t *testing.T) {
	dir := t.TempDir()
	mk := func(season, ep int, jerl string) models.Episode {
		base := filepath.Join(dir, fmt.Sprintf("Show.S%02dE%02d", season, ep))
		require.NoError(t, os.WriteFile(base+".mkv", []byte("x"), 0o644))
		require.NoError(t, os.WriteFile(base+".en.srt", []byte(srt(
			[3]string{"00:00:01,000", "00:00:03,000", "Jerlamarel told me."},
			[3]string{"00:00:05,000", "00:00:07,000", "Go find Jerlamarel."},
			[3]string{"00:00:09,000", "00:00:11,000", "Jerlamarel is waiting."},
			[3]string{"00:00:13,000", "00:00:15,000", "Haniwa, come."},
			[3]string{"00:00:17,000", "00:00:19,000", "Where is Haniwa?"},
			[3]string{"00:00:21,000", "00:00:23,000", "Haniwa knows."},
		)), 0o644))
		require.NoError(t, os.WriteFile(base+".zh-TW.srt", []byte(srt(
			[3]string{"00:00:01,000", "00:00:03,000", jerl + "告訴我"},
			[3]string{"00:00:05,000", "00:00:07,000", "去找" + jerl},
			[3]string{"00:00:09,000", "00:00:11,000", jerl + "在等"},
			[3]string{"00:00:13,000", "00:00:15,000", "哈妮娃，過來"},
			[3]string{"00:00:17,000", "00:00:19,000", "哈妮娃在哪"},
			[3]string{"00:00:21,000", "00:00:23,000", "哈妮娃知道"},
		)), 0o644))
		return models.Episode{ID: fmt.Sprintf("s%de%d", season, ep), SeriesID: "s1", SeasonNumber: season, EpisodeNumber: ep, FilePath: models.NewNullString(base + ".mkv")}
	}
	run := func(t *testing.T, eps []models.Episode) (*fakeGlossary, MineResult) {
		g := &fakeGlossary{}
		m := NewOfficialSubtitleMiner(fakeEpisodes{bySeries: map[string][]models.Episode{"s1": eps}},
			fakeSeries{rows: []models.Series{{ID: "s1", Title: "See"}}}, fakeScopes{}, g, nil, nil, nil)
		res, err := m.MineSeries(context.Background(), "s1")
		require.NoError(t, err)
		return g, res
	}
	rows := func(g *fakeGlossary) map[string]string { // "scope|src" → zh
		out := map[string]string{}
		for _, r := range append(append([]models.GlossaryTerm{}, g.inserted...), g.upserted...) {
			out[r.Scope+"|"+r.TermSrc] = r.TermZh
		}
		return out
	}

	t.Run("more episodes win; each season keeps its drawer", func(t *testing.T) {
		g, res := run(t, []models.Episode{mk(1, 1, "謝拉馬威"), mk(1, 2, "謝拉馬威"), mk(2, 1, "傑拉馬瑞"), mk(2, 2, "傑拉馬瑞"), mk(2, 3, "傑拉馬瑞")})
		r := rows(g)
		assert.Equal(t, "傑拉馬瑞", r["tmdb:tv:s1|Jerlamarel"], "3 episodes vs 2")
		assert.Equal(t, "謝拉馬威", r["tmdb:tv:s1:s1|Jerlamarel"])
		assert.Equal(t, "傑拉馬瑞", r["tmdb:tv:s1:s2|Jerlamarel"])
		assert.Equal(t, "哈妮娃", r["tmdb:tv:s1|Haniwa"])
		_, s1Haniwa := r["tmdb:tv:s1:s1|Haniwa"]
		assert.False(t, s1Haniwa, "a name the seasons agree on gets no season drawer")
		assert.Equal(t, 1, res.TermsSplit)
		assert.Equal(t, []string{"tmdb:tv:s1"}, g.cleared, "drawers are cleared before being rewritten")
	})
	t.Run("a tie goes to the latest season", func(t *testing.T) {
		g, _ := run(t, []models.Episode{mk(1, 1, "謝拉馬威"), mk(1, 2, "謝拉馬威"), mk(2, 1, "傑拉馬瑞"), mk(2, 2, "傑拉馬瑞")})
		assert.Equal(t, "傑拉馬瑞", rows(g)["tmdb:tv:s1|Jerlamarel"])
	})
	t.Run("one season only → no drawers at all", func(t *testing.T) {
		g, res := run(t, []models.Episode{mk(1, 1, "謝拉馬威"), mk(1, 2, "謝拉馬威"), mk(1, 3, "謝拉馬威")})
		r := rows(g)
		assert.Equal(t, "謝拉馬威", r["tmdb:tv:s1|Jerlamarel"])
		_, has := r["tmdb:tv:s1:s1|Jerlamarel"]
		assert.False(t, has)
		assert.Equal(t, 0, res.TermsSplit)
	})
}

// disc-2026-10-glossary-prune-harvest-garble: a harvested guess the official
// English never says is deleted; a harvested term the English does say, a
// confirmed row, and non-harvest rows stay; a thin corpus prunes nothing.
func TestMineSeries_PrunesHarvestedGarble(t *testing.T) {
	dir := t.TempDir()
	var eps []models.Episode
	for i := 1; i <= 4; i++ {
		base := filepath.Join(dir, fmt.Sprintf("Show.S01E0%d", i))
		require.NoError(t, os.WriteFile(base+".mkv", []byte("x"), 0o644))
		require.NoError(t, os.WriteFile(base+".en.srt", []byte(fourLines(false, "")), 0o644))
		require.NoError(t, os.WriteFile(base+".zh-TW.srt", []byte(fourLines(true, "謝拉馬威")), 0o644))
		eps = append(eps, models.Episode{ID: fmt.Sprintf("e%d", i), SeriesID: "s1", FilePath: models.NewNullString(base + ".mkv")})
	}
	existing := []models.GlossaryTerm{
		{ID: "garble-1", TermSrc: "Chola Morel", TermZh: "丘拉·莫瑞爾", Source: models.GlossarySourceSubtitle},
		{ID: "garble-2", TermSrc: "Cofoun", TermZh: "科鋒", Source: models.GlossarySourceSubtitle},
		{ID: "said", TermSrc: "Jerlamarel", TermZh: "傑拉瑪瑞爾", Source: models.GlossarySourceSubtitle}, // said → replaced, not pruned
		{ID: "confirmed", TermSrc: "Sungrave", TermZh: "日升之地", Source: models.GlossarySourceSubtitle, Confirmed: true},
		{ID: "tmdb", TermSrc: "Timat Dijon", TermZh: "提馬特", Source: models.GlossarySourceMetadata},
	}
	t.Run("enough coverage → garble goes, the rest stays", func(t *testing.T) {
		g := &fakeGlossary{existing: append([]models.GlossaryTerm{}, existing...)}
		m := NewOfficialSubtitleMiner(fakeEpisodes{bySeries: map[string][]models.Episode{"s1": eps}},
			fakeSeries{rows: []models.Series{{ID: "s1", Title: "See"}}}, fakeScopes{}, g, nil, nil, nil)
		res, err := m.MineSeries(context.Background(), "s1")
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"garble-1", "garble-2"}, g.deleted)
		assert.Equal(t, 2, res.TermsPruned)
		assert.Equal(t, []string{"Chola Morel", "Cofoun"}, res.Pruned)
		assert.Equal(t, 1, res.TermsReplaced, "the harvested Jerlamarel is REPLACED by the official rendering, not pruned")
		require.Len(t, g.upserted, 1)
		assert.Equal(t, "謝拉馬威", g.upserted[0].TermZh)
	})
	t.Run("a mention only in an English line the Chinese file lacks still counts", func(t *testing.T) {
		// CR 2: the prune corpus is every English cue, not only the aligned ones.
		dir := t.TempDir()
		var eps2 []models.Episode
		for i := 1; i <= 2; i++ {
			base := filepath.Join(dir, fmt.Sprintf("Show.S01E0%d", i))
			require.NoError(t, os.WriteFile(base+".mkv", []byte("x"), 0o644))
			en := fourLines(false, "") + srt([3]string{"00:00:40,000", "00:00:42,000", "Chola Morel is here."})
			require.NoError(t, os.WriteFile(base+".en.srt", []byte(en), 0o644))
			require.NoError(t, os.WriteFile(base+".zh-TW.srt", []byte(fourLines(true, "謝拉馬威")), 0o644))
			eps2 = append(eps2, models.Episode{ID: fmt.Sprintf("x%d", i), SeriesID: "s1", FilePath: models.NewNullString(base + ".mkv")})
		}
		g := &fakeGlossary{existing: []models.GlossaryTerm{{ID: "garble-1", TermSrc: "Chola Morel", TermZh: "丘拉·莫瑞爾", Source: models.GlossarySourceSubtitle}}}
		m := NewOfficialSubtitleMiner(fakeEpisodes{bySeries: map[string][]models.Episode{"s1": eps2}},
			fakeSeries{rows: []models.Series{{ID: "s1", Title: "See"}}}, fakeScopes{}, g, nil, nil, nil)
		_, err := m.MineSeries(context.Background(), "s1")
		require.NoError(t, err)
		assert.Empty(t, g.deleted)
	})
	t.Run("a season with files but no official subtitles protects the show (CR 1)", func(t *testing.T) {
		dir := t.TempDir()
		var eps2 []models.Episode
		for i := 1; i <= 4; i++ {
			base := filepath.Join(dir, fmt.Sprintf("Show.S01E0%d", i))
			require.NoError(t, os.WriteFile(base+".mkv", []byte("x"), 0o644))
			require.NoError(t, os.WriteFile(base+".en.srt", []byte(fourLines(false, "")), 0o644))
			require.NoError(t, os.WriteFile(base+".zh-TW.srt", []byte(fourLines(true, "謝拉馬威")), 0o644))
			eps2 = append(eps2, models.Episode{ID: fmt.Sprintf("a%d", i), SeriesID: "s1", SeasonNumber: 1, FilePath: models.NewNullString(base + ".mkv")})
		}
		for i := 1; i <= 2; i++ { // season 2: files, no zh yet — its new names live only in the harvest
			base := filepath.Join(dir, fmt.Sprintf("Show.S02E0%d", i))
			require.NoError(t, os.WriteFile(base+".mkv", []byte("x"), 0o644))
			eps2 = append(eps2, models.Episode{ID: fmt.Sprintf("b%d", i), SeriesID: "s1", SeasonNumber: 2, FilePath: models.NewNullString(base + ".mkv")})
		}
		g := &fakeGlossary{existing: []models.GlossaryTerm{{ID: "new-name", TermSrc: "Wren", TermZh: "蘭恩", Source: models.GlossarySourceSubtitle}}}
		m := NewOfficialSubtitleMiner(fakeEpisodes{bySeries: map[string][]models.Episode{"s1": eps2}},
			fakeSeries{rows: []models.Series{{ID: "s1", Title: "See"}}}, fakeScopes{}, g, nil, nil, nil)
		res, err := m.MineSeries(context.Background(), "s1")
		require.NoError(t, err)
		assert.Equal(t, 4, res.EpisodesUsed)
		assert.Empty(t, g.deleted, "4 of 6 would pass the half rule; the unmined season stops it")
	})
	t.Run("one episode is not enough to judge", func(t *testing.T) {
		g := &fakeGlossary{existing: append([]models.GlossaryTerm{}, existing...)}
		m := NewOfficialSubtitleMiner(fakeEpisodes{bySeries: map[string][]models.Episode{"s1": eps[:1]}},
			fakeSeries{rows: []models.Series{{ID: "s1", Title: "See"}}}, fakeScopes{}, g, nil, nil, nil)
		res, err := m.MineSeries(context.Background(), "s1")
		require.NoError(t, err)
		assert.Empty(t, g.deleted)
		assert.Equal(t, 0, res.TermsPruned)
	})
}
