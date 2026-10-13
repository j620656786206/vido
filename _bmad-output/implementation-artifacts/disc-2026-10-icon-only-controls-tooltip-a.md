# Disc：只有圖示的按鈕加提示（a：導覽與共用元件）

Status: done

**Source:** `disc-2026-10-icon-only-controls-tooltip`（2026-10-13 從 `disc-2026-10-icon-only-native-title-audit` 拆出）。2026-10-13 Alexyu：「做 disc-2026-10-icon-only-controls-tooltip」。約 40 處太大，依「該拆就拆」照畫面區域拆三張：

- **a（這張）**：導覽與共用元件——分頁、篩選欄收合、側邊欄收合、媒體庫檢視切換、首頁輪播左右與 Hero、詳情頁與工作區的返回箭頭。
- **b**：`disc-2026-10-icon-only-controls-tooltip-b`——各頁的列動作與設定（下載、設定首頁區塊、媒體庫卡片、精靈、探索卡片選單、複製路徑、線上字幕下載、密碼顯示、掃描卡）。
- **c**：`disc-2026-10-icon-only-controls-tooltip-c`——自己畫的 ✕ 與欄位、標籤裡的清除 ✕。

## 設計裁定（沿用 a 單，Sally 2026-10-13）

- 只有圖示的**按鈕**：包 `components/ui/Tooltip`，滑鼠移過、鍵盤切到時出現（預設延遲）。提示是短短的動作名稱（「上一頁」「收合篩選」），`aria-label` 照舊（可以比提示多帶片名等上下文）。
- **只在手機上才是純圖示的按鈕不加**：手機沒有滑鼠移過，點下去就是執行動作（a 單裁定），提示永遠看不到。這幾處 `aria-label` 已經夠螢幕閱讀器用：媒體庫手機篩選（LibraryBrowseV2:599）、下載手機排序（DownloadsBrowseV2:409）、手機頂欄搜尋／返回（AppShellV2:96/122）、批次動作列（SelectionToolbar，桌機有字、手機才只剩圖示）。
- 提示框的位置：頁面上方的按鈕 `side="bottom"`，列裡的動作 `side="top"`，左右箭頭朝外（左箭頭 `left`、右箭頭 `right`）。
- 不畫新設計稿（提示框是現成元件，規則已寫在 DESIGN.md §Tooltips）。

## Acceptance Criteria

1. 以下按鈕包 `Tooltip`，提示文字＝下表：
   - `ui/Pagination.tsx` 上一頁／下一頁（數字頁有字，不加）。
   - `ui/FilterRailShell.tsx` 收合篩選。
   - `shell/AppSidebar.tsx` 收合側邊欄（展開那顆已經有）。
   - `library/ViewToggle.tsx` 格狀檢視／列表檢視。
   - `homepage/RecentlyAddedRowV2.tsx`、`homepage/ExploreBlock.tsx` 向左／向右捲動。
   - `homepage/HeroBanner.tsx` 上一部／下一部；下面的圓點提示「第 N 部：片名」（有片名時）或「第 N 部」。
   - `media/DetailHeroV2.tsx`、`media/DetailStatesV2.tsx` 返回媒體庫。（`subtitle/GenerationWorkspaceV2.tsx` 的返回活動是 `sm:hidden`，只在手機出現，照裁定不加。）
2. 圖示加 `aria-hidden`（還沒加的）。`aria-label`、`data-testid`、className 都不變。
3. 測試：每個檔案至少一條「鍵盤切到會出現提示」。
4. 檢查：vitest、`tsc`、`pnpm run lint:all`、`format:check`、visual（基準圖不應該變）；grep `tests/e2e` 確認沒有用到會變的名字。

## Tasks / Subtasks

- [x] T1 共用：Pagination、FilterRailShell、AppSidebar、ViewToggle
- [x] T2 首頁：RecentlyAddedRowV2、ExploreBlock、HeroBanner
- [x] T3 返回箭頭：DetailHeroV2、DetailStatesV2、GenerationWorkspaceV2
- [x] T4 測試與檢查

### Time-dependent visual coverage

N/A。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5），Amelia dev-story，2026-10-13。

### Debug Log References

- 突變測試：把 `Pagination.tsx` 還原成沒有提示框的版本，新測試變紅。還原後全綠。

### Completion Notes List

- 9 個檔案、14 顆按鈕包 `Tooltip`：分頁上／下一頁、收合篩選、收合側邊欄、格狀／列表檢視、首頁兩條輪播的左右、Hero 的上一部／下一部與圓點（「第 2 部：片名」）、詳情頁三個返回箭頭。
- 圖示補上 `aria-hidden`（分頁、檢視切換、輪播、Hero）。`aria-label`、`data-testid`、className 都沒動。
- Hero 圓點在 `map` 裡，`key` 移到外層的 `Tooltip`。
- 生成工作區的「返回活動」只在手機出現（`sm:hidden`），照裁定不加。
- 新的測試小工具 `src/test-utils/tooltip.ts` 的 `expectTooltipOnFocus`：確認沒有 `title`、鍵盤切到後提示框出現在按鈕外面。9 個 spec 各加一條。
- 檢查：
  - vitest 全部 288 檔、4,463 個測試全綠。
  - `tsc`、`pnpm run lint:all`（0 errors）、`format:check` 綠。
  - 本機 visual 綠，基準圖沒有變。
  - `aria-label` 沒變，`tests/e2e` 不受影響。

### File List

- `apps/web/src/components/ui/Pagination.tsx`、`FilterRailShell.tsx`（＋ spec）
- `apps/web/src/components/shell/AppSidebar.tsx`（＋ spec）
- `apps/web/src/components/library/ViewToggle.tsx`（＋ spec）
- `apps/web/src/components/homepage/RecentlyAddedRowV2.tsx`、`ExploreBlock.tsx`、`HeroBanner.tsx`（＋ spec）
- `apps/web/src/components/media/DetailHeroV2.tsx`、`DetailStatesV2.tsx`（＋ spec）
- `apps/web/src/test-utils/tooltip.ts`（新）
- `_bmad-output/implementation-artifacts/disc-2026-10-icon-only-controls-tooltip-{a,b,c}.md`（新）、`sprint-status.yaml`
