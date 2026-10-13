# Disc：批次選片清單說明「已聽寫的部分不重算」（c2b：畫面）

Status: done

**Source:** `disc-2026-09-generation-resume-c2-consent-list-deduction` 拆出的畫面那半。2026-10-13 Alexyu 從 backlog 選 1。

**設計稿**：2026-10-13 Alexyu 在 Pencil 跑完 Sally 的提示詞（`disc-2026-10-consent-list-resume-note-pen-prompt.md`）。F15-D-v2 列 `KxRtV`，註記 `VCmYz`。

## 設計裁定（Sally，2026-10-13）

- 一部片的價錢比同季其他集低很多，沒有說明會像算錯。所以有扣的列要說明原因。
- **放在那一列的第二行（路線說明那行）最後面**，同樣的字級和顏色（12、`$text-muted`），不加徽章、不加顏色：
  「無文字字幕軌 → 語音辨識 + 翻譯 · 1 小時 38 分 · 已聽寫 40 分鐘，不重算」
- 分鐘數用跟片長同一個格式（`formatRuntime`：不滿一小時「40 分鐘」，超過「1 小時 5 分」）。比 backlog 原句「已做的部分不重算」多給一個數字：說「扣了多少」比只說「有扣」誠實。
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

- [x] T0 設計稿
- [x] T1 型別＋`routeSubtitle`（AC #1、#2）
- [x] T2 測試、gallery 樣本、visual（AC #3）

### Time-dependent visual coverage

N/A。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5），Amelia dev-story，2026-10-13。

### Completion Notes List

- **Sally review**：提示詞的文字、字級、顏色都對；沒有新的溢出（clipping 數 162 不變；`全面啟動` 那列金額被裁切是原本就有的）。
  - 提示詞把分鐘寫成「40 分」，但片長格式不滿一小時是「40 分鐘」。Sally 用 MCP 改了那一句和註記，用選單存檔，另一個 commit。
  - 改的那一列在 `f15-d-v2.png` 的可視範圍外（影集群組在畫面下面），重出的 PNG 只有雜訊，所以沒更新截圖。
- `routeSubtitle` 改成把路線、片長、「已聽寫 N，不重算」三段用「 · 」接起來，空的跳過。新的 `resumedLabel` 只對 `route === 'asr'` 且四捨五入後大於 0 的列回句子。
- `GenerationCandidate` 加 `resumedAsrMinutes?`。
- gallery 新樣本 `generation-consent/resumed`：用搜尋「怪奇」打開影集群組，S01E02 有 20 分鐘已聽寫、價錢比 S01E01 低。
- 檢查：
  - vitest：subtitle＋services 39 檔、1,120 個測試全綠。
  - `tsc`、`pnpm run lint:all`（0 errors）、`format:check`、`check-design-refs` 綠。
  - 本機 visual 綠，新增一張 `-darwin` 基準。
  - `tests/e2e` 沒有用到 `consent-row-subtitle`。

### File List

- `ux-design.pen`、`_bmad-output/pen-node-ids.json`、`_bmad-output/pen-tokens.json`
- `apps/web/src/services/subtitleService.ts`
- `apps/web/src/components/subtitle/consent/CandidateListPanel.tsx`、`.spec.tsx`
- `apps/web/src/routes/test/-gallery.fixtures.tsx`
- `tests/visual/components.visual.spec.ts-snapshots/components/generation-consent/resumed/`
- `_bmad-output/implementation-artifacts/disc-2026-10-consent-list-resume-note.md`、`sprint-status.yaml`
