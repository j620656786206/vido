# Story: Deduplicate METADATA_ Error Codes — Rule 11 Canonicalization

Status: backlog

**Origin:** Winston (Architect) architectural review of retro-10-AI3 Rule 7 expansion — 2026-04-20.
**Priority:** MEDIUM (Rule 11 smell affecting live wire contract; not a user-visible bug).
**Scope estimate:** 2–3 Go files touched, ~30 LOC delta, possibly 1 boundaries_test.go amendment.

## Problem

`apps/api/internal/metadata/provider.go` is the canonical source of `METADATA_*` wire-contract error codes (package doc comment: _"Common error codes for metadata providers"_). It declares 7 codes at lines 232–247:

```
METADATA_NO_RESULTS, METADATA_TIMEOUT, METADATA_RATE_LIMITED, METADATA_UNAVAILABLE,
METADATA_INVALID_REQUEST, METADATA_ALL_FAILED, METADATA_CIRCUIT_OPEN
```

`apps/api/internal/retry/metadata_integration.go` **violates Rule 11** in two ways:

1. **Lines 9–15** — mirrors 5 of the canonical codes as a sibling `const (...)` block with comment "Metadata error codes from provider.go". This is an explicit acknowledgment of the duplication.
2. **Lines 107, 120, 133, 144, 156, 166, 186, 195** — `ClassifyMetadataError()` constructs `RetryableError` values that introduce **4 new wire codes not declared in provider.go**:
   - `METADATA_GATEWAY_ERROR`
   - `METADATA_NETWORK_ERROR`
   - `METADATA_NOT_FOUND`
   - `METADATA_UNKNOWN_ERROR`

The second item is the more serious defect: the retry package is silently expanding the wire contract beyond what the canonical `metadata` package declares. Frontend/ops see error codes with no authoritative definition.

## Acceptance Criteria

1. Given a reader greps `apps/api/internal/` for `"METADATA_`, when the grep returns hits, then **every quoted `METADATA_*` string constant lives in `apps/api/internal/metadata/provider.go`** and nowhere else (except test assertions and the Rule 7 CR workflow inline list).
2. Given `apps/api/internal/retry/metadata_integration.go` after the change, when reading the file, then the local `const (...)` block (currently lines 9–15) is **deleted** and replaced by `import "github.com/vido/api/internal/metadata"` + usage as `metadata.ErrCodeTimeout` (etc.).
3. Given `ClassifyMetadataError()` in retry after the change, when reading the function, then the four retry-only codes (`METADATA_GATEWAY_ERROR`, `METADATA_NETWORK_ERROR`, `METADATA_NOT_FOUND`, `METADATA_UNKNOWN_ERROR`) are **promoted to exported constants in `metadata/provider.go`** (with doc comments following the existing style) and referenced via `metadata.ErrCode*` rather than hard-coded string literals.
4. Given `go test ./...` runs after the change, when it completes, then all tests pass with no behavioral drift — the wire contract values are byte-identical to pre-change (this is a refactor, not a rename).
5. Given Rule 19 boundaries (`apps/api/internal/boundaries_test.go`), when executed, then the retry → metadata import direction is verified **allowed** (retry is a leaf-ish utility; metadata is a leaf per project-context.md Rule 19). If the leaf list needs amendment, update `project-context.md` Rule 19 list in the same commit.
6. Given the CR auto-fix prefix map in `_bmad/bmm/workflows/4-implementation/code-review/instructions.xml:149` currently maps `metadata/** or retry/**  → METADATA_`, when this story completes, then the map can stay as-is (both packages still legitimately emit METADATA_ codes via the canonical constants). No XML edit required.

## Out of Scope

- Renaming `METADATA_*` codes (the prefix is correct per Winston's Item 2 approval).
- Restructuring `ClassifyMetadataError()` logic — the string-matching classifier is preserved; only the constant references change.
- Touching provider implementations under `metadata/tmdb/`, `metadata/douban/`, etc. — those already use their source-specific prefixes.

## References

- Winston (Architect) verdict — retro-10-AI3 Winston-prompt file, Item 1 ruling (2026-04-20)
- Rule 11 source: `project-context.md#rule-11-interface-location`
- Rule 19 source: `project-context.md#rule-19-package-dependency-boundaries` (leaf package list)
- Canonical file: `apps/api/internal/metadata/provider.go:232-247`
- Offending mirror: `apps/api/internal/retry/metadata_integration.go:9-15, 107, 120, 133, 144, 156, 166, 186, 195`
