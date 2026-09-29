# Story: disc-2026-09-batch-reparse-progress-in-dialog — 批次重新解析的對話框顯示比對進度

Status: done

## Story

身為在媒體庫勾了幾部片按「重新解析」的人，我要對話框告訴我比對跑到第幾部、成功幾部、失敗幾部，
而不是只有一句「比對在背景進行」，讓我知道要不要繼續等、有沒有東西壞掉。

## 背景（查到的事，附親自讀過的位置）

- #574 起批次重新解析真的會跑比對（`EnrichmentService.RequestRun`）。後端每 5 部送一次 `enrich_progress`
  SSE（`enrichment_service.go` `EnrichmentProgress`：`total`／`processed`／`succeeded`／`failed`／`skipped`／
  `current_title`／`is_active`），結束送 `enrich_complete`（`EnrichmentResult`：`total`／`succeeded`／`failed`／`skipped`／`duration`）。
- 前端 `hooks/useEnrichmentRefresh.ts` 只聽 `enrich_complete` → `invalidateQueries(libraryKeys.all)`；
  `components/library/BatchProgress.tsx` 完成態多一行 `note`；`LibraryBrowseV2.tsx` 在 `successCount > 0` 時開監聽。
- ⚠️ `total` 是**整輪 pending 的數量**，可能大於這次勾的 N（掃描留下的 pending 列在同一輪）。
- ⚠️ 排在別人後面的那一輪：第一個 `enrich_complete` 是前一輪的；之後會再來 `enrich_progress`。
- Rule 23：畫面上的時間／數字只能來自事件，不讀瀏覽器時鐘。
- 設計稿：C2-D（`dcf67`）只畫了工具列，沒畫對話框 → 新增規格稿 **C24-D**（三態並排；DialogFrame `m6KMPr` ＋ ProgressBar `TzhFY` 母版）。
  提示詞在下方〈設計交付〉，由 Alexyu 貼給 Pencil inline agent（feedback_pen_inline_agent_workflow）。

## Party-mode 裁定（2026-09-29：John／Winston／Sally／Bob／Amelia）

- **John**：只做對話框內的進度，不進活動中心、不做通知；比對中可隨時關閉。
- **Winston**：`useEnrichmentRefresh` 擴成同時聽 `enrich_progress`，回傳進度；完成仍 invalidate。完成後再收到進度要能回到「比對中」。
- **Sally**：三態文案——①「重新解析中／已排入比對 N 項，等待開始…」②「比對中／本輪整理 T 部（含你勾的 N 部）」＋進度條「目前：片名｜p / T」＋「成功 s・失敗 f・略過 k」③「比對完成／成功 s・失敗 f — 清單已更新」＋原有失敗清單。總數文案**不可**寫成「你勾的 T 部」。手機同一個對話框，不另外畫。
- **Bob**：AC 五條、測試四層、一個 PR。

## Acceptance Criteria

1. 送出且 `successCount > 0`：對話框標題「重新解析中」，副標「已排入比對 N 項，等待開始…」；第一筆 `enrich_progress` 到達後切成「比對中」：副標「本輪整理 T 部（含你勾的 N 部）」、進度條（`processed / total`）、「目前：{current_title}」、「成功 s・失敗 f・略過 k」。
2. `enrich_complete`：標題「比對完成」，副標「成功 s・失敗 f — 清單已更新」；清單自動重抓（既有行為）；原本的失敗清單（`errors`）照舊顯示。
3. 完成後若再收到 `enrich_progress`，回到 AC #1 的「比對中」。
4. `successCount === 0`：維持現況（「操作完成」，無三態、不開 SSE）。
5. 三態下都只有「關閉」一顆鈕，隨時可關；關閉後不再顯示進度，清單仍靠 `enrich_complete` 更新。
6. 測試：hook spec（MockEventSource 送 progress／complete／再 progress）、`BatchProgress.spec`（三態）、`LibraryBrowseV2.spec`（進度接線）、gallery 夾具新增三態（`-linux` 基準走 CI bootstrap）。
7. `BatchProgress.tsx` 檔頭 `Design ref` 改指 C24-D 的節點 ID；Rule 21 lint 通過。

## Tasks / Subtasks

- [x] Task 0 — 設計稿 C24-D（Alexyu 跑提示詞 → Sally MCP 唯讀 review → 截圖、`SCREENS`、commit）— PR #578
- [x] Task 1 — `useEnrichmentRefresh` → 回傳 `{ phase: 'queued'|'running'|'done', progress, result }`，同時聽兩個事件（AC #1–#3）
- [x] Task 2 — `BatchProgress` 新增 `matching` prop 與三態畫面（AC #1, #2, #5, #7）
- [x] Task 3 — `LibraryBrowseV2` 接線（AC #1, #4）
- [x] Task 4 — 測試四層與夾具（AC #6）

## 設計交付（給 Pencil inline agent 的提示詞）

見 `_bmad-output/implementation-artifacts/disc-2026-09-batch-reparse-progress-in-dialog.pen-prompt.md`。

## Dev Agent Record

### Completion Notes List

- **hook**：`useEnrichmentRefresh(enabled, runId)` 回傳 `EnrichmentMatching | null`（queued／running／done），同時聽 `enrich_progress` 與 `enrich_complete`；完成仍 `invalidateQueries(libraryKeys.all)`。壞掉的 payload 忽略不炸。`runId` 是自審抓到的 bug：第二次重新解析時 watch 已經是 on，`false→true` 被 React 合併成沒變，會一直顯示上一輪的「比對完成」——改成每次 +1 讓 effect 重連並回到「已排入」。
- **對話框**：`BatchProgress` 新增 `matching` prop，三態照 C24-D：標題「重新解析中／比對中／比對完成」、副標「已排入比對 N 項，等待開始…／本輪整理 T 部（含你勾的 N 部）／成功 s・失敗 f — 清單已更新」、進度條 `processed/total`、「目前：片名｜p / T」、「成功 s・失敗 f・略過 k」。`isComplete` 之前（請求還在飛）忽略 `matching`。拿掉上一個 PR 暫時加的 `note` prop。
- **媒體庫頁**：`batchProgress.kind` 記住這次是哪種批次，只有 `reparse` 且 `successCount > 0` 才把 `matching` 傳進去（批次刪除時就算背景有比對也不會顯示）；`current` 改記實際排到的數量。
- **測試**：hook 4、對話框 13（含三態 4 條）、媒體庫頁 34（新增「刪除不顯示比對」）、路由 7，全綠；lint:all、typecheck 綠。
- **視覺**：畫廊新增 3 個夾具（`library-batch-progress-matching-{queued,running,done}`），darwin 基準本機產出，逐張與 `c24-d.png` 對過：文案、進度條比例、失敗清單位置一致。`-linux` 走 CI bootstrap。
- **沒做**：失敗清單顯示的是列 id 不是片名（後端 `BatchError` 只有 id，既有行為；設計稿畫的是片名）→ 另立 `disc-2026-09-batch-error-shows-id-not-title`。
