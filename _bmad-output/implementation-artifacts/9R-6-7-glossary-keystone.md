# Story 9R-6 + 9R-7 — Glossary keystone (schema + generalized translation)

Status: review

**Epic:** epic-9R-subtitle-route-c (Track 2 — Keystone) · **Owner:** dev (Amelia) · **Date:** 2026-07-05
**Priority:** P0 keystone · **Feasibility:** PROVEN (Route C POC)
**Shipped together** — 9R-7 depends on 9R-6's schema and both are small; one branch/PR.

## Why (ADR Decision 3)

The per-show glossary is the differentiator no OSS subtitle tool provides and the fix for
proper-noun drift the POC surfaced (the same title rendered 隱形戰士/隱形特務 across runs;
"The Deep" → 深海怪物 because the model lacked the character roster). One infra serves BOTH
subtitle translation (9R-7) AND `.nfo` metadata localization (9R-13).

## 9R-6 — `show_glossary` schema + repository

- **Migration 028** (`028_create_show_glossary_table.go`): `show_glossary(id, media_id,
  term_src, term_zh, language DEFAULT 'zh-Hant', source CHECK subtitle|metadata|manual DEFAULT
  manual, confirmed DEFAULT 0, created_at, updated_at)`; **UNIQUE(media_id, term_src,
  language)** + a `media_id` lookup index.
  - `media_id` = the **local movie/series id** (string; movie int ids stringified) — the
    identifier the generation pipeline (9R-10), the detail 管理字幕 surface, and the 9R-15 REST
    routes (`/{movies|series}/:id/glossary`) all hold. Matches the F6 design.
- **Model** `models.GlossaryTerm` + source constants + `Validate()`.
- **Repository** `GlossaryRepository` (registered in both `NewRepositories` constructors):
  `Upsert` (ON CONFLICT(media,term,language) updates in place — a re-mined term refreshes, no
  duplicate), `ListByMedia`, `LookupByMedia(confirmedOnly)` (returns the `term_src→term_zh` map
  the translation layer injects), `Update`, `Confirm`, `Delete`.
  - Real-sqlite integration tests via the production migration runner (Rule 15 — no hand-copied
    schema): upsert/list, conflict-updates-in-place, confirmed-only lookup filter,
    update/confirm/delete + not-found, validation.

## 9R-7 — generalize `TranslationService` → `TranslationRequest{Fields, Glossary}`

- **Generic carrier types** (`services`): `TranslationField{Key, Text}` (arbitrary keyed unit
  — a subtitle block or a metadata field like `plot`/`title`), `GlossaryPair{Source, Target}`,
  `TranslationRequest{Fields, Glossary}`.
- **Glossary-aware prompt** (`prompts`): `BuildGlossarySection` + `BuildSubtitleTranslatorPrompt
  WithGlossary` prepend a **MANDATORY fixed-rendering** block (`Source → Target`, do-not-
  retranslate). A nil/empty glossary yields a **byte-identical** prompt to the pre-9R-7 builder
  (no-regression, asserted).
- **Service methods:**
  - `Translate` (unchanged signature — back-compat) now delegates to
    `TranslateWithGlossary(..., nil, ...)`.
  - `TranslateWithGlossary` — the subtitle path with a glossary threaded into every batch prompt
    (batching + context-window preserved).
  - `TranslateRequest(ctx, TranslationRequest)` — the **generic single-batch** entry point for
    9R-13 metadata localization (a handful of named fields, no batching); fail-soft (a field
    with no returned translation keeps its original Text).
- **Tests:** glossary injected into subtitle prompt + blank-source filtered; no-glossary prompt
  unchanged; `TranslateRequest` honors glossary + preserves field keys + fail-soft; prompt-level
  section tests.

## Acceptance Criteria

**9R-6:** 1 ✅ migration adds `show_glossary` (media-keyed; term_src/term_zh/language/source/
confirmed/timestamps; unique on media+term+lang). 2 ✅ repository CRUD + lookup-by-media.
3 ✅ tests for migration + repo.

**9R-7:** 1 ✅ new generic request type carrying fields + glossary map; subtitle translation
refactored onto it (no behavior regression — existing translation tests green). 2 ✅ glossary
terms injected into the prompt (do-not-retranslate + use-this-rendering). 3 ✅ test: glossary
term honored in output.

## Dev Notes

- **Wiring the pipeline to LOAD + pass the glossary is 9R-10** (orchestration) — this story
  delivers the storage + the translation seam. `TranslateWithGlossary`/`TranslateRequest` are
  the entry points 9R-10 (subtitle) and 9R-13 (metadata) call after `GlossaryRepository.
  LookupByMedia`. See Discovery Triage.
- **9R-15** (glossary HTTP API, backlog) exposes `GlossaryRepository` over REST for the F6 UI —
  the repository is deliberately the shared layer both 9R-15 and the pipeline use.
- Rule 15 respected: `glossaryColumns` keeps INSERT/SELECT/scan in sync.

### Discovery Triage

- **YES — deferred consumers, all already tracked (no new entries):**
  - **① in-scope note (not a discovery):** the glossary is not yet *loaded* by any live flow —
    that wiring is **9R-10** (pipeline: mine terms → `LookupByMedia` → `TranslateWithGlossary`)
    and **9R-13** (metadata: `TranslateRequest`). Both are existing sprint-status stories; this
    story deliberately stops at the seam so 9R-10/13 don't block on it.
  - **③ already-filed:** `9R-15-glossary-http-api` (backlog, filed by 9R-UX) is the REST surface
    over this repository — unchanged.

### References

- [Source: architecture/adr-subtitle-route-c-generation.md#Decision-3] — glossary keystone.
- [Source: subtitle-route-c-stories-2026-06.md#9R-6,9R-7] — ACs.
- [Source: internal/database/migrations/027_create_requests_table.go] — migration + repo pattern.

## Dev Agent Record

### Agent Model Used

claude-fable-5 (dev)

### Completion Notes List

- 9R-6 + 9R-7 implemented and unit-tested; full api suite + staticcheck green. Glossary storage
  + generalized glossary-aware translation seam landed; pipeline/metadata consumers (9R-10/13)
  and REST surface (9R-15) call into it.

### File List

- `apps/api/internal/database/migrations/028_create_show_glossary_table.go` (+ test)
- `apps/api/internal/models/glossary.go`
- `apps/api/internal/repository/glossary_repository.go` (+ test)
- `apps/api/internal/repository/registry.go` (register GlossaryRepository)
- `apps/api/internal/ai/prompts/subtitle_translator.go` (+ test) — glossary section
- `apps/api/internal/services/translation_service.go` (+ test) — generic types + methods
- `_bmad-output/implementation-artifacts/sprint-status.yaml`

## Change Log

| Date | Change |
|---|---|
| 2026-07-05 | 9R-6 + 9R-7 implemented (dev): migration 028 + GlossaryTerm model + GlossaryRepository (upsert/list/lookup/update/confirm/delete, real-sqlite tests); TranslationRequest/Field/GlossaryPair types + glossary-aware prompt + TranslateWithGlossary/TranslateRequest (back-compat Translate unchanged). Full suite + staticcheck green. Status → review. |
