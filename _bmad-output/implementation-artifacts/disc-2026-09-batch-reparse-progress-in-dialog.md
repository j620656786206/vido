# Story: disc-2026-09-batch-reparse-progress-in-dialog — 批次重新解析的對話框顯示比對進度

Status: ready-for-dev（等設計稿 C24-D 畫好、Sally 審過才動工）

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

- [ ] Task 0 — 設計稿 C24-D（Alexyu 跑提示詞 → Sally MCP 唯讀 review → 截圖、`SCREENS`、commit）
- [ ] Task 1 — `useEnrichmentRefresh` → 回傳 `{ phase: 'queued'|'running'|'done', progress, result }`，同時聽兩個事件（AC #1–#3）
- [ ] Task 2 — `BatchProgress` 新增 `matching` prop 與三態畫面（AC #1, #2, #5, #7）
- [ ] Task 3 — `LibraryBrowseV2` 接線（AC #1, #4）
- [ ] Task 4 — 測試四層與夾具（AC #6）

## 設計交付（給 Pencil inline agent 的提示詞）

見 `_bmad-output/implementation-artifacts/disc-2026-09-batch-reparse-progress-in-dialog.pen-prompt.md`。

## Dev Agent Record

（待填）
