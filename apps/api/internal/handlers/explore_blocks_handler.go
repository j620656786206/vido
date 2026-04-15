package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/services"
)

// Error codes (Rule 7 — {SOURCE}_{ERROR_TYPE}).
const (
	errCodeExploreBlockNotFound  = "EXPLORE_BLOCK_NOT_FOUND"
	errCodeExploreBlockValidation = "EXPLORE_BLOCK_VALIDATION_FAILED"
)

// ExploreBlocksHandler handles HTTP requests for homepage explore block CRUD
// and content retrieval. Story 10.3.
type ExploreBlocksHandler struct {
	service services.ExploreBlockServiceInterface
}

// NewExploreBlocksHandler builds a new handler.
func NewExploreBlocksHandler(service services.ExploreBlockServiceInterface) *ExploreBlocksHandler {
	return &ExploreBlocksHandler{service: service}
}

// RegisterRoutes mounts the explore-block routes under the provided API group.
//
// NOTE: /reorder is registered BEFORE /:id so the literal path wins over the param route.
func (h *ExploreBlocksHandler) RegisterRoutes(rg *gin.RouterGroup) {
	blocks := rg.Group("/explore-blocks")
	{
		blocks.GET("", h.ListBlocks)
		blocks.POST("", h.CreateBlock)
		blocks.PUT("/reorder", h.ReorderBlocks)
		blocks.GET("/:id", h.GetBlock)
		blocks.PUT("/:id", h.UpdateBlock)
		blocks.DELETE("/:id", h.DeleteBlock)
		blocks.GET("/:id/content", h.GetBlockContent)
	}
}

// ListBlocks handles GET /api/v1/explore-blocks
// @Summary List explore blocks
// @Tags explore-blocks
// @Produce json
// @Success 200 {object} APIResponse{data=object}
// @Router /api/v1/explore-blocks [get]
func (h *ExploreBlocksHandler) ListBlocks(c *gin.Context) {
	blocks, err := h.service.GetAllBlocks(c.Request.Context())
	if err != nil {
		slog.Error("Failed to list explore blocks", "error", err)
		InternalServerError(c, "Failed to list explore blocks")
		return
	}
	// Never send null — the UI expects an array.
	if blocks == nil {
		blocks = []models.ExploreBlock{}
	}
	SuccessResponse(c, gin.H{"blocks": blocks})
}

// GetBlock handles GET /api/v1/explore-blocks/:id
func (h *ExploreBlocksHandler) GetBlock(c *gin.Context) {
	id := c.Param("id")
	block, err := h.service.GetBlock(c.Request.Context(), id)
	if err != nil {
		slog.Error("Failed to get explore block", "id", id, "error", err)
		handleExploreBlockError(c, err)
		return
	}
	SuccessResponse(c, block)
}

// CreateBlock handles POST /api/v1/explore-blocks
func (h *ExploreBlocksHandler) CreateBlock(c *gin.Context) {
	var req services.CreateExploreBlockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequestError(c, errCodeExploreBlockValidation, "Invalid request body: "+err.Error())
		return
	}

	block, err := h.service.CreateBlock(c.Request.Context(), req)
	if err != nil {
		slog.Error("Failed to create explore block", "error", err)
		handleExploreBlockError(c, err)
		return
	}
	CreatedResponse(c, block)
}

// UpdateBlock handles PUT /api/v1/explore-blocks/:id
func (h *ExploreBlocksHandler) UpdateBlock(c *gin.Context) {
	id := c.Param("id")

	var req services.UpdateExploreBlockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequestError(c, errCodeExploreBlockValidation, "Invalid request body: "+err.Error())
		return
	}

	block, err := h.service.UpdateBlock(c.Request.Context(), id, req)
	if err != nil {
		slog.Error("Failed to update explore block", "id", id, "error", err)
		handleExploreBlockError(c, err)
		return
	}
	SuccessResponse(c, block)
}

// DeleteBlock handles DELETE /api/v1/explore-blocks/:id
func (h *ExploreBlocksHandler) DeleteBlock(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.DeleteBlock(c.Request.Context(), id); err != nil {
		slog.Error("Failed to delete explore block", "id", id, "error", err)
		handleExploreBlockError(c, err)
		return
	}
	SuccessResponse(c, gin.H{"deleted": true})
}

// ReorderBlocks handles PUT /api/v1/explore-blocks/reorder with body
// {"ordered_ids": ["id1", "id2", ...]} and returns the updated list.
func (h *ExploreBlocksHandler) ReorderBlocks(c *gin.Context) {
	var req struct {
		OrderedIDs []string `json:"ordered_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequestError(c, errCodeExploreBlockValidation, "Invalid request body: ordered_ids is required")
		return
	}

	blocks, err := h.service.ReorderBlocks(c.Request.Context(), req.OrderedIDs)
	if err != nil {
		slog.Error("Failed to reorder explore blocks", "error", err)
		handleExploreBlockError(c, err)
		return
	}
	if blocks == nil {
		blocks = []models.ExploreBlock{}
	}
	SuccessResponse(c, gin.H{"blocks": blocks})
}

// GetBlockContent handles GET /api/v1/explore-blocks/:id/content
// Fetches TMDb discover results for the block (1-hour cache).
func (h *ExploreBlocksHandler) GetBlockContent(c *gin.Context) {
	id := c.Param("id")
	content, err := h.service.GetBlockContent(c.Request.Context(), id)
	if err != nil {
		slog.Error("Failed to get explore block content", "id", id, "error", err)
		handleExploreBlockError(c, err)
		return
	}
	SuccessResponse(c, content)
}

// handleExploreBlockError maps service errors to HTTP responses.
func handleExploreBlockError(c *gin.Context, err error) {
	if errors.Is(err, repository.ErrExploreBlockNotFound) {
		ErrorResponse(c, http.StatusNotFound, errCodeExploreBlockNotFound,
			"Explore block not found",
			"Verify the block ID is correct.")
		return
	}
	var validationErr *models.ValidationError
	if errors.As(err, &validationErr) {
		BadRequestError(c, errCodeExploreBlockValidation, err.Error())
		return
	}
	InternalServerError(c, err.Error())
}
