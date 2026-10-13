# Disc：AI 帳號沒錢時，批次整批停下來（b：畫面）

Status: blocked — 等設計稿

**Source:** 同 `disc-2026-10-batch-stop-on-credit-exhausted-a`。2026-10-13 Alexyu：「沒錢就整批停，補生成工作區那一列新的文字『AI 帳號沒有額度』的設計稿」。

**擋住的原因**：要等 Alexyu 在 Pencil 跑完 Sally 的兩段提示詞（`disc-2026-10-batch-stop-on-credit-exhausted-pen-prompts.md`），存檔、出截圖。

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

## Acceptance Criteria（設計稿到了再由 Bob 對稿補齊）

1. 前端認得 `credit_exhausted`：加進所有批次狀態型別和「結束狀態」清單（`useGenerationBatchProgress.ts`、`useGenerationJobsFeed.ts`、`generationWorkspace.ts`、`services/subtitleService.ts`），收到就關掉 SSE，不會卡在「進行中」。
2. 批次對話框（`GenerationBatchDialogV2.tsx`）照 F9b：橫幅文字、儲值連結（新分頁開 `billing_url`）、「下次繼續」行為跟預算上限一樣。
3. 生成工作區（`GenerationWorkspaceV2.tsx`）：橫幅照 F9b 原句；狀態小標寫「沒有額度」。
4. 紀錄列（`generationEventCopy.ts`）照 F11-SPEC-LOG：
   - 單部帳號沒錢：⑤b 赭色「已停止」。
   - 批次層：⑪「AI 帳號沒有額度 · 本批次 · OpenAI」。
5. 測試：每個檔案的 spec，加 gallery 的 F9b 樣本與 visual 基準。
6. 拿掉或改名 data-testid 前先 grep `tests/e2e`。

## Tasks / Subtasks

- [ ] T0 等設計稿（Alexyu 跑提示詞 → Sally review）
- [ ] T1–T5 設計稿到了再拆

### Time-dependent visual coverage

待補。

## Dev Agent Record
