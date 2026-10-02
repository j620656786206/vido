package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/services"
)

// GlossaryHandler serves the per-show glossary REST surface (Story 9R-15) that
// the F6 review UI drives: list / add / edit / confirm / confirm-all / delete.
type GlossaryHandler struct {
	service  services.GlossaryServiceInterface
	exchange GlossaryExchanger
}

// GlossaryExchanger is the export/import surface (sub-8-1, Rule 11);
// *services.GlossaryExchangeService satisfies it. nil = routes answer 404.
type GlossaryExchanger interface {
	Export(ctx context.Context, mediaID string) (*services.GlossaryExport, error)
	Import(ctx context.Context, mediaID string, file *services.GlossaryExport) (*services.GlossaryImportResult, error)
}

// WithExchange enables GET …/export and POST …/import (sub-8-1).
func (h *GlossaryHandler) WithExchange(x GlossaryExchanger) *GlossaryHandler {
	h.exchange = x
	return h
}

// glossaryImportMaxBytes caps an import body: a few hundred terms is a few
// tens of KB; 2 MB is generous and still stops a pasted video file.
const glossaryImportMaxBytes = 2 << 20

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
		g.GET("/export", h.Export)  // sub-8-1
		g.POST("/import", h.Import) // sub-8-1
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
// Export handles GET /api/v1/media/:id/glossary/export (sub-8-1 AC #2).
//
// @Summary Download a title's glossary as a vido-glossary file
// @Description The file (`services.GlossaryExport` [@contract-v1]) is served as an attachment, NOT wrapped in the APIResponse envelope — it is the file itself. Only titles matched to TMDb can be shared: a local:* glossary answers 409 GLOSSARY_NOT_SHAREABLE.
// @Tags glossary
// @Produce json
// @Param id path string true "movie or series id"
// @Success 200 {object} services.GlossaryExport
// @Router /media/{id}/glossary/export [get]
func (h *GlossaryHandler) Export(c *gin.Context) {
	if h.exchange == nil {
		NotFoundError(c, "Glossary export")
		return
	}
	file, err := h.exchange.Export(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.writeErr(c, err, "export glossary")
		return
	}
	name := "vido-glossary-" + strings.NewReplacer(":", "-", "/", "-").Replace(file.Scope) + ".json"
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, name))
	c.JSON(http.StatusOK, file)
}

// Import handles POST /api/v1/media/:id/glossary/import (sub-8-1 AC #2).
//
// @Summary Merge a friend's vido-glossary file into this title's glossary
// @Description Accepts the file as the raw JSON body or as multipart field `file`. Insert-only: new terms land as source=community, unconfirmed; a term this title already has is never overwritten — the same rendering counts as skipped, a different one is returned in `conflicts` for the user to settle. 400 GLOSSARY_IMPORT_INVALID (not a vido-glossary v1 file / over limits), 400 GLOSSARY_SCOPE_MISMATCH (exported from another title), 409 GLOSSARY_NOT_SHAREABLE (this title has no TMDb match).
// @Tags glossary
// @Accept json,mpfd
// @Produce json
// @Param id path string true "movie or series id"
// @Success 200 {object} APIResponse "services.GlossaryImportResult [@contract-v1]"
// @Router /media/{id}/glossary/import [post]
func (h *GlossaryHandler) Import(c *gin.Context) {
	if h.exchange == nil {
		NotFoundError(c, "Glossary import")
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, glossaryImportMaxBytes)
	var raw io.Reader = c.Request.Body
	if strings.HasPrefix(c.ContentType(), "multipart/") {
		fh, err := c.FormFile("file")
		if err != nil {
			ErrorResponse(c, http.StatusBadRequest, "GLOSSARY_IMPORT_INVALID",
				"沒有收到檔案。", "請選一個 vido-glossary 的 .json 檔。")
			return
		}
		f, err := fh.Open()
		if err != nil {
			ErrorResponse(c, http.StatusBadRequest, "GLOSSARY_IMPORT_INVALID", "檔案打不開。", "")
			return
		}
		defer f.Close()
		raw = f
	}
	var file services.GlossaryExport
	if err := json.NewDecoder(raw).Decode(&file); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "GLOSSARY_IMPORT_INVALID",
			"這不是 Vido 的詞彙表檔。", "請用 Vido「匯出」產生的 .json 檔。")
		return
	}
	res, err := h.exchange.Import(c.Request.Context(), c.Param("id"), &file)
	if err != nil {
		h.writeErr(c, err, "import glossary")
		return
	}
	SuccessResponse(c, res)
}

func (h *GlossaryHandler) writeErr(c *gin.Context, err error, op string) {
	var ve *models.ValidationError
	switch {
	case errors.Is(err, services.ErrGlossaryNotShareable):
		ErrorResponse(c, http.StatusConflict, "GLOSSARY_NOT_SHAREABLE",
			"這部片還沒對到 TMDb，詞彙表沒辦法分享。", "先在詳情頁把它對到正確的 TMDb 條目。")
	case errors.Is(err, services.ErrGlossaryScopeMismatch):
		ErrorResponse(c, http.StatusBadRequest, "GLOSSARY_SCOPE_MISMATCH",
			"這個檔案是別部片的詞彙表。", "請到那部片的詞彙表匯入，或請對方匯出這一部。")
	case errors.Is(err, services.ErrGlossaryImportInvalid):
		ErrorResponse(c, http.StatusBadRequest, "GLOSSARY_IMPORT_INVALID", err.Error(), "請用 Vido「匯出」產生的 .json 檔。")
	case errors.As(err, &ve):
		ValidationError(c, ve.Error())
	case errors.Is(err, repository.ErrGlossaryTermNotFound):
		NotFoundError(c, "Glossary term")
	default:
		slog.Error("glossary handler error", "op", op, "error", err)
		InternalServerError(c, "詞彙操作失敗")
	}
}
