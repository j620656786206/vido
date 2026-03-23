package subtitle

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test fixtures — representative subtitle excerpts.
const (
	// Pure Traditional Chinese (zh-Hant)
	traditionalSample = `1
00:00:01,000 --> 00:00:03,000
這個世界已經變了

2
00:00:04,000 --> 00:00:06,000
從前的時間過得很慢

3
00:00:07,000 --> 00:00:09,000
現在一切都變了

4
00:00:10,000 --> 00:00:12,000
讓我們回到過去

5
00:00:13,000 --> 00:00:15,000
機會來了就應該把握
開門見面說聽問題
經過長時間的運動
兩個人從這邊走遠了`

	// Pure Simplified Chinese (zh-Hans)
	simplifiedSample = `1
00:00:01,000 --> 00:00:03,000
这个世界已经变了

2
00:00:04,000 --> 00:00:06,000
从前的时间过得很慢

3
00:00:07,000 --> 00:00:09,000
现在一切都变了

4
00:00:10,000 --> 00:00:12,000
让我们回到过去

5
00:00:13,000 --> 00:00:15,000
机会来了就应该把握
开门见面说听问题
经过长时间的运动
两个人从这边走远了`

	// English only
	englishSample = `1
00:00:01,000 --> 00:00:03,000
Hello World

2
00:00:04,000 --> 00:00:06,000
This is a test subtitle file`

	// Mixed content (roughly 50/50)
	mixedSample = `1
00:00:01,000 --> 00:00:03,000
這個世界已經變了
这个问题很难回答

2
00:00:04,000 --> 00:00:06,000
從前的時間過得很慢
让我们开始学习`
)

func TestDetect_TraditionalChinese(t *testing.T) {
	result := Detect([]byte(traditionalSample))
	assert.Equal(t, LangTraditional, result.Language)
	assert.Greater(t, result.TraditionalRatio, 0.90)
	assert.Greater(t, result.TraditionalCount, 0)
	assert.Greater(t, result.TotalCJK, 0)
}

func TestDetect_SimplifiedChinese(t *testing.T) {
	result := Detect([]byte(simplifiedSample))
	assert.Equal(t, LangSimplified, result.Language)
	assert.Less(t, result.TraditionalRatio, 0.10)
	assert.Greater(t, result.SimplifiedCount, 0)
}

func TestDetect_MixedContent(t *testing.T) {
	result := Detect([]byte(mixedSample))
	assert.Equal(t, LangAmbiguous, result.Language)
	assert.GreaterOrEqual(t, result.TraditionalRatio, 0.30)
	assert.LessOrEqual(t, result.TraditionalRatio, 0.70)
}

func TestDetect_EnglishOnly(t *testing.T) {
	result := Detect([]byte(englishSample))
	assert.Equal(t, LangUndetermined, result.Language)
	assert.Equal(t, 0, result.TotalCJK)
}

func TestDetect_EmptyContent(t *testing.T) {
	result := Detect([]byte{})
	assert.Equal(t, LangUndetermined, result.Language)
	assert.Equal(t, 0, result.TotalCJK)
}

func TestDetect_BOMHandling(t *testing.T) {
	// UTF-8 BOM + Traditional Chinese content
	bom := []byte{0xEF, 0xBB, 0xBF}
	content := append(bom, []byte(traditionalSample)...)

	result := Detect(content)
	assert.Equal(t, LangTraditional, result.Language)
}

func TestDetect_BoundaryThreshold71(t *testing.T) {
	// Create content with ~71% traditional-unique characters
	// Use 71 traditional + 29 simplified unique characters
	var sb strings.Builder
	trad := []rune{'這', '個', '來', '進', '過', '還', '從', '為', '們', '會',
		'著', '沒', '給', '讓', '動', '關', '開', '長', '問', '時',
		'應', '點', '經', '機', '頭', '現', '實', '說', '種', '見',
		'邊', '後', '嗎', '裡', '聽', '遠', '運', '兩', '幾', '發',
		'無', '書', '東', '馬', '車', '雲', '風', '飛', '鳥', '魚',
		'龍', '門', '電', '號', '樂', '寫', '買', '賣', '紅', '綠',
		'藍', '銀', '鐵', '錢', '鐘', '鋼', '鬧', '聞', '陽', '陰',
		'隊'}
	simp := []rune{'这', '个', '来', '进', '过', '还', '从', '为', '们', '会',
		'着', '没', '给', '让', '动', '关', '开', '长', '问', '时',
		'应', '点', '经', '机', '头', '现', '实', '说', '种', '边'}

	for _, r := range trad[:71] {
		sb.WriteRune(r)
	}
	for _, r := range simp[:29] {
		sb.WriteRune(r)
	}

	result := Detect([]byte(sb.String()))
	// 71 / (71+29) = 0.71 > 0.70 threshold
	assert.Equal(t, LangTraditional, result.Language, "71%% traditional should be zh-Hant")
}

func TestDetect_BoundaryThreshold69(t *testing.T) {
	// 69% traditional → should be ambiguous
	var sb strings.Builder
	trad := []rune{'這', '個', '來', '進', '過', '還', '從', '為', '們', '會',
		'著', '沒', '給', '讓', '動', '關', '開', '長', '問', '時',
		'應', '點', '經', '機', '頭', '現', '實', '說', '種', '見',
		'邊', '後', '嗎', '裡', '聽', '遠', '運', '兩', '幾', '發',
		'無', '書', '東', '馬', '車', '雲', '風', '飛', '鳥', '魚',
		'龍', '門', '電', '號', '樂', '寫', '買', '賣', '紅', '綠',
		'藍', '銀', '鐵', '錢', '鐘', '鋼', '鬧', '聞', '陽'}
	simp := []rune{'这', '个', '来', '进', '过', '还', '从', '为', '们', '会',
		'没', '给', '让', '动', '关', '开', '长', '问', '时', '应',
		'点', '经', '机', '头', '现', '实', '说', '种', '边', '听',
		'远'}

	for _, r := range trad[:69] {
		sb.WriteRune(r)
	}
	for _, r := range simp[:31] {
		sb.WriteRune(r)
	}

	result := Detect([]byte(sb.String()))
	assert.Equal(t, LangAmbiguous, result.Language, "69%% traditional should be zh (ambiguous)")
}

func TestDetect_BoundaryExact70(t *testing.T) {
	// Exactly 70% traditional → should be ambiguous (threshold is >70%, not >=)
	var sb strings.Builder
	trad := []rune{'這', '個', '來', '進', '過', '還', '從', '為', '們', '會',
		'著', '沒', '給', '讓', '動', '關', '開', '長', '問', '時',
		'應', '點', '經', '機', '頭', '現', '實', '說', '種', '見',
		'邊', '後', '嗎', '裡', '聽', '遠', '運', '兩', '幾', '發',
		'無', '書', '東', '馬', '車', '雲', '風', '飛', '鳥', '魚',
		'龍', '門', '電', '號', '樂', '寫', '買', '賣', '紅', '綠',
		'藍', '銀', '鐵', '錢', '鐘', '鋼', '鬧', '聞', '陽', '陰'}
	simp := []rune{'这', '个', '来', '进', '过', '还', '从', '为', '们', '会',
		'没', '给', '让', '动', '关', '开', '长', '问', '时', '应',
		'点', '经', '机', '头', '现', '实', '说', '种', '边', '听'}

	for _, r := range trad[:70] {
		sb.WriteRune(r)
	}
	for _, r := range simp[:30] {
		sb.WriteRune(r)
	}

	result := Detect([]byte(sb.String()))
	// 70/100 = 0.70, NOT > 0.70, so ambiguous
	assert.Equal(t, LangAmbiguous, result.Language, "exactly 70%% traditional should be zh (ambiguous)")
}

func TestDetect_BoundaryExact30(t *testing.T) {
	// Exactly 30% traditional → should be zh-Hans (threshold is ≤30%)
	var sb strings.Builder
	trad := []rune{'這', '個', '來', '進', '過', '還', '從', '為', '們', '會',
		'著', '沒', '給', '讓', '動', '關', '開', '長', '問', '時',
		'應', '點', '經', '機', '頭', '現', '實', '說', '種', '見'}
	simp := []rune{'这', '个', '来', '进', '过', '还', '从', '为', '们', '会',
		'没', '给', '让', '动', '关', '开', '长', '问', '时', '应',
		'点', '经', '机', '头', '现', '实', '说', '种', '边', '听',
		'远', '运', '两', '几', '发', '无', '书', '东', '马', '车',
		'云', '风', '飞', '鸟', '鱼', '龙', '门', '电', '号', '乐',
		'写', '买', '卖', '红', '绿', '蓝', '银', '铁', '钱', '钟',
		'钢', '闹', '闻', '间', '阳', '阴', '队', '际', '陆', '险'}

	for _, r := range trad[:30] {
		sb.WriteRune(r)
	}
	for _, r := range simp[:70] {
		sb.WriteRune(r)
	}

	result := Detect([]byte(sb.String()))
	// 30/100 = 0.30, ≤ 0.30, so zh-Hans
	assert.Equal(t, LangSimplified, result.Language, "exactly 30%% traditional should be zh-Hans")
}

func TestDetect_SharedCharactersOnly(t *testing.T) {
	// Characters common to both simplified and traditional (e.g., 人, 大, 好, 中)
	content := "人大好中國的是不了在有我他這"
	// Note: some of these may be in the sets; use truly shared ones
	sharedContent := "人大好小多少上下左右前中天地日月山水"

	result := Detect([]byte(sharedContent))
	// Should be ambiguous or undetermined since no unique chars
	assert.Contains(t, []string{LangAmbiguous, LangUndetermined}, result.Language)
	_ = content
}

func TestDetect_ContentOnlyNoFilename(t *testing.T) {
	// AC 8: Detection must be content-based only
	// The function signature takes []byte, not a filename — this test
	// verifies the API doesn't accept filename parameters
	result := Detect([]byte(traditionalSample))
	assert.Equal(t, LangTraditional, result.Language)
	// Function signature enforces content-only: Detect(content []byte) DetectionResult
}

func TestDetectFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.srt")
	err := os.WriteFile(path, []byte(traditionalSample), 0644)
	require.NoError(t, err)

	result, err := DetectFromFile(path)
	require.NoError(t, err)
	assert.Equal(t, LangTraditional, result.Language)
}

func TestDetectFromFile_NotFound(t *testing.T) {
	_, err := DetectFromFile("/nonexistent/path.srt")
	assert.Error(t, err)
}

func TestDetect_LargeContent(t *testing.T) {
	// Create content larger than maxDetectionBytes
	var sb strings.Builder
	for sb.Len() < 200*1024 {
		sb.WriteString(traditionalSample)
	}

	result := Detect([]byte(sb.String()))
	assert.Equal(t, LangTraditional, result.Language)
}

func TestLanguageConstants(t *testing.T) {
	assert.Equal(t, "zh-Hant", LangTraditional)
	assert.Equal(t, "zh-Hans", LangSimplified)
	assert.Equal(t, "zh", LangAmbiguous)
	assert.Equal(t, "und", LangUndetermined)
}

func BenchmarkDetect(b *testing.B) {
	// Create a realistic ~100KB subtitle content
	var sb strings.Builder
	for sb.Len() < 100*1024 {
		sb.WriteString(traditionalSample)
	}
	content := []byte(sb.String())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Detect(content)
	}
}
