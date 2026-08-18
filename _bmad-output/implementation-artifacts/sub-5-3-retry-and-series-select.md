# Story 5.3: 失敗重試＋整季/整劇便捷選取 —— F8 失敗列一鍵回同意清單、F15 series 群組勾選

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

**Epic:** `epic-subtitle-pipeline-m3`（M3 第一波，A 群組）· **Risk: 🟡 MED-LOW（純增量 UI＋additive 欄位；同意紅線不可回歸）** · **前端為主（FE 4 task / BE 1 task）**
**Source:** sprint-status `epic-subtitle-pipeline-m3` seed（A:失敗重試＋整季/整劇便捷選取 —— F8 失敗列一鍵回到同意清單預選、F15 series 層級群組勾選;重試必須重新走同意不可繞過）
**Cross-stack split check:** backend tasks = 1, frontend tasks = 4 → 單一 story（規則要求**兩側皆 >3** 才拆；BE 1 未觸發）

---

## Story

As a NAS owner with a TV-heavy library,
I want to re-run just the items that failed with one click, and tick a whole show or season at once instead of hunting for its episodes one row at a time,
so that recovering from a partial batch and consenting to a season's cost both take seconds — while every retry still walks through the same priced-consent screen.

---

## Context — 這個 story 為什麼存在

M3 A 群組（營運強韌性）第一棒。同意流程（sub-4-1/4-2/4-3）把「批次」交付大半之後，剩下兩個日常摩擦點：

**(1) 失敗之後沒有路。** F8 批次面板在終局（complete／error／budget_ceiling）會把失敗列標紅（`failedIds`，dialog 內從 per-item SSE 累積，`GenerationBatchDialogV2.tsx:534,591-600`），但**沒有任何 affordance** —— 使用者只能關掉、重開產生字幕、在整張清單裡憑記憶重找剛剛失敗的那幾部。而重試的全部基建其實都在：`GenerationConsentView` 已支援 `preselectedIds`（D1 圖書館選取路徑，`:59`）、`forceAnalyze` 已存在（CR sub-4-3 H2）、失敗項目的 `subtitle_status` 已回復 retryable（sub-5-1 AC #3）所以重新分析必然重新列舉它們。缺的只是**一條接線**。

**(2) 影集庫的勾選是逐集酷刑。** F15 候選清單是扁平列（`CandidateRow`，每列一個 checkbox），一部 24 集的劇要點 24 下。D1 裁定（2026-08-10）讓影集進了批次，但清單沒有跟著長出 series 層級的操作。候選項目目前**完全沒有 series 身分**（`GenerationCandidate` 只有 media_id/media_type/title，episode 的 title 是「集名 SxxEyy」；`generation_candidates.go:85-95`）—— 所以這半是 BE additive 欄位 ＋ FE 群組化。

### 🚨 authoring 盤點的三個關鍵事實

**(a) 重試必經同意是「結構性」的,不用另外防。** 到達 `onStartBatch` 的唯一路徑是 F16 confirm（`GenerationConsentView.handleConfirm`）；`startGenerationBatch` 在 dialog 內只有 `handleStartConsented` 一個呼叫點。重試 = 回到 consent phase 帶預選,先天不可能繞過金額確認。AC #7 加一條守衛測試把這個結構釘死,防未來長出捷徑。

**(b) 下次繼續 目前會丟掉使用者的付費選擇。** budget_ceiling 的 `handleResume`（`:660-664`）回到 consent 後**零預選** → fallback 到預設選擇（僅 extract）。使用者本來同意的付費 ASR 清單被靜默清空,得整份重勾。這跟 F8 失敗重試是**同一條機制**（終局 → consent 帶預選）,lane ① 吸收為 AC #4。

**(c) 失敗名單是 session-local 的。** `failedIds` 只活在 dialog state;attach 模式（batch 由別的 session 啟動,status probe 不帶 items[]）本來就以「誠實降級」出貨（`disc-2026-07-generation-batch-status-items` 追蹤中）。本 story **不**解 attach 的失敗名單 —— 重試按鈕在 attach 模式不出現（沒有名單就不畫按鈕,capability-honor),AC #5 記錄。

---

## Acceptance Criteria

### AC #1 — BE：候選項目帶 series 身分（additive,無 bump）

`GenerationCandidate` 增三個 additive 欄位:

```go
// episodes only；movies 三者皆零值。FE 以 series_id != "" 判別群組歸屬,
// 不以 season_number 判別（S00 特別篇的 0 是合法值,不可 omitempty 掉）。
SeriesID     string `json:"series_id,omitempty"`
SeriesTitle  string `json:"series_title,omitempty"`
SeasonNumber int    `json:"season_number"`
```

- `models.Episode` 已有 `SeriesID`/`SeasonNumber`（`episode.go:12,15`）;**series title 需查 series 表** → 新窄介面 `CandidateSeriesTitleResolver { FindByID(ctx, id) (*models.Series, error) }`（Rule 11,`*repository.SeriesRepository` 滿足,main.go 注入,建構子加參數 —— `selfHostedASR`/`defaultBudgetUSD` 先例）。
- **sweep 內 memo**:一個 series 只查一次（`map[string]string` 於單次 enumerate 生命週期;20 季的劇不打 500 次 DB）。
- **fail-soft（Rule 13 case 3）**:resolver nil 或單一 series 查詢失敗 → 該列 `series_title` 留空、`series_id` 照填,Debug log 一行,**不得**讓整個 sweep 失敗（sub-5-1 CR M2 的 degrade 精神;FE 對空 title 以 series_id 群組、header 顯示「未知影集」）。
- **Rule 20**:sub-4-1 AC #7 `[@contract-v1]` 候選信封的 additive 欄位 —— 既有 key 不動,**不 bump**,ack ＋ Change Log（sub-5-1 `default_budget_usd` 同款先例）。wire-shape 測試釘 snake_case key。
- 既有排序（title,id）**不動** —— 顯示序由 FE 群組化決定（AC #2）,BE 零行為回歸。

### AC #2 — FE：F15 series／season 群組勾選（三序同源不可破）

`consentSelection.ts` 新增純函式（unit-test 隔離,既有檔案慣例）:

- `groupCandidates(candidates)` → 依 `series_id` 分組:series 區段（首見序）＋ 電影扁平列（不另設「電影」header —— 扁平列是既有核定型態,零漂移）。season 子層 **僅當該劇跨 ≥2 季**才出現（單季劇 series header 即整季,少一層噪音）。
- `groupOrder(candidates)` → 群組化後的**新陣列順序**。🔴 **三序同源紅線**:顯示序 ＝ 送出序 ＝ F18 `feasibleCount` 累加序 —— `seedList` 以 `groupOrder` 重排 `candidates` state 本身,`computeTotals`／`handleConfirm` 零改動地繼承同一順序。三個獨立順序是被 sub-4-3 AC 明文禁止的類別（三處金額同源的順序版）。
- 群組 checkbox 語意:**對群組的全部 listable 項目操作,與 route 篩選 chips 無關** —— chips 是純檢視濾鏡,勾選語意跟既有 全選（`handleToggleAll` 對全體操作,非可見子集）一致,不引入第二套語意。
- header checkbox 三態:全選 checked／部分 `indeterminate`（`ref` 設 `el.indeterminate`,`aria-checked="mixed"`）／全不選 unchecked。`aria-label`:「選取整部 {series_title}」／「選取第 {n} 季」。
- 群組 header 顯示該群組的 小計金額＋route 徽章計數（重用 `computeTotals` 的分類邏輯,不長第二套加總）。
- **設計語彙零新增**:checkbox／route 徽章／金額字體全部重用 `CandidateRow` 既有 token;header 列 = 既有列型的加粗變體。`.pen` 同步走 lane ③（見 AC #6）。

### AC #3 — FE：F8 失敗列一鍵重試（必經同意）

- 終局 footer（complete／error／budget_ceiling）新增 **「重試失敗項目」** 按鈕,**僅當 `failedIds ∩ items ≠ ∅`** 時渲染（attach 模式 items=[] → 永不渲染,AC #5）。
- 點擊 → `resetBatch()` 回到 consent phase,`preselectedIds` = 失敗項目 ids、`forceAnalyze = true`（庫的候選現實剛變過:成功項該消失、失敗項重新列舉 —— CR sub-4-3 H2 的既有語意）。dialog 內以 `retryIds` state 傳遞（`preselectedIds` prop 要求 render-stable 陣列,`:57` 明文）。
- **既有交集語意沿用**:預選與新鮮候選清單交集;交集為空 fallback 預設選擇（shipped 行為,不改）。失敗項因 status 已回復 retryable 必然重新列舉,交集實務上非空。
- 🔴 **同意紅線**:重試路徑**不得**直接呼叫 `startGenerationBatch` —— 唯一入口仍是 F16 confirm 的 `onStartBatch`。守衛測試（AC #7）釘住。

### AC #4 — FE：下次繼續 預選未完成項（lane ① 吸收）

`handleResume`（budget_ceiling）從「零預選」改為預選**未完成項** = `deriveRowStates` 判為 paused／pending ＋ failed 的列（該 selector 已存在,`GenerationBatchDialogV2.tsx:71-92` 一帶;抽出 `remainingIds(items, progress, failedIds)` 純函式供兩個入口共用）。使用者已同意過的付費選擇不再被靜默丟棄;重新定價＋重新確認照舊（forceAnalyze 同 AC #3）。workspace 側的 `onResume`（`GenerationWorkspaceV2.tsx:620`,走 `onLaunch` 開 dialog）**不動** —— workspace 沒有 items/failedIds,沿用其誠實降級。

### AC #5 — attach 模式誠實降級（記錄,不解）

attach 模式（items[] 未知）:重試按鈕不渲染、下次繼續 維持零預選 fallback。根因是 status probe 不帶 items[]（`disc-2026-07-generation-batch-status-items`,既有 backlog）——名單持久化屬該條目,本 story 不擴張。F8 attach 視圖的既有「honest note」文案不動。

### AC #6 — `.pen` 設計同步（lane ③,function-first 裁定）

F15 群組 header 與 F8 重試按鈕是核定設計（2026-08-10 定稿）沒有的結構元素。**authoring 裁定 function-first**:(1) M3 seed 裁定（2026-08-12,晚於設計定稿）明文點名這兩個 affordance,產品方向已裁;(2) 兩者**全部重用**核定語彙（checkbox 列型、半選工具列既已核定部分選取語意、終局 footer 按鈕型）,零新 token／零新元件類;(3) 對照 sub-5-2 F5 先例(程式碼先行,設計註記後補)。立案 `backlog-f15-f8-group-retry-pen-annotation`（雙向）—— Sally 出提示詞 → Inline-Agent → MCP review → 重產 f15-d-v2/f15-m-v2/f8 截圖。**若 Alexyu 要改走 design-first,本 story 降回 blocked 等設計輪** —— 這是明示的可推翻裁定。

### AC #7 — 測試

- **(a) BE**:series 欄位填值（episode 帶三欄、movie 三欄零值）／sweep 內 memo（fake resolver 計數:同 series 多集只查一次）／fail-soft（resolver 錯誤 → 該列 title 空、sweep 照常完成、其餘列不受影響）／wire-shape snake_case（`default_budget_usd` 測試同款）。
- **(b) 選擇器**:`groupCandidates`（分組、首見序、單季不出 season 層、S00 正常分組）／`groupOrder` 重排後 `computeTotals` 的 `feasibleCount` 走新序（三序同源斷言:顯示序陣列 === 送出 ids 序）／`remainingIds`（paused＋pending＋failed;成功列排除）。
- **(c) Panel**:series header 渲染＋小計／三態 checkbox（含 `aria-checked="mixed"`）／群組 toggle 對全群組操作且不受 chips 影響／a11y label。
- **(d) Dialog**:終局出現重試按鈕（failedIds 非空）→ 點擊回 consent 且 `GenerationConsentView` 收到 preselectedIds＋forceAnalyze／attach 模式（items=[]）不渲染按鈕／**同意紅線守衛:整條重試流程 `startGenerationBatch` 零呼叫,直到 F16 confirm**／下次繼續 預選未完成項。
- **(e) 視覺基準線**:consent 相關 gallery fixtures（`-gallery.fixtures.tsx:3695+`）若含 episode 資料,群組 header 會改變渲染 → 重產受影響 fixtures 的 `-darwin` 基準線;`-linux` 由 CI `Visual Regression` workflow 的 bootstrap PR 補（CLAUDE.md 慣例,**不得**本機產 -linux）。必要時為「含群組的 F15」新增 fixture（Rule 22 邊界內）。
- 全回歸閘門:`go test ./...`、`pnpm nx test web`、`pnpm run lint:all`、`format:check`。

---

## Tasks / Subtasks

- [x] **Task 1 — BE additive series 欄位（AC: #1）** 🔴 BE
  - [x] `GenerationCandidate` 三欄＋`CandidateSeriesTitleResolver` 窄介面＋建構子參數＋main.go 注入
  - [x] sweep 內 memo＋fail-soft＋測試（填值／memo 計數／degrade／wire shape）＋Rule 20 ack

- [x] **Task 2 — 群組選擇器與三序同源（AC: #2）** 🟡 FE
  - [x] `consentSelection.ts`:`groupCandidates`／`groupOrder`／`remainingIds` 純函式＋spec
  - [x] `seedList` 以 `groupOrder` 重排 candidates state（三序同源斷言）

- [x] **Task 3 — CandidateListPanel 群組渲染（AC: #2）** 🟡 FE
  - [x] series／season header 列（小計＋徽章計數＋三態 checkbox＋a11y）;電影列零改動
  - [x] 群組 toggle 接 `GenerationConsentView`（對全群組操作,chips 無關）

- [x] **Task 4 — F8 重試與 下次繼續 預選（AC: #3, #4, #5）** 🟡 FE
  - [x] 終局 footer 重試按鈕（條件渲染）＋`retryIds` state＋consent 預選接線
  - [x] `handleResume` 改預選 `remainingIds`;attach 降級確認（不渲染）＋同意紅線守衛測試

- [x] **Task 5 — 測試、基準線與收尾（AC: #6, #7）**
  - [x] 視覺基準線:受影響 consent fixtures 重產 `-darwin`（`-linux` 走 CI bootstrap PR）
  - [x] 立案 `backlog-f15-f8-group-retry-pen-annotation`（雙向）＋契約清點（1 ack、0 bump）＋全回歸

（後端 task 1 個、前端 4 個 —— 未觸發跨端拆分門檻。）

---

## Dev Notes

### 既有可重用零件（不要重造）

| 需求 | 現成零件 |
| --- | --- |
| consent 預選 | `GenerationConsentView.preselectedIds`（`:59`,D1 路徑;交集＋fallback 語意已定） |
| 強制重新分析 | `forceAnalyze`（CR sub-4-3 H2;`postTerminal` 已在終局後設 true —— 重試路徑天然拿到新鮮清單） |
| 失敗名單 | `GenerationBatchDialogV2` `failedIds` state（`:534`,per-item SSE 累積,running 時才記錄） |
| 列狀態推導 | `deriveRowStates(items, progress, failedIds)`（dialog 檔內;`remainingIds` 從它抽） |
| 群組小計 | `computeTotals` 的 route 分類邏輯（三處金額同源;群組小計走同一分類,不另起爐灶） |
| BE 建構子注入先例 | `selfHostedASR`／`defaultBudgetUSD` 參數（`generation_candidates.go:201-208`） |
| additive-no-bump 先例 | sub-5-1 `default_budget_usd`（同一個 [@contract-v1] 信封,ack 格式照抄） |
| 窄介面先例 | `CandidateMovieFinder`／`CandidateEpisodeFinder`（`:56-66`;Rule 11） |
| 三態 checkbox a11y | 無 in-repo 先例 —— `el.indeterminate` ＋ `aria-checked="mixed"`,jsx-a11y 掃過 |
| 視覺基準線流程 | 19-4/19-5 慣例:`pnpm run test:visual`＋`-darwin` 本機、`-linux` CI bootstrap PR |

### 關鍵決策（authoring 已裁）

- **series title 由 BE 供給**而非 FE 另打 series API:候選信封是這張畫面唯一的資料來源(sub-4-1 設計),讓 FE 為 header 發 N 個 series 請求違反該設計且引入載入態複雜度。
- **`season_number` 不 omitempty**:S00 特別篇的 0 是合法值;FE 以 `series_id != ""` 判別,不以 season 判別。
- **群組 toggle 對全群組**（非可見子集）:與既有 全選 語意一致;chips 維持純檢視濾鏡。兩套勾選語意並存是使用者無法建立心智模型的類別。
- **三序同源用「重排 state」實現**而非渲染層映射:改一處(`seedList`),`computeTotals`/`handleConfirm`/`feasibleCount` 零改動繼承 —— 渲染層映射會讓送出序與顯示序各自維護,正是要防的漂移。
- **重試按鈕只在 dialog**（F8）:workspace 沒有 failedIds(container 從未接,`GenerationWorkspaceV2.tsx` 只有 gallery fixture 在餵),它的終局已有 onResume→onLaunch 路徑;為 workspace 補失敗名單屬 `disc-2026-07-generation-batch-status-items` 的持久化範圍。
- **function-first vs design-first**（AC #6）:明示可推翻。理由三點記錄在 AC 內;推翻成本 = 本 story 轉 blocked、先跑 Sally 設計輪。
- **下次繼續 預選吸收進本 story**（lane ①）:與 F8 重試同一條機制(終局→consent 帶預選),分開做會兩次動同一個 footer。

### seam 資料層觸及（retro-m2-AI3 慣例）

- `GenerationCandidateService`:enumerate 內新增 series 表讀（memo 化,每 series 一次）;零新表、零 migration。
- `SeriesRepository.FindByID`:既有方法,零改動。
- FE:純 state／渲染;`subtitleService.GenerationCandidate` 型別加三個 optional 欄位(snakeToCamel 通用映射,sub-5-1 已驗)。
- **零新 Rule 7 error code**（prefix 數維持 16）、零 SSE 事件改動、零 endpoint 改動。

### 已知限制（記錄,不在本 story 解）

- attach 模式無失敗名單 → 無重試按鈕（AC #5;`disc-2026-07-generation-batch-status-items` 追蹤持久化）。
- `failedIds` 不跨 session:重開頁面後終局面板消失,失敗項只能從重新分析的清單手動勾（候選清單本來就會列出它們 —— 資訊不丟,只是少了預選便利）。
- `.pen` F15/F8 在 lane ③ 補齊前,設計與實作在群組 header／重試按鈕上不一致（已立案、雙向、明示裁定）。
- 群組 header 的小計是**選中項**小計還是**全群組**小計 —— authoring 裁定:顯示「已選 n/N ・ $小計(選中)」,與 footer 的選中總額語意一致;dev 若發現排版塞不下,降級為只顯示 n/N 計數並記錄。

### 契約姿態（Rule 20）

- 消費 sub-4-1 AC #7 `[@contract-v1]`（候選信封）:additive 三欄,**不 bump**,ack ＋ Change Log ＋（`generation_candidates_handler.go` 已有 inline stamp,sub-5-1 補的 —— 無需再補）。
- `GenerationBatchItem`／batch 202 shape／`generation_batch_progress` SSE／D2／D6:**全部不動** ⇒ 0 bump ⇒ 無 stale-mark 義務。
- 本 story 不產生新 stamp（群組化是渲染層行為,`groupCandidates` 是 repo 內純函式,無跨 story 消費者）。

### Time-dependent visual coverage

`N/A — no wall-clock-reading components touched.`（群組化與重試皆純狀態推導;Rule 23 掃過:無 `Date.now()`／`new Date()` 新增。）

### References

- [Source: sprint-status `epic-subtitle-pipeline-m3` seed] — A 群組:F8 失敗列預選重試、F15 series 群組勾選、重試必經同意
- [Source: `apps/web/src/components/subtitle/GenerationBatchDialogV2.tsx:508-720`] — failedIds 累積、handleResume 零預選現況、preselectedIds 通道
- [Source: `apps/web/src/components/subtitle/consent/GenerationConsentView.tsx:50-109`] — preselectedIds 交集語意、forceAnalyze、seedList
- [Source: `apps/web/src/components/subtitle/consent/consentSelection.ts`] — computeTotals／feasibleCount 走序、defaultSelection
- [Source: `apps/web/src/components/subtitle/consent/CandidateListPanel.tsx:50-110`] — CandidateRow 列型（header 列的語彙來源）
- [Source: `apps/web/src/components/subtitle/GenerationWorkspaceV2.tsx:530-628`] — workspace container 無 failedIds、onResume→onLaunch
- [Source: `apps/api/internal/services/generation_candidates.go:56-115,168-208,460-560`] — 候選型別、窄介面、enumerate、episodeTitle、排序
- [Source: `apps/api/internal/models/episode.go:12-15`] — SeriesID／SeasonNumber
- [Source: `apps/api/internal/repository/series_repository.go:102`] — FindByID（title 查點）
- [Source: `apps/api/internal/repository/episode_repository.go:157-164`] — 列舉排序（series,season,episode）
- [Source: `apps/web/src/routes/test/-gallery.fixtures.tsx:3695+`] — consent fixtures（基準線影響面）
- [Source: sprint-status `disc-2026-07-generation-batch-status-items`] — attach 模式 items[] 缺口（AC #5 的邊界）
- [Source: `project-context.md`] — Rule 11/13/20/21/22/23/24

---

## Dev Agent Record

### Agent Model Used

Claude Fable 5 (claude-fable-5)

### Debug Log References

- Full API suite: `go test ./... -count=1` — 34 packages ok, 0 failures; `go vet` + `gofmt -l` clean.
- Full web suite: `pnpm nx test web -- --run` — 233 files / **2648** tests green (+28 by this story).
- `pnpm run lint:all` — 0 errors; prettier clean（3 檔first-pass 格式修正後全綠）。
- Visual: `playwright test --project=visual --update-snapshots` — 1 passed；**只有新 fixture 的基準線被寫入**（`git status` 證明既有 F15/F18 baselines byte-identical ⇒ 舊列型零視覺回歸）。
- Falsification: 重試預選接線（`retryIds ?? selectedMediaIds`）被刻意破壞後 2 個 spec 轉紅,復原後全綠 —— 測試真的在守。
- 本機 webServer 需 `GEMINI_API_KEY=visual-dummy`（見 Discovery Triage lane ③）。

### Completion Notes List

- **🔗 AC Drift: NONE**（grep `preselectedIds|下次繼續|feasibleCount|重試` 過 `_bmad-output/implementation-artifacts/*.md` → hits 在 sub-4-2/sub-4-3/ux3-ai-2,逐一讀過皆 REUSE:preselectedIds 交集語意、forceAnalyze、consent-必經路徑全部沿用不改;下次繼續 的行為變更（零預選→預選未完成項）是本 story AC #4 的明文交付,且 sub-4-3 的 AC 只規定「回到 consent、重新確認」——預選內容不在其契約面）。
- **📎 Contract Stamps: FOUND（1 upstream ack;0 bumps）** — **confirmed against `[@contract-v1]` (Story sub-4-1 AC #7)**:候選信封 additive 四欄（見 AC #1 note）,既有 key 不動,不 bump（sub-5-1 `default_budget_usd` 先例);inline stamp 註解已在 handler（sub-5-1 補),無需再補。本 story 產生 0 個新 stamp ⇒ 無下游 stale-mark 義務。
- **🎭 A11y Pre-Flight: PASS**（6 個 FE 檔 scoped eslint 0 warnings、0 introduced）。手動四類:無新圖片/無 modal 結構變更/無 async-reveal 變更;新自訂 widget = 三態 checkbox —— 原生 `<input type="checkbox">`（鍵盤免費）+ `aria-label`（選取整部/選取第 n 季）+ `aria-checked="mixed"` + `el.indeterminate`（`consent-select-all` 既有 idiom）。
- **🎨 UX Verification: 依 AC #6 function-first 裁定** — F15 群組 header 與 F8 重試按鈕重用核定語彙（checkbox 列型/終局 footer 按鈕型/`--error-tint` 徽章色 = 失敗列既有色);`.pen` 註記由 `backlog-f15-f8-group-retry-pen-annotation` 追蹤（雙向已立）。新 gallery fixture 的 header 帶 design-coverage-gap 註解（Rule 21 第 4 形式）。
- **AC #1** — `GenerationCandidate` + `series_id`/`series_title`(omitempty) + `season_number`/`episode_number`(**非** omitempty,S00/E00 為合法零值)。**Authoring 修正（in-flight,記錄於此）**:AC 原列三欄,實作加了第四欄 `episode_number` —— 季內顯示序的唯一可靠來源（BE 全域排序是 (title,id),集名互異時會打亂集序;解析 title 的 SxxEyy 子字串是脆弱替代)。`CandidateSeriesTitleResolver` 窄介面 + 建構子參數 + main.go 注入 `repos.Series`;sweep 內 memo（每 series 一次,含失敗 memo）;fail-soft（nil resolver／查詢失敗 → title 空、id 照填、sweep 照常）。7 個 BE 測試含 memo 計數與 wire-shape。
- **AC #2** — `groupCandidates`/`groupOrder` 純函式:電影扁平段先、series 首見序、季內 season→episode 升冪、單季不出 season 層、S00=特別篇。**三序同源**以 `seedList` 重排 state 實現（一處改,`computeTotals`/`handleConfirm` 零改動繼承);view-seam 測試釘住送出序 = 群組序 ≠ 後端 title 序。Panel:series/season header 列（已選 n/N ・ 選中小計,金額 verbatim from `estimated_usd`）、三態 checkbox、群組 toggle 對全群組（chips 純檢視濾鏡,spec 釘住）。無 seriesId 的列（舊伺服器/電影）維持出貨版扁平渲染,spec 證明零 header。
- **AC #3** — 終局 footer「重試失敗項目」按鈕（`--error-tint`,`gen-batch-retry-failed-btn`）:僅當 `failedRowIds` 非空才由 container 傳入 `onRetryFailed` ⇒ attach 模式（items=[]）永不渲染。`failedRowIds` 從 `deriveRowStates` 推導 —— 預選集合 = 使用者「看到」標紅的列（budget_ceiling 被中斷的 in-flight 項 renders 已暫停 ⇒ 歸 remainingIds,spec 釘住 9R-16 caveat 的一致性）。點擊 → `retryIds` state → consent `preselectedIds`,`postTerminal` 既有機制供給 forceAnalyze。**同意紅線測試**:整條重試流程 `startGenerationBatch` 呼叫數不變,直到 F16 confirm。
- **AC #4** — `handleResume` 預選 `remainingIds`（paused+stopped+failed）;consumed-on-start:`handleStartConsented` 開頭清 `retryIds`,spec 證明後續 clean ceiling 的 下次繼續 零殘留。workspace 側 `onResume→onLaunch` 不動。
- **AC #5** — attach 降級由「無 items ⇒ failedRowIds 空 ⇒ onRetryFailed undefined ⇒ 按鈕不渲染」的鏈自然成立,panel spec 直接釘 absent-prop 不渲染。
- **AC #7(e)** — 新 gallery fixture `generation-consent/grouped`（1 電影 + 雙季影集含 S00）;`-darwin` 基準線已產,**`-linux` 由 CI Visual Regression workflow 的 bootstrap PR 補**（CLAUDE.md 慣例）。既有 consent fixtures 補 `onToggleGroup: noop`（新必要 prop）,渲染 byte 不變（update-snapshots 全跑後 git 只見新檔）。
- **Pre-existing fix**: error-phase 重試按鈕以零參數呼叫 `bootstrap()`（CR sub-4-3 M3 加了必要的 `isCancelled` 守衛參數後未同步此呼叫點）→ runtime TypeError,重試永遠只會再顯示錯誤。修為 `bootstrap(() => false)` + 新 spec 證明重試真的重新載入到 list phase。tsc 噪音中的其餘錯誤為 repo 既有（stash 驗證 506 行 baseline）,不屬本 story。

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - **YES** — filed at authoring time（2026-08-18）：
    - **① expand-scope-in-place** — budget_ceiling 的 下次繼續 回 consent 後零預選,使用者已同意的付費選擇被靜默丟棄（fallback 到僅 extract 的預設選擇）→ 吸收為 **AC #4**（與 F8 重試同一條「終局→consent 帶預選」機制）。
    - **③ backlog-with-carry-forward-link** — `backlog-f15-f8-group-retry-pen-annotation`：F15 群組 header 與 F8 重試按鈕的 `.pen` 設計註記（Sally ＋ Inline-Agent 流程;function-first 裁定見 AC #6,明示可推翻）。非阻塞。
  - 既有條目確認不重複立案:attach 模式名單缺口已由 `disc-2026-07-generation-batch-status-items` 追蹤（AC #5 引用,不另立）。
  - **YES** — filed at implementation time（2026-08-18）：
    - **③ backlog-with-carry-forward-link** — `backlog-visual-webserver-needs-ai-key`：本機 `pnpm run test:visual` 的 playwright webServer 直接 `go run ./cmd/api`,而 `AI_PROVIDER` 預設 gemini 且無金鑰時 `NewAIService` 失敗 → `os.Exit(1)` → **無金鑰的開發機跑不了視覺測試**（本次以 `GEMINI_API_KEY=visual-dummy` 繞過）。sub-5-2 才剛確立「keyless boot 不得癱瘓服務」的方向,parse-path AI service 卻仍是 boot-fatal。非阻塞（CI 自帶 env）。

### File List

**Backend**

- `apps/api/internal/services/generation_candidates.go` — AC #1: 四個 additive 欄位、`CandidateSeriesTitleResolver`、`resolveSeriesTitle` fail-soft、sweep 內 memo、candidateRow series 身分
- `apps/api/internal/services/generation_candidates_test.go` — 7 個 sub-5-3 測試（填值/S00/memo 計數/fail-soft/nil resolver/wire shape ×2）+ 建構子 call sites 機械遷移
- `apps/api/cmd/api/main.go` — `repos.Series` 注入候選服務

**Frontend**

- `apps/web/src/services/subtitleService.ts` — `GenerationCandidate` 四個 optional 欄位
- `apps/web/src/components/subtitle/consent/consentSelection.ts` — `groupCandidates`/`groupOrder` + `CandidateGroup`/`CandidateSeasonSection` 型別
- `apps/web/src/components/subtitle/consent/consentSelection.spec.ts` — 9 個群組/三序同源測試
- `apps/web/src/components/subtitle/consent/CandidateListPanel.tsx` — `GroupHeaderRow`（三態 checkbox + 已選小計）、`seasonLabel`、群組渲染、`onToggleGroup` prop
- `apps/web/src/components/subtitle/consent/CandidateListPanel.spec.tsx` — 9 個群組渲染/語意測試 + harness 補 prop
- `apps/web/src/components/subtitle/consent/GenerationConsentView.tsx` — `seedList` 以 `groupOrder` 重排 state、`handleToggleGroup`、pre-existing bootstrap 重試修復
- `apps/web/src/components/subtitle/consent/GenerationConsentView.spec.tsx` — 3 個 view-seam 測試（送出序/群組 header/重試修復）
- `apps/web/src/components/subtitle/GenerationBatchDialogV2.tsx` — `failedRowIds`/`remainingIds` 純函式、panel `onRetryFailed` + 重試按鈕、container `retryIds` state 與接線
- `apps/web/src/components/subtitle/GenerationBatchDialogV2.spec.tsx` — 9 個測試（selector ×3、panel ×3、container 重試/預選/清除 ×3,含同意紅線守衛）
- `apps/web/src/routes/test/-gallery.fixtures.tsx` — 新 `generation-consent/grouped` fixture + 既有兩個 consent fixtures 補 `onToggleGroup: noop`

**Visual**

- `tests/visual/components.visual.spec.ts-snapshots/components/generation-consent/grouped/default-visual-darwin.png` — 新基準線（`-linux` 走 CI bootstrap PR）

**Docs / tracking**

- `_bmad-output/implementation-artifacts/sprint-status.yaml` — sub-5-3 ready-for-dev → in-progress → review

---

## Change Log

| Date | Change |
| --- | --- |
| 2026-08-18 | create-story：M3 A 群組建檔。F8 失敗重試（結構性必經同意）＋ F15 series/season 群組勾選（BE additive 三欄,無 bump）。lane ①×1（下次繼續 預選未完成項）＋ lane ③×1（`.pen` 註記）。function-first 裁定明示可推翻（AC #6）。 |
| 2026-08-18 | Task 1 (AC #1)：BE additive 四欄（authoring 三欄 + in-flight 補 `episode_number` 供季內確定性排序）+ 窄介面 + memo + fail-soft + main.go 注入;7 測試。 |
| 2026-08-18 | Task 2-3 (AC #2)：`groupCandidates`/`groupOrder` + seedList 重排 state（三序同源一處改）+ Panel 群組渲染（三態 checkbox/已選小計/特別篇）;18 FE 測試;舊伺服器扁平渲染零回歸（baselines byte-identical）。 |
| 2026-08-18 | Task 4 (AC #3/#4/#5)：`failedRowIds`/`remainingIds`（自 deriveRowStates 推導,與使用者所見一致）+ 終局重試按鈕 + `retryIds` 預選接線 + 下次繼續 預選未完成項 + consumed-on-start 清除;同意紅線守衛測試（startGenerationBatch 零直呼);attach 降級 absent-prop 不渲染。Pre-existing fix:error-phase bootstrap 重試 TypeError。 |
| 2026-08-18 | Senior Developer Review (Opus 5 adversarial, 換模型慣例 — implementation by Fable 5) — 1H/2M/5L, all adjudicated and fixed in-session. **H1**: AC #2 明文要求群組 header「route 徽章計數（重用 computeTotals,不長第二套加總）」—— 徽章根本沒做,小計還是自己 `reduce` 的第二套加總(Completion Notes 反而聲稱 "no second totals engine")→ header 改走 `computeTotals(items, selectedIds, null)`,同時補上選中 route 徽章。**M1**: 重試失敗項目 在 budget_ceiling 與 下次繼續 並排,而 failed ⊂ remaining ⇒ 點窄的那顆會靜默丟掉使用者已同意的暫停項 —— 正是 AC #4 要防的損失 → 該按鈕限 complete/cancelled/error。**M2**: series 區段用「首見序」,而首見序來自後端 per-EPISODE 的 (title,id) 排序 ⇒ 影集順序在字母排序的電影列旁看起來隨機,且新增一集早排序的集名會讓整部劇跳到最前 → 改依 series title 排序,未知影集沉底。**L1** failedRows 每 render 新陣列使自己的 useCallback 失效(bugfix-19-4b-1 同類)→ useMemo。**L2** groupCandidates+Set 每 render 重算(預算輸入每個按鍵都重排 1,200 項)→ useMemo。**L3** 原生 checkbox 上多餘的 aria-checked,與出貨的 consent-select-all idiom 不一致 → 移除,只用 native indeterminate。**L4** 兩個 `TestAnalyze_*` 從不呼叫 Analyze(實為 struct wire-shape 測試)+ struct 註解 "all three" 已成四欄 → 更名/更正。**L5** 群組列全被 chips 過濾掉時不畫 header 的分支無測試 → 補。+5 測試;grouped 視覺基準線因 header 改版重產。修後全綠:api 34 pkg、web 233/2653、lint 0 errors、visual 1 passed。Status review → done。 |
| 2026-08-18 | Task 5 (AC #6/#7)：新 grouped gallery fixture + `-darwin` 基準線（唯一新檔,既有 byte-identical);lane ③ 已於 authoring 立案;全回歸綠（api 34 pkg、web 233/2648、lint 0 errors、visual 1 passed）。Status in-progress → review。 |

---

## Senior Developer Review (AI)

**Reviewer model:** Claude Opus 5（換模型 adversarial CR 慣例 — implementation by Fable 5）· **Date:** 2026-08-18 · **Outcome:** Changes Requested → all findings adjudicated and fixed in-session → **Approve (done)**

**Mandatory checks:** 🔒 Rule 7 Wire Format: PASS（in-review Go 檔 0 個新 error-code 常數;prefix 數維持 16）· 🔒 Rule 20 Contract Bump: N/A（0 bumps;消費側 ack 行 `confirmed against [@contract-v1] (Story sub-4-1 AC #7)` 已驗證存在,additive 四欄不 bump 成立）· 🔒 Rule 25 Mega-line: N/A（project-context.md 未觸及）· **Git vs File List: 0 discrepancies**（14 modified + 新快照目錄,逐一對帳）。

### Findings & resolutions（1H / 2M / 5L）

- [x] **[H1] AC #2 的群組 header 有一半沒做,而 Completion Notes 聲稱做了。** AC #2 明文:「群組 header 顯示該群組的 小計金額＋**route 徽章計數**（**重用 `computeTotals` 的分類邏輯,不長第二套加總**）」。實作只有小計、沒有徽章,且小計是 `selected.reduce((s, i) => s + i.estimatedUsd, 0)` —— 一套自己長出來的加總,正是 AC 禁止的東西;Completion Notes 卻寫 "amounts come verbatim… no second totals engine"。FIXED:`GroupHeaderRow` 改用 `computeTotals(items, selectedIds, null)`（ceiling 給 null —— 預算裁決是 footer 的全域職責）,一次拿到 `selectedCount`/`selectedExtractCount`/`selectedAsrCount`/`selectedTotalUsd`;補上選中 route 徽章（抽取 n / 語音辨識 m,計數為 0 時不畫）。+2 測試。
- [x] **[M1] `重試失敗項目` 在 budget_ceiling 與 `下次繼續` 並排,而前者是後者的真子集。** `deriveRowStates` 在 budget_ceiling 下把尾端 `pausedCount` 列標 paused、之前的失敗列標 failed ⇒ 兩顆按鈕同時出現,點窄的那顆只預選 failed,**靜默丟棄使用者已同意的 paused 項**。這與本 story AC #4 的立案理由（「已同意的付費選擇不再被靜默丟棄」）自相矛盾。FIXED:重試按鈕限 `complete`/`cancelled`/`error`;budget_ceiling 交給語意完整的 `下次繼續`。+1 測試釘住兩顆按鈕的分工。
- [x] **[M2] series 區段的「首見序」實際上是隨機序。** 首見序由後端 **per-EPISODE** 的 `(title, id)` 排序決定 —— 影集區段的順序因此與影集本身無關:在字母排序的電影列旁看起來隨機,且**新增一集排序靠前的集名就能讓整部劇跳到清單最前**。FIXED:series 區段改依 series title 排序,未知影集（BE 查詢降級的空 title）沉底,id 作決定性 tie-break;`groupOrder` 是同一個函式 ⇒ 顯示/送出/feasible 三序仍同源。+1 測試,原「首見序」測試改寫。
- [x] **[L1] `failedRows` 每 render 產生新陣列,使 `handleRetryFailed` 的 `useCallback` 完全失效**（deps 含該陣列）—— repo 有 bisect regression gate 專門守這個 unstable-callback-prop 類別（bugfix-19-4b-1）。今天無功能影響（panel 未 memo）,但屬純儀式 + 每次 SSE tick 重走全部列。FIXED:`useMemo`。
- [x] **[L2] `groupCandidates(candidates)` 與 `visibleIds` Set 每 render 重算。** 分組會對每個 series 桶排序,而本 panel 在**預算輸入每個按鍵**都重繪 ⇒ 1,200 項的媒體庫每打一個字就重排一次（改版前只有一次 O(n) filter）。FIXED:兩者都 `useMemo`。
- [x] **[L3] 原生 checkbox 上多餘的 `aria-checked`。** native `indeterminate` 已經對 AT 曝露 mixed;額外的 `aria-checked` 是冗餘 ARIA、有與原生狀態脫鉤的風險,且**與出貨的 `consent-select-all` idiom 不一致**（那顆只用 ref+indeterminate)。原測試還把這個寫法釘住了。FIXED:移除 attribute,測試改斷言 native `indeterminate`/`checked` 且 `not.toHaveAttribute('aria-checked')`。
- [x] **[L4] 測試名不符實 + 註解漂移。** `TestAnalyze_SeriesFieldsSerializeAsSnakeCase` / `TestAnalyze_MovieWireShapeOmitsSeriesKeys` **都沒有呼叫 `Analyze`**（是 struct tag 的 wire-shape 測試）—— 與 sub-5-2 CR L2 同一類;另 struct 註解寫 "movies leave all **three**",in-flight 補了 `EpisodeNumber` 後已是四欄。FIXED:更名 `TestGenerationCandidate_*`、註解更正。
- [x] **[L5] 分支無測試:群組的列被 chips 全數過濾時 header 應完全不畫**（否則會出現一顆管不到任何可見列的 checkbox）。實作是對的,但零覆蓋。FIXED:補測試（100% asr 的影集 + `filter='extract'`）。

### Reviewer verifications that held（未再由 orchestrator 重查）

wire→FE 映射走 `snakeToCamel` 泛型轉換 ⇒ 四個新欄位真的到得了前端（非只在單元測試的手構物件裡成立）· `startTracking` 的 `START` reducer 同步設 `status:'running'`,因此 `setRetryIds(undefined)` 不會在 consent 仍掛載時改變 `preselectedIds` 而觸發整輪 re-bootstrap（本來是我最擔心的 HIGH,查證後不成立）· start 失敗時 `retryIds` 保留（`setRetryIds` 在 await 之後、catch 之外）· `remainingIds`/`failedRowIds` 走 `deriveRowStates` ⇒ 與使用者所見一致,9R-16 的 paused-勝過-failed 裁定被沿用 · bisect/visual gate 皆無硬編 fixture 數（`toBeGreaterThan(0)`）⇒ 新 fixture 不破閘 · E2E consent stubs 全是電影、無 `series_id` ⇒ 群組化對 e2e 惰性 · `usd()` 走 `toFixed(2)` ⇒ 浮點求和無顯示漂移 · Map 迭代序對 UUID 字串鍵為插入序（M2 修正後已不依賴它）· 既有 F15/F18 baselines byte-identical。

### ⚠️ 合併前須知（非 findings）

本 story 新增了一個 gallery fixture,`-linux` 基準線只能由 CI 產生 ⇒ **PR 的 `Visual Regression / PR` check 會紅**,直到 `Visual Regression / Main` 自動開出的 `chore(visual): bootstrap N missing -linux baselines` PR 合併為止（CLAUDE.md 明文流程;**不可**本機產 `-linux`）。