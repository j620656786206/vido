# Story 13.4a: *arr DVR Plugin — Infra + Radarr + Movie Fulfilment

Status: ready-for-dev

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

- [ ] Task 1 (AC #1, #7): `internal/plugins/` — `plugin.go` (DVRPlugin/PluginConfig/QueueItem/AddOptions), `errors.go` (PluginError + DVR_*/PLUGIN_* codes).
- [ ] Task 2 (AC #2): `internal/plugins/radarr/client.go(+_test)` — reused http.Client + X-Api-Key + limiter (`tmdb/client.go:124-144` construction) + buildURL(`/api/v3`) + TestConnection/AddMovie/GetQueue/GetQualityProfiles/GetRootFolders.
- [ ] Task 3 (AC #5): `internal/plugins/manager.go(+_test)` — registration, settings+secrets config load, fingerprint-cached clients, health state, 60s scheduler (`retry/scheduler.go` lifecycle), `connection_history` writes (+ `ValidServiceNames`/constants extension).
- [ ] Task 4 (AC #3, #4): `services/dvr_settings_service.go(+_test)` + `handlers/dvr_settings_handler.go(+_test)` — GET/PUT/test + profiles/root-folders routes for `radarr` (parameterize by plugin name — 13-4b adds `sonarr` with zero handler duplication); PUT test-before-save guard; api_key → secretsService.
- [ ] Task 5 (AC #6): `services/fulfilment_service.go(+_test)` + extend `services/request_service.go` (optional nil-safe dep; movie branch only) — status/external_id/fulfilment_source transition + graceful pending+error_message.
- [ ] Task 6 (AC #7): Rule 7 sync — project-context.md (codes + prefix list + mega-line, Rule 25 discipline on conflict) + CR instructions.xml Step 3.
- [ ] Task 7 (AC #5, #9): main.go — DI (manager, services, handler), `dvrSettingsHandler.RegisterRoutes(apiV1)`, scheduler start (goroutine zone ~:637-668) + stop (shutdown block ~:690-716).
- [ ] Task 8 (AC #9): full gates — `pnpm nx test api`, `pnpm lint:all`, Rule 15 self-check.

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

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### Discovery Triage

- **Authoring-time (SM, 2026-07-04):** ③ `13-6-arr-settings-ui` — FE settings UI for *arr config is in NO epic story (BE endpoints are curl-usable; UI needed for non-technical config; likely needs a small flow-c design addendum). Filed in sprint-status.yaml with bidirectional link. Also: stranded-pending fulfilment retry absorbed into the 13-3 seed scope (sprint-status comment updated — ①-style into a planned story, not a new entry).
- **Dev additions:** if this story discovers more out-of-scope work, classify per Rule 24 (①/②/③) with tracked entry IDs; prose-only mentions are banned.

### File List
