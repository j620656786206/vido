# Disc：只有圖示的按鈕加提示（b：列動作與設定）

Status: ready-for-dev

**Source:** 同 a 單。

## 設計裁定（沿用 a 單，Sally 2026-10-13）

- 只有圖示的**按鈕**：包 `components/ui/Tooltip`，滑鼠移過、鍵盤切到時出現（預設延遲）。提示是短短的動作名稱（「上一頁」「收合篩選」），`aria-label` 照舊（可以比提示多帶片名等上下文）。
- **只在手機上才是純圖示的按鈕不加**：手機沒有滑鼠移過，點下去就是執行動作（a 單裁定），提示永遠看不到。這幾處 `aria-label` 已經夠螢幕閱讀器用：媒體庫手機篩選（LibraryBrowseV2:599）、下載手機排序（DownloadsBrowseV2:409）、手機頂欄搜尋／返回（AppShellV2:96/122）、批次動作列（SelectionToolbar，桌機有字、手機才只剩圖示）。
- 提示框的位置：頁面上方的按鈕 `side="bottom"`，列裡的動作 `side="top"`，左右箭頭朝外（左箭頭 `left`、右箭頭 `right`）。
- 不畫新設計稿（提示框是現成元件，規則已寫在 DESIGN.md §Tooltips）。

## Acceptance Criteria

1. 以下按鈕包 `Tooltip`（提示＝動作名稱，不含片名）：
   - `downloads/DownloadRowActions.tsx` 暫停／繼續／重試、更多動作；`downloads/DownloadDetailSheet.tsx` 更多動作。
   - `settings/ExploreBlocksSettings.tsx` 上移／下移／編輯／刪除。
   - `settings/LibraryCard.tsx` 更多選項；`settings/LibraryEditModal.tsx` 移除路徑、新增路徑。
   - `setup/MediaLibrarySetupStep.tsx` 移除此媒體庫。
   - `media/PosterCard.tsx` 更多選項。
   - `media/LocalDetailV2.tsx` 複製檔案路徑（桌機只有圖示）。
   - `subtitle/ManageSubtitleDialogV2.tsx` 線上字幕的下載。
   - `auth/LoginForm.tsx` 顯示／隱藏密碼。
   - `scanner/ScanProgressCard.tsx` 最小化、取消掃描。
2. 測試、檢查同 a 單。

## Tasks / Subtasks

- [ ] T1–T3 照上面分檔
- [ ] T4 測試與檢查

### Time-dependent visual coverage

N/A。

## Dev Agent Record
