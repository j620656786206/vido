package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func gateRouter(healthy func() bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := r.Group("/api/v1", DatabaseGate(healthy))
	api.GET("/movies", func(c *gin.Context) {
		SuccessResponse(c, gin.H{"ok": true})
	})
	return r
}

func TestDatabaseGate_HealthyPassesThrough(t *testing.T) {
	r := gateRouter(func() bool { return true })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/movies", nil))

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDatabaseGate_UnhealthyFailsFastWithUniformCode(t *testing.T) {
	r := gateRouter(func() bool { return false })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/movies", nil))

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), ErrCodeDatabaseUnavailable)
	assert.Contains(t, w.Body.String(), "資料庫目前無法使用")
}
