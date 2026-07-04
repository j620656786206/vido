# Story 13.4a: *arr DVR Plugin — Infra + Radarr + Movie Fulfilment

Status: review

**Epic:** Epic 13 — Request System · **FR:** P3-004 (G-4) · **Artery #2 (part 1)** · **BACKEND-ONLY**
**Depends on: 13-1a merged** (requests table + RequestService must exist — this story extends them).
**Split:** 13-4 is BACKEND-ONLY (no cross-stack split) but LARGE → size-split per the epic's pre-flag: **13-4a = plugin infra + `DVRPlugin` + config/health + Radarr + movie fulfilment** / 13-4b = Sonarr + series fulfilment.

## Story

As a NAS owner with Radarr configured,
I want my movie 想要 requests automatically routed to Radarr (search + grab + download hand-off),
so that a request actually gets fulfilled — the realistic path that makes Vido an Overseerr replacement.

## Acceptance Criteria

1. **[@contract-v1] `DVRPlugin` interface (§7 shape — consumed by 13-4b/13-3a/13-2a).** **Given** the new greenfield package `internal/plugins/`, **then** it defines exactly:

   ```go
   type PluginConfig struct {
       URL    string `json:"url"`
       APIKey string `json:"-"`   // never serialized/logged (masking handler is NOT wired — json:"-" is the guard)
   }
   type QueueItem struct {        // normalized across Radarr/Sonarr for 13-3a
       ExternalID   int64   // *arr's own movie/series id
       Title        string
       Status       string  // *arr raw status string
       Size         int64
       SizeLeft     int64
       DownloadID   string  // torrent hash — 13-3a joins this to the qBT monitor
   }
   type DVRPlugin interface {
       Name() string
       TestConnection(ctx context.Context, config PluginConfig) error
       AddMovie(ctx context.Context, tmdbID int64, opts AddOptions) (externalID int64, err error)
       AddSeries(ctx context.Context, tmdbID int64, opts AddOptions) (externalID int64, err error)
       GetQueue(ctx context.Context) ([]QueueItem, error)
   }
   type AddOptions struct { QualityProfileID int64; RootFolderPath string; SearchNow bool }
   ```

   Radarr's `AddSeries` returns a typed `DVR_NOT_SUPPORTED` error (movie-only plugin); Sonarr's `AddMovie` mirrors that in 13-4b. Changing this interface later = Rule 20 bump + downstream stale-mark.

2. **Radarr client (`internal/plugins/radarr/`).** **Given** a configured base URL + API key, **then** the client implements `DVRPlugin` against Radarr API v3 (`{url}/api/v3/...`, header `X-Api-Key`, single reused `http.Client` 10s timeout, `*rate.Limiter` 10 req/s burst 10 — LAN-local observed-safe ceiling, `limiter.Wait(ctx)` first line per Rule 27 ①):
   - `TestConnection` → `GET /api/v3/system/status` (200 + parseable version = pass; 401 → `DVR_AUTH_FAILED`);
   - `AddMovie` → `POST /api/v3/movie` body `{tmdb_id→tmdbId, qualityProfileId, rootFolderPath, monitored: true, addOptions: {searchForMovie: <SearchNow>}}` → returns Radarr movie `id`; a 400 "already exists" maps to `DVR_ADD_FAILED` with the upstream message;
   - `GetQueue` → `GET /api/v3/queue?pageSize=100` (paginated `{records: […]}`) → normalized `[]QueueItem` (movie queue items carry `movieId` → `ExternalID`, `downloadId` → `DownloadID`);
   - `GetQualityProfiles` (`GET /api/v3/qualityprofile`) + `GetRootFolders` (`GET /api/v3/rootfolder`) — client-level extras (not on the `DVRPlugin` interface) used by config validation + the settings passthrough (AC #4).
     All tested against `httptest.NewServer` (house convention — `qbittorrent/client_test.go` structure).

3. **Config persistence — settings table + secrets (RULING: no new table, no migration).** **Given** §7 requires per-plugin config in SQLite, **then** follow the qBittorrent precedent EXACTLY: settings keys `radarr.url`, `radarr.enabled`, `radarr.quality_profile_id`, `radarr.root_folder_path` via `SettingsRepository`, and `radarr.api_key` through `secretsService.Store/Retrieve` (AES-256-GCM at rest). The settings table IS SQLite — the §7 mandate is satisfied; a dedicated `plugin_configs` table is deliberately rejected (reuse-over-reinvent; kills the 027/028 migration-ordering coupling with 13-1a). **13-4a ships ZERO migrations.**

4. **[@contract-v1] Config endpoints (consumed by the 13-6 settings UI).** Mirror the qBT settings triad (`qbittorrent_handler.go:144-151` shapes):
   - `GET /api/v1/settings/radarr` → config sans key (`has_api_key: true|false`, never the key itself) + live health block `{status: "healthy"|"unhealthy"|"unconfigured", last_checked_at, message}`;
   - `POST /api/v1/settings/radarr/test` → tests body config if provided, else saved config (`TestConnectionWithConfig` pattern);
   - `PUT /api/v1/settings/radarr` → **server-side test-before-save guard (NEW vs qBT precedent, §7 mandate "must pass before save"):** runs `TestConnection` with the incoming config INSIDE SaveConfig and refuses persistence (409 `DVR_TEST_FAILED`) on failure — not just UI-driven test-then-save;
   - `GET /api/v1/settings/radarr/quality-profiles` + `GET /api/v1/settings/radarr/root-folders` → passthrough (needed to choose valid `quality_profile_id`/`root_folder_path`; consumed by 13-6).
     Handlers/service follow the `qbittorrent_service.go` / `qbittorrent_handler.go` split.

5. **Plugin manager + 60s health check (self-contained — RULING: do NOT extend `ServicesHealth`).** A `plugins.Manager` registers configured plugins at startup, caches clients by config fingerprint (`URL|APIKey` — `download_service.go:24-60` pattern), owns health state, and runs a 60s scheduler (copy `retry/scheduler.go` Start/Stop/`sync.WaitGroup`/`stopCh` lifecycle; interval configurable, default 60s per §7). Health transitions write `connection_history` events via the existing `ConnectionHistoryRepository` (add `radarr`/`sonarr` to `services.ValidServiceNames` + `models/degradation.go` service-name constants so `GET /api/v1/health/services/:service/history` works). The hardcoded 5-service `ServicesHealth`/`totalServices` model (`monitor.go:55,229`, `degradation.go:237-277`) is left UNTOUCHED — plugin health surfaces via AC #4's settings GET. Scheduler started in main.go's goroutine zone with its own `ctx,cancel` and stopped in the graceful-shutdown block.

6. **[@contract-v1] Movie fulfilment semantics (consumed by 13-3a/13-5).** **Given** 13-1a's `RequestService.Create` currently leaves every row `pending`, **then** a new `FulfilmentServiceInterface` (implemented over the plugin manager) is injected into `RequestService` as an OPTIONAL dependency (nil-safe — 13-1a behavior is preserved exactly when absent/unconfigured):
   - movie request created AND Radarr enabled+healthy → synchronous `AddMovie(tmdbID, opts from config, SearchNow: true)` → on success update the row: `status='searching'`, `external_id=<radarr id>`, `fulfilment_source='arr'`, `updated_at` — the create response then carries `status:"searching"` (within the 13-1a enum contract; no shape change, no bump — see Dev Notes ack);
   - Radarr unconfigured / unhealthy / AddMovie error → request row STAYS `pending` with a clear zh-TW `error_message` (e.g. `Radarr 未設定` / `Radarr 連線失敗`), the POST still returns **201** (graceful degradation — fulfilment is best-effort, never fails the request), and the failure is slog-logged (Rule 13: recorded, never swallowed);
   - tv requests are NOT fulfilled by this story (no Sonarr yet) → stay `pending` with `error_message='Sonarr 尚未支援（13-4b）'`-class reason; **retry of stranded `pending` rows is EXPLICITLY handed off to 13-3a's reconcile loop** (recorded in sprint-status 13-3 seed — do not build a retry loop here).

7. **Rule 7 — activate `DVR_*` (new prefix) + first `PLUGIN_*` use.** New codes in `plugins/errors.go` (local-package convention like `qbittorrent/types.go:42-47`): `DVR_NOT_CONFIGURED`, `DVR_CONNECTION_FAILED`, `DVR_AUTH_FAILED`, `DVR_TIMEOUT`, `DVR_ADD_FAILED`, `DVR_TEST_FAILED`, `DVR_NOT_SUPPORTED`; plus first live use of reserved `PLUGIN_INIT_FAILED` / `PLUGIN_HEALTH_CHECK_FAILED` in the manager. In the SAME change: extend the `project-context.md` Rule 7 code block + authoritative prefix list (13→14 prefixes… 15 counting 13-1a's `REQUEST_*` — coordinate: whichever story merges second reconciles the list) + mega-line entry + sync `code-review/instructions.xml` Step 3 prefix list + date. Typed `PluginError{Code, Message, Cause}` lifts into `APIError.Code` via `errors.As` (handler pattern `qbittorrent_handler.go:120-138`).

8. **Rule 27 pillars — documented posture.** ① limiter per AC #2; ② cache **N/A-justified**: Radarr is LAN-local and every call is a command or live-state read (queue freshness is the point) — deviation recorded here; ③ degrade per AC #6 + health gate; ④ `DVR_*` per AC #7; ⑤ `api_key` via secretsService, `json:"-"`, `MaskSecret()` at any log site — NEVER log the key raw (the slog MaskingHandler exists but is NOT wired; do not rely on it).

9. **Tests + gates.** Radarr client (httptest: auth header, status/movie/queue/profiles endpoints, 401→`DVR_AUTH_FAILED`, timeout); manager (health transitions + history writes + fingerprint rebuild); config service (test-before-save refusal path, secrets round-trip with mock secrets service); fulfilment (movie success transition / radarr-down stays-pending / tv untouched / nil-dep no-op); handler (endpoint shapes, key never in GET response). `pnpm nx test api` + `pnpm lint:all` green; Rule 15 wiring self-check (main.go DI + routes + scheduler start/stop + shutdown).

## Tasks / Subtasks

- [x] Task 1 (AC #1, #7): `internal/plugins/` — `plugin.go` (DVRPlugin/PluginConfig/QueueItem/AddOptions), `errors.go` (PluginError + DVR_*/PLUGIN_* codes).
- [x] Task 2 (AC #2): `internal/plugins/radarr/client.go(+_test)` — reused http.Client + X-Api-Key + limiter (`tmdb/client.go:124-144` construction) + buildURL(`/api/v3`) + TestConnection/AddMovie/GetQueue/GetQualityProfiles/GetRootFolders.
- [x] Task 3 (AC #5): `internal/plugins/manager.go(+_test)` — registration, settings+secrets config load, fingerprint-cached clients, health state, 60s scheduler (`retry/scheduler.go` lifecycle), `connection_history` writes (+ `ValidServiceNames`/constants extension).
- [x] Task 4 (AC #3, #4): `services/dvr_settings_service.go(+_test)` + `handlers/dvr_settings_handler.go(+_test)` — GET/PUT/test + profiles/root-folders routes for `radarr` (parameterize by plugin name — 13-4b adds `sonarr` with zero handler duplication); PUT test-before-save guard; api_key → secretsService.
- [x] Task 5 (AC #6): `services/fulfilment_service.go(+_test)` + extend `services/request_service.go` (optional nil-safe dep; movie branch only) — status/external_id/fulfilment_source transition + graceful pending+error_message.
- [x] Task 6 (AC #7): Rule 7 sync — project-context.md (codes + prefix list + mega-line, Rule 25 discipline on conflict) + CR instructions.xml Step 3.
- [x] Task 7 (AC #5, #9): main.go — DI (manager, services, handler), `dvrSettingsHandler.RegisterRoutes(apiV1)`, scheduler start (goroutine zone ~:637-668) + stop (shutdown block ~:690-716).
- [x] Task 8 (AC #9): full gates — `pnpm nx test api`, `pnpm lint:all`, Rule 15 self-check.

## Dev Notes

### Developer context — copy-map (scouted 2026-07-04)

- **Client skeleton:** `internal/qbittorrent/client.go` (single reused `http.Client` :37-52, `buildURL` :56-60, `TestConnection` :192-219, local error type + codes `types.go:23-47`). *arr is SIMPLER: static `X-Api-Key` header — no login/cookie/re-auth machinery needed.
- **Limiter:** `internal/tmdb/client.go:124-144` (`rate.NewLimiter(rate.Every(...), burst)` + `Wait(ctx)` first line). `golang.org/x/time` already in go.mod:18.
- **Cached client by fingerprint:** `services/download_service.go:24-60`.
- **Settings + secrets:** keys pattern `services/qbittorrent_service.go:14-19`; `SaveConfig` routes password → `secretsService.Store` (:93); `GetConfig` decrypts (:49-71); GET response never returns the secret (`QBConfigResponse` :35-40). Secrets = AES-256-GCM, key from `ENCRYPTION_KEY` env or machine-id (`internal/crypto/`).
- **Test-before-save:** `TestConnectionWithConfig` (`qbittorrent_service.go:122-132`) + handler body-or-saved branch (`qbittorrent_handler.go:100-141`). The server-side PUT guard is NEW (qBT's is UI-driven only) — that delta is deliberate (§7).
- **Scheduler:** `internal/retry/scheduler.go` (:62-74 fields, :109-146 Start/Stop, :156-170 run loop). Simpler ticker variant: `health/monitor.go:302-323`.
- **Health/history:** `connection_history` (mig 014, repo `connection_history_repository.go`, events `connected/disconnected/error/recovered`); `ValidServiceNames` at `services/connection_history_service.go:39-52`; do NOT touch the 5-service `ServicesHealth` (`degradation.go:237-277`, `monitor.go:55,229` hardcodes).
- **main.go zones:** secrets :137, health wiring :229-251, handlers :526-530, routes :595-629, goroutine starts :637-668, shutdown :674-729.
- **`internal/plugins/` confirmed absent** — greenfield; PLUGIN_* codes are reserved-only (zero code refs today), this story gives them their first live use.

### *arr API facts (web-verified 2026-07-04 — step-4 research)

- Radarr v3+ (current v5 still `/api/v3`): `POST /api/v3/movie` requires `tmdbId, qualityProfileId, rootFolderPath` (+ `title` tolerated; server resolves metadata), `monitored`, `addOptions.searchForMovie`. Auth via `X-Api-Key` header. TestConnection = `GET /api/v3/system/status`. Queue = `GET /api/v3/queue` paginated `{page, pageSize, totalRecords, records[]}`; records carry `movieId`, `downloadId`, `status`, `size`, `sizeleft`.
- Radarr is TMDB-native — no id resolution needed (the Sonarr TVDB gotcha is 13-4b's problem, not this story's).

### Contract stamps + acks (Rule 20)

- **Stamps [@contract-v1]:** AC #1 (`DVRPlugin` interface — consumers 13-4b/13-3a/13-2a), AC #4 (settings endpoints — consumer 13-6), AC #6 (fulfilment semantics `pending→searching` + `external_id`/`fulfilment_source` writes — consumers 13-3a/13-5).
- **Acks:** confirmed against `[@contract-v1]` (Story 13-1a AC #2, AC #3) — request resource shape untouched; this story only WRITES `status/external_id/fulfilment_source/error_message/updated_at` within the stamped 5-value enum. Create response may now carry `status:"searching"` — a value change inside the contracted enum, not a shape/semantic break (13-1b already treats pending+searching+downloading uniformly as active); no bump.

### Scope walls

- NO Sonarr / no TVDB resolution (13-4b). NO status poller/SSE (13-3a). NO season/episode selection (13-2a). NO stranded-pending retry loop (handed to 13-3a's reconcile — recorded in the 13-3 sprint-status seed). NO FE (settings UI = 13-6 backlog, filed at this authoring). NO migration (AC #3 ruling).

### Latest-tech note

New package deps: NONE (x/time, x/sync, testify, gin all present). Radarr API verified current 2026-07 (sources in story PR / create-story session): Servarr wiki + ArrAPI docs + Radarr repo issues.

### Project Structure Notes

- New: `internal/plugins/{plugin,errors,manager}.go(+tests)`, `internal/plugins/radarr/client.go(+test)`, `services/{dvr_settings_service,fulfilment_service}.go(+tests)`, `handlers/dvr_settings_handler.go(+test)`; edits: `services/request_service.go` (13-1a file — optional dep), `services/connection_history_service.go`, `models/degradation.go` (constants only), `cmd/api/main.go`, `project-context.md`, CR `instructions.xml`.
- Commit scope `feat(13-4a): …`; branch off `main`; gh `j620656786206`.

### Time-dependent visual coverage

- N/A — backend-only story; no `apps/web/src/components/**` files touched.

### References

- [Source: _bmad-output/planning-artifacts/epics/epic-13-request-system.md#13-4]
- [Source: project-context.md#§7-Plugin-Architecture + Rule-7/13/14/15/19/20/24/27]
- [Source: _bmad-output/implementation-artifacts/13-1a-one-click-request.md#AC-2/#AC-3 ([@contract-v1])]
- [Source: apps/api/internal/qbittorrent/client.go + services/qbittorrent_service.go (precedent chain)]
- [Source: https://github.com/Sonarr/Sonarr/wiki/Series-Lookup + https://arrapi.kometa.wiki/en/latest/radarr.html (API verification)]

## Change Log

| Date       | Change                                                                                                                                                                                                 |
| ---------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 2026-07-04 | Story created (SM create-story, yolo). Size-split 13-4 → 13-4a (infra+Radarr+movie fulfilment) / 13-4b (Sonarr+series). Rulings: settings-table config (NO migration); self-contained health scheduler (no ServicesHealth extension); fulfilment-on-create synchronous, stranded-pending retry → 13-3a. [@contract-v1] on AC #1/#4/#6. Discovery ③: 13-6-arr-settings-ui filed. Status → ready-for-dev. |
| 2026-07-04 | Task 1 (AC #1/#7): greenfield `internal/plugins/` — DVRPlugin/PluginConfig/QueueItem/AddOptions per [@contract-v1] AC #1 shape + PluginError with 7 DVR_* codes + 2 reserved PLUGIN_* codes. TDD: errors_test.go (Error/Unwrap/errors.As, code values, api_key json:"-" guard, compile-time DVRPlugin fake). |
| 2026-07-04 | Task 2 (AC #2): `radarr.Client` implements DVRPlugin vs API v3 — X-Api-Key, reused 10s http.Client, 10rps/burst-10 limiter (Wait first line, Rule 27 ①), TestConnection(version-parse gate, 401→DVR_AUTH_FAILED), AddMovie(tmdbId/monitored/searchForMovie body; 400→DVR_ADD_FAILED w/ upstream msg), GetQueue (paginated, movieId→ExternalID/downloadId→DownloadID, maxQueuePages=20 guard), GetQualityProfiles/GetRootFolders extras, AddSeries→DVR_NOT_SUPPORTED, timeout→DVR_TIMEOUT. 16 httptest cases. |
| 2026-07-04 | Task 3 (AC #5): `plugins.Manager` — Register/GetClient (fingerprint `URL\|APIKey` cache, download_service pattern), settings+secrets LoadConfig (SettingKey* helpers shared with Task 4 service — defined in plugins to keep Rule 19 direction), health state (healthy/unhealthy/unconfigured + nil LastCheckedAt pre-check), CheckHealth transitions → connection_history (connected/disconnected/recovered, monitor.go mapping; first live PLUGIN_INIT_FAILED/PLUGIN_HEALTH_CHECK_FAILED use), 60s scheduler w/ immediate startup sweep (retry/scheduler.go stopCh+WaitGroup lifecycle). models.ServiceNameRadarr/Sonarr + ValidServiceNames extended (ServicesHealth untouched per ruling). 10 manager tests + extended TestIsValidServiceName. |
| 2026-07-04 | Task 4 (AC #3/#4): `DVRSettingsService` + `DVRSettingsHandler` — GET (sans key, has_api_key + live health block) / PUT (server-side test-before-save INSIDE SaveConfig → 409 DVR_TEST_FAILED refusal; disabled-save skips probe — turning a plugin off must not need a reachable server) / POST test (body-or-saved) / quality-profiles + root-folders passthrough via plugins.ProfileLister (types hoisted to plugins pkg so 13-4b sonarr = one string, zero duplication) + Manager.TestConfig (throwaway-client probe). Empty body api_key keeps stored key. 10 service tests + 10 handler tests. |
| 2026-07-04 | Task 5 (AC #6): `FulfilmentService.FulfilRequest` ([@contract-v1] semantics) — movie branch: IsConfigured + health gate (lazy one-shot CheckHealth kills the boot-edge race) + config-derived AddOptions guard → synchronous AddMovie(SearchNow) → repo.UpdateFulfilment(searching/external_id/fulfilment_source='arr') with in-memory mutation so the 201 carries the transition; ALL failure paths stay pending + zh-TW error_message (未設定/連線失敗/新增失敗/設定不完整) + slog (Rule 13); tv → 'Sonarr 尚未支援（13-4b）'. RequestService.SetFulfilmentService optional nil-safe dep (13-1a constructor untouched). NEW repo method RequestRepository.UpdateFulfilment (AC #6-implied write path; real-DB tests incl. rows-affected→ErrRequestNotFound). models.RequestFulfilmentSource* constants added. 8 fulfilment + 2 request-service + 3 repo test cases. |
| 2026-07-04 | Task 6 (AC #7): Rule 7 sync — project-context.md code block +7 DVR_* codes, authoritative prefix list 14→15 (`DVR_`), mega-line 13-4a entry prepended (13-1a demoted to Prior, tail kept once, prettier-verified — bare underscores in the entry initially made prettier non-idempotent/mangled emphasis; fixed by backticking every `_`-bearing token, Rule 25 verification caught it); code-review/instructions.xml Step 3 prefix list + sync date → 2026-07-04 (15 sources). Reconciles 13-1a's "15th at 13-4a merge" note. |
| 2026-07-04 | Task 7 (AC #5/#9): main.go wiring — pluginManager DI (settings+secrets+connection-history repos, 60s default) + radarr factory registration + fulfilmentService → requestService.SetFulfilmentService + dvrSettingsService/Handler("radarr") + RegisterRoutes(apiV1) + scheduler start (own ctx/cancel in goroutine zone, non-fatal on error) + stop in graceful-shutdown block. Swagger: annotations-only per 13-1a precedent (no docs/ pipeline exists in apps/api). |
| 2026-07-04 | Task 8 (AC #9): full gates GREEN — `pnpm nx test api` PASS (full suite), `pnpm nx test web` 2322/2322 PASS, `pnpm lint:all` 0 errors / 123 warnings (all in untouched FE files — story is backend-only), prettier clean, `test:cleanup` no orphans. Rule 15 self-check PASS (DI/routes/scheduler-lifecycle verified; no migrations; UpdateFulfilment uses existing 027 columns w/ real-DB round-trip test). Known pre-existing flake TestScannerService_SSEBroadcast_ScanCancelled recurred intermittently mid-dev (stash-verified on clean main; existing entry preexisting-fail-scanner-sse-scan-cancelled-flake covers it — no new entry per bugfix-10-1/10-5/10-6 precedent); green on the gate run. Status → review. |

## Dev Agent Record

### Agent Model Used

claude-fable-5 (Fable 5)

### Implementation Plan

Copy-map-driven, TDD red-green per task: (1) greenfield `internal/plugins` contract types + typed errors → (2) Radarr v3 client vs httptest (qbittorrent/client_test.go structure) → (3) Manager (fingerprint cache = download_service pattern; scheduler = retry/scheduler.go lifecycle; transitions → connection_history = health/monitor.go mapping) → (4) settings service/handler split (qbittorrent_service/handler precedent + NEW server-side PUT guard) → (5) FulfilmentService over the manager + optional nil-safe RequestService dep + repo UpdateFulfilment → (6) Rule 7 doc/CR sync → (7) main.go DI/routes/lifecycle → (8) full gates. Key layering call: `SettingKey*` helpers + `QualityProfile`/`RootFolder`/`ProfileLister` live in `plugins` so services→plugins stays one-directional (Rule 19) and 13-4b sonarr reuses everything with one registration string.

### Debug Log References

- Step-7 full-suite: TestScannerService_SSEBroadcast_ScanCancelled intermittent failure — reproduced 3/3 on clean main via `git stash -u` (pre-existing; tracked entry preexisting-fail-scanner-sse-scan-cancelled-flake); passed on the final `pnpm nx test api` gate run.
- project-context.md mega-line: first insertion left bare `_` tokens (`DVR_`, code-name list) → prettier emphasis-pairing became non-idempotent and mangled text; reverted via `git checkout`, re-inserted with every underscore-bearing token backticked → `prettier --check` clean (Rule 25 verification caught it).

### Completion Notes List

- 🔗 AC Drift: NONE (checked 'pending|searching|status|fulfil' across `_bmad-output/implementation-artifacts/13-1a-one-click-request.md` — 8 hits, all REUSE not DRIFT: 13-1a AC #9 capability-honor explicitly delegates fulfilment to 13-4; the request resource shape (AC #2/#3) is untouched; create response may now carry `status:"searching"` — a value inside the stamped 5-value enum, no bump)
- 📎 Contract Stamps: FOUND (3 stamped ACs in this story — AC #1 DVRPlugin interface / AC #4 settings endpoints / AC #6 fulfilment semantics, all [@contract-v1] producer-side; upstream 13-1a AC #2/#3 [@contract-v1] consumed with the ack line in Dev Notes; versions reconcile, zero bumps, no stale-marks needed)
- 🎭 A11y Pre-Flight: N/A (100% backend — no apps/web/ files touched)
- 🎨 UX Verification: SKIPPED — no UI changes in this story (backend-only; FE settings UI = 13-6 backlog)
- Rule 27 posture (AC #8): ① 10 req/s burst-10 limiter, `Wait(ctx)` first line ✓ · ② cache N/A-justified (Radarr is LAN-local; every call is a command or live-state read — queue freshness is the point) ✓ · ③ degrade: health gate + EVERY fulfilment failure stays pending w/ zh-TW reason, POST always 201 ✓ · ④ `DVR_*` new prefix + first live `PLUGIN_*` use ✓ · ⑤ api_key via secretsService (AES-256-GCM), `json:"-"` guard, key never in any GET response or log ✓
- Documented deviations (all within AC spirit): (a) PUT test-before-save probe runs only when `enabled=true` — disabling a plugin must not require a reachable server; (b) `QualityProfile`/`RootFolder` + optional `ProfileLister` interface hoisted to the `plugins` package (NOT on the stamped `DVRPlugin` — AC #1 shape stays exact) so 13-4b sonarr reuses the passthrough with zero duplication; (c) NEW repo method `RequestRepository.UpdateFulfilment` — the AC #6 row-write has no Rule-4-compliant path without it (scoped: existing 027 columns only, zero migrations); (d) FulfilmentService runs one lazy `CheckHealth` when a request arrives before the scheduler's first sweep (kills the boot-edge spurious-annotation race); (e) empty PUT/test body `api_key` falls back to the stored key (GET never echoes the key, so the UI cannot round-trip it).
- Pre-existing failure handling (Epic 9c retro AI-2): TestScannerService_SSEBroadcast_ScanCancelled — option 2 already satisfied by existing backlog entry `preexisting-fail-scanner-sse-scan-cancelled-flake` (filed 2026-05-04); no new entry per bugfix-10-1/10-5/10-6 precedent.

### Discovery Triage

- **Authoring-time (SM, 2026-07-04):** ③ `13-6-arr-settings-ui` — FE settings UI for *arr config is in NO epic story (BE endpoints are curl-usable; UI needed for non-technical config; likely needs a small flow-c design addendum). Filed in sprint-status.yaml with bidirectional link. Also: stranded-pending fulfilment retry absorbed into the 13-3 seed scope (sprint-status comment updated — ①-style into a planned story, not a new entry).
- **Dev additions (2026-07-04):** ZERO new out-of-scope discoveries during implementation. The authoring-time ③ (13-6-arr-settings-ui) and the stranded-pending-retry absorption into the 13-3 seed remain the only triaged items. The pre-existing scanner-SSE flake resurfaced but its tracked entry (`preexisting-fail-scanner-sse-scan-cancelled-flake`) predates this story — no new lane needed.

### File List

- apps/api/internal/plugins/plugin.go (new)
- apps/api/internal/plugins/errors.go (new)
- apps/api/internal/plugins/errors_test.go (new)
- apps/api/internal/plugins/radarr/client.go (new)
- apps/api/internal/plugins/radarr/client_test.go (new)
- apps/api/internal/plugins/manager.go (new)
- apps/api/internal/plugins/manager_test.go (new)
- apps/api/internal/models/degradation.go (modified — ServiceNameRadarr/Sonarr constants)
- apps/api/internal/services/connection_history_service.go (modified — ValidServiceNames + radarr/sonarr)
- apps/api/internal/services/connection_history_service_test.go (modified — validity cases)
- apps/api/internal/services/dvr_settings_service.go (new)
- apps/api/internal/services/dvr_settings_service_test.go (new)
- apps/api/internal/handlers/dvr_settings_handler.go (new)
- apps/api/internal/handlers/dvr_settings_handler_test.go (new)
- apps/api/internal/services/fulfilment_service.go (new)
- apps/api/internal/services/fulfilment_service_test.go (new)
- apps/api/internal/services/request_service.go (modified — optional nil-safe fulfilment dep)
- apps/api/internal/services/request_service_test.go (modified — fulfilment wiring + nil-safe tests, mock UpdateFulfilment)
- apps/api/internal/repository/request_repository.go (modified — UpdateFulfilment)
- apps/api/internal/repository/request_repository_test.go (modified — UpdateFulfilment real-DB tests)
- apps/api/internal/models/request.go (modified — RequestFulfilmentSource* constants)
- apps/api/cmd/api/main.go (modified — plugin manager DI, radarr registration, fulfilment/settings services + handler, routes, scheduler start/stop)
- project-context.md (modified — Rule 7 DVR_* codes + prefix list 14→15 + mega-line 13-4a entry)
- _bmad/bmm/workflows/4-implementation/code-review/instructions.xml (modified — Step 3 prefix list + sync date 2026-07-04)
- _bmad-output/implementation-artifacts/sprint-status.yaml (modified — 13-4a status tracking)
