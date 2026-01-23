package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/metadata"
	"github.com/vido/api/internal/services"
)

// MetadataHandler handles HTTP requests for metadata operations.
// It uses the multi-source fallback chain for metadata search.
type MetadataHandler struct {
	service services.MetadataServiceInterface
}

// NewMetadataHandler creates a new MetadataHandler with the given service.
func NewMetadataHandler(service services.MetadataServiceInterface) *MetadataHandler {
	return &MetadataHandler{
		service: service,
	}
}

// SearchMetadataResponse represents the search response format
type SearchMetadataResponse struct {
	Source         string                   `json:"source"`
	Results        []metadata.MetadataItem  `json:"results"`
	TotalCount     int                      `json:"totalCount"`
	Page           int                      `json:"page"`
	TotalPages     int                      `json:"totalPages"`
	FallbackStatus *metadata.FallbackStatus `json:"fallbackStatus,omitempty"`
}

// SearchMetadata handles GET /api/v1/metadata/search
// Searches for metadata using the multi-source fallback chain
// @Summary Search metadata across multiple sources
// @Description Search for movie/TV metadata using fallback chain (TMDb → Douban → Wikipedia)
// @Tags metadata
// @Accept json
// @Produce json
// @Param query query string true "Search query"
// @Param mediaType query string false "Media type: movie or tv" default(movie)
// @Param year query int false "Release year filter"
// @Param page query int false "Page number" default(1)
// @Param language query string false "Language code" default(zh-TW)
// @Success 200 {object} APIResponse{data=SearchMetadataResponse}
// @Failure 400 {object} APIResponse{error=APIError}
// @Failure 500 {object} APIResponse{error=APIError}
// @Router /api/v1/metadata/search [get]
func (h *MetadataHandler) SearchMetadata(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		ErrorResponse(c, http.StatusBadRequest, "METADATA_INVALID_REQUEST",
			"Search query is required",
			"Please provide a 'query' parameter")
		return
	}

	mediaType := c.DefaultQuery("mediaType", "movie")
	language := c.DefaultQuery("language", "zh-TW")

	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	year := 0
	if yearStr := c.Query("year"); yearStr != "" {
		if y, err := strconv.Atoi(yearStr); err == nil && y > 0 {
			year = y
		}
	}

	req := &services.SearchMetadataRequest{
		Query:     query,
		MediaType: mediaType,
		Year:      year,
		Page:      page,
		Language:  language,
	}

	result, fallbackStatus, err := h.service.SearchMetadata(c.Request.Context(), req)
	if err != nil {
		ErrorResponse(c, http.StatusBadRequest, "METADATA_INVALID_REQUEST",
			err.Error(),
			"Please check your request parameters")
		return
	}

	// Build response
	response := SearchMetadataResponse{
		Results:        []metadata.MetadataItem{},
		FallbackStatus: fallbackStatus,
	}

	if result != nil {
		response.Source = string(result.Source)
		response.Results = result.Items
		response.TotalCount = result.TotalCount
		response.Page = result.Page
		response.TotalPages = result.TotalPages
	}

	SuccessResponse(c, response)
}

// GetProviders handles GET /api/v1/metadata/providers
// Returns information about registered metadata providers
// @Summary Get metadata providers
// @Description Get list of registered metadata providers and their status
// @Tags metadata
// @Accept json
// @Produce json
// @Success 200 {object} APIResponse{data=[]services.ProviderInfo}
// @Router /api/v1/metadata/providers [get]
func (h *MetadataHandler) GetProviders(c *gin.Context) {
	providers := h.service.GetProviders()
	SuccessResponse(c, providers)
}

// RegisterRoutes registers all metadata routes on the given router group
func (h *MetadataHandler) RegisterRoutes(rg *gin.RouterGroup) {
	metadataGroup := rg.Group("/metadata")
	{
		metadataGroup.GET("/search", h.SearchMetadata)
		metadataGroup.GET("/providers", h.GetProviders)
	}
}
