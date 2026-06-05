---
name: Run Prettier before commit
description: Always run pnpm run format:check locally before committing to avoid CI formatting failures
type: feedback
originSessionId: 46053282-464e-4afb-87a5-b8aa371c55ad
---
Run `pnpm run format:check` (or `npx prettier --check .`) before committing changes that touch .ts/.tsx/.md files. The CI pipeline runs Prettier as a format gate and will fail if files aren't formatted.

**Why:** CI failed on `fix: format 4 files to pass Prettier CI check` because agent-edited files weren't run through Prettier before push. This caused a wasted CI cycle.

**How to apply:** After editing TypeScript/React/Markdown files, run `pnpm run format:check` and fix with `npx prettier --write <files>` before committing. This is especially important when subagents edit files (they don't run Prettier).
