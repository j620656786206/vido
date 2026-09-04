# Story 6.8b: 確認框顯示模型選擇 + 價錢 + 時間（前端）

Status: in-progress

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

- [ ] **Task 1 — 設計（AC: #1）**：`.pen` F16/F19 更新 + 重出截圖 — **提示詞已備妥，等 Alexyu 在 Pencil 執行**（[[feedback_pen_inline_agent_workflow]]）
- [x] **Task 2 — service／型別／query（AC: #2）**
- [x] **Task 3 — 元件與 selector（AC: #3, #4, #5）**
- [x] **Task 4 — 測試與 fixtures（AC: #6）**

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

Claude Opus 5 (1M context)

### Completion Notes List

**四點值得 Alexyu 過目的裁量（其餘依 AC 實作）：**

1. **估時是「換算」出來的，不是後端直接給的。** 後端的 `estimated_minutes_by_model`
   是**整輪掃描**的分鐘數，但確認框問的是「我勾的這幾部要跑多久」。後端的公式是
   `片長 × 該模型的時間佔比`（`estimateMinutes`，extract／asr 同一佔比），所以
   「已勾片長 ÷ 全部可寫入片長」這個比例可以精準換算過去 —— 前端因此**沒有**自己抄一份
   佔比常數，後端改校準時這裡會跟著改。若後端沒送分鐘數，時間欄直接不顯示（不編造）。

2. **舊伺服器只給一列，不是三列。** `estimates_by_model` 缺席時（pre-sub-6-8a），
   只列出預設模型。把 Haiku 用預設模型的價錢列出來會是**金額欄位裡的謊**，而這整個
   story 的目的正好相反。

3. **「eval-1 實測品質最穩」只有真的握有最高評級的預設列才能講。** 操作者把
   `CLAUDE_MODEL` 設成 haiku 時，預設列是 B 級 —— 那句話就會變成假的，所以改成
   由 `isBestGrade` 決定，不是由 `isDefault` 決定。

4. **價差雙向顯示。** AC #4 只寫「省 $X」，但目錄裡有比預設更貴的模型（Opus 4.8）。
   選了貴的卻不講貴多少，跟藏起便宜選項是同一種隱瞞，所以也顯示「多 $X」。

**其他：**

- 確認框現在會長高（三列模型 + 明細），手機上可能超出視窗。改成 `max-h-[85vh]` +
  body 捲動，標題列與按鈕列固定 —— 確認鍵被推出畫面等於死路。
- sub-6-6 交接的 FE 半邊一併做掉：`POST /settings/keys/test` 的 additive `model`
  現在顯示為「金鑰驗證成功 · 已驗證：{model}」；舊伺服器沒送就不宣稱。
- 本機 `-darwin` 視覺基線已產生三張；`-linux` 依 CLAUDE.md 等 CI bootstrap PR。
  跑基線時發現 `retry-retry-notifications` 與 `glossary-panel-v2/seeded` 兩張
  **既有** darwin 基線有 ~1000px 漂移（diff 落在字型描邊，與本 story 無關、未動）。

### Discovery Triage

- ① expand-scope-in-place — 確認框加了模型區塊後會超出手機視窗 → 同一檔改成 body 捲動。
- ① expand-scope-in-place — sub-6-6 留下的 FE 半邊（`model` 顯示）→ 本 story 消費端一併做。
- ③ backlog-with-carry-forward-link — `.pen` **沒有 F16-M／F19-M 手機框**（只有
  F15-M-v2）。確認框在程式碼裡沒有手機分支（各寬度都是置中 `max-w-md` 對話框），
  所以本輪不新造手機框；AC #1 的「含手機版」待 Alexyu 裁定是否要補
  → `backlog-f16-f19-mobile-pen-frames`（owner: Sally）。

### File List

- `apps/web/src/components/subtitle/consent/ModelPicker.tsx`、`ModelPicker.spec.tsx`（new）
- `apps/web/src/components/subtitle/consent/consentSelection.ts`、`consentSelection.spec.ts`（modified）
- `apps/web/src/components/subtitle/consent/ConfirmGenerationDialog.tsx`、`ConfirmGenerationDialog.spec.tsx`（modified）
- `apps/web/src/components/subtitle/consent/CandidateListPanel.tsx`（modified）
- `apps/web/src/components/subtitle/consent/GenerationConsentView.tsx`、`GenerationConsentView.spec.tsx`（modified）
- `apps/web/src/components/subtitle/GenerationBatchDialogV2.tsx`（modified）
- `apps/web/src/hooks/useTranslationModels.ts`（new）
- `apps/web/src/services/subtitleService.ts`、`keySettingsService.ts`、`apps/web/src/hooks/useKeySettings.ts`（modified）
- `apps/web/src/components/settings/ApiKeysForm.tsx`、`ApiKeysForm.spec.tsx`（modified）
- `apps/web/src/routes/test/-gallery.fixtures.tsx`（modified）
- `tests/visual/components.visual.spec.ts-snapshots/components/generation-consent/{f16-model-default,f16-model-haiku,f19-over-budget-haiku}/default-visual-darwin.png`（new）
- `ux-design.pen`、`_bmad-output/screenshots/flow-f-subtitle-v2/`（Task 1，待 Alexyu 執行後補）
- `project-context.md`、`_bmad-output/implementation-artifacts/sprint-status.yaml`

### Change Log

| Date | Change |
| --- | --- |
| 2026-09-04 | Task 2 — `subtitleService.getModels()`；`TranslationModelInfo`／`ModelEstimate` 型別；`GenerationBatchStartParams.modelId`；`useTranslationModels` hook（Rule 5，key `['settings','models']`）。 |
| 2026-09-04 | Task 3 — `consentSelection` 加 `candidateUsd`／`computeTotals(prices)`／`modelChoices()`（估時依已勾片長比例換算）；新 `ModelPicker`；`ConfirmGenerationDialog` 掛入並改為 body 捲動；`CandidateListPanel` 逐列改讀該模型價；`GenerationConsentView` 持有 modelId 並串到三處金額與 batch payload。 |
| 2026-09-04 | Task 4 — 12 個新 spec（selector／picker／容器）＋ 三個 gallery fixture 與 `-darwin` 基線；sub-6-6 FE 半邊（`已驗證：{model}`）。 |
