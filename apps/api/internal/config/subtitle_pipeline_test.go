package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoad_SubtitlePipelineMode covers the D5 feature flag (story sub-1-6 AC #1).
//
// The flag is an ENV VAR rather than a settings row on purpose: migration 029's
// cleanup of the table-backed `new_shell_enabled` is the cautionary precedent —
// a table-backed flag needs a migration to retire and can drift per-install.
func TestLoad_SubtitlePipelineMode(t *testing.T) {
	tests := []struct {
		name    string
		env     string
		want    string
		wantErr bool
	}{
		{
			name: "unset defaults to legacy",
			env:  "",
			want: SubtitlePipelineModeLegacy,
		},
		{
			name: "legacy is accepted explicitly",
			env:  "legacy",
			want: SubtitlePipelineModeLegacy,
		},
		{
			name: "pipeline switches the batch seam",
			env:  "pipeline",
			want: SubtitlePipelineModePipeline,
		},
		{
			name:    "an unknown mode fails fast at startup",
			env:     "generate",
			wantErr: true,
		},
		{
			name:    "case matters — the value is a wire-ish contract, not a hint",
			env:     "Pipeline",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.env != "" {
				t.Setenv("VIDO_SUBTITLE_PIPELINE_MODE", tc.env)
			}

			cfg, err := Load()

			if tc.wantErr {
				require.Error(t, err, "an unknown mode must fail at startup, not silently fall back to legacy")
				assert.Contains(t, err.Error(), "VIDO_SUBTITLE_PIPELINE_MODE")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, cfg.SubtitlePipelineMode)
		})
	}
}

// TestConfig_SubtitlePipelineEnabled — the wiring reads the flag ONCE at startup
// through this predicate (AC #1: "flag read once at startup into the wiring, not
// per item"), so nothing downstream ever compares the raw string.
func TestConfig_SubtitlePipelineEnabled(t *testing.T) {
	assert.False(t, (&Config{SubtitlePipelineMode: SubtitlePipelineModeLegacy}).SubtitlePipelineEnabled())
	assert.True(t, (&Config{SubtitlePipelineMode: SubtitlePipelineModePipeline}).SubtitlePipelineEnabled())
	assert.False(t, (&Config{}).SubtitlePipelineEnabled(),
		"a zero-value Config must not silently enable the pipeline")
}
