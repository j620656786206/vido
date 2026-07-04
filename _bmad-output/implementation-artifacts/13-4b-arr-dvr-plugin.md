# Story 13.4b: *arr DVR Plugin — Sonarr + Series Fulfilment

Status: review

**Epic:** Epic 13 — Request System · **FR:** P3-004 (G-4) · **Artery #2 (part 2)** · **BACKEND-ONLY**
**Depends on: 13-4a merged** (plugin infra, `DVRPlugin`, manager, settings handler, fulfilment service) **and 13-1a** (transitively).
**Split:** second half of the 13-4 size-split. Pairs forward with 13-2a (season/episode selection extends this story's `AddSeries`).

## Story

As a NAS owner with Sonarr configured,
I want my TV 想要 requests routed to Sonarr as whole-series adds (search + grab + download),
so that series requests are fulfilled just like movies — completing the *arr fulfilment engine.

## Acceptance Criteria

1. **⚠️ THE TVDB GOTCHA — TMDB→TVDB resolution (web-verified 2026-07-04).** Sonarr's `POST /api/v3/series` **hard-requires `tvdbId`** (a TMDB-only add fails validation: `'Tvdb Id' must be greater than '0'` — Sonarr/Sonarr#7565), and `GET /api/v3/series/lookup` officially documents ONLY name and `term=tvdb:{id}` searches (a `tmdb:` prefix is NOT documented — do not rely on it). **Then** the resolution flow is:
   1. `tmdbService.GetTVExternalIDs(ctx, tmdbID)` — **NEW thin wrapper on the EXISTING `internal/tmdb` client** (Rule 27 reuse: rides the shared limiter/cache/key; zero new client) calling TMDB `GET /tv/{tmdb_id}/external_ids` → `tvdb_id`;
   2. `tvdb_id` empty/0 → the series does not exist on TVDB (Sonarr fundamental limitation) → typed `DVR_TVDB_NOT_FOUND`, request row → `status='failed'` + zh-TW `error_message`（`此影集不在 TVDB 上，Sonarr 無法搜尋`）— an honest terminal failure, NOT a stranded pending;
   3. Sonarr `GET /api/v3/series/lookup?term=tvdb:{tvdbId}` → take the first result's full series object (title/titleSlug/images/seasons — Sonarr wants the lookup-shaped body on POST).

2. **Sonarr client (`internal/plugins/sonarr/`).** Implements `DVRPlugin` (13-4a AC #1 [@contract-v1] — `AddMovie` returns `DVR_NOT_SUPPORTED`) against Sonarr v4 `/api/v3`, mirroring the Radarr client structure (reused `http.Client`, `X-Api-Key`, 10 req/s limiter, httptest tests):
   - `TestConnection` → `GET /api/v3/system/status`;
   - `AddSeries(tmdbID, opts)` → resolution flow (AC #1) then `POST /api/v3/series` with the lookup object + `{qualityProfileId, rootFolderPath, monitored: true, seasons: <all monitored>, addOptions: {monitor: "all", searchForMissingEpisodes: <SearchNow>}}` → returns Sonarr series `id`. **Whole-series only** — season/episode granularity is 13-2a's `[@contract-v1]`-bump territory, not here;
   - `GetQueue` → `GET /api/v3/queue?pageSize=100` → normalized `[]QueueItem` (`seriesId` → `ExternalID`, `downloadId` → `DownloadID`);
   - `GetQualityProfiles` / `GetRootFolders` passthroughs.
     Version note: target **Sonarr v4** (`languageProfileId` removed in v4; v3 would require it — if `system/status` reports v3, `TestConnection` fails with a clear `DVR_TEST_FAILED`「需要 Sonarr v4」 message rather than half-working).

3. **Sonarr config endpoints — zero handler duplication.** **Given** 13-4a's settings service/handler are parameterized by plugin name (13-4a AC #4 [@contract-v1]), **then** `sonarr` lights up as: `GET/PUT /api/v1/settings/sonarr`, `POST /api/v1/settings/sonarr/test`, `GET /api/v1/settings/sonarr/{quality-profiles,root-folders}` — settings keys `sonarr.url`/`sonarr.enabled`/`sonarr.quality_profile_id`/`sonarr.root_folder_path`, `sonarr.api_key` via secretsService, PUT test-before-save guard — all by registration/config, no copied handler code. Manager registers the Sonarr plugin into the SAME 60s health scheduler + `connection_history` (`sonarr` already in `ValidServiceNames` per 13-4a AC #5).

4. **Series fulfilment.** `FulfilmentService`'s tv branch (13-4a left it `pending` with a 13-4b placeholder reason) now routes: tv request + Sonarr enabled+healthy → `AddSeries` → `status='searching'`, `external_id=<sonarr id>`, `fulfilment_source='arr'`; Sonarr unconfigured/unhealthy/add-error → stays `pending` + zh-TW reason (201, graceful, slog-logged); `DVR_TVDB_NOT_FOUND` → `failed` per AC #1.2 (the ONE fulfilment error that is terminal — retrying can't fix TVDB absence). Movie flow (13-4a) untouched.

5. **Rule 7 — extend `DVR_*` (existing prefix, no new prefix).** Add `DVR_TVDB_NOT_FOUND` to `plugins/errors.go` + the `project-context.md` Rule 7 code block (list update only — the `DVR_` prefix + CR sync happened in 13-4a; no instructions.xml change unless the code list is mirrored there).

6. **Tests + gates.** Sonarr client httptest suite (lookup tvdb-term → POST body assertion incl. seasons/addOptions; 401; v3-version rejection; queue normalization); external-IDs wrapper test (tmdb client httptest — hit path + missing-tvdb path); fulfilment tv branch (success / down→pending / no-tvdb→failed); settings parameterization (sonarr keys + secrets round-trip). `pnpm nx test api` + `pnpm lint:all`; Rule 15 wiring check (manager registration in main.go DI).

## Tasks / Subtasks

- [x] Task 1 (AC #1): `internal/tmdb` — add `GetTVExternalIDs(ctx, tvID)` endpoint wrapper (+ service method on `TMDbServiceInterface`) + httptest tests (present + absent tvdb_id).
- [x] Task 2 (AC #1, #2): `internal/plugins/sonarr/client.go(+_test)` — TestConnection (v4 gate) / lookup `term=tvdb:` / AddSeries whole-series / GetQueue / profiles / root-folders; limiter + X-Api-Key + reused client per 13-4a patterns.
- [x] Task 3 (AC #3): register Sonarr in `plugins.Manager` + settings parameterization (`sonarr.*` keys, secrets, health scheduler) + main.go DI.
- [x] Task 4 (AC #4): `FulfilmentService` tv branch — searching/failed transitions + graceful pending; update `request_service` tests for the tv path.
- [x] Task 5 (AC #5): Rule 7 code-list update (`DVR_TVDB_NOT_FOUND`) in project-context.md (+ mega-line entry, Rule 25 discipline).
- [x] Task 6 (AC #6): full gates — `pnpm nx test api`, `pnpm lint:all`, Rule 15 self-check.

## Dev Notes

### Developer context

- **Everything structural comes from 13-4a** — interface, errors, manager, scheduler, settings service/handler, fulfilment service, Radarr client as the reference implementation. This story = one new client + one TMDb wrapper + one fulfilment branch + config registration. If 13-4a's shapes don't fit, that's a 13-4a contract conversation (Rule 20), not an ad-hoc fork here.
- **TMDb wrapper:** follow existing endpoint-wrapper style in `internal/tmdb/tv.go` (`GetTVShowDetails` :246 as the shape); external_ids response = `{id, imdb_id, tvdb_id, …}` — only `tvdb_id int64` is needed. Cache TTL: ride the client's default (external IDs are immutable-ish; 24h fine).
- **Sonarr lookup→POST idiom:** POST the lookup result object enriched with config fields — hand-building a minimal series body is the classic source of Sonarr 400s. Assert the POST body in tests (seasons array present, `monitor:"all"`, `searchForMissingEpisodes` true).
- **`seasons` on the requests row stays NULL** — whole-series adds don't write seasons/episodes JSON; that column activates in 13-2a.

### Contract stamps + acks (Rule 20)

- **Stamps [@contract-v1]:** AC #2 `AddSeries` whole-series semantics + normalized queue mapping (consumers: 13-2a — will BUMP this when adding season/episode selection; 13-3a — queue mapping).
- **Acks:** confirmed against `[@contract-v1]` (Story 13-4a AC #1 DVRPlugin interface, AC #4 settings endpoints, AC #6 fulfilment semantics); confirmed against `[@contract-v1]` (Story 13-1a AC #2, AC #3) — writes stay within the request-resource contract; adds the first `failed` writer (`DVR_TVDB_NOT_FOUND` path), still inside the stamped enum.

### Scope walls

- NO season/episode selection (13-2a bumps AC #2). NO status poller/SSE (13-3a). NO stranded-pending retry (13-3a reconcile). NO FE (13-6). NO new prefix, NO migration.

### Latest-tech note

No new deps. Sonarr facts web-verified 2026-07-04: tvdbId required (Sonarr#7565), lookup `tvdb:` term (Sonarr wiki Series-Lookup), v4 dropped languageProfileId (Servarr wiki). TMDB `GET /tv/{id}/external_ids` is a stable v3 endpoint (existing client's API family).

### Project Structure Notes

- New: `internal/plugins/sonarr/client.go(+test)`; edits: `internal/tmdb/tv.go(+test)`, `services/tmdb_service.go`, `internal/plugins/manager.go`, `services/{dvr_settings_service,fulfilment_service}.go(+tests)`, `cmd/api/main.go`, `project-context.md`.
- Commit scope `feat(13-4b): …`; branch off `main`; gh `j620656786206`.

### Time-dependent visual coverage

- N/A — backend-only story; no `apps/web/src/components/**` files touched.

### References

- [Source: _bmad-output/implementation-artifacts/13-4a-arr-dvr-plugin.md#AC-1/#AC-4/#AC-6 ([@contract-v1])]
- [Source: _bmad-output/planning-artifacts/epics/epic-13-request-system.md#13-4]
- [Source: https://github.com/Sonarr/Sonarr/issues/7565 (tvdbId required) + https://github.com/Sonarr/Sonarr/wiki/Series-Lookup (tvdb: term)]
- [Source: project-context.md#§7 + Rule-7/20/27]

## Change Log

| Date       | Change                                                                                                                                                     |
| ---------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 2026-07-04 | Story created (SM create-story, yolo). Second half of 13-4 size-split. TVDB-resolution ruling: TMDB external_ids → Sonarr lookup tvdb: term (tmdb: prefix undocumented, rejected); no-TVDB = terminal `failed`. Sonarr v4-only gate. [@contract-v1] on AC #2. Status → ready-for-dev. |
| 2026-07-04 | Task 1 (AC #1): `tmdb.TVExternalIDs` + `Client.GetTVExternalIDs` (GET /tv/{id}/external_ids, language-neutral) + `CacheService.GetTVExternalIDs` (providersClient path per GetWatchProviders precedent — Rule 27 ② cache-before-limiter, default 24h TTL, key `tmdb:tv/{id}/external_ids`) + `TMDbService`/interface method. tvdb_id null → 0 (no error) = the AC #1.2 signal. 3 httptest cases; 4 mocks patched (MockClient/MockCacheService/mockTMDbServiceForExplore/mockTMDbServiceForNFO). |
| 2026-07-04 | Task 2 (AC #1/#2): `sonarr.Client` implements DVRPlugin + ProfileLister — v4 gate in TestConnection (major<4 → DVR_TEST_FAILED「需要 Sonarr v4」), AddSeries = TVDBResolver (interface defined in sonarr pkg, Rule 19 services-free) → tvdb==0 → DVR_TVDB_NOT_FOUND → lookup `term=tvdb:{id}` → POST lookup-shaped object enriched (qualityProfileId/rootFolderPath/monitored/all-seasons-monitored/addOptions monitor:"all"+searchForMissingEpisodes), AddMovie → DVR_NOT_SUPPORTED, GetQueue seriesId→ExternalID (paginated + cap warn), profiles/rootfolders. doRequest mirrors radarr (shared-helper extraction deferred per ADR Decision 3). New code `DVR_TVDB_NOT_FOUND` in plugins/errors.go. 12 httptest cases incl. POST-body assertion. |
| 2026-07-04 | Task 3 (AC #3): main.go registers sonarr factory (TVDBResolverFunc closure over shared tmdbService — Rule 27 reuse) + handler `NewDVRSettingsHandler(svc, "radarr", "sonarr")`. Settings/health/scheduler light up by registration alone. New test TestDVRSettingsService_SonarrParameterization (sonarr.* keys + secrets round-trip, radarr keys untouched). |
| 2026-07-04 | Task 4 (AC #4): FulfilmentService generalized — one shared `fulfil(pluginName, displayName, add)` gate→add→transition flow via method expressions (DVRPlugin.AddMovie/AddSeries); tv→sonarr replaces the 13-4a placeholder; `failTerminally` writes status='failed' + zh-TW reason with source/external NULL for DVR_TVDB_NOT_FOUND (the ONE terminal error). Movie flow byte-for-byte semantics preserved (all 13-4a movie tests green untouched). 5 new tv-branch tests + 1 request-service tv-path test. |
| 2026-07-04 | Task 5 (AC #5): Rule 7 code-list +DVR_TVDB_NOT_FOUND (existing DVR_ prefix, count stays 15, no CR-workflow change) + mega-line 13-4b entry prepended (13-4a demoted to Prior; all underscore tokens backticked — prettier clean first try; Rule 25 verification: 3 entries survive). |
| 2026-07-04 | Task 6 (AC #6): gates GREEN — `pnpm nx test api` PASS (uncached; known tracked scanner-SSE flake fired on 2 warm-up runs, 0 failures on the recorded run), `pnpm nx test web` 2322/2322, `pnpm lint:all` 0 errors, prettier clean, gofmt clean on touched files, test:cleanup no orphans. Rule 15 self-check PASS (manager registration + handler param + resolver closure in main.go DI; zero migrations; no new routes beyond parameterization). Status → review. |

## Dev Agent Record

### Agent Model Used

claude-fable-5 (Fable 5)

### Implementation Plan

Everything structural rides 13-4a; this story = one TMDb wrapper + one client + registration + one fulfilment branch. TDD red-green per task. Key layering calls: (a) `GetTVExternalIDs` plumbed through the CacheService providersClient path (the GetWatchProviders language-neutral precedent — Rule 27 ② cache-before-limiter); (b) `TVDBResolver` interface DEFINED in the sonarr package and implemented as a main.go closure over the shared TMDb service, so sonarr never imports services (Rule 19); (c) `FulfilmentService` refactored to one plugin-parameterized flow using Go method expressions (`plugins.DVRPlugin.AddMovie/AddSeries`) instead of a copied tv branch — the 13-4a movie tests pass untouched, proving the semantics held.

### Debug Log References

- Interface extension fan-out: adding `GetTVExternalIDs` to `TMDbServiceInterface`/`ClientInterface`/`CacheServiceInterface` broke 4 test mocks (MockClient, MockCacheService, mockTMDbServiceForExplore, mockTMDbServiceForNFO) — all patched with trivial stubs.
- Known tracked flake `TestScannerService_SSEBroadcast_ScanCancelled` fired on two warm-up `nx test api` runs, 0 failures on the third (recorded) run — same load-dependent intermittency as documented in 13-4a; no new entry (existing `preexisting-fail-scanner-sse-scan-cancelled-flake`).

### Completion Notes List

- 🔗 AC Drift: NONE (checked 'tv requests|Sonarr 尚未支援|failed' across 13-4a/13-1a stories — all REUSE not DRIFT: 13-4a AC #6 explicitly delegates tv fulfilment to 13-4b and its placeholder reason was contractually temporary; movie flow untouched — 13-4a movie tests pass unmodified; the first `failed` writer stays inside the 13-1a stamped 5-value enum, no bump)
- 📎 Contract Stamps: FOUND (this story stamps AC #2 [@contract-v1] — AddSeries whole-series semantics + seriesId→ExternalID queue mapping, consumers 13-2a/13-3a; acks upstream 13-4a AC #1/#4/#6 ×v1 + 13-1a AC #2/#3 ×v1 — all versions reconcile, zero bumps, no stale-marks)
- 🎭 A11y Pre-Flight: N/A (100% backend — no apps/web/ files touched)
- 🎨 UX Verification: SKIPPED — no UI changes (backend-only; FE settings UI = 13-6 backlog)
- Rule 27 posture: ① sonarr client 10 req/s limiter Wait-first ✓ · ② external-ids cached 24h BEFORE the TMDb limiter (providersClient path) ✓ · ③ degrade: health gate + graceful pending + the one honest terminal failed ✓ · ④ `DVR_TVDB_NOT_FOUND` under the existing prefix ✓ · ⑤ `sonarr.api_key` via secretsService, `json:"-"` guard ✓
- Documented deviations (within AC spirit): (a) doRequest/mapTransportError/truncate mirrored from the radarr client rather than extracted — ADR Decision 3 defers a shared base until a THIRD client hand-rolls the triplet; (b) lookup enrichment marks ALL season entries monitored=true (AC-literal "<all monitored>", including specials — Sonarr's addOptions.monitor:"all" governs the effective server-side state); (c) resolver errors (TMDb down) map to DVR_CONNECTION_FAILED → graceful pending, NOT terminal — only a confirmed tvdb_id==0 is terminal.

### Discovery Triage

- N/A — no out-of-scope work discovered. (The authoring-time handoffs stand: season/episode selection = 13-2a AC #2 bump; reconcile/retry = 13-3a; FE = 13-6.)

### File List

- apps/api/internal/tmdb/tv.go (modified — TVExternalIDs + Client.GetTVExternalIDs)
- apps/api/internal/tmdb/tv_test.go (modified — 3 external-ids cases)
- apps/api/internal/tmdb/client.go (modified — ClientInterface method)
- apps/api/internal/tmdb/cache.go (modified — CacheServiceInterface + CacheService.GetTVExternalIDs)
- apps/api/internal/tmdb/fallback_test.go (modified — MockClient stub)
- apps/api/internal/services/tmdb_service.go (modified — interface + service method)
- apps/api/internal/services/tmdb_service_test.go (modified — MockCacheService stub)
- apps/api/internal/services/explore_block_service_test.go (modified — mock stub)
- apps/api/internal/services/enrichment_nfo_test.go (modified — mock stub)
- apps/api/internal/plugins/errors.go (modified — DVR_TVDB_NOT_FOUND)
- apps/api/internal/plugins/sonarr/client.go (new)
- apps/api/internal/plugins/sonarr/client_test.go (new)
- apps/api/internal/services/fulfilment_service.go (modified — generalized fulfil + failTerminally + tv routing)
- apps/api/internal/services/fulfilment_service_test.go (modified — sonarr env + 5 tv-branch tests)
- apps/api/internal/services/dvr_settings_service_test.go (modified — fakeDVRPlugin AddSeries controls + sonarr parameterization test)
- apps/api/internal/services/request_service_test.go (modified — tv-path fulfilment test)
- apps/api/cmd/api/main.go (modified — sonarr registration w/ resolver closure + handler param)
- project-context.md (modified — Rule 7 code list + mega-line 13-4b entry)
- _bmad-output/implementation-artifacts/sprint-status.yaml (modified — status tracking)
