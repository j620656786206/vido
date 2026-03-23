package subtitle

import (
	"bytes"
	"fmt"
	"log/slog"
	"sync"

	"github.com/longbridgeapp/opencc"
)

// Supported OpenCC conversion profiles.
const (
	ProfileS2TWP = "s2twp" // Simplified → Traditional (Taiwan standard + Taiwan phrases)
)

// Converter wraps OpenCC for Chinese variant conversion.
// It uses the pure Go opencc binding (no external binary required).
//
// Converter is safe for concurrent use. The underlying opencc library performs
// read-only dictionary lookups after initialization, and non-default profiles
// are cached with a sync.Map to avoid per-call allocation.
type Converter struct {
	cc        *opencc.OpenCC
	available bool
	cache     sync.Map // profile string → *opencc.OpenCC
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
	c := &Converter{cc: cc, available: true}
	c.cache.Store(ProfileS2TWP, cc)
	return c, nil
}

// IsAvailable returns true if OpenCC can perform conversions.
func (c *Converter) IsAvailable() bool {
	if c == nil {
		return false
	}
	return c.available
}

// Convert converts subtitle content using the specified OpenCC profile.
// On any error, returns the original content unchanged along with the error
// (graceful degradation — unconverted subtitle is better than no subtitle).
func (c *Converter) Convert(content []byte, profile string) ([]byte, error) {
	if c == nil {
		return content, fmt.Errorf("opencc: nil converter")
	}

	if len(content) == 0 {
		return content, nil
	}

	if !c.available {
		return content, fmt.Errorf("opencc: converter not available")
	}

	// Strip UTF-8 BOM if present
	bom := []byte{0xEF, 0xBB, 0xBF}
	stripped := bytes.TrimPrefix(content, bom)
	hasBOM := len(stripped) < len(content)

	input := string(stripped)

	// Look up or create the converter for this profile (cached)
	cc, err := c.getOrCreateCC(profile)
	if err != nil {
		return content, fmt.Errorf("opencc: unsupported profile %q: %w", profile, err)
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

	// Pre-allocate result with BOM space if needed
	if hasBOM {
		result := make([]byte, 0, len(bom)+len(output))
		result = append(result, bom...)
		result = append(result, output...)
		return result, nil
	}

	return []byte(output), nil
}

// getOrCreateCC returns a cached OpenCC instance for the given profile,
// creating and caching one if it doesn't exist yet.
func (c *Converter) getOrCreateCC(profile string) (*opencc.OpenCC, error) {
	if v, ok := c.cache.Load(profile); ok {
		return v.(*opencc.OpenCC), nil
	}
	cc, err := opencc.New(profile)
	if err != nil {
		return nil, err
	}
	// Store-or-load: if another goroutine raced us, use theirs and discard ours
	actual, _ := c.cache.LoadOrStore(profile, cc)
	return actual.(*opencc.OpenCC), nil
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
