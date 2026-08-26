import { describe, it, expect } from 'vitest';
import { formatUsdShort } from './formatUsdShort';

// H8-SPEC-v3 folding rules (ux3-1-7 AC #6): precision folds as magnitude grows
// so the worst case stays at 9–10 characters — no truncation, no tooltip.
describe('formatUsdShort (H8-SPEC-v3 金額顯示規則)', () => {
  it('< $10 → one decimal', () => {
    expect(formatUsdShort(1.2)).toBe('$1.2');
    expect(formatUsdShort(0.42)).toBe('$0.4');
    expect(formatUsdShort(9.99)).toBe('$10'); // rounds up across the boundary
  });

  it('whole dollars fold the trailing .0 away ($5, not $5.0)', () => {
    expect(formatUsdShort(5)).toBe('$5');
    expect(formatUsdShort(0)).toBe('$0');
  });

  it('$10–999 → integer', () => {
    expect(formatUsdShort(12)).toBe('$12');
    expect(formatUsdShort(123.45)).toBe('$123');
    expect(formatUsdShort(999.4)).toBe('$999');
  });

  it('≥ $1000 → k with one decimal', () => {
    expect(formatUsdShort(1234)).toBe('$1.2k');
    expect(formatUsdShort(1000)).toBe('$1k');
    expect(formatUsdShort(99000)).toBe('$99k');
  });

  it('garbage in → an honest $0, never NaN text', () => {
    expect(formatUsdShort(Number.NaN)).toBe('$0');
    expect(formatUsdShort(-3)).toBe('$0');
    expect(formatUsdShort(Number.POSITIVE_INFINITY)).toBe('$0');
  });
});
