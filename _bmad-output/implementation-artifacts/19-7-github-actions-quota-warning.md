# Story 19.7: GitHub Actions Month-End TestSprite Monthly-Run Watchdog (Design-Drift Audit — Phase 2 CI)

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->
<!-- @contract-v1 on AC #1–#3 (workflow trigger model + Issue de-dup contract + Issue body format that Rule 22 retros / future credit-alerting stories may grep). -->
<!-- 🔗 AC Drift: FOUND→RESOLVED (correct-course 2026-05-20 — re-scoped against 19-6 [@contract-v2]; originally drafted against superseded v1) · 📎 Contract Stamps: v1×3 this story; consumes ONLY 19-6 [@contract-v2] AC #2 (queue schema_version:2 + last_run block); live-credit dependency on 19-6 AC #4/#5/#6 REMOVED · 🔒 Rule 7: N/A (pure CI / 0 Go) · 🎨 UX: N/A (no `.pen` modification, no UI change) -->
<!-- markers-block-end -->

## Story

As a maintainer of the Vido TestSprite integration who set up the monthly consumer (19-6),
I want a GitHub Actions workflow that runs on the 28th of every month at 03:00 UTC, inspects the `last_run` block of `_bmad-output/audit/testsprite-queue.yaml` (written by 19-6, schema_version 2), and opens a GitHub Issue when 19-6's last monthly run failed (`api-failure` / `test-failures-only`) or did not run at all (`last_run` absent or stale beyond 35 days),
so that the 19-6 consumer's failure modes (auth expired silently, target unreachable, cron skipped, workflow disabled, commit-back blocked) surface in the GitHub UI three days before month-end — leaving time for a manual `workflow_dispatch` of 19-6 to catch up — instead of a broken monthly pass being discovered a month later.

## Acceptance Criteria

1. [@contract-v1] **A new GitHub Actions workflow file `.github/workflows/testsprite-quota-warning.yml` exists** with exactly two triggers: (a) `schedule: - cron: '0 3 28 * *'` (28th of month, 03:00 UTC — chosen because it gives ≥3 days of lead time before month-end credit expiry at 23:59 UTC on the last day, even in February; running on the 30th or 31st risks the cron not firing in February at all — `0 3 31 * *` wouldn't fire in Feb / Apr / Jun / Sep / Nov; the 28th is the safest "near-month-end" cron); (b) `workflow_dispatch` (manual trigger for testing the workflow itself OR for a same-day re-check after firing 19-6's `workflow_dispatch` to catch up). NO `push` / `pull_request` / `schedule:` with other crons. Workflow `name:` is `TestSprite Quota Warning`. Single job `name: TestSprite Quota Warning / Check`. `runs-on: ubuntu-24.04` (pinned per 19-5 / 19-6 AC #5 rationale — deterministic runner; do NOT use `-latest`).

2. [@contract-v1] **Issue de-dup logic — at most ONE open watchdog Issue at any time.** The workflow MUST search the repo for an open Issue with the label `testsprite-quota-warning` (and EXACTLY this label — used as the dedup key) BEFORE opening a new one. Behaviour (4 branches; "UNHEALTHY" / "HEALTHY" defined in AC #4):
   - **No existing open Issue + 19-6 UNHEALTHY**: open a new Issue with the AC #3 title/body and apply the `testsprite-quota-warning` label.
   - **Existing open Issue + 19-6 UNHEALTHY**: post a new comment on the existing Issue with the current health reason + a timestamp + a link to this workflow run — DO NOT open a duplicate. The existing Issue becomes the running thread for the month's watchdog sequence.
   - **No existing open Issue + 19-6 HEALTHY**: no-op; workflow exits cleanly without opening anything.
   - **Existing open Issue + 19-6 HEALTHY**: post a final comment `✅ Resolved — 19-6's last_run is healthy again ({status}, finished {finished_at}). Closing.` and CLOSE the Issue (auto-close — the situation that triggered it has been remedied, either by 19-6's cron landing a fresh healthy run or by a manual `workflow_dispatch` catching up). This closes the loop without requiring the user to manually close issues each month.
   
   The `testsprite-quota-warning` label MUST be created if it doesn't exist (use `gh label create testsprite-quota-warning --color FFA500 --description "TestSprite monthly run watchdog (story 19-7)" --force` — `--force` makes it idempotent). The label colour `FFA500` (orange) signals "warning, not error" — consistent with `requires-manual-review` (used by 19-5 bootstrap PR).

3. [@contract-v1] **Issue body format.** The Issue title and body MUST follow this template (the workflow renders placeholders via `gh issue create` or `actions/github-script@v7`):
   
   - **Title:** `[TestSprite] Monthly run watchdog: {reason} ({YYYY-MM})` — e.g. `[TestSprite] Monthly run watchdog: 19-6 last run failed — api-failure (2026-05)`
   - **Body** (Markdown HEREDOC):
     ```markdown
     ## Why this Issue was opened
     
     The month-end watchdog (`testsprite-quota-warning.yml`, story 19-7) found story 19-6's monthly TestSprite cron in an UNHEALTHY state:
     
     **{reason}**
     
     19-6 spends the monthly TestSprite credit budget on journey-test coverage. An unhealthy `last_run` means that coverage did not happen this cycle — and Free-plan credits expire at calendar-month rollover (~{X} days from now) with no carry-forward.
     
     ## What to do
     
     1. Fire `workflow_dispatch` on [TestSprite Monthly]({link-to-19-6-workflow}) to run the monthly pass now.
     2. After it completes, re-fire `workflow_dispatch` on this workflow ([TestSprite Quota Warning]({link-to-this-workflow})) to re-check — when 19-6's `last_run` is healthy again, this Issue auto-closes.
     3. If 19-6's run fails, investigate root cause (most common: `TESTSPRITE_API_KEY` rotated / expired, `TESTSPRITE_TARGET_URL` unreachable, GitHub-Actions branch-protection blocking the commit-back push).
     
     ## Audit trail snapshot
     
     - **19-6 last_run status:** `{last_run.status | "null — never run"}`
     - **19-6 last_run finished:** {last_run.finished_at | "never"}
     - **19-6 last_run test cases:** {last_run.test_ids_run.length}
     - **19-6 last_run URL:** {last_run.run_url | "n/a"}
     - **Queue schema_version:** {schema_version}
     - **Queue file:** [`_bmad-output/audit/testsprite-queue.yaml`]({queue-file-permalink-at-current-sha})
     
     ## Workflow run
     
     This Issue was opened by [TestSprite Quota Warning run #{run_id}]({this-run-url}).
     
     <!-- testsprite-quota-warning-marker: do not edit; consumed by 19-7's dedup logic -->
     ```
     
   The trailing HTML-comment marker is a defensive de-dup hint — `gh issue list --label testsprite-quota-warning --state open` is the primary dedup key per AC #2, but a future maintainer who hand-edits the label off the Issue won't get duplicates because the workflow can also fall back to grepping the body for the marker comment.

4. **The `_bmad-output/audit/testsprite-queue.yaml` `last_run` block (schema_version 2) is 19-7's single source of truth for 19-6's health.** 19-7 makes NO live TestSprite API call and installs NO TestSprite CLI — it is a pure repo-file reader + GitHub-Issue writer. The workflow MUST:
   - Verify `schema_version == 2` via `yq`. Absent or != 2 → treat as a HEALTH FAILURE (19-6's queue contract drifted) → open/maintain an Issue per AC #2; do NOT crash.
   - Read `last_run.status` (enum: `success | budget-exhausted | api-failure | test-failures-only | null`) and `last_run.finished_at` (ISO-8601 UTC or null).
   - NOT read `last_run.credits_*` — those fields are permanently null in 19-6 [@contract-v2] (the bare `@testsprite/testsprite-mcp` CLI cannot query live account-info; only the MCP-server mode exposes `testsprite_check_account_info`). A future `19-7a` story may add live-credit alerting via an MCP-stdio dance — explicitly OUT of scope here.

   19-6 is judged **UNHEALTHY** (→ warn) when ANY of: (a) `schema_version != 2`; (b) `last_run.status` ∈ { `api-failure`, `test-failures-only` }; (c) `last_run.finished_at` is null (19-6 never ran) OR older than 35 days — one full monthly cycle + slack; covers cron skipped / workflow disabled / branch-protection blocking the commit-back so the queue never updated. Otherwise **HEALTHY** (`last_run.status` ∈ { `success`, `budget-exhausted` } AND `finished_at` ≤ 35 days old) → no warn.

5. **No production / source-tree edits.** Created: `.github/workflows/testsprite-quota-warning.yml`. Modified: `project-context.md` Last Updated header (one entry) + the TestSprite Journey Test Workflow section's new "Monthly Cron Workflow (story 19-6)" sub-section (added by 19-6) gets an addendum paragraph referencing 19-7's month-end alert; `sprint-status.yaml` (status bump). Zero edits under `apps/web/`, `apps/api/`, `testsprite_tests/`, `_bmad-output/audit/testsprite-queue.yaml` (this story only READS the queue file; it does NOT write to it — 19-6 owns writes), `tests/visual/`, `playwright.config.ts`, `package.json`, `pnpm-lock.yaml`. `ux-design.pen` NOT modified → screenshot workflow does not trigger.

6. **No TestSprite secret — no new secret of any kind.** The re-scope removed the live account-info call, so 19-7 does NOT consume `secrets.TESTSPRITE_API_KEY` and does NOT consume `vars.TESTSPRITE_TARGET_URL`. The ONLY token used is the auto-issued `secrets.GITHUB_TOKEN`, for the `gh` / github-script Issue + label operations (search/create/comment/close/label-create). Job-scoped permissions: `{ contents: read, issues: write }` (minimum-privilege — `contents: read` for `actions/checkout` to read `_bmad-output/audit/testsprite-queue.yaml`; `issues: write` for the Issue/label operations). No `pull-requests:` permission needed. 19-7 is therefore contract-independent of 19-6's AC #5/#6 secret surface — it depends ONLY on 19-6 [@contract-v2] AC #2 (queue-file `schema_version: 2` + `last_run` block shape).

7. **Failure semantics.** Workflow exit code ≠ 0 ONLY on: (a) the `gh` CLI / github-script call to create/comment/close the Issue or create the label fails (e.g. `issues: write` permission misconfigured at the repo or org level); (b) the queue file is absent or unparseable as YAML (genuine repo corruption — distinct from `schema_version != 2`, which is a *detected* drift the watchdog reports via an Issue, exit 0). A detected-UNHEALTHY 19-6 + Issue successfully opened/updated = exit 0 (the watchdog did its job). HEALTHY 19-6 + no Issue = exit 0 (the desired steady state). The workflow MUST NOT exit ≠ 0 just because there's already an open Issue (that's the expected state of a real, still-active warning condition).

8. **Concurrency control.** `concurrency: { group: 'testsprite-quota-warning', cancel-in-progress: false }` — same `cancel-in-progress: false` rationale as 19-6: a mid-run cancellation could leave the Issue in an inconsistent state (e.g. comment posted but auto-close skipped). Manual `workflow_dispatch` while the scheduled run is in progress queues behind it. Different concurrency group from 19-6 (`testsprite-monthly`) so the two workflows can run independently — they don't conflict at the file level (19-6 writes queue file, 19-7 only reads it).

9. **Regression + framework hygiene.** `pnpm lint:all` = 0 errors / ≤122 warnings (workflow file is YAML — `prettier --check .` covers it). `actionlint` clean. `pnpm nx test web` + `pnpm nx test api` pass (no source touched). `pnpm test:e2e --list` count unchanged. `pnpm run test:visual` green. `pnpm run test:cleanup` no orphans. `ux-design.pen` untouched.

10. **`project-context.md` updates.** (a) `Last Updated` header gets a 19-7 entry following the existing 19-4 / 19-4b / 19-5 / 19-6 format; (b) the "Monthly Cron Workflow (story 19-6)" sub-section (added by 19-6 Task 3) gets a final paragraph appended that names 19-7's workflow file, what it watches (19-6's `last_run` health — failure status or >35-day staleness, no live credit count), and where the resulting Issue surfaces (Issues tab, label `testsprite-quota-warning`). No NEW sub-section — 19-7 is the watchdog sibling of 19-6's primary loop, not a free-standing workflow concept; appending keeps the docs flat.

## Tasks / Subtasks

- [x] Task 1: Author the watchdog workflow file (AC: #1, #2, #3, #4, #6, #7, #8)
  - [x] `.github/workflows/testsprite-quota-warning.yml` — name `TestSprite Quota Warning`, triggers (cron `0 3 28 * *` + workflow_dispatch only), single job `TestSprite Quota Warning / Check` on `ubuntu-24.04`. Job-scoped permissions: `{ contents: read, issues: write }`. Concurrency: `{ group: 'testsprite-quota-warning', cancel-in-progress: false }`.
  - [x] Steps: `actions/checkout@v4` (default depth) + label-ensure (`gh label create testsprite-quota-warning --color FFA500 --description "TestSprite monthly run watchdog (story 19-7)" --force`). NO `pnpm/action-setup` / `setup-node` / `pnpm install` / TestSprite-CLI install steps — the watchdog is a pure file reader.
  - [x] Health-evaluation step (bash + `yq`): reads the queue file; verifies `schema_version == 2`; reads `last_run.status` + `last_run.finished_at`; computes `healthy`/`unhealthy` per AC #4 (status enum + 35-day staleness); exports the verdict + `reason` / `reason_short` strings.
  - [x] Issue-CRUD step (`actions/github-script@v7`): paginated dedup search by label `testsprite-quota-warning` (warns if >1 open Issue); executes the AC #2 4-branch decision tree (no-Issue+unhealthy → create, Issue+unhealthy → comment, no-Issue+healthy → no-op, Issue+healthy → ✅-close). Inputs passed via `env:` (not `${{ }}` interpolation) to avoid script injection.
  - [x] Top-of-file comment block: 28th-of-month cron rationale, AC #4 source-of-truth (queue file, schema v2, NO live API, NO secret), AC #2 dedup-label contract, correct-course re-scope note, links to story 19-7 + sibling 19-6.
- [x] Task 2: Validate the workflow (AC: #1, #9)
  - [x] `actionlint .github/workflows/testsprite-quota-warning.yml` → 0 issues (one fix: a literal `${{ }}` inside a JS comment was mis-parsed by actionlint's expression scanner; reworded).
  - [x] `pnpm exec prettier --check` clean — workflow file + all touched docs.
  - [x] Push the workflow on a feature branch; observe it in the Actions tab — ⏸️ DEFERRED per user decision (2026-05-20): owner-operational follow-up — requires a push to GitHub (see Dev Agent Record → Completion Notes).
  - [x] Fire `workflow_dispatch` to validate — ⏸️ DEFERRED per user decision (2026-05-20): owner-operational follow-up (depends on the push above).
  - [x] Cleanup the validation Issue — ⏸️ DEFERRED per user decision (2026-05-20): owner-operational follow-up (depends on the `workflow_dispatch` run above).
- [x] Task 3: Documentation (AC: #5, #10)
  - [x] `project-context.md` Last Updated header → prepended a `2026-05-20` story-19-7 entry.
  - [x] `project-context.md` "Monthly Cron Workflow (story 19-6)" sub-section → appended the month-end-watchdog addendum paragraph.
- [x] Task 4: Close-out regression (AC: #5, #9)
  - [x] `pnpm lint:all` → 0 errors / 122 warnings (exact baseline match).
  - [x] `pnpm nx test web` (1841 tests) + `pnpm nx test api` (Go suites) pass.
  - [x] `pnpm test:e2e --list` → 1663 tests / 36 files (unchanged — 19-7 adds 0 specs).
  - [x] `pnpm run test:visual` — ran; 1 **pre-existing** failure (`library-recently-added` `default`-state emoji-glyph drift), unrelated to 19-7 (0 rendering surfaces touched), filed as backlog `preexisting-fail-library-recently-added-visual` per the dev-story Epic-9c-Retro-AI-2 protocol. See Completion Notes.
  - [x] `pnpm run test:cleanup` no orphans.
  - [x] `ux-design.pen` untouched → screenshot workflow not triggered.
  - [x] Sprint-status entry: `ready-for-dev` → `in-progress` → `review` — all three transitions applied (`→ review` on 2026-05-20 after the user elected to defer Task 2's live validation).

## Dev Notes

### Why this story exists / where it sits in epic-19

- **19-7 is the watchdog for 19-6.** Six failure modes 19-6 cannot self-detect: (i) `TESTSPRITE_API_KEY` rotated / expired silently — 19-6 fails its account-info call, exits ≠ 0, fires an email — but only if the user reads workflow-failure emails (commonly muted); (ii) `TESTSPRITE_TARGET_URL` Variable unset or pointing at a stale tunnel — 19-6 succeeds at account-info but every test case fails; queue commits-back but with `last_run.status: "test-failures-only"` (or "api-failure"); (iii) GitHub-Actions branch-protection blocking 19-6's commit-back push — 19-6 exits ≠ 0 mid-run after consuming credits but failing to record them; (iv) user manually consumed 30+ credits via local `workflow_dispatch` between cron runs, starving the monthly cron of remaining budget; (v) the cron itself didn't fire (GitHub Actions has a documented "scheduled workflows may be delayed or skipped during periods of high load" — affects free-tier accounts more); (vi) 19-6 was disabled (user clicked "Disable workflow" in the Actions UI, possibly forgot). 19-7 (Mechanism C, re-scoped via correct-course 2026-05-20) catches modes (i), (ii), (iii), (v), (vi) by reading 19-6's `last_run.status` + run staleness at month-end — an `api-failure` / `test-failures-only` status, or a `last_run` that is null or >35 days stale, all flag an unhealthy 19-6, and the user has ~3 days to act. Mode (iv) "manual consumption starved the budget" is the one mode that genuinely needs a live credit count, which 19-6 [@contract-v2] does not expose — explicitly deferred to a potential future `19-7a`.
- **Why open a GitHub Issue instead of email / Slack / Discord:** (a) the existing project tooling is GitHub-centric (sprint-status / story files in repo, PRs as the work unit) — staying in GitHub keeps the alert in the same place as the work; (b) Issues are auditable (`git log` won't show, but `gh issue list --label testsprite-quota-warning --state all` gives the history); (c) no extra integration (no webhook URL, no bot token, no Slack workspace dependency); (d) Issues are dismissible (close = acknowledged) and reopenable (the dedup logic in AC #2 makes the workflow re-use rather than spam).
- **Why 28th-of-month, not 30th or 31st** (AC #1 deep-dive): cron's `* * 31 * *` does NOT trigger in months with 30 days (Apr, Jun, Sep, Nov) and never in Feb; `* * 30 * *` skips Feb; `* * 29 * *` mostly works but skips Feb in non-leap-years. `* * 28 * *` fires every month including Feb non-leap-year (Feb has 28 days every year). The 28th gives 3 days lead time in 31-day months, 2 days in 30-day, 0–1 in Feb — enough to manually catch up with one `workflow_dispatch` of 19-6 (~120 credits consumed in ~2–3 minutes given TestSprite's per-case latency).
- **Why a `last_run`-health watchdog, not a live credit threshold (Mechanism C, correct-course 2026-05-20):** the original design checked `remaining_credits > 30` via a live `testsprite_check_account_info` call. 19-6's Phase 2 implementation discovery proved the bare `@testsprite/testsprite-mcp` CLI exposes no such subcommand (only `generateCodeAndExecute` + `server`), and 19-6 [@contract-v2] leaves all `last_run.credits_*` fields null — so no credit count is available to 19-7 at all. Mechanism C instead judges 19-6 healthy/unhealthy from `last_run.status` + staleness (AC #4). The 35-day staleness window: 19-6 runs `0 3 1 * *`, 19-7 runs `0 3 28 * *`, so within healthy operation the freshest 19-6 run is ≤27 days old when 19-7 fires; >35 days means a whole monthly cycle was missed.
- **Dependency:** consumes ONLY **19-6 [@contract-v2] AC #2 (queue-file `schema_version: 2` + `last_run` block shape)**. **Confirmed against [@contract-v2] (Story 19-6 AC #2)** — `schema_version: 2`, `last_run` fields {`started_at`, `finished_at`, `credits_at_start`, `credits_at_end`, `credits_consumed`, `test_ids_run`, `status`, `run_url`}; the three `credits_*` fields are permanently null in v2 and are NOT read by 19-7. The READ is defensive — if `schema_version != 2`, AC #4 treats it as a HEALTH FAILURE (open an Issue) rather than crashing or rendering garbage. 19-7 no longer depends on 19-6 AC #4 (commit-message format) or AC #5/#6 (secrets) — the correct-course re-scope removed the live-credit + secret surface entirely.
- **What this story doesn't do:** doesn't write to `_bmad-output/audit/testsprite-queue.yaml` (read-only consumer); doesn't make any live TestSprite API call or install the TestSprite CLI; doesn't open an Issue per individual test-case failure (it reads the run-level `last_run.status` — a `test-failures-only` value flags the whole run, not each case; a future story could add per-failure Issue creation if useful); doesn't track a live remaining-credit count (mode (iv) — deferred to a potential `19-7a`).

### Architecture / constraints — read before implementing

- **Pure CI / DevOps story.** 0 Go, 0 frontend source, 0 tests authored. Cross-stack split: 0 backend, 0 frontend → single story, trivially correct.
- **Issue dedup via label, not title-substring-match.** Labels are stable, machine-queryable, and survive title edits (a user might rename the Issue title; the label persists unless explicitly removed). Title-substring matches break on rename. The HTML-comment marker in the Issue body (AC #3) is a tertiary defence — if a user removes the label AND renames the title, the marker survives Markdown preview and `gh issue list --search "testsprite-quota-warning-marker"` still finds the orphan.
- **`actions/github-script@v7` vs raw `gh` CLI:** `github-script` is the cleaner choice here because (a) the Issue body has multi-line Markdown that's painful to escape in a shell HEREDOC inside another HEREDOC (workflow YAML), (b) the dedup logic is conditional with branching (4 branches per AC #2) — `gh` chains of `if/then` against `gh issue list` get gnarly; JavaScript reads better, (c) `github-script` auto-handles `GITHUB_TOKEN` injection. Trade-off: adds the `actions/github-script` action dep — pinned `@v7` per current major. Document the choice in the workflow top-comment.
- **No TestSprite CLI, no Node toolchain.** The correct-course re-scope (Mechanism C) makes 19-7 a pure repo-file reader: `actions/checkout@v4` + `yq` (pre-installed on `ubuntu-24.04`) + `gh` / `actions/github-script@v7`. No `pnpm/action-setup`, no `setup-node`, no `pnpm install`, no `npx @testsprite/testsprite-mcp`. This is a deliberate simplification over the original design and eliminates any CLI-version drift with 19-6.
- **No `testsprite_check_account_info` call.** The original design's open question — whether account-info consumes credits — is moot under Mechanism C: 19-7 makes no TestSprite API call of any kind, reading only the in-repo queue file. A future `19-7a` wanting a live credit count would have to run the CLI in MCP-server mode and speak JSON-RPC to the `testsprite_check_account_info` tool — out of scope here.
- **The Issue body's permalink to `testsprite-queue.yaml`** (AC #3) MUST use a SHA-pinned URL (`github.com/{owner}/{repo}/blob/{SHA}/_bmad-output/audit/testsprite-queue.yaml`), NOT a branch-pinned one (`/blob/main/...`), because the file mutates monthly. The current `GITHUB_SHA` (the SHA the workflow checked out) is the right pin — captures the queue state at the moment the warning fired. Use `${{ github.sha }}` from the workflow context. If a future maintainer clicks the link 6 months later, they see the queue state as it was at warning time, not the current state.
- **Permission scoping** — `issues: write` is the minimum required for label create/Issue create/Issue comment/Issue close. `contents: read` is for the `actions/checkout@v4` step (need to read the queue file). No `actions: write`, no `pull-requests: write`, no `pages: write`. Both at the JOB level, not workflow level.
- **Auto-close behaviour edge case** — AC #2's "auto-close when 19-6 is healthy again" branch: if a user manually closes the Issue between 19-7 runs while 19-6 is still unhealthy, the next 19-7 run sees "no open Issue" and opens a NEW one (correct behaviour — user closed prematurely, the alert resurfaces). If a user manually reopens a closed Issue after auto-close, the next 19-7 run sees "open Issue + 19-6 healthy", posts the ✅ Resolved comment again, and closes it again. Both edge cases are acceptable churn — the dedup rule is robust to either.
- **Issue comment vs new Issue** for a re-fire while 19-6 is still unhealthy: AC #2 explicitly says "comment on existing Issue, do NOT open duplicate". An unresolved warning accrues 1 comment per scheduled run (1 per month). If the user fires `workflow_dispatch` ad-hoc 5 times in a day to validate, the Issue gets 5 comments. Acceptable — the comment thread tells the story of how 19-6's health state evolved across runs.

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
- **Rule 13 error handling (workflow-level):** the `gh` / github-script Issue operations use `set -e` (API failures = hard fail per AC #7 (a)). The `yq` read fails hard only if the queue file is missing/unparseable (AC #7 (b)); a `schema_version != 2` is caught and routed to the UNHEALTHY branch (open an Issue), not a crash.
- **`pnpm run test:cleanup`:** N/A in CI (ephemeral runner); applies to local Task 2 manual run.

### Rule 21 / Rule 22 / Rule 20 linkage

- **Rule 21 (Component-to-Design Node Traceability):** N/A (no component files).
- **Rule 22 (Epic Retro Design-Drift Audit):** indirectly supportive — 19-7's Issues are the surfacing layer for "19-6 ran into trouble"; Rule 22 retros that include a "did our test infra work this epic" check can grep `gh issue list --label testsprite-quota-warning --state all --search "in:title TestSprite"` for the epic's monthly health. No tooling-line change needed in project-context.md Rule 22 block — that block is about visual-regression diff tooling; TestSprite is journey-level (separate tracking surface).
- **Rule 20 (AC Contract Versioning):** stamps `[@contract-v1]` on AC #1–#3 (workflow trigger model + Issue dedup contract + Issue body format). **Upstream consumed:** confirmed against 19-6 **[@contract-v2]** AC #2 (queue file `schema_version: 2` + `last_run` block shape). The original draft acked 19-6 [@contract-v1]; correct-course 2026-05-20 re-acked to v2 after 19-6's CR closeout bump (commit `8094ca4`). No upstream contract bump by this story; no consumption of 19-6 AC #4 (commit-message format) — that cross-link was dropped in the re-scope. Downstream consumers: future Rule 22 retros may grep Issue titles/bodies; a future `19-7a` may extend the Issue body format.
- **Rule 7 (Error Codes):** N/A — pure CI / 0 Go.

### Latest tech information

- **`actions/github-script@v7`** — current stable; runs a Node.js script with `octokit`-style GitHub API access. Pinned `@v7` per current major (matches the project's documented major-pin convention used by `actions/checkout@v4`, `pnpm/action-setup@v4`, `actions/setup-node@v4`).
- **`gh` CLI** — pre-installed on `ubuntu-24.04` runners; version stays roughly current. Used for label create (`gh label create … --force`) — `actions/github-script` doesn't expose labels-CRUD as cleanly.
- **`yq` (`mikefarah/yq`)** — pre-installed on `ubuntu-24.04` runners; used to read `schema_version` + the `last_run` block from the queue file. No TestSprite CLI / npm package is used by this workflow.
- **GitHub Actions `schedule:` cron** — POSIX 5-field; `0 3 28 * *` = minute 0, hour 3 UTC, day-of-month 28, any month, any day-of-week. Documented as "may be delayed or skipped during high load on free-tier" — see Dev Notes failure mode (v) and (vi).
- **`secrets.GITHUB_TOKEN`** — auto-issued; permissions: `{ contents: read, issues: write }` job-scoped.

### References

- [Source: _bmad-output/planning-artifacts/sprint-change-proposal-2026-05-20.md] — correct-course re-scope: the original charter's `testsprite_check_account_info` / ">30 credits" mechanism was unimplementable (no CLI subcommand; v2 credits null); 19-7 re-framed to a `last_run`-health watchdog (Mechanism C).
- [Source: _bmad-output/implementation-artifacts/19-6-github-actions-testsprite-monthly.md] — sibling story; the consumer this story watches. Provides 19-6 [@contract-v2] AC #2 (queue file `schema_version: 2` + `last_run` block shape) consumed here. 19-6's Phase 2 CLI-shape discovery is the evidence that `testsprite_check_account_info` is not a CLI subcommand.
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

claude-opus-4-7 — Amelia (BMM Developer Agent)

### Debug Log References

- `actionlint` initially failed on the `script:` block — a literal `${{ }}` inside a JavaScript comment was parsed by actionlint's expression scanner ("unexpected end of input"). Fixed by rewording the comment ("workflow-expression interpolation"). Re-run: 0 issues.
- `pnpm run test:visual` failed twice consecutively on `library-recently-added` `default` state — investigated via the Playwright diff PNG (`test-results/.../library-recently-added/default-diff.png`): diff is localized to emoji/icon glyphs. Determined pre-existing (see Completion Notes).

### Completion Notes List

- **Story re-scoped before implementation via correct-course (2026-05-20).** The dev-story Step 2 AC Contract Drift Check found 19-7 (drafted 2026-05-18) targeted 19-6 `[@contract-v1]`; 19-6's CR closeout (commit `8094ca4`, 2026-05-19) had already bumped AC #2/#3/#4 to `[@contract-v2]`. The original mechanism — a live `testsprite_check_account_info` credit check — is unimplementable (the bare `@testsprite/testsprite-mcp` CLI exposes no such subcommand). Mechanism C was selected by the user (Alexyu): a queue-file `last_run` health watchdog. Full record: `_bmad-output/planning-artifacts/sprint-change-proposal-2026-05-20.md`.
- 🔗 **AC Drift: FOUND→RESOLVED** — resolved by the correct-course re-scope; this story now consumes only 19-6 `[@contract-v2]` AC #2 (queue `schema_version: 2` + `last_run` block shape).
- 📎 **Contract Stamps: FOUND** — 19-7 stamps `[@contract-v1]` on AC #1–#3; upstream 19-6 `[@contract-v2]` AC #2 consumed, ack lines present in Dev Notes "Dependency" + "Rule 20 linkage".
- 🔒 **Rule 7: N/A** — pure CI / 0 Go.
- 🎨 **UX Verification: SKIPPED** — no UI changes (AC #5: `ux-design.pen` untouched, 0 `apps/web/` edits).
- **Task 1 complete** — `.github/workflows/testsprite-quota-warning.yml` authored: cron `0 3 28 * *` + `workflow_dispatch`, `ubuntu-24.04`, job permissions `{ contents: read, issues: write }`, concurrency `testsprite-quota-warning` / `cancel-in-progress: false`. Pure repo-file reader — `yq` health-eval step + `actions/github-script@v7` 4-branch Issue dedup. Uses NO TestSprite secret and installs NO TestSprite CLI.
- **Task 2 — local validation done; live validation DEFERRED (owner-operational follow-up).** `actionlint` 0 issues + `prettier --check` clean. The push-to-GitHub + `workflow_dispatch` live run + validation-Issue cleanup mirror the 19-5 (branch-protection) / 19-6 (`workflow_dispatch` live run) owner-operational deferral precedent. ⏸️ **FOLLOW-UP:** push `.github/workflows/testsprite-quota-warning.yml`, then fire `workflow_dispatch` — against the current queue (`last_run.status: null`, never run) the watchdog will correctly evaluate UNHEALTHY (never-run branch) and open exactly ONE Issue; verify a re-fire comments rather than duplicates; then close the validation Issue (or let it auto-close on 19-6's first healthy run).
- **Task 4 — pre-existing test failure handled per Epic-9c-Retro-AI-2 protocol.** `pnpm run test:visual` fails on `library-recently-added` `default` state — the diff is localized to the 🎬 poster-placeholder emoji + a metadata-source badge glyph (emoji/icon-glyph rendering drift vs the committed `-darwin` baseline), consistent across 2 runs (not a flake). Story 19-7 changed 0 rendering surfaces (workflow YAML + `project-context.md` + `_bmad-output/**` docs only — verified via `git status`) → pre-existing, not introduced here. Option chosen: **FILE** (non-trivial — a baseline re-bless needs the 19-4b/19-5 `requires-manual-review` UX gate, and AC #5 forbids `tests/visual/` edits) — tracked as `preexisting-fail-library-recently-added-visual: backlog` in `sprint-status.yaml`. All other regression gates green: `lint:all` 0/122, `nx test web` 1841 pass, `nx test api` pass, `test:e2e --list` 1663/36, `test:cleanup` no orphans.

### File List

**Created:**

- `.github/workflows/testsprite-quota-warning.yml` — the month-end watchdog workflow (story 19-7 deliverable).
- `_bmad-output/planning-artifacts/sprint-change-proposal-2026-05-20.md` — the correct-course re-scope proposal.

**Modified:**

- `project-context.md` — Last Updated header (19-7 entry) + "Monthly Cron Workflow (story 19-6)" sub-section addendum paragraph.
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — 19-7 `ready-for-dev` → `in-progress`; added the `preexisting-fail-library-recently-added-visual` backlog row.
- `_bmad-output/implementation-artifacts/19-7-github-actions-quota-warning.md` — this story file (correct-course re-scope + Tasks checkboxes + Dev Agent Record + Change Log + Status).

## Change Log

| Date | Change |
| ---- | ------ |
| 2026-05-18 | SM Bob /create-story (YOLO) — story drafted ready-for-dev. Pure CI / DevOps; 0 Go / 0 frontend source / 0 tests authored → single story (cross-stack split N/A; backend tasks = 0). 10 ACs (#1–#3 stamped `[@contract-v1]`), 4 tasks. Key SM decisions recorded in Dev Notes: (1) **28th-of-month cron** — `0 3 28 * *` is the safest "near-month-end" pattern; `*/30/31` cron expressions skip Feb / 30-day months. 28th gives 2–3 days lead time for manual catch-up. (2) **Threshold > 30, not ≥ 30** — 30 is 19-6's `reserved_credits` floor; exact 30 = correct steady state, no warning; > 30 = unused credits about to expire. (3) **Issue dedup via label (`testsprite-quota-warning`), not title-substring** — labels are stable + machine-queryable; HTML-comment marker in body is tertiary defence. (4) **`actions/github-script@v7` over raw `gh` CLI** — cleaner for multi-line Markdown Issue body + 4-branch conditional dedup decision tree; auto-handles GITHUB_TOKEN. (5) **Auto-close on threshold-drop** — when remaining ≤ 30 AND an open Issue exists, the workflow posts `✅ Resolved` and closes; user doesn't manually close monthly. (6) **No new secrets** — reuses 19-6's `TESTSPRITE_API_KEY`; account-info is API-only, no `TESTSPRITE_TARGET_URL` consumption. (7) **Account-info assumed FREE** (no per-call credit cost) — documented as defensive assumption; fallback paths sketched in Dev Notes if validation reveals otherwise. (8) **Six failure modes 19-6 cannot self-detect** explicitly enumerated in Dev Notes — auth rotation, target unreachable, branch-protection block, manual-consumption starve, cron skip, workflow disabled. 19-7 catches ALL via the live-credit threshold check. (9) **Issue body permalink uses `${{ github.sha }}`-pinned URL** to `testsprite-queue.yaml`, not `/blob/main/` — captures queue state at warning-time. Consumes 19-6 [@contract-v1] AC #2 (schema_version + last_run shape) + AC #4 (commit-message format) per Rule 20 ack — confirmed against both. 🔒 Rule 7 Wire Format: N/A (pure CI / 0 Go). 🎨 UX: N/A (no .pen modification). |
| 2026-05-18 | [@contract-v0→v1] AC #1–#3 stamped on creation — what's defined: workflow file at `.github/workflows/testsprite-quota-warning.yml` + name `TestSprite Quota Warning` + cron `0 3 28 * *` UTC + `workflow_dispatch` trigger + `ubuntu-24.04` runner pin (AC #1); dedup contract — at most one open Issue with label `testsprite-quota-warning` at any time; 4-branch decision tree (no-Issue+breached → create, Issue+breached → comment, no-Issue+OK → no-op, Issue+OK → ✅-close); label-create idempotent via `--force` with colour `FFA500` (AC #2); Issue title format `[TestSprite] {N} credits unused at month-end ({YYYY-MM})` + body Markdown template with Why/What-to-do/Audit-trail-snapshot/Workflow-run sections + trailing `<!-- testsprite-quota-warning-marker -->` HTML comment (AC #3). What breaks downstream: future Rule 22 retros / a quota-warning-Slack-bridge story may grep Issue titles or bodies — silently changing the title format breaks the grep; the marker comment is the tertiary dedup key if the label is hand-removed — silently dropping it weakens the dedup robustness. Upstream consumption: confirmed against [@contract-v1] (Story 19-6 AC #2 — queue file `schema_version: 1` + last_run block shape consumed in audit-trail-snapshot rendering), confirmed against [@contract-v1] (Story 19-6 AC #4 — commit-message format referenced indirectly via the audit-trail cross-link in the Issue body). No upstream contract bump — this story READS 19-6's contract surface, does not modify it. |
| 2026-05-20 | **correct-course re-scope** (dev-story HALT trigger; full proposal `_bmad-output/planning-artifacts/sprint-change-proposal-2026-05-20.md`). dev-story Step 2 AC Contract Drift Check found 19-7 was drafted `2026-05-18` against 19-6 `[@contract-v1]`; 19-6 CR closeout `2026-05-19` (commit `8094ca4`) had bumped AC #2/#3/#4 to `[@contract-v2]`. Three conflicts: (1) **BLOCKING** — AC #1/#4/#7 + Task 1 required calling `testsprite_check_account_info` via the CLI, but the bare `@testsprite/testsprite-mcp` CLI has only `generateCodeAndExecute` + `server` subcommands (19-6 Phase 2 discovery); (2) Dev Notes hard-coded `schema_version = 1`, actual is `2`; (3) AC #3 Issue body rendered `last_run.credits_consumed`, permanently null in v2. Mechanism C selected (Alexyu): 19-7 re-framed from a live-credit threshold to a queue-file `last_run` watchdog — UNHEALTHY when `status` ∈ {api-failure, test-failures-only}, or `last_run` null / >35-day stale, or `schema_version != 2`. Rewritten: title, marker block, Story, AC #2/#3/#4/#6/#7/#10, Tasks 1–4, Dev Notes. Kept: AC #1/#5/#8/#9. Contract dependency narrowed to 19-6 `[@contract-v2]` AC #2 only — the live-credit + `TESTSPRITE_API_KEY` secret dependency on 19-6 AC #4/#5/#6 is removed (19-7 needs no TestSprite secret). Live-credit alerting deferred to a potential future `19-7a`. No `[@contract-v*]` bump on 19-7's own stamps — story never shipped, v1 re-drafted in place. |
| 2026-05-20 | DEV Amelia /dev-story (Opus 4.7 1M ctx) — implementation after the correct-course re-scope. Created `.github/workflows/testsprite-quota-warning.yml` (month-end `last_run` watchdog — `yq` health-eval + `actions/github-script@v7` 4-branch Issue dedup; cron `0 3 28 * *` + `workflow_dispatch`; `ubuntu-24.04`; job permissions `{ contents: read, issues: write }`; no TestSprite secret, no TestSprite CLI). **Task 1 + Task 3** complete. **Task 2** local validation done (`actionlint` 0 issues — fixed a literal `${{ }}` in a JS comment; `prettier --check` clean); push + `workflow_dispatch` live run + Issue cleanup ⏸️ DEFERRED as an owner-operational follow-up (mirrors 19-5/19-6 precedent). **Task 4** regression: `pnpm lint:all` 0 errors / 122 warnings, `pnpm nx test web` 1841 pass, `pnpm nx test api` pass, `pnpm test:e2e --list` 1663/36, `pnpm run test:cleanup` no orphans; `pnpm run test:visual` surfaced 1 PRE-EXISTING failure (`library-recently-added` `default`-state emoji-glyph drift — 19-7 changed 0 rendering surfaces) → filed `preexisting-fail-library-recently-added-visual` backlog row in `sprint-status.yaml` per the Epic-9c-Retro-AI-2 protocol. Status: `in-progress` → `review` — the user elected (2026-05-20) to defer Task 2's push / `workflow_dispatch` live validation as an owner-operational follow-up, per the 19-5/19-6 precedent. |
