package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/tmdb"
)

// disc-2026-10-season-episode-count-unknown AC #1/#3/#4.

var (
	_ SeasonDetailsRepo = (*repository.SeasonRepository)(nil)
	_ TVDetailsFetcher  = (*TMDbService)(nil)
)

type fakeSeasonRepo struct {
	gaps    []repository.SeasonDetailsGap
	seasons map[string][]models.Season
	updates []models.Season
}

func (f *fakeSeasonRepo) FindSeriesNeedingSeasonDetails(context.Context, int) ([]repository.SeasonDetailsGap, error) {
	var out []repository.SeasonDetailsGap
	for _, g := range f.gaps {
		for _, se := range f.seasons[g.SeriesID] {
			if !se.EpisodeCount.Valid || !se.TMDbID.Valid {
				out = append(out, g)
				break
			}
		}
	}
	return out, nil
}
func (f *fakeSeasonRepo) FindBySeriesID(_ context.Context, id string) ([]models.Season, error) {
	return append([]models.Season(nil), f.seasons[id]...), nil
}
func (f *fakeSeasonRepo) Update(_ context.Context, se *models.Season) error {
	f.updates = append(f.updates, *se)
	rows := f.seasons[se.SeriesID]
	for i := range rows {
		if rows[i].ID == se.ID {
			rows[i] = *se
		}
	}
	return nil
}

type fakeTVDetails struct {
	byID  map[int]*tmdb.TVShowDetails
	err   error
	calls []int
}

func (f *fakeTVDetails) GetTVShowDetails(_ context.Context, id int) (*tmdb.TVShowDetails, error) {
	f.calls = append(f.calls, id)
	if f.err != nil {
		return nil, f.err
	}
	return f.byID[id], nil
}

func seeTMDb() *tmdb.TVShowDetails {
	return &tmdb.TVShowDetails{Seasons: []tmdb.Season{
		{ID: 1001, SeasonNumber: 1, Name: "第 1 季", EpisodeCount: 8, PosterPath: strPtr("/s1.jpg"), AirDate: strPtr("2019-11-01")},
		{ID: 1002, SeasonNumber: 2, Name: "第 2 季", EpisodeCount: 8, PosterPath: strPtr("/s2.jpg"), AirDate: strPtr("2021-08-27")},
	}}
}

func TestSeasonDetailsBackfill_FillsEmptySeasons(t *testing.T) {
	repo := &fakeSeasonRepo{
		gaps: []repository.SeasonDetailsGap{{SeriesID: "see", TMDbID: 85421}},
		seasons: map[string][]models.Season{"see": {
			{ID: "s1", SeriesID: "see", SeasonNumber: 1},
			{ID: "s2", SeriesID: "see", SeasonNumber: 2},
		}},
	}
	fetch := &fakeTVDetails{byID: map[int]*tmdb.TVShowDetails{85421: seeTMDb()}}
	svc := NewSeasonDetailsBackfillService(repo, fetch, nil)

	res := svc.Run(context.Background())
	assert.Equal(t, SeasonDetailsBackfillResult{Series: 1, Seasons: 2}, res)
	s1 := repo.seasons["see"][0]
	assert.EqualValues(t, 8, s1.EpisodeCount.Int64)
	assert.Equal(t, "第 1 季", s1.Name.String)
	assert.Equal(t, "/s1.jpg", s1.PosterPath.String)
	assert.Equal(t, "2019-11-01", s1.AirDate.String)
	assert.EqualValues(t, 1001, s1.TMDbID.Int64)

	// Whole now → off the worklist; a second pass asks TMDb nothing.
	svc.Run(context.Background())
	assert.Len(t, fetch.calls, 1)
}

func TestSeasonDetailsBackfill_SeasonTMDbLacksIsSkippedAndNotRetried(t *testing.T) {
	repo := &fakeSeasonRepo{
		gaps: []repository.SeasonDetailsGap{{SeriesID: "see", TMDbID: 85421}},
		seasons: map[string][]models.Season{"see": {
			{ID: "s1", SeriesID: "see", SeasonNumber: 1},
			{ID: "s9", SeriesID: "see", SeasonNumber: 9}, // a mis-numbered folder
		}},
	}
	fetch := &fakeTVDetails{byID: map[int]*tmdb.TVShowDetails{85421: seeTMDb()}}
	svc := NewSeasonDetailsBackfillService(repo, fetch, nil)

	assert.Equal(t, 1, svc.Run(context.Background()).Seasons)
	assert.False(t, repo.seasons["see"][1].EpisodeCount.Valid)
	svc.Run(context.Background())
	assert.Len(t, fetch.calls, 1, "still incomplete — not re-fetched every scan")
}

func TestSeasonDetailsBackfill_FailureIsNotRetriedThisProcess(t *testing.T) {
	repo := &fakeSeasonRepo{
		gaps:    []repository.SeasonDetailsGap{{SeriesID: "x", TMDbID: 1}},
		seasons: map[string][]models.Season{"x": {{ID: "s1", SeriesID: "x", SeasonNumber: 1}}},
	}
	fetch := &fakeTVDetails{err: errors.New("TMDb 503")}
	svc := NewSeasonDetailsBackfillService(repo, fetch, nil)
	assert.Equal(t, 1, svc.Run(context.Background()).Failed)
	svc.Run(context.Background())
	assert.Len(t, fetch.calls, 1)
	assert.Empty(t, repo.updates)
}

func TestMergeSeasonDetails_NeverBlanksAStoredValue(t *testing.T) {
	se := models.Season{Name: models.NewNullString("第 1 季"), PosterPath: models.NewNullString("/old.jpg"),
		EpisodeCount: models.NewNullInt64(8), TMDbID: models.NewNullInt64(1001)}
	assert.False(t, MergeSeasonDetails(&se, 0, "", "", nil, nil, 0), "TMDb gave nothing — nothing changes")
	assert.Equal(t, "/old.jpg", se.PosterPath.String)

	assert.True(t, MergeSeasonDetails(&se, 1001, "第 1 季", "", strPtr("/new.jpg"), nil, 8))
	assert.Equal(t, "/new.jpg", se.PosterPath.String)
	assert.False(t, MergeSeasonDetails(&se, 1001, "第 1 季", "", strPtr("/new.jpg"), nil, 8), "same values — no write")
}

// AC #2: opening a season keeps what TMDb just said about it.
func TestSeriesService_GetSeasonEpisodes_RemembersSeasonDetails(t *testing.T) {
	series := newSeasonSeries(t, 85421)
	details := &tmdb.SeasonDetails{ID: 1001, SeasonNumber: 1, Name: "第 1 季", PosterPath: strPtr("/s1.jpg"),
		AirDate: strPtr("2019-11-01"), Episodes: make([]tmdb.EpisodeInfo, 8)}
	for i := range details.Episodes {
		details.Episodes[i].EpisodeNumber = i + 1
	}
	svc := newSeasonServiceUnderTest(series, nil, nil, details, nil)
	seasons := &stubSeasonRepo{row: &models.Season{ID: "s1", SeriesID: "series-1", SeasonNumber: 1}}
	svc.SetSeasonRepo(seasons)

	resp, err := svc.GetSeasonEpisodes(context.Background(), "series-1", 1)
	require.NoError(t, err)
	require.Len(t, seasons.updates, 1)
	assert.EqualValues(t, 8, seasons.updates[0].EpisodeCount.Int64)
	assert.Equal(t, 8, resp.Season.EpisodeCount, "the header in this response has the count too")
	assert.Equal(t, "/s1.jpg", resp.Season.PosterPath)

	_, err = svc.GetSeasonEpisodes(context.Background(), "series-1", 1)
	require.NoError(t, err)
	assert.Len(t, seasons.updates, 1, "unchanged — no second write")
}

type stubSeasonRepo struct {
	repository.SeasonRepositoryInterface
	row     *models.Season
	updates []models.Season
}

func (s *stubSeasonRepo) FindBySeriesAndNumber(context.Context, string, int) (*models.Season, error) {
	cp := *s.row
	return &cp, nil
}
func (s *stubSeasonRepo) Update(_ context.Context, se *models.Season) error {
	s.updates = append(s.updates, *se)
	*s.row = *se
	return nil
}

// CR M1: an incomplete series rests, it is not forgotten — after triedFor it
// is asked again (an unaired season airs, a TMDb outage passes).
func TestSeasonDetailsBackfill_RestingSeriesIsAskedAgainLater(t *testing.T) {
	repo := &fakeSeasonRepo{
		gaps:    []repository.SeasonDetailsGap{{SeriesID: "x", TMDbID: 1}},
		seasons: map[string][]models.Season{"x": {{ID: "s1", SeriesID: "x", SeasonNumber: 1}}},
	}
	fetch := &fakeTVDetails{err: errors.New("TMDb 503")}
	svc := NewSeasonDetailsBackfillService(repo, fetch, nil)
	clock := time.Date(2026, 10, 8, 10, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return clock }

	svc.Run(context.Background())
	clock = clock.Add(triedFor - time.Minute)
	svc.Run(context.Background())
	assert.Len(t, fetch.calls, 1, "still resting")

	clock = clock.Add(2 * time.Minute)
	fetch.err = nil
	fetch.byID = map[int]*tmdb.TVShowDetails{1: {Seasons: []tmdb.Season{{ID: 9, SeasonNumber: 1, EpisodeCount: 10}}}}
	assert.Equal(t, 1, svc.Run(context.Background()).Seasons)
	assert.Len(t, fetch.calls, 2)
}

// CR L3: the series-level count only fills a gap; it never overwrites the
// exact count the season's own episode list wrote.
func TestSeasonDetailsBackfill_SeriesCountOnlyFillsAGap(t *testing.T) {
	repo := &fakeSeasonRepo{
		gaps: []repository.SeasonDetailsGap{{SeriesID: "see", TMDbID: 85421}},
		seasons: map[string][]models.Season{"see": {
			{ID: "s1", SeriesID: "see", SeasonNumber: 1, EpisodeCount: models.NewNullInt64(9)}, // from the season list
			{ID: "s2", SeriesID: "see", SeasonNumber: 2},
		}},
	}
	fetch := &fakeTVDetails{byID: map[int]*tmdb.TVShowDetails{85421: seeTMDb()}}
	NewSeasonDetailsBackfillService(repo, fetch, nil).Run(context.Background())
	assert.EqualValues(t, 9, repo.seasons["see"][0].EpisodeCount.Int64)
	assert.EqualValues(t, 8, repo.seasons["see"][1].EpisodeCount.Int64)
}

func TestSeasonDetailsBackfill_TriggerCoalesces(t *testing.T) {
	repo := &fakeSeasonRepo{
		gaps:    []repository.SeasonDetailsGap{{SeriesID: "see", TMDbID: 85421}},
		seasons: map[string][]models.Season{"see": {{ID: "s1", SeriesID: "see", SeasonNumber: 1}}},
	}
	fetch := &fakeTVDetails{byID: map[int]*tmdb.TVShowDetails{85421: seeTMDb()}}
	svc := NewSeasonDetailsBackfillService(repo, fetch, nil)
	ctx := context.Background()
	svc.Trigger(ctx)
	svc.Trigger(ctx)
	svc.Trigger(ctx)
	require.Eventually(t, func() bool {
		svc.mu.Lock()
		defer svc.mu.Unlock()
		return !svc.running
	}, 2*time.Second, 10*time.Millisecond)
	assert.Len(t, fetch.calls, 1, "the queued re-run finds nothing left to fill")
}
