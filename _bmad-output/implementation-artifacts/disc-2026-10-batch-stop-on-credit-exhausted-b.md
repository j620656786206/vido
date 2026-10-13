# Disc：AI 帳號沒錢時，批次整批停下來（b：畫面）

Status: review

**Source:** 同 `disc-2026-10-batch-stop-on-credit-exhausted-a`。2026-10-13 Alexyu：「沒錢就整批停，補生成工作區那一列新的文字『AI 帳號沒有額度』的設計稿」。

**設計稿**：2026-10-13 Alexyu 在 Pencil 跑完 Sally 的兩段提示詞（`disc-2026-10-batch-stop-on-credit-exhausted-pen-prompts.md`），Sally 用 MCP 逐字核對過。F9b-D-v2 節點 `YHnnC`，截圖 `flow-f-subtitle-v2/f9b-d-v2.png`；F11-SPEC-LOG 新列 `xkmMv`（⑤b）、`IDRsT`（⑪）。

⚠️ **a 和 b 要一起部署**（理由見 a）。

## 設計裁定（Sally，2026-10-13）

- **帳號沒錢是「已停止」，不是「失敗」。** 跟預算上限同一類：東西沒壞，是錢的問題，用赭色（warning）。單部的紀錄列從現在的紅色「失敗 · 片名 · AI 帳號沒有額度」改成赭色「已停止 · 片名 · AI 帳號沒有額度」。
- **批次停下來的樣子照抄預算上限（F9）**，只換橫幅文字，再加一個「去 OpenAI 儲值 ↗」連結。剩下的片子照舊寫「已暫停 — 下次繼續」，按鈕照舊是「關閉」「下次繼續」。
- 新設計稿：
  - **F9b-D-v2**（從 F9-D-v2 `JMqPg` 複製）：批次對話框，帳號沒錢。
  - **F11-SPEC-LOG**（`tT3JX`）加兩種列：⑤b 單部已停止（帳號沒錢）、⑪ 批次層「AI 帳號沒有額度」。

## Story

身為自己付 AI 費用的人，
我希望批次因為帳號沒錢停下來時，畫面直接告訴我「哪一家沒錢、去哪裡儲值」，
這樣我知道不是片子壞了，儲值後按「下次繼續」就好。

## Acceptance Criteria

1. 前端認得 `credit_exhausted`：加進所有批次狀態型別和「結束狀態」清單（`useGenerationBatchProgress.ts`、`useGenerationJobsFeed.ts`、`generationWorkspace.ts`、`services/subtitleService.ts`），收到就關掉 SSE，不會卡在「進行中」。
2. 批次對話框（`GenerationBatchDialogV2.tsx`）照 F9b：橫幅文字、儲值連結（新分頁開 `billing_url`）、「下次繼續」行為跟預算上限一樣。
3. 生成工作區（`GenerationWorkspaceV2.tsx`）：橫幅照 F9b 原句；狀態小標寫「沒有額度」。
4. 紀錄列（`generationEventCopy.ts`）照 F11-SPEC-LOG：
   - 單部帳號沒錢：⑤b 赭色「已停止」。
   - 批次層：⑪「AI 帳號沒有額度 · 本批次 · OpenAI」。
5. 測試：每個檔案的 spec，加 gallery 的 F9b 樣本（批次對話框、生成工作區）與 visual 基準。
7. 儲值連結只接受 `https://` 開頭的網址；沒有網址（舊伺服器）就不顯示連結，名稱寫「AI」。
6. 拿掉或改名 data-testid 前先 grep `tests/e2e`。

## Tasks / Subtasks

- [x] T0 設計稿（Alexyu 跑提示詞 → Sally review）
- [x] T1 型別與結束狀態清單（AC #1）
- [x] T2 批次對話框 F9b（AC #2、#7）
- [x] T3 生成工作區（AC #3）
- [x] T4 紀錄列 ⑤b／⑪（AC #4）
- [x] T5 測試、gallery、visual（AC #5、#6）

### Time-dependent visual coverage

N/A — 新畫面不讀現在時間。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5），Amelia dev-story，2026-10-13。

### Completion Notes List

- **Sally review（2026-10-13）**：兩段提示詞的文字和顏色逐字相符。兩處處理：
  - 提示詞 2 把註記字級寫成裸數字 13，`check-design-tokens.py` 會擋。Sally 用 MCP 改成 `$Type/Label/Size`（只改這一個屬性），用選單存檔。
  - 儲值連結設計稿沒有底線。實作加了底線：只靠顏色分辨連結不符合 WCAG 1.4.1。追認，不回頭改稿。
- 新檔 `creditStop.ts`（廠商名稱、播報句、只收 https 網址）和 `CreditStopMessage.tsx`（F9b 橫幅文字＋連結）。對話框和工作區共用同一段文字。
- 前端認得 `credit_exhausted`：`subtitleService.ts`、`useGenerationBatchProgress.ts`（收到就關 SSE，帶 `creditProvider`／`billingUrl`）、`useGenerationJobsFeed.ts`、`generationWorkspace.ts`（新 `isPausedStop`）。
- 批次對話框：F9b 橫幅；排隊列、「關閉」「下次繼續」跟預算上限共用（`isPausedStop`）；不顯示「重試失敗項目」，理由同預算上限（CR M1）。
- 生成工作區：F9b 橫幅、狀態小標「沒有額度」、紀錄區底下「已停止（AI 帳號沒有額度）」。`EventLogPane` 的 `budgetStopped` 改成 `stoppedLabel`。
- 紀錄列：`FailureCopy.budget` 改名 `stopped`（意思變成「為了錢停下」，不只預算）。單部帳號沒錢改成赭色「已停止」（⑤b）；批次新增 ⑪「AI 帳號沒有額度 · 本批次 · OpenAI」。
- 檢查：
  - `vitest`（subtitle、hooks、services、activity）：77 檔、1,535 個測試全綠。
  - `tsc -p tsconfig.app.json` 綠。
  - `pnpm run lint:all`（0 errors）、`format:check` 綠。
  - `check-design-refs.py`、`check-design-tokens.py` 綠。
  - 本機 visual 綠；新增兩張 `-darwin` 基準，`-linux` 由 CI 補。
  - 沒有拿掉或改名 data-testid；`tests/e2e` 沒有引用改到的文字。

### Senior Developer Review (AI) — 2026-10-13

結論：0 High／0 Medium／1 Low，接受。

- **L1（接受）**：手機版（F8-M）沒有 F9b 的手機稿。橫幅用 flex-wrap，在手機上會換行，跟預算上限的手機樣子一樣。沒有另外加手機的 visual 樣本。

### File List

- `ux-design.pen`、`scripts/export-pen-screenshots.py`、`_bmad-output/screenshots/flow-f-subtitle-v2/f9b-d-v2.png`、`f11-spec-log.png`、`_bmad-output/pen-node-ids.json`、`_bmad-output/pen-tokens.json`
- `apps/web/src/components/subtitle/creditStop.ts`（新）、`creditStop.spec.ts`（新）、`CreditStopMessage.tsx`（新）
- `apps/web/src/components/subtitle/GenerationBatchDialogV2.tsx`、`.spec.tsx`
- `apps/web/src/components/subtitle/GenerationWorkspaceV2.tsx`、`.spec.tsx`
- `apps/web/src/components/subtitle/generationWorkspace.ts`、`.spec.ts`
- `apps/web/src/components/subtitle/generationEventCopy.ts`、`.spec.ts`
- `apps/web/src/hooks/useGenerationBatchProgress.ts`、`.spec.ts`
- `apps/web/src/hooks/useGenerationJobsFeed.ts`、`.spec.ts`
- `apps/web/src/services/subtitleService.ts`
- `apps/web/src/routes/test/-gallery.fixtures.tsx`
- `tests/visual/components.visual.spec.ts-snapshots/components/generation-batch-dialog-v2/credit_exhausted/`、`generation-workspace-v2/credit_exhausted/`
- `_bmad-output/implementation-artifacts/disc-2026-10-batch-stop-on-credit-exhausted-b.md`、`sprint-status.yaml`
