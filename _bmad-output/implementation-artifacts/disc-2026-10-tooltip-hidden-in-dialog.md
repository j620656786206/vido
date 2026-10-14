# Disc：對話框裡的提示說明被對話框蓋住，看不到

Status: done

**Source:** 2026-10-14 NAS 驗證（正式機 image `67e86ef2`＝#773）。滑鼠移到對話框右上角 ✕、或選片清單裡「資料夾無法寫入」的標籤，提示說明都沒出現。Alexyu：「1修」。

## 查到的事（2026-10-14，main `451889dd`）

- 提示框元件 `apps/web/src/components/ui/Tooltip.tsx:78-84`：`BaseTooltip.Portal` → `BaseTooltip.Positioner`（**沒有 z-index**）→ `BaseTooltip.Popup className="z-[80] …"`。
- Base UI 的 Positioner 用 transform 定位，所以它是一個疊層（stacking context）。Popup 的 `z-[80]` 只在 Positioner 裡面比大小；Positioner 本身在 body 層的 z 是 `auto`。
- 對話框 `apps/web/src/components/ui/Dialog.tsx:31`（遮罩）和 `:73`（內容）都是 `z-50`。所以提示框整個被壓在對話框底下。
- NAS 上用瀏覽器 JS 量過：提示框的 DOM 在、`data-open` 有、Popup `z-index: 80`、Positioner `z-index: auto`；對話框 `z-index: 50`。在提示框中心點 `elementFromPoint` 拿到的是對話框，不是提示框。
- 一般頁面（沒有對話框）沒有比它高的層，所以 #766～#768 的提示框正常。
- 受影響：#770（5 個對話框的 ✕「關閉視窗」）、#771（選片清單「未匹配／資料夾無法寫入／無法估價」的原因）、`ModelPicker.tsx` 等所有放在對話框或 Sheet（`z-[70]/z-[71]`，`Sheet.tsx:75`）裡的提示框。
- 最高的對話框是登出確認（`LogoutButton.tsx:139-140`，`z-[80]`）。
- 為什麼測試沒抓到：`Tooltip.spec.tsx` 在 jsdom 只檢查文字有沒有出現；jsdom 不算疊層。`tests/e2e/episode-subtitle-badge.spec.ts` 用 `toBeVisible`，被蓋住也算 visible。

## Acceptance Criteria

1. 提示框的疊層放在 Positioner 上（Popup 上的拿掉），值比畫面上所有對話框、Sheet 都高：用 `z-[90]`（登出確認是 80）。
2. 一般頁面的提示框行為不變（位置、延遲、手機點開）。
3. 真瀏覽器測試：開一個對話框，滑鼠移到 ✕，提示框出現，**而且**在提示框中心點拿到的元素就是提示框本身（不是被蓋住）。修之前這條要紅。
4. 單元測試：提示框的疊層 class 在 Positioner 上。

## Tasks / Subtasks

- [x] T1 `Tooltip.tsx`：`z-[90]` 移到 Positioner，註解寫明原因
- [x] T2 `Tooltip.spec.tsx`：Positioner 帶疊層 class
- [x] T3 `tests/e2e/tooltip-in-dialog.spec.ts`：對話框 ✕ 的提示框在最上層（用 `elementFromPoint`）；先在修之前跑一次確認會紅

### Time-dependent visual coverage

N/A（提示框要滑鼠移過才出現，visual 基準不拍）。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5），2026-10-14。

### Completion Notes List

- `Tooltip.tsx`：Popup 上的 `z-[80]` 拿掉，Positioner 加 `z-[90]`，註解寫明原因。
- 真瀏覽器測試 `tests/e2e/tooltip-in-dialog.spec.ts` 兩條：
  - 儲存篩選（手刻的 z-50 對話框）的 ✕。
  - 產生字幕（共用的 Radix 對話框，就是 #770／#771 加提示框的那種）的 ✕。
  - 修之前兩條都紅（「the tooltip is painted above the dialog」），修之後 chromium、firefox 都綠。手機專案跳過（沒有滑鼠移過）。
- 一個要注意的地方：Radix 的對話框打開時，會把 `<body>` 設成 `pointer-events:none`，提示框掛在 body 底下也跟著繼承，`elementFromPoint` 會跳過它。所以測試在量的那一下，暫時把提示框和 Positioner 的 pointer-events 打開，問的是「誰畫在最上面」。
- `Tooltip.spec.tsx` 加一條：疊層 class 在 Positioner，不在 Popup。
- 檢查：vitest（ui／shell／consent／search 40 檔 624 個）、`lint:all` 0 errors、`format:check`。

### Senior Developer Review (AI) — 2026-10-14（Fable 5.1 跨模型）

結論：0 High／1 Medium／3 Low。

- **M1（已修）**：原本的 E2E 只測了手刻的儲存篩選對話框，沒測共用的 Radix 對話框；而且照原寫法，換成 Radix 對話框會「修好也紅」（body 的 pointer-events:none）。已補第二條測試，並改成量的那一下打開 pointer-events。
- **L1（已修）**：註解說 Positioner 是 `position:fixed`，其實預設是 absolute；結論一樣（有 transform 就自成疊層）。改成「用 transform 定位」。
- **L2（接受）**：z-[90] 也會蓋過右下角的通知（z-50～60）。提示框只在滑鼠停著時出現，可以接受。
- **L3（已處理）**：story 勾選與 sprint-status 更新。

### File List

- `apps/web/src/components/ui/Tooltip.tsx`、`Tooltip.spec.tsx`
- `tests/e2e/tooltip-in-dialog.spec.ts`（新）
- `_bmad-output/implementation-artifacts/disc-2026-10-tooltip-hidden-in-dialog.md`、`sprint-status.yaml`
