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
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/services"
)

// --- Mock service ---

type mockExploreBlockService struct {
	getAllResp []models.ExploreBlock
	getAllErr  error

	getResp *models.ExploreBlock
	getErr  error

	createReq  services.CreateExploreBlockRequest
	createResp *models.ExploreBlock
	createErr  error

	updateID   string
	updateReq  services.UpdateExploreBlockRequest
	updateResp *models.ExploreBlock
	updateErr  error

	deleteID  string
	deleteErr error

	reorderIDs  []string
	reorderResp []models.ExploreBlock
	reorderErr  error

	seedErr error

	contentID   string
	contentResp *services.ExploreBlockContent
	contentErr  error
}

func (m *mockExploreBlockService) GetAllBlocks(ctx context.Context) ([]models.ExploreBlock, error) {
	return m.getAllResp, m.getAllErr
}
func (m *mockExploreBlockService) GetBlock(ctx context.Context, id string) (*models.ExploreBlock, error) {
	return m.getResp, m.getErr
}
func (m *mockExploreBlockService) CreateBlock(ctx context.Context, req services.CreateExploreBlockRequest) (*models.ExploreBlock, error) {
	m.createReq = req
	return m.createResp, m.createErr
}
func (m *mockExploreBlockService) UpdateBlock(ctx context.Context, id string, req services.UpdateExploreBlockRequest) (*models.ExploreBlock, error) {
	m.updateID = id
	m.updateReq = req
	return m.updateResp, m.updateErr
}
func (m *mockExploreBlockService) DeleteBlock(ctx context.Context, id string) error {
	m.deleteID = id
	return m.deleteErr
}
func (m *mockExploreBlockService) ReorderBlocks(ctx context.Context, orderedIDs []string) ([]models.ExploreBlock, error) {
	m.reorderIDs = orderedIDs
	return m.reorderResp, m.reorderErr
}
func (m *mockExploreBlockService) SeedDefaultsIfEmpty(ctx context.Context) error { return m.seedErr }
func (m *mockExploreBlockService) GetBlockContent(ctx context.Context, id string) (*services.ExploreBlockContent, error) {
	m.contentID = id
	return m.contentResp, m.contentErr
}

var _ services.ExploreBlockServiceInterface = (*mockExploreBlockService)(nil)

func newExploreBlocksRouter(svc services.ExploreBlockServiceInterface) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewExploreBlocksHandler(svc)
	api := r.Group("/api/v1")
	h.RegisterRoutes(api)
	return r
}

func doJSON(t *testing.T, r *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func parseEnvelope(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var env map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	return env
}

// --- List ---

func TestExploreBlocksHandler_List_Empty(t *testing.T) {
	mock := &mockExploreBlockService{getAllResp: nil}
	r := newExploreBlocksRouter(mock)

	rec := doJSON(t, r, http.MethodGet, "/api/v1/explore-blocks", nil)
	assert.Equal(t, http.StatusOK, rec.Code)

	env := parseEnvelope(t, rec)
	assert.Equal(t, true, env["success"])
	data := env["data"].(map[string]any)
	blocks, ok := data["blocks"].([]any)
	require.True(t, ok, "blocks must be array (not null) per API contract")
	assert.Empty(t, blocks)
}

func TestExploreBlocksHandler_List_WithBlocks(t *testing.T) {
	mock := &mockExploreBlockService{
		getAllResp: []models.ExploreBlock{
			{ID: "1", Name: "A", ContentType: models.ExploreBlockContentMovie},
			{ID: "2", Name: "B", ContentType: models.ExploreBlockContentTV},
		},
	}
	r := newExploreBlocksRouter(mock)

	rec := doJSON(t, r, http.MethodGet, "/api/v1/explore-blocks", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	env := parseEnvelope(t, rec)
	data := env["data"].(map[string]any)
	blocks := data["blocks"].([]any)
	assert.Len(t, blocks, 2)
}

func TestExploreBlocksHandler_List_ServiceError(t *testing.T) {
	mock := &mockExploreBlockService{getAllErr: errors.New("db down")}
	r := newExploreBlocksRouter(mock)

	rec := doJSON(t, r, http.MethodGet, "/api/v1/explore-blocks", nil)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// --- Get by ID ---

func TestExploreBlocksHandler_Get_NotFound(t *testing.T) {
	mock := &mockExploreBlockService{getErr: repository.ErrExploreBlockNotFound}
	r := newExploreBlocksRouter(mock)

	rec := doJSON(t, r, http.MethodGet, "/api/v1/explore-blocks/missing", nil)
	assert.Equal(t, http.StatusNotFound, rec.Code)
	env := parseEnvelope(t, rec)
	errObj := env["error"].(map[string]any)
	assert.Equal(t, "EXPLORE_BLOCK_NOT_FOUND", errObj["code"])
}

// --- Create ---

func TestExploreBlocksHandler_Create_Success(t *testing.T) {
	mock := &mockExploreBlockService{
		createResp: &models.ExploreBlock{
			ID:          "new-id",
			Name:        "熱門電影",
			ContentType: models.ExploreBlockContentMovie,
			MaxItems:    20,
		},
	}
	r := newExploreBlocksRouter(mock)

	rec := doJSON(t, r, http.MethodPost, "/api/v1/explore-blocks", map[string]any{
		"name":         "熱門電影",
		"content_type": "movie",
		"max_items":    20,
	})
	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, "熱門電影", mock.createReq.Name)
	assert.Equal(t, "movie", mock.createReq.ContentType)
	assert.Equal(t, 20, mock.createReq.MaxItems)
}

func TestExploreBlocksHandler_Create_ValidationError(t *testing.T) {
	mock := &mockExploreBlockService{
		createErr: &models.ValidationError{Field: "name", Message: "block name is required"},
	}
	r := newExploreBlocksRouter(mock)

	rec := doJSON(t, r, http.MethodPost, "/api/v1/explore-blocks", map[string]any{
		"name":         "",
		"content_type": "movie",
	})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	env := parseEnvelope(t, rec)
	errObj := env["error"].(map[string]any)
	assert.Equal(t, "EXPLORE_BLOCK_VALIDATION_FAILED", errObj["code"])
}

func TestExploreBlocksHandler_Create_BadJSON(t *testing.T) {
	mock := &mockExploreBlockService{}
	r := newExploreBlocksRouter(mock)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/explore-blocks", bytes.NewBufferString("{not-json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- Update ---

func TestExploreBlocksHandler_Update_Success(t *testing.T) {
	mock := &mockExploreBlockService{
		updateResp: &models.ExploreBlock{ID: "abc", Name: "已改", ContentType: models.ExploreBlockContentMovie, MaxItems: 20},
	}
	r := newExploreBlocksRouter(mock)

	rec := doJSON(t, r, http.MethodPut, "/api/v1/explore-blocks/abc", map[string]any{
		"name": "已改",
	})
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "abc", mock.updateID)
	require.NotNil(t, mock.updateReq.Name)
	assert.Equal(t, "已改", *mock.updateReq.Name)
}

func TestExploreBlocksHandler_Update_NotFound(t *testing.T) {
	mock := &mockExploreBlockService{updateErr: repository.ErrExploreBlockNotFound}
	r := newExploreBlocksRouter(mock)

	rec := doJSON(t, r, http.MethodPut, "/api/v1/explore-blocks/xxx", map[string]any{"name": "x"})
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// --- Delete ---

func TestExploreBlocksHandler_Delete_Success(t *testing.T) {
	mock := &mockExploreBlockService{}
	r := newExploreBlocksRouter(mock)

	rec := doJSON(t, r, http.MethodDelete, "/api/v1/explore-blocks/del-me", nil)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "del-me", mock.deleteID)
}

func TestExploreBlocksHandler_Delete_NotFound(t *testing.T) {
	mock := &mockExploreBlockService{deleteErr: repository.ErrExploreBlockNotFound}
	r := newExploreBlocksRouter(mock)

	rec := doJSON(t, r, http.MethodDelete, "/api/v1/explore-blocks/missing", nil)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// --- Reorder ---

func TestExploreBlocksHandler_Reorder_Success(t *testing.T) {
	mock := &mockExploreBlockService{
		reorderResp: []models.ExploreBlock{
			{ID: "2", Name: "B", ContentType: models.ExploreBlockContentMovie},
			{ID: "1", Name: "A", ContentType: models.ExploreBlockContentMovie},
		},
	}
	r := newExploreBlocksRouter(mock)

	rec := doJSON(t, r, http.MethodPut, "/api/v1/explore-blocks/reorder", map[string]any{
		"ordered_ids": []string{"2", "1"},
	})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, []string{"2", "1"}, mock.reorderIDs)

	env := parseEnvelope(t, rec)
	data := env["data"].(map[string]any)
	blocks := data["blocks"].([]any)
	require.Len(t, blocks, 2)
	assert.Equal(t, "B", blocks[0].(map[string]any)["name"])
}

func TestExploreBlocksHandler_Reorder_MissingIDs(t *testing.T) {
	mock := &mockExploreBlockService{}
	r := newExploreBlocksRouter(mock)

	rec := doJSON(t, r, http.MethodPut, "/api/v1/explore-blocks/reorder", map[string]any{})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestExploreBlocksHandler_Reorder_UnknownIDFromService(t *testing.T) {
	mock := &mockExploreBlockService{reorderErr: repository.ErrExploreBlockNotFound}
	r := newExploreBlocksRouter(mock)

	rec := doJSON(t, r, http.MethodPut, "/api/v1/explore-blocks/reorder", map[string]any{
		"ordered_ids": []string{"unknown"},
	})
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// --- Content ---

func TestExploreBlocksHandler_Content_Success(t *testing.T) {
	mock := &mockExploreBlockService{
		contentResp: &services.ExploreBlockContent{
			BlockID:     "abc",
			ContentType: "movie",
			TotalItems:  1,
		},
	}
	r := newExploreBlocksRouter(mock)

	rec := doJSON(t, r, http.MethodGet, "/api/v1/explore-blocks/abc/content", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "abc", mock.contentID)
}

func TestExploreBlocksHandler_Content_NotFound(t *testing.T) {
	mock := &mockExploreBlockService{contentErr: repository.ErrExploreBlockNotFound}
	r := newExploreBlocksRouter(mock)

	rec := doJSON(t, r, http.MethodGet, "/api/v1/explore-blocks/missing/content", nil)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// --- Route ordering regression: /reorder must not collide with /:id ---

func TestExploreBlocksHandler_ReorderRouteDoesNotCollideWithID(t *testing.T) {
	mock := &mockExploreBlockService{
		reorderResp: []models.ExploreBlock{},
	}
	r := newExploreBlocksRouter(mock)

	rec := doJSON(t, r, http.MethodPut, "/api/v1/explore-blocks/reorder", map[string]any{
		"ordered_ids": []string{},
	})
	require.Equal(t, http.StatusOK, rec.Code, "PUT /reorder must hit ReorderBlocks, not UpdateBlock with id=reorder")
	assert.Nil(t, mock.updateResp, "UpdateBlock must NOT be invoked when id='reorder'")
	assert.Empty(t, mock.updateID)
}
