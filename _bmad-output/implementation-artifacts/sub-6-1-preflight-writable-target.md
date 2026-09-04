# Story 6.1: pre-flight 檢查目標資料夾可寫 —— 先驗證，再花錢（後端）

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a BYOK NAS owner,
I want the pipeline to refuse to spend a single API dollar on a media file whose folder it cannot write to,
so that "translate 900 cues, then `permission denied` at placement" — which burned real money twice in eval-1 — can never happen again.

## Context — 這個 story 為什麼存在

M3 第二波第一棒（eval-1 後續 P0-1，party-mode 2026-09-03 裁定）。eval-1 實跑兩次「翻完才失敗」：(1) Unraid 模板把 `/media` 掛成 `ro`，placer 寫檔失敗；(2) 容器 `PUID/PGID=1000` 對上 `nobody:users` 的資料夾，同樣翻完才 permission denied，燒了 $0.196。兩者的共同形狀是**先花錢、最後才驗證**。

現況：`preflightSkip`（`pipeline.go:600`）只回答「是不是已經有可接受的 sidecar」；`placer.Place`（`placer.go:78`）只檢查 `mediaDir` **存在**，不檢查**可寫**。中間隔著 ffprobe、抽軌、幾十次 LLM 呼叫。

## Acceptance Criteria

1. **可寫探針在任何付費呼叫之前。** `ProcessItem`（`process_item.go:32`）在 `preflightSkip` 之後、路由／抽軌之前，對 `filepath.Dir(item.FilePath)` 做一次真實寫入探針：在該資料夾建立 `.vido-write-probe-<random>` 暫存檔並立即刪除（`os.CreateTemp` + `os.Remove`）。**不得**只靠 `os.Stat` 權限位判斷——NFS／SMB／Unraid 的 mode bit 與實際可寫常不一致，eval-1 的 PGID 案例正是 mode bit 看起來可寫。探針失敗 → 該 item 以新錯誤 `SUBTITLE_TARGET_NOT_WRITABLE` 終止，**零 ffprobe、零抽軌、零 LLM**。

2. **Rule 7 錯誤碼。** 新增 `ErrSubtitleTargetNotWritable` sentinel 於 `subtitle/errors.go`（`SUBTITLE_` 既有前綴，code-list update only，prefix 數不變）；`project-context.md` Rule 7 清單同步加入 `SUBTITLE_TARGET_NOT_WRITABLE`。handler 層的 zh-TW envelope：message「目標資料夾無法寫入」、suggestion 指向 `docs/deployment.md` 的權限段（見 AC #5）。

3. **失敗不燒錢也不髒狀態。** 探針失敗的 item：`subtitle_runs` 記一筆 `failed` 且 `cost_usd=0`（沿用既有 failed 寫入路徑）；media row 的 `subtitle_status` 依 9R-10b 的 FreeOnly brake 語意**放回原值**，不得留在 in-progress。SSE `subtitle_progress` 送 `stage=failed` 帶 message。

4. **同意畫面提前亮紅燈（F15）。** `GenerationCandidateService` 分析（`generation_candidates.go`）對每個候選也跑同一探針（抽成 `subtitle.ProbeWritable(dir) error` 供兩處共用，Rule 27 reuse-over-reinvent），結果以 **additive** 欄位 `writable bool` + `blocker string`（空＝無阻礙）掛在 `GenerationCandidate`（sub-4-1 `[@contract-v1]` additive 不 bump，記 ack 與 Change Log，比照 sub-5-1 AC #5 先例）。`writable=false` 的候選：**預設不勾選、估價不計入合計**、列上顯示 error 徽章「資料夾無法寫入」；summary 加 `unwritable_count`。同一資料夾多集只探一次（快取 per-analysis，Rule 14 有界 map）。

5. **文件。** `docs/deployment.md` + `docs/deployment.zh-TW.md`（Rule 17）新增「檔案權限」段：Unraid 模板 `/media` 必須 `Read/Write`；`PGID` 建議設為片庫資料夾的 group（Unraid 預設 `users`=100）；列出 `SUBTITLE_TARGET_NOT_WRITABLE` 的排錯步驟。

6. **測試。** (a) `ProbeWritable`：可寫目錄 nil、唯讀目錄（`os.Chmod 0555`，root 環境下 skip）回 error、不存在目錄回 error；(b) `ProcessItem`：探針失敗時 translator／extractor／ffprobe fakes **零呼叫**（斷言 call count），run 記 failed 且 cost 0；(c) 候選分析：`writable=false` 不入預設選取、不計入 `estimated_total_usd`、`unwritable_count` 正確、同資料夾只探一次；(d) 全回歸。

## Tasks / Subtasks

- [x] **Task 1 — 探針與錯誤碼（AC: #1, #2）**
  - [x] `subtitle/writable.go`：`ProbeWritable(dir string) error`（CreateTemp + Remove，錯誤包 `%w` ErrSubtitleTargetNotWritable）
  - [x] `subtitle/errors.go` 新 sentinel；`project-context.md` Rule 7 清單同步
- [x] **Task 2 — ProcessItem 接線（AC: #1, #3）**
  - [x] `process_item.go` 在 preflightSkip 後插入探針；失敗路徑走既有 failed 寫入＋status 回滾＋SSE
  - [x] 測試：零付費呼叫斷言
- [x] **Task 3 — 候選分析曝露（AC: #4）**
  - [x] `GenerationCandidate` 加 `writable`／`blocker`，summary 加 `unwritable_count`；per-analysis 目錄快取
  - [x] Swagger 註解＋sub-4-1 契約 ack／Change Log
- [x] **Task 4 — FE 消費（AC: #4 FE 半）**
  - [x] `subtitleService.ts` 型別加欄位；`consentSelection.ts` 預設選取與合計排除 `writable=false`；`CandidateListPanel` 顯示徽章
  - [x] specs：排除邏輯、徽章渲染
- [x] **Task 5 — 文件（AC: #5）**

（後端 task 3 個、前端 1 個 —— 未觸發跨端拆分門檻。）

## Dev Notes

### 既有可重用零件（不要重造）

| 需求 | 現成零件 |
| --- | --- |
| 早退點 | `Pipeline.preflightSkip` `pipeline.go:600`（本 story 在它**之後**加第二道閘） |
| 失敗寫入與 status 回滾 | `process_item.go` 既有 failed 路徑；9R-10b FreeOnly brake 的「放回原值」語意（`media_store.go:95-99` 註解） |
| 候選信封 additive 先例 | sub-5-1 AC #5（`DefaultBudgetUSD` ride `AnalysisSnapshot`，不 bump） |
| 有界快取 | Rule 14；per-analysis `map[dir]error` 隨分析結束丟棄即可 |
| SSE stage 詞彙 | `PipelineStage` stamped（sub-1-3）——**只用既有 `failed`，不新增 stage** |

### 為什麼是真寫探針而不是 stat

eval-1 發現 2：資料夾 `nobody:users`（gid 100）、容器 gid 1000，`ls -l` 顯示 group 可寫，但容器程序不在該 group。`os.Stat` 的 mode bit 會說「可寫」，只有真的建檔才會失敗。CreateTemp 是最便宜的真實驗證。

### 架構合規

- Rule 4/19：探針放 `subtitle` 套件（leaf-ish，`services` 已 import 它的介面方向不變）；`generation_candidates.go` 在 services 層呼叫 `subtitle.ProbeWritable` **會不會**造成 services→subtitle 的 import？**會**——Rule 19 禁止。因此探針放 `internal/fsprobe`（新 leaf 套件，零 internal 依賴）或直接放 `services` 並由 subtitle 透過既有 port 注入。**裁定：新 leaf 套件 `internal/fsprobe`**，兩邊都 import 它；`boundaries_test.go` 的 leaf 清單加 `fsprobe`。
- Rule 13：探針錯誤一律 propagate，不 log-and-continue。
- Rule 20：sub-4-1 AC #7 additive ack。

### Project Structure Notes

- 新檔：`apps/api/internal/fsprobe/writable.go` + `_test.go`；`docs/deployment*.md` 段落。
- 改檔：`subtitle/errors.go`、`subtitle/process_item.go`、`services/generation_candidates.go`、`handlers/generation_candidates_handler.go`（Swagger）、`web/src/services/subtitleService.ts`、`web/src/components/subtitle/consent/{consentSelection.ts,CandidateListPanel.tsx}`、`project-context.md`。

### Time-dependent visual coverage

- N/A — no wall-clock-reading components touched（徽章是純 props）。

### References

- eval-1 story「執行中發現的產品問題」1、2；「後續 Backlog」P0-1 — `_bmad-output/implementation-artifacts/eval-1-translation-blind-eval.md`
- `apps/api/internal/subtitle/placer.go:78-120`（現況只檢查目錄存在）
- Rule 7 / 13 / 14 / 17 / 19 / 20 — `project-context.md`

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（dev-story，2026-09-04）

### Debug Log References

- `go test ./internal/fsprobe ./internal/subtitle ./internal/services ./internal/` 綠；`pnpm exec vitest run apps/web/src/components/subtitle/consent` 82 tests 綠。

### Completion Notes List

- **探針放新 leaf 套件 `internal/fsprobe`**（Dev Notes 裁定；Rule 19：services↛subtitle）：`ProbeWritable(dir)` 真建檔＋刪檔，不看 mode bit；`ErrNotWritable` sentinel；`boundaries_test.go` leaf 清單與 project-context Rule 19 同步。
- pipeline：`WithWritableProbe` option（預設 `fsprobe.ProbeWritable`）；`ProcessItem` Step 2b 在 run row 建立後、路由前探針；失敗 → `failItem`（run `failed`、$0、SSE `failed`）再 `restoreMediaStatus` 把 media row 放回 Load 時的狀態（FreeOnly brake 語意，AC #3）。錯誤碼 `SUBTITLE_TARGET_NOT_WRITABLE`（Rule 7 code-list update only，prefix 17 不變）。
- 候選分析：`probeWritable` 欄位（`defaultWritableProbe` 變數，測試 `TestMain` 換成寬鬆版）；每次分析每個目錄探一次（map 快取）；`GenerationCandidate.Writable/Blocker` + `Summary.UnwritableCount`，unwritable 的估價不進 `EstimatedTotalUSD`。sub-4-1 AC #7 `[@contract-v1]` additive 不 bump（confirmed against `[@contract-v1]` (Story sub-4-1 AC #7)）。
- FE：型別 optional（舊伺服器視為可寫）；`isWritable`／`selectableIds`；`defaultSelection`、全選、整劇／整季、`preselectedIds` 交集全部排除 unwritable；列：checkbox `disabled` + 「資料夾無法寫入」error 徽章（tooltip 帶 blocker）+ `data-writable`；全選的「全部」以可選列計。
- 文件：`docs/deployment.md` 新增「File Permissions」段（ro mount、PGID 100、排錯三步）。**Rule 17 缺口**：`docs/deployment.zh-TW.md` 本來就不存在 → lane ③ `backlog-deployment-doc-zhtw-twin`。
- 🔗 AC Drift: NONE (checked: 'preflightSkip|failItem|estimated_total_usd|defaultSelection' across _bmad-output/implementation-artifacts/*.md — sub-1-5b AC #2 pre-flight（sidecar 閘門）不變、本 story 在其後加第二道；sub-4-3 AC #2「預設選取＝全部 extract」語意加上「且可寫」是收窄不是改變；REUSE not DRIFT)
- 📎 Contract Stamps: FOUND (2 across 2 files — sub-4-1 AC #7 `[@contract-v1]` additive ack；sub-1-5b `ProcessItemOptions`／`ProcessOutcome` `[@contract-v1]` 未改)
- 🎭 A11y Pre-Flight: PASS (2 components checked — CandidateListPanel／GenerationConsentView, 0 jsx-a11y warnings on touched files, 0 introduced by this story; disabled checkbox keeps its aria-label, badge text is real text not colour-only)
- 🔌 Route Sync: N/A (no new backend route; candidates envelope additive on existing GET)
- 🎨 UX Verification: PARTIAL — `flow-f-subtitle-v2/f15-d-v2.png` has no「資料夾無法寫入」state; the badge reuses the route-badge idiom (error tint, 12px) and the disabled checkbox is native. Design frame for the unwritable row is handed to sub-6-10b's F15 `.pen` update (noted there).
- 全回歸：`pnpm nx test api` ✅、`pnpm nx test web` 255 files / 3130 tests ✅、`test:cleanup` 無殘留；`pnpm lint:all` go vet／staticcheck 過、eslint 0 errors、prettier 唯一紅字為未追蹤本機檔。

### Discovery Triage

- ③ backlog-with-carry-forward-link — `docs/deployment.zh-TW.md` 不存在（Rule 17 既有缺口）→ `backlog-deployment-doc-zhtw-twin`（filed 2026-09-04）。
- ① expand-scope-in-place — 探針改放 leaf 套件 `internal/fsprobe`（Dev Notes 已裁定，Task 1 路徑更新）。

### Change Log

| Date | Change |
| --- | --- |
| 2026-09-04 | Task 1 — `internal/fsprobe` leaf 套件（ProbeWritable + 5 tests）；`ErrSubtitleTargetNotWritable`；Rule 7／Rule 19 清單、`boundaries_test.go`。 |
| 2026-09-04 | Task 2 — `WithWritableProbe`、`ProcessItem` Step 2b、`restoreMediaStatus`；`TestProcessItem_UnwritableTargetSpendsNothing`（零 LLM／零 route／run failed／status 還原）。 |
| 2026-09-04 | Task 3 — 候選分析 writable/blocker/unwritable_count + 每目錄一次探針；兩條測試 + `TestMain` 寬鬆探針。 |
| 2026-09-04 | Task 4 — FE 型別、`isWritable`／`selectableIds`、bulk 選取排除、列徽章與 disabled；specs +5。 |
| 2026-09-04 | Task 5 — `docs/deployment.md` File Permissions 段；zh-TW twin 缺口立案。 |

### File List

- `apps/api/internal/fsprobe/writable.go`、`writable_test.go`（new）
- `apps/api/internal/subtitle/errors.go`、`pipeline.go`、`process_item.go`（modified）+ `process_item_test.go`
- `apps/api/internal/services/generation_candidates.go`（modified）+ `generation_candidates_test.go`
- `apps/api/internal/boundaries_test.go`（modified）
- `apps/web/src/services/subtitleService.ts`、`apps/web/src/components/subtitle/consent/{consentSelection.ts,CandidateListPanel.tsx,GenerationConsentView.tsx}`（modified）+ `consentSelection.spec.ts`、`CandidateListPanel.spec.tsx`
- `docs/deployment.md`、`project-context.md`（modified）
- `_bmad-output/implementation-artifacts/sub-6-1-preflight-writable-target.md`、`sub-6-10b-candidate-identity-frontend.md`（note）、`sprint-status.yaml`
