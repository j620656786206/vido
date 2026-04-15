package services

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/tmdb"
)

// --- Mocks ---

type mockTMDbServiceForExplore struct {
	discoverMoviesResp  *tmdb.SearchResultMovies
	discoverMoviesErr   error
	discoverMoviesCalls []tmdb.DiscoverParams

	discoverTVResp  *tmdb.SearchResultTVShows
	discoverTVErr   error
	discoverTVCalls []tmdb.DiscoverParams
}

func (m *mockTMDbServiceForExplore) SearchMovies(ctx context.Context, q string, p int) (*tmdb.SearchResultMovies, error) {
	return &tmdb.SearchResultMovies{}, nil
}
func (m *mockTMDbServiceForExplore) SearchTVShows(ctx context.Context, q string, p int) (*tmdb.SearchResultTVShows, error) {
	return &tmdb.SearchResultTVShows{}, nil
}
func (m *mockTMDbServiceForExplore) GetMovieDetails(ctx context.Context, id int) (*tmdb.MovieDetails, error) {
	return &tmdb.MovieDetails{}, nil
}
func (m *mockTMDbServiceForExplore) GetTVShowDetails(ctx context.Context, id int) (*tmdb.TVShowDetails, error) {
	return &tmdb.TVShowDetails{}, nil
}
func (m *mockTMDbServiceForExplore) FindByExternalID(ctx context.Context, id, src string) (*tmdb.FindByExternalIDResponse, error) {
	return &tmdb.FindByExternalIDResponse{}, nil
}
func (m *mockTMDbServiceForExplore) GetTrendingMovies(ctx context.Context, w string, p int) (*tmdb.SearchResultMovies, error) {
	return &tmdb.SearchResultMovies{}, nil
}
func (m *mockTMDbServiceForExplore) GetTrendingTVShows(ctx context.Context, w string, p int) (*tmdb.SearchResultTVShows, error) {
	return &tmdb.SearchResultTVShows{}, nil
}
func (m *mockTMDbServiceForExplore) DiscoverMovies(ctx context.Context, params tmdb.DiscoverParams) (*tmdb.SearchResultMovies, error) {
	m.discoverMoviesCalls = append(m.discoverMoviesCalls, params)
	if m.discoverMoviesErr != nil {
		return nil, m.discoverMoviesErr
	}
	return m.discoverMoviesResp, nil
}
func (m *mockTMDbServiceForExplore) DiscoverTVShows(ctx context.Context, params tmdb.DiscoverParams) (*tmdb.SearchResultTVShows, error) {
	m.discoverTVCalls = append(m.discoverTVCalls, params)
	if m.discoverTVErr != nil {
		return nil, m.discoverTVErr
	}
	return m.discoverTVResp, nil
}
func (m *mockTMDbServiceForExplore) GetMovieVideos(ctx context.Context, id int) (*tmdb.VideosResponse, error) {
	return &tmdb.VideosResponse{}, nil
}
func (m *mockTMDbServiceForExplore) GetTVShowVideos(ctx context.Context, id int) (*tmdb.VideosResponse, error) {
	return &tmdb.VideosResponse{}, nil
}

var _ TMDbServiceInterface = (*mockTMDbServiceForExplore)(nil)

// --- SQLite test fixture (reuses the schema the runtime migration creates) ---

func setupExploreBlockServiceDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	schema := `
		CREATE TABLE explore_blocks (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			content_type TEXT NOT NULL CHECK(content_type IN ('movie', 'tv')),
			genre_ids TEXT NOT NULL DEFAULT '',
			language TEXT NOT NULL DEFAULT '',
			region TEXT NOT NULL DEFAULT '',
			sort_by TEXT NOT NULL DEFAULT '',
			max_items INTEGER NOT NULL DEFAULT 20,
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX idx_explore_blocks_sort_order ON explore_blocks(sort_order);
		CREATE TABLE cache_entries (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			type TEXT NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`
	_, err = db.Exec(schema)
	require.NoError(t, err)

	t.Cleanup(func() { db.Close() })
	return db
}

func newExploreService(t *testing.T, mock *mockTMDbServiceForExplore) (*ExploreBlockService, *sql.DB) {
	t.Helper()
	db := setupExploreBlockServiceDB(t)
	repo := repository.NewExploreBlockRepository(db)
	cacheRepo := repository.NewCacheRepository(db)
	svc := NewExploreBlockService(repo, mock, cacheRepo)
	return svc, db
}

// --- CRUD tests ---

func TestExploreBlockService_CreateBlock_Success(t *testing.T) {
	svc, _ := newExploreService(t, &mockTMDbServiceForExplore{})
	ctx := context.Background()

	block, err := svc.CreateBlock(ctx, CreateExploreBlockRequest{
		Name:        "  熱門電影  ",
		ContentType: "movie",
		SortBy:      "popularity.desc",
		MaxItems:    20,
	})
	require.NoError(t, err)
	assert.Equal(t, "熱門電影", block.Name, "name is trimmed")
	assert.Equal(t, models.ExploreBlockContentMovie, block.ContentType)
	assert.Equal(t, 0, block.SortOrder, "first block gets sort_order 0")
	assert.NotEmpty(t, block.ID)
}

func TestExploreBlockService_CreateBlock_AssignsIncrementingSortOrder(t *testing.T) {
	svc, _ := newExploreService(t, &mockTMDbServiceForExplore{})
	ctx := context.Background()

	b1, err := svc.CreateBlock(ctx, CreateExploreBlockRequest{
		Name: "第一", ContentType: "movie", MaxItems: 20,
	})
	require.NoError(t, err)

	b2, err := svc.CreateBlock(ctx, CreateExploreBlockRequest{
		Name: "第二", ContentType: "tv", MaxItems: 20,
	})
	require.NoError(t, err)

	assert.Equal(t, 0, b1.SortOrder)
	assert.Equal(t, 1, b2.SortOrder)
}

func TestExploreBlockService_CreateBlock_DefaultMaxItems(t *testing.T) {
	svc, _ := newExploreService(t, &mockTMDbServiceForExplore{})
	ctx := context.Background()

	block, err := svc.CreateBlock(ctx, CreateExploreBlockRequest{
		Name: "沒給上限", ContentType: "movie", // MaxItems omitted → 0
	})
	require.NoError(t, err)
	assert.Equal(t, models.ExploreBlockDefaultMaxItems, block.MaxItems)
}

func TestExploreBlockService_CreateBlock_ValidationErrors(t *testing.T) {
	svc, _ := newExploreService(t, &mockTMDbServiceForExplore{})
	ctx := context.Background()

	cases := []struct {
		name string
		req  CreateExploreBlockRequest
	}{
		{"empty name", CreateExploreBlockRequest{Name: "   ", ContentType: "movie", MaxItems: 10}},
		{"bad content type", CreateExploreBlockRequest{Name: "X", ContentType: "series", MaxItems: 10}},
		{"max_items out of range", CreateExploreBlockRequest{Name: "X", ContentType: "movie", MaxItems: 500}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.CreateBlock(ctx, tc.req)
			require.Error(t, err)
			var ve *models.ValidationError
			assert.True(t, errors.As(err, &ve), "expected ValidationError, got %T", err)
		})
	}
}

func TestExploreBlockService_UpdateBlock_PartialFields(t *testing.T) {
	svc, _ := newExploreService(t, &mockTMDbServiceForExplore{})
	ctx := context.Background()

	block, err := svc.CreateBlock(ctx, CreateExploreBlockRequest{
		Name: "原名", ContentType: "movie", MaxItems: 10,
	})
	require.NoError(t, err)

	newName := "新名字"
	newMax := 25
	updated, err := svc.UpdateBlock(ctx, block.ID, UpdateExploreBlockRequest{
		Name:     &newName,
		MaxItems: &newMax,
	})
	require.NoError(t, err)
	assert.Equal(t, "新名字", updated.Name)
	assert.Equal(t, 25, updated.MaxItems)
	assert.Equal(t, models.ExploreBlockContentMovie, updated.ContentType, "content type untouched")
}

func TestExploreBlockService_UpdateBlock_NotFound(t *testing.T) {
	svc, _ := newExploreService(t, &mockTMDbServiceForExplore{})
	newName := "nope"
	_, err := svc.UpdateBlock(context.Background(), "missing-id", UpdateExploreBlockRequest{Name: &newName})
	require.Error(t, err)
	assert.True(t, errors.Is(err, repository.ErrExploreBlockNotFound))
}

func TestExploreBlockService_DeleteBlock(t *testing.T) {
	svc, _ := newExploreService(t, &mockTMDbServiceForExplore{})
	ctx := context.Background()

	block, err := svc.CreateBlock(ctx, CreateExploreBlockRequest{
		Name: "Del", ContentType: "movie", MaxItems: 10,
	})
	require.NoError(t, err)

	require.NoError(t, svc.DeleteBlock(ctx, block.ID))

	_, err = svc.GetBlock(ctx, block.ID)
	require.Error(t, err)
}

func TestExploreBlockService_ReorderBlocks(t *testing.T) {
	svc, _ := newExploreService(t, &mockTMDbServiceForExplore{})
	ctx := context.Background()

	a, err := svc.CreateBlock(ctx, CreateExploreBlockRequest{Name: "A", ContentType: "movie", MaxItems: 10})
	require.NoError(t, err)
	b, err := svc.CreateBlock(ctx, CreateExploreBlockRequest{Name: "B", ContentType: "movie", MaxItems: 10})
	require.NoError(t, err)

	list, err := svc.ReorderBlocks(ctx, []string{b.ID, a.ID})
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, "B", list[0].Name)
	assert.Equal(t, "A", list[1].Name)
}

func TestExploreBlockService_SeedDefaultsIfEmpty(t *testing.T) {
	svc, _ := newExploreService(t, &mockTMDbServiceForExplore{})
	ctx := context.Background()

	require.NoError(t, svc.SeedDefaultsIfEmpty(ctx))

	blocks, err := svc.GetAllBlocks(ctx)
	require.NoError(t, err)
	require.Len(t, blocks, 3, "three default blocks seeded (AC #5)")
	assert.Equal(t, "熱門電影", blocks[0].Name)
	assert.Equal(t, "熱門影集", blocks[1].Name)
	assert.Equal(t, "近期新片", blocks[2].Name)
	assert.Equal(t, models.ExploreBlockContentMovie, blocks[0].ContentType)
	assert.Equal(t, models.ExploreBlockContentTV, blocks[1].ContentType)
}

func TestExploreBlockService_SeedDefaultsIfEmpty_Idempotent(t *testing.T) {
	svc, _ := newExploreService(t, &mockTMDbServiceForExplore{})
	ctx := context.Background()

	require.NoError(t, svc.SeedDefaultsIfEmpty(ctx))
	require.NoError(t, svc.SeedDefaultsIfEmpty(ctx)) // second run

	blocks, err := svc.GetAllBlocks(ctx)
	require.NoError(t, err)
	assert.Len(t, blocks, 3, "still three; seeds don't duplicate")
}

// --- Content fetch tests ---

func TestExploreBlockService_GetBlockContent_Movie(t *testing.T) {
	mock := &mockTMDbServiceForExplore{
		discoverMoviesResp: &tmdb.SearchResultMovies{
			Results: []tmdb.Movie{
				{ID: 1, Title: "Film A", VoteAverage: 8.5, VoteCount: 1000, ReleaseDate: "2024-05-01"},
				{ID: 2, Title: "Film B", VoteAverage: 7.2, VoteCount: 200, ReleaseDate: "2024-08-10"},
				{ID: 3, Title: "Filler", VoteAverage: 2.0, VoteCount: 3, ReleaseDate: "2024-01-01"}, // low quality — filtered
			},
		},
	}
	svc, _ := newExploreService(t, mock)
	// Fixed-clock filter so FarFuture math is deterministic
	svc.SetContentFilter(NewContentFilterServiceWithClock(func() time.Time {
		return time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC)
	}))
	ctx := context.Background()

	block, err := svc.CreateBlock(ctx, CreateExploreBlockRequest{
		Name: "熱門電影", ContentType: "movie", SortBy: "popularity.desc", MaxItems: 20,
	})
	require.NoError(t, err)

	content, err := svc.GetBlockContent(ctx, block.ID)
	require.NoError(t, err)
	require.NotNil(t, content)
	assert.Equal(t, block.ID, content.BlockID)
	assert.Equal(t, "movie", content.ContentType)
	require.Len(t, content.Movies, 2, "low-quality Filler removed")
	assert.Equal(t, 1, content.Movies[0].ID)
	assert.Equal(t, 2, content.TotalItems)
}

func TestExploreBlockService_GetBlockContent_TV(t *testing.T) {
	mock := &mockTMDbServiceForExplore{
		discoverTVResp: &tmdb.SearchResultTVShows{
			Results: []tmdb.TVShow{
				{ID: 10, Name: "劇集 A", VoteAverage: 8.0, VoteCount: 500, FirstAirDate: "2023-06-01"},
				{ID: 11, Name: "劇集 B", VoteAverage: 7.5, VoteCount: 120, FirstAirDate: "2024-01-15"},
			},
		},
	}
	svc, _ := newExploreService(t, mock)
	svc.SetContentFilter(NewContentFilterServiceWithClock(func() time.Time {
		return time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC)
	}))
	ctx := context.Background()

	block, err := svc.CreateBlock(ctx, CreateExploreBlockRequest{
		Name: "劇集", ContentType: "tv", SortBy: "popularity.desc", MaxItems: 20,
	})
	require.NoError(t, err)

	content, err := svc.GetBlockContent(ctx, block.ID)
	require.NoError(t, err)
	require.NotNil(t, content)
	assert.Equal(t, "tv", content.ContentType)
	assert.Len(t, content.TVShows, 2)
	assert.Empty(t, content.Movies)
}

func TestExploreBlockService_GetBlockContent_MaxItemsCap(t *testing.T) {
	movies := make([]tmdb.Movie, 25)
	for i := range movies {
		movies[i] = tmdb.Movie{
			ID:          i + 1,
			Title:       "Movie",
			VoteAverage: 9,
			VoteCount:   1000,
			ReleaseDate: "2024-01-01",
		}
	}
	mock := &mockTMDbServiceForExplore{
		discoverMoviesResp: &tmdb.SearchResultMovies{Results: movies},
	}
	svc, _ := newExploreService(t, mock)
	svc.SetContentFilter(NewContentFilterServiceWithClock(func() time.Time {
		return time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	}))
	ctx := context.Background()

	block, err := svc.CreateBlock(ctx, CreateExploreBlockRequest{
		Name: "Top10", ContentType: "movie", MaxItems: 10,
	})
	require.NoError(t, err)

	content, err := svc.GetBlockContent(ctx, block.ID)
	require.NoError(t, err)
	assert.Len(t, content.Movies, 10, "capped at max_items=10")
}

func TestExploreBlockService_GetBlockContent_Caches(t *testing.T) {
	mock := &mockTMDbServiceForExplore{
		discoverMoviesResp: &tmdb.SearchResultMovies{
			Results: []tmdb.Movie{{ID: 1, Title: "A", VoteAverage: 7, VoteCount: 200, ReleaseDate: "2024-01-01"}},
		},
	}
	svc, _ := newExploreService(t, mock)
	svc.SetContentFilter(NewContentFilterServiceWithClock(func() time.Time {
		return time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	}))
	ctx := context.Background()

	block, err := svc.CreateBlock(ctx, CreateExploreBlockRequest{
		Name: "Cached", ContentType: "movie", MaxItems: 10,
	})
	require.NoError(t, err)

	_, err = svc.GetBlockContent(ctx, block.ID)
	require.NoError(t, err)
	_, err = svc.GetBlockContent(ctx, block.ID)
	require.NoError(t, err)

	assert.Len(t, mock.discoverMoviesCalls, 1, "second call served from cache — TMDb not hit again")
}

func TestExploreBlockService_GetBlockContent_TMDbError(t *testing.T) {
	mock := &mockTMDbServiceForExplore{
		discoverMoviesErr: errors.New("boom"),
	}
	svc, _ := newExploreService(t, mock)
	ctx := context.Background()

	block, err := svc.CreateBlock(ctx, CreateExploreBlockRequest{
		Name: "X", ContentType: "movie", MaxItems: 10,
	})
	require.NoError(t, err)

	_, err = svc.GetBlockContent(ctx, block.ID)
	require.Error(t, err)
}

func TestExploreBlockService_GetBlockContent_NotFound(t *testing.T) {
	svc, _ := newExploreService(t, &mockTMDbServiceForExplore{})
	_, err := svc.GetBlockContent(context.Background(), "nope")
	require.Error(t, err)
	assert.True(t, errors.Is(err, repository.ErrExploreBlockNotFound))
}

func TestExploreBlockService_UpdateBlock_InvalidatesCache(t *testing.T) {
	mock := &mockTMDbServiceForExplore{
		discoverMoviesResp: &tmdb.SearchResultMovies{
			Results: []tmdb.Movie{{ID: 1, Title: "A", VoteAverage: 7, VoteCount: 200, ReleaseDate: "2024-01-01"}},
		},
	}
	svc, _ := newExploreService(t, mock)
	svc.SetContentFilter(NewContentFilterServiceWithClock(func() time.Time {
		return time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	}))
	ctx := context.Background()

	block, err := svc.CreateBlock(ctx, CreateExploreBlockRequest{
		Name: "Cached", ContentType: "movie", MaxItems: 10,
	})
	require.NoError(t, err)

	_, err = svc.GetBlockContent(ctx, block.ID) // populate cache
	require.NoError(t, err)

	newName := "已改名"
	_, err = svc.UpdateBlock(ctx, block.ID, UpdateExploreBlockRequest{Name: &newName})
	require.NoError(t, err)

	_, err = svc.GetBlockContent(ctx, block.ID) // should refetch
	require.NoError(t, err)

	assert.Len(t, mock.discoverMoviesCalls, 2, "update invalidates cache; second get refetches")
}
