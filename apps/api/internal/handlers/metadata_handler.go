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

// UpdateMetadataRequestBody represents the request body for updating metadata (Story 3.8 - AC2)
type UpdateMetadataRequestBody struct {
	MediaType    string   `json:"mediaType"`
	Title        string   `json:"title"`
	TitleEnglish string   `json:"titleEnglish,omitempty"`
	Year         int      `json:"year"`
	Genres       []string `json:"genres,omitempty"`
	Director     string   `json:"director,omitempty"`
	Cast         []string `json:"cast,omitempty"`
	Overview     string   `json:"overview,omitempty"`
	PosterURL    string   `json:"posterUrl,omitempty"`
}

// UpdateMetadata handles PUT /api/v1/media/{id}/metadata (Story 3.8 - AC2)
// Allows users to manually edit all metadata fields for a media item
// @Summary Update media metadata
// @Description Manually update metadata for a movie or series (AC2)
// @Tags metadata
// @Accept json
// @Produce json
// @Param id path string true "Media ID"
// @Param request body UpdateMetadataRequestBody true "Update metadata request"
// @Success 200 {object} APIResponse{data=services.UpdateMetadataResponse}
// @Failure 400 {object} APIResponse{error=APIError}
// @Failure 404 {object} APIResponse{error=APIError}
// @Failure 500 {object} APIResponse{error=APIError}
// @Router /api/v1/media/{id}/metadata [put]
func (h *MetadataHandler) UpdateMetadata(c *gin.Context) {
	mediaID := c.Param("id")
	if mediaID == "" {
		ErrorResponse(c, http.StatusBadRequest, "METADATA_UPDATE_INVALID_REQUEST",
			"Media ID is required",
			"Please provide a valid media ID in the URL path")
		return
	}

	var req UpdateMetadataRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "METADATA_UPDATE_INVALID_REQUEST",
			"Invalid request body",
			"Please provide a valid JSON request")
		return
	}

	// Apply default media type
	if req.MediaType == "" {
		req.MediaType = "movie"
	}

	serviceReq := &services.UpdateMetadataRequest{
		ID:           mediaID,
		MediaType:    req.MediaType,
		Title:        req.Title,
		TitleEnglish: req.TitleEnglish,
		Year:         req.Year,
		Genres:       req.Genres,
		Director:     req.Director,
		Cast:         req.Cast,
		Overview:     req.Overview,
		PosterURL:    req.PosterURL,
	}

	result, err := h.service.UpdateMetadata(c.Request.Context(), serviceReq)
	if err != nil {
		if err == services.ErrUpdateMetadataTitleRequired || err == services.ErrUpdateMetadataYearRequired {
			ErrorResponse(c, http.StatusBadRequest, "VALIDATION_REQUIRED_FIELD",
				err.Error(),
				"Please provide all required fields (title, year)")
			return
		}
		if err == services.ErrUpdateMetadataNotFound {
			ErrorResponse(c, http.StatusNotFound, "METADATA_UPDATE_NOT_FOUND",
				"Media item not found",
				"Please verify the media ID is correct")
			return
		}
		ErrorResponse(c, http.StatusInternalServerError, "METADATA_UPDATE_FAILED",
			err.Error(),
			"Please try again later")
		return
	}

	SuccessResponse(c, result)
}

// UploadPoster handles POST /api/v1/media/{id}/poster (Story 3.8 - AC3)
// Allows users to upload a custom poster image for a media item
// @Summary Upload custom poster
// @Description Upload a custom poster image for a movie or series (AC3)
// @Tags metadata
// @Accept multipart/form-data
// @Produce json
// @Param id path string true "Media ID"
// @Param mediaType query string false "Media type: movie or series" default(movie)
// @Param file formData file true "Poster image file (jpg, png, webp, max 5MB)"
// @Success 200 {object} APIResponse{data=services.UploadPosterResponse}
// @Failure 400 {object} APIResponse{error=APIError}
// @Failure 404 {object} APIResponse{error=APIError}
// @Failure 500 {object} APIResponse{error=APIError}
// @Router /api/v1/media/{id}/poster [post]
func (h *MetadataHandler) UploadPoster(c *gin.Context) {
	mediaID := c.Param("id")
	if mediaID == "" {
		ErrorResponse(c, http.StatusBadRequest, "POSTER_UPLOAD_INVALID_REQUEST",
			"Media ID is required",
			"Please provide a valid media ID in the URL path")
		return
	}

	mediaType := c.DefaultQuery("mediaType", "movie")

	// Get the uploaded file
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		ErrorResponse(c, http.StatusBadRequest, "POSTER_UPLOAD_INVALID_REQUEST",
			"File is required",
			"Please upload an image file (jpg, png, or webp)")
		return
	}
	defer file.Close()

	// Read file data
	fileData := make([]byte, header.Size)
	_, err = file.Read(fileData)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "POSTER_UPLOAD_FAILED",
			"Failed to read file",
			"Please try again")
		return
	}

	// Detect content type from file header
	contentType := http.DetectContentType(fileData)

	serviceReq := &services.UploadPosterRequest{
		MediaID:     mediaID,
		MediaType:   mediaType,
		FileData:    fileData,
		FileName:    header.Filename,
		ContentType: contentType,
		FileSize:    header.Size,
	}

	result, err := h.service.UploadPoster(c.Request.Context(), serviceReq)
	if err != nil {
		if err == services.ErrPosterInvalidFormat {
			ErrorResponse(c, http.StatusBadRequest, "POSTER_INVALID_FORMAT",
				err.Error(),
				"Please upload a jpg, png, or webp image")
			return
		}
		if err == services.ErrPosterTooLarge {
			ErrorResponse(c, http.StatusBadRequest, "POSTER_TOO_LARGE",
				err.Error(),
				"Please upload an image smaller than 5MB")
			return
		}
		if err == services.ErrUploadPosterNotFound {
			ErrorResponse(c, http.StatusNotFound, "POSTER_UPLOAD_NOT_FOUND",
				"Media item not found",
				"Please verify the media ID is correct")
			return
		}
		ErrorResponse(c, http.StatusInternalServerError, "POSTER_UPLOAD_FAILED",
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

	// Media-centric routes for metadata editing (Story 3.8)
	mediaGroup := rg.Group("/media")
	{
		mediaGroup.PUT("/:id/metadata", h.UpdateMetadata)  // Story 3.8 - AC2
		mediaGroup.POST("/:id/poster", h.UploadPoster)     // Story 3.8 - AC3
	}
}
