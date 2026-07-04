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
	"github.com/vido/api/internal/testutil"
	"github.com/vido/api/internal/tmdb"
)

// mockRequestRepo is a hand-written in-memory mock of RequestRepositoryInterface.
type mockRequestRepo struct {
	created   []*models.Request
	createErr error
	listResp  []models.Request
	listErr   error
	active    *models.Request // returned by FindActiveByTMDbID when non-nil
	activeErr error           // overrides the derived behavior when non-nil
}

var _ repository.RequestRepositoryInterface = (*mockRequestRepo)(nil)

func (m *mockRequestRepo) Create(ctx context.Context, request *models.Request) error {
	if m.createErr != nil {
		return m.createErr
	}
	request.ID = "test-id"
	m.created = append(m.created, request)
	return nil
}

func (m *mockRequestRepo) List(ctx context.Context) ([]models.Request, error) {
	return m.listResp, m.listErr
}

func (m *mockRequestRepo) FindActiveByTMDbID(ctx context.Context, tmdbID int64, mediaType string) (*models.Request, error) {
	if m.activeErr != nil {
		return nil, m.activeErr
	}
	if m.active != nil {
		return m.active, nil
	}
	return nil, repository.ErrRequestNotFound
}

func (m *mockRequestRepo) UpdateFulfilment(ctx context.Context, id string, status string, fulfilmentSource, externalID, errorMessage models.NullString) (time.Time, error) {
	return time.Now(), nil
}

func (m *mockRequestRepo) ListActive(ctx context.Context) ([]models.Request, error) {
	return nil, nil
}

func (m *mockRequestRepo) UpdateStatus(ctx context.Context, id string, status string, errMsg string) (time.Time, error) {
	return time.Now(), nil
}

// mockTMDbForRequests embeds the shared explore mock (same package) and
// overrides only the two detail lookups the request service uses.
type mockTMDbForRequests struct {
	mockTMDbServiceForExplore
	movieDetails *tmdb.MovieDetails
	movieErr     error
	tvDetails    *tmdb.TVShowDetails
	tvErr        error
}

func (m *mockTMDbForRequests) GetMovieDetails(ctx context.Context, id int) (*tmdb.MovieDetails, error) {
	if m.movieErr != nil {
		return nil, m.movieErr
	}
	return m.movieDetails, nil
}

func (m *mockTMDbForRequests) GetTVShowDetails(ctx context.Context, id int) (*tmdb.TVShowDetails, error) {
	if m.tvErr != nil {
		return nil, m.tvErr
	}
	return m.tvDetails, nil
}

func newRequestServiceForTest(repo *mockRequestRepo, tmdbMock *mockTMDbForRequests, ownedMovies, ownedSeries []int64) *RequestService {
	movieRepo := new(testutil.MockMovieRepository)
	seriesRepo := new(testutil.MockSeriesRepository)
	movieRepo.On("FindOwnedTMDbIDs", context.Background(), []int64{550}).Return(ownedMovies, nil).Maybe()
	movieRepo.On("FindOwnedTMDbIDs", context.Background(), []int64{1399}).Return(ownedMovies, nil).Maybe()
	seriesRepo.On("FindOwnedTMDbIDs", context.Background(), []int64{550}).Return(ownedSeries, nil).Maybe()
	seriesRepo.On("FindOwnedTMDbIDs", context.Background(), []int64{1399}).Return(ownedSeries, nil).Maybe()
	return NewRequestService(repo, tmdbMock, movieRepo, seriesRepo)
}

// stubFulfilment records the FulfilRequest call and simulates the 13-4a
// success transition in place.
type stubFulfilment struct {
	calls    int
	lastReq  *models.Request
	simulate func(request *models.Request)
}

func (s *stubFulfilment) FulfilRequest(ctx context.Context, request *models.Request) {
	s.calls++
	s.lastReq = request
	if s.simulate != nil {
		s.simulate(request)
	}
}

func TestRequestService_CreateRequest_WithFulfilment(t *testing.T) {
	ctx := context.Background()
	movieDetails := &tmdb.MovieDetails{}
	movieDetails.Title = "鬥陣俱樂部"

	repo := &mockRequestRepo{}
	svc := newRequestServiceForTest(repo, &mockTMDbForRequests{movieDetails: movieDetails}, nil, nil)
	fulfilment := &stubFulfilment{simulate: func(request *models.Request) {
		request.Status = models.RequestStatusSearching
		request.FulfilmentSource = models.NewNullString(models.RequestFulfilmentSourceArr)
		request.ExternalID = models.NewNullString("42")
	}}
	svc.SetFulfilmentService(fulfilment)

	created, err := svc.CreateRequest(ctx, CreateMediaRequestRequest{TMDbID: 550, MediaType: "movie"})
	require.NoError(t, err)

	assert.Equal(t, 1, fulfilment.calls, "fulfilment must run synchronously on create (AC #6)")
	assert.Same(t, created, fulfilment.lastReq, "fulfilment receives the created row")
	assert.Equal(t, models.RequestStatusSearching, created.Status,
		"the create response carries the transition (searching stays within the 13-1a enum — no bump)")
	assert.Equal(t, "42", created.ExternalID.String)
}

func TestRequestService_CreateRequest_TVWithFulfilment(t *testing.T) {
	// 13-4b — tv requests flow through the same fulfilment seam; the create
	// response carries whatever transition the tv branch produced.
	ctx := context.Background()
	tvDetails := &tmdb.TVShowDetails{}
	tvDetails.Name = "冰與火之歌"

	repo := &mockRequestRepo{}
	svc := newRequestServiceForTest(repo, &mockTMDbForRequests{tvDetails: tvDetails}, nil, nil)
	fulfilment := &stubFulfilment{simulate: func(request *models.Request) {
		request.Status = models.RequestStatusSearching
		request.FulfilmentSource = models.NewNullString(models.RequestFulfilmentSourceArr)
		request.ExternalID = models.NewNullString("7")
	}}
	svc.SetFulfilmentService(fulfilment)

	created, err := svc.CreateRequest(ctx, CreateMediaRequestRequest{TMDbID: 1399, MediaType: "tv"})
	require.NoError(t, err)

	assert.Equal(t, 1, fulfilment.calls)
	assert.Equal(t, models.RequestMediaTypeTV, fulfilment.lastReq.MediaType)
	assert.Equal(t, models.RequestStatusSearching, created.Status)
	assert.Equal(t, "7", created.ExternalID.String)
}

func TestRequestService_CreateRequest_NilFulfilmentIsNoOp(t *testing.T) {
	// 13-1a behavior preserved exactly when the optional dep is absent.
	ctx := context.Background()
	movieDetails := &tmdb.MovieDetails{}
	movieDetails.Title = "x"

	repo := &mockRequestRepo{}
	svc := newRequestServiceForTest(repo, &mockTMDbForRequests{movieDetails: movieDetails}, nil, nil)

	created, err := svc.CreateRequest(ctx, CreateMediaRequestRequest{TMDbID: 550, MediaType: "movie"})
	require.NoError(t, err)
	assert.Equal(t, models.RequestStatusPending, created.Status)
	assert.False(t, created.FulfilmentSource.Valid)
	assert.False(t, created.ErrorMessage.Valid)
}

func TestRequestService_CreateRequest_Movie(t *testing.T) {
	ctx := context.Background()
	movieDetails := &tmdb.MovieDetails{}
	movieDetails.Title = "鬥陣俱樂部"

	repo := &mockRequestRepo{}
	svc := newRequestServiceForTest(repo, &mockTMDbForRequests{movieDetails: movieDetails}, nil, nil)

	created, err := svc.CreateRequest(ctx, CreateMediaRequestRequest{TMDbID: 550, MediaType: "movie"})
	require.NoError(t, err)

	assert.Equal(t, "test-id", created.ID)
	assert.Equal(t, "鬥陣俱樂部", created.Title, "title must be the server-side zh-TW resolve, never client input")
	assert.Equal(t, models.RequestStatusPending, created.Status, "rows are born pending (AC #9 capability-honor)")
	assert.False(t, created.FulfilmentSource.Valid)
	require.Len(t, repo.created, 1)
}

func TestRequestService_CreateRequest_EmptyTitleFallback(t *testing.T) {
	ctx := context.Background()

	repo := &mockRequestRepo{}
	svc := newRequestServiceForTest(repo, &mockTMDbForRequests{movieDetails: &tmdb.MovieDetails{}}, nil, nil)

	created, err := svc.CreateRequest(ctx, CreateMediaRequestRequest{TMDbID: 550, MediaType: "movie"})
	require.NoError(t, err)
	assert.Equal(t, "TMDB-550", created.Title, "no-usable-title edge stores a deterministic placeholder, never an empty NOT-NULL title (CR L1)")
}

func TestRequestService_CreateRequest_TV(t *testing.T) {
	ctx := context.Background()
	tvDetails := &tmdb.TVShowDetails{}
	tvDetails.Name = "權力遊戲"

	repo := &mockRequestRepo{}
	svc := newRequestServiceForTest(repo, &mockTMDbForRequests{tvDetails: tvDetails}, nil, nil)

	created, err := svc.CreateRequest(ctx, CreateMediaRequestRequest{TMDbID: 1399, MediaType: "tv"})
	require.NoError(t, err)
	assert.Equal(t, "權力遊戲", created.Title, "tv titles come from Name (not Title)")
}

func TestRequestService_CreateRequest_Validation(t *testing.T) {
	ctx := context.Background()
	svc := newRequestServiceForTest(&mockRequestRepo{}, &mockTMDbForRequests{}, nil, nil)

	t.Run("zero tmdb_id rejected", func(t *testing.T) {
		_, err := svc.CreateRequest(ctx, CreateMediaRequestRequest{TMDbID: 0, MediaType: "movie"})
		var validationErr *models.ValidationError
		require.ErrorAs(t, err, &validationErr)
		assert.Equal(t, "tmdb_id", validationErr.Field)
	})

	t.Run("media_type 'series' rejected — requests speak TMDB vocabulary", func(t *testing.T) {
		_, err := svc.CreateRequest(ctx, CreateMediaRequestRequest{TMDbID: 550, MediaType: "series"})
		var validationErr *models.ValidationError
		require.ErrorAs(t, err, &validationErr)
		assert.Equal(t, "media_type", validationErr.Field)
	})
}

func TestRequestService_CreateRequest_Guards(t *testing.T) {
	ctx := context.Background()

	t.Run("already in library → ErrRequestAlreadyInLibrary (AC #5)", func(t *testing.T) {
		repo := &mockRequestRepo{}
		svc := newRequestServiceForTest(repo, &mockTMDbForRequests{}, []int64{550}, nil)

		_, err := svc.CreateRequest(ctx, CreateMediaRequestRequest{TMDbID: 550, MediaType: "movie"})
		assert.ErrorIs(t, err, ErrRequestAlreadyInLibrary)
		assert.Empty(t, repo.created, "no row may be created for owned media")
	})

	t.Run("active duplicate → ErrRequestDuplicate (AC #4)", func(t *testing.T) {
		repo := &mockRequestRepo{active: &models.Request{ID: "existing"}}
		svc := newRequestServiceForTest(repo, &mockTMDbForRequests{}, nil, nil)

		_, err := svc.CreateRequest(ctx, CreateMediaRequestRequest{TMDbID: 550, MediaType: "movie"})
		assert.ErrorIs(t, err, repository.ErrRequestDuplicate)
		assert.Empty(t, repo.created)
	})

	t.Run("unknown tmdb_id → TMDb error propagates typed (AC #2)", func(t *testing.T) {
		notFound := &tmdb.TMDbError{Code: tmdb.ErrCodeNotFound, Message: "Not found"}
		repo := &mockRequestRepo{}
		svc := newRequestServiceForTest(repo, &mockTMDbForRequests{movieErr: notFound}, nil, nil)

		_, err := svc.CreateRequest(ctx, CreateMediaRequestRequest{TMDbID: 550, MediaType: "movie"})
		var tmdbErr *tmdb.TMDbError
		require.ErrorAs(t, err, &tmdbErr)
		assert.Equal(t, tmdb.ErrCodeNotFound, tmdbErr.Code)
		assert.Empty(t, repo.created, "no row may be created when the target does not resolve")
	})

	t.Run("duplicate-check infra error propagates (Rule 13)", func(t *testing.T) {
		repo := &mockRequestRepo{activeErr: errors.New("db down")}
		svc := newRequestServiceForTest(repo, &mockTMDbForRequests{}, nil, nil)

		_, err := svc.CreateRequest(ctx, CreateMediaRequestRequest{TMDbID: 550, MediaType: "movie"})
		assert.ErrorContains(t, err, "duplicate check")
	})
}

func TestRequestService_ListRequests(t *testing.T) {
	ctx := context.Background()

	repo := &mockRequestRepo{listResp: []models.Request{{ID: "r1"}, {ID: "r2"}}}
	svc := newRequestServiceForTest(repo, &mockTMDbForRequests{}, nil, nil)

	requests, err := svc.ListRequests(ctx)
	require.NoError(t, err)
	assert.Len(t, requests, 2)

	t.Run("repo error propagates", func(t *testing.T) {
		repo.listErr = errors.New("boom")
		_, err := svc.ListRequests(ctx)
		assert.ErrorContains(t, err, "list requests")
	})
}
