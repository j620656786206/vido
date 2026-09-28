---
name: ship
description: Full auto-to-merge delivery pipeline for vido — adversarial review → new branch → conventional commit → PR → CI self-heal → merge. Use when the user says "ship", "ship it", "/ship", or asks to take current changes all the way to a merged PR. Pauses only for genuine product/architecture decisions.
---

# /ship — vido auto-to-merge pipeline

Take the current working-tree changes (or a just-finished BMAD dev story) **all the way to a merged PR without stopping**, except to ask the user when a real product or architecture decision arises.

This skill is vido-specific: it bakes in vido's gh account, nx/pnpm tooling, conventional-commit style, and the `-linux` visual-baseline CI quirk. Do not generalize it to other repos.

## Hard rules (never violate)

- **Never commit to `main`.** Always create a NEW feature branch first, based off `main` (not off another feature/skill branch).
- **Never use git worktrees.** Use a direct `git checkout -b`.
- **Every `gh` call runs as `j620656786206`, pinned per command — never by switching.** Start each shell that calls `gh` with `export GH_TOKEN=$(gh auth token --user j620656786206)`. Do **not** rely on `gh auth switch`: the active account is machine-global, another session (work repos use `alexyu-tvbs`) flips it mid-pipeline, and a `gh pr create` then fails as a non-collaborator. (retro-dsr-AI2, carries retro-m2-AI2.)
- **`-linux` visual baselines CANNOT be generated locally** (this machine is darwin; CI runs ubuntu). Never run `test:visual:update` and commit the resulting `-*.png` to fix a visual-regression failure — those would be wrong-platform PNGs. See "CI self-heal → Visual Regression" below.
- Stay autonomous. Only pause for a genuine product/architecture decision — not for routine lint fixes, baseline bootstraps, or gh account switches.

## Network: retry, don't stop

github.com is sometimes unreachable from this machine for minutes at a time (`ssh: Could not resolve hostname github.com`, `error connecting to api.github.com`) while the rest of the internet works. That is not a failure of the work — retry:

- **push / fetch / pull**: loop up to ~20 times with a 20s pause, e.g. `for i in $(seq 1 20); do git push -q -u origin "$B" && break; sleep 20; done`. After a pull, check the expected commit actually arrived (`git log --oneline -1 | grep -q "#NNN"`) instead of trusting the exit code.
- **`gh pr create`**: write the body to a file first (`--body-file`), then loop; on each miss also try `gh pr list --head "$B" --json url` — the create may have succeeded before the connection dropped. Never create the PR twice.
- **Watching CI and merging** can run as one background command (create → `gh pr checks --watch` → merge only if every check is `pass`/`skipping`); you are notified when it ends.
- Only report "blocked" if github.com stays unreachable for the whole retry window, and say which step (push, PR, checks, merge) is still pending.

## Pipeline

### 1. Adversarial self-review
- Review the current diff adversarially (use the BMAD adversarial review task at `_bmad/core/tasks/review-adversarial-general.xml`, or `/code-review high` if no BMAD story context).
- Fix all **in-scope** issues, each with a test. Out-of-scope findings → note them in the PR body, don't fix.

### 2. New branch off main
- Determine scope/ticket from the work (BMAD story id, `pg-XXXXX`, `retro-NN`, etc.).
- `git checkout main && git pull`, then `git checkout -b <type>/<scope>-<slug>` matching existing naming (`feat/pg-13453-...`, `retro-11-ai1-...`, `docs(11)`-style scopes).
- If changes are already on `main`, move them onto the new branch (do NOT commit them to main).

### 3. Verify locally before commit
- `pnpm run lint:all` (nx run-many lint + root lint + format:check). Auto-fix with `pnpm run lint:fix && pnpm run format` if it fails, then re-run.
- Run the relevant tests: `pnpm run test:ci` for the CI-tagged suite, or the story-specific grep (`pnpm run test:e2e -- --grep @story-N-M`). For new/risky E2E, run burn-in: `pnpm run test:burn-in`.
- If `ux-design.pen` changed, follow the UX screenshots workflow in `CLAUDE.md` (regen via `scripts/export-pen-screenshots.py`, stage only design-changed PNGs) before committing.

### 4. Conventional commit
- One or more conventional commits: `<type>(<scope>): <summary>` — match the history style (`feat(retro-11): ...`, `fix(media-detail): ...`, `docs(8-11): ...`, `chore(visual): ...`).
- Husky pre-commit hooks will run; let them. If they reject, fix and retry.

### 5. Push + open PR
- `git push -u origin <branch>` (with the retry loop above).
- `gh pr create --body-file …` (retry loop above; `GH_TOKEN` pinned) with a title mirroring the commit and a body containing: what changed, test evidence (which suites ran green), and any out-of-scope review findings. End the body with the Claude Code attribution.

### 6. CI self-heal loop
Poll CI with `gh pr checks --watch`. Fix failures autonomously:

- **Lint / format** → `pnpm run lint:fix && pnpm run format`, commit `chore: lint`, push.
- **Unit / E2E regression** → diagnose, fix in-scope with a test, push. If a test is genuinely flaky, confirm via burn-in before touching it.
- **Visual Regression (`-linux` baselines)** → `-linux` PNGs are produced by CI only (retro-dsr-AI5, carries retro-m25-AI2). The flow that works, used on every PR from dsr-2 on:
  1. **Before pushing**, for every fixture whose look changed: update the `-darwin` PNG locally and **`git rm` the stale `-linux` PNG**. A change below the snapshot threshold is **not** rewritten by `test:visual:update` — if the picture's content changed (e.g. a sentence), delete the `-darwin` file too and run `test:visual:update-missing` so the baseline shows the new content.
  2. After the PR is open, dispatch the bootstrap on the **feature branch**: `gh workflow run "Visual Regression" --ref <branch>`. It opens a `chore(visual): bootstrap N missing -linux baselines (incremental)` PR **targeting your branch**.
  3. Verify that PR only adds `-linux.png` files (plus the auto-updated `_bmad-output/audit/visual-baseline-19-4.md`), then `gh pr merge <N> --squash --delete-branch`. This lands the baselines on your branch; the PR's own checks then re-run green. No `--admin` and no rebase onto main needed.
  - If `Visual Regression / PR` shows a **real diff** (not just missing baselines), that's a genuine change → pause and show the diff artifact to the user.
- **Docker / other** → diagnose from logs; fix if in-scope.

Repeat until all checks are green.

### 7. Merge
- When all checks are green, address any bot/human review comments inline first.
- Merge (`gh pr merge --squash --delete-branch` to match the squash-with-`(#NN)` history style).
- Report a final summary: merged PR link, what shipped, suites that passed, and anything deferred.

## When to pause and ask
- A real product decision (scope, UX behavior, what a feature should do).
- A real architecture decision (new dependency, data-model change, cross-cutting refactor).
- A visual-regression **real diff** (design actually changed) — confirm intent before baselining.
- Anything ambiguous about the user's intent. Routine mechanics (lint, baselines, gh switch, flaky-retry) are NOT pause-worthy.
