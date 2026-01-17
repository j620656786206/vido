package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/parser"
	"github.com/vido/api/internal/services"
)

// ParseRequest represents the request body for parsing a single filename.
type ParseRequest struct {
	Filename string `json:"filename" binding:"required"`
}

// ParseBatchRequest represents the request body for parsing multiple filenames.
type ParseBatchRequest struct {
	Filenames []string `json:"filenames" binding:"required"`
}

// ParserHandler handles HTTP requests for filename parsing operations.
type ParserHandler struct {
	service services.ParserServiceInterface
}

// NewParserHandler creates a new ParserHandler with the given service.
func NewParserHandler(service services.ParserServiceInterface) *ParserHandler {
	return &ParserHandler{
		service: service,
	}
}

// Parse handles POST /api/v1/parser/parse
// Parses a single filename and returns extracted metadata
// @Summary Parse a single filename
// @Description Parse a media filename and extract metadata (title, year, quality, etc.)
// @Tags parser
// @Accept json
// @Produce json
// @Param request body ParseRequest true "Filename to parse"
// @Success 200 {object} APIResponse{data=parser.ParseResult}
// @Failure 400 {object} APIResponse{error=APIError}
// @Router /api/v1/parser/parse [post]
func (h *ParserHandler) Parse(c *gin.Context) {
	var req ParseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "VALIDATION_INVALID_FORMAT",
			"Invalid request format",
			"Please provide a valid JSON body with a 'filename' field")
		return
	}

	if req.Filename == "" {
		ErrorResponse(c, http.StatusBadRequest, "VALIDATION_REQUIRED_FIELD",
			"Filename is required",
			"Please provide a 'filename' in the request body")
		return
	}

	result := h.service.ParseFilename(req.Filename)
	SuccessResponse(c, result)
}

// ParseBatch handles POST /api/v1/parser/parse-batch
// Parses multiple filenames and returns results for each
// @Summary Parse multiple filenames
// @Description Parse multiple media filenames and extract metadata for each
// @Tags parser
// @Accept json
// @Produce json
// @Param request body ParseBatchRequest true "Filenames to parse"
// @Success 200 {object} APIResponse{data=[]parser.ParseResult}
// @Failure 400 {object} APIResponse{error=APIError}
// @Router /api/v1/parser/parse-batch [post]
func (h *ParserHandler) ParseBatch(c *gin.Context) {
	var req ParseBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "VALIDATION_INVALID_FORMAT",
			"Invalid request format",
			"Please provide a valid JSON body with a 'filenames' array")
		return
	}

	if len(req.Filenames) == 0 {
		ErrorResponse(c, http.StatusBadRequest, "VALIDATION_REQUIRED_FIELD",
			"At least one filename is required",
			"Please provide a non-empty 'filenames' array in the request body")
		return
	}

	results := h.service.ParseBatch(req.Filenames)
	SuccessResponse(c, results)
}

// RegisterRoutes registers parser routes on the given router group.
func (h *ParserHandler) RegisterRoutes(rg *gin.RouterGroup) {
	parser := rg.Group("/parser")
	{
		parser.POST("/parse", h.Parse)
		parser.POST("/parse-batch", h.ParseBatch)
	}
}

// Ensure ParserHandler properly uses the parser package
var _ = parser.ParseStatusSuccess
