# disc-2026-10-reparse-dialog-stuck-over-tunnel：從外網按「重新解析」，對話框不會再卡在「處理中 0 / 1」

Status: review

<!-- SM Bob create-story 2026-10-27；行號為 main `3207a7d1`。Rule 24 lane ③，Alexyu 10/27 要求開票並指定做這張。 -->

**Source:** 2026-10-27 12:21，Alexyu 從外網（`https://vido.alexunraidhome.org`，Cloudflare Tunnel＋Access）在影集頁勾選將夜→「重新解析」，對話框停在「重新解析中… 處理中 0 / 1…」。後端其實 2 秒就做完了（將夜換成 2026 版、第 1 季 20 集）。

## 查到的事（正式機 log，2026-10-27）

| 時間 | 後端 | 代表什麼 |
|---|---|---|
| 12:21:51.536 | `SSE client registered` | 對話框打開進度串流 |
| 12:21:54.519 | `POST /library/batch/reparse` 200，2.9ms | **剛好 3 秒後**才送出＝`ENRICHMENT_CONNECT_TIMEOUT_MS`（3000）逾時那條路：瀏覽器 3 秒內沒收到串流的開頭（`onopen` 沒觸發），伺服器卻早就送了 |
| 12:21:56.260 | `enrichment completed … succeeded=1` | 比對 2 秒做完，`enrich_complete` 已廣播 |
| 12:22:13 | `GET /library/tv` 200，0.3ms | Alexyu 按重新整理；伺服器馬上回了，瀏覽器的開發者工具卻顯示這個請求一直「Pending」 |
| 12:22:32 | `SSE events stream closed`（開了 40 秒） | 舊頁面關掉、串流斷了 |
| 12:22:33 | 一連串 API 都正常 | **串流一斷，卡住的回應馬上都到了** |

- 對話框顯示「0 / 1」（不是「本輪整理…」）代表**連那個 POST 的回應都沒回到瀏覽器**（`current` 是 POST 回來後才設）。
- 伺服器端沒有問題：本機 `curl -N` 打 `/api/v1/events`，標頭與 `event: connected` 立刻出來，沒有 gzip（`compressionMiddleware` 已排除 `/api/v1/events`），也有 `X-Accel-Buffering: no`。
- 同一時間 Alexyu 正在外網看 Jellyfin（他 10/27 確認）。NAS 上只有一個 cloudflared（`Unraid-Cloudflared-Tunnel`，QUIC），Jellyfin、Nextcloud、Immich、Vido 全部走它；cloudflared log 裡 Vido 的串流和 Jellyfin 的影片串流都在同一條連線（`connIndex=0`、tpe01）。家裡的上傳頻寬被影片吃掉時，Vido 的小回應可能被拖慢——這是可能的助因，沒有量化。NAS 本身沒事（10/27 12:37：可用記憶體 20.5 GB、無 OOM、Vido 140 MB、182 個請求 0 個 5xx）。
- 結論（高度懷疑、未在外網直接重現）：**Cloudflare 那一段把串流卡住，而且串流卡著的時候，同一頁的其他回應也回不來**。我沒辦法自己從外網重現（Cloudflare Access 要 Alexyu 的信箱驗證碼）。

## Acceptance Criteria

1. **串流沒開就關掉**：重新解析前開的進度串流，3 秒內沒打開（或打開前就出錯）→ 關掉，不要讓它一直掛著卡住頁面的其他請求。打開之後才斷線的，照舊交給瀏覽器自己重連。
2. **輪詢當退路**：POST 確定排進比對（`successCount > 0`）之後，每 2 秒問一次 `GET /api/v1/scanner/enrich/status`：
   - `is_active = true` 且有總數 → 顯示「進行中」與數字、目前的片名。
   - `is_active = false` → 顯示「完成」與這一輪的結果、重新讀媒體庫，停止輪詢。
   - 問失敗 → 下一輪再問。
   - 串流先送來「完成」→ 停止輪詢。
   - 關掉對話框／離開頁面 → 停止輪詢。
3. **空的追加輪次不蓋掉結果**：完成時狀態是全 0（後面那輪沒東西做），保留這一輪顯示過的數字（跟 `disc-2026-10-reparse-dialog-empty-followup-pass` 同一個規則）。
4. **沒排進任何一部就不輪詢**（照舊直接關串流）。
5. **伺服器**：兩個 SSE 端點的 `Cache-Control` 改成 `no-cache, no-transform`，叫前面的代理（Cloudflare）不要壓縮、改寫串流——這是 Cloudflare 會照做的標準開關，可能直接解掉根因（要部署後 Alexyu 從外網試）。
6. **測試**：
   - 串流 3 秒沒開→關掉；開之前出錯→關掉；開之後出錯→不關。
   - 輪詢：進行中→完成（有數字、重新讀媒體庫、之後不再問）；問失敗會再問；全 0 保留數字；串流的完成會停止輪詢；stop／unmount 停止輪詢。
   - 媒體庫頁：排進比對才呼叫 `track()`，沒排進不呼叫。
   - SSE handler：標頭含 `no-transform`、第一個事件立刻是 `connected`。

## 不要做的事

- 不改其他 10 個只靠 SSE 的地方（掃描進度、字幕生成、下載進度…）。盤點結果列在下面，另立 `disc-2026-10-sse-over-tunnel-other-streams`，等部署後確認 `no-transform` 有沒有解掉根因再決定。
- 不改 Cloudflare 的設定（那是 Alexyu 帳號的事）。
- 不改對話框的樣子。

## Tasks

- [x] Task 1 — `useEnrichmentRefresh`：沒開的串流關掉；新增 `track()` 輪詢（AC 1～3）
- [x] Task 2 — `scannerService.getEnrichStatus`；`LibraryBrowseV2` 排進比對後呼叫 `track()`（AC 2、4）
- [x] Task 3 — 兩個 SSE 端點加 `no-transform`＋handler 測試（AC 5）
- [x] Task 4 — 測試、lint、typecheck、vet、staticcheck（AC 6）

## Dev Notes

- 對話框流程：`LibraryBrowseV2.tsx` `executeBatchAction` 的 reparse 分支（先 `watch()` 再 POST，`disc-2026-10-reparse-dialog-misses-instant-pass`）。
- 狀態端點：`GET /api/v1/scanner/enrich/status` → `EnrichmentProgress{total, processed, succeeded, failed, skipped, current_title, is_active}`（`scanner_handler.go:224`）。POST 回來之前 `RequestRun` 已經拿到單飛旗標（或排了下一輪），所以 POST 之後第一次問到 `is_active=false` 就是做完了。
- 只靠 SSE 的其他地方（盤點）：`useScanProgress`、`useDownloadProgress`、`useGenerationProgress`、`useGenerationJobsFeed`、`useGenerationBatchProgress`、`useGenerationCandidatesProgress`、`useSubtitleBatchProgress`、`useSubtitleSearch`、`useRequestProgress`、`components/parse/useParseProgress`。

### Time-dependent visual coverage

- N/A — 對話框外觀沒變。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5），2026-10-27。

### Completion Notes List

- `useEnrichmentRefresh`：`watch()` 逾時或開之前出錯就 `close()`；新增 `track()`，每 2 秒問狀態，完成就停；`doneRef` 讓完成之後晚到的串流進度不會把畫面拉回「進行中」；串流的 `enrich_complete` 會停止輪詢。
- `LibraryBrowseV2`：`successCount > 0` 時呼叫 `track()`。
- SSE：`sse/handler.go`、`parse_progress_handler.go` 的 `Cache-Control` 加 `no-transform`；新增 `sse/handler_test.go`（真的 HTTP server，讀第一行）。
- 反向驗證：拿掉逾時的 `close()` 與輪詢的排程 → 新的 5 個測試紅。
- 試著用瀏覽器工具從外網量串流，被 Cloudflare Access 擋下（要信箱驗證碼），沒有登入，改用 log 的時間點推論（見上表）。
- 測試：`useEnrichmentRefresh.spec`、`LibraryBrowseV2.spec`、`routes/library.spec`、`BatchProgress` 共 99 個全過；`go test ./internal/sse/ -race`、`./internal/handlers/` 全過；vet、staticcheck 乾淨。
- **部署後要 Alexyu 驗**：從外網勾一部→重新解析，對話框幾秒內走到「完成」。若 `no-transform` 生效，串流本身就會通；若沒有，輪詢也會在 2～4 秒內收尾。

### File List

- `apps/web/src/hooks/useEnrichmentRefresh.ts`、`.spec.ts`
- `apps/web/src/services/scannerService.ts`
- `apps/web/src/components/library/LibraryBrowseV2.tsx`、`.spec.tsx`
- `apps/web/src/routes/library.spec.tsx`
- `apps/api/internal/sse/handler.go`、`handler_test.go`（新）
- `apps/api/internal/handlers/parse_progress_handler.go`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
