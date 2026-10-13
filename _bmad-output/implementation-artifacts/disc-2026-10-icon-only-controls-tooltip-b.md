# Disc：只有圖示的按鈕加提示（b：列動作與設定）

Status: done

**Source:** 同 a 單。

## 設計裁定（沿用 a 單，Sally 2026-10-13）

- 只有圖示的**按鈕**：包 `components/ui/Tooltip`，滑鼠移過、鍵盤切到時出現（預設延遲）。提示是短短的動作名稱（「上一頁」「收合篩選」），`aria-label` 照舊（可以比提示多帶片名等上下文）。
- **只在手機上才是純圖示的按鈕不加**：手機沒有滑鼠移過，點下去就是執行動作（a 單裁定），提示永遠看不到。這幾處 `aria-label` 已經夠螢幕閱讀器用：媒體庫手機篩選（LibraryBrowseV2:599）、下載手機排序（DownloadsBrowseV2:409）、手機頂欄搜尋／返回（AppShellV2:96/122）、批次動作列（SelectionToolbar，桌機有字、手機才只剩圖示）。
- 提示框的位置：頁面上方的按鈕 `side="bottom"`，列裡的動作 `side="top"`，左右箭頭朝外（左箭頭 `left`、右箭頭 `right`）。
- 不畫新設計稿（提示框是現成元件，規則已寫在 DESIGN.md §Tooltips）。

## Acceptance Criteria

1. 以下按鈕包 `Tooltip`（提示＝動作名稱，不含片名）：
   - `downloads/DownloadRowActions.tsx` 暫停／繼續／重試、更多動作（桌機的選單版）。手機用的「更多動作」（`useSheet`）與 `downloads/DownloadDetailSheet.tsx` 只在手機出現，照裁定不加。
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

- [x] T1–T3 照上面分檔
- [x] T4 測試與檢查

### Time-dependent visual coverage

N/A。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5），Amelia dev-story，2026-10-13。

### Completion Notes List

- 10 個檔案、16 顆按鈕包 `Tooltip`。提示是動作名稱（「暫停」「更多動作」「上移」「更多操作」「移除路徑」「新增路徑」「移除此媒體庫」「更多選項」「複製檔案路徑」「下載」「顯示／隱藏密碼」「最小化」「取消掃描」），`aria-label` 照舊帶片名或路徑。
- 兩個下拉選單（下載列「更多動作」、媒體庫卡片 ⋮）把 `Tooltip` 包在 `DropdownMenu.Trigger` 外面。
- 只在手機出現的「更多動作」（下載卡片的 sheet 版、下載詳情 sheet）不加。
- 掃描卡的 ✕ 意思是「取消掃描」，提示寫清楚；名字維持「取消」。
- **visual 有一張真的變了，是故意的**：`settings-media-library-manager/focus`。這個樣本的「focus」狀態是用鍵盤切到第一個能按的東西，也就是媒體庫卡片的 ⋮，現在會出現「更多操作」提示框——正是這張要的行為。刪掉舊的 `-darwin` 重出、`git rm` 舊的 `-linux`，`-linux` 交給 CI 補。
- 測試：9 個 spec 各加「鍵盤切到會出現提示」（共用 a 單的 `src/test-utils/tooltip.ts`）。
- 檢查：vitest 全部 4,463 個測試、`tsc`、`pnpm run lint:all`（0 errors）、`format:check`、本機 visual 全綠。

### File List

- `apps/web/src/components/downloads/DownloadRowActions.tsx`、`settings/ExploreBlocksSettings.tsx`、`settings/LibraryCard.tsx`、`settings/LibraryEditModal.tsx`、`setup/MediaLibrarySetupStep.tsx`、`media/PosterCard.tsx`、`media/LocalDetailV2.tsx`、`subtitle/ManageSubtitleDialogV2.tsx`、`auth/LoginForm.tsx`、`scanner/ScanProgressCard.tsx`（含各自的 spec）
- `tests/visual/components.visual.spec.ts-snapshots/components/settings-media-library-manager/focus-visual-darwin.png`（重出）、`focus-visual-linux.png`（刪，CI 補）
- `_bmad-output/implementation-artifacts/disc-2026-10-icon-only-controls-tooltip-b.md`、`sprint-status.yaml`
