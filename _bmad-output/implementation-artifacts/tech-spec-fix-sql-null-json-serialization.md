---
title: 'Fix sql.Null* JSON Serialization'
slug: 'fix-sql-null-json-serialization'
created: '2026-03-27'
status: 'implementation-complete'
stepsCompleted: [1, 2, 3, 4]
tech_stack: [Go, database/sql, encoding/json, stretchr/testify]
files_to_modify:
  - apps/api/internal/models/types.go (NEW)
  - apps/api/internal/models/types_test.go (NEW)
  - apps/api/internal/models/movie.go
  - apps/api/internal/models/series.go
  - apps/api/internal/models/season.go
  - apps/api/internal/models/episode.go
  - apps/api/internal/services/converters.go
  - apps/api/internal/repository/learning_repository.go
  - apps/api/internal/repository/series_repository.go
  - apps/api/internal/models/movie_test.go
  - apps/api/internal/models/series_test.go
  - apps/api/internal/models/season_test.go
  - apps/api/internal/models/episode_test.go
code_patterns: [embedded struct, custom JSON marshaler, Go sql nullable types]
test_patterns: [JSON roundtrip test, table-driven tests, testify assert/require]
---

# Tech-Spec: Fix sql.Null* JSON Serialization

**Created:** 2026-03-27

## Overview

### Problem Statement

Go's `sql.NullString`, `sql.NullInt64`, `sql.NullFloat64`, `sql.NullBool`, and `sql.NullTime` types serialize to JSON as objects (`{"String":"value","Valid":true}`) instead of plain values (`"value"`). This causes React error #31 ("Objects are not valid as a React child") on the library page when rendering the 4385 scanned media items. All 4 model types (Movie, Series, Season, Episode) are affected.

### Solution

Create custom nullable wrapper types in `models/types.go` that embed the `sql.Null*` types and implement `json.Marshaler` / `json.Unmarshaler`. Replace all `sql.Null*` field types in the 4 model structs with the custom types. This is the standard Go community pattern (same approach as `guregu/null` package) — one file, automatic coverage for all endpoints.

### Scope

**In Scope:**
- New `models/types.go` with custom `NullString`, `NullInt64`, `NullFloat64`, `NullBool`, `NullTime` types
- Replace `sql.Null*` → custom types in Movie, Series, Season, Episode models
- Update `converters.go` references (constructor calls like `sql.NullString{...}`)
- Update repository files that construct `sql.Null*` directly
- JSON roundtrip tests for all 5 custom types
- Existing test files that construct `sql.Null*` in test data

**Out of Scope:**
- Response DTO layer (deferred to v5 if needed)
- Handler changes (none required — JSON serialization is transparent)
- Frontend changes (fix is backend-only)
- API contract changes (same JSON keys, correct value types)

## Context for Development

### Codebase Patterns

- Models use `db:"column"` tags for sqlx and `json:"camelCase,omitempty"` for API responses
- Some `NullString` fields use `json:"-"` (CreditsJSON, SeasonsJSON, etc.) — these are internal storage and don't need the fix but should still use the custom type for consistency
- `converters.go` constructs nullable values like `sql.NullString{String: val, Valid: true}`
- Repository files use `sql.NullString` as local variables for scan targets — these stay as `sql.NullString` since they're not serialized
- Tests use `testify` assert/require, table-driven pattern

### Files to Reference

| File | Purpose |
| ---- | ------- |
| `apps/api/internal/models/movie.go` | Movie model — 9 NullString (+3 json:"-"), 4 NullInt64, 3 NullFloat64, 1 NullTime = 17 exposed fields |
| `apps/api/internal/models/series.go` | Series model — 10 NullString (+3 json:"-"), 4 NullInt64, 3 NullFloat64, 1 NullBool, 1 NullTime = 19 exposed fields |
| `apps/api/internal/models/season.go` | Season model — 4 NullString, 2 NullInt64, 1 NullFloat64 = 7 fields |
| `apps/api/internal/models/episode.go` | Episode model — 6 NullString, 2 NullInt64, 1 NullFloat64 = 9 fields |
| `apps/api/internal/services/converters.go` | Constructs model instances from TMDb data |
| `apps/api/internal/repository/learning_repository.go` | Uses `sql.NullString` as scan targets + has `nullString()` helper |
| `apps/api/internal/repository/series_repository.go` | Uses `sql.NullString` in subtitle update |

### Technical Decisions

- **Embedding over aliasing**: Use `struct { sql.NullString }` embedding so `Scan()` and `Value()` interfaces pass through automatically
- **`omitempty` behavior**: When `Valid=false`, marshal to JSON `null`. With `omitempty` tag, the field is omitted entirely — matching current expected behavior
- **Backward compatibility**: The `json:"-"` tagged fields (CreditsJSON, etc.) are unaffected by the marshaler since they're excluded from JSON output
- **Repository scan targets**: Local variables in repositories that use `sql.NullString` for scanning can stay as `sql.NullString` — they're not serialized. Only model struct fields need the custom type
- **Constructor pattern**: Provide helper functions `NewNullString(s string)`, `NewNullInt64(i int64)`, etc. to replace verbose `sql.NullString{String: val, Valid: true}` constructors in converters.go

## Implementation Plan

### Tasks

- [x] Task 1: Create `models/types.go` with 5 custom nullable types
  - File: `apps/api/internal/models/types.go` (NEW)
  - Action: Define `NullString`, `NullInt64`, `NullFloat64`, `NullBool`, `NullTime` structs that embed corresponding `sql.Null*` types. Implement `MarshalJSON()` and `UnmarshalJSON()` for each. Add constructor helpers `NewNullString(s string)`, `NewNullInt64(i int64)`, etc.
  - Notes: When `Valid=false`, `MarshalJSON` returns `[]byte("null")`. When `Valid=true`, marshal the inner value. `UnmarshalJSON` handles both `null` and typed values. Embedding ensures `Scan()` and `Value()` pass through.

- [x] Task 2: Create `models/types_test.go` with JSON roundtrip tests
  - File: `apps/api/internal/models/types_test.go` (NEW)
  - Action: Table-driven tests for all 5 types covering: (a) valid value → marshal → correct JSON primitive, (b) null value → marshal → `null`, (c) JSON primitive → unmarshal → valid value, (d) JSON `null` → unmarshal → invalid value. Also test constructor helpers.
  - Notes: Use testify assert. This is the critical regression guard.

- [x] Task 3: Replace `sql.Null*` in Movie model
  - File: `apps/api/internal/models/movie.go`
  - Action: Replace all `sql.NullString` → `NullString`, `sql.NullInt64` → `NullInt64`, `sql.NullFloat64` → `NullFloat64`, `sql.NullTime` → `NullTime` in the `Movie` struct field type declarations. Remove `"database/sql"` import if no longer needed (check for remaining `sql.*` usage in methods like `SetCredits`).
  - Notes: 20 fields total (17 exposed + 3 json:"-"). Keep `db:` and `json:` tags unchanged. Methods that construct `sql.NullString{...}` internally (e.g., `SetCredits`) need updating to `NullString{sql.NullString{...}}`.

- [x] Task 4: Replace `sql.Null*` in Series model
  - File: `apps/api/internal/models/series.go`
  - Action: Same replacement pattern as Task 3. Series has 22 fields (19 exposed + 3 json:"-") plus `NullBool` for `InProduction`.
  - Notes: Methods `SetCredits`, `SetSeasons`, `SetNetworks` construct `sql.NullString` internally — update these.

- [x] Task 5: Replace `sql.Null*` in Season model
  - File: `apps/api/internal/models/season.go`
  - Action: Same replacement pattern. 7 fields total.
  - Notes: Simpler model, no internal methods that construct nullable values.

- [x] Task 6: Replace `sql.Null*` in Episode model
  - File: `apps/api/internal/models/episode.go`
  - Action: Same replacement pattern. 9 fields total.
  - Notes: Simpler model, no internal methods that construct nullable values.

- [x] Task 7: Update `converters.go` constructors
  - File: `apps/api/internal/services/converters.go`
  - Action: Replace all `sql.NullString{String: val, Valid: true}` with `models.NewNullString(val)`, and similarly for other types. Update import from `"database/sql"` to use models package.
  - Notes: ~40 constructor calls to replace. Can use the helper functions to make this cleaner than the current verbose format.

- [x] Task 8: Update repository files
  - File: `apps/api/internal/repository/learning_repository.go`
  - Action: Update the `nullString()` helper function to return `models.NullString` instead of `sql.NullString`. Update any places where `sql.NullString` is used as a model field value (not scan targets — those stay as `sql.NullString`).
  - File: `apps/api/internal/repository/series_repository.go`
  - Action: Update `sql.NullString{String: path, Valid: path != ""}` to `models.NewNullString(path)` (line 805-806).

- [x] Task 9: Update existing test files
  - Files: `apps/api/internal/models/movie_test.go`, `series_test.go`, `season_test.go`, `episode_test.go`
  - Action: Replace `sql.NullString{String: "val", Valid: true}` → `NullString{sql.NullString{String: "val", Valid: true}}` or use the constructor `NewNullString("val")` in test data. Similarly for other Null* types.
  - Notes: Only test data construction needs changing — test assertions remain the same.

- [x] Task 10: Verify compilation and run all tests
  - Action: Run `go build ./...` and `go test ./...` from `apps/api/`. Ensure zero compilation errors and all existing tests pass.
  - Notes: If any test fails, it's a regression — fix before proceeding.

### Acceptance Criteria

- [x] AC 1: Given a Movie with `Overview = NullString{sql.NullString{String: "A great film", Valid: true}}`, when serialized to JSON, then `overview` field is `"A great film"` (string), not `{"String":"A great film","Valid":true}` (object)
- [x] AC 2: Given a Movie with `Overview = NullString{sql.NullString{Valid: false}}`, when serialized to JSON with `omitempty` tag, then `overview` field outputs `null` (Note: Go's json.Marshaler types are not omitted by omitempty; `null` is safe for React rendering)
- [x] AC 3: Given a valid JSON string `{"overview":"A great film"}`, when deserialized into a Movie struct, then `Overview.String` is `"A great film"` and `Overview.Valid` is `true`
- [x] AC 4: Given a JSON with `{"overview":null}`, when deserialized into a Movie struct, then `Overview.Valid` is `false`
- [x] AC 5: Given all 5 custom types (NullString, NullInt64, NullFloat64, NullBool, NullTime), when used in database scan operations, then `Scan()` and `Value()` work identically to the original `sql.Null*` types
- [x] AC 6: Given the updated models, when all existing tests in `apps/api/` are run, then all tests pass with zero failures
- [ ] AC 7 (deferred to deploy): Given the updated Docker image deployed to Unraid, when browsing the library page at `/library`, then media items render without React error #31

## Additional Context

### Dependencies

- No external libraries needed — uses only Go stdlib (`database/sql`, `encoding/json`, `time`)
- No API contract changes — same JSON keys, just corrected value types
- No frontend changes needed — React will receive primitives instead of objects

### Testing Strategy

- **Unit tests**: `types_test.go` — JSON marshal/unmarshal roundtrip for all 5 types (valid + null cases)
- **Regression tests**: All existing model tests must continue to pass
- **Integration verification**: Deploy updated Docker image to Unraid and verify library page renders
- **No new E2E tests needed** — this is a serialization fix, not a feature change

### Notes

- **Risk**: Near zero. All changes are compile-time verified. Embedding preserves database behavior.
- **Future consideration**: If v5 introduces a public API with different response shapes, a DTO layer can be added on top of these custom types without conflict.
- **Performance**: No measurable impact — custom marshalers are trivial operations (nil check + delegate).
