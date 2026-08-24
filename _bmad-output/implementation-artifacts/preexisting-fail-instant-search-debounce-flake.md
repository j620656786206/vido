# Bugfix: deflake `InstantSearchBar` debounce spec — wait on the spy, not on the dropdown

Status: done

**Origin:** `preexisting-fail-instant-search-debounce-flake` (backlog since 2026-06-05, filed during `disc-flaky-visual-media-detail-panel` per Epic 9c retro AI-2). Fails rarely on the full parallel web suite (`unifiedSearch` spy 0 calls); most recently red on PR #267's first CI run (2026-08-24, an unrelated Go-only diff — rerun went green). Same class as the two solved sleep/race flakes (`bugfix-scanner-sse-cancel-flake`, `preexisting-fail-generation-batch-cancel-mid-item-flake`): asserting on a side effect the awaited condition does not actually guarantee. **Test-only.**

---

## Story

As the team,
I want the InstantSearchBar spec to await the exact condition it asserts,
so that a saturated CI runner cannot turn a passing debounce into a red PR on unrelated changes.

## Root cause (verified against code)

`InstantSearchBar.spec.tsx:107-120`: the test `waitFor`s the **dropdown** (`search-suggestions`) and then synchronously asserts the **spy**:

- The dropdown opens on `open = focused && debouncedQuery.length >= MIN_QUERY_LENGTH` (`InstantSearchBar.tsx:55`) — i.e. as soon as the 300 ms `useDebounce` fires, **before** React Query has invoked the mocked `tmdbService.unifiedSearch` (the dropdown has a loading state). On an idle machine the query fn runs in the same flush, so the spy assertion passes; on a saturated runner the gap between "dropdown rendered" and "query fn invoked" is real → `expected "spy" to be called with [ '你的名字' ]` with 0 calls. Exactly the failure recorded in the backlog entry and on PR #267.
- The synchronous half of the test (`not.toBeInTheDocument()` + `not.toHaveBeenCalled()` right after `fireEvent.change`) is safe: it runs in the same macrotask, the 300 ms timer cannot have fired.

Two sibling assertions in the same file lean on the default 1 s `waitFor` timeout for a chain that includes the 300 ms debounce **plus** query resolution **plus** a router navigation (`:146-149`, `:160-171`) — the same load-sensitivity, just with more headroom; they have not been observed red but are one bad scheduler pause away.

## Fix (test-only, `apps/web/src/components/search/InstantSearchBar.spec.tsx`)

| # | Change |
|---|--------|
| F1 | In the debounce test: `await waitFor(() => expect(tmdbService.unifiedSearch).toHaveBeenCalledWith('你的名字'))` — the spy IS the awaited condition; then assert the dropdown (which by then must be rendered — keep the existing `getByTestId` assertion after it). |
| F2 | The two navigation tests: before asserting rendered suggestion text, `await waitFor` on the spy call first (same pattern), so the text wait starts only after the fetch actually ran. Keep every existing assertion. |
| F3 | Leave the `350 ms` sleep in the "shorter than 2 characters" test — it asserts a **negative** (never called), where sleeping longer only makes the assertion stronger; converting it to fake timers would drag React Query and `use-debounce` into fake-timer interop for zero flake-risk gain. Record this decision in a comment. |

No `vi.useFakeTimers()`: the backlog entry suggested a fake-timer audit, but `use-debounce` + React Query + `waitFor` under fake timers is a known interop tarpit (every `waitFor` then needs manual timer advancement), and the actual defect is an ordering assumption, not timer reality. State this in the story-completion note.

## Acceptance Criteria

1. **AC #1** — The debounce test awaits the `unifiedSearch` spy call (with the exact query) via `waitFor` before any dropdown/spy synchronous assertion; the pre-debounce synchronous negative assertions stay.
2. **AC #2** — The two navigation tests wait on the spy before waiting on rendered suggestion text.
3. **AC #3** — Stress: the file passes `pnpm nx test web -- --run src/components/search/InstantSearchBar.spec.tsx` 20× consecutively AND the full `pnpm nx test web` twice, with 8 CPU busy-loops pinning the machine during one full run (the condition the flake needs).
4. **AC #4** — Component code untouched (`InstantSearchBar.tsx` git diff 0); no test deleted, no assertion weakened (every existing `expect` survives, possibly reordered into/after a `waitFor`).

## Tasks / Subtasks

- [x] **Task 1 — F1 + F2** in `InstantSearchBar.spec.tsx`; comment on F3.
- [x] **Task 2 — Verify** (AC #3, #4): 20× file loop green; full web suite ×2 green (one under CPU pinning); `git diff apps/web/src/components/search/InstantSearchBar.tsx` = 0; `pnpm nx lint web`; `format:check`.
- [x] **Task 3 — Record**: Dev Agent Record + sprint-status; note the fake-timer decision.

### Cross-stack split check — Frontend 3 / Backend 0 ⇒ no split.

## Dev Notes

- Precedents: `bugfix-scanner-sse-cancel-flake.md` (drive sync from the asserted event) and PR #266 (`eventsUntilTerminal`) — same principle: *await the thing you assert*.
- `waitFor` polls with its own act() handling; asserting a `vi.fn` spy inside it is standard RTL practice — no extra flush needed.
- Do NOT raise global `testTimeout` or `waitFor` timeouts as the "fix" — that hides the ordering bug instead of removing it. If a specific `waitFor` still needs more than 1 s under pinned CPU on AC #3's stress run, raising the timeout *on that one call* is acceptable and must be noted.
- A11y pre-flight applies (`apps/web` touched): spec-only change, so jsx-a11y warnings on touched files should be 0 introduced — record the gate result anyway.
- Rule 23 check: the spec does not read wall-clock beyond the 350 ms sleep; no fixture states involved — `N/A — no wall-clock-reading components touched` (component untouched).

### References

- [Source: `apps/web/src/components/search/InstantSearchBar.spec.tsx:107-120, 128-135, 138-171`]
- [Source: `apps/web/src/components/search/InstantSearchBar.tsx:49-60` — `useDebounce(300)`; `:55` — dropdown opens before the fetch resolves]
- [Source: sprint-status entry `preexisting-fail-instant-search-debounce-flake` (2026-06-05); PR #267 first CI run (2026-08-24) — the same signature in the wild]

## Dev Agent Record

### Agent Model Used

Claude Fable 5 (claude-fable-5) — 2026-08-24.

### Debug Log References

- None needed — the file went green first run after the reorder; the stress evidence is the deliverable.

### Completion Notes List

- F1: debounce test now `waitFor`s `unifiedSearch` called with `'你的名字'`, then asserts the dropdown synchronously (by then it must exist). F2: both navigation tests wait on the spy before waiting on rendered text. F3: the 350 ms negative-assertion sleep kept, with the fake-timer decision recorded in a comment.
- No `waitFor` timeout was raised anywhere — the ordering fix alone survived the stress runs.
- **Stress (AC #3)**: file ×20 consecutive green; full `nx test web` ×2 green (237 files, 2763 tests), one of them under 8 CPU busy-loops; `nx test api` green (untouched, full-regression gate).
- **AC #4**: `git diff apps/web/src/components/search/InstantSearchBar.tsx` = 0 lines; all 8 tests survive, no assertion removed (two reordered into `waitFor`).
- Gates: `nx lint web` green · `format:check` green · 0 orphaned workers.
- 🎭 A11y Pre-Flight: PASS (1 file touched — spec only; 0 jsx-a11y warnings introduced; component untouched so the 4 recurring classes are unaffected — the existing `aria-activedescendant` assertions still run).
- Code review (fresh-context agent, same model): no blocking findings — the "dropdown exists once the fetch ran" guarantee was independently proven (shared `debouncedQuery` state + act/task-boundary reasoning); one residual noted: the three spy waits keep ~700 ms headroom inside the 1 s `waitFor` budget (irreducible with real timers; per-call raise pre-authorised if ever needed).
- 🔗 AC Drift: N/A (test-only; the debounce contract "no fetch before 300 ms, fetch after" is asserted unchanged). 📎 Contract Stamps: NONE. 🎨 UX Verification: SKIPPED (no visual change).

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - `N/A — no out-of-scope work discovered`.

### File List

- `apps/web/src/components/search/InstantSearchBar.spec.tsx`
- `_bmad-output/implementation-artifacts/preexisting-fail-instant-search-debounce-flake.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`

## Change Log

| Date | Change |
|------|--------|
| 2026-08-24 | code-review: no blocking findings; residual 700ms headroom noted. Status → done. |
| 2026-08-24 | dev-story (Amelia): spy-first waits in 3 tests; file ×20 + full suite ×2 (one CPU-pinned) green. Status → review. |
| 2026-08-24 | Story created (create-story). Root cause: the test awaits the dropdown, which opens on the debounce BEFORE React Query invokes the mocked fetch — the spy is asserted synchronously in that gap. |
