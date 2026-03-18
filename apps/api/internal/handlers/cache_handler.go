package handlers

import (
	"errors"
	"log/slog"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/services"
)

// CacheHandler handles HTTP requests for cache management
type CacheHandler struct {
	statsService   services.CacheStatsServiceInterface
	cleanupService services.CacheCleanupServiceInterface
}

// NewCacheHandler creates a new CacheHandler
func NewCacheHandler(stats services.CacheStatsServiceInterface, cleanup services.CacheCleanupServiceInterface) *CacheHandler {
	return &CacheHandler{
		statsService:   stats,
		cleanupService: cleanup,
	}
}

// GetCacheStats handles GET /api/v1/settings/cache
func (h *CacheHandler) GetCacheStats(c *gin.Context) {
	stats, err := h.statsService.GetCacheStats(c.Request.Context())
	if err != nil {
		slog.Error("Failed to get cache stats", "error", err)
		InternalServerError(c, "Failed to retrieve cache statistics")
		return
	}

	SuccessResponse(c, stats)
}

// ClearAllCache handles DELETE /api/v1/settings/cache
func (h *CacheHandler) ClearAllCache(c *gin.Context) {
	olderThanDays := c.Query("older_than_days")

	if olderThanDays != "" {
		days, err := strconv.Atoi(olderThanDays)
		if err != nil || days <= 0 {
			BadRequestError(c, "VALIDATION_INVALID_FORMAT", "older_than_days must be a positive integer")
			return
		}

		result, err := h.cleanupService.ClearCacheByAge(c.Request.Context(), days)
		if err != nil {
			slog.Error("Failed to clear cache by age", "error", err, "days", days)
			InternalServerError(c, "Failed to clear cache")
			return
		}

		SuccessResponse(c, result)
		return
	}

	// No query param — clear all cache types
	var totalRemoved int64
	var totalBytes int64
	for _, cacheType := range services.ValidCacheTypes {
		result, err := h.cleanupService.ClearCacheByType(c.Request.Context(), cacheType)
		if err != nil {
			slog.Warn("Failed to clear cache type", "type", cacheType, "error", err)
			continue
		}
		totalRemoved += result.EntriesRemoved
		totalBytes += result.BytesReclaimed
	}

	SuccessResponse(c, &services.CleanupResult{
		Type:           "all",
		EntriesRemoved: totalRemoved,
		BytesReclaimed: totalBytes,
	})
}

// ClearCacheByType handles DELETE /api/v1/settings/cache/:type
func (h *CacheHandler) ClearCacheByType(c *gin.Context) {
	cacheType := c.Param("type")

	result, err := h.cleanupService.ClearCacheByType(c.Request.Context(), cacheType)
	if err != nil {
		if errors.Is(err, services.ErrInvalidCacheType) {
			BadRequestError(c, "CACHE_TYPE_INVALID", "Unknown cache type: "+cacheType)
			return
		}
		slog.Error("Failed to clear cache by type", "error", err, "type", cacheType)
		ErrorResponse(c, 500, "CACHE_CLEAR_FAILED", "Failed to clear cache", "Please try again later.")
		return
	}

	SuccessResponse(c, result)
}

// RegisterRoutes registers cache management routes
func (h *CacheHandler) RegisterRoutes(rg *gin.RouterGroup) {
	cache := rg.Group("/settings/cache")
	{
		cache.GET("", h.GetCacheStats)
		cache.DELETE("", h.ClearAllCache)
		cache.DELETE("/:type", h.ClearCacheByType)
	}
}
