# Story bugfix：程式碼的字級跟上設計系統——不再有 10／11／13／15px

Status: review

## Story

As the person reading Vido on a phone and a desktop,
I want every piece of text to use one of the eight sizes the design system defines,
so that labels and error messages are never smaller than 12px and every size carries a real line height.

## Context

升級自 `disc-2026-09-type-scale-even-migration`（P0）＋ `disc-2026-09-code-still-uses-10px`（P1）。

### ⚖️ 裁定（Alexyu 2026-09-28 選 A）

照 DESIGN.md「為什麼沒有 11／13／15」與「12px 是地板」段（`DESIGN.md:344-422`，PR #410 2026-09-10 寫下）已經定的做法；對照圖：https://claude.ai/artifact/WK8pMdv4bYnWXUM62fweZp
- 15px 不再用（程式碼已經 0 處）。
- 11px（含等寬讀數）→ 12px `text-xs`；10px → 12px `text-xs`；13px → 14px `text-sm`。
- 導覽小字 12px、字重 500，選中加粗（`MobileTabBar.tsx:56` 既有的 `font-medium`／`font-bold` 保留）。
- 跟中文並排、要被讀的等寬讀數用 14（DESIGN.md:366）——本次盤點的 11px 等寬讀數都是純數字欄位（年份、片長、評分、大小），走 12。

### 🔴 查到的事（2026-09-28，親自 grep／讀過；AI4）

1. 非測試檔剩 59 處奇數字級、35 個檔（`grep -rn "text-\[1[0135]px\]" apps/web/src | grep -v .spec.`）：11px × 48、10px × 6、13px × 5、15px × 0。待辦上的「13px × 72、15px × 4」是 2026-09-10 的舊數字。
2. 13px 五處：`SavePresetDialog.tsx:110, :128`、`EpisodeList.tsx:262`、`NfoLocalizeAction.tsx:70, :283`。
3. 10px 六處：`RecentMediaPanel.tsx:123`、`NewMediaNotifications.tsx:105`、`NewMediaToast.tsx:34`、`ParseCompleteToast.tsx:58, :60`、`routes/test/gallery.tsx:372`（gallery 頁的標籤）。
4. 設計稿側已完成：`.pen` 只剩八階 `Type/*` 變數（`DESIGN.md:383-396`），`check-design-tokens.py` 守門裸字級。
5. 沒有守門擋程式碼再寫回奇數字級。

## Acceptance Criteria

1. `apps/web/src`（含 `routes/test/gallery.tsx`）不再有 `text-[10px]`／`text-[11px]`／`text-[13px]`／`text-[15px]`：11、10 → `text-xs`；13 → `text-sm`。其他 class 不動。
2. 新增守門測試：掃 `apps/web/src/**/*.{ts,tsx}`（排除 `.spec.`），出現上述任一 → 失敗並列出檔案:行號。
3. 既有 spec 斷言舊 class 的，改成新 class（刻意變更，Completion Notes 列出）。
4. 視覺基準：受影響夾具 darwin 更新、過期 `-linux` `git rm`，交給 CI bootstrap（`/ship` Visual Regression 段）。
5. impeccable 檢查（Alexyu 指定）：改完後用 impeccable 對字級做一次稽核，結果寫進 Completion Notes；發現的問題能修就修、不能修就立案。
6. `pnpm nx test web`、`pnpm run lint:all`、typecheck 綠；關閉兩張 disc；DESIGN.md「程式碼尚未跟上」一段更新為現況。

## Tasks / Subtasks

- [x] Task 1 — 守門測試先紅（AC #2）
- [x] Task 2 — 59 處改掉、spec 同步（AC #1, #3）
- [x] Task 3 — 視覺基準、impeccable 稽核、DESIGN.md、收尾（AC #4, #5, #6）

## Dev Agent Record

### Completion Notes List

- **59 處全改**：`text-[11px]`×48、`text-[10px]`×6 → `text-xs`(12)；`text-[13px]`×5 → `text-sm`(14)；15px 程式碼本來就是 0。35 個原始檔。
- **守門測試** `apps/web/src/styles-type-scale.spec.ts`：先紅（59 筆）→ 改完綠。**Mutation check**：把 `ThemeToggle.tsx:99` 的 `text-xs` 改回 `text-[11px]` → 測試紅、並指出 `components/shell/ThemeToggle.tsx:99`；還原後綠。
- **Spec 同步**：5 個 spec 的正向斷言改新 class；負向 `not.toContain` 保留（仍然有意義）。
- **過期註解**：6 處註解還在說「11px 是刻意的、別清掉」（HomeReadoutBand 檔頭、InFlightBadge×2、SidebarGroupLabel、LibraryStatesV2、ManageSubtitleDialogV2 一行 TODO）——改成現況。量測紀錄型的註解（「在 11px 量到 3.58:1」）是歷史事實，不動。
- **AC drift**：`tests/e2e` 沒有任何斷言綁 `text-[1Xpx]`（grep 0 筆）。
- **視覺基準**：42 張 `-darwin.png` 重拍（通知 retry 那張是重拍雜訊，已還原）；對應 42 張 `-linux.png` 刪掉，交給 CI bootstrap。
- **impeccable 稽核（AC #5）**：`detect.mjs` 對 35 個檔零發現。人工稽核三條，**都不在本張範圍，立案**：① 行高——`text-xs` 自帶 16px、`text-sm` 自帶 20px，表上要 18／22.75；② 字重——Label 應 500，改到的 54 處 `text-xs` 有 41 處沒 `font-medium`（→ `disc-2026-09-type-line-height-weight-drift`）；③ `text-3xl`×5、`text-5xl`×1 不在表上、手機降階不存在（→ `disc-2026-09-type-scale-display-and-mobile-step`）。對比度：字變大 1px，沒有任何一處變差。
- **DESIGN.md** §Hierarchy「程式碼尚未跟上」段落改寫為現況＋剩下三件事。
- **收掉的 disc**：`type-scale-even-migration`（P0）、`code-still-uses-10px`（P1）、`11px-micro-label-not-on-type-scale`（P3）。
- 驗證：web 4308/4308、lint 0 errors、prettier 綠、typecheck 綠。

### File List

- `apps/web/src/styles-type-scale.spec.ts`（新）
- 35 個 `apps/web/src/**` 原始檔（字級）＋ 6 處註解
- 5 個 spec：`LibraryStatesV2`、`GenerationProgressV2`、`GlossaryRowV2`、`AvailabilityBadge` 等
- `apps/web/tests/visual/**`：42 `-darwin.png` 更新、42 `-linux.png` 刪除
- `DESIGN.md`、`_bmad-output/implementation-artifacts/sprint-status.yaml`

## Change Log

| Date | Change |
| --- | --- |
| 2026-09-28 | 建單＋開工（Alexyu 選 A）：由 P0／P1 兩張 disc 升級 |
| 2026-09-28 | 開發完成：59 處改完、守門測試、impeccable 稽核（立 2 案）、DESIGN.md 更新 → review |
