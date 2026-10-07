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
}

func (g *fakeGlossary) ListByScope(context.Context, string) ([]models.GlossaryTerm, error) {
	return g.existing, nil
}

// ReplaceUnconfirmedGuess mirrors the repository's SQL guard: insert when
// absent; overwrite only an unconfirmed harvest / earlier-mining row whose
// rendering differs; leave everything else alone.
func (g *fakeGlossary) ReplaceUnconfirmedGuess(_ context.Context, t *models.GlossaryTerm) (bool, error) {
	for i, e := range g.existing {
		if !strings.EqualFold(e.TermSrc, t.TermSrc) {
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
