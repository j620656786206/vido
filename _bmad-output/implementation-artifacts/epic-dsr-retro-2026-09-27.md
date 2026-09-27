# Epic DSR Retrospective — Design-System Reconciliation（2026-09-27）

Facilitator: Bob (SM) · Project Lead: Alexyu · Participants: Sally (UX), John (PM), Amelia (Dev), Murat/Dana (QA view)

## 1. Epic summary

- **Scope:** one reconciliation story per UX flow (A B C D E F H I J K L M N; no Flow G) — make the shipped app match `ux-design.pen` v2 and fix what the audit surfaced along the way.
- **Delivery:** 13/13 flow stories done; 38 story files in total (5 flows split: dsr-1 → 1b-a/a2/b/c, dsr-2 → 2b-a/2b-b, dsr-3 → 3a–3f, dsr-4 → 4b-1/4b-2, dsr-6 → 6a–6f with 6d → a/b/c and 6d-c → c-1/c-2). PRs #413 → #535.
- **Follow-up chain after the epic (same week):** custom posters served (#536) → upload entry design + dialog v2 (#539) → poster field (#542) → orphan sweep (#545) → backups include posters (#547) → **P0 restore fixed on real databases (#549)** → editor year no longer truncates release dates (#551) → day-theme text on artwork + CI guard (#553).
- **Discoveries:** ~129 `disc-2026-09-*` entries filed inside the epic block (113 backlog / 16 done at retro time).

## 2. What went well

1. **Adversarial `/ship` review found real product bugs, not nits** — dsr-5 (both scan-complete links broken, counts false), dsr-13 (API keys entered in the wizard were never used), dsr-6d-b (batch stuck "running", selection eaten), dsr-6d-c-1 (terminal results lost on tab switch), dsr-3f (table clipped buttons at 768px), dsr-1b-a (search ignored filters → dsr-1b-a2).
2. **Split by layout, not "behaviour first, looks later"** (`feedback_split_oversized_stories`) — every sub-story shipped independently and merged cleanly.
3. **Feature-branch baseline bootstrap** (`gh workflow run "Visual Regression" --ref <branch>` → merge the bootstrap PR back) became routine from dsr-2 on.
4. **Mutation checks as standard practice** — recorded in 26 of 46 story files; several caught tests that could not fail.
5. **Discoveries filed at the moment of finding** turned one poster bug into a traceable chain that ended by fixing a P0 nobody knew about (restore).

## 3. What did not go well (patterns)

1. **Fixtures / mocks too simple hid real bugs (8+ cases).** dsr-2 (every real TMDb 404 became 500; test mocked `ApiError(404)`), dsr-2b-a (timeout path green only under mocks), dsr-6d-c-2 (autoscroll passed only with fake `scrollHeight`), dsr-6f-2 (e2e fed three short terms; a long term broke the row), dsr-1 (hand-written labels), and the worst: **bugfix-restore-fails-on-real-database** — restore tests used a single un-keyed `test_data` table, so restore was broken on every real database and nobody knew until a local smoke run.
2. **Tests that could not fail / first-round mutations that stayed green (~10).** dsr-6d-a, 6d-c-1, 6e-1, 6e-2, 6f-2, 6f-3, dsr-7, dsr-3f, poster-upload-a (`pointerDown` never reached Radix), bugfix-custom-posters (whitelist `.*`).
3. **Story Context "facts" wrong at dev time (5+ stories).** dsr-2, dsr-7 (3 AC errors), dsr-10 (3 of 4 AC #7 items), dsr-11, dsr-4b, poster-upload-b ("`Validate()` never called" — it was).
4. **Friction that repeats every story:** `-linux` baseline bootstrap (~25 bootstrap PRs), Pencil save ritual and schema gotchas, gh account flipping to `alexyu-tvbs`, flaky network to github.com.
5. **Previous retro follow-through was zero.** All four action items of `epic-subtitle-pipeline-m2-5-retro-2026-08-12.md` are still backlog:
   - ⏳ retro-m2-AI2 (gh account guard) — practised via memory, never written into `/ship`.
   - ⏳/❌ retro-m25-AI1 (e2e in AC-drift sweep) — recurred in dsr-4 (CI e2e red on stale copy).
   - ⏳ retro-m25-AI2 (document bootstrap path) — replaced in practice by the feature-branch flow, still undocumented.
   - ❌ retro-m2-AI4 (InstantSearchBar flake) — no recurrence recorded in this epic.

## 4. Key insights

1. **"Green" only means something if the fixture has the real shape.** Every serious miss this epic was a test that ran against a simpler world than production.
2. **A story's 🔴 facts are hypotheses until code is read at file:line.** Write them that way.
3. **Retro action items that live only in memory or a doc do not happen.** They must change a skill, a workflow file, or a CI check.

## 5. Action items (tracked in sprint-status under "Epic DSR Retro Action Items")

| ID | Action | Owner / route | Priority | Done when |
| --- | --- | --- | --- | --- |
| retro-dsr-AI1 | Real-shape fixture rule: any AC touching persistence or an external contract needs ≥1 test on the real shape (migration-backed DB, real error from the real handler) — add to `project-context.md` and the dev-story checklist | SM → QD | HIGH | rule text in both files; referenced by next dev-story run |
| retro-dsr-AI2 | Write gh-account guard (`GH_TOKEN=$(gh auth token --user j620656786206)`) and push/PR network retry into `.claude/skills/ship/SKILL.md` (carries retro-m2-AI2) | QD | HIGH | SKILL.md updated; retro-m2-AI2 closed |
| retro-dsr-AI3 | Add `tests/e2e` to the AC-drift sweep in `dev-story/instructions.xml` (carries retro-m25-AI1) | QD | MED | instructions mention e2e grep; retro-m25-AI1 closed |
| retro-dsr-AI4 | create-story: every 🔴 fact must cite file:line verified by reading code (not an agent summary) | SM | MED | create-story instructions/checklist updated |
| retro-dsr-AI5 | Document the feature-branch `-linux` bootstrap flow in `/ship` (replace the old admin-merge text; carries retro-m25-AI2) | QD | MED | SKILL.md updated; retro-m25-AI2 closed |
| retro-dsr-AI6 | Status hygiene: fix drifted entries (`library-search-ignores-filters` backend shipped in #511; stale epic headers) and triage the 113 backlog `disc-2026-09-*` into P0–P3 order | SM | LOW | drift fixed; triage list in sprint-status |

retro-m2-AI4 (InstantSearchBar flake) stays backlog unchanged — no recurrence evidence this epic.

## 6. Open technical debt carried forward (selection)

`disc-2026-09-type-scale-even-migration` (P0, needs rulings), `disc-2026-09-code-still-uses-10px` (P1), `disc-2026-09-rule21-node-existence-unguarded`, `disc-2026-09-batch-reparse-never-runs`, `disc-2026-09-solo-run-unwritable-folder-pays-asr`, `disc-2026-09-manual-search-hides-source-errors`, `disc-2026-09-unmatched-filter-vs-parse-status`, `disc-2026-09-metadata-retry-queue-never-writes`, `disc-2026-09-setup-complete-claims-done-before-submit`, `infra-vr-pr-bootstrap-gap`.

## 7. Readiness

- Testing & quality: all flows shipped with CI green; the real-shape gap is addressed by AI1.
- Deployment: merged to main; production NAS runs the released image.
- Blockers: none for starting new work. No next epic is defined yet (candidates: subtitle `sub-7-x` stories, ready-for-dev).
- **Epic update required:** No.

## 8. Commitments

6 action items (2 HIGH), 0 preparation tasks, 0 critical-path blockers. Review action items before the next epic starts.
