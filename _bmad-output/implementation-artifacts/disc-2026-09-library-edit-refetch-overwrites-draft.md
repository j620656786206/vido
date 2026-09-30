# Story: disc-2026-09-library-edit-refetch-overwrites-draft — 編輯媒體庫時，加一條路徑不會把還沒存的名稱蓋掉

Status: done

<!-- SM Bob create-story 2026-09-30。Alexyu：「做下一張」。原單：disc-2026-09-scanner-custom-modals-a11y 的 /ship 對抗式 CR（既有問題）。
     行號為 main `51e52308`；每條「查到的事」都是 SM 本次親自讀過的位置。 -->

## Story

身為在「編輯媒體庫」框裡改東西的人，
我先改了名稱或類型、再新增或刪除一條路徑，
改好的名稱與類型不能被默默變回原本的值。

## 背景（查到的事）

- `components/settings/LibraryEditModal.tsx:48-54`：`useEffect` 依 `existingLibrary` 把名稱／類型／自動字幕設回伺服器的值。新增或刪除路徑成功後媒體庫清單重抓，`existingLibrary` 變成新的物件 → effect 再跑一次 → 使用者還沒存的修改被蓋掉。
- 同檔 `:57-78` `handleSave`：存檔途中把框關掉，存完仍呼叫 `onClose`；`MediaLibraryManager.tsx:74-78` 的 `onClose` 是共用的 `setEditModal({ open: false })`，可能關掉之後又打開的框。

## Acceptance Criteria

1. 表單只在第一次拿到媒體庫資料時填入；之後的重抓不再覆蓋使用者的輸入。
2. 打開時資料還沒到，到了之後照樣填入。
3. 存檔完成時框已經關了 → 不呼叫 `onClose`、不寫錯誤。
4. 測試（先紅後綠）：重抓後保留名稱／類型／自動字幕；資料晚到仍填入；關閉後完成的存檔不呼叫 `onClose`。

## Tasks / Subtasks

- [x] Task 1 — 只填一次＋卸載後不呼叫 `onClose`＋測試
- [x] Task 2 — 全量測試、lint、typecheck

## Dev Notes

- `MediaLibraryManager` 以 `key={libraryId ?? 'new'}` 掛載，確保換一個媒體庫就是新的表單（目前對話框擋住外部點擊，實際換不了，這是保險）。

### Time-dependent visual coverage

- N/A — no visual change.

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（Amelia）

### Completion Notes List

- RED：「重抓後保留」與「關閉後完成的存檔」兩條在舊碼失敗。GREEN：`seededRef` 只填一次；`mountedRef` 擋掉卸載後的 `onClose`／`setError`。
- 自我對抗檢查（變更小）：StrictMode 雙重掛載下 `mountedRef` 由 effect 重設為 true、`seededRef` 第二次略過（值已填好）；新增模式不受影響；媒體庫在框開著時被刪除 → 表單保留輸入、存檔時由既有錯誤處理顯示。
- 全量：`pnpm nx test web` 291 檔／4,416 條綠；`lint:all`、`web:typecheck` 綠。
- 🔗 AC Drift: NONE。📎 Contract Stamps: NONE。🎭 A11y Pre-Flight: PASS（無標記變動）。🎨 UX Verification: SKIPPED — 無畫面變化。

### Discovery Triage

- 無。

### File List

- apps/web/src/components/settings/LibraryEditModal.tsx、LibraryEditModal.spec.tsx
- apps/web/src/components/settings/MediaLibraryManager.tsx
- _bmad-output/implementation-artifacts/disc-2026-09-library-edit-refetch-overwrites-draft.md、sprint-status.yaml

## Change Log

| Date       | Change                                          |
| ---------- | ----------------------------------------------- |
| 2026-09-30 | create-story（SM Bob）；dev-story（Amelia）→ review。 |
| 2026-09-30 | PR #621 合併 → done。 |
