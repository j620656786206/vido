# Epic 20 — Test Quality Hardening (data-path coverage & masked-test cleanup)

**Status:** ready-for-dev (Story 20-1) · planned (20-2, 20-3)
**Owner:** SM Bob (prep) → Dev Amelia (impl) · TEA Murat (test policy) · ARCH Winston (rules)
**Origin:** Discovered 2026-06-14 during the Phase-2 v2 pilot's first real-data test
(seeded a local library on localhost) — the TV-detail season accordion rendered
empty. Root-cause analysis exposed a backend bug **and** a systemic test blind
spot that let it through.

## Why this epic exists

The season bug was not a one-off — it is a **symptom of a coverage hole**: the
project has **no E2E/integration test-data seeding**, so every test that needs
real data either (a) self-skips when the DB is empty, or (b) is permanently
skipped "until proper test data seeding." CI runs against an empty DB, so an
entire layer — **the real DB read/write data path** — is invisible to the suite.
"Tests green ≠ feature works" (brief P10) made concrete.

### Audit evidence (2026-06-14 skipped-test sweep)

| Finding | Where | Risk | Class |
|---|---|---|---|
| **Season-summary endpoint reads a dead column** | `series_season.go:56 GetSeasons` → `FindByID().GetSeasons()` reads `series.seasons` JSON, but `seriesSelectColumns`/`scanSeries` never select/scan it. The real data is in the `seasons` table (mig 015). | **HIGH** (broken feature) | the bug |
| **`POST /metadata/apply` has NO passing E2E** | `manual-search.api.spec.ts` — 4 `test.skip('[P1] …metadata/apply…')` "Skip until we have proper test data seeding" | **MED-HIGH** (real feature, untested e2e) | masked gap |
| **Detail E2E self-skips on empty DB** | `media-detail.spec.ts` — ~10 `test.skip(!movie, 'No movies available')`; on CI's empty DB the whole suite skips → false green | **HIGH** (process) | systemic blind spot |
| **3 whole API describe blocks skipped** | `api.spec.ts` — `describe.skip` (Health/Movies/Response-format) "Skip API tests if backend is not running" — superseded by the `*.api.spec.ts` files | LOW | dead/superseded |
| Go `testing.Short()` perf/integration skips (5) | various `*_test.go` | none | legit |
| `qBittorrent`/`TMDB_API_KEY` env-guard skips | downloads/tmdb api specs | none | legit (external dep) |
| poster-too-large E2E skip (unit-covered) | `metadata-editor.api.spec.ts:369` | none | legit (documented, covered in `metadata_handler_test.go`) |

JS/TS **unit** specs: 0 skips (clean). The masking is concentrated in the
**data-dependent E2E** layer.

## Stories

### 20-1 — Fix the season-summary read path (the bug) — *ready-for-dev*
`GetSeasons` must read the canonical `seasons` table (`season_repository.
FindBySeriesID`), not the dead `series.seasons` JSON column. Wire `repos.Seasons`
into `SeriesService`; map `[]models.Season` → `[]SeasonSummary`. Add a
**repository/integration test against a real sqlite DB** that seeds a series +
seasons rows and asserts `GET /series/:id/seasons` returns them (the test that
would have caught this). Deprecate the dead `series.seasons` column (drop-column
migration may be a separate follow-up). File: `bugfix-20-1-season-summary-read.md`.

### 20-2 — E2E test-data seeding + un-skip masked data-path tests — *planned*
Introduce reusable E2E/integration **test-data seeding** (fixtures/factory or a
seed endpoint scoped to test env). Then **un-skip** the permanently-skipped
`POST /metadata/apply` tests and convert the `media-detail.spec.ts`
`test.skip(!data,…)` self-skips into seeded, actually-running tests. Policy
(Murat): a "needs real data" test must NOT silently self-skip in CI — it seeds,
or it is a tracked gap, never a false green. Delete/restore the superseded
`api.spec.ts` describe-skip blocks (confirm coverage exists in the `*.api.spec.ts`
files first).

### 20-3 — Process hardening (rules) — *planned*
- **Rule 15 (DB Column Sync) extension:** when a column is added/changed, the
  repo's **SELECT column list AND row scan** must be synced too — not just
  model/migration/INSERT/UPDATE. (The exact gap that hid 20-1.)
- **Rule 24 (Discovery Triage) reinforcement:** when a migration introduces a
  replacement (e.g. the `seasons` table superseding the `series.seasons` column),
  the superseded mechanism's retirement + all readers re-pointed must be triaged
  as a tracked item at that moment (not left dual-living). (Alexyu confirmed
  2026-06-14: strengthen Rule 24, not Rule 20.)
- TEA `code-review`/CI: flag permanently-skipped `*.api.spec.ts` tests as debt.

## Definition of Done
- 20-1 merged; the season accordion shows real seasons (verified on localhost
  with seeded data). 20-2 seeding in place + the masked tests run green for real.
  20-3 rules updated in `project-context.md` (+ CR sync if applicable).
