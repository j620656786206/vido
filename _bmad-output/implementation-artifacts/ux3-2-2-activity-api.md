# Story ux3-2-2 — Activity hub aggregate API (`GET /api/v1/activity`)

**Epic:** ux3-activity-hub (UX Redesign Phase 3) · **Status:** done (BE) · **Type:** backend
**Pairs with:** ux3-2-1 design (`flow-k-activity-v2`) · FE follow-up = ux3-2-3 `/activity` route

## What

The composition endpoint behind the v2 Activity hub (ADR **D4-1**). Mirrors the
`status_summary` pattern exactly: a service that **reads existing services** (Rule 4) and
is **fail-soft per section** (B1/F3) — a degraded source marks ONLY its own section
`unavailable` and the endpoint never returns an error envelope. Always `200` with a
per-section payload.

## Contract — `GET /api/v1/activity`

snake_case on the wire; the web client camelCases at the `fetchApi` boundary (Rule 18).

```jsonc
{
  "success": true,
  "data": {
    "active_jobs": {
      "status": "ok",
      "jobs": [
        { "kind": "scan",           "percent_done": 62, "detail": "/media/movies/…", "current": 1234 },
        { "kind": "subtitle_batch", "percent_done": 40, "detail": "ep.mkv", "current": 12, "total": 30 }
      ]
    },
    "pending":   { "status": "ok", "parse_count": 8 },
    "downloads": { "status": "ok", "downloading": 3, "queued": 5, "total": 8 },
    "recent": {
      "status": "ok",
      "events": [
        { "kind": "parse", "result": "completed", "detail": "done.mkv", "at": "2026-06-15T10:00:00Z" },
        { "kind": "parse", "result": "failed",    "detail": "bad.mkv",  "at": "2026-06-15T09:40:00Z" }
      ]
    }
  }
}
```

- Every section carries `status` (`"ok"` | `"unavailable"`) and an `error` string when
  unavailable. `jobs` / `events` are always arrays (never null) — safe to render.
- `kind` / `result` are stable enums; **all human copy + icons live on the web client**
  (i18n) — the backend stays copy-free.
- `active_jobs` is `ok` with an **empty** `jobs` array when nothing is running (not
  `unavailable`) — "no job" is a valid state, not an error.
- `downloads.queued` = `all − downloading − completed − seeding` (paused/stalled/queued/
  errored), clamped ≥ 0.

## Real vs. greenfield (Rule 24 — honour backend capability, don't fabricate)

| Section | Source (live today) | Notes |
|---|---|---|
| `active_jobs.scan` | `ScannerService.GetProgress()` / `IsScanActive()` | ✅ live |
| `active_jobs.subtitle_batch` | `subtitle.BatchProcessor.ActivityProgress()` | ✅ live (new primitive adapter — keeps `services` free of a `subtitle` import → no cycle) |
| `active_jobs` **AI correction** | — | ⛔ **greenfield**: no AI-job tracking exists; omitted until it lands |
| `pending.parse_count` | `ParseJobRepository.GetPending()` (existing iface) | ✅ live (capped at 500) |
| `downloads` | `DownloadService.GetDownloadCounts()` | ✅ live |
| `recent.events` | `ParseJobRepository.ListAll()` filtered to completed/failed | ✅ live, **parse events only** — scan/subtitle/AI *completion* isn't persisted yet (a future activity-log table); v1 honestly shows parse terminal events, capped at 8 |

## Footprint

- New: `services/activity_service.go` (+ `_test.go`, 7 cases incl. each section's fail-soft
  + nil-source degradation), `handlers/activity_handler.go`.
- Reused interfaces `scanStateSource` + `downloadCountSource` from `status_summary_service.go`.
- `subtitle/batch.go`: +`ActivityProgress()` primitive adapter (no new repo/interface/mock surface).
- `cmd/api/main.go`: wired `activityService`/`activityHandler` + registered the route.
- Gates: `go build ./...`, `go vet`, `gofmt` clean, `go test ./internal/{services,handlers,subtitle}` green.

## Next — ux3-2-3 (FE)

`/activity` route + components from the `flow-k-activity-v2` design (reuse `ActivityRow-v2`),
consume this contract, wire the sidebar/mobile-tab 活動 destination live. E2E reuses real
seeding (no data-dependent self-skips).
