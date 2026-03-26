package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	// defaultPublicDir is the default directory for static web assets in Docker.
	defaultPublicDir = "/app/public"

	// cacheImmutable sets 1-year cache for hashed assets (JS, CSS, images, fonts).
	cacheImmutable = "public, max-age=31536000, immutable"

	// cacheNoStore ensures index.html is never cached so users always get the latest SPA.
	cacheNoStore = "no-store, no-cache, must-revalidate"
)

// getPublicDir returns the static assets directory path.
// Uses VIDO_PUBLIC_DIR env var if set, otherwise defaults to /app/public.
func getPublicDir() string {
	if dir := os.Getenv("VIDO_PUBLIC_DIR"); dir != "" {
		return dir
	}
	return defaultPublicDir
}

// registerStaticRoutes sets up static file serving and SPA fallback on the router.
// Must be called AFTER all API routes are registered.
func registerStaticRoutes(router *gin.Engine, publicDir string) {
	// Only register if the public directory exists (skip in dev mode without built frontend)
	if _, err := os.Stat(publicDir); os.IsNotExist(err) {
		return
	}

	// Serve static assets directory with immutable cache headers.
	// Vite outputs hashed filenames (e.g., index-DpxxWXUv.js) so immutable caching is safe.
	assetsDir := filepath.Join(publicDir, "assets")
	if _, err := os.Stat(assetsDir); err == nil {
		router.Use(assetCacheMiddleware(cacheImmutable))
		router.Static("/assets", assetsDir)
	}

	// Serve favicon and other root-level static files
	for _, name := range []string{"favicon.ico", "favicon.svg", "robots.txt"} {
		fullPath := filepath.Join(publicDir, name)
		if _, err := os.Stat(fullPath); err == nil {
			localName := name // capture for closure
			router.GET("/"+localName, cacheControlMiddleware(cacheImmutable), func(c *gin.Context) {
				c.File(filepath.Join(publicDir, localName))
			})
		}
	}

	// SPA fallback — serves index.html for all non-API, non-asset routes.
	// This enables client-side routing (TanStack Router).
	router.NoRoute(spaFallbackHandler(publicDir))
}

// assetCacheMiddleware sets Cache-Control headers only for requests under /assets/.
func assetCacheMiddleware(value string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/assets/") {
			c.Header("Cache-Control", value)
		}
		c.Next()
	}
}

// spaFallbackHandler returns a Gin handler that serves index.html for non-API routes
// and returns a JSON 404 for unmatched API routes.
func spaFallbackHandler(publicDir string) gin.HandlerFunc {
	indexPath := filepath.Join(publicDir, "index.html")
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// API routes that don't match should return JSON 404
		if strings.HasPrefix(path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "NOT_FOUND",
					"message": "API endpoint not found",
				},
			})
			return
		}

		// Health endpoint is handled by explicit route, but just in case
		if path == "/health" {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		// Serve index.html with no-cache headers for SPA fallback
		c.Header("Cache-Control", cacheNoStore)
		c.File(indexPath)
	}
}

// cacheControlMiddleware sets the Cache-Control header on responses.
func cacheControlMiddleware(value string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", value)
		c.Next()
	}
}

// securityHeadersMiddleware adds standard security headers to all responses,
// replicating the headers previously set by Nginx.
func securityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Frame-Options", "SAMEORIGIN")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Next()
	}
}
