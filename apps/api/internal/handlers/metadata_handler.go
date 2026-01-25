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

// ManualSearchRequest represents the request body for manual search
type ManualSearchRequest struct {
	Query     string `json:"query"`
	MediaType string `json:"mediaType"`
	Year      int    `json:"year,omitempty"`
	Source    string `json:"source"`
}

// ManualSearch handles POST /api/v1/metadata/manual-search (Story 3.7)
// Allows users to manually search for metadata across selected sources
// @Summary Manual metadata search
// @Description Search for movie/TV metadata on specific source(s) (AC1, AC4)
// @Tags metadata
// @Accept json
// @Produce json
// @Param request body ManualSearchRequest true "Manual search request"
// @Success 200 {object} APIResponse{data=services.ManualSearchResponse}
// @Failure 400 {object} APIResponse{error=APIError}
// @Failure 500 {object} APIResponse{error=APIError}
// @Router /api/v1/metadata/manual-search [post]
func (h *MetadataHandler) ManualSearch(c *gin.Context) {
	var req ManualSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "MANUAL_SEARCH_INVALID_REQUEST",
			"Invalid request body",
			"Please provide a valid JSON request with 'query' field")
		return
	}

	// Validate query is present
	if req.Query == "" {
		ErrorResponse(c, http.StatusBadRequest, "MANUAL_SEARCH_INVALID_REQUEST",
			"Search query is required",
			"Please provide a 'query' parameter")
		return
	}

	// Apply defaults
	if req.MediaType == "" {
		req.MediaType = "movie"
	}
	if req.Source == "" {
		req.Source = "all"
	}

	serviceReq := &services.ManualSearchRequest{
		Query:     req.Query,
		MediaType: req.MediaType,
		Year:      req.Year,
		Source:    req.Source,
	}

	result, err := h.service.ManualSearch(c.Request.Context(), serviceReq)
	if err != nil {
		// Check for specific errors
		if err == services.ErrManualSearchInvalidSource {
			ErrorResponse(c, http.StatusBadRequest, "MANUAL_SEARCH_INVALID_SOURCE",
				err.Error(),
				"Valid sources: 'tmdb', 'douban', 'wikipedia', or 'all'")
			return
		}
		ErrorResponse(c, http.StatusBadRequest, "MANUAL_SEARCH_INVALID_REQUEST",
			err.Error(),
			"Please check your request parameters")
		return
	}

	SuccessResponse(c, result)
}

// ApplyMetadataRequestBody represents the request body for applying metadata (Story 3.7 - AC3)
type ApplyMetadataRequestBody struct {
	MediaID      string `json:"mediaId"`
	MediaType    string `json:"mediaType"`
	SelectedItem struct {
		ID     string `json:"id"`
		Source string `json:"source"`
	} `json:"selectedItem"`
	LearnPattern bool `json:"learnPattern,omitempty"`
}

// ApplyMetadata handles POST /api/v1/metadata/apply (Story 3.7 - AC3)
// Applies selected metadata from manual search to a media item
// @Summary Apply metadata to media
// @Description Apply selected metadata from search results to a movie or series (AC3)
// @Tags metadata
// @Accept json
// @Produce json
// @Param request body ApplyMetadataRequestBody true "Apply metadata request"
// @Success 200 {object} APIResponse{data=services.ApplyMetadataResponse}
// @Failure 400 {object} APIResponse{error=APIError}
// @Failure 404 {object} APIResponse{error=APIError}
// @Failure 500 {object} APIResponse{error=APIError}
// @Router /api/v1/metadata/apply [post]
func (h *MetadataHandler) ApplyMetadata(c *gin.Context) {
	var req ApplyMetadataRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "APPLY_METADATA_INVALID_REQUEST",
			"Invalid request body",
			"Please provide a valid JSON request")
		return
	}

	// Validate required fields
	if req.MediaID == "" {
		ErrorResponse(c, http.StatusBadRequest, "APPLY_METADATA_INVALID_REQUEST",
			"mediaId is required",
			"Please provide the media item ID")
		return
	}

	if req.SelectedItem.ID == "" || req.SelectedItem.Source == "" {
		ErrorResponse(c, http.StatusBadRequest, "APPLY_METADATA_INVALID_REQUEST",
			"selectedItem with id and source is required",
			"Please provide the selected metadata item")
		return
	}

	// Apply defaults
	if req.MediaType == "" {
		req.MediaType = "movie"
	}

	serviceReq := &services.ApplyMetadataRequest{
		MediaID:   req.MediaID,
		MediaType: req.MediaType,
		SelectedItem: services.SelectedMetadataItem{
			ID:     req.SelectedItem.ID,
			Source: req.SelectedItem.Source,
		},
		LearnPattern: req.LearnPattern,
	}

	result, err := h.service.ApplyMetadata(c.Request.Context(), serviceReq)
	if err != nil {
		if err == services.ErrApplyMetadataNotFound {
			ErrorResponse(c, http.StatusNotFound, "APPLY_METADATA_NOT_FOUND",
				"Media item not found",
				"Please verify the media ID is correct")
			return
		}
		ErrorResponse(c, http.StatusInternalServerError, "APPLY_METADATA_FAILED",
			err.Error(),
			"Please try again later")
		return
	}

	SuccessResponse(c, result)
}

// RegisterRoutes registers all metadata routes on the given router group
func (h *MetadataHandler) RegisterRoutes(rg *gin.RouterGroup) {
	metadataGroup := rg.Group("/metadata")
	{
		metadataGroup.GET("/search", h.SearchMetadata)
		metadataGroup.GET("/providers", h.GetProviders)
		metadataGroup.POST("/manual-search", h.ManualSearch) // Story 3.7
		metadataGroup.POST("/apply", h.ApplyMetadata)        // Story 3.7 - AC3
	}
}
