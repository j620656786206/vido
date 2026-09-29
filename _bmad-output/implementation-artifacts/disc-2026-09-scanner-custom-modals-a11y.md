# Story: disc-2026-09-scanner-custom-modals-a11y — 媒體庫設定的編輯框按 Esc 關得掉、⋮ 選單點旁邊會收

Status: review

<!-- SM Bob create-story 2026-09-29。Alexyu 從推薦清單選 B。不需要產品裁定、不需要新設計稿（外觀照 E5-D／E5-M、E1-D）。
     行號為 main `849eb8d8`，改動前；每條「查到的事」都是 SM 本次親自讀過的位置。 -->

## Story

身為在「設定 → 媒體庫」編輯媒體庫的人，
我要「編輯／新增媒體庫」的視窗按 Esc 就關、用鍵盤不會跑到後面的頁面，
也要 ⋮ 選單按 Esc 或點旁邊就收起來——跟 App 裡其他對話框、選單一樣。

## 背景（查到的事）

- `apps/web/src/components/settings/LibraryEditModal.tsx:104-112`：自製 `fixed inset-0` 遮罩＋卡片，**沒有 `role="dialog"`、沒有 `aria-modal`、沒有 Esc、焦點不會被圈在框內**；遮罩沒有 onClick（點外面不會關）。右上 ✕ 在 `:117-124`。標題是 `<h3>`（`:114-116`）。
- `apps/web/src/components/settings/LibraryCard.tsx:67-101`：⋮ 選單是 `absolute z-10` 的 div，只靠按鈕切換（`:70`），**沒有 Esc、點外面不會關**，也沒有 `role="menu"`。
- 現成原語：
  - `components/ui/Dialog.tsx`（Radix Dialog；`DialogContent` 預設置中、`max-w-lg`、`p-6`、附一顆 ✕，`closeClassName` 可調）。先例 `settings/RestoreConfirmDialog.tsx:79-128`（dsr-3f 同樣從自製浮層換過來）。
  - Radix DropdownMenu：`components/downloads/DownloadRowActions.tsx:130-180`（`modal={false}`，避免 body 殘留 `pointer-events:none` 吞掉下一個點擊）。
- 設計稿：`flow-e-scanner/e5-d.png`、`e5-m.png`（E5-M 是置中卡片、四周留邊，不是底部抽屜）；卡片 `e1-d.png`。
- 畫廊夾具 `routes/test/-gallery.fixtures.tsx:4358-4400` `settings-library-edit-modal`（註解寫「Inline fixed inset-0 overlay (NOT Radix portal)」，三態 default／hover／focus）。`9R-UX-library-edit-modal-hover-focus-states` 記錄這三張基準一模一樣。
- 使用處：`MediaLibraryManager.tsx:75-79`（開關由父層 `editModal.open` 控制）。

## Acceptance Criteria

1. `LibraryEditModal` 改用 `ui/Dialog`：`role="dialog"`、`aria-modal`、標題為 `DialogTitle`（「編輯媒體庫」／「新增媒體庫」）、焦點圈在框內、關閉後焦點回到觸發處（Radix 預設）。
   - **Esc 關閉**（呼叫 `onClose`）；**但選字中的 Esc 不關**（`isComposing`／`keyCode 229`——框裡有名稱、路徑兩個文字框；對應 `disc-2026-09-dialog-esc-closes-mid-composition` 在本框先修）。
   - **點遮罩不關**（維持現狀：這是編輯表單，誤觸會丟掉輸入；先例 `MetadataEditorDialog.tsx:106-109`）。
   - 開啟時焦點落在「名稱」輸入框。
   - 外觀照舊：`max-w-md`、`rounded-[var(--radius-lg)]`、邊框、`p-4 sm:p-6`、手機四周留 20px（原 `p-5`）；沿用原本右上 ✕（隱藏 `DialogContent` 內建的那顆，避免兩顆）。
2. `LibraryCard` 的 ⋮ 選單改用 Radix DropdownMenu（`modal={false}`）：Esc 與點外面會收、`role="menu"`／`menuitem`、方向鍵可移動；項目「編輯」「刪除」行為不變；外觀照舊（`w-32`、邊框、陰影、項目樣式）。
3. 既有測試全部照過（必要時只改「怎麼打開選單」的操作方式，不改斷言）；新增：Esc 關框、選字中 Esc 不關、點遮罩不關、開啟焦點在名稱、有 `role="dialog"`；選單 Esc 收、點外面收、選「編輯」呼叫 `onEdit`、選「刪除」出現確認。先紅後綠。
4. 畫廊：`settings-library-edit-modal` 改為 portal 夾具（`statesOnly: ['default']`，同 `settings-restore-confirm-dialog`），更新註解；重出 `-darwin` 基準，`-linux` 由 CI bootstrap。

## Tasks / Subtasks

- [x] Task 1 — `LibraryEditModal` → `ui/Dialog`＋測試（AC #1, #3）
- [x] Task 2 — `LibraryCard` 選單 → DropdownMenu＋測試（AC #2, #3）
- [x] Task 3 — 畫廊夾具與基準；`pnpm nx test web`、`lint:all`、typecheck（AC #4）

## Dev Notes

- `DialogContent` 需要 `aria-describedby={undefined}`（沒有 Description 時避免 Radix 警告，先例 `MetadataEditorDialog.tsx:105`）。
- 手機寬度：`DialogContent` 預設 `w-full` 會貼齊螢幕邊；用 `w-[calc(100%-2.5rem)]` 保留原本 `p-5` 的 20px 邊。
- 不要改 `MediaLibraryManager`（仍以條件渲染控制開關，Dialog 固定 `open`，同 `RestoreConfirmDialog`）。
- `9R-UX-library-edit-modal-hover-focus-states`：夾具改 `statesOnly: ['default']` 是 portal 夾具的機械需求（零尺寸的狀態 div 無法 hover／focus），不是替 Sally 裁定「要不要 hover/focus 設計」——該條目補記，不結案。

### Project Structure Notes

- 純前端：`components/settings/LibraryEditModal.tsx`、`LibraryCard.tsx`＋各自 spec、畫廊夾具、視覺基準。前端 3 個 task → 單張。

### Time-dependent visual coverage

- N/A — no wall-clock-reading components touched.

### References

- 原條目：`sprint-status.yaml` `disc-2026-09-scanner-custom-modals-a11y`（dsr-3 立案）
- 先例：dsr-3f（`RestoreConfirmDialog` 換 `ui/Dialog`）

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（Amelia）

### Debug Log References

- RED：新增對話框 7 條、選單 5 條；改碼前 7 條紅（選單的「Esc 收」「點外面收」在舊碼上因為根本沒有 `role=menu` 而空洞地過，改碼後才有意義）。
- 既有測試只改一處「怎麼找」：`never implies that scanning…` 原用 `container.querySelector`，對話框改在 portal 後改用 `screen.getByTestId`，斷言不變。
- 視覺：第一次重出基準時 `settings-library-card`／`-media-library-manager`／`-scanner-settings` 高度少 2px——⋮ 按鈕少了外層 block 包裝，失去行框高度。加回 `<div>` 包裝後三者與既有基準一致（零差異）。

### Completion Notes List

- `LibraryEditModal` → `ui/Dialog`：`role="dialog"`＋名稱（`DialogTitle`）、焦點圈住、Esc 關（選字中 Esc 不關）、點遮罩不關（`onPointerDownOutside`／`onInteractOutside` preventDefault，同 `MetadataEditorDialog`）、開啟焦點在「名稱」、隱藏內建 ✕（保留標題列那顆）、`w-[calc(100%-2.5rem)] max-w-md` 保留手機 20px 邊。
- `LibraryCard` ⋮ → Radix DropdownMenu（`modal={false}`，同 `DownloadRowActions`）：Esc／點外面收、`role=menu`／`menuitem`、方向鍵、`data-[highlighted]` 底色；外觀照舊。
- 畫廊：`settings-library-edit-modal` 改 portal 夾具（`statesOnly: ['default']`、移除 `width`，同 `settings-restore-confirm-dialog`）；刪除 6 張舊基準（含 hover／focus 與 3 張 `-linux`），重出 `default-visual-darwin.png`（差異：名稱框有焦點框、遮罩採 `ui/Dialog` 的淡入——與其他 Radix 對話框基準一致）。`-linux` 由 CI bootstrap。
- 全量回歸：`pnpm nx test web` 291 檔／4,371 條綠；`lint:all`、`web:typecheck` 綠（純前端，未跑 api）。
- /ship 對抗式 CR：1 HIGH／3 MED／3 LOW，全在本張範圍內處理：
  - HIGH-1（關閉後焦點掉到 `<body>`——沒有 `Dialog.Trigger`，Radix 無處可還；AC #1 寫了要還）→ `onCloseAutoFocus` 依 id 找回開啟者：編輯 → 該卡片的 ⋮、新增 → 「新增媒體庫」（不存元素：開啟它的選單項目當下就被移除）。兩條測試（Radix 下一個 tick 才還焦點，用 `waitFor`）；拿掉處理器會紅。
  - MED-2（TestSprite 快取腳本會壞）→ 手動改 `TC110`、`TC070`：「編輯」改找 `menuitem`；`TC110` 的 `<select>` 絕對 XPath（對話框移到 portal 後失效）改用 `library-type-select` testid。Playwright `tests/e2e` 未碰這兩個元件。
  - MED-3（框不能捲動，矮螢幕／多路徑時「儲存」被切掉；舊版也一樣）→ `max-h-[calc(100dvh-2.5rem)] overflow-y-auto`。
  - MED-4（「點外面不關」測試空洞：Radix 下一個 tick 才掛監聽）→ 先等一個 tick；拿掉 preventDefault 會紅。
  - LOW-5（從 ⋮ 開編輯時，選單還焦點給 ⋮ → 對話框焦點圈再搶回、名稱全選）→ 選單的 `onCloseAutoFocus` 在「編輯」時 preventDefault。
  - LOW-6（無關閉動畫）、LOW-7（Firefox 原生下拉的 Esc 未驗證）：記錄，不處理。
  - 既有、範圍外：新增／刪路徑後重抓資料會蓋掉未存的名稱／類型；存檔途中關框、存完仍呼叫 `onClose`（見 Discovery Triage）。
- 🔗 AC Drift: NONE（Story 7b-4 的開關、存檔、路徑增刪行為不變；只換容器與選單原語）。
- 📎 Contract Stamps: NONE（無 wire contract）。
- 🎭 A11y Pre-Flight: PASS（2 components；新增 role=dialog／aria-modal／menu 語意；0 jsx-a11y warnings introduced）。
- 🎨 UX Verification: PASS — 卡片、選單外觀與 E1-D／E5-D／E5-M 一致；三個卡片相關基準零差異。

### Discovery Triage

- ① — 本框的「選字中 Esc 關掉對話框」（`disc-2026-09-dialog-esc-closes-mid-composition` 的一個實例）已在 AC #1 併入修掉；該條目其餘對話框仍待處理（已補記）。
- ③ — CR 既有問題：新增／刪除路徑後清單重抓，`useEffect`（`LibraryEditModal.tsx:48-54`）會覆蓋使用者改了但還沒存的名稱／類型 → `disc-2026-09-library-edit-refetch-overwrites-draft`（P3）。
- ③ — `9R-UX-library-edit-modal-hover-focus-states`：夾具改 portal 後只拍 default，三張相同基準的成本問題消失；「要不要 hover／focus 設計」仍由 Sally 裁定（已補記，不結案）。

### File List

- apps/web/src/components/settings/LibraryEditModal.tsx
- apps/web/src/components/settings/LibraryEditModal.spec.tsx
- apps/web/src/components/settings/LibraryCard.tsx
- apps/web/src/components/settings/LibraryCard.spec.tsx
- apps/web/src/routes/test/-gallery.fixtures.tsx
- tests/visual/components.visual.spec.ts-snapshots/components/settings-library-edit-modal/（刪 6 張、新增 default-visual-darwin.png）
- _bmad-output/implementation-artifacts/disc-2026-09-scanner-custom-modals-a11y.md
- _bmad-output/implementation-artifacts/sprint-status.yaml

## Change Log

| Date       | Change                                         |
| ---------- | ---------------------------------------------- |
| 2026-09-29 | create-story（SM Bob）。                       |
| 2026-09-29 | dev-story（Amelia）：兩個元件換原語；14 條新測試；基準更新 → review。 |
| 2026-09-29 | /ship 對抗式 CR：修焦點歸還、捲動、兩個 TestSprite 腳本、兩條測試；立案 1 條。 |
