package handlers

import (
	"errors"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/services"
)

// GlossaryHandler serves the per-show glossary REST surface (Story 9R-15) that
// the F6 review UI drives: list / add / edit / confirm / confirm-all / delete.
type GlossaryHandler struct {
	service services.GlossaryServiceInterface
}

// NewGlossaryHandler builds a GlossaryHandler.
func NewGlossaryHandler(service services.GlossaryServiceInterface) *GlossaryHandler {
	return &GlossaryHandler{service: service}
}

// RegisterRoutes mounts the glossary routes. :id is the local movie/series
// id — the same key the generation pipeline (9R-10) and .nfo localizer (9R-13)
// use, so the UI, the pipeline, and the REST surface share one glossary. The
// wildcard must be named :id to match the existing /media/:id/* routes
// (metadata_handler) — gin panics on differing wildcard names per segment.
func (h *GlossaryHandler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/media/:id/glossary")
	{
		g.GET("", h.List)
		g.POST("", h.Add)
		g.POST("/confirm-all", h.ConfirmAll)
		g.PUT("/:termId", h.Edit)
		g.POST("/:termId/confirm", h.Confirm)
		g.DELETE("/:termId", h.Delete)
	}
}

// List handles GET /api/v1/media/:id/glossary
func (h *GlossaryHandler) List(c *gin.Context) {
	terms, err := h.service.List(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.writeErr(c, err, "list glossary")
		return
	}
	if terms == nil {
		terms = []models.GlossaryTerm{}
	}
	SuccessResponse(c, gin.H{"terms": terms})
}

type glossaryAddRequest struct {
	TermSrc   string `json:"term_src"`
	TermZh    string `json:"term_zh"`
	Language  string `json:"language"`
	Source    string `json:"source"`
	Confirmed bool   `json:"confirmed"`
}

// Add handles POST /api/v1/media/:id/glossary
func (h *GlossaryHandler) Add(c *gin.Context) {
	var req glossaryAddRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequestError(c, "VALIDATION_INVALID_FORMAT", "Invalid request body")
		return
	}
	term := &models.GlossaryTerm{
		MediaID:   c.Param("id"), // route wins — never trust a body media_id
		TermSrc:   req.TermSrc,
		TermZh:    req.TermZh,
		Language:  req.Language,
		Source:    req.Source,
		Confirmed: req.Confirmed,
	}
	if err := h.service.Add(c.Request.Context(), term); err != nil {
		h.writeErr(c, err, "add glossary term")
		return
	}
	CreatedResponse(c, term)
}

type glossaryEditRequest struct {
	TermZh    string `json:"term_zh"`
	Confirmed bool   `json:"confirmed"`
}

// Edit handles PUT /api/v1/media/:id/glossary/:termId
func (h *GlossaryHandler) Edit(c *gin.Context) {
	var req glossaryEditRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequestError(c, "VALIDATION_INVALID_FORMAT", "Invalid request body")
		return
	}
	if err := h.service.Edit(c.Request.Context(), c.Param("id"), c.Param("termId"), req.TermZh, req.Confirmed); err != nil {
		h.writeErr(c, err, "edit glossary term")
		return
	}
	NoContentResponse(c)
}

// Confirm handles POST /api/v1/media/:id/glossary/:termId/confirm
func (h *GlossaryHandler) Confirm(c *gin.Context) {
	if err := h.service.Confirm(c.Request.Context(), c.Param("id"), c.Param("termId")); err != nil {
		h.writeErr(c, err, "confirm glossary term")
		return
	}
	NoContentResponse(c)
}

// ConfirmAll handles POST /api/v1/media/:id/glossary/confirm-all
func (h *GlossaryHandler) ConfirmAll(c *gin.Context) {
	n, err := h.service.ConfirmAll(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.writeErr(c, err, "confirm all glossary terms")
		return
	}
	SuccessResponse(c, gin.H{"confirmed": n})
}

// Delete handles DELETE /api/v1/media/:id/glossary/:termId
func (h *GlossaryHandler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), c.Param("id"), c.Param("termId")); err != nil {
		h.writeErr(c, err, "delete glossary term")
		return
	}
	NoContentResponse(c)
}

// writeErr maps service errors to the standard wire codes (no new Rule 7 prefix
// — reuses VALIDATION_*/DB_NOT_FOUND/INTERNAL_ERROR).
func (h *GlossaryHandler) writeErr(c *gin.Context, err error, op string) {
	var ve *models.ValidationError
	switch {
	case errors.As(err, &ve):
		ValidationError(c, ve.Error())
	case errors.Is(err, repository.ErrGlossaryTermNotFound):
		NotFoundError(c, "Glossary term")
	default:
		slog.Error("glossary handler error", "op", op, "error", err)
		InternalServerError(c, "詞彙操作失敗")
	}
}
