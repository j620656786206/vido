// Package prompts provides prompt templates for AI-powered text processing.
package prompts

import "fmt"

// TerminologyCorrectorSystemPrompt instructs Claude to fix cross-strait Chinese
// terminology differences while preserving proper nouns and subtitle formatting.
const TerminologyCorrectorSystemPrompt = `You are a Traditional Chinese (Taiwan) subtitle proofreader.
Your ONLY job is to fix cross-strait terminology differences in text that has already been converted from Simplified to Traditional Chinese by OpenCC.

## What to fix (examples):
- 軟件 → 軟體
- 內存 → 記憶體
- 數據 → 資料
- 視頻 → 影片
- 網絡 → 網路
- 信息 → 資訊
- 硬盤 → 硬碟
- 程序 → 程式
- 默認 → 預設
- 文件夾 → 資料夾
- 操作系統 → 作業系統
- 寬帶 → 寬頻
- 光標 → 游標
- 鏈接 → 連結
- 上傳 → 上傳 (correct, no change)
- 下載 → 下載 (correct, no change)

## What NOT to change:
- Proper nouns: person names (人名), place names (地名), movie/show titles (作品名)
- Already correct Taiwan Traditional Chinese terms
- Punctuation and formatting (timestamps, SRT structure, blank lines)
- Non-Chinese text (English, Japanese, etc.)
- Grammar or sentence structure — only vocabulary substitution

## Rules:
1. ONLY substitute mainland China terms with Taiwan equivalents
2. Do NOT rewrite sentences or change meaning
3. Preserve ALL whitespace, newlines, and subtitle block structure exactly
4. If no corrections are needed, return the text unchanged
5. Return ONLY the corrected text, no explanations or annotations`

// BuildTerminologyCorrectorPrompt generates the user prompt for terminology correction.
func BuildTerminologyCorrectorPrompt(subtitleContent string) string {
	return fmt.Sprintf("Correct the cross-strait terminology in the following subtitle text:\n\n%s", subtitleContent)
}

// TerminologyTestPair represents a known cross-strait term mapping for testing.
type TerminologyTestPair struct {
	Input    string // Mainland China term (post-OpenCC)
	Expected string // Taiwan equivalent
	Category string // Category of the term
}

// KnownTerminologyPairs contains test cases for validating terminology correction.
var KnownTerminologyPairs = []TerminologyTestPair{
	{Input: "這個軟件很好用", Expected: "這個軟體很好用", Category: "tech"},
	{Input: "內存不足，請關閉程序", Expected: "記憶體不足，請關閉程式", Category: "tech"},
	{Input: "請檢查數據是否正確", Expected: "請檢查資料是否正確", Category: "tech"},
	{Input: "這個視頻的畫質很好", Expected: "這個影片的畫質很好", Category: "media"},
	{Input: "網絡連接超時", Expected: "網路連線逾時", Category: "tech"},
	{Input: "收到一條新信息", Expected: "收到一條新資訊", Category: "general"},
	{Input: "硬盤空間不足", Expected: "硬碟空間不足", Category: "tech"},
	{Input: "默認設置已恢復", Expected: "預設設定已恢復", Category: "tech"},
	{Input: "文件夾裡有三個文件", Expected: "資料夾裡有三個檔案", Category: "tech"},
	{Input: "操作系統需要更新", Expected: "作業系統需要更新", Category: "tech"},
	// Proper noun preservation — these should NOT be changed
	{Input: "周杰倫的新專輯", Expected: "周杰倫的新專輯", Category: "proper-noun"},
	{Input: "我們去北京旅遊", Expected: "我們去北京旅遊", Category: "proper-noun"},
}
