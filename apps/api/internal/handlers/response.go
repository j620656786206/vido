package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// APIResponse represents the standard API response format
// Success: {"success": true, "data": {...}}
// Error: {"success": false, "error": {...}}
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
}

// APIError represents the error response format
type APIError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
}

// PaginatedResponse wraps data with pagination info
type PaginatedResponse struct {
	Items      interface{} `json:"items"`
	Page       int         `json:"page"`
	PageSize   int         `json:"pageSize"`
	TotalItems int         `json:"totalItems"`
	TotalPages int         `json:"totalPages"`
}

// SuccessResponse sends a success response with data
func SuccessResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    data,
	})
}

// CreatedResponse sends a 201 Created response with data
func CreatedResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, APIResponse{
		Success: true,
		Data:    data,
	})
}

// NoContentResponse sends a 204 No Content response
func NoContentResponse(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// ErrorResponse sends an error response with the given status code
func ErrorResponse(c *gin.Context, statusCode int, code, message, suggestion string) {
	c.JSON(statusCode, APIResponse{
		Success: false,
		Error: &APIError{
			Code:       code,
			Message:    message,
			Suggestion: suggestion,
		},
	})
}

// BadRequestError sends a 400 Bad Request error
func BadRequestError(c *gin.Context, code, message string) {
	ErrorResponse(c, http.StatusBadRequest, code, message, "Please check your request parameters.")
}

// NotFoundError sends a 404 Not Found error
func NotFoundError(c *gin.Context, resource string) {
	ErrorResponse(c, http.StatusNotFound, "DB_NOT_FOUND",
		resource+" not found",
		"Verify the ID is correct and the resource exists.")
}

// InternalServerError sends a 500 Internal Server Error
func InternalServerError(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR",
		message,
		"Please try again later or contact support.")
}

// ValidationError sends a 400 Bad Request for validation failures
func ValidationError(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusBadRequest, "VALIDATION_ERROR",
		message,
		"Please check the required fields and their formats.")
}
