// Implements: <utility — no .pen counterpart>
import { cn } from '@/lib/utils';

/**
 * A determinate spinner: the same 16px ring the Loader2 spinner draws, filled
 * to `value` percent (disc-2026-11-online-sub-download-progress). Alexyu's
 * ruling: show how far a download has got IN the spinner, never as an extra
 * line of text — so the number lives only in `label` for screen readers.
 */
export function ProgressRing({
  value,
  label,
  className,
}: {
  /** 0–100. */
  value: number;
  /** Accessible name, e.g. 「下載中 40%」. */
  label: string;
  className?: string;
}) {
  const r = 9;
  const c = 2 * Math.PI * r;
  const pct = Math.min(100, Math.max(0, value));
  return (
    <svg
      viewBox="0 0 24 24"
      role="progressbar"
      aria-label={label}
      aria-valuemin={0}
      aria-valuemax={100}
      aria-valuenow={Math.round(pct)}
      className={cn('h-4 w-4 -rotate-90', className)}
    >
      <circle
        cx="12"
        cy="12"
        r={r}
        fill="none"
        stroke="currentColor"
        strokeOpacity={0.25}
        strokeWidth={2.5}
      />
      <circle
        cx="12"
        cy="12"
        r={r}
        fill="none"
        stroke="currentColor"
        strokeWidth={2.5}
        strokeLinecap="round"
        strokeDasharray={c}
        strokeDashoffset={c * (1 - pct / 100)}
        className="transition-[stroke-dashoffset] duration-[var(--motion-state)] ease-[var(--ease-settle)] motion-reduce:transition-none"
      />
    </svg>
  );
}
