package subtitle

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Supported OpenCC conversion profiles.
const (
	ProfileS2TWP = "s2twp" // Simplified → Traditional (Taiwan standard + Taiwan phrases)
)

// Converter wraps the official C++ OpenCC CLI for Chinese variant conversion.
type Converter struct {
	available bool
	helper    *openCCHelper
}

// openCCHelper invokes the official C++ OpenCC CLI.
type openCCHelper struct {
	path    string
	config  string
	timeout time.Duration
}

// NewConverter creates a Converter initialized with the s2twp profile.
// Returns an error if the helper is unavailable; callers can still use the
// degraded converter, which returns original content with an error.
func NewConverter() (*Converter, error) {
	path := os.Getenv("VIDO_OPENCC_BIN")
	if path == "" {
		path = "opencc"
	}
	resolved, err := exec.LookPath(path)
	if err != nil {
		return &Converter{available: false}, fmt.Errorf("opencc helper: %w", err)
	}
	config := os.Getenv("VIDO_OPENCC_CONFIG")
	if config == "" {
		config = "/usr/share/opencc/s2twp.json"
	}
	slog.Info("OpenCC C++ helper initialized", "profile", ProfileS2TWP, "binary", resolved, "config", config)
	return &Converter{available: true, helper: &openCCHelper{path: resolved, config: config, timeout: 30 * time.Second}}, nil
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
	if c.helper == nil {
		return content, fmt.Errorf("opencc: helper not initialized")
	}

	output, err := c.helper.convert(input, profile)
	if err != nil {
		return content, fmt.Errorf("opencc helper: %w", err)
	}
	if hasBOM {
		return append(append([]byte{}, bom...), []byte(output)...), nil
	}
	return []byte(output), nil
}

func (h *openCCHelper) convert(input, profile string) (string, error) {
	config := h.config
	if profile != ProfileS2TWP {
		config = filepath.Join(filepath.Dir(h.config), profile+".json")
		if _, err := os.Stat(config); err != nil {
			return "", fmt.Errorf("unsupported profile %q: %w", profile, err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()
	// OpenCC's CLI requires explicit input/output paths for streaming. The
	// procfs devices keep the helper stateless and avoid temporary files.
	cmd := exec.CommandContext(ctx, h.path, "-c", config, "-i", "/dev/stdin", "-o", "/dev/stdout")
	cmd.Stdin = bytes.NewBufferString(input)
	output, err := cmd.Output()
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("exit status %d: %s", exitErr.ExitCode(), bytes.TrimSpace(exitErr.Stderr))
		}
		return "", err
	}
	return string(output), nil
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
