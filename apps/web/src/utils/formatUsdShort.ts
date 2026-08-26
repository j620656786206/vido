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
  if (amount >= 1000) {
    const k = amount / 1000;
    // toFixed would print $1.0k for exactly 1000 — fold the trailing zero away
    // the same way the sub-$10 branch does.
    return `$${trimTrailingZero(k.toFixed(1))}k`;
  }
  if (amount >= 10) return `$${Math.round(amount)}`;
  return `$${trimTrailingZero(amount.toFixed(1))}`;
}

function trimTrailingZero(fixed: string): string {
  return fixed.endsWith('.0') ? fixed.slice(0, -2) : fixed;
}
