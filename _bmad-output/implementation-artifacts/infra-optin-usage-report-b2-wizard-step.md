# Story infra-optin-usage-report-b2: 首次設定精靈多一步「匿名使用回報」，完成頁列出選擇

Status: ready-for-dev

**Epic:** standalone（`infra-optin-usage-report` 家族）· **Priority:** P1 · **Size:** S（前端 only）
**Source:** 設計 N6-D `GQae8`、N1–N5／N3-M 六個進度點、N5 摘要列（PR #666）；後端 `SetupConfig.usage_report_enabled`、`ValidateStep("usage-report")`（PR #665）。
**Split（SM Bob 2026-10-05）：** 由 `infra-optin-usage-report-b-frontend` 依版面拆出；與 b1 互不相依。

## Story

As a new Vido user going through first-run setup,
I want one clear step that asks whether to send the anonymous weekly count, defaulting to no,
so that I decide once, with the facts in front of me, before setup finishes.

## Context —— 查到的事（2026-10-05，SM Bob）

- 步驟陣列 `apps/web/src/components/setup/SetupWizard.tsx:34-52`；`handleNext` 依 step id 組資料（`:78-86`）；`handleFinish` 逐欄送 `completeSetup`（`:127-135`）。
- `SetupConfig` 型別 `apps/web/src/services/setupService.ts:18-27`。
- 完成頁摘要列 `apps/web/src/components/setup/CompleteStep.tsx:25-35`。
- 進度點 `StepProgress.tsx` 由 `WIZARD_STEPS` 決定數量，加一步自動變 6 點。
- `SetupWizard.spec.tsx` 寫死步數的測試：`:66`「step 1 of 5」、`:114`「5 step dots」、`:124`、`:162`。
- 後端：`ValidateStep("usage-report")` 回 nil；`SetupConfig.usage_report_enabled`（`omitempty`，預設 false）。

## Acceptance Criteria

1. 新步驟 id `usage-report`，在 `api-keys` 與 `complete` 之間；元件 `UsageReportStep.tsx`，`Design ref` N6-D（GQae8）。
2. 內容照 N6-D 定稿：標題、說明、開關列（預設關）、「會送／絕不送」兩欄（`--bg-secondary`＋`--border-subtle`、等高）、「看完整說明 →」連結、只有上一步／下一步。
3. `SetupConfig` 加 `usageReportEnabled?: boolean`；`handleNext` 對 `usage-report` 送 `usageReportEnabled`；`handleFinish` 帶上它（未操作＝`false`）。
4. 完成頁摘要多一列「匿名使用回報｜開啟／關閉」（關閉用 `--text-muted`，同「未設定」）。
5. 進度點變 6 個；既有寫死步數的測試更新為 6；`SetupWizard.tsx` 檔頭 Design ref 加 N6-D。
6. a11y：開關 `role="switch"`＋`aria-checked`＋`aria-labelledby`。
7. 測試：步驟渲染、開關切換、送出的 `completeSetup` 內容（開／關）、完成頁兩態、步數 6。
8. Gallery fixture（N6 預設關閉）＋visual 基準（`-darwin` 本機，`-linux` CI）。
9. 文件：移除 `docs/usage-report(.zh-TW).md` 與 README 中「開關下一版提供」的說法（b1 已處理設定頁那半）。

## Tasks / Subtasks

- [ ] Task 1: `UsageReportStep.tsx`＋測試（AC #1、#2、#6）
- [ ] Task 2: `SetupWizard.tsx`／`setupService.ts` 串接＋步數測試更新（AC #3、#5、#7）
- [ ] Task 3: `CompleteStep.tsx` 摘要列＋測試（AC #4）
- [ ] Task 4: Gallery＋visual（AC #8）
- [ ] Task 5: 文件（AC #9）

### Time-dependent visual coverage
N/A — no wall-clock-reading components touched。

## Dev Agent Record

### Agent Model Used

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created（SM Bob，2026-10-05）

### Discovery Triage

### File List
