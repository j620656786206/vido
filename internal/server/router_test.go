package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alexyu/vido/internal/config"
	"github.com/gin-gonic/gin"
)

func TestHealthCheck(t *testing.T) {
	// Set up gin in test mode
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		method         string
		expectedStatus int
		checkJSON      bool
	}{
		{
			name:           "GET /health returns 200 OK",
			method:         "GET",
			expectedStatus: http.StatusOK,
			checkJSON:      true,
		},
		{
			name:           "POST /health not allowed",
			method:         "POST",
			expectedStatus: http.StatusNotFound,
			checkJSON:      false,
		},
		{
			name:           "PUT /health not allowed",
			method:         "PUT",
			expectedStatus: http.StatusNotFound,
			checkJSON:      false,
		},
		{
			name:           "DELETE /health not allowed",
			method:         "DELETE",
			expectedStatus: http.StatusNotFound,
			checkJSON:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create config
			cfg := &config.Config{
				Port:        "8080",
				Env:         "test",
				LogLevel:    "info",
				CORSOrigins: []string{"http://localhost:3000"},
				APIVersion:  "v1",
			}

			// Create server
			server := New(cfg)

			// Create request
			req := httptest.NewRequest(tt.method, "/health", nil)
			w := httptest.NewRecorder()

			// Serve request
			server.router.ServeHTTP(w, req)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check JSON response for successful requests
			if tt.checkJSON {
				var response HealthResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				if err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}

				// Check status field
				if response.Status != "ok" {
					t.Errorf("expected status 'ok', got '%s'", response.Status)
				}

				// Check timestamp field exists and is valid RFC3339
				if response.Timestamp == "" {
					t.Error("expected timestamp to be set, got empty string")
				}

				_, err = time.Parse(time.RFC3339, response.Timestamp)
				if err != nil {
					t.Errorf("timestamp is not valid RFC3339 format: %v", err)
				}

				// Check Content-Type header
				contentType := w.Header().Get("Content-Type")
				if contentType != "application/json; charset=utf-8" {
					t.Errorf("expected Content-Type 'application/json; charset=utf-8', got '%s'", contentType)
				}
			}
		})
	}
}

func TestHealthCheckCORS(t *testing.T) {
	// Set up gin in test mode
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name          string
		origin        string
		corsOrigins   []string
		expectAllowed bool
	}{
		{
			name:          "Allowed origin",
			origin:        "http://localhost:3000",
			corsOrigins:   []string{"http://localhost:3000"},
			expectAllowed: true,
		},
		{
			name:          "Multiple allowed origins - match first",
			origin:        "http://localhost:3000",
			corsOrigins:   []string{"http://localhost:3000", "https://example.com"},
			expectAllowed: true,
		},
		{
			name:          "Multiple allowed origins - match second",
			origin:        "https://example.com",
			corsOrigins:   []string{"http://localhost:3000", "https://example.com"},
			expectAllowed: true,
		},
		{
			name:          "Wildcard origin",
			origin:        "https://any-domain.com",
			corsOrigins:   []string{"*"},
			expectAllowed: true,
		},
		{
			name:          "Disallowed origin",
			origin:        "https://malicious.com",
			corsOrigins:   []string{"http://localhost:3000"},
			expectAllowed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create config
			cfg := &config.Config{
				Port:        "8080",
				Env:         "test",
				LogLevel:    "info",
				CORSOrigins: tt.corsOrigins,
				APIVersion:  "v1",
			}

			// Create server
			server := New(cfg)

			// Create request with Origin header
			req := httptest.NewRequest("GET", "/health", nil)
			req.Header.Set("Origin", tt.origin)
			w := httptest.NewRecorder()

			// Serve request
			server.router.ServeHTTP(w, req)

			// Check status code (should always be 200 for health)
			if w.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", w.Code)
			}

			// Check CORS headers
			if tt.expectAllowed {
				allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
				if allowOrigin != tt.origin {
					t.Errorf("expected Access-Control-Allow-Origin: %s, got: %s", tt.origin, allowOrigin)
				}

				allowCredentials := w.Header().Get("Access-Control-Allow-Credentials")
				if allowCredentials != "true" {
					t.Errorf("expected Access-Control-Allow-Credentials: true, got: %s", allowCredentials)
				}

				allowMethods := w.Header().Get("Access-Control-Allow-Methods")
				if allowMethods == "" {
					t.Error("expected Access-Control-Allow-Methods header, got none")
				}

				allowHeaders := w.Header().Get("Access-Control-Allow-Headers")
				if allowHeaders == "" {
					t.Error("expected Access-Control-Allow-Headers header, got none")
				}
			} else {
				allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
				if allowOrigin != "" {
					t.Errorf("expected no Access-Control-Allow-Origin header, got: %s", allowOrigin)
				}
			}
		})
	}
}

func TestHealthCheckPreflightCORS(t *testing.T) {
	// Set up gin in test mode
	gin.SetMode(gin.TestMode)

	// Create config
	cfg := &config.Config{
		Port:        "8080",
		Env:         "test",
		LogLevel:    "info",
		CORSOrigins: []string{"http://localhost:3000"},
		APIVersion:  "v1",
	}

	// Create server
	server := New(cfg)

	// Create preflight OPTIONS request
	req := httptest.NewRequest("OPTIONS", "/health", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "GET")
	w := httptest.NewRecorder()

	// Serve request
	server.router.ServeHTTP(w, req)

	// Check status code (should be 204 No Content for OPTIONS)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", w.Code)
	}

	// Check CORS headers are present
	allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
	if allowOrigin != "http://localhost:3000" {
		t.Errorf("expected Access-Control-Allow-Origin: http://localhost:3000, got: %s", allowOrigin)
	}

	allowMethods := w.Header().Get("Access-Control-Allow-Methods")
	if allowMethods == "" {
		t.Error("expected Access-Control-Allow-Methods header, got none")
	}

	allowHeaders := w.Header().Get("Access-Control-Allow-Headers")
	if allowHeaders == "" {
		t.Error("expected Access-Control-Allow-Headers header, got none")
	}
}

func TestRequestIDMiddleware(t *testing.T) {
	// Set up gin in test mode
	gin.SetMode(gin.TestMode)

	// Create config
	cfg := &config.Config{
		Port:        "8080",
		Env:         "test",
		LogLevel:    "info",
		CORSOrigins: []string{"http://localhost:3000"},
		APIVersion:  "v1",
	}

	// Create server
	server := New(cfg)

	t.Run("generates request ID when not provided", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		requestID := w.Header().Get("X-Request-ID")
		if requestID == "" {
			t.Error("expected X-Request-ID header in response, got none")
		}
	})

	t.Run("preserves existing request ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		req.Header.Set("X-Request-ID", "test-id-123")
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		requestID := w.Header().Get("X-Request-ID")
		if requestID != "test-id-123" {
			t.Errorf("expected X-Request-ID 'test-id-123', got '%s'", requestID)
		}
	})
}

func TestErrorExampleEndpoint(t *testing.T) {
	// Set up gin in test mode
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		queryParam     string
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "No error type - success response",
			queryParam:     "",
			expectedStatus: http.StatusOK,
			expectedCode:   "",
		},
		{
			name:           "Validation error",
			queryParam:     "validation",
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "VALIDATION_ERROR",
		},
		{
			name:           "Not found error",
			queryParam:     "notfound",
			expectedStatus: http.StatusNotFound,
			expectedCode:   "NOT_FOUND",
		},
		{
			name:           "Internal error",
			queryParam:     "internal",
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   "INTERNAL_ERROR",
		},
		{
			name:           "Unauthorized error",
			queryParam:     "unauthorized",
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   "UNAUTHORIZED",
		},
		{
			name:           "Forbidden error",
			queryParam:     "forbidden",
			expectedStatus: http.StatusForbidden,
			expectedCode:   "FORBIDDEN",
		},
		{
			name:           "Panic recovery",
			queryParam:     "panic",
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create config
			cfg := &config.Config{
				Port:        "8080",
				Env:         "test",
				LogLevel:    "info",
				CORSOrigins: []string{"http://localhost:3000"},
				APIVersion:  "v1",
			}

			// Create server
			server := New(cfg)

			// Create request
			url := "/api/v1/error-example"
			if tt.queryParam != "" {
				url += "?type=" + tt.queryParam
			}
			req := httptest.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()

			// Serve request
			server.router.ServeHTTP(w, req)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check response format
			if tt.expectedCode != "" {
				// Error response
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				if err != nil {
					t.Fatalf("failed to unmarshal error response: %v", err)
				}

				errorObj, ok := response["error"].(map[string]interface{})
				if !ok {
					t.Fatal("expected error object in response")
				}

				code, ok := errorObj["code"].(string)
				if !ok {
					t.Fatal("expected code in error object")
				}

				if code != tt.expectedCode {
					t.Errorf("expected error code '%s', got '%s'", tt.expectedCode, code)
				}

				message, ok := errorObj["message"].(string)
				if !ok || message == "" {
					t.Error("expected message in error object")
				}
			} else {
				// Success response
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				if err != nil {
					t.Fatalf("failed to unmarshal success response: %v", err)
				}

				message, ok := response["message"].(string)
				if !ok || message == "" {
					t.Error("expected message in success response")
				}
			}
		})
	}
}

func TestSwaggerEndpoint(t *testing.T) {
	// Set up gin in test mode
	gin.SetMode(gin.TestMode)

	// Create config
	cfg := &config.Config{
		Port:        "8080",
		Env:         "test",
		LogLevel:    "info",
		CORSOrigins: []string{"http://localhost:3000"},
		APIVersion:  "v1",
	}

	// Create server
	server := New(cfg)

	// Test swagger UI endpoint
	req := httptest.NewRequest("GET", "/swagger/index.html", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	// Should return 200 or 301 redirect
	if w.Code != http.StatusOK && w.Code != http.StatusMovedPermanently {
		t.Errorf("expected status 200 or 301, got %d", w.Code)
	}
}

func TestAPIv1RouteGroup(t *testing.T) {
	// Set up gin in test mode
	gin.SetMode(gin.TestMode)

	// Create config
	cfg := &config.Config{
		Port:        "8080",
		Env:         "test",
		LogLevel:    "info",
		CORSOrigins: []string{"http://localhost:3000"},
		APIVersion:  "v1",
	}

	// Create server
	server := New(cfg)

	t.Run("API v1 route group exists", func(t *testing.T) {
		// Test that /api/v1/error-example is accessible
		req := httptest.NewRequest("GET", "/api/v1/error-example", nil)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		// Should return 200 (not 404)
		if w.Code != http.StatusOK {
			t.Errorf("expected status 200 for /api/v1/error-example, got %d", w.Code)
		}
	})

	t.Run("Non-existent v1 route returns 404", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/nonexistent", nil)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		// Should return 404
		if w.Code != http.StatusNotFound {
			t.Errorf("expected status 404 for non-existent route, got %d", w.Code)
		}
	})
}

func TestServerIntegration(t *testing.T) {
	// Set up gin in test mode
	gin.SetMode(gin.TestMode)

	// Create config
	cfg := &config.Config{
		Port:        "8080",
		Env:         "test",
		LogLevel:    "info",
		CORSOrigins: []string{"http://localhost:3000"},
		APIVersion:  "v1",
	}

	// Create server
	server := New(cfg)

	t.Run("All middleware applied correctly", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		// Check that all middleware worked
		// 1. Recovery middleware - didn't panic
		// 2. Logger middleware - added request ID
		// 3. CORS middleware - added CORS headers
		// 4. Error handler - not triggered for success

		// Status should be 200
		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		// Request ID should be present
		if w.Header().Get("X-Request-ID") == "" {
			t.Error("expected X-Request-ID header, got none")
		}

		// CORS headers should be present
		if w.Header().Get("Access-Control-Allow-Origin") == "" {
			t.Error("expected Access-Control-Allow-Origin header, got none")
		}

		// Response should be valid JSON
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
	})
}

func TestRouterExport(t *testing.T) {
	// Set up gin in test mode
	gin.SetMode(gin.TestMode)

	// Create config
	cfg := &config.Config{
		Port:        "8080",
		Env:         "test",
		LogLevel:    "info",
		CORSOrigins: []string{"http://localhost:3000"},
		APIVersion:  "v1",
	}

	// Create server
	server := New(cfg)

	// Test Router() method
	router := server.Router()
	if router == nil {
		t.Fatal("expected Router() to return non-nil router")
	}

	// Verify it's the same router instance
	if router != server.router {
		t.Error("expected Router() to return the same router instance")
	}
}
