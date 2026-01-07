package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexyu/vido/internal/config"
	"github.com/gin-gonic/gin"
)

func TestAppError(t *testing.T) {
	tests := []struct {
		name        string
		appErr      *AppError
		wantError   string
		wantUnwrap  bool
		wantMessage string
	}{
		{
			name: "error without wrapped error",
			appErr: &AppError{
				Code:       "TEST_ERROR",
				Message:    "test error message",
				StatusCode: http.StatusBadRequest,
			},
			wantError:   "test error message",
			wantUnwrap:  false,
			wantMessage: "test error message",
		},
		{
			name: "error with wrapped error",
			appErr: &AppError{
				Code:       "TEST_ERROR",
				Message:    "test error",
				StatusCode: http.StatusInternalServerError,
				Err:        errors.New("wrapped error"),
			},
			wantError:   "test error: wrapped error",
			wantUnwrap:  true,
			wantMessage: "test error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.appErr.Error(); got != tt.wantError {
				t.Errorf("AppError.Error() = %v, want %v", got, tt.wantError)
			}

			if tt.wantUnwrap {
				if tt.appErr.Unwrap() == nil {
					t.Error("AppError.Unwrap() = nil, want non-nil")
				}
			} else {
				if tt.appErr.Unwrap() != nil {
					t.Error("AppError.Unwrap() = non-nil, want nil")
				}
			}
		})
	}
}

func TestNewValidationError(t *testing.T) {
	err := NewValidationError("invalid input")

	if err.Code != "VALIDATION_ERROR" {
		t.Errorf("Code = %v, want VALIDATION_ERROR", err.Code)
	}
	if err.Message != "invalid input" {
		t.Errorf("Message = %v, want 'invalid input'", err.Message)
	}
	if err.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode = %v, want %v", err.StatusCode, http.StatusBadRequest)
	}
}

func TestNewNotFoundError(t *testing.T) {
	err := NewNotFoundError("resource not found")

	if err.Code != "NOT_FOUND" {
		t.Errorf("Code = %v, want NOT_FOUND", err.Code)
	}
	if err.Message != "resource not found" {
		t.Errorf("Message = %v, want 'resource not found'", err.Message)
	}
	if err.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode = %v, want %v", err.StatusCode, http.StatusNotFound)
	}
}

func TestNewInternalError(t *testing.T) {
	wrappedErr := errors.New("database error")
	err := NewInternalError("failed to process", wrappedErr)

	if err.Code != "INTERNAL_ERROR" {
		t.Errorf("Code = %v, want INTERNAL_ERROR", err.Code)
	}
	if err.Message != "failed to process" {
		t.Errorf("Message = %v, want 'failed to process'", err.Message)
	}
	if err.StatusCode != http.StatusInternalServerError {
		t.Errorf("StatusCode = %v, want %v", err.StatusCode, http.StatusInternalServerError)
	}
	if err.Err != wrappedErr {
		t.Errorf("Err = %v, want %v", err.Err, wrappedErr)
	}
}

func TestNewUnauthorizedError(t *testing.T) {
	err := NewUnauthorizedError("authentication required")

	if err.Code != "UNAUTHORIZED" {
		t.Errorf("Code = %v, want UNAUTHORIZED", err.Code)
	}
	if err.Message != "authentication required" {
		t.Errorf("Message = %v, want 'authentication required'", err.Message)
	}
	if err.StatusCode != http.StatusUnauthorized {
		t.Errorf("StatusCode = %v, want %v", err.StatusCode, http.StatusUnauthorized)
	}
}

func TestNewForbiddenError(t *testing.T) {
	err := NewForbiddenError("access denied")

	if err.Code != "FORBIDDEN" {
		t.Errorf("Code = %v, want FORBIDDEN", err.Code)
	}
	if err.Message != "access denied" {
		t.Errorf("Message = %v, want 'access denied'", err.Message)
	}
	if err.StatusCode != http.StatusForbidden {
		t.Errorf("StatusCode = %v, want %v", err.StatusCode, http.StatusForbidden)
	}
}

func TestErrorHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		Env: "development",
	}

	tests := []struct {
		name             string
		setupHandler     func(*gin.Context)
		wantStatus       int
		wantErrorCode    string
		wantErrorMessage string
	}{
		{
			name: "handles validation error",
			setupHandler: func(c *gin.Context) {
				c.Error(NewValidationError("invalid email format"))
			},
			wantStatus:       http.StatusBadRequest,
			wantErrorCode:    "VALIDATION_ERROR",
			wantErrorMessage: "invalid email format",
		},
		{
			name: "handles not found error",
			setupHandler: func(c *gin.Context) {
				c.Error(NewNotFoundError("user not found"))
			},
			wantStatus:       http.StatusNotFound,
			wantErrorCode:    "NOT_FOUND",
			wantErrorMessage: "user not found",
		},
		{
			name: "handles internal error",
			setupHandler: func(c *gin.Context) {
				c.Error(NewInternalError("database error", errors.New("connection failed")))
			},
			wantStatus:       http.StatusInternalServerError,
			wantErrorCode:    "INTERNAL_ERROR",
			wantErrorMessage: "database error",
		},
		{
			name: "handles unauthorized error",
			setupHandler: func(c *gin.Context) {
				c.Error(NewUnauthorizedError("authentication required"))
			},
			wantStatus:       http.StatusUnauthorized,
			wantErrorCode:    "UNAUTHORIZED",
			wantErrorMessage: "authentication required",
		},
		{
			name: "handles forbidden error",
			setupHandler: func(c *gin.Context) {
				c.Error(NewForbiddenError("access denied"))
			},
			wantStatus:       http.StatusForbidden,
			wantErrorCode:    "FORBIDDEN",
			wantErrorMessage: "access denied",
		},
		{
			name: "handles generic error",
			setupHandler: func(c *gin.Context) {
				c.Error(errors.New("generic error"))
			},
			wantStatus:       http.StatusInternalServerError,
			wantErrorCode:    "INTERNAL_ERROR",
			wantErrorMessage: "An internal server error occurred",
		},
		{
			name: "no error returns no response",
			setupHandler: func(c *gin.Context) {
				c.Status(http.StatusOK)
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/test", nil)
			c.Set("request_id", "test-request-id")

			// Setup handler
			tt.setupHandler(c)

			// Apply error handler middleware
			ErrorHandler(cfg)(c)

			// Check status code
			if w.Code != tt.wantStatus {
				t.Errorf("Status = %v, want %v", w.Code, tt.wantStatus)
			}

			// If we expect an error response, check it
			if tt.wantErrorCode != "" {
				var response ErrorResponse
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				if response.Error.Code != tt.wantErrorCode {
					t.Errorf("Error code = %v, want %v", response.Error.Code, tt.wantErrorCode)
				}

				if response.Error.Message != tt.wantErrorMessage {
					t.Errorf("Error message = %v, want %v", response.Error.Message, tt.wantErrorMessage)
				}
			}
		})
	}
}

func TestRecovery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		Env: "development",
	}

	tests := []struct {
		name             string
		panicValue       interface{}
		wantStatus       int
		wantErrorCode    string
		wantErrorMessage string
	}{
		{
			name:             "recovers from string panic",
			panicValue:       "something went wrong",
			wantStatus:       http.StatusInternalServerError,
			wantErrorCode:    "INTERNAL_ERROR",
			wantErrorMessage: "An internal server error occurred",
		},
		{
			name:             "recovers from error panic",
			panicValue:       errors.New("panic error"),
			wantStatus:       http.StatusInternalServerError,
			wantErrorCode:    "INTERNAL_ERROR",
			wantErrorMessage: "An internal server error occurred",
		},
		{
			name:             "recovers from nil panic",
			panicValue:       nil,
			wantStatus:       http.StatusInternalServerError,
			wantErrorCode:    "INTERNAL_ERROR",
			wantErrorMessage: "An internal server error occurred",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, router := gin.CreateTestContext(w)

			// Apply recovery middleware
			router.Use(Recovery(cfg))

			// Add a route that panics
			router.GET("/panic", func(c *gin.Context) {
				c.Set("request_id", "test-request-id")
				panic(tt.panicValue)
			})

			c.Request = httptest.NewRequest("GET", "/panic", nil)
			router.ServeHTTP(w, c.Request)

			// Check status code
			if w.Code != tt.wantStatus {
				t.Errorf("Status = %v, want %v", w.Code, tt.wantStatus)
			}

			// Check error response
			var response ErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			if response.Error.Code != tt.wantErrorCode {
				t.Errorf("Error code = %v, want %v", response.Error.Code, tt.wantErrorCode)
			}

			if response.Error.Message != tt.wantErrorMessage {
				t.Errorf("Error message = %v, want %v", response.Error.Message, tt.wantErrorMessage)
			}
		})
	}
}

func TestRecoveryNoError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		Env: "production",
	}

	w := httptest.NewRecorder()
	c, router := gin.CreateTestContext(w)

	// Apply recovery middleware
	router.Use(Recovery(cfg))

	// Add a route that doesn't panic
	router.GET("/ok", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	c.Request = httptest.NewRequest("GET", "/ok", nil)
	router.ServeHTTP(w, c.Request)

	// Check status code
	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestGetRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name      string
		setupCtx  func(*gin.Context)
		wantID    string
	}{
		{
			name: "returns request ID from context",
			setupCtx: func(c *gin.Context) {
				c.Set("request_id", "test-id-123")
			},
			wantID: "test-id-123",
		},
		{
			name: "returns empty string when request ID not set",
			setupCtx: func(c *gin.Context) {
				// Don't set request_id
			},
			wantID: "",
		},
		{
			name: "returns empty string when request ID is wrong type",
			setupCtx: func(c *gin.Context) {
				c.Set("request_id", 12345)
			},
			wantID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/test", nil)

			tt.setupCtx(c)

			got := getRequestID(c)
			if got != tt.wantID {
				t.Errorf("getRequestID() = %v, want %v", got, tt.wantID)
			}
		})
	}
}
