# Story 7.9: 鎖「逐 cue 對應、不得合併拆分」—— 時間位移歸零（prompt + gate，後端）

Status: ready-for-dev

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

- [ ] **Task 1 — prompt `m1-v3` + 規則 3 修辭（AC: #1, #3）**
- [ ] **Task 2 — gate `misaligned`（AC: #2）**
- [ ] **Task 3 — 回歸評測（AC: #4）**
- [ ] **Task 4 — 測試（AC: #5）**

## Dev Notes

- `GateReason*` 是 sub-1-3 stamped 詞彙的一部分？確認：若 `PipelineStage`／gate reason 有 `[@contract]` stamp，加值屬 additive，記 ack。
- 與 sub-6-2（transient 失敗 → stubborn）並存：misaligned 是 semantic 失敗，走 quality retry 不走 transport retry。

### Time-dependent visual coverage

- N/A。

### References

- eval-1「T4/T5 v2／v3」0 分組成；`prompts/subtitle_translator.go:21-71`；`subtitle/quality_gate.go`

## Dev Agent Record

### Agent Model Used

### Completion Notes List

### Discovery Triage

- （dev 填）

### File List
