package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/database"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/services"
)

// DatabaseHealth represents the database health status
type DatabaseHealth struct {
	Status          string `json:"status"`           // "healthy", "degraded", "unhealthy"
	Latency         int64  `json:"latency"`          // Latency in milliseconds
	WALEnabled      bool   `json:"walEnabled"`       // Whether WAL mode is active
	WALMode         string `json:"walMode"`          // Current journal mode
	SyncMode        string `json:"syncMode"`         // Current synchronous mode
	OpenConnections int    `json:"openConnections"`  // Current open connections
	IdleConnections int    `json:"idleConnections"`  // Current idle connections
	Error           string `json:"error,omitempty"`  // Error message if unhealthy
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status   string          `json:"status"`
	Service  string          `json:"service"`
	Database *DatabaseHealth `json:"database,omitempty"`
}

// HealthCheckHandler creates a health check handler with database dependency
func HealthCheckHandler(db *database.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		response := HealthResponse{
			Status:  "healthy",
			Service: "vido-api",
		}

		// Check database health
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		dbHealth := db.Health(ctx)

		// Convert database health to response format
		response.Database = &DatabaseHealth{
			Status:          dbHealth.Status,
			Latency:         dbHealth.Latency.Milliseconds(),
			WALEnabled:      dbHealth.WALEnabled,
			WALMode:         dbHealth.WALMode,
			SyncMode:        dbHealth.SyncMode,
			OpenConnections: dbHealth.OpenConnections,
			IdleConnections: dbHealth.IdleConnections,
			Error:           dbHealth.Error,
		}

		// If database is unhealthy or degraded, reflect in overall status
		if dbHealth.Status == "unhealthy" {
			response.Status = "unhealthy"
			c.JSON(http.StatusServiceUnavailable, response)
			return
		} else if dbHealth.Status == "degraded" {
			response.Status = "degraded"
			c.JSON(http.StatusOK, response)
			return
		}

		c.JSON(http.StatusOK, response)
	}
}

// HealthCheck returns the health status of the API (legacy, for backwards compatibility)
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{
		Status:  "healthy",
		Service: "vido-api",
	})
}

// ServiceHealthHandler handles GET /api/v1/health/services
// Returns health status of all external services and degradation level.
type ServiceHealthHandler struct {
	degradationService services.DegradationServiceInterface
}

// NewServiceHealthHandler creates a new ServiceHealthHandler.
func NewServiceHealthHandler(degradationService services.DegradationServiceInterface) *ServiceHealthHandler {
	return &ServiceHealthHandler{
		degradationService: degradationService,
	}
}

// GetServicesHealth returns the health status of all external services.
// @Summary Get external services health status
// @Description Returns the health status of TMDb, Douban, Wikipedia, and AI services
// @Tags Health
// @Produce json
// @Success 200 {object} models.HealthStatusResponse
// @Router /api/v1/health/services [get]
func (h *ServiceHealthHandler) GetServicesHealth(c *gin.Context) {
	if h.degradationService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "HEALTH_NOT_CONFIGURED",
				"message": "Health monitoring is not configured",
			},
		})
		return
	}

	status := h.degradationService.GetHealthStatus()

	// Determine HTTP status based on degradation level
	httpStatus := http.StatusOK
	if status.DegradationLevel == models.DegradationOffline {
		httpStatus = http.StatusServiceUnavailable
	}

	c.JSON(httpStatus, gin.H{
		"success": true,
		"data":    status,
	})
}
