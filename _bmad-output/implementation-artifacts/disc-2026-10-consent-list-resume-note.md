# Disc：批次選片清單說明「已聽寫的部分不重算」（c2b：畫面）

Status: blocked — 等設計稿

**Source:** `disc-2026-09-generation-resume-c2-consent-list-deduction` 拆出的畫面那半。2026-10-13 Alexyu 從 backlog 選 1。

**擋住的原因**：要等 Alexyu 在 Pencil 跑 Sally 的提示詞（`disc-2026-10-consent-list-resume-note-pen-prompt.md`）。

## 設計裁定（Sally，2026-10-13）

- 一部片的價錢比同季其他集低很多，沒有說明會像算錯。所以有扣的列要說明原因。
- **放在那一列的第二行（路線說明那行）最後面**，同樣的字級和顏色（12、`$text-muted`），不加徽章、不加顏色：
  「無文字字幕軌 → 語音辨識 + 翻譯 · 1 小時 38 分 · 已聽寫 40 分，不重算」
- 分鐘數用跟片長同一個格式（`runtimeLabel` 那套：「1 小時 5 分」「40 分」）。比 backlog 原句「已做的部分不重算」多給一個數字：說「扣了多少」比只說「有扣」誠實。
- 只有 `resumed_asr_minutes` 大於 0 的列才加。抽取路線不會有。
- 頁尾總計、群組小計不另外說明，數字本來就是扣過的。
- 單部「管理字幕」對話框不在這張範圍（那裡沒有可以比較的旁邊幾列）。

## Acceptance Criteria

1. `GenerationCandidate` 型別加 `resumedAsrMinutes?: number`。
2. `routeSubtitle` 在 `resumedAsrMinutes > 0` 時，最後加「 · 已聽寫 N，不重算」，N 用片長同一個格式。
3. 測試：有扣、沒扣、0、抽取路線。
4. 手機版那行本來就會換行，不另外處理。
5. 拿掉或改名 data-testid 前先 grep `tests/e2e`。

## Tasks / Subtasks

- [ ] T0 等設計稿
- [ ] T1 型別＋`routeSubtitle`（AC #1、#2）
- [ ] T2 測試、gallery 樣本、visual（AC #3）

### Time-dependent visual coverage

N/A。

## Dev Agent Record
