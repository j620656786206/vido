# Sprint Change Proposal — 2026-05-20

**Workflow:** correct-course (BMM 4-implementation)
**Triggered by:** dev-story HALT on Story `19-7-github-actions-quota-warning`
**Author:** Amelia (Developer Agent)
**Mode:** Incremental
**Status:** Approved by user (Alexyu) — all six edit proposals approved 2026-05-20

---

## Section 1 — Issue Summary

### Problem statement

Story `19-7-github-actions-quota-warning` (epic-19, status `ready-for-dev`) cannot be
implemented as written. Its central mechanism — fetching the live remaining-credit
count by calling `testsprite_check_account_info` "via the CLI" — does not exist, and
the story was drafted against a contract that has since been superseded.

### Discovery context

The defect was caught by the dev-story workflow's Step 2 **AC Contract Drift Check**
(Epic 10 Retro AI-2 control) — *before* any code was written — when the Developer
Agent began executing the story.

### Root cause

Story 19-7 was drafted **2026-05-18**. Its sibling story 19-6 then completed a CR
closeout on **2026-05-19** (commit `8094ca4`) that bumped 19-6 AC #2/#3/#4 from
`[@contract-v1]` to `[@contract-v2]`. Story 19-7 still references 19-6's superseded
v1 contract throughout.

### Evidence (three concrete conflicts)

1. **BLOCKING — `testsprite_check_account_info` is not a CLI subcommand.**
   19-7 AC #1/#4/#7 + Task 1 require calling `testsprite_check_account_info` via
   `npx @testsprite/testsprite-mcp@0.0.37`. 19-6's Phase 2 implementation discovery
   proved the bare CLI exposes only `generateCodeAndExecute` and `server`;
   account-info is reachable **only** through the MCP-server-mode tool interface.
   - Evidence: `.github/workflows/testsprite-monthly.yml:80-89` (CLI-shape discovery
     comment block); `project-context.md:1178` ("MCP-server-only tool, not a
     bare-CLI subcommand").

2. **`schema_version` hard-coded to 1.**
   19-7 Dev Notes "Dependency" states `schema_version = 1`. The actual queue file is
   `schema_version: 2`.
   - Evidence: `_bmad-output/audit/testsprite-queue.yaml:47`.

3. **`credits_consumed` rendered in the Issue body is always null in v2.**
   19-7 AC #3's Issue body template renders `{last_run.credits_consumed}`. In 19-6
   `[@contract-v2]` the `credits_at_start` / `credits_at_end` / `credits_consumed`
   fields are permanently null. 19-7's own Dev Notes fallback (b) — "derive remaining
   credits from `last_run.credits_at_end`" — is dead for the same reason.
   - Evidence: `_bmad-output/audit/testsprite-queue.yaml:55-57` and the queue-file
     header comment `:10-22`, which explicitly earmarks a future story "19-7a" for
     live account-info querying.

---

## Section 2 — Impact Analysis

### Epic Impact

- **epic-19 (Design-Implementation Drift Audit), `in-progress`:** completes as
  planned. Its goal (drift-prevention CI) is unaffected. 19-7 is one of 8 stories and
  needs only a re-scope — no new epic, no epic removal, no resequencing.
- **19-8 (`comprehensive-component-sweep`, `ready-for-dev`):** the only remaining
  epic-19 story. Unrelated to TestSprite quota — **no impact**.

### Story Impact

- **19-7 itself:** AC #2/#3/#4/#6/#7 + Tasks 1–2 + Dev Notes rewritten; AC
  #1/#5/#8/#9/#10 kept (or trivially adjusted). See Section 4.
- **19-6:** **no change.** 19-6 `[@contract-v2]` is the correct, shipped state — 19-7
  is the artifact that drifted, not 19-6. Rollback of 19-6 was evaluated and rejected.
- **Optional future `19-7a`:** live-credit alerting (the MCP-stdio dance) is deferred,
  not required. Out of scope for this proposal.

### Artifact Conflicts

| Artifact | Conflict? | Notes |
|----------|-----------|-------|
| PRD | No | 19-7 is epic-19 CI tooling, not a PRD feature; no MVP-scope impact |
| Architecture | No | No system component / data model / API contract change |
| UI/UX | No | 19-7 AC #5 keeps `ux-design.pen` untouched; zero UI change |
| CI / workflows | **Yes** | `.github/workflows/testsprite-quota-warning.yml` design changes |
| `project-context.md` | **Yes** | Task 3 doc addendum content changes to watchdog wording |
| `testsprite-queue.yaml` | Read-only | 19-7 must read `schema_version: 2` (no writes) |

### Technical Impact

- **Net simplification.** The re-scope *removes* the TestSprite CLI install, the
  `pnpm install` step, and the `TESTSPRITE_API_KEY` secret dependency. 19-7 becomes a
  pure repo-file reader + GitHub-Issue writer.
- **Narrowed contract dependency.** 19-7 now depends only on 19-6 `[@contract-v2]`
  AC #2 (queue-file `schema_version: 2` + `last_run` block shape). The dependency on
  19-6 AC #4/#5/#6 (live credits + secret surface) is removed.
- **No downstream consumers of 19-7's own contract exist yet** — re-drafting its
  `[@contract-v1]` stamps in place is safe (no version bump required; the story never
  shipped).

---

## Section 3 — Recommended Approach

### Path forward evaluation

| Option | Verdict | Effort | Risk |
|--------|---------|--------|------|
| **1. Direct Adjustment** — re-scope 19-7 within epic-19 | ✅ **Selected** | Medium | Low |
| 2. Rollback — revert 19-6 | ❌ Not viable — would revert correct, shipped work | — | — |
| 3. PRD MVP Review | N/A — epic-19 is not PRD-MVP scope | — | — |

### Selected approach: Direct Adjustment + Mechanism C

Within Direct Adjustment, three mechanisms for obtaining 19-6's health signal were
evaluated:

- **Mechanism A — MCP-stdio dance:** run `npx ...@0.0.37 server` and call the
  `testsprite_check_account_info` MCP tool over JSON-RPC. Preserves the literal
  credit-threshold semantics. Rejected for now: higher complexity, unknown tool
  name/params, unknown whether account-info consumes credits.
- **Mechanism B — derive from the queue file:** **dead** — v2's `credits_*` fields
  are all null; the queue file carries no credit numbers at all.
- **Mechanism C — watchdog on `last_run.status` + run staleness:** **SELECTED by
  user (Alexyu).** Uses only data provably present in v2. Robustly covers 5 of the 6
  19-6 failure modes the story enumerates.

### Rationale

- Mechanism C uses only the v2 fields that demonstrably exist; zero MCP-stdio unknowns.
- 5 of 6 failure modes 19-7's user story enumerates are detectable from
  `last_run.status` + staleness (only mode (iv) "manual consumption starved the
  budget" needs a live credit count — deferred to a potential future `19-7a`).
- The queue-file header comment already earmarks live-credit querying as a separate
  "19-7a" — Mechanism C aligns with the project's own documented intent.
- `project-context.md:1199` already delegates regression alerting to 19-7; the
  watchdog framing fulfils that delegation directly.

### Effort / Risk / Timeline

- **Effort:** Medium — single-story rewrite, no new code subsystems; the workflow is
  *simpler* than the original design.
- **Risk:** Low — no live external API; validation (`workflow_dispatch`) is a pure
  repo-read + Issue-CRUD, safely re-runnable.
- **Timeline:** No epic-19 timeline impact — 19-7 was not yet started; the corrected
  story enters dev-story immediately after this proposal is applied.

---

## Section 4 — Detailed Change Proposals

All changes apply to
`_bmad-output/implementation-artifacts/19-7-github-actions-quota-warning.md`.

### EP-1 — Story title + header marker block

- **Title:** `# Story 19.7: GitHub Actions Month-End TestSprite Monthly-Run Watchdog
  (Design-Drift Audit — Phase 2 CI)`
- **Marker comment** updated to:
  `🔗 AC Drift: FOUND→RESOLVED (correct-course 2026-05-20 — re-scoped against 19-6
  [@contract-v2]; originally drafted against superseded v1) · 📎 Contract Stamps:
  v1×3 this story; consumes ONLY 19-6 [@contract-v2] AC #2 (queue schema_version:2 +
  last_run block); live-credit dependency on 19-6 AC #4/#5/#6 REMOVED · 🔒 Rule 7:
  N/A · 🎨 UX: N/A`

### EP-2 — User Story paragraph

> As a maintainer of the Vido TestSprite integration who set up the monthly consumer
> (19-6), I want a GitHub Actions workflow that runs on the 28th of every month at
> 03:00 UTC, inspects the `last_run` block of `_bmad-output/audit/testsprite-queue.yaml`
> (written by 19-6, schema_version 2), and opens a GitHub Issue when 19-6's last
> monthly run failed (`api-failure` / `test-failures-only`) or did not run at all
> (`last_run` absent or stale beyond 35 days), so that the 19-6 consumer's failure
> modes (auth expired silently, target unreachable, cron skipped, workflow disabled,
> commit-back blocked) surface in the GitHub UI three days before month-end — leaving
> time for a manual `workflow_dispatch` of 19-6 to catch up — instead of a broken
> monthly pass being discovered a month later.

### EP-3 — Acceptance Criteria

**AC #1 — KEPT verbatim** (triggers / runner / job name / concurrency; never
referenced credits).

**AC #2 — REWRITTEN:**

> 2. [@contract-v1] **Issue de-dup logic — at most ONE open watchdog Issue at any
>    time.** The workflow MUST search the repo for an open Issue with the label
>    `testsprite-quota-warning` (EXACTLY this label — the dedup key) BEFORE opening a
>    new one. Behaviour (4 branches; "UNHEALTHY"/"HEALTHY" defined in AC #4):
>    - **No existing open Issue + 19-6 UNHEALTHY:** open a new Issue with the AC #3
>      title/body and apply the `testsprite-quota-warning` label.
>    - **Existing open Issue + 19-6 UNHEALTHY:** post a new comment on the existing
>      Issue with the current health reason + a timestamp + a link to this workflow
>      run — DO NOT open a duplicate.
>    - **No existing open Issue + 19-6 HEALTHY:** no-op; workflow exits cleanly.
>    - **Existing open Issue + 19-6 HEALTHY:** post a final comment
>      `✅ Resolved — 19-6's last_run is healthy again ({status}, finished
>      {finished_at}). Closing.` and CLOSE the Issue.
>
>    The `testsprite-quota-warning` label MUST be created if it doesn't exist
>    (`gh label create testsprite-quota-warning --color FFA500 --description
>    "TestSprite monthly run watchdog (story 19-7)" --force`). Colour `FFA500`
>    (orange) signals "warning, not error".

**AC #3 — REWRITTEN (Issue body template):**

> 3. [@contract-v1] **Issue body format.**
>    - **Title:** `[TestSprite] Monthly run watchdog: {reason} ({YYYY-MM})` — e.g.
>      `[TestSprite] Monthly run watchdog: 19-6 last run failed — api-failure
>      (2026-05)`
>    - **Body** (Markdown HEREDOC):
>
>      ```markdown
>      ## Why this Issue was opened
>
>      The month-end watchdog (`testsprite-quota-warning.yml`, story 19-7) found
>      story 19-6's monthly TestSprite cron in an UNHEALTHY state:
>
>      **{reason}**
>
>      19-6 spends the monthly TestSprite credit budget on journey-test coverage.
>      An unhealthy `last_run` means that coverage did not happen this cycle — and
>      Free-plan credits expire at calendar-month rollover (~{X} days from now) with
>      no carry-forward.
>
>      ## What to do
>
>      1. Fire `workflow_dispatch` on [TestSprite Monthly]({link-to-19-6-workflow})
>         to run the monthly pass now.
>      2. After it completes, re-fire `workflow_dispatch` on this workflow
>         ([TestSprite Quota Warning]({link-to-this-workflow})) to re-check — when
>         19-6's `last_run` is healthy again, this Issue auto-closes.
>      3. If 19-6's run fails, investigate root cause (most common:
>         `TESTSPRITE_API_KEY` rotated/expired, `TESTSPRITE_TARGET_URL` unreachable,
>         GitHub-Actions branch-protection blocking the commit-back push).
>
>      ## Audit trail snapshot
>
>      - **19-6 last_run status:** `{last_run.status | "null — never run"}`
>      - **19-6 last_run finished:** {last_run.finished_at | "never"}
>      - **19-6 last_run test cases:** {last_run.test_ids_run.length}
>      - **19-6 last_run URL:** {last_run.run_url | "n/a"}
>      - **Queue schema_version:** {schema_version}
>      - **Queue file:** [`_bmad-output/audit/testsprite-queue.yaml`]({queue-file-permalink-at-current-sha})
>
>      ## Workflow run
>
>      This Issue was opened by [TestSprite Quota Warning run #{run_id}]({this-run-url}).
>
>      <!-- testsprite-quota-warning-marker: do not edit; consumed by 19-7's dedup logic -->
>      ```
>
>    The trailing HTML-comment marker is a tertiary de-dup hint behind the primary
>    label key.

**AC #4 — REWRITTEN (source of truth + health rule):**

> 4. **The `_bmad-output/audit/testsprite-queue.yaml` `last_run` block
>    (schema_version 2) is 19-7's single source of truth for 19-6's health.** 19-7
>    makes NO live TestSprite API call and installs NO TestSprite CLI — it is a pure
>    repo-file reader + GitHub-Issue writer. The workflow MUST:
>    - Verify `schema_version == 2` via `yq`. Absent or != 2 → treat as a HEALTH
>      FAILURE (19-6's queue contract drifted) → open/maintain an Issue per AC #2; do
>      NOT crash.
>    - Read `last_run.status` (enum: `success | budget-exhausted | api-failure |
>      test-failures-only | null`) and `last_run.finished_at` (ISO-8601 UTC or null).
>    - NOT read `last_run.credits_*` — permanently null in 19-6 `[@contract-v2]`
>      (the bare CLI cannot query live account-info). A future `19-7a` story may add
>      live-credit alerting via an MCP-stdio dance — explicitly OUT of scope here.
>
>    19-6 is judged **UNHEALTHY** (→ warn) when ANY of:
>    - (a) `schema_version != 2`;
>    - (b) `last_run.status` ∈ { `api-failure`, `test-failures-only` };
>    - (c) `last_run.finished_at` is null (never run) OR older than 35 days — one
>      full monthly cycle + slack; covers cron skipped / workflow disabled /
>      branch-protection blocking the commit-back so the queue never updated.
>
>    Otherwise **HEALTHY** (`last_run.status` ∈ { `success`, `budget-exhausted` } AND
>    `finished_at` ≤ 35 days old) → no warn.

**AC #5 — KEPT** (no production / source-tree edits; file list unchanged — it never
referenced the CLI).

**AC #6 — REWRITTEN:**

> 6. **No TestSprite secret — no new secret of any kind.** The re-scope removed the
>    live account-info call, so 19-7 does NOT consume `secrets.TESTSPRITE_API_KEY`
>    and does NOT consume `vars.TESTSPRITE_TARGET_URL`. The ONLY token used is the
>    auto-issued `secrets.GITHUB_TOKEN`, for the `gh` / github-script Issue + label
>    operations. Job-scoped permissions: `{ contents: read, issues: write }` —
>    `contents: read` for `actions/checkout` to read the queue file; `issues: write`
>    for label/Issue CRUD. No `pull-requests:` permission. 19-7 is therefore
>    contract-independent of 19-6's AC #5/#6 secret surface — it depends ONLY on
>    19-6 `[@contract-v2]` AC #2 (queue-file `schema_version: 2` + `last_run` block
>    shape).

**AC #7 — REWRITTEN:**

> 7. **Failure semantics.** Workflow exit code ≠ 0 ONLY on: (a) the `gh` CLI /
>    github-script call to create/comment/close the Issue or create the label fails
>    (e.g. `issues: write` misconfigured); (b) the queue file is absent or
>    unparseable as YAML (genuine repo corruption — distinct from
>    `schema_version != 2`, which is a *detected* drift the watchdog reports via an
>    Issue, exit 0). A detected-unhealthy 19-6 + Issue successfully opened/updated =
>    exit 0 (the watchdog did its job). Healthy 19-6 + no Issue = exit 0 (steady
>    state). The workflow MUST NOT exit ≠ 0 merely because an open Issue already
>    exists.

**AC #8 — KEPT verbatim** (concurrency `{ group: 'testsprite-quota-warning',
cancel-in-progress: false }`).

**AC #9 — KEPT verbatim** (regression + framework hygiene; YAML covered by
`prettier --check .` + `actionlint`).

**AC #10 — KEPT with minor wording change.** Sub-clause (b): the `project-context.md`
"Monthly Cron Workflow (story 19-6)" addendum paragraph now names 19-7's workflow
file, what it watches (19-6's `last_run` health — failure status or >35-day
staleness), and where the resulting Issue surfaces (Issues tab, label
`testsprite-quota-warning`). The original ">30 remaining" threshold wording is
dropped.

### EP-4 — Tasks / Subtasks (full replacement)

> - [ ] Task 1: Author the watchdog workflow file (AC: #1, #2, #3, #4, #6, #7, #8)
>   - [ ] `.github/workflows/testsprite-quota-warning.yml` — name, triggers
>     (cron `0 3 28 * *` + `workflow_dispatch` only), single job on `ubuntu-24.04`.
>     Job-scoped permissions `{ contents: read, issues: write }`. Concurrency
>     `{ group: 'testsprite-quota-warning', cancel-in-progress: false }`.
>   - [ ] Steps: (1) `actions/checkout@v4` (default depth — only needs the queue file
>     at HEAD); (2) **Ensure `testsprite-quota-warning` label exists** —
>     `gh label create testsprite-quota-warning --color FFA500 --description
>     "TestSprite monthly run watchdog (story 19-7)" --force`. NO `pnpm` / Node /
>     TestSprite-CLI install steps — the watchdog is a pure file reader.
>   - [ ] Health-evaluation step (bash + `yq`, pre-installed on `ubuntu-24.04`): read
>     `_bmad-output/audit/testsprite-queue.yaml`; verify `schema_version == 2`; read
>     `last_run.status` + `last_run.finished_at`; compute `healthy`/`unhealthy` per
>     AC #4 (status enum + 35-day staleness); export the verdict + a human-readable
>     reason string.
>   - [ ] Issue-CRUD step via `actions/github-script@v7`: search open Issues with
>     label `testsprite-quota-warning` (`gh issue list --state open --label
>     testsprite-quota-warning --json number,body,title`); execute the AC #2 4-branch
>     decision tree.
>   - [ ] Top-of-file comment block: 28th-of-month cron rationale, AC #4
>     source-of-truth (queue file, schema v2, NO live API), AC #2 dedup-label
>     contract, links to story 19-7 + sibling 19-6, and the correct-course re-scope
>     note (was credit-threshold, now `last_run` watchdog).
> - [ ] Task 2: Validate the workflow (AC: #1, #9)
>   - [ ] `actionlint .github/workflows/testsprite-quota-warning.yml` clean.
>   - [ ] `pnpm exec prettier --check .github/workflows/testsprite-quota-warning.yml`
>     clean.
>   - [ ] Push the workflow on a feature branch; observe it appears in the Actions
>     tab as a valid scheduled workflow.
>   - [ ] Fire `workflow_dispatch` to validate — pure repo-read + Issue-CRUD, no
>     TestSprite call, free + safely re-runnable. Confirm: (a) the label is created
>     idempotently; (b) the dedup search runs; (c) with the current queue
>     (`last_run.status: null`, never run) the watchdog evaluates UNHEALTHY
>     (staleness branch — `finished_at` null) and opens ONE Issue with the AC #3
>     template; (d) re-firing comments on the same Issue and does not duplicate.
>   - [ ] Cleanup the validation Issue (close with a "test run" comment) — or note it
>     will auto-close once 19-6's first real cron run lands a fresh `last_run`.
> - [ ] Task 3: Documentation (AC: #5, #10)
>   - [ ] `project-context.md` `Last Updated` header → add a 19-7 entry.
>   - [ ] `project-context.md` "Monthly Cron Workflow (story 19-6)" sub-section →
>     append a final paragraph describing the watchdog.
> - [ ] Task 4: Close-out regression (AC: #5, #9)
>   - [ ] `pnpm lint:all` → 0 errors, ≤122 warnings.
>   - [ ] `pnpm nx test web` + `pnpm nx test api` pass.
>   - [ ] `pnpm test:e2e --list` → 1663 unchanged.
>   - [ ] `pnpm run test:visual` green against committed baselines.
>   - [ ] `pnpm run test:cleanup` no orphans.
>   - [ ] `ux-design.pen` untouched → screenshot workflow not triggered.
>   - [ ] Sprint-status entry: `ready-for-dev` → `in-progress` → `review`.

### EP-5 — Dev Notes (targeted rewrites)

1. **"Why this story exists" — 6 failure modes:** annotate that the watchdog covers
   modes **(i) auth fail, (ii) target unreachable, (iii) branch-protection blocking
   commit-back, (v) cron skipped, (vi) workflow disabled** via `last_run.status` +
   staleness. Mode **(iv) "manual consumption starved the budget"** genuinely needs a
   live credit count → explicitly marked OUT of scope, deferred to a potential future
   `19-7a`.
2. **"Dependency" section:** re-ack as `Confirmed against [@contract-v2] (Story 19-6
   AC #2)` — `schema_version: 2`, `last_run` block shape including `credits_*` (v2:
   permanently null, NOT read by 19-7). Remove the v1 reference and the AC #4
   commit-message-format cross-link (no longer consumed).
3. **Delete dead content:** the fallback (b) "derive remaining credits from
   `last_run.credits_at_end`" (v2: null), and the "is account-info free?" uncertainty
   paragraph (Mechanism C makes no account-info call).
4. **Add:** the Mechanism C selection rationale, and the 35-day staleness-threshold
   derivation (19-6 runs `0 3 1 * *`, 19-7 runs `0 3 28 * *` → within healthy
   operation the freshest 19-6 run is ≤27 days old when 19-7 fires; >35 days means a
   whole monthly cycle was missed).
5. **Note** that 19-7's first-ever run against the current queue file
   (`last_run.status: null`) will correctly evaluate UNHEALTHY and open one Issue —
   this is a true signal (19-6 has not been wired up / run yet), not a false positive.

### EP-6 — Change Log (new row)

> | 2026-05-20 | correct-course re-scope (dev-story HALT trigger). AC Drift Check
> found 19-7 was drafted 2026-05-18 against 19-6 `[@contract-v1]`; 19-6 CR closeout
> 2026-05-19 (commit `8094ca4`) bumped to `[@contract-v2]`. Three conflicts: (1) AC
> #1/#4/#7 + Task 1 required via-CLI `testsprite_check_account_info` — the bare CLI
> has no such subcommand; (2) Dev Notes hard-coded `schema_version=1`, actual is 2;
> (3) AC #3 body rendered `last_run.credits_consumed`, null in v2. Mechanism C
> selected (Alexyu): 19-7 re-framed from a live-credit threshold to a queue-file
> `last_run` watchdog (status enum + 35-day staleness). AC #2/#3/#4/#6/#7 + Tasks
> 1–2 + Dev Notes rewritten; AC #1/#5/#8/#9/#10 kept (or trivially adjusted). 19-7's
> contract dependency narrowed to 19-6 `[@contract-v2]` AC #2 only (AC #4/#5/#6
> live-credit + secret dependency removed). Live-credit alerting deferred to a
> potential future `19-7a`. No `[@contract-v*]` bump on 19-7's own stamps — never
> shipped, v1 re-drafted in place. Ref: `_bmad-output/planning-artifacts/sprint-change-proposal-2026-05-20.md`. |

---

## Section 5 — Implementation Handoff

### Scope classification: **Minor**

A single `ready-for-dev` story that has never been implemented, with no downstream
consumers of its contract, no epic/PRD/architecture/UX structural change. The
re-scope is fully specified above (exact OLD→NEW). It can be implemented directly by
the development team.

### Handoff

| Recipient | Responsibility |
|-----------|----------------|
| Developer Agent (Amelia) | Apply EP-1…EP-6 to `19-7-github-actions-quota-warning.md`, then resume the dev-story workflow on the corrected story. |
| Owner (Alexyu) | Post-merge: no new secret to wire (the re-scope removed `TESTSPRITE_API_KEY` from 19-7's needs). |

### Sequencing

1. Apply the approved edit proposals to the 19-7 story file.
2. Resume `dev-story` on 19-7 (it stays `ready-for-dev` → dev-story will move it to
   `in-progress`).
3. Implement, validate, and close out per the rewritten Tasks 1–4.

### Success criteria

- 19-7 story file reflects all six approved edit proposals; the AC Contract Drift
  Check passes (drift RESOLVED).
- `.github/workflows/testsprite-quota-warning.yml` makes no live TestSprite call,
  uses no `TESTSPRITE_API_KEY`, passes `actionlint`.
- `workflow_dispatch` validation opens exactly one dedup'd Issue against the current
  queue state and re-comments (not duplicates) on re-fire.
- Regression gate (AC #9) green.

### Out of scope (tracked, not lost)

- **`19-7a`** — live remaining-credit alerting via an MCP-stdio call to
  `testsprite_check_account_info`. Covers 19-6 failure mode (iv) "manual consumption
  starved the budget". Not required for 19-7; create only if/when the user wants it.

---

*Generated by the correct-course workflow — BMM 4-implementation.*
