# Story ux3-4-2b — Downloads SSE live-progress broadcaster (backend)

**Epic:** ux3-downloads-v2 (UX Redesign Phase 3, Epic 4) · **Status:** ready-for-dev
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
2. **AC2 `[@contract-v1]` (event payload shape):** each broadcast carries `Data` = the current downloads snapshot in the **same item shape as `GET /api/v1/downloads`** (reuse the existing `DownloadItem` / `qbittorrent.Torrent` serialization — hash, name, progress, dlspeed/upspeed, eta, size, status, plus `parse_status` when present). **Key casing MUST match what the FE consumer reads** (mirror the `scanner_service` snake_case caveat — `broadcastScanComplete` uses snake_case to match `useScanProgress.ts`). This `download_progress` payload is the contract the FE story (ux3-4-3) acks — stamp `[@contract-v1]`.
3. **AC3 (single gated server poll):** a new background service polls `DownloadService.GetAllDownloads(ctx, "all", …)` on a ticker (default **~2s**, see decision #2 — honors Epic 14's <1s-ish freshness target while staying qBT-rate-friendly) and broadcasts AC2. **Before each poll it checks `sseHub.ClientCount()`; when 0 it skips the qBittorrent call entirely** (no clients → no poll → no broadcast).
4. **AC4 (clean lifecycle, no goroutine leak — Rule 14):** the service exposes `Start(ctx)` + `Stop()`, honors **both** `ctx.Done()` and `Stop()`, `defer ticker.Stop()`, and `Stop()` is idempotent — mirroring `CacheSweepScheduler` / `BackupScheduler`. A qBittorrent poll error is logged and the loop **continues** (never panics the goroutine; `*qbittorrent.ConnectionError` while qBT is unconfigured/unreachable is expected and logged at DEBUG/WARN, not spammed at ERROR).
5. **AC5 (wired into main + graceful shutdown):** constructed in `apps/api/cmd/api/main.go` alongside the other schedulers (it has `downloadService` `:163` and `sseHub` `:351` in scope), started in its own goroutine with a cancellable context, and on shutdown `cancel()` + `Stop()` are invoked — matching the backup/scan/cache-sweep wiring pattern.
6. **AC6 (testability — narrow interface, Rule 11):** the broadcaster depends on a **narrow interface** for the hub (e.g. `progressSink interface { Broadcast(sse.Event); ClientCount() int }`) that `*sse.Hub` satisfies, so tests inject a fake — no real hub/HTTP needed.
7. **AC7 (tests):** unit tests cover — (a) clients connected → a tick polls `GetAllDownloads` and `Broadcast`s one `download_progress` event with the snapshot; (b) **`ClientCount()==0` → NO poll, NO broadcast** (the gate); (c) a `GetAllDownloads` error is swallowed and the loop continues; (d) `Stop()` / `ctx` cancellation returns promptly and is idempotent — all with a mocked `DownloadService` + fake sink + short injected interval, **no real `time.Sleep` flakiness** (mirror `cache_sweep_scheduler_test.go`). `go vet` + `staticcheck` clean (Rule 12); no migration.

## Tasks / Subtasks

- [ ] **Task 1 — Event type (AC: #1)**
  - [ ] `apps/api/internal/sse/hub.go` `:16-23`: add `EventDownloadProgress EventType = "download_progress"`.
- [ ] **Task 2 — `DownloadProgressBroadcaster` service (AC: #2, #3, #4, #6)**
  - [ ] New file `apps/api/internal/services/download_progress_broadcaster.go`. Struct holds `downloadSvc DownloadServiceInterface`, `sink progressSink` (the narrow hub interface), `interval time.Duration`, `mu`/`stopCh`/`stopped` (copy `CacheSweepScheduler`'s lifecycle shape verbatim).
  - [ ] `Start(ctx)` → ticker loop; each tick: `if s.sink.ClientCount() == 0 { continue }`; else `items, err := s.downloadSvc.GetAllDownloads(ctx, "all", "added_on", "desc")`; on err log+continue (DEBUG/WARN for `*qbittorrent.ConnectionError`); else `s.sink.Broadcast(sse.Event{ID: uuid.New().String(), Type: sse.EventDownloadProgress, Data: <snapshot, AC2 shape>})`.
  - [ ] `Stop()` idempotent (`mu`+`stopped`+`close(stopCh)`); `defer ticker.Stop()`; honor `ctx.Done()` + `stopCh` (mirror `cache_sweep_scheduler.go` `run`/`Stop`).
  - [ ] **Payload casing (AC2):** reuse the same serialization as the `GET /downloads` list item so the FE reads one shape; verify the JSON key casing the FE expects (snake_case per the `scanner_service.broadcastScanComplete` precedent + Rule 18 boundary on the FE).
- [ ] **Task 3 — Wire into `cmd/api/main.go` (AC: #5)**
  - [ ] Construct after the cache-sweep scheduler (~`:409`): `downloadProgressBroadcaster := services.NewDownloadProgressBroadcaster(downloadService, sseHub)`; start in its own goroutine with a cancellable ctx (~`:642`, beside the other `go …Start(ctx)`); shutdown `cancel()` + `Stop()` (~`:682`).
- [ ] **Task 4 — Tests (AC: #7)**
  - [ ] New `apps/api/internal/services/download_progress_broadcaster_test.go`: mocked `DownloadServiceInterface` + fake `progressSink` (configurable `ClientCount`, captures `Broadcast` calls). Cases: clients→poll+broadcast; **no-clients→no poll/broadcast**; poll-error→continue; `Stop`/ctx-cancel idempotent + prompt. Short injected interval, channel-signalled (no real sleep).
  - [ ] `cd apps/api && go test ./internal/services/ -run DownloadProgressBroadcaster -v` (+ `-race`); `pnpm lint:all` (Rule 12).

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

_(to be filled by dev agent)_

### Debug Log References

### Completion Notes List

### Discovery Triage

- **Did this story discover any work outside its current scope?** **YES — one, pre-noted:**
  - **③ — `/sync/maindata` incremental polling** would reduce per-poll bytes vs the full-list poll, but needs a new qBT client method + RID state tracking. Deferred (YAGNI for single-user NAS; the ClientCount gate already bounds load). File a `backlog` entry only if poll bandwidth ever proves a problem.
- Reference: `project-context.md` Rule 24; origin: this story's design.

### File List

_(to be filled by dev agent)_

- `apps/api/internal/sse/hub.go` (MODIFIED — `EventDownloadProgress` const)
- `apps/api/internal/services/download_progress_broadcaster.go` (NEW — gated poll → broadcast service)
- `apps/api/internal/services/download_progress_broadcaster_test.go` (NEW)
- `apps/api/cmd/api/main.go` (MODIFIED — construct + start + shutdown wiring)

## Change Log

| Date       | Change                                                                                                                              |
| ---------- | --------------------------------------------------------------------------------------------------------------------------------- |
| 2026-06-30 | Story created (SM create-story) as the SSE sibling of ux3-4-2. Server-side gated poll → SSE fan-out (Epic 14 H-1 / P3-012); `ClientCount()` gate (already exists) removes idle qBT load. `[@contract-v1]` on the `download_progress` payload (FE ux3-4-3 acks). 4 tasks, backend-only, no split. Status → ready-for-dev. |
