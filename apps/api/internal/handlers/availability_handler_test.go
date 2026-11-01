package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vido/api/internal/services"
)

// MockAvailabilityService satisfies services.AvailabilityServiceInterface for
// handler-level tests — mirrors the mock pattern used in movie_handler_test.go.
type MockAvailabilityService struct {
	mock.Mock
}

func (m *MockAvailabilityService) CheckOwned(ctx context.Context, tmdbIDs []int64) ([]int64, error) {
	args := m.Called(ctx, tmdbIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]int64), args.Error(1)
}

func (m *MockAvailabilityService) CheckOwnedByType(ctx context.Context, mediaType string, tmdbIDs []int64) ([]int64, error) {
	args := m.Called(ctx, mediaType, tmdbIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]int64), args.Error(1)
}

func (m *MockAvailabilityService) LibraryID(ctx context.Context, mediaType string, tmdbID int64) (string, error) {
	args := m.Called(ctx, mediaType, tmdbID)
	return args.String(0), args.Error(1)
}

var _ services.AvailabilityServiceInterface = (*MockAvailabilityService)(nil)

func setupAvailabilityRouter(handler *AvailabilityHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	apiV1 := router.Group("/api/v1")
	handler.RegisterRoutes(apiV1)
	return router
}

func TestAvailabilityHandler_CheckOwned(t *testing.T) {
	t.Run("returns owned_ids subset on success (AC #1, #4)", func(t *testing.T) {
		svc := new(MockAvailabilityService)
		svc.On("CheckOwned", mock.Anything, []int64{603, 157336, 999999}).
			Return([]int64{603, 157336}, nil)

		handler := NewAvailabilityHandler(svc)
		router := setupAvailabilityRouter(handler)

		body, _ := json.Marshal(CheckOwnedRequest{TMDbIDs: []int64{603, 157336, 999999}})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/media/check-owned", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Success bool               `json:"success"`
			Data    CheckOwnedResponse `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		assert.True(t, resp.Success)
		assert.ElementsMatch(t, []int64{603, 157336}, resp.Data.OwnedIDs)
		svc.AssertExpectations(t)
	})

	t.Run("empty array request is valid — binding 'required' only rejects missing field", func(t *testing.T) {
		svc := new(MockAvailabilityService)
		svc.On("CheckOwned", mock.Anything, []int64{}).Return([]int64{}, nil)

		handler := NewAvailabilityHandler(svc)
		router := setupAvailabilityRouter(handler)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/media/check-owned",
			bytes.NewBufferString(`{"tmdb_ids": []}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("missing tmdb_ids field returns 400", func(t *testing.T) {
		svc := new(MockAvailabilityService)
		handler := NewAvailabilityHandler(svc)
		router := setupAvailabilityRouter(handler)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/media/check-owned",
			bytes.NewBufferString(`{}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		svc.AssertNotCalled(t, "CheckOwned")
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		svc := new(MockAvailabilityService)
		handler := NewAvailabilityHandler(svc)
		router := setupAvailabilityRouter(handler)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/media/check-owned",
			bytes.NewBufferString(`{"tmdb_ids": not-json`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		svc.AssertNotCalled(t, "CheckOwned")
	})

	t.Run("over-limit request returns 400 without calling service", func(t *testing.T) {
		svc := new(MockAvailabilityService)
		handler := NewAvailabilityHandler(svc)
		router := setupAvailabilityRouter(handler)

		over := make([]int64, availabilityCheckOwnedMaxIDs+1)
		for i := range over {
			over[i] = int64(i + 1)
		}
		body, _ := json.Marshal(CheckOwnedRequest{TMDbIDs: over})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/media/check-owned", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		svc.AssertNotCalled(t, "CheckOwned")
	})

	t.Run("service error returns 500", func(t *testing.T) {
		svc := new(MockAvailabilityService)
		svc.On("CheckOwned", mock.Anything, mock.Anything).
			Return(nil, errors.New("db down"))

		handler := NewAvailabilityHandler(svc)
		router := setupAvailabilityRouter(handler)

		body, _ := json.Marshal(CheckOwnedRequest{TMDbIDs: []int64{603}})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/media/check-owned", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("nil owned result is normalized to empty array in JSON", func(t *testing.T) {
		svc := new(MockAvailabilityService)
		// Service returns nil, nil — handler must normalise to [] so the
		// frontend never receives `owned_ids: null`.
		svc.On("CheckOwned", mock.Anything, []int64{111}).Return(nil, nil)

		handler := NewAvailabilityHandler(svc)
		router := setupAvailabilityRouter(handler)

		body, _ := json.Marshal(CheckOwnedRequest{TMDbIDs: []int64{111}})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/media/check-owned", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"owned_ids":[]`)
	})
}

func TestAvailabilityHandler_LibraryID(t *testing.T) {
	get := func(t *testing.T, svc *MockAvailabilityService, query string) (int, map[string]any) {
		t.Helper()
		router := setupAvailabilityRouter(NewAvailabilityHandler(svc))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/media/library-id?"+query, nil))
		var body map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &body)
		return w.Code, body
	}

	t.Run("owned title → its library id", func(t *testing.T) {
		svc := new(MockAvailabilityService)
		svc.On("LibraryID", mock.Anything, "tv", int64(93740)).Return("series-uuid", nil)
		code, body := get(t, svc, "type=tv&tmdb_id=93740")
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, map[string]any{"id": "series-uuid"}, body["data"])
	})

	t.Run("not in the library → id null, still 200", func(t *testing.T) {
		svc := new(MockAvailabilityService)
		svc.On("LibraryID", mock.Anything, "movie", int64(603)).Return("", nil)
		code, body := get(t, svc, "type=movie&tmdb_id=603")
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, map[string]any{"id": nil}, body["data"])
	})

	t.Run("bad type or tmdb_id → 400, service not called", func(t *testing.T) {
		for _, q := range []string{"type=book&tmdb_id=1", "type=tv&tmdb_id=abc", "type=tv&tmdb_id=0", "type=tv"} {
			svc := new(MockAvailabilityService)
			code, _ := get(t, svc, q)
			assert.Equal(t, http.StatusBadRequest, code, q)
			svc.AssertNotCalled(t, "LibraryID", mock.Anything, mock.Anything, mock.Anything)
		}
	})

	t.Run("lookup error → 500", func(t *testing.T) {
		svc := new(MockAvailabilityService)
		svc.On("LibraryID", mock.Anything, "tv", int64(1)).Return("", errors.New("db down"))
		code, _ := get(t, svc, "type=tv&tmdb_id=1")
		assert.Equal(t, http.StatusInternalServerError, code)
	})
}
