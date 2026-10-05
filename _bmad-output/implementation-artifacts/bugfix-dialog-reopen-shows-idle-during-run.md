# Story bugfix：生成字幕跑到一半關掉「管理字幕」再打開，會直接看到進度，不會再看到「生成字幕 $0.80」

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As the person who started generating subtitles for a film or an episode and then closed the「管理字幕」window,
I want reopening the window to show the run's progress right away,
so that I can see how far it has got — and am never shown a paid「生成字幕」button for something that is already running.

## Context

**回報**：Alexyu，2026-10-05，**P1**。真實重現：在 NAS 上對影集《末日光明》S01E02（單集）按「生成字幕」，關掉視窗再打開，看到的是「還沒開始」的畫面（按鈕「生成字幕 $0.80」），不是生成進度。

**設計稿已經寫明這是錯的，不需要新設計**：F1 規格的助語「生成進行中」是「本集正在生成字幕——開啟即接續顯示進度，不會重複啟動」（`_bmad-output/implementation-artifacts/9R-UX-episode-row-cta-design.md:255`）。進度畫面 `GenerationProgressV2`（F3-D-v2 `JbXai`）已經存在，只是打開視窗時沒有走到那裡。

### 🔴 查到的事（main `0426d635`；行號皆為現況）

1. **視窗一打開永遠是 idle**：`genView` 的初始值是 `'idle'`（`apps/web/src/components/subtitle/ManageSubtitleDialogV2.tsx:202`）。
2. **只有按下按鈕才會切到進度**：`trigger` 的 `onSuccess` 在「已啟動」或後端回 409「已在進行中」時才 `setGenView('progress')` ＋ `generation.startTracking(mediaId)`（`ManageSubtitleDialogV2.tsx:278-291`）。打開時沒有任何地方先問「這部片現在是不是正在跑」。
3. **關掉視窗會把畫面重設回 idle**：`handleOpenChange(false)` 會 `generation.reset()`、`setGenView('idle')`（`ManageSubtitleDialogV2.tsx:304-324`），而工作在伺服器上繼續跑（檔頭註解 `ManageSubtitleDialogV2.tsx:45-46`）。單集版的視窗由 `SeasonAccordion.tsx:143-144` 依 `subtitleEpisode` 掛載，關掉即卸載，重開就是新的 idle；電影版（`LocalDetailV2.tsx:442-443`）一直掛著、靠上面的重設回到 idle。兩條路的結果相同。
4. **按鈕在 idle 畫面是可以按的**：`ButtonCost label="生成字幕"`（`ManageSubtitleDialogV2.tsx:661-669`）只看估價（`costView.cost`）與 `trigger.isPending`。按下去後端會回 409、畫面才接上進度——不會重複啟動，但使用者看到的是一顆標著價錢、好像會再花一次錢的按鈕。
5. **後端已經知道答案**：`TranscriptionService.IsInProgress(mediaID)`（`apps/api/internal/services/transcription_service.go:469-475`）讀的是同一張單飛表 `inProgress`（`transcription_service.go:198`；`acquireJob` 在 `:678-687` 寫入），手動點的（solo）與批次的都在裡面。兩條觸發路徑本來就用它回 409（`apps/api/internal/handlers/transcription_handler.go:139`（電影）、`:223`（單集））。
6. **活動頁的「進行中」清單不適合拿來用**：`ActivityService.activeJobsSection`（`apps/api/internal/services/activity_service.go:144-188`）只回 `kind`／`detail`（片名字串），**沒有 media id**；而且 `ActivityProgress` 刻意略過批次工作（`transcription_service.go:733-744`）。拿片名去比對會誤判同名片、也看不到批次裡的這一集。
7. **進度串流本身就能接上已在跑的工作**：`useGenerationProgress.startTracking(mediaId)` 的檔頭寫明它也是 409 接續的路（`apps/web/src/hooks/useGenerationProgress.ts:30-31`），`START` 先把畫面放在「抽取音訊」（`useGenerationProgress.ts:140-141`），下一個事件到了就跳到真正的階段。

### ⚖️ 建單裁定（2026-10-05，SM；Alexyu 可在 review 推翻）

1. **加一條只讀的小路由問「正在跑嗎」**：`GET /api/v1/movies/:id/transcribe/status`、`GET /api/v1/episodes/:id/transcribe/status` → `{ "in_progress": bool }`。只讀單飛表，不查資料庫、不 ffprobe、不花錢，所以很快。
   - **不選**「把 `in_progress` 塞進估價回應」：估價可能排在 ffprobe 限流後面（`transcription_handler.go:340` 的說明），等它回來才切到進度會慢；而且「價錢」與「正在跑嗎」是兩個問題，混在一起之後估價失敗就連帶不知道在不在跑。
   - **不選**活動頁清單（見 🔴 #6）。
   - 單集路由跟觸發路由一樣，只在有接單集查詢時才掛（`transcription_handler.go:90-100` 的慣例）。
2. **視窗打開時先問，問到「在跑」就直接顯示進度並接上串流**；問到「沒在跑」或問失敗 → 照舊顯示 idle（失敗時保守：409 仍會擋重複啟動）。
3. **還沒問到答案前，「生成字幕」不可按**：按鈕顯示成載入中（`ButtonCost` 的 `loading` 狀態，與估價還沒回來時同一個樣子，`apps/web/src/components/ui/ButtonCost.tsx:8-15`），所以不會出現「打開那一瞬間按得到」的空檔。
4. **只在 idle 時接手**：已經在進度／失敗／設定錯誤畫面時，問到的答案不改畫面。

## Acceptance Criteria

1. **後端：新路由回報「這部片／這一集現在是不是在生成字幕」。**
   - `GET /api/v1/movies/:id/transcribe/status` 與 `GET /api/v1/episodes/:id/transcribe/status` 回 `200 {success:true, data:{in_progress:bool}}`，值來自 `IsInProgress(id)`；不查資料庫。
   - 單集路由只在 handler 有接單集查詢時掛載（沒接 → 404），與 `POST /episodes/:id/transcribe` 相同。
   - Go 測試：電影在跑 → true、沒在跑 → false；單集同；確認查的是路徑上的那個 id；沒接單集查詢 → 單集路由 404。
2. **前端：打開時正在跑 → 直接顯示進度。**（電影、單集各一）
   - 打開視窗就問狀態；回 `inProgress: true` → 顯示生成進度（標題「生成字幕 — …」、`GenerationProgressV2`），並以同一個 media id 接上進度串流；畫面上**沒有**「生成字幕」按鈕，也**沒有**呼叫觸發 API。
3. **前端：沒在跑 → 照舊。**（電影、單集各一）
   - 回 `inProgress: false` → 照舊顯示 idle 與帶價錢的「生成字幕」，按下去照舊啟動。
4. **前端：答案回來前不可按；問失敗不擋路。**
   - 狀態還沒回來時「生成字幕」是載入中、按了不會觸發。
   - 狀態問失敗 → 照舊 idle、可按（409 仍擋重複啟動）。
   - 每次打開都重新問（關掉時清掉快取，與估價同一個做法 `ManageSubtitleDialogV2.tsx:313-319`）。
   - 影集（series）層級不問（它沒有生成路由）。
5. **真實形狀（Rule 28）**：一條 API e2e 打真的後端路由，確認回應形狀是 `{success:true, data:{in_progress:false}}`；前端 service 測試用同一個形狀（snake_case → camelCase）。
6. **不准回歸**：`ManageSubtitleDialogV2.spec`、`transcription_handler_test.go`、`transcription_episode_handler_test.go` 其他案例全綠；視覺 gallery 的三個 `ManageSubtitleDialogV2` fixture 要種「沒在跑」的狀態，畫面不變（CI 沒有後端）。
7. **CI**：`pnpm run lint:all`、`pnpm nx test web`、`go test ./...`、`staticcheck`（2026.1）、`pnpm run format:check` 綠；紅／守（Rule 16）。

## Tasks / Subtasks

- [x] **Task 1 — 後端：`GET …/transcribe/status`（AC: #1）**＋handler 測試（先紅）
- [x] **Task 2 — 前端 service＋hook（AC: #4, #5）**：`getTranscriptionStatus`、`useTranscriptionStatus`＋測試
- [x] **Task 3 — 視窗接上（AC: #2, #3, #4）**：先寫電影／單集「在跑／沒在跑」四條會紅的測試，再改
- [x] **Task 4 — gallery fixture 種狀態、API e2e（AC: #5, #6）**
- [x] **Task 5 — 驗證與收尾（AC: #7）**

## Dev Notes

### 這張的重點

- 問題出在「打開時沒有問」，不是進度畫面本身——進度畫面、串流、409 接續都是現成的，不要動它們。
- **不要順手改「現有字幕」那一區的顯示**：另一張進行中的工作（`ManageSubtitleDialogV2` 顯示真實字幕軌）也在改這個檔案、會在本張之後合併。本張只動 genView 的起點與按鈕可按與否，盡量少碰 `buildTrackRows`／`renderTracksSection`。
- `data-testid` 一律不拿掉、不改名；新增的才可以自取。改之前先 grep `tests/e2e`。

### 上游契約（Rule 20）

- `POST /movies/:id/transcribe`、`POST /episodes/:id/transcribe` 的 409 `TRANSCRIPTION_IN_PROGRESS` 行為不變（本張只讀不寫）。
- `GET …/transcribe/estimate`（dsr-6a AC #2 `[@contract-v1]`）形狀不變——本張刻意不把狀態塞進去。confirmed against `[@contract-v1]` (Story dsr-6a AC #2)：不動。

### 已知限制（不在本張修 → `disc-2026-10-reattach-progress-starts-at-extracting`）

- 串流接上後，畫面先停在「抽取音訊」直到下一個事件到（🔴 #7）。翻譯階段事件很密，但若剛好在一段很長的語音辨識中間打開，會停一陣子。409 接續本來就是這樣；要修得讓後端記住目前階段並在狀態路由回傳。
- 「問到在跑」與「串流連上」之間若工作剛好結束，完成事件會錯過、畫面停在進度。視窗間隔極短（毫秒級），409 接續同樣有此窗口。

### Time-dependent visual coverage

- N/A — no wall-clock-reading components touched.

### References

- `apps/web/src/components/subtitle/ManageSubtitleDialogV2.tsx:45-46, 202, 278-291, 304-324, 661-669`
- `apps/web/src/hooks/useGenerationProgress.ts:30-31, 140-141`
- `apps/web/src/hooks/useTranscriptionEstimate.ts`（hook 寫法範本）
- `apps/api/internal/handlers/transcription_handler.go:90-100, 139, 223, 340`
- `apps/api/internal/services/transcription_service.go:198, 469-475, 678-687, 733-744`
- `apps/api/internal/services/activity_service.go:144-188`
- `_bmad-output/implementation-artifacts/9R-UX-episode-row-cta-design.md:255`（「開啟即接續顯示進度，不會重複啟動」）

## Dev Agent Record

### Agent Model Used

Claude（Claude Code，雲端環境）

### Debug Log References

- 本機真後端冒煙：`GET /api/v1/movies/abc/transcribe/status` → `{"success":true,"data":{"in_progress":false}}`；episodes 同。
- 本機真瀏覽器冒煙（暫時 spec，跑完刪除）：狀態 stub 成 `in_progress:true`、SSE stub 一個 `translation_progress 42%` → 打開視窗直接是「生成字幕 — …」進度、步驟停在「翻譯中 42%」、畫面沒有「生成字幕」鍵、觸發 POST 次數 0。

### Completion Notes List

- **先紅後綠**：後端 4 條 handler 測試先 404 紅；前端 4 條（電影／單集「打開時在跑」、「答案回來前不可按」、「開始→關→重開」）先紅，「沒在跑照舊」與 series 兩條是防回歸、本來就綠。
- **突變檢查（Rule 16）**：①拿掉關視窗時清狀態快取 → 「開始→關→重開」紅；②生成鍵改回只看估價 → 「答案回來前不可按」紅；③拿掉打開時接手的 effect → 四條「在跑」測試紅（即首輪紅燈）。
- **真實形狀（Rule 28）**：新增 `tests/e2e/transcription-status.api.spec.ts` 打真後端（本機 2/2 綠）；前端 service 測試用同一個 body。
- **e2e**：`manage-subtitle-mobile.spec.ts`、`glossary-mobile.spec.ts` 補上狀態 stub（回「沒在跑」）讓畫面確定；本機 chromium 7/7 綠。沒有拿掉或改名任何 `data-testid`。
- **視覺**：三個 `ManageSubtitleDialogV2` gallery fixture 種「沒在跑」，畫面應不變（不需要新基準）。
- **驗證**：`pnpm run lint:all` 綠（含 go vet＋staticcheck 2026.1＋format:check）；`pnpm nx run web:typecheck` 綠；vitest 298 檔／4579 條全綠（`nx test web` 的包裝腳本在此容器結束時送 SIGTERM 導致非 0，vitest 本身 exit 0）；`go test ./...` 只有 `TestBackupService_Restore_PosterFailureKeepsTheRestoredDatabase` 紅——在未修改的 `main` 上同樣紅，原因是此容器以 root 執行（權限型失敗測試對 root 無效），CI 不是 root。
- 沒動「現有字幕」區塊（另一張字幕軌工作會改那裡）。

### Discovery Triage

- ③ backlog-with-carry-forward-link → `disc-2026-10-reattach-progress-starts-at-extracting`（P3）：接上已在跑的工作時進度先停在「抽取音訊」、以及「問到在跑→串流連上」之間工作剛好結束的極短窗口。409 接續本來就有同樣的限制。
- 本機 `go test` 的 backup 測試在 root 下失敗：環境因素、`main` 上同樣失敗，非產品問題，不立案。

### File List

- `apps/api/internal/handlers/transcription_handler.go`（新路由 `GET /{movies|episodes}/:id/transcribe/status`）
- `apps/api/internal/handlers/transcription_handler_test.go`（mock 記錄被問的 id）
- `apps/api/internal/handlers/transcription_status_handler_test.go`（新）
- `apps/web/src/services/transcriptionService.ts`、`.spec.ts`
- `apps/web/src/hooks/useTranscriptionStatus.ts`、`.spec.tsx`（新）
- `apps/web/src/components/subtitle/ManageSubtitleDialogV2.tsx`、`.spec.tsx`
- `apps/web/src/routes/test/-gallery.fixtures.tsx`
- `tests/e2e/transcription-status.api.spec.ts`（新）
- `tests/e2e/manage-subtitle-mobile.spec.ts`、`tests/e2e/glossary-mobile.spec.ts`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`、本檔
