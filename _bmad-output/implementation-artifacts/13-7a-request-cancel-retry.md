# Story 13-7a: Request cancel + retry — backend (DELETE + retry endpoints, *arr queue-remove capability)

Status: done

> **Split note (SM Bob, 2026-07-05):** 13-7 counted 5 backend + 5 frontend tasks → MANDATORY a/b split (Epic 8 Retro Agreement 5). This is the **backend** half; `13-7b-request-cancel-retry` (FE) depends on this story's endpoints.
>
> **Scope ruling — design over seed:** the sprint seed said "重試/取消 on failed rows", but the authoritative `.pen` (RequestRow-v2 `LkjRd`, L1 list `K7fiy`) draws **取消 ONLY on pending rows** and **重試 ONLY on failed rows**; searching/downloading/completed rows have the action-area `enabled:false`. This story implements exactly that matrix. Consequence: cancel never touches *arr (a pending row has NO `external_id` — fulfilment success transitions to searching), so **`DVRPlugin` [@contract-v1] is NOT bumped**.

## Story

As a Vido user who filed a media request,
I want to cancel a pending request and retry a failed one from the backend API,
so that the 想要清單 rows drawn in the design (取消 on pending, 重試 on failed) can actually act, and a dead request never blocks me from re-requesting.

## Acceptance Criteria

1. **`DELETE /api/v1/requests/{id}` [@contract-v1] — cancel a PENDING request.** Atomic conditional hard-delete (`DELETE FROM requests WHERE id = ? AND status = 'pending'`). Responses: **204** No Content on success; **404** `DB_NOT_FOUND` when the id does not exist; **409** `REQUEST_NOT_CANCELLABLE` (zh-TW message, e.g. `此請求已在處理中，無法取消`) when the row exists but status ≠ pending (distinguish via `FindByID` after a 0-rows-affected delete). Hard-delete (NOT a status write): `cancelled` is not in the migration-027 5-status CHECK enum, and deletion naturally frees the `idx_requests_active_unique` partial index so the same `(tmdb_id, media_type)` is immediately re-requestable.
2. **`POST /api/v1/requests/{id}/retry` [@contract-v1] — retry a FAILED request.** Only `failed` rows are retryable: **404** `DB_NOT_FOUND` unknown id; **409** `REQUEST_NOT_RETRYABLE` (zh-TW message) when status ≠ failed; **200** `{"success":true,"data":<updated request resource>}` on success (resource shape = 13-1a [@contract-v1], unchanged). Two failure flavors, split on `external_id`:
   - **(a) terminal-failed, `external_id` NULL** (e.g. `DVR_TVDB_NOT_FOUND`): reset row → `status='pending'`, clear `error_message` (source/external stay NULL), then best-effort synchronous `FulfilRequest` exactly like create (graceful degradation per 13-4a AC #6 — every failure path stays pending with zh-TW `error_message`; the 200 carries whatever state fulfilment produced).
   - **(b) queue-failed, `external_id` present** (poller wrote `下載發生錯誤…`, media already in *arr): locate the errored queue item via `GetQueue()` (join on `QueueItem.ExternalID`); if found, remove it via the NEW `QueueRemover` capability (AC 3) with blocklist + re-search semantics so *arr grabs a different release; then reset row → `status='searching'`, clear `error_message` (KEEP `external_id`/`fulfilment_source` — the media stays managed by *arr; the 15s poller re-derives from queue evidence). If the queue item is already gone, just reset → searching (poller Rule 4 holds searching while *arr hunts). Queue-remove failure is best-effort: slog WARN + still reset → searching (never 500 the retry because *arr hiccuped).
3. **`plugins.QueueRemover` optional capability — DVRPlugin untouched.** New optional interface in `apps/api/internal/plugins/plugin.go` mirroring the `ProfileLister` pattern (type-assert at call site; `DVRPlugin` [@contract-v1] stays byte-for-byte identical — NO Rule 20 bump): `RemoveQueueItem(ctx context.Context, queueID int64, opts …) error` wrapping *arr `DELETE /api/v3/queue/{id}` with the blocklist + re-search flags. ⚠️ **Verify the exact v3 query-param names against the live Radarr/Sonarr API docs before coding** (Radarr v3+: `removeFromClient`/`blocklist`/`skipRedownload` — do NOT trust memory; the retry semantic wants blocklist=true, redownload NOT skipped). Implement in BOTH `radarr` and `sonarr` clients (Rule 27 pillars ride the existing clients: `limiter.Wait(ctx)` first line, existing 10s http.Client, typed `PluginError` → `DVR_*` codes).
4. **Service + repository extensions (Rule 11 interfaces).** `RequestServiceInterface` += `CancelRequest(ctx, id) error`, `RetryRequest(ctx, id) (*models.Request, error)`. `RequestRepositoryInterface` += `FindByID(ctx, id) (*models.Request, error)`, `DeleteIfPending(ctx, id) (int64, error)` (rows-affected), `ResetForRetry(ctx, id, status, clearExternal bool) (*models.Request, error)` (clears `error_message`, optionally external/source, bumps `updated_at`, returns updated row). Zero-rows paths map to `ErrRequestNotFound`. Handler → Service → Repo layering (Rule 4); handler owns only `:id` presence validation.
5. **Poller-race honesty.** The 15s `RequestStatusPoller` Rule-5 branch re-fulfils pending rows concurrently. The conditional delete makes the DB outcome atomic; in the residual race window the poller's in-flight `FulfilRequest` can complete `AddMovie` against *arr AFTER the delete — `UpdateFulfilment` then hits 0 rows → `ErrRequestNotFound`, logged (13-4a behavior), and the movie stays added in *arr with NO request row (an orphaned *arr entry). Consequence: a later re-request of that title hits *arr's "already exists" 400 → `DVR_ADD_FAILED` → `stayPending` on every tick (see `disc-2026-07-arr-already-exists-loop`, filed by this story — pre-existing class, NOT fixed here). Document BOTH the log outcome and the orphan consequence in the code comment + cover with a service test simulating delete-after-FindActive. Retry's status writes use the same `UpdateStatus`/reset writers the poller uses; no new locking.
6. **Rule 7 error codes.** New codes under the EXISTING `REQUEST_` prefix (code-list update only — no new prefix, no CR-workflow sync): `REQUEST_NOT_CANCELLABLE`, `REQUEST_NOT_RETRYABLE` (both 409, zh-TW message + suggestion). 404 reuses `DB_NOT_FOUND` via `NotFoundError` (glossary/movie precedent). Swagger `@Failure` annotations updated on both new handlers; run `swag init` if annotations changed (Rule 15).
7. **Tests (co-located, real-sqlite for repo per 13-1a precedent).** Repo: conditional-delete matrix (pending deletes / non-pending 0 rows / unknown 0 rows), `ResetForRetry` field matrix. Service: cancel state matrix, retry flavors (a)/(b)/(queue-item-gone)/(queue-remove-fails best-effort), race test (AC 5). Handler httptest: 204 / 404 / 409×2 / 200-with-resource. Plugins: `RemoveQueueItem` httptest for radarr + sonarr (URL/method/params asserted). Full Go suite + `staticcheck` + `pnpm lint:all` green.
8. **Contract acks + stamps.** This story ACKS: confirmed against [@contract-v1] (Story 13-1a AC #2/#3 — request resource shape unchanged); confirmed against [@contract-v1] (Story 13-3a AC #2/#4 — status derivation + `request_progress` payload untouched; ⚠️ a hard-deleted row drops out of the snapshot, and 13-3b's planned merge PRESERVES absent rows as stale-terminal — see the cross-tab-phantom note in Dev Notes + the 13-3b stale-mark filed by this story); confirmed against [@contract-v1] (Story 13-4a AC #1 — `DVRPlugin` interface untouched via optional-capability pattern; AC #6 — fulfilment semantics reused verbatim for retry flavor (a)). This story STAMPS [@contract-v1] on AC 1 + AC 2 (endpoint shapes) — consumer: 13-7b.

## Tasks / Subtasks

- [x] Task 1: Repository extensions (AC: 1, 2, 4)
  - [ ] `FindByID`, `DeleteIfPending`, `ResetForRetry` in `request_repository.go` + interface; sentinels reused
  - [ ] Real-sqlite tests (matrices per AC 7)
- [x] Task 2: `QueueRemover` capability (AC: 3)
  - [ ] Verify Radarr/Sonarr v3 `DELETE /queue/{id}` param names (web/API docs) — record findings in Dev Agent Record
  - [ ] Optional interface in `plugins/plugin.go` (ProfileLister pattern) + radarr + sonarr impls + httptest
- [x] Task 3: Service methods (AC: 2, 4, 5)
  - [ ] `CancelRequest` (conditional delete → 0-rows → FindByID → not-found vs not-cancellable)
  - [ ] `RetryRequest` (flavor split on `external_id`; queue lookup + best-effort remove; reset writers; re-fulfil for flavor (a))
  - [ ] Service tests incl. race + degradation paths
- [x] Task 4: Handler + routes (AC: 1, 2, 6)
  - [ ] `DELETE /:id` + `POST /:id/retry` in `request_handler.go` `RegisterRoutes`; new 409 codes; Swagger annotations (+ `swag init` if needed)
  - [ ] httptest coverage per AC 7
- [x] Task 5: Gates (AC: 7)
  - [ ] Full `go test ./...`, `go vet`, staticcheck, `pnpm lint:all`, prettier on touched md/yaml

## Dev Notes

> ⚠️ **STALE [@contract-v1→v2]**: upstream 13-4b AC #2 bumped 2026-08-19 by 13-2a — `AddSeries` is now selection-aware (`AddOptions.Seasons/Episodes` additive; empty selection = v1 行為 byte-identical). Re-confirm against v2 before dev — 預期結論：cancel 僅作用於 pending（external_id NULL、無 *arr un-add），selection 分支不影響本 story 的取消/重試語意，但需在實作時明文 ack v2。See 13-4b Change Log.

### Code anchors (all MERGED on main — verified 2026-07-05)

- Routes: `apps/api/internal/handlers/request_handler.go:34-40` (`GET ""` + `POST ""` only today; error consts :17-20 `REQUEST_DUPLICATE`/`REQUEST_ALREADY_IN_LIBRARY`).
- Model: `apps/api/internal/models/request.go:36-50` — [@contract-v1] resource, JSON tags `id, tmdb_id, media_type, title, status, fulfilment_source, external_id, seasons, episodes, error_message, requested_at, updated_at`; wire `media_type` ∈ `movie|tv`.
- Migration 027: 5-status CHECK (`pending|searching|downloading|completed|failed`), partial unique `idx_requests_active_unique ON (tmdb_id, media_type) WHERE status IN ('pending','searching','downloading')` — comment says "a failed request must be retryable".
- Repo: `apps/api/internal/repository/request_repository.go:25-45` (no FindByID/Delete today); `ErrRequestNotFound` :16.
- Service: `apps/api/internal/services/request_service.go:26-29` interface; optional fulfilment dep via `SetFulfilmentService` :120-122.
- Fulfilment: `apps/api/internal/services/fulfilment_service.go` — `FulfilRequest` :65-76, success → `UpdateFulfilment(searching, arr, externalID)` :138-141, `failTerminally` :165-187, `stayPending` :192-213. Retry flavor (a) reuses `FulfilRequest` as-is.
- Poller: `apps/api/internal/services/request_status_poller.go` — 15s; Rule-5 re-fulfil of external-less ACTIVE pending :285-287 (⚠️ failed rows are NOT active → user-triggered retry is the only path back, hence this story); `reconcileExternal` :291-338 (not-in-queue + not-downloading → searching :334-336 — this is why retry flavor (b) resets to searching); queue-failed writer :370 (`下載發生錯誤，請重試或檢查下載器`, external_id KEPT).
- Plugins: `apps/api/internal/plugins/plugin.go:39-45` `DVRPlugin` [@contract-v1] (Name/TestConnection/AddMovie/AddSeries/GetQueue — NO remove); `ProfileLister` :64-67 = the optional-capability precedent to mirror; radarr/sonarr clients currently GET/POST only. `plugins.Manager` resolves configured clients (13-4a); fulfilment shows the client-acquisition pattern to copy.
- DELETE handler templates: `movie_handler.go:223-239` (pure 204) — cancel follows this; `download_handler.go:355-383` (`RemoveDownload`, external side-effect + query param) — retry/queue-remove follows this shape; `retry_handler.go:162` mounts a `DELETE /:id` precedent.
- Response helpers: `handlers/response.go` — `NoContentResponse` :50-53, `NotFoundError` :73-77 (`DB_NOT_FOUND`), `ErrorResponse`, `SuccessResponse`.

### Semantics decisions (already ruled — do not re-litigate)

- **Cancel = pending-only hard DELETE.** No `cancelled` enum value, no migration, no *arr un-add. Pending ⇒ `external_id` NULL (the only writer of external_id also writes status=searching). Searching/downloading cancel is a DESIGN decision (action-area disabled) — see `disc-2026-07-cancel-active-requests` backlog entry; do NOT absorb it here.
- **Retry keeps the 5-status enum intact** — it only moves failed → pending (flavor a) or failed → searching (flavor b) using existing writers. No SSE changes: the poller snapshot self-corrects next tick.
- **Cancel × 13-3b SSE-merge phantom (known, accepted):** 13-3b AC #2 specifies "cached rows ABSENT from the snapshot are stale-terminal → keep as-is" — that rule predates hard-delete cancel, so a row cancelled in tab A would be PRESERVED as a phantom pending row in tab B's cache until refetch (bounded by the 30s staleTime; the acting tab is corrected by 13-7b's optimistic remove + invalidate). This story STALE-MARKED 13-3b (Dev Notes + sprint-status, 2026-07-05): its merge must preserve absent rows ONLY when the cached status is terminal (completed/failed) and DROP absent ACTIVE rows (deleted-while-active). No change in this story.
- **Best-effort external calls** (Rule 13 + fulfilment precedent): *arr failures during retry/cancel NEVER 500 the user action; log via slog with request_id, annotate `error_message` where meaningful.

### Project Structure Notes

- All changes under `apps/api/internal/{handlers,services,repository,plugins}` + tests. No FE, no migrations, no main.go wiring expected (handler group already registered — Rule 15: verify `requestHandler.RegisterRoutes(apiV1)` covers the new routes; grep before assuming).
- No `.pen` change, no screenshots, no visual baselines.

### Time-dependent visual coverage

- N/A — backend-only story; no `apps/web/src/components/**` files touched.

### References

- [Source: sprint-status.yaml `13-7-request-cancel-retry` seed (filed by 13-1b Discovery Triage, Rule 24 lane ③)]
- [Source: _bmad-output/implementation-artifacts/13-1a-one-click-request.md AC #2/#3 [@contract-v1]]
- [Source: _bmad-output/implementation-artifacts/13-3a-request-status-tracking.md — poller/reconcile + request_progress [@contract-v1]]
- [Source: _bmad-output/implementation-artifacts/13-4a-arr-dvr-plugin.md AC #1/#6 [@contract-v1]; 13-4b AC #2]
- [Source: ux-design.pen — RequestRow-v2 `LkjRd` action matrix; L1 `K7fiy` rows `daLMM`(pending+取消) / `XfjmU`→`iyYqV`(failed+重試)]
- [Source: project-context.md Rules 4/7/11/13/14/15/19/20/24/27]

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（2026-10-02）

### Debug Log References

### Completion Notes List

- **Contract ack [@contract-v2] (13-4b AC #2, selection-aware `AddSeries`)**：取消只作用在 pending（`external_id` NULL、不碰 *arr）；重試 (a) 重跑 `FulfilRequest`，它本來就把列上存的季／集選取帶進 `AddOptions`——v2 對本單語意零影響。其餘 ack 同 AC #8。
- **AC #3 偏離（有理由，CR 後定案）**：能力介面是 `RemoveQueueItems(ctx, externalID, downloadIDs []string) (removed int, err error)`——呼叫端指名「哪幾個下載（種子 hash）」要刪，client 只負責找到對應的佇列紀錄 `id` 去 DELETE。原因：`plugins.QueueItem`（[@contract-v1] `GetQueue` 的回傳型別）沒有佇列紀錄自己的 `id`；而「哪一筆算壞掉」**必須和 poller 判 failed 的規則一模一樣**，所以抽出 `deriveQueueState(item, torrents)`（poller 的 `reconcileQueued` 與重試共用）：*arr 自己說 `failed`，或 qBittorrent 種子是 error。`services.DVRQueueCleaner` 用 poller 同一組來源（plugin manager＋`downloadService`）算出要刪的 hash。
- **Task 2 參數查證**：2026-10-02 抓 Radarr `develop` 與 Sonarr `v5-develop` 的 v3 OpenAPI，`DELETE /api/v3/queue/{id}` 的 query 參數為 `removeFromClient`（預設 true）、`blocklist`（預設 false）、`skipRedownload`（預設 false）、`changeCategory`。重試送 `removeFromClient=true&blocklist=true&skipRedownload=false`。
- **重試 (b) 的順序**：先查重複、再清 *arr 佇列、最後把列改回 searching。先查重複是為了不要在回 409 之前就已經叫 *arr blocklist／重搜；清佇列在 reset 之前，是因為反過來的話 poller 會看到還是壞的那筆、又把列翻回 failed。服務測試在清佇列的當下斷言列仍是 failed。
- **AC #2(b) 改成誠實失敗（CR）**：清佇列失敗時**不再**「照樣 reset」——那樣 15 秒後 poller 會把列翻回 failed，使用者完全不知道重試沒生效。現在列維持 failed，回 **502 `REQUEST_RETRY_CLEANUP_FAILED`**（「沒能清掉壞掉的下載，這次重試沒有生效」）。新碼在既有 `REQUEST_` 前綴下，Rule 7 碼表已更新。佇列裡已經沒有壞掉的那筆（被 *arr 自己清掉）不算失敗，照常 reset。
- **Rule 24 範圍內補洞**：重試把 failed 改回 active 狀態，可能撞上「使用者在失敗後又按了一次想要」產生的新 active 列（partial unique index）→ repo 映射成 `ErrRequestDuplicate` → 409 `REQUEST_DUPLICATE`（沿用既有碼與文案）。`ResetForRetry` 以 `status='failed'` 為條件，兩次重試或與 poller 競速只有一個會贏，輸的回 409 `REQUEST_NOT_RETRYABLE`。
- `ResetForRetry` 簽名比故事寫的少了 `clearExternal`：兩種 flavor 都不需要清（(a) 本來就是 NULL、(b) 要保留）。
- 404 沿用 `DB_NOT_FOUND`，但訊息改成中文（`NotFoundError` 只有英文）。
- Swagger：只補註解、不跑 `swag init`（`disc-2026-09-swagger-docs-stale` 的既定做法）。
- 主程式：`requestService.SetQueueCleaner(services.NewDVRQueueCleaner(pluginManager, downloadService))`。
- **Adversarial CR（2026-10-02，獨立 agent，含 router 實建 probe）**：
  - ✅ 修：初版用 *arr 的 `warning` 判壞——但 Radarr／Sonarr 的 qBittorrent client 把 `stalledDL`、`metaDL`、`missingFiles`、`error` 全映射成 warning，影集重試會連「只是暫時沒種子」的健康集數一起刪檔＋blocklist。改成上面「與 poller 同一條規則、指名 hash」。
  - ✅ 修：初版在 poller 因 qBT 判 failed、但 *arr 狀態還沒跟上（*arr 約每分鐘才刷新）時，重試什麼都沒刪就 reset，下一輪又翻回 failed。指名 hash 後這個時間差消失。
  - ✅ 修：清佇列失敗改回 502、列維持 failed（見上）；重複檢查移到清佇列之前。
  - 📝 不修（既有）：重試 (a) 與 poller 同時 `FulfilRequest` 的雙重新增競速，`stayPending` 用舊快照覆寫可能抹掉 `external_id`——create 路徑早就有同一個競速，且 (a) 今天只在 Sonarr「TVDB 查無」會發生，歸 `disc-2026-07-arr-already-exists-loop`。
  - 📝 不修：Sonarr 3.0.6／Radarr 4.0 以前的參數叫 `blacklist`；舊版只會刪不會 blocklist。
  - ✅ 驗證無誤：取消的條件式刪除不會被任何後續 UPDATE 復活；兩次重試只有一次贏；Gin 三條路由（GET coverage／DELETE :id／POST :id/retry）實建無衝突；Radarr `QueueController` 的 DELETE 是同步的、下一輪 poller 看到「不在佇列」會維持 searching。
- staticcheck：本機舊的 `~/go/bin/staticcheck`（go1.25 編）分析不完整，改用 `~/go/bin/staticcheck-2026.1 ./...`，零輸出。

### Discovery Triage

Authoring-time discoveries (SM Bob, 2026-07-05, filed in sprint-status.yaml):

- **③ `disc-2026-07-cancel-active-requests`** — the design deliberately disables the action-area on searching/downloading rows, so cancelling an ACTIVE (already-in-*arr) request is unsupported: it would need *arr un-add (`DELETE /movie/{id}`/`DELETE /series/{id}`) + a design addition. A stuck-searching row blocks re-request via the partial unique index — real but out-of-scope UX hole. Bidirectional: entry names this story.
- **③ `disc-2026-07-arr-already-exists-loop`** — requesting a title that already exists in *arr (orphaned by the AC-5 cancel race, or added manually in *arr) loops forever: `AddMovie` 400 already-exists → `DVR_ADD_FAILED` → `stayPending` re-attempted every 15s with no user-visible exit (retry only works on failed rows). Pre-existing resilience gap surfaced by the cancel-race analysis; candidate fix = map already-exists to adopt-existing-entry. Bidirectional: entry names this story.
- **Stale-mark on 13-3b (ready-for-dev draft, Rule-20-style):** 13-3b AC #2's preserve-absent-rows merge rule predates hard-delete cancel → phantom-row hazard. Annotated 2026-07-05 in 13-3b's Dev Notes + sprint-status entry: preserve absent rows only when cached status is terminal; drop absent ACTIVE rows.
- Seed-vs-design discrepancy ("取消 on failed rows" per seed; design draws 重試 only) — resolved IN-SCOPE by following the `.pen` (recorded in the split note; no tracked entry needed — narrowing, not new work).

(Dev: add further in-flight discoveries per Rule 24 before marking done.)

### File List

- apps/api/internal/plugins/plugin.go（`QueueRemover`）
- apps/api/internal/plugins/radarr/client.go、client_test.go
- apps/api/internal/plugins/sonarr/client.go、client_test.go
- apps/api/internal/repository/request_repository.go、request_repository_test.go
- apps/api/internal/services/request_service.go
- apps/api/internal/services/request_status_poller.go（抽出 `deriveQueueState`）
- apps/api/internal/services/request_cancel_retry.go（新）
- apps/api/internal/services/request_cancel_retry_test.go（新）
- apps/api/internal/services/request_service_test.go、fulfilment_service_test.go、request_status_poller_test.go（fake 補方法）
- apps/api/internal/handlers/request_handler.go、request_handler_test.go
- apps/api/cmd/api/main.go
- project-context.md（Rule 7 碼表）
