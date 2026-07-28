package subtitle

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Story sub-1-3 AC #6.2: SUBTITLE_ sentinel wire format + errors.Is identity ---

func subtitleSentinels() []struct {
	name string
	err  error
	code string
} {
	return []struct {
		name string
		err  error
		code string
	}{
		{"extract failed", ErrSubtitleExtractFailed, "SUBTITLE_EXTRACT_FAILED"},
		{"no text source", ErrSubtitleNoTextSource, "SUBTITLE_NO_TEXT_SOURCE"},
		{"translate failed", ErrSubtitleTranslateFailed, "SUBTITLE_TRANSLATE_FAILED"},
		{"timestamp mismatch", ErrSubtitleTimestampMismatch, "SUBTITLE_TIMESTAMP_MISMATCH"},
	}
}

// TestSubtitleSentinelWireFormat locks the Rule 7 {SOURCE}_{ERROR_TYPE} shape.
// The code prefix is what sub-1-6 lifts into ApiResponse.error.code, so the
// literal is a wire contract even though the sentinel itself is Go-internal.
func TestSubtitleSentinelWireFormat(t *testing.T) {
	for _, tt := range subtitleSentinels() {
		t.Run(tt.name, func(t *testing.T) {
			require.Error(t, tt.err)
			msg := tt.err.Error()

			assert.Equal(t, tt.code+": ", msg[:len(tt.code)+2],
				"the message must START with the exact %q code followed by ': '", tt.code)
			assert.Greater(t, len(msg), len(tt.code)+2,
				"a bare code with no human-readable tail is not an error message")
			assert.True(t, strings.HasPrefix(tt.code, "SUBTITLE_"),
				"Rule 7 D7: new codes extend the EXISTING SUBTITLE_ prefix — no new prefix")
		})
	}
}

// TestSubtitleSentinelErrorsIsRoundTrip covers the consumers' actual usage
// shape: sub-1-4/1-5a wrap with %w and classify with errors.Is.
func TestSubtitleSentinelErrorsIsRoundTrip(t *testing.T) {
	for _, tt := range subtitleSentinels() {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := fmt.Errorf("extract track 3: %w", tt.err)
			assert.ErrorIs(t, wrapped, tt.err, "wrapping must preserve sentinel identity")

			doubleWrapped := fmt.Errorf("pipeline item 42: %w", wrapped)
			assert.ErrorIs(t, doubleWrapped, tt.err, "identity must survive nested wrapping")

			for _, other := range subtitleSentinels() {
				if other.code == tt.code {
					continue
				}
				assert.NotErrorIs(t, wrapped, other.err,
					"%s must not match %s — the sentinels are distinct classifications", tt.code, other.code)
			}
		})
	}
}
