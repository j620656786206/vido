package services

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/tmdb"
)

// ─── disc-2026-10-rematch-keeps-old-season-rows ─────────────────────────────
//
// 將夜 on the NAS (2026-10-27): the show was re-matched to the 2026 animation
// but its season 1 row still said 60 episodes — the 2018 drama's.

// everNight2026 is the new show: one 20-episode season, no season poster or
// overview on TMDb.
func everNight2026() *tmdb.TVShowDetails {
	return &tmdb.TVShowDetails{
		TVShow:  tmdb.TVShow{ID: 282136, Name: "將夜", FirstAirDate: "2026-04-23"},
		Seasons: []tmdb.Season{{ID: 400001, SeasonNumber: 1, Name: "第 1 季", EpisodeCount: 20, AirDate: strptr("2026-04-23")}},
	}
}

// oldSeason is a row filled from the 2018 drama.
func oldSeason(seriesID string, number int) *models.Season {
	return &models.Season{
		ID:           fmt.Sprintf("%s-s%d", seriesID, number),
		SeriesID:     seriesID,
		SeasonNumber: number,
		TMDbID:       models.NewNullInt64(int64(111783 + number)),
		Name:         models.NewNullString("將夜 第一季"),
		Overview:     models.NewNullString("寧缺與桑桑"),
		PosterPath:   models.NewNullString("/drama.jpg"),
		AirDate:      models.NewNullString("2018-10-31"),
		EpisodeCount: models.NewNullInt64(60),
		VoteAverage:  models.NewNullFloat64(7.4),
	}
}

type seasonsEnv struct {
	ctx        context.Context
	series     *repository.SeriesRepository
	seasons    *repository.SeasonRepository
	tmdb       *matchTMDbFake
	enrichment *EnrichmentService
}

func newSeasonsEnv(t *testing.T) *seasonsEnv {
	t.Helper()
	db := setupTestDB(t)
	env := &seasonsEnv{
		ctx:     context.Background(),
		series:  repository.NewSeriesRepository(db),
		seasons: repository.NewSeasonRepository(db),
		tmdb:    &matchTMDbFake{tvResp: everNight2026()},
	}
	env.enrichment = NewEnrichmentService(repository.NewMovieRepository(db), &mockPQParserService{}, &mockPQMetadataService{},
		nil, env.tmdb, nil, nil, nil)
	env.enrichment.SetSeriesRepo(env.series)
	env.enrichment.SetSeasonRepository(env.seasons)
	return env
}

// seedDrama stores 將夜 matched to the 2018 drama with seasons 1 and 2.
func (e *seasonsEnv) seedDrama(t *testing.T) *models.Series {
	t.Helper()
	s := &models.Series{
		ID:             fmt.Sprintf("s-%d", time.Now().UnixNano()),
		Title:          "將夜",
		TMDbID:         models.NewNullInt64(83612),
		MetadataSource: models.NewNullString(string(models.MetadataSourceTMDb)),
		ParseStatus:    models.ParseStatusSuccess,
	}
	require.NoError(t, e.series.Create(e.ctx, s))
	require.NoError(t, e.seasons.Create(e.ctx, oldSeason(s.ID, 1)))
	require.NoError(t, e.seasons.Create(e.ctx, oldSeason(s.ID, 2)))
	return s
}

func (e *seasonsEnv) season(t *testing.T, seriesID string, number int) *models.Season {
	t.Helper()
	se, err := e.seasons.FindBySeriesAndNumber(e.ctx, seriesID, number)
	require.NoError(t, err)
	require.NotNil(t, se, "the season row must never be deleted")
	return se
}

func TestApplyTMDbMatch_NewShowReplacesTheSeasonRows(t *testing.T) {
	env := newSeasonsEnv(t)
	s := env.seedDrama(t)

	_, err := env.enrichment.ApplyTMDbMatch(env.ctx, MediaKindSeries, s.ID, 282136)
	require.NoError(t, err)

	s1 := env.season(t, s.ID, 1)
	assert.Equal(t, int64(400001), s1.TMDbID.Int64)
	assert.Equal(t, "第 1 季", s1.Name.String)
	assert.Equal(t, int64(20), s1.EpisodeCount.Int64, "the 2026 show's 20, not the drama's 60")
	assert.Equal(t, "2026-04-23", s1.AirDate.String)
	assert.False(t, s1.PosterPath.Valid, "no season poster on TMDb: the drama's must not stay")
	assert.False(t, s1.Overview.Valid)
	assert.False(t, s1.VoteAverage.Valid)

	s2 := env.season(t, s.ID, 2)
	assert.False(t, s2.TMDbID.Valid, "the new show has no season 2")
	assert.False(t, s2.Name.Valid)
	assert.False(t, s2.Overview.Valid)
	assert.False(t, s2.PosterPath.Valid)
	assert.False(t, s2.AirDate.Valid)
	assert.False(t, s2.EpisodeCount.Valid, "劇集數未知, not the drama's 60")
	assert.False(t, s2.VoteAverage.Valid)
}

func TestApplyTMDbMatch_SameShowKeepsTheSeasonRows(t *testing.T) {
	env := newSeasonsEnv(t)
	env.tmdb.tvResp.ID = 83612
	s := env.seedDrama(t)

	_, err := env.enrichment.ApplyTMDbMatch(env.ctx, MediaKindSeries, s.ID, 83612)
	require.NoError(t, err)

	assert.Equal(t, oldSeason(s.ID, 1).EpisodeCount, env.season(t, s.ID, 1).EpisodeCount,
		"the same show: the exact count written when the season was opened stays")
	assert.Equal(t, "/drama.jpg", env.season(t, s.ID, 2).PosterPath.String)
}

func TestResetSeasonsForNewShow_TMDbFailureLeavesTheRows(t *testing.T) {
	env := newSeasonsEnv(t)
	env.tmdb.tvErr = errors.New("TMDb down")
	s := env.seedDrama(t)

	env.enrichment.resetSeasonsForNewShow(env.ctx, s.ID, models.NewNullInt64(83612), models.NewNullInt64(282136))

	assert.Equal(t, int64(60), env.season(t, s.ID, 1).EpisodeCount.Int64, "one network error must not wipe the seasons")
	assert.Equal(t, "/drama.jpg", env.season(t, s.ID, 2).PosterPath.String)
}

// recordingSeasons stands in for the season repository on the mocked
// enrichment paths (search, tvshow.nfo).
type recordingSeasons struct {
	rows    []models.Season
	updated []models.Season
}

func (r *recordingSeasons) FindBySeriesID(context.Context, string) ([]models.Season, error) {
	return append([]models.Season(nil), r.rows...), nil
}

func (r *recordingSeasons) Update(_ context.Context, se *models.Season) error {
	r.updated = append(r.updated, *se)
	return nil
}

func TestEnrichSeries_EveryRematchPathResetsTheSeasons(t *testing.T) {
	cases := []struct {
		name   string
		folder string
		nfo    string
	}{
		{"tvshow.nfo id", "將夜", `<tvshow><uniqueid default="true" type="tmdb">282136</uniqueid></tvshow>`},
		{"search, picked by year", "將夜 (2026)", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := newSeriesIdentityEnv(t, "將夜")
			env.tmdb.tvDetails = everNight2026()
			seasons := &recordingSeasons{rows: []models.Season{*oldSeason("series-1", 1)}}
			env.svc.SetSeasonRepository(seasons)
			s := seriesAt(env.showDir(t, tc.folder, tc.nfo), "將夜")
			s.TMDbID = models.NewNullInt64(83612) // the wrong earlier match

			require.NoError(t, env.svc.enrichSeries(context.Background(), s))

			require.Len(t, seasons.updated, 1)
			assert.Equal(t, int64(20), seasons.updated[0].EpisodeCount.Int64)
			assert.Equal(t, int64(400001), seasons.updated[0].TMDbID.Int64)
		})
	}
}

func TestEnrichSeries_SameShowAgainDoesNotTouchTheSeasons(t *testing.T) {
	env := newSeriesIdentityEnv(t, "將夜")
	env.tmdb.tvDetails = everNight2026()
	seasons := &recordingSeasons{rows: []models.Season{*oldSeason("series-1", 1)}}
	env.svc.SetSeasonRepository(seasons)
	s := seriesAt(env.showDir(t, "將夜 (2026)", ""), "將夜")
	s.TMDbID = models.NewNullInt64(282136) // already the right show

	require.NoError(t, env.svc.enrichSeries(context.Background(), s))

	assert.Empty(t, seasons.updated)
}
