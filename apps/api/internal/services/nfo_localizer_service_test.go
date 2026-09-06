package services

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
)

func sampleMovie(t *testing.T, dir string) models.Movie {
	t.Helper()
	return models.Movie{
		ID:            "42",
		Title:         "Inception",
		OriginalTitle: models.NewNullString("Inception"),
		ReleaseDate:   "2010",
		Overview:      models.NewNullString("A thief who steals corporate secrets through dream-sharing."),
		Genres:        []string{"Science Fiction", "Action"},
		VoteAverage:   models.NewNullFloat64(8.4),
		TMDbID:        models.NewNullInt64(27205),
		CreditsJSON:   models.NewNullString(`{"cast":[{"name":"Leonardo DiCaprio","character":"Dom Cobb"}],"crew":[{"name":"Christopher Nolan","job":"Director"}]}`),
		FilePath:      models.NewNullString(filepath.Join(dir, "Inception (2010).mkv")),
	}
}

// glossaryReturningStub returns fixed terms for LookupByScope.
type glossaryReturningStub struct{ terms map[string]string }

func (g *glossaryReturningStub) Upsert(ctx context.Context, t *models.GlossaryTerm) error { return nil }
func (g *glossaryReturningStub) InsertIfAbsent(ctx context.Context, t *models.GlossaryTerm) (bool, error) {
	return false, nil
}
func (g *glossaryReturningStub) ListByScope(ctx context.Context, scope string) ([]models.GlossaryTerm, error) {
	return nil, nil
}
func (g *glossaryReturningStub) LookupByScope(ctx context.Context, scope string, confirmedOnly bool) (map[string]string, error) {
	return g.terms, nil
}
func (g *glossaryReturningStub) Update(ctx context.Context, id, termZh string, confirmed bool) (time.Time, error) {
	return time.Time{}, nil
}
func (g *glossaryReturningStub) Confirm(ctx context.Context, id string) (time.Time, error) {
	return time.Time{}, nil
}
func (g *glossaryReturningStub) ConfirmAllByScope(ctx context.Context, scope string) (int64, error) {
	return 0, nil
}
func (g *glossaryReturningStub) MigrateScope(ctx context.Context, from, to string) (int64, int64, error) {
	return 0, 0, nil
}
func (g *glossaryReturningStub) Delete(ctx context.Context, id string) error { return nil }

func newLocalizer(t *testing.T, completerResp string, terms map[string]string) *NFOLocalizerService {
	t.Helper()
	completer := &mockTranslationCompleter{response: completerResp}
	svc := NewNFOLocalizerService(NewTranslationService(completer, nil), &glossaryReturningStub{terms: terms}, nil)
	require.NotNil(t, svc)
	return svc
}

func TestNFOLocalizer_NoOriginal_WritesFilenameSlot(t *testing.T) {
	dir := t.TempDir()
	movie := sampleMovie(t, dir)
	// 5 fields (title, plot, 2 genres, 1 role) → index-prefixed response.
	resp := "[1] 全面啟動\n[2] 一名盜賊竊取企業機密\n[3] 科幻\n[4] 動作\n[5] 唐姆·柯布"
	svc := newLocalizer(t, resp, map[string]string{"Dom Cobb": "唐姆·柯布"})

	res, err := svc.LocalizeMovieNFO(context.Background(), movie)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "Inception (2010).nfo"), res.Path)
	assert.False(t, res.Replaced)

	out, _ := os.ReadFile(res.Path)
	body := string(out)
	assert.Contains(t, body, "<title>全面啟動</title>")
	assert.Contains(t, body, "一名盜賊竊取企業機密")
	assert.Contains(t, body, "<genre>科幻</genre>")
	assert.Contains(t, body, "唐姆·柯布")
	// Preserved fields: original title, year, uniqueid, person name.
	assert.Contains(t, body, "<originaltitle>Inception</originaltitle>")
	assert.Contains(t, body, "<year>2010</year>")
	assert.Contains(t, body, `type="tmdb"`)
	assert.Contains(t, body, "Leonardo DiCaprio")
}

func TestNFOLocalizer_OriginalAtFilenameSlot_WritesMovieNfoAdditive(t *testing.T) {
	dir := t.TempDir()
	movie := sampleMovie(t, dir)
	orig := filepath.Join(dir, "Inception (2010).nfo")
	require.NoError(t, os.WriteFile(orig, []byte("<movie><title>Inception</title></movie>"), 0o644))

	svc := newLocalizer(t, "[1] 全面啟動\n[2] 劇情\n[3] 科幻\n[4] 動作\n[5] 柯布", nil)
	res, err := svc.LocalizeMovieNFO(context.Background(), movie)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "movie.nfo"), res.Path, "free-slot: original at <basename>.nfo → write movie.nfo")

	// Original untouched.
	origContent, _ := os.ReadFile(orig)
	assert.Equal(t, "<movie><title>Inception</title></movie>", string(origContent))
}

func TestNFOLocalizer_OriginalAtMovieNfo_WritesFilenameSlotAdditive(t *testing.T) {
	dir := t.TempDir()
	movie := sampleMovie(t, dir)
	orig := filepath.Join(dir, "movie.nfo")
	require.NoError(t, os.WriteFile(orig, []byte("ORIG"), 0o644))

	svc := newLocalizer(t, "[1] 全面啟動\n[2] 劇情\n[3] 科幻\n[4] 動作\n[5] 柯布", nil)
	res, err := svc.LocalizeMovieNFO(context.Background(), movie)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "Inception (2010).nfo"), res.Path)

	origContent, _ := os.ReadFile(orig)
	assert.Equal(t, "ORIG", string(origContent), "movie.nfo original preserved")
}

func TestNFOLocalizer_BothSlotsOccupied_BackupAndReplace(t *testing.T) {
	dir := t.TempDir()
	movie := sampleMovie(t, dir)
	fnSlot := filepath.Join(dir, "Inception (2010).nfo")
	require.NoError(t, os.WriteFile(fnSlot, []byte("ORIGINAL-FN"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "movie.nfo"), []byte("ORIGINAL-MOVIE"), 0o644))

	svc := newLocalizer(t, "[1] 全面啟動\n[2] 劇情\n[3] 科幻\n[4] 動作\n[5] 柯布", nil)
	res, err := svc.LocalizeMovieNFO(context.Background(), movie)
	require.NoError(t, err)
	assert.True(t, res.Replaced)
	assert.Equal(t, fnSlot, res.Path)
	assert.Equal(t, fnSlot+".orig", res.BackupPath)

	// The original was preserved in the .orig backup.
	backup, _ := os.ReadFile(fnSlot + ".orig")
	assert.Equal(t, "ORIGINAL-FN", string(backup), "original must survive in the .orig backup")
	// The slot now holds the zh-TW nfo.
	newContent, _ := os.ReadFile(fnSlot)
	assert.Contains(t, string(newContent), "全面啟動")
}

func TestNFOLocalizer_GlossaryInjectedIntoPrompt(t *testing.T) {
	dir := t.TempDir()
	movie := sampleMovie(t, dir)
	completer := &mockTranslationCompleter{response: "[1] 全面啟動\n[2] 劇情\n[3] 科幻\n[4] 動作\n[5] 唐姆·柯布"}
	svc := NewNFOLocalizerService(NewTranslationService(completer, nil), &glossaryReturningStub{terms: map[string]string{"Dom Cobb": "唐姆·柯布"}}, nil)

	_, err := svc.LocalizeMovieNFO(context.Background(), movie)
	require.NoError(t, err)
	require.Len(t, completer.calls, 1)
	assert.Contains(t, completer.calls[0].UserPrompt, "Dom Cobb → 唐姆·柯布")
	// All localizable fields batched in one request.
	assert.True(t, strings.Contains(completer.calls[0].UserPrompt, "Inception"))
}

func TestNFOLocalizer_NilWhenTranslationUnavailable(t *testing.T) {
	assert.Nil(t, NewNFOLocalizerService(nil, nil, nil))
}

// TestNFOLocalizer_KeylessBootConstructsAndRecoversWhenAKeyArrives pins the CR
// sub-2-1a M1 fix: since keys hot-reload, the constructor must NOT freeze a
// boot-time IsConfigured() snapshot into a nil service (a permanently-404 route
// no later key save could revive). Availability is a per-call question.
func TestNFOLocalizer_KeylessBootConstructsAndRecoversWhenAKeyArrives(t *testing.T) {
	secrets := &fakeSecrets{}
	holder := NewClaudeProviderHolder(NewKeyResolver(secrets, EnvKeys{}, nil), "", nil)

	svc := NewNFOLocalizerService(NewTranslationService(holder, nil), nil, nil)

	require.NotNil(t, svc, "a keyless boot must still construct the localizer")
	assert.False(t, svc.IsAvailable(), "…but it declines while no key resolves")

	// The settings page stores a key at runtime — no restart.
	require.NoError(t, secrets.Store(context.Background(), SecretNameClaude, "sk-late-arrival"))

	assert.True(t, svc.IsAvailable(), "a saved key must revive the feature without a restart")
}

// ─── 9R-13a: TV (tvshow.nfo + per-episode .nfo) ─────────────────────────────

// showTree builds a real on-disk layout so path assertions mean something:
//
//	<tmp>/Buffy/          ← series.FilePath (the SHOW FOLDER, not a file)
//	<tmp>/Buffy/Season01/S01E01.mkv
func showTree(t *testing.T) (showDir, episodePath string) {
	t.Helper()
	showDir = filepath.Join(t.TempDir(), "Buffy")
	seasonDir := filepath.Join(showDir, "Season01")
	require.NoError(t, os.MkdirAll(seasonDir, 0o755))
	episodePath = filepath.Join(seasonDir, "Buffy.S01E01.mkv")
	require.NoError(t, os.WriteFile(episodePath, []byte("fake"), 0o644))
	return showDir, episodePath
}

func sampleSeries(showDir string) models.Series {
	return models.Series{
		ID:            "series-1",
		Title:         "Buffy the Vampire Slayer",
		OriginalTitle: models.NewNullString("Buffy the Vampire Slayer"),
		FirstAirDate:  "1997",
		Overview:      models.NewNullString("A teenager battles the forces of darkness."),
		Genres:        []string{"Drama", "Fantasy"},
		VoteAverage:   models.NewNullFloat64(8.3),
		TMDbID:        models.NewNullInt64(95),
		Status:        models.NewNullString("Ended"),
		CreditsJSON:   models.NewNullString(`{"cast":[{"name":"Sarah Michelle Gellar","character":"Buffy Summers"}]}`),
		FilePath:      models.NewNullString(showDir),
	}
}

func sampleEpisode(episodePath string) models.Episode {
	return models.Episode{
		ID:            "episode-1",
		SeriesID:      "series-1",
		SeasonNumber:  1,
		EpisodeNumber: 1,
		Title:         models.NewNullString("Welcome to the Hellmouth"),
		Overview:      models.NewNullString("Buffy arrives in Sunnydale."),
		AirDate:       models.NewNullString("1997-03-10"),
		TMDbID:        models.NewNullInt64(11111),
		FilePath:      models.NewNullString(episodePath),
	}
}

// 🔴 The single most important assertion in this story: tvshow.nfo lands INSIDE
// the show folder. series.FilePath is already the folder, so a stray
// filepath.Dir() would put it in the library ROOT — invisible to every player
// and polluting the user's library. Asserted on the FULL path, not a substring.
func TestNFOLocalizer_TVShow_WritesInsideTheShowFolder(t *testing.T) {
	showDir, _ := showTree(t)
	series := sampleSeries(showDir)
	// 5 fields: title, plot, 2 genres, 1 role.
	resp := "[1] 魔法奇兵\n[2] 少女對抗黑暗勢力\n[3] 劇情\n[4] 奇幻\n[5] 巴菲·桑默斯"
	svc := newLocalizer(t, resp, nil)

	res, err := svc.LocalizeTVShowNFO(context.Background(), series)
	require.NoError(t, err)

	assert.Equal(t, filepath.Join(showDir, "tvshow.nfo"), res.Path)
	assert.NotEqual(t, filepath.Join(filepath.Dir(showDir), "tvshow.nfo"), res.Path,
		"a stray filepath.Dir() would write into the library root")
	assert.False(t, res.Replaced)

	body, err := os.ReadFile(res.Path)
	require.NoError(t, err)
	out := string(body)
	assert.Contains(t, out, "<tvshow>")
	assert.Contains(t, out, "<title>魔法奇兵</title>")
	assert.Contains(t, out, "<genre>劇情</genre>")
	assert.Contains(t, out, "巴菲·桑默斯")
	// Preserved: original title, year, uniqueid, person name, status.
	assert.Contains(t, out, "<originaltitle>Buffy the Vampire Slayer</originaltitle>")
	assert.Contains(t, out, "<year>1997</year>")
	assert.Contains(t, out, `type="tmdb"`)
	assert.Contains(t, out, "Sarah Michelle Gellar")
	assert.Contains(t, out, "<status>Ended</status>")
}

func TestNFOLocalizer_TVShow_BacksUpExistingThenReplaces(t *testing.T) {
	showDir, _ := showTree(t)
	target := filepath.Join(showDir, "tvshow.nfo")
	require.NoError(t, os.WriteFile(target, []byte("<tvshow><title>ORIGINAL</title></tvshow>"), 0o644))

	svc := newLocalizer(t, "[1] 魔法奇兵\n[2] 少女對抗黑暗勢力\n[3] 劇情\n[4] 奇幻\n[5] 巴菲", nil)
	res, err := svc.LocalizeTVShowNFO(context.Background(), sampleSeries(showDir))
	require.NoError(t, err)

	assert.True(t, res.Replaced)
	assert.Equal(t, target+".orig", res.BackupPath)
	backup, err := os.ReadFile(res.BackupPath)
	require.NoError(t, err)
	assert.Contains(t, string(backup), "ORIGINAL", "the user's original must survive verbatim")

	written, _ := os.ReadFile(target)
	assert.Contains(t, string(written), "魔法奇兵")
}

// Re-running localization must never let a previously localized file become the
// "original" — the backup is written once and then left alone forever.
func TestNFOLocalizer_TVShow_ExistingBackupIsNeverOverwritten(t *testing.T) {
	showDir, _ := showTree(t)
	target := filepath.Join(showDir, "tvshow.nfo")
	require.NoError(t, os.WriteFile(target, []byte("SECOND RUN CONTENT"), 0o644))
	require.NoError(t, os.WriteFile(target+".orig", []byte("THE TRUE ORIGINAL"), 0o644))

	svc := newLocalizer(t, "[1] 魔法奇兵\n[2] 少女對抗黑暗勢力\n[3] 劇情\n[4] 奇幻\n[5] 巴菲", nil)
	_, err := svc.LocalizeTVShowNFO(context.Background(), sampleSeries(showDir))
	require.NoError(t, err)

	backup, _ := os.ReadFile(target + ".orig")
	assert.Equal(t, "THE TRUE ORIGINAL", string(backup))
}

func TestNFOLocalizer_TVShow_NoFolderPathIsAnError(t *testing.T) {
	svc := newLocalizer(t, "[1] x", nil)
	series := sampleSeries("")
	series.FilePath = models.NullString{}
	_, err := svc.LocalizeTVShowNFO(context.Background(), series)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no folder path")
}

// 🔴 An episode's FilePath IS a file, so the .nfo goes beside it — inside the
// season folder, NOT in the show folder.
func TestNFOLocalizer_Episode_WritesBesideTheEpisodeFile(t *testing.T) {
	showDir, episodePath := showTree(t)
	svc := newLocalizer(t, "[1] 歡迎來到地獄口\n[2] 巴菲來到桑尼戴爾", nil)

	res, err := svc.LocalizeEpisodeNFO(context.Background(), sampleEpisode(episodePath), "魔法奇兵")
	require.NoError(t, err)

	assert.Equal(t, filepath.Join(filepath.Dir(episodePath), "Buffy.S01E01.nfo"), res.Path)
	assert.NotEqual(t, filepath.Join(showDir, "Buffy.S01E01.nfo"), res.Path,
		"the episode nfo belongs in the SEASON folder, beside its video")

	body, _ := os.ReadFile(res.Path)
	out := string(body)
	assert.Contains(t, out, "<episodedetails>", "Kodi keys on this exact root element")
	assert.Contains(t, out, "<title>歡迎來到地獄口</title>")
	assert.Contains(t, out, "<plot>巴菲來到桑尼戴爾</plot>")
	assert.Contains(t, out, "<showtitle>魔法奇兵</showtitle>")
	assert.Contains(t, out, "<season>1</season>")
	assert.Contains(t, out, "<episode>1</episode>")
	assert.Contains(t, out, "<aired>1997-03-10</aired>")
	assert.NotContains(t, out, "<actor>", "episodes carry no credits — no invented cast")
}

func TestNFOLocalizer_Episode_BacksUpExistingThenReplaces(t *testing.T) {
	_, episodePath := showTree(t)
	target := filepath.Join(filepath.Dir(episodePath), "Buffy.S01E01.nfo")
	require.NoError(t, os.WriteFile(target, []byte("ORIGINAL EPISODE NFO"), 0o644))

	svc := newLocalizer(t, "[1] 歡迎來到地獄口\n[2] 巴菲來到桑尼戴爾", nil)
	res, err := svc.LocalizeEpisodeNFO(context.Background(), sampleEpisode(episodePath), "魔法奇兵")
	require.NoError(t, err)

	assert.True(t, res.Replaced)
	backup, _ := os.ReadFile(target + ".orig")
	assert.Equal(t, "ORIGINAL EPISODE NFO", string(backup))
}

// 🔴 The glossary must key on the parent SERIES. A per-episode key gives every
// episode its own private vocabulary, so one character is rendered two ways
// across a season — the bug sub-5-5 CR H1 fixed on the subtitle side.
func TestNFOLocalizer_Episode_GlossaryKeysOnTheParentSeries(t *testing.T) {
	_, episodePath := showTree(t)
	spy := &glossaryKeySpy{}
	svc := NewNFOLocalizerService(
		NewTranslationService(&mockTranslationCompleter{response: "[1] 歡迎來到地獄口\n[2] 巴菲來到桑尼戴爾"}, nil),
		spy, nil)
	require.NotNil(t, svc)

	_, err := svc.LocalizeEpisodeNFO(context.Background(), sampleEpisode(episodePath), "魔法奇兵")
	require.NoError(t, err)

	require.Len(t, spy.keys, 1)
	// No resolver wired → the SeriesID under its sub-7-1 name (local drawer).
	assert.Equal(t, "local:series-1", spy.keys[0], "must be keyed by the SeriesID, never the episode id")
}

// sub-7-1 AC #5(d): with a resolver wired the localizer reads the RESOLVED
// scope, and still keys the resolve on the SeriesID for an episode.
func TestNFOLocalizer_Episode_ReadsGlossaryByResolvedScope(t *testing.T) {
	_, episodePath := showTree(t)
	spy := &glossaryKeySpy{}
	svc := NewNFOLocalizerService(
		NewTranslationService(&mockTranslationCompleter{response: "[1] 歡迎來到地獄口\n[2] 巴菲來到桑尼戴爾"}, nil),
		spy, nil)
	require.NotNil(t, svc)
	resolver := &recordingResolver{scope: "tmdb:tv:95"}
	svc.SetGlossaryScopeResolver(resolver)

	_, err := svc.LocalizeEpisodeNFO(context.Background(), sampleEpisode(episodePath), "魔法奇兵")
	require.NoError(t, err)

	assert.Equal(t, []string{"series-1"}, resolver.asked, "the resolver is asked with the SeriesID")
	assert.Equal(t, []string{"tmdb:tv:95"}, spy.keys, "the repository is read by the resolved scope")
}

// recordingResolver is a GlossaryScopeResolverInterface that answers one scope
// and remembers what it was asked.
type recordingResolver struct {
	scope string
	asked []string
}

func (r *recordingResolver) Resolve(_ context.Context, mediaID string) (string, error) {
	r.asked = append(r.asked, mediaID)
	return r.scope, nil
}

// A translation response that omits a key leaves that field alone.
func TestNFOLocalizer_Episode_MissingTranslationKeepsTheOriginal(t *testing.T) {
	_, episodePath := showTree(t)
	// Only [1] comes back — [2] (plot) is missing.
	svc := newLocalizer(t, "[1] 歡迎來到地獄口", nil)

	res, err := svc.LocalizeEpisodeNFO(context.Background(), sampleEpisode(episodePath), "魔法奇兵")
	require.NoError(t, err)

	body, _ := os.ReadFile(res.Path)
	assert.Contains(t, string(body), "<title>歡迎來到地獄口</title>")
	assert.Contains(t, string(body), "<plot>Buffy arrives in Sunnydale.</plot>",
		"a missing key keeps the original — per-field fail-soft")
}

func TestNFOLocalizer_Episode_NoFilePathIsAnError(t *testing.T) {
	svc := newLocalizer(t, "[1] x", nil)
	ep := sampleEpisode("")
	ep.FilePath = models.NullString{}
	_, err := svc.LocalizeEpisodeNFO(context.Background(), ep, "魔法奇兵")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no file path")
}

// ─── whole-show batch ───────────────────────────────────────────────────────

type stubEpisodeLister struct {
	episodes []models.Episode
	err      error
	calls    int
}

func (l *stubEpisodeLister) FindBySeriesID(ctx context.Context, seriesID string) ([]models.Episode, error) {
	l.calls++
	if l.err != nil {
		return nil, l.err
	}
	return l.episodes, nil
}

type glossaryKeySpy struct {
	glossaryReturningStub
	keys []string
}

func (g *glossaryKeySpy) LookupByScope(ctx context.Context, scope string, confirmedOnly bool) (map[string]string, error) {
	g.keys = append(g.keys, scope)
	return nil, nil
}

func TestNFOLocalizer_Batch_SkipsEpisodesWithoutFilesAndKeepsGoing(t *testing.T) {
	showDir, episodePath := showTree(t)
	second := filepath.Join(filepath.Dir(episodePath), "Buffy.S01E02.mkv")
	require.NoError(t, os.WriteFile(second, []byte("fake"), 0o644))

	withFile := sampleEpisode(episodePath)
	withFile2 := sampleEpisode(second)
	withFile2.ID = "episode-2"
	withFile2.EpisodeNumber = 2
	noFile := sampleEpisode("")
	noFile.ID = "episode-3"
	noFile.FilePath = models.NullString{}

	lister := &stubEpisodeLister{episodes: []models.Episode{withFile, withFile2, noFile}}
	svc := newLocalizer(t, "[1] 譯名\n[2] 譯述\n[3] 劇情\n[4] 奇幻\n[5] 角色", nil)
	svc.SetEpisodeLister(lister)

	res, err := svc.LocalizeSeriesNFOWithEpisodes(context.Background(), sampleSeries(showDir))
	require.NoError(t, err)

	assert.Equal(t, 2, res.Succeeded)
	assert.Equal(t, 0, res.Failed)
	assert.Equal(t, 1, res.Skipped)
	assert.Equal(t, 1, lister.calls, "the series is enumerated exactly once — no N+1")
	assert.FileExists(t, filepath.Join(showDir, "tvshow.nfo"))
	assert.FileExists(t, filepath.Join(filepath.Dir(episodePath), "Buffy.S01E01.nfo"))
	assert.FileExists(t, filepath.Join(filepath.Dir(second), "Buffy.S01E02.nfo"))
}

// The show file is already on disk by the time enumeration runs; a lister
// failure must degrade to show-only rather than report total failure.
func TestNFOLocalizer_Batch_ListerFailureStillReportsTheShowFile(t *testing.T) {
	showDir, _ := showTree(t)
	svc := newLocalizer(t, "[1] 魔法奇兵\n[2] 少女對抗黑暗勢力\n[3] 劇情\n[4] 奇幻\n[5] 巴菲", nil)
	svc.SetEpisodeLister(&stubEpisodeLister{err: assertAnError})

	res, err := svc.LocalizeSeriesNFOWithEpisodes(context.Background(), sampleSeries(showDir))
	require.NoError(t, err)
	assert.NotNil(t, res.Show)
	assert.Empty(t, res.Episodes)
	assert.Equal(t, 0, res.Succeeded)
	assert.FileExists(t, filepath.Join(showDir, "tvshow.nfo"))
}

func TestNFOLocalizer_Batch_NoListerWiredLocalizesShowOnly(t *testing.T) {
	showDir, _ := showTree(t)
	svc := newLocalizer(t, "[1] 魔法奇兵\n[2] 少女對抗黑暗勢力\n[3] 劇情\n[4] 奇幻\n[5] 巴菲", nil)

	res, err := svc.LocalizeSeriesNFOWithEpisodes(context.Background(), sampleSeries(showDir))
	require.NoError(t, err)
	assert.NotNil(t, res.Show)
	assert.Empty(t, res.Episodes)
}

var assertAnError = errorString("episode lookup exploded")

type errorString string

func (e errorString) Error() string { return string(e) }

// CR M1: a 24-episode run used to re-read the SAME series glossary 24 times.
// After the fix the whole-show batch reads it twice in total — once for the
// show file, once for the episode loop — regardless of episode count.
func TestNFOLocalizer_Batch_LoadsTheShowGlossaryOncePerRunNotPerEpisode(t *testing.T) {
	showDir, episodePath := showTree(t)
	seasonDir := filepath.Dir(episodePath)
	episodes := []models.Episode{sampleEpisode(episodePath)}
	for i := 2; i <= 4; i++ {
		path := filepath.Join(seasonDir, "Buffy.S01E0"+string(rune('0'+i))+".mkv")
		require.NoError(t, os.WriteFile(path, []byte("fake"), 0o644))
		ep := sampleEpisode(path)
		ep.ID = "episode-" + string(rune('0'+i))
		ep.EpisodeNumber = i
		episodes = append(episodes, ep)
	}

	spy := &glossaryKeySpy{}
	svc := NewNFOLocalizerService(
		NewTranslationService(&mockTranslationCompleter{response: "[1] 譯名\n[2] 譯述\n[3] 劇情\n[4] 奇幻\n[5] 角色"}, nil),
		spy, nil)
	require.NotNil(t, svc)
	svc.SetEpisodeLister(&stubEpisodeLister{episodes: episodes})

	res, err := svc.LocalizeSeriesNFOWithEpisodes(context.Background(), sampleSeries(showDir))
	require.NoError(t, err)
	require.Equal(t, 4, res.Succeeded)

	assert.Len(t, spy.keys, 2,
		"one glossary read for the show file + one for the whole episode loop — never one per episode")
	for _, k := range spy.keys {
		assert.Equal(t, "local:series-1", k)
	}
}

// CR M3: tvshow.nfo is JOINED onto series.FilePath, so a path that is not a
// directory must fail with something an operator can act on rather than an
// os.WriteFile "not a directory" from deep inside the write.
func TestNFOLocalizer_TVShow_FilePathPointingAtAFileIsAClearError(t *testing.T) {
	_, episodePath := showTree(t)
	series := sampleSeries(episodePath) // a FILE, not the show folder

	svc := newLocalizer(t, "[1] x", nil)
	_, err := svc.LocalizeTVShowNFO(context.Background(), series)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a readable directory")
	assert.NotContains(t, err.Error(), "localize fields",
		"the guard must fire before any translation is paid for")
}

func TestNFOLocalizer_TVShow_MissingFolderIsAClearError(t *testing.T) {
	svc := newLocalizer(t, "[1] x", nil)
	_, err := svc.LocalizeTVShowNFO(context.Background(), sampleSeries(filepath.Join(t.TempDir(), "gone")))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a readable directory")
}
