# Story: disc-2026-09-batch-generation-no-asr-key-warning — 批次產生字幕，一打開就先說「金鑰還沒設定」

Status: done

<!-- SM Bob create-story 2026-09-29。Alexyu：「做 disc-2026-09-batch-generation-no-asr-key-warning」。
     需要新畫面（批次對話框多一塊提示），依 .pen 協作模式：Sally 出提示詞 → Alexyu 跑 Inline AI Agent → Sally MCP 唯讀 review → Dev。
     行號為 main `4629dcfd`；.pen 節點 id 由 SM 本次用 Pencil MCP 唯讀讀取。 -->

## Story

身為想一次幫很多部片產生字幕的人，
當伺服器還沒有產生字幕所需的金鑰時，
我要在打開「產生字幕」批次對話框的當下就看到「還沒設定、去哪裡設定」，
而不是挑完幾十部片、按下「開始產生」才被告知無法開始。

## 背景（查到的事）

- 後端 `apps/api/internal/handlers/generation_batch_handler.go:105-117`：`StartGenerationBatch` 一開頭 `if !h.processor.IsAvailable()` → 503 `TRANSCRIPTION_DISABLED`「字幕生成功能未啟用」＋建議（兩把金鑰都點名，`:108-116` 註解說明 pipeline 模式看翻譯金鑰、legacy 看 ASR 金鑰）。`IsAvailable` 定義於 `services/generation_batch.go:317-321`。
- 狀態端點 `GetGenerationBatchStatus`（`:218-225`）只回 `{running, progress, last}`，**不含可用性**。
- 前端 `components/subtitle/GenerationBatchDialogV2.tsx:875-905`：對話框一打開就呼叫 `getGenerationBatchStatus()`（恢復進行中的批次），之後進入同意清單（分析候選片）。按「開始產生」失敗時 `:1054-1055` 把錯誤訊息放進 `startError`，由 `consent/CandidateListPanel.tsx:1114-1123` 顯示在按鈕上方。所以今天的流程是：分析、挑片、按鈕，**最後**才看到「字幕生成功能未啟用」。
- 單片（管理字幕）早就會先說：`components/subtitle/generateCostView.ts:115-116`（`asrAvailable` → `ASR_NOT_CONFIGURED_LINE`，J9-D ⑥），畫面是 F5-D-v2 的 `warning-panel`（`RrbLx`：`$warning-tint` 底、標題「語音辨識尚未設定」、說明、「前往設定」）。`git grep -n "asrAvailable\|asr_available" -- apps/web/src/components` 只有 `generateCostView.ts:115` 一處——批次與工作區沒看任何可用性訊號。
- 「開始產生」停用時必須說原因（`CandidateListPanel.tsx:1183-1186` 註解，dsr-6e-1 AC #2）；按鈕在 `:1199-1213`。
- 批次對話框有兩個入口：活動中心（`ActivityHub.tsx:321,365`）與媒體庫（`LibraryBrowseV2.tsx:868`），都用同一個 `GenerationBatchDialogV2`。
- 設計稿：批次同意清單是 `f15-d-v2`（`pwMzT`；對話框 `DBPu5`、內容 `RTmxa`、底列 `rrOWa`、開始鈕 `iK0BE`＝`Component/Button/Primary`、預算提示 `twwOH`）。**沒有「尚未設定」的批次畫面**（`note-gen-capability` `c4FIoB` 也沒提）。

## Acceptance Criteria

1. **後端 `[@contract-v1]`**：`GET /subtitles/generation-batch/status` 多回 `available`（bool）＝`processor.IsAvailable()`。既有 `running`／`progress`／`last` 不變。
2. **前端**：對話框打開時讀 `available`。`false` 時，同意清單最上方顯示提示（依 Task 0 設計稿 F28-D-v2）：
   - 標題「字幕生成尚未設定」、說明「批次產生字幕需要翻譯（Claude）與語音辨識（ASR）金鑰。到金鑰設定儲存後就能開始；清單可以先看。」、「前往設定」連到 `/settings/keys`。
   - 「開始產生」停用，停用原因寫在按鈕旁原本的提示位置：「尚未設定金鑰，無法開始」。
   - 清單照常顯示、可以勾選（只是不能開始）。
3. `available` 為 `true` 或讀不到（舊後端、探測失敗）→ 行為完全不變（讀不到不擋：不能因為探測失敗就鎖住按鈕；真的不可用時 503 的既有錯誤訊息仍會出現）。
4. 測試：Go — status 在可用／不可用時帶 `available`。前端 — 不可用：提示、連結、按鈕停用與原因；可用：沒有提示、按鈕照舊；探測失敗：不擋。先紅後綠。
5. 視覺：畫廊新增「尚未設定」夾具與 `-darwin` 基準；`-linux` 由 CI bootstrap；與設計稿比對。

## Tasks / Subtasks

- [x] **Task 0（設計，Alexyu 執行）**— 把下方提示詞貼給 Pencil 的 Inline AI Agent；⌘S；`python3 scripts/export-pen-screenshots.py`；回報完成。
- [x] Task 1 — 後端 `available` ＋測試（AC #1, #4）
- [x] Task 2 — 前端讀 `available`、提示、停用與原因＋測試（AC #2–#4）
- [x] Task 3 — 畫廊夾具與基準；全量測試、lint、typecheck（AC #5）

## Sally 的 Pencil 提示詞（Task 0，整段貼給 Inline AI Agent）

```
在 ux-design.pen 新增一張桌面稿，示範「批次產生字幕」對話框在伺服器還沒有產生字幕所需金鑰時的樣子。

1. 複製現有的「f15-d-v2」（節點 pwMzT），放在同一個群組裡、f25-d-v2（UYud2）右邊 100px 處（y 與 pwMzT 相同）。
   新稿名稱：F28-D-v2 · 批次生成 — 尚未設定金鑰。不要動 pwMzT 本身。

2. 在新稿的內容區（pwMzT 內 body「RTmxa」的複本）最前面、summary-block 之前，插入一塊提示：
   直接複製 F5-D-v2 裡的 warning-panel（節點 RrbLx）到這個位置，然後改它裡面的三段文字：
   - 標題（原「語音辨識尚未設定」）改成：字幕生成尚未設定
   - 說明（原「生成字幕需要雲端語音辨識（ASR）金鑰。請至金鑰設定儲存後即可使用。」）改成：
     批次產生字幕需要翻譯（Claude）與語音辨識（ASR）金鑰。到金鑰設定儲存後就能開始；清單可以先看。
   - 按鈕文字維持：前往設定
   warning-panel 的寬度 fill_container，外觀（$warning-tint 底、圓角、內距、圖示）保持和 RrbLx 一樣。
   注意：複製一般節點後，子節點會換新 id，請複製完再用名稱找到三段文字各自修改。

3. 底列（rrOWa 的複本）：
   - 右邊「開始產生」按鈕（iK0BE 的複本，Component/Button/Primary）改成停用樣式：整顆 opacity 0.5，文字維持「開始產生」。
   - 預算區的提示文字（twwOH 的複本，原「達到上限會自動暫停，可稍後續跑」）改成：尚未設定金鑰，無法開始
     顏色改 $warning-text。

4. 其他（清單、勾選、預估金額）保持和 f15-d-v2 一樣——意思是清單可以先看、先勾，只是不能開始。
5. 確認沒有任何子節點被裁切，然後存檔（File ▸ Save）。
```

## Dev Notes

- 探測失敗或舊後端（沒有 `available`）一律當作可用——鎖按鈕的前提是「確定不可用」。
- 提示放在清單**上方**、按鈕停用原因放在**按鈕旁**：前者一打開就看得到，後者回答「為什麼按不了」。
- 不要改 503 的既有處理（真的按下去仍然得到既有錯誤）。
- 工作區（`GenerationWorkspaceV2`）的「批次產生」入口開的是同一個對話框，本張不另外改工作區畫面。

### Time-dependent visual coverage

- N/A — no wall-clock-reading components touched.

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（Amelia）

### Completion Notes List

- 設計：Alexyu 於 2026-09-29 畫好 F28-D-v2（`N6dxid`，commit `7f9377b8`）。Sally MCP 唯讀核對：`warning-panel`（`aOLpr`，`$warning-tint`、fill_container）三段文字、`btn-start` opacity 0.5、`budget-hint`「尚未設定金鑰，無法開始」`$warning-text`，全部與提示詞一致；SCREENS 已加 `N6dxid`。
- RED → GREEN：Go `TestGetGenerationBatchStatus_ReportsUnavailable`＋Idle 鍵集含 `available`；面板 3 條、對話框 4 條（含「先 false、關閉、重開探測失敗 → 不擋」）先紅後綠。
- 對抗式 review（subagent）：無 High／Med。修掉 3 條 Low——關閉時把「尚未設定」歸零（否則重開遇探測失敗會沿用舊值，違反 AC #3）、補上會抓到它的測試（原測試從初始值起跑，空洞）、停用按鈕用 `aria-describedby` 指向原因。1 條文案 Low 記 Discovery Triage。
- 全量：`pnpm nx test web` 291 檔／4,382 條綠；`pnpm nx test api` 綠；`lint:all`、`web:typecheck` 綠。
- 🔗 AC Drift: NONE。
- 📎 Contract Stamps: `available` on `GET /subtitles/generation-batch/status` `[@contract-v1]`（handler、`subtitleService.ts` 型別註解）。
- 🎭 A11y Pre-Flight: PASS（前往設定是真按鈕；停用原因以 `aria-describedby` 綁到按鈕）。
- 🎨 UX Verification: PASS — 畫廊 `generation-consent/not-configured` 與 F28-D-v2 比對：提示位置、樣式、三段文字、停用鈕與原因一致。差異：設計稿「使用：Claude（你的金鑰）」是從 f15 原樣帶來，實作依真實狀態顯示「（尚未設定金鑰）」——實作較正確，不改。夾具只放 3 列（多一塊提示後 5 列會超過 800 高的截圖視窗）。

### Discovery Triage

- ③ — 提示說明寫「需要翻譯（Claude）與語音辨識（ASR）金鑰」，但用自架語音辨識（`selfHostedAsr`）時其實只需要 Claude 金鑰；文案是本張指定的，要不要依模式分兩種說法需 Sally／Alexyu 決定 → `disc-2026-09-not-configured-copy-self-hosted-asr`（P4）。

### File List

- apps/api/internal/handlers/generation_batch_handler.go、generation_batch_handler_test.go
- apps/web/src/services/subtitleService.ts
- apps/web/src/components/subtitle/GenerationBatchDialogV2.tsx、GenerationBatchDialogV2.spec.tsx
- apps/web/src/components/subtitle/consent/GenerationConsentView.tsx
- apps/web/src/components/subtitle/consent/CandidateListPanel.tsx、CandidateListPanel.spec.tsx
- apps/web/src/components/activity/ActivityHub.tsx
- apps/web/src/components/library/LibraryBrowseV2.tsx
- apps/web/src/routes/test/-gallery.fixtures.tsx
- tests/visual/components.visual.spec.ts-snapshots/components/generation-consent/not-configured/default-visual-darwin.png（新）
- _bmad-output/implementation-artifacts/disc-2026-09-batch-generation-no-asr-key-warning.md、sprint-status.yaml

## Change Log

| Date       | Change                                                    |
| ---------- | --------------------------------------------------------- |
| 2026-09-29 | create-story（SM Bob）＋ Sally 的 Pencil 提示詞。          |
| 2026-09-29 | 設計稿 F28-D-v2（Alexyu）；dev-story（Amelia）＋對抗式 review → review。 |
| 2026-09-29 | PR #600 合併（`-linux` 基準 #601）→ done。 |
