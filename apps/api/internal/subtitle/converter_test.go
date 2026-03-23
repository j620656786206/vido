package subtitle

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConverter(t *testing.T) {
	c, err := NewConverter()
	require.NoError(t, err)
	assert.True(t, c.IsAvailable())
}

func TestConverter_IsAvailable_Degraded(t *testing.T) {
	c := &Converter{available: false}
	assert.False(t, c.IsAvailable())

	// Should return original content on convert
	input := []byte("测试内容")
	result, err := c.ConvertS2TWP(input)
	assert.Error(t, err)
	assert.Equal(t, input, result, "degraded mode should return original content")
}

// Task 5.2: Basic s2twp conversion (character-level)
func TestConverter_BasicConversion(t *testing.T) {
	c, err := NewConverter()
	require.NoError(t, err)

	input := []byte("简体字测试")
	result, err := c.ConvertS2TWP(input)
	require.NoError(t, err)

	output := string(result)
	assert.Contains(t, output, "簡體字")
	assert.Contains(t, output, "測試")
	assert.NotEqual(t, string(input), output, "output should differ from simplified input")
}

// Task 5.3: Taiwan phrase substitution (AC: 2)
func TestConverter_TaiwanPhraseSubstitution(t *testing.T) {
	c, err := NewConverter()
	require.NoError(t, err)

	tests := []struct {
		simplified  string
		traditional string
	}{
		{"软件", "軟體"},
		{"内存", "記憶體"},
		{"网络", "網路"},
		{"信息", "資訊"},
		{"程序", "程式"},
		{"打印机", "印表機"},
		{"硬盘", "硬碟"},
	}

	for _, tt := range tests {
		t.Run(tt.simplified+"→"+tt.traditional, func(t *testing.T) {
			result, err := c.ConvertS2TWP([]byte(tt.simplified))
			require.NoError(t, err)
			assert.Equal(t, tt.traditional, string(result),
				"phrase substitution: %s → %s", tt.simplified, tt.traditional)
		})
	}
}

// Task 5.4: Idempotent conversion (AC: 3)
func TestConverter_Idempotent_TraditionalInput(t *testing.T) {
	c, err := NewConverter()
	require.NoError(t, err)

	traditional := "這是繁體中文測試，軟體程式記憶體"
	result, err := c.ConvertS2TWP([]byte(traditional))
	require.NoError(t, err)
	assert.Equal(t, traditional, string(result), "traditional input should pass through unchanged")
}

// Task 5.5: Mixed content (AC: 4)
func TestConverter_MixedContent(t *testing.T) {
	c, err := NewConverter()
	require.NoError(t, err)

	input := "Hello 世界 123 测试 test@example.com"
	result, err := c.ConvertS2TWP([]byte(input))
	require.NoError(t, err)

	output := string(result)
	assert.Contains(t, output, "Hello")
	assert.Contains(t, output, "123")
	assert.Contains(t, output, "test@example.com")
	assert.Contains(t, output, "測試") // Chinese converted
}

// Task 5.6 + Task 4: SRT format preservation (AC: 4)
func TestConverter_SRTFormatPreservation(t *testing.T) {
	c, err := NewConverter()
	require.NoError(t, err)

	input := `1
00:01:23,456 --> 00:01:25,789
这是一个测试
这里有<i>斜体</i>和<b>粗体</b>

2
00:01:26,000 --> 00:01:28,000
软件程序信息

`

	result, err := c.ConvertS2TWP([]byte(input))
	require.NoError(t, err)

	output := string(result)

	// Task 4.1: Timing codes preserved
	assert.Contains(t, output, "00:01:23,456 --> 00:01:25,789")
	assert.Contains(t, output, "00:01:26,000 --> 00:01:28,000")

	// Task 4.2: Sequence numbers preserved
	assert.True(t, strings.HasPrefix(output, "1\n"), "first sequence number preserved")
	assert.Contains(t, output, "\n2\n")

	// Task 4.3: HTML tags preserved
	assert.Contains(t, output, "<i>")
	assert.Contains(t, output, "</i>")
	assert.Contains(t, output, "<b>")
	assert.Contains(t, output, "</b>")

	// Task 4.4: Line breaks preserved
	assert.Contains(t, output, "\n")

	// Chinese text converted
	assert.Contains(t, output, "測試")
	assert.Contains(t, output, "軟體")
	assert.Contains(t, output, "程式")
	// Note: OpenCC s2twp maps 信息 → 資訊 (not 訊息)
	assert.Contains(t, output, "資訊")
}

// Task 5.7: Graceful degradation (AC: 5)
func TestConverter_GracefulDegradation(t *testing.T) {
	// Create a converter that's not available
	c := &Converter{available: false}

	input := []byte("原始内容 original content")
	result, err := c.ConvertS2TWP(input)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not available")
	assert.Equal(t, input, result, "should return original content on failure")
}

// Task 5.8: IsAvailable
func TestConverter_IsAvailable(t *testing.T) {
	c, err := NewConverter()
	require.NoError(t, err)
	assert.True(t, c.IsAvailable())

	degraded := &Converter{available: false}
	assert.False(t, degraded.IsAvailable())
}

// Task 5.9: NeedsConversion
func TestNeedsConversion(t *testing.T) {
	tests := []struct {
		language string
		expected bool
	}{
		{"zh-Hans", true},
		{"zh-Hant", false},
		{"zh", false},
		{"und", false},
		{"en", false},
		{"ja", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.language, func(t *testing.T) {
			assert.Equal(t, tt.expected, NeedsConversion(tt.language))
		})
	}
}

// Task 5.10: Empty input
func TestConverter_EmptyInput(t *testing.T) {
	c, err := NewConverter()
	require.NoError(t, err)

	result, err := c.ConvertS2TWP([]byte{})
	assert.NoError(t, err)
	assert.Empty(t, result)

	result2, err := c.ConvertS2TWP(nil)
	assert.NoError(t, err)
	assert.Nil(t, result2)
}

// Task 5.11: BOM handling
func TestConverter_BOMHandling(t *testing.T) {
	c, err := NewConverter()
	require.NoError(t, err)

	// UTF-8 BOM + simplified content
	bom := []byte{0xEF, 0xBB, 0xBF}
	input := append(bom, []byte("软件测试")...)

	result, err := c.ConvertS2TWP(input)
	require.NoError(t, err)

	// Should preserve BOM
	assert.True(t, len(result) >= 3 && result[0] == 0xEF && result[1] == 0xBB && result[2] == 0xBF,
		"BOM should be preserved")

	// Content should be converted
	output := string(result[3:])
	assert.Contains(t, output, "軟體")
	assert.Contains(t, output, "測試")
}

// Task 5.6 extra: Windows-style line endings
func TestConverter_WindowsLineEndings(t *testing.T) {
	c, err := NewConverter()
	require.NoError(t, err)

	input := "第一行\r\n第二行测试\r\n第三行"
	result, err := c.ConvertS2TWP([]byte(input))
	require.NoError(t, err)

	output := string(result)
	assert.Contains(t, output, "測試")
	// Line endings should be preserved (OpenCC passes through non-CJK)
	assert.Contains(t, output, "\r\n")
}

// Task 6: Benchmark
func BenchmarkConvertS2TWP(b *testing.B) {
	c, err := NewConverter()
	if err != nil {
		b.Fatal(err)
	}

	// Create ~100KB simplified content
	var sb strings.Builder
	sample := "这是一个测试字幕文件，包含了软件程序信息网络内存等常见词汇。\n"
	for sb.Len() < 100*1024 {
		sb.WriteString(sample)
	}
	content := []byte(sb.String())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = c.ConvertS2TWP(content)
	}
}
