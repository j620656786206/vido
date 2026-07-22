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
- **gh account must be `j620656786206`** for all PR/CI operations. Verify with `gh auth status`; if the active account is `alexyu-tvbs`, run `gh auth switch --user j620656786206` before any `gh` PR/CI call. Re-verify if a `gh` call fails with a permissions error.
- **`-linux` visual baselines CANNOT be generated locally** (this machine is darwin; CI runs ubuntu). Never run `test:visual:update` and commit the resulting `-*.png` to fix a visual-regression failure — those would be wrong-platform PNGs. See "CI self-heal → Visual Regression" below.
- Stay autonomous. Only pause for a genuine product/architecture decision — not for routine lint fixes, baseline bootstraps, or gh account switches.

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
- If `ux-design.pen` changed, follow the UX screenshots workflow in `AGENTS.md` (regen via `scripts/export-pen-screenshots.py`, stage only design-changed PNGs) before committing.

### 4. Conventional commit

- One or more conventional commits: `<type>(<scope>): <summary>` — match the history style (`feat(retro-11): ...`, `fix(media-detail): ...`, `docs(8-11): ...`, `chore(visual): ...`).
- Husky pre-commit hooks will run; let them. If they reject, fix and retry.

### 5. Push + open PR

- `git push -u origin <branch>`.
- `gh pr create` with a title mirroring the commit and a body containing: what changed, test evidence (which suites ran green), and any out-of-scope review findings. End the body with the Codex attribution.

### 6. CI self-heal loop

Poll CI with `gh pr checks --watch`. Fix failures autonomously:

- **Lint / format** → `pnpm run lint:fix && pnpm run format`, commit `chore: lint`, push.
- **Unit / E2E regression** → diagnose, fix in-scope with a test, push. If a test is genuinely flaky, confirm via burn-in before touching it.
- **Visual Regression (`-linux` baselines)** → This is the known vido quirk. When the `Visual Regression / PR` check fails **purely because `-linux.png` baselines are missing** (no real visual diff), the dedicated `Visual Regression` workflow auto-opens a separate `chore(visual): bootstrap N missing -linux baselines (incremental)` PR. Do NOT regenerate baselines locally. Instead:
  1. Find that bootstrap PR (`gh pr list --search "bootstrap linux baselines"`).
  2. Verify it only adds `-linux.png` files (no source changes), then merge it.
  3. Rebase the feature branch on updated `main` and re-run CI.
  - If the visual check shows a **real diff** (not just missing baselines), that's a genuine change → pause and show the diff artifact to the user.
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
