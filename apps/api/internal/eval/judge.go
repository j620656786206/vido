package eval

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/vido/api/internal/ai"
)

// JudgeVersion is the id of the rubric below. Any wording change bumps it so
// two reports can only be compared when they were judged the same way.
const JudgeVersion = "judge-v1"

// DefaultJudgeModel is the fixed judge. It is deliberately NOT the model under
// test's own id: every model is judged by the same reader.
const DefaultJudgeModel = "claude-sonnet-5"

// JudgeBatchSize is how many cues one judge call scores. 20 keeps a batch
// well inside one response while amortising the rubric over the batch.
const JudgeBatchSize = 20

// JudgeNote is the bias disclosure every report and rating carries (AC #3).
const JudgeNote = "裁判為 Claude（" + DefaultJudgeModel + "），評 Claude 模型時有自評偏誤；人評校準落地前，分數宜打折看。"

// JudgeItem is one cue as the judge sees it.
type JudgeItem struct {
	ID        string   `json:"id"`
	Text      string   `json:"en"`
	Refs      []string `json:"refs"`
	Candidate string   `json:"zh"`
}

// JudgeScore is the judge's verdict on one cue.
type JudgeScore struct {
	ID    string `json:"id"`
	Score int    `json:"score"`
	Why   string `json:"why,omitempty"`
}

// ErrJudgeResponse wraps every malformed-reply case so callers can retry once
// and then give up without guessing which parse step failed.
var ErrJudgeResponse = errors.New("eval: judge reply unusable")

// JudgeSystemPrompt is the rubric. The 0/1/2 scale is eval-1's, verbatim in
// spirit: 0 = 看不懂或翻錯（不改不能看）, 1 = 看得懂但生硬／不像台灣話, 2 =
// 自然、不用改. The style bullets are the working rules Taiwanese subtitle
// translators publish (口語、短、不翻譯腔、台灣用語) — principles, not any
// copyrighted text.
func JudgeSystemPrompt() string {
	return strings.TrimSpace(`
你是台灣院線與串流字幕的資深審稿人。你會拿到一批英文字幕句、每句的一到數個「可接受參考譯法」，以及待評的繁體中文譯文。
請只評「待評譯文」，參考譯法只是告訴你意思與可接受的風格；用字不同不扣分。

評分（每句一個整數）：
0 ＝ 看不懂、翻錯意思、漏翻、留英文、出現簡體字、人名或數字錯 —— 不改不能看。
1 ＝ 意思對但生硬、翻譯腔、語氣不對、用語不像台灣人會說的（例：軟件、視頻、質量＝品質）—— 勉強能看。
2 ＝ 自然，像台灣人翻的，不用改 —— 口語、精簡、俚語與雙關有處理、人名一致、語域（正式／口語）貼合原文。

規則：
- 譬喻與俚語直譯而失去原意算 0；直譯但勉強看得懂算 1。
- 字幕以短為美；多餘的「的」「了」堆疊、長句不斷算 1。
- 不要因為譯文比參考譯法更好或更短而扣分。
- 只回 JSON 陣列，不要任何前後文字：[{"id":"…","score":0,"why":"十字以內"}, …]，每一個 id 都要出現一次。`)
}

// JudgeUserPrompt serialises a batch for the judge.
func JudgeUserPrompt(items []JudgeItem) (string, error) {
	if len(items) == 0 {
		return "", errors.New("eval: empty judge batch")
	}
	b, err := json.MarshalIndent(items, "", " ")
	if err != nil {
		return "", err
	}
	return "請評分以下 " + fmt.Sprint(len(items)) + " 句：\n" + string(b), nil
}

// ParseJudgeResponse decodes the judge's reply and checks it covers exactly
// the requested ids with in-range scores. Code fences and prose around the
// array are tolerated; anything else is ErrJudgeResponse.
func ParseJudgeResponse(raw string, want []JudgeItem) (map[string]JudgeScore, error) {
	start := strings.Index(raw, "[")
	end := strings.LastIndex(raw, "]")
	if start < 0 || end < start {
		return nil, fmt.Errorf("%w: no JSON array", ErrJudgeResponse)
	}
	var scores []JudgeScore
	if err := json.Unmarshal([]byte(raw[start:end+1]), &scores); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrJudgeResponse, err)
	}
	out := make(map[string]JudgeScore, len(want))
	for _, s := range scores {
		if s.Score < 0 || s.Score > 2 {
			return nil, fmt.Errorf("%w: score %d out of range for %s", ErrJudgeResponse, s.Score, s.ID)
		}
		out[s.ID] = s
	}
	for _, w := range want {
		if _, ok := out[w.ID]; !ok {
			return nil, fmt.Errorf("%w: missing score for %s", ErrJudgeResponse, w.ID)
		}
	}
	return out, nil
}

// JudgeBatch scores one batch with the given completer (a Claude provider
// pinned to the judge model). It retries a malformed reply once.
func JudgeBatch(ctx context.Context, judge ai.TextCompleter, items []JudgeItem) (map[string]JudgeScore, error) {
	user, err := JudgeUserPrompt(items)
	if err != nil {
		return nil, err
	}
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		raw, err := judge.CompleteText(ctx, JudgeSystemPrompt(), user, 4096)
		if err != nil {
			return nil, err
		}
		scores, perr := ParseJudgeResponse(raw, items)
		if perr == nil {
			return scores, nil
		}
		lastErr = perr
	}
	return nil, lastErr
}
