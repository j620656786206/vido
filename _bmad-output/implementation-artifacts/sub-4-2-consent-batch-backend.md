# Story 4.2: 同意後批次執行 —— 混合 id 清單、影集、使用者核准預算上限（後端寫側）

Status: ready-for-dev

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a NAS owner,
I want the batch I confirmed on the cost screen to run exactly as quoted — the items I picked (movies and episodes alike), down the cheapest honest route, stopping at the ceiling I approved,
so that the amount I consented to on F16 is the amount the system is actually allowed to spend.

## Context — 這個 story 為什麼存在

sub-4-1（done）交付了 BE「讀側」：候選清單、路線預測、成本估算。**本 story 是 BE「寫側」**——使用者在 F15 勾選、在 F16 按下「確認並開始」之後，後端要照著同意的內容執行。FE 篩選畫面是 **sub-4-3**。

三個裁定約束本 story：

1. **Alexyu 2026-08-07 三件一體裁定**（`backlog-pipeline-cost-consent`）：批次帶總預算上限（「基建現成——9R-16 ctx-attached shared Budget + sub-3-1 的 ErrBudgetExceeded→pause 語意，管線端已支援」），且既有 9R-16 preview/confirm 是可延伸基底「**但那套目前驅動 Route C 引擎，需 rewire 到 D2 pipeline**」。
2. **D1 裁定（Alexyu 2026-08-10，design-prompt §5-quinquies）**：批次要支援影集。驗收原文：「混合電影＋影集的批次可成功送出並各自寫回正確資料表；不再出現『任一 id 非電影就整批 400』」。
3. **sub-4-1 的守衛**：`internal/cost_consent_test.go` 禁止任何生產程式呼叫 `.EnqueueMissing(`（整庫 sweep）。知情批次必須走**明確 id 清單**，不是 sweep。

今日寫側的三個洞（agent 盤點確認）：

- `Start(ctx, scope, mediaIDs)` **不收預算參數**（`generation_batch.go:185`）；上限來自建構時固定的 `p.budgetUSD`（`:94`，env `AI_RUN_BUDGET_USD`，`config.go:133` 預設 5.0）——使用者核准的數字無處進線。
- `generationCandidateFinder`（`:80-84`）三個方法全是 movie-repo 形狀；`collectItems` scope=selected 對非電影 id 的 400 是「movies 表查不到」的副作用而非型別檢查（`:249-266`）；`GenerationBatchItem`（`:63-69`）**沒有 MediaType 欄位**。
- `process` 迴圈呼叫 `RunTranscription(..., WithTranslation())` **未帶 `WithMediaType`**（`:323`）——預設是 movie（`transcription_service.go:344-350`），影集項目會**寫錯表**；且 Route C = 一律付費 ASR，可抽取項目在 F15 被報成近乎免費，寫側若照舊執行，同意畫面的報價就是謊言。

## Acceptance Criteria

1. **使用者核准的預算上限進線。** `POST /subtitles/generation-batch` request 新增選填欄位 `budget_usd`（float）：
   - 未提供 → 沿用 `cfg.AIRunBudgetUSD`（既有 FE 零行為變化）。
   - 提供則必須 > 0，否則 400 `VALIDATION_INVALID_FORMAT`。**絕不**把使用者輸入映射為「無上限」——`ai.NewBudget(<=0)` 的語意是 unlimited（`budget.go:93`），一個手滑的 0 不可以變成不設防。
   - `Start()` 收預算參數（新參數或 request-scoped struct 皆可），`generation_batch.go:213` 的 `ai.NewBudget(p.budgetUSD)` 改用每批次核准值；`p.budgetUSD` 降級為「未提供時的預設」。SSE 的 `budget_usd`/`spent_usd` 讀 `budget.Snapshot()`（`:395-416`），自動跟著正確。

2. **明確 id 清單支援混合電影＋影集（D1）。** scope=selected 的 id 解析改雙來源：先 `MovieRepository.FindByID`，查無再 `EpisodeRepository.FindByID`（`episode_repository.go:107`）；兩邊都查無才拒絕。**維持 reject-not-filter**：任一 id 無效仍整批 400（同意畫面的誠實性——送出的清單就是 F16 確認的金額對應的清單，默默丟項等於執行另一個沒被同意過的批次）。`GenerationBatchItem` 新增 `MediaType`（`models.SubtitleRunMediaMovie|Episode` 內部詞彙，同 FR12 的 movie|episode，**非** TMDB 的 movie|tv）；202 回應 items 與 SSE 的對應欄位為 additive 變更。影集無 `file_path` 者比照電影現行規則處理（selected → reject）。

3. **scope=missing 維持 movies-only（明確不變）。** 既有 `GET .../preview`（`{total_items}`）是 movies-only 且 sub-4-1 已決定原樣保留；若 missing 批次擴影集而 preview 計數不擴，舊 FE 的「缺字幕 N」會與實際批次數矛盾。影集進批次一律走明確 id 清單（sub-4-3 的 F15 本來就是逐項勾選後送 id）。於 handler 註解 + Swagger 註記這個刻意的不對稱。

4. **執行路線誠實（pipeline mode）：批次改騎 D2 pipeline。** `generationRunner` port（`generation_batch.go:73-76`）改為攜帶 media type 的執行 port（**Rule 19**：services 不可 import subtitle，port 用 services 側詞彙；`cmd/api` 新增 adapter 包 `subtitlePipeline.ProcessItem(ctx, subtitle.MediaRef{ID, MediaType}, subtitle.ProcessItemOptions{Force: false})`——比照 `pipelineASRAdapter`／`route_predictor_adapter.go` 先例）。pipeline mode（`cfg.SubtitlePipelineEnabled()`）注入 D2 adapter，可抽取項目走免費抽取＋翻譯、`no_text_source` 才落 ASR（sub-3-1/3-2 已完工的 leg）。
   **批次直接循序呼叫 `ProcessItem`，不走 WorkerPool 佇列。** 理由：批次要的是單飛＋共享預算＋可取消＋逐項進度，pool 佇列一項都給不了（無 batch 身分、無佇列移除＝cancel 做不到、ctx 是行程全域掛不了 per-batch budget、2 worker 亂序）。pool 與 FR12 端點**原樣保留**（`ProcessItem` 本就設計給 2 個 worker 並發呼叫，多一個循序呼叫者安全）。`cost_consent_test.go` 維持綠燈——它禁的是 `.EnqueueMissing(` 呼叫點，明確 id 的 `ProcessItem` 不在禁區。`main.go:631-633` 的註解（「sub-4-2's consented batch will too [drive the pool]」）是 sub-4-1 實作者的預期而非裁定，**同步修正為實情**（comment-only）。

5. **legacy mode 回退：維持 Route C 引擎（含影集修正）。** legacy 下 `subtitlePipeline` 為 nil（`main.go:573-596`），無 D2 可騎；批次維持注入 `transcriptionService` 執行器，但呼叫補上 `WithMediaType(item.MediaType)`（sub-3-2 已把寫回與 resume 依型別分派做好；漏傳的預設值是 movie ⇒ 影集寫錯表）。兩種 mode 的 runner 由 `main.go` 依 `cfg.SubtitlePipelineEnabled()` 選擇注入，processor 本身 mode-agnostic。除 `WithMediaType` 與預算參數外，legacy 行為與今日一致。

6. **預算上限＝共享單一 `ai.Budget`，`ErrBudgetExceeded`＝暫停非失敗。** 沿用既有機制：批次 detached ctx 以 `ai.WithBudget` 掛共享 budget（`generation_batch.go:214-215`）、逐項前置 `Exceeded()` 檢查 → `budget_ceiling` 終態＋`paused_count`（`:306-310`）、runner 錯誤 `errors.Is(err, ai.ErrBudgetExceeded)` → 同樣歸類暫停（`:335`）。**新增驗證義務**：D2 路徑的兩條腿都必須把花費記到同一顆批次 budget——翻譯 LLM 走 `governed()` 的 ctx budget 前置短路（`governor.go:70-73`），ASR 腿走 sub-3-1 `pauseASRItem` 的包裝 sentinel（`process_item.go:392,427`；run 記 failed＋訊息、media row 不動以保 resume 標記）；**adapter 不可吞掉可 `errors.Is` 分類的錯誤鏈**。已知限制不變：**軟上限**（`Exceeded()` 用 `>=` 且在呼叫前檢查，實際花費可能略超）——文案與 Swagger 不可承諾「絕不超過」。暫停後重新送同清單可續跑（未完成項目仍缺 zh-Hant ⇒ 仍是候選；pipeline 側 resume-aware）。

7. **單飛、取消、進度、錯誤碼行為保留。** 409 `TRANSCRIPTION_BATCH_RUNNING` 單飛、`POST /cancel`、SSE `generation_batch_progress` 全保留；新增欄位一律 additive。**Rule 7 零新前綴、零新碼**（`VALIDATION_`/`TRANSCRIPTION_` 現有碼夠用）。Rule 20：`generation_batch_progress`／202 回應是 9R-16 的 stamped 面（FE ux3-subtitle-v2-batch acked，已 done＝frozen）——additive 欄位是否 bump 由 dev-story 的 Contract Stamp Check 裁定並記 Change Log；`transcription_*` SSE [@contract-v2]、D2 媒體狀態 [@contract-v3]、D6 `PipelineStage`：**皆不動**。

8. **測試。** 至少涵蓋：(a) 混合批次 1 movie＋1 episode → 各自寫回正確資料表（movies vs episodes 的 subtitle 欄位）；(b) episode id 於 scope=selected 不再 400、未知 id 仍整批 400；(c) `budget_usd` 提供→蓋過 env 預設（斷言 budget 以 request 值建構）、未提供→env 預設、<=0→400；(d) pipeline mode：可抽取項目**不觸發 ASR**（route-honest 的核心斷言）；(e) legacy mode：除 `WithMediaType`/預算參數外行為不變（既有 16 個 generation_batch 測試全綠或僅機械更新）；(f) 中途觸頂：已完成保留、`budget_ceiling`＋`paused_count` 正確、runner 回傳的 `ErrBudgetExceeded` 被歸類為暫停非失敗；(g) 單飛 409 與 cancel 不變；(h) `cost_consent_test.go` 綠燈。

## Tasks / Subtasks

- [ ] **Task 1 — 預算參數進線（AC: #1）**
  - [ ] `GenerationBatchStartRequest` 加 `budget_usd`（選填、>0 驗證、400 路徑）；handler → `Start` 傳遞
  - [ ] `Start` 簽名改造；`ai.NewBudget` 改吃每批次值；`p.budgetUSD` 明確註解為 default-only
  - [ ] Swagger 註解同步（含軟上限措辭）

- [ ] **Task 2 — 混合 id 解析＋MediaType（AC: #2, #3）**
  - [ ] finder port 擴為雙來源（movie `FindByID` ＋ episode `FindByID`；`main.go` 注入 `repos.Movies`＋`repos.Episodes`）
  - [ ] `GenerationBatchItem.MediaType`＋episode 的 `toItem` 對應（title 用 episode 自身欄位組 `SxxEyy`／fallback，**不加 series join**——SSE `current_item` 是輔助顯示，sub-4-3 的清單自帶完整標題）
  - [ ] reject-not-filter 保留；scope=missing movies-only 註記（handler 註解＋Swagger）

- [ ] **Task 3 — 引擎接線：D2 adapter＋mode 選擇（AC: #4, #5）**
  - [ ] runner port 改帶 media type；`cmd/api/generation_batch_runner_adapter.go`（新檔）包 `subtitlePipeline.ProcessItem`
  - [ ] legacy 分支：`transcriptionService` runner 補 `WithMediaType`
  - [ ] `main.go` 依 `cfg.SubtitlePipelineEnabled()` 注入對應 runner；`:631-633` 註解修正（comment-only）
  - [ ] 確認 pool／FR12／`cost_consent_test.go` 零觸動

- [ ] **Task 4 — 預算共享與暫停分類驗證（AC: #6）**
  - [ ] 測試：批次 budget 經 ctx 傳入後，翻譯與 ASR 的花費記到同一顆 budget（`RecordLLM`/`RecordASR` 同源斷言）
  - [ ] 測試：adapter 保留 `ai.ErrBudgetExceeded` 的 `errors.Is` 鏈；觸頂 → `budget_ceiling` 暫停語意

- [ ] **Task 5 — 契約清點與文件（AC: #7）**
  - [ ] AC Drift Check＋Contract Stamp Check（dev-story Step 2 強制項）：additive 欄位的 bump 裁定＋Change Log
  - [ ] 若 SSE payload 有新欄位：`docs/sse-event-types.md`＋`.zh-TW.md` 同步（Rule 17）
  - [ ] `docs/deployment.md` 補「pipeline mode 下批次走抽取優先路線」行為變更說明

- [ ] **Task 6 — 測試與回歸（AC: #8）**
  - [ ] AC #8 的 8 類案例全數落地（RED→GREEN 逐項）
  - [ ] 全回歸閘門：`pnpm nx test api`＋`pnpm nx test web`＋`pnpm run lint:all`＋gofmt/vet

（後端 task 6 個、前端 0 個 —— 未觸發跨端拆分門檻。）

## Dev Notes

### 既有可重用零件（不要重造）

| 需求 | 現成零件 |
| --- | --- |
| 批次單飛／取消／SSE 進度／budget_ceiling 終態 | `GenerationBatchProcessor` 全套（`generation_batch.go:90-416`）——本 story 是改造不是重寫 |
| 影集缺字幕查詢 | `EpisodeRepository.FindByID` `:107`／`FindMissingZhHantSubtitle` `:160`（後者本 story 用不到——missing 不擴） |
| 媒體型別感知寫回＋resume | sub-3-2 的 `WithMediaType`（`transcription_service.go:362`）＋ episode writer/reader ports（`:44,53`） |
| D2 路線引擎（抽取 > ASR、episode 已支援） | `subtitle.Pipeline.ProcessItem`（`batch.go:102`、`process_item.go:30`）＋ `MediaRef{ID, MediaType}`（`pipeline.go:75`，[@contract-v1]） |
| 共享預算＋前置短路＋暫停 sentinel | `ai.Budget`/`WithBudget`/`Exceeded`（`budget.go`）、`governed()`（`governor.go:68-73`）、`ai.ErrBudgetExceeded`（`types.go:33`）、`pauseASRItem`（`process_item.go:427`） |
| Rule 19 跨界 adapter 先例 | `cmd/api/asr_adapter.go`（sub-3-2）、`cmd/api/route_predictor_adapter.go`（sub-4-1） |
| FR12 明確 id 進管線的形狀參考 | `subtitle_pipeline_handler.go:59-65`（`media_type oneof=movie series episode` 內部詞彙） |

### 架構裁決記錄（為什麼是這個形狀）

- **為何直呼 `ProcessItem` 不走 pool**：AC #4 已列四個理由。補充：走 pool 需要發明 batch 身分、佇列移除、per-item budget 選項與完成回呼——全是新機制；直呼只重用既有循序迴圈。
- **為何 mode-dependent runner**：legacy 下 pipeline 不存在（nil）；既有 FE 對話框在 legacy 也要能跑批次。強行統一等於在 legacy 重建半個 pipeline。
- **為何 reject-not-filter**：同意畫面的清單＝金額的對應物。任何默默過濾都會讓實際執行偏離 F16 確認的內容。
- **為何 missing 不擴影集**：preview 端點凍結（sub-4-1 AC #7）；擴了批次不擴計數＝舊 FE 自相矛盾。新流程根本不用 missing scope。
- **執行順序**：依提交順序循序處理，不重排。FE（sub-4-3）想要「免費先做」可以自行把 extract 路線的 id 排前面——後端不猜。

### seam 資料層觸及（retro-m2-AI3 模板義務）

本 story 騎的每條 seam 的實際資料層範圍：

- `GenerationBatchProcessor.finder`：今日 3 方法全查 `movies` 表；改造後 movie 路徑不變、episode 路徑查 `episodes` 表（`FindByID`）。
- `TranscriptionService.RunTranscription`（legacy runner）：movie → `movies.UpdateSubtitleStatus`；episode → `episodes.UpdateEpisodeSubtitleStatus`；resume 讀取同型別分派（sub-3-2 完工，`main.go:528-536` 四個 setter 已接）。
- `subtitle.Pipeline.ProcessItem`（D2 runner）：`subtitle_runs`（RunStore）、`movies`/`episodes`（MediaStore——`NewMediaStore(repos.Movies, repos.Series, repos.Episodes)`，`main.go:580`）、`cache_entries`（SegmentCache）；ASR 腿經 `pipelineASRAdapter` 回到 TranscriptionService＝**同一個寫者**，無雙寫。
- `ai.Budget`：純記憶體，零 DB；行程重啟即歸零（暫停後續跑＝新批次新 budget，符合 F18「可提高上限或稍後續跑」）。

### 已知限制（記錄，不在本 story 解）

- **軟上限**：`Exceeded()` 用 `>=` 且呼叫前檢查，單一超大呼叫可略超上限。UI/文件措辭已在 sub-4-1 統一為「達到上限會自動暫停」。
- **估價是下界**：F15 的 extract 估價在 SDH 過濾歸零時實際會落 ASR（`router.go:123-130`）——這正是 ceiling 存在的理由：估價會錯，上限不會。
- **batch 與 pool 的去重互不知情**：批次直呼 `ProcessItem` 的項目不在 pool 的 `inFlight`，FR12 同項並發可能重複處理（今日 Route C batch 與 FR12 之間**本來就有**同類競態，非本 story 引入）→ lane ③ `backlog-batch-pool-dedup-overlap`。
- 既有：Gemini 呼叫不計費（`backlog-gemini-cost-metering`）、自架 ASR 事後記帳高估（`backlog-selfhosted-asr-actual-cost`）。

### 契約姿態（Rule 20）

- `MediaRef` [@contract-v1]（`pipeline.go:75`）：services 側 runner port 以 mirror 語彙攜帶相同兩欄位——confirmed against [@contract-v1]；Rule 19 mirror-types，若加 parity 測試比照 `route_prediction_parity_test.go`。
- `generation_batch_progress` SSE 與 202 回應（9R-16 stamp，下游 ux3-subtitle-v2-batch done＝frozen）：只 additive；bump 與否於實作時裁定並記 Change Log（additive-only 的先例是「不 bump、記 ack」，但以 Contract Stamp Check 為準）。
- 不動：`transcription_*` [@contract-v2]、D2 [@contract-v3]、D6 `PipelineStage`、FR12 端點、preview 端點、sub-4-1 三個 candidates 端點。

### Project Structure Notes

- `apps/api/internal/services/generation_batch.go`（＋`_test.go`）— 主改造面
- `apps/api/internal/handlers/generation_batch_handler.go` — `budget_usd`、Swagger、missing 註記
- `apps/api/cmd/api/generation_batch_runner_adapter.go` — NEW：D2 pipeline adapter（Rule 19）
- `apps/api/cmd/api/main.go` — runner mode 選擇注入、`:631-633` 註解修正、finder 注入 `repos.Episodes`
- `apps/api/internal/services/transcription_service.go` — 預期零改動（`WithMediaType` 已存在）
- `docs/sse-event-types*.md`／`docs/deployment.md` — 視 payload/行為變更同步
- 前端：**本 story 不動**（FE 是 sub-4-3；既有 `subtitleService.ts`/`GenerationBatchDialogV2.tsx` 靠 additive 相容）

### Time-dependent visual coverage

`N/A — no wall-clock-reading components touched.` 後端 story，不新增或修改任何前端元件。

### References

- [Source: `_bmad-output/planning-artifacts/design-prompt-cost-consent-2026-08-09.md#5-quinquies`] — D1 裁定原文與 BE 範圍
- [Source: sprint-status.yaml `backlog-pipeline-cost-consent`] — 2026-08-07 三件一體裁定＋「需 rewire 到 D2 pipeline」原文
- [Source: `_bmad-output/implementation-artifacts/sub-4-1-cost-preview-backend.md`] — 前一 story 情報：`generationCandidateFinder` movies-only 是「查不到」副作用、preview 端點凍結、軟上限措辭
- [Source: `apps/api/internal/services/generation_batch.go:63-84,185,213-215,233-266,292-343`] — 本 story 改造面的現況
- [Source: `apps/api/internal/services/transcription_service.go:344-362`] — `WithMediaType` 與 movie 預設的寫錯表風險
- [Source: `apps/api/internal/subtitle/process_item.go:347-355,392,427`] — 暫停非失敗語意
- [Source: `apps/api/internal/ai/budget.go:93,98` + `governor.go:68-73` + `types.go:33`] — budget 建構、`>=` 前置檢查、sentinel
- [Source: `apps/api/internal/cost_consent_test.go:16-65`] — 守衛測試禁的是 `.EnqueueMissing(` 呼叫點
- [Source: `apps/api/cmd/api/main.go:573-640,755-757`] — mode 分支、pool 保留註解、batch 建構注入點
- [Source: `project-context.md`] — Rule 3／7／11／17／19／20／24

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - **YES** — filed at authoring time：
    - **③ backlog-with-carry-forward-link** — `backlog-batch-pool-dedup-overlap`：批次直呼 `ProcessItem` 的項目不在 WorkerPool `inFlight` 去重範圍內，與 FR12 對同一 media 的並發請求可能重複處理（今日 Route C batch 與 FR12 之間已存在同類競態，非本 story 引入；管線冪等性讓後果限於浪費一次處理）。非阻塞。

### File List
