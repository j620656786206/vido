# Story: Fix libs/shared-types ESLint Config — Port Legacy CJS to Flat Config

Status: ready-for-dev

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

- [ ] Task 1: Remove legacy broken CJS config (AC: #1)
  - [ ] 1.1 Delete `libs/shared-types/eslint.config.cjs`

- [ ] Task 2: Create new flat-config `eslint.config.mjs` (AC: #2)
  - [ ] 2.1 Create `libs/shared-types/eslint.config.mjs` with the following contents:
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
  - [ ] 2.2 Verify `jsonc-eslint-parser` resolves — it is already available under `libs/shared-types/node_modules/` and also in the root workspace. If resolution fails, run `pnpm add -D -w jsonc-eslint-parser` (workspace root) rather than adding a per-lib dep.
  - [ ] 2.3 Verify `@nx/eslint-plugin` is installed at the root (it provides the `@nx/dependency-checks` rule). It is — no action needed unless the rule reports unknown-plugin.

- [ ] Task 3: Verify the inferred Nx lint target still resolves (AC: #3)
  - [ ] 3.1 Run `pnpm nx show project shared-types --json | grep -o '"lint"'` — confirm `@nx/eslint/plugin` still infers a `lint` target because `eslint.config.mjs` is present.
  - [ ] 3.2 Run `pnpm nx run shared-types:lint` — expect exit 0.
  - [ ] 3.3 If the inputs list in the inferred target still mentions the old `.cjs` path, that is harmless (stale cache). Clear with `pnpm nx reset` if needed.

- [ ] Task 4: Fault-injection test — prove `@nx/dependency-checks` is live (AC: #4)
  - [ ] 4.1 In `libs/shared-types/src/index.ts` (or any `.ts` file that is part of the library barrel), temporarily add `import 'lodash';` (a package NOT listed in `libs/shared-types/package.json` dependencies).
  - [ ] 4.2 Run `pnpm nx run shared-types:lint` — it MUST fail with a `@nx/dependency-checks` error naming `lodash`.
  - [ ] 4.3 Revert the injected import. Run lint again — it MUST pass.
  - [ ] 4.4 Do NOT commit the injected import. This step only verifies the rule runs.

- [ ] Task 5: Root-lint and lint:all regression check (AC: #5, #6)
  - [ ] 5.1 Run `pnpm run lint` from repo root — expect exit 0.
  - [ ] 5.2 Run `pnpm run lint:all` from repo root — expect exit 0.
  - [ ] 5.3 If root lint now flags `libs/shared-types/eslint.config.mjs` itself, extend the root `eslint.config.mjs` ignores to include `**/eslint.config.{js,cjs,mjs,ts,cts,mts}` (root already ignores `*.config.js` / `*.config.ts` but NOT `.mjs` — verify against current root config before editing). Prefer NOT modifying root ignores unless a concrete error appears.

- [ ] Task 6: Build + broader affected regression (AC: #7, #8)
  - [ ] 6.1 Run `pnpm nx run shared-types:build` — expect exit 0.
  - [ ] 6.2 Run `pnpm nx affected -t lint` — expect no new failures (compare to pre-change baseline from `main`).
  - [ ] 6.3 Run `pnpm nx affected -t test` if shared-types is a dep of any testable project — expect no new failures.

- [ ] Task 7: Update sprint-status.yaml (done by SM create-story — devs do NOT re-edit)
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

Claude Opus 4.6 (1M context) — SM agent (Bob) create-story workflow, YOLO mode

### Debug Log References

### Completion Notes List

### Change Log

### File List

- `libs/shared-types/eslint.config.cjs` — DELETE (broken legacy transitional file)
- `libs/shared-types/eslint.config.mjs` — CREATE (new flat config extending root + `@nx/dependency-checks` for `package.json`)
