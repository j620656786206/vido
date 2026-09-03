# Story 7.4: 內建台灣用語詞庫 + OTT 風格在地化程度（後端為主，設定頁一格）

Status: ready-for-dev

## Story

As a Taiwanese viewer,
I want translations that read like Netflix or Apple TV subtitles — 影片 not 視頻, 全聯 not「超市」when the scene calls for it —
so that the output never feels like a machine transliterating a mainland phrasebook.

## Context

party-mode 2026-09-03（Mary／Alexyu）：這是 A 路線裡**唯一不用等累積**的部分，出貨即生效。兩種性質不同的東西，**不要混成一張表**：

| | (a) 查表型詞庫 | (b) OTT 風格在地化 |
| --- | --- | --- |
| 例 | 視頻→影片、質量→品質、信息→資訊、屏幕→螢幕；Life360（被翻成「360 號公路」）、常見品牌／App 名 | "the grocery store" → 全聯／家樂福／喜互惠；"the convenience store" → 超商／小七 |
| 機制 | 後處理 replace（OpenCC s2twp 之後）＋ prompt glossary 注入 | prompt **風格規則 + 範例**，不是查表 |
| 風險 | 低（可逆、可測） | **口味**——有人討厭美劇裡出現全聯 → 必須是**使用者可調的開關** |

## Acceptance Criteria

1. **(a) 詞庫檔。** `apps/api/internal/ai/prompts/lexicon/zh-tw.yaml`：兩段——`replacements`（簡→台，含 OpenCC s2twp 漏掉的常見詞，先收 ≥ 60 筆，來源註明）與 `terms`（品牌／App／機構 ≥ 40 筆，如 Life360、Venmo、Costco、7-Eleven、CVS、DMV、IRS）。embed 進 binary；版本號進 `PromptVersion` 的組成（改詞庫 = 新版本 → cache 語意誠實）。
2. **(a) 注入與後處理。** `terms` 併入 `BuildGlossarySection` 的**全域**段（在 show 詞彙表之前、show 詞彙表優先覆蓋）；`replacements` 在 OpenCC 之後、quality gate 之前做整詞替換（Unicode 邊界，不碰英文）。`GlossaryVersion` 含詞庫版本。
3. **(b) 在地化程度開關。** 設定 `SUBTITLE_LOCALIZATION_LEVEL`（env + settings 表，預設 `standard`）三檔：`literal`（不在地化，超市就是超市）／`standard`（台灣用語，但不換品牌）／`ott`（Netflix／Apple 風格，可用在地品牌與俚語）。prompt 依檔位插入一段風格規則＋ 3–5 個範例（`ott` 檔附「只在場景是日常生活且原文是泛稱時才替換；專有品牌名不換」的護欄）。檔位進 `PromptVersion`。
4. **設定頁。** 字幕設定區一格 radio（三檔，各一句說明＋一個例子）；Rule 21 header 循 `ApiKeysForm` 先例 ride 設定 shell。
5. **評測掛鉤。** `standard` 與 `ott` 各跑 sub-7-8 的黃金樣本一次（$0.05 內），把差異記在 story Completion Notes；**不得**因為分數改預設檔位，那是 Alexyu 的產品裁定。
6. **測試。** 詞庫 YAML schema 驗證測試（重複鍵、空值）；replace 邊界（「視頻」在「電視頻道」裡**不得**被換——需詞邊界或白名單）；三檔 prompt 快照；設定讀寫；FE radio spec。

## Tasks / Subtasks

- [ ] **Task 1 — 詞庫檔 + 載入 + 版本（AC: #1）**
- [ ] **Task 2 — 注入與後處理（AC: #2）**
- [ ] **Task 3 — 在地化檔位 + prompt 段（AC: #3）**
- [ ] **Task 4 — 設定頁一格（AC: #4）**
- [ ] **Task 5 — 評測與測試（AC: #5, #6）**

（後端 4 task、前端 1 —— 不觸發拆分。）

## Dev Notes

- 「視頻／電視頻道」這類誤換是這個 story 最容易翻車的地方；replace 必須是**詞級**不是子字串級，測試要有反例表。
- 詞庫是**跨片**資產（scope=`global:zh-TW` 的概念），但本 story 先做 embed 檔，不進 `show_glossary` 表——sub-7-1 的 scope 命名空間留了位子，將來要讓使用者編輯再搬進 DB。
- PRD 既有規則：大陸出品內容保留簡體不轉換（`production_countries` 含 CN）→ 詞庫替換也要**跳過**這類內容。

### Time-dependent visual coverage

- N/A — 設定頁 radio 無時鐘。

### References

- party-mode 2026-09-03（喜互惠／全聯 討論）；eval-1 零分例 Life360；`subtitle/converter.go`（s2twp）；`prompts/subtitle_translator.go:21-71`

## Dev Agent Record

### Agent Model Used

### Completion Notes List

### Discovery Triage

- （dev 填）

### File List
