# Spike: AC Contract Versioning (retro-10-AI5)

**Date:** 2026-04-22
**Source:** BMAD Party Mode session — Winston (Architect), Mary (Analyst), Bob (SM)
**Time budget:** ≤2h (per AI-5 row in `_bmad-output/implementation-artifacts/sprint-status.yaml`)
**Purpose:** Spike research output to feed `sm /create-story` for `retro-10-AI5-cross-story-contract-versioning`. Documents the 3 options considered, trade-off matrix, drift-type coverage analysis, and the three decision points resolved. Acts as the source-of-truth artifact for the downstream story's Dev Notes.

---

## Problem Statement

Epic 10 retro Pattern #2: **cross-story AC drift**. When Story A defines an AC that specifies a contract (endpoint shape, request count, path, etc.) and Story B later references that AC as a precondition, the AC can silently change shape between Story A's landing and Story B's execution — breaking B without any signal.

- `retro-10-AI2` landed the dev-story Step 2 **AC Drift Check** (greps for story-ID references). Reactive-to-proactive step 1.
- `retro-10-AI4` (just completed 2026-04-22) caught a meta-example: the anti-drift story itself cited a wrong precedent path. Pattern is recurrent enough to need a second defense layer.

**AI-5 scope:** decide whether/how to version ACs to surface drift at story-authoring time.

---

## Options Considered (Winston's matrix)

### Option 1: Header stamp in AC text

Each cross-referenced AC carries a `[@contract-v1]` prefix.

```
AC #3 [@contract-v1]: POST /api/v1/foo returns {id, status, createdAt}
```

- Grep pattern: `grep -nE '\[@contract-v[0-9]+\]' story.md`
- Bump protocol: author manually bumps to `[@contract-v2]` + Change Log entry when contract changes
- Downstream acknowledgement: referencing stories must record `confirmed against [@contract-vN]` in Dev Notes

### Option 2: Test-snapshot tag binding AC to test hash

Tests tag themselves with `AC:X-Y.Z contract:v1`. CI check enforces every tagged AC has a test with matching version; shape drift without bump → CI fails.

### Option 3: Defer

Continue using retro-10-AI2 AC Drift Check + adversarial CR as the sole defense.

---

## Trade-off Matrix

| Criterion                 | Opt 1: Header stamp     | Opt 2: Snapshot tag                  | Opt 3: Defer            |
| ------------------------- | ----------------------- | ------------------------------------ | ----------------------- |
| **Upfront cost**          | 🟢 0.5h (convention doc) | 🟡 2–3h (CI + tag parser + migration) | 🟢 0h                    |
| **Per-story cost**        | 🟡 1 extra token per AC  | 🟡 Tag + snapshot dual-write          | 🟢 0                     |
| **Enforcement**           | 🔴 Honor-system + grep   | 🟢 CI automatic                       | 🔴 Reviewer eyes only    |
| **Retro noise (forward)** | 🟡 Medium                | 🟢 Low (CI catches)                   | 🔴 High (3 epics precedent) |
| **False-positive risk**   | 🟢 Zero                  | 🔴 High (dynamic fields, fragile)    | 🟢 Zero                  |

**Observation:** Opt 2's 2–3h upfront already exceeds the AI-5 ≤2h spike budget — this itself is a signal that Opt 2 is engineering-over-solution territory.

---

## Drift-Type Coverage (Mary's retro-backed validation)

Cross-referenced against `_bmad-output/implementation-artifacts/*.md` retro history:

| Drift Type                                    | Historical instances                                        | Opt 1 catches? | Opt 2 catches? | Opt 3 catches? |
| --------------------------------------------- | ----------------------------------------------------------- | -------------- | -------------- | -------------- |
| **A) Path rename cross-story**                | Story 10-4 `/movies/check-owned → /media/check-owned`       | ✅ grep hit on `@contract-v1` reference | ⚠️ requires path in snapshot | ❌ retro-only |
| **B) Handler lives on unexpected file**       | retro-10-AI4 H1 (meta-irony)                                | ❌ not AC-level | ❌ not AC-level | ❌ retro-only (CR responsibility) |
| **C) Response shape cross-story drift**       | Epic 8 TD3/TD4                                              | ⚠️ author must remember to bump | ✅ CI automatic | ❌ retro-only |
| **D) Request-count contract**                 | Story 10-5 lazy-load 1 vs N                                 | ✅             | ✅             | ❌ (retro-10-AI1 pre-flight partially covers) |
| **E) Error code prefix**                      | retro-10-AI3                                                | N/A            | N/A            | N/A (Rule 7 CR check handles) |

**Key insight (Mary):** Honor-system adherence in BMAD pipeline historically converges to ~100% when tooled + checklisted. retro-10-AI2 bootstrap audit: "24 hits across 13 files, all REUSE not DRIFT" — the checklist-forced grep converts informal discipline into tooled discipline. **Adherence is not a function of author diligence; it is a function of zero-friction tooling.**

---

## Consensus Recommendation

**Opt 1 (Header stamp) + grep helper integration into dev-story Step 2.**

- **Total cost:** ~1.5h (0.5h spike doc already sunk + 30min checklist patch + 30min dev-story/instructions.xml Step 2 extension). Within AI-5 budget.
- **Coverage:** drift-types A + C + D. Type B is CR's domain (retro-10-AI4 H1 precedent). Type E already closed by Rule 7 check.
- **Enforcement strength:** Medium-high — honor-system + grep = tooled honor-system.
- **Forward retro noise:** expected near-zero.

**Opt 2 rejected** for engineering-over-solution (2–3h + CI infra + fragile snapshot maintenance outweighs the gap it closes).
**Opt 3 rejected** — Pattern #2 has recurred 3 times across 3 epics; purely reactive defense is empirically insufficient.

---

## Resolved Decision Points

### DP1: `@contract-vN` placement — **PREFIX with square brackets**

Consensus format: `AC #N [@contract-v1]: Given/When/Then...`

- Rationale: grep-friendly (`\[@contract-v[0-9]+\]` is zero-ambiguity), human-scannable on the left, survives multi-line ACs.
- Square brackets chosen over parentheses to avoid collision with natural-language parentheses in AC text.

### DP2: Bump trigger — **MANUAL + mandatory Change Log entry + downstream acknowledgement**

Bump protocol:

1. AC text `[@contract-v1]` → `[@contract-v2]`
2. Change Log MUST carry entry: `| Date | [@contract-v1→v2] AC #N: {what changed, what breaks}`
3. Downstream stories referencing the AC MUST record `confirmed against [@contract-vN]` in Dev Notes

- Rationale (Mary): bump is a **declaration** of breaking change, not a silent update. Forces author intentionality.
- Rationale (Bob): third rule is the critical one — prevents the "author bumped, but 3 downstream stories didn't resync" second-order drift.

### DP3: Historical AC retrofit — **NO retrofit. Forward-only. Historical = implicit `v0 frozen`**

- Cost avoided: ~250+ manual stamps across Epic 1–10.
- Invariant: un-stamped AC ≡ `@contract-v0` (frozen). Any bump to `v1` is a breaking-change event requiring Change Log.
- Fallback (Bob): when a NEW story references a historical un-stamped AC, the new story's author may stamp the historical AC `[@contract-v1]` in-place as part of the new story's scope (OR defer to a follow-up). Natural-attrition retrofit — zero cost ballooning.

---

## Draft Story Shape (for `/create-story` to refine)

```markdown
# Story: AC Contract Versioning (retro-10-AI5)

Status: ready-for-dev   (post-/create-story will upgrade)

## Story

As a Dev Agent / CR reviewer reading a story that references an AC from
another story as a contract baseline,
I want each cross-referenced AC to optionally carry a `[@contract-vN]`
version stamp + mandatory Change Log on bump + downstream acknowledgement,
so that cross-story AC drift (Pattern #2 from Epic 10 retro) is caught
at story-authoring time via grep, not at retro time via forensics.

## Acceptance Criteria

1. Given a story author writing an AC that defines a contract (endpoint
   shape, request count, path, etc.), when the AC lands for the first time,
   then the AC text MAY carry a `[@contract-v1]` prefix (MAY not MUST — only
   cross-referenced ACs need the stamp; rest remain implicit `v0`).

2. Given an AC already stamped `[@contract-vN]`, when the author changes
   the contract shape/semantics, then (a) the stamp MUST bump to `v(N+1)`
   AND (b) the Change Log MUST carry an entry formatted:
   `| Date | [@contract-vN→v(N+1)] AC #N: {what changed, what breaks}`.

3. Given a downstream story references an AC from another story as a
   precondition ("per Story X-Y AC #N"), when the downstream story is
   authored, then Dev Notes MUST include `confirmed against [@contract-vN]`
   — a non-silent acknowledgement of which version was validated.

4. Given the grep helper `grep -nE '\[@contract-v[0-9]+\]' <file>`, when
   run against any story file, then it lists every stamped AC with line
   numbers — used in dev-story Step 2 AC Drift Check (retro-10-AI2) as
   an additional pass.

5. Given dev-story/instructions.xml Step 2 AC Drift Check action today
   greps for story-ID references, when this story lands, then Step 2
   MUST also run the `\[@contract-v[0-9]+\]` grep and record a
   `📎 Contract Stamps: {found/none/N/A}` line in Completion Notes
   (three-state rule, matches retro-10-AI2 pattern).

6. Given the retrofit strategy is forward-only, when a new story would
   reference a historical (unstamped) AC, then the author MUST either
   (a) stamp the historical AC `[@contract-v1]` in-place as part of the
   new story's scope, OR (b) defer and file a follow-up. Historical ACs
   are implicitly frozen (`v0 implied`).

## Tasks / Subtasks

- [ ] Task 1: Update `_bmad/bmm/workflows/4-implementation/dev-story/instructions.xml`
      Step 2 to append `\[@contract-v[0-9]+\]` grep to the AC Drift Check
      action + `📎 Contract Stamps` Completion Notes line (AC #4, #5)
  - [ ] 1.1 Append grep command to existing drift-check action block
  - [ ] 1.2 Add `{{contract_stamps_result}}` binding (mirrors retro-10-AI2
        three-state pattern: FOUND / NONE / N/A)
  - [ ] 1.3 xmllint PASS verification
- [ ] Task 2: Extend `project-context.md` with a new sub-section (SM to judge
      placement — likely new sub-section under Rule 15 OR new Rule 17)
      titled "AC Contract Versioning" documenting the `[@contract-vN]` stamp
      shape + bump rule + downstream ack rule + forward-only retrofit
      (AC #1, #2, #3, #6)
- [ ] Task 3: Add Change Log hygiene + grep helper docs to
      `_bmad/bmm/workflows/4-implementation/dev-story/checklist.md` under
      `## ✅ Implementation Completion` (AC #2, #3)
- [ ] Task 4: Full regression gate (pnpm lint:all, nx test api, nx test web)
      — docs-only story, zero code regression expected (matches retro-10-AI2/AI-3/AI-4 precedent)

## Dev Notes

### Spike Output Reference
This story's scope, ACs, and tasks are derived from:
`_bmad-output/implementation-artifacts/spike-10-AI5-ac-contract-versioning.md`
(BMAD Party Mode session 2026-04-22, Winston + Mary + Bob).

### Why MAY not MUST on AC #1
Over-stamping creates noise. Only cross-story-referenced ACs need versioning.
Empirical baseline from Epic 8–10: ~15% of ACs are cross-referenced. A MUST
rule would force ~85% unnecessary stamps, net-negative cost.

### Historical ACs stay unstamped
Retrofit cost estimated 250+ manual stamps for zero immediate value
(historical ACs are frozen by definition — they aren't being referenced by
new stories today unless by a specific new-story author's choice).
Forward-only strategy: stamp on reference.

## References
- `_bmad-output/implementation-artifacts/spike-10-AI5-ac-contract-versioning.md` — this spike doc (party-mode decisions + matrix + coverage analysis)
- `_bmad-output/implementation-artifacts/retro-10-AI2-ac-contract-drift-check.md` — sibling AC Drift Check at dev-story Step 2 (pattern to mirror)
- `_bmad-output/implementation-artifacts/retro-10-AI4-http-route-client-method-gap.md` — sibling Rule 15 extension precedent (shape + checklist pattern)
- `_bmad-output/implementation-artifacts/epic-10-retro-2026-04-20.md` — Pattern #2 source
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — `retro-10-AI5-cross-story-contract-versioning: backlog` entry (to transition ready-for-dev → in-progress on dev-story start)
```

---

## Out of Scope

- **Automation of contract-hash computation (Opt 2).** Rejected on cost. Revisit if Opt 1 + grep proves insufficient after 2 epics of forward data.
- **Cross-system contract versioning (API consumers, SDK bindings).** This spike targets internal story-to-story AC references only.
- **Retrofit of Epic 1–10 historical ACs.** Explicitly out per DP3 resolution.

## Next Action

SM (Bob) runs `/bmad:bmm:workflows:create-story` with:
- **Source doc:** this spike file
- **Output:** `_bmad-output/implementation-artifacts/retro-10-AI5-ac-contract-versioning.md`
- **Sprint-status transition:** `backlog → ready-for-dev` on story file save
