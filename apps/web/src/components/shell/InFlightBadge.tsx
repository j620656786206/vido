// Implements: <utility — no .pen counterpart>
/**
 * The 活動 count badge — one component for the three shells that draw it
 * (expanded sidebar, collapsed rail, mobile tab bar), because it is the app's
 * single instance of motion licence ③ and that licence needs one place to live.
 *
 * ③ 它真的在跑 — the ONLY thing on screen allowed to move without a gesture.
 * The rule from styles.css:「動的東西＝正在發生的事」. This badge earns it
 * literally: `useInflightJobCount()` returns the number of jobs the backend
 * says are running right now, the badge is not rendered at all below 1, and it
 * stops the instant the last job lands. If it is moving, work IS happening —
 * so the motion is a true statement, not decoration. Do not lift this pattern
 * onto a static count; that would make the app lie in a second channel, the
 * way a green 完成 badge on a running job lies in the first.
 *
 * WHAT breathes is deliberate: the accent WASH behind the digit, never the
 * digit. A number that fades is a number you have to wait to read, and this
 * badge exists to be read at a glance from across the room (the 「回來查」
 * posture). So the figure sits at full opacity on a halo that is alive.
 *
 * `motion-safe:` is belt-and-braces over the global reduced-motion net in
 * styles.css. Under reduced motion the badge is simply still — the count is
 * the information, the breath was only ever the emphasis.
 */
import { cn } from '../../lib/utils';

interface InFlightBadgeProps {
  /** Jobs running right now. The caller renders nothing when this is 0. */
  count: number;
  /** 'rail' = the 10px corner chip; 'row' = the 11px end-of-row pill. */
  variant: 'rail' | 'row';
  testId: string;
  className?: string;
}

export function InFlightBadge({ count, variant, testId, className }: InFlightBadgeProps) {
  const rail = variant === 'rail';
  return (
    // A one-cell GRID, not relative+absolute. Two callers pass their own
    // positioning (`absolute right-0.5 top-0.5` on the rail, `-right-2 -top-1`
    // on the mobile tab), and a `relative` in the base class list would collide
    // with their `absolute` — a conflict Tailwind resolves by stylesheet order,
    // not by class order, so it would break silently and only in one shell.
    // Stacking in a grid cell needs no position at all, and the cell sizes
    // itself to the padded digit so the halo can never be the wrong size.
    <span
      aria-hidden="true"
      data-testid={testId}
      className={cn(
        'grid rounded-full font-mono leading-none text-[var(--accent-text)]',
        rail ? 'text-[10px]' : 'text-[11px]',
        className
      )}
    >
      <span
        aria-hidden="true"
        className="col-start-1 row-start-1 rounded-full bg-[var(--accent-subtle)] motion-safe:animate-breathe"
      />
      <span
        className={cn('col-start-1 row-start-1 text-center', rail ? 'px-1 py-px' : 'px-1.5 py-0.5')}
      >
        {count}
      </span>
    </span>
  );
}
