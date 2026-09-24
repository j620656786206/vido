package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	"github.com/gin-gonic/gin"
)

// posterFileName is the ONLY shape a served file may have: what
// images.ImageProcessor writes — <mediaID>.jpg and <mediaID>-thumb.jpg, where
// the media ID is a UUID. Anything else (a "..", a sub-directory, another
// extension, an encoded slash) is simply not a poster and gets a 404.
var posterFileName = regexp.MustCompile(`^[A-Za-z0-9_-]+\.jpg$`) // "-thumb" is covered by the class

// PosterFileHandler serves posters the user uploaded through the metadata
// editor (Story 3.8 AC3). The upload stored `poster_path = "/posters/<id>.jpg"`
// but nothing ever served that path, so a custom poster never showed up
// (bugfix-custom-posters-served-and-not-cache). The frontend maps
// "/posters/<file>" to "{API}/posters/<file>".
type PosterFileHandler struct {
	dir string
}

// NewPosterFileHandler serves files from dir (data/posters).
func NewPosterFileHandler(dir string) *PosterFileHandler {
	return &PosterFileHandler{dir: dir}
}

// RegisterRoutes registers GET /posters/:file under the given group.
func (h *PosterFileHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/posters/:file", h.Serve)
	rg.HEAD("/posters/:file", h.Serve)
}

// Serve handles GET /api/v1/posters/:file.
func (h *PosterFileHandler) Serve(c *gin.Context) {
	name := c.Param("file")
	if !posterFileName.MatchString(name) {
		c.Status(http.StatusNotFound)
		return
	}
	// The name matched a pattern with no separators, so Join cannot leave dir.
	path := filepath.Join(h.dir, name)
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		c.Status(http.StatusNotFound)
		return
	}
	// Re-uploading a poster overwrites the same file name, so never
	// "immutable": the browser revalidates with Last-Modified instead.
	c.Header("Cache-Control", "no-cache")
	c.Header("Content-Type", "image/jpeg")
	c.File(path)
}
