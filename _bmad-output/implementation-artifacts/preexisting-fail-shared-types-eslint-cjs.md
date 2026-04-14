# Story: Fix libs/shared-types ESLint Config — Port Legacy CJS to Flat Config

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a developer,
I want `libs/shared-types/eslint.config.cjs` replaced with a working flat-config `eslint.config.mjs` that extends the root config,
so that `nx run shared-types:lint` succeeds without requiring a non-existent legacy `.eslintrc.json` and the `@nx/dependency-checks` rule actually runs to validate the published library's `package.json`.

## Acceptance Criteria

1. Given `libs/shared-types/eslint.config.cjs` is the legacy broken config, when the fix is applied, then this file no longer exists (replaced, not kept alongside).
2. Given the replacement `libs/shared-types/eslint.config.mjs`, when ESLint loads it, then it imports the root `eslint.config.mjs` flat config (no reference to any legacy `.eslintrc.json`) and adds the library-specific `@nx/dependency-checks` override for `**/*.json` files.
3. Given the new config, when `pnpm nx run shared-types:lint` runs, then it exits with code 0 (no `Cannot find module '../../.eslintrc.json'` error, no rule failures).
4. Given the new config, when `pnpm nx run shared-types:lint` is run against a deliberately-bad `package.json` (temporarily add an undeclared import to `src/index.ts`), then `@nx/dependency-checks` flags the missing dep — proving the rule is actually live. Revert the injection before committing.
5. Given the fix, when `pnpm run lint` (root `eslint .`) runs, then `libs/shared-types/src/**/*.ts` is still covered by the root flat config (no regression in root lint coverage).
6. Given the fix, when `pnpm run lint:all` runs (the convenience command from retro-9-AI4 — `pnpm nx run api:lint && pnpm run lint && pnpm run format:check`), then it exits with code 0.
7. Given the fix, when `pnpm nx run shared-types:build` runs, then build still succeeds (fix must not break build).
8. Given the fix, when the full affected lint run `pnpm nx affected -t lint` is run after the change, then no new failures are introduced on other projects.

## Tasks / Subtasks

- [x] Task 1: Remove legacy broken CJS config (AC: #1)
  - [x] 1.1 Delete `libs/shared-types/eslint.config.cjs`

- [x] Task 2: Create new flat-config `eslint.config.mjs` (AC: #2)
  - [x] 2.1 Create `libs/shared-types/eslint.config.mjs` with the following contents:
    ```js
    import rootConfig from '../../eslint.config.mjs';
    import jsoncParser from 'jsonc-eslint-parser';

    export default [
      ...rootConfig,
      {
        files: ['**/*.json'],
        languageOptions: {
          parser: jsoncParser,
        },
        rules: {
          '@nx/dependency-checks': [
            'error',
            {
              ignoredFiles: ['{projectRoot}/eslint.config.{js,cjs,mjs,ts,cts,mts}'],
            },
          ],
        },
      },
    ];
    ```
  - [x] 2.2 Verify `jsonc-eslint-parser` resolves — it resolves via pnpm hoist from root `node_modules/jsonc-eslint-parser/` (no per-lib copy under `libs/shared-types/node_modules/`; pnpm's hoisted layout is sufficient for ESLint's module resolution from the library cwd). If resolution fails, run `pnpm add -D -w jsonc-eslint-parser` (workspace root) rather than adding a per-lib dep.
  - [x] 2.3 Verify `@nx/eslint-plugin` is installed at the root (it provides the `@nx/dependency-checks` rule). It is — no action needed unless the rule reports unknown-plugin. **ACTION TAKEN:** Rule reported unknown-plugin (story anticipated this contingency). Installed `@nx/eslint-plugin@22.3.3` via `pnpm add -D -w` and registered it in the config's `plugins` block (keyed as `'@nx'`).

- [x] Task 3: Verify the inferred Nx lint target still resolves (AC: #3)
  - [x] 3.1 Run `pnpm nx show project shared-types --json | grep -o '"lint"'` — confirm `@nx/eslint/plugin` still infers a `lint` target because `eslint.config.mjs` is present.
  - [x] 3.2 Run `pnpm nx run shared-types:lint` — expect exit 0.
  - [x] 3.3 If the inputs list in the inferred target still mentions the old `.cjs` path, that is harmless (stale cache). Clear with `pnpm nx reset` if needed. Ran `pnpm nx reset` once after adding the plugin dep.

- [x] Task 4: Fault-injection test — prove `@nx/dependency-checks` is live (AC: #4)
  - [x] 4.1 In `libs/shared-types/src/index.ts` (or any `.ts` file that is part of the library barrel), temporarily add `import 'lodash';` (a package NOT listed in `libs/shared-types/package.json` dependencies).
  - [x] 4.2 Run `pnpm nx run shared-types:lint` — it MUST fail with a `@nx/dependency-checks` error naming `lodash`. **RESULT:** `error: The "shared-types" project uses the following packages, but they are missing from "dependencies": - lodash  @nx/dependency-checks` — rule is live.
  - [x] 4.3 Revert the injected import. Run lint again — it MUST pass. Confirmed pass.
  - [x] 4.4 Do NOT commit the injected import. This step only verifies the rule runs. `src/index.ts` reverted to original contents (single `export * from './lib/shared-types';`).

- [x] Task 5: Root-lint and lint:all regression check (AC: #5, #6)
  - [x] 5.1 Run `pnpm run lint` from repo root — expect exit 0. **RESULT:** 0 errors, 108 warnings, exit 0.
  - [x] 5.2 Run `pnpm run lint:all` from repo root — expect exit 0. **RESULT:** exit 0 (api:lint + root lint + format:check all pass).
  - [x] 5.3 If root lint now flags `libs/shared-types/eslint.config.mjs` itself, extend the root `eslint.config.mjs` ignores to include `**/eslint.config.{js,cjs,mjs,ts,cts,mts}` (root already ignores `*.config.js` / `*.config.ts` but NOT `.mjs` — verify against current root config before editing). Prefer NOT modifying root ignores unless a concrete error appears. **NO ACTION NEEDED** — no flag on the new `.mjs` file; root ignores untouched.

- [x] Task 6: Build + broader affected regression (AC: #7, #8)
  - [x] 6.1 Run `pnpm nx run shared-types:build` — expect exit 0. **PASS**.
  - [x] 6.2 Run `pnpm nx affected -t lint` — expect no new failures. **PASS** (3 projects: api + shared-types + web; 0 errors, 107 warnings = same as pre-change baseline).
  - [x] 6.3 Run `pnpm nx affected -t test` if shared-types is a dep of any testable project — expect no new failures. **PASS** (2 projects — `api:test` + `web:test`; `shared-types` has no `test` target, only `lint`/`build`/`nx-release-publish`, so it is not picked up by `affected -t test`).

- [x] Task 7: Update sprint-status.yaml (done by SM create-story — devs do NOT re-edit)
  - [x] 7.1 `preexisting-fail-shared-types-eslint-cjs` updated from `backlog` → `ready-for-dev` (SM workflow step 6).

## Dev Notes

### Root Cause

`libs/shared-types/eslint.config.cjs` is a **doubly-broken** legacy transitional file:

1. **Missing dependency.** Line 1: `const baseConfig = require('../../.eslintrc.json');` — but `.eslintrc.json` does not exist at the repo root (and never did after the flat-config migration). The repo root uses `eslint.config.mjs` (ESLint flat config, post-`eslint@9`).
2. **Incompatible format even if the file existed.** The code does `module.exports = [ ...baseConfig, ... ]` — spreading the loaded JSON into a flat-config array. Legacy `.eslintrc.json` is a single object with nested keys (`extends`, `rules`, `overrides`, …), not an array. Spreading an object into an array produces `[ "extends", "rules", ... ]` (string keys) — invalid flat-config entries. So even re-creating `.eslintrc.json` would not fix this file.

### Why It Is Masked

- Root `pnpm run lint` runs `eslint .` from the workspace root. ESLint's flat-config resolution finds the root `eslint.config.mjs` before walking into `libs/shared-types/`, so it never loads the broken cjs file.
- `libs/shared-types/eslint.config.cjs` is only loaded when ESLint is invoked with a cwd inside `libs/shared-types/` — which is exactly what `@nx/eslint/plugin`'s inferred `lint` target does (`eslint .` with `cwd: "libs/shared-types"`).
- retro-9-AI4 introduced `pnpm run lint:all` which runs root `pnpm run lint` — still not invoking the per-project target, so this failure stayed hidden even after retro-9-AI4 closed.
- The failure surfaces only when someone runs `pnpm nx run shared-types:lint` or `pnpm nx run-many -t lint` — which is what retro-9-AI4 triage found.

### Why Flat Config Instead of Delete

The sprint-status entry offered two fix paths: *"delete cjs or port to flat config."* Porting is preferred for this project because:

- **`shared-types` is tagged `npm:public`** (visible via `pnpm nx show project shared-types --json`) and has an `nx-release-publish` target configured with `manifestRootsToUpdate: ["dist/{projectRoot}"]`. It is set up as a published npm library.
- **`@nx/dependency-checks` is a library-specific rule** that validates the published `package.json` declares every dependency actually imported from source. Losing it on a published library means drift between runtime imports and declared deps could ship to consumers.
- Port effort is minimal (~10 LOC); a plain delete would also require acknowledging the lost check and documenting it, which is comparable work with worse downstream risk.

### Why the `.mjs` Extension (not `.cjs`)

- `libs/shared-types/package.json` sets `"type": "commonjs"`. ESLint's flat-config resolver treats `.mjs` as ESM regardless of the package `type`, so `.mjs` is the safest choice when we want to `import` the root flat config (which is itself ESM — root is named `eslint.config.mjs`).
- A `.cjs` replacement could work via `require()` of the root config, but the root config is ESM; `require('../../eslint.config.mjs')` from a CJS file would fail under Node's ESM rules without a dynamic `import()`. ESM-to-ESM is simpler and matches the prevailing flat-config convention used elsewhere in the Nx docs.
- Nx's eslint plugin auto-infers a `lint` target for any of `eslint.config.{js,cjs,mjs,ts,cts,mts}` — `.mjs` is fully supported.

### What `@nx/dependency-checks` Validates

For a library project with an `eslint.config.*`, Nx's dependency-checks rule lints `package.json` to ensure:

- Every runtime import in `src/**` that resolves to an external package has a matching entry in the library's `package.json` `dependencies` (or `peerDependencies`).
- The version ranges declared in the library are compatible with the root workspace lock.
- No stale/unused deps remain in the library's `package.json`.

Today this is effectively off for `shared-types` (the config fails to load). After the fix, it becomes active — which is the intended state for a published library.

### Scope Boundaries — What NOT to Do

- **DO NOT** delete `libs/shared-types/eslint.config.cjs` without creating the `.mjs` replacement. Deleting alone would remove both the broken file AND the dependency-checks rule for a published library.
- **DO NOT** create a new root `.eslintrc.json` to satisfy the old require. The repo is fully migrated to flat config; re-adding legacy config would fight the modern tooling.
- **DO NOT** modify root `eslint.config.mjs` unless Task 5.3 actually reports a concrete failure lint-ing the new `.mjs` file. Root config already ignores `*.config.js`/`*.config.ts` — check behavior before adding more ignores.
- **DO NOT** add `jsonc-eslint-parser` as a per-library dependency. If the module doesn't resolve, add it at the workspace root (`-w`). Per-library devDeps are discouraged in this pnpm workspace.
- **DO NOT** modify `libs/shared-types/project.json`. The `lint` target is *inferred* by `@nx/eslint/plugin` from `nx.json` — adding a manual `lint` target would duplicate/shadow it and diverge from convention (`apps/web` and `apps/api` also rely on plugin inference, with one exception for `api:lint` which is a Go-specific custom target).
- **DO NOT** touch the sprint-status.yaml status line yourself — the SM workflow already flipped it to `ready-for-dev`.
- **DO NOT** change `libs/shared-types/package.json` `type` from `commonjs` to `module`. That would break `@nx/js:tsc` build output format (CJS) that downstream consumers may depend on.

### Scope of Source Coverage

- Root `eslint.config.mjs` globs `**/*.ts` and `**/*.tsx` with no `libs/**` ignore — so `libs/shared-types/src/**/*.ts` IS linted by the root config via `pnpm run lint`. The per-project target is additive (it enables `@nx/dependency-checks` for `package.json`), not required for source coverage.
- After the fix, shared-types TS files get linted **twice** in CI-style runs (once by root, once by per-project). This is a known tradeoff — the per-project run layers in the `@nx/dependency-checks` rule; overlap is harmless because results concur.

### Expected File Contents (Reference)

`libs/shared-types/eslint.config.mjs` (new, ~15 LOC):

```js
import rootConfig from '../../eslint.config.mjs';
import jsoncParser from 'jsonc-eslint-parser';

export default [
  ...rootConfig,
  {
    files: ['**/*.json'],
    languageOptions: {
      parser: jsoncParser,
    },
    rules: {
      '@nx/dependency-checks': [
        'error',
        {
          ignoredFiles: ['{projectRoot}/eslint.config.{js,cjs,mjs,ts,cts,mts}'],
        },
      ],
    },
  },
];
```

### Testing Standards Summary

- No unit tests for a config file; verification is operational (the lint target itself is the test).
- **Fault injection is mandatory (Task 4)** — a green lint run alone doesn't prove the added rule is firing. The temporary `import 'lodash';` in `src/index.ts` forces `@nx/dependency-checks` to complain; seeing that failure then disappear after revert is the only way to confirm the rule is live.
- Follow the retro-9-AI4 pattern: run `pnpm run lint:all` before committing. CI ran `pnpm run lint` sharded, so local `lint:all` parity is the green-light signal.

### Project Structure Notes

- Aligned with unified project structure: per-library ESLint config lives at library root as `eslint.config.{mjs,cjs,ts,...}`. No conflict with nx.json plugin inference.
- No source code in `libs/shared-types/src/**` changes for this story.
- `libs/shared-types/package.json` unchanged.
- `libs/shared-types/project.json` unchanged (lint target stays plugin-inferred).

### References

- [Source: libs/shared-types/eslint.config.cjs:1] — `const baseConfig = require('../../.eslintrc.json');` (broken require — this file is deleted in Task 1)
- [Source: eslint.config.mjs:1-184] — Root flat config (the thing the new `.mjs` imports)
- [Source: nx.json:18-23] — `@nx/eslint/plugin` registration with `targetName: "lint"` — this is what auto-infers `shared-types:lint`
- [Source: libs/shared-types/project.json:1-31] — No manual `lint` target; relies on plugin inference
- [Source: libs/shared-types/package.json:1-10] — `"type": "commonjs"` (unchanged)
- [Source: package.json:7-10] — `"lint"`, `"lint:fix"`, `"lint:all"` scripts (the retro-9-AI4 convenience commands)
- [Source: _bmad-output/implementation-artifacts/sprint-status.yaml:371-373] — retro-9-AI4 closure note + this story's backlog entry
- [Source: _bmad-output/implementation-artifacts/retro-9-AI4-local-lint-command.md] — Context on why this failure was surfaced (the retro that added the local lint command and triaged per-project lint failures)

## Dev Agent Record

### Agent Model Used

- SM story creation: Claude Opus 4.6 (1M context), YOLO mode
- DEV implementation: Claude Opus 4.6 (1M context) — Amelia /dev-story, 2026-04-14

### Debug Log References

- Baseline failure reproduced: `pnpm nx run shared-types:lint` → `Cannot find module '../../.eslintrc.json'`
- Post-fix first run: `@nx/dependency-checks` rule reports plugin `@nx` not found → added plugin import + registration
- Fault-injection (`import 'lodash'` into `src/index.ts`): rule correctly flagged `lodash` missing from `package.json` → proves rule is live
- Post-revert: lint exits 0 (cached after Nx detected file reverted to prior content)

### Completion Notes List

- All 6 tasks + subtasks complete; all 8 ACs satisfied.
- **Pre-existing fix (inline)**: Created `$(go env GOPATH)/bin` directory (`/Users/tvbs/go/bin`) so `api:lint`'s staticcheck auto-install step (added in retro-9-AI3) can complete its `mv` from `$STATICCHECK_TMP`. Without this, `api:lint` fails on any machine that has never had a `go install` run. Unrelated to the shared-types ESLint change; surfaced because `lint:all` (required by AC #6) runs `api:lint` as its first step. One-time machine-local setup, no repo files modified for this fix.
- **Dependency added**: `@nx/eslint-plugin@22.3.3` (workspace devDep). Story's Dev Notes anticipated this contingency ("no action needed unless the rule reports unknown-plugin"). Version pinned to match `@nx/js` / `nx` / `@nx/web` at `22.3.3`.
- **Flat-config tweak vs story template**: Story provided a 15-LOC template that omits explicit plugin registration. Added `plugins: { '@nx': nxEslintPlugin }` inside the JSON override block — this is a hard requirement of ESLint flat config (no implicit plugin resolution from rule names, unlike the legacy `.eslintrc` world). Net effect: ~18 LOC, still trivial.
- Build output format unchanged (CJS, per `"type": "commonjs"`).
- No source-file changes to `src/**` (index.ts reverted verbatim after fault injection).

### UX Verification

🎨 UX Verification: SKIPPED — no UI changes in this story (build-tool config only).

### Change Log

| Date       | Change                                                                                                                                                                                                                                                                                                                                                  |
|------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| 2026-04-14 | Replaced `libs/shared-types/eslint.config.cjs` (broken legacy CJS require of non-existent `.eslintrc.json`) with `libs/shared-types/eslint.config.mjs` (flat config extending root + `@nx/dependency-checks`).                                                                                                                                           |
| 2026-04-14 | Added `@nx/eslint-plugin@22.3.3` to root `devDependencies` (pnpm-lock.yaml regenerated). Needed to satisfy `@nx/dependency-checks` rule; story Dev Notes pre-authorized this contingency ("unless the rule reports unknown-plugin"). Plugin registered via `plugins: { '@nx': nxEslintPlugin }` in the new `.mjs` config.                                  |
| 2026-04-14 | Fault-injection test confirmed `@nx/dependency-checks` is live: temporary `import 'lodash'` in `src/index.ts` → rule errors on missing dep → import reverted → rule green. Proves the published-lib safety net is now active (previously silently off because the broken CJS never loaded).                                                              |
| 2026-04-14 | **CR fix (Amelia /code-review, M1):** Extended `lint:all` in `package.json` — replaced `pnpm nx run api:lint && pnpm run lint && pnpm run format:check` with `pnpm nx run-many -t lint && pnpm run lint && pnpm run format:check`. Reason: the pre-fix `lint:all` never invoked `shared-types:lint`, so the `@nx/dependency-checks` safety net this story just resurrected would NOT be exercised by CI-parity gate — silent-regression risk identical to the one that produced this story. Verified via fault-injection: `import 'lodash'` in `src/index.ts` → `lint:all` exits 1 with `@nx/dependency-checks` error → reverted → `lint:all` exits 0. |
| 2026-04-14 | **CR fix (M2/L1):** Corrected two narrative inaccuracies in this story file — Task 6.3's "api Go tests + shared-types" claim (shared-types has no `test` target; actual run hits `api:test` + `web:test`), and Task 2.2's "already available under `libs/shared-types/node_modules/`" claim (pnpm hoisted, only at root). No functional change.                  |

### File List

- `libs/shared-types/eslint.config.cjs` — **DELETED** (broken legacy transitional file)
- `libs/shared-types/eslint.config.mjs` — **CREATED** (new flat config extending root + `@nx/dependency-checks` for `package.json`, with explicit `@nx` plugin registration)
- `package.json` — **MODIFIED** — two changes: (1) added `@nx/eslint-plugin@22.3.3` to `devDependencies` (dev), and (2) **CR M1:** widened `lint:all` script from `pnpm nx run api:lint && …` to `pnpm nx run-many -t lint && …` so `shared-types:lint` (and any future per-project lint targets) participate in the CI-parity regression gate.
- `pnpm-lock.yaml` — **MODIFIED** (regenerated by `pnpm add -D -w @nx/eslint-plugin@22.3.3`)

## Senior Developer Review (AI)

**Reviewer:** Amelia (Claude Opus 4.6 /code-review) on 2026-04-14
**Outcome:** **APPROVED WITH FIXES APPLIED**

**AC verification (independent, re-ran from clean cache where possible):**

| AC | Check | Result |
|----|-------|--------|
| #1 | `libs/shared-types/eslint.config.cjs` absent | ✅ (`ls` confirms only `.mjs`) |
| #2 | `.mjs` imports root + adds `@nx/dependency-checks` | ✅ (file inspected) |
| #3 | `nx run shared-types:lint` exits 0 | ✅ (re-run, cache hit) |
| #4 | Fault injection (`import 'lodash'`) fires the rule | ✅ (reviewer independently injected + reverted) |
| #5 | `pnpm run lint` exits 0 | ✅ (0 errors, 108 warnings) |
| #6 | `pnpm run lint:all` exits 0 | ✅ (green under both original and CR-widened command) |
| #7 | `nx run shared-types:build` exits 0 | ✅ |
| #8 | `nx affected -t lint` and `-t test` clean | ✅ (`api:lint` + `shared-types:lint` + `web:lint` green; `api:test` + `web:test` green) |

**Findings & fixes applied (via option [1] auto-fix):**

- **M1 (regression-gate gap) — FIXED**: `lint:all` widened to `nx run-many -t lint` so `@nx/dependency-checks` is exercised on every CI-parity run, not just explicit `nx run shared-types:lint`. Fault-injection confirmed the widened command now fails on undeclared imports. Net risk eliminated: silent drift between a published lib's runtime imports and its declared deps can no longer slip past `lint:all`.
- **M2 (Task 6.3 narrative) — FIXED**: Corrected the claim that `affected -t test` hits "api + shared-types" — actual targets are `api:test` + `web:test` (shared-types has no test target).
- **L1 (Task 2.2 wording) — FIXED**: Replaced the false "already available under `libs/shared-types/node_modules/`" with the accurate pnpm-hoist explanation.
- **L2 (Change Log scope bleed) — FIXED**: The `$GOPATH/bin` machine-local fix was removed from the Change Log table (retained in Completion Notes, where it belongs — it's environmental, not a repo change).

**Net new repo change from CR:** `package.json` (single-line `lint:all` script edit); zero code changes in `libs/shared-types/`.

**Status transition:** `review` → `done` (all ACs satisfied, all CR findings addressed).
