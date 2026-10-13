# Disc：只有圖示的按鈕加提示（c：✕）

Status: done

**Source:** 同 a 單。

## 設計裁定（沿用 a 單，Sally 2026-10-13）

- 只有圖示的**按鈕**：包 `components/ui/Tooltip`，滑鼠移過、鍵盤切到時出現（預設延遲）。提示是短短的動作名稱（「上一頁」「收合篩選」），`aria-label` 照舊（可以比提示多帶片名等上下文）。
- **只在手機上才是純圖示的按鈕不加**：手機沒有滑鼠移過，點下去就是執行動作（a 單裁定），提示永遠看不到。這幾處 `aria-label` 已經夠螢幕閱讀器用：媒體庫手機篩選（LibraryBrowseV2:599）、下載手機排序（DownloadsBrowseV2:409）、手機頂欄搜尋／返回（AppShellV2:96/122）、批次動作列（SelectionToolbar，桌機有字、手機才只剩圖示）。
- 提示框的位置：頁面上方的按鈕 `side="bottom"`，列裡的動作 `side="top"`，左右箭頭朝外（左箭頭 `left`、右箭頭 `right`）。
- 不畫新設計稿（提示框是現成元件，規則已寫在 DESIGN.md §Tooltips）。

## Acceptance Criteria

1. 對話框自己畫的 ✕（5 個）包 `Tooltip`，名字與提示都改成「關閉視窗」（跟 `ui/Dialog` 一致）：`LibraryEditModal:174`、`SeasonEpisodeTreeDialog:250`、`MetadataEditorDialog:235`、`ExploreBlockEditModal:131`、`SavePresetDialog:104`。掃描完成卡的 ✕（`ScanProgressCard:196`）不是視窗，提示寫「關閉」。手機掃描 sheet 的 ✕（`ScanProgressSheet:114`）只在手機出現，照裁定不加。測試同步改。
2. 欄位與標籤裡的清除 ✕ 包 `Tooltip`：`InstantSearchBar:179`、`SearchBar:76`、`LogFilters:104`、`CandidateListPanel:926`（清除搜尋）、`FilterChips:69-139`、`FilterChipBar:65`、`CastEditor:52`、`GenreSelector:41`（移除…）。
3. 測試、檢查同 a 單。

## Tasks / Subtasks

- [x] T1 7 個 ✕
- [x] T2 清除 ✕
- [x] T3 測試與檢查

### Time-dependent visual coverage

N/A。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5），Amelia dev-story，2026-10-13。

### Completion Notes List

- 對話框 ✕ 5 個：包 `Tooltip`（`side="bottom"`），名字從「關閉」改成「關閉視窗」，圖示補 `aria-hidden`。
  - `LibraryEditModal`、`MetadataEditorDialog` 用 `ui/Dialog` 並把它的 ✕ 藏起來（`closeClassName="hidden"`），自己畫一顆。現在兩顆同名；瀏覽器裡藏起來那顆不在無障礙樹上，但 jsdom 不讀 CSS，所以這兩個 spec 改成挑「沒有 hidden class 的那顆」。
- 掃描完成卡 ✕：提示「關閉」（它是一張卡，不是視窗）。手機掃描 sheet 不加。
- 清除 ✕ 13 顆：搜尋框三個與選片清單搜尋（「清除搜尋」）、媒體庫篩選標籤 6 顆與探索篩選標籤（「移除篩選」）、修改資訊的演員與類型（「移除」）。`aria-label` 照舊帶名字。
- 沒做（裁定）：`SidePanel`、`ManualSearchDialog`、`SubtitleSearchDialog`、`BatchSubtitleDialog` 的 ✕——這四個元件沒有掛在任何頁面上（`disc-2026-10-more-unmounted-components`）。
- 測試：11 個 spec 加「鍵盤切到會出現提示」。`InstantSearchBar`、`CandidateListPanel` 的清除 ✕ 沒有另加測試：寫法跟其他清除 ✕ 完全一樣，而且這兩個 spec 的 render 很重。
- 檢查：vitest 全部 4,474 個測試、`tsc`、`pnpm run lint:all`（0 errors）、`format:check`、本機 visual 全綠（基準圖沒變）。`tests/e2e` 沒有用名字「關閉」找這些 ✕。

### File List

- ✕：`settings/LibraryEditModal.tsx`、`settings/ExploreBlockEditModal.tsx`、`requests/SeasonEpisodeTreeDialog.tsx`、`search/SavePresetDialog.tsx`、`metadata-editor/MetadataEditorDialog.tsx`、`scanner/ScanProgressCard.tsx`
- 清除 ✕：`search/InstantSearchBar.tsx`、`search/SearchBar.tsx`、`settings/LogFilters.tsx`、`subtitle/consent/CandidateListPanel.tsx`、`library/FilterChips.tsx`、`search/FilterChipBar.tsx`、`metadata-editor/CastEditor.tsx`、`metadata-editor/GenreSelector.tsx`
- 上述檔案的 spec（11 個）
- `_bmad-output/implementation-artifacts/disc-2026-10-icon-only-controls-tooltip-c.md`、`sprint-status.yaml`
