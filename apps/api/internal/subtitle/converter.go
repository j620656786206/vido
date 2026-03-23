package subtitle

import (
	"bytes"
	"fmt"
	"log/slog"

	"github.com/longbridgeapp/opencc"
)

// Supported OpenCC conversion profiles.
const (
	ProfileS2TWP = "s2twp" // Simplified → Traditional (Taiwan standard + Taiwan phrases)
)

// Converter wraps OpenCC for Chinese variant conversion.
// It uses the pure Go opencc binding (no external binary required).
type Converter struct {
	cc        *opencc.OpenCC
	available bool
}

// NewConverter creates a Converter initialized with the s2twp profile.
// Returns an error if OpenCC initialization fails, but the Converter is still
// usable in degraded mode (IsAvailable returns false, Convert returns originals).
func NewConverter() (*Converter, error) {
	cc, err := opencc.New(ProfileS2TWP)
	if err != nil {
		slog.Warn("OpenCC initialization failed — converter will operate in degraded mode",
			"profile", ProfileS2TWP,
			"error", err,
		)
		return &Converter{available: false}, fmt.Errorf("opencc init: %w", err)
	}

	slog.Info("OpenCC converter initialized", "profile", ProfileS2TWP)
	return &Converter{cc: cc, available: true}, nil
}

// IsAvailable returns true if OpenCC can perform conversions.
func (c *Converter) IsAvailable() bool {
	return c.available
}

// Convert converts subtitle content using the specified OpenCC profile.
// On any error, returns the original content unchanged along with the error
// (graceful degradation — unconverted subtitle is better than no subtitle).
func (c *Converter) Convert(content []byte, profile string) ([]byte, error) {
	if len(content) == 0 {
		return content, nil
	}

	if !c.available {
		return content, fmt.Errorf("opencc: converter not available")
	}

	// Strip UTF-8 BOM if present
	stripped := bytes.TrimPrefix(content, []byte{0xEF, 0xBB, 0xBF})
	hasBOM := len(stripped) < len(content)

	input := string(stripped)

	// If a different profile is requested, create a temporary converter
	var cc *opencc.OpenCC
	if profile != ProfileS2TWP {
		var err error
		cc, err = opencc.New(profile)
		if err != nil {
			return content, fmt.Errorf("opencc: unsupported profile %q: %w", profile, err)
		}
	} else {
		cc = c.cc
	}

	output, err := cc.Convert(input)
	if err != nil {
		slog.Warn("OpenCC conversion failed — returning original content",
			"profile", profile,
			"error", err,
			"content_length", len(content),
		)
		return content, fmt.Errorf("opencc: conversion failed: %w", err)
	}

	result := []byte(output)

	// Restore BOM if original had one
	if hasBOM {
		result = append([]byte{0xEF, 0xBB, 0xBF}, result...)
	}

	return result, nil
}

// ConvertS2TWP is a convenience method for Simplified → Traditional (Taiwan phrases).
// This is the primary conversion profile for Vido's zh-TW subtitle pipeline.
//
// Calling ConvertS2TWP on already-Traditional text is safe (idempotent) —
// OpenCC's s2twp profile only transforms simplified-unique characters and
// mainland phrases, leaving traditional text unchanged.
func (c *Converter) ConvertS2TWP(content []byte) ([]byte, error) {
	return c.Convert(content, ProfileS2TWP)
}

// NeedsConversion returns true only for Simplified Chinese ("zh-Hans").
// Returns false for "zh-Hant" (already traditional), "zh" (ambiguous),
// "und" (undetermined), or any non-Chinese language.
func NeedsConversion(language string) bool {
	return language == LangSimplified
}
