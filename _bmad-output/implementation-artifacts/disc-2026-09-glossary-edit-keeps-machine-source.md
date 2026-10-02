# Story disc-2026-09-glossary-edit-keeps-machine-source: 在詞彙表改過的譯名自動算「已確認」，來源不變 — 後端＋前端

Status: review

<!-- SM Bob create-story 2026-10-02（自 sprint-status 同名條目轉成單；⚖️ Alexyu 2026-10-02 裁定選項 1）。行號為 main `9d264d6c`。sub-8-1 匯入規則的前置。 -->

## Story

身為會在詞彙表面板裡改譯名的人，
我改過一個機器抓來的詞，就代表我看過、認可了新譯名——不該還掛著「未確認」要我再按一次，
而它最早從哪來（字幕／TMDb／官方字幕）是事實，不該因為我改了就被抹掉。

## 背景（查到的事）

- 面板「編輯」送 `PUT {term_zh, confirmed: 該列原本的值}`（`GlossaryPanelV2.tsx:303-308`）→ 原本未確認的詞改完仍未確認。
- 後端 `GlossaryService.Edit` → `GlossaryRepository.Update(id, termZh, confirmed)` 只動 `term_zh`／`confirmed`，`source` 不變（`glossary_repository.go:227-239`）。
- `models/glossary.go:19` 把 `manual` 定義成「entered/edited by the user」，與實際行為不符。
- sub-8-1 匯入規則「既有 confirmed=1 或 manual → 跳過」：這條裁定後，使用者改過的詞一定是 confirmed=1，自然被保護。

## ⚖️ 裁定（Alexyu 2026-10-02，選項 1）

編輯＝使用者看過並改了 → **自動確認**；**來源保留**（出處是事實）；不另加「已修改」旗標。

## Acceptance Criteria

1. **後端強制**：`GlossaryService.Edit` 一律以 `confirmed=true` 寫入，不論 body 帶什麼——規則放在服務層，任何 client（含未來的 CLI／匯入工具）都一樣。`PUT` body 形狀不變（0 bump），`confirmed` 欄位保留但文件註明「編輯一律確認」。`source` 不動。
2. **前端一致**：面板編輯送 `confirmed: true`，存完那一列的「未確認」標記消失（refetch 後由伺服器值決定）。
3. **定義修正**：`GlossarySourceManual` 註解改為「由使用者新增」；編輯不改來源。
4. **測試**：服務層「body 帶 false 仍寫 true、source 不動」；前端「原本未確認的列改完送 true」。

## Tasks / Subtasks

- [x] Task 1 — 服務層強制確認＋模型註解（AC #1, #3）
- [x] Task 2 — 前端送 true（AC #2）
- [x] Task 3 — 測試（AC #4）

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（Amelia）

### Completion Notes List

- `GlossaryService.Edit` 忽略傳入的 `confirmed`、一律 `repo.Update(..., true)`；handler 與 PUT body 不變。`GlossarySourceManual` 註解改正。
- `GlossaryPanelV2` 編輯送 `confirmed: true`；原本「preserving the row confirmed flag」的 spec 改寫成「未確認的列改完送 true」。
- 測試：服務層 1 條（body false → 寫 true）、前端 1 條改寫；`go test ./...`、web 全量綠。

### Discovery Triage

- 無新單。sub-8-1 的「既有 confirmed=1 或 manual → 跳過」前提因此成立。

### File List

- apps/api/internal/services/glossary_service.go、glossary_service_test.go；apps/api/internal/models/glossary.go
- apps/web/src/components/subtitle/GlossaryPanelV2.tsx、GlossaryPanelV2.spec.tsx
- \_bmad-output/implementation-artifacts/disc-2026-09-glossary-edit-keeps-machine-source.md、sprint-status.yaml

## Change Log

| Date       | Change                                                          |
| ---------- | --------------------------------------------------------------- |
| 2026-10-02 | create-story（SM Bob）＋裁定入檔＋dev-story（Amelia）→ review。 |
