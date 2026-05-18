# Story 19.7: GitHub Actions Month-End TestSprite Quota Warning (Design-Drift Audit — Phase 2 CI)

Status: ready-for-dev

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->
<!-- @contract-v1 on AC #1–#3 (workflow trigger model + Issue de-dup contract + Issue body format that Rule 22 retros / future credit-alerting stories may grep). -->
<!-- 🔗 AC Drift: N/A (new CI subsystem) · 📎 Contract Stamps: NEW (v1×3 this story; consumes 19-6 [@contract-v1] AC #2 schema_version + AC #4 commit-message format — see Dev Notes ack) · 🔒 Rule 7: N/A (pure CI / 0 Go) · 🎨 UX: N/A (no `.pen` modification, no UI change) -->
<!-- markers-block-end -->

## Story

As a maintainer of the Vido TestSprite integration who set up the monthly consumer (19-6),
I want a GitHub Actions workflow that runs on the 28th of every month at 03:00 UTC, checks TestSprite's remaining credits via the account-info API, and opens a GitHub Issue when more than 30 credits remain unused,
so that the 19-6 consumer's failure modes (auth expired silently, target unreachable, cron skipped, manual consumption starved the reserve) surface in the GitHub UI three days before month-end — leaving time for a manual `workflow_dispatch` of 19-6 to catch up — instead of credits silently evaporating at the calendar-month rollover and forcing me to discover the gap a month later.

## Acceptance Criteria

1. [@contract-v1] **A new GitHub Actions workflow file `.github/workflows/testsprite-quota-warning.yml` exists** with exactly two triggers: (a) `schedule: - cron: '0 3 28 * *'` (28th of month, 03:00 UTC — chosen because it gives ≥3 days of lead time before month-end credit expiry at 23:59 UTC on the last day, even in February; running on the 30th or 31st risks the cron not firing in February at all — `0 3 31 * *` wouldn't fire in Feb / Apr / Jun / Sep / Nov; the 28th is the safest "near-month-end" cron); (b) `workflow_dispatch` (manual trigger for testing the workflow itself OR for a same-day re-check after firing 19-6's `workflow_dispatch` to catch up). NO `push` / `pull_request` / `schedule:` with other crons. Workflow `name:` is `TestSprite Quota Warning`. Single job `name: TestSprite Quota Warning / Check`. `runs-on: ubuntu-24.04` (pinned per 19-5 / 19-6 AC #5 rationale — deterministic runner; do NOT use `-latest`).

2. [@contract-v1] **Issue de-dup logic — at most ONE open quota-warning Issue at any time.** The workflow MUST search the repo for an open Issue with the label `testsprite-quota-warning` (and EXACTLY this label — used as the dedup key) BEFORE opening a new one. Behaviour:
   - **No existing open Issue + threshold breached** (`remaining_credits > 30`): open a new Issue with the AC #3 title/body and apply the `testsprite-quota-warning` label.
   - **Existing open Issue + threshold breached**: post a new comment on the existing Issue with the updated remaining-credits number + a timestamp + a link to this workflow run — DO NOT open a duplicate. The existing Issue becomes the running thread for the month's quota-alert sequence.
   - **No existing open Issue + threshold NOT breached** (`remaining_credits ≤ 30`): no-op; workflow exits cleanly without opening anything.
   - **Existing open Issue + threshold NOT breached**: post a final comment `✅ Resolved — remaining credits dropped to {N} (≤30 reserved floor). Closing.` and CLOSE the Issue (auto-close — the situation that triggered it has been remedied, either by 19-6's cron consuming credits or by a manual `workflow_dispatch` catching up). This closes the loop without requiring the user to manually close issues each month.
   
   The `testsprite-quota-warning` label MUST be created if it doesn't exist (use `gh label create testsprite-quota-warning --color FFA500 --description "TestSprite monthly credits about to expire (story 19-7)" --force` — `--force` makes it idempotent). The label colour `FFA500` (orange) signals "warning, not error" — consistent with `requires-manual-review` (used by 19-5 bootstrap PR).

3. [@contract-v1] **Issue body format.** The Issue title and body MUST follow this template (the workflow renders placeholders via `gh issue create` or `actions/github-script@v7`):
   
   - **Title:** `[TestSprite] {N} credits unused at month-end ({YYYY-MM})` — e.g. `[TestSprite] 47 credits unused at month-end (2026-05)`
   - **Body** (Markdown HEREDOC):
     ```markdown
     ## Why this Issue was opened
     
     TestSprite reports **{N} credits remaining** as of {YYYY-MM-DD HH:MM UTC}, which exceeds the 30-credit reserved floor (story 19-6 AC #3). These credits expire at calendar-month rollover (~{X} days from now) and cannot be carried forward on the Free plan.
     
     ## What to do
     
     1. Fire `workflow_dispatch` on [TestSprite Monthly]({link-to-19-6-workflow}) — it will consume up to {N - 30} additional credits before stopping at the reserved floor.
     2. After it completes, re-fire `workflow_dispatch` on this workflow ([TestSprite Quota Warning]({link-to-this-workflow})) to verify the threshold dropped — when it does, this Issue auto-closes.
     3. If 19-6's run also fails, investigate root cause (most common: `TESTSPRITE_API_KEY` rotated / expired, `TESTSPRITE_TARGET_URL` unreachable, GitHub-Actions branch-protection blocking the commit-back push).
     
     ## Audit trail snapshot
     
     - **Last 19-6 run status:** `{last_run.status}` ({last_run.finished_at | "never run"})
     - **Last 19-6 run consumed:** {last_run.credits_consumed} credits across {last_run.test_ids_run.length} cases
     - **Last 19-6 run URL:** {last_run.run_url | "n/a"}
     - **Queue file:** [`_bmad-output/audit/testsprite-queue.yaml`]({queue-file-permalink-at-current-sha})
     
     ## Workflow run
     
     This Issue was opened by [TestSprite Quota Warning run #{run_id}]({this-run-url}).
     
     <!-- testsprite-quota-warning-marker: do not edit; consumed by 19-7's dedup logic -->
     ```
     
   The trailing HTML-comment marker is a defensive de-dup hint — `gh issue list --label testsprite-quota-warning --state open` is the primary dedup key per AC #2, but a future maintainer who hand-edits the label off the Issue won't get duplicates because the workflow can also fall back to grepping the body for the marker comment.

4. **`testsprite_check_account_info` is the live source of truth for `remaining_credits`.** The workflow MUST NOT trust the `_bmad-output/audit/testsprite-queue.yaml`'s `last_run.credits_at_end` field as ground truth (it's a snapshot from possibly ~28 days ago, before any manual consumption between cron runs). The audit-trail snapshot in AC #3's body is just context — the threshold decision uses ONLY the live API response. Field naming defensiveness applies (mirrors 19-6 AC #3): probe for `remaining_credits` / `credits_remaining` / `available_credits` and fail-closed if none of the expected fields are present.

5. **No production / source-tree edits.** Created: `.github/workflows/testsprite-quota-warning.yml`. Modified: `project-context.md` Last Updated header (one entry) + the TestSprite Journey Test Workflow section's new "Monthly Cron Workflow (story 19-6)" sub-section (added by 19-6) gets an addendum paragraph referencing 19-7's month-end alert; `sprint-status.yaml` (status bump). Zero edits under `apps/web/`, `apps/api/`, `testsprite_tests/`, `_bmad-output/audit/testsprite-queue.yaml` (this story only READS the queue file; it does NOT write to it — 19-6 owns writes), `tests/visual/`, `playwright.config.ts`, `package.json`, `pnpm-lock.yaml`. `ux-design.pen` NOT modified → screenshot workflow does not trigger.

6. **Secrets reuse.** The workflow consumes the SAME `secrets.TESTSPRITE_API_KEY` that 19-6 uses — no new secret added. `vars.TESTSPRITE_TARGET_URL` is NOT consumed (account-info hits TestSprite's central API, not the user's Vido instance). The auto-issued `secrets.GITHUB_TOKEN` is used for the `gh issue` operations (search/create/comment/close) and label management; job-scoped permissions: `{ contents: read, issues: write }` (minimum-privilege — `contents: read` for checkout if the workflow needs to read `_bmad-output/audit/testsprite-queue.yaml`; `issues: write` for the Issue/label operations). No `pull-requests:` permission needed.

7. **Failure semantics.** Workflow exit code ≠ 0 ONLY on: (a) `TESTSPRITE_API_KEY` invalid/expired (auth failure on account-info); (b) `gh` CLI / GitHub API call to create/comment/close the Issue fails (e.g. `issues: write` permission misconfigured at the repo or org level). Threshold-not-breached + no existing Issue = exit 0, no Issue created (the desired steady state). The workflow MUST NOT exit ≠ 0 just because there's already an open Issue (that's the expected month-late state of a real warning condition).

8. **Concurrency control.** `concurrency: { group: 'testsprite-quota-warning', cancel-in-progress: false }` — same `cancel-in-progress: false` rationale as 19-6: a mid-run cancellation could leave the Issue in an inconsistent state (e.g. comment posted but auto-close skipped). Manual `workflow_dispatch` while the scheduled run is in progress queues behind it. Different concurrency group from 19-6 (`testsprite-monthly`) so the two workflows can run independently — they don't conflict at the file level (19-6 writes queue file, 19-7 only reads it).

9. **Regression + framework hygiene.** `pnpm lint:all` = 0 errors / ≤122 warnings (workflow file is YAML — `prettier --check .` covers it). `actionlint` clean. `pnpm nx test web` + `pnpm nx test api` pass (no source touched). `pnpm test:e2e --list` count unchanged. `pnpm run test:visual` green. `pnpm run test:cleanup` no orphans. `ux-design.pen` untouched.

10. **`project-context.md` updates.** (a) `Last Updated` header gets a 19-7 entry following the existing 19-4 / 19-4b / 19-5 / 19-6 format; (b) the "Monthly Cron Workflow (story 19-6)" sub-section (added by 19-6 Task 3) gets a final paragraph appended that names 19-7's workflow file, what its threshold is (>30 remaining), and where the resulting Issue surfaces (Issues tab, label `testsprite-quota-warning`). No NEW sub-section — 19-7 is the warning sibling of 19-6's primary loop, not a free-standing workflow concept; appending keeps the docs flat.

## Tasks / Subtasks

- [ ] Task 1: Author the workflow file (AC: #1, #2, #3, #4, #6, #7, #8)
  - [ ] `.github/workflows/testsprite-quota-warning.yml` — name, triggers (cron + workflow_dispatch only), single job on `ubuntu-24.04`. Job-scoped permissions: `{ contents: read, issues: write }`. Concurrency: `{ group: 'testsprite-quota-warning', cancel-in-progress: false }`.
  - [ ] Steps: (1) `actions/checkout@v4` (default depth — only needs the queue file at the current commit, no history); (2) `pnpm/action-setup@v4` v9; (3) `actions/setup-node@v4` with `.nvmrc` + `cache: 'pnpm'`; (4) `pnpm install --frozen-lockfile`; (5) **Ensure `testsprite-quota-warning` label exists** — `gh label create testsprite-quota-warning --color FFA500 --description "TestSprite monthly credits about to expire (story 19-7)" --force` (idempotent via `--force`); (6) Install TestSprite CLI — `npx @testsprite/testsprite-mcp@0.0.37` (same version pin as 19-6).
  - [ ] Main step is a bash/`actions/github-script@v7` step that: (a) calls `testsprite_check_account_info` via the CLI, parses `remaining_credits` defensively (probe `remaining_credits` / `credits_remaining` / `available_credits`; fail-closed if none found); (b) reads `_bmad-output/audit/testsprite-queue.yaml`'s `last_run:` block via `yq` for the audit-trail-snapshot fields used in the AC #3 Issue body; (c) searches existing open Issues with label `testsprite-quota-warning` via `gh issue list --state open --label testsprite-quota-warning --json number,body,title` (`--json` makes the output machine-parseable; pick the first result if multiple exist — multiple-open Issues would be a bug to investigate, not silently accept); (d) executes the AC #2 decision tree (4 branches: no-Issue+breached, Issue+breached, no-Issue+OK, Issue+OK). Use `actions/github-script@v7` for the Issue CRUD (cleaner than `gh` CLI for HEREDOC body templating + auto-handles GITHUB_TOKEN auth).
  - [ ] Top-of-file comment block: rationale (28th-of-month cron picks because Feb), AC #6 secret name + reuse-from-19-6, AC #2 dedup-label as the contract, link back to story 19-7 + sibling story 19-6.
- [ ] Task 2: Validate the workflow locally (AC: #1, #9)
  - [ ] `actionlint .github/workflows/testsprite-quota-warning.yml` clean.
  - [ ] `pnpm exec prettier --check .github/workflows/testsprite-quota-warning.yml` clean (or `--write` and accept reformat).
  - [ ] Push the workflow; observe it appears in the Actions tab as a valid scheduled workflow.
  - [ ] Fire `workflow_dispatch` manually to validate — this is a READ-only operation on TestSprite (account-info only, no test consumption), so it's free to run any number of times. Confirm: (a) account-info call succeeds against the same Secret 19-6 uses; (b) the dedup search returns 0 Issues (clean state); (c) if remaining > 30, an Issue IS opened with the AC #3 template populated; (d) if remaining ≤ 30, no Issue is opened (or an existing test Issue gets closed if you opened one manually for testing).
  - [ ] Cleanup any test-Issue opened during validation — close it manually with a "test run, ignore" comment, OR design the test so it lands in the steady state (e.g. set the threshold temporarily to `9999` via a workflow_dispatch input → guaranteed no breach → no Issue).
- [ ] Task 3: Documentation (AC: #5, #10)
  - [ ] `project-context.md` Last Updated header → add a 19-7 entry (one-paragraph block dated `2026-05-{day}`) summarising: month-end quota warning, dedup-via-label, AC #6 secret reuse, mirror format of 19-6's entry.
  - [ ] `project-context.md` "Monthly Cron Workflow (story 19-6)" sub-section (under TestSprite Journey Test Workflow) → append a final paragraph: "**Month-end quota warning (story 19-7).** A sibling workflow `.github/workflows/testsprite-quota-warning.yml` runs `0 3 28 * *` UTC and opens a GitHub Issue tagged `testsprite-quota-warning` when remaining credits > 30 (the reserved floor from 19-6 AC #3). Issue auto-closes when the threshold drops back. See `_bmad-output/implementation-artifacts/19-7-github-actions-quota-warning.md` for the dedup contract + Issue body schema."
- [ ] Task 4: Close-out regression (AC: #5, #9)
  - [ ] `pnpm lint:all` → 0 errors, ≤122 warnings.
  - [ ] `pnpm nx test web` + `pnpm nx test api` pass.
  - [ ] `pnpm test:e2e --list` → 1663 unchanged.
  - [ ] `pnpm run test:visual` green against committed baselines.
  - [ ] `pnpm run test:cleanup` no orphans.
  - [ ] `ux-design.pen` untouched → screenshot workflow not triggered.
  - [ ] Sprint-status entry: `ready-for-dev` → `in-progress` → `review`.

## Dev Notes

### Why this story exists / where it sits in epic-19

- **19-7 is the watchdog for 19-6.** Six failure modes 19-6 cannot self-detect: (i) `TESTSPRITE_API_KEY` rotated / expired silently — 19-6 fails its account-info call, exits ≠ 0, fires an email — but only if the user reads workflow-failure emails (commonly muted); (ii) `TESTSPRITE_TARGET_URL` Variable unset or pointing at a stale tunnel — 19-6 succeeds at account-info but every test case fails; queue commits-back but with `last_run.status: "test-failures-only"` (or "api-failure"); (iii) GitHub-Actions branch-protection blocking 19-6's commit-back push — 19-6 exits ≠ 0 mid-run after consuming credits but failing to record them; (iv) user manually consumed 30+ credits via local `workflow_dispatch` between cron runs, starving the monthly cron of remaining budget; (v) the cron itself didn't fire (GitHub Actions has a documented "scheduled workflows may be delayed or skipped during periods of high load" — affects free-tier accounts more); (vi) 19-6 was disabled (user clicked "Disable workflow" in the Actions UI, possibly forgot). 19-7 catches ALL six by reading the live remaining-credits count at month-end — if > 30 remain, something went wrong upstream, and the user has 3 days to act.
- **Why open a GitHub Issue instead of email / Slack / Discord:** (a) the existing project tooling is GitHub-centric (sprint-status / story files in repo, PRs as the work unit) — staying in GitHub keeps the alert in the same place as the work; (b) Issues are auditable (`git log` won't show, but `gh issue list --label testsprite-quota-warning --state all` gives the history); (c) no extra integration (no webhook URL, no bot token, no Slack workspace dependency); (d) Issues are dismissible (close = acknowledged) and reopenable (the dedup logic in AC #2 makes the workflow re-use rather than spam).
- **Why 28th-of-month, not 30th or 31st** (AC #1 deep-dive): cron's `* * 31 * *` does NOT trigger in months with 30 days (Apr, Jun, Sep, Nov) and never in Feb; `* * 30 * *` skips Feb; `* * 29 * *` mostly works but skips Feb in non-leap-years. `* * 28 * *` fires every month including Feb non-leap-year (Feb has 28 days every year). The 28th gives 3 days lead time in 31-day months, 2 days in 30-day, 0–1 in Feb — enough to manually catch up with one `workflow_dispatch` of 19-6 (~120 credits consumed in ~2–3 minutes given TestSprite's per-case latency).
- **Why threshold = 30, not 0 or 50:** 30 == 19-6's `reserved_credits` floor (the explicit reserve for ad-hoc manual runs). When `remaining > 30`, it means either (a) ad-hoc didn't consume the reserve AND the cron didn't consume the cap (something broke), OR (b) the user just hasn't done ad-hoc runs this month and the cron behaved exactly as designed — but the un-consumed reserve about to expire is signal worth a low-noise nudge regardless. Choosing 30 (not 0) means "warn even if technically the cron did its job, because un-consumed credits are wasted credits". Choosing 30 (not 50) avoids false positives when the cron consumes exactly 120 (leaving 30 — borderline) and natural daily fluctuation pushes remaining briefly to 31 due to a partial-batch retry. The threshold is `> 30`, NOT `≥ 30` — exact 30-remaining = the reserved floor exactly = no warning.
- **Dependency:** consumes **19-6 [@contract-v1] AC #2 (schema_version: 1, last_run block shape)** for the audit-trail-snapshot in AC #3's Issue body. **Confirmed against [@contract-v1] (Story 19-6 AC #2)** — schema_version = 1, last_run fields {`started_at`, `finished_at`, `credits_at_start`, `credits_at_end`, `credits_consumed`, `test_ids_run`, `status`, `run_url`}. If 19-6 ever bumps to `schema_version: 2`, this story's queue-file-read step MUST re-check the field names. The READ is defensive — if the schema_version != 1, the audit-trail-snapshot in the Issue body MUST gracefully degrade to "Audit trail unavailable (queue schema_version mismatch — investigate)" rather than crashing or rendering garbage. **Confirmed against [@contract-v1] (Story 19-6 AC #4)** — commit-message format `chore(testsprite): monthly run {YYYY-MM} — ...` is referenced in 19-7's Issue body's "Last 19-6 run URL" derivation (the run URL comes from `last_run.run_url` in the queue file, not from `git log`, but the commit-message-format-stability is what makes future Rule 22 retros able to `git log --grep='chore(testsprite): monthly run'` for trend analysis; 19-7 doesn't directly consume it but its Issue body cross-references the audit trail).
- **What this story doesn't do:** doesn't write to `_bmad-output/audit/testsprite-queue.yaml` (read-only consumer); doesn't open Issues for test-case failures (that's a different signal — 19-6's `last_run.status: "test-failures-only"` is per-case, not per-month — a future story could add per-failure Issue creation if useful, but 19-7's signal is the QUOTA, not the TESTS); doesn't consume any TestSprite credits beyond the free account-info API call (the account-info endpoint is metadata, not test execution — no credit cost).

### Architecture / constraints — read before implementing

- **Pure CI / DevOps story.** 0 Go, 0 frontend source, 0 tests authored. Cross-stack split: 0 backend, 0 frontend → single story, trivially correct.
- **Issue dedup via label, not title-substring-match.** Labels are stable, machine-queryable, and survive title edits (a user might rename the Issue title; the label persists unless explicitly removed). Title-substring matches break on rename. The HTML-comment marker in the Issue body (AC #3) is a tertiary defence — if a user removes the label AND renames the title, the marker survives Markdown preview and `gh issue list --search "testsprite-quota-warning-marker"` still finds the orphan.
- **`actions/github-script@v7` vs raw `gh` CLI:** `github-script` is the cleaner choice here because (a) the Issue body has multi-line Markdown that's painful to escape in a shell HEREDOC inside another HEREDOC (workflow YAML), (b) the dedup logic is conditional with branching (4 branches per AC #2) — `gh` chains of `if/then` against `gh issue list` get gnarly; JavaScript reads better, (c) `github-script` auto-handles `GITHUB_TOKEN` injection. Trade-off: adds the `actions/github-script` action dep — pinned `@v7` per current major. Document the choice in the workflow top-comment.
- **TestSprite CLI installation in 19-7 — same `0.0.37` pin as 19-6.** If 19-6 bumps the pin (e.g. to `0.0.38`), 19-7's bump is its own PR (parallel — 19-6 and 19-7 are siblings; their CLI installations are independent — neither blocks the other). Cross-story drift on the CLI version is acceptable in the short term but should be reconciled in a subsequent housekeeping PR.
- **`testsprite_check_account_info` cost** — the TestSprite docs are not crystal-clear on whether account-info itself consumes credits. The 2026-03-15 local-bootstrap memo's "150-credit Free plan" never explicitly mentions account-info charges. **Defensive assumption: account-info is FREE.** If implementation reveals it's NOT free (e.g. first manual `workflow_dispatch` shows the credit count dropping after the call), pivot to either (a) ratchet 19-7's cron to less frequent than monthly (e.g. weekly at most, but month-end-only suffices for the warning purpose), or (b) compute remaining credits by reading 19-6's `last_run.credits_at_end` + (no manual consumption since) — but that loses the "live source of truth" guarantee AC #4 currently provides. Both fallbacks are out of scope for this story's primary implementation; address only if validation reveals the assumption is wrong.
- **The Issue body's permalink to `testsprite-queue.yaml`** (AC #3) MUST use a SHA-pinned URL (`github.com/{owner}/{repo}/blob/{SHA}/_bmad-output/audit/testsprite-queue.yaml`), NOT a branch-pinned one (`/blob/main/...`), because the file mutates monthly. The current `GITHUB_SHA` (the SHA the workflow checked out) is the right pin — captures the queue state at the moment the warning fired. Use `${{ github.sha }}` from the workflow context. If a future maintainer clicks the link 6 months later, they see the queue state as it was at warning time, not the current state.
- **Permission scoping** — `issues: write` is the minimum required for label create/Issue create/Issue comment/Issue close. `contents: read` is for the `actions/checkout@v4` step (need to read the queue file). No `actions: write`, no `pull-requests: write`, no `pages: write`. Both at the JOB level, not workflow level.
- **Auto-close behaviour edge case** — AC #2's "auto-close when threshold drops back" branch: if a user manually closes the Issue between 19-7 runs while the threshold is still breached, the next 19-7 run sees "no open Issue" and opens a NEW one (correct behaviour — user closed prematurely, the alert resurfaces). If a user manually reopens a closed Issue after auto-close, the next 19-7 run sees "open Issue + threshold OK", posts the ✅ Resolved comment again, and closes it again. Both edge cases are acceptable churn — the dedup rule is robust to either.
- **Issue comment vs new Issue** for a re-fire on the same threshold breach: AC #2 explicitly says "comment on existing Issue, do NOT open duplicate". This means after 28 days of an unresolved warning, the Issue accrues 1 comment per scheduled run (1 per month, since the warning workflow runs monthly). If the user fires `workflow_dispatch` ad-hoc 5 times in a day to validate, the Issue gets 5 comments. Acceptable — the comments tell a story of "warning persisted across N runs, threshold was {N1} → {N2} → {N3}".

### Project Structure Notes

- **New files:** `.github/workflows/testsprite-quota-warning.yml`.
- **Modified files:** `project-context.md` (Last Updated header + the 19-6 sub-section addendum paragraph); `_bmad-output/implementation-artifacts/sprint-status.yaml` (19-7 status); this story file.
- **Read-only consumed:** `_bmad-output/audit/testsprite-queue.yaml` (for the audit-trail snapshot in the Issue body).
- **Auto-generated:** GitHub Issues with label `testsprite-quota-warning` (lifecycle: monthly cron creates / re-comments / auto-closes; manual user actions on the Issue are tolerated per Dev Notes "auto-close edge case").
- **Out of scope:** the actual remediation when a warning fires (user's operational action — fire 19-6's `workflow_dispatch`, OR investigate root cause); any change to 19-6 (this story is read-only on 19-6's contract surface); any change to TestSprite's TC*.py test cases; any per-failure Issue creation (that's a different signal — see Dev Notes "What this story doesn't do").

### Testing standards (project-context.md)

- **No new test code.** The deliverable is a CI workflow. Validation = `actionlint` clean + Task 2 manual `workflow_dispatch` end-to-end.
- **Rule 12 lint gate:** `pnpm lint:all` 0 errors / ≤122 warnings. YAML covered by `prettier --check .`. `actionlint` is the additional check.
- **Rule 16 assertion quality:** N/A.
- **Rule 13 error handling (workflow-level):** the account-info call is `set -e` (auth failure = hard fail per AC #7). The `gh issue` operations use `set -e` (API failures = hard fail). The yq read of the queue file is `set +e` with a fallback to "audit trail unavailable" rendering per Dev Notes.
- **`pnpm run test:cleanup`:** N/A in CI (ephemeral runner); applies to local Task 2 manual run.

### Rule 21 / Rule 22 / Rule 20 linkage

- **Rule 21 (Component-to-Design Node Traceability):** N/A (no component files).
- **Rule 22 (Epic Retro Design-Drift Audit):** indirectly supportive — 19-7's Issues are the surfacing layer for "19-6 ran into trouble"; Rule 22 retros that include a "did our test infra work this epic" check can grep `gh issue list --label testsprite-quota-warning --state all --search "in:title TestSprite"` for the epic's monthly health. No tooling-line change needed in project-context.md Rule 22 block — that block is about visual-regression diff tooling; TestSprite is journey-level (separate tracking surface).
- **Rule 20 (AC Contract Versioning):** stamps `[@contract-v1]` on AC #1–#3 (workflow trigger model + Issue dedup contract + Issue body format). Downstream consumers: future Rule 22 retros may grep Issue titles/bodies for trend analysis; future credit-alerting stories (e.g. quota-warning-Slack-bridge if the user ever wants extra surfacing) may extend the Issue body format. **Upstream consumed:** confirmed against 19-6 [@contract-v1] AC #2 (queue file schema_version: 1 + last_run block shape) + AC #4 (commit-message format, indirectly via the audit-trail cross-link). No upstream contract bump.
- **Rule 7 (Error Codes):** N/A — pure CI / 0 Go.

### Latest tech information

- **`actions/github-script@v7`** — current stable; runs a Node.js script with `octokit`-style GitHub API access. Pinned `@v7` per current major (matches the project's documented major-pin convention used by `actions/checkout@v4`, `pnpm/action-setup@v4`, `actions/setup-node@v4`).
- **`gh` CLI** — pre-installed on `ubuntu-24.04` runners; version stays roughly current. Used for label create (`gh label create … --force`) — `actions/github-script` doesn't expose labels-CRUD as cleanly.
- **`@testsprite/testsprite-mcp@0.0.37`** — same pin as 19-6 Task 2. Sibling-story version-drift caveat per Dev Notes.
- **GitHub Actions `schedule:` cron** — POSIX 5-field; `0 3 28 * *` = minute 0, hour 3 UTC, day-of-month 28, any month, any day-of-week. Documented as "may be delayed or skipped during high load on free-tier" — see Dev Notes failure mode (v) and (vi).
- **`secrets.GITHUB_TOKEN`** — auto-issued; permissions: `{ contents: read, issues: write }` job-scoped.

### References

- [Source: _bmad-output/implementation-artifacts/sprint-status.yaml:528] — the 19-7 charter line: `.github/workflows/testsprite-quota-warning.yml` cron `0 3 28 * *` UTC, checks remaining credits via `testsprite_check_account_info`, opens GitHub Issue if unused > 30 credits, non-intrusive surfacing.
- [Source: _bmad-output/implementation-artifacts/19-6-github-actions-testsprite-monthly.md] — sibling story; the consumer this story watches. Provides the [@contract-v1] AC #2 (queue file schema_version + last_run block) + AC #4 (commit-message format) consumed here.
- [Source: _bmad-output/audit/testsprite-queue.yaml] (created by 19-6) — the file 19-7 reads for the audit-trail snapshot. NOT written by this story.
- [Source: project-context.md §§L1138–L1160 "TestSprite Journey Test Workflow" — Monthly Cron Workflow (story 19-6) sub-section] — created by 19-6 Task 3; this story's Task 3 appends a paragraph.
- [Source: .github/workflows/testsprite-monthly.yml] (created by 19-6) — the workflow 19-7's Issue body links to ("Fire `workflow_dispatch` on [TestSprite Monthly]").
- [Source: _bmad-output/implementation-artifacts/19-5-github-actions-visual-regression-pr.md] — pattern reference for workflow structure (ubuntu-24.04 pin, job-scoped permissions, concurrency block, top-of-file comment).
- [Source: project-context.md#Rule-12-Code-Quality-Checks-CI-based] — `pnpm lint:all` baseline (122 warnings); YAML coverage via `prettier --check .`.
- [Source: project-context.md#Rule-20-AC-Contract-Versioning] — stamp + ack format; this story stamps [@contract-v1] on AC #1–#3.
- [Source: GitHub Actions docs — `schedule:` cron behaviour on Free tier] — scheduled workflows may be skipped during high load; reinforces 19-7's existence as a watchdog (cf. Dev Notes failure modes v/vi).
- [Source: GitHub Issues / Labels CLI docs (`gh issue create` / `gh issue list --json` / `gh label create --force`)] — the CRUD primitives the workflow uses.
- [Source: actions/github-script@v7 docs] — the Node.js scripting action for cleaner Issue body templating.

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List

## Change Log

| Date | Change |
| ---- | ------ |
| 2026-05-18 | SM Bob /create-story (YOLO) — story drafted ready-for-dev. Pure CI / DevOps; 0 Go / 0 frontend source / 0 tests authored → single story (cross-stack split N/A; backend tasks = 0). 10 ACs (#1–#3 stamped `[@contract-v1]`), 4 tasks. Key SM decisions recorded in Dev Notes: (1) **28th-of-month cron** — `0 3 28 * *` is the safest "near-month-end" pattern; `*/30/31` cron expressions skip Feb / 30-day months. 28th gives 2–3 days lead time for manual catch-up. (2) **Threshold > 30, not ≥ 30** — 30 is 19-6's `reserved_credits` floor; exact 30 = correct steady state, no warning; > 30 = unused credits about to expire. (3) **Issue dedup via label (`testsprite-quota-warning`), not title-substring** — labels are stable + machine-queryable; HTML-comment marker in body is tertiary defence. (4) **`actions/github-script@v7` over raw `gh` CLI** — cleaner for multi-line Markdown Issue body + 4-branch conditional dedup decision tree; auto-handles GITHUB_TOKEN. (5) **Auto-close on threshold-drop** — when remaining ≤ 30 AND an open Issue exists, the workflow posts `✅ Resolved` and closes; user doesn't manually close monthly. (6) **No new secrets** — reuses 19-6's `TESTSPRITE_API_KEY`; account-info is API-only, no `TESTSPRITE_TARGET_URL` consumption. (7) **Account-info assumed FREE** (no per-call credit cost) — documented as defensive assumption; fallback paths sketched in Dev Notes if validation reveals otherwise. (8) **Six failure modes 19-6 cannot self-detect** explicitly enumerated in Dev Notes — auth rotation, target unreachable, branch-protection block, manual-consumption starve, cron skip, workflow disabled. 19-7 catches ALL via the live-credit threshold check. (9) **Issue body permalink uses `${{ github.sha }}`-pinned URL** to `testsprite-queue.yaml`, not `/blob/main/` — captures queue state at warning-time. Consumes 19-6 [@contract-v1] AC #2 (schema_version + last_run shape) + AC #4 (commit-message format) per Rule 20 ack — confirmed against both. 🔒 Rule 7 Wire Format: N/A (pure CI / 0 Go). 🎨 UX: N/A (no .pen modification). |
| 2026-05-18 | [@contract-v0→v1] AC #1–#3 stamped on creation — what's defined: workflow file at `.github/workflows/testsprite-quota-warning.yml` + name `TestSprite Quota Warning` + cron `0 3 28 * *` UTC + `workflow_dispatch` trigger + `ubuntu-24.04` runner pin (AC #1); dedup contract — at most one open Issue with label `testsprite-quota-warning` at any time; 4-branch decision tree (no-Issue+breached → create, Issue+breached → comment, no-Issue+OK → no-op, Issue+OK → ✅-close); label-create idempotent via `--force` with colour `FFA500` (AC #2); Issue title format `[TestSprite] {N} credits unused at month-end ({YYYY-MM})` + body Markdown template with Why/What-to-do/Audit-trail-snapshot/Workflow-run sections + trailing `<!-- testsprite-quota-warning-marker -->` HTML comment (AC #3). What breaks downstream: future Rule 22 retros / a quota-warning-Slack-bridge story may grep Issue titles or bodies — silently changing the title format breaks the grep; the marker comment is the tertiary dedup key if the label is hand-removed — silently dropping it weakens the dedup robustness. Upstream consumption: confirmed against [@contract-v1] (Story 19-6 AC #2 — queue file `schema_version: 1` + last_run block shape consumed in audit-trail-snapshot rendering), confirmed against [@contract-v1] (Story 19-6 AC #4 — commit-message format referenced indirectly via the audit-trail cross-link in the Issue body). No upstream contract bump — this story READS 19-6's contract surface, does not modify it. |
