# Story 7.9: 鎖「逐 cue 對應、不得合併拆分」—— 時間位移歸零（prompt + gate，後端）

Status: review

<!-- dev-story 2026-10-01（Amelia）。行號為 main `c4913371`。原稿兩處與現實不符：① `SubtitleTranslatorPromptVersion` 已是 `m1-v3`（sub-7-4 先用掉了）→ 本單 bump 到 **`m1-v4`**；② sub-7-8 已落地，AC #4 的回歸評測改用 `cmd/grade`，但本機無 key，真跑交 Alexyu（指令在 Completion Notes）。 -->

## Story

As a viewer,
I want each translated line to sit on the cue it came from,
so that「我是拳擊手」never lands on the cue where the next line says「在擂臺上殺死了一個人」.

## Context

eval-1 全檔評分：484 個 0 分裡 **120 個是時間位移**（兩模型皆有、Haiku 較多）；抽樣版 19 個 0 分「以 cue 時間位移為主」。根因：批次 10 句被當一段重排（`SubtitleTranslatorBatchSize=10`），模型把內容搬到相鄰 cue。同時 eval-1 已知槓桿 4：規則 3「人名保留英文」與 `===TERMS===`「回報你決定的中文譯名」語意拉扯。

## Acceptance Criteria

1. **prompt 版本 `m1-v3`。** 規則明寫：每個 `[N]` 的輸出只能翻譯該 `[N]` 的內容；跨 cue 的句子在原文哪一行斷，譯文就在哪一行斷；不得把上一行的補語搬到下一行；輸出 cue 數必須等於輸入 cue 數。範例 2 組（正／反）。並**解掉規則 3 與 TERMS 的矛盾**：規則 3 改為「人名依 glossary；glossary 沒有的人名首次出現時給中文譯名並在 `===TERMS===` 回報」。`SubtitleTranslatorPromptVersion` bump → cache 語意自然分版。
2. **gate 加位移檢查。** `checkChunk`（`quality_gate.go:57`）新 reason `misaligned`：對每個 cue，取原文的「錨點 token」（數字、英文專有名詞、glossary 命中詞）；若譯文缺該錨點而**相鄰** cue 的譯文多出它 → 判 misaligned，走既有 semantic retry（`maxQualityRetries`）。誤判率要低：只在錨點明確時觸發，測試含反例。
3. **上下文窗說明。** 不改 `SubtitleTranslatorContextWindow=5`（eval-1 已知槓桿 1 另案）；但 context blocks 在 prompt 中標明「僅供參考，不要翻譯」已存在——確認並加測試。
4. **回歸評測。** 用 sub-7-8 黃金樣本（含 40 句陷阱）跑 `m1-v2` vs `m1-v3` 各一次（Sonnet），misaligned 類 0 分數必須下降、其餘類不得上升；結果記 Completion Notes。若 7-8 未落地，用 `eval/zeros-full.csv` 的 120 個位移例（含前後文的 idx 已在 CSV）挑 30 句手動對照。
5. **測試。** prompt 快照；gate misaligned 正／反例（含數字錨點、人名錨點、無錨點不觸發）；retry 迴圈與 stubborn 語意不變。

## Tasks / Subtasks

- [x] **Task 1 — prompt `m1-v3` + 規則 3 修辭（AC: #1, #3）**
- [x] **Task 2 — gate `misaligned`（AC: #2）**
- [~] **Task 3 — 回歸評測（AC: #4）**（規則層代理指標已跑：黃金樣本三組 time_shift 陷阱對全部由新 gate 偵測；真跑 Sonnet 前後版需 key，交 Alexyu，見 Completion Notes）
- [x] **Task 4 — 測試（AC: #5）**

## Dev Notes

- `GateReason*` 是 sub-1-3 stamped 詞彙的一部分？確認：若 `PipelineStage`／gate reason 有 `[@contract]` stamp，加值屬 additive，記 ack。
- 與 sub-6-2（transient 失敗 → stubborn）並存：misaligned 是 semantic 失敗，走 quality retry 不走 transport retry。

### Time-dependent visual coverage

- N/A。

### References

- eval-1「T4/T5 v2／v3」0 分組成；`prompts/subtitle_translator.go:21-71`；`subtitle/quality_gate.go`

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（Amelia）

### Completion Notes List

- **Task 1 prompt `m1-v4`**（原稿寫 m1-v3，但 sub-7-4 已用掉；`PromptVersionFor` 隨之變 `m1-v4+zh-tw-lex-1+<level>`，兩條腿的 segment cache 自然分版）。新段「Per-cue alignment — this is NOT optional」：每個 [N] 只翻 [N]、斷句跟原文同處、不得合併拆分、輸出 cue 數＝輸入；一組正例（拳擊手／擂台）＋一組反例（內容搬到上一句）。規則 3 改：「人名依 Glossary；沒有的就依台灣字幕慣例給中文譯名、全程一致、在 ===TERMS=== 回報；品牌／產品名留英文」——與 trailer 段「不要列保留英文的詞」不再矛盾。P11 pin digest 同步更新（`aced9e9…` → `896ab9f…`）；四處測試的版本字串改 m1-v4。
- **Task 2 gate `misaligned`**：`GateReasonMisaligned`（additive）。`AnchorsFor(chunk, glossary, harvested)` 取每句錨點：多位數數字（單一位數不算——「4 wives」→「四個老婆」是好翻譯）、原文提到的 glossary 詞的固定譯法、原文提到的 harvest 詞（含本 chunk 的 ===TERMS===）的譯法；**不**用英文人名當錨點（m1-v4 會把人名翻成中文，英文 token 本來就該消失）。判定：某錨點不在自己譯文、卻出現在 Index±1 的鄰句譯文、且該鄰句原文自己沒有這個錨點 → misaligned。只是漏掉數字（換成「兩點半」）不觸發；鄰句原文也有同錨點不觸發；沒錨點永不觸發。管線：錨點以**整個 chunk** 算、重試時把 `final`（已接受的鄰句）當 neighbours 傳入，單獨重送的 cue 仍比得到它漂去的那一句；走既有 `maxQualityRetries` semantic retry，stubborn 語意不變。`CheckChunk` 簽名與行為 byte-for-byte 不變（有測）；新增 `CheckChunkAnchored`。
- **Task 3 回歸評測**：本機無 key，真跑交 Alexyu。代理指標：黃金樣本 `trap-14～19` 三組 time_shift 陷阱對，用新 gate 的判定邏輯（eval `time_shift` 規則同一形狀）全部可偵測。⏳ 要跑的指令（main 合併後，兩次各約 $0.16）：`git checkout c4913371 && CLAUDE_API_KEY=… go run ./cmd/grade --model claude-sonnet-5 --trace --out /tmp/m1v3.json`、`git checkout main && … --out /tmp/m1v4.json`，比 `rule_fails_by_reason.time_shift`（應下降）、`zero_rate`（不得上升）、其餘 reason（不得上升）；結果回填本節。
- **Task 4 測試**：gate 7 組（對齊／往前漂／往後漂／只漏不漂／鄰句原文共享錨點／無錨點／先到的類別優先）＋重試時 neighbours 路徑＋`CheckChunk` 不變；`AnchorsFor` 規則（多位數、單位數不算、glossary、whole-word、空譯法略過）；pipeline 一條（漂移的 cue 只重送它自己、帶 glossary 錨點、回來後 stubborn=0）；prompt pin。`go test ./...` 全綠、vet 綠、staticcheck 新套件乾淨。
- 🔗 AC Drift：AC #1 版本號 m1-v3 → m1-v4（已被占用）；AC #2「英文專有名詞」錨點改為 glossary／harvest 譯法（英文 token 在 m1-v4 下本來就會消失，用它會大量誤判）。🔗 AC #3：context 段落「do NOT translate, for reference only」既有（`BuildSubtitleTranslatorPromptWithGlossary`），pin 測試已含該段；未另加測試。

### Discovery Triage

- 無新單。

### File List

- apps/api/internal/ai/prompts/subtitle_translator.go、subtitle_translator_test.go、lexicon_test.go
- apps/api/internal/subtitle/quality_gate.go、quality_gate_test.go、pipeline.go、pipeline_test.go、lexicon_pipeline_test.go
- apps/api/internal/services/transcription_translation_test.go
- _bmad-output/implementation-artifacts/sub-7-9-per-cue-alignment-lock.md、sub-7-8c-local-preview-run.md、sub-7-8-model-ratings-feed.md、sprint-status.yaml


## Change Log

| Date       | Change                                   |
| ---------- | ---------------------------------------- |
| 2026-09-04 | create-story。 |
| 2026-10-01 | dev-story（Amelia）→ review；AC #4 真跑待 Alexyu。 |
