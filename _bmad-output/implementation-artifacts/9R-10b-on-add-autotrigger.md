# Story 9R-10b: 入庫自動觸發 —— 常設同意下的「早上起來字幕就好了」

Status: done

**Unblocked 2026-08-20** —— `9R-UX-auto-generation-toggle-design` done（Sally MCP review PASS），PR #246 已 merge，
`flow-e-scanner/e5-d.png`・`e5-m.png`・`flow-j-specs/j4-d.png` 皆在 main 上。

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story.
     ⚖️ AC #1 已裁定 2026-08-19（Alexyu）：「花錢須同意」—— 付費動作不得自動執行。
     🔬 2026-08-20 create-story 重跑：AC #2–#9 依「實際程式碼」重寫（原版有 6 處對現況的錯誤假設，見 Dev Notes §程式碼盤點）。
     ⚖️ 2026-08-20 Alexyu 選 B（等設計）：Task 5 由 lane ③ 升格 lane ②，本 story 轉 blocked。 -->

**Epic:** 9R 尾款（Rule-24 ③ from 9R-10, 2026-07-05）· **Risk: 🔴 HIGH —— 花費同意紅線**（2026-08-07 裁定 + `cost_consent_test.go` repo guard）
**Source:** sprint-status `9R-10b-on-add-autotrigger`；2026-08-19 What'Sub 對抗重審列為無悔四項之 N4 —— 同時是 P（每日體驗）、O（最小可愛 demo）、C（購買觸發點）三情境的共同核心。
**機器盤點：** 管線（9R-10）、批次協奏（9R-16）、成本預估＋同意＋預算（sub-4-1/4-2/4-3, M2.5）、詞彙庫閉環（sub-5-5）、scan-complete 掛點（`scanner_service.go:314`）皆已出貨。

---

## Story

As a NAS owner who adds new episodes at night,
I want items that arrive with a Chinese embedded track to be finished automatically for free, and items that would cost money to be queued up priced and waiting for one tap,
so that by morning the library is partly subtitled and truthfully badged —— without the system ever spending money I didn't consent to.

---

## Context —— 為什麼這是 HIGH risk，以及紅線的原文

**2026-08-07 事故與裁定**（`apps/api/internal/cost_consent_test.go` 檔頭原文）：pipeline 模式首次上線時 scan-complete 曾直接掛整庫 sweep，一次掃描 enqueue 1,026 項、約 2/3 走付費 ASR、估算 ~US$200，使用者全程沒看到數字。裁定：「**scanning updates metadata and nothing else；paid generation is chosen explicitly on a screen that shows the estimate first**」。repo guard `TestScanMustNotAutoEnqueueSubtitleGeneration` 令 `.EnqueueMissing(` 在生產程式碼零呼叫者。

本 story **不是**解除那條紅線 —— 是在紅線**之下**開一條「完全不花錢」的自動化通道。

## AC #1 —— ⚖️ 花錢須同意（Alexyu 裁定 2026-08-19）

裁定原文：「**9R-10b 花錢須同意**」。落地語意：

| 動作類別 | 自動觸發行為 |
|---|---|
| **零花費**：繁中內嵌軌 passthrough（`deliver_direct`）、簡中/混合軌 OpenCC s2twp（`convert_then_deliver`）、ffprobe 探測＋軌道抽取＋語言偵測 | ✅ **可全自動執行**（per-library 開關，預設 OFF） |
| **付費**：LLM 翻譯（`translate`）、ASR 轉錄（`no_text_source`） | 🔴 **自動一律止步** —— 不呼叫、不排隊、不扣款。項目原樣留在「缺繁中」集合裡，下次使用者開同意流程（F14–F20）自然帶著金額出現 |
| 常設預算上限（standing budget）模式 | ❌ 本次不做 —— 若未來想要「設一次月上限、之後全自動」，另立 story 重新裁定 |

淨效果：**「免費的自動做完，要花錢的原地等你一鍵同意」**。

> ⚠️ **AC #1 的關鍵工程後果**（原版 story 沒寫、但決定整個實作形狀）：
> 現有的**估價用**路線預測器（`services.RoutePrediction`）只有 `extract`／`asr`／`skip` 三類，
> 而 `extract` 的註解白紙黑字寫著「**NOT the same as free: an English track still pays for LLM translation**」
> （`generation_candidates.go:31-35`）。**免費層在估價層面是分不出來的。**
> 唯一分得出「繁中直送／簡中轉換／英文翻譯」的地方是 `subtitle.Router.SelectAndRoute`
> （`router.go:289 routeForVariant`）—— 而它是**抽取之後**才知道的。抽取＝本機 ffmpeg，**免費**。
> ⇒ 免費層必須「跑管線前半段，在付費動作前煞車」，不能靠估價預先分流。這就是 AC #3 的由來。

---

## Acceptance Criteria

### AC #2 —— 政策設定（per-library opt-in，預設 OFF）

> 🔄 **2026-08-20 修訂**：初稿提議把政策塞進 settings JSON key。**改用 `media_libraries` 的布林欄位** ——
> 該表**已有一模一樣的先例** `auto_detect`（`020_create_media_libraries.go:22` 欄位、`media_library.go:29` 欄位標籤、
> `media_library_repository.go:63/79/99/179` 四處 CRUD 全通）。用欄位可以一次寫入、無跨資源部分失敗、
> FE 沿用既有 `updateLibrary` mutation；用 settings key 則 modal 儲存要寫兩個資源（library + settings），
> 其中一個失敗就是靜默不一致。**欄位勝出。**

- **DB**：新 migration 加 `media_libraries.auto_subtitle INTEGER NOT NULL DEFAULT 0`（**預設 0 = OFF**，比照
  `auto_detect ... DEFAULT 0`）。
- **Model**：`models.MediaLibrary` 加 `AutoSubtitle bool \`db:"auto_subtitle" json:"auto_subtitle"\``。
- **Repo**：`media_library_repository.go` 的 INSERT / 兩處 SELECT / UPDATE **四處都要補**（`auto_detect` 在哪出現，
  `auto_subtitle` 就在哪出現 —— 漏掉 SELECT 會讓值永遠讀成 false，漏掉 UPDATE 會讓它永遠存不進去）。
- **Service**：`UpdateLibraryRequest` 加 `AutoSubtitle *bool`，在 `media_library_service.go:101-115` 的
  load-then-patch 區塊照 `req.Name`/`req.ContentType`/`req.SortOrder` 的**指標選填**慣例加一段。
  **不新增 endpoint** —— 沿用既有 library update 路由。
- **單次上限** `max_per_run`：**不是 per-library 設定**，用套件層常數（預設 20），與 `PipelineConcurrencyM1`
  同一種「寫死一個安全值」的慣例。要調再另立 story。
- **FE**：`LibraryEditModal.tsx` 新增 checkbox。文案與版面**由 `9R-UX-auto-generation-toggle-design` 裁定**，
  dev 照設計稿實作，**不得自創**。
  ⚠️ 已知 a11y 前提：該 modal 是**手刻**的（不是 `ui/Dialog` Radix），沒有 focus trap —— 本 story
  **不負責**修它（範圍紅線），但新 checkbox 本身要有 `htmlFor`/`id` 配對與 44px 觸控目標。
- 📌 **順手發現、不處理**：`UpdateLibraryRequest` 目前**沒有** `AutoDetect` 欄位，⇒ `auto_detect` 這個既有欄位
  無法經 API 修改（只有 create 時能定）。與本 story 無關，見 Discovery Triage。

### AC #3 —— 免費層閘門（本 story 的技術核心）`[@contract-v1]`

- `subtitle.ProcessItemOptions`（`pipeline.go:97`）新增 `FreeOnly bool`（additive；該 struct 目前**未帶** `[@contract-vN]` 戳記 ⇒ Rule 20 隱含 v0，additive 欄位比照 `default_budget_usd` 先例**不需 bump**，但本 AC 為新戳記 v1，下游消費者＝本 story 的 scan callback）。
- `process_item.go:142-158` 的 route switch 在 `FreeOnly=true` 時：
  - `RouteDeliverDirect` / `RouteConvertThenDeliver` → **照常執行**（OpenCC 是本機的，`converter.go`）。
  - `RouteTranslate` → **不得**進入 `p.deliverable(...)` 的 `RouteTranslate` 分支（`process_item.go:263`），亦即**不得**呼叫 `TranslateTrack`。
  - `RouteNoTextSource` → **不得**進入 `transcribeFallback`（`process_item.go:402`）。
  - `RouteSkip` → 行為不變（照舊 `recordSkip`）。
- 煞車時的狀態語意：項目必須**回到／留在「缺繁中」集合**，讓 `FindMissingZhHantSubtitle` 下次仍撈得到它（否則它會從同意清單消失 —— 那是靜默吞掉使用者的內容，比不做還糟）。
  dev 須在 Completion Notes 明確記載採用哪一種：(a) 不寫任何 run／不改 `subtitle_status`（最小侵入，推薦），或 (b) 寫一筆 run 但保持 status 不變。**禁止**寫入任何終局 status（`found`/`skipped`/`no_text_source`）。
- **不得**改動 `ProcessItem` 在 `FreeOnly=false`（既有全部呼叫端）時的任何行為 —— 現有測試逐字綠。

### AC #4 —— 觸發線（scan-complete 組合）

- 掛點以**組合**方式接入：`main.go:660` 目前在 pipeline 分支裡**重複**呼叫了一次 `scannerService.SetOnScanComplete(postScanEnrichment)`（同一個 callback 設兩次，實質 no-op）。**那一行就是本 story 要換掉的位置**。
- sub-1-6 AC #2 紅線原文（`main.go:432-435`）：「SetOnScanComplete holds exactly ONE callback … must **WRAP** this body, never call the setter a second time」。⇒ 建立 `subtitle.ComposeScanCallback(prev func(), next func()) func()`，`postScanEnrichment` **byte-unchanged** 先行、自動觸發後行。
  ⚠️ `subtitle.ComposeScanCallback` **目前不存在** —— `main.go:435` 只是一句「未來會有」的註解（`grep -rn ComposeScanCallback` 僅命中該註解）。dev 要新建它。
- **觸發集合的誠實界定**：`SetOnScanComplete(fn func())` 不帶參數，`ScanResult`（`scanner_service.go:50`）只有計數沒有 id ⇒ **「本次 scan 新增的項目」在 callback 端拿不到**。
  **本 story 的裁定：不改 setter 簽章**（那是 sub-1-6 紅線），改用**既有的整庫「缺繁中」列舉**：`repos.Movies/Episodes.FindMissingZhHantSubtitle`（`generation_candidates.go:60-70` 的同一個述詞，單一事實來源），再套三道過濾：
  1. 該項目所屬 library 開了開關（電影＝`movies.library_id`；**分集沒有 `library_id`**，須經 `series.library_id` 解析 —— `movie.go:288` / `series.go:104` 有，`episode.go` 沒有）；
  2. `subtitle_status ∈ {not_searched, not_found, untranslated}`；
  3. 取前 `max_per_run`（預設 20）。
  這在「免費層」前提下是安全的：跑滿 20 項的最壞情況是本機 CPU 時間，**帳單為零**。
- 排程掃描（`scan_scheduler.go`）與手動掃描共用同一條 callback，政策在 callback 內判定，不在觸發源判定。
- 全流程在 goroutine 內、`context.Background()`，比照 `postScanEnrichment` 既有形狀（`main.go:436-449`）。

### AC #5 —— 「入待同意清單」＝零新程式碼（範圍刪除，不是遺漏）

- 同意清單**不是佇列**：`GenerationCandidateService`（`generation_candidates.go`）在使用者開啟流程時**即時**從 `FindMissingZhHantSubtitle` 重算並估價（sub-4-3 CR H2 已把 stale 快照修成「掃描後／批次後強制重新分析」）。
- ⇒ 被 AC #3 煞車的付費項目**自動就在**下次的 F15 清單裡帶著金額。**本 story 不新增待同意佇列、不新增 badge、不新增通知**。
- 掃描完成的入口 F17（「{n} 部影片缺繁中字幕」＋「產生字幕 →」）**已由 sub-4-3 Task 5 出貨**，本 story 不動它。
- 驗收方式：整合測試證明「auto 跑完後，一個 `translate` 路線的項目仍被 `FindMissingZhHantSubtitle` 撈到」。

### AC #6 —— repo guard 保留並加固

- `TestScanMustNotAutoEnqueueSubtitleGeneration`（`internal/cost_consent_test.go`）**原樣保留、一個字不改**。其不變量（整庫 sweep 零生產呼叫者）在本 story 下**未變** —— 自動路徑不呼叫 `EnqueueMissing`。
  ⚠️ 該檔頭寫著「whoever does it has to **delete this test**」—— 那是給「解除付費 sweep」的人看的。**本 story 不是那個人**；刪掉它就是 CR High。
- **新增互補 guard**（同檔或手足檔），以**測試**斷言自動路徑的免費性。優先用行為斷言而非字串掃描：
  以 fake `ChunkTranslator` / fake `SpeechTranscriber` 跑一輪 `FreeOnly=true` 的 `ProcessItem`，對 `translate` 與 `no_text_source` 兩種夾具斷言 **fake 的呼叫計數 == 0**。
  測試註解須引用 2026-08-19「花錢須同意」裁定與 2026-08-07 事故鏈。

### AC #7 —— 誠實狀態

- 自動觸發完成的項目走既有 SSE／狀態機（`subtitle_status` 生命週期、Activity 生成列）；**徽章語意零新增**。
- 與 bugfix-j 串接：partial-failure verdict（`untranslated`）在自動路徑同樣生效 —— 自動化**不得**放大不誠實。
- 觀測：自動觸發啟動與結束各記**一行** `slog.Info`（比照 `main.go:661-669` 的 `scan_auto_enqueue` 慣例），帶 `library_ids`、`considered`、`processed`、`deferred_paid` 四個計數。**不得**每項一行（sub-4-1 AC #5 先例）。

### AC #8 —— 測試

- **AC #3 閘門 table-driven**（Go，RED-first）：四種 `RouteKind` × `FreeOnly ∈ {true,false}` = 8 例 → 斷言「是否呼叫 translator／transcriber」＋「終局 status」。
- **AC #6 免費性 guard**：translator/transcriber fake 呼叫計數 == 0（見上）。
- **AC #4 組合 callback**：`ComposeScanCallback` 單元測試 —— prev 先跑、next 後跑、prev panic 不吞掉 next（或明確記錄選擇的語意）；並釘住 `postScanEnrichment` 主體 byte-unchanged（sub-1-6 AC #2 回歸釘）。
- **政策過濾器 table-driven**：開關 OFF／ON × 電影／分集（分集走 series 解析）× status 三態 × 超過/未超過 `max_per_run` → 斷言選中集合。
- **AC #5 整合**：free-only 跑完後 `translate` 項目仍在 `FindMissingZhHantSubtitle` 結果內。
- **FE**：`LibraryEditModal.spec.tsx` 補 checkbox 的 render／toggle／送出 payload 三例。
- 閘門：`nx run api:test`＋`staticcheck` 全綠；`nx run web:test` 全綠；`pnpm format:check` 綠（Rule 15 / feedback_format_before_commit）。
  ⚠️ 原版 AC 寫的「既有 26 Claude-touching 測試」查無此數（今日 `grep -rln Claude --include=*_test.go` = 18 檔）—— 已刪除該不可驗證的宣稱，改用上面的具名閘門。

### AC #9 —— 範圍紅線

- 不動每次同意流程（F14–F20）—— 手動批次照舊。
- 不動 `EnqueueMissing`、不動 `GenerationBatchProcessor`、不動 `WorkerPool`。
- 不做常設預算／月上限（AC #1 明文排除）⇒ **本 story 完全不碰 `ai.Budget`**（免費層花費恆為 0）。
- 不做下載完成觸發（Epic 13 story 13-5 職權）；不做 per-item 重試自動化（sub-5-3 已交付手動重試）。
- 不改 `SetOnScanComplete` 簽章、不改 `ScanResult` 形狀。

---

## Tasks / Subtasks

- [x] **Task 1 — AC #3 免費層閘門（管線）**
  - [x] `pipeline.go` `ProcessItemOptions` 加 `FreeOnly bool`（doc comment 引用兩次裁定 ＋ additive-on-v1 理由）
  - [x] `process_item.go` route switch 前置 `if opts.FreeOnly` 分流：`translate`／`no_text_source` 於**付費呼叫之前**走 `deferPaidItem`
  - [x] 煞車狀態語意 = **(b) 變體**（見 Completion Notes「煞車語意」）：run 記 `skipped`＋理由，media 還原成**載入時的原值**
  - [x] RED-first：8 例 table-driven（`TestProcessItem_FreeOnlyGate`）＋2 例留存性（`..._FreeOnlyBrakeKeepsItemMissing`）＋1 例 skip 不受影響
- [x] **Task 2 — AC #4 觸發線（組合 callback ＋ 政策過濾器）**
  - [x] 新建 `subtitle.ComposeScanCallback`（復刻 sub-1-6 已驗證語義）＋6 例單元測試
  - [x] 新建 `subtitle.AutoGenerator`：讀 policy → `FindMissingZhHantSubtitle` → 三道過濾 → 逐項 `ProcessItem{FreeOnly:true}`
  - [x] 分集走 `series.library_id`（每個 series 只查一次，非每集一次）
  - [x] `main.go:660` 換成組合；`postScanEnrichment` 主體 **byte-unchanged**（diff 僅 657-680 一塊）
  - [x] 政策過濾器 table-driven（11 例）＋19 例 AutoGenerator 總計
- [x] **Task 3 — AC #2 政策設定（BE 側）**
  - [x] Migration `031_add_media_library_auto_subtitle`（idempotent，`DEFAULT 0`）＋4 例測試
  - [x] `models.MediaLibrary.AutoSubtitle` ＋ repo **四處** CRUD 全補
  - [x] `UpdateLibraryRequest.AutoSubtitle *bool` ＋ service load-then-patch
  - [x] **零新增 endpoint**
  - [x] 測試：repo 往返 6 例（含 INSERT/兩處 SELECT/UPDATE 各自的失效模式）＋ service 指標選填 3 例
- [x] **Task 4 — AC #6 guard ＋ AC #7 觀測**
  - [x] `cost_consent_free_lane_test.go`（9 例）—— 走**真 Pipeline** 斷言 translator/transcriber 呼叫計數 == 0；註解引用 2026-08-07 事故與 2026-08-19 裁定
  - [x] **fault injection 驗證**：把閘門改成 `if false` → 6 例轉紅；還原 → 全綠
  - [x] `cost_consent_test.go` `git diff` = **0 行**
  - [x] 兩行 `slog.Info`（started／finished）＋`considered`/`processed`/`deferred_paid`/`failed`
- [x] **Task 5 — AC #2 FE 開關**（`9R-UX-auto-generation-toggle-design` done，PR #246 已 merge）
  - [x] checkbox ＋三句定稿文案，逐字照 E5-D
  - [x] `LibraryEditModal.spec.tsx` **7 例**（原 2 例 ＋ 新 5 例：label 綁定／兩段文案／不得出現「掃描」／欄位順序／payload 含 `autoSubtitle`）
  - [x] **Rule 21** 檔頭改為 `// Design ref: ux-design.pen Screen E5-D (hUVYm) · E5-M (P0P82x) · J4-D (sPzZT)`
        —— 順帶關掉 `bugfix-libraryeditmodal-wrong-design-ref`
  - [x] **➕ Rule 24 ① 吸收（authoring 缺口）**：`LibraryCard` footer 顯示開關狀態
        （`2 個資料夾 · 316 個項目 · 自動處理免費字幕`，末段 `--success`；關閉時整段不出現）
        ＋新建 `LibraryCard.spec.tsx` 3 例。理由見 Discovery Triage ①。
- [x] **Task 6 — AC #8 收尾閘門**
  - [x] `pnpm nx test api` 全綠／`pnpm nx test web` **2698 例全綠**（233 檔）
  - [x] `pnpm nx lint api`（釘版 staticcheck-2026.1）綠／`pnpm nx lint web` 綠／`pnpm format:check` 綠
  - [x] AC #5 整合：`TestFreeLane_DeferredItemStaysOnTheConsentList`（斷言零 `zh-Hant` 語言戳記、零 sidecar 路徑）

---

## Dev Notes

### 🔬 程式碼盤點（2026-08-20 grep 實查，非推測）—— 原版 story 的三處錯誤假設

| 原版假設 | 實況 | 影響 |
|---|---|---|
| 「候選分析＋估價」可分辨免費/付費 | `RoutePrediction` 只有 `extract`/`asr`/`skip`，且 `extract` 註解明寫「NOT the same as free」（`generation_candidates.go:31-35`） | ⇒ 免費層改用「跑到 router 再煞車」（AC #3） |
| 需要把付費項目「掛入待同意清單」 | 同意清單是**即時重算**的，不是佇列（`generation_candidates.go` + sub-4-3 CR H2 forceAnalyze） | ⇒ 這部分**零程式碼**（AC #5），story 縮小一大塊 |
| 觸發集合＝「本次 scan 新增/變更的項目」 | callback 無參數、`ScanResult` 只有計數 | ⇒ 改用整庫「缺繁中」列舉＋cap（AC #4）；免費層下安全 |
| 「預算足/不足/觸頂」測試維度 | AC #1 已排除常設預算；免費層花費恆 0 | ⇒ 該維度刪除（AC #9） |
| 「26 個 Claude-touching 測試」 | 今日 18 個檔 | ⇒ 改為具名閘門（AC #8） |
| 「cost_consent guard 交接／改名」 | 其不變量未變 | ⇒ **原樣保留**，另加互補 guard（AC #6） |
| 政策放 settings JSON key | `media_libraries` 已有 `auto_detect` 布林欄位先例，全套 CRUD 通了 | ⇒ 改用 `auto_subtitle` 欄位，省掉跨資源部分失敗（AC #2 修訂） |

### 關鍵檔案與行號（dev 直接跳）

| 用途 | 路徑 |
|---|---|
| 免費/付費路線的**唯一**判定點 | `apps/api/internal/subtitle/router.go:289` `routeForVariant` |
| route switch（要加閘門） | `apps/api/internal/subtitle/process_item.go:142-158` |
| 付費呼叫點（要擋住） | `process_item.go:263`（translate 分支）／`process_item.go:402` `transcribeFallback` |
| options struct | `apps/api/internal/subtitle/pipeline.go:97` |
| scan-complete 掛點（要換的那一行） | `apps/api/cmd/api/main.go:660` |
| 不可動的 enrichment 主體 | `apps/api/cmd/api/main.go:436-449` |
| callback 觸發條件 | `apps/api/internal/services/scanner_service.go:314` |
| 「缺繁中」列舉述詞 | `generation_candidates.go:60-70`（`FindMissingZhHantSubtitle`） |
| repo guard（不可動） | `apps/api/internal/cost_consent_test.go` |
| per-library 布林欄位**先例** | `apps/api/internal/database/migrations/020_create_media_libraries.go:22`（`auto_detect`） |
| repo 四處 CRUD（照抄 `auto_detect`） | `apps/api/internal/repository/media_library_repository.go:63,79,99,179` |
| load-then-patch 指標選填慣例 | `apps/api/internal/services/media_library_service.go:101-115` |
| FE 開關落點 | `apps/web/src/components/settings/LibraryEditModal.tsx` |

### 架構規則對照

- **Rule 4 / Rule 19**：自動觸發邏輯放 `internal/subtitle`（`subtitle → services` 是**允許**方向；`services → subtitle` 是**禁止**的，會成環）。若要放 `internal/services`，只能透過窄介面（Rule 11）反向注入 —— 但 `ComposeScanCallback` 的註解已把它指定在 `subtitle` 套件，照辦。
- **Rule 11**：新的列舉／設定讀取一律走窄介面（比照 `CandidateMovieFinder` 只有一個方法的形狀），由 `main.go` 注入。
- **Rule 2**：`slog` only；**Rule 3**：使用者可見文案 zh-TW（本 story 的使用者可見面只有 FE checkbox 文案）。
- **Rule 7**：本 story **不新增** error code（免費層失敗沿用既有 `SUBTITLE_*` sentinel，`internal/subtitle/errors.go`）。
- **Rule 9**：測試與程式碼同目錄；**Rule 16**：斷言要斷在行為上（呼叫計數／集合內容），不是「沒 panic」。
- **Rule 20**：本 story 新增戳記 `AC #3 [@contract-v1]`（`ProcessItemOptions.FreeOnly`）。上游 `ProcessOutcome`/`MediaRef` 已是 `[@contract-v1]` 且**未變更** ⇒ dev 於 Completion Notes 記：
  `📎 Contract Stamps: FOUND (1 new v1 in this story: AC #3 ProcessItemOptions.FreeOnly; upstream ProcessOutcome/MediaRef v1 unchanged, ack recorded, no bump owed)`
- **Rule 24**：Task 5 的設計稿缺口是已知的 ③ 案（見該 Task）；其餘發現照 Discovery Triage 欄位填。

### 跨端拆分檢查（Epic 8 Retro Agreement 5 / Epic 9c Retro AI-1）

BE 任務 4（Task 1/2/3/4）· FE 任務 1（Task 5）⇒ **FE 側 1 ≤ 3，不觸發強制拆分**。單一 story 成立。

### 前一個 story 的教訓（9R-10a，2026-08-19 CR APPROVED-WITH-FIXES，0H/2M/3L 全修）

1. **M1 —— 錯誤不可一律吞成 404/正常。** `FindByID` 的任何錯誤原本一律回 404，SQLite 鎖死（NAS FUSE 問題已實際發生）時使用者被告知「找不到」。**本 story 的對應風險：** 自動觸發是背景跑的，一個吞掉的錯誤**完全無聲**。⇒ 列舉／政策讀取失敗必須 `slog.Error` 且**中止本輪**，不得半途靜默。
2. **M2 —— 「聲稱依序實作」但缺一道測試。** ⇒ AC #8 的 8 例 table-driven 要**真的 8 例**，缺一例就是 CR finding。
3. **L1 —— Completion Notes 的宣稱要查證過才寫**（該次「新訊息全數 zh-TW」不精確）。⇒ 本 story 的 Completion Notes 若宣稱「零付費呼叫」，必須附上 guard 測試名稱作為證據。
4. **L2 —— 例行狀況用 `slog.Warn` 不用 `slog.Error`。** ⇒ 「本輪沒有可自動處理的項目」是例行，用 `Debug`/`Info`。
5. **紅線 2 先例 —— 不要順手改鄰居。** 9R-10a 明文「不要順手改電影路由」。⇒ 本 story：**不要順手改** `EnqueueMissing`、`GenerationBatchProcessor`、`WorkerPool`、F14–F20。

### 近期 commit 脈絡

`26c64f60 feat(9R-10c)` 分集列逐集字幕入口（FE，剛併）· `28921c79 feat(9R-10a)` 單集手動生成入口（BE）· `d876f0d2 chore(visual)` -linux baseline bootstrap。
⇒ 手動觸發（電影＋單集）兩端都已就位，本 story 是把「免費的那半」自動化。

### Project Structure Notes

- 新檔預期：`apps/api/internal/subtitle/scan_callback.go`（＋`_test.go`）、`apps/api/internal/subtitle/auto_generation.go`（＋`_test.go`）。命名可由 dev 定，但**必須**落在 `internal/subtitle`（Rule 19）。
- `main.go` 只新增「建構＋注入＋換掉 :660 那一行」，不搬既有結構。
- **有 DB migration**：`media_libraries.auto_subtitle`（AC #2 修訂後）。編號接續 `apps/api/internal/database/migrations/` 現有最大號。

### Time-dependent visual coverage

- **N/A —— 本 story 唯一觸及的 `apps/web` 元件是 `LibraryEditModal.tsx`，不讀 `Date.now()`/`new Date()`/`Date.UTC()`/`Date.parse()`。** 無 visual fixture 新增（checkbox 走既有 modal spec，非 gallery fixture）。
- Reference: `project-context.md` Rule 23。

### References

- [Source: apps/api/internal/cost_consent_test.go] —— 2026-08-07 事故原文與解除條件
- [Source: apps/api/internal/subtitle/router.go#routeForVariant] —— 免費/付費路線的唯一判定點
- [Source: apps/api/internal/subtitle/process_item.go#ProcessItem] —— route switch 與兩個付費呼叫點
- [Source: apps/api/cmd/api/main.go#L432-L449, #L640-L670] —— sub-1-6 AC #2 紅線與重複 setter 呼叫
- [Source: apps/api/internal/services/generation_candidates.go#RoutePrediction] —— 估價層分不出免費層的證據
- [Source: _bmad-output/implementation-artifacts/sub-4-1-cost-preview-backend.md] —— 成本預估 API
- [Source: _bmad-output/implementation-artifacts/sub-4-2-consent-batch-backend.md] —— 同意批次
- [Source: _bmad-output/implementation-artifacts/sub-4-3-cost-consent-frontend.md] —— F14–F20 消費面、F17 掃描完成入口、CR H2 forceAnalyze
- [Source: _bmad-output/implementation-artifacts/9R-10a-series-episode-trigger.md#Senior Developer Review] —— 前一 story 的 5 項 CR 教訓
- [Source: project-context.md] —— Rules 2/3/4/7/9/11/15/16/19/20/21/23/24

---

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

**🔒 煞車語意（AC #3 要求明確記載）—— 採 (b) 的變體，理由如下。**
story 提供的 (a)「不寫任何 run／不改 `subtitle_status`」**在該位置不可能**：路線判定發生在
`process_item.go` 的 step 3，而 run row 在 step 2 就已 `pending→running`、media row 也已寫成 `extracting`。
故實作為：run 記 `skipped` ＋ `error_message` 說明為何延後（**成本同意功能保留稽核軌跡有價值**），
media row **還原成載入時的原值**（新增 `MediaItem.SubtitleStatus` 快照）。
**不用一律 `not_searched`**：那會抹掉 bugfix-j 的 `untranslated` 判定，讓已有英文 SRT、只差翻譯的項目
反過來宣稱「從未搜尋過」—— 正是 AC #7「自動化不得放大不誠實」禁止的事。
另外查證：`missingZhHantSubtitleWhere`（`movie_repository.go:898`）**只看 `subtitle_language`，不看 `subtitle_status`**，
所以「留在同意清單」由「不寫 zh-Hant 語言」結構性保證；status 還原是為了讓**本 story 自己的** AC #4 過濾器下次仍撈得到。

**💰 零付費宣稱的證據**（9R-10a L1 教訓：宣稱要附證據）：
`TestFreeLane_NeverReachesPaidPorts`（走真 `Pipeline`，斷言 `fakeTranslator.calls` 與 `fakeSpeechTranscriber.calls` 皆為空）
＋ `TestProcessItem_FreeOnlyGate` 8 例 ＋ `TestAutoGenerator_AlwaysProcessesFreeOnly`。
**且經 fault injection 反證**：將 `if opts.FreeOnly` 改為 `if false` → 6 例轉紅；還原 → 全綠。
`internal/cost_consent_test.go` `git diff` = **0 行**。

**📎 Contract Stamps: FOUND (1 additive-on-v1 in this story; upstream unchanged, no bump owed)**
—— `ProcessItemOptions.FreeOnly`。⚠️ **story 對此的敘述有誤已更正**：該 struct **本來就帶** `[@contract-v1]`
（`pipeline.go:96`），不是「未帶戳記的隱含 v0」。處置結論不變但依據改為同檔既有先例
`TranslateResult.HarvestedTerms`（`pipeline.go:67`：「ADDITIVE on v1 … the `default_budget_usd` precedent — no bump」）：
零值即現行行為的 additive 欄位不 bump。上游 `MediaRef`/`ProcessOutcome`（皆 `[@contract-v1]`）形狀未動。
`MediaItem` 未帶戳記，新增 `SubtitleStatus` 為純 additive。

**🔗 AC Drift: FOUND — sub-4-1 AC #1 → 本 story AC #4。**
sub-4-1 AC #1 把 scan→generation 的自動派工**整條拆掉**（其自身亦是對 sub-1-6 AC #2 的 drift，已記於 sub-4-1:188）。
本 story 在同一個掛點重新接上一條路徑 —— 但**語義不同**：sub-4-1 拆掉的是**付費**整庫 sweep（`EnqueueMissing`），
本 story 接上的是**零花費**單向通道（`FreeOnly`），且 per-library 預設 OFF。
sub-4-1 為 `done`＝frozen，其 AC #1 未 stamp（隱含 v0，forward-only）⇒ **不欠 stale-mark**。
其守護的不變量（付費 sweep 零生產呼叫者）**未被破壞**，`cost_consent_test.go` 原樣通過。
grep 依據：`SetOnScanComplete|scan-complete|EnqueueMissing` across `_bmad-output/implementation-artifacts/*.md`。

**🎭 A11y Pre-Flight: PASS**（1 個觸及元件；`npx eslint` 對本 story 三個觸及檔 **0 warnings / 0 errors**）。
四類逐項：**modal focus** —— `LibraryEditModal` 是手刻 overlay、無 focus trap，**屬既有狀況，AC #2 明文不在本 story 範圍**，
新 checkbox 未加深該問題；**aria-live** —— 本次無非同步揭露內容；**keyboard/ARIA** —— checkbox 為原生 `<input type="checkbox">`，
`htmlFor`/`id` 配對經 `getByLabelText` 斷言；**觸控目標** —— 開關列 `min-h-[44px]`，label 包住整列擴大命中範圍。

**⚠️ 一個我查證後撤回的宣稱**（避免留下未驗證斷言）：撰寫過程中曾以為
「label 包住 input 又同時 `htmlFor` 指向它會雙重觸發」，並據此改成同層結構。
**實測反證**：巢狀版本在修好測試 mock 後同樣 7/7 綠 —— 真正原因是 mock 每次 render 回傳新物件、
使 hydrate effect 重跑重置狀態。已改回巢狀（觸控目標更大）並移除該錯誤註解。
該 effect 的重置行為是**既有**問題，已依 Rule 24 ③ 立案（見下）。

**🔧 順序微調（如實記錄）**：Task 3（DB 欄位）在 Task 2 的最後一項（`main.go` 接線）**之前**完成 ——
接線需要 `AutoLibraryPolicy` 的具體實作，而該實作依賴 `media_libraries.auto_subtitle` 欄位，否則無法編譯。
Task 2 的其餘三項（`ComposeScanCallback`、`AutoGenerator`、政策過濾器測試）皆按序先行，且全部以窄介面（Rule 11）撰寫，
未依賴 Task 3 的具體型別。

**🖼️ Visual regression（第一次推送後 CI 抓到，story 的第三處事實錯誤）**：
story 的「Time-dependent visual coverage」節寫「**無 visual fixture 新增**（checkbox 走既有 modal spec，非 gallery fixture）」——
**「無新增」為真，但漏了「既有 fixture 會需要 rebless」**：`settings-library-edit-modal` 是**既有的** gallery fixture，
modal 多一個欄位必然讓它的三個基準不匹配。AC #8 的閘門清單也沒有 `test:visual`，所以本機沒跑到，CI 才紅。
處置依 `tests/visual/README.md` 的 Baseline-update discipline ＋ `6bbd3fb4`(sub-4-3) 先例：
`-darwin` 本機重產（`CI=true` ＋只起 Vite dev server，比照 workflow 的 PR job）、`-linux` **刪除**交 CI incremental bootstrap；
基準獨立成 commit（`363ed45a`），不與邏輯混。全量重產只 4 張變動，第 4 張是既有本機漂移（已 restore）。

**⚠️ 交付狀態：`Visual Regression / PR` 這一支會維持紅色，直到本 PR merge —— 這是工作流的設計，不是未修完。**
`.github/workflows/visual-regression.yml` 有兩個 job：`pr`（`if: event_name == 'pull_request'`，**只驗證**）與
`main`（`if: event_name == 'push' || 'workflow_dispatch'`，**bootstrap 在這裡**）。
`-linux` 基準只能由 ubuntu runner 產生，而產生它的 job **不在 PR 事件上跑**。
⇒ 依既有流程：**merge 本 PR → main-push 觸發 incremental bootstrap → 自動開出
`chore(visual): bootstrap 3 missing -linux baselines (incremental)` PR（帶 `requires-manual-review` 交 Sally）→ merge 它**。
先例完全相同：`6bbd3fb4`(sub-4-3 #218) merge 後才有 `d876f0d2`(#240)。

**查證失敗性質為「純缺基準」**（決定它會走 incremental 而非 steady-state 失敗）：
第二輪 CI 的 4 筆失敗全是 `A snapshot doesn't exist ... settings-library-edit-modal/{default,hover,focus}-visual-linux.png`，
**零 pixel-diff、零 other**。其餘 13 支 check 全綠（Docker、E2E ×4、Unit、Build、Serve Smoke、Lint & Format…）。

**🧯 順帶抓到的型別破口**：`MediaLibrary.autoSubtitle` 設必填後，三個 gallery fixture 的 `satisfies` 缺欄位 ⇒
`tsc --noEmit` 從 main 的 147 錯變成 151。**本專案不以 tsc 為 CI 閘門**（Vite build 不做型別檢查，
CI 的 Build 也因此綠燈），所以這是我主動比對 main 才發現的。已補齊，回到 147（零新增）。

**🧪 一個被我自己推翻的 staticcheck 疑慮**：中途以非釘版 staticcheck 跑出兩筆 U1000
（`config.go:282`、`images/processor.go:206`），一度準備立案為 pre-existing。
改用專案**釘版** `staticcheck-2026.1`（`apps/api/project.json:33`，即 CI 用的那支）後 **全綠** ——
該兩筆是工具版本差異的產物，非真實既有失敗，故**不立案**。

### 🔬 Dev 實作期發現的兩處 story 事實錯誤（已查證，影響實作）

1. **`ProcessItemOptions` **有** `[@contract-v1]` 戳記**（`pipeline.go:96`，就在 `type ProcessItemOptions struct` 正上方）。
   story AC #3 寫「該 struct 目前**未帶** `[@contract-vN]` 戳記 ⇒ Rule 20 隱含 v0」—— **不正確**。
   ⇒ 處置不變但理由改寫：依**同檔既有先例** `TranslateResult.HarvestedTerms`（`pipeline.go:67`：
   「ADDITIVE on v1（sub-5-5 AC #3, the `default_budget_usd` precedent — no bump）」），
   零值即現行行為的 additive 欄位**不 bump**。已在 `ProcessItemOptions` 的 doc comment 明文寫下這個判斷。
2. **`subtitle.ComposeScanCallback` 曾經存在，是被 sub-4-1 刪掉的**（sub-1-6 `f2214299` 建立 → sub-4-1 `ac4083f5` 移除）。
   story 說它「目前不存在 —— `main.go:435` 只是一句『未來會有』的註解」—— 前半對、後半錯：那是一句**指向已刪除函式的過期註解**。
   ⇒ 實作將**復刻其已驗證的語義**（prev 先跑、`next == nil` 時原樣回傳 `prev`、兩者皆 nil 回 no-op、
   組合本身 inline 以便斷言順序），而不是重新發明。

### ➕ AC #2 補述（Rule 24 ① 就地擴充，2026-08-20 dev）

authoring 的 AC #2 只列了 modal 內的 checkbox，**漏了 Sally 裁定 3 的卡片顯示**
（J4-D 區塊 E 有完整 specimen）。少了它就會重演 Sally 明文要防的「隱形布林」——
使用者得逐一點進 modal 才知道哪個媒體庫開著。屬小、同一表面、且在 AC #2「使用者怎麼看到、怎麼改」的精神內
⇒ 依 Rule 24 ① **就地吸收**，並以本節＋Task 5 新增子項追蹤（不做「安靜修掉」）。

- `LibraryCard` footer 沿用**既有頓點語法**擴充，**不新增徽章**：路徑列已有一套彩色圓點狀態語彙，
  第二套會跟它搶同一眼。末段用 `--success`（＝同意流程裡「這個不用錢」的顏色語意）。
- 關閉時**整段不出現**（不是灰掉），避免把「沒開」渲染成一種狀態。

### 🎨 UX Verification（dev-story Step 9，強制）

比對對象：`_bmad-output/screenshots/flow-e-scanner/e5-d.png`（`hUVYm`）、`e5-m.png`（`P0P82x`）、
`flow-j-specs/j4-d.png`（`sPzZT`，文案與免費/付費界線的權威來源）。

| 區域 | 設計規格 | 實作 | 相符 | 需修 |
|---|---|---|---|---|
| 欄位順序 | 名稱 → 類型 → 資料夾路徑 → **開關（最後）** | 同 | ✅ | — |
| 開關前的分隔 | 分隔線隔開前三欄與開關 | `border-t border-[var(--border-subtle)]/50 pt-4` | ✅ | — |
| 控制項型態 | checkbox（非 toggle switch） | 原生 `<input type="checkbox">` | ✅ | — |
| 標籤文案 | `新檔入庫後，自動完成免費的字幕處理` | 逐字相同 | ✅ | — |
| 說明第一段 | `影片內建繁體中文字幕會直接沿用，簡體字幕自動轉成繁體。這些都在本機執行，不會產生費用。` | 逐字相同 | ✅ | — |
| 說明第二段 | `需要 AI 翻譯或語音辨識的影片不會自動處理，它們會留在「產生字幕」清單裡，標好預估金額等你確認。` | 逐字相同 | ✅ | — |
| 說明縮排 | 對齊標籤文字起點（checkbox 20 ＋ gap 10） | `pl-[30px]` | ✅ | — |
| 說明字級/色 | 小一級、次要色 | `text-xs text-[var(--text-secondary)]` | ✅ | — |
| 勾選色 | accent 藍 | `accent-[var(--accent-primary)]` | ✅ | — |
| 觸控目標（E5-M） | ≥44px | `min-h-[44px]` ＋ label 包住整列 | ✅ | — |
| 「掃描」二字 | 不得出現 | spec 斷言該區塊 `textContent` 不含「掃描」 | ✅ | — |
| 底部按鈕文案 | 設計畫「取消／**儲存**」 | 已出貨「取消／**儲存變更**」(edit)、「取消／**建立**」(create) | ⚠️ | **不改** —— 見下 |

**唯一不符項的處置**：按鈕文案。這**不是實作偏離設計**，而是**設計稿沿用了 Sally 提示詞中未查證的字串**
（提示詞逕自寫「儲存」，未先讀 `LibraryEditModal.tsx:258` 的既有標籤）。依 AC #9 範圍紅線
（不順手改鄰居 / 9R-10a 紅線 2），dev **不改既有按鈕文案** —— 那會連帶影響 create 模式的「建立」，屬另一個決定。
已依 Rule 24 ③ 立 `drift-e5d-save-button-label` 交 Sally 修設計稿。

**J4-D 規格頁比對**（規格頁本身非實作畫面，比對其 specimen）：

| 區域 | 設計規格 | 實作 | 相符 |
|---|---|---|---|
| 控制項狀態 · 關閉 | 空 checkbox ＋同一句標籤 | 預設 `autoSubtitle=false` | ✅ |
| 控制項狀態 · 開啟 | 藍勾 ＋標籤 ＋兩段完整說明 | 同 | ✅ |
| 卡片 footer · 關閉 | `2 個資料夾 · 316 個項目` | 同（末段不渲染） | ✅ |
| 卡片 footer · 開啟 | 上句 ＋ ` · 自動處理免費字幕`，末段 success 綠 | `text-[var(--success)]`，前段維持 muted | ✅ |
| 徽章 | **不得**新增徽章／彩色圓點 | 純文字，沿用頓點語法 | ✅ |

**未比對項**：E5-D／E5-M 是**只畫 modal 本身**的畫面（H3/I3 慣例），不含頁面外框，故無版面層級可比對項。

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - **② spawn-blocking-story** — **FE 開關無 `.pen` 設計節點**。⚖️ Alexyu 2026-08-20 裁定設計先行 ⇒ 已建
    `9R-UX-auto-generation-toggle-design`（`sprint-status.yaml` ＋ story 檔 `9R-UX-auto-generation-toggle-design.md`），
    本 story Status = `blocked`，雙向連結成立。
  - **③ backlog-with-carry-forward-link** — `UpdateLibraryRequest` 缺 `AutoDetect` 欄位 ⇒ 既有 `auto_detect`
    無法經 API 修改。發現於 AC #2 改用欄位時（`media_library_service.go:101-115` 只 patch Name/ContentType/SortOrder）。
    非阻塞、與本 story 無關 ⇒ 見 `sprint-status.yaml` `bugfix-library-autodetect-not-updatable`。
  - **① expand-scope-in-place** —— **`LibraryCard` footer 的開關狀態顯示**。Sally 裁定 3 與 J4-D 區塊 E 都有規格，
    但 authoring 的 AC #2／Task 5 都沒列到。小、同表面、在 AC #2 精神內 ⇒ 就地吸收，
    追蹤於 **AC #2 補述節 ＋ Task 5 新增子項**（非安靜修改）。
  - **③ `bugfix-libraryeditmodal-hydrate-resets-form`** —— `LibraryEditModal` 的 hydrate `useEffect` deps 是
    query data 物件身分，refetch（如按「新增路徑」觸發 invalidate）會把使用者**尚未儲存**的表單值打回伺服器值。
    `name`/`contentType` 早已如此，本 story 的 `autoSubtitle` 只是繼承同一模式、未擴大。非阻塞，已立案。
  - **③ `drift-e5d-save-button-label`** —— 設計 E5-D 畫「儲存」，已出貨為「儲存變更」/「建立」。
    成因是提示詞未查證既有標籤；依 AC #9 不改既有文案，交 Sally 修設計稿。非阻塞，已立案。
  - **③ `9R-UX-library-edit-modal-hover-focus-states`** —— `settings-library-edit-modal` 的
    default/hover/focus 三張 `-darwin` 基準 **md5 完全相同，且變更前也相同**（既有狀況）。
    依 `feedback_identical_rendering_is_sally_decision` **交 Sally 裁決**，dev 不自選方案。
  - **已關閉**：`bugfix-libraryeditmodal-wrong-design-ref` → `done`（Task 5 修正檔頭）。

### File List

**Backend**

| 檔案 | 動作 |
|---|---|
| `apps/api/internal/subtitle/pipeline.go` | modified — `MediaItem.SubtitleStatus` 快照欄位；`ProcessItemOptions.FreeOnly`（additive on v1） |
| `apps/api/internal/subtitle/process_item.go` | modified — route switch 前置 FreeOnly 閘門；新 `deferPaidItem` |
| `apps/api/internal/subtitle/media_store.go` | modified — 三個 loader 各補 `SubtitleStatus` |
| `apps/api/internal/subtitle/auto_generation.go` | **new** — `AutoGenerator`＋`AutoLibraryPolicy`／`AutoSeriesLibraryResolver` 窄介面＋`AutoGenerationMaxPerRun=20` |
| `apps/api/internal/subtitle/scan_callback.go` | **new** — `ComposeScanCallback`（復刻 sub-1-6 語義） |
| `apps/api/internal/database/migrations/031_add_media_library_auto_subtitle.go` | **new** — `auto_subtitle INTEGER NOT NULL DEFAULT 0` |
| `apps/api/internal/models/media_library.go` | modified — `MediaLibrary.AutoSubtitle` |
| `apps/api/internal/repository/media_library_repository.go` | modified — INSERT／2× SELECT／UPDATE 四處 |
| `apps/api/internal/services/media_library_service.go` | modified — `UpdateLibraryRequest.AutoSubtitle *bool` ＋ load-then-patch |
| `apps/api/cmd/api/auto_subtitle_policy_adapter.go` | **new** — repo → `AutoLibraryPolicy` 橋接（Rule 19，放組合根） |
| `apps/api/cmd/api/main.go` | modified — `SetOnScanComplete` 換成 `ComposeScanCallback(postScanEnrichment, autoGenerator.ScanCallback())`；enrichment 主體未動 |

**Backend tests**

| 檔案 | 動作 |
|---|---|
| `apps/api/internal/subtitle/process_item_freeonly_test.go` | **new** — 13 例（8 例閘門矩陣＋留存性＋skip 不受影響） |
| `apps/api/internal/subtitle/auto_generation_test.go` | **new** — 19 例（11 例政策矩陣＋錯誤中止＋FreeOnly 恆真） |
| `apps/api/internal/subtitle/scan_callback_test.go` | **new** — 6 例（順序／各跑一次／三種 nil／panic 語義） |
| `apps/api/internal/subtitle/cost_consent_free_lane_test.go` | **new** — 9 例成本 guard（走真 Pipeline） |
| `apps/api/internal/database/migrations/031_..._test.go` | **new** — 4 例（既有列預設 OFF／新列預設 OFF／冪等／Down 安全） |
| `apps/api/internal/repository/media_library_repository_test.go` | modified — 測試 schema 補欄位；新增 6 例往返 |
| `apps/api/internal/services/media_library_service_test.go` | **new** — 3 例指標選填語義 |

**Frontend**

| 檔案 | 動作 |
|---|---|
| `apps/web/src/services/mediaLibraryService.ts` | modified — `MediaLibrary.autoSubtitle`、`UpdateLibraryRequest.autoSubtitle?` |
| `apps/web/src/components/settings/LibraryEditModal.tsx` | modified — 第四欄位 checkbox＋兩段說明；Rule 21 檔頭更正 |
| `apps/web/src/components/settings/LibraryEditModal.spec.tsx` | modified — mock 身分穩定化；新增 5 例（共 7） |
| `apps/web/src/components/settings/LibraryCard.tsx` | modified — footer 開關狀態（Rule 24 ①） |
| `apps/web/src/components/settings/LibraryCard.spec.tsx` | **new** — 3 例 |
| `apps/web/src/routes/test/-gallery.fixtures.tsx` | modified — 4 個 library fixture 補 `autoSubtitle`（CR M3 補列） |
| `tests/visual/.../settings-library-edit-modal/{default,hover,focus}-visual-darwin.png` | modified — rebless（CR M3 補列） |
| `tests/visual/.../settings-library-edit-modal/{default,hover,focus}-visual-linux.png` | **deleted** — 交 CI bootstrap（CR M3 補列，已由 #248 補回） |

**未改動（刻意）**：`apps/api/internal/cost_consent_test.go`（`git diff` 0 行）、`WorkerPool`、
`GenerationBatchProcessor`、F14–F20 同意流程、`scanner_service.go`、`SetOnScanComplete` 簽章。

**AC drift 參照來源**：`_bmad-output/implementation-artifacts/sub-4-1-cost-preview-backend.md`
（AC drift reference — see Completion Notes）

## Change Log

| 日期 | 變更 |
|---|---|
| 2026-08-20 | **Task 1（AC #3）** —— `ProcessItemOptions.FreeOnly` 免費層閘門。`translate`／`no_text_source` 在 `FreeOnly` 下於付費呼叫**之前**走新的 `deferPaidItem`：run 記 `skipped`＋理由，media 還原成載入時的原值（不寫路徑、不寫語言）。`MediaItem` 新增 `SubtitleStatus` 快照。13 例新測試。 |
| 2026-08-20 | **Task 2（AC #4）** —— 新建 `ComposeScanCallback`（復刻 sub-1-6 已驗證語義）與 `AutoGenerator`（policy → 列舉 → library/status/cap 三道過濾 → 逐項 `FreeOnly` 處理）。`main.go` 的 scan-complete slot 改為組合，`postScanEnrichment` 主體 byte-unchanged。25 例新測試。 |
| 2026-08-20 | **Task 3（AC #2 BE）** —— migration 031 `media_libraries.auto_subtitle DEFAULT 0`、model 欄位、repo 四處 CRUD、`UpdateLibraryRequest.AutoSubtitle *bool`（指標選填，缺席不覆寫）。零新增 endpoint。13 例新測試。 |
| 2026-08-20 | **Task 4（AC #6/#7）** —— `cost_consent_free_lane_test.go` 走真 Pipeline 斷言付費埠呼叫計數 0，並以 fault injection 反證（閘門停用 → 6 例轉紅）。`cost_consent_test.go` 一字未改。`AutoGenerator` 起訖各一行 `slog.Info` ＋四個計數。 |
| 2026-08-20 | **Task 5（AC #2 FE）** —— `LibraryEditModal` 第四欄位 checkbox＋三句定稿文案（逐字照 E5-D），Rule 21 檔頭更正為 `E5-D (hUVYm) · E5-M (P0P82x) · J4-D (sPzZT)`。**Rule 24 ① 吸收**：`LibraryCard` footer 顯示開關狀態。FE 共 10 例。 |
| 2026-08-20 | **Task 6（AC #8）** —— `nx test api` 全綠、`nx test web` 2698 例全綠、`nx lint api`（釘版 staticcheck-2026.1）綠、`nx lint web` 綠、`format:check` 綠。 |

---

## Senior Developer Review (AI)

**日期：** 2026-08-20 ｜ **審查者：** Amelia（⚠️ **自審** —— 實作者與審查者同一 context，結構上弱於換模型審查）
**結果：** **APPROVED-WITH-FIXES** —— 2 High / 3 Medium / 2 Low，**High 與 Medium 全數修復並以 fault injection 反證**
**修復 PR：** `fix/9R-10b-cr-findings`（story 主體已於 #247 merge）

### 強制檢查

| 檢查 | 結果 |
|---|---|
| 🔒 Rule 7 Wire Format | **PASS**（本 story 的 Go 檔零 error-code 字串常數） |
| 🔒 Rule 20 Contract Bump | **N/A**（零 bump；`FreeOnly` 為 additive-on-v1） |
| 🔒 Rule 25 Mega-line | **N/A**（`project-context.md` 未觸及） |
| Git vs Story File List | **1 筆落差** → M3 |

### Findings

**🔴 H1 —— 付費項目永久佔住配額，把免費項目餓死** · ✅ FIXED
`preflightSkip`（`pipeline.go:581`）**只看 sidecar 是否存在**；被 `deferPaidItem` 煞車的項目永遠沒有 sidecar ⇒
永遠不會被 pre-flight 跳過。而 `FindMissingZhHantSubtitle` 是 `ORDER BY title COLLATE NOCASE ASC, id ASC` ⇒
**每次都是同樣的前 20 筆**。若字母序前 20 筆恰好都是英文軌，則每次掃描都白燒 20 次 ffprobe+ffmpeg 抽取，
而排在後面、真的有內建繁中軌的項目**永遠輪不到** —— 功能看起來完全沒作用。
**修復**：新增匯出常數 `DeferredPaidRunPrefix`（`process_item.go:549`）作為 run `error_message` 前綴，
新增窄介面 `AutoDeferredRunLister`（沿用既有 `ListByStatus`，**零 schema 變更、零新 repo 方法**），
`collect()` 每輪**一次**查詢建立排除集合。port 為選填：未接線時仍正確，只是會重探。
**反證**：拿掉排除 → 4 例轉紅。

**🔴 H2 —— 一個孤兒分集中止整輪，而且原測試的 fake 契約錯誤** · ✅ FIXED
`SeriesRepository.FindByID`（`series_repository.go:108-110`）查無資料時回傳**包 `sql.ErrNoRows` 的 error**，
**不是** `(nil, nil)`。原 `collect()` 對任何 error 都 `return nil, err` ⇒ **一集孤兒分集就讓整輪歸零**，
連前面收好的電影一起丟，且每次掃描重複發生。
**更嚴重的是**：原測試 `TestAutoGenerator_SelectionPolicy` 有一例「episode with an unresolvable series is skipped」**是綠的** ——
因為 fake 對查不到的 id 回 `(nil, nil)`，**比它替身的真物件更寬容**，在斷言一個生產程式碼沒有的行為。
這正是 9R-10a CR M1 的鏡像版，而本 story 的 Dev Notes 還抄了那條教訓。
**修復**：`isSeriesNotFound()` 以 `errors.Is(err, sql.ErrNoRows)` 分流 —— 孤兒跳過、真故障（DB 鎖死）仍中止；
**fake 改為回傳真 repo 的 error 形狀**，並補 `TestAutoGenerator_GenuineSeriesLookupFailureStillAborts` 釘住分界。
**反證**：還原成 error 一律致命 → 4 例轉紅。

**🟡 M1 —— 沒有 single-flight 護欄** · ✅ FIXED
兄弟元件全有（`GenerationCandidateService` 的 `ErrAnalysisRunning`、`GenerationBatchProcessor` 的 `IsRunning`），
`AutoGenerator` 沒有 ⇒ 手動掃描與排程掃描前後腳完成、或掃描撞上手動批次時，兩輪跑同一份清單，
兩個 `ProcessItem` 競寫同一個 sidecar 路徑。
**修復**：`sync.Mutex` + `running` 旗標，第二輪直接 no-op。**反證**：拿掉 → 2 例轉紅。

**🟡 M2 —— 建立模式靜默丟棄勾選** · ✅ FIXED
`handleSave` 只在 edit 分支傳 `autoSubtitle`；FE 與 Go 的 `CreateLibraryRequest` **都沒有這個欄位**。
使用者新建媒體庫時勾選 → 按「建立」→ 設定消失，**零錯誤、零回饋**。原 spec 只測 edit 模式故未捕捉。
**修復**：兩側 `CreateLibraryRequest` 補欄位（create 用**純 bool**，缺席即 false＝正確預設；
與 update 的**指標**選填語義刻意不同）＋ FE modal create 分支傳值 ＋ 4 例新測試（Go 2、FE 2）。
**反證**：拿掉 create 分支的欄位 → 2 例轉紅。

**🟡 M3 —— File List 漏 7 個檔** · ✅ FIXED
git 有但 story 未列：`-gallery.fixtures.tsx` ＋ 6 張 visual 基準。已補入 File List。

**🟢 L2 —— `deferred_paid` 計數方式脆弱** · ✅ FIXED
原以 `outcome.Kind == RouteTranslate/RouteNoTextSource` 判斷，只因 FreeOnly 恆真才正確
（`FreeOnly=false` 時**成功**的翻譯 run 也是同一個 Kind）。改為 `isDeferredOutcome()` 讀 run 的 marker。

**🟢 L1 —— `context.Background()` 無逾時、無關機掛鉤** · ⏭️ **未修，已立案**
卡住的 ffmpeg 會把 goroutine 釘到 process 結束；WorkerPool 有 `Stop()`，這裡沒有對應機制。
修它需要 `main.go` 的 lifecycle 決定（要不要納入 graceful-shutdown、逾時取多久），
超出 CR 的就地修復範圍 ⇒ Rule 24 ③ `bugfix-autogenerator-no-timeout-or-shutdown`。

### 看過、決定保留

- **`AutoGenerationMaxPerRun` 維持套件常數**（非設定）—— 與 `PipelineConcurrencyM1` 同慣例；story AC #2 明文。
- **`ListByStatus(skipped, 0)` 不設上限** —— 排除集合只取 media id，NAS 規模的 skipped 列數在可接受範圍；
  設上限反而會讓排除不完整、H1 部分復發。
- **`deferPaidItem` 仍寫 `skipped` run 而非新 run 狀態** —— `SubtitleRunStatus` 是已出貨 enum，
  延後**確實是一種 skip**，差別在理由，而理由正是 `error_message` 承載的東西。

### 修後閘門（全綠）

`pnpm nx test api` EXIT=0、零 FAIL ｜ `pnpm nx test web` **2704 例 / 234 檔全綠** ｜
`pnpm nx lint api`（釘版 staticcheck-2026.1）綠 ｜ `pnpm nx lint web` 綠 ｜ `pnpm format:check` 綠 ｜
`tsc --noEmit` **147**（＝main 基準，零新增）。
Visual 基準**不受影響**：本次 FE 變更只有 create 分支的 payload 一行，**零 DOM 變動**。

