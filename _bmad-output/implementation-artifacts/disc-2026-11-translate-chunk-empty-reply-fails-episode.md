# Story disc-2026-11-translate-chunk-empty-reply-fails-episode：AI 回空白時切半重送，不讓整集失敗

Status: done

**Source:** sprint-status 同名待辦（P1，PR #946 建單）。⚖️ Alexyu 2026-11-01：「先 1，然後修第一件」。

## 問題

2026-11-01 在 NAS 上用 claude-sonnet-5 翻《天蠍行動》S01 的片內英文字幕：

| 集 | 結果 | 實際 | 預估 |
|---|---|---|---|
| E1 | 完成 | $0.44 | $0.19 |
| E2 | 失敗（第 426–476 句） | $0.10 | $0.19 |
| E3 | 失敗（第 1272–1325 句） | $0.34 | $0.19 |

兩次失敗的錯誤一樣：`translate chunk [...]: AI_INVALID_RESPONSE: Cannot parse AI response`。訊息裡**沒有** `AI_RESPONSE_TRUNCATED`，所以是 `ai/claude.go` `CompleteTextWithUsage` 收到「沒有文字、stop_reason 也不是 max_tokens」的回覆。stop_reason 沒記錄下來，最可能是 `refusal`：Claude 的安全分類器中途擋下回覆（《天蠍行動》有駭客、炸彈的台詞），要等下一次失敗的訊息才能證實。

`subtitle/pipeline.go` `translateSplitting` 只在「截斷」時切半重送，其他空白回覆直接讓整集失敗。前面已經翻好、已經付錢的句子全部丟掉。

## Acceptance Criteria

1. [x] Claude 回空白、stop_reason 不是 max_tokens 時，錯誤同時是 `ErrAIInvalidResponse`（舊的判斷照常成立）和新的 `ErrAIEmptyReply`，訊息帶 `stop_reason=…` 和 `output_tokens=…`。OpenAI 相容的供應商回空白（`finish_reason` 不是 `length`）同樣處理，帶 `finish_reason=…`。
2. [x] 片內／片旁英文字幕翻譯（`translateSplitting`）和語音辨識那條（`completeSplitting`）遇到「截斷」或「空白回覆」都切半重送，切到單句還是空白就那一句保留英文，不讓整集失敗。保留英文的句數太多時，原本的上限照樣讓這次執行失敗。
3. [x] 有文字但解析不了的回覆（單純的 `ErrAIInvalidResponse`）照舊不切半、直接失敗（`TestTranslateTrack_OtherInvalidRepliesStillStopTheRun` 不動）。
4. [x] 空白回覆也要付錢：Claude 在回錯誤時一併回傳這次的用量，`TranslateChunk` 把它傳上去，花費照算。
5. [x] 測試：Claude refusal、OpenAI content_filter、`IsShrinkableReply`、兩條翻譯路線切半、單句被擋保留英文、被擋的請求用量有算進去。

## Dev Agent Record

### 做了什麼

- `ai/thinking.go`：新增 `ErrAIEmptyReply` 和 `IsShrinkableReply(err)`（截斷或空白都算）。
- `ai/claude.go` `CompleteTextWithUsage`：空白回覆分兩種，max_tokens → 截斷（照舊）；其他 → `ErrAIInvalidResponse` + `ErrAIEmptyReply` + `(stop_reason=…, output_tokens=…)`。兩種都把 `Usage` 跟錯誤一起回傳。
- `ai/openai_compat.go`：空白回覆且 `finish_reason` 不是 `length` → `ErrAIEmptyReply`，帶 `finish_reason`。
- `services/translation_service.go`：`TranslateChunk` 失敗時回傳 `res.Usage`；`completeSplitting` 改用 `IsShrinkableReply`。
- `subtitle/pipeline.go` `translateSplitting`：改用 `IsShrinkableReply`，log 文字改成「回覆沒有文字」。

### 判斷

- `end_turn` 但沒有文字也算空白回覆、一樣切半：stop_reason 還沒證實是 refusal，只認 refusal 的話萬一猜錯就等於沒修。最壞情況是整段 51 句一路切到單句（約 100 次小請求，系統提示有快取），而保留英文的上限會先讓這次執行失敗，不會無限花錢。
- 只修「整集失敗」。E1 比預估貴 2.3 倍、log 裡大量「回原文」（echoed）是另一件事，沒有動。

### 測試

- `go test ./...` 全過；`go vet ./...`、`gofmt`（改到的檔）、`staticcheck`（ai／subtitle／services）乾淨。
- 新測試：`TestClaudeProvider_RefusalIsAnEmptyReplyWithItsStopReason`、`TestOpenAICompat_ContentFilterWithNoTextIsAnEmptyReply`、`TestIsShrinkableReply_OnlyNoTextReplies`、`TestTranslateTrack_EmptyReplyChunkIsSplitAndOnlyTheTriggerKeepsEnglish`（含用量）、`TestTranslateWithGlossary_EmptyReplyBatchIsSplitInHalf`、`TestTranslateWithGlossary_OtherInvalidReplyIsNotSplit`。

### 正式機驗證（2026-11-01）

- 部署 `4db26ab0` 後 Alexyu 用手機重跑天蠍 E2、E3，兩集都完成：E2 1,502 句、$0.49，E3 1,549 句、$0.44。
- log 證實原因是 `stop_reason=refusal`（`output_tokens=0`）。E2 的 426–476、E3 的 1272–1325 等五段被擋，切半重送後都翻完；切到單句還是被擋的只有 E2 第 1411 句「It'll be aerosolized in...」（生化病毒劇情），保留英文。
- 其他保留英文的 30 句幾乎都是人名（O'Brien!、Happy?、Cabe Gallo），屬於「回原文」，不是這次的問題。
- 每集實際花費是預估的 2.5 倍，另立 `disc-2026-11-translate-estimate-ignores-cue-density`。

## Change Log

- 2026-11-01：建立＋實作（Alexyu 在外面，同一個 session 接著做）。
- 2026-11-01：正式機重跑天蠍 E2／E3 驗證通過，狀態改 done。
