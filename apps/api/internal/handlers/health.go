package handlers

import (
	"context"
	"net/http"
	"strconv"
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
	WALEnabled      bool   `json:"wal_enabled"`       // Whether WAL mode is active
	WALMode         string `json:"wal_mode"`          // Current journal mode
	SyncMode        string `json:"sync_mode"`         // Current synchronous mode
	OpenConnections int    `json:"open_connections"`  // Current open connections
	IdleConnections int    `json:"idle_connections"`  // Current idle connections
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
	historyService     services.ConnectionHistoryServiceInterface
}

// NewServiceHealthHandler creates a new ServiceHealthHandler.
func NewServiceHealthHandler(degradationService services.DegradationServiceInterface) *ServiceHealthHandler {
	return &ServiceHealthHandler{
		degradationService: degradationService,
	}
}

// SetHistoryService sets the connection history service for history endpoints
func (h *ServiceHealthHandler) SetHistoryService(svc services.ConnectionHistoryServiceInterface) {
	h.historyService = svc
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

// GetConnectionHistory returns connection history for a specific service.
// @Summary Get connection history for a service
// @Description Returns recent connection status change events for the specified service
// @Tags Health
// @Produce json
// @Param service path string true "Service name (e.g., qbittorrent)"
// @Param limit query int false "Number of events to return (default 20)"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/health/services/{service}/history [get]
func (h *ServiceHealthHandler) GetConnectionHistory(c *gin.Context) {
	if h.historyService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "CONNECTION_HISTORY_ERROR",
				"message": "Connection history is not configured",
			},
		})
		return
	}

	service := c.Param("service")
	if service == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_REQUIRED_FIELD",
				"message": "Service name is required",
			},
		})
		return
	}

	if !services.IsValidServiceName(service) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_INVALID_FORMAT",
				"message": "Invalid service name",
			},
		})
		return
	}

	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	events, err := h.historyService.GetHistory(c.Request.Context(), service, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "CONNECTION_HISTORY_ERROR",
				"message": "Failed to retrieve connection history",
			},
		})
		return
	}

	if events == nil {
		events = []models.ConnectionEvent{}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    events,
	})
}
