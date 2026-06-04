package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/services"
	"github.com/vido/api/internal/tmdb"
)

// mockUnifiedSearchService is a mock UnifiedSearchServiceInterface.
type mockUnifiedSearchService struct {
	result *services.UnifiedSearchResult
	err    error
	calls  []string // captured queries
}

func (m *mockUnifiedSearchService) Search(_ context.Context, query string, _ int) (*services.UnifiedSearchResult, error) {
	m.calls = append(m.calls, query)
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

func setupSearchRouter(h *SearchHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := r.Group("/api/v1")
	h.RegisterRoutes(api)
	return r
}

func TestSearchHandler_MissingQuery_Returns400(t *testing.T) {
	mock := &mockUnifiedSearchService{}
	router := setupSearchRouter(NewSearchHandler(mock))

	rec := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/search", nil)
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var resp APIResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.False(t, resp.Success)
	// service must NOT be called when q is missing
	assert.Empty(t, mock.calls)
}

func TestSearchHandler_BlankQuery_Returns400(t *testing.T) {
	mock := &mockUnifiedSearchService{}
	router := setupSearchRouter(NewSearchHandler(mock))

	rec := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/search?q=%20%20", nil)
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Empty(t, mock.calls)
}

func TestSearchHandler_Success(t *testing.T) {
	mock := &mockUnifiedSearchService{
		result: &services.UnifiedSearchResult{
			Query:   "iron",
			Page:    1,
			Movies:  []tmdb.Movie{{ID: 1, Title: "鋼鐵人"}},
			TVShows: []tmdb.TVShow{},
			People:  []tmdb.Person{},
		},
	}
	router := setupSearchRouter(NewSearchHandler(mock))

	rec := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/search?q=iron&page=1", nil)
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp APIResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.Equal(t, []string{"iron"}, mock.calls)
}

func TestSearchHandler_ServiceError_Returns500(t *testing.T) {
	mock := &mockUnifiedSearchService{err: errors.New("tmdb unavailable")}
	router := setupSearchRouter(NewSearchHandler(mock))

	rec := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/search?q=iron", nil)
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	var resp APIResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.False(t, resp.Success)
}
