package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/services"
	"github.com/vido/api/internal/tmdb"
)

// mockRequestService is a hand-written mock of RequestServiceInterface.
type mockRequestService struct {
	createResp   *models.Request
	createErr    error
	listResp     []models.Request
	listErr      error
	lastCreate   services.CreateMediaRequestRequest
	coverageResp *services.RequestCoverage
	coverageErr  error
	lastCoverage int64
	cancelErr    error
	lastCancel   string
	retryResp    *models.Request
	retryErr     error
	lastRetry    string
}

func (m *mockRequestService) CancelRequest(ctx context.Context, id string) error {
	m.lastCancel = id
	return m.cancelErr
}

func (m *mockRequestService) RetryRequest(ctx context.Context, id string) (*models.Request, error) {
	m.lastRetry = id
	return m.retryResp, m.retryErr
}

func (m *mockRequestService) TVCoverage(ctx context.Context, tmdbID int64) (*services.RequestCoverage, error) {
	m.lastCoverage = tmdbID
	if m.coverageErr != nil {
		return nil, m.coverageErr
	}
	if m.coverageResp != nil {
		return m.coverageResp, nil
	}
	return &services.RequestCoverage{
		Owned:             map[string][]int{},
		RequestedSeasons:  []int{},
		RequestedEpisodes: map[string][]int{},
	}, nil
}

var _ services.RequestServiceInterface = (*mockRequestService)(nil)

func (m *mockRequestService) CreateRequest(ctx context.Context, req services.CreateMediaRequestRequest) (*models.Request, error) {
	m.lastCreate = req
	if m.createErr != nil {
		return nil, m.createErr
	}
	return m.createResp, nil
}

func (m *mockRequestService) ListRequests(ctx context.Context) ([]models.Request, error) {
	return m.listResp, m.listErr
}

func setupRequestRouter(service services.RequestServiceInterface) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewRequestHandler(service)
	h.RegisterRoutes(r.Group("/api/v1"))
	return r
}

func TestRequestHandler_ListRequests(t *testing.T) {
	t.Run("returns requests newest first under data.requests", func(t *testing.T) {
		svc := &mockRequestService{listResp: []models.Request{{ID: "r2", Title: "second"}, {ID: "r1", Title: "first"}}}
		r := setupRequestRouter(svc)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/requests", nil))

		require.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			Success bool `json:"success"`
			Data    struct {
				Requests []models.Request `json:"requests"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.True(t, resp.Success)
		require.Len(t, resp.Data.Requests, 2)
		assert.Equal(t, "r2", resp.Data.Requests[0].ID)
	})

	t.Run("empty list serializes as [] not null ([@contract-v1] AC #3)", func(t *testing.T) {
		r := setupRequestRouter(&mockRequestService{})

		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/requests", nil))

		require.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"requests":[]`)
	})

	t.Run("service error → 500 envelope", func(t *testing.T) {
		r := setupRequestRouter(&mockRequestService{listErr: errors.New("boom")})

		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/requests", nil))
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestRequestHandler_CreateRequest(t *testing.T) {
	post := func(r *gin.Engine, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/requests", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		return w
	}

	t.Run("201 with the full request resource ([@contract-v1] AC #2)", func(t *testing.T) {
		created := &models.Request{ID: "new-id", TMDbID: 550, MediaType: "movie", Title: "鬥陣俱樂部", Status: "pending"}
		svc := &mockRequestService{createResp: created}
		r := setupRequestRouter(svc)

		w := post(r, `{"tmdb_id": 550, "media_type": "movie"}`)
		require.Equal(t, http.StatusCreated, w.Code)

		var resp struct {
			Success bool           `json:"success"`
			Data    models.Request `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.True(t, resp.Success)
		assert.Equal(t, "new-id", resp.Data.ID)
		assert.Equal(t, "pending", resp.Data.Status)
		assert.Equal(t, int64(550), svc.lastCreate.TMDbID, "snake_case body must bind")
		assert.Contains(t, w.Body.String(), `"fulfilment_source":null`, "nullable columns serialize as null")
	})

	t.Run("malformed body → 400 VALIDATION_INVALID_FORMAT", func(t *testing.T) {
		r := setupRequestRouter(&mockRequestService{})
		w := post(r, `{"tmdb_id": "not-a-number"}`)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "VALIDATION_INVALID_FORMAT")
	})

	t.Run("missing tmdb_id → 400 VALIDATION_REQUIRED_FIELD", func(t *testing.T) {
		svc := &mockRequestService{createErr: fmt.Errorf("validation: %w", &models.ValidationError{Field: "tmdb_id", Message: "tmdb_id is required and must be a positive integer"})}
		r := setupRequestRouter(svc)
		w := post(r, `{"media_type": "movie"}`)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "VALIDATION_REQUIRED_FIELD")
	})

	t.Run("duplicate → 409 REQUEST_DUPLICATE (AC #4)", func(t *testing.T) {
		svc := &mockRequestService{createErr: fmt.Errorf("tmdb_id 550: %w", repository.ErrRequestDuplicate)}
		r := setupRequestRouter(svc)
		w := post(r, `{"tmdb_id": 550, "media_type": "movie"}`)
		assert.Equal(t, http.StatusConflict, w.Code)
		assert.Contains(t, w.Body.String(), "REQUEST_DUPLICATE")
		assert.Contains(t, w.Body.String(), "已有進行中的請求")
	})

	t.Run("already owned → 409 REQUEST_ALREADY_IN_LIBRARY (AC #5)", func(t *testing.T) {
		svc := &mockRequestService{createErr: fmt.Errorf("tmdb_id 550: %w", services.ErrRequestAlreadyInLibrary)}
		r := setupRequestRouter(svc)
		w := post(r, `{"tmdb_id": 550, "media_type": "movie"}`)
		assert.Equal(t, http.StatusConflict, w.Code)
		assert.Contains(t, w.Body.String(), "REQUEST_ALREADY_IN_LIBRARY")
	})

	t.Run("unknown tmdb_id → typed TMDB_NOT_FOUND passes through with 404 (AC #2)", func(t *testing.T) {
		svc := &mockRequestService{createErr: fmt.Errorf("resolve tmdb target: %w", &tmdb.TMDbError{
			Code: tmdb.ErrCodeNotFound, Message: "資源不存在", StatusCode: http.StatusNotFound,
		})}
		r := setupRequestRouter(svc)
		w := post(r, `{"tmdb_id": 999999999, "media_type": "movie"}`)
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "TMDB_NOT_FOUND")
	})

	t.Run("unexpected error → 500", func(t *testing.T) {
		svc := &mockRequestService{createErr: errors.New("db exploded")}
		r := setupRequestRouter(svc)
		w := post(r, `{"tmdb_id": 550, "media_type": "movie"}`)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// --- 13-2a: selection wire + coverage endpoint ---

func TestRequestHandler_CreateRequest_SelectionPassthrough(t *testing.T) {
	// The optional seasons/episodes body fields reach the service DTO intact
	// ([@contract-v1] 13-2a AC #1).
	svc := &mockRequestService{createResp: &models.Request{ID: "r1"}}
	r := setupRequestRouter(svc)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/requests",
		bytes.NewBufferString(`{"tmdb_id": 1399, "media_type": "tv", "seasons": [1], "episodes": {"2": [5, 6]}}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, []int{1}, svc.lastCreate.Seasons)
	assert.Equal(t, map[string][]int{"2": {5, 6}}, svc.lastCreate.Episodes)
}

func TestRequestHandler_CreateRequest_InvalidSelectionMapsTo400(t *testing.T) {
	svc := &mockRequestService{createErr: &services.InvalidSelectionError{Reason: "此影集沒有第 9 季"}}
	r := setupRequestRouter(svc)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/requests",
		bytes.NewBufferString(`{"tmdb_id": 1399, "media_type": "tv", "seasons": [9]}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"REQUEST_INVALID_SELECTION"`)
	assert.Contains(t, w.Body.String(), "此影集沒有第 9 季",
		"the zh-TW reason renders WITHOUT the English sentinel text (Rule 3)")
	assert.NotContains(t, w.Body.String(), "invalid season/episode selection")
}

func TestRequestHandler_TVCoverage(t *testing.T) {
	t.Run("200 with the coverage shape ([@contract-v1] AC #5)", func(t *testing.T) {
		svc := &mockRequestService{coverageResp: &services.RequestCoverage{
			Owned:             map[string][]int{"1": {1, 2}},
			RequestedSeasons:  []int{3},
			RequestedEpisodes: map[string][]int{},
			ActiveRequest:     true,
		}}
		r := setupRequestRouter(svc)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/requests/tv/1399/coverage", nil))

		require.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, int64(1399), svc.lastCoverage)
		assert.Contains(t, w.Body.String(), `"owned":{"1":[1,2]}`)
		assert.Contains(t, w.Body.String(), `"requested_seasons":[3]`)
		assert.Contains(t, w.Body.String(), `"active_request":true`)
	})

	t.Run("non-numeric tmdb_id → 400", func(t *testing.T) {
		r := setupRequestRouter(&mockRequestService{})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/requests/tv/abc/coverage", nil))
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error → 500", func(t *testing.T) {
		r := setupRequestRouter(&mockRequestService{coverageErr: errors.New("boom")})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/requests/tv/1399/coverage", nil))
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// --- Story 13-7a: cancel / retry ---

func doRequestCall(r *gin.Engine, method, path string) (*httptest.ResponseRecorder, APIResponse) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	r.ServeHTTP(w, req)
	var resp APIResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	return w, resp
}

func TestRequestHandler_CancelRequest(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"pending → 204", nil, http.StatusNoContent, ""},
		{"unknown → 404", fmt.Errorf("x: %w", repository.ErrRequestNotFound), http.StatusNotFound, "DB_NOT_FOUND"},
		{"not pending → 409", fmt.Errorf("x: %w", services.ErrRequestNotCancellable), http.StatusConflict, "REQUEST_NOT_CANCELLABLE"},
		{"unexpected → 500", errors.New("disk"), http.StatusInternalServerError, "INTERNAL_ERROR"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &mockRequestService{cancelErr: tc.err}
			w, resp := doRequestCall(setupRequestRouter(svc), http.MethodDelete, "/api/v1/requests/req-1")
			assert.Equal(t, tc.status, w.Code)
			assert.Equal(t, "req-1", svc.lastCancel)
			if tc.code == "" {
				assert.Empty(t, w.Body.String(), "204 has no body")
				return
			}
			require.NotNil(t, resp.Error)
			assert.Equal(t, tc.code, resp.Error.Code)
			assert.NotEmpty(t, resp.Error.Message)
		})
	}
}

func TestRequestHandler_RetryRequest(t *testing.T) {
	t.Run("failed → 200 with the updated resource", func(t *testing.T) {
		svc := &mockRequestService{retryResp: &models.Request{ID: "req-1", Status: models.RequestStatusSearching, Title: "片"}}
		w, _ := doRequestCall(setupRequestRouter(svc), http.MethodPost, "/api/v1/requests/req-1/retry")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "req-1", svc.lastRetry)
		var body struct {
			Success bool `json:"success"`
			Data    struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		assert.True(t, body.Success)
		assert.Equal(t, "req-1", body.Data.ID)
		assert.Equal(t, "searching", body.Data.Status)
	})

	cases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"unknown → 404", fmt.Errorf("x: %w", repository.ErrRequestNotFound), http.StatusNotFound, "DB_NOT_FOUND"},
		{"not failed → 409", fmt.Errorf("x: %w", services.ErrRequestNotRetryable), http.StatusConflict, "REQUEST_NOT_RETRYABLE"},
		{"re-requested since → 409", fmt.Errorf("x: %w", repository.ErrRequestDuplicate), http.StatusConflict, "REQUEST_DUPLICATE"},
		{"*arr cleanup failed → 502", fmt.Errorf("x: %w", services.ErrRetryCleanupFailed), http.StatusBadGateway, "REQUEST_RETRY_CLEANUP_FAILED"},
		{"unexpected → 500", errors.New("disk"), http.StatusInternalServerError, "INTERNAL_ERROR"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &mockRequestService{retryErr: tc.err}
			w, resp := doRequestCall(setupRequestRouter(svc), http.MethodPost, "/api/v1/requests/req-1/retry")
			assert.Equal(t, tc.status, w.Code)
			require.NotNil(t, resp.Error)
			assert.Equal(t, tc.code, resp.Error.Code)
		})
	}
}
