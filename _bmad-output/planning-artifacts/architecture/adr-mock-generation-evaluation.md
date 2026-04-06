# ADR: Mock Generation Tool Evaluation

**Date:** 2026-04-06
**Status:** Recommendation — Not Yet Adopted
**Source:** Epic 9c Retro Action Item AI-3

---

## Context

The Vido Go backend has 57 mock structs with 357 manually maintained methods across 42 test files. All mocks use `testify/mock` with hand-written method implementations. When an interface changes, every mock implementation must be manually updated — a recurring friction point observed during Epic 9c (6+ files needed `GetStats` mock stubs for a single new method).

### Current State

| Metric | Value |
|--------|-------|
| Mock structs | 57 (2 centralized + 55 scattered) |
| Mock methods | 357 (40 centralized, 317 scattered) |
| Interfaces needing mocks | 68+ |
| Code generation | None (`//go:generate` = 0) |
| Mock tool | testify/mock only (manual) |

---

## Options Evaluated

### Option A: mockery (github.com/vektra/mockery)

**Pros:**
- Auto-generates testify/mock implementations from interfaces
- `.mockery.yaml` config for project-wide rules
- `//go:generate mockery` per-interface or `mockery --all` bulk
- Output is standard testify/mock — no test rewrite needed
- Active maintenance, wide adoption

**Cons:**
- New dependency (dev-only)
- Generated code adds to repo size (or needs CI step)
- Migration of 357 existing methods is non-trivial
- Some complex return types need manual type assertion tweaks

### Option B: gomock (go.uber.org/mock)

**Pros:**
- Google-backed, well-maintained
- `mockgen` generates from interfaces or source
- Strong IDE support

**Cons:**
- Different API than testify/mock — requires rewriting ALL test expectations
- `EXPECT().Return()` style vs current `On().Return()` style
- Migration cost is very high (357 methods + all test call sites)

### Option C: Status Quo (manual testify/mock)

**Pros:**
- No new dependencies
- Team already familiar
- Full control over mock behavior

**Cons:**
- 357 methods to maintain manually
- Interface changes cascade to 5-10+ files
- Error-prone type casting boilerplate
- Scales linearly with codebase growth

---

## Recommendation: Option A (mockery) — Gradual Adoption

**Rationale:** mockery generates standard testify/mock code, so existing tests don't need rewriting. The migration can be incremental — new interfaces use mockery immediately, existing mocks migrate opportunistically.

### Adoption Plan

**Phase 1 — Setup (one-time):**
1. `go install github.com/vektra/mockery/v2@latest`
2. Create `.mockery.yaml` at project root with output config
3. Add `//go:generate mockery --name=InterfaceName` to new interfaces

**Phase 2 — New interfaces only:**
- All new interfaces get `//go:generate mockery` directive
- Generated mocks go to `internal/testutil/generated/`
- Manual mocks remain as-is for existing interfaces

**Phase 3 — Opportunistic migration (during story work):**
- When a story modifies an interface, migrate its mock to mockery
- Delete the hand-written mock, replace with generated one
- No dedicated "migration sprint" needed

### NOT Recommended Now

- Bulk migration of all 357 methods (too disruptive, no user value)
- gomock adoption (requires rewriting test expectations)
- Mandatory enforcement (let it prove value first)

---

## Decision

**Defer adoption until a story naturally requires adding 3+ methods to an existing interface.** At that point, the dev should:
1. Install mockery
2. Generate the mock instead of hand-writing
3. Validate the approach works with our test patterns

This keeps the evaluation LOW priority while ensuring the tool gets real-world validation before broader adoption.
