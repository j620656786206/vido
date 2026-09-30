# Story: bugfix-learned-patterns-ui-orphaned — 刪掉點不到的「學習規則」前端，功能另立單重啟

Status: review

<!-- SM Bob create-story 2026-09-30。Alexyu 兩輪 Party Mode 後裁定：「把這功能留下來，照上面 1 到 5 的步驟去做」——
     本張是步驟 2＋3（刪前端孤兒、把帳改正）；步驟 4 的功能單是 `feat-learning-rules-relaunch`（sprint-status backlog＋PRD FR35）。
     原單：TestSprite B4（TC057-059）2026-08-31。行號為 main `3e6e354a`。 -->

## Story

身為 Vido 的維護者，
我不要程式碼裡留著一個寫好卻沒掛路由的設定頁、一個沒人呼叫的提示框，
帳上卻寫著 Story 3-9「完成」——這會讓人以為「學習規則」能用，其實從來沒存過任何一條規則。

## 背景（查到的事）

- **前端孤兒**：`components/learning/LearnedPatternsSettings.tsx`、`LearnPatternPrompt.tsx`（各含 spec）只被 `routes/test/-gallery.fixtures.tsx` 引用；`hooks/useLearning.ts`、`services/learning.ts` 只被這兩個元件與 fixtures 引用。`routes/settings/` 沒有任何路由掛它們，`SettingsLayout.tsx` 的 `SETTINGS_CATEGORIES` 也沒有。
- **從沒存過規則**：正式選片框 `components/media/ManualMatchDialogV2.tsx:118-122` 送 `applyMetadata` 時沒有 `learnPattern`；dev-only 的 `ManualSearchDialog.tsx:112` 送 `learnPattern: true`，但後端 `services/metadata_service.go:811-816` 只 `slog.Debug` 一行 TODO。
- **學了也不釘 ID**：`parser_service.go:485-542` 命中規則回 `LearnedTmdbID`，`enrichment_service.go:632-636` 只拿 `Title` 再搜 TMDb。
- **e2e 空殼**：`tests/e2e/learning.spec.ts` 三條測試的斷言是 `body` 可見／`if (sectionVisible)`——永遠 PASS、什麼都沒驗。
- **後端引擎是好的**：`internal/learning/{pattern,matcher}.go`、`learning_service.go`、`learning_repository.go`、`/api/v1/learning/patterns` 三個端點＋`/stats`，測試齊全。**本張不動後端。**
- **PRD 漏搬**：v3 FR24「從使用者修正中學習」在 v4 遷移時消失，v4 的 FR24 是別的需求（選 AI 供應商）；`epics.md` Epic 15–18 皆無此功能。

## Acceptance Criteria

1. `apps/web/src/components/learning/`、`hooks/useLearning.ts`、`services/learning.ts` 刪除；圖庫 fixtures 不再引用它們；`git grep -i "LearnedPatternsSettings\|LearnPatternPrompt\|useLearning\|services/learning" -- apps/web/src` 零命中。
2. `tests/e2e/learning.spec.ts` 刪除（空殼測試）。
3. 後端 `internal/learning/`、`learning_service.go`、`learning_handler.go` 與其測試**一行不改**（`git diff --stat -- apps/api` 為空）。
4. 帳改正：`sprint-status.yaml` 的 `3-9-filename-mapping-learning-system` 加上稽核更正註（狀態維持 done，Epic 3 已封存）；`3-9-filename-mapping-learning-system.md` 檔頭加同一段。
5. 功能重啟入口存在：`sprint-status.yaml` 新增 `feat-learning-rules-relaunch: backlog`（傘狀，列出三個接頭、季集設計洞、管理頁安全閥、Sally 稿前置、吸收的舊單）；`prd.md` 補回 FR35。
6. web 全量測試、lint、typecheck 綠。

## Tasks / Subtasks

- [x] Task 1 — 刪前端孤兒與空殼 e2e、清 fixtures import 與註解
- [x] Task 2 — 帳改正（3-9 註、story 檔頭）＋功能單（sprint-status、PRD FR35）
- [x] Task 3 — 全量測試、lint、typecheck

## Dev Notes

- `ManualSearchDialog.tsx:112` 的 `learnPattern: true` 與 `useManualSearch` 的 `learnPattern` 參數**刻意保留**：那是 dev-only `/test/manual-search` 路由，e2e 綁著它的 DOM；重啟時由 FE 單決定要走哪個選片框。
- `tests/support/helpers/api-helpers.ts:285-320, 566-570` 的 learning API helper 保留——它打的是仍在的後端端點，重啟時 BE 單會用到。
- `eslint-rules/no-emoji-in-ui.spec.ts:60` 的註解原本點名 `LearnPatternPrompt` 的 ✓，改成中性描述；規則本身不變。

### Time-dependent visual coverage

- N/A — 刪掉的兩個 fixture（`learning-learn-pattern-prompt`、`learning-learned-patterns-settings`）在 `tests/visual` 沒有已提交的基準（bucket P／Task 3 的 `screen-section` 節點從未 bless）。

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（Amelia）

### Completion Notes List

- 刪除 8 個前端檔＋1 個空殼 e2e；`routeTree.gen.ts` 由 TanStack 外掛重新產生（−21 行 `PendingRoute`）。
- AC1 grep 零命中；AC3 `git diff --stat -- apps/api` 為空。
- 全量：`pnpm nx test web` 288 檔／4,391 條綠（刪前 291／4,416：少掉的 3 檔 25 條正是 `pending.spec`＋兩個 learning spec）；`web:typecheck` 綠；eslint 改到的 4 檔 0 errors（1 個既有 unused-directive warning，不在本張改動行）；prettier 全過。
- 🔗 AC Drift: NONE。📎 Contract Stamps: NONE（後端零改動）。🎭 A11y Pre-Flight: PASS（只刪不加）。🎨 UX Verification: SKIPPED — 沒有新畫面；刪掉的兩個元件設計稿裡本來就沒有對應畫面（檔頭自述「no current screen frame」）。

### Discovery Triage

- /ship 對抗式自審抓到兩件漏的：① 兩個刪掉的 fixture 在 `tests/visual/…-snapshots/components/learning-*/` 各有 darwin＋linux 共 12 張基準 → 一併 `git rm`，`_bmad-output/audit/visual-baseline-19-4.md` 拿掉兩列、design-coverage gap 7→6；② TestSprite TC057–059 測的就是這個元件 → 立 `plan-drift-tc057-059-learned-patterns-ui-retired`（backlog）。
- 功能單已在 AC5 建好（`feat-learning-rules-relaunch`）。

### File List

- apps/web/src/components/learning/（整個目錄刪除）
- apps/web/src/hooks/useLearning.ts、apps/web/src/services/learning.ts（刪除）
- apps/web/src/routes/test/-gallery.fixtures.tsx
- apps/web/src/eslint-rules/no-emoji-in-ui.spec.ts（註解）
- tests/e2e/learning.spec.ts（刪除）、tests/e2e/manual-search.spec.ts（註解）
- tests/visual/components.visual.spec.ts-snapshots/components/learning-learn-pattern-prompt/、learning-learned-patterns-settings/（12 張 PNG 刪除）
- _bmad-output/audit/visual-baseline-19-4.md
- _bmad-output/implementation-artifacts/bugfix-learned-patterns-ui-orphaned.md、3-9-filename-mapping-learning-system.md、sprint-status.yaml
- _bmad-output/planning-artifacts/prd.md（FR35）

## Change Log

| Date       | Change                                                            |
| ---------- | ----------------------------------------------------------------- |
| 2026-09-30 | create-story（SM Bob）；dev-story（Amelia）→ review。同 PR 與 `disc-2026-09-pending-page-hardcoded-empty` 一起出。 |
