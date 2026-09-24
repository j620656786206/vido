package handlers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupPosterRouter(t *testing.T) (*gin.Engine, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	root := t.TempDir()
	dir := filepath.Join(root, "posters")
	require.NoError(t, os.Mkdir(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "0fe13b88-a374-4039-9e8a-f80556eb7c78.jpg"), []byte("JPEGDATA"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "0fe13b88-a374-4039-9e8a-f80556eb7c78-thumb.jpg"), []byte("THUMB"), 0o644))
	// A file INSIDE the dir that is not a poster name: only the whitelist keeps it out.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("SECRET"), 0o644))
	// Something that must never be reachable: a sibling of the posters dir.
	require.NoError(t, os.WriteFile(filepath.Join(root, "vido.db"), []byte("SECRET"), 0o644))

	r := gin.New()
	NewPosterFileHandler(dir).RegisterRoutes(r.Group("/api/v1"))
	return r, dir
}

func getPoster(r *gin.Engine, path string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestPosterFileHandler_ServesUploadedPosterAndThumb(t *testing.T) {
	r, _ := setupPosterRouter(t)

	w := getPoster(r, "/api/v1/posters/0fe13b88-a374-4039-9e8a-f80556eb7c78.jpg")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "JPEGDATA", w.Body.String())
	assert.Equal(t, "image/jpeg", w.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
	assert.NotEmpty(t, w.Header().Get("Last-Modified"))

	w = getPoster(r, "/api/v1/posters/0fe13b88-a374-4039-9e8a-f80556eb7c78-thumb.jpg")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "THUMB", w.Body.String())
}

func TestPosterFileHandler_HeadAnswersWithoutBody(t *testing.T) {
	r, _ := setupPosterRouter(t)
	req, _ := http.NewRequest(http.MethodHead, "/api/v1/posters/0fe13b88-a374-4039-9e8a-f80556eb7c78.jpg", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "image/jpeg", w.Header().Get("Content-Type"))
}

func TestPosterFileHandler_RejectsAnythingThatIsNotAPosterName(t *testing.T) {
	r, _ := setupPosterRouter(t)
	for _, p := range []string{
		"/api/v1/posters/..%2Fvido.db",
		"/api/v1/posters/%2e%2e%2fvido.db",
		"/api/v1/posters/../vido.db",
		"/api/v1/posters/vido.db",
		"/api/v1/posters/x.png",
		"/api/v1/posters/.jpg",
		"/api/v1/posters/a%2Fb.jpg",
		"/api/v1/posters/a.b.jpg",
		"/api/v1/posters/notes.txt",
	} {
		w := getPoster(r, p)
		assert.NotEqual(t, http.StatusOK, w.Code, p)
		assert.NotContains(t, w.Body.String(), "SECRET", p)
	}
}

func TestPosterFileHandler_MissingFileIs404(t *testing.T) {
	r, _ := setupPosterRouter(t)
	w := getPoster(r, "/api/v1/posters/does-not-exist.jpg")
	assert.Equal(t, http.StatusNotFound, w.Code)
}
