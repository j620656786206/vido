package ai

import (
	"unicode"
)

// Language represents a detected language.
type Language string

const (
	LanguageUnknown            Language = "unknown"
	LanguageEnglish            Language = "en"
	LanguageJapanese           Language = "ja"
	LanguageKorean             Language = "ko"
	LanguageChineseSimplified  Language = "zh-CN"
	LanguageChineseTraditional Language = "zh-TW"
)

// String returns the human-readable language name.
func (l Language) String() string {
	switch l {
	case LanguageEnglish:
		return "English"
	case LanguageJapanese:
		return "Japanese"
	case LanguageKorean:
		return "Korean"
	case LanguageChineseSimplified:
		return "Simplified Chinese"
	case LanguageChineseTraditional:
		return "Traditional Chinese"
	default:
		return "Unknown"
	}
}

// DetectLanguage detects the primary language of the given text.
// It uses Unicode character ranges to identify CJK languages.
// Priority: Japanese (if hiragana/katakana present) > Korean > Chinese > English
func DetectLanguage(text string) Language {
	if text == "" {
		return LanguageUnknown
	}

	// Count character types
	var (
		hasHiragana bool
		hasKatakana bool
		hasKorean   bool
		hasChinese  bool
		hasLatin    bool
	)

	for _, r := range text {
		switch {
		case isHiragana(r):
			hasHiragana = true
		case isKatakana(r):
			hasKatakana = true
		case isKorean(r):
			hasKorean = true
		case isChinese(r):
			hasChinese = true
		case isLatin(r):
			hasLatin = true
		}
	}

	// Japanese detection: presence of hiragana or katakana is definitive
	if hasHiragana || hasKatakana {
		return LanguageJapanese
	}

	// Korean detection
	if hasKorean {
		return LanguageKorean
	}

	// Chinese detection
	if hasChinese {
		if isSimplifiedChinese(text) {
			return LanguageChineseSimplified
		}
		return LanguageChineseTraditional
	}

	// English/Latin detection
	if hasLatin {
		return LanguageEnglish
	}

	return LanguageUnknown
}

// isHiragana checks if a rune is a Hiragana character.
// Hiragana range: U+3040 - U+309F
func isHiragana(r rune) bool {
	return r >= 0x3040 && r <= 0x309F
}

// isKatakana checks if a rune is a Katakana character.
// Katakana range: U+30A0 - U+30FF
func isKatakana(r rune) bool {
	return r >= 0x30A0 && r <= 0x30FF
}

// isKorean checks if a rune is a Hangul character.
// Hangul Syllables range: U+AC00 - U+D7A3
// Hangul Jamo range: U+1100 - U+11FF
func isKorean(r rune) bool {
	return (r >= 0xAC00 && r <= 0xD7A3) || (r >= 0x1100 && r <= 0x11FF)
}

// isChinese checks if a rune is a CJK ideograph.
// CJK Unified Ideographs: U+4E00 - U+9FFF
// CJK Extension A: U+3400 - U+4DBF
func isChinese(r rune) bool {
	return (r >= 0x4E00 && r <= 0x9FFF) || (r >= 0x3400 && r <= 0x4DBF)
}

// isLatin checks if a rune is a Latin character.
func isLatin(r rune) bool {
	return unicode.IsLetter(r) && r < 0x0100
}

// containsHiragana checks if the text contains any Hiragana characters.
func containsHiragana(text string) bool {
	for _, r := range text {
		if isHiragana(r) {
			return true
		}
	}
	return false
}

// containsKatakana checks if the text contains any Katakana characters.
func containsKatakana(text string) bool {
	for _, r := range text {
		if isKatakana(r) {
			return true
		}
	}
	return false
}

// containsKorean checks if the text contains any Hangul characters.
func containsKorean(text string) bool {
	for _, r := range text {
		if isKorean(r) {
			return true
		}
	}
	return false
}

// containsChinese checks if the text contains any CJK ideographs.
func containsChinese(text string) bool {
	for _, r := range text {
		if isChinese(r) {
			return true
		}
	}
	return false
}

// simplifiedOnlyChars contains characters that exist only in Simplified Chinese.
// These are commonly used characters that have different forms in Traditional Chinese.
var simplifiedOnlyChars = map[rune]bool{
	// Common simplified characters
	'这': true, '国': true, '时': true, '为': true, '会': true,
	'动': true, '发': true, '开': true, '对': true, '经': true,
	'问': true, '说': true, '还': true, '进': true, '头': true,
	'个': true, '两': true, '长': true, '业': true, '实': true,
	'学': true, '门': true, '见': true, '关': true, '东': true,
	'无': true, '专': true, '乐': true, '书': true, '习': true,
	'马': true, '鸟': true, '鱼': true, '龙': true, '龟': true,
	'车': true, '贝': true, '风': true, '飞': true, '齐': true,
	'办': true, '让': true, '计': true, '认': true, '议': true,
	'设': true, '证': true, '话': true, '该': true, '请': true,
	'语': true, '读': true, '谁': true, '调': true, '论': true,
	'识': true, '远': true, '运': true, '连': true, '选': true,
	'达': true, '边': true, '迁': true, '迟': true, '适': true,
	'递': true, '遗': true, '电': true, '灭': true, '击': true,
	'犹': true, '狱': true, '狮': true, '独': true, '狭': true,
	'猎': true, '简': true, '体': true,
}

// traditionalOnlyChars contains characters that exist only in Traditional Chinese.
var traditionalOnlyChars = map[rune]bool{
	// Common traditional characters
	'這': true, '國': true, '時': true, '為': true, '會': true,
	'動': true, '發': true, '開': true, '對': true, '經': true,
	'問': true, '說': true, '還': true, '進': true, '頭': true,
	'個': true, '兩': true, '長': true, '業': true, '實': true,
	'學': true, '門': true, '見': true, '關': true, '東': true,
	'無': true, '專': true, '樂': true, '書': true, '習': true,
	'馬': true, '鳥': true, '魚': true, '龍': true, '龜': true,
	'車': true, '貝': true, '風': true, '飛': true, '齊': true,
	'辦': true, '讓': true, '計': true, '認': true, '議': true,
	'設': true, '證': true, '話': true, '該': true, '請': true,
	'語': true, '讀': true, '誰': true, '調': true, '論': true,
	'識': true, '遠': true, '運': true, '連': true, '選': true,
	'達': true, '邊': true, '遷': true, '遲': true, '適': true,
	'遞': true, '遺': true, '電': true, '滅': true, '擊': true,
	'猶': true, '獄': true, '獅': true, '獨': true, '狹': true,
	'獵': true,
}

// isSimplifiedChinese checks if the Chinese text is in Simplified Chinese.
// It counts character occurrences unique to each script variant.
func isSimplifiedChinese(text string) bool {
	simplifiedCount := 0
	traditionalCount := 0

	for _, r := range text {
		if simplifiedOnlyChars[r] {
			simplifiedCount++
		}
		if traditionalOnlyChars[r] {
			traditionalCount++
		}
	}

	// If we found any script-specific characters, use them to decide
	if simplifiedCount > 0 || traditionalCount > 0 {
		return simplifiedCount > traditionalCount
	}

	// If no script-specific characters found, assume Traditional (safer default for Taiwan market)
	return false
}
