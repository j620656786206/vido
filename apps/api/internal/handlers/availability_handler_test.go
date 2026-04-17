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
