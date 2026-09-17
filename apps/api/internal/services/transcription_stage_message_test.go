package services

// Story dsr-6b AC #9 — the progress line under the generation stepper is shown
// verbatim by the dialog, the batch dialog and the generation workspace. It
// used to be English ("Transcribing audio with Whisper API").

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTranscriptionStageMessage_IsTraditionalChinese(t *testing.T) {
	cases := []struct {
		phase string
		want  string
	}{
		{"extracting", "正在提取音訊"},
		{"transcribing", "正在轉錄音訊"},
		{"translating", "正在翻譯成繁體中文"},
	}
	for _, tc := range cases {
		t.Run(tc.phase, func(t *testing.T) {
			assert.Equal(t, tc.want, transcriptionStageMessage(tc.phase))
		})
	}
}

func TestTranslationProgressMessage_CarriesTheRoundedPercent(t *testing.T) {
	assert.Equal(t, "正在翻譯成繁體中文（45%）", translationProgressMessage(45.2))
	assert.Equal(t, "正在翻譯成繁體中文（0%）", translationProgressMessage(0))
	assert.Equal(t, "正在翻譯成繁體中文（100%）", translationProgressMessage(99.6))
	// Batches of 8 land exactly on .5 (12.5, 37.5, 62.5, 87.5). The stepper beside
	// this line rounds with JS Math.round (half up); fmt's %.0f rounds half to
	// EVEN and would print 62% next to a 63% — both must say 63.
	assert.Equal(t, "正在翻譯成繁體中文（63%）", translationProgressMessage(62.5))
	assert.Equal(t, "正在翻譯成繁體中文（13%）", translationProgressMessage(12.5))
}
