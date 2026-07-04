# Story 13.3a: Request Status Tracking — Backend Reconciler + request_progress SSE

Status: ready-for-dev

**Epic:** Epic 13 — Request System · **FR:** P3-003 (G-3) · **Artery #3 (BE half)** · **BACKEND-ONLY**
**Depends on: 13-1a merged** (requests table/repo/service) **+ 13-4a merged** (plugins.Manager + DVRPlugin.GetQueue + fulfilment service). 13-4b (Sonarr) is a SOFT dep — series requests simply stay `pending` (graceful) until it lands; the reconciler needs no change when it does.
**Split:** 13-3 is cross-stack → 13-3a (BE, this) / 13-3b (FE). Success criterion: status reflects real *arr/qBT state with **<30s latency**.

## Story

As a user with requests in flight,
I want each request's status to track reality (搜尋中 → 下載中 x% → 已入庫 / 失敗) automatically on the server,
so that the 想要清單 tells the truth without me refreshing or guessing.

## Acceptance Criteria

1. **Reconciler service (`RequestStatusPoller`) — always-on, 15s tick.** A new `services/request_status_poller.go` copies the `DownloadProgressBroadcaster` anatomy (narrow sink iface + ticker + Start/Stop lifecycle, `download_progress_broadcaster.go:50-181`) with ONE deliberate deviation, recorded here: **the `ClientCount()==0` gate moves from the POLL to the BROADCAST**. ux3-4-2b skips polling when nobody watches because its only job is SSE; this reconciler has SSE-independent server duties (DB status truth, scan triggering, pending retry, the 13-5 completed seam) so it always reconciles. Cheap-idle gate instead: **if `ListActive()` returns 0 rows (no pending/searching/downloading requests), the tick returns before ANY external call** — an idle NAS does zero *arr/qBT traffic. Interval 15s (comfortably beats the <30s criterion; all sources are LAN/local).

2. **[@contract-v1] Status derivation (single source of truth for the 5-enum).** Each tick, for every ACTIVE request row, derive in this priority order:

   | # | Condition (evaluated in order) | → status | Notes |
   |---|---|---|---|
   | 1 | `tmdb_id` ∈ `AvailabilityService.CheckOwned` (bulk, one call per tick — reuse Story 10-4 service, `availability_service.go:43-88`) | `completed` | Terminal. Vido's OWN library is the truth for 已入庫 — deliberately NOT *arr's `hasFile` (no DVRPlugin interface change needed). Fires the 13-5 seam (AC #6). |
   | 2 | `external_id` set AND a queue record matches it in `Manager` plugins' `GetQueue` (join `QueueItem.ExternalID`; refine via `QueueItem.DownloadID` → qBT torrent map when available) | `downloading` + ephemeral `progress` = (size−sizeleft)/size, or the joined qBT `Torrent.Progress` when the hash matches | Queue item in an errored state (*arr `trackedDownloadStatus`-equivalent surfaced by the client, or joined qBT status `error`) → `failed` + `error_message`. |
   | 3 | Was `downloading` last tick, queue record GONE, not yet owned | **hold `downloading`** (import window = the N1 整理中 phase) + trigger AC #3 scan | NEVER regress to `searching` — no 6th enum value invented (capability-honor). |
   | 4 | `external_id` set, no queue record, never reached downloading | `searching` | *arr is still hunting. |
   | 5 | no `external_id` | stays `pending` → **retry fulfilment** via `FulfilmentServiceInterface` (the ABSORBED 13-4a handoff): re-attempt at most once per tick per row; respects plugin health gate | zh-TW `error_message` refreshed on each failed attempt. |

   Persisted transitions go through a new `RequestRepository.UpdateStatus(ctx, id, status, errorMessage)` modeled on `ParseJobRepository.UpdateStatus` (`parse_job_repository.go:152-186` — subset UPDATE + `updated_at` + RowsAffected guard). `progress` is EPHEMERAL (SSE payload only — never a column; no migration in this story). A qBT-status→request-status mapping table is written as its own function with the `MapQBState` style (`qbittorrent/torrent.go:27-63`): `downloading/queued/checking/stalled` → downloading; `completed/seeding/paused(UP)` → import window (row 3); `error` → failed.

3. **Completion actually happens on a default install — scan trigger (RULING, product-critical).** Scouted fact: there is NO filesystem watcher and the default `scan_schedule` is `manual` (`scan_scheduler.go:262-271`) — an *arr-imported file is INVISIBLE to Vido until a scan, so rule-2.1's ownership check would never flip and the pipeline would dead-end at 下載中 100%. **Then:** when a request ENTERS the import window (rule 3 above — first tick where its download vanished from the queue/completed in qBT), the poller triggers ONE `ScannerService.StartScan` — debounced: skip if a scan is already running, and at most one poller-initiated scan per interval of 2 minutes regardless of how many requests entered the window together (a burst of completions shares one scan). Failure to start the scan is slog-logged, never fatal (Rule 13); the ownership check simply catches up on a later scan.

4. **[@contract-v1] `request_progress` SSE event.** Add `EventRequestProgress EventType = "request_progress"` to `sse/hub.go` (one-line addition alongside `EventDownloadProgress`, hub.go:29). After each tick, **if `ClientCount() > 0`**, broadcast ONE event whose `Data` is a **bare array** (nil→`[]`, never null — broadcaster convention) of ALL non-terminal-stale requests (all rows in active statuses + rows that transitioned THIS tick, so the FE sees the final `completed`/`failed` frame), each item:

   ```json
   {
     "id": "…", "tmdb_id": 550, "media_type": "movie", "title": "鬥陣俱樂部",
     "status": "downloading", "progress": 0.42,
     "fulfilment_source": "arr", "external_id": 123,
     "error_message": null, "requested_at": "…", "updated_at": "…"
   }
   ```

   = the 13-1a request resource + ephemeral `progress` (0–1 float, present only when meaningful, mirroring `Torrent.Progress`). Wire format is the house Event envelope (`{id,type,data}` inside `event:`/`data:` framing — handler.go:63-71). Consumers: 13-3b (acks this AC).

5. **Per-section fail-soft (Rule 13 + broadcaster error taxonomy).** One dead source degrades only its column of the derivation: *arr GetQueue error → rules 2/3 skip queue evidence this tick (rows hold status; slog per the `lastPollErr` WARN-dedup pattern, broadcaster lines 123-141); qBT down → progress refinement skipped (queue-based % still works); CheckOwned error → completion detection skipped this tick. The loop NEVER exits on a source error; ticks are independent.

6. **13-5 seam — completed hook (build the socket, not the appliance).** The poller exposes an optional nil-safe `OnRequestCompleted func(ctx context.Context, req models.Request)` callback field, invoked exactly ONCE per request transition into `completed` (guarded by the status change, not by observation — re-ticks must not re-fire: idempotence lives in the transition edge). 13-5 plugs the Epic-8 subtitle trigger into this seam; THIS story wires nothing into it (capability-honor).

7. **Tests + gates.** Poller unit tests with fake sink + mock repos/services (the broadcaster test file is the template): idle-gate (0 active rows → zero source calls), each derivation row (owned→completed exactly-once hook fire; queue-match→downloading+progress; vanished-queue→hold+single debounced scan; searching; pending-retry), broadcast gating (ClientCount 0 → no Broadcast, reconcile still ran), fail-soft per source, nil→`[]`. Repo `UpdateStatus` test on in-memory sqlite. `pnpm nx test api` + `pnpm lint:all`; Rule 15 wiring check (main.go construct + start/stop in shutdown block, mirroring `downloadProgressBroadcaster` at main.go:424/660-663/705-707).

## Tasks / Subtasks

- [ ] Task 1 (AC #4): `sse/hub.go` — add `EventRequestProgress` constant (+ hub test line).
- [ ] Task 2 (AC #2): `repository/request_repository.go` — add `ListActive(ctx)` + `UpdateStatus(ctx, id, status, errorMessage)` (ParseJob template) + in-memory-sqlite tests. (New unstamped methods on the 13-1a repo — additive, no contract bump.)
- [ ] Task 3 (AC #1, #2, #5): `services/request_status_poller.go(+_test)` — ticker/Start/Stop/narrow-sink lifecycle; derivation engine incl. qBT-status→request-status mapping func; per-source fail-soft; pending-retry via `FulfilmentServiceInterface`.
- [ ] Task 4 (AC #3): import-window scan trigger — `ScannerServiceInterface` dep, running-scan + 2-min debounce guards; tests for single-fire on burst.
- [ ] Task 5 (AC #4): snapshot builder + gated Broadcast (bare array, transitioned-this-tick rows included); tests.
- [ ] Task 6 (AC #6): `OnRequestCompleted` seam + exactly-once transition-edge test.
- [ ] Task 7 (AC #7): main.go wiring (construct near :424 zone, start in goroutine zone, cancel+Stop in shutdown) + full gates (`pnpm nx test api`, `pnpm lint:all`, Rule 15 self-check).

## Dev Notes

### Developer context — copy-map (scouted 2026-07-04)

- **Template to clone:** `services/download_progress_broadcaster.go` — struct :50-63, narrow `progressSink{Broadcast; ClientCount}` iface :21-29 (+ `var _` check), run-loop :93-109, tick gate :116-118, error taxonomy :123-141, nil→`[]` + Broadcast :160-167, idempotent Stop :172-180; test file alongside. Interval const style :19 (ours: `defaultRequestStatusInterval = 15 * time.Second`).
- **SSE:** Event struct hub.go:33-37; `Broadcast` :137-144 (non-blocking, cap 256); `ClientCount` :154-158; wire = whole Event as JSON (handler.go:63-71).
- **Joins:** `DownloadService.GetAllDownloads(ctx,"all","added_on","desc")` returns `[]qbittorrent.Torrent` (NOT keyed — build `map[hash]Torrent` yourself); `Torrent` fields torrent.go:96-113 (`Hash/Progress/Status/ETA/...`); `MapQBState` torrent.go:27-63 is the STYLE precedent — the request mapping is a NEW smaller-vocabulary function, not a drop-in.
- **Ownership:** reuse `AvailabilityService.CheckOwned(ctx, []int64)` (Story 10-4, merges movie+series `FindOwnedTMDbIDs`, batched, no N+1) — do NOT hit the repos directly from the poller. Its doc comment even says the "requested" state awaits Epic 13 — this is that story.
- **Scanner:** `ScannerService.StartScan` (scanner_service.go:116); scheduler intervals manual/hourly/daily ONLY (scan_scheduler.go:262-271) — hence AC #3. Check the service's existing already-running guard before adding your own.
- ***arr queue:** via 13-4a's `plugins.Manager` — iterate REGISTERED+healthy plugins' `GetQueue()`; when only Radarr exists (pre-13-4b), series rows simply never match rule 2 (they're `pending` anyway — Sonarr fulfilment doesn't exist yet). Zero reconciler changes when 13-4b lands.
- **No migration, no new columns:** `DownloadID`/progress are per-tick in-memory joins. If a future story wants persisted progress, THAT story takes the migration + Rule-20 bump.

### Contract stamps + acks (Rule 20)

- **Stamps [@contract-v1]:** AC #2 (derivation semantics — the enum's runtime meaning; consumers 13-3b/13-5/13-2a), AC #4 (`request_progress` payload — consumer 13-3b).
- **Acks:** confirmed against `[@contract-v1]` (Story 13-1a AC #2, AC #3) — resource shape reused verbatim in the SSE payload (+ ephemeral `progress` field is payload-only, list endpoint untouched); confirmed against `[@contract-v1]` (Story 13-4a AC #1 DVRPlugin/QueueItem, AC #6 fulfilment semantics — the pending-retry consumes the same nil-safe service); confirmed against `[@contract-v1]` (Story 13-4b AC #2 queue mapping `seriesId→ExternalID`).

### Scope walls

- NO FE (13-3b). NO subtitle trigger (13-5 — only the AC #6 seam). NO season/episode awareness (13-2a). NO persisted progress/download_id columns. NO changes to ux3-4-2b's broadcaster or the downloads SSE.

### Latest-tech note

No new dependency; no external API beyond clients built in 13-4a/b. Web research not required this story.

### Project Structure Notes

- New: `services/request_status_poller.go(+_test)`; edits: `sse/hub.go`, `repository/request_repository.go(+_test)` (13-1a file, additive methods), `cmd/api/main.go`.
- Commit scope `feat(13-3a): …`; branch off `main`; gh `j620656786206`.

### Time-dependent visual coverage

- N/A — backend-only story; no `apps/web/src/components/**` files touched.

### References

- [Source: _bmad-output/planning-artifacts/epics/epic-13-request-system.md#13-3]
- [Source: apps/api/internal/services/download_progress_broadcaster.go (ux3-4-2b pattern)]
- [Source: _bmad-output/implementation-artifacts/13-1a-one-click-request.md#AC-1/#AC-2/#AC-3 + 13-4a-arr-dvr-plugin.md#AC-1/#AC-6 + 13-4b-arr-dvr-plugin.md#AC-2]
- [Source: project-context.md#§8-SSE-Hub + Rule-7/11/13/14/15/19/20/24]
- [Source: memory project_qbt_state_mapping (qBT 4.x/5.0+ state names)]

## Change Log

| Date       | Change |
| ---------- | ------ |
| 2026-07-04 | Story created (SM create-story, yolo). Cross-stack split 13-3 → a/b. Rulings: gate moved poll→broadcast (reconciler has SSE-independent duties) with ListActive idle-gate; completed = Vido-library ownership (no DVRPlugin change); import-window holds `downloading` (no 6th state) + debounced scan trigger (default install has manual-only scans — product-critical); progress ephemeral (no migration). [@contract-v1] on AC #2/#4. 13-5 seam = OnRequestCompleted. Status → ready-for-dev. |

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - If **NO**: state `N/A — no out-of-scope work discovered`.
  - If **YES**: classify each per Rule 24 (①/②/③) with tracked entry IDs; prose-only mentions are banned.

### File List
