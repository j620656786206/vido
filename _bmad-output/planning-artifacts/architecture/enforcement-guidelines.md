# Enforcement Guidelines

## All AI Agents MUST:

1. **Read `project-context.md` FIRST** (if exists) before implementing any code
2. **Follow naming conventions EXACTLY** - No deviations allowed
3. **Use the unified backend** (`/apps/api`) - Never add code to root `/cmd` or `/internal`
4. **Implement error handling** using `AppError` types with proper error codes
5. **Use `slog` for logging** - Never use `fmt.Println`, `log.Print`, or other logging libraries
6. **Follow the API response format** - All responses must use `ApiResponse<T>` wrapper
7. **Write tests** alongside implementation - `*_test.go` or `*.spec.tsx` co-located
8. **Use TanStack Query** for server state - Never use Zustand/Redux for API data
9. **Follow layered architecture** - Handlers → Services → Repositories (no shortcuts)
10. **Validate inputs** at handler/component level before processing
11. **No authentication middleware** - Vido v4 is single-user with no auth (deferred to v5.0)

## Plugin Validation Rules:

1. **TestConnection before save** - Always call `plugin.TestConnection(ctx, config)` before persisting plugin configuration to SQLite. If TestConnection fails, reject the config with a descriptive error.
2. **Health check on startup** - When the application starts, run `TestConnection()` on all registered plugins. Log warnings for unreachable plugins but do not block startup.
3. **Graceful degradation** - If a plugin becomes unavailable at runtime, mark it as degraded and continue operating. Never crash or block other plugins due to one plugin's failure.
4. **Config validation** - Each plugin must validate its required config fields (URL, API key, etc.) before attempting TestConnection.
5. **Plugin interface compliance** - All plugins must implement the base `Plugin` interface (`Name()`, `TestConnection()`). Type-specific plugins must implement their full interface (`MediaServerPlugin`, `DownloaderPlugin`, or `DVRPlugin`).

## Pattern Verification Checklist:

Before committing code, verify:

- [ ] File and variable naming follows conventions
- [ ] API endpoints use `/api/v1/{resource}` pattern
- [ ] Database tables/columns use `snake_case`
- [ ] Error responses include error code, message, and suggestion
- [ ] Dates are ISO 8601 strings in JSON
- [ ] Tests are co-located with source files
- [ ] No code added to deprecated `/cmd` or root `/internal`
- [ ] Logging uses `slog` with structured fields
- [ ] API responses use standard wrapper format
- [ ] TanStack Query used for server state, NOT Zustand
- [ ] No authentication/JWT middleware added (v4 is single-user)
- [ ] Plugin configs validated with TestConnection before persistence
- [ ] Plugin health checks registered with plugin manager

## Pattern Violations:

**If you find pattern violations during code review:**

1. **Document the violation** - Note location and pattern broken
2. **Fix immediately** if in new code (< 1 week old)
3. **Create refactoring task** if in existing code (add to Phase 5 testing)
4. **Update this document** if pattern needs clarification

## Updating Patterns:

**Process for pattern changes:**

1. **Identify need** - Why does pattern need to change?
2. **Propose change** - Document new pattern and rationale
3. **User approval** - Get user sign-off before adopting
4. **Update document** - Modify this architecture.md file
5. **Refactor existing code** - Update all code to new pattern
6. **Verify compliance** - Ensure all agents follow new pattern

---
