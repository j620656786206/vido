package secrets

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaskSecret(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", "(not set)"},
		{"very short", "abc", "****"},
		{"exactly 8 chars", "12345678", "****"},
		{"9 chars", "123456789", "1234****6789"},
		{"typical API key", "sk-1234567890abcdef", "sk-1****cdef"},
		{"long string", "this-is-a-very-long-api-key-value", "this****alue"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskSecret(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMaskSecretFull(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", "(not set)"},
		{"short string", "abc", "****"},
		{"long string", "sk-1234567890abcdef", "****"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskSecretFull(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsSensitiveField(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"api_key", "api_key", true},
		{"API_KEY", "API_KEY", true},
		{"tmdb_api_key", "tmdb_api_key", true},
		{"password", "password", true},
		{"Password", "Password", true},
		{"user_password", "user_password", true},
		{"token", "token", true},
		{"access_token", "access_token", true},
		{"secret", "secret", true},
		{"client_secret", "client_secret", true},
		{"auth", "auth", true},
		{"authorization", "authorization", true},
		{"credential", "credential", true},
		{"encryption_key", "encryption_key", true},
		{"username", "username", false},
		{"email", "email", false},
		{"name", "name", false},
		{"status", "status", false},
		{"created_at", "created_at", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSensitiveField(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMaskingHandler_MasksSensitiveFields(t *testing.T) {
	var buf bytes.Buffer
	baseHandler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	maskingHandler := NewMaskingHandler(baseHandler)
	logger := slog.New(maskingHandler)

	// Log with sensitive fields
	logger.Info("test message",
		"api_key", "sk-1234567890abcdef",
		"password", "secret123",
		"username", "john",
	)

	output := buf.String()

	// api_key should be masked
	assert.Contains(t, output, "api_key")
	assert.NotContains(t, output, "sk-1234567890abcdef")
	assert.Contains(t, output, "sk-1****cdef")

	// password should be masked
	assert.Contains(t, output, "password")
	assert.NotContains(t, output, "secret123")
	assert.Contains(t, output, "****")

	// username should NOT be masked
	assert.Contains(t, output, "username")
	assert.Contains(t, output, "john")
}

func TestMaskingHandler_Enabled(t *testing.T) {
	var buf bytes.Buffer
	baseHandler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})
	maskingHandler := NewMaskingHandler(baseHandler)

	ctx := context.Background()

	// Should respect base handler's level
	assert.False(t, maskingHandler.Enabled(ctx, slog.LevelDebug))
	assert.False(t, maskingHandler.Enabled(ctx, slog.LevelInfo))
	assert.True(t, maskingHandler.Enabled(ctx, slog.LevelWarn))
	assert.True(t, maskingHandler.Enabled(ctx, slog.LevelError))
}

func TestMaskingHandler_WithAttrs(t *testing.T) {
	var buf bytes.Buffer
	baseHandler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	maskingHandler := NewMaskingHandler(baseHandler)

	// Add attributes with sensitive data
	handlerWithAttrs := maskingHandler.WithAttrs([]slog.Attr{
		slog.String("api_key", "sk-test-key-12345"),
	})

	logger := slog.New(handlerWithAttrs)
	logger.Info("test")

	output := buf.String()
	assert.Contains(t, output, "sk-t****2345")
	assert.NotContains(t, output, "sk-test-key-12345")
}

func TestMaskingHandler_WithGroup(t *testing.T) {
	var buf bytes.Buffer
	baseHandler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	maskingHandler := NewMaskingHandler(baseHandler)

	handlerWithGroup := maskingHandler.WithGroup("config")
	logger := slog.New(handlerWithGroup)

	logger.Info("test", "api_key", "sk-group-test-key")

	output := buf.String()
	// Should still mask even within groups
	assert.NotContains(t, output, "sk-group-test-key")
}

func TestMaskingHandler_NonStringValues(t *testing.T) {
	var buf bytes.Buffer
	baseHandler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	maskingHandler := NewMaskingHandler(baseHandler)
	logger := slog.New(maskingHandler)

	// Log with non-string sensitive field (should still mask)
	logger.Info("test",
		"api_key", 12345, // integer value
		"count", 100,
	)

	output := buf.String()
	// Sensitive field with non-string should be fully masked
	assert.Contains(t, output, "api_key=****")
	// Non-sensitive field should not be masked
	assert.Contains(t, output, "count=100")
}

func TestSensitivePatterns(t *testing.T) {
	// Test that sensitive patterns match expected patterns from AC #4
	expectedPatterns := []string{"_key", "_secret", "password", "token"}

	for _, pattern := range expectedPatterns {
		found := false
		for _, p := range sensitivePatterns {
			if strings.Contains(p, pattern) || strings.Contains(pattern, p) {
				found = true
				break
			}
		}
		if !found {
			// Check if the pattern itself would be caught
			testField := "test" + pattern
			if IsSensitiveField(testField) {
				found = true
			}
		}
		assert.True(t, found, "Expected pattern %q to be in sensitive patterns", pattern)
	}
}
