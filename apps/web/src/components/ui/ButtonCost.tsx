// Implements: Component/ButtonCost/Default (qAERt) + Component/ButtonCost/Loading (zhIx7) + Component/ButtonCost/Disabled (dqE4G)
// Source: ux-design.pen (Pencil app)
/**
 * A control that spends money (story dsr-6a AC #4, DESIGN.md「會花錢的動作要有記號」,
 * spec screen J9-D).
 *
 * The `$` amount IS the marker — no coin icon, no colour of its own. The marker
 * is fixed because it is bound to clickability: whenever this button can be
 * pressed, an amount is on it. So there are exactly three looks:
 *
 * - `ready`       — accent fill, label + amount (`≈` only when the runtime is assumed)
 * - `loading`     — same fill, a fixed-width skeleton where the amount will be;
 *                   not clickable yet (J9-D ④ — and the width must not jump)
 * - `unavailable` — disabled, no amount; the caller writes the reason in a line
 *                   next to it and points `aria-describedby` at that line
 *
 * `busy` (the paid request is in flight) keeps the content as-is: no spinner is
 * inserted, because inserting one widens the button mid-click.
 */
import * as React from 'react';
import { cn } from '../../lib/utils';
import { usdWithEstimate } from '../../lib/currency';

export type ButtonCostState =
  | { status: 'ready'; usd: number; approximate: boolean }
  | { status: 'loading' }
  | { status: 'unavailable' };

export interface ButtonCostProps extends Omit<
  React.ComponentProps<'button'>,
  'children' | 'disabled'
> {
  label: string;
  cost: ButtonCostState;
  busy?: boolean;
  'data-testid'?: string;
}

export function ButtonCost({
  label,
  cost,
  busy = false,
  onClick,
  className,
  type = 'button',
  'data-testid': testId,
  ...rest
}: ButtonCostProps) {
  const unavailable = cost.status === 'unavailable';
  const loading = cost.status === 'loading';
  const clickable = cost.status === 'ready' && !busy;

  const handleClick = (event: React.MouseEvent<HTMLButtonElement>) => {
    if (!clickable) {
      event.preventDefault();
      return;
    }
    onClick?.(event);
  };

  return (
    <button
      {...rest}
      type={type}
      data-testid={testId}
      data-cost-status={cost.status}
      disabled={unavailable || busy}
      aria-disabled={loading ? true : undefined}
      aria-busy={busy ? true : undefined}
      onClick={handleClick}
      className={cn(
        // The border is there in EVERY state (transparent unless disabled), so
        // loading → unavailable does not grow the button by 2px.
        'inline-flex min-h-[44px] items-center justify-center gap-2 whitespace-nowrap rounded-[var(--radius-md)] border px-5 text-sm font-semibold transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)]',
        unavailable
          ? 'cursor-not-allowed border-[var(--border-subtle)] bg-[var(--bg-tertiary)] text-[var(--text-disabled)]'
          : 'border-transparent bg-[var(--accent-primary)] text-[var(--text-on-accent)]',
        clickable && 'hover:bg-[var(--accent-pressed)]',
        loading && 'cursor-default',
        busy && 'cursor-wait',
        className
      )}
    >
      <span>{label}</span>
      {cost.status === 'ready' && (
        <>
          {' '}
          <span
            data-testid={testId ? `${testId}-amount` : undefined}
            className="inline-block min-w-[5ch] font-mono font-semibold tabular-nums"
          >
            {usdWithEstimate(cost.usd, cost.approximate)}
          </span>
        </>
      )}
      {loading && (
        <>
          <span
            aria-hidden="true"
            data-testid={testId ? `${testId}-amount-skeleton` : undefined}
            // font-mono: `5ch` must be measured in the AMOUNT's font, or the
            // skeleton and the number it becomes differ in width.
            className="inline-block h-3 w-[5ch] rounded-[var(--radius-sm)] bg-[var(--text-on-accent)] font-mono opacity-25"
          />
          <span className="sr-only">正在估算費用</span>
        </>
      )}
    </button>
  );
}
