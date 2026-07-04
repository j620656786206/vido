# Story 13.4b: *arr DVR Plugin — Sonarr + Series Fulfilment

Status: ready-for-dev

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

- [ ] Task 1 (AC #1): `internal/tmdb` — add `GetTVExternalIDs(ctx, tvID)` endpoint wrapper (+ service method on `TMDbServiceInterface`) + httptest tests (present + absent tvdb_id).
- [ ] Task 2 (AC #1, #2): `internal/plugins/sonarr/client.go(+_test)` — TestConnection (v4 gate) / lookup `term=tvdb:` / AddSeries whole-series / GetQueue / profiles / root-folders; limiter + X-Api-Key + reused client per 13-4a patterns.
- [ ] Task 3 (AC #3): register Sonarr in `plugins.Manager` + settings parameterization (`sonarr.*` keys, secrets, health scheduler) + main.go DI.
- [ ] Task 4 (AC #4): `FulfilmentService` tv branch — searching/failed transitions + graceful pending; update `request_service` tests for the tv path.
- [ ] Task 5 (AC #5): Rule 7 code-list update (`DVR_TVDB_NOT_FOUND`) in project-context.md (+ mega-line entry, Rule 25 discipline).
- [ ] Task 6 (AC #6): full gates — `pnpm nx test api`, `pnpm lint:all`, Rule 15 self-check.

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
