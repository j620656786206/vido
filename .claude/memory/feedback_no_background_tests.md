---
name: No background tests + mandatory cleanup verification
description: Never run test suites in background, and always verify no orphaned processes remain after every test execution
type: feedback
---

1. Never use `run_in_background` for test suite execution (vitest, nx test, playwright).
2. After EVERY test execution, verify no orphaned processes remain by running `pnpm run test:cleanup`.
3. If orphaned processes are found, kill them immediately with `pnpm run test:cleanup:all`.
4. Test execution is NOT complete until cleanup verification passes.

**Why:** Vitest fork workers can become orphaned even in foreground mode. This happened repeatedly on 2026-03-15 — documented rules existed but were not followed because the dev workflow Step 7 lacked an explicit cleanup mandate. Rules without execution are meaningless.

**How to apply:** After every `nx run web:test`, `go test`, or any test command completes, run `pnpm run test:cleanup` as the final step. This is non-negotiable — treat it as part of the test execution itself, not an optional follow-up.
