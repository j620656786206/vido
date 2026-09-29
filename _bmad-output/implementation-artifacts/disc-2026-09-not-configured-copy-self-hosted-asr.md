# Story: disc-2026-09-not-configured-copy-self-hosted-asr — 「尚未設定」提示只點名真正缺的那把金鑰

Status: review

<!-- SM Bob create-story 2026-09-29。Alexyu 裁定 B：「分兩種說法」。
     行號為 main `dc9b0ec6`；.pen 節點 id 由 SM 本次用 Pencil MCP 唯讀讀取。 -->

## Story

身為沒設好金鑰就打開「產生字幕」的人，
我要提示只告訴我**真正缺的那一把**金鑰，
而不是兩把都點名，讓我去設一把其實不需要的。

## 背景（查到的事）

- 立案時以為分界是「自架語音辨識」。實際查碼後，**分界是伺服器的生成模式**：
  - **pipeline 模式**（NAS 正式環境開著這個）：`available` 只看 Claude 金鑰——`cmd/api/main.go:745` `subtitleCapabilityGate := keyResolver.Has(…, services.KeyClaude)`，`:1021-1028` 把它交給 `pipelineGenerationRunner`；`cmd/api/generation_batch_runner_adapter.go:52-62` 註解：沒有 ASR 的部署**照樣可用**（抽取＋翻譯能跑，ASR 那段逐片降級）。所以這個模式下「不可用」＝缺 Claude 金鑰，跟 ASR 金鑰、自不自架都無關。
  - **legacy 模式**：`main.go:1020` 用 `RouteCGenerationRunner`，`services/generation_batch_runner.go:28-31` → `services/transcription_service.go:452-462`：只看 FFmpeg＋ASR 金鑰。FFmpeg 內建在 Docker 映像檔，所以實務上「不可用」＝缺 ASR 金鑰。
- 現在的提示（#600）一律寫「需要翻譯（Claude）與語音辨識（ASR）金鑰」（`apps/web/src/components/subtitle/consent/CandidateListPanel.tsx:828`）——兩種模式都多點名了一把。
- 狀態端點 `GET /subtitles/generation-batch/status` 只回 `available`（`handlers/generation_batch_handler.go:228`），前端不知道是哪一把。handler 建構在 `main.go:1038`，模式在 `:1021` 已知（`subtitlePipeline != nil`）。
- 設計稿 F28-D-v2（`N6dxid`）的說明文字節點是 `pMCK7`（`wp-sub`）；說明上方標題 `cap-F28-D-v2`（`Uk8ZM`）。

## Acceptance Criteria

1. **後端 `[@contract-v1]`**：status 在 `available: false` 時多回 `missing_key`：pipeline 模式 `"claude"`、legacy 模式 `"asr"`。`available: true` 時不回。既有欄位不變。
2. **前端**：提示說明依 `missing_key` 換句：
   - `"claude"` →「批次產生字幕需要翻譯（Claude）金鑰。到金鑰設定儲存後就能開始；清單可以先看。」
   - `"asr"` →「批次產生字幕需要語音辨識（ASR）金鑰。到金鑰設定儲存後就能開始；清單可以先看。」
   - 沒有這個欄位（舊後端）或其他值 → 維持現在兩把都點名的句子。
   - 標題、「前往設定」、停用原因不變。
3. 測試：Go — pipeline／legacy 各自回對的 `missing_key`、可用時不回。前端 — 三種說明各一條。先紅後綠。
4. 視覺：畫廊 `generation-consent/not-configured` 改用 `"claude"`（正式環境的情況），更新 `-darwin` 基準、刪掉舊 `-linux` 讓 CI 重產；與設計稿比對。

## Tasks / Subtasks

- [x] **Task 0（設計，Alexyu 執行）**— 把下方提示詞貼給 Pencil 的 Inline AI Agent；⌘S；`python3 scripts/export-pen-screenshots.py`；回報完成。
- [x] Task 1 — 後端 `missing_key`＋測試（AC #1, #3）
- [x] Task 2 — 前端換句＋測試（AC #2, #3）
- [x] Task 3 — 畫廊夾具與基準；全量測試、lint、typecheck（AC #4）

## Sally 的 Pencil 提示詞（Task 0，整段貼給 Inline AI Agent）

```
在 ux-design.pen 修改設計稿「F28-D-v2 · 批次生成 — 尚未設定金鑰」（節點 N6dxid）的一段文字。

1. 找到 N6dxid 裡的說明文字節點 pMCK7（名稱 wp-sub，目前內容是「批次產生字幕需要翻譯（Claude）與語音辨識（ASR）金鑰。到金鑰設定儲存後就能開始；清單可以先看。」）。
   把內容改成：批次產生字幕需要翻譯（Claude）金鑰。到金鑰設定儲存後就能開始；清單可以先看。
   只改文字，字級、顏色、寬度都不要動。

2. 找到畫面上方的標題文字 Uk8ZM（名稱 cap-F28-D-v2，目前是「F28 · 批次產生字幕 — 尚未設定金鑰（桌面）」）。
   改成：F28 · 批次產生字幕 — 尚未設定金鑰（桌面；舊版語音辨識模式時說明改成「需要語音辨識（ASR）金鑰」）
   確認它不會和右邊或下方的東西重疊；太長的話讓它換行，並把它往上移到不碰到 N6dxid 為止。

3. 其他都不要動。存檔（File ▸ Save）。
```

## Dev Notes

- 用「模式」決定缺哪一把，不去問兩把金鑰各自在不在：`available` 的判斷本來就只看一把（依模式），`missing_key` 必須跟它同一個來源，才不會說錯。
- 模式在開機時就固定（`cfg.SubtitlePipelineEnabled()`），所以 handler 在建構時設定一次即可。
- 不改 503 的既有訊息（按下去仍是既有錯誤）。

### Time-dependent visual coverage

- N/A — no wall-clock-reading components touched.

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（Amelia）

### Completion Notes List

- 設計：Alexyu 畫好（commit `3af86bb5`）。Sally MCP 唯讀核對：`pMCK7` 改成只講 Claude 金鑰、`Uk8ZM` 標題註明舊模式說法，其餘未動。
- RED → GREEN：Go `TestGetGenerationBatchStatus_MissingKey`（4 種情況）、`TestGenerationGateKeyFor`；前端三種說明各一條、對話框傳遞 `missingKey`、`available:true` 帶 key 時忽略、關掉再開清掉 key。
- 對抗式 review（subagent）：無 High／Med。採納：模式→金鑰抽成 `GenerationGateKeyFor` 並加測試（原本只在 `main.go` 的 if 裡，寫反測試抓不到）；補兩條前端測試；FFmpeg 限制寫進註解。
- 已知、不修：legacy 模式兩把都沒設時，提示只講 ASR；補了 ASR 之後可以開始，但因為沒有 Claude 金鑰會跳過翻譯、拿到原文字幕（不會再報錯）。正式環境跑 pipeline 模式，影響小。
- 全量：`pnpm nx test web` 291 檔綠；`pnpm nx test api` 綠；`lint:all`、`web:typecheck` 綠。
- 🔗 AC Drift: FOUND — 立案時的分界「自架 ASR」更正為「生成模式」（見背景）。
- 📎 Contract Stamps: `missing_key` on `GET /subtitles/generation-batch/status` `[@contract-v1]`。
- 🎭 A11y Pre-Flight: PASS（只換文字）。
- 🎨 UX Verification: PASS — 畫廊 `generation-consent/not-configured`（`claude`）與 F28-D-v2 說明文字一致。

### Discovery Triage

- 無新立案。（review 提到單片「管理字幕」的 ASR 提示一律講 ASR 金鑰——單片那條路確實需要 ASR，不算錯。）

### File List

- apps/api/internal/handlers/generation_batch_handler.go、generation_batch_handler_test.go
- apps/api/cmd/api/main.go
- apps/web/src/services/subtitleService.ts
- apps/web/src/components/subtitle/GenerationBatchDialogV2.tsx、GenerationBatchDialogV2.spec.tsx
- apps/web/src/components/subtitle/consent/GenerationConsentView.tsx
- apps/web/src/components/subtitle/consent/CandidateListPanel.tsx、CandidateListPanel.spec.tsx
- apps/web/src/routes/test/-gallery.fixtures.tsx
- tests/visual/…/generation-consent/not-configured/default-visual-darwin.png（更新）
- _bmad-output/implementation-artifacts/disc-2026-09-not-configured-copy-self-hosted-asr.md、sprint-status.yaml

## Change Log

| Date       | Change                                                             |
| ---------- | ------------------------------------------------------------------ |
| 2026-09-29 | create-story（SM Bob）＋ Sally 的 Pencil 提示詞；分界更正為「模式」。 |
| 2026-09-29 | 設計稿（Alexyu）；dev-story（Amelia）＋對抗式 review → review。 |
