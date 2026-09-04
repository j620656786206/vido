# Story 6.8b: 確認框顯示模型選擇 + 價錢 + 時間（前端）

Status: ready-for-dev

**Depends on:** sub-6-8a (backend API must be ready) — `GET /settings/models`、`estimates_by_model`、`estimated_minutes_by_model`、batch `model_id`.

## Story

As a BYOK NAS owner,
I want the confirm step to show me each model's price for *this* batch, its measured quality grade and rough time, with Sonnet preselected,
so that saving money with Haiku is a choice I see and make — and the 2.7× gap is never hidden.

## Context — 這個 story 為什麼存在

sub-6-8a 的消費面。設計語彙來自 party-mode Sally 2026-09-03：

```
選擇翻譯模型
● Claude Sonnet 5      這批約 $0.53   品質 A   約 11 分鐘   （預設）
○ Claude Haiku 4.5     這批約 $0.21   品質 B   約 9 分鐘
○ Gemini 2.5 Flash     這批約 $0.06   品質 尚未評測 · 可花約 $0.01 試跑 20 句
```

## Acceptance Criteria

1. **UX 先行（Sally）。** F16／F19 金額確認框（`ConfirmGenerationDialog.tsx`，Design ref `gmOt6`／`KThbY`）加「翻譯模型」區塊：radio 清單，每列＝display name＋本批估價＋品質等級徽章＋預估時間；預設勾 `is_default`。`ux-design.pen` 更新 F16/F19 兩個 frame（含手機版），依 `CLAUDE.md` 流程重出 `flow-f-subtitle-v2/` 對應截圖並同 commit。**未評測模型**顯示「尚未評測」中性徽章＋一行「可花約 $0.01 試跑 20 句」（本 story 只顯示文案；試跑功能屬 P1-8）。

2. **資料流。** `subtitleService.ts` 加 `getModels()`（TanStack Query，key `['settings','models']`，Rule 5/18）；`CandidateAnalysisSnapshot` 型別加 `estimatesByModel`／`estimatedMinutesByModel`（optional，舊伺服器 fallback：只顯示預設模型一列）；`GenerationBatchStartParams.modelId`。

3. **金額同源。** 選定模型後，F15 摘要條／footer／確認框三處金額**同一 selector**改讀該模型估價（sub-4-3 AC #2 的「禁止三份獨立加總」紅線延續）；預算上限 F18 判定也用選定模型的合計。

4. **價差明示。** 非預設模型被選時，該列下方顯示對比文案「比 Sonnet 省 $X（Y%）」；預設列顯示「eval-1 實測品質最穩」。Haiku 選中時確認按鈕文案不變（不做勸退，只做告知）。

5. **a11y 與慣例。** radio group `role="radiogroup"` + `aria-label="翻譯模型"`；44px 觸控；Rule 21 header 更新為多畫面併列；jsx-a11y 零新增 warning。

6. **測試。** specs：預設選中 is_default、切換後三處金額同步、F18 依選定模型、舊伺服器 fallback 單列、未評測徽章、payload 含 `model_id`；visual gallery fixtures：f16-model-default／f16-model-haiku／f19-over-budget-haiku（`-darwin` 本機、`-linux` 等 CI bootstrap PR）；dev-story Step 9 截圖比對。

## Tasks / Subtasks

- [ ] **Task 1 — 設計（AC: #1）**：Sally 更新 `.pen` F16/F19（+ 手機）、重出截圖
- [ ] **Task 2 — service／型別／query（AC: #2）**
- [ ] **Task 3 — 元件與 selector（AC: #3, #4, #5）**
- [ ] **Task 4 — 測試與 fixtures（AC: #6）**

## Dev Notes

- **Inherited from sub-6-6 (AC #3 FE half):** `POST /settings/keys/test` 200 now carries additive `model` (the verified model id). `ApiKeysForm` should render「已驗證：{model}」next to the valid state; codes `AI_UNAUTHORIZED` / `AI_MODEL_NOT_FOUND` are now distinct from `AI_PROVIDER_ERROR` if the form wants to branch.

- `usd()` helper 已抽共用（`lib/currency`）。
- `consentSelection.ts` 是三處金額的唯一 selector——模型維度加在這裡，不要在元件裡各算各的。
- 時間顯示用「約 N 分鐘」，來源 `estimated_minutes_by_model`；未知 → 不顯示時間欄。

### Time-dependent visual coverage

- N/A — no wall-clock-reading components touched（估時來自後端數字）。

### References

- sub-6-8a AC #2/#3/#4（`[@contract-v1]` models 端點，記 ack）
- sub-4-3 AC #2/#3/#4（金額同源、F18、F16/F19）
- `apps/web/src/components/subtitle/consent/{ConfirmGenerationDialog.tsx,consentSelection.ts,GenerationConsentView.tsx}`

## Dev Agent Record

### Agent Model Used

### Completion Notes List

### Discovery Triage

- （dev 填）

### File List
