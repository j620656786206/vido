import { divideUsd, fixedUsd } from '../lib/currency';

/**
 * H8-SPEC-v3 amount folding (ux3-1-7 AC #6): precision folds as magnitude
 * grows, so the worst-case readout string stays at 9–10 characters and the
 * band cell never needs truncation or a tooltip.
 *
 *   < $10    → one decimal   ($1.2)
 *   $10–999  → integer       ($123)
 *   ≥ $1000  → k, one decimal ($1.2k)
 */
export function formatUsdShort(amount: number): string {
  if (!Number.isFinite(amount) || amount < 0) return '$0';
  // CR M5: rounds through lib/currency, not native toFixed/Math.round. Those
  // round the DOUBLE, which disagrees with the decimal rounding the consent
  // and execution screens now use at `.x5` boundaries — and this band renders
  // the same spent_usd / budget_usd figures those screens do. A half-converted
  // money path is worse than an unconverted one: two surfaces, same number,
  // different answer.
  if (amount >= 1000) {
    return `$${trimTrailingZero(fixedUsd(divideUsd(amount, 1000), 1))}k`;
  }
  if (amount >= 10) return `$${fixedUsd(amount, 0)}`;
  return `$${trimTrailingZero(fixedUsd(amount, 1))}`;
}

function trimTrailingZero(fixed: string): string {
  return fixed.endsWith('.0') ? fixed.slice(0, -2) : fixed;
}
