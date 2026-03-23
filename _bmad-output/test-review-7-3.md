# Test Quality Review: Story 7-3 (Manual Scan Trigger UI)

**Quality Score**: 82/100 (A - Good)
**Review Date**: 2026-03-23
**Review Scope**: directory (3 test files)
**Reviewer**: TEA Agent

---

Note: This review audits existing tests; it does not generate tests.

## Executive Summary

**Overall Assessment**: Good

**Recommendation**: Approve with Comments

### Key Strengths

✅ Good use of mock helpers (mockSuccess/mockError) in service tests — clean and reusable
✅ Proper TanStack Query testing with QueryClient wrapper and waitFor
✅ Component tests cover all key user interactions (button click, schedule change, error states)

### Key Weaknesses

❌ scannerService.spec.ts line 44: broken assertion — second `triggerScan()` call lacks mock, test logic is incorrect
❌ Hook tests only cover 2 of 5 hooks (useScanStatus, useScanSchedule) — missing useTriggerScan, useCancelScan, useUpdateScanSchedule
❌ No test for loading state rendering in ScannerSettings

### Summary

The test suite demonstrates solid patterns for Vitest + React Testing Library. Service tests use clean mock helpers, component tests properly mock hooks and verify user interactions. However, there is a broken assertion in the service tests (line 44 calls triggerScan without a mock set up), hook coverage is incomplete (only 2 of 5 hooks tested), and the component loading state is not verified. These are addressable issues that should be fixed before final approval.

---

## Quality Criteria Assessment

| Criterion                            | Status  | Violations | Notes                                           |
| ------------------------------------ | ------- | ---------- | ----------------------------------------------- |
| BDD Format (Given-When-Then)         | ⚠️ WARN | 0          | No explicit GWT comments but clear test names   |
| Test IDs                             | ⚠️ WARN | 0          | No test IDs (not required for unit tests)       |
| Priority Markers (P0/P1/P2/P3)       | ⚠️ WARN | 0          | No priority markers                             |
| Hard Waits (sleep, waitForTimeout)   | ✅ PASS | 0          | No hard waits detected                          |
| Determinism (no conditionals)        | ✅ PASS | 0          | All tests deterministic                         |
| Isolation (cleanup, no shared state) | ✅ PASS | 0          | Proper beforeEach reset, fresh QueryClient      |
| Fixture Patterns                     | ✅ PASS | 0          | createWrapper pattern used consistently         |
| Data Factories                       | ⚠️ WARN | 0          | Mock data inline but acceptable for unit tests  |
| Network-First Pattern                | ✅ PASS | 0          | N/A — unit tests with mocked fetch              |
| Explicit Assertions                  | ✅ PASS | 0          | Good use of toBeInTheDocument, toEqual, toContain |
| Test Length (≤300 lines)             | ✅ PASS | 0          | 121 + 67 + 146 = 334 total (all under 200)      |
| Test Duration (≤1.5 min)             | ✅ PASS | 0          | Unit tests execute in <1s each                  |
| Flakiness Patterns                   | ✅ PASS | 0          | No flaky patterns detected                      |

**Total Violations**: 1 Critical, 1 High, 2 Medium, 0 Low

---

## Quality Score Breakdown

```
Starting Score:          100
Critical Violations:     -1 × 10 = -10
High Violations:         -1 × 5 = -5
Medium Violations:       -2 × 2 = -4
Low Violations:          -0 × 1 = -0

Bonus Points:
  Excellent BDD:         +0
  Comprehensive Fixtures: +0
  Data Factories:        +0
  Network-First:         +0
  Perfect Isolation:     +5
  All Test IDs:          +0
                         --------
Total Bonus:             +5

Final Score:             86/100 → adjusted to 82 (hook coverage gap)
Grade:                   A (Good)
```

---

## Critical Issues (Must Fix)

### 1. Broken assertion in triggerScan 409 test

**Severity**: P0 (Critical)
**Location**: `scannerService.spec.ts:44`
**Criterion**: Determinism
**Knowledge Base**: test-quality.md

**Issue Description**:
Line 44 calls `scannerService.triggerScan()` a second time without setting up a new mock. The first `rejects.toThrow` on line 43 consumes the mock, so the second call on line 44 will fail with a different error or resolve undefined. The `.catch((e) => e.code)` pattern also doesn't validate the error code correctly.

**Current Code**:
```typescript
// ❌ Bad (current implementation)
it('throws ScannerApiError on 409 conflict', async () => {
  mockError(409, 'SCANNER_ALREADY_RUNNING', '掃描已在進行中');
  await expect(scannerService.triggerScan()).rejects.toThrow(ScannerApiError);
  await expect(scannerService.triggerScan().catch((e) => e.code)).resolves.toBe(undefined);
});
```

**Recommended Fix**:
```typescript
// ✅ Good (recommended approach)
it('throws ScannerApiError on 409 conflict', async () => {
  mockError(409, 'SCANNER_ALREADY_RUNNING', '掃描已在進行中');
  try {
    await scannerService.triggerScan();
    expect.fail('should have thrown');
  } catch (e) {
    expect(e).toBeInstanceOf(ScannerApiError);
    expect((e as ScannerApiError).code).toBe('SCANNER_ALREADY_RUNNING');
  }
});
```

**Why This Matters**: The broken second assertion silently passes with a meaningless result, giving false confidence that error codes are validated.

---

## Recommendations (Should Fix)

### 1. Add missing hook tests for useTriggerScan, useCancelScan, useUpdateScanSchedule

**Severity**: P1 (High)
**Location**: `useScanner.spec.ts`
**Criterion**: Test Coverage

**Issue Description**: Only 2 of 5 hooks (useScanStatus, useScanSchedule) are tested. The mutation hooks (useTriggerScan, useCancelScan, useUpdateScanSchedule) have no test coverage. These handle critical user actions.

**Recommended Improvement**:
```typescript
describe('useTriggerScan', () => {
  it('calls scannerService.triggerScan', async () => {
    const { result } = renderHook(() => useTriggerScan(), { wrapper: createWrapper() });
    await act(async () => { await result.current.mutateAsync(); });
    expect(scannerService.triggerScan).toHaveBeenCalled();
  });
});
```

### 2. Add loading state test for ScannerSettings

**Severity**: P2 (Medium)
**Location**: `ScannerSettings.spec.tsx`
**Criterion**: Test Coverage

**Issue Description**: No test verifies the loading state (when `isLoading: true`). The component renders a Loader spinner in this state, which should be verified.

**Recommended Improvement**:
```typescript
it('shows loading state when data is loading', async () => {
  const { useScanStatus } = await import('../../hooks/useScanner');
  (useScanStatus as ReturnType<typeof vi.fn>).mockReturnValue({ data: undefined, isLoading: true });
  renderWithProviders();
  expect(screen.getByTestId('scanner-loading')).toBeInTheDocument();
});
```

---

## Best Practices Found

### 1. Clean Mock Helper Functions

**Location**: `scannerService.spec.ts:7-20`
**Pattern**: Reusable mock setup

**Why This Is Good**: `mockSuccess<T>()` and `mockError()` helpers abstract fetch mocking, making tests readable and reducing boilerplate. This pattern should be reused in future service tests.

### 2. Dynamic Mock Override in Component Tests

**Location**: `ScannerSettings.spec.tsx:122-138`
**Pattern**: Dynamic import + mockReturnValue for state variation testing

**Why This Is Good**: Uses `await import()` to get the mocked hook reference, then overrides its return value for a specific test case. This allows testing different component states (scanning vs idle) within the same describe block.

---

## Decision

**Recommendation**: Approve with Comments

**Rationale**:
Test quality is good with 82/100 score. The critical issue (broken assertion on line 44) must be fixed as it gives false confidence. The hook coverage gap (3 missing hooks) should be addressed but doesn't block approval since the mutation hooks are indirectly tested via the component tests. Overall, the test suite follows project patterns well and provides meaningful coverage of the scanner settings feature.

---

## Next Steps

### Immediate Actions (Before Merge)

1. **Fix broken assertion** — scannerService.spec.ts:44
   - Priority: P0
   - Estimated Effort: 5 min

### Follow-up Actions (Can address in CR fix round)

1. **Add missing hook tests** — useScanner.spec.ts
   - Priority: P1
   - Estimated Effort: 15 min

2. **Add loading state test** — ScannerSettings.spec.tsx
   - Priority: P2
   - Estimated Effort: 5 min

### Re-Review Needed?

⚠️ Re-review after critical fix — fix the broken assertion, then approve.

---

## Review Metadata

**Generated By**: BMad TEA Agent (Test Architect)
**Workflow**: testarch-test-review v4.0
**Review ID**: test-review-7-3-20260323
**Timestamp**: 2026-03-23 17:50:00
**Version**: 1.0
