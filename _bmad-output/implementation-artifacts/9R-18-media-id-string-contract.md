# Story 9R-18: media_id string contract — align Route C chain with UUID movie PKs

Status: done

> **Origin (party-mode ruling, 2026-07-06, Winston+Amelia+Murat 3-0, Alexyu ratified):** movie/series PKs are UUID STRINGS (`library_service.go:161`, `parse_queue_service.go:81/216/273/312` — `uuid.New().String()`; prod data confirmed via retro-8-TD4), but the entire Route C chain assumed int64 numeric ids — `POST /movies/:id/transcribe` ParseInts the id (400 on every real movie), the 9R-16 batch orchestrator ParseInt-SKIPS every UUID movie (preview counts 38, start enumerates 0), and the FE `Number(uuid)` = NaN. The int64 assumption came from a TMDB-id reflex in 9R-10, not any ADR — **it is a bug, not a design**. Fix: string media ids end-to-end, consistent with the glossary API (already string `:mediaId`) and `models.Movie.ID`. Filed as 🔴 `disc-2026-07-movie-id-int64-contract-mismatch` by the ux3-subtitle-v2-batch adversarial CR.
>
> **Scope wall:** this story fixes code ON MAIN (Route C BE + slice-1 FE). The HELD branch `feat/ux3-subtitle-v2-batch` adapts itself at rebase (drops its `Number()`/`String()` conversions, re-acks v2) — NOT this story's scope.

## Story

As a Vido user whose library was scanned into UUID-keyed movie rows (i.e., every real library),
I want the Route C generation chain to address media by the ids the database actually has,
so that 生成字幕 and batch generation operate on my actual movies instead of 400-ing or silently enumerating zero.

## Acceptance Criteria

1. **Transcribe route takes string ids.** `POST /api/v1/movies/{id}/transcribe` treats `:id` as an opaque STRING (remove `strconv.ParseInt`, `transcription_handler.go:62`); validation = non-empty only. A UUID id reaches the service untouched. 400 `VALIDATION_INVALID_FORMAT` remains only for empty ids. Swagger updated.
2. **TranscriptionService keyed by string.** `inProgress map[int64]string` → `map[string]string` (`transcription_service.go:90-91`); the 8 int64 signatures (`loadGlossary:153`/`IsInProgress:179`/`StartTranscription:190`/`RunTranscription:225`/`acquireJob:260`/`runPipeline:289`/`translateAndPersist:411`/`translateSRT:545`) + the handler-side interface (`transcription_handler.go:25-26`) take `mediaID string`; the two `strconv.FormatInt` shims (`:157` glossary, `:446` writeback) disappear. No behavior change beyond the type.
3. **SSE payloads carry string media ids [@contract-v1].** The 5 `transcription_*`/`translation_progress` EVENT KINDS across their 6 emission sites (`transcription_service.go:314/349/390/416/566` + failJob `~:719-733` — translating emits from two places): `media_id` becomes a STRING (the UUID). `generation_batch_progress` (`generation_batch.go:400-421`): `current_media_id` becomes a STRING (`items[].media_id` is the 202 HTTP body — AC 4's scope, not this SSE payload). Wire key names/casing unchanged; only the value type. (These payloads ship to the FE fixed in AC 5 within the SAME PR — no mixed-version window.)
4. **Generation-batch API takes/returns string ids [@contract-v1→v2 bump].** Start body `media_ids: [<string>...]`; 202 `items[].media_id` string; the orchestrator's ParseInt-skip in `toItem()` is DELETED (it was the silent UUID-dropper); scope=selected repo lookups use the string ids directly. **Rule 20 bump mechanics (producer-side, mandatory):** in `9R-16-batch-generation-endpoint.md`, bump the stamped ACs #1/#2/#3/#7/#9 `[@contract-v1]` → `[@contract-v2]`; ⚠️ the file carries a SIXTH stamp on AC 12 (writeback) — **deliberately NOT bumped** (its id parameter is already string; record this in the Change Log so the stamp-grep doesn't stall). The file has NO Change Log section yet — CREATE one, one row per bumped AC (`{what changed: media ids int64→string, what breaks downstream: any consumer parsing/emitting numeric ids}` + the AC-12-not-bumped note). Run the consumer grep, and stale-mark the ONE not-done consumer `ux3-subtitle-v2-batch` (Dev Notes + sprint-status entry — pre-written at authoring, confirm idempotently; it re-acks v2 at its rebase).
5. **Slice-1 FE consumes string ids.** `useGenerationProgress.ts`: `media_id` filter compares STRINGS — inline types at `:40/:154/:254` (drop numeric coercion; specs at `useGenerationProgress.spec.ts:52/112` use `media_id: 42/777` → UUID strings per AC 7); `transcriptionService.ts:35`: `startTranscription(movieId: string)` (URL interpolation unchanged); `ManageSubtitleDialogV2.tsx:176/:184`: remove the two `Number(mediaId)` calls (the prop is already string, `:118`); `LocalDetailV2.tsx` already passes strings (`:271/:280`) — verify only. All changed types are INLINE in these files (`apps/web/src/types/` has no mediaId types).
6. **Integration guard (Murat, blocking).** NEW real-sqlite integration test that creates a movie through the REAL creation path (`library_service`/scanner path → `uuid.New().String()` id, with file_path + missing zh-Hant), then exercises the chain over HTTP handlers. No single existing template — COMPOSE two proven halves: `library_service_test.go:~30-55` (`setupTestDB`: real sqlite tmpfile + `migrations.NewRunner`+`RegisterAll`+`Up()` + real repos + real creation path) + `glossary_handler_test.go:60-66` (gin.TestMode + `RegisterRoutes` + httptest). Chain to assert: `POST /movies/{uuid}/transcribe` (asserts NOT 400 — 503-or-202 depending on availability gate), `GET /subtitles/generation-batch/preview` counts it, `POST /subtitles/generation-batch {scope:"missing"}` enumerates it (`items[0].media_id == uuid`). This test is the permanent tripwire for id-format assumptions.
7. **Fixture convention (Murat).** All NEW specs in this story use UUID-string media ids. Add the convention note to the story-touched spec files where numeric ids were used as fixtures (comment: media ids are UUID strings — mirror the prod creation path, do not invent numeric ids). Sweep the Route C spec files touched here; do NOT sweep the whole repo (out of scope).
8. **No collateral id changes.** TMDB ids stay numeric everywhere (`tmdb_id` is genuinely an integer — different concept). `requests.tmdb_id`, discover/TMDB routes, *arr `external_id` parsing (13-3a poller) are UNTOUCHED. Glossary API unchanged (already string). Verify with a diff review that no `tmdb_id` surface was renamed/retyped.
9. **Tests + gates.** Updated unit/httptest suites green with string ids (UUID fixtures per AC 7); the AC 6 integration guard green; full `go test ./...` + `pnpm nx test web` + `pnpm lint:all` 0 errors + builds + prettier. `go test -race` on touched Go packages (known pre-existing `disc-2026-07-retry-mock-race` failure in retry mocks is exempt — do not chase).
10. **Tracking.** sprint-status: 🔴 `disc-2026-07-movie-id-int64-contract-mismatch` annotated `resolved-by: 9R-18`; `ux3-subtitle-v2-batch` entry stale-marked per AC 4. This story's entry → review at dev completion.

## Tasks / Subtasks

- [x] Task 1 (BE): handler + service string keying (AC: 1, 2)
  - [x] transcription_handler `:id` string-through; service map/signature flip; httptest with UUID ids
- [x] Task 2 (BE): SSE + batch API string ids (AC: 3, 4)
  - [x] 5 transcription payloads + generation_batch payload/body/items; DELETE toItem ParseInt-skip; orchestrator string plumbing; payload-shape tests updated
  - [x] Rule 20 bump in 9R-16.md (v2 stamps + Change Log rows + consumer grep + stale-mark)
- [x] Task 3 (BE): integration guard (AC: 6)
  - [x] Real-creation-path UUID movie → transcribe/preview/start chain over HTTP handlers
- [x] Task 4 (FE): slice-1 string ids (AC: 5)
  - [x] useGenerationProgress filter, transcriptionService types, dialog/detail pass-through, Number() removals, specs to UUID fixtures
- [x] Task 5: convention + tracking + gates (AC: 7, 8, 9, 10)
  - [x] Fixture-convention notes; tmdb_id untouched diff-check; sprint-status edits; full gates

**Cross-stack split check:** backend tasks = 3, frontend tasks = 1 (+1 shared) → no a/b split. ✓

## Dev Notes

### Verified anchors (this session, 2026-07-06)

- UUID creation: `apps/api/internal/services/library_service.go:161`, `parse_queue_service.go:81/216/273/312`.
- ParseInt entry: `apps/api/internal/handlers/transcription_handler.go:62`.
- Single-flight map: `apps/api/internal/services/transcription_service.go:78-79` (`mu` + `inProgress map[int64]string`).
- Writeback already string: `translateAndPersist` calls `UpdateSubtitleStatus(ctx, <string id>, …)` — the 9R-16 CR asserted `"42"` string round-trip; after this story the int64→string conversion shim there DISAPPEARS (id stays string end-to-end).
- Batch: `apps/api/internal/handlers/generation_batch_handler.go` (body/DTO types), `apps/api/internal/services/generation_batch.go` (`toItem()` ParseInt-skip = the silent dropper; 11-key SSE map).
- SSE payload emission sites: `transcription_service.go` ~:240-345 (extracting/transcribing/translating/complete) + failJob ~:616-622.
- FE slice-1: `apps/web/src/hooks/useGenerationProgress.ts` (media_id filter), `apps/web/src/services/transcriptionService.ts`, `apps/web/src/components/subtitle/ManageSubtitleDialogV2.tsx`, wiring in `LocalDetailV2.tsx`.
- Already-correct string surfaces (do NOT touch): glossary (`/media/:mediaId/glossary`), `MovieRepository` methods (string ids), `movies.id TEXT PRIMARY KEY`.
- NOT media ids (do NOT touch): `tmdb_id` (numeric, correct), *arr `external_id` (numeric *arr ids, poller ParseInt correct), request rows' `tmdb_id`.

### Rulings (party-mode 2026-07-06 — do not re-litigate)

- String ids everywhere on the media axis; PK migration to numeric REJECTED.
- Same-PR atomicity: BE payload type change + FE consumer change ship together (no compatibility shim, no dual-type acceptance — single-user NAS, no rolling deploy).
- The held `feat/ux3-subtitle-v2-batch` branch self-adapts at rebase (drops conversions, re-acks v2) — leave its files alone here.
- 9R-10's transcribe route carries NO contract stamp (verified) → direct fix, no bump. The SSE `transcription_*` payloads likewise unstamped → this story STAMPS them [@contract-v1] as part of AC 3 (first formalization, consumers: slice-1 + held batch branch).

### Project Structure Notes

- BE: `apps/api/internal/{handlers,services}` + tests. FE: 4 slice-1 files + specs. No migrations, no new routes, no .pen, no visual-baseline changes expected (no rendered output changes — ids aren't displayed).

### Time-dependent visual coverage

- N/A — no wall-clock-reading components touched (type-level changes only; no new rendering).

### References

- [Source: party-mode ruling 2026-07-06 (this session); ux3-subtitle-v2-batch CR finding H2]
- [Source: sprint-status.yaml disc-2026-07-movie-id-int64-contract-mismatch (filed by the batch CR)]
- [Source: _bmad-output/implementation-artifacts/9R-16-batch-generation-endpoint.md (stamps being bumped)]
- [Source: project-context.md Rules 6/7/15/16/20/24; retro-8-TD4 (prod UUID data)]

## Dev Agent Record

### Agent Model Used

Claude Fable 5 (claude-fable-5) — Amelia (BMAD DEV), 2026-07-06.

### Debug Log References

- `go vet ./...` chased the compile-error trail after the map/signature flips — every int64 caller surfaced and was converted (no grep-and-hope).
- Full `go test ./...` green; `go test -race` on `internal/{handlers,services}`: handlers clean; services' ONLY race is the pre-existing exempt `disc-2026-07-retry-mock-race` (`MockRetryRepository` in retry_service_test.go — untouched, not chased). Touched-scope `-race -run 'Transcription|GenerationBatch|Translate|RouteC|ResolveBudget'` fully clean.
- `pnpm nx test web`: 223 files / 2399 tests green (full suite, no flake retry needed). `pnpm lint:all`: 0 errors (122 pre-existing warnings). Both builds green. Prettier clean.

### Completion Notes List

- **AC 1/2:** `transcription_handler.go` ParseInt dropped — `:id` is opaque, non-empty-only validation (400 branch kept defensively; gin routing can't produce an empty `:id`, so the httptest exercises it via a hand-built context). All 8 service signatures + `inProgress map[string]string` + handler interface flipped; both `strconv.FormatInt` shims (glossary lookup :157, writeback :446) deleted — the string id flows end-to-end. No swagger annotations exist on the transcribe route (apps/api has no swag-gen — 9R-16 precedent); the route comment documents the string id.
- **AC 3:** all 6 emission sites now emit the UUID string `media_id` by construction (they broadcast the `mediaID string` param). SSE const block stamped `[@contract-v1]` (first formalization). `generation_batch_progress` `current_media_id` → string ([@contract-v2] comment).
- **AC 4:** `MediaIDs []string` body + interface + orchestrator (`Start`/`collectItems` `[]string`); `toItem()` ParseInt-skip DELETED (the silent UUID-dropper) — a UUID row with a file path now always enumerates; only file-less rows skip. Rule 20 bump done in 9R-16.md: AC 1/2/3/7/9 v1→v2, NEW Change Log section (one row per bumped AC + explicit AC-12-not-bumped note — its `UpdateSubtitleStatus` id param was already string). Consumer grep: ONE not-done consumer `ux3-subtitle-v2-batch` — sprint-status stale-mark verified pre-written (idempotent), ⚠️ STALE line ADDED to that story's Dev Notes (was not present).
- **AC 5:** `useGenerationProgress` `mediaId?: string` / `mediaIdRef<string|null>` / `startTracking(mediaId: string)` — strict string equality filter; `transcriptionService.startTranscription(movieId: string)`; `ManageSubtitleDialogV2` both `Number(mediaId)` calls removed (prop already string); `LocalDetailV2` verified pass-through only (`mediaId={id}` at :271/:280 — untouched).
- **AC 6 (integration guard):** NEW `handlers/route_c_uuid_integration_test.go` — real sqlite + full production migrations + real repos + REAL creation path (`LibraryService.SaveMovieFromTMDb` → `uuid.New().String()`), then over HTTP: transcribe NOT-400 (503-or-202 per availability; local run took the 202 full-pass — ffmpeg present), preview `total_items==1`, batch start 202 with `items[0].media_id == <minted uuid>` AND the runner received the UUID. Note: `services.NewGenerationBatchProcessor` accepts the stub runner cross-package (unexported-interface param, structural satisfaction) — no test-only export needed.
- **AC 7:** UUID-fixture convention comment added to every touched spec (Go: generation_batch_test / transcription_generation_test / transcription_service_test / transcription_translation_test / transcription_handler_test / generation_batch_handler_test / integration guard; FE: useGenerationProgress.spec / transcriptionService.spec / ManageSubtitleDialogV2.spec). Route-C-scope sweep only, per AC.
- **AC 8:** diff-review clean — zero `tmdb_id`/`owned_ids`/`external_id` lines in the code diff (only a sprint-status description mention). Glossary API untouched.
- **Behavior-note:** `generation_batch_test` "SkipsMalformedRows" was renamed/inverted to `UUIDIDsEnumerated` — the "non-numeric id skip" it asserted IS the bug this story deletes; UUID rows with files must enumerate.

### Discovery Triage

- No new discoveries. Pre-existing `disc-2026-07-retry-mock-race` reconfirmed under `-race` (exempt, already filed by the 9R-16 CR — no action). No new error codes (Rule 7: none expected, none added).

### Code Review Record (AI) — 2026-07-06, adversarial CR (same session)

**Verdict: APPROVED-WITH-FIXES-APPLIED** (1M/1L fixed, 1L accepted-open, 2 INFO). Rule 7 PASS (only pre-existing `VALIDATION_INVALID_FORMAT`/`TRANSCRIPTION_DISABLED` in diff — no new codes). Rule 20 PASS (5 v2 stamps verified; Change Log rows two-stage-verified; AC-12-not-bumped note present; CR re-ran the consumer grep — exactly ONE not-done consumer `ux3-subtitle-v2-batch`, stale-marked in BOTH its Dev Notes and sprint-status; `9-2a`/`ux3-subtitle-v2` are done → frozen, no retro-mark). Rule 25 N/A-PASS (project-context.md untouched, correctly). File List = exact match vs `git diff --name-only` (21/21).

- **[M1, FIXED]** `route_c_uuid_integration_test.go` — the 202 transcribe leg spawned the DETACHED async pipeline goroutine (context.Background + 5-min service timeout, ffmpeg child on the fake mkv) with NO drain; it could outlive the test, race `t.TempDir()`/db cleanup, and pollute the package run. Fix: bounded single-flight-slot drain after leg 1 (202-gated; `require.False` on timeout). Verified 3× with ffmpeg present (drain ~ms — extract fails fast) — hermeticity: WhisperClient("test-key") is unreachable (extraction always fails on the 8-byte fake), sseHub nil-guarded, subtitleWriter unwired.
- **[L2, FIXED]** 9R-16.md Change Log rows lacked the canonical greppable `[@contract-v1→v2] AC #N:` per-row token (Rule 20's two-stage verify keys on `AC #N:`; the token lived only in the preamble). Fix: date + bump token prepended to each of the 5 rows; header cell renamed. Substance (both sub-tokens per row) was already compliant.
- **[L3, ACCEPTED-OPEN]** `transcription_handler.go` empty-id 400 branch is unreachable through gin routing (a `:id` segment cannot match empty; `/movies//transcribe` never reaches the handler) — defensive keep is justified, and both the handler comment and the hand-built-context test document exactly this. No action.
- **[INFO]** Full `go test ./...` flaked on runs 1–2 on `TestScannerService_SSEBroadcast_ScanCancelled` — the already-filed `preexisting-fail-scanner-sse-scan-cancelled-flake` (passes 5/5 isolated; full-suite run 3 green). Not chased. The Dev log's "no flake" was that run's luck, not a contradiction.
- **[INFO]** `pnpm lint:all` 0 errors / 123 warnings (Dev logged 122) — both sampled warnings are in files this story never touched; count drift is pre-existing.

**CR-verified adversarial sweeps:** residual-numeric grep clean (only comments + `ConfirmAll` int64 row-COUNT returns + PosterCard's correct tmdb-axis `Number`); all 6 SSE emission sites broadcast the string `mediaID` param by construction; FE filter is strict string `!==` (`useGenerationProgress.ts:180`), no `==` coercion anywhere on the media axis; collateral byte-check clean (zero `tmdb_id`/`owned_ids`/`external_id` code lines in diff; glossary files untouched); transcribe guard is non-tautological (`NotEqual 400` + `Contains {202,503}` + 503-body must be `TRANSCRIPTION_DISABLED` — a 400 regression fails it); single-flight semantics type-only (same-media 409 both directions tested; leading-zero int collision class actually REMOVED by stringification); batch: missing-scope enumerates only repo movie rows, selected-scope rejects unknown/file-less ids before any `RunTranscription`.

**Post-fix gates:** integration guard 3×/3 green (202 path, ffmpeg present); `go test ./internal/{handlers,services}` green; `-race` handlers full + services touched-scope clean (retry-mock race exempt, not hit in scope); full `go test ./...` green (run 3; known scanner flake noted); `pnpm nx test web` 2399/2399; `pnpm lint:all` 0 errors; prettier clean; gofmt clean.

### File List

- apps/api/internal/handlers/transcription_handler.go (modified)
- apps/api/internal/handlers/transcription_handler_test.go (modified)
- apps/api/internal/handlers/generation_batch_handler.go (modified)
- apps/api/internal/handlers/generation_batch_handler_test.go (modified)
- apps/api/internal/handlers/route_c_uuid_integration_test.go (NEW — AC 6 guard)
- apps/api/internal/services/transcription_service.go (modified)
- apps/api/internal/services/transcription_service_test.go (modified)
- apps/api/internal/services/transcription_generation_test.go (modified)
- apps/api/internal/services/transcription_translation_test.go (modified)
- apps/api/internal/services/generation_batch.go (modified)
- apps/api/internal/services/generation_batch_test.go (modified)
- apps/web/src/hooks/useGenerationProgress.ts (modified)
- apps/web/src/hooks/useGenerationProgress.spec.ts (modified)
- apps/web/src/services/transcriptionService.ts (modified)
- apps/web/src/services/transcriptionService.spec.ts (modified)
- apps/web/src/components/subtitle/ManageSubtitleDialogV2.tsx (modified)
- apps/web/src/components/subtitle/ManageSubtitleDialogV2.spec.tsx (modified)
- _bmad-output/implementation-artifacts/9R-16-batch-generation-endpoint.md (Rule 20 bump + Change Log)
- _bmad-output/implementation-artifacts/ux3-subtitle-v2-batch.md (STALE mark in Dev Notes)
- _bmad-output/implementation-artifacts/sprint-status.yaml (9R-18 status; stale/resolved-by lines pre-written)
- _bmad-output/implementation-artifacts/9R-18-media-id-string-contract.md (this file)
