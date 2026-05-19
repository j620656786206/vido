# Story: Bugfix 19.4b-1 Follow-up — Bisect Spec CI Coverage Restoration (Playwright Dev-Mode Project)

Status: ready-for-dev

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->
<!-- SM Bob /create-story (YOLO) 2026-05-19 — filed by story 19-5 CR (Amelia /code-review 2026-05-19) during status-bump re-examination. Original draft listed this as 19-5 finding "M3"; on authorship-chain re-examination, moved to a standalone follow-up scoped to bugfix-19-4b-1 (root cause: bisect spec's design + story 19-4 CR M1's PROD-gate, both predate 19-5). -->
<!-- This story consumes 19-4 [@contract-v1] AC #1 unchanged (existing `visual` project untouched) and STAMPS a new [@contract-v1] on AC #2 (the new `bisect` Playwright project's identity + testMatch + dev-mode webServer). Adds the new project as a SIBLING to existing 6 projects in `playwright.config.ts`; additive change, no breaking modification. -->
<!-- 🔗 AC Drift: NEW (additive — 19-4 [@contract-v1] AC #1 consumed unchanged; project-list widening, not modification). · 📎 Contract Stamps: NEW (v1×1 candidate on AC #2 — `bisect` project identity; final stamp decision deferred to dev). · 🔒 Rule 7 Wire Format: N/A (pure FE + test-infra; no Go). · 🎨 UX: N/A (bisect is a regression-gate spec, not a UI; `ux-design.pen` untouched ⇒ CLAUDE.md screenshot workflow NOT triggered). -->
<!-- markers-block-end -->

## Story

As a frontend maintainer running CI on PRs that touch `apps/web/src/components/parse/**` (especially `useParseProgress.ts` or any callback-prop-stable hook that follows the same pattern),
I want the bisect spec `tests/e2e/bisect-bugfix-19-4b-1.spec.ts` to actually run in CI as the **AC #2 regression gate it was authored to be** (bugfix-19-4b-1's `multi.warnCount === 0` + per-fixture `offenderCount === 0` integration assertion),
so that a future PR that re-introduces the callback-prop-identity churn (the bugfix-19-4b-1 root cause: fresh `{ onConnected, … }` object literal per render → `handleEvent` deps churn → `useEffect` re-fires → setProgress loop → React-18 depth limiter at 50) is caught at PR-review time — not only by the narrower `useParseProgress.spec.ts` unit test that covers callback identity stability but does **not** exercise the integrated multi-fixture render-loop scenario the bisect spec walks across 123 fixtures.

## Acceptance Criteria

1. **Bisect spec moved to a dedicated location.** `tests/e2e/bisect-bugfix-19-4b-1.spec.ts` MUST be relocated to `tests/bisect/bisect-bugfix-19-4b-1.spec.ts` (mirrors the proven `tests/visual/` pattern: dedicated directory → dedicated Playwright project → fewer config surfaces). After the move, the file MUST NOT exist under `tests/e2e/` and MUST NOT be picked up by the `chromium` / `firefox` / `mobile-chrome` / `mobile-safari` projects (which have NO `testMatch` filter today and therefore sweep everything under `tests/e2e/`). Existing in-file imports / relative paths (`OUT_DIR = path.resolve(__dirname, '../../_bmad-output/...')`) work unchanged after the move — both `tests/e2e/` and `tests/bisect/` are at the same depth relative to repo root.

2. [@contract-v1] **New `bisect` Playwright project added to `playwright.config.ts`.** A new project block MUST be added to the `projects:` array (anywhere — recommend placement adjacent to the existing `visual` project so the two dev-tool-style projects sit together) with the following identity:
   - `name: 'bisect'` (lowercase, no hyphen, matches `--project=bisect` invocation pattern).
   - `testDir: path.resolve(__dirname, './tests/bisect')` (own directory, isolates from feature-E2E sweep, mirrors `visual` project pattern at L150).
   - `testMatch: ['**/bisect-*.spec.ts']` (explicit redundancy today — `tests/bisect/` only holds bisect specs — but mandated by AC #1 spirit so future non-bisect specs under `tests/bisect/` MUST NOT be swept into this project).
   - `use: { ...devices['Desktop Chrome'] }` (chromium-only: the in-spec `test.skip(browserName !== 'chromium', ...)` becomes redundant once project-scoped; AC #3 decides whether to keep or strip the in-spec guard).
   - **No `viewport` / `colorScheme` / `reducedMotion` overrides** — the bisect spec measures render-cycle behaviour (console warnings), not pixels; the defaults are fine.
   - **Dedicated `webServer:` entry** inside the project block (NOT a global webServer) that starts `nx serve web` (Vite dev mode, port 4200) — because `/test/gallery` is gated behind `!import.meta.env.PROD` (`apps/web/src/routes/test/gallery.tsx:90-97`, story 19-4 CR M1 safety boundary) and is unreachable in any prod build. Mirror story 19-5's `.github/workflows/visual-regression.yml:184-196` wait-loop pattern if the global `webServer` config cannot be project-scoped (it cannot — Playwright's `webServer` is global, not per-project; so the workflow-level `webServer` startup is the right surface — see AC #5).
   - **Downstream impact (what breaks if this contract is silently changed):** future bugfix-N stories that need dev-mode integration regression gates depend on the `bisect` project's identity to invoke `--project=bisect` and on the `**/bisect-*.spec.ts` testMatch to auto-pick-up new bisect specs. If a future change renames the project, narrows `testMatch`, or switches the webServer to prod build, the regression coverage silently regresses — same failure class as story 19-5 H2 (path filter dead pattern).

3. **CI skip removed from the bisect spec.** `tests/bisect/bisect-bugfix-19-4b-1.spec.ts:132-135` (the lines that are currently L132-L135 in `tests/e2e/bisect-bugfix-19-4b-1.spec.ts` before the move) contains:
   ```ts
   test.skip(
     process.env.CI === 'true',
     'bisect probe is local-only — prod-build CI cannot reach /test/gallery (Access Denied gate)'
   );
   ```
   This block MUST be removed (the predicate is no longer true once the spec runs under the `bisect` project with dev-mode webServer). The accompanying long comment (L125-L131 today, explaining the prod-build gate) MUST be rewritten to a 1–2-line note about the dev-mode requirement, OR deleted entirely if AC #2's project config + this story's commit message provide sufficient archaeological context. **The browser-agnostic skip (`test.skip(browserName !== 'chromium', ...)` at L121-L124) MAY be kept as defence-in-depth or removed as redundant** — implementer's call; document in Completion Notes.

4. **New npm scripts.** `package.json` MUST gain:
   ```json
   "test:bisect": "playwright test --project=bisect"
   ```
   Mirroring the `test:visual` / `test:visual:update` pattern (but **no `:update` script** — bisect spec doesn't produce snapshots; it asserts numeric counts). The `test:e2e` script's `--project=` list MUST NOT include `--project=bisect` (the bisect project is conceptually a regression-gate spec, not a feature-E2E spec; running it in every `pnpm test:e2e` invocation is wasteful — ~3-5 min wall-clock for a probe that only matters when `apps/web/src/components/parse/**` or `tests/bisect/**` changes).

5. **CI wiring — new standalone workflow.** A new file `.github/workflows/bisect-regression.yml` MUST be created. Required shape:
   - **Triggers:** `pull_request` against `main` / `develop` AND `push` to `main` / `develop` AND `workflow_dispatch`. Use the story-19-5 H2-corrected always-trigger pattern: trigger on ALL PRs, then internal `dorny/paths-filter@v3` early-exit on irrelevant PRs (so the check name appears consistently and can be set as a required branch-protection check without the Catch-22 story 19-5 documented). Path filter MUST cover at minimum: `apps/web/src/components/parse/**`, `apps/web/src/routes/test/gallery.tsx`, `apps/web/src/routes/test/-gallery.fixtures.tsx`, `tests/bisect/**`, `playwright.config.ts`, `package.json`, `.github/workflows/bisect-regression.yml`. (Narrow filter — the bisect spec exists specifically to guard the bugfix-19-4b-1 root cause, which lives in the parse hook + gallery fixture stack. Widening to `apps/web/src/components/**` would dilute the signal.)
   - **Runner:** `runs-on: ubuntu-24.04` (mirror story 19-5 AC #5's pin policy — same `actions/runner-images` reproducibility lever; same `ImageVersion`-capture pattern if any audit-trail line ever appended).
   - **Steps:** checkout → pnpm setup → Node setup with `.nvmrc` + `cache: 'pnpm'` → `pnpm install --frozen-lockfile` → `npx playwright install --with-deps chromium` → start `nx serve web` in background with fail-fast wait-loop (mirror story 19-5 M2 pattern: track `ready` flag, `exit 1` + log tail on timeout) → `pnpm run test:bisect` → upload diff artefacts on failure (`if: always() && steps.filter.outputs.bisect == 'true'`).
   - **Permissions:** job-level `{contents: read}` only (the bisect workflow opens no PRs, commits nothing — purely verify-only). Minimum-privilege per story 19-5's AC #2 pattern.
   - **Concurrency:** `group: bisect-regression-${{ github.workflow }}-${{ github.ref }}` with `cancel-in-progress: true` for PRs; separate group + `cancel-in-progress: false` for main pushes (story 19-5 AC #7 pattern).
   - **Job names:** top-level `name: Bisect Regression`; PR job `name: PR` (renders as `Bisect Regression / PR`); main job `name: Main` (renders as `Bisect Regression / Main`).
   - **No `webServer:` reliance on `playwright.config.ts`** — the workflow starts `nx serve web` explicitly (mirrors 19-5), so the global `webServer:` config short-circuits to `[]` in CI (as it does today, line 167). The dev server is the workflow's responsibility, not Playwright's.

6. **`test:e2e --list` count returns to bugfix-19-4b-1's stale-charter baseline.** After the move, `pnpm test:e2e --list` MUST report **1663 tests / 36 files** (the pre-bugfix-19-4b-1 baseline, restored because the bisect spec is no longer swept into the chromium/firefox/mobile-* projects). The current 1667/37 count includes the bisect spec × 4 effective browser projects after webkit-core's `testMatch` filter exclusion (chromium + firefox + mobile-chrome + mobile-safari, all sweeping `tests/e2e/`); removing the file from `tests/e2e/` strips all 4 entries. Record the count delta (1667/37 → 1663/36) in Completion Notes; this is observable in `pnpm test:e2e --list` output, not a contract claim.

7. **Regression hygiene gates.** `pnpm lint:all` MUST stay **0 errors / 122 warnings** (the bugfix-10-7 / 19-3 / 19-4 / 19-4b / 19-5 baseline); `pnpm exec prettier --check` MUST pass on all touched files (workflow file + `playwright.config.ts` + `package.json` + moved bisect spec + this story file + `sprint-status.yaml`); `actionlint .github/workflows/bisect-regression.yml` MUST report **0 issues** (mirror story 19-5 lint discipline); `pnpm nx test web` + `pnpm nx test api` MUST PASS (no source changes outside `playwright.config.ts` and the bisect spec move); `pnpm run test:visual` MUST PASS against the committed 262 `-darwin` + 262 `-linux` baselines with **0 re-blessings** (visual project unaffected — bisect's project-scoped settings don't touch the visual project's config); `pnpm run test:bisect` MUST PASS locally against `nx serve web` running on 4200 (the spec was already passing locally pre-CI-skip; the only behavioural change in this story is "now runs under its own project / webServer" not "fixes anything inside the spec"); `pnpm run test:cleanup` MUST report no orphaned processes; `ux-design.pen` MUST be git-status clean (CLAUDE.md screenshot workflow NOT triggered — this is pure CI / test infra; zero design surface touched).

8. **Rule 20 ack — 19-4 [@contract-v1] AC #1.** Story 19-4's [@contract-v1] AC #1 defines the existence and identity of the `visual` Playwright project (project name, chromium-base, 1280×800 viewport, dark color scheme, reducedMotion, maxDiffPixelRatio). This story consumes that contract **unchanged** — the `visual` project's existing block in `playwright.config.ts` is not modified. Adding a SIBLING `bisect` project to the `projects:` array is an additive widening of the project list, not a modification of any existing project. Per Rule 20: confirmed-against `[@contract-v1] (Story 19-4 AC #1)`; no upstream contract bump needed. **This story STAMPS its own [@contract-v1] on AC #2** (the new `bisect` project's name + testMatch + chromium-base; documented in the AC #2 "Downstream impact" bullet for future Rule 22 retros + bugfix-N drift stories).

9. **Operational follow-up (owner-driven, post-merge).** Branch-protection rule for `main` / `develop`: Settings → Branches → Branch protection rules → "Require status checks to pass before merging" → tick `Bisect Regression / PR` (same half-CI / half-policy pattern as story 19-5 AC #2). The workflow MAKES the check appear in PR UI; making it `required` is a one-click web-UI action the workflow cannot perform itself. Surface this in Completion Notes as a follow-up the user/owner does post-merge — single web-UI click; the story's contract is the *check existing and being available to mark required*, not the policy click.

## Tasks / Subtasks

> **Cross-Stack Split Check** (Agreement 5, Epic 8 + 9c Retro): Backend tasks = **0** / Frontend tasks = **0** / Test-infra tasks = **4** → single story trivially correct; the `>3 each side` threshold is not met (this isn't even a "stack" boundary — it's a CI-tooling + Playwright-config boundary, same shape as story 19-5).
>
> **Scope inheritance:** the bugfix-19-4b-1 root-cause fix (callback identity via `useRef` in `useParseProgress.ts`) is **already in main** and is the unit-tested regression target (`useParseProgress.spec.ts`'s "does not reconnect on parent rerender with fresh inline options (bugfix-19-4b-1)" test). This story restores the **integration-level** regression gate the bisect spec was authored to be. No source-code change to `apps/web/src/components/parse/**` is in scope.

- [ ] **Task 1: Move bisect spec + add `bisect` Playwright project** (AC: #1, #2, #3)
  - [ ] **1.1** `git mv tests/e2e/bisect-bugfix-19-4b-1.spec.ts tests/bisect/bisect-bugfix-19-4b-1.spec.ts` (preserves git blame; creates the new directory). Verify `_bmad-output/implementation-artifacts/` relative path in the spec still resolves: `OUT_DIR = path.resolve(__dirname, '../../_bmad-output/implementation-artifacts')` — both `tests/e2e/` and `tests/bisect/` are at depth 2 from repo root, so the `../../` traversal works unchanged.
  - [ ] **1.2** Add the new project block to `playwright.config.ts`'s `projects:` array (placement: immediately after the `visual` project at L148-L162, so the two regression-gate / dev-tool projects sit together). Required fields per AC #2: `name: 'bisect'`, `testDir: path.resolve(__dirname, './tests/bisect')`, `testMatch: ['**/bisect-*.spec.ts']`, `use: { ...devices['Desktop Chrome'] }`. Add a comment block above the project explaining: (a) why bisect lives here (story 19-4b-1 regression gate, story 19-5 CR reclassification origin), (b) why no `viewport`/`colorScheme` (measures console behaviour, not pixels), (c) why NOT in `tests/e2e/` (excluded from feature-E2E sweep — `test:e2e --list` count discipline).
  - [ ] **1.3** Remove the CI skip (`test.skip(process.env.CI === 'true', ...)` + preceding multi-line comment block describing the prod-build gate) from `tests/bisect/bisect-bugfix-19-4b-1.spec.ts:125-135` (post-move line numbers). Reduce the leading comment block to 2-3 lines: "Runs under the `bisect` Playwright project (dev-mode webServer); does NOT run under the feature-E2E projects (chromium/firefox/mobile-*). See `.github/workflows/bisect-regression.yml` for CI wiring."
  - [ ] **1.4** Decide whether to keep or strip the in-spec `test.skip(browserName !== 'chromium', ...)` guard (L121-L124 post-move). **Recommended: keep as defence-in-depth** (project config can be edited; in-spec guard is harder to silently misconfigure). Document the decision in Completion Notes as a one-line rationale.
  - [ ] **1.5** Verify local: `pnpm playwright test --project=bisect` against `nx serve web` running on 4200. Expected: 1 passed, ~3-5 min wall-clock, OUT_PATH (`_bmad-output/implementation-artifacts/bisect-bugfix-19-4b-1-dev.json`) written.

- [ ] **Task 2: npm scripts + CI workflow file** (AC: #4, #5)
  - [ ] **2.1** Add to `package.json` after the `test:visual:update` line:
    ```json
    "test:bisect": "playwright test --project=bisect",
    ```
    Do NOT add `--project=bisect` to `test:e2e` / `test:e2e:ui` / `test:e2e:headed` / `test:e2e:debug` (bisect is a focused regression gate, not part of the feature-E2E suite — running it on every developer `pnpm test:e2e` invocation is wasteful).
  - [ ] **2.2** Create `.github/workflows/bisect-regression.yml`. Reference template: `.github/workflows/visual-regression.yml` (story 19-5) — copy its top-of-file documentation pattern, dorny-filter early-exit, fail-fast wait-loop, concurrency split, permissions block, artifact upload. Adjust: workflow name `Bisect Regression`; paths filter per AC #5 (narrower than visual's filter — only the parse hook + gallery fixture stack + bisect spec + playwright config + this workflow + package.json); both jobs invoke `pnpm run test:bisect` after `nx serve web` is ready; no bootstrap branch (bisect has no per-platform baselines to commit); artifact name `bisect-regression-results-{pr|main}-${{ github.run_id }}` (JSON output + Playwright traces).
  - [ ] **2.3** Run `actionlint .github/workflows/bisect-regression.yml` clean (use `brew install actionlint` if not on PATH; same setup story 19-5 used). Run `pnpm exec prettier --check .github/workflows/bisect-regression.yml playwright.config.ts package.json` clean.
  - [ ] **2.4** Verify the workflow appears in the GitHub Actions UI after pushing the branch (workflow file appears on push regardless of trigger match — same pattern story 19-5 used).

- [ ] **Task 3: Verify e2e count delta + regression gates** (AC: #6, #7)
  - [ ] **3.1** `pnpm test:e2e --list` returns **1663 tests / 36 files** (delta: -4 tests / -1 file vs the current 1667/37 baseline). Record the exact pre/post numbers in Completion Notes — this is observable verification, not a contract.
  - [ ] **3.2** `pnpm lint:all` 0 errors / 122 warnings (baseline match exact — no source-code changes outside `playwright.config.ts` mean lint surface is unchanged; the new workflow is YAML / not under ESLint scope).
  - [ ] **3.3** `pnpm run test:visual` PASS against the committed 262 darwin + 262 linux baselines, 0 re-blessings. The visual project's config is byte-identical pre/post this story (the new `bisect` project is a sibling addition; doesn't touch `visual`'s block at playwright.config.ts L148-L162).
  - [ ] **3.4** `pnpm run test:bisect` PASS locally (~3-5 min wall-clock).
  - [ ] **3.5** `pnpm nx test web` + `pnpm nx test api` PASS (no source changes).
  - [ ] **3.6** `pnpm run test:cleanup` no orphans.
  - [ ] **3.7** `git status ux-design.pen` clean. `git status _bmad-output/screenshots/` clean (CLAUDE.md screenshot workflow correctly does NOT trigger — no `.pen` modification).

- [ ] **Task 4: Documentation + Rule 20 ack + close** (AC: #8, #9)
  - [ ] **4.1** Add a Rule 20 ack line in Dev Notes: "confirmed against [@contract-v1] (Story 19-4 AC #1 — `visual` project identity); no upstream bump." Add the `[@contract-v0→v1]` Change Log row stamping AC #2 (new `bisect` project's identity + testMatch + chromium-base) with the standard {what changed, what breaks downstream} framing.
  - [ ] **4.2** Operational follow-up note in Completion Notes (per AC #9): branch-protection toggle for `Bisect Regression / PR` is the owner's post-merge web-UI click; the workflow's contract is the check existing and being mark-as-required-capable.
  - [ ] **4.3** **No `project-context.md` Rule 22 edit needed** — Rule 22 explicitly scopes to the design-drift surface (component visual baselines), and bisect is a different regression class (callback-identity / render-cycle hygiene). Do NOT extend the Rule 22 tooling sentence with this workflow; that would dilute Rule 22's narrative. If `project-context.md` Last Updated header is touched, scope the entry to "story bugfix-19-4b-1-followup landed Bisect Regression workflow" and keep it brief.
  - [ ] **4.4** Update sprint-status.yaml: `bugfix-19-4b-1-followup-bisect-spec-ci-coverage: backlog → ready-for-dev → in-progress → review` with the standard Completion Notes summary line. Story Status header → `review` at close-out.

## Dev Notes

### Why this story exists / its origin in the 19-x family

- **Origin:** filed by story 19-5 CR (Amelia /code-review 2026-05-19) during status-bump re-examination. The 19-5 CR originally listed this as finding "M3"; on authorship-chain re-examination (which story introduced what), it became clear that:
  - The `/test/gallery` `!import.meta.env.PROD` PROD-gate (`gallery.tsx:90-97`) was added by **story 19-4 CR M1** (security: NAS deployment must not see dev-only gallery route).
  - The bisect spec (`tests/e2e/bisect-bugfix-19-4b-1.spec.ts`) was authored by **bugfix-19-4b-1** as the AC #2 regression gate for "Phase A multi-fixture browse = 0 warnings AND per-fixture offenderCount = 0".
  - The bisect spec's design (`expect(ids.length > 0)` after navigating to `/test/gallery`) is inherently incompatible with the prod-build CI shard in `test.yml` (which uses `nx build --configuration=production` + `npx serve dist` → PROD-gate fires → `ids` stays empty → assertion fails).
  - Story 19-5 commit `e70f84a` ("fix(19-5): skip bisect spec in CI (prod-build can't reach /test/gallery)") added `test.skip(process.env.CI === 'true', ...)` as an **inline patch** to unblock 19-5's ruleset enforcement (under the post-19-5 ruleset, any failing check on a PR — including the previously-tolerated bisect-spec failure — blocks merge). 19-5's commit message explicitly framed this as "Pre-existing fix #2 (Epic 9c Retro AI-2 inline option)" — meaning "surface patch to unblock the current story's ruleset push; the proper root-cause fix is a follow-up scoped to the original story that authored the broken spec".
  - The proper root-cause fix (this story) needs a NEW Playwright project running the bisect spec against Vite dev mode (so the PROD-gate doesn't fire). That structurally requires modifying `playwright.config.ts` — which is part of story 19-4's [@contract-v1] surface (project list) — and is therefore outside story 19-5's CI-subsystem scope. Inheriting back to bugfix-19-4b-1 (which authored the spec) was the correct decoupling.
- **Forward-looking value:** the dev-mode webServer pattern this story introduces (a Playwright project with its own `nx serve web` startup, isolated from the feature-E2E suite's prod-build pattern) is a reusable substrate for any future regression gate that needs dev-only route access. Specifically: any `/test/<dev-route>` URL gated behind `!import.meta.env.PROD` (gallery, search-debugger, etc.) can be regression-tested under this project's webServer without compromising the prod-gate's safety boundary. If a future bugfix lands a similar dev-only regression-gate spec, it can `tests/bisect/<name>.spec.ts` and inherit the project's config trivially.

### Architecture / constraints — read before implementing

- **Pure CI / test-infra story. 0 Go, 0 frontend source code, 0 component tests authored, 0 migrations, 0 swagger.** Cross-stack split check: backend task count = 0, frontend-source task count = 0 → single story is trivially correct.
- **Why a new project instead of unblocking the existing chromium/firefox sweep.** The existing chromium/firefox/mobile-* projects have NO `testMatch` filter; they sweep everything under `tests/e2e/`. Three rejected alternatives to a new `bisect` project:
  - **(a) Add `testMatch` filters to all 4 e2e projects excluding `**/bisect-*.spec.ts`.** Rejected: 4 surfaces to maintain; future bisect specs would need to remember to add themselves to each project's exclude list; brittle.
  - **(b) Add `testIgnore: ['**/bisect-*.spec.ts']` to the global `use:` config.** Rejected: would exclude the bisect spec from the `bisect` project too (testIgnore is project-inherited unless overridden). Requires per-project re-overriding, defeating the simplification.
  - **(c) Add a separate `webServer` block per spec via Playwright's setup-hook pattern.** Rejected: Playwright's `webServer:` is global, not per-spec; no setup-hook path exists. Per-project webServer is also not natively supported — but per-project `webServer` IS allowed if defined at the test-runner config level by conditioning on `process.env.PROJECT` or similar. Convoluted; the cleaner path is "new project, new directory, dedicated webServer in the workflow".
- **webServer in workflow, not in playwright.config.ts.** Playwright's `webServer:` block in `playwright.config.ts:167-189` already short-circuits to `[]` in CI (`process.env.CI === 'true'`). The visual-regression workflow's pattern is to start `nx serve web` in a workflow step; the bisect workflow mirrors this. Reasons: (i) workflow-step startup provides fail-fast log capture that `webServer.command:` doesn't (story 19-5 M2 motivation), (ii) avoids coupling the `playwright.config.ts` `webServer:` block to multiple CI workflows with different needs (visual + bisect both need dev mode; future workflows might need prod build).
- **Why not bake bisect into the visual-regression workflow.** Three considerations:
  - **(i) Wall-clock budget.** Visual workflow: ~45s (1 spec × 123 fixtures × ~0.35s/screenshot diff). Bisect workflow: ~3-5 min (1 spec × 123 fixtures × ~1.5s/navigation+settle). Combining them in a single workflow ~5x's the wall-clock; the visual workflow's "fast feedback on every visual-touching PR" virtue degrades.
  - **(ii) Trigger-path separation.** Visual workflow's path filter watches the entire design-rendering surface (`apps/web/src/components/**` + `routes/**` + styles + Tailwind config + visual specs). Bisect workflow's path filter is narrower (`apps/web/src/components/parse/**` + gallery fixtures + bisect specs only). Coupling the two would force the bisect spec to run on every Visual Regression PR (~10x more PRs), wasting CI minutes.
  - **(iii) Failure-mode signal clarity.** A visual diff failure = "rendering changed unintentionally" → reviewer eyeballs PNGs. A bisect failure = "render-cycle inefficiency re-introduced" → reviewer reads console-warning stacks. Different debugging mental models; separate checks keep the signal-to-noise high.
- **Why "narrow path filter" specifically for parse hook.** bugfix-19-4b-1's root cause was a single hook (`useParseProgress.ts`) and one upstream component (`FloatingParseProgressCard.tsx:54`'s inline-object literal). The bisect spec's Phase A (multi-fixture browse) covers the gallery's harness behaviour; Phase B (per-fixture isolation) walks all 123 fixtures. The class of bug the bisect spec actually guards against — callback-prop-identity churn that becomes a setState loop — lives specifically in components that pass `{ on*: ... }` callback bundles to hook consumers. Today, that pattern lives in the parse subsystem only (per bugfix-19-4b-1's Bucket-A verdict: single offender = `FloatingParseProgressCard`). If a future PR touches another component family that exhibits the same anti-pattern, the path filter widens via a future bugfix story; do NOT pre-emptively widen here.
- **Permissions: minimum-privilege.** The bisect workflow needs only `contents: read` (checkout). It opens no PRs, commits nothing, has no Linux-baseline bootstrap analog. Scope at the job level, not workflow-wide.

### Decisions captured before dev starts

- **AC #2 [@contract-v1] stamp candidate.** Recommended: STAMP. The new `bisect` project's identity (name, testMatch, chromium-base) is consumed by `pnpm run test:bisect`, `.github/workflows/bisect-regression.yml`, and any future bugfix that wants to add a dev-mode regression spec. A silent rename or testMatch narrowing would silently lose regression coverage. Same pattern as story 19-4 [@contract-v1] AC #1 stamped the `visual` project's identity for the same downstream-protection reason.
- **Browser-agnostic skip in the bisect spec.** Keep it (defence-in-depth). The project's `use: { ...devices['Desktop Chrome'] }` already pins chromium — but if a future config edit removes that and reverts to multi-browser default, the in-spec guard ensures the spec doesn't silently start running on firefox / webkit / mobile (where the React-internal warning text may differ, leading to false negatives). Cost: 4 lines. Benefit: belt-and-braces against project-config drift.
- **No `:update` script for bisect.** The bisect spec asserts numeric counts (`multi.warnCount === 0`, `offenderCount === 0`), not snapshots. There's nothing to update. The JSON output at `_bmad-output/implementation-artifacts/bisect-bugfix-19-4b-1-dev.json` is a side-effect artefact for human review, not a baseline; `git status` should not show it as a routine change.
- **Branch-protection enablement is owner-driven.** Same half-CI / half-policy pattern as story 19-5 AC #2. The workflow makes the check appear; the owner ticks `Required` in Settings → Branches post-merge. Do not attempt to automate this from the workflow side (GitHub Actions cannot self-elevate to admin permissions for branch-protection writes).

### Project Structure Notes

- **New files:**
  - `tests/bisect/bisect-bugfix-19-4b-1.spec.ts` (moved from `tests/e2e/`; new directory created in the move).
  - `.github/workflows/bisect-regression.yml`.
- **Modified files:**
  - `playwright.config.ts` — new `bisect` project block added; `visual` project block unchanged (19-4 [@contract-v1] preserved).
  - `package.json` — new `test:bisect` script added.
  - `_bmad-output/implementation-artifacts/sprint-status.yaml` — status row updated.
  - This story file — tasks marked, Dev Agent Record populated, Change Log appended.
  - Possibly `project-context.md` Last Updated header — minimal entry (Task 4.3 explicitly defers the Rule 22 tooling line edit; do NOT touch Rule 22 prose).
- **Files NOT modified:**
  - `apps/web/src/**` — 0 frontend source edits (ESLint `local/implements-pen-node-id` trivially green).
  - `apps/api/**` — 0 Go edits.
  - `tests/visual/**` — 0 spec / fixture / baseline edits (visual project's spec + baselines untouched).
  - `tests/e2e/**` excluding the moved file — 0 edits to other E2E specs.
  - `tests/support/**` — 0 edits (global-setup tolerance fix already landed in story 19-5).
  - `.github/workflows/visual-regression.yml` — 0 edits (separate workflow per "Why not bake bisect into the visual-regression workflow" decision above).
  - `.github/workflows/test.yml` — 0 edits (the E2E shard step's `--project=chromium --project=webkit-core` list is unchanged; bisect runs in its own workflow).
  - `ux-design.pen` — untouched (CLAUDE.md screenshot workflow NOT triggered).
- **Out of scope:**
  - Widening the path filter to `apps/web/src/components/**` (see "Why narrow path filter" above).
  - Adding a `:update` script for bisect (no snapshots to update).
  - Adding bisect to `pnpm test:e2e` (regression-gate, not feature-E2E).
  - Restructuring the bisect spec's Phase A / Phase B assertions (the spec's assertion design is bugfix-19-4b-1 [@contract] — bisect-19-4b-1 closure-stamped during its own CR).
  - SHA-pinning third-party actions in the new workflow (same trade-off as story 19-5 L2 — repository-wide decision; document in workflow header but do NOT block this story on it).

### Testing standards (project-context.md)

- **No new test code in this story** — the bisect spec already exists; this story relocates it and unblocks its CI execution.
- **Validation:** (a) `pnpm run test:bisect` PASSES locally against `nx serve web`; (b) `actionlint` clean on the new workflow; (c) `pnpm test:e2e --list` reverts to 1663/36 (verifies the move successfully excluded the spec from feature-E2E projects); (d) the new workflow appears in GitHub Actions UI on push.
- **Rule 12 lint gate:** `pnpm lint:all` → 0 errors / 122 warnings baseline (unchanged — no source touched outside `playwright.config.ts` which is TS but linted as part of the broader sweep; `package.json` is JSON; YAML is covered by Prettier).
- **Rule 13 error handling:** the new workflow's `nx serve web` step MUST have fail-fast (mirror story 19-5 M2 pattern); the `find` / `bash` discipline from 19-5's bootstrap step is not relevant here (no bootstrap branch).
- **Rule 16 assertion quality:** the bisect spec's existing assertions (`expect(multi.warnCount).toBe(0)`, `expect(offenderCount).toBe(0)`) are real gates with specific failure messages — verified during bugfix-19-4b-1 CR (CR H1 finding: AC #2 regression gate "now an actual gate, not a stale-JSON dumper"). No change to assertion shape in this story.

### Rule 21 / Rule 22 / Rule 20 linkage

- **Rule 21 (Component-to-Design Node Traceability):** N/A — no `apps/web/src/components/**/*.tsx` source edits.
- **Rule 22 (Epic Retro Design-Drift Audit):** N/A — Rule 22's tooling line scopes to component visual baselines (the `visual` Playwright project + `pnpm run test:visual`). Bisect is a different regression class (render-cycle hygiene / callback identity); explicitly NOT a Rule 22 tooling extension (Task 4.3 decision).
- **Rule 20 (AC Contract Versioning):** see AC #8. Upstream consumption: 19-4 [@contract-v1] AC #1 confirmed-against (no bump). This story STAMPS [@contract-v1] on AC #2.
- **Rule 7 (Error Codes):** N/A — no Go.

### Latest tech information (already in the codebase)

- **Playwright projects** — pinned version in `package.json` devDependencies; current config supports per-project `webServer:` indirectly via the global `webServer:` array (which is what visual + bisect both bypass in CI by relying on workflow-level server startup).
- **`actions/checkout@v4`**, **`pnpm/action-setup@v4`**, **`actions/setup-node@v4`**, **`actions/upload-artifact@v4`** — same versions and patterns as story 19-5 / `test.yml`.
- **`dorny/paths-filter@v3`** — story 19-5 introduced; current stable. Same major-version pin trade-off (LOW finding per 19-5 L2; same rationale).
- **`ubuntu-24.04` runner** — same pin as story 19-5 AC #5; image bumps follow same deliberate-rebless flow.
- **No new third-party actions introduced** (no `peter-evans/create-pull-request` analog — bisect workflow opens no PRs).

### References

- [Source: `_bmad-output/implementation-artifacts/sprint-status.yaml`] — the `bugfix-19-4b-1-followup-bisect-spec-ci-coverage` backlog row this story consumes (the SM-bootstrap line carries the full reclassification rationale).
- [Source: `_bmad-output/implementation-artifacts/19-5-github-actions-visual-regression-pr.md`] — story 19-5 Senior Developer Review § "Reclassification (2026-05-19 amendment)" subsection + Change Log 2026-05-19 ✅ STATUS BUMP row.
- [Source: `_bmad-output/implementation-artifacts/bugfix-19-4b-1-gallery-max-update-depth-warnings.md`] — parent bugfix story; AC #2 ("zero warnings post-fix") regression gate the bisect spec was authored to enforce.
- [Source: `_bmad-output/implementation-artifacts/19-4-playwright-visual-snapshot-baseline.md`] — pattern reference for the `visual` Playwright project (story 19-4's [@contract-v1] AC #1).
- [Source: `tests/e2e/bisect-bugfix-19-4b-1.spec.ts`] — the spec to move and unblock; specifically L121-L135 (the two skip blocks) are the surgical-edit targets.
- [Source: `apps/web/src/routes/test/gallery.tsx:90-97`] — the `!import.meta.env.PROD` PROD-gate; the structural reason the spec needs dev-mode webServer.
- [Source: `playwright.config.ts:148-162`] — the `visual` project block; pattern reference for the new `bisect` project.
- [Source: `playwright.config.ts:167-189`] — the global `webServer:` block; reference for why CI uses workflow-level server startup instead.
- [Source: `.github/workflows/visual-regression.yml`] — workflow template; story 19-5 patterns to mirror (always-trigger + internal paths-filter; fail-fast wait-loop; concurrency split; minimum-privilege permissions; `runs-on: ubuntu-24.04`).
- [Source: `package.json:17-18`] — the `test:visual` / `test:visual:update` scripts; pattern reference for the new `test:bisect` script.
- [Source: `project-context.md` Rule 22 + Rule 20] — the rule cross-references this story consumes.

## Dev Agent Record

### Agent Model Used

_To be filled in by dev agent during /dev-story execution._

### Debug Log References

_To be filled in by dev agent._

### Completion Notes List

_To be filled in by dev agent. Required entries at close-out:_
- AC #1 verification (`git mv` clean; spec runs at new path).
- AC #2 [@contract-v1] stamp confirmation + downstream-impact restatement.
- AC #3 CI-skip removal + browser-agnostic-skip decision (keep / strip) with rationale.
- AC #4 npm script added; `test:e2e` script unchanged.
- AC #5 workflow file created; `actionlint` clean; prettier clean.
- AC #6 `pnpm test:e2e --list` count delta verified (1667/37 → 1663/36).
- AC #7 lint / visual / bisect / unit / cleanup / `ux-design.pen` all gates green.
- AC #8 Rule 20 ack line + Change Log [@contract-v0→v1] row.
- AC #9 branch-protection owner-follow-up note.

### File List

_To be filled in by dev agent._

## Change Log

| Date | Change |
| ---- | ------ |
| 2026-05-19 | 📝 SM Bob /create-story (YOLO) COMPLETE. backlog → ready-for-dev. Story file: `bugfix-19-4b-1-followup-bisect-spec-ci-coverage.md` (9 ACs — AC #2 stamp-candidate [@contract-v1]; 4 tasks; pure CI / test-infra / 0 Go / 0 frontend source / 0 new test code → single story per cross-stack split check). Consumes story 19-4 `[@contract-v1]` AC #1 unchanged (existing `visual` project untouched; new `bisect` project is additive sibling). Origin: filed by story 19-5 CR (Amelia /code-review 2026-05-19) as a reclassified "M3" finding — the bisect spec's CI skip introduced in 19-5 commit `e70f84a` was an inline "Pre-existing fix #2 (Epic 9c Retro AI-2 inline option)" patching over a structural mismatch between the spec's design (assumes dev-mode `/test/gallery` reachable) and the prod-build CI shard (PROD-gate from story 19-4 CR M1 makes `/test/gallery` return "Access Denied"). Root-cause fix = new Playwright project with dev-mode webServer, which structurally requires `playwright.config.ts` edits — that's a 19-4 [@contract-v1] surface widening, outside 19-5's CI-subsystem scope. Inheriting back to bugfix-19-4b-1 (the spec's author) is the correct decoupling. Key SM decisions: (a) **New project, new directory** (`tests/bisect/`) — mirrors `tests/visual/` pattern; cleaner than per-spec testMatch surgery on the 4 e2e projects. (b) **Standalone workflow** (`.github/workflows/bisect-regression.yml`) — NOT bundled into visual-regression workflow (wall-clock + trigger-path + signal-clarity reasons in Dev Notes); separate concurrency group; narrow path filter on `apps/web/src/components/parse/**` + gallery fixture stack + bisect specs only. (c) **AC #2 [@contract-v1] stamp recommended** — `bisect` project identity is consumed by `test:bisect` script + bisect-regression workflow + future bugfix dev-mode regression specs; silent rename = silent regression-coverage loss (same failure class as story 19-5 H2 path-filter dead-pattern). (d) **In-spec browser-agnostic skip kept as defence-in-depth** — belt-and-braces against project-config drift; 4 lines, negligible cost. (e) **No Rule 22 tooling-line extension** — Rule 22 scopes to design-drift visual baselines; bisect is a different regression class (render-cycle hygiene). (f) **Operational follow-up**: branch-protection toggle for `Bisect Regression / PR` is the owner's post-merge web-UI click (same half-CI / half-policy pattern as story 19-5 AC #2). Depends on bugfix-19-4b-1 (done) + story 19-5 (done) + story 19-4 (done). 🔗 AC Drift: NEW (additive — 19-4 [@contract-v1] consumed unchanged; project-list widening). 📎 Contract Stamps: NEW (v1×1 candidate on AC #2). 🔒 Rule 7: N/A. 🎨 UX: N/A — `ux-design.pen` untouched. → DEV Amelia /dev-story next (different LLM context per workflow tip; CR Amelia /code-review after with a third). |
