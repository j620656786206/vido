# Test Execution Process Safety

## Principle

Never run test suites (vitest, playwright, nx test) in background mode. Always execute test commands in the foreground and wait for completion before proceeding.

## Rationale

Background test execution creates orphaned worker processes:

1. Test runners (vitest) fork multiple worker processes (one per CPU core)
2. When the parent shell process exits (e.g., background task completes or is abandoned), worker child processes become orphans
3. Orphaned workers continue consuming CPU/memory indefinitely until manually killed
4. Multiple abandoned test runs compound the problem, eventually starving system resources

**Incident (2026-03-15):** A background `nx test web` command left 11 vitest worker processes running at ~15-100% CPU each. They persisted until manually killed with `kill`.

## Correct Patterns

### Always foreground

```bash
# ✅ CORRECT — foreground, wait for completion
npx vitest --run src/routes/library.spec.tsx

# ✅ CORRECT — full suite, foreground
npx nx test web -- --run

# ❌ WRONG — background execution
npx nx test web -- --run &

# ❌ WRONG — run_in_background tool parameter
# (creates same orphan problem via shell backgrounding)
```

### Never duplicate test runs

```bash
# ❌ WRONG — launching same tests twice (background + foreground)
# Background: npx nx test web (still running)
# Foreground: npx vitest --run same-file.spec.tsx (duplicate)

# ✅ CORRECT — wait for first run, then proceed
npx nx test web -- --run
# (after completion) proceed with next task
```

### Cleanup if orphans occur

```bash
# Find orphaned vitest workers
ps aux | grep vitest | grep -v grep

# Kill them
kill <pid1> <pid2> ...

# Or use project cleanup script
pnpm run test:cleanup:all
```

## Applicability

- Local development (Claude Code sessions)
- CI pipelines (use proper job orchestration instead of background processes)
- Any environment where test runners fork worker processes
