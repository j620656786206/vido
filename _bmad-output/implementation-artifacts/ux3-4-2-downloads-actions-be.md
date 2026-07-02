# Story ux3-4-2 — Downloads card-action endpoints (backend)

**Epic:** ux3-downloads-v2 (UX Redesign Phase 3, Epic 4) · **Status:** done
**Owner:** dev · **Type:** backend (cross-stack BE half) · **FRs:** PH3-M3 (Epic 14 v2)

## Story

As a NAS owner managing my download queue,
I want **pause / resume / remove** actions on each download exposed as backend endpoints,
so that **the Downloads v2 page (ux3-4-1 design) can actually control downloads — not just watch them — closing the "no card actions" gap, with the existing qBittorrent connection-error contract preserved.**

## ⚠️ Scope correction (read FIRST) — this is a FULL vertical build, not route-wiring

The design story (ux3-4-1) assumed the qBittorrent client "almost certainly has Pause/Remove methods, just unwired." **Verified 2026-06-30 — that is FALSE at every layer:**

- **Client** (`apps/api/internal/qbittorrent/client.go`) is **read-only**: `Login` / `Ping` / `TestConnection` / `GetTorrents` / `GetTorrentDetails` only. **No pause/resume/delete.**
- **Service** (`DownloadService`): `GetAllDownloads` / `GetDownloadDetails` / `GetDownloadCounts` only.
- **Handler** (`DownloadHandler.RegisterRoutes`): **three GETs only** (`""`, `/counts`, `/:hash`).

So this story builds the action path **end-to-end**: qBittorrent client methods → service methods → handler endpoints → routes → tests. (This is exactly the "method-exists ≠ wired" claim-verification lesson from `retro-cand-sprint-note-claim-verification` — the claim was grep-checked and corrected.)

## Out of scope (siblings / deferred)

- **Download SSE / live progress (P3-012 / Epic 14 H-1)** — a **separate sibling story** (suggested `ux3-4-2b-downloads-sse-be`), NOT this one. This story is the **actions** half only.
- **NZBGet (H-2), completion notifications (H-3), unified multi-source dashboard (H-4)** — additive Epic 14 features, deferred.

## Acceptance Criteria

1. **AC1 `[@contract-v1]` (pause):** `POST /api/v1/downloads/:hash/pause` pauses the torrent and returns `{ success: true }` (no body data required). An empty/missing `:hash` → `400` `VALIDATION_*` via `ValidationError`.
2. **AC2 `[@contract-v1]` (resume):** `POST /api/v1/downloads/:hash/resume` resumes the torrent, same success/validation shape as AC1.
3. **AC3 `[@contract-v1]` (remove):** `DELETE /api/v1/downloads/:hash?deleteFiles=true|false` removes the torrent from qBittorrent. **`deleteFiles` defaults to `false`** (remove from client, **keep files on disk** — maps to the design's `移除（保留檔案）`; `true` = `移除（連同檔案刪除）`). Returns `{ success: true }`.
3a. **`[@contract-v1]` request/response shape is the contract the FE story (ux3-4-3) acks** — method + path + the `deleteFiles` param name/semantics + the success/error envelope. Stamp `[@contract-v1]` on AC1/AC2/AC3.
4. **AC4 (qBT 4.x / 5.0+ version split):** pause/resume hit the **version-correct** WebAPI endpoint — qBT **4.x** `POST /torrents/pause` / `/torrents/resume`; qBT **5.0+** `POST /torrents/stop` / `/torrents/start` (the 4.x names 404 on 5.x). `delete` is `POST /torrents/delete` on both. Mirror the existing 4.x/5.0 handling precedent (`internal/qbittorrent/torrent.go` state mapping already branches on version — see Dev Notes "qBT State Mapping" / project-context). Version is obtainable via the existing `getVersion` / `TestConnection` (`VersionInfo.AppVersion`).
5. **AC5 (client layer):** `Client` gains `PauseTorrents(ctx, hashes []string)`, `ResumeTorrents(ctx, hashes []string)`, `DeleteTorrents(ctx, hashes []string, deleteFiles bool)` — each a `POST` with form body `hashes=h1|h2` (qBT pipe-joined-hashes convention; `delete` adds `deleteFiles=true|false`), built via `buildURL`, authed via `ensureAuth` + `doWithAuth`, returning a `*ConnectionError` on non-2xx (mirror `GetTorrents`' error wrapping). **Singular-hash convenience is fine, but accept a slice so the FE batch-ops (D2-D-v2) reuse the same method.**
6. **AC6 (`doWithAuth` body-preservation — correctness, do NOT skip):** the existing `doWithAuth` re-auth retry **rebuilds the retried request with a `nil` body** (`client.go:134`). For these new **POST-with-form-body** actions a 401/403 retry would silently send an **empty** body → a no-op pause/delete that *looks* successful. Fix the retry to preserve the body (set `req.GetBody` / re-marshal the form on retry), or route the action POSTs through a body-safe helper. Add a test that asserts the body survives a forced re-auth.
7. **AC7 (service layer):** `DownloadService` gains `PauseDownload(ctx, hash)`, `ResumeDownload(ctx, hash)`, `RemoveDownload(ctx, hash, deleteFiles bool)`, added to `DownloadServiceInterface`; they fetch the client (same `getClient`/config path as the GET methods) and call the AC5 client methods.
8. **AC8 (error contract preserved — reuse, no new Rule-7 code):** action endpoints reuse the **existing** qBT error mapping — `*qbittorrent.ConnectionError` + `qbtErrorToHTTPStatus` (`NotConfigured`→503 + `SETUP_REQUIRED` marker, `AuthFailed`→502, `Timeout`→504, `ConnectionFailed`/default→502). **No new `QBITTORRENT_*` error code is added** (the existing five cover the surface; qBT's pause/resume/delete are idempotent and 200 even for unknown hashes, so there is no new "action-failed" surface to name). If a distinct action-failure code is ever needed it is a Rule-7 extension (sync `code-review/instructions.xml`) — explicitly **out of scope** here.
9. **AC9 (wiring — Rule 15 self-check):** the three routes are registered in `DownloadHandler.RegisterRoutes` (`POST /downloads/:hash/pause`, `POST /downloads/:hash/resume`, `DELETE /downloads/:hash`) **and** verified reachable — the handler is already constructed + `RegisterRoutes`-ed in `main.go` (handler `:515`), so no new main.go wiring is needed beyond confirming the routes appear. **Grep-verify** the routes register (don't assume).
10. **AC10 (Swagger + tests + lint):** Swaggo annotations added to the 3 new handler methods (Rule 15 — `swag` regen if annotations changed); unit tests at all three layers (AC11); `go vet` + `staticcheck` clean (Rule 12); no migration, no schema change.
11. **AC11 (tests):** **client** — a mock qBT HTTP server asserts each method issues the correct **method + path + form body**, the **version-correct** pause/resume path (AC4), and the **body survives a forced 401→re-auth retry** (AC6). **service** — each method calls the right client method with the right args (mocked client). **handler** — `POST .../pause|resume` + `DELETE .../:hash?deleteFiles=...` → `200 {success:true}`; a `*ConnectionError` → the correctly mapped status (503/502/504); empty `:hash` → `400`. Co-located `_test.go` (Rule 9).

## Tasks / Subtasks

- [x] **Task 1 — qBittorrent client action methods (AC: #4, #5, #6)**
  - [x] File: `apps/api/internal/qbittorrent/client.go`. Add `PauseTorrents` / `ResumeTorrents` / `DeleteTorrents` mirroring `GetTorrents` (`:228`) for auth/error wrapping, but `http.MethodPost` with a `application/x-www-form-urlencoded` body (`hashes=` pipe-joined; `delete` adds `deleteFiles`).
  - [x] **Version routing (AC4):** resolve qBT major version (reuse `getVersion`/`TestConnection`; cache on the `Client` like `lastLoginAt` if a per-call `getVersion` is too chatty) and select `/torrents/pause`|`/resume` (4.x) vs `/torrents/stop`|`/start` (5.0+). Cross-ref the `torrent.go` state-mapping version branch for the canonical version-detection idiom.
  - [x] **Fix `doWithAuth` body loss (AC6):** make the 401/403 retry (`:134`) preserve the request body for POSTs (e.g. set `req.GetBody` when building the action request, and have `doWithAuth` use it on retry) + a regression test.
- [x] **Task 2 — Service action methods (AC: #7)**
  - [x] File: `apps/api/internal/services/download_service.go`. Add `PauseDownload` / `ResumeDownload` / `RemoveDownload(…, deleteFiles bool)` to `DownloadServiceInterface` (`:14`) + impl; use the existing `getClient(config)` path (`:44`) the GET methods use.
- [x] **Task 3 — Handler endpoints + routes + Swagger (AC: #1, #2, #3, #8, #9, #10)**
  - [x] File: `apps/api/internal/handlers/download_handler.go`. Add `PauseDownload` / `ResumeDownload` / `RemoveDownload` handlers mirroring `GetDownloadDetails` (`:194`) for hash validation + the `errors.As(&connErr)` → `qbtErrorToHTTPStatus` mapping (reuse, AC8). `RemoveDownload` reads `deleteFiles` via `c.DefaultQuery("deleteFiles","false")` → `strconv.ParseBool`. Register in `RegisterRoutes` (`:266`): `downloads.POST("/:hash/pause", …)`, `.POST("/:hash/resume", …)`, `.DELETE("/:hash", …)`. Add Swaggo annotations.
- [x] **Task 4 — Verify wiring (AC: #9)**
  - [x] `main.go:515` already does `handlers.NewDownloadHandler(downloadService)` + `RegisterRoutes`. Confirm (grep) the 3 new routes appear; no new construction needed. (Rule 15: grep-verify, don't assume.)
- [x] **Task 5 — Tests (AC: #11) + gates**
  - [x] `client_test.go` (mock `httptest` qBT): method/path/body per action; version-routed pause/resume; **body-survives-reauth** (AC6). `download_service_test.go`: each action calls the right client method. `download_handler_test.go`: 200 success + mapped connection-error status + empty-hash 400.
  - [x] `cd apps/api && go test ./internal/qbittorrent/ ./internal/services/ ./internal/handlers/ -run 'Pause|Resume|Remove|Delete|Action' -v`; `pnpm lint:all` (Rule 12); `swag init` if annotations changed.

## Dev Notes

### Verified anchors (2026-06-30)

- **Read-only client** `apps/api/internal/qbittorrent/client.go`: `buildURL` `:47`, `Login` `:55`, `ensureAuth` `:112`, **`doWithAuth` `:121` (the nil-body retry bug at `:134` — AC6)**, `GetTorrents` `:228` (the POST pattern to mirror), `GetTorrentDetails` `:296`.
- **Error codes** `apps/api/internal/qbittorrent/types.go:43-46` + `torrent.go:140` — `QBITTORRENT_{CONNECTION_FAILED,AUTH_FAILED,TIMEOUT,NOT_CONFIGURED,TORRENT_NOT_FOUND}` (Rule 7; reuse — AC8). `ConnectionError` struct carries `Code`.
- **Handler** `apps/api/internal/handlers/download_handler.go`: `qbtErrorToHTTPStatus` `:25`, `SetupRequiredMarker` `:18` (bugfix-10-2 `[@contract-v1]`), `GetDownloadDetails` `:194` (the validate+map pattern), `RegisterRoutes` `:266`.
- **Service** `apps/api/internal/services/download_service.go`: `DownloadServiceInterface` `:14`, `getClient` `:44`, GET methods `:86/:141/:166`, compile-time assert `:201`.
- **Wiring** `apps/api/cmd/api/main.go`: `downloadService` `:163`, `downloadHandler` + RegisterRoutes `:515`.

### qBT WebAPI action endpoints + the version split (AC4 — the key design decision)

- qBT **4.x**: `POST /api/v2/torrents/pause` and `/torrents/resume`, body `hashes=<h1>|<h2>` (or `hashes=all`).
- qBT **5.0+**: those were **renamed** to `/torrents/stop` and `/torrents/start` (the 4.x names 404). `delete` stayed `POST /api/v2/torrents/delete`, body `hashes=…&deleteFiles=true|false`.
- The codebase **already** branches on the qBT 4.x/5.0 difference for **state mapping** (`torrent.go` — "Covers both qBittorrent 4.x (pausedDL/pausedUP) and 5.0+ (stoppedDL/stoppedUP)"; project-context "qBittorrent State Mapping" decision). Use the **same version-detection idiom** so action-endpoint selection and state mapping agree on the version.

### The `doWithAuth` body-loss caveat (AC6 — correctness, not optional)

`doWithAuth` retries a 401/403 by rebuilding the request with a **`nil` body** (`client.go:134`). The current GET callers have no body so it's invisible — but a POST pause/delete that re-auths mid-flight would send an **empty** body and qBT would treat it as a no-op (or `hashes=` empty → nothing happens) while returning 200. The action looks successful but did nothing. **Must** be fixed (preserve the body on retry) with a test that forces a 401 and asserts the second request still carries `hashes=…`.

### Contract / FE handoff

- Stamp `[@contract-v1]` on AC1/AC2/AC3 (method + path + `deleteFiles` semantics + success/error envelope). The FE build story **ux3-4-3** consumes it and must record `confirmed against [@contract-v1]` (Rule 20).
- Reuses the bugfix-10-2 `[@contract-v1]` qBT-error envelope (`SetupRequiredMarker`, `qbtErrorToHTTPStatus`) unchanged — no bump to that contract.

### Rule compliance

- Rule 1 (backend `/apps/api`), Rule 4/11 (handler→service→client; interfaces in their packages), **Rule 7 (REUSE `QBITTORRENT_*`, no new code → no code-review/instructions.xml sync)**, Rule 9 (co-located tests), Rule 12 (lint:all), **Rule 13** (propagate every error — the `doWithAuth` fix is squarely this), **Rule 15** (grep-verify routes register + Swagger updated; the whole story originates from a Rule-15 method-exists≠wired audit).
- **Cross-stack split check:** 5 tasks, all backend, 0 frontend → single story, **no split** (the FE is the separate ux3-4-3).

### Time-dependent visual coverage

- **N/A — no `apps/web/src/components` touched.** Backend-only.

### References

- [Source: `ux3-4-1-downloads-design.md` — Decision #2 / Discovery Triage ③] — the design that requires these actions (this story corrects its "client likely has the methods" assumption).
- [Source: `apps/api/internal/qbittorrent/client.go`, `…/types.go`, `…/torrent.go`] — read-only client + error codes + version split (anchors above).
- [Source: `apps/api/internal/handlers/download_handler.go`, `…/services/download_service.go`] — handler/service to extend.
- [Source: `project-context.md` — "qBittorrent State Mapping" decision; Rule 1/4/7/9/12/13/15] — governing rules + the 4.x/5.0 precedent.
- [Source: `epics/epic-14-download-management-v2.md`] — Epic 14 v2 scope (H-1 SSE is the separate sibling).

## Dev Agent Record

### Agent Model Used

claude-opus-4-8[1m] (Amelia — Dev Agent, BMM dev-story workflow)

### Debug Log References

- `go test ./internal/qbittorrent/ ./internal/services/ ./internal/handlers/ -run 'Pause|Resume|Remove|Delete|Action|ParseQBMajor'` — all green (incl. AC6 `TestClient_PauseTorrents_BodySurvivesReauth`).
- `go test ./...` (full backend) — green except the pre-existing tracked flake `TestScannerService_SSEBroadcast_ScanCancelled` (see Completion Notes).
- `go vet` + pinned `staticcheck-2026.1` on the 3 changed packages — clean. `pnpm lint:all` — 0 errors (123 pre-existing warnings in untouched web files), prettier clean.
- AC9 grep: `downloads.POST("/:hash/pause"|"/:hash/resume")` + `downloads.DELETE("/:hash")` confirmed in `RegisterRoutes`; `main.go` already constructs+registers the handler (`cmd/api/main.go:524`,`:608`) — no new wiring.

### Completion Notes List

- **Implementation (client→service→handler, all backend):**
  - `client.go`: `PauseTorrents`/`ResumeTorrents`/`DeleteTorrents([]string, …)` (AC5) via a shared `doFormAction` (auth + form-POST + non-2xx→`*ConnectionError`); version-routed pause/resume — qBT 4.x `/torrents/pause`|`/resume` vs 5.0+ `/torrents/stop`|`/start` (AC4) via cached `majorVersion` + `parseQBMajorVersion` (fallback 4.x on unparseable, logged); `delete` unchanged both versions; hashes pipe-joined for batch reuse (D2-D).
  - **AC6 (`doWithAuth` body loss) fixed:** the 401/403 retry now rewinds the body via `req.GetBody` and clones headers, so a mid-flight re-auth re-sends the form intact. GET callers have nil `GetBody` → nil body → byte-identical to before (existing `TestClient_DoWithAuth_RetriesOn401` still green). Regression test `TestClient_PauseTorrents_BodySurvivesReauth` asserts the retried POST still carries `hashes=abc123`.
  - `download_service.go`: `PauseDownload`/`ResumeDownload`/`RemoveDownload(…, deleteFiles bool)` on the interface (AC7) + `clientForAction` reusing the GET methods' config-fetch + not-configured guard.
  - `download_handler.go`: 3 handlers + `writeActionError` (reuses `qbtErrorToHTTPStatus` — bugfix-10-2 [@contract-v1] — with NO TorrentNotFound branch since qBT actions are idempotent-200; AC8, no new Rule-7 code); `deleteFiles` via `DefaultQuery("deleteFiles","false")`→`ParseBool` (default false = keep files, AC3); success is `SuccessResponse(c, nil)` → exactly `{"success":true}` (Data omitempty). Routes registered in `RegisterRoutes` (AC9).
- **AC10 Swagger:** Swaggo annotations added to all 3 handler methods. No committed swagger artifact to regen — `apps/api/docs/` is not git-tracked, no `swag` binary present, and `cmd/api/main.go` does not import a generated docs package; annotations live as source comments exactly like the existing GET endpoints. `swag init` therefore N/A here.
- **🔗 AC Drift: NONE** (checked: `downloads/:hash|deleteFiles|PauseTorrents|doWithAuth|/torrents/(pause|resume|stop|start|delete)` across `_bmad-output/implementation-artifacts/*.md` — hits are prior GET endpoints (Story 4-2, bugfix-10-2), the `doWithAuth` lazy-auth from bugfix-5, and the downstream ux3-4-3 pre-ack — all REUSE not DRIFT. The AC6 body-preservation fix keeps GET-caller behavior byte-identical, so bugfix-5 AC 2.3's retry contract is strengthened, not changed.)
- **📎 Contract Stamps: FOUND** (3 stamped ACs in this story — AC1/AC2/AC3 `[@contract-v1]`, the new action-endpoint contract. Upstream: reuses bugfix-10-2 `[@contract-v1]` qBT-error envelope UNCHANGED — no bump. Downstream `ux3-4-3` already pre-acks `confirmed against [@contract-v1]`. All versions reconcile.)
- **🎭 A11y Pre-Flight: N/A** (100% backend — no `apps/web/` files touched).
- **Pre-existing failure (Epic 9c Retro AI-2 → option 2, already filed):** the full-suite gate surfaced `TestScannerService_SSEBroadcast_ScanCancelled` flaking (pass/fail/pass over 3 identical isolated runs). This is the **already-tracked** `preexisting-fail-scanner-sse-scan-cancelled-flake` (filed 2026-05-04, SSE-Hub↔assertion race); this story touched ZERO scanner code, so no new entry — cited here per the recording rule.
- **Rule 15 self-check win:** extending `DownloadServiceInterface` broke a second mock (`mockDownloadService` in `internal/workers/parse_worker_test.go`, which receives the full interface) — caught by `go test ./...`; the 3 no-op methods were added there. `fakeDownloads`/activity mocks use narrower interfaces and were unaffected.
- **Web vitest not run:** story touches 0 `apps/web/` files (backend-only); `pnpm lint:all` already exercised eslint over `apps/web` with 0 errors, and the a11y pre-flight is N/A — so the React suite carries no signal for this change (its own flakes are separately tracked). CI (/ship) runs the full web suite regardless.

### Discovery Triage

- **Did this story discover any work outside its current scope?** **YES — two, both pre-noted:**
  - **③ — download SSE / live progress (P3-012 / Epic 14 H-1)** is the other half of the Downloads-v2 BE need and is **explicitly a separate sibling** (`ux3-4-2b-downloads-sse-be`, to be created). Not absorbed here (distinct concern: real-time push vs request/response actions; would push this story over single-dev size). Bidirectional link: ← ux3-4-1 design Discovery Triage.
  - **③ — `doWithAuth` nil-body retry is a latent pre-existing bug** (`client.go:134`) that only bites body-carrying requests. This story is the first POST-with-body caller, so it **fixes it in-scope (AC6)** rather than deferring (expand-in-place ① would also be defensible, but it's a prerequisite for AC1-3 correctness, so it's absorbed with a dedicated AC + test).
- Reference: `project-context.md` Rule 24; origin: this story's capability audit.

### File List

- `apps/api/internal/qbittorrent/client.go` (MODIFIED — Pause/Resume/DeleteTorrents + version routing + doFormAction + doWithAuth body/header preservation; **CR M1**: `verMu sync.Mutex` guarding version resolution)
- `apps/api/internal/qbittorrent/client_test.go` (MODIFIED — action methods + version routing + delete form body + body-survives-reauth + parseQBMajorVersion; **CR**: +Content-Type-on-retry (L2), +version-fetch-fail (L3), +concurrent-no-race `-race` (M1))
- `apps/api/internal/services/download_service.go` (MODIFIED — Pause/Resume/RemoveDownload + clientForAction + interface)
- `apps/api/internal/services/download_service_test.go` (MODIFIED — action success/not-configured/config-error tests)
- `apps/api/internal/handlers/download_handler.go` (MODIFIED — 3 handlers + writeActionError + routes + Swagger)
- `apps/api/internal/handlers/download_handler_test.go` (MODIFIED — 3 mock methods + action success/delete-parse/error-mapping/empty-hash tests; **CR L3**: +Resume/Remove empty-hash)
- `apps/api/internal/workers/parse_worker_test.go` (MODIFIED — `mockDownloadService` gains the 3 new interface methods; Rule 15 self-check surfaced it)
- `_bmad-output/implementation-artifacts/sprint-status.yaml` (MODIFIED — ux3-4-2-downloads-actions-be: ready-for-dev → in-progress → review)

## Senior Developer Review (AI)

**Reviewer:** Amelia (dev agent, adversarial code-review workflow) · **Date:** 2026-07-02 · **Outcome:** Approve (all High/Medium fixed)

Adversarial CR of this story's own implementation (weak case — same model that wrote it; flagged for a fresh-context re-review if desired). Mandatory gates: 🔒 **Rule 7 Wire Format PASS** (0 new codes; `QBITTORRENT_*` reused unchanged), 🔒 **Rule 20 Contract Bump N/A** (creates `[@contract-v1]`, no `→` bump), 🔒 **Rule 25 Mega-line N/A** (`project-context.md` untouched), **Git ↔ File List: 0 discrepancies**. Findings: 0 High · 1 Medium · 4 Low · 1 Info.

**Fixed (M1 + L2 + L3):**

- ✅ **[MED] M1** — data race on the cached `qbtMajorVer` (one `*Client` is shared across concurrent actions via `DownloadService.getClient`): guarded version resolution with `verMu sync.Mutex` (`client.go`) + added `TestClient_MajorVersion_ConcurrentNoRace` (8 goroutines, passes under `-race`).
- ✅ **[LOW] L2** — `TestClient_PauseTorrents_BodySurvivesReauth` now also asserts the form `Content-Type` survives the re-auth `Header.Clone()`.
- ✅ **[LOW] L3** — added `TestDownloadHandler_ResumeAndRemove_EmptyHash` (Resume/Remove empty-hash → 400) + `TestClient_PauseTorrents_VersionFetchFails` (version-lookup failure → `ConnectionError`).

**Noted, accepted (not fixed):**

- **[LOW] L1** — `qbtMajorVer` has no TTL; a qBT 4→5 upgrade without a config change or app restart would keep routing to the 4.x `/pause` until the client is recreated. Rare edge case.
- **[LOW] L4** — `majorVersion` + `doFormAction` both call `ensureAuth` (harmless no-op; `doFormAction`'s is required for the delete path, which skips version resolution).
- **[INFO] I1** — pre-existing Rule-11 breadth: `parse_worker` consumes the wide `DownloadServiceInterface` though it only needs `GetAllDownloads`; extending the interface forced a mock stub. Out of scope.
- Pre-existing flaky `TestScannerService_SSEBroadcast_ScanCancelled` — already tracked (`preexisting-fail-scanner-sse-scan-cancelled-flake`).

Post-fix: `go vet` + `staticcheck-2026.1` clean; 3-package suite green (incl. `-race`); `pnpm lint:all` 0 errors + prettier clean.

## Change Log

| Date       | Change                                                                                                                          |
| ---------- | ----------------------------------------------------------------------------------------------------------------------------- |
| 2026-06-30 | Story created (SM create-story). Capability audit corrected the design's assumption — qBT client is read-only, so this is a full vertical build (client→service→handler). Flags: qBT 4.x/5.0 endpoint split (AC4), `doWithAuth` nil-body retry fix (AC6). `[@contract-v1]` on AC1-3 for FE ack. SSE = separate sibling. Status → ready-for-dev. |
| 2026-07-02 | Adversarial code-review (Amelia, dev). Fixed **M1** (cached-version data race → `verMu` mutex + concurrent `-race` test), **L2** (Content-Type assertion on re-auth retry), **L3** (Resume/Remove empty-hash + version-fetch-fail tests). L1/L4/I1 accepted as notes. Gates: Rule 7 PASS, Rule 20/25 N/A, Git↔File-List clean. 3-pkg suite + `-race` green; lint:all 0 errors. Status → done. |
| 2026-07-02 | Implemented (dev-story, Amelia). Client: version-routed Pause/Resume (4.x pause/resume vs 5.0+ stop/start) + Delete + `doFormAction`; `doWithAuth` now rewinds body + clones headers on re-auth (AC6, byte-identical for GET callers). Service: Pause/Resume/RemoveDownload + not-configured guard. Handler: 3 endpoints + `writeActionError` (reuses `qbtErrorToHTTPStatus`, no new Rule-7 code) + routes; `{success:true}` via `SuccessResponse(c,nil)`. Tests at all 3 layers incl. AC6 body-survives-reauth; extended-interface break in `workers/parse_worker_test.go` mock fixed (Rule 15). `go test ./...` green modulo tracked flake `preexisting-fail-scanner-sse-scan-cancelled-flake`; `pnpm lint:all` 0 errors + prettier clean. AC1-11 met. Status → review. |
