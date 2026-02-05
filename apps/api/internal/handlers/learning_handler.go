package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/services"
)

// LearningHandler handles HTTP requests for filename pattern learning.
// It uses services.LearningServiceInterface for business logic, following the
// Handler → Service → Repository → Database architecture.
type LearningHandler struct {
	service services.LearningServiceInterface
}

// NewLearningHandler creates a new LearningHandler with the given service.
func NewLearningHandler(service services.LearningServiceInterface) *LearningHandler {
	return &LearningHandler{
		service: service,
	}
}

// CreatePatternRequest represents the request body for learning a new pattern
type CreatePatternRequest struct {
	Filename     string `json:"filename" binding:"required"`
	MetadataID   string `json:"metadataId" binding:"required"`
	MetadataType string `json:"metadataType" binding:"required,oneof=movie series"`
	TmdbID       int    `json:"tmdbId,omitempty"`
}

// PatternListResponse represents the response for listing patterns
type PatternListResponse struct {
	Patterns   []*models.FilenameMapping `json:"patterns"`
	TotalCount int                         `json:"totalCount"`
	Stats      *services.PatternStats      `json:"stats,omitempty"`
}

// Create handles POST /api/v1/learning/patterns
// Creates a new learned pattern from a filename correction
func (h *LearningHandler) Create(c *gin.Context) {
	var req CreatePatternRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ValidationError(c, "Invalid request body: "+err.Error())
		return
	}

	slog.Info("Learning pattern from correction",
		"filename", req.Filename,
		"metadataId", req.MetadataID,
		"metadataType", req.MetadataType,
	)

	serviceReq := services.LearnFromCorrectionRequest{
		Filename:     req.Filename,
		MetadataID:   req.MetadataID,
		MetadataType: req.MetadataType,
		TmdbID:       req.TmdbID,
	}

	pattern, err := h.service.LearnFromCorrection(c.Request.Context(), serviceReq)
	if err != nil {
		slog.Error("Failed to learn pattern", "error", err, "filename", req.Filename)
		ErrorResponse(c, http.StatusInternalServerError, "LEARNING_SAVE_FAILED",
			"無法學習此規則",
			"請稍後再試，或檢查檔案名稱格式。")
		return
	}

	slog.Info("Pattern learned successfully",
		"patternId", pattern.ID,
		"pattern", pattern.Pattern,
	)

	CreatedResponse(c, pattern)
}

// List handles GET /api/v1/learning/patterns
// Returns all learned patterns with statistics
func (h *LearningHandler) List(c *gin.Context) {
	ctx := c.Request.Context()

	patterns, err := h.service.ListPatterns(ctx)
	if err != nil {
		slog.Error("Failed to list patterns", "error", err)
		ErrorResponse(c, http.StatusInternalServerError, "LEARNING_QUERY_FAILED",
			"無法取得學習規則清單",
			"請稍後再試。")
		return
	}

	// Get stats (don't fail if stats retrieval fails)
	stats, err := h.service.GetPatternStats(ctx)
	if err != nil {
		slog.Warn("Failed to get pattern stats", "error", err)
		// Continue without stats
	}

	response := PatternListResponse{
		Patterns:   patterns,
		TotalCount: len(patterns),
		Stats:      stats,
	}

	SuccessResponse(c, response)
}

// Delete handles DELETE /api/v1/learning/patterns/:id
// Removes a learned pattern by ID
func (h *LearningHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		BadRequestError(c, "VALIDATION_REQUIRED_FIELD", "Pattern ID is required")
		return
	}

	slog.Info("Deleting pattern", "patternId", id)

	if err := h.service.DeletePattern(c.Request.Context(), id); err != nil {
		slog.Error("Failed to delete pattern", "error", err, "patternId", id)
		ErrorResponse(c, http.StatusInternalServerError, "LEARNING_DELETE_FAILED",
			"無法刪除此規則",
			"請確認規則 ID 是否正確，或稍後再試。")
		return
	}

	NoContentResponse(c)
}

// GetStats handles GET /api/v1/learning/stats
// Returns statistics about learned patterns
func (h *LearningHandler) GetStats(c *gin.Context) {
	stats, err := h.service.GetPatternStats(c.Request.Context())
	if err != nil {
		slog.Error("Failed to get pattern stats", "error", err)
		ErrorResponse(c, http.StatusInternalServerError, "LEARNING_QUERY_FAILED",
			"無法取得學習統計資訊",
			"請稍後再試。")
		return
	}

	SuccessResponse(c, stats)
}

// RegisterRoutes registers all learning routes on the given router group
func (h *LearningHandler) RegisterRoutes(rg *gin.RouterGroup) {
	learning := rg.Group("/learning")
	{
		learning.POST("/patterns", h.Create)
		learning.GET("/patterns", h.List)
		learning.DELETE("/patterns/:id", h.Delete)
		learning.GET("/stats", h.GetStats)
	}
}
