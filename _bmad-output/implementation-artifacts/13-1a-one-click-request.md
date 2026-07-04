# Story 13.1a: One-Click Request — Backend (requests model + endpoints)

Status: done

**Epic:** Epic 13 — Request System · **FR:** P3-001 (G-1) · **Artery #1 (BE half)**
**Split:** 13-1 was cross-stack (7 BE + 7 FE tasks > 3/3 threshold) → split per Epic 8 Agreement 5. This is the **backend** half; `13-1b-one-click-request.md` (FE) depends on this story.
**Sequencing:** Epic 14 (ux3-4-x) is fully done → Epic 13 DEV is unblocked. GATE-A (13-0 design) is open.

## Story

As a user browsing the explore/detail surfaces,
I want my one-click 想要 request recorded as a durable `pending` request,
so that Vido can later acquire the title for me (fulfilment = 13-4) without me leaving the page.

## Acceptance Criteria

1. **Migration 027 — `requests` table (the Epic 13 data foundation).** **Given** a fresh or existing DB, **when** the server boots, **then** `internal/database/migrations/027_create_requests_table.go` (self-registered via `init()`, `NewMigrationBase(27, "create_requests_table")`) creates:

   ```sql
   CREATE TABLE IF NOT EXISTS requests (
     id             TEXT PRIMARY KEY,                -- uuid.New().String() in repo
     tmdb_id        INTEGER NOT NULL,
     media_type     TEXT NOT NULL CHECK(media_type IN ('movie','tv')),
     title          TEXT NOT NULL,                   -- zh-TW-resolved, server-side
     status         TEXT NOT NULL DEFAULT 'pending'
                    CHECK(status IN ('pending','searching','downloading','completed','failed')),
     fulfilment_source TEXT CHECK(fulfilment_source IN ('arr','builtin')),  -- NULL until 13-4 claims it
     external_id    TEXT,                            -- Sonarr/Radarr id, set by 13-4
     seasons        TEXT,                            -- JSON, reserved — written only by 13-2
     episodes       TEXT,                            -- JSON, reserved — written only by 13-2
     error_message  TEXT,
     requested_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
     updated_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
   );
   CREATE INDEX IF NOT EXISTS idx_requests_status ON requests(status);
   CREATE INDEX IF NOT EXISTS idx_requests_tmdb_id ON requests(tmdb_id);
   -- Duplicate-request guard, DB level (service level is AC #4):
   CREATE UNIQUE INDEX IF NOT EXISTS idx_requests_active_unique
     ON requests(tmdb_id, media_type)
     WHERE status IN ('pending','searching','downloading');
   ```

   `Down` = `DROP TABLE IF EXISTS requests`. Column set is **exactly** the epic data model — no extra columns (no `poster_path`: `Component/RequestRow-v2` renders title + status token + progress + action, no poster; adding a column later is a migration + Rule 20 contract bump, decided then, not now).

2. **[@contract-v1] `POST /api/v1/requests` — create.** **Given** body `{"tmdb_id": 550, "media_type": "movie"}` (only these two fields; snake_case per Rule 6), **when** the target resolves via the existing Epic-2 TMDb integration, **then** the server stores the **server-side-resolved zh-TW title** (never a client-sent title) and returns **201** `CreatedResponse` with the full request resource:

   ```json
   {
     "success": true,
     "data": {
       "id": "…uuid…",
       "tmdb_id": 550,
       "media_type": "movie",
       "title": "鬥陣俱樂部",
       "status": "pending",
       "fulfilment_source": null,
       "external_id": null,
       "seasons": null,
       "episodes": null,
       "error_message": null,
       "requested_at": "2026-07-04T12:00:00Z",
       "updated_at": "2026-07-04T12:00:00Z"
     }
   }
   ```

   Errors (Rule 3 envelope, zh-TW messages): invalid/missing body fields → 400 `VALIDATION_REQUIRED_FIELD` / `VALIDATION_INVALID_FORMAT`; unknown `tmdb_id` → 404 `TMDB_NOT_FOUND` (reuse the existing prefix — the failure IS a TMDb lookup miss); duplicate → AC #4; already owned → AC #5.

3. **[@contract-v1] `GET /api/v1/requests` — list.** **Given** any number of stored requests, **then** the endpoint returns **200** `{"success":true,"data":{"requests":[…request resources per AC #2 shape…]}}` ordered `requested_at DESC`, and an empty table yields `"requests": []` (never `null` — coerce nil slice, mirror `filter_presets_handler.go:56-59`). No pagination (single-user NAS, small list — matches the `filter_presets`/`libraries` precedent; revisit only if a future story needs it).

4. **Duplicate-request guard.** **Given** an ACTIVE request (`status` ∈ pending/searching/downloading) exists for the same `(tmdb_id, media_type)`, **when** POST is called again, **then** respond **409** with code `REQUEST_DUPLICATE` and no new row. `completed`/`failed` requests do **NOT** block re-requesting (a failed request must be retryable; a completed one is guarded by AC #5 once the media lands). Enforced at BOTH levels: service pre-check (clean error) + the AC #1 partial unique index (race safety) — an index-violation error surfaced from the repo maps to `REQUEST_DUPLICATE`, not a 500.

5. **Already-in-library guard.** **Given** the `(tmdb_id, media_type)` already resolves to owned media (`movies`/`series` repo `FindByTMDbID`, `repository/interfaces.go:21`/`:106` — `media_type='movie'`→movies, `'tv'`→series), **when** POST is called, **then** respond **409** with code `REQUEST_ALREADY_IN_LIBRARY` and no row (FE shows 已入庫 with no action, but the API must not trust the FE).

6. **Rule 7 — new `REQUEST_*` prefix (subsystem-level obligation).** New codes `REQUEST_DUPLICATE`, `REQUEST_ALREADY_IN_LIBRARY` establish the 14th prefix. **Then** in the SAME change: extend the authoritative prefix list in `project-context.md` (Rule 7 section + prepend a "Last Updated" mega-line entry per its convention — Rule 25 applies if a rebase conflicts) AND sync `_bmad/bmm/workflows/4-implementation/code-review/instructions.xml` Step 3 "Rule 7 Wire Format Check" (inline prefix list + HTML comment sync date). This mirrors the `DVR_*` obligation the epic already flags for 13-4.

7. **Layering + wiring (Rules 4/11/15).** `RequestRepositoryInterface` (repository pkg) + `Requests` field added to the `Repositories` struct AND both factories in `repository/registry.go`; `RequestServiceInterface` (services pkg) with `var _` compile-time check; handler depends only on the service interface; `requestHandler.RegisterRoutes(apiV1)` added to the `/api/v1` block in `cmd/api/main.go` (~line 590-629) with service/handler instantiation in the existing DI section; Swaggo annotations on both endpoints (run `swag init` if the project's flow requires). Grep-verify the route registration per Rule 15 before marking done.

8. **Tests (Rule 9 co-located, testify, hand-written mocks — NO sqlmock/gomock).** Migration test (in-memory `sqlite`, `Up(tx)`, assert schema + partial-index behavior); repository test (in-memory DB, real SQL: Create / List ordering / active-dup index violation / nil-slice); service test (mock repo + mock TMDb service: resolve-title path, movie vs tv branch, dup guard, owned guard, TMDB miss); handler test (mock service + `httptest`: 201 shape, 409s, 400, envelope). `pnpm nx test api`, `pnpm nx lint api`, then `pnpm lint:all` green before push.

9. **Capability-honor (Rule 24 corollary).** This story records intent ONLY: every row is born `pending`; NO fulfilment call, NO status transition, NO SSE event is implemented here (fulfilment = 13-4, transitions + `request_progress` SSE = 13-3a, partial seasons/episodes = 13-2a). The 5-value `status` CHECK is the target enum — the single source of truth downstream stories render against.

## Tasks / Subtasks

- [x] Task 1 (AC #1): Migration `027_create_requests_table.go` — copy `023_create_filter_presets.go` structure; self-register `init()`; Up/Down; co-located migration test incl. asserting the partial unique index blocks a second active `(tmdb_id, media_type)` but allows one after `status='failed'`.
- [x] Task 2 (AC #2): Model `internal/models/request.go` — struct with dual `db:`/`json:` tags (snake_case JSON per AC #2 shape), `models.NullString` for nullable columns, `Validate() error` returning `*models.ValidationError` (tmdb_id > 0; media_type ∈ movie|tv).
- [x] Task 3 (AC #3, #4, #7): Repository `internal/repository/request_repository.go` — interface + `*sql.DB` impl: `Create`, `List` (ordered `requested_at DESC`), `FindActiveByTMDbID(ctx, tmdbID, mediaType)`; sentinel `ErrRequestNotFound` + `ErrRequestDuplicate` (map unique-index violation); uuid PK + `time.Now()` per house pattern; register in `Repositories` struct + both factories; repo test on in-memory DB.
- [x] Task 4 (AC #2, #4, #5): Service `internal/services/request_service.go` — `RequestServiceInterface`; DTO `CreateMediaRequestRequest{TMDbID, MediaType}` (naming: NOT `CreateRequestRequest` — reads awkwardly against the house `CreateXRequest` convention); inject `RequestRepositoryInterface` + `TMDbServiceInterface` + movie/series repos; flow: validate → owned guard (`FindByTMDbID`) → active-dup guard → TMDb resolve (`GetMovieDetails`/`GetTVShowDetails`, title = `Title`/`Name` — zh-TW arrives free via the client's language-fallback chain) → create `pending` row; service test with mocks.
- [x] Task 5 (AC #2, #3, #7): Handler `internal/handlers/request_handler.go` — `RegisterRoutes(rg)` → `rg.Group("/requests")` `POST("")`/`GET("")`; `ShouldBindJSON` → service; `handleRequestError` mapping (errors.Is/As → `REQUEST_DUPLICATE` 409 / `REQUEST_ALREADY_IN_LIBRARY` 409 / `TMDB_NOT_FOUND` 404 / validation 400, Rule 3 envelope + zh-TW message + suggestion); Swaggo annotations; wire service + handler + `RegisterRoutes(apiV1)` in `cmd/api/main.go`; handler test via httptest.
- [x] Task 6 (AC #6): Rule 7 sync — add `REQUEST_DUPLICATE`, `REQUEST_ALREADY_IN_LIBRARY` + `REQUEST_` prefix to `project-context.md` (codes block + authoritative prefix list + mega-line entry) and `code-review/instructions.xml` Step 3 prefix list + sync date.
- [x] Task 7 (AC #8): Full verification — `pnpm nx test api` → `pnpm lint:all` (vet, staticcheck, eslint, prettier — prettier will touch the md edits) → Rule 15 self-check (main.go wiring grep; SELECT column list == scan list == INSERT list).

## Dev Notes

### Developer context — read these first

- **1:1 template chain (copy this shape, do NOT invent):** `filter_presets` — migration `internal/database/migrations/023_create_filter_presets.go`, handler `internal/handlers/filter_presets_handler.go`, service `internal/services/filter_preset_service.go`, repo `internal/repository/filter_preset_repository.go`, model `internal/models/filter_preset.go`, tests co-located at each layer (`filter_presets_handler_test.go:19-75` is the httptest skeleton).
- **Migration system:** custom in-house runner (NOT goose/golang-migrate); latest = `026` → **yours is 027**; self-register via `init()` ONLY (do NOT edit `registry.go` — see comment in `026:14-15`); runs automatically at boot inside per-migration transactions; helper `columnExists` exists at `020:82` (not needed here).
- **Response envelope:** `internal/handlers/response.go` — `SuccessResponse`/`CreatedResponse`/`ErrorResponse` + shortcuts. List wraps in `gin.H{"requests": …}` with nil→`[]` coercion.
- **TMDb resolve (Epic 2, reuse — Rule 27 pillar ✅ by reuse):** inject the existing `services.TMDbServiceInterface` singleton (built at `cmd/api/main.go:192-197`); `GetMovieDetails(ctx, id)` / `GetTVShowDetails(ctx, id)` already run the zh-TW→zh-CN→en fallback chain + tiered cache + rate limiter. ZERO new external client, key, limiter, or `TMDB_*` code.
- **DB conventions:** driver `modernc.org/sqlite` (name `"sqlite"`); `foreign_keys(on)` + WAL enabled per-connection; TEXT-uuid PKs; JSON-in-TEXT is the established pattern (`filter_presets.filters`, `movies.genres`) — `seasons`/`episodes` follow it in 13-2; nullable via `models.Null*` types (`internal/models/types.go`).
- **`media_type` vocabulary — deliberate ruling:** `'movie'|'tv'` (TMDB/FE vocabulary — the FE route is `media/$type.$id` with `tv`, and requests target TMDB entities), NOT `media_libraries.content_type`'s `'movie'|'series'` (that column classifies local folders). Do not "fix" this to `series`; the divergence is intentional and recorded here.
- **`fulfilment_source` stays NULL at create** — 13-4 (the fulfilment engine) claims rows and sets `arr` + `external_id`. SQLite CHECK passes on NULL, no special handling needed.
- **Greenfield confirmed:** zero domain-level `request`/`wishlist` code in `apps/api` (scouted 2026-07-04); `internal/plugins/` does not exist yet (13-4 creates it). You own the `requests` namespace.
- **Rule 13:** every repo/service error propagates or logs-and-returns; index-violation from SQLite must be detected (`strings.Contains(err.Error(), "UNIQUE constraint failed")` or the driver's typed error) and mapped to `ErrRequestDuplicate` — never leaked as a raw 500.

### Contract stamps (Rule 20)

- AC #2 + AC #3 are stamped **[@contract-v1]** — the request-resource JSON shape and the create/list endpoints. Known consumers at authoring: **13-1b** (acks in its Dev Notes), later 13-2a (extends create body), 13-3a/b (status pipeline + list rendering). Any shape change bumps v1→v2 + Change Log row + downstream stale-mark grep per Rule 20 🔁.

### Previous story intelligence (13-0, done 2026-07-04, PR #118)

- GATE-A open: `flow-l-requests-v2` L1–L8 shipped; `Component/RequestRow-v2` registered. The `searching` status renders `warning-tint`/「搜尋中」(DL-v2 §2.5 + Rule TY-3) — FE concern, but the enum name `searching` in AC #1 is that exact contract.
- 13-0 deliberately did NOT touch flow-i/flow-b frames; L2 carries the button affordance spec — irrelevant to BE except: the button's `已請求·處理中` state is driven by this story's list endpoint.

### Latest-tech note

No new third-party dependency is introduced (uuid, testify, gin, modernc/sqlite all repo-pinned; TMDb client reused). Web research skipped — nothing to version-check.

### Project Structure Notes

- All code under `apps/api` (Rule 1); new files: `migrations/027_create_requests_table.go(+_test)`, `models/request.go`, `repository/request_repository.go(+_test)`, `services/request_service.go(+_test)`, `handlers/request_handler.go(+_test)`; edits: `repository/registry.go`, `cmd/api/main.go`, `project-context.md`, `code-review/instructions.xml`.
- Conventional commit scope: `feat(13-1a): …`; branch off `main` (never off another feature branch); gh account `j620656786206`.

### Time-dependent visual coverage

- N/A — backend-only story; no `apps/web/src/components/**` files touched.

### References

- [Source: _bmad-output/planning-artifacts/epics/epic-13-request-system.md#Data-model + #13-1]
- [Source: _bmad-output/planning-artifacts/prd/functional-requirements.md#P3-001]
- [Source: project-context.md#Rule-3/4/6/7/9/11/13/15/19/20/24/27]
- [Source: _bmad-output/implementation-artifacts/13-0-requests-design.md (GATE-A, status vocabulary)]
- [Source: _bmad-output/planning-artifacts/ux-redesign/01-design-language-v2.md#§2.5 (status→token, enum alignment)]

## Change Log

| Date       | Change                                                                                                                                       |
| ---------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| 2026-07-04 | Story created (SM create-story, yolo). Cross-stack split 13-1 → 13-1a (BE, this) / 13-1b (FE). [@contract-v1] stamped on AC #2/#3. New Rule-7 `REQUEST_*` prefix obligation (AC #6). Status → ready-for-dev. |
| 2026-07-04 | Adversarial code review (CR): 0H/3M/2L found, all fixed in-session (real-migration test setup, 4xx log levels, @Failure annotations, TMDB-{id} title fallback, fulfilment_source CHECK test). Status → done. |
| 2026-07-04 | Implemented (dev-story): mig 027 + model + repo (+registry) + service + handler + main.go wiring (Tasks 1–5); Rule 7 三處同步 REQUEST\_ = 14th prefix (Task 6); full gates green — `pnpm nx test api`, `pnpm nx test web`, `pnpm lint:all` (Task 7). 26 new test cases across 4 layers. Status → review. |

## Senior Developer Review (AI)

**Date:** 2026-07-04 · **Outcome:** Approve (all findings fixed in-session) · **Reviewer:** adversarial CR workflow (claude-fable-5)

- **Git vs File List:** 0 discrepancies (10 new + 4 edits, exact match).
- **🔒 Rule 7 Wire Format:** PASS (4 codes checked — REQUEST_DUPLICATE / REQUEST_ALREADY_IN_LIBRARY / VALIDATION_REQUIRED_FIELD / VALIDATION_INVALID_FORMAT, all under registered prefixes).
- **🔒 Rule 20 Contract Bump:** N/A (no stamp bumps — new v1 stamps only).
- **🔒 Rule 25 Mega-line:** N/A (clean single-author prepend, no rebase/merge).
- **Findings (0 High / 3 Medium / 2 Low) — ALL FIXED:**
  - [x] M1 repo test carried a hand-copied 027 schema literal (drift risk) → setup now applies the REAL migration chain via `migrations.NewRunner` + `GetAll()` + `Up()`.
  - [x] M2 expected 4xx flows logged at Error (log noise) → `handleRequestError` logs Debug for duplicate/owned/tmdb-miss/validation, Error only for unexpected.
  - [x] M3 Swagger lacked @Failure contract rows → 400/404/409 (+500 on GET) annotated with their Rule-7 codes.
  - [x] L1 `resolveTitle` empty-title pathological edge → deterministic `TMDB-{id}` placeholder + service test.
  - [x] L2 `fulfilment_source` CHECK untested → migration test case added.
- **Post-fix gates:** request tests 4 layers green; `pnpm nx test api` exit 0 (verified standalone); `pnpm lint:all` green.

## Dev Agent Record

### Agent Model Used

claude-fable-5 (Claude Fable 5)

### Debug Log References

- Repo List test: SQLite `datetime()` can't parse the Go driver's time serialization → test fixed to pass a Go `time.Time` arg for the older-timestamp UPDATE (implementation unaffected).
- prettier non-idempotent on project-context.md mega-line: bare underscore tokens (`tmdb_id`, `media_type`) parsed as markdown emphasis → oscillating reflow. Fixed by backtick-wrapping every underscore-bearing token in the new entry (Rule 25's CJK warning generalizes to bare `_`).

### Completion Notes List

- 🔗 AC Drift: N/A (new requests subsystem — no prior story AC covers `/api/v1/requests` or the requests table; `main.go`/`registry.go` edits are additive registration only, no existing contract's observable behavior changed)
- 📎 Contract Stamps: FOUND (2 stamped ACs in this story — AC #2/#3 [@contract-v1], producer side; no upstream stamped refs consumed: 13-0 is a design story with no stamps, implicit v0)
- 🎭 A11y Pre-Flight: N/A (100% backend — no apps/web/ files touched)
- 🎨 UX Verification: SKIPPED — no UI changes in this story
- Implementation exactly per AC: rows born `pending` (AC #9 verified by service+repo tests); dup guard at BOTH levels (service pre-check + partial-unique index mapped to `ErrRequestDuplicate`, migration test proves failed-rows don't block re-request); owned guard via `FindOwnedTMDbIDs` bulk (typed, no error-string matching); zh-TW title server-side via existing TMDbService (movie→Title, tv→Name, Original* fallback); `TMDB_NOT_FOUND` passes through typed with its own status code.
- Rule 7: `REQUEST_` registered as the 14th prefix in project-context.md (codes block + authoritative list + mega-line entry) and code-review instructions.xml Step 3 (list + sync date 2026-07-04). 13-4a adds `DVR_` as 15th and reconciles the count at its merge (noted in both).
- Swagger: Swaggo annotations added on both endpoints (house pattern = annotations only; no swag-init target exists in project.json).
- Full regression: `pnpm nx test api` ✅, `pnpm nx test web` ✅ (orphan-process cleanup verified: none), `pnpm lint:all` ✅ (vet/staticcheck/eslint 0 errors/prettier). Pre-existing failures encountered: NONE.

### Discovery Triage

- N/A — no out-of-scope work discovered. (The prettier mega-line gotcha was fixed in-place inside this story's own Task-6 edit — same surface, not out-of-scope; documented in Debug Log for the next mega-line editor.)

### File List

- apps/api/internal/database/migrations/027_create_requests_table.go (new)
- apps/api/internal/database/migrations/027_create_requests_table_test.go (new)
- apps/api/internal/models/request.go (new)
- apps/api/internal/repository/request_repository.go (new)
- apps/api/internal/repository/request_repository_test.go (new)
- apps/api/internal/repository/registry.go (modified — Requests field + both factories)
- apps/api/internal/services/request_service.go (new)
- apps/api/internal/services/request_service_test.go (new)
- apps/api/internal/handlers/request_handler.go (new)
- apps/api/internal/handlers/request_handler_test.go (new)
- apps/api/cmd/api/main.go (modified — service :215, handler :540, route :624)
- project-context.md (modified — Rule 7 codes + 14-prefix list + mega-line entry)
- \_bmad/bmm/workflows/4-implementation/code-review/instructions.xml (modified — Step 3 prefix list + sync date)
- \_bmad-output/implementation-artifacts/sprint-status.yaml (modified — status tracking)
