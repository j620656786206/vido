# Story ux3-4-2b — Downloads SSE live-progress broadcaster (backend)

**Epic:** ux3-downloads-v2 (UX Redesign Phase 3, Epic 4) · **Status:** done
**Owner:** dev · **Type:** backend (cross-stack BE half) · **FRs:** PH3-M3 — **Epic 14 H-1 / P3-012**

## Story

As a NAS owner watching my downloads,
I want download progress / speed / ETA pushed to the page in real time over SSE instead of every browser polling the API,
so that **the Downloads v2 page updates live (<1s) and the "polling storms / console bursts" pain is gone — one server-side poll fans out to all clients, and qBittorrent is not polled at all when nobody is watching.**

## The mechanism (read FIRST) — move polling from N browsers to ONE gated server poll

Today the Downloads page **polls** `GET /api/v1/downloads` from every open tab (the "polling storms" pain, redesign brief Part 4 D1). This story replaces that with the **server-side poller → SSE fan-out** pattern (the standard Epic 14 H-1 design):

- A single background goroutine polls qBittorrent on a short ticker and **broadcasts** a `download_progress` event to the SSE hub; all connected clients receive it. N client-polls → 1 server-poll.
- **The poll is gated on `Hub.ClientCount() > 0`** — when nobody is on the Downloads page, the broadcaster does **not** touch qBittorrent at all (Rule 14; also why this *removes* polling load rather than relocating it).
- qBittorrent has no push/webhook for live progress, so a server-side poll is the only source — but doing it **once, gated** is the win.

## Out of scope (siblings / deferred)

- **Card actions (pause/resume/remove)** — the sibling `ux3-4-2-downloads-actions-be` (request/response actions; distinct concern). This story is **progress push only**.
- **NZBGet (H-2), notifications (H-3), unified dashboard (H-4)** — additive Epic 14 features, deferred.
- **`/sync/maindata` incremental (RID) polling** — an efficiency optimization over the full-list poll; **deferred** (YAGNI for a single-user NAS — see decision #3).

## Acceptance Criteria

1. **AC1 (new event type):** `sse.EventDownloadProgress = "download_progress"` is added to `apps/api/internal/sse/hub.go` (the `EventType` const block, `:16-23`). (A `download_complete` event is **out of scope** — the FE derives completion from the progress payload's per-item status; note in code.)
2. **AC2 `[@contract-v1]` (event payload shape):** each broadcast carries `Data` = the current downloads snapshot as a bare `[]qbittorrent.Torrent` array, **same PER-ITEM shape as each item in `GET /api/v1/downloads`** (hash, name, progress, `download_speed`/`upload_speed`, eta, size, status, added_on — the raw snake_case `Torrent` JSON). **Key casing MUST match what the FE consumer reads:** the wire is snake_case and the FE applies `snakeToCamel` on BOTH paths (`downloadService.fetchApi` on GET, `useScanProgress`-style `JSON.parse(e.data).data` on SSE), so raw `Torrent` JSON is byte-symmetric with GET per item. This `download_progress` payload is the contract the FE story (ux3-4-3) acks — stamp `[@contract-v1]`.
   - **⚠️ The push DELIBERATELY DIVERGES from `GET /downloads` in three ways the FE (ux3-4-3) MUST reconcile in `setQueryData` — MERGE, do not replace (CR H1):**
     1. **NO `parse_status`** — it is a handler-layer enrichment (`handlers.DownloadItem` = `Torrent` + `parse_status`, built from `parseQueueSvc`); the broadcaster lives in `services` and importing `handlers` = import cycle, so `parse_status` is ABSENT (`omitempty`). The FE must PRESERVE existing `parse_status` across the ~2s snapshot merge, or completed-download parse badges vanish every tick.
     2. **Bare array, NOT the paginated envelope** — GET returns `{items, page, pageSize, totalItems, totalPages}`; `download_progress.Data` is just the item array. The FE maps it into `.items`.
     3. **Full, UNPAGINATED list** — every torrent, not one page. The FE must not let a push blow away the current page window.
3. **AC3 (single gated server poll):** a new background service polls `DownloadService.GetAllDownloads(ctx, "all", …)` on a ticker (default **~2s**, see decision #2 — honors Epic 14's <1s-ish freshness target while staying qBT-rate-friendly) and broadcasts AC2. **Before each poll it checks `sseHub.ClientCount()`; when 0 it skips the qBittorrent call entirely** (no clients → no poll → no broadcast).
4. **AC4 (clean lifecycle, no goroutine leak — Rule 14):** the service exposes `Start(ctx)` + `Stop()`, honors **both** `ctx.Done()` and `Stop()`, `defer ticker.Stop()`, and `Stop()` is idempotent — mirroring `CacheSweepScheduler` / `BackupScheduler`. A qBittorrent poll error is logged and the loop **continues** (never panics the goroutine; `*qbittorrent.ConnectionError` while qBT is unconfigured/unreachable is expected and logged at DEBUG/WARN, not spammed at ERROR).
5. **AC5 (wired into main + graceful shutdown):** constructed in `apps/api/cmd/api/main.go` alongside the other schedulers (it has `downloadService` `:163` and `sseHub` `:351` in scope), started in its own goroutine with a cancellable context, and on shutdown `cancel()` + `Stop()` are invoked — matching the backup/scan/cache-sweep wiring pattern.
6. **AC6 (testability — narrow interface, Rule 11):** the broadcaster depends on a **narrow interface** for the hub (e.g. `progressSink interface { Broadcast(sse.Event); ClientCount() int }`) that `*sse.Hub` satisfies, so tests inject a fake — no real hub/HTTP needed.
7. **AC7 (tests):** unit tests cover — (a) clients connected → a tick polls `GetAllDownloads` and `Broadcast`s one `download_progress` event with the snapshot; (b) **`ClientCount()==0` → NO poll, NO broadcast** (the gate); (c) a `GetAllDownloads` error is swallowed and the loop continues; (d) `Stop()` / `ctx` cancellation returns promptly and is idempotent — all with a mocked `DownloadService` + fake sink + short injected interval, **no real `time.Sleep` flakiness** (mirror `cache_sweep_scheduler_test.go`). `go vet` + `staticcheck` clean (Rule 12); no migration.

## Tasks / Subtasks

- [x] **Task 1 — Event type (AC: #1)**
  - [x] `apps/api/internal/sse/hub.go` `:16-23`: add `EventDownloadProgress EventType = "download_progress"`. (Added with a doc comment noting NO `download_complete` event — FE derives completion from per-item `status` — and the `[@contract-v1]` payload pointer.)
- [x] **Task 2 — `DownloadProgressBroadcaster` service (AC: #2, #3, #4, #6)**
  - [x] New file `apps/api/internal/services/download_progress_broadcaster.go`. Struct holds `downloadSvc DownloadServiceInterface`, `sink progressSink` (the narrow hub interface), `interval time.Duration`, `mu`/`stopCh`/`stopped` (copies `CacheSweepScheduler`'s lifecycle shape).
  - [x] `Start(ctx)`→`run(ctx, interval)` ticker loop; each tick (`tick(ctx)`): `if b.sink.ClientCount() == 0 { return }`; else `torrents, err := b.downloadSvc.GetAllDownloads(ctx, "all", "added_on", "desc")`; on err log+continue (DEBUG for `*qbittorrent.ConnectionError` **and** context-cancel shutdown race, WARN for other errors — never broadcasts on error); else `b.sink.Broadcast(sse.Event{ID: uuid.New().String(), Type: sse.EventDownloadProgress, Data: <snapshot>})`. **No cold-start poll** (documented: gate skips it at boot anyway; FE seeds via its own initial GET).
  - [x] `Stop()` idempotent (`mu`+`stopped`+`close(stopCh)`); `defer ticker.Stop()`; honors `ctx.Done()` + `stopCh` (mirrors `cache_sweep_scheduler.go` `run`/`Stop`).
  - [x] **Payload casing (AC2):** broadcasts raw `[]qbittorrent.Torrent` — its JSON tags are already snake_case (`hash/name/progress/download_speed/upload_speed/eta/size/status/added_on`), the exact keys `GET /downloads` exposes per item (`handlers.DownloadItem` embeds `Torrent`). `parse_status` is a handler-only enrichment (importing `handlers` from `services` = import cycle) — left absent (`omitempty`); FE preserves it across merges. `nil`→`[]` normalized so the payload is never JSON `null`.
- [x] **Task 3 — Wire into `cmd/api/main.go` (AC: #5)**
  - [x] Constructed after the cache-sweep scheduler: `downloadProgressBroadcaster := services.NewDownloadProgressBroadcaster(downloadService, sseHub)`; started in its own goroutine with a cancellable `downloadProgressCtx` (beside the other `go …Start(ctx)`); shutdown `downloadProgressCancel()` + `Stop()` (beside the cache-sweep stop).
- [x] **Task 4 — Tests (AC: #7)**
  - [x] New `apps/api/internal/services/download_progress_broadcaster_test.go`: hand-rolled `mockDownloadSvc` (`DownloadServiceInterface`) + `fakeProgressSink` (configurable `ClientCount`, captures `Broadcast`, optional signal chan). Cases: clients→poll+broadcast (+event type/id/`[]Torrent` payload); **no-clients→no poll/broadcast**; poll-error→swallowed+no-broadcast; ConnectionError→quiet; nil→empty-array; ticker-driven broadcast (signal, not sleep); `Stop`/ctx-cancel prompt + idempotent; `Start` public entrypoint. `+ compile-time `var _ progressSink = (*sse.Hub)(nil)` proves AC6.
  - [x] `go test ./internal/services/ -run DownloadProgressBroadcaster -v -race` → PASS (all sub-tests); `go vet` + `staticcheck` clean; `pnpm lint:all` 0 errors + prettier clean (Rule 12).

## Dev Notes

### Verified anchors (2026-06-30)

- **SSE hub** `apps/api/internal/sse/hub.go`: `EventType` const block `:16-23` (add the new type), `Event{ID,Type,Data}` `:27`, `Broadcast(event)` `:131`, **`ClientCount() int` already exists `:148`** (the gate — no hub change needed beyond the const).
- **Producer precedent** `apps/api/internal/services/scanner_service.go:509-514` — `sse.Event{ID: uuid.New().String(), Type: EventScanProgress, Data: progress}`; `broadcastScanComplete` `:520+` documents the **snake_case-to-match-frontend** casing rule (AC2).
- **Lifecycle precedent** `apps/api/internal/services/cache_sweep_scheduler.go` — `Start(ctx)`/`run`/`sweep`/`Stop` (idempotent `mu`+`stopped`+`close(stopCh)`, `defer ticker.Stop()`, ctx+stopCh select). Copy this shape; swap `sweep` for `poll+broadcast`.
- **Data source** `apps/api/internal/services/download_service.go:86` `GetAllDownloads(ctx, filter, sort, order)` — reuse; same `getClient` config path.
- **Wiring** `apps/api/cmd/api/main.go`: `downloadService` `:163`, `sseHub := sse.NewHub()` `:351`, scheduler start sites ~`:642`, shutdown ~`:682`.

### Key design decisions

1. **Server-side poll → SSE fan-out (not qBT push).** qBittorrent exposes no live-progress webhook; the only source is a poll. The win is doing it **once on the server, gated on connected clients**, vs every browser polling — this is Epic 14 H-1 / P3-012's "SSE replacing polling."
2. **Cadence ~2s default (decision, tunable).** Epic 14 success criterion is "<1s from state change to UI"; a ~2s tick is a pragmatic qBT-friendly default for a single-user NAS (drop toward 1s if the felt latency is too slow). A settings key (`download_progress_interval_ms`) is **optional/future** — not added now (the ClientCount gate already removes idle load; YAGNI).
3. **Full-list poll, not `/sync/maindata` incremental.** v1 reuses `GetAllDownloads` (full snapshot) for simplicity and one shared payload shape with the GET endpoint. The qBT `/sync/maindata?rid=N` incremental API would cut per-poll bytes but needs new client work + RID state; **deferred** (Discovery Triage ③).
4. **Lazy-SSE is the FE's job (project-context §8).** The FE (ux3-4-3) must lazy-connect `EventSource` only when the Downloads page is active (never on mount — breaks Playwright `networkidle`). The BE gate (`ClientCount==0 → skip`) complements this: zero clients → zero qBT traffic.

### Rule compliance

- Rule 1 (backend `/apps/api`), Rule 4/11 (service + **narrow `progressSink` interface**, AC6), Rule 7 (no new error code — reuses qBT errors, logged-not-surfaced), Rule 9 (co-located test), Rule 12 (lint:all), **Rule 13** (poll error logged + loop continues, never swallowed-and-continue-as-success), **Rule 14** (ctx-honoring goroutine + idempotent Stop + the ClientCount gate = bounded work).
- **Cross-stack split check:** 4 tasks, all backend, 0 frontend → single story, **no split** (FE consumer = ux3-4-3).

### Time-dependent visual coverage

- **N/A — no `apps/web/src/components` touched.** Backend-only.

### References

- [Source: `ux3-4-1-downloads-design.md` — Decision #3 / Discovery Triage ③] — the design requiring real-time (not polling) progress.
- [Source: `ux3-4-2-downloads-actions-be.md`] — the sibling actions BE half (this is the progress half).
- [Source: `apps/api/internal/sse/hub.go`] — event types + `Broadcast` + `ClientCount`.
- [Source: `apps/api/internal/services/scanner_service.go`, `…/cache_sweep_scheduler.go`] — producer + lifecycle precedents.
- [Source: `epics/epic-14-download-management-v2.md` — H-1] · [Source: `project-context.md` §8 SSE Hub + frontend lazy-SSE pattern; Rule 1/4/11/13/14].

## Dev Agent Record

### Agent Model Used

claude-opus-4-8[1m] (Amelia, dev-story workflow) — 2026-07-03

### Debug Log References

- `go test ./internal/services/ -run DownloadProgressBroadcaster -v -race` → all PASS.
- Full backend suite `go test ./...` → EXIT=0, 33 packages, 0 failures (tracked `preexisting-fail-scanner-sse-scan-cancelled-flake` did not recur this run).
- `CI=true pnpm nx test web --watch=false` → 202 files / 2251 tests PASS, "No test processes found" (no orphaned workers).
- `go vet` + `staticcheck` (services/sse/cmd) clean; `pnpm lint:all` → 0 errors, 123 pre-existing FE warnings (0 in touched files), prettier clean.

### Completion Notes List

- **AC1** — `sse.EventDownloadProgress = "download_progress"` added to `hub.go` const block, with a doc comment that there is deliberately NO `download_complete` event (FE derives completion from each item's `status`).
- **AC2 `[@contract-v1]`** — broadcast `Data` is the raw `[]qbittorrent.Torrent` snapshot. Its JSON tags are already snake_case and identical to the per-item shape `GET /api/v1/downloads` returns (`handlers.DownloadItem` embeds `qbittorrent.Torrent`), so the FE (ux3-4-3) reads ONE shape. **Design note:** `parse_status` is a handler-layer enrichment requiring `parseQueueSvc`; `DownloadItem` lives in the `handlers` package, and importing `handlers` from `services` would be an import cycle — so the push carries the embedded `Torrent` only, `parse_status` absent (`omitempty`). The FE preserves `parse_status` across snapshot merges (its concern per ux3-4-3). `nil`→`[]` normalized so the payload is always a JSON array.
- **AC3 (gated poll)** — `tick()` checks `sink.ClientCount()` FIRST; `==0` returns before touching qBittorrent (no poll, no broadcast). Default cadence `defaultDownloadProgressInterval = 2s` (decision #2). No `download_progress_interval_ms` settings key (YAGNI — the gate already removes idle load).
- **AC4 (lifecycle, Rule 14)** — `Start`/`Stop` + `run(ctx, interval)` split (for test injection), `defer ticker.Stop()`, honors both `ctx.Done()` and `stopCh`, `Stop()` idempotent (`mu`+`stopped`+`close(stopCh)`). Poll errors logged + loop continues: `*qbittorrent.ConnectionError` (unconfigured/unreachable) and context-cancel shutdown races → DEBUG; other errors → WARN; never broadcasts on error; never panics the goroutine. NO cold-start poll (unlike CacheSweepScheduler — documented rationale: gate skips it at boot, FE seeds via its own initial GET).
- **AC5 (main wiring)** — constructed after the cache-sweep scheduler (`downloadService` + `sseHub` in scope), started `go downloadProgressBroadcaster.Start(downloadProgressCtx)`, shutdown `downloadProgressCancel()` + `Stop()` — mirrors the cache-sweep/scan/backup pattern.
- **AC6 (narrow interface, Rule 11)** — `progressSink interface { Broadcast(sse.Event); ClientCount() int }`; `*sse.Hub` satisfies it (compile-time `var _ progressSink = (*sse.Hub)(nil)`); tests inject `fakeProgressSink`, no real hub/HTTP.
- **AC7 (tests)** — deterministic: gate/poll/broadcast/error paths asserted via direct `tick()` calls (no sleep); ticker path via a signal channel; Stop/ctx via long-interval + close-and-wait (mirrors `cache_sweep_scheduler_test.go`). `-race` clean.
- **🔗 AC Drift: NONE** (checked: `'download_progress|EventDownloadProgress|contract-v'` across `_bmad-output/implementation-artifacts/*.md` — all `download_progress` hits are THIS story or the downstream FE consumer ux3-4-3 which *acks* (not drift); the sibling ux3-4-2's `[@contract-v1]` is a distinct pause/resume/remove contract; `hub.go`/`main.go` edits are additive — no prior external contract altered).
- **📎 Contract Stamps: FOUND** (1 stamped AC in this story — AC2 `download_progress` payload `[@contract-v1]`, freshly-minted v1, NO bump → no Change-Log bump row required per Rule 20; downstream ux3-4-3 acks via `confirmed against [@contract-v1]`, that story not yet implemented).
- **🎭 A11y Pre-Flight: N/A (100% backend — no apps/web/ files touched).**
- **Pre-existing failures:** none introduced; the tracked scanner-SSE flake did not recur (no new tracking entry needed).
- **Discovery Triage ③** — `/sync/maindata` incremental polling remains deferred (YAGNI; the ClientCount gate already bounds load). No new out-of-scope work discovered during implementation.

### Discovery Triage

- **Did this story discover any work outside its current scope?** **YES — one, pre-noted:**
  - **③ — `/sync/maindata` incremental polling** would reduce per-poll bytes vs the full-list poll, but needs a new qBT client method + RID state tracking. Deferred (YAGNI for single-user NAS; the ClientCount gate already bounds load). File a `backlog` entry only if poll bandwidth ever proves a problem.
- Reference: `project-context.md` Rule 24; origin: this story's design.

### File List

- `apps/api/internal/sse/hub.go` (MODIFIED — `EventDownloadProgress = "download_progress"` const + doc comment)
- `apps/api/internal/services/download_progress_broadcaster.go` (NEW — gated poll → broadcast service + narrow `progressSink` iface)
- `apps/api/internal/services/download_progress_broadcaster_test.go` (NEW — `mockDownloadSvc` + `fakeProgressSink`, AC7 cases)
- `apps/api/cmd/api/main.go` (MODIFIED — construct + start goroutine + graceful-shutdown wiring)
- `_bmad-output/implementation-artifacts/sprint-status.yaml` (MODIFIED — story status ready-for-dev → in-progress → review)

## Senior Developer Review (AI)

**Reviewer:** Amelia (adversarial code-review workflow) · **Date:** 2026-07-03 · **Outcome:** Approve (all High + Medium fixed)

**Mandatory checks:** 🔒 Rule 7 Wire Format **PASS** (changed Go files define no error-code consts; existing `QBITTORRENT_*` reused) · 🔒 Rule 20 Contract Bump **N/A** (no bump — v1 fresh origin) · 🔒 Rule 25 Mega-line **N/A** (`project-context.md` untouched) · Git ↔ File List: **0 discrepancies**.

**Findings & resolutions (1 High, 1 Medium, 2 Low — all fixed):**

- ✅ **[High] H1 — `[@contract-v1]` under-specified the GET↔SSE shape deltas.** The stamped `download_progress` payload diverges from `GET /downloads` in three un-documented ways (no `parse_status`; bare array vs paginated `{items,…}` envelope; full unpaginated list vs one page). ux3-4-3 acks this stamp and calls `setQueryData` — an ambiguous stamp would wipe parse badges / mis-map the envelope / break pagination. **Fix:** spelled out all three deltas + the merge-not-replace obligation explicitly in AC2 AND the broadcaster's payload comment. No wire change (contract still v1 — clarified, not bumped).
- ✅ **[Med] M1 — Tests didn't lock AC3 poll args or the ~2s default.** `mockDownloadSvc` ignored its args, so a filter/sort regression was invisible. **Fix:** mock now captures `(filter, sort, order)`; AC7a asserts `"all"/"added_on"/"desc"`; new `TestDownloadProgressBroadcaster_DefaultInterval` asserts `interval == 2s`.
- ✅ **[Low] L1 — WARN spam on persistent non-connection poll errors (~30/min at 2s cadence).** **Fix:** added `lastPollErr` dedup — first occurrence WARNs, identical repeats DEBUG, a successful poll resets. New `TestDownloadProgressBroadcaster_WarnThrottle` locks the state machine.
- ✅ **[Low] L2 — Ticker test didn't join its goroutine.** **Fix:** `TestDownloadProgressBroadcaster_Run` now joins via a `done` channel after `Stop()`.

**Post-fix validation:** broadcaster tests `-race` green (incl. 3 new cases); full Go suite 0-fail; `go vet` + `staticcheck` + `pnpm lint:all` (0 errors) + prettier clean. Web vitest not re-run (0 apps/web/ files touched in the fix round; passed earlier this session).

## Change Log

| Date       | Change                                                                                                                              |
| ---------- | --------------------------------------------------------------------------------------------------------------------------------- |
| 2026-06-30 | Story created (SM create-story) as the SSE sibling of ux3-4-2. Server-side gated poll → SSE fan-out (Epic 14 H-1 / P3-012); `ClientCount()` gate (already exists) removes idle qBT load. `[@contract-v1]` on the `download_progress` payload (FE ux3-4-3 acks). 4 tasks, backend-only, no split. Status → ready-for-dev. |
| 2026-07-03 | Implemented all 4 tasks (dev-story, Amelia). AC1 `EventDownloadProgress` const. AC2 `[@contract-v1]` broadcast `Data = []qbittorrent.Torrent` (snake_case, same per-item shape as GET /downloads; `parse_status` left to the handler layer — import-cycle-avoided). AC3 gated ~2s poll (`ClientCount()==0`→skip). AC4 CacheSweepScheduler-shape lifecycle (ctx+stopCh, idempotent Stop, `defer ticker.Stop()`, no cold-start poll, poll-error DEBUG/WARN + loop continues). AC5 main.go construct+start+shutdown. AC6 narrow `progressSink` iface (`*sse.Hub` satisfies; compile-time asserted). AC7 `-race`-clean deterministic tests (no real sleep). Full Go suite 0-fail; web suite 2251-pass; go vet + staticcheck + lint:all clean; prettier clean. AC Drift NONE, Contract Stamps FOUND (v1 origin, no bump), A11y N/A. Status → review. |
| 2026-07-03 | Adversarial code-review (Amelia) — 1H/1M/2L, ALL fixed. H1: AC2 + payload comment now spell out the 3 GET↔SSE deltas (no parse_status / bare array vs envelope / unpaginated) + FE merge-not-replace obligation (contract clarified, still v1). M1: tests lock AC3 poll args (`all/added_on/desc`) + 2s default interval. L1: `lastPollErr` WARN-throttle for persistent non-connection errors. L2: ticker test joins its goroutine. Post-fix: `-race` green, full Go 0-fail, vet/staticcheck/lint:all/prettier clean. Rule 7 PASS, Rule 20/25 N/A, Git↔File-List clean. Status → done. Ready for /ship → unblocks FE ux3-4-3 GATE B (with ux3-4-2). |
