# Disc：即時活動的失敗原因 B（前端）——生成畫面的即時活動跟活動記錄講同一句失敗原因

Status: done

**Source:** 同 A 單。`disc-2026-10-activity-subtitle-run-time-b`（PR #828）合併後，dev 回報：生成字幕畫面的「即時活動」認不出「寫不進資料夾」和「中途停止（取消）」，同一次失敗那邊寫「生成失敗」，活動記錄寫「影片所在的資料夾寫不進字幕」。
⚖️ **Alexyu 2026-10-19 裁定 2A：** 即時活動改用活動記錄那份失敗原因文字（`FAILURE_COPY_BY_REASON`）。
⚖️ **SM 裁定（Bob）：** 後端在即時事件送原因代碼（A 單），前端改讀代碼、**刪掉英文字串比對**（`failureCopy` 的 `TABLE`）。理由與拆單見 A 單開頭。

設計稿：**無版面變更，沿用 F11-SPEC-LOG `tT3JX`**（`_bmad-output/screenshots/flow-f-subtitle-v2/f11-spec-log.png`）的 ⑥「失敗」列與 ⑤／⑤b「已停止」列——列的種類不變，只是原因那一格多兩種字。生成工作區本體 F11-D-v2 `l8FsB`／F11-M-v2 `PXB0z`。

**依賴：** 照 A 單 AC #1 `[@contract-v1]` 寫，**可以先開工**。⛔ **合併順序：A 先合併，本單後合併。** 本單刪掉字串比對；若後端還沒送 `failure_reason`，今天認得的「已達預算上限」「AI 帳號沒有額度」等也會退成「生成失敗」。

## 查到的事（2026-10-19，讀 `origin/main` 9331d1e2）

- **即時活動的文字從哪來**：`feedRowView`（`apps/web/src/components/subtitle/generationEventCopy.ts:182-329`）的 `case 'failed'`（`:234-255`）：
  - 單次生成（`row.reason === null`）→ `failureCopy(row.error)`；`stopped`（預算、額度）→ 黃色「已停止」（`:238-247`），其他 → `failedView` 紅色「失敗」（`:248`、`:331-340`）。
  - 批次成員 `reason === 'error'` 且有 `row.error` → `failureCopy(row.error).text`；其他 reason → `queueRowLabel`（`generationQueueRow.ts:48-54`：沒有可用的字幕來源／這部正在別處處理／生成失敗）。
- **`failureCopy` 是英文子字串比對**（`generationEventCopy.ts:90-121`），只認預算、音軌、ffmpeg、語音辨識未設定、額度、金鑰、逾時七種；`FAILURE_COPY_BY_REASON`（`:64-81`）十種代碼的文字已是同一份物件，`TABLE` 只是引用它。`:38-46` 註解自己寫明「文字共用，比對規則不共用，所以即時活動對寫不進資料夾仍寫生成失敗」。
- **hook 收到什麼**（`apps/web/src/hooks/useGenerationJobsFeed.ts`）：
  - `TranscriptionPayload`（`:170-184`）有 `error?`、`message?`，**沒有原因代碼**；`ChangedItem`（`:191-197`）只有 `status`、`reason`。
  - `FeedRow` 的 failed（`:96-103`）存 `reason` 與原始 `error`（「never displayed as-is」）。
  - 單次失敗：`SINGLE_FAILED`（`:416-450`）把 `payload.error ?? payload.message` 存進 row（`:447`）。
  - 批次成員：`memberErrors`（宣告 `:154`、初值 `:165`、`:330`、`:428-436`、`:482-485`、`:495`、`:521`、`:532`、`:556-559`）先記住同一部片 `transcription_failed` 的字串，`changed_item` 失敗且 `reason === 'error'` 時拿出來（`:521`）。翻譯路線的批次失敗不會送 `transcription_failed`，所以拿到 `null` → 「生成失敗」。
- **A 單之後事件會帶 `failure_reason`**（經 `snakeToCamel` 變 `failureReason`）：`transcription_failed` 每一個都有；`changed_item.failure_reason` 一定在，`failed`＋`error` 時是十個代碼之一，其餘 `""`。
- **共用查表已存在**：`failureCopyForReason(reason)`（`generationEventCopy.ts:84-88`），未知／空字串／`null` → `other`「生成失敗」，用 `hasOwnProperty` 擋 `constructor` 之類。活動記錄用它（`recentEventCopy.ts:79`）。
- **列的版面**：`FeedRowItem`（`GenerationWorkspaceV2.tsx:383-422`）——片名那格 `min-w-0 shrink`＋`truncate`，原因那格 `shrink-0`（`:398-405`，「The film name may truncate; the reason after it stays whole」）。「影片所在的資料夾寫不進字幕」13 個字，比「生成失敗」長很多，**窄畫面上片名會被擠得更短**。
- **測試現況**：
  - `generationEventCopy.spec.ts:12-80` `describe('failureCopy')` 全是英文字串 → 文字；A 單 AC #5 已把每一條搬到 Go 測試。`:160-176`（④）、`:178-192`（⑤）、`:194-209`（⑤b）用 `error:` 字串餵 `feedRowView`。`:341-380` `describe('FAILURE_COPY_BY_REASON')` 保留。
  - `useGenerationJobsFeed.spec.ts` 送 `transcription_failed` 帶 `error` 的有 `:386-405`、`:503-510`、`:550-556`、`:609-627`（「reason "error" keeps the member transcription_failed text」）、`:813-820`、`:888-895`；期望值 `error:` 在 `:401`、`:605`、`:624`。
  - `GenerationWorkspaceV2.spec.tsx:655-722`：#13「原始字串（含伺服器路徑）絕不上畫面」與「單次預算停止是已停止」用 `error:` 餵列。
- **畫廊夾具與視覺基準**：即時活動的失敗列夾具只有 `-gallery.fixtures.tsx:7284-7285`（`reason: 'busy_elsewhere'`／`'skipped'`、`error: null`），走 `queueRowLabel`，**文字不會變** → `generation-workspace-v2/*` 基準（darwin＋linux 各 5 組）不應變。
- **E2E**：`tests/e2e/generation-workspace.spec.ts`、`generation-workspace-mobile.spec.ts` 只數 `workspace-feed-row` 列數與找「翻譯中」（`:147`、`:203`、`:214`、mobile `:155`），stub 不送 `transcription_failed`；`:63` 的批次失敗是 `reason: 'skipped'`。→ **E2E 與 data-testid 不受影響。**

## Story

身為只有我一個人在用的 NAS 片庫主人，
我想要生成字幕畫面的即時活動，對失敗的原因跟活動記錄講同一句話（例如「影片所在的資料夾寫不進字幕」），
這樣我在生成畫面當下就知道要去修資料夾，而不是看到「生成失敗」再跑去活動頁查。

## Acceptance Criteria

1. **型別（ack A 單 AC #1 `[@contract-v1]`）：** `TranscriptionPayload` 加 `failureReason?: string`；`ChangedItem` 加 `failureReason?: string`；`GenerationBatchItemState`（`apps/web/src/services/subtitleService.ts:173-176`）加 `failureReason?: string`（註解：後端一定送，`''` 表示不是 `error` 失敗；可選只是為了舊夾具）。
2. **hook 只存代碼，不存原始錯誤：** `FeedRow` failed 的 `error: string | null` 換成 `failureReason: string | null`。
   - 單次失敗：`failureReason = payload.failureReason || null`。
   - 批次成員失敗：`failureReason = changed.reason === 'error' ? (changed.failureReason || null) : null`。
   - `memberErrors` 整個拿掉（含 `SINGLE_FAILED` 對批次成員只 settle、不再記字串）——代碼已經在 `changed_item` 上，不用再拿 `transcription_failed` 去配。
   - 斷言：整個 `feed` 的 `JSON.stringify` 不含送進來的原始 `error` 字串（例如 `permission denied`、`/media/`）。
3. **畫面文字（`feedRowView` failed）：**
   - 單次：`copy = failureCopyForReason(row.failureReason)`；`copy.stopped` → 黃色「已停止」（`budget_exceeded`、`credit_exhausted`，跟今天一樣）；其餘 → 紅色「失敗」＋`copy.text`。
   - 批次 `reason === 'error'`：`failureCopyForReason(row.failureReason).text`（紅色「失敗」）；其他 reason 照舊 `queueRowLabel`。
   - 結果：`target_not_writable` →「失敗 · 片名 · 影片所在的資料夾寫不進字幕」；`cancelled` →「失敗 · 片名 · 中途停止」；沒有代碼／未知 →「生成失敗」。讀螢幕軟體的 `announce` 跟著變（`失敗：片名，影片所在的資料夾寫不進字幕`）。
4. **刪掉字串比對：** 刪 `TABLE`（`generationEventCopy.ts:90-111`）、`FALLBACK`（`:113`）、`failureCopy()`（`:115-121`），全 repo 不再有 `failureCopy(` 呼叫（`git grep -n "failureCopy(" -- apps` 只剩 `failureCopyForReason`）。檔頭（`:8-20`）與 `RunFailureReason` 上方（`:38-46`）註解改成「原因代碼由後端判斷（`models.SubtitleRunFailureReason`），即時事件與活動記錄都帶代碼；前端只查 `FAILURE_COPY_BY_REASON`」。
5. **兩邊一致的測試（會變紅的那種）：** `generationEventCopy.spec.ts` 新增：對十個代碼逐一，
   - 單次失敗列的原因那格（`feedRowView(...).parts[1]`）`===` `FAILURE_COPY_BY_REASON[code].text`；
   - 而且 `===` 活動記錄同一個代碼的說明後半段（`recentSubtitleDetail({ kind:'subtitle', result:'failed', mediaTitle:'奧本海默', failureReason: code, … })` 以 ` · ${text}` 結尾）。
   故意改壞驗證：在 `feedRowView` 把任一代碼寫死成別的字，或回到用 `row.error` 比對 → 紅。
6. **既有行為不退步（改寫既有測試，不是刪掉）：**
   - ④（`:160-176`）：`reason: 'error', failureReason: 'timeout'` → `['奧本海默', '處理逾時']`；`failureReason: null` → `生成失敗`；`skipped` 照舊。新增 `failureReason: 'target_not_writable'` →「影片所在的資料夾寫不進字幕」。
   - ⑤、⑤b（`:178-209`）：`failureReason: 'budget_exceeded'`／`'credit_exhausted'` → 黃色已停止，文字不變。
   - `describe('failureCopy')`（`:12-80`）刪除（A 單 AC #5 已在 Go 端覆蓋同一組字串）；`describe('FAILURE_COPY_BY_REASON')` 保留。
   - `useGenerationJobsFeed.spec.ts`：AC #2 的三種情況各一條；把 `:609-627` 改成「批次 `reason: 'error'` 取 `changed_item.failure_reason`」，並加「同一部片先收到 `transcription_failed` 也不影響，以 `changed_item` 為準」；其他送 `error` 的測試補 `failure_reason`，期望值改 `failureReason`。
   - `GenerationWorkspaceV2.spec.tsx:655-722`：列改用 `failureReason`；#13 保留「頁面不含 `/mnt/media`」斷言（改成即使事件帶了原始字串，hook→畫面也不會出現——用 hook＋假 EventSource 或直接斷言 `FeedRow` 型別沒有 `error` 欄位都可以，擇一寫清楚）；加一列 `target_not_writable` 斷言畫面文字。
7. **夾具：** `-gallery.fixtures.tsx:7284-7285` 的 `error: null` 改 `failureReason: null`（畫面不變）。
8. **視覺基準不變：** `AI_PROVIDER=claude pnpm test:visual --grep generation-workspace-v2`（不加 `--update-snapshots`）全綠；**不重產、不 commit 任何 png**。
9. **窄畫面量一次：** 在生成工作區手機寬度（390，F11-M-v2）放一列「失敗 · <長片名> · 影片所在的資料夾寫不進字幕」（hook 測試資料或 dev 頁皆可），量片名那格實際寬度並寫進 Completion Notes。片名仍有字可看（不是只剩「…」）→ 過；只剩「…」或原因被截 → **停下來找 Sally**，不要自己改版面。
10. `pnpm nx run web:test`（subtitle＋hooks＋activity 相關 spec）、`pnpm run lint:all`（看「✖ N problems (X errors」那行）、typecheck、`pnpm run format:check` 綠。

## Tasks / Subtasks

- [x] **T1 型別＋hook（AC #1、#2）**：`subtitleService.ts`、`useGenerationJobsFeed.ts`（`TranscriptionPayload`、`ChangedItem`、`FeedRow`、`SINGLE_FAILED`、`BATCH`、拿掉 `memberErrors`）＋ `useGenerationJobsFeed.spec.ts`。
- [x] **T2 畫面文字（AC #3、#5）**：`generationEventCopy.ts` `case 'failed'` 改查 `failureCopyForReason`；新增 AC #5 一致性測試。
- [x] **T3 刪字串比對（AC #4、#6）**：刪 `TABLE`／`FALLBACK`／`failureCopy`、改註解；改寫 `generationEventCopy.spec.ts`、`GenerationWorkspaceV2.spec.tsx` 的既有案例。
- [x] **T4 夾具＋收尾（AC #7–#10）**：夾具欄位改名；visual 不更新跑一次；窄畫面量測；lint／typecheck／format。

## Dev Notes

> ⚠️ STALE [@contract-v1→v2]: upstream disc-2026-10-activity-subtitle-run-time-a AC #1 bumped 2026-10-19; re-confirm against v2 before dev — see disc-2026-10-run-timeout-not-cancel

### 上游合約已變（`disc-2026-10-run-timeout-not-cancel`，合併順序在本單之前，2026-10-19）

- `failure_reason` 值域變成**十一個**代碼：多了 `queue_timeout`，`FAILURE_COPY_BY_REASON.queue_timeout = { text: '處理逾時', stopped: false }` 已經在（⚖️ Sally 2026-10-19 追認：文字跟 `timeout` 相同、紅色失敗）。
- AC #5「對十個代碼逐一」→ 改成對 `Object.keys(FAILURE_COPY_BY_REASON)` 逐一，以後再加代碼就不用改測試。
- `TABLE`／`failureCopy()` 上一張沒動（即時活動實際收不到 `queue_timeout`：自動生成不送 `transcription_failed`），本單照原計畫刪掉。

### A 單實作確認（2026-10-19，A 單合併）

- 鍵名與值域照上面：`transcription_failed.failure_reason` 每一個都有（十一個代碼之一）；`changed_item`／`items[]` 每一項的 `failure_reason` 一定在，`failed`＋`reason: 'error'` 才是代碼，其餘 `''`。
- 多一個地方帶這個鍵：`POST /api/v1/subtitles/generation-batch` 202 回應的 `progress.items[]`（同一個型別），AC #1 在 `GenerationBatchItemState` 加 `failureReason?` 就涵蓋。
- 後端分類的是「跟紀錄一樣切過（約 1 KB）」的文字，所以即時活動與活動記錄連「錯誤訊息很長、關鍵字被切掉」的情況也講同一句（都是「生成失敗」）。
- 前端 `describe('failureCopy')` 的每個輸入已在 Go 端 `TestSubtitleRunFailureReason_LiveLogStrings` 落地，結果全部一致，AC #6 可以照計畫刪掉那組測試。

### 不要做的事

- 不要在前端保留任何「英文錯誤字串 → 原因」的比對當備援（那正是兩邊說法不一致的來源）。沒有代碼就是「生成失敗」。
- 不要改 `FAILURE_COPY_BY_REASON` 的文字或 `stopped`（活動記錄共用；文字是 ⚖️ Sally 2026-10-19 追認過的）。
- 不要動 `queueRowLabel` 與批次的 `skipped`／`busy_elsewhere` 文字（不在裁定範圍）。
- 不要改詳情對話框的失敗面板（`GenerationProgressV2`／`useGenerationProgress`）：它顯示後端的 `message`／`error`，不在本裁定範圍。
- 不要改列的版面（`FeedRowItem` 的 `shrink`／`truncate`）；AC #9 量完有問題就找 Sally。
- 不要跑 `test:visual:update`，不要產或 commit `-linux.png`。

### 預設做法（要改請 Sally 裁定）

- **「中途停止」在即時活動是紅色「失敗」**：沿用 `FAILURE_COPY_BY_REASON.cancelled.stopped = false`（活動記錄同一個物件）。實際上很少出現——單次生成沒有取消按鈕（`GenerationWorkspaceV2.tsx:11-12`「cancel is batch-wide only」），批次取消走「批次已取消」那一列（`generationEventCopy.ts:296-305`）；會看到「中途停止」的主要是等抽軌名額逾時被放棄。若 Sally 認為該是灰色／黃色「已停止」，那是改 F11-SPEC-LOG 的事，另開單。
- **F11-SPEC-LOG 有沒有列舉原因文字**：請 Sally 用 Pencil MCP 唯讀看 `tT3JX` ⑥ 列。只是示意一句（例如「生成失敗」）就不用改稿；若列了完整清單，補兩種字是設計跟進，不擋本單。

### 測試標準

- Vitest＋Testing Library；斷言用可見文字、testid 與 `FeedRow` 物件（Rule 16）。不要背景跑測試。

### E2E data-testid

- 沒有新增、改名或移除 testid（`workspace-feed-row` 不變）。E2E 不受影響。

### 視覺基準

- **不影響**（夾具的失敗列不走這條路，見「查到的事」）。

### Project Structure Notes

- 動到：`apps/web/src/services/subtitleService.ts`、`apps/web/src/hooks/useGenerationJobsFeed.ts`（＋spec）、`apps/web/src/components/subtitle/generationEventCopy.ts`（＋spec）、`apps/web/src/components/subtitle/GenerationWorkspaceV2.spec.tsx`、`apps/web/src/routes/test/-gallery.fixtures.tsx`（只改兩個欄位名）、`_bmad-output/implementation-artifacts/sprint-status.yaml`。
- 跟 `disc-2026-10-activity-run-time-sr-label` 沒有共用檔案。

### References

- [Source: _bmad-output/implementation-artifacts/disc-2026-10-live-activity-failure-reasons-a.md AC #1 `[@contract-v1]`]
- [Source: apps/web/src/components/subtitle/generationEventCopy.ts:1-121, 182-340]
- [Source: apps/web/src/components/subtitle/generationQueueRow.ts:48-54]
- [Source: apps/web/src/hooks/useGenerationJobsFeed.ts:96-103, 154-165, 170-197, 416-450, 482-559]
- [Source: apps/web/src/services/subtitleService.ts:165-176]
- [Source: apps/web/src/components/subtitle/GenerationWorkspaceV2.tsx:11-12, 383-422；.spec.tsx:655-722]
- [Source: apps/web/src/components/activity/recentEventCopy.ts:76-88]
- [Source: apps/web/src/routes/test/-gallery.fixtures.tsx:7270-7300]
- [Source: project-context.md Rule 16]

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（dev agent，2026-10-19）

### Debug Log References

- `npx vitest run src/components/subtitle src/hooks src/components/activity src/services src/routes`：91 檔／1788 條全綠。
- `pnpm nx run web:typecheck` 綠；`pnpm run lint:all`「✖ 170 problems (0 errors, 170 warnings)」（全是既有警告，改動的檔案沒有新警告）；format:check 綠。
- `AI_PROVIDER=claude pnpm test:visual`（不更新）全綠，沒有任何 png 變動。（`--grep generation-workspace-v2` 抓不到測試：視覺測試是單一 test 跑全部夾具，所以整套跑一次。）
- Go 註解（`models/subtitle_run_failure.go`、`_test.go` 兩處提到「-b 合併後」）改成現在式；`go vet`、`staticcheck`、`go test ./internal/models` 綠。
- 故意改壞驗證：`feedRowView` 把 `cancelled` 寫死成「已取消」→ 一致性測試 `cancelled` 與單次失敗測試變紅，還原後綠。

### Completion Notes List

- **合約照 main 上的程式確認**（#844）：`transcription_failed.failure_reason`、`changed_item.failure_reason`／`items[].failure_reason`，十一個代碼（含 `queue_timeout`）；`''` 視同沒有代碼。
- **AC #2**：`FeedRow` failed 只存 `failureReason`；`TranscriptionPayload` 拿掉 `error?`（hook 不再讀，詳情對話框自己的型別不受影響）；`memberErrors` 全部移除。批次成員的 `transcription_failed` 現在只結束那一步，不再記任何字串。
- **AC #5**：一致性測試對 `Object.keys(FAILURE_COPY_BY_REASON)` 逐一（十一個），同時比單次列、批次 `error` 列、活動記錄說明三處。
- **AC #6 #13**：選「hook＋假 EventSource」——在 `GenerationWorkspaceV2.spec.tsx` 用真的 hook 接假串流，事件帶原始字串（含 `/mnt/media` 路徑），斷言整頁不含路徑、`permission denied`、`no audio track`。
- **AC #9 窄畫面量測（本機真頁面、假 SSE，片名「魔戒首部曲：魔戒現身 加長版導演剪輯特別收藏版」＋原因「影片所在的資料夾寫不進字幕」）**：
  - 390（F11-M-v2，展開即時活動）：列寬 356px，原因 182px 完整，**片名那格 34px＝只剩 1 個字＋「…」**（「魔…」「和…」「花…」，連 4 個字的「花月殺手」也只剩 1 字）。照 AC 字面（不是只剩「…」、原因沒被截）算過，但片名實際上認不出來 → 回報給 Alexyu／Sally 裁定，沒有改版面。
  - 1440：即時活動欄寬 398px，片名那格 76px（約 5 字），「和平使者 S01E01」會變「和平使者 …」，集數看不到。
- **UX 驗證**：本機 `/activity?view=generation` 收到寫不進資料夾、取消、排隊逾時、其他四種單次失敗＋一個批次翻譯後寫不進，截 1440／390；活動頁同四部的說明逐字相同（「和平使者 S01E01 · 影片所在的資料夾寫不進字幕」「芭比 · 中途停止」「奧本海默 · 處理逾時」「沙丘：第二部 · 生成失敗」）。列的樣子跟 F11-SPEC-LOG ⑥ 一致（紅色三角＋失敗・片名・原因）。
- **自我審查發現、範圍外**：同一個畫面左邊的「生成佇列」列，`error` 失敗的項目仍寫「生成失敗」（`queueRowLabel` 不看代碼），右邊即時活動已寫「影片所在的資料夾寫不進字幕」。本單 Dev Notes 明定不動 `queueRowLabel`，資料已經在 `items[].failure_reason`，要不要跟進待裁定。

### File List

- apps/web/src/services/subtitleService.ts
- apps/web/src/hooks/useGenerationJobsFeed.ts
- apps/web/src/hooks/useGenerationJobsFeed.spec.ts
- apps/web/src/components/subtitle/generationEventCopy.ts
- apps/web/src/components/subtitle/generationEventCopy.spec.ts
- apps/web/src/components/subtitle/GenerationWorkspaceV2.spec.tsx
- apps/web/src/routes/test/-gallery.fixtures.tsx
- apps/api/internal/models/subtitle_run_failure.go（只改註解）
- apps/api/internal/models/subtitle_run_failure_test.go（只改註解）
- _bmad-output/implementation-artifacts/disc-2026-10-live-activity-failure-reasons-b.md
- _bmad-output/implementation-artifacts/sprint-status.yaml
