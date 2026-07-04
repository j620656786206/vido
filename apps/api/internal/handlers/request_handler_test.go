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
	createResp *models.Request
	createErr  error
	listResp   []models.Request
	listErr    error
	lastCreate services.CreateMediaRequestRequest
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
