# Story infra-optin-usage-report-b2: 首次設定精靈多一步「匿名使用回報」，完成頁列出選擇

Status: review

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

- [x] Task 1: `UsageReportStep.tsx`＋測試（AC #1、#2、#6）
- [x] Task 2: `SetupWizard.tsx`／`setupService.ts` 串接＋步數測試更新（AC #3、#5、#7）
- [x] Task 3: `CompleteStep.tsx` 摘要列＋測試（AC #4）
- [x] Task 4: Gallery＋visual（AC #8）
- [x] Task 5: 文件（AC #9）

### Time-dependent visual coverage
N/A — no wall-clock-reading components touched。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（2026-10-05，dev-story／Amelia）

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created（SM Bob，2026-10-05）
- **Task 1**：`components/setup/UsageReportStep.tsx`（Design ref N6-D）——標題、說明、開關（`data.usageReportEnabled === true` 才是開，預設關）、「會送／絕不送」兩欄（CSS grid 兩欄，列預設 stretch → 自動等高；`--bg-secondary`＋`--border-subtle`）、「看完整說明 →」（共用 b1 的 `USAGE_REPORT_DOCS_URL`）、`StepNav` 只傳 `onBack`／`onNext`（不傳 `onSkip` → 沒有跳過）。
- **Task 2**：`SetupWizard.tsx` 在 `api-keys` 與 `complete` 之間加 `{ id: 'usage-report', title: '匿名回報' }`；`handleNext` 對它送 `usageReportEnabled`；`handleFinish` 帶 `usageReportEnabled: formData.usageReportEnabled === true`（沒動過＝`false`）；檔頭 Design ref 加 N6-D。`setupService.ts` 的 `SetupConfig` 加 `usageReportEnabled?: boolean`（`camelToSnake` → `usage_report_enabled`，對上 a2）。
- **Task 3**：`CompleteStep.tsx` 摘要多一列「匿名使用回報｜開啟／關閉」，關閉走既有「未設定」的 `--text-muted` 色調。
- **測試**：`UsageReportStep.spec.tsx` 6 測（預設關、兩欄內容、開關寫入資料、開→關、只有上一步／下一步、連結）；`CompleteStep.spec.tsx` +2（關閉為 muted、開啟）；`SetupWizard.spec.tsx` 步數 5→6、6 個步驟點、走完全程時新步驟沒有跳過、`completeSetup` 預設帶 `usageReportEnabled: false`、新增「打開開關後完成頁顯示開啟且送出 true」；另外 5 個既有測試多走一步（都是跳過 API 金鑰後直接按完成的路徑）。
- **Task 4**：gallery 新增 `setup-usage-report-step`（N6-D 預設關閉）；`setup-step-progress` fixture 改 6 步；`setup-complete-step` 因多一列而改變。三組 `-darwin` 基準在本機重產（舊的 `-darwin` 刪掉再 `update-missing`），**舊的 `-linux` 已 `git rm`**，交給 CI bootstrap；`test:visual` 全套重跑綠。
- **Task 5**：兩份文件移除「下一版提供」的說明，改寫成「首次設定精靈問一次」＋「設定 → 連線設定最底下有開關」。
- **Gate**：`nx test web` 297 檔／4564 測全綠；`lint:all` 0 errors（警告 172，與 main 相同）；`prettier --check .` 綠；`test:visual` 綠；`test:cleanup` 無殘留。
- TestSprite TC108（「走完精靈」）的步驟是「有跳過就按跳過、直到最後一步」，新步驟只有下一步，計畫文字仍適用，不需改。
- 🔗 AC Drift: FOUND（預期內）— `SetupWizard.spec.tsx` 寫死的「5 步」改為 6 步；dsr N 系列（N1–N5）的步驟點數由設計單 PR #666 先改為 6，程式跟上。
- 📎 Contract Stamps: NONE。
- 🎭 A11y Pre-Flight: PASS（1 個元件；開關 `role="switch"`＋`aria-checked`＋`aria-labelledby`、44px；兩欄用 `h3`＋`ul`；eslint jsx-a11y 對新檔 0 警告）。
- 🎨 UX Verification：

| Area | Design（N6-D） | Implementation | Match? |
| --- | --- | --- | --- |
| 標題／說明 | H 級標題＋說明兩行 | `text-lg semibold`＋`text-sm secondary` | ✅ |
| 開關列 | 標籤＋提示，右側 Switch Off | 同 | ✅ |
| 兩欄方塊 | `bg-secondary`＋`border-subtle`、等高 | grid 兩欄 stretch、同 token | ✅ |
| 連結 | `accent-text` 小字 | `text-xs accent-text` | ✅ |
| 按鈕 | 上一步＋下一步（填滿） | `StepNav` 無跳過 | ✅ |
| 進度點 | 6 點（亮到第 5） | `StepProgress` 依步驟陣列 6 點 | ✅ |
| 完成頁 | 多一列「匿名使用回報｜關閉」 | 同，muted | ✅ |

### Discovery Triage

- N/A — no out-of-scope work discovered

### File List

- apps/web/src/components/setup/UsageReportStep.tsx（新）、UsageReportStep.spec.tsx（新）
- apps/web/src/components/setup/SetupWizard.tsx、SetupWizard.spec.tsx
- apps/web/src/components/setup/CompleteStep.tsx、CompleteStep.spec.tsx
- apps/web/src/services/setupService.ts
- apps/web/src/routes/test/-gallery.fixtures.tsx
- tests/visual/components.visual.spec.ts-snapshots/components/setup-usage-report-step/default-visual-darwin.png（新）
- tests/visual/components.visual.spec.ts-snapshots/components/setup-step-progress/、setup-complete-step/（`-darwin` 重產、`-linux` 刪除）
- docs/usage-report.md、docs/usage-report.zh-TW.md
- _bmad-output/implementation-artifacts/sprint-status.yaml

### Change Log

| Date | Change |
| --- | --- |
| 2026-10-05 | dev-story：精靈第 5 步「匿名使用回報」、完成頁摘要列、6 個步驟點、gallery＋darwin 基準、文件移除「下一版」；Status → review |
