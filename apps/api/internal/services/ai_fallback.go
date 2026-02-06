package services

import (
	"context"
	"errors"
	"strings"

	"github.com/vido/api/internal/parser"
)

// AIFallbackMessage returns the user-friendly message shown when AI is unavailable.
// Per NFR-R11: "AI 服務暫時無法使用，使用基本解析"
func AIFallbackMessage() string {
	return "AI 服務暫時無法使用，使用基本解析"
}

// IsAIServiceError checks if the error is a service-level error that warrants fallback.
// Service errors include: quota exceeded, rate limits, timeouts, connection issues.
// Returns true for errors that should trigger regex fallback.
func IsAIServiceError(err error) bool {
	if err == nil {
		return false
	}

	// Check for context deadline exceeded
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	errStr := strings.ToLower(err.Error())
	serviceErrors := []string{
		"quota exceeded",
		"rate limit",
		"service unavailable",
		"timeout",
		"connection refused",
		"503",
		"429",
		"deadline exceeded",
	}

	for _, pattern := range serviceErrors {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// ApplyRegexFallback marks a parse result as coming from regex fallback.
// This should be called when regex parsing is used because AI was unavailable.
func ApplyRegexFallback(result *parser.ParseResult) {
	result.MetadataSource = parser.MetadataSourceRegexFallback
	result.DegradationMessage = AIFallbackMessage()
}

// CreateFallbackResult creates a parse result for when all parsing methods fail.
// This is used when neither AI nor regex can parse the filename.
func CreateFallbackResult(filename string, attemptedSources []string) *parser.ParseResult {
	return &parser.ParseResult{
		OriginalFilename:   filename,
		Status:             parser.ParseStatusNeedsAI,
		MediaType:          parser.MediaTypeUnknown,
		Confidence:         0,
		DegradationMessage: "無法自動識別此檔案",
		ErrorMessage:       "All parsing methods failed",
		MetadataSource:     parser.MetadataSource("none"),
	}
}
