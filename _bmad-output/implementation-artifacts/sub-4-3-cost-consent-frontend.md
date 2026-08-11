# Story 4.3: 成本同意篩選畫面 —— F14–F20（前端）

Status: done

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

- [x] **Task 1 — service 層與共用 helper（AC: #9 部分）**
  - [x] `subtitleService`：`getGenerationCandidates()`（狀態信封型別）、`startCandidateAnalysis()`（202/409 discriminated union）、`cancelCandidateAnalysis()`；`GenerationBatchStartParams.budgetUsd`；`GenerationBatchItem.mediaType`
  - [x] 抽共用 `usd()` 到 `apps/web/src/lib/`（兩處既有重複改 import）
  - [x] service spec（信封解包、409 判別、`camelToSnake` 出線含 `budget_usd`）

- [x] **Task 2 — SSE hooks（AC: #1, #8）**
  - [x] 新 `useGenerationCandidatesProgress`（clone `useGenerationBatchProgress` 骨架；terminal＝ready|cancelled|error）
  - [x] 逐項追蹤擴充：`subtitle_progress` family join（`stage` 欄位、`media_id` 嚴格相等、D6 詞彙只讀）
  - [x] hook specs（lazy-connect、terminal 關流、雙 family、外來 media 丟棄）

- [x] **Task 3 — 同意面板元件（AC: #2, #3, #5）**
  - [x] `consent/` 目錄：F14 分析面板、F15 清單面板（含 F18 超限狀態與 F15-M 響應式）、F20 空狀態——全部 prop-driven
  - [x] 選取 state＋三處金額同源 selector＋F18 預計完成 N 部計算
  - [x] panel specs

- [x] **Task 4 — 確認對話框 F16/F19（AC: #4）**
  - [x] `ui/Dialog` 基底、明細三行、超限 variant（warning tint／「仍要開始」）
  - [x] spec（未超限/超限文案與按鈕、合計與 F15 同源）

- [x] **Task 5 — container 整合與入口（AC: #4, #6, #7, #9）**
  - [x] `GenerationBatchDialogV2` idle 分支換成同意流程；confirm → start payload；F8 分支零觸動
  - [x] `SelectionToolbar`/`LibraryBrowseV2` 混合 ids＋`excludedSeriesCount` 三處移除（含 spec/fixture）
  - [x] F17：`ScanProgress.tsx` 掛 preview query → `ScanProgressCard`/`ScanProgressSheet` 新 prop＋連結；`library.tsx` `generate` 參數（Rule 26 係數化）＋ `LibraryBrowseV2` 開啟邏輯
  - [x] container spec 更新＋新案例

- [x] **Task 6 — 慣例落地（AC: #10a/b）**
  - [x] Rule 21 headers（八個節點 id）；a11y 四類逐項；44px 稽核
  - [x] gallery fixtures（六個以上狀態）＋`-darwin` baselines

- [x] **Task 7 — 契約清點、回歸與 UX 驗證（AC: #10c/d/e）**
  - [x] AC Drift Check＋Contract Stamp Check（強制項）：ack 9R-16 AC #1 [@contract-v3]／AC #2/#3/#7/#9 [@contract-v2]／D6 `PipelineStage` [@contract-v1]／`transcription_*` [@contract-v2]；**回頭在 sub-4-1 story 檔對 AC #7/#8 補 stamp `[@contract-v1]`**（sub-4-1 明文延後至「sub-4-3 確定跨 story 消費時」——現在確定了）並記 ack
  - [x] `pnpm nx test api`＋`pnpm nx test web`＋`lint:all`＋visual（darwin）
  - [x] Step 9 UX 截圖比對：flow-f-subtitle-v2 八張逐區對照，落差修正或記錄

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

## Senior Developer Review (AI)

**Date:** 2026-08-11 · **Reviewer model:** Claude Opus 5（實作為 Fable 5 —— 換模型慣例：對抗式審查由 Opus 5 subagent 執行、Fable 5 逐項驗證後裁定與修復） · **Outcome:** Approve (after same-session fixes) · **Findings:** 2 High / 5 Medium / 2 Low

**強制檢查：** 🔒 Rule 7 Wire Format **N/A**（純前端，無 Go error-code 檔在範圍內）· 🔒 Rule 20 Contract Bump **N/A**（無 `→` bump token；sub-4-1 的兩個 `[@contract-v1]` 是首次 stamp 非 bump）· 🔒 Rule 25 Mega-line **N/A**（project-context.md 未觸及）· Git vs File List **0 落差**（26 路徑）· 10 條 AC 全數有實作證據、checkbox 稽核 0 未勾。

**Action Items：**

- [x] **[H1] Workspace 的「下次繼續」仍會啟動未經同意的整庫批次。** 對話框已改「回同意清單」，但 Activity workspace 同名按鈕仍直呼 `startGenerationBatch({scope:'missing'})`——無 media_ids、無 budget_usd、無確認畫面，正是 2026-08-07 裁定要消滅的路徑。**修復：** `onResume={onLaunch}`——經 launch 路徑開啟對話框的同意階段（F15 重選 → F16 確認）；容器內未用的 queryClient 一併清除。
- [x] **[H2] 候選清單在首次分析後永久凍結。** bootstrap 只在 `idle|cancelled|error` 時 POST analyze；掃描後（F17 入口）與批次結束後 GET 都回舊 `ready` 快照——新檔案永遠進不了 F15、剛做完的項目被重報價重預選（執行時撞 preflight skip 被計 fail）。且 `analyzed_at` 不上畫面，stale 完全不可見。**修復：** `forceAnalyze` prop 鏈（consent view ← dialog ← LibraryBrowseV2）：F17 deep-link 開啟與批次 terminal 後的下一次 consent render 一律強制重新分析（免費、可取消、有 F14 可視進度）。**AC #1 語意修訂記錄**：「ready → 直接 F15」僅適用於同一 process 內容未變動的重開；上述兩個時點視快照為 stale。
- [x] **[M3] unmount 期間 in-flight promise 會 leak EventSource（reviewer 以 probe test 實證）。** bootstrap 無 cancelled guard；unmount 後 promise resolve → `startTracking()` 在死掉的 hook 上開新連線，無人關閉。**修復：** hook `startTracking` 加 `mountedRef` 前置擋（+回歸測試：unmount 後 startTracking 零連線）；bootstrap 改收 `isCancelled` token，所有 await 後寫入前檢查。
- [x] **[M4] 批次執行中開對話框會踢起隱形的整庫 ffprobe 掃描。** mount 時 `isIdle` 為真 → consent bootstrap 先於恢復探測跑完 → POST analyze 開跑（NAS 上與批次的 ffmpeg/ASR 撞 I/O），探測回來 consent unmount，使用者從沒看到 F14 與取消鈕（並疊加 M3 的 leak）。**修復：** container `probed` 狀態——恢復探測 settle 前不 render consent view（+測試：探測未決時 stub 不存在、決後出現）。
- [x] **[M5] 錯過 terminal `ready` 事件時 F14 永久轉圈。** ready 只走 SSE；小片庫亞秒完成可搶在 EventSource 註冊前、或 10s backoff 期間掉幀。**修復：** `phase==='analyzing'` 期間每 5s GET 快照作 safety net（ready→清單、cancelled→關閉、error→錯誤畫面；SSE 仍是 fast path）。+fake-timers 測試。
- [x] **[M6] F17 計數可能是上一次掃描的數字。** query 掛全域 5 分鐘 staleTime、只有批次 terminal 會 invalidate——兩次掃描間隔 <5 分鐘時第二個 toast 顯示第一次的計數（cached 0 時整行與連結直接消失）。**修復：** 該 query `staleTime: 0`（toast 出現即 refetch）。
- [x] **[M7] 同媒體的「搜尋」完成會誤終結生成追蹤。** 搜尋引擎是 `subtitle_progress` 的第二個 producer（同 media_id、共享 terminal 詞彙）——生成中途跑搜尋，搜尋的 `complete` 會讓生成 UI 顯示完成並關流。**修復：** D6 terminal 只在本追蹤 session 觀察過 pipeline stage（probing/extracting/translating）後才生效（`d6PipelineSeenRef`）；attach-degraded 錯過 stages 時 terminal 讓位給批次事件（本就 authoritative）。+雙向測試（搜尋 complete 被忽略、其後真 pipeline terminal 生效）。
- [x] **[L8] 確認框的 `confirming` 狀態在生產不可達＋in-flight 可被靜默關閉。** `handleConfirm` 先關框再 `onStartBatch`，spinner/disabled 永不渲染；且 starting 期間 Escape 可關框，付費批次已啟動而無任何回饋。**修復：** 確認框在 start in-flight 期間保持開啟（成功→整個 view unmount 自然關閉；失敗→startError effect 關框並在清單面板顯示錯誤）；starting 期間 onCancel 擋掉。+生命週期測試。
- [x] **[L9] 三處測試無法因其名義原因而失敗。** (a) candidates backoff 測試改為真正斷言 fresh stream 的 terminal 關流且無後續重連；(b) D6 外來 media 測試改用 translating（filter 刪除即紅）、unmapped-stage 測試先進入 distinct 狀態再斷言不動；(c) container spec 以 stub 隔離 consent view——**accepted with rationale**：consent spec 覆蓋「真實選取→onStartBatch(ids, budget)」、container spec 覆蓋「onStartBatch→wire payload」，接縫是兩個 typed props 的直通，重複整合層的成本大於殘餘風險；記錄於此供未來 e2e 補位。

**Reviewer 驗證後排除（不列 findings）**：`computeTotals`/`handleConfirm` 走同一未過濾順序（提交順序＝F18 前綴和順序）；D6 payload keys 與 `progress_sse.go` 核對正確（episode UUID 含）；`budget_usd` snake-case 出線正確；`preselectedIds` 兩個呼叫端皆傳 state array（props doc 已明文要求 render-stable）；取消路徑雙重 `onClose` 冪等；移除 props 零殘留引用；F17 card/sheet gating 一致。

**修復後驗證：** web 233 檔/**2612** 測試全綠（CR 修復 +9 測試）· api 全綠 · lint 0 errors · prettier 乾淨 · 無殘留 worker · visual 基準零額外變動（CR 修復全為行為面）。

## Dev Agent Record

### Agent Model Used

Claude Fable 5 — dev-story workflow, 2026-08-11

### Debug Log References

- RED→GREEN：新測試先對未實作面紅燈（candidates service 方法、`useGenerationCandidatesProgress`、consent 面板、雙 family join、F18 hint 隱藏）→ 實作後綠；既有 dialog spec 對被取代的 idle 契約紅燈 → 重寫至 consent 時代契約後綠。
- 全回歸：`pnpm nx test api` ✅ · `pnpm nx test web` ✅（**233 檔 / 2603 測試**，+5 檔/+56 測試）· `pnpm run lint:all` **0 errors**（warnings 較基線持平且本 story 觸及檔零新增）· prettier 乾淨 · 無殘留 vitest worker。
- Visual：全套 `--project=visual` update→verify 兩輪皆綠（本機 dev server + `CI=true` 繞過需金鑰的 Go webServer——CI 工作流本來就只起前端）。基準變更集乾淨：僅本 story 的 fixtures（見 File List）。
- 原始 tsc 錯誤數 **149→139**（淨減 10，皆為移除舊碼；新檔零錯誤——原生 `tsc -p` 非本 repo 閘門，其餘為 pre-existing route-type 噪音）。

### Completion Notes List

- **🔗 AC Drift: FOUND** — (1) **ux3-subtitle-v2-batch 的 idle 分支契約**（scope chips「缺字幕的項目 N」／「已選項目」segments、`gen-batch-empty-scope`、`開始生成` 按鈕、`excludedSeriesCount` 註記）：被 consent 流程整體取代（本 story AC #9 的目的）；該 story done＝frozen，不欠 stale-mark，其 spec/fixture 同 change 更新。(2) **ux3-subtitle-v2-batch AC 5「客戶端過濾影集」**：D1 之後語意反轉（混合 ids 直送），`SelectionToolbar`→`LibraryBrowseV2` 的過濾與 `excludedSeriesCount` 三處移除。(3) 對話框標題「批次生成字幕」→「產生字幕」（設計 F15/F8 標題統一）。grep 範圍：`excludedSeriesCount|selectedMovieIds|批次生成字幕|gen-batch-scope` across web src ＋ implementation-artifacts。
- **📎 Contract Stamps: FOUND（0 bumps produced；2 stamps produced upstream；4 acks recorded）** — 依 sub-4-1 的明文延遲條款，本 story 回頭在 sub-4-1 story 檔對 **AC #7、AC #8 蓋上 `[@contract-v1]`**（forward-only retrofit：首次跨 story 消費時 stamp，無 bump 無 Change Log row 義務）。Acks as consumer：confirmed against `[@contract-v3]`（Story 9R-16 AC #1，as bumped by sub-4-2——`budget_usd`＋`items[].media_type`＋混合 episode UUIDs，`startGenerationBatch` 送出面與 202 解析面皆比對）· confirmed against `[@contract-v2]`（Story 9R-16 AC #2/#3/#7/#9——status/cancel/preview 端點與 `generation_batch_progress` 11-key payload 原樣消費）· confirmed against `[@contract-v1]`（Story sub-4-1 AC #7/#8，本次新蓋——狀態信封／候選形狀／`generation_candidates_progress` counts-only 條款）· confirmed against `[@contract-v1]`（Story sub-1-3 AC #1 `PipelineStage`——D6 詞彙 READ-ONLY 消費於雙 family join，零新值）· `transcription_*` `[@contract-v2]`（sub-3-2——episode UUID 為合法 `media_id`，join 用嚴格字串相等）。
- **🎭 A11y Pre-Flight: PASS**（新元件 6 個＋觸及元件 5 個檢查；jsx-a11y 掃描本 story 觸及檔 **0 warnings 新增**；四類逐項：modal focus——F16/F19 與 consent shell 全走 Radix `ui/Dialog`（focus trap／Escape／aria-modal 免費）；aria-live——`consent-phase-live` sr-only 區＋F14 進度 `aria-live="polite"`＋`role="progressbar"` 帶 valuemin/max/now；keyboard/ARIA——chips `aria-pressed`、checkbox 全帶 aria-label、indeterminate 用 ref 慣例；觸控目標——所有可按元素 `min-h-[44px]`／`h-11`。lazy-load 合約：F17 計數 query `enabled` 雙閘（isVisible && isComplete），註解如實描述。）
- **🎨 UX Verification: PASS（with 1 fix applied）** — 對照 `_bmad-output/screenshots/flow-f-subtitle-v2/` 八張定稿（含第三輪金額修訂；本 session 稍早我以 Pencil MCP 逐節點驗過同一份定稿的文字層與色彩，比對基準即該驗收記錄）：

  | Area | Design Spec | Implementation | Match? | Fix |
  |------|------------|----------------|--------|-----|
  | F14 進度＋免費說明＋取消 | 進度條＋「分析字幕軌 N / M」＋本機免費說明 | `AnalysisProgressPanel` 同文案同結構 | ✅ | — |
  | F15 列金額 | 抽取 `$0.05`/`$0.04` success 綠 mono；ASR warning 橘；`≈` on 片長未知 | `estimated_usd` 原值渲染，色彩/字型 token 同 | ✅ | — |
  | F15 chip 標記 | 「僅翻譯費」success／「付費」warning | 同 | ✅ | — |
  | F15 工具列 | 半選 checkbox＋「選取全部可抽取（僅翻譯費）」＋清除（手機縮短標籤） | 同（sm: 斷點切換標籤） | ✅ | — |
  | F15 三處金額 | 摘要條/footer/明細一致 | 單一 `computeTotals` selector | ✅ | — |
  | F16/F19 明細與按鈕 | 兩行明細＋合計；「確認並開始」/「仍要開始」；F19 warning tint 拉開 | 同；F19 用 warning-tint vs F16 中性 bg-tertiary（對比大於設計顧慮的兩 warning tint 差） | ✅ | — |
  | F17 toast | 統計行下「N 部影片缺繁中字幕」＋第三連結「產生字幕 →」；無掃描即生成暗示 | 同（count>0 才顯示；cancelled 不顯示） | ✅ | — |
  | **F18 footer 小字** | **第三輪已刪除**（與警示列語義重複） | 初版仍顯示 → **修正：over-budget 時隱藏**，＋2 測試釘住 | ✅ (fixed) | 🎨 UX Fix |
  | F18 警示列/按鈕 | 「預估 $X 已超過上限 $Y——預計可完成約 N 部…」；按鈕「開始產生（將於上限暫停）」可按 | 同（N＝前綴和） | ✅ | — |
  | F20 空狀態 | 主文＋說明＋僅「關閉」 | 同 | ✅ | — |

- **架構落地與 story 規劃一致**：consent 流程（`consent/` 5 元件＋純邏輯 selector）取代 dialog idle 分支；F8 執行分支、恢復探測、terminal invalidation 原樣保留（race A/B 測試續綠）；`下次繼續` 語意更新＝回到同意清單（新同意，非未經同意的自動重跑）。
- **Deep link**：`library.tsx` `generate` 參數用寬鬆 boolean 係數化（Rule 26——`true|1|'1'|'true'`），`LibraryBrowseV2` 開啟後以 `replace: true` 清參數（8-11 先例）。
- **已知行為（記錄）**：library 選取含 series 列時，series ROW id 與候選（episodes）不相交 → 不預選、fallback 預設 extract-only；其影集仍完整出現在 F15 清單可手動勾選。屬可接受行為非缺陷（FE 無 series→episodes 對映資料）。
- **Pre-existing fix or file（Epic 9c AI-2）**：FILED —— `preexisting-fail-parse-progress-darwin-baseline`（sprint-status backlog）：`parse-floating-parse-progress-card` 的 `-darwin` 基準在本機 verify 紅（本 story 未觸及 parse/**；full-suite update 時它被重寫、restore 後 verify 即紅＝本機環境漂移非本 story 造成）。基準檔已 restore 未提交；CI 用 `-linux` 基準不受影響。
- **-linux baselines**：新 fixtures（`generation-consent/*` 六個、`scanner-scan-complete-toast`）只產 `-darwin`；`-linux` 依規定等 CI `Visual Regression` workflow 的 bootstrap PR，**未在本機產**。舊 `generation-batch-dialog-v2/{running,budget_ceiling}` 的 `-linux` 基準將因本 story 變更而在 CI 顯示 diff → 屬預期的 deliberate-rebless（見 PR 說明）。

### Change Log

| Date | Change |
| ---- | ------ |
| 2026-08-11 | Task 1：`subtitleService` candidates 三方法＋型別（狀態信封/候選/彙總）、`GenerationBatchStartParams.budgetUsd`、`GenerationBatchItem.mediaType`（v3 對齊）；`lib/currency.ts` 抽共用 `usd()`（兩處重複改 import）；service spec +5。 |
| 2026-08-11 | Task 2：NEW `useGenerationCandidatesProgress`（lazy SSE，terminal＝ready/cancelled/error）＋ `useGenerationProgress` 雙 family join（D6 `subtitle_progress`：probing/extracting→extracting、translating→translating、complete/failed/skipped terminal；未對映 stage 忽略；`media_id` 嚴格相等含 episode UUID）；hook specs +15。 |
| 2026-08-11 | Task 3–4：`consent/` 目錄——`consentSelection.ts`（單一 totals selector＋前綴和 feasibleCount＋parseBudgetInput 0≠unlimited）、`AnalysisProgressPanel`（F14）、`CandidateListPanel`（F15/F18/F15-M）、`ConsentEmptyState`（F20）、`ConfirmGenerationDialog`（F16/F19）、`GenerationConsentView`（consent 容器）；specs +42。 |
| 2026-08-11 | Task 5：dialog container idle→consent 重接（`selectedMediaIds` 混合 ids、`handleStartConsented` 送 `{scope:selected, media_ids, budget_usd}`、`下次繼續`→回同意清單）；panel 剝離 idle 分支；`SelectionToolbar` 入口混合 ids＋`excludedSeriesCount` 三處移除；F17（`ScanProgress` preview query→card/sheet 新 prop＋連結；`library.tsx` `generate` 參數）；dialog spec 重寫至新契約。 |
| 2026-08-11 | Task 6：Rule 21 headers（八節點 Design ref）；gallery fixtures——idle fixture 移除、+6 consent fixtures＋F17 完成 toast fixture、items 補 mediaType；`-darwin` baselines：+7 新、2 更新（F8 running/budget_ceiling）、idle 2 刪除；a11y 四類逐項通過。 |
| 2026-08-11 | Senior Developer Review（Opus 5 審 Fable 5）：2H/5M/2L 全處理 —— H1 workspace 下次繼續改走 onLaunch（同意路徑）；H2 forceAnalyze 鏈（F17 deep-link＋批次 terminal 後強制重析，AC #1 語意修訂）；M3 mountedRef＋cancelled token 堵 EventSource leak；M4 probe gate；M5 analyzing 5s 輪詢 safety net；M6 F17 query staleTime:0；M7 D6 terminal 需先見 pipeline stage（搜尋汙染隔離）；L8 確認框 in-flight 生命週期；L9 測試強化。+9 測試。修後 web 233/2612 全綠。Status review → done。 |
| 2026-08-11 | Task 7：sub-4-1 AC #7/#8 補 stamp `[@contract-v1]`（依其明文延遲條款）；UX 比對修正 F18 footer 小字（over-budget 隱藏＋2 測試）；全回歸 web 233/2603 ✅ api ✅ lint 0 errors；`preexisting-fail-parse-progress-darwin-baseline` 立案。story → review。 |

### File List

- `apps/web/src/services/subtitleService.ts` — candidates API＋`budgetUsd`＋`mediaType`＋v3 契約註解
- `apps/web/src/services/subtitleService.spec.ts` — +5 測試（信封 camelize、409 判別、budget_usd 出線）
- `apps/web/src/lib/currency.ts` — NEW：共用 `usd()`
- `apps/web/src/hooks/useGenerationCandidatesProgress.ts` — NEW：分析進度 lazy SSE hook
- `apps/web/src/hooks/useGenerationCandidatesProgress.spec.ts` — NEW：8 測試
- `apps/web/src/hooks/useGenerationProgress.ts` — 雙 family join（D6 `subtitle_progress`）
- `apps/web/src/hooks/useGenerationProgress.spec.ts` — +7 測試（stage 對映/terminal/外來丟棄/episode 追蹤）
- `apps/web/src/components/subtitle/consent/consentSelection.ts` — NEW：純邏輯 selector
- `apps/web/src/components/subtitle/consent/consentSelection.spec.ts` — NEW：11 測試
- `apps/web/src/components/subtitle/consent/AnalysisProgressPanel.tsx` — NEW：F14
- `apps/web/src/components/subtitle/consent/CandidateListPanel.tsx` — NEW：F15/F18/F15-M
- `apps/web/src/components/subtitle/consent/CandidateListPanel.spec.tsx` — NEW：12 測試
- `apps/web/src/components/subtitle/consent/ConsentEmptyState.tsx` — NEW：F20
- `apps/web/src/components/subtitle/consent/ConfirmGenerationDialog.tsx` — NEW：F16/F19
- `apps/web/src/components/subtitle/consent/ConfirmGenerationDialog.spec.tsx` — NEW：3 測試
- `apps/web/src/components/subtitle/consent/GenerationConsentView.tsx` — NEW：consent 容器
- `apps/web/src/components/subtitle/consent/GenerationConsentView.spec.tsx` — NEW：9 測試
- `apps/web/src/components/subtitle/GenerationBatchDialogV2.tsx` — idle→consent 重接、panel 剝離 idle、標題統一
- `apps/web/src/components/subtitle/GenerationBatchDialogV2.spec.tsx` — 重寫至 consent 時代契約（24 測試）
- `apps/web/src/components/subtitle/GenerationWorkspaceV2.tsx` — `usd()` 改共用 import；CR H1：onResume→onLaunch（同意路徑）
- `apps/web/src/components/library/LibraryBrowseV2.tsx` — 混合 ids、excluded 移除、`generate` deep link
- `apps/web/src/components/scanner/ScanProgress.tsx` — F17 preview query（rules-of-hooks 安全位置）
- `apps/web/src/components/scanner/ScanProgressCard.tsx` — F17 計數行＋「產生字幕 →」連結
- `apps/web/src/components/scanner/ScanProgressSheet.tsx` — F17 手機雙生
- `apps/web/src/routes/library.tsx` — `generate` 搜尋參數（Rule 26 係數化）
- `apps/web/src/routes/test/-gallery.fixtures.tsx` — consent fixtures ×6＋F17 fixture、idle fixture 移除、items 補 mediaType
- `tests/visual/components.visual.spec.ts-snapshots/components/generation-consent/**` — NEW：6 組 `-darwin` 基準
- `tests/visual/components.visual.spec.ts-snapshots/components/scanner-scan-complete-toast/**` — NEW：`-darwin` 基準
- `tests/visual/components.visual.spec.ts-snapshots/components/generation-batch-dialog-v2/{running,budget_ceiling}/default-visual-darwin.png` — 更新（deliberate rebless）
- `tests/visual/components.visual.spec.ts-snapshots/components/generation-batch-dialog-v2/idle/**` — DELETED（fixture 移除）
- `_bmad-output/implementation-artifacts/sub-4-1-cost-preview-backend.md` — AC #7/#8 `[@contract-v1]` stamp（依延遲條款，見 Completion Notes）
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — sub-4-3 in-progress → review＋pre-existing 立案
- `_bmad-output/implementation-artifacts/sub-4-3-cost-consent-frontend.md` — 本 story 檔

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - **YES** — filed at authoring time：
    - **③ backlog-with-carry-forward-link** — `backlog-consent-toast-count-episodes`：F17 計數來源（凍結 preview 端點）movies-only，候選清單含影集 → toast 計數可能低於 F15 實際項目數。誠實修法＝BE 提供含影集的輕量計數（或 candidates 端點加 count-only 模式），屬後端範圍。非阻塞（低估不高估，方向安全）。
    - **③ backlog-with-carry-forward-link** — `backlog-budget-default-config-endpoint`：預算輸入預填 $5.00 為 FE 常數；operator 改 `AI_RUN_BUDGET_USD` 後預填值不跟隨（送出值仍 WYSIWYG 正確，僅預填漂移）。誠實修法＝BE 曝露預設值的 config 讀點，屬後端範圍。非阻塞。

### File List
