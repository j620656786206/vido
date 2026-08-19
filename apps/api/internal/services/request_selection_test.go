package services

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/tmdb"
)

// --- 13-2a AC #1: canonicalization ---

func TestCanonicalizeSelection(t *testing.T) {
	t.Run("empty selection is the whole-title request", func(t *testing.T) {
		sel, err := canonicalizeSelection(models.RequestMediaTypeTV, nil, nil)
		require.NoError(t, err)
		assert.Nil(t, sel)
	})

	t.Run("movie with any selection is rejected", func(t *testing.T) {
		_, err := canonicalizeSelection(models.RequestMediaTypeMovie, []int{1}, nil)
		assert.ErrorIs(t, err, ErrRequestInvalidSelection)
	})

	t.Run("seasons are sorted and deduped", func(t *testing.T) {
		sel, err := canonicalizeSelection(models.RequestMediaTypeTV, []int{3, 1, 3}, nil)
		require.NoError(t, err)
		assert.Equal(t, []int{1, 3}, sel.Seasons)
	})

	t.Run("episodes are sorted and deduped per season", func(t *testing.T) {
		sel, err := canonicalizeSelection(models.RequestMediaTypeTV, nil, map[string][]int{"2": {6, 5, 6}})
		require.NoError(t, err)
		assert.Equal(t, map[int][]int{2: {5, 6}}, sel.Episodes)
	})

	t.Run("a season on both sides is rejected", func(t *testing.T) {
		_, err := canonicalizeSelection(models.RequestMediaTypeTV, []int{2}, map[string][]int{"2": {1}})
		assert.ErrorIs(t, err, ErrRequestInvalidSelection)
	})

	t.Run("malformed inputs are rejected", func(t *testing.T) {
		cases := []struct {
			name     string
			seasons  []int
			episodes map[string][]int
		}{
			{"negative season", []int{-1}, nil},
			{"non-numeric season key", nil, map[string][]int{"x": {1}}},
			{"empty episode list", nil, map[string][]int{"2": {}}},
			{"non-positive episode", nil, map[string][]int{"2": {0}}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := canonicalizeSelection(models.RequestMediaTypeTV, tc.seasons, tc.episodes)
				assert.ErrorIs(t, err, ErrRequestInvalidSelection)
			})
		}
	})
}

// --- 13-2a AC #1: canonical JSON columns roundtrip ---

func TestSelectionColumnsRoundtrip(t *testing.T) {
	t.Run("whole title stays NULL/NULL — byte-identical to pre-13-2a rows", func(t *testing.T) {
		seasons, episodes, err := selectionColumns(nil)
		require.NoError(t, err)
		assert.False(t, seasons.Valid)
		assert.False(t, episodes.Valid)

		sel, err := parseSelectionColumns(seasons, episodes)
		require.NoError(t, err)
		assert.Nil(t, sel)
	})

	t.Run("mixed selection roundtrips through the canonical JSON", func(t *testing.T) {
		in := &RequestSelection{Seasons: []int{1, 3}, Episodes: map[int][]int{2: {5, 6}}}
		seasons, episodes, err := selectionColumns(in)
		require.NoError(t, err)
		assert.Equal(t, "[1,3]", seasons.String)
		assert.Equal(t, `{"2":[5,6]}`, episodes.String)

		out, err := parseSelectionColumns(seasons, episodes)
		require.NoError(t, err)
		assert.Equal(t, in, out)
	})

	t.Run("malformed stored JSON is a hard error, never fail-soft", func(t *testing.T) {
		_, err := parseSelectionColumns(models.NewNullString("not-json"), models.NullString{})
		require.Error(t, err, "acting on a half-parsed selection could fetch the wrong episodes")
	})
}

// --- 13-2a AC #3: episode-level owned guard ---

func treeTVDetails() *tmdb.TVShowDetails {
	d := &tmdb.TVShowDetails{}
	d.Name = "怪奇物語"
	d.Seasons = []tmdb.Season{
		{SeasonNumber: 1, EpisodeCount: 2},
		{SeasonNumber: 2, EpisodeCount: 3},
	}
	return d
}

func TestCheckSelectionOwnership(t *testing.T) {
	owned := map[int][]int{1: {1, 2}, 2: {1}} // S1 fully owned, S2E1 owned

	t.Run("no overlap is allowed — the point of partial requests", func(t *testing.T) {
		sel := &RequestSelection{Episodes: map[int][]int{2: {2, 3}}}
		assert.NoError(t, checkSelectionOwnership(sel, owned, treeTVDetails()))
	})

	t.Run("entirely-owned selection maps to already-in-library", func(t *testing.T) {
		sel := &RequestSelection{Seasons: []int{1}, Episodes: map[int][]int{2: {1}}}
		err := checkSelectionOwnership(sel, owned, treeTVDetails())
		assert.ErrorIs(t, err, ErrRequestAlreadyInLibrary)
	})

	t.Run("partial overlap is an honest rejection, never a silent trim", func(t *testing.T) {
		sel := &RequestSelection{Episodes: map[int][]int{2: {1, 2}}}
		err := checkSelectionOwnership(sel, owned, treeTVDetails())
		assert.ErrorIs(t, err, ErrRequestInvalidSelection)
	})

	t.Run("a whole-season pick over a partially-owned season overlaps", func(t *testing.T) {
		sel := &RequestSelection{Seasons: []int{2}}
		err := checkSelectionOwnership(sel, owned, treeTVDetails())
		assert.ErrorIs(t, err, ErrRequestInvalidSelection)
	})

	t.Run("nothing owned means nothing to guard", func(t *testing.T) {
		sel := &RequestSelection{Seasons: []int{1}}
		assert.NoError(t, checkSelectionOwnership(sel, nil, treeTVDetails()))
	})
}

// --- 13-2a create flow (service level) ---

func partialCreateReq(seasons []int, episodes map[string][]int) CreateMediaRequestRequest {
	return CreateMediaRequestRequest{TMDbID: 1399, MediaType: "tv", Seasons: seasons, Episodes: episodes}
}

func seasonDetailsFixture(season int, episodes ...int) *tmdb.SeasonDetails {
	d := &tmdb.SeasonDetails{SeasonNumber: season}
	for _, n := range episodes {
		d.Episodes = append(d.Episodes, tmdb.EpisodeInfo{EpisodeNumber: n})
	}
	return d
}

func TestRequestService_CreatePartialRequest_HappyPath(t *testing.T) {
	repo := &mockRequestRepo{}
	tmdbMock := &mockTMDbForRequests{tvDetails: treeTVDetails(),
		seasonDetails: map[int]*tmdb.SeasonDetails{2: seasonDetailsFixture(2, 1, 2, 3)}}
	svc := newRequestServiceForTest(repo, tmdbMock, nil, nil)

	created, err := svc.CreateRequest(context.Background(),
		partialCreateReq([]int{1}, map[string][]int{"2": {3, 2}}))
	require.NoError(t, err)

	require.Len(t, repo.created, 1)
	assert.Equal(t, "[1]", created.Seasons.String, "canonical seasons JSON lands in the column")
	assert.Equal(t, `{"2":[2,3]}`, created.Episodes.String, "episodes are canonicalized (sorted) before storage")
	assert.Equal(t, "怪奇物語", created.Title)
	assert.Equal(t, models.RequestStatusPending, created.Status)
}

func TestRequestService_CreateWholeTitle_ColumnsStayNull(t *testing.T) {
	// RED-line: the 13-1a whole-title path must stay byte-identical — no
	// selection means NULL/NULL exactly like every pre-13-2a row.
	tvDetails := &tmdb.TVShowDetails{}
	tvDetails.Name = "怪奇物語"
	repo := &mockRequestRepo{}
	svc := newRequestServiceForTest(repo, &mockTMDbForRequests{tvDetails: tvDetails}, nil, nil)

	created, err := svc.CreateRequest(context.Background(), CreateMediaRequestRequest{TMDbID: 1399, MediaType: "tv"})
	require.NoError(t, err)
	assert.False(t, created.Seasons.Valid)
	assert.False(t, created.Episodes.Valid)
}

func TestRequestService_CreatePartialRequest_DuplicateGuardHolds(t *testing.T) {
	// ⚖️ A ruling: a partial request does NOT bypass one-active-per-title.
	repo := &mockRequestRepo{active: &models.Request{ID: "existing"}}
	svc := newRequestServiceForTest(repo, &mockTMDbForRequests{tvDetails: treeTVDetails()}, nil, nil)

	_, err := svc.CreateRequest(context.Background(), partialCreateReq([]int{2}, nil))
	assert.ErrorIs(t, err, repository.ErrRequestDuplicate)
}

func TestRequestService_CreatePartialRequest_TMDBValidation(t *testing.T) {
	tmdbMock := &mockTMDbForRequests{tvDetails: treeTVDetails(),
		seasonDetails: map[int]*tmdb.SeasonDetails{2: seasonDetailsFixture(2, 1, 2, 3)}}
	svc := newRequestServiceForTest(&mockRequestRepo{}, tmdbMock, nil, nil)

	t.Run("unknown season is rejected", func(t *testing.T) {
		_, err := svc.CreateRequest(context.Background(), partialCreateReq([]int{9}, nil))
		assert.ErrorIs(t, err, ErrRequestInvalidSelection)
	})

	t.Run("unknown episode is rejected", func(t *testing.T) {
		_, err := svc.CreateRequest(context.Background(), partialCreateReq(nil, map[string][]int{"2": {99}}))
		assert.ErrorIs(t, err, ErrRequestInvalidSelection)
	})
}

func TestRequestService_CreatePartialRequest_OwnedGuardBranches(t *testing.T) {
	// Local series present: S1 fully owned (2/2 per TMDB episode_count),
	// S2E1 owned out of 3.
	episodes := []models.Episode{ownedEpisode(1, 1), ownedEpisode(1, 2), ownedEpisode(2, 1)}
	newSvc := func(repo *mockRequestRepo) *RequestService {
		tmdbMock := &mockTMDbForRequests{tvDetails: treeTVDetails(),
			seasonDetails: map[int]*tmdb.SeasonDetails{2: seasonDetailsFixture(2, 1, 2, 3)}}
		return newRequestServiceWithEpisodes(repo, tmdbMock, nil, []int64{1399}, episodes)
	}

	t.Run("requesting the un-owned remainder is allowed", func(t *testing.T) {
		repo := &mockRequestRepo{}
		created, err := newSvc(repo).CreateRequest(context.Background(),
			partialCreateReq(nil, map[string][]int{"2": {2, 3}}))
		require.NoError(t, err, "a partially-owned show requesting what it is missing is the entire point of 13-2a")
		assert.Equal(t, `{"2":[2,3]}`, created.Episodes.String)
	})

	t.Run("an entirely-owned selection is already-in-library", func(t *testing.T) {
		_, err := newSvc(&mockRequestRepo{}).CreateRequest(context.Background(),
			partialCreateReq([]int{1}, nil))
		assert.ErrorIs(t, err, ErrRequestAlreadyInLibrary)
	})

	t.Run("an overlapping selection is rejected honestly", func(t *testing.T) {
		_, err := newSvc(&mockRequestRepo{}).CreateRequest(context.Background(),
			partialCreateReq(nil, map[string][]int{"2": {1, 2}}))
		assert.ErrorIs(t, err, ErrRequestInvalidSelection)
	})
}

// --- 13-2a AC #5: coverage ---

func TestRequestService_TVCoverage(t *testing.T) {
	episodes := []models.Episode{ownedEpisode(1, 1), ownedEpisode(1, 2)}

	t.Run("owned plus active partial request", func(t *testing.T) {
		active := &models.Request{ID: "r1",
			Seasons:  models.NewNullString("[3]"),
			Episodes: models.NewNullString(`{"2":[5]}`)}
		repo := &mockRequestRepo{active: active}
		svc := newRequestServiceWithEpisodes(repo, &mockTMDbForRequests{}, nil, []int64{1399}, episodes)

		cov, err := svc.TVCoverage(context.Background(), 1399)
		require.NoError(t, err)
		assert.Equal(t, map[string][]int{"1": {1, 2}}, cov.Owned)
		assert.Equal(t, []int{3}, cov.RequestedSeasons)
		assert.Equal(t, map[string][]int{"2": {5}}, cov.RequestedEpisodes)
		assert.True(t, cov.ActiveRequest)
		assert.False(t, cov.WholeSeriesRequested)
	})

	t.Run("active whole-series request", func(t *testing.T) {
		repo := &mockRequestRepo{active: &models.Request{ID: "r1"}}
		svc := newRequestServiceWithEpisodes(repo, &mockTMDbForRequests{}, nil, nil, nil)

		cov, err := svc.TVCoverage(context.Background(), 1399)
		require.NoError(t, err)
		assert.True(t, cov.WholeSeriesRequested)
		assert.True(t, cov.ActiveRequest)
	})

	t.Run("brand-new show returns the empty-but-valid shape", func(t *testing.T) {
		svc := newRequestServiceWithEpisodes(&mockRequestRepo{}, &mockTMDbForRequests{}, nil, nil, nil)

		cov, err := svc.TVCoverage(context.Background(), 1399)
		require.NoError(t, err)
		assert.NotNil(t, cov.Owned)
		assert.NotNil(t, cov.RequestedSeasons)
		assert.NotNil(t, cov.RequestedEpisodes)
		assert.False(t, cov.ActiveRequest)
		assert.False(t, cov.WholeSeriesRequested)
	})
}

// --- CR 13-2a M1: selection size ceilings bound upstream amplification ---

func TestCanonicalizeSelection_SizeCeilings(t *testing.T) {
	t.Run("too many seasons is rejected", func(t *testing.T) {
		seasons := make([]int, 0, maxSelectedSeasons+1)
		for i := 1; i <= maxSelectedSeasons+1; i++ {
			seasons = append(seasons, i)
		}
		_, err := canonicalizeSelection(models.RequestMediaTypeTV, seasons, nil)
		assert.ErrorIs(t, err, ErrRequestInvalidSelection,
			"every selected season costs one TMDB fetch on create and one Sonarr fetch on fulfilment")
	})

	t.Run("too many episodes is rejected", func(t *testing.T) {
		episodes := map[string][]int{}
		for season := 1; season <= 10; season++ {
			list := make([]int, 0, 300)
			for e := 1; e <= 300; e++ {
				list = append(list, e)
			}
			episodes[strconv.Itoa(season)] = list
		}
		_, err := canonicalizeSelection(models.RequestMediaTypeTV, nil, episodes)
		assert.ErrorIs(t, err, ErrRequestInvalidSelection)
	})

	t.Run("a realistic long-running show still passes", func(t *testing.T) {
		seasons := make([]int, 0, 40)
		for i := 1; i <= 40; i++ {
			seasons = append(seasons, i)
		}
		sel, err := canonicalizeSelection(models.RequestMediaTypeTV, seasons, map[string][]int{"41": {1, 2, 3}})
		require.NoError(t, err)
		assert.Len(t, sel.Seasons, 40)
	})
}

// --- CR 13-2a H1 (service half): the poller's completion predicate ---

func TestRequestService_SelectionFullyOwned(t *testing.T) {
	episodes := []models.Episode{ownedEpisode(1, 1), ownedEpisode(1, 2), ownedEpisode(2, 1)}
	svc := newRequestServiceWithEpisodes(&mockRequestRepo{},
		&mockTMDbForRequests{tvDetails: treeTVDetails()}, nil, []int64{1399}, episodes)

	t.Run("selection not yet landed is not owned", func(t *testing.T) {
		full, err := svc.SelectionFullyOwned(context.Background(), 1399,
			models.NullString{}, models.NewNullString(`{"2":[2,3]}`))
		require.NoError(t, err)
		assert.False(t, full)
	})

	t.Run("landed selection is owned", func(t *testing.T) {
		full, err := svc.SelectionFullyOwned(context.Background(), 1399,
			models.NewNullString("[1]"), models.NullString{})
		require.NoError(t, err)
		assert.True(t, full, "season 1 is fully present (2/2 per TMDB episode_count)")
	})

	t.Run("whole-title rows defer to the title-level rule", func(t *testing.T) {
		full, err := svc.SelectionFullyOwned(context.Background(), 1399, models.NullString{}, models.NullString{})
		require.NoError(t, err)
		assert.True(t, full)
	})

	t.Run("malformed stored selection surfaces as an error, never as completed", func(t *testing.T) {
		_, err := svc.SelectionFullyOwned(context.Background(), 1399, models.NewNullString("nope"), models.NullString{})
		require.Error(t, err)
	})
}
