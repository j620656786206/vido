# Disc：只有圖示的按鈕加提示（c：✕）

Status: ready-for-dev

**Source:** 同 a 單。

## 設計裁定（沿用 a 單，Sally 2026-10-13）

- 只有圖示的**按鈕**：包 `components/ui/Tooltip`，滑鼠移過、鍵盤切到時出現（預設延遲）。提示是短短的動作名稱（「上一頁」「收合篩選」），`aria-label` 照舊（可以比提示多帶片名等上下文）。
- **只在手機上才是純圖示的按鈕不加**：手機沒有滑鼠移過，點下去就是執行動作（a 單裁定），提示永遠看不到。這幾處 `aria-label` 已經夠螢幕閱讀器用：媒體庫手機篩選（LibraryBrowseV2:599）、下載手機排序（DownloadsBrowseV2:409）、手機頂欄搜尋／返回（AppShellV2:96/122）、批次動作列（SelectionToolbar，桌機有字、手機才只剩圖示）。
- 提示框的位置：頁面上方的按鈕 `side="bottom"`，列裡的動作 `side="top"`，左右箭頭朝外（左箭頭 `left`、右箭頭 `right`）。
- 不畫新設計稿（提示框是現成元件，規則已寫在 DESIGN.md §Tooltips）。

## Acceptance Criteria

1. 自己畫的 ✕（7 個）包 `Tooltip`，提示「關閉視窗」（跟 `ui/Dialog` 一致）：`LibraryEditModal:174`、`SeasonEpisodeTreeDialog:250`、`MetadataEditorDialog:235`、`ExploreBlockEditModal:131`、`SavePresetDialog:104`、`ScanProgressCard:196`、`ScanProgressSheet:114`。名字跟提示一致（「關閉視窗」），測試與 e2e 同步改。
2. 欄位與標籤裡的清除 ✕ 包 `Tooltip`：`InstantSearchBar:179`、`SearchBar:76`、`LogFilters:104`、`CandidateListPanel:926`（清除搜尋）、`FilterChips:69-139`、`FilterChipBar:65`、`CastEditor:52`、`GenreSelector:41`（移除…）。
3. 測試、檢查同 a 單。

## Tasks / Subtasks

- [ ] T1 7 個 ✕
- [ ] T2 清除 ✕
- [ ] T3 測試與檢查

### Time-dependent visual coverage

N/A。

## Dev Agent Record
