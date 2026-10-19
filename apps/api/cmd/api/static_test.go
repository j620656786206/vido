package main

import (
	"bytes"
	"mime"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestPublicDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Create index.html
	require.NoError(t, os.WriteFile(filepath.Join(dir, "index.html"), []byte("<!DOCTYPE html><html><body>SPA</body></html>"), 0644))

	// Create assets directory with a JS file
	assetsDir := filepath.Join(dir, "assets")
	require.NoError(t, os.MkdirAll(assetsDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(assetsDir, "main.abc123.js"), []byte("console.log('app')"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(assetsDir, "style.def456.css"), []byte("body{}"), 0644))

	// Create favicon
	require.NoError(t, os.WriteFile(filepath.Join(dir, "favicon.ico"), []byte("icon"), 0644))

	return dir
}

func setupRouter(publicDir string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add security headers middleware
	router.Use(securityHeadersMiddleware())

	// Simulate API routes (must be registered BEFORE static routes)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	router.GET("/api/v1/movies", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": []string{}})
	})

	// Register static routes AFTER API routes
	registerStaticRoutes(router, publicDir)

	return router
}

func TestSPAFallback_ServesIndexHTML(t *testing.T) {
	publicDir := setupTestPublicDir(t)
	router := setupRouter(publicDir)

	// Request a client-side route (e.g., /library)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/library", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "SPA")
	assert.Equal(t, cacheNoStore, w.Header().Get("Cache-Control"))
}

func TestSPAFallback_DeepRoute(t *testing.T) {
	publicDir := setupTestPublicDir(t)
	router := setupRouter(publicDir)

	// Deep nested client-side route
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/library/movies/123", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "SPA")
	assert.Equal(t, cacheNoStore, w.Header().Get("Cache-Control"))
}

func TestAPIRoute_Returns404JSON(t *testing.T) {
	publicDir := setupTestPublicDir(t)
	router := setupRouter(publicDir)

	// Non-existent API route should return JSON 404, not index.html
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/nonexistent", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "NOT_FOUND")
	assert.NotContains(t, w.Body.String(), "SPA")
}

func TestAPIRoute_ExistingEndpoint(t *testing.T) {
	publicDir := setupTestPublicDir(t)
	router := setupRouter(publicDir)

	// Existing API route should work normally
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/movies", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "data")
}

func TestHealthEndpoint_WorksNormally(t *testing.T) {
	publicDir := setupTestPublicDir(t)
	router := setupRouter(publicDir)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")
}

func TestStaticAssets_ImmutableCacheControl(t *testing.T) {
	publicDir := setupTestPublicDir(t)
	router := setupRouter(publicDir)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/assets/main.abc123.js", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, cacheImmutable, w.Header().Get("Cache-Control"))
	assert.Contains(t, w.Body.String(), "console.log")
}

func TestStaticAssets_CSSFile(t *testing.T) {
	publicDir := setupTestPublicDir(t)
	router := setupRouter(publicDir)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/assets/style.def456.css", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, cacheImmutable, w.Header().Get("Cache-Control"))
}

func TestFavicon_Served(t *testing.T) {
	publicDir := setupTestPublicDir(t)
	router := setupRouter(publicDir)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/favicon.ico", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, cacheImmutable, w.Header().Get("Cache-Control"))
}

func TestSecurityHeaders_Present(t *testing.T) {
	publicDir := setupTestPublicDir(t)
	router := setupRouter(publicDir)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, "SAMEORIGIN", w.Header().Get("X-Frame-Options"))
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"))
	assert.Equal(t, "strict-origin-when-cross-origin", w.Header().Get("Referrer-Policy"))
}

func TestRegisterStaticRoutes_NoPublicDir(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Register with non-existent dir — should not panic
	registerStaticRoutes(router, "/nonexistent/path")

	// Router should still work, just no static routes
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetPublicDir_Default(t *testing.T) {
	// Unset env var to test default
	os.Unsetenv("VIDO_PUBLIC_DIR")
	assert.Equal(t, defaultPublicDir, getPublicDir())
}

func TestGetPublicDir_CustomEnv(t *testing.T) {
	t.Setenv("VIDO_PUBLIC_DIR", "/custom/path")
	assert.Equal(t, "/custom/path", getPublicDir())
}

func TestRootPath_ServesIndexHTML(t *testing.T) {
	publicDir := setupTestPublicDir(t)
	router := setupRouter(publicDir)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "SPA")
	assert.Equal(t, cacheNoStore, w.Header().Get("Cache-Control"))
}

// setupCompressedRouter mirrors main.go's order: security headers, gzip, then
// static routes — so the assertions below see what a browser sees.
func setupCompressedRouter(t *testing.T) *gin.Engine {
	t.Helper()
	publicDir := setupTestPublicDir(t)
	assetsDir := filepath.Join(publicDir, "assets")
	// A woff2 starts with the "wOF2" signature; the body after it is opaque.
	woff2 := append([]byte("wOF2"), bytes.Repeat([]byte{0x00, 0x01, 0x02, 0x03}, 512)...)
	require.NoError(t, os.WriteFile(filepath.Join(assetsDir, "noto-sans-tc-115-wght-normal-AbC123.woff2"), woff2, 0644))
	require.NoError(t, os.WriteFile(filepath.Join(assetsDir, "index-Xyz789.css"), []byte(strings.Repeat("body{color:red}", 200)), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(assetsDir, "poster-Q1w2e3.png"), []byte("\x89PNG\r\n\x1a\n"+strings.Repeat("x", 512)), 0644))

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(securityHeadersMiddleware())
	router.Use(compressionMiddleware())
	router.GET("/api/v1/events", func(c *gin.Context) {
		c.Header("Content-Type", "text/event-stream")
		c.String(http.StatusOK, strings.Repeat("data: ping\n\n", 200))
	})
	router.GET("/api/v1/movies", func(c *gin.Context) {
		c.String(http.StatusOK, strings.Repeat(`{"id":"m"}`, 200))
	})
	registerStaticRoutes(router, publicDir)
	return router
}

func getWithGzip(router *gin.Engine, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", path, nil)
	req.Header.Set("Accept-Encoding", "gzip")
	router.ServeHTTP(w, req)
	return w
}

// woff2 is already Brotli-compressed: gzipping it again costs NAS CPU and makes
// it bigger. It must still go out as font/woff2 with the 1-year cache.
func TestCompression_Woff2NotGzipped(t *testing.T) {
	router := setupCompressedRouter(t)

	w := getWithGzip(router, "/assets/noto-sans-tc-115-wght-normal-AbC123.woff2")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Header().Get("Content-Encoding"))
	assert.Equal(t, "font/woff2", w.Header().Get("Content-Type"))
	assert.Equal(t, cacheImmutable, w.Header().Get("Cache-Control"))
	assert.True(t, bytes.HasPrefix(w.Body.Bytes(), []byte("wOF2")))
}

// WithExcludedExtensions replaces the library defaults, so the image defaults
// have to survive the woff2 addition.
func TestCompression_ImagesStillNotGzipped(t *testing.T) {
	router := setupCompressedRouter(t)

	w := getWithGzip(router, "/assets/poster-Q1w2e3.png")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Header().Get("Content-Encoding"))
}

// Text assets keep their gzip — excluding woff2 must not switch it off globally.
func TestCompression_CSSStillGzipped(t *testing.T) {
	router := setupCompressedRouter(t)

	w := getWithGzip(router, "/assets/index-Xyz789.css")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "gzip", w.Header().Get("Content-Encoding"))
}

func TestGzipExcludedExtensions_ExactList(t *testing.T) {
	assert.Equal(t, []string{".png", ".gif", ".jpeg", ".jpg", ".woff2"}, gzipExcludedExtensions)
}

// The SSE stream must stay uncompressed (http.Flusher) — the exclusion moved
// into compressionMiddleware with this change, so pin it here.
func TestCompression_SSEStreamNotGzipped(t *testing.T) {
	router := setupCompressedRouter(t)

	w := getWithGzip(router, "/api/v1/events")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Header().Get("Content-Encoding"))
}

// Ordinary API JSON keeps its gzip.
func TestCompression_APIStillGzipped(t *testing.T) {
	router := setupCompressedRouter(t)

	w := getWithGzip(router, "/api/v1/movies")

	assert.Equal(t, "gzip", w.Header().Get("Content-Encoding"))
}

// Registered in init(), not left to the sniffer: alpine has no mime.types.
func TestWoff2MimeTypeRegistered(t *testing.T) {
	assert.Equal(t, "font/woff2", mime.TypeByExtension(".woff2"))
}
