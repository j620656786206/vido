# ADR: OpenCC backend migration to the official C++ helper

**Status:** In progress (opt-in implementation)

## Decision

Use the official C++ OpenCC CLI as the production conversion backend, invoked by the Go API through stdin/stdout. Preserve the existing `ConvertS2TWP` contract and keep the backend selectable during migration.

## Why

- The upstream OpenCC project is actively maintained and Apache-2.0 licensed.
- `s2twp` quality and phrase dictionaries remain the compatibility target.
- A subprocess avoids cgo and reduces cross-compilation risk for NAS `amd64`/`arm64` images.
- The adapter boundary makes rollback possible while golden-output parity is measured.

## Guardrails

- 30-second timeout; conversion failures return the original bytes plus an error.
- UTF-8 BOM preservation and profile allow-list are unchanged.
- The current Go backend is development fallback only; it must be removed before a permissive project license is declared because its dependency graph includes GPL-2.0 `cedar-go`.

## Exit criteria

Golden corpus parity, multi-architecture image tests, NAS smoke tests, and SBOM/license review must pass before switching the default backend or publishing a project-wide MIT/Apache license.
