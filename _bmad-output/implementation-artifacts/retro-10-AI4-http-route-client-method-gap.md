# Story: HTTP Route ↔ Client Method Sync Verification (Rule 15 Extension)

Status: done

## Story

As a Dev Agent (Amelia) implementing a story that references an **existing client method** or **existing endpoint** as a given,
I want Rule 15 self-verification (and the dev-story checklist that enforces it) to explicitly require confirming the server-side HTTP route is registered in `main.go`,
so that story scope doesn't silently expand mid-implementation like it did in Story 10-2 (Go client had `tmdb.GetMovieVideos`, but the `/api/v1/tmdb/movies/:id/videos` HTTP route on `tmdbHandler` was never wired — DEV had to add unplanned backend exposure).

## Acceptance Criteria

1. Given a developer following `_bmad/bmm/workflows/4-implementation/dev-story/checklist.md`, when a story task references an existing client method or endpoint as a precondition (e.g., "videos endpoint already exists in client", "movie search method already registered"), then the checklist has an explicit item that requires the dev to grep `apps/api/cmd/api/main.go` for the corresponding `{handler}.RegisterRoutes(apiV1)` call AND grep the handler file for the exact HTTP method + path, and record the verification result in Dev Agent Record → Completion Notes
2. Given the grep confirms the route IS registered, when the dev completes the task, then Completion Notes records `🔌 Route Sync: {METHOD} {path} verified at {file:line}` — auditable, non-silent (retro-10-AI2 pattern)
3. Given the grep finds the route is NOT registered (client method orphaned), when the dev encounters this gap, then the dev MUST expand the story scope to include the route registration (new task, new AC if behavior changes, new File List entry) OR halt and ask the user — not silently assume the client call will work
4. Given Rule 15 in `project-context.md` today lists three sub-sections (main.go Wiring, DB Column Sync, Swagger), when this story lands, then Rule 15 gains a fourth sub-section titled "HTTP Route ↔ Client Method Sync" that documents the gap + the grep procedure + the Story 10-2 precedent, keeping the authoritative rule source consistent with the checklist
5. Given both files are updated, when `pnpm lint:all` runs post-change, then it passes with zero regressions (markdown + checklist text edits only; no code change; no ESLint/Prettier/go-vet impact)
6. Given the sprint-status.yaml entry `retro-10-AI4-http-route-client-method-gap`, when this story completes dev-story → code-review, then the status transitions `backlog → ready-for-dev → in-progress → review → done`, and a completion note captures the final line ranges in both `project-context.md` (Rule 15) and `dev-story/checklist.md`

## Tasks / Subtasks

- [x] Task 1: Add "HTTP Route ↔ Client Method Sync" checklist item to `_bmad/bmm/workflows/4-implementation/dev-story/checklist.md` (AC: #1, #2, #3)
  - [x] 1.1 Placement: insert under the existing "## ✅ Implementation Completion" section (currently lines ~32–39), AFTER the "**Dependencies Within Scope**" item and BEFORE the "## 🧪 Testing & Quality Assurance" section. Rationale: this is a scope-completeness check, not a testing check. Keeps grouping clean.
  - [x] 1.2 Draft the new checklist item with this shape:
    ```markdown
    - [ ] **HTTP Route ↔ Client Method Sync (Epic 10 Retro AI-4):** If a story task
      references an "existing client method" or "existing endpoint" as a given,
      do NOT assume the HTTP route is wired up — a Go client method or a frontend
      service call can exist without a server-side route (the runtime result is a
      404, silent in `go test` / unit tests, fatal in real use).

      Verification procedure (run for EACH such task reference):
        1. Find the client call's HTTP method + path. Examples:
           - Go client: `apps/api/internal/tmdb/client.go::GetMovieVideos` →
             `GET https://api.themoviedb.org/...` (external — OUT of scope for this
             check; external routes are TMDb/qBT/etc.)
           - Internal API client: `apps/api/internal/*/client.go` for plugin clients,
             OR `apps/web/src/services/*.ts` for frontend services calling our own
             backend → THIS is the in-scope case.
        2. For in-scope (our backend) routes, grep: `grep -n "RegisterRoutes"
           apps/api/cmd/api/main.go` to locate the handler registration, then
           read the handler file's `RegisterRoutes` method for the exact method +
           path match (e.g., `group.GET("/movies/:id/videos", h.GetMovieVideos)`).
        3. Record result in Dev Agent Record → Completion Notes:
           - If registered: `🔌 Route Sync: {HTTP_METHOD} {path} verified at
             {handler_file}:{line} (registered in main.go:{line})`
           - If NOT registered: HALT. The task cannot be completed as written.
             Expand story scope to include route registration, OR ask the user
             whether to defer to a follow-up story.
        4. If the story is 100% frontend OR 100% docs (no Go handler impact),
           record: `🔌 Route Sync: N/A (no backend route touched)`.

      Why this exists: Story 10-2 Task 3.3 said "videos endpoint already exists
      in client". The Go client method `tmdb.GetMovieVideos` (in
      `apps/api/internal/tmdb/client.go`) did exist — but the internal backend
      route to expose it to the frontend,
      `GET /api/v1/tmdb/movies/:id/videos` → `tmdbHandler.GetMovieVideos`
      (`apps/api/internal/handlers/tmdb_handler.go:440`), was NOT registered.
      DEV had to add the backend route exposure mid-story, silently expanding
      scope without a matching AC update. This check surfaces that gap at
      task-complete time, not at CR time.
    ```
  - [x] 1.3 Verify the new item renders correctly as rendered markdown (bullets, code spans, no unescaped angle brackets).

- [x] Task 2: Extend Rule 15 in `project-context.md` with the same gap + procedure (AC: #4)
  - [x] 2.1 Current Rule 15 has three sub-sections under the `Before marking a story task complete, verify:` header: **main.go Wiring**, **DB Column Sync**, **Swagger**. Insert a new fourth sub-section **HTTP Route ↔ Client Method Sync** AFTER the existing three, before Rule 15's closing code fence.
  - [x] 2.2 Draft the new sub-section:
    ```
    HTTP Route ↔ Client Method Sync:
      ✅ If a task description says "endpoint already exists in client" or
         "method already registered", grep apps/api/cmd/api/main.go for the
         corresponding {handler}.RegisterRoutes(apiV1) call AND verify the
         exact HTTP method + path in the handler file.
      ✅ Client method existing ≠ HTTP route registered. Assume nothing.
      ✅ If route is missing, expand story scope (new task + AC) before
         continuing. Do not silently add it.
      ❌ Trusting a client method's existence as proof the server route is wired.
      📌 Precedent (Epic 10 Retro AI-4, Story 10-2 Task 3.3): the Go client
         method tmdb.GetMovieVideos in apps/api/internal/tmdb/client.go existed,
         but the internal backend route GET /api/v1/tmdb/movies/:id/videos →
         tmdbHandler.GetMovieVideos (apps/api/internal/handlers/tmdb_handler.go:440)
         was never wired — DEV had to add it mid-story, silently expanding scope.
    ```
  - [x] 2.3 Update the "Last Updated" header line at the top of `project-context.md` to note the Rule 15 extension (pattern: same as prior retro updates, e.g., "Rule 15 HTTP Route ↔ Client Method Sync extension (retro-10-AI4)").
  - [x] 2.4 Do NOT renumber rules. Just append the new sub-section inside Rule 15's existing block. Rule 16 and onward stay put.

- [x] Task 3: Verify zero code regressions + update sprint-status.yaml (AC: #5, #6)
  - [x] 3.1 Run `pnpm lint:all` — PASS expected (markdown files are not linted by ESLint/Prettier/go-vet/staticcheck per this repo's lint scope)
  - [x] 3.2 Run `pnpm nx test api` — PASS expected (zero Go code change)
  - [x] 3.3 Run `pnpm nx test web` — PASS expected (zero frontend code change)
  - [x] 3.4 Mark `retro-10-AI4-http-route-client-method-gap: ready-for-dev` at creation (this step). Transitions during execution: ready-for-dev → in-progress → review → done. On completion, record in sprint-status.yaml comment the final line ranges in both edited files (e.g., "checklist.md line 40, project-context.md Rule 15 sub-section added at lines X–Y").

## Dev Notes

### Root Cause

Epic 10 retro (2026-04-20) Pattern #4: "Story scope drift — 'client method exists' ≠ 'HTTP route exists'". Two cases surfaced in Epic 10:

- **Story 10-2 Task 3.3**: Wrote "/videos endpoint already exists in client". The Go client method `tmdb.GetMovieVideos` in `apps/api/internal/tmdb/client.go` existed and could hit TMDb. But the **internal backend route** to expose that to the frontend (`GET /api/v1/tmdb/movies/:id/videos` → `tmdbHandler.GetMovieVideos` at `apps/api/internal/handlers/tmdb_handler.go:440`) was not registered. DEV had to add the backend route as unplanned scope. Caught only because the frontend TrailerModal integration test 404'd in dev.

- **Story 10-4 path rename** (different flavor): `/movies/check-owned` was renamed to `/media/check-owned` mid-implementation after realizing the endpoint queries both `movies` and `series` tables. This is a naming drift, not a wiring gap — handled separately via path-naming review. OUT of scope for THIS story; AI-4 targets the wiring gap specifically.

Rule 15 today already enforces "new routes added to router setup" — but that targets the **author side** (don't forget to register the route you just wrote). The Epic 10 pattern is the **consumer side**: don't trust that an existing client method implies an existing route. These are different failure modes.

### Why checklist + Rule 15 (not workflow instructions.xml)?

Prior retros landed workflow-level gates inside `instructions.xml`:
- retro-9-AI1 Full Regression Gate → `dev-story/instructions.xml` Step 7
- retro-9c-AI2 Fix-or-File → `dev-story/instructions.xml` Step 7
- retro-10-AI2 AC Drift Check → `dev-story/instructions.xml` Step 2
- retro-10-AI3 Rule 7 Wire Format → `code-review/instructions.xml` Step 3

This story uses the **checklist.md + project-context.md Rule 15** route instead, mirroring retro-10-AI1 (Frontend Perf+A11y Pre-Flight Checklist). Rationale:

1. **Rule 15 is already the canonical home for "before marking a task complete, verify X".** Adding the gap check to Rule 15 keeps the three-step Rule-15 mental model intact (wiring, DB, Swagger, and now route↔client sync).
2. **This check is a spot verification, not a workflow gate.** Not every story has a "client method exists" precondition. A per-item checklist entry fires only when the situation applies; adding a MANDATORY instruction.xml action would add noise to the 80% of stories that don't have this case.
3. **LOW priority** (per retro categorization) warrants a lighter-touch intervention than the MEDIUM retro actions (AI-2, AI-3). Checklist items are easy to skim and cite; workflow-level MANDATORY actions carry more ceremonial weight and are reserved for gates that must fire every time.

### Concrete grep pattern for verification

In this repo's pattern (Gin-based handlers registered through `{handler}.RegisterRoutes(apiV1)` called from `apps/api/cmd/api/main.go:513`):

```bash
# Step 1: Is the handler registered at all?
grep -n "RegisterRoutes" apps/api/cmd/api/main.go

# Step 2: Does the handler expose the specific method + path?
# (Reader: adjust 'tmdb_handler.go' and 'videos' per your specific case;
#  for Story 10-2 the route lives on tmdbHandler, not movieHandler)
grep -nE '(GET|POST|PUT|DELETE|PATCH).*"/(videos|movies|series)' \
  apps/api/internal/handlers/tmdb_handler.go
```

Example (Story 10-2 retrospective reconstruction):

```
$ grep -n "videos" apps/api/internal/handlers/tmdb_handler.go
# BEFORE Story 10-2 fix: zero hits → route missing
# AFTER Story 10-2 fix: hits on the handler method (line 292), Swaggo @Router
#   annotation (line 304), and the GET registration in RegisterRoutes (line 440)
```

### Rule 15 current state (from project-context.md lines 413–434)

```
### Rule 15: Pre-commit Self-verification

Before marking a story task complete, verify:

main.go Wiring:
  ✅ New handlers/services registered in main.go dependency injection
  ✅ New routes added to router setup
  ❌ Implementing handler but forgetting to wire it up

DB Column Sync:
  ✅ New model fields have corresponding migration ALTER/CREATE
  ✅ Repository INSERT/UPDATE SQL includes ALL model fields
  ❌ Adding model field but missing it in repository SQL or migration

Swagger:
  ✅ New/changed endpoints have updated Swaggo annotations
  ✅ Run swag init if annotations changed
  ❌ Changing API contract without updating docs
```

The new sub-section slots in after "Swagger:" and before the closing code fence. Rule 15's three existing sub-sections are each 3-bullet blocks; the new one matches that shape for visual consistency.

### Out of Scope

- Automating the route-registered check (a Go `TestAllClientMethodsHaveRoutes` test). Too brittle because "client method" is a loose concept that spans both `apps/api/internal/*/client.go` (plugin clients calling external services) and `apps/web/src/services/*.ts` (frontend calling our backend). The first group intentionally points to external routes. Automation requires structured annotations; deferred to a future retro if needed.
- Addressing Story 10-4's path-rename pattern (`/movies/check-owned` → `/media/check-owned`). That's a different class of issue (semantic naming) and belongs to a future UX/API-design conventions doc, not Rule 15.
- Touching `code-review/instructions.xml` or `code-review/checklist.md`. CR already catches wire-contract gaps via its git-vs-story-file-list audit and test-runtime failures. AI-4 targets the author side (prevent the gap at implementation) not the review side.

### References

- [Source: `_bmad-output/implementation-artifacts/epic-10-retro-2026-04-20.md#challenges`] Pattern #4 — Story scope drift, "client method exists" ≠ "HTTP route exists"
- [Source: `_bmad-output/implementation-artifacts/epic-10-retro-2026-04-20.md#action-items`] AI-4 row (SM, LOW priority)
- [Source: `_bmad-output/implementation-artifacts/sprint-status.yaml`] `retro-10-AI4-http-route-client-method-gap: backlog` entry + Agreement 4 tracking rule
- [Source: `project-context.md#rule-15-pre-commit-self-verification`] lines 413–434 — target insertion point for the new sub-section
- [Source: `_bmad/bmm/workflows/4-implementation/dev-story/checklist.md`] "## ✅ Implementation Completion" section — target insertion point for the new checklist item
- [Source: `apps/api/cmd/api/main.go:513`] `apiV1 := router.Group("/api/v1")` — anchor for route-registration grep
- [Source: `apps/api/internal/handlers/movie_handler.go`] Reference for per-handler `RegisterRoutes` pattern
- [Precedent: `retro-10-AI1-frontend-perf-a11y-checklist`] Same shape (checklist-only update), same precedent-citation style
- [Precedent: `retro-10-AI2-ac-contract-drift-check`] Same "record result in Completion Notes — silence is not an option" audit rule

## Dev Agent Record

### Agent Model Used

Amelia (BMM Dev Agent) / Claude Opus 4.7 (1M context) — `/bmad:bmm:agents:dev` → `/bmad:bmm:workflows:dev-story`, invoked 2026-04-22.

### Debug Log References

- `pnpm lint:all` (repo root, 2026-04-22): 0 errors, 129 pre-existing warnings (no new warnings — docs-only changes cannot introduce lint regressions); `prettier --check .` PASS.
- `pnpm nx test api` (2026-04-22): PASS, Go backend green (partially cached per Nx).
- `pnpm nx test web` (2026-04-22): 144 files / 1738 tests, all PASS; `test:cleanup:all` ran automatically, both spawned PIDs (19909, 4379) exited cleanly — no orphaned workers.
- Task 1.3 rendering verification (post-CR M3 fix, 2026-04-22): the new checklist item at `checklist.md:40-71` was previewed via the IDE markdown preview and via `grep -nE 'escaped|<|>' checklist.md` to confirm no unescaped angle brackets broke the listing; bullets (✅/❌/📌 replacements), nested numbered list (1-4), and `↔` U+2194 character all render correctly.

### Completion Notes List

- 🔗 AC Drift: N/A (bootstrap of new workflow rule — no prior story AC covers the shape or count of Rule 15 sub-sections, the internal item set of `dev-story/checklist.md`, or HTTP Route ↔ Client Method Sync semantics; checked grep patterns `Rule 15` / `HTTP Route|Client Method|RegisterRoutes` / `Perf.A11y` across `_bmad-output/implementation-artifacts/*.md` — 24 hits across 13 unique files, all REUSE not DRIFT: prior Rule 15 citations are compliance checks against its existing 3-subsection contract, not specs on the count; prior RegisterRoutes mentions are implementation patterns, not AC contracts).
- 🔌 Route Sync: N/A (no backend route touched — this is a 100% docs/workflow story modifying `_bmad/bmm/workflows/4-implementation/dev-story/checklist.md` and `project-context.md` only; no Go handler, service, or repository code was added or changed).
- 🎨 UX Verification: SKIPPED — no UI changes in this story (zero files under `apps/web/`).
- AC #1 satisfied: new checklist item inserted at `_bmad/bmm/workflows/4-implementation/dev-story/checklist.md:40–71`, under `## ✅ Implementation Completion`, directly after "Dependencies Within Scope" and before `## 🧪 Testing & Quality Assurance`. Content matches the story Task 1.2 template verbatim.
- AC #2 satisfied: the new checklist item's step 3 explicitly mandates `🔌 Route Sync: {HTTP_METHOD} {path} verified at {handler_file}:{line} (registered in main.go:{line})` in Completion Notes when a route is registered — non-silent, auditable, matches the retro-10-AI2 three-state audit rule. (Post-CR: `{METHOD}` tightened to `{HTTP_METHOD}` to disambiguate from Go receiver method — L3 fix.)
- AC #3 satisfied: step 3 branch for "NOT registered" explicitly directs the developer to HALT and either expand scope or ask the user — no silent scope expansion path.
- AC #4 satisfied: Rule 15 in `project-context.md:421–455` now has a 4th sub-section `HTTP Route ↔ Client Method Sync:` at lines 441–454, matching the shape of existing main.go Wiring / DB Column Sync / Swagger blocks (3 ✅ bullets + 1 ❌ bullet) plus an explicit `📌 Precedent` line citing Story 10-2 Task 3.3 (added via CR H2 fix to satisfy AC #4's "Story 10-2 precedent" requirement). Rules 16+ unchanged (Rule 16 now at line 457). "Last Updated" header at line 7 updated to `2026-04-22` with retro-10-AI4 citation.
- AC #5 satisfied: `pnpm lint:all` PASS (0 errors). Markdown files are not processed by `go vet` / `staticcheck` / ESLint; Prettier confirms all formatting compliant.
- AC #6 satisfied: sprint-status.yaml entry `retro-10-AI4-http-route-client-method-gap` transitioned `ready-for-dev → in-progress → review → done` across dev-story + CR. Final post-CR line ranges: `checklist.md:40–72` (item grew 1 line due to longer "Why this exists" paragraph with correct path), `project-context.md:441–454` (Rule 15 sub-section grew 5 lines due to precedent addition; Rule 16 now at line 457).
- 🔥 CR Fix Round (Amelia self-review, 2026-04-22, adversarial): 2 HIGH + 3 MEDIUM + 3 LOW findings, all fixed via option [1] (L4 withdrawn — 6-space continuation aligns with `- [ ] ` prefix, standard markdown):
  - **H1 Precedent path factually wrong in checklist "Why this exists" block.** Original cited `/movies/:id/videos` on `movieHandler.GetMovieVideos`; actual Story 10-2 route is `/api/v1/tmdb/movies/:id/videos` on `tmdbHandler.GetMovieVideos` at `apps/api/internal/handlers/tmdb_handler.go:440` (verified by grep — `movie_handler.go` has zero `videos` hits; `tmdb_handler.go` has the actual route). Fixed in `checklist.md:67-76`, story Root Cause at line 90, and story narrative at line 9. Meta-irony noted: the anti-precedent-drift story itself shipped a drifted precedent.
  - **H2 AC #4 under-satisfied — Story 10-2 precedent missing from Rule 15 sub-section body.** AC #4 requires "the gap + the grep procedure + the Story 10-2 precedent" all inside the sub-section. Original landed 2 of 3 (gap + procedure). Fixed by adding `📌 Precedent (Epic 10 Retro AI-4, Story 10-2 Task 3.3): ...` line citing the real `tmdbHandler.GetMovieVideos` case at `project-context.md:450-454`; parity with `checklist.md` "Why this exists" block now holds per AC #4's "keeping the authoritative rule source consistent with the checklist" clause.
  - **M1 Story Dev Notes concrete grep example used wrong file.** `grep ... apps/api/internal/handlers/movie_handler.go` returns 0 hits; fixed to `apps/api/internal/handlers/tmdb_handler.go` in story lines 120-131 (bash example + retrospective reconstruction block).
  - **M2 Rule 15 sub-section heading broke sibling-symmetry.** Others: `main.go Wiring:`, `DB Column Sync:`, `Swagger:` (2-3 word labels). Original: `HTTP Route ↔ Client Method Sync (Epic 10 Retro AI-4):` (retro-tag appended). Fixed by dropping `(Epic 10 Retro AI-4)` from heading (retro citation recovered via the new `📌 Precedent` line inside the body). Matches sibling style.
  - **M3 Task 1.3 "Verify renders correctly" marked [x] without evidence.** Fixed by adding rendering-verification entry to Debug Log References (2026-04-22 dated, per the audit pattern Epic 8 TD3/TD4 retro introduced).
  - **L1 Story Task 1.2 draft showed `- [x]` placeholder; actual landed `- [ ]`.** Fixed draft at story line 26 to match landed implementation.
  - **L2 Story Task 2.2 draft typo "Ac" (lowercase c).** Fixed to `AC` at story line 72; also synced draft to reflect dropped retro tag + new precedent line (M2 + H2 parity).
  - **L3 `{METHOD}` placeholder ambiguous.** Fixed to `{HTTP_METHOD}` in `checklist.md:58` and story line 46 draft.
  - **L4 Withdrawn:** 6-space continuation indent in the new checklist item aligns with text after `- [ ] ` (6-char prefix); it is the standard markdown list-item continuation, not a regression. No change needed.

### File List

- `_bmad/bmm/workflows/4-implementation/dev-story/checklist.md` — added new "HTTP Route ↔ Client Method Sync (Epic 10 Retro AI-4)" checklist item under `## ✅ Implementation Completion` (lines 40–76 post-CR). Post-CR edits: `{METHOD}`→`{HTTP_METHOD}` (L3) and "Why this exists" paragraph rewritten with correct Story 10-2 path `/api/v1/tmdb/movies/:id/videos` on `tmdbHandler.GetMovieVideos` at `tmdb_handler.go:440` (H1).
- `project-context.md` — added 4th sub-section to Rule 15 at lines 441–454 (grew from 9 → 14 lines due to CR H2 precedent line); updated "Last Updated" header at line 7. Post-CR edits: dropped `(Epic 10 Retro AI-4)` from sub-section heading for sibling symmetry (M2) and added explicit `📌 Precedent (Epic 10 Retro AI-4, Story 10-2 Task 3.3): ...` line inside the body to satisfy AC #4 "Story 10-2 precedent" literal text (H2). Rule 16 now at line 457 (was 452 pre-CR).
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — entry `retro-10-AI4-http-route-client-method-gap` status `ready-for-dev → in-progress → review → done` across dev-story + CR. Final comment records post-CR line ranges.
- `_bmad-output/implementation-artifacts/retro-10-AI4-http-route-client-method-gap.md` — this story file: all 14 Tasks/Subtasks checkboxes marked [x], Status → done (post-CR), Dev Agent Record populated with dev + CR entries, Change Log records both landing + CR fix rounds. CR-round edits: Root Cause block (line 90), narrative story header (line 9), Task 1.2 + Task 2.2 drafts (lines 26, 46, 72), concrete grep example (lines 120-131), Completion Notes entries for AC #2/#4/#6 + a full CR Fix Round block, Debug Log M3 entry.

### Change Log

| Date       | Change                                                                                                                                                                                                                           |
| ---------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 2026-04-22 | Added "HTTP Route ↔ Client Method Sync (Epic 10 Retro AI-4)" checklist item to `_bmad/bmm/workflows/4-implementation/dev-story/checklist.md` at lines 40–71, directly after "Dependencies Within Scope" and before Testing section. Item mandates grep `main.go` for `RegisterRoutes` when a task references "existing client method" / "existing endpoint" as a given, with three-state audit record (PASS / HALT / N/A) in Completion Notes. (AC #1, #2, #3) |
| 2026-04-22 | Extended Rule 15 in `project-context.md` with new 4th sub-section "HTTP Route ↔ Client Method Sync (Epic 10 Retro AI-4)" at lines 441–449, matching existing sub-section shape (3 ✅ + 1 ❌). Rules 16+ unchanged. Updated "Last Updated" header at line 7. (AC #4)                                                                   |
| 2026-04-22 | Full regression gate PASS: `pnpm lint:all` 0 errors, `pnpm nx test api` PASS, `pnpm nx test web` 1738/1738 PASS, cleanup verified. (AC #5)                                                                                                                                                                                                  |
| 2026-04-22 | Sprint-status.yaml `retro-10-AI4-http-route-client-method-gap` entry transitioned `ready-for-dev → in-progress`; will transition to `review` as Step 10 saves. Final line ranges recorded in this Completion Notes list for CR comment backfill. (AC #6) |
| 2026-04-22 | 🔥 **CR Fix Round (Amelia self-review, adversarial):** 2 HIGH + 3 MEDIUM + 3 LOW fixed (L4 withdrawn). **H1:** corrected Story 10-2 precedent path `/movies/:id/videos`→`/api/v1/tmdb/movies/:id/videos` and handler `movieHandler`→`tmdbHandler` in `checklist.md:67-76` + story Root Cause + narrative header (meta-irony: the anti-precedent-drift story itself shipped a drifted precedent). **H2:** added `📌 Precedent` line inside Rule 15 sub-section at `project-context.md:450-454` to satisfy AC #4's literal "Story 10-2 precedent" requirement. **M1:** concrete grep example `movie_handler.go`→`tmdb_handler.go` in story lines 120-131. **M2:** dropped `(Epic 10 Retro AI-4)` tag from Rule 15 sub-section heading for sibling symmetry (tag recovered via H2 precedent line). **M3:** added Task 1.3 rendering-verification entry to Debug Log (closes Epic 8 TD3/TD4 audit pattern). **L1:** story Task 1.2 draft `- [x]`→`- [ ]`. **L2:** story Task 2.2 draft "Ac"→"AC" + synced retro tag / precedent line removal/addition. **L3:** `{METHOD}`→`{HTTP_METHOD}` in `checklist.md:58` + draft. Final state: Rule 15 at `project-context.md:441-454`, Rule 16 at line 457. Story Status `review → done`. |
| 2026-04-22 | Sprint-status.yaml `retro-10-AI4-http-route-client-method-gap` entry final transition `review → done` set by CR workflow. Comment updated with post-CR line ranges. (AC #6) |
