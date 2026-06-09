# Story: project-context.md Rule 26 — TanStack Router Search-Param Coercion (lone-numeric trap)

Status: ready-for-dev

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a Dev Agent (Amelia) wiring `validateSearch` guards on TanStack Router routes,
I want a numbered `project-context.md` rule documenting that the default search parser JSON-parses a **lone numeric** query value (`?genre=16`) into a `number` — which fails `typeof === 'string'` guards and **silently drops** the param,
so that the next route author coerces with `String()` / a `toCsvString`-style helper up front instead of re-discovering this gotcha a third time at adversarial CR.

## Acceptance Criteria

1. A new `### Rule 26: TanStack Router Search-Param Coercion (lone-numeric trap)` section is added to `project-context.md`, inserted AFTER Rule 25 and BEFORE the `---` that precedes the `## 🧪 Known dev-mode artifacts` section (so the numbered-rule block stays contiguous). House style is followed: `### Rule N: Title` heading + a fenced code block carrying the explanation with `✅ CORRECT` / `❌ WRONG` examples, closed by a `📌 Precedent:` line.

2. The rule body states the mechanism precisely: TanStack Router's **default** search parser JSON-parses each query value; a param holding a SINGLE numeric value (`?genre=16`, `?platform=8`) is parsed into a `number` (not a string); a `typeof x === 'string'` `validateSearch` guard is then `false` and the param is silently dropped; a multi-value form (`?genre=16,28`) stays a string and slips through — so the bug only manifests on single-value deep links and evades casual manual testing.

3. The `✅ CORRECT` example shows the canonical `toCsvString(value)` coercion (number → `String(value)` before the guard) and the `❌ WRONG` example shows the bare `typeof search.genre === 'string'` guard that drops the lone-numeric case. The rule cites the canonical helper location `apps/web/src/routes/discover.tsx::toCsvString`, scopes itself to `validateSearch` guards under `apps/web/src/routes/**` for params writable as a bare number (CSV id-lists: genre, platform, person ids), and notes the contrast cases: genuinely-numeric params use a `toOptionalNumber`-style coercion, and a string-enum guard (`subtitleStatus`) is only safe when the value can NEVER be all-digits.

4. The `📌 Precedent:` line names BOTH recurrences with their stories: 11-2 (`?genre=16` / `?platform=8` single-value deep links dropped → fixed with `toCsvString()` + defensive `String()`, E2E-guarded, caught by CR as a HIGH) and 8-11 (`subtitleStatus` param, same class). It frames the "two strikes across two epics" rationale per Epic 11 Retro Insight 3 / Rule 22 ("codify a framework gotcha the moment it hits twice").

5. The `project-context.md` L7 `**Last Updated:**` mega-line is updated per **Rule 25**: a new newest-first `(...)` entry for this story is PREPENDED, the former lead entry is demoted to a `Prior:` entry, the shared older tail is kept exactly once, and the entry text is **English-only** (a CJK char makes prettier reflow the whole line). Post-edit, `pnpm exec prettier --check project-context.md` passes and the entry count does not shrink.

6. This is a pure docs change (no Go, no TS, no ESLint rule, no baseline regeneration). `pnpm exec prettier --check project-context.md` passes; `pnpm lint:all`, `pnpm nx test api`, and `pnpm nx test web` are unaffected (run as the standard regression gate, all green). sprint-status transitions `backlog → ready-for-dev → in-progress → review → done`.

## Tasks / Subtasks

- [ ] **Task 1: Author Rule 26 in `project-context.md`** (AC: #1, #2, #3, #4)
  - [ ] 1.1 Locate the insertion point: end of Rule 25 (`### Rule 25: Mega-line Rebase Conflict Resolution`, body + `📌 Precedent` block) and the `---` separator before `## 🧪 Known dev-mode artifacts`. Insert Rule 26 between the Rule 25 precedent block and that `---`.
  - [ ] 1.2 Add the Rule 26 section using the draft in Dev Notes → "Proposed Rule 26 text" (adjust wording as needed but preserve every AC #2–#4 requirement). Match the `### Rule N:` + fenced-block + `✅`/`❌` + `📌 Precedent:` house format of Rules 5 / 18 / 25.
  - [ ] 1.3 Verify the code in the `✅ CORRECT` block matches the REAL canonical helper at `apps/web/src/routes/discover.tsx:37-41` (`toCsvString` — `typeof === 'string'` early return, `typeof === 'number' && Number.isFinite` → `String(value)`, else `undefined`). Do not invent a divergent signature.

- [ ] **Task 2: Update the `Last Updated` mega-line per Rule 25** (AC: #5)
  - [ ] 2.1 PREPEND a new English-only `(...)` entry to L7 leading the `**Last Updated:**` line, e.g.: `2026-06-09 (retro-11-AI2 — Epic 11 retro action item (MED): SM Bob + DEV authored **Rule 26 (TanStack Router Search-Param Coercion)** … lone-numeric `?genre=16` JSON-parsed to number fails `typeof==='string'` guard → silently dropped; coerce via toCsvString/String(). Precedents 11-2 + 8-11. Pure docs — 0 Go, 0 FE, 0 ESLint, 0 baseline.)`.
  - [ ] 2.2 Demote the former lead entry (retro-19-P4) to a `Prior:` entry immediately after the new lead; keep the shared older `Prior:`/`Earlier:` tail exactly once (do NOT duplicate or drop any existing entry).
  - [ ] 2.3 Entry text is English-only (no CJK anywhere on the mega-line — Rule 25 + the known prettier+CJK reflow gotcha).

- [ ] **Task 3: Verification** (AC: #6)
  - [ ] 3.1 `pnpm exec prettier --check project-context.md` → PASS (if it reflows, a CJK char or formatting slip leaked in — fix and re-check).
  - [ ] 3.2 Confirm Rule numbering is contiguous (… 24, 25, 26) and the `## 🧪 Known dev-mode artifacts` section still follows immediately after Rule 26 + `---`.
  - [ ] 3.3 Grep the mega-line for both `retro-11-AI2` (new) and `retro-19-P4` (demoted-but-present) to prove no entry was dropped; entry count ≥ prior count.
  - [ ] 3.4 Run the standard regression gate (`pnpm lint:all`, `pnpm nx test api`, `pnpm nx test web`) — all green; this story changes no code so the result must match the prior baseline (0 errors / ~122 warnings).

- [ ] **Task 4: sprint-status transitions** (AC: #6)
  - [ ] 4.1 `retro-11-AI2-tanstack-numeric-param-note: ready-for-dev` at story creation (this `/create-story` step).
  - [ ] 4.2 `ready-for-dev → in-progress` on `/dev-story` start; `in-progress → review` on completion (note final Rule 26 line range).
  - [ ] 4.3 `review → done` on `/code-review` pass.

## Dev Notes

### Root Cause (Epic 11 Retro, 2026-06-09, Pattern #2)

TanStack Router's default search parser JSON-parses each query value. A lone numeric value (`?genre=16`) becomes the `number` `16`, so a `validateSearch` guard written `typeof search.genre === 'string'` evaluates `false` and the param is dropped — the deep link silently loses its filter. It only reproduces on single-value links (a multi-value `?genre=16,28` stays a string), which is why it slipped past manual testing in 11-2 and was caught only by adversarial E2E/CR. The retro flagged this as a recurring framework gotcha (11-2 + 8-11 = two strikes across two epics) and Insight 3 says: codify the moment a pattern hits twice. This is that codification — a sibling to Rule 5 (TanStack Query) in the frontend-convention set.

### Proposed Rule 26 text (draft — DEV may refine wording, must preserve AC #2–#4)

````markdown
### Rule 26: TanStack Router Search-Param Coercion (lone-numeric trap)

```typescript
// TanStack Router's DEFAULT search parser JSON-parses each query value. A param
// holding a SINGLE numeric value (`?genre=16`, `?platform=8`) is parsed into a
// `number`, NOT a string. A validateSearch guard written `typeof x === 'string'`
// is then false for the lone-numeric case and SILENTLY DROPS the param — the
// deep link looks unfiltered. A multi-value form (`?genre=16,28`) stays a string
// and slips through, so the bug shows ONLY on single-value deep links and is
// easy to miss in manual testing.

// ✅ CORRECT — coerce number → string before the guard (CSV-string params)
function toCsvString(value: unknown): string | undefined {
  if (typeof value === 'string') return value || undefined;
  if (typeof value === 'number' && Number.isFinite(value)) return String(value);
  return undefined;
}
validateSearch: (search) => ({ genre: toCsvString(search.genre) }),

// ❌ WRONG — lone `?genre=16` arrives as number 16, guard false, param dropped
validateSearch: (search) => ({
  genre: typeof search.genre === 'string' ? search.genre : undefined,
}),
```

Applies to every `validateSearch` under `apps/web/src/routes/**` whose param can
be written as a bare number in a deep link (CSV id-lists: genre, platform, person
ids; any single-id filter). Canonical helper: `apps/web/src/routes/discover.tsx`
`toCsvString`. Genuinely-numeric params use a `toOptionalNumber`-style coercion
instead; a string-enum guard (e.g. `subtitleStatus`) is only safe when the value
can NEVER be all-digits.

📌 Precedent: recurred twice. Story 11-2 (persistent filter chip UI) — `?genre=16`
/ `?platform=8` single-value deep links silently dropped the filter; fixed with
`toCsvString()` + defensive `String()` coercion, E2E-guarded (CR caught it as a
HIGH). Story 8-11 (batch subtitle UI) — same class on the `subtitleStatus` param.
Two strikes across two epics → codified here per Epic 11 Retro Insight 3 / Rule 22
(codify a framework gotcha the moment it hits twice).
````

### Why a doc rule, not an ESLint rule (scope)

The retro scoped AI-2 to a **note** (MED), not automation — consistent with how retro-10-AI2 deferred grep-automation because the judgement ("is this param numeric-writable?") needs human context. A lint rule that flags every `typeof === 'string'` guard on a search param would false-positive on genuine string enums. The existing route-param review naturally covers it now that the rule is documented. Automation is explicitly out of scope (a possible future enhancement, not this story). This also matches the pure-docs nature of sibling Rules 24/25.

### Why no CR-workflow sync (contrast with Rules 20/24/25)

Rules 20, 24, 25 each got a paired `code-review/instructions.xml` Step 3 MANDATORY check because they govern cross-story **contracts** / process invariants where a silent miss strands other work. Rule 26 is a single-file frontend coding convention (sibling to Rules 5/6/18) — a miss is a self-contained bug caught by the route's own E2E, not a cross-story landmine. So no CR sync is added, keeping this MED-priority story tight. (If route-param drops recur a third time despite the rule, escalating to a CR check or lint rule becomes the follow-up.)

### Mega-line hazard (Rule 25 + known gotcha)

L7 is a single physical line. The new entry MUST be prepended English-only; a CJK character makes prettier reflow the entire line and can mask a dropped entry. Demote retro-19-P4 to `Prior:`, keep the shared tail once, then `prettier --check` + grep both IDs. This is the exact failure Rule 25 exists to prevent — follow it to the letter.

### Project Structure Notes

- Only `project-context.md` is edited (Rule 26 body + L7 mega-line). No code, no workflow files, no `.pen`, no screenshots.
- Rule 26 slots between Rule 25 (ends ~L1054) and the `---` / `## 🧪 Known dev-mode artifacts` (~L1056) — keeps the numbered block contiguous.

### Time-dependent visual coverage

- **Does this story add/modify any `apps/web/src/components/**/*.{ts,tsx}` that reads `Date.now()` / `new Date()` / `Date.UTC()` / `Date.parse()`?**
  - **NO** — `N/A — no wall-clock-reading components touched`. This story edits only `project-context.md`; zero components.
- Reference: `project-context.md` Rule 23.

### References

- [Source: `_bmad-output/implementation-artifacts/epic-11-retro-2026-06-09.md#challenges`] Pattern #2 — TanStack Router numeric search-param coercion gotcha (recurred 11-2 + 8-11)
- [Source: `_bmad-output/implementation-artifacts/epic-11-retro-2026-06-09.md#action-items`] retro-11-AI2 row (SM, MED)
- [Source: `apps/web/src/routes/discover.tsx:34-41`] canonical `toCsvString` helper + its explanatory comment (the ✅ pattern Rule 26 codifies)
- [Source: `apps/web/src/routes/library.tsx:118,122`] `genres` (CSV) + `subtitleStatus` (8-11) guards — the surfaces the rule governs
- [Source: `project-context.md` L7 + Rule 25 (L1008-1054)] mega-line update protocol
- [Source: `project-context.md` Rule 5 (L254-265) / Rule 18 (L518)] house format for a frontend-convention rule
- [Precedent: `retro-10-AI2-ac-contract-drift-check`] prior retro process/docs story (MANDATORY action) — sibling pattern, though AI-2 here is doc-only with no workflow sync

### Out of Scope

- An ESLint rule enforcing search-param coercion (false-positive-prone on string enums) — deferred; escalate only if a third recurrence appears.
- A `code-review/instructions.xml` CR-workflow sync (Rule 26 is a self-contained convention, not a cross-story contract).
- Auditing/retrofitting every existing `validateSearch` guard in `apps/web/src/routes/**` for latent lone-numeric drops — the rule is forward-looking; `library.tsx::subtitleStatus` is a string-enum (values like `missing`/`none`, never all-digits) so it is safe as-is. If a concrete vulnerable guard is found, file it per Rule 24 (Discovery Triage).

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - Expected default: `N/A — no out-of-scope work discovered`. If, while authoring Rule 26, the DEV greps `apps/web/src/routes/**` and finds a `validateSearch` guard that IS vulnerable to the lone-numeric drop (a numeric-writable CSV/id param still using a bare `typeof === 'string'` guard), file it per Rule 24 lane ③ (`backlog`/`bugfix-N` entry, bidirectional link) at discovery time — do NOT silently fix it inside this docs story.
- Reference: `project-context.md` Rule 24.

### File List

## Change Log

| Date | Change | Author |
| ---- | ------ | ------ |
| 2026-06-09 | Story created (SM Bob /create-story, YOLO) — backlog → ready-for-dev. Pure docs: add `project-context.md` Rule 26 (TanStack Router lone-numeric search-param coercion trap) + L7 mega-line update per Rule 25. Precedents 11-2 + 8-11. No code / no ESLint / no CR sync. | Bob (SM) |
