# Story 9R-15 — Glossary HTTP API (REST surface for the F6 review UI)

Status: review

**Epic:** epic-9R-subtitle-route-c · **Owner:** dev (Amelia) · **Date:** 2026-07-05
**Origin:** Rule-24 ③ from 9R-UX create-story — the F6 名詞對照表 UI (9R-UX AC4) had no HTTP surface
(9R-6/7 shipped migration + repo + service-side consumption only). **Depends:** 9R-6 ✅ / 9R-7 ✅.

## Why

The glossary is stored (9R-6) and CONSUMED by generation (9R-10) + .nfo localization (9R-13), but
the F6 review loop — where the user confirms/corrects auto-mined terms — needs a REST surface. This
story exposes `GlossaryRepository` over HTTP so the UI (and the pipeline, and the localizer) all
share one glossary keyed by the same media id.

## What shipped

### Routes (`internal/handlers/glossary_handler.go`)
Mounted at `/api/v1/media/:mediaId/glossary` — `mediaId` is the **local movie/series id**, the same
key `9R-10` (pipeline) and `9R-13` (localizer) use, so UI ↔ pipeline ↔ REST share one glossary.

| Verb | Path | F6 action |
|---|---|---|
| GET | `/media/:mediaId/glossary` | list (`{data:{terms:[]}}`, never null) |
| POST | `/media/:mediaId/glossary` | 新增 (upsert; 201) |
| PUT | `/media/:mediaId/glossary/:termId` | 編輯 (term_zh + confirmed; 204) |
| POST | `/media/:mediaId/glossary/:termId/confirm` | 確認 (204) |
| POST | `/media/:mediaId/glossary/confirm-all` | 全部確認 (`{data:{confirmed:N}}`) |
| DELETE | `/media/:mediaId/glossary/:termId` | 刪除 (204) |

**Security:** the route `mediaId` is authoritative on Add — a body `media_id` can't cross-write
another show's glossary.

### Service (`internal/services/glossary_service.go`)
`GlossaryService` (Rule 4 Handler→Service→Repository) with input validation; wraps the repo.

### Repository
Added `GlossaryRepository.ConfirmAll(ctx, mediaID) (int64, error)` — one `UPDATE … WHERE
media_id=? AND confirmed=0`, returns rows changed (F6 「全部確認」).

### Rule 7
**No new prefix.** Reuses existing wire codes: `VALIDATION_ERROR` / `VALIDATION_INVALID_FORMAT`
(bad input), `DB_NOT_FOUND` (unknown term), `INTERNAL_ERROR` — so no code-review CR-sync.

## Acceptance Criteria (from the 9R-15 backlog entry)

1. ✅ REST CRUD for per-media glossary terms (list/add/edit/confirm/delete).
2. ✅ **全部確認** batch-confirm (the F6 verb the repo lacked) — new `ConfirmAll`.
3. ✅ Gates the Epic 6 (ux3-subtitle-v2) FE glossary section — the surface it calls now exists;
   verbs match the F6 design (確認/全部確認/編輯/刪除/新增).
4. ✅ Tests: repo `ConfirmAll` (only unconfirmed flip, other media untouched); handler routes via
   httptest (list/never-null, add-route-media-wins, confirm-all, edit+delete, not-found→DB_NOT_FOUND,
   validation→VALIDATION_ERROR).

## Dev Notes

- Glossary population/mining (auto-extract terms during a run) is still a separate concern — this
  is the human review + edit surface. When a mining step lands, the auto-mined terms arrive
  `unconfirmed` and this API is how the user reviews them.
- Media-keyed by local id, not `/{movies|series}/:id` — the glossary is movie/series-agnostic
  (media_id string), so one route tree serves both; the Epic 6 FE passes the detail page's id.

### Discovery Triage

- **N/A — no out-of-scope work discovered.** Closes the 9R-UX ③ (glossary HTTP surface).

### References

- [Source: subtitle-route-c-stories-2026-06.md / 9R-UX Discovery ③] — origin.
- [Source: internal/repository/glossary_repository.go] — the repo exposed (+ ConfirmAll).
- [Source: internal/handlers/request_handler.go] — handler pattern mirrored.

## Dev Agent Record

### Agent Model Used

claude-fable-5 (dev)

### Completion Notes List

- Glossary REST surface (GlossaryHandler + GlossaryService + repo ConfirmAll) shipped: 6 routes
  under /media/:mediaId/glossary matching the F6 verbs; route media-id authoritative; no new Rule 7
  prefix. Full suite + staticcheck green. Unblocks the Epic 6 FE glossary section.

### File List

- `apps/api/internal/handlers/glossary_handler.go` (+ test)
- `apps/api/internal/services/glossary_service.go`
- `apps/api/internal/repository/glossary_repository.go` (+ test) — ConfirmAll
- `apps/api/cmd/api/main.go` — wire handler
- `_bmad-output/implementation-artifacts/sprint-status.yaml`

## Change Log

| Date | Change |
|---|---|
| 2026-07-05 | 9R-15 implemented (dev): 6-route glossary REST surface (/media/:mediaId/glossary — list/add/edit/confirm/confirm-all/delete) + GlossaryService + repo ConfirmAll (F6 全部確認). Route media-id authoritative; reuses existing Rule 7 codes (no CR-sync). Tests: repo + handler httptest. Full suite + staticcheck green. Closes 9R-UX ③. Status → review. |
