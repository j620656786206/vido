// Package secrets provides secret masking functionality for safe logging.
package secrets

import (
	"context"
	"log/slog"
	"strings"
)

// Sensitive field name patterns for automatic masking
var sensitivePatterns = []string{
	"_key", "secret", "password", "token", "credential",
	"api_key", "apikey", "auth", "encryption",
}

// MaskSecret masks a secret value, showing only first 4 and last 4 characters.
// Returns "(not set)" for empty strings.
// Returns "****" for strings with 8 or fewer characters.
func MaskSecret(value string) string {
	if value == "" {
		return "(not set)"
	}
	if len(value) <= 8 {
		return "****"
	}
	return value[:4] + "****" + value[len(value)-4:]
}

// MaskSecretFull completely masks a secret value.
// Returns "(not set)" for empty strings.
// Returns "****" for any non-empty value.
func MaskSecretFull(value string) string {
	if value == "" {
		return "(not set)"
	}
	return "****"
}

// IsSensitiveField checks if a field name indicates sensitive data.
// Returns true if the field name contains any sensitive pattern (case-insensitive).
func IsSensitiveField(fieldName string) bool {
	lower := strings.ToLower(fieldName)
	for _, pattern := range sensitivePatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}

// MaskingHandler wraps an slog.Handler to automatically mask sensitive fields.
// Fields with names matching sensitive patterns will have their string values masked.
type MaskingHandler struct {
	inner slog.Handler
}

// NewMaskingHandler creates a new MaskingHandler that wraps the given handler.
// It automatically masks values of fields that match sensitive name patterns.
func NewMaskingHandler(inner slog.Handler) *MaskingHandler {
	return &MaskingHandler{inner: inner}
}

// Enabled implements slog.Handler.
func (h *MaskingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

// Handle implements slog.Handler.
// It creates a new record with sensitive fields masked before passing to the inner handler.
func (h *MaskingHandler) Handle(ctx context.Context, r slog.Record) error {
	// Create new record with masked attributes
	newRecord := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)

	r.Attrs(func(a slog.Attr) bool {
		newRecord.AddAttrs(h.maskAttr(a))
		return true
	})

	return h.inner.Handle(ctx, newRecord)
}

// WithAttrs implements slog.Handler.
// It masks any sensitive attributes before adding them.
func (h *MaskingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	maskedAttrs := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		maskedAttrs[i] = h.maskAttr(a)
	}
	return &MaskingHandler{inner: h.inner.WithAttrs(maskedAttrs)}
}

// WithGroup implements slog.Handler.
func (h *MaskingHandler) WithGroup(name string) slog.Handler {
	return &MaskingHandler{inner: h.inner.WithGroup(name)}
}

// maskAttr masks the attribute value if the key matches a sensitive pattern.
func (h *MaskingHandler) maskAttr(a slog.Attr) slog.Attr {
	if IsSensitiveField(a.Key) {
		// Mask string values, fully mask non-string values
		if str, ok := a.Value.Any().(string); ok {
			return slog.String(a.Key, MaskSecret(str))
		}
		return slog.String(a.Key, "****")
	}
	return a
}

// Compile-time interface verification
var _ slog.Handler = (*MaskingHandler)(nil)
