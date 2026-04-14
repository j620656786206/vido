# Sprint Change Proposal — Story 10-1a Discover YearRange Validation

**Date:** 2026-04-14
**Epic:** Epic 10 — Homepage TV Wall
**Scope:** Minor (Direct Adjustment)
**Status:** Approved by Alexyu (party-mode session 2026-04-14)

---

## 1. Issue Summary

During the test automation review (`/bmad:bmm:agents:tea TA`) for Story 10-1 (TMDb Trending & Discover API, status `review`), a P2/P3 coverage gap was identified: the `parseDiscoverParams` helper accepts `year_gte > year_lte` without validation and passes the reversed range through to TMDb. TMDb's behavior for reversed `primary_release_date.gte` / `primary_release_date.lte` pairs is undefined — it can return empty results, ignore one field, or reject the request, with no contract guarantee.

**Example:** `GET /api/v1/tmdb/discover/movies?year_gte=2030&year_lte=2020` currently returns a silently successful empty page that gets cached for 1 hour (per Story 10-1 AC #5) — blocking recovery and masking frontend bugs.

## 2. Impact Analysis

### Epic Impact
- Epic 10 can still be completed as planned
- No new Epics needed
- No priority changes
- Does NOT block Story 10-2 (Hero Banner) — Hero Banner consumes trending, not discover

### Story Impact

| Story | Status | Change Required |
|-------|--------|----------------|
| 10-1 TMDb Trending/Discover API | review | No code change; gap handed off to 10-1a |
| 10-1a Discover YearRange Validation | **ready-for-dev (NEW)** | Add error code + handler-layer validation + 7 test cases |

### Artifact Impact
- `_bmad-output/planning-artifacts/epics/epic-10-homepage-tv-wall.md` — add D-1a line to Stories list ✅ done
- `_bmad-output/implementation-artifacts/10-1a-discover-year-range-validation.md` — new story file ✅ done
- `project-context.md` — no change (Rule 18 ApiResponse envelope already covers the 400 shape)

## 3. Recommended Approach

**Direct Adjustment** — Add a narrow follow-up story inside Epic 10's current sprint.

- **Effort:** Low — 2 files to modify (`errors.go`, `tmdb_handler.go`), 2 test files to extend. Amelia's estimate: ~15 minutes including tests.
- **Risk:** Low — validation lives at handler layer only (AC #6 makes this explicit), so internal TMDb client / service / cache callers are unaffected.

## 4. Decision Rationale

Three options were considered in party-mode (Murat / Winston / John / Amelia):

| Option | Verdict | Reason |
|--------|---------|--------|
| A. HTTP 400 + `INVALID_YEAR_RANGE` | ✅ Chosen | Predictable contract; no poisoned cache; fails fast at API boundary |
| B. Auto-swap `year_gte` ↔ `year_lte` | ❌ Rejected | "Magic" fix masks upstream UI bugs; delays bug discovery |
| C. Passthrough (status quo) | ❌ Rejected | Relies on undefined TMDb behavior; caches empty results for 1h |

## 5. Follow-ups / Out of Scope

- **Frontend client-side validation** — OUT of Story 10-1a scope. A separate frontend story will add `year_gte <= year_lte` validation in the discover form UI so users get inline feedback instead of a 400. Not blocked by 10-1a and not in this proposal.
