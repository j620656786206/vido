package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrCodeDatabaseUnavailable is the single error code every /api/v1 route
// returns while the database supervisor reports the database as down
// (bugfix-i-3-db-dead-returns-200). One condition, one code, one message —
// instead of ten handlers each failing in their own way, which is what made
// one data-layer incident look like ten distinct bugs on the first NAS deploy.
const ErrCodeDatabaseUnavailable = "DATABASE_UNAVAILABLE"

// DatabaseGate returns a middleware that fails every request fast with a
// uniform 503 DATABASE_UNAVAILABLE while healthy() reports false. healthy is a
// cached atomic read (database.Supervisor.Healthy) — no per-request ping, so a
// dead database costs microseconds per request instead of a 3s ping timeout.
// The root /health endpoint lives OUTSIDE the gated group and keeps answering
// with the honest 503 + database detail for probes and the frontend banner.
func DatabaseGate(healthy func() bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if healthy() {
			c.Next()
			return
		}
		ErrorResponse(c, http.StatusServiceUnavailable, ErrCodeDatabaseUnavailable,
			"資料庫目前無法使用",
			"所有需要資料庫的功能暫時失效。請檢查 NAS 的儲存掛載與磁碟空間;系統每 30 秒會自動嘗試復原,毋須重裝。")
		c.Abort()
	}
}
