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
