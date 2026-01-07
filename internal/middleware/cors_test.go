package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexyu/vido/internal/config"
	"github.com/gin-gonic/gin"
)

func TestCORS(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		corsOrigins    []string
		requestOrigin  string
		requestMethod  string
		expectAllowed  bool
		expectStatus   int
	}{
		{
			name:          "Allowed origin - exact match",
			corsOrigins:   []string{"http://localhost:3000", "https://example.com"},
			requestOrigin: "http://localhost:3000",
			requestMethod: "GET",
			expectAllowed: true,
			expectStatus:  http.StatusOK,
		},
		{
			name:          "Allowed origin - different case",
			corsOrigins:   []string{"http://localhost:3000"},
			requestOrigin: "HTTP://LOCALHOST:3000",
			requestMethod: "GET",
			expectAllowed: true,
			expectStatus:  http.StatusOK,
		},
		{
			name:          "Wildcard origin",
			corsOrigins:   []string{"*"},
			requestOrigin: "https://any-domain.com",
			requestMethod: "GET",
			expectAllowed: true,
			expectStatus:  http.StatusOK,
		},
		{
			name:          "Disallowed origin",
			corsOrigins:   []string{"http://localhost:3000"},
			requestOrigin: "https://malicious.com",
			requestMethod: "GET",
			expectAllowed: false,
			expectStatus:  http.StatusOK,
		},
		{
			name:          "Preflight OPTIONS request - allowed origin",
			corsOrigins:   []string{"http://localhost:3000"},
			requestOrigin: "http://localhost:3000",
			requestMethod: "OPTIONS",
			expectAllowed: true,
			expectStatus:  http.StatusNoContent,
		},
		{
			name:          "Preflight OPTIONS request - disallowed origin",
			corsOrigins:   []string{"http://localhost:3000"},
			requestOrigin: "https://malicious.com",
			requestMethod: "OPTIONS",
			expectAllowed: false,
			expectStatus:  http.StatusNoContent,
		},
		{
			name:          "No origin header",
			corsOrigins:   []string{"http://localhost:3000"},
			requestOrigin: "",
			requestMethod: "GET",
			expectAllowed: false,
			expectStatus:  http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create config with test origins
			cfg := &config.Config{
				CORSOrigins: tt.corsOrigins,
			}

			// Create test router
			router := gin.New()
			router.Use(CORS(cfg))
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			// Create request
			req := httptest.NewRequest(tt.requestMethod, "/test", nil)
			if tt.requestOrigin != "" {
				req.Header.Set("Origin", tt.requestOrigin)
			}

			// Record response
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Check status code
			if w.Code != tt.expectStatus {
				t.Errorf("Expected status %d, got %d", tt.expectStatus, w.Code)
			}

			// Check CORS headers
			if tt.expectAllowed {
				// Should have Access-Control-Allow-Origin header
				allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
				if allowOrigin != tt.requestOrigin {
					t.Errorf("Expected Access-Control-Allow-Origin: %s, got: %s", tt.requestOrigin, allowOrigin)
				}

				// Should allow credentials
				allowCredentials := w.Header().Get("Access-Control-Allow-Credentials")
				if allowCredentials != "true" {
					t.Errorf("Expected Access-Control-Allow-Credentials: true, got: %s", allowCredentials)
				}

				// Should have allowed methods
				allowMethods := w.Header().Get("Access-Control-Allow-Methods")
				if allowMethods == "" {
					t.Error("Expected Access-Control-Allow-Methods header, got none")
				}

				// Should have allowed headers
				allowHeaders := w.Header().Get("Access-Control-Allow-Headers")
				if allowHeaders == "" {
					t.Error("Expected Access-Control-Allow-Headers header, got none")
				}

				// Check for required headers
				requiredHeaders := []string{"Authorization", "Content-Type"}
				for _, header := range requiredHeaders {
					if !containsHeader(allowHeaders, header) {
						t.Errorf("Expected %s in Access-Control-Allow-Headers, got: %s", header, allowHeaders)
					}
				}
			} else {
				// Should not have Access-Control-Allow-Origin header
				allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
				if allowOrigin != "" {
					t.Errorf("Expected no Access-Control-Allow-Origin header, got: %s", allowOrigin)
				}
			}
		})
	}
}

func TestIsOriginAllowed(t *testing.T) {
	tests := []struct {
		name           string
		origin         string
		allowedOrigins []string
		expected       bool
	}{
		{
			name:           "Exact match",
			origin:         "http://localhost:3000",
			allowedOrigins: []string{"http://localhost:3000"},
			expected:       true,
		},
		{
			name:           "Case insensitive match",
			origin:         "HTTP://LOCALHOST:3000",
			allowedOrigins: []string{"http://localhost:3000"},
			expected:       true,
		},
		{
			name:           "Wildcard",
			origin:         "https://any-domain.com",
			allowedOrigins: []string{"*"},
			expected:       true,
		},
		{
			name:           "Not in list",
			origin:         "https://malicious.com",
			allowedOrigins: []string{"http://localhost:3000", "https://example.com"},
			expected:       false,
		},
		{
			name:           "Empty origin",
			origin:         "",
			allowedOrigins: []string{"http://localhost:3000"},
			expected:       false,
		},
		{
			name:           "Multiple origins - match second",
			origin:         "https://example.com",
			allowedOrigins: []string{"http://localhost:3000", "https://example.com"},
			expected:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isOriginAllowed(tt.origin, tt.allowedOrigins)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for origin %s with allowed %v",
					tt.expected, result, tt.origin, tt.allowedOrigins)
			}
		})
	}
}

// Helper function to check if a header is in the comma-separated list
func containsHeader(headerList, header string) bool {
	headers := splitAndTrim(headerList, ",")
	for _, h := range headers {
		if h == header {
			return true
		}
	}
	return false
}

// Helper function to split and trim strings
func splitAndTrim(s, sep string) []string {
	parts := []string{}
	for _, part := range splitString(s, sep) {
		trimmed := trimSpace(part)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

// Helper to split string
func splitString(s, sep string) []string {
	if s == "" {
		return []string{}
	}
	result := []string{}
	current := ""
	for i := 0; i < len(s); i++ {
		if i <= len(s)-len(sep) && s[i:i+len(sep)] == sep {
			result = append(result, current)
			current = ""
			i += len(sep) - 1
		} else {
			current += string(s[i])
		}
	}
	result = append(result, current)
	return result
}

// Helper to trim whitespace
func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}
