package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/alexyu/vido/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func TestLogger(t *testing.T) {
	// Set up gin in test mode
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		method         string
		path           string
		requestID      string
		expectedStatus int
		wantRequestID  bool
	}{
		{
			name:           "GET request logs successfully",
			method:         "GET",
			path:           "/test",
			requestID:      "",
			expectedStatus: http.StatusOK,
			wantRequestID:  true,
		},
		{
			name:           "POST request with existing request ID",
			method:         "POST",
			path:           "/api/test",
			requestID:      "test-request-id-123",
			expectedStatus: http.StatusCreated,
			wantRequestID:  true,
		},
		{
			name:           "Request with query parameters",
			method:         "GET",
			path:           "/test?foo=bar&baz=qux",
			requestID:      "",
			expectedStatus: http.StatusOK,
			wantRequestID:  true,
		},
		{
			name:           "404 Not Found",
			method:         "GET",
			path:           "/notfound",
			requestID:      "",
			expectedStatus: http.StatusNotFound,
			wantRequestID:  true,
		},
		{
			name:           "500 Internal Server Error",
			method:         "GET",
			path:           "/error",
			requestID:      "",
			expectedStatus: http.StatusInternalServerError,
			wantRequestID:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture log output
			var buf bytes.Buffer
			log.Logger = zerolog.New(&buf)

			// Create test router
			router := gin.New()
			router.Use(Logger())

			// Add test handlers
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "ok"})
			})
			router.POST("/api/test", func(c *gin.Context) {
				c.JSON(http.StatusCreated, gin.H{"message": "created"})
			})
			router.GET("/notfound", func(c *gin.Context) {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			})
			router.GET("/error", func(c *gin.Context) {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			})

			// Create request
			req := httptest.NewRequest(tt.method, tt.path, nil)
			if tt.requestID != "" {
				req.Header.Set(RequestIDHeader, tt.requestID)
			}

			// Create response recorder
			w := httptest.NewRecorder()

			// Serve request
			router.ServeHTTP(w, req)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check request ID header
			if tt.wantRequestID {
				responseRequestID := w.Header().Get(RequestIDHeader)
				if responseRequestID == "" {
					t.Error("expected X-Request-ID header in response, got none")
				}

				if tt.requestID != "" && responseRequestID != tt.requestID {
					t.Errorf("expected X-Request-ID to be %s, got %s", tt.requestID, responseRequestID)
				}
			}

			// Check that log output is not empty
			logOutput := buf.String()
			if logOutput == "" {
				t.Error("expected log output, got empty string")
			}
		})
	}
}

func TestRequestID(t *testing.T) {
	// Set up gin in test mode
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name             string
		existingID       string
		expectGeneration bool
	}{
		{
			name:             "generates new request ID when not present",
			existingID:       "",
			expectGeneration: true,
		},
		{
			name:             "preserves existing request ID",
			existingID:       "existing-id-123",
			expectGeneration: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test router
			router := gin.New()
			router.Use(RequestID())

			// Add test handler
			router.GET("/test", func(c *gin.Context) {
				requestID, exists := c.Get("request_id")
				if !exists {
					t.Error("request_id not found in context")
				}

				if requestID == "" {
					t.Error("request_id is empty")
				}

				c.JSON(http.StatusOK, gin.H{"request_id": requestID})
			})

			// Create request
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.existingID != "" {
				req.Header.Set(RequestIDHeader, tt.existingID)
			}

			// Create response recorder
			w := httptest.NewRecorder()

			// Serve request
			router.ServeHTTP(w, req)

			// Check status code
			if w.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", w.Code)
			}

			// Check request ID header
			responseRequestID := w.Header().Get(RequestIDHeader)
			if responseRequestID == "" {
				t.Error("expected X-Request-ID header in response, got none")
			}

			if tt.existingID != "" && responseRequestID != tt.existingID {
				t.Errorf("expected X-Request-ID to be %s, got %s", tt.existingID, responseRequestID)
			}

			if tt.expectGeneration && responseRequestID == "" {
				t.Error("expected generated request ID, got empty string")
			}
		})
	}
}

func TestInitLogger(t *testing.T) {
	tests := []struct {
		name     string
		logLevel string
		env      string
	}{
		{
			name:     "development mode with debug level",
			logLevel: "debug",
			env:      "development",
		},
		{
			name:     "production mode with info level",
			logLevel: "info",
			env:      "production",
		},
		{
			name:     "production mode with warn level",
			logLevel: "warn",
			env:      "production",
		},
		{
			name:     "production mode with error level",
			logLevel: "error",
			env:      "production",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create config
			cfg := &config.Config{
				LogLevel: tt.logLevel,
				Env:      tt.env,
			}

			// Initialize logger (this should not panic)
			InitLogger(cfg)

			// Capture log output
			var buf bytes.Buffer
			log.Logger = zerolog.New(&buf)

			// Write a test log
			log.Info().Msg("test message")

			// The test passes if InitLogger doesn't panic
		})
	}
}

func TestInitLogger_DefaultLevel(t *testing.T) {
	// Create config with invalid log level
	cfg := &config.Config{
		LogLevel: "invalid",
		Env:      "development",
	}

	// Initialize logger (should default to info level)
	InitLogger(cfg)

	// Capture log output
	var buf bytes.Buffer
	log.Logger = zerolog.New(&buf).With().Timestamp().Logger()

	// Write a test log
	log.Info().Msg("test message")

	// Check that log output is not empty
	if buf.String() == "" {
		t.Error("expected log output, got empty string")
	}
}

func TestInitLogger_ConsoleOutput(t *testing.T) {
	// Save original stdout
	oldStdout := os.Stdout

	// Create config for development
	cfg := &config.Config{
		LogLevel: "info",
		Env:      "development",
	}

	// Initialize logger (should use console output)
	InitLogger(cfg)

	// Restore stdout
	os.Stdout = oldStdout

	// The test passes if InitLogger doesn't panic
}
