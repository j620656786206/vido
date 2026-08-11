# Story 4.3: 成本同意篩選畫面 —— F14–F20（前端）

Status: ready-for-dev

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a NAS owner,
I want a screen that shows me exactly which files would get subtitles, by which route, for roughly how much — and lets me pick, set a ceiling, and confirm before a single cent is spent,
so that pressing "產生字幕" can never again silently commit my whole library to paid work.

## Context — 這個 story 為什麼存在

M2.5 的最後一棒。BE 讀側（sub-4-1）與寫側（sub-4-2）皆已 done：候選清單／路線預測／估價 API、`budget_usd`、混合電影＋影集批次、route-honest 引擎全部就緒。設計 8 個畫面三輪定稿（第三輪 2026-08-11：抽取路線據實顯示小額翻譯費，PR #216）。**本 story 把消費面補上**：F14 分析中 → F15 候選清單（F15-M 手機）→ F16/F19 金額確認 → 既有 F8 執行畫面；F17 掃描完成入口；F18 超限狀態；F20 空狀態。

**產品裁定鏈**（實作不得偏離）：2026-08-07 三件一體（掃描只 metadata＋篩選畫面＋總預算上限）；D1 批次支援影集（2026-08-10）；抽取標籤據實顯示（2026-08-11，§5-sexies——FE **不做**「免費」四捨五入，直接渲染後端 `estimated_usd`）。

## Acceptance Criteria

1. **F14 分析中。** 開啟產生字幕流程時：先 `GET /subtitles/generation-candidates` 讀狀態信封；`status=idle|cancelled|error` → `POST .../analyze`（202；409 `TRANSCRIPTION_ANALYSIS_RUNNING` 視同已在分析，**不是錯誤**——比照 `transcriptionService.ts:46-52` 的雙重判別 discriminated-union 模式，不 throw）；`analyzing` → 顯示 F14：進度條＋「分析字幕軌 {analyzed} / {total}」＋「…本機執行，不會產生費用」＋次要按鈕「取消」（`POST .../analyze/cancel`）。進度由新 SSE hook 驅動（`generation_candidates_progress`，payload `{status, analyzed, total, error}`）；`ready` → 轉 F15（result 走 GET 取，事件不含 result——sub-4-1 CR L2 既定契約）。**Lazy-connect 鐵律**：比照 `useGenerationBatchProgress.ts` 逐行慣例（`startTracking` 觸發連線、絕不 mount 即連、terminal＝`ready|cancelled|error` 即關流、10s backoff、`mountedRef` 守衛）。

2. **F15 候選清單（核心）。** `result.candidates` 渲染清單：每列 checkbox＋縮圖＋主標（影集格式「{title} SxxEyy」——後端 `title` 已含 SxxEyy，FE 不重組）＋路線副標（可抽取「內嵌英文字幕 → 翻譯」／需辨識「無文字字幕軌 → 語音辨識 + 翻譯」／`runtime_known=false`「片長未知，以 45 分鐘估算」＋金額前 `≈`）＋右側路線徽章與金額（抽取＝success 綠、語音辨識＝warning 橘，**金額一律直接渲染 `estimated_usd`**，`$X.XX` 用既有 `usd()` 格式——先抽成共用 helper，目前重複於 `GenerationBatchDialogV2.tsx:105` 與 `GenerationWorkspaceV2.tsx:54`）。**`route=skip` 項目不入清單不入計數**（後端 skipped 計數不報價；設計無 skip 列語彙）。篩選 chips「全部 {n}」「可抽取內嵌 {extract_count}」（帶「僅翻譯費」標記）「需語音辨識 {asr_count}」（付費標記），沿用 `aria-pressed` chip 慣例（dialog L341-379，`h-11`）。工具列：全選 checkbox（indeterminate 用 `DownloadsTableV2.tsx:88-99` 的 ref 慣例）＋「已選 {x} / {n}」＋「選取全部可抽取（僅翻譯費）」＋「清除選取」。**預設選取＝全部 extract 路線項目**（準則④ 最低成本；付費 ASR 預設不選）。摘要條／footer 左段／明細**三處金額同源即時一致**（單一 selector 計算，禁止三份獨立加總）。F15-M：手機 bottom-sheet 版（既有 Radix className 切換慣例，dialog L308-323；工具列縮短標籤「全部可抽取」「清除」）。

3. **預算上限與 F18 超限。** 預算輸入框：預填 `$5.00`、client 端驗證 > 0（鏡射 server 400 規則）；**送出時一律帶 `budget_usd`＝畫面顯示值**（WYSIWYG 同意——使用者確認的數字就是強制的上限；env 預設是 legacy 對話框的 server 端 fallback，不是本流程的資料來源）。當「已選預估總額 > 上限」→ F18 狀態：摘要條金額轉 warning 橘、輸入框 warning 邊框、footer 上方警示列「預估 $X 已超過上限 $Y —— 預計可完成約 **N** 部後暫停，其餘保留在佇列，可提高上限或稍後續跑。」（**N＝依清單順序對已選項目累計估價、未達上限的項目數**——與後端「依序處理、呼叫前檢查」語意一致，標示為預估）、主按鈕改「開始產生（將於上限暫停）」**且維持可按**（知情選擇非錯誤）。文案不得承諾「絕不超過」（軟上限）。

4. **F16／F19 金額確認。** 按「開始產生」→ 確認對話框（`ui/Dialog.tsx` Radix 慣例，非手刻）：主文「即將為 {x} 部影片產生字幕」＋明細「語音辨識 {a} 部　預估 $X」「抽取 + 翻譯 {e} 部　預估 $Y」＋分隔線「合計預估 $Z」（加粗；三數與 F15 同源）。未超限＝F16（中性提示區塊＋「確認並開始」）；超限＝F19（warning 提示區塊——tint 依第三輪拉深一階、合計 warning 橘、「仍要開始」、文案含預計完成 N 部）。確認 → `POST /subtitles/generation-batch {scope:"selected", media_ids:[已選順序], budget_usd}` → 轉入**既有** F8 執行畫面（running／budget_ceiling／terminal 分支原樣重用）；409 `TRANSCRIPTION_BATCH_RUNNING` 沿用既有恢復路徑（`startBatchTracking(progress)`）。

5. **F20 空狀態。** `result` 的 extract＋asr 候選數為 0 → 「所有影片都有繁中字幕了」＋說明＋footer 僅「關閉」（**無**「開始產生」）。沿用既有 EmptyLibrary 插圖語彙。

6. **F17 掃描完成入口。** `ScanProgressCard`（＋`ScanProgressSheet` 手機雙生）完成態：既有行動連結列**新增第三個連結**「產生字幕 →」，統計行下方新增次要色「{n} 部影片缺繁中字幕」；`n` 由 `generationBatchPreviewKey` query 供給（**prop-driven**：query 在 `ScanProgress.tsx` 容器層跑，卡片維持純 props 讓 fixture 可測；`n=0` 時整行與連結**不顯示**）。連結導向 library route 並開啟同意流程：`library.tsx` `validateSearch` 新增 `generate` 參數（**Rule 26**：lone-numeric JSON-parse 陷阱——用 gallery.tsx 的寬鬆 boolean 係數化慣例，勿用 `typeof x === 'string'` 守衛），`LibraryBrowseV2` 讀到即開 dialog 並清掉參數（8-11 `subtitleStatus` deep-link 先例）。**不得出現任何暗示掃描本身會產生字幕的字樣**。

7. **混合選取入口＋影集過濾移除（D1 尾款）。** `SelectionToolbar` 的批次入口改傳**混合** movie＋episode ids；移除三處 `excludedSeriesCount` 管線（`LibraryBrowseV2.tsx:279-282,430-439,700-705`、dialog props L209/235 與註記 L389-397、對應 spec 斷言與 gallery fixture）。帶選取開啟時：分析 ready 後預選＝「使用者選取 ∩ 候選清單」（取代預設 extract-only；交集為空則回退預設）。`ActivityHub` 兩個入口（L269/L313）走同一流程。

8. **F8 逐項 stage 雙 family join。** sub-4-2 行為註記落地：pipeline mode 下抽取路線項目逐項進度走 D6 `subtitle_progress`（payload `{media_id, media_type, stage, message}`，**欄位名 `stage` 非 `phase`**），ASR 項目維持 `transcription_*`。擴充逐項追蹤（`useGenerationProgress` 或其 wrapper）同時監聽兩個 family、皆以 `media_id` 嚴格字串相等過濾（episode UUID 也是合法值——sub-3-2 [@contract-v2] 已定 media row id 語意）；D6 stage 詞彙**只讀不擴**（`PipelineStage` stamped）。既有單集對話框消費零回歸。

9. **架構整合姿態。** 同意流程**取代** `GenerationBatchDialogV2` 的 idle 分支（scope chips＋「缺字幕的項目 N」聚合數字走入歷史）；running／budget_ceiling／terminal（F8）分支、on-open 恢復探測、terminal invalidation（`libraryKeys.all`＋preview key）**原樣保留**。新面板拆成獨立 prop-driven 元件（`components/subtitle/consent/` 建議），container 只做接線——維持既有「presentational panel＋container」可測拆分。`subtitleService` 補齊：candidates 三方法＋型別、`GenerationBatchStartParams.budgetUsd`、`GenerationBatchItem.mediaType`（後端已回傳，FE 型別過期）。

10. **慣例與測試。** (a) **Rule 21**：所有新元件掛 `// Design ref: ux-design.pen Screen ...` header——節點：F14 `nBT3M`／F15-D `pwMzT`／F15-M `fdu4y`／F16 `gmOt6`／F17 `I3Wb0p`／F18 `zBik1`／F19 `KThbY`／F20 `D7MOm`（多畫面元件用 `·` 併列，`GenerationWorkspaceV2.tsx:1` 先例）。(b) **a11y**（Epic 11 pre-flight 四類）：確認框走 `ui/Dialog`（focus trap 免費）；F14 `role="progressbar"`＋`aria-valuemin/max/now`；狀態轉換 sr-only `aria-live="polite"`（dialog L333-335 先例）；checkbox／chips aria 標籤；44px 觸控目標（`min-h-[44px]`／`h-11`）。(c) **測試**：panel specs（選取數學、三處金額同源、F18 N 部計算、skip 排除、空狀態）＋ container spec（analyze 409 恢復、confirm payload 含 `budget_usd`＋混合 ids 順序、F17 參數開啟、`vi.hoisted` mock 慣例）＋新 SSE hook spec＋service spec；**visual gallery fixtures**（f14 analyzing／f15 default／f18 over-budget／f20 empty／f16、f19 confirm——`-gallery.fixtures.tsx` `GalleryFixture` 形狀 L279-339，UUID 字串 ids，`penNode:'screen-section'` 慣例）——**`-darwin` baselines 本機產、`-linux` 一律等 CI bootstrap PR，絕不本機 `test:visual:update` 產 linux**。(d) 全回歸＋**dev-story Step 9 UX 截圖比對強制**：對照 `_bmad-output/screenshots/flow-f-subtitle-v2/` 的 f14–f20 八張（含第三輪金額修訂）。(e) jsx-a11y：本 story 觸及檔零新增 warning（既有 118 個屬 retro-11-AI1b，不擴大清理）。

## Tasks / Subtasks

- [ ] **Task 1 — service 層與共用 helper（AC: #9 部分）**
  - [ ] `subtitleService`：`getGenerationCandidates()`（狀態信封型別）、`startCandidateAnalysis()`（202/409 discriminated union）、`cancelCandidateAnalysis()`；`GenerationBatchStartParams.budgetUsd`；`GenerationBatchItem.mediaType`
  - [ ] 抽共用 `usd()` 到 `apps/web/src/lib/`（兩處既有重複改 import）
  - [ ] service spec（信封解包、409 判別、`camelToSnake` 出線含 `budget_usd`）

- [ ] **Task 2 — SSE hooks（AC: #1, #8）**
  - [ ] 新 `useGenerationCandidatesProgress`（clone `useGenerationBatchProgress` 骨架；terminal＝ready|cancelled|error）
  - [ ] 逐項追蹤擴充：`subtitle_progress` family join（`stage` 欄位、`media_id` 嚴格相等、D6 詞彙只讀）
  - [ ] hook specs（lazy-connect、terminal 關流、雙 family、外來 media 丟棄）

- [ ] **Task 3 — 同意面板元件（AC: #2, #3, #5）**
  - [ ] `consent/` 目錄：F14 分析面板、F15 清單面板（含 F18 超限狀態與 F15-M 響應式）、F20 空狀態——全部 prop-driven
  - [ ] 選取 state＋三處金額同源 selector＋F18 預計完成 N 部計算
  - [ ] panel specs

- [ ] **Task 4 — 確認對話框 F16/F19（AC: #4）**
  - [ ] `ui/Dialog` 基底、明細三行、超限 variant（warning tint／「仍要開始」）
  - [ ] spec（未超限/超限文案與按鈕、合計與 F15 同源）

- [ ] **Task 5 — container 整合與入口（AC: #4, #6, #7, #9）**
  - [ ] `GenerationBatchDialogV2` idle 分支換成同意流程；confirm → start payload；F8 分支零觸動
  - [ ] `SelectionToolbar`/`LibraryBrowseV2` 混合 ids＋`excludedSeriesCount` 三處移除（含 spec/fixture）
  - [ ] F17：`ScanProgress.tsx` 掛 preview query → `ScanProgressCard`/`ScanProgressSheet` 新 prop＋連結；`library.tsx` `generate` 參數（Rule 26 係數化）＋ `LibraryBrowseV2` 開啟邏輯
  - [ ] container spec 更新＋新案例

- [ ] **Task 6 — 慣例落地（AC: #10a/b）**
  - [ ] Rule 21 headers（八個節點 id）；a11y 四類逐項；44px 稽核
  - [ ] gallery fixtures（六個以上狀態）＋`-darwin` baselines

- [ ] **Task 7 — 契約清點、回歸與 UX 驗證（AC: #10c/d/e）**
  - [ ] AC Drift Check＋Contract Stamp Check（強制項）：ack 9R-16 AC #1 [@contract-v3]／AC #2/#3/#7/#9 [@contract-v2]／D6 `PipelineStage` [@contract-v1]／`transcription_*` [@contract-v2]；**回頭在 sub-4-1 story 檔對 AC #7/#8 補 stamp `[@contract-v1]`**（sub-4-1 明文延後至「sub-4-3 確定跨 story 消費時」——現在確定了）並記 ack
  - [ ] `pnpm nx test api`＋`pnpm nx test web`＋`lint:all`＋visual（darwin）
  - [ ] Step 9 UX 截圖比對：flow-f-subtitle-v2 八張逐區對照，落差修正或記錄

（前端 task 7 個、後端 0 個 —— 未觸發跨端拆分門檻。）

## Dev Notes

### 消費的後端契約（全部已 merged，形狀勿再猜）

| 面 | 形狀 | 來源 |
| --- | --- | --- |
| 候選狀態信封 | `{status: idle\|analyzing\|ready\|cancelled\|error, analyzed, total, result?, analyzed_at?, error?}` | `generation_candidates.go:137-146` |
| 候選項 | `{media_id, media_type, title, route: extract\|asr\|skip, runtime_minutes, runtime_known, estimated_usd}` | `:85-94` |
| 彙總 | `{extract_count, asr_count, skipped_count, estimated_total_usd, self_hosted_asr}` | `:98-108` |
| 分析端點 | GET 恆 200 信封；POST analyze 202／409 `TRANSCRIPTION_ANALYSIS_RUNNING`；POST cancel 200 | `generation_candidates_handler.go:35-94` |
| 分析 SSE | `generation_candidates_progress` `{status, analyzed, total, error}`，250ms 節流，**不含 result** | `hub.go:53`、`generation_candidates.go:336-350` |
| 批次 start | `{scope, media_ids(混合 UUID), budget_usd?>0}`；202 `items[{media_id,title,media_type}]`；400 任一 id 無效整批拒 | `generation_batch_handler.go:59-121` [@contract-v3] |
| 執行 SSE | `generation_batch_progress` 11 keys 不變；逐項：extract→`subtitle_progress`（`stage`）、ASR→`transcription_*` | sub-4-2 行為註記 |

### 既有可重用零件（不要重造）

| 需求 | 現成零件 |
| --- | --- |
| Dialog／focus trap／Escape | `ui/Dialog.tsx`（Radix）；手機 sheet＝dialog L308-323 className 切換 |
| F8 執行分支＋恢復＋invalidation | `GenerationBatchDialogV2` running/terminal 全套——**只換 idle** |
| lazy SSE 骨架 | `useGenerationBatchProgress.ts`（backoff、mountedRef、terminal 關流） |
| indeterminate checkbox | `DownloadsTableV2.tsx:88-99` ref 慣例 |
| aria-pressed chips＋44px | dialog L341-379 |
| sr-only live region／progressbar | dialog L333-335／L459-465 |
| 409 判別不 throw | `transcriptionService.ts:46-52` |
| 縮圖列 | `LibraryListRowV2.tsx`＋`lib/image.ts getImageUrl`（`QueueRow` 是無海報 placeholder） |
| toast 行動連結慣例 | `ScanProgressCard.tsx:165-190` |
| deep-link 參數 | `library.tsx:25-42`（8-11 `subtitleStatus` 先例）＋ Rule 26 係數化（gallery.tsx boolean） |
| 測試骨架 | `GenerationBatchDialogV2.spec.tsx`（`vi.hoisted` mock＋panel/container 雙層） |

### 關鍵決策（authoring 時已裁，實作不再開放）

- **budget_usd＝WYSIWYG**：一律送畫面顯示值。畫面沒顯示的 env 預設不可能被「同意」；預填常數 $5.00 與後端 `AI_RUN_BUDGET_USD` 預設一致，operator 改 env 造成的預填漂移已記 backlog（見 Discovery Triage）。
- **F17 計數用凍結的 preview 端點**（movies-only）：最便宜且不觸發分析；候選清單含影集故 F15 實際項目數可能大於 toast 計數——已記 backlog，文案照設計「{n} 部影片」不改。
- **route=skip 不入清單**：設計無 skip 列語彙、後端不報價；`skipped_count` 留在 summary 不上畫面。
- **「免費」字樣禁用**：§5-sexies 裁定。FE 渲染 `estimated_usd` 原值，不做門檻四捨五入。
- **F18 的 N 部**：清單順序前綴和。這是預估（實際由後端逐項前置檢查決定），文案已含「預計」「約」。

### 已知限制（記錄，不在本 story 解）

- 軟上限：實際花費可能略超（文案已對齊）。抽取估價是下界（SDH 歸零仍落 ASR）——F15 註記「實際費用依內容長度而定」涵蓋。
- 每 hook 一條 EventSource（現況 9 條）：本 story +1（candidates）。共享 SSE hub 重構屬既有技術債，不擴大。
- `self_hosted_asr=true` 時 ASR 估價為 0：畫面照實渲染 $0.00（後端單一真實來源），不特別標示。

### Time-dependent visual coverage

- **Does this story add/modify any `apps/web/src/components/**/*.{ts,tsx}` that reads `Date.now()` / `new Date()` / `Date.UTC()` / `Date.parse()`?**
  - **N/A — no wall-clock-reading components touched**（八個畫面皆無時間顯示；`analyzed_at` 不上畫面。若實作中引入相對時間顯示，立即回到 Rule 23 全套：≥2 fixture states＋`clockTime`＋marker）。

### References

- [Source: `_bmad-output/planning-artifacts/design-prompt-cost-consent-2026-08-09.md`] — §3/§5-ter 畫面規格、§4 準則（②④已修訂）、§5-quinquies D1、§5-sexies 第三輪裁定
- [Source: `_bmad-output/screenshots/flow-f-subtitle-v2/`] — f14–f20 定稿截圖（Step 9 比對基準）
- [Source: `_bmad-output/implementation-artifacts/sub-4-1-cost-preview-backend.md` / `sub-4-2-consent-batch-backend.md`] — 上游契約與行為註記
- [Source: `apps/web/src/components/subtitle/GenerationBatchDialogV2.tsx`] — 整合宿主與慣例大全
- [Source: `project-context.md`] — Rule 5/18/21/23/26；jsx-a11y scope；visual baseline 工作流

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - **YES** — filed at authoring time：
    - **③ backlog-with-carry-forward-link** — `backlog-consent-toast-count-episodes`：F17 計數來源（凍結 preview 端點）movies-only，候選清單含影集 → toast 計數可能低於 F15 實際項目數。誠實修法＝BE 提供含影集的輕量計數（或 candidates 端點加 count-only 模式），屬後端範圍。非阻塞（低估不高估，方向安全）。
    - **③ backlog-with-carry-forward-link** — `backlog-budget-default-config-endpoint`：預算輸入預填 $5.00 為 FE 常數；operator 改 `AI_RUN_BUDGET_USD` 後預填值不跟隨（送出值仍 WYSIWYG 正確，僅預填漂移）。誠實修法＝BE 曝露預設值的 config 讀點，屬後端範圍。非阻塞。

### File List
