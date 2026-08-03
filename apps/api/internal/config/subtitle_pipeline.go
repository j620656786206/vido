package config

import "fmt"

// Subtitle generation pipeline mode (architecture D5, story sub-1-6 AC #1).
//
// The flag exists so the M1 pipeline can ship dark and be switched on per
// install without a code change. It is an ENV VAR, not a settings row:
// migration 029's cleanup of the table-backed `new_shell_enabled` flag is the
// cautionary precedent — a table-backed flag needs a migration to retire,
// drifts per-install, and cannot be read before the DB is open.
const (
	// SubtitlePipelineModeLegacy keeps the Epic-8 search engine as the batch
	// seam's processor. This is the default: an install that never sets the
	// variable behaves exactly as it did before M1 shipped.
	SubtitlePipelineModeLegacy = "legacy"

	// SubtitlePipelineModePipeline routes the batch seam to the generation
	// pipeline (extract → route → translate → place) and turns on the scanner
	// auto-enqueue and the manual run endpoint.
	SubtitlePipelineModePipeline = "pipeline"
)

// validSubtitlePipelineModes is the closed set. An unknown value is a startup
// error rather than a silent fallback to legacy: a typo'd `VIDO_SUBTITLE_PIPELINE_MODE=pipelien`
// that quietly kept the old behaviour would look exactly like "the pipeline is
// broken" to an operator who believes they enabled it.
var validSubtitlePipelineModes = map[string]struct{}{
	SubtitlePipelineModeLegacy:   {},
	SubtitlePipelineModePipeline: {},
}

// validateSubtitlePipelineMode rejects anything outside the closed set.
func validateSubtitlePipelineMode(mode string) error {
	if _, ok := validSubtitlePipelineModes[mode]; !ok {
		return fmt.Errorf("invalid VIDO_SUBTITLE_PIPELINE_MODE %q: must be %q or %q",
			mode, SubtitlePipelineModeLegacy, SubtitlePipelineModePipeline)
	}
	return nil
}

// SubtitlePipelineEnabled is the single predicate the wiring reads at startup
// (AC #1 — "flag read once at startup into the wiring, not per item"). Nothing
// downstream of main.go compares the raw string, which is what keeps the flag
// out of the pipeline stages, the handlers and the scanner path (D5's ban).
func (c *Config) SubtitlePipelineEnabled() bool {
	return c.SubtitlePipelineMode == SubtitlePipelineModePipeline
}
