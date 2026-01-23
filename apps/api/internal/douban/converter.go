package douban

import (
	"log/slog"
	"sync"

	"github.com/longbridgeapp/opencc"
)

// ChineseConverter handles Simplified to Traditional Chinese conversion
type ChineseConverter struct {
	converter *opencc.OpenCC
	logger    *slog.Logger
	mu        sync.RWMutex
	initErr   error
	initOnce  sync.Once
}

// NewChineseConverter creates a new Chinese converter
// Uses s2twp profile: Simplified to Traditional (Taiwan + phrases)
func NewChineseConverter(logger *slog.Logger) *ChineseConverter {
	if logger == nil {
		logger = slog.Default()
	}
	return &ChineseConverter{
		logger: logger,
	}
}

// init lazily initializes the converter
func (c *ChineseConverter) init() error {
	c.initOnce.Do(func() {
		// Use s2twp for best Traditional Chinese (Taiwan) conversion
		// s2twp = Simplified to Traditional (Taiwan) with phrases
		converter, err := opencc.New("s2twp")
		if err != nil {
			c.logger.Error("Failed to initialize OpenCC converter",
				"profile", "s2twp",
				"error", err,
			)
			c.initErr = err
			return
		}
		c.converter = converter
		c.logger.Info("OpenCC converter initialized",
			"profile", "s2twp",
		)
	})
	return c.initErr
}

// ToTraditional converts Simplified Chinese text to Traditional Chinese (Taiwan)
func (c *ChineseConverter) ToTraditional(simplified string) (string, error) {
	if simplified == "" {
		return "", nil
	}

	if err := c.init(); err != nil {
		return simplified, err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.converter == nil {
		return simplified, nil
	}

	traditional, err := c.converter.Convert(simplified)
	if err != nil {
		c.logger.Warn("Failed to convert to Traditional Chinese",
			"error", err,
			"input_length", len(simplified),
		)
		return simplified, err
	}

	return traditional, nil
}

// IsTraditional checks if the text appears to be Traditional Chinese
// This is a heuristic check based on common Traditional Chinese characters
func (c *ChineseConverter) IsTraditional(text string) bool {
	if text == "" {
		return false
	}

	// Common Traditional Chinese characters that are different from Simplified
	traditionalChars := []rune{
		'國', '學', '體', '機', '關', '發', '電', '頭', '時',
		'東', '車', '書', '長', '門', '問', '開', '間', '馬',
		'風', '飛', '魚', '鳥', '麗', '黃', '點', '龍', '齊',
	}

	// Count occurrences of Traditional characters
	traditionalCount := 0
	runeText := []rune(text)

	for _, char := range runeText {
		for _, tc := range traditionalChars {
			if char == tc {
				traditionalCount++
				break
			}
		}
	}

	// If more than 5% of characters are Traditional-specific, assume it's Traditional
	threshold := len(runeText) / 20
	if threshold < 1 {
		threshold = 1
	}

	return traditionalCount >= threshold
}

// ConvertIfSimplified converts to Traditional only if the text appears to be Simplified
func (c *ChineseConverter) ConvertIfSimplified(text string) (string, error) {
	if text == "" {
		return "", nil
	}

	// If already Traditional, return as-is
	if c.IsTraditional(text) {
		return text, nil
	}

	return c.ToTraditional(text)
}

// globalConverter is a shared converter instance
var globalConverter *ChineseConverter
var globalConverterOnce sync.Once

// GetGlobalConverter returns the global Chinese converter instance
func GetGlobalConverter() *ChineseConverter {
	globalConverterOnce.Do(func() {
		globalConverter = NewChineseConverter(nil)
	})
	return globalConverter
}

// ToTraditional is a convenience function using the global converter
func ToTraditional(simplified string) (string, error) {
	return GetGlobalConverter().ToTraditional(simplified)
}

// ConvertIfSimplified is a convenience function using the global converter
func ConvertIfSimplified(text string) (string, error) {
	return GetGlobalConverter().ConvertIfSimplified(text)
}
