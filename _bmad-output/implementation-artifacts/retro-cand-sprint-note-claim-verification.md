# Retro Candidate: Verify Factual Runtime Claims in Sprint Notes

Status: backlog

<!-- ⚠️ THIS IS A RETRO CANDIDATE (process asset), NOT a ready-for-dev code story.
     It has NO decided deliverable yet: the disposition (new project-context Rule vs an
     extension of Rule 15/24 vs a checklist note) is RESERVED for ARCH Winston to rule on at
     the next ux3 / Epic-11 retrospective. This document is SM decision-prep — it stages the
     problem, evidence, and options so the ceremony can decide fast. It deliberately does NOT
     author any Rule. Kept `backlog` (NOT promoted) until the retro picks a disposition; at that
     point it converts into a concrete tracked action item (à la retro-8-P1/P2/P3). -->

## What this is (read FIRST)

- **Type:** retro candidate / process-improvement proposal. **Owner of the decision:** ARCH (Winston) at the retro ceremony (run via the SM **ER — Epic Retrospective** workflow). **Owner of this prep doc:** SM (Bob).
- **Why it is not a normal story:** there is no code or doc edit to make *yet* — the proposal itself is sound, but *how* to institutionalize it (Rule / Rule-extension / checklist) is a deliberate fork the retro must choose. Authoring a `project-context.md` Rule now would pre-empt that ceremony, which its own sprint-note forbids.
- **Convert-to-action trigger:** when the retro picks option ①/②/③ below, this entry becomes a concrete tracked item (new `project-context.md` Rule, or an edit to `create-story`/`sprint-planning` instructions, etc.) and follows the normal dev/QD/TW path to `done` — mirroring how `retro-8-P1/P2/P3` were tracked and completed.

## Problem statement

A **factual claim about runtime behavior**, written into a `sprint-status.yaml` note **from memory and never grep-verified**, was **doubly false** — and went unchallenged until a Party-Mode review during a later, otherwise-unrelated story.

> The offending note (paraphrased): *"only the AI cache is swept."*
> Ground truth (verified 2026-06-24): the AI cache is **NOT** swept; the cache that **does** self-sweep is **Douban**. True state: **3 orphaned caches** (`cache_entries`, `ai_cache`, `offline_cache` — each has a `ClearExpired*` method but **zero scheduled caller**) **+ 1 self-sweep** (`douban_cache`, built-in `cleanupLoop` goroutine).

## Root cause (Mary / Analyst)

The note **conflated "the method `ClearExpiredCache` EXISTS" with "it is SCHEDULED / wired / called."** That is the exact **Rule 15 failure class** — *"a client method existing ≠ the HTTP route is registered; assume nothing"* — but manifesting in **narrative prose** (a sprint-note) instead of in code. It is also kin to **Rule 24's prose-only-mention ban** (a claim living only in narrative, untracked/unverified).

## Why it matters (impact evidence)

- The false claim **mis-scoped reality**: had it been believed, `ai_cache` + `offline_cache` would have stayed silently orphaned (no scheduled expiry) indefinitely. It took an unrelated story's Party-Mode review to catch it.
- It is the **sprint-note analogue of a whole family of "exists ≠ wired" hallucinations** the project has already paid for in code (Rule 15 origin: `tmdb.GetMovieVideos` client method existed but its route was never registered, Epic 10 Retro AI-4). The same trust-the-claim error in prose is cheaper to make and harder to catch (no compiler, no test).

## The proposal (what the retro is being asked to ratify)

> **Any factual claim in a `sprint-status.yaml` note (or any planning narrative) asserting RUNTIME behavior — "X is swept / wired / called / scheduled / registered / enabled" — MUST be grep-verified at write time, or explicitly hedged as `unverified:` / `(claim, not verified)`.**

The claim is about *runtime wiring*, not *method existence*; the cheap check is a grep for the **scheduled caller / registration site**, not just the method definition.

## Disposition options (the fork the retro must pick) — SM pre-read

| # | Option | Pros | Cons | SM lean |
| - | ------ | ---- | ---- | ------- |
| **①** | **New numbered `project-context.md` Rule** (e.g. Rule 28 "Sprint-Note Claim Verification") | Highest visibility; joins the canonical Rule set; unambiguous citation in reviews | `project-context.md` is already very large; a *writing-discipline* rule is heavier than a *code-pattern* rule; hard to grep-enforce in code-review | — |
| **②** | **Extend Rule 15** (Pre-commit Self-verification — "method-exists ≠ wired") **with a "narrative-claim corollary"** | Reuses the *exact* existing mental model — the failure **is** Rule 15's class, just in prose; minimal canon growth; one place already teaches "exists ≠ wired" | Rule 15 is framed around code self-verification; adding a prose surface slightly broadens its scope | ✅ **most defensible** |
| **③** | **Checklist note only** (no Rule) — add a one-liner to the SM `create-story` / `sprint-planning` workflow instructions where sprint-notes are authored | Lightest; lives exactly where the bad claim is written; zero Rule-canon bloat | Lowest visibility; checklists are easy to skip; not enforced anywhere | viable fallback |

**SM pre-read recommendation (non-binding — ARCH rules):** **Option ② (extend Rule 15)**. The defect is *literally* the Rule 15 "exists ≠ wired" failure expressed in narrative; a short corollary ("this principle applies to factual runtime claims in sprint-notes / planning prose too — grep the wiring or mark it unverified") reuses the established concept with the least overhead, and Rule 15 already lives in the self-verification headspace. If the retro wants enforcement teeth, pair ② with ③ (a checklist nudge at sprint-note authoring time). Reserve ① for if the retro judges this important enough to warrant top-level Rule visibility.

## Explicitly NOT done now

- **No `project-context.md` Rule authored.** No edit to Rule 15/24. No checklist edit. This doc only *prepares the decision*. (Per the sprint-note: "NOT authoring a project-context Rule now (retro/architect ceremony owns that).")
- This item **does not gate** any in-flight story; it is **non-blocking**.

## Acceptance Criteria (for the eventual action item, AFTER the retro picks a disposition)

> These are intentionally provisional — the chosen option determines which apply. Recorded here so the conversion-to-action is mechanical once the retro rules.

1. The retro selects exactly one disposition (① / ② / ③) and records the rationale in the retro doc.
2. The selected artifact is authored: ① a new numbered Rule in `project-context.md` (+ the "Last Updated" mega-line entry per Rule 25, + any code-review/instructions.xml sync if it becomes grep-checkable); OR ② a Rule 15 corollary paragraph (+ mega-line entry); OR ③ a checklist line in `_bmad/bmm/workflows/4-implementation/create-story/instructions.xml` (and/or `sprint-planning`).
3. The proposal text is captured verbatim: factual runtime claims ("swept/wired/called/scheduled/registered/enabled") in sprint-notes/planning prose must be grep-verified at write time or hedged `unverified:`.
4. This `sprint-status.yaml` entry is updated to the concrete action-item form and driven to `done`.

## Dev Notes

### Why "method-exists ≠ wired" keeps recurring (the pattern to name)

- **Rule 15 (code):** a client method existing ≠ its HTTP route registered (Epic 10 origin: `tmdb.GetMovieVideos`).
- **Rule 15 DB corollary:** a column existing in the model ≠ the repo SELECT/scan loads it (bugfix-20-1: `series.seasons`).
- **This candidate (prose):** a `ClearExpired*` method existing ≠ a scheduled caller invokes it (`ai_cache`/`offline_cache`).
- The throughline: **existence of a capability is not evidence the capability is invoked.** Verifying the *call site / wiring*, not the *definition*, is the cheap fix in all three.

### Process precedent for conversion

- `retro-8-P1` → "Added mandatory tracking rule to `retrospective/instructions.md` step 11"; `retro-8-P2` → "cross-stack split check in `create-story/instructions.xml` step 5"; `retro-8-P3` → "checkbox audit in `dev-story`". These show retro candidates becoming concrete instruction/Rule edits once ruled on — the same path this item will take.

### Time-dependent visual coverage

- **N/A — no `apps/web/src/components` touched.** Pure process/docs proposal.

### References

- [Source: `_bmad-output/implementation-artifacts/infra-cache-entries-expiry-sweep.md` → Discovery Triage] — the originating Party-Mode finding (3 orphaned + 1 self-sweep) and the process root-cause filing.
- [Source: `_bmad-output/implementation-artifacts/infra-ai-offline-cache-expiry-sweep.md`] — the corrective story that proves the claim was false (sibling, ready-for-dev).
- [Source: `project-context.md` Rule 15 (Pre-commit Self-verification — "method-exists ≠ wired" + HTTP-Route↔Client-Method Sync + DB Column Sync)] — the closest existing Rule; Option ② extension target.
- [Source: `project-context.md` Rule 24 (Discovery Triage — prose-only-mention ban)] — kin Rule (a claim living only in narrative).
- [Source: `project-context.md` Rule 25 (Mega-line edit discipline)] — applies if option ①/② edits `project-context.md`.
- [Source: `_bmad/bmm/workflows/4-implementation/retrospective/workflow.yaml`] — the ER ceremony that owns the disposition decision.

## Dev Agent Record

_(Not applicable until the retro rules a disposition and this converts to a concrete action item.)_

### Discovery Triage

- **Did this story discover any work outside its current scope?** **N/A — this IS a discovery-tracking artifact** (a retro candidate filed per Rule 24 ③ from `infra-cache-entries-expiry-sweep`). It introduces no new out-of-scope findings of its own.
- Reference: `project-context.md` Rule 24; origin: `infra-cache-entries-expiry-sweep` Party-Mode review (2026-06-24).

## Change Log

| Date       | Change                                                                                                                                            |
| ---------- | ----------------------------------------------------------------------------------------------------------------------------------------------- |
| 2026-06-30 | SM decision-prep authored (create-story). Problem/evidence/root-cause documented; 3 disposition options laid out with SM pre-read lean (② extend Rule 15). NO Rule authored — disposition reserved for the next ux3/Epic-11 retro (ARCH Winston rules). Status kept `backlog`. |
