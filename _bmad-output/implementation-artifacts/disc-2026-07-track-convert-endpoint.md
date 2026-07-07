# Story disc-2026-07-track-convert-endpoint: Convert an existing local 簡中 subtitle track to 繁中 (standalone OpenCC endpoint)

Status: done

<!-- Rule-24 ③ discovery from ux3-subtitle-v2 (DEV Amelia, 2026-07-05). BACKEND-ONLY. -->

> **Origin (Rule 24 ③):** The F1 (`ManageSubtitleDialogV2`) design draws a `轉為繁中（簡轉繁）` action (+ CN-variant `仍要轉換` override, `.pen` note v16pVI) on an EXISTING local 簡中 track. It shipped as a capability-honored INFO line only, because **OpenCC s2twp runs solely inside the download post-process (`subtitle_handler.go` DownloadSubtitle) and the Route C generation pipeline (`transcription_service`) — there is NO standalone convert-an-existing-track route.** This story adds that endpoint. It is BACKEND-ONLY; the FE button that consumes it is a downstream follow-up (see Discovery Triage — it is additionally coupled to `disc-2026-07-production-countries-detail-api` for the CN warning display).

## Story

As a Vido user who already has a 簡體中文 subtitle sitting next to a movie,
I want a one-click "轉為繁中" that converts that existing local track to 繁體中文 via the same OpenCC s2twp pipeline the download/generation flows use,
so that I get a proper zh-Hant sidecar without re-downloading or re-generating a subtitle I already have.

## Acceptance Criteria

1. **New endpoint `POST /api/v1/subtitles/convert`.** Registered on the EXISTING `SubtitleHandler` (`internal/handlers/subtitle_handler.go`) inside its `/subtitles` route group (`RegisterRoutes`). No new handler struct, no new constructor dependency — the handler already holds `converter *subtitle.Converter`, `placer *subtitle.Placer`, `movieRepo`/`seriesRepo` (`subtitle.SubtitleStatusUpdater`), and `sseHub`. No `main.go` change required beyond the route already living inside `RegisterRoutes`. [@contract-v1]

2. **Request body (snake_case, Rule 6 — mirrors `SubtitleDownloadRequest`):** [@contract-v1]
   ```json
   {
     "media_id":        "<opaque string, required>",
     "media_type":      "movie" | "series",   // required, binding oneof
     "media_file_path": "/abs/path/Movie.2024.1080p.mkv",  // required
     "source_language": "zh-Hans"             // optional, default "zh-Hans"
   }
   ```
   `media_file_path` is provided by the client exactly as the existing `DownloadSubtitle` flow does (the local-detail payload exposes `movies.file_path` → `json:"file_path"`, so the FE already has it). Bind with `c.ShouldBindJSON`; on bind failure → `ValidationError` (400 `VALIDATION_*`). `media_type` NOT in `{movie, series}` → 400.

3. **Locate the source track.** From `media_file_path` + `source_language`, resolve the sidecar the placer would have written: `{name-without-media-ext}.{source_language}.srt` (and `.ass`). The source file MUST be found on disk next to the media file; probe `.srt` then `.ass`. If neither exists → 404 `SUBTITLE_NOT_FOUND` (`該語言的字幕檔不存在`). Path handling reuses the placer's safety posture: `media_file_path` must be absolute and `filepath.Clean`-ed; the resolved source path must remain within the media directory (reject traversal). Do NOT trust `source_language` as a raw path segment — it flows into a filename, so validate it against `subtitle.safeTagPattern`-equivalent (alphanumeric + hyphen) or map it through `normalizeLanguageTag` semantics before building the name.

4. **Convert + place (reuse the pipeline, do NOT reinvent).** Read the source bytes → `subtitle.Detect(data)`:
   - If detected `LangTraditional` (`zh-Hant`) → 409 `SUBTITLE_ALREADY_TRADITIONAL` (`字幕已是繁體中文，無需轉換`). No-op guard (idempotent; the user asked to convert a 簡中 track — if it's already 繁中 there is nothing to do).
   - If `LangSimplified` (`zh-Hans`) or `LangAmbiguous` (`zh`) → convert via the EXISTING `h.converter.ConvertS2TWP(data)`. Guard `h.converter != nil && h.converter.IsAvailable()` first; if OpenCC is in degraded mode → 503 `SUBTITLE_CONVERT_UNAVAILABLE` (`繁簡轉換服務目前無法使用`). On `ConvertS2TWP` error → 500 `SUBTITLE_CONVERT_FAILED`.
   - Otherwise (`und` / non-Chinese) → 400 `SUBTITLE_NOT_SIMPLIFIED` (`該字幕不是簡體中文，無法轉換`).
   Then place via the EXISTING `h.placer.Place(subtitle.PlaceRequest{MediaFilePath, SubtitleData: converted, Language: subtitle.LangTraditional, Format: <source ext w/o dot>})` → writes `{name}.zh-Hant.srt` next to the media file (the placer backs up any pre-existing `zh-Hant` sidecar to `.bak` automatically). On placer error → 500 `SUBTITLE_PLACE_FAILED`. **[@contract-v1]** (the error-code set 503/404/409/400/500 is part of the wire contract the FE consumer acks).

5. **Non-destructive.** The output filename (`.zh-Hant.srt`) differs from the source (`.zh-Hans.srt`), so the source 簡中 file is LEFT IN PLACE — the item ends with both tracks. Do NOT delete the source. (A future "replace/cleanup" behaviour is explicitly out of scope.)

6. **Persist status — pointer flip ONLY.** On success, update the DB via the EXISTING `updateSubtitleDB(ctx, media_id, media_type, placeResult.SubtitlePath, placeResult.Language, score)` helper → `subtitle_status = found`, `subtitle_language = zh-Hant`, `subtitle_path = <new path>`. Use a nominal `score` (e.g. `1.0` — a user-confirmed local track). A DB-update failure is NON-fatal (file is already placed): log via slog and still return success, exactly as `DownloadSubtitle` does. **Do NOT hand-edit the `subtitle_tracks` JSON blob** (the ffprobe/external-scan column) — `UpdateSubtitleStatus` does not touch it, and neither does `DownloadSubtitle`; a subsequent library rescan reconciles that JSON. The new 繁中 track surfaces to the FE immediately via the flipped `subtitle_status==='found'` pointer (the dialog's `buildTrackRows` renders a "字幕引擎" row from `subtitle_status`/`subtitle_language`), so no blob rewrite is needed. Rewriting the blob is out of scope and error-prone — do not.

7. **CN policy is NOT enforced server-side (explicit-action = intent).** An explicit convert click IS the `仍要轉換` override — the endpoint converts unconditionally and does NOT read `production_countries` / apply the §9b `ConvertNever` skip (that policy governs AUTOMATIC conversion during download/generation, not a user's explicit request). Therefore this endpoint has NO dependency on `disc-2026-07-production-countries-detail-api`; the CN warning stays a FE display concern.

8. **Synchronous, no SSE, no job.** Single-file OpenCC on one SRT is a sub-second in-memory transform → respond `200` synchronously with `{"subtitle_path": "<abs>", "language": "zh-Hant", "source_path": "<abs>"}` (snake_case, wrapped in the standard `{success, data}` envelope via `SuccessResponse`). Do NOT spawn a background job or emit `subtitle_progress` SSE — the FE flips the badge by refetching after the 200 (same as the download flow's DB-write → invalidate pattern). **[@contract-v1]** (this 200 response shape is part of the wire contract the FE consumer acks).

9. **Tests + gates.** Colocated `subtitle_handler_test.go` cases covering: (a) happy path zh-Hans→zh-Hant (file written with correct name + content converted + DB updated); (b) ambiguous `zh` also converts; (c) source not found → 404 `SUBTITLE_NOT_FOUND`; (d) already-traditional → 409 `SUBTITLE_ALREADY_TRADITIONAL`; (e) non-Chinese → 400 `SUBTITLE_NOT_SIMPLIFIED`; (f) converter unavailable → 503 `SUBTITLE_CONVERT_UNAVAILABLE`; (g) placer error → 500 `SUBTITLE_PLACE_FAILED`; (h) bad body / bad `media_type` → 400; (i) path-traversal `source_language` / non-absolute `media_file_path` rejected. Use a real `subtitle.Converter` (OpenCC is pure-Go, no external binary) and a temp dir for placement (mirror `placer_test.go`). `go test ./...`, `go vet ./...`, `staticcheck ./...` (via `pnpm lint:all`) all green.

## Tasks / Subtasks

- [x] **Task 1: Convert handler method + route + DTO** (AC: 1, 2, 8)
  - [x] Add `SubtitleConvertRequest` DTO (snake_case, `binding` tags mirroring `SubtitleDownloadRequest`) in `subtitle_handler.go`.
  - [x] Add `ConvertSubtitle(c *gin.Context)` method; register `subtitles.POST("/convert", h.ConvertSubtitle)` in `RegisterRoutes`.
  - [x] Return `200` `{subtitle_path, language, source_path}` via `SuccessResponse`.
- [x] **Task 2: Source-track resolution** (AC: 3)
  - [x] Helper to build `{name}.{source_language}.{srt|ass}` from `media_file_path` — reused via NEW exported `subtitle.BuildSubtitleFilename` + `subtitle.NormalizeLanguageTag` (thin wrappers over the private placer helpers; no logic duplicated).
  - [x] Absolute-path + `filepath.Clean` guard on `media_file_path`; `NormalizeLanguageTag` canonicalizes + collapses unsafe tags to `und` (path-traversal guard); resolved path stays within the media dir.
  - [x] Probe `.srt` then `.ass`; 404 `SUBTITLE_NOT_FOUND` if neither.
- [x] **Task 3: Convert + place + persist** (AC: 4, 5, 6, 7)
  - [x] `subtitle.Detect` guard branches (already-traditional 409 / not-simplified 400 / converter-unavailable 503).
  - [x] `h.converter.ConvertS2TWP` → `h.placer.Place` (Language `zh-Hant`, Format from source ext) → `updateSubtitleDB` (non-fatal on error).
  - [x] New error codes `SUBTITLE_CONVERT_UNAVAILABLE`, `SUBTITLE_ALREADY_TRADITIONAL`, `SUBTITLE_NOT_SIMPLIFIED` under the EXISTING `SUBTITLE_` prefix (no Rule 7 prefix-set change / no CR-workflow sync).
- [x] **Task 4: Tests** (AC: 9)
  - [x] `subtitle_handler_test.go` cases (a)–(i) above; real converter + temp dir (9 new tests, all green).
  - [x] `nx test api` full backend suite green; `pnpm lint:all` (go vet + staticcheck + eslint + prettier) → 0 errors.

**Cross-stack split check:** backend tasks = 4, frontend tasks = 0 → single story, no a/b split.

## Dev Notes

### Backend surface (code-verified 2026-07-07 — do NOT re-derive)

**Reuse map (everything the endpoint needs already exists on `SubtitleHandler`):**
- `internal/subtitle/converter.go` — `Converter.ConvertS2TWP(content []byte) ([]byte, error)` (s2twp; idempotent on already-繁 text; graceful-degrades, returns original + error on failure). `Converter.IsAvailable() bool` (false when OpenCC init failed — degraded mode). `subtitle.NeedsConversion(lang) bool` = `lang == LangSimplified` only.
- `internal/subtitle/detector.go` — `subtitle.Detect(data) → {Language}`; language constants `LangTraditional = "zh-Hant"`, `LangSimplified = "zh-Hans"`, `LangAmbiguous = "zh"`, `LangUndetermined = "und"`. **Detection is content-based (Bazarr fix) — trust `Detect`, not the filename tag.**
- `internal/subtitle/placer.go` — `Placer.Place(PlaceRequest{MediaFilePath, SubtitleData, Language, Format, Score}) → (*PlaceResult{SubtitlePath, Language, BackupPath}, error)`. Naming: `Movie.2024.mkv` + `zh-Hant` → `Movie.2024.zh-Hant.srt` (`buildSubtitleFilename`, UNEXPORTED). Placer guards absolute-path + dir-exists + path-traversal, backs up existing target `.bak`, writes atomically. `normalizeLanguageTag` + `safeTagPattern` (UNEXPORTED) sanitize tags.
- `internal/handlers/subtitle_handler.go` — the home. `DownloadSubtitle` (line ~206) is the EXACT template: takes `media_file_path` in the body, `Detect` → `shouldConvert` → `ConvertS2TWP` → `placer.Place` → `updateSubtitleDB`. `updateSubtitleDB(ctx, mediaID, mediaType, path, language, score) error` (line ~372) switches movie/series over `SubtitleStatusUpdater.UpdateSubtitleStatus`. `SubtitleDownloadRequest` (line ~107) is the DTO shape to mirror.
- `handlers/response.go` — `SuccessResponse`, `ErrorResponse(c, status, code, msg, suggestion)`, `BadRequestError`, `ValidationError`, `NotFoundError`, `InternalServerError`, `APIResponse`/`APIError`.

**Source-file tag tolerance + optional scan fallback.** Our OWN download/generation pipeline writes simplified sidecars as `.zh-Hans.srt` (via the placer's `normalizeLanguageTag`), so `source_language: "zh-Hans"` + constructed `{name}.zh-Hans.{srt|ass}` locates the common case deterministically. But a user's manually-dropped 簡中 file may be tagged `.chs.srt` / `.zh-CN.srt` / `.簡體.srt` (all mapped to `zh-Hans` by `normalizeLanguageTag`, and picked up by `DetectExternalSubtitles` which parses ANY `.{suffix}.{ext}`). Two acceptable strategies: (a) MINIMAL — trust the `source_language` the FE sends (it comes from the parsed track language) and build the name from it; 404 if not found. (b) ROBUST (optional enhancement) — if the constructed name misses, fall back to scanning the media dir (à la `DetectExternalSubtitles`) for a sidecar whose CONTENT `subtitle.Detect`s as `LangSimplified`. Either way, **the convert/guard decision (AC #4) is driven by `subtitle.Detect` on the file CONTENT, never the filename tag** (content-based detection is the whole point of the §9b engine — the filename can lie). Pick (a) for this story unless (b) is trivial; note the choice in Completion Notes.

**Alternative persistence primitive:** `subtitle.Manager.PlaceAndRecord(ctx, mediaID, mediaType, PlaceRequest)` (`internal/subtitle/manager.go:30`) does place+DB atomically with orphan-file cleanup on DB failure. `SubtitleHandler` does NOT currently hold a `*Manager` (only `placer` + `updateSubtitleDB`), so **stay with the handler's existing `placer.Place` + `updateSubtitleDB` pattern** (the `DownloadSubtitle` precedent) rather than injecting a new dep — the DB-write is non-fatal here anyway, so the atomic-compensation value is low.

**Handler construction (already wired — `main.go:633`):**
```go
subtitleHandler := handlers.NewSubtitleHandler(
    subtitleProviders, subtitleScorer, subtitleConverter, subtitlePlacer,
    sseHub, repos.Movies, repos.Series,
)
```
`subtitleConverter` = `subtitle.NewConverter()` (main.go:473). `subtitlePlacer` = `subtitle.NewPlacer(DefaultPlacerConfig())` (main.go:475, `BackupExisting:true`). **The convert endpoint adds ZERO wiring in `main.go` — just a new method + a route line inside the already-registered `RegisterRoutes`.**

**Media file path source:** `models.Movie.FilePath NullString `db:"file_path" json:"file_path,omitempty"`` (movie.go:138) — exposed on the local-detail payload, so the FE has it (this is what `DownloadSubtitle` already relies on). Movie IDs are opaque strings (UUIDs, 9R-18) — do not assume int64 (transcribe treats `:id` as opaque string, `transcription_handler.go:59`). This endpoint takes `media_id` in the BODY (not a path param), so id-type is a non-issue.

### Design decision: why `POST /subtitles/convert` (body-based) over `POST /movies/:id/subtitles/convert` (id-based)

Chosen to MIRROR the sibling `DownloadSubtitle` endpoint exactly: same handler, same `/subtitles` group, same body carrying `media_id` + `media_type` + `media_file_path`, same `converter → placer → updateSubtitleDB` flow. This keeps the diff to one method + one route line with zero new constructor dependencies and zero `main.go` change, and it handles movie AND series through one path (the id-based `/movies/:id/...` form would need a series twin + a movie/series file-path getter injected into `SubtitleHandler`, which it does not currently hold — it only has `SubtitleStatusUpdater`, not a full repo). The discovery note's illustrative `POST /api/v1/media/{id}/subtitles/convert` was a shape suggestion, not a contract. **This decision is surfaced for ratification at story hand-off (see Story Author Questions).**

### Error codes (Rule 7)

All codes are under the **already-registered `SUBTITLE_` prefix** — reused `SUBTITLE_NOT_FOUND`, `SUBTITLE_CONVERT_FAILED`, `SUBTITLE_PLACE_FAILED` (all live in code today) + three NEW codes under the SAME prefix: `SUBTITLE_CONVERT_UNAVAILABLE` (503), `SUBTITLE_ALREADY_TRADITIONAL` (409), `SUBTITLE_NOT_SIMPLIFIED` (400). **Adding codes under an EXISTING registered prefix does NOT require a `project-context.md` prefix-set edit or a `code-review/instructions.xml` Step 3 sync** — that obligation is only for NEW prefixes (per the Rule 7 authoritative-prefix-set note, prefix count stays 16). No mega-line (Rule 25) edit either — no prefix added.

### Architecture compliance

- Rule 1: all code in `/apps/api`. Rule 2: `slog` only. Rule 3: `{success, data}` / `{success, error:{code,message,suggestion}}`. Rule 4: Handler → (subtitle pkg + repo); the subtitle package IS the "service" layer here (converter/placer are domain services), consistent with how `DownloadSubtitle` is structured. Rule 6: snake_case JSON, `/api/v1` path, plural resource. Rule 10: `/api/v1/subtitles/convert`. Rule 13: propagate/log-and-halt every error; the ONLY intentionally-non-fatal path is the post-place DB update (documented, mirrors `DownloadSubtitle`). Rule 15: no migration, no new model field, no new route-vs-client drift (single new route); Swagger — `apps/api` has no swag generation (consolidation P1.2 pending), so no annotation regen (matches 9R-16 closeout).
- Rule 27 (External Integration): N/A — OpenCC is an in-process pure-Go library, not a network upstream. No rate limit / cache / key concerns.
- Idempotency: re-running convert on an item that already has a `zh-Hant` sidecar re-backs-up (`.bak`) and rewrites it — acceptable (placer's documented behaviour). The `SUBTITLE_ALREADY_TRADITIONAL` guard fires only when the SOURCE (`zh-Hans`) track's content is detected traditional, not when a separate `zh-Hant` sidecar exists.

### Project Structure Notes

- Touch ONLY: `internal/handlers/subtitle_handler.go` (+ `subtitle_handler_test.go`), and OPTIONALLY tiny exported helpers in `internal/subtitle/placer.go` (e.g. export `BuildSubtitleFilename` / a `SafeLanguageTag` validator) if the handler should reuse the naming logic instead of duplicating it — prefer reuse. No migration, no new file, no `main.go` change, no frontend, no `.pen`.
- Naming: Go `snake_case.go`, `PascalCase` structs, `SubtitleConvertRequest` DTO.

### Time-dependent visual coverage

N/A — backend-only story; no `apps/web/src/components/**` files touched, no wall-clock reads.

### References

- [Source: apps/api/internal/handlers/subtitle_handler.go — `DownloadSubtitle` (~206) is the template; `RegisterRoutes` (~61); `updateSubtitleDB` (~372); `SubtitleDownloadRequest` (~107)]
- [Source: apps/api/internal/subtitle/converter.go — `ConvertS2TWP`, `IsAvailable`, `NeedsConversion`, `ProfileS2TWP`]
- [Source: apps/api/internal/subtitle/placer.go — `Place`, `PlaceRequest`/`PlaceResult`, `buildSubtitleFilename`, `normalizeLanguageTag`, `safeTagPattern`, `DefaultPlacerConfig{BackupExisting:true}`]
- [Source: apps/api/internal/subtitle/detector.go — `Detect`, `LangTraditional/Simplified/Ambiguous/Undetermined` constants]
- [Source: apps/api/internal/handlers/transcription_handler.go — opaque-string movie id (9R-18), file-path validation exemplar]
- [Source: apps/api/cmd/api/main.go:473-475, 633-636 — converter/placer construction + `NewSubtitleHandler` wiring]
- [Source: apps/api/internal/models/movie.go:138 — `FilePath` `json:"file_path"`]
- [Source: project-context.md §9b Subtitle Engine Pipeline (OpenCC s2twp, CN policy), Rule 7 (16-prefix set), Rules 1/3/4/6/13/27]
- [Source: _bmad-output/implementation-artifacts/ux3-subtitle-v2.md — Discovery Triage (origin), Sally gate note on VXof3 filename/⋯ menu]
- [Source: sprint-status.yaml entries `disc-2026-07-track-convert-endpoint` (this), `disc-2026-07-production-countries-detail-api`, `chore-pen-subtitle-v2-design-sync` (party-mode ruling: John — BE unscheduled 2-3 sprints; filename-keep/menu-drop)]

## Dev Agent Record

### Agent Model Used

Claude Opus 4.8 (claude-opus-4-8[1m]) — DEV Amelia, 2026-07-07

### Debug Log References

- New convert tests (targeted): `go test ./internal/handlers/ -run TestSubtitleHandler_Convert -v` → 10/10 PASS (cases a–i + series).
- Full backend suite: `pnpm nx test api` → **PASS**; post-CR `go test ./...` (apps/api) → 34 pkgs ok / 0 FAIL.
- `pnpm lint:all` → **0 errors** (125 pre-existing `apps/web` warnings, none introduced — this story adds 0 TS); go vet + staticcheck clean; `prettier --check .` clean.
- `go vet ./internal/handlers/ ./internal/subtitle/` → exit 0.

### Completion Notes List

- Implemented ratified **contract A**: `POST /api/v1/subtitles/convert` on the EXISTING `SubtitleHandler` — new `SubtitleConvertRequest` DTO + `ConvertSubtitle` method + one route line in `RegisterRoutes`. **Zero `main.go` change, zero new constructor dependency** (handler already holds `converter`/`placer`/`movieRepo`/`seriesRepo`).
- Reuse over reinvention: converts via `h.converter.ConvertS2TWP`, places via `h.placer.Place`, persists via the existing `h.updateSubtitleDB` — same flow as `DownloadSubtitle`. Source-track path resolved through NEW exported `subtitle.BuildSubtitleFilename` + `subtitle.NormalizeLanguageTag` (thin wrappers over the private placer helpers — no logic duplicated, per Task 2's reuse mandate).
- **Content-based decision, not filename:** the convert/guard branch keys on `subtitle.Detect(data).Language` (zh-Hant→409, zh-Hans/zh→convert, else→400) — a `.zh-Hans.srt` file that actually contains Traditional text correctly yields 409 (test `_AlreadyTraditional`).
- **Non-destructive** (AC #5): output is `{name}.zh-Hant.{ext}`, a different filename from the `.zh-Hans` source, so the source is preserved (verified in `_Success`). Placer backs up any pre-existing zh-Hant to `.bak`.
- **Pointer-flip only** (AC #6): `updateSubtitleDB` flips `subtitle_status=found`/`subtitle_language=zh-Hant`/`subtitle_path`; the `subtitle_tracks` JSON blob is deliberately untouched (rescan reconciles). DB write is non-fatal (logged, still 200).
- **CN policy not enforced server-side** (AC #7): explicit click = intent; no `production_countries` read → no dependency on `disc-2026-07-production-countries-detail-api`.
- **Rule 7:** 3 new codes (`SUBTITLE_CONVERT_UNAVAILABLE` 503, `SUBTITLE_ALREADY_TRADITIONAL` 409, `SUBTITLE_NOT_SIMPLIFIED` 400) + reused `SUBTITLE_NOT_FOUND`/`SUBTITLE_CONVERT_FAILED`/`SUBTITLE_PLACE_FAILED` — all under the ALREADY-REGISTERED `SUBTITLE_` prefix → prefix count stays 16, NO `project-context.md` prefix-set edit, NO `code-review/instructions.xml` sync, NO mega-line (Rule 25) edit.
- **Source-tag strategy chosen (a) — minimal:** builds the sidecar path from the (normalized) `source_language`, default `zh-Hans`; no dir-scan fallback (b) added (YAGNI — our pipeline writes canonical `.zh-Hans.srt`). Non-canonical user tags are handled by `NormalizeLanguageTag` mapping (zh-cn/chs/简体 → zh-Hans).
- Status codes use raw ints (503/404/409/400/500) to match the file's existing `ErrorResponse(c, 500, …)` style; no `net/http` import needed (added only `os` + `path/filepath`).
- 🔗 **AC Drift:** NONE (checked `grep -rn "subtitles/convert\|ConvertS2TWP\|convert.*existing.*track" _bmad-output/implementation-artifacts/*.md` — the only prior hits are this story + the `ux3-subtitle-v2` origin note; no prior AC ships a convert-existing-track contract — this is a new subsystem surface, all REUSE of the subtitle pipeline, no behavior change to a prior AC).
- 📎 **Contract Stamps:** FOUND (1 stamp: this story's `[@contract-v1]` on AC #1/#2 for the future FE consumer; no upstream stamped AC consumed — the reused subtitle/placer helpers are pre-Rule-20 internal code, not contract-stamped).
- 🎭 **A11y Pre-Flight:** N/A (100% backend — no `apps/web/` files touched).
- 🎨 **UX Verification:** SKIPPED — no UI changes in this story (backend-only; the FE `轉為繁中` button is a deferred downstream consumer, see Discovery Triage).
- **Full-regression note:** `pnpm nx test web` NOT run — this story touches zero `apps/web/` files (backend-only), so the web suite cannot be affected by the change (same rationale as the a11y-preflight 100%-backend carve-out). `nx test api` is the meaningful gate and is green.
- **Test-process cleanup:** N/A — no vitest/JS test workers spawned (Go `go test` only); no orphaned processes possible.

### Change Log

| Date | Change |
|------|--------|
| 2026-07-07 | Implemented `POST /api/v1/subtitles/convert` (contract A) — `SubtitleConvertRequest` DTO + `ConvertSubtitle` handler + route on existing `SubtitleHandler`; converts an existing local 簡中 sidecar → 繁中 via OpenCC s2twp, non-destructive, sync 200. |
| 2026-07-07 | Exported `subtitle.BuildSubtitleFilename` + `subtitle.NormalizeLanguageTag` (thin wrappers over private placer helpers) for cross-package sidecar-path resolution. |
| 2026-07-07 | Added 10 handler tests (cases a–i + series); `nx test api` green, `lint:all` 0 errors. Story → review. |
| 2026-07-07 | Adversarial CR: +`[@contract-v1]` on AC #4/#8 (full wire contract), +series-path test. `go test ./...` 34 pkgs ok. Story → done. |

### Senior Developer Review (AI) — adversarial CR (2026-07-07)

Reviewer: DEV Amelia (same-session adversarial pass, per user option 1). Outcome: **APPROVED-WITH-FIXES** — all fixes applied and re-verified green.

Gates: 🔒 Rule 7 Wire Format: **PASS** (all `SUBTITLE_*` / `VALIDATION_*` inline codes use registered prefixes; no constants, prefix count stays 16) · 🔒 Rule 20 Contract Bump: **N/A** (fresh `[@contract-v1]`, no bump) · 🔒 Rule 25 Mega-line: **N/A** (`project-context.md` untouched). No HIGH/CRITICAL. Task-vs-code audit: all `[x]` genuinely done. ACs #1–#9: all IMPLEMENTED.

Findings + resolution:

- **[MED fixed] Contract incompleteness (Rule 20).** The wire contract the future FE consumer acks spans the request (AC #2), the 200 response shape (AC #8), AND the error-code set (AC #4), but only AC #1/#2 were stamped. → Added `[@contract-v1]` to AC #4 (503/404/409/400/500 codes) and AC #8 (response shape).
- **[LOW fixed] `series` path untested.** `updateSubtitleDB`'s `series` branch had zero coverage (only `movie`). → Added `TestSubtitleHandler_Convert_Series` (asserts the `seriesRepo` pointer flip); green.
- **[LOW acknowledged — no code change] `media_id` ↔ `media_file_path` not cross-validated.** A bogus `media_id` silently no-ops the DB write (non-fatal) yet still returns 200; the endpoint trusts the body-supplied path + id and does no `GetByID` existence check. This EXACTLY mirrors the sibling `DownloadSubtitle` trust model (single-user app; both take `media_file_path` from the body without validating it against `media_id`). Left as-is for consistency; a future hardening story could add `GetByID`-based path resolution to BOTH endpoints together rather than diverging one.

Post-fix gates: `go test ./...` (apps/api) → 34 pkgs ok / 0 FAIL; 10/10 convert tests green; `go vet` clean; `pnpm lint:all` 0 errors; `prettier --check` clean.

### Discovery Triage

- **Downstream FE consumer — the `轉為繁中` button that calls this endpoint (out of scope, non-blocking).** This story ships the BE endpoint only. Wiring the F1 `ManageSubtitleDialogV2` convert action is a SEPARATE FE story because it is coupled to (a) the `.pen` convert affordance, currently "aspirational" / dropped per the `chore-pen-subtitle-v2-design-sync` party-mode ruling (2026-07-06), and (b) `disc-2026-07-production-countries-detail-api` for the CN `仍要轉換` warning display. Not filed as a new entry yet — the FE surface is design-blocked; file it when the `.pen` convert affordance is ratified. **[@contract-v1]** is stamped on AC #1/#2 so that future FE consumer acks the endpoint contract (Rule 20).
- **Cross-story relationship (Party Mode 2026-07-08) — shares the dialog surface with `disc-2026-07-production-countries-detail-api`.** This endpoint's future FE convert BUTTON and that story's §9b CN INFO LINE live in the SAME `ManageSubtitleDialogV2` surface but are INDEPENDENTLY shippable: the info line is un-design-blocked (wired by the production-countries story), the button stays design-blocked pending the `.pen` affordance. **No change to this (merged) endpoint** — its server-side independence from `production_countries` (explicit click = `仍要轉換` override) was re-affirmed correct by the team.
- (Dev: add any further in-flight discoveries here per Rule 24 before marking done.)

### File List

Modified:

- `apps/api/internal/handlers/subtitle_handler.go` — `SubtitleConvertRequest` DTO, `ConvertSubtitle` handler, `POST /subtitles/convert` route; imports `os` + `path/filepath`.
- `apps/api/internal/handlers/subtitle_handler_test.go` — 10 convert tests (cases a–i + series) + `setupConvertHandler`/`writeSidecar`/`doConvert`/`convErrCode` helpers; imports `path/filepath`.
- `apps/api/internal/subtitle/placer.go` — exported `NormalizeLanguageTag` + `BuildSubtitleFilename` wrappers.
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — `disc-2026-07-track-convert-endpoint` → `review`.
- `_bmad-output/implementation-artifacts/disc-2026-07-track-convert-endpoint.md` — this file (checkboxes, Dev Agent Record, Change Log, File List, Status).

## Story Author Decisions (ratified by Alexyu 2026-07-07)

1. **Endpoint shape → A (body-based `POST /api/v1/subtitles/convert`).** Mirrors `DownloadSubtitle`; zero new wiring. The RESTful `/media/:id/...` id-based form was NOT chosen. This is the locked [@contract-v1] shape (AC #1/#2) — do NOT re-litigate at dev time.
2. **Scope → BE-only** (endpoint + tests). FE `轉為繁中` button wiring stays deferred (design-blocked — see Discovery Triage). No cross-stack split.
