package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/metadata"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/parser"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/tmdb"
)

// dsr-2b-a: an unmatched library item finally has working ways back —
// applying a TMDb match the user picked (AC #1), re-matching one item on
// demand (AC #2), and auto-enrichment never overwriting what the user chose or
// typed (AC #4). Most of these run against the real SQLite schema: the bug this
// story exists for was a write path that "succeeded" without writing, which a
// mocked repository cannot see (Rule 15, bugfix-20-1).

// ─── fixtures ───────────────────────────────────────────────────────────────

type matchTMDbFake struct {
	mockTMDbServiceForNFO
	tvResp   *tmdb.TVShowDetails
	tvErr    error
	movieIDs []int
	tvIDs    []int
}

func (m *matchTMDbFake) GetMovieDetails(ctx context.Context, id int) (*tmdb.MovieDetails, error) {
	m.movieIDs = append(m.movieIDs, id)
	return m.mockTMDbServiceForNFO.GetMovieDetails(ctx, id)
}

func (m *matchTMDbFake) GetTVShowDetails(_ context.Context, id int) (*tmdb.TVShowDetails, error) {
	m.tvIDs = append(m.tvIDs, id)
	if m.tvErr != nil {
		return nil, m.tvErr
	}
	return m.tvResp, nil
}

// recordingParser remembers what it was asked to parse.
type recordingParser struct {
	mockPQParserService
	mu     sync.Mutex
	inputs []string
}

func (p *recordingParser) ParseFilenameWithContext(ctx context.Context, filename string) *parser.ParseResult {
	p.mu.Lock()
	p.inputs = append(p.inputs, filename)
	p.mu.Unlock()
	return p.mockPQParserService.ParseFilenameWithContext(ctx, filename)
}

// ParseFilename is the context-free variant; recorded with a marker so a test
// can prove the cancellable variant was used (a 60s re-match deadline is
// meaningless if the AI parse underneath cannot be cancelled).
func (p *recordingParser) ParseFilename(filename string) *parser.ParseResult {
	p.mu.Lock()
	p.inputs = append(p.inputs, "noctx:"+filename)
	p.mu.Unlock()
	return p.result
}

// hookedMetadata runs onSearch before answering — the seam for "the user acts
// while enrichment is mid-flight" and for a search that outlives its deadline.
type hookedMetadata struct {
	mockPQMetadataService
	onSearch func(ctx context.Context)
	calls    int
}

func (m *hookedMetadata) SearchMetadata(ctx context.Context, req *SearchMetadataRequest) (*metadata.SearchResult, *metadata.FallbackStatus, error) {
	m.calls++
	if m.onSearch != nil {
		m.onSearch(ctx)
	}
	return m.mockPQMetadataService.SearchMetadata(ctx, req)
}

func tmdbMovieMatch(id, title string) *metadata.SearchResult {
	return &metadata.SearchResult{
		Source: models.MetadataSourceTMDb,
		Items:  []metadata.MetadataItem{{ID: id, Title: title, MediaType: metadata.MediaTypeMovie}},
	}
}

type matchEnv struct {
	ctx        context.Context
	movies     *repository.MovieRepository
	series     *repository.SeriesRepository
	tmdb       *matchTMDbFake
	parser     *recordingParser
	meta       *hookedMetadata
	seeder     *fakeGlossarySeeder
	scopes     *fakeScopeResolver
	enrichment *EnrichmentService
}

func newMatchEnv(t *testing.T) *matchEnv {
	t.Helper()
	db := setupTestDB(t)
	env := &matchEnv{
		ctx:    context.Background(),
		movies: repository.NewMovieRepository(db),
		series: repository.NewSeriesRepository(db),
		tmdb:   &matchTMDbFake{},
		parser: &recordingParser{mockPQParserService: mockPQParserService{result: &parser.ParseResult{
			Status: parser.ParseStatusSuccess, MediaType: parser.MediaTypeMovie, CleanedTitle: "Fight Club",
		}}},
		meta:   &hookedMetadata{},
		seeder: &fakeGlossarySeeder{credits: seedTestCredits, pairs: seedTestPairs},
		scopes: &fakeScopeResolver{scope: "tmdb:movie:550"},
	}
	env.enrichment = NewEnrichmentService(env.movies, env.parser, env.meta, nil, env.tmdb, nil, nil, nil)
	env.enrichment.SetSeriesRepo(env.series)
	env.enrichment.SetGlossarySeeder(env.seeder, env.scopes)
	return env
}

func (e *matchEnv) seedMovie(t *testing.T, m *models.Movie) *models.Movie {
	t.Helper()
	if m.ID == "" {
		m.ID = fmt.Sprintf("m-%d", time.Now().UnixNano())
	}
	require.NoError(t, e.movies.Create(e.ctx, m))
	return m
}

func (e *matchEnv) seedSeries(t *testing.T, s *models.Series) *models.Series {
	t.Helper()
	if s.ID == "" {
		s.ID = fmt.Sprintf("s-%d", time.Now().UnixNano())
	}
	require.NoError(t, e.series.Create(e.ctx, s))
	return s
}

func (e *matchEnv) movie(t *testing.T, id string) *models.Movie {
	t.Helper()
	m, err := e.movies.FindByID(e.ctx, id)
	require.NoError(t, err)
	return m
}

func (e *matchEnv) oneSeries(t *testing.T, id string) *models.Series {
	t.Helper()
	s, err := e.series.FindByID(e.ctx, id)
	require.NoError(t, err)
	return s
}

func strptr(s string) *string { return &s }

// ─── AC #1: applying a TMDb match the user picked ──────────────────────────

func TestApplyTMDbMatch_Movie_WritesTheMatchAndReadsBack(t *testing.T) {
	env := newMatchEnv(t)
	env.tmdb.getMovieDetailsResp = &tmdb.MovieDetails{
		Movie: tmdb.Movie{ID: 550, Title: "鬥陣俱樂部", OriginalTitle: "Fight Club", Overview: "失眠的上班族",
			ReleaseDate: "1999-10-15", PosterPath: strptr("/fc.jpg"), BackdropPath: strptr("/fc-bg.jpg"), VoteAverage: 8.4},
		Runtime: 139, Genres: []tmdb.Genre{{ID: 18, Name: "劇情"}},
	}
	m := env.seedMovie(t, &models.Movie{Title: "[Raws] fc.mkv", ParseStatus: models.ParseStatusFailed})

	item, err := env.enrichment.ApplyTMDbMatch(env.ctx, MediaKindMovie, m.ID, 550)
	require.NoError(t, err)

	got := env.movie(t, m.ID)
	assert.Equal(t, int64(550), got.TMDbID.Int64)
	assert.Equal(t, "鬥陣俱樂部", got.Title)
	assert.Equal(t, "/fc.jpg", got.PosterPath.String)
	assert.Equal(t, int64(139), got.Runtime.Int64)
	assert.Equal(t, models.ParseStatusSuccess, got.ParseStatus)
	assert.Equal(t, string(models.MetadataSourceManual), got.MetadataSource.String,
		"a user-picked match is the user's word — nothing automatic may overwrite it")
	require.NotNil(t, got.Credits, "the detail page reads local credits for manual rows, so the cast must be stored")
	assert.Equal(t, "布萊恩·克蘭斯頓", got.Credits.Cast[0].Name)
	assert.Equal(t, []string{"movie/550"}, env.seeder.fetchArgs)
	assert.Equal(t, []string{m.ID}, env.scopes.calls)

	assert.Equal(t, &EnrichedItem{ID: m.ID, ParseStatus: models.ParseStatusSuccess, Title: "鬥陣俱樂部", TMDbID: 550}, item)
}

func TestApplyTMDbMatch_Series_WritesTheMatchAndReadsBack(t *testing.T) {
	env := newMatchEnv(t)
	env.tmdb.tvResp = &tmdb.TVShowDetails{
		TVShow: tmdb.TVShow{ID: 1396, Name: "絕命毒師", OriginalName: "Breaking Bad", Overview: "化學老師",
			FirstAirDate: "2008-01-20", PosterPath: strptr("/bb.jpg"), VoteAverage: 8.9},
		Genres: []tmdb.Genre{{ID: 18, Name: "劇情"}},
	}
	s := env.seedSeries(t, &models.Series{Title: "breaking bad s01", ParseStatus: models.ParseStatusFailed})

	item, err := env.enrichment.ApplyTMDbMatch(env.ctx, MediaKindSeries, s.ID, 1396)
	require.NoError(t, err)

	got := env.oneSeries(t, s.ID)
	assert.Equal(t, int64(1396), got.TMDbID.Int64)
	assert.Equal(t, "絕命毒師", got.Title)
	assert.Equal(t, "Breaking Bad", got.OriginalTitle.String)
	assert.Equal(t, "/bb.jpg", got.PosterPath.String)
	assert.Equal(t, "2008-01-20", got.FirstAirDate)
	assert.Equal(t, []string{"劇情"}, got.Genres)
	assert.Equal(t, models.ParseStatusSuccess, got.ParseStatus)
	assert.Equal(t, string(models.MetadataSourceManual), got.MetadataSource.String)
	require.NotNil(t, got.Credits)
	assert.Equal(t, []string{"tv/1396"}, env.seeder.fetchArgs)
	assert.Equal(t, []int{1396}, env.tmdb.tvIDs)
	assert.Empty(t, env.tmdb.movieIDs, "a series id must never be looked up as a movie")
	assert.Equal(t, "絕命毒師", item.Title)
}

func TestApplyTMDbMatch_MissingItemIsNotFound(t *testing.T) {
	env := newMatchEnv(t)
	env.tmdb.getMovieDetailsResp = &tmdb.MovieDetails{Movie: tmdb.Movie{ID: 550, Title: "x"}}

	_, err := env.enrichment.ApplyTMDbMatch(env.ctx, MediaKindMovie, "no-such-id", 550)
	assert.ErrorIs(t, err, ErrEnrichItemNotFound)
}

func TestApplyTMDbMatch_TMDbErrorsSurviveWrappingAndLeaveTheRowAlone(t *testing.T) {
	for name, wrap := range map[string]func(error) error{
		"wrapped once":  func(e error) error { return fmt.Errorf("failed to get movie details: %w", e) },
		"wrapped twice": func(e error) error { return fmt.Errorf("cache: %w", fmt.Errorf("client: %w", e)) },
	} {
		t.Run(name, func(t *testing.T) {
			env := newMatchEnv(t)
			env.tmdb.getMovieDetailsErr = wrap(tmdb.NewNotFoundError(550))
			m := env.seedMovie(t, &models.Movie{Title: "[Raws] fc.mkv", ParseStatus: models.ParseStatusFailed})

			_, err := env.enrichment.ApplyTMDbMatch(env.ctx, MediaKindMovie, m.ID, 550)

			var tmdbErr *tmdb.TMDbError
			require.True(t, errors.As(err, &tmdbErr), "the handler maps TMDb errors with errors.As")
			assert.Equal(t, tmdb.ErrCodeNotFound, tmdbErr.Code)
			got := env.movie(t, m.ID)
			assert.Equal(t, "[Raws] fc.mkv", got.Title)
			assert.Equal(t, models.ParseStatusFailed, got.ParseStatus)
		})
	}
}

func TestApplyTMDbMatch_RefusedWhileBatchEnrichmentRuns(t *testing.T) {
	env := newMatchEnv(t)
	m := env.seedMovie(t, &models.Movie{Title: "x.mkv", ParseStatus: models.ParseStatusFailed})
	env.enrichment.isEnriching = true

	_, err := env.enrichment.ApplyTMDbMatch(env.ctx, MediaKindMovie, m.ID, 550)
	assert.ErrorIs(t, err, ErrEnrichmentAlreadyRunning)
	assert.Empty(t, env.tmdb.movieIDs)
}

// ─── AC #4: auto-enrichment never overwrites what the user chose ───────────

func TestBatchEnrichment_ManualRowKeepsItsDataAndRefreshesLocalFacts(t *testing.T) {
	env := newMatchEnv(t)
	dir := t.TempDir()
	mkv := filepath.Join(dir, "Fight.Club.1999.mkv")
	require.NoError(t, os.WriteFile(mkv, []byte("x"), 0o644))
	m := env.seedMovie(t, &models.Movie{
		Title: "我親手選的片", TMDbID: models.NewNullInt64(550), PosterPath: models.NewNullString("/mine.jpg"),
		FilePath: models.NewNullString(mkv), MetadataSource: models.NewNullString(string(models.MetadataSourceManual)),
		ParseStatus: models.ParseStatusSuccess,
	})
	// The file changed: the scanner knocks the row back to pending, and a new
	// sidecar subtitle appeared next to it.
	require.NoError(t, env.movies.UpdateParseStatus(env.ctx, m.ID, models.ParseStatusPending))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Fight.Club.1999.srt"), []byte("1\n00:00:01,000 --> 00:00:02,000\nhi\n"), 0o644))
	env.meta.searchResult = tmdbMovieMatch("999", "完全錯的片")

	_, err := env.enrichment.StartEnrichment(env.ctx)
	require.NoError(t, err)

	got := env.movie(t, m.ID)
	assert.Equal(t, "我親手選的片", got.Title)
	assert.Equal(t, int64(550), got.TMDbID.Int64)
	assert.Equal(t, "/mine.jpg", got.PosterPath.String)
	assert.Equal(t, models.ParseStatusSuccess, got.ParseStatus)
	assert.Zero(t, env.meta.calls, "a manual row is never searched")
	assert.True(t, got.SubtitleTracks.Valid, "the file changed — local analysis still runs")
	assert.Contains(t, got.SubtitleTracks.String, `"format":"srt"`)
}

func TestBatchEnrichment_UserTurnsRowManualMidRun_MatchIsNotWritten(t *testing.T) {
	env := newMatchEnv(t)
	m := env.seedMovie(t, &models.Movie{Title: "Fight.Club.mkv", ParseStatus: models.ParseStatusPending})
	env.meta.searchResult = tmdbMovieMatch("999", "完全錯的片")
	// The batch already holds its snapshot; the user edits while TMDb answers.
	env.meta.onSearch = func(ctx context.Context) {
		row := env.movie(t, m.ID)
		row.Title = "使用者剛存的片名"
		row.MetadataSource = models.NewNullString(string(models.MetadataSourceManual))
		row.ParseStatus = models.ParseStatusSuccess
		require.NoError(t, env.movies.UpdateEnrichedMetadata(ctx, row))
	}

	_, err := env.enrichment.StartEnrichment(env.ctx)
	require.NoError(t, err)

	got := env.movie(t, m.ID)
	assert.Equal(t, "使用者剛存的片名", got.Title)
	assert.False(t, got.TMDbID.Valid, "the stale snapshot's match must not land")
	assert.Equal(t, models.ParseStatusSuccess, got.ParseStatus)
}

func TestBatchEnrichment_UserTurnsRowManualMidRun_FailureIsNotWritten(t *testing.T) {
	env := newMatchEnv(t)
	m := env.seedMovie(t, &models.Movie{Title: "zzqx.mkv", ParseStatus: models.ParseStatusPending})
	env.meta.searchResult = &metadata.SearchResult{} // no match
	env.meta.onSearch = func(ctx context.Context) {
		row := env.movie(t, m.ID)
		row.Title = "使用者剛存的片名"
		row.MetadataSource = models.NewNullString(string(models.MetadataSourceManual))
		row.ParseStatus = models.ParseStatusSuccess
		require.NoError(t, env.movies.UpdateEnrichedMetadata(ctx, row))
	}

	_, _ = env.enrichment.StartEnrichment(env.ctx)

	got := env.movie(t, m.ID)
	assert.Equal(t, "使用者剛存的片名", got.Title)
	assert.Equal(t, models.ParseStatusSuccess, got.ParseStatus, "a stale 'failed' must not overwrite the user's save")
}

func TestBatchEnrichment_ManualSeriesKeepsItsData(t *testing.T) {
	env := newMatchEnv(t)
	s := env.seedSeries(t, &models.Series{
		Title: "我填的影集", TMDbID: models.NewNullInt64(1396),
		MetadataSource: models.NewNullString(string(models.MetadataSourceManual)), ParseStatus: models.ParseStatusPending,
	})
	env.meta.searchResult = &metadata.SearchResult{Source: models.MetadataSourceTMDb,
		Items: []metadata.MetadataItem{{ID: "1", Title: "錯的", MediaType: metadata.MediaTypeTV}}}

	_, err := env.enrichment.StartEnrichment(env.ctx)
	require.NoError(t, err)

	got := env.oneSeries(t, s.ID)
	assert.Equal(t, "我填的影集", got.Title)
	assert.Equal(t, int64(1396), got.TMDbID.Int64)
	assert.Equal(t, models.ParseStatusSuccess, got.ParseStatus)
	assert.Zero(t, env.meta.calls)
}

// ─── AC #2: re-matching one item on demand ─────────────────────────────────

func TestEnrichOne_Movie_ParsesTheFilenameNotTheCurrentTitle(t *testing.T) {
	env := newMatchEnv(t)
	m := env.seedMovie(t, &models.Movie{
		Title: "上次配錯的片名", FilePath: models.NewNullString("/lib/movies/Fight.Club.1999.1080p.mkv"),
		ParseStatus: models.ParseStatusFailed,
	})
	env.meta.searchResult = tmdbMovieMatch("550", "Fight Club")

	item, err := env.enrichment.EnrichOne(env.ctx, MediaKindMovie, m.ID)
	require.NoError(t, err)

	assert.Equal(t, []string{"Fight.Club.1999.1080p.mkv"}, env.parser.inputs)
	assert.Equal(t, models.ParseStatusSuccess, item.ParseStatus)
	assert.Equal(t, "Fight Club", item.Title)
	assert.Equal(t, int64(550), item.TMDbID)
}

func TestEnrichOne_Movie_NoPathFallsBackToTitle_AndNoMatchIsAResult(t *testing.T) {
	env := newMatchEnv(t)
	m := env.seedMovie(t, &models.Movie{Title: "zzqx-e2e-nonsense"})
	env.meta.searchResult = &metadata.SearchResult{}

	item, err := env.enrichment.EnrichOne(env.ctx, MediaKindMovie, m.ID)
	require.NoError(t, err, "ran but found nothing is a result, not an error")

	assert.Equal(t, []string{"zzqx-e2e-nonsense"}, env.parser.inputs)
	assert.Equal(t, models.ParseStatusFailed, item.ParseStatus)
	assert.Equal(t, models.ParseStatusFailed, env.movie(t, m.ID).ParseStatus)
}

func TestEnrichOne_ManualMovie_OnlyRefreshesAndSucceeds(t *testing.T) {
	env := newMatchEnv(t)
	m := env.seedMovie(t, &models.Movie{
		Title: "我填的", MetadataSource: models.NewNullString(string(models.MetadataSourceManual)),
		ParseStatus: models.ParseStatusFailed, // edited before dsr-2b-a: stuck manual+failed
	})

	item, err := env.enrichment.EnrichOne(env.ctx, MediaKindMovie, m.ID)
	require.NoError(t, err)

	assert.Equal(t, models.ParseStatusSuccess, item.ParseStatus)
	assert.Equal(t, "我填的", item.Title)
	assert.Empty(t, env.parser.inputs)
	assert.Zero(t, env.meta.calls)
}

func TestEnrichOne_Series_UsesTheTitleAndSucceeds(t *testing.T) {
	env := newMatchEnv(t)
	s := env.seedSeries(t, &models.Series{Title: "Breaking Bad", ParseStatus: models.ParseStatusFailed})
	env.parser.result = &parser.ParseResult{CleanedTitle: "Breaking Bad"}
	env.meta.searchResult = &metadata.SearchResult{Source: models.MetadataSourceTMDb,
		Items: []metadata.MetadataItem{{ID: "1396", Title: "Breaking Bad", TitleZhTW: "絕命毒師", MediaType: metadata.MediaTypeTV}}}

	item, err := env.enrichment.EnrichOne(env.ctx, MediaKindSeries, s.ID)
	require.NoError(t, err)

	assert.Equal(t, []string{"Breaking Bad"}, env.parser.inputs)
	assert.Equal(t, models.ParseStatusSuccess, item.ParseStatus)
	assert.Equal(t, "絕命毒師", item.Title)
	assert.Equal(t, int64(1396), item.TMDbID)
}

func TestEnrichOne_NotFound(t *testing.T) {
	env := newMatchEnv(t)
	_, err := env.enrichment.EnrichOne(env.ctx, MediaKindMovie, "nope")
	assert.ErrorIs(t, err, ErrEnrichItemNotFound)
	_, err = env.enrichment.EnrichOne(env.ctx, MediaKindSeries, "nope")
	assert.ErrorIs(t, err, ErrEnrichItemNotFound)
}

func TestEnrichOne_RefusedWhileBatchRuns_AndDoesNotTakeTheBatchLock(t *testing.T) {
	env := newMatchEnv(t)
	m := env.seedMovie(t, &models.Movie{Title: "x.mkv", ParseStatus: models.ParseStatusPending})

	env.enrichment.isEnriching = true
	_, err := env.enrichment.EnrichOne(env.ctx, MediaKindMovie, m.ID)
	assert.ErrorIs(t, err, ErrEnrichmentAlreadyRunning)
	env.enrichment.isEnriching = false

	// While a single re-match runs, a post-scan batch must still start (its
	// work would otherwise be silently dropped), and cancelling it must not
	// panic on a nil channel.
	env.meta.searchResult = &metadata.SearchResult{}
	env.meta.onSearch = func(context.Context) {
		assert.False(t, env.enrichment.IsEnrichmentActive(), "single re-match must not hold the batch flag")
		assert.Error(t, env.enrichment.CancelEnrichment(), "nothing batch-shaped to cancel")
	}
	_, err = env.enrichment.EnrichOne(env.ctx, MediaKindMovie, m.ID)
	require.NoError(t, err)
}

func TestEnrichOne_RealDeadlineIsReportedAsTimeout(t *testing.T) {
	env := newMatchEnv(t)
	m := env.seedMovie(t, &models.Movie{Title: "slow.mkv", ParseStatus: models.ParseStatusPending})
	// Mirrors the orchestrator: on cancel it returns (nil, status) with NO
	// error, so a timeout can only be seen through ctx.Err().
	env.meta.onSearch = func(ctx context.Context) { <-ctx.Done() }
	env.meta.searchResult = nil

	ctx, cancel := context.WithTimeout(env.ctx, 50*time.Millisecond)
	defer cancel()
	_, err := env.enrichment.EnrichOne(ctx, MediaKindMovie, m.ID)

	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.NotErrorIs(t, err, ErrEnrichPersist,
		"reported as a timeout, not as the write that failed on the expired context")
}

// A write failure must surface as an error — never a 200 carrying the row's
// untouched 'pending'.
type failingWriteMovieRepo struct {
	mockMovieRepoForNFO
	row *models.Movie
}

func (r *failingWriteMovieRepo) FindByID(context.Context, string) (*models.Movie, error) {
	cp := *r.row
	return &cp, nil
}

func (r *failingWriteMovieRepo) UpdateEnrichedMetadata(context.Context, *models.Movie) error {
	return errors.New("database is locked")
}

func TestEnrichOne_WriteFailureIsAPersistError(t *testing.T) {
	repo := &failingWriteMovieRepo{row: &models.Movie{ID: "m-1", Title: "x.mkv", ParseStatus: models.ParseStatusPending}}
	svc := NewEnrichmentService(repo,
		&mockPQParserService{result: &parser.ParseResult{Status: parser.ParseStatusSuccess, CleanedTitle: "X"}},
		&mockPQMetadataService{searchResult: tmdbMovieMatch("550", "Fight Club")}, nil, nil, nil, nil, nil)

	_, err := svc.EnrichOne(context.Background(), MediaKindMovie, "m-1")
	assert.ErrorIs(t, err, ErrEnrichPersist)

	svc.metadataService = &mockPQMetadataService{searchResult: &metadata.SearchResult{}} // no match → failed write
	_, err = svc.EnrichOne(context.Background(), MediaKindMovie, "m-1")
	assert.ErrorIs(t, err, ErrEnrichPersist)
}

// ─── /ship adversarial review follow-ups ───────────────────────────────────

// CR #1: on the failure path the takeover check must come AFTER local
// analysis (the probe can take seconds on a NAS disk that has to spin up), so
// a user save made during the probe is still seen. The recording repo looks
// at the in-memory row at the moment of the check: the sidecar subtitle found
// by local analysis must already be on it.
type orderRecordingMovieRepo struct {
	mockMovieRepoForNFO
	watched          *models.Movie
	analysedAtCheck  bool
	checkedBeforeRun bool
}

func (r *orderRecordingMovieRepo) FindByID(context.Context, string) (*models.Movie, error) {
	if r.watched != nil {
		r.analysedAtCheck = r.watched.SubtitleTracks.Valid
		r.checkedBeforeRun = true
	}
	return nil, nil
}

func TestPersistFailedWithLocalAnalysis_TakeoverCheckedAfterTheProbe(t *testing.T) {
	repo := &orderRecordingMovieRepo{}
	s := &EnrichmentService{movieRepo: repo, logger: slog.Default()}
	movie := newMovieWithSidecar(t)
	repo.watched = movie

	require.NoError(t, s.persistFailedWithLocalAnalysis(context.Background(), movie))

	require.True(t, repo.checkedBeforeRun, "the takeover check must run")
	assert.True(t, repo.analysedAtCheck, "local analysis must already have run when the check happens")
}

// CR #2: with a real ffprobe and a codec already stored, the manual refresh
// used to do nothing (applyFFprobeTechInfo skips once a codec is set) — the
// test above passed only because its env had no ffprobe. A fake ffprobe on
// PATH proves the file is re-probed.
func fakeFFprobeOnPath(t *testing.T, codec string) *FFprobeService {
	t.Helper()
	dir := t.TempDir()
	script := "#!/bin/sh\ncat <<'JSON'\n" +
		`{"streams":[{"index":0,"codec_type":"video","codec_name":"` + codec + `","width":3840,"height":2160}],"format":{"duration":"120.0"}}` +
		"\nJSON\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ffprobe"), []byte(script), 0o755))
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	svc := NewFFprobeService(1, 5*time.Second, nil)
	require.True(t, svc.IsAvailable())
	return svc
}

func TestBatchEnrichment_ManualRowIsReprobedEvenWithACodecStored(t *testing.T) {
	env := newMatchEnv(t)
	env.enrichment.ffprobeService = fakeFFprobeOnPath(t, "hevc")
	dir := t.TempDir()
	mkv := filepath.Join(dir, "Movie.mkv")
	require.NoError(t, os.WriteFile(mkv, []byte("x"), 0o644))
	m := env.seedMovie(t, &models.Movie{
		Title: "我親手選的片", FilePath: models.NewNullString(mkv), VideoCodec: models.NewNullString("h264"),
		MetadataSource: models.NewNullString(string(models.MetadataSourceManual)), ParseStatus: models.ParseStatusPending,
	})
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Movie.srt"), []byte("1\n00:00:01,000 --> 00:00:02,000\nhi\n"), 0o644))

	_, err := env.enrichment.StartEnrichment(env.ctx)
	require.NoError(t, err)

	got := env.movie(t, m.ID)
	assert.Equal(t, "H.265", got.VideoCodec.String, "the changed file is probed again (was h264)")
	assert.Contains(t, got.SubtitleTracks.String, `"format":"srt"`)
	assert.Equal(t, "我親手選的片", got.Title)
}

// CR #3: a row handled after the batch loaded its snapshot (here: matched by
// a single re-match) is skipped, not re-parsed from its new title.
func TestBatchEnrichment_SkipsRowsHandledSinceTheSnapshot(t *testing.T) {
	env := newMatchEnv(t)
	a := env.seedMovie(t, &models.Movie{Title: "a.mkv", ParseStatus: models.ParseStatusPending})
	b := env.seedMovie(t, &models.Movie{Title: "b.mkv", ParseStatus: models.ParseStatusPending})
	env.meta.searchResult = &metadata.SearchResult{}
	env.meta.onSearch = func(ctx context.Context) {
		if env.meta.calls != 1 {
			return
		}
		// Whichever row is searched first, the OTHER one gets matched meanwhile.
		other := b.ID
		if env.parser.inputs[len(env.parser.inputs)-1] == "b.mkv" {
			other = a.ID
		}
		row := env.movie(t, other)
		row.Title, row.TMDbID = "鬥陣俱樂部", models.NewNullInt64(550)
		row.MetadataSource = models.NewNullString(string(models.MetadataSourceTMDb))
		row.ParseStatus = models.ParseStatusSuccess
		require.NoError(t, env.movies.UpdateEnrichedMetadata(ctx, row))
	}

	result, err := env.enrichment.StartEnrichment(env.ctx)
	require.NoError(t, err)

	assert.Equal(t, 1, env.meta.calls, "the handled row is not searched again")
	assert.Equal(t, 1, result.Skipped)
	for _, id := range []string{a.ID, b.ID} {
		if row := env.movie(t, id); row.TMDbID.Valid {
			assert.Equal(t, "鬥陣俱樂部", row.Title)
			assert.Equal(t, models.ParseStatusSuccess, row.ParseStatus)
		}
	}
}

// CR #4: the deadline can expire after the row write, during the glossary
// resolve that follows it. The re-match did succeed and must say so.
type blockingScopes struct{}

func (blockingScopes) Resolve(ctx context.Context, _ string) (string, error) {
	<-ctx.Done()
	return "", ctx.Err()
}

func TestEnrichOne_DeadlineAfterTheWriteIsStillASuccess(t *testing.T) {
	env := newMatchEnv(t)
	env.enrichment.SetGlossarySeeder(env.seeder, blockingScopes{})
	m := env.seedMovie(t, &models.Movie{Title: "Fight.Club.mkv", ParseStatus: models.ParseStatusPending})
	env.meta.searchResult = tmdbMovieMatch("550", "Fight Club")

	ctx, cancel := context.WithTimeout(env.ctx, 300*time.Millisecond)
	defer cancel()
	item, err := env.enrichment.EnrichOne(ctx, MediaKindMovie, m.ID)

	require.NoError(t, err)
	assert.Equal(t, models.ParseStatusSuccess, item.ParseStatus)
	assert.Equal(t, int64(550), item.TMDbID)
}

// CR #6: when the new match brings no cast (fetch failed / no credits
// client), the previous wrong match's cast must not stay on a manual row —
// the detail page reads local credits for manual rows.
func TestApplyTMDbMatch_ClearsTheOldCastWhenNoNewCastArrives(t *testing.T) {
	env := newMatchEnv(t)
	env.seeder.fetchErr = errors.New("tmdb credits: timeout")
	env.tmdb.getMovieDetailsResp = &tmdb.MovieDetails{Movie: tmdb.Movie{ID: 550, Title: "鬥陣俱樂部"}}
	m := env.seedMovie(t, &models.Movie{Title: "錯配的片", ParseStatus: models.ParseStatusSuccess})
	require.NoError(t, env.movies.UpdateCredits(env.ctx, m.ID, &models.Credits{Cast: []models.CastMember{{Name: "錯的演員"}}}))
	require.NotNil(t, env.movie(t, m.ID).Credits)

	_, err := env.enrichment.ApplyTMDbMatch(env.ctx, MediaKindMovie, m.ID, 550)
	require.NoError(t, err)

	assert.Nil(t, env.movie(t, m.ID).Credits, "the wrong film's cast is gone; the page falls back to live TMDb credits")
}

// CR #7: service-level apply failures (AC #7 lists them explicitly).
func TestApplyTMDbMatch_TMDbTimeoutSurvivesWrapping(t *testing.T) {
	env := newMatchEnv(t)
	env.tmdb.getMovieDetailsErr = fmt.Errorf("failed to get movie details: %w", tmdb.NewTimeoutError(nil))
	m := env.seedMovie(t, &models.Movie{Title: "x.mkv", ParseStatus: models.ParseStatusFailed})

	_, err := env.enrichment.ApplyTMDbMatch(env.ctx, MediaKindMovie, m.ID, 550)

	var tmdbErr *tmdb.TMDbError
	require.True(t, errors.As(err, &tmdbErr))
	assert.Equal(t, tmdb.ErrCodeTimeout, tmdbErr.Code)
}

func TestApplyTMDbMatch_WriteFailureIsAPersistError(t *testing.T) {
	repo := &failingWriteMovieRepo{row: &models.Movie{ID: "m-1", Title: "x.mkv", ParseStatus: models.ParseStatusFailed}}
	svc := NewEnrichmentService(repo, nil, nil, nil,
		&mockTMDbServiceForNFO{getMovieDetailsResp: &tmdb.MovieDetails{Movie: tmdb.Movie{ID: 550, Title: "鬥陣俱樂部"}}},
		nil, nil, nil)

	_, err := svc.ApplyTMDbMatch(context.Background(), MediaKindMovie, "m-1", 550)
	assert.ErrorIs(t, err, ErrEnrichPersist)
}
