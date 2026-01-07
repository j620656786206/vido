package server

import (
	"net/http"
	"time"

	"github.com/alexyu/vido/internal/middleware"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// setupRoutes configures all routes for the server
func (s *Server) setupRoutes() {
	// Swagger documentation - available in all environments
	s.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Health check endpoint - no prefix
	s.router.GET("/health", s.healthCheck)

	// API v1 route group
	v1 := s.router.Group("/api/v1")
	{
		// Example endpoints demonstrating error handling
		v1.GET("/error-example", s.errorExample)
		// Future API endpoints will be added here
	}
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string `json:"status" example:"ok"`
	Timestamp string `json:"timestamp" example:"2024-01-07T10:30:00Z"`
}

// healthCheck handles the health check endpoint
// @Summary      Check API health status
// @Description  Returns the health status of the API server
// @Tags         health
// @Produce      json
// @Success      200  {object}  HealthResponse
// @Router       /health [get]
func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// ErrorExampleResponse represents the error example usage message
type ErrorExampleResponse struct {
	Message string `json:"message" example:"Use ?type=validation|notfound|internal|unauthorized|forbidden|panic to test error handling"`
}

// errorExample demonstrates error handling middleware
// @Summary      Demonstrate error handling
// @Description  Test endpoint to demonstrate different error types and the error handling middleware
// @Tags         examples
// @Produce      json
// @Param        type  query  string  false  "Error type to trigger" Enums(validation, notfound, internal, unauthorized, forbidden, panic)
// @Success      200  {object}  ErrorExampleResponse
// @Failure      400  {object}  middleware.ErrorResponse
// @Failure      401  {object}  middleware.ErrorResponse
// @Failure      403  {object}  middleware.ErrorResponse
// @Failure      404  {object}  middleware.ErrorResponse
// @Failure      500  {object}  middleware.ErrorResponse
// @Router       /api/v1/error-example [get]
func (s *Server) errorExample(c *gin.Context) {
	errorType := c.Query("type")

	switch errorType {
	case "validation":
		c.Error(middleware.NewValidationError("Invalid email format"))
	case "notfound":
		c.Error(middleware.NewNotFoundError("User not found"))
	case "internal":
		c.Error(middleware.NewInternalError("Database connection failed", nil))
	case "unauthorized":
		c.Error(middleware.NewUnauthorizedError("Authentication required"))
	case "forbidden":
		c.Error(middleware.NewForbiddenError("Access denied"))
	case "panic":
		panic("This is a test panic")
	default:
		c.JSON(http.StatusOK, gin.H{
			"message": "Use ?type=validation|notfound|internal|unauthorized|forbidden|panic to test error handling",
		})
	}
}
