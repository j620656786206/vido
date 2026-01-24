package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected Language
	}{
		// Chinese Traditional
		{name: "Traditional Chinese title", text: "鬼滅之刃", expected: LanguageChineseTraditional},
		{name: "Traditional Chinese anime", text: "進擊的巨人", expected: LanguageChineseTraditional},

		// Chinese Simplified
		{name: "Simplified Chinese title", text: "鬼灭之刃", expected: LanguageChineseSimplified},
		{name: "Simplified Chinese anime", text: "进击的巨人", expected: LanguageChineseSimplified},

		// Japanese
		{name: "Japanese with hiragana", text: "進撃の巨人", expected: LanguageJapanese},
		{name: "Pure hiragana", text: "しんげき", expected: LanguageJapanese},
		{name: "Pure katakana", text: "シンゲキ", expected: LanguageJapanese},
		{name: "Japanese mixed", text: "デーモンスレイヤー", expected: LanguageJapanese},

		// Korean
		{name: "Korean title", text: "오징어 게임", expected: LanguageKorean},
		{name: "Korean movie", text: "기생충", expected: LanguageKorean},

		// English
		{name: "English title", text: "Demon Slayer", expected: LanguageEnglish},
		{name: "English movie", text: "The Matrix", expected: LanguageEnglish},

		// Mixed - should detect dominant language
		{name: "Mixed Chinese English", text: "鬼滅之刃 Demon Slayer", expected: LanguageChineseTraditional},
		{name: "Mixed English numbers", text: "Demon Slayer 2020", expected: LanguageEnglish},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectLanguage(tt.text)
			assert.Equal(t, tt.expected, result, "DetectLanguage(%q) = %v, want %v", tt.text, result, tt.expected)
		})
	}
}

func TestLanguage_String(t *testing.T) {
	tests := []struct {
		lang     Language
		expected string
	}{
		{LanguageEnglish, "English"},
		{LanguageJapanese, "Japanese"},
		{LanguageChineseSimplified, "Simplified Chinese"},
		{LanguageChineseTraditional, "Traditional Chinese"},
		{LanguageKorean, "Korean"},
		{LanguageUnknown, "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.lang.String())
		})
	}
}

func TestContainsHiragana(t *testing.T) {
	assert.True(t, containsHiragana("の"))
	assert.True(t, containsHiragana("あいうえお"))
	assert.False(t, containsHiragana("ABC"))
	assert.False(t, containsHiragana("漢字"))
}

func TestContainsKatakana(t *testing.T) {
	assert.True(t, containsKatakana("デーモン"))
	assert.True(t, containsKatakana("アイウエオ"))
	assert.False(t, containsKatakana("ABC"))
	assert.False(t, containsKatakana("あいうえお"))
}

func TestContainsKorean(t *testing.T) {
	assert.True(t, containsKorean("한글"))
	assert.True(t, containsKorean("오징어"))
	assert.False(t, containsKorean("ABC"))
	assert.False(t, containsKorean("漢字"))
}

func TestContainsChinese(t *testing.T) {
	assert.True(t, containsChinese("漢字"))
	assert.True(t, containsChinese("鬼滅之刃"))
	assert.False(t, containsChinese("ABC"))
	assert.False(t, containsChinese("あいうえお"))
}

func TestIsSimplifiedChinese(t *testing.T) {
	tests := []struct {
		text     string
		expected bool
	}{
		// Simplified Chinese
		{"鬼灭之刃", true},
		{"进击的巨人", true},
		{"简体字", true},

		// Traditional Chinese
		{"鬼滅之刃", false},
		{"進擊的巨人", false},
		{"繁體字", false},

		// Non-Chinese
		{"ABC", false},
		{"あいうえお", false},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			result := isSimplifiedChinese(tt.text)
			assert.Equal(t, tt.expected, result, "isSimplifiedChinese(%q) = %v, want %v", tt.text, result, tt.expected)
		})
	}
}
