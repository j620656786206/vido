// Design ref: ux-design.pen Screen 4 Detail Panel Desktop (RgSxQ)
import { cn } from '../../lib/utils';

export type AvailabilityBadgeVariant = 'owned' | 'requested';

export interface AvailabilityBadgeProps {
  variant: AvailabilityBadgeVariant;
  className?: string;
}

// Visually consistent with the sibling badges inside PosterCard (new-badge,
// metadata-source, type) — same font size, padding, and rounding. The colour
// layer uses CSS variables from the design system so a future light-theme
// swap continues to work.
// 已有 wore text-white on the jade fill — measured 2.17:1 (critique R2 P0,
// the last nest of the token-debt class). --text-on-accent is the ink cut
// for exactly this job: 8.36:1 on --success. requested keeps its dark ink
// (5.71:1, passes).
const variantClasses: Record<AvailabilityBadgeVariant, string> = {
  owned: 'bg-[var(--success)] text-[var(--text-on-accent)]',
  requested: 'bg-[var(--warning)] text-[var(--text-on-accent)]',
};

const variantLabels: Record<AvailabilityBadgeVariant, string> = {
  owned: '已有',
  requested: '已請求',
};

const variantTestIds: Record<AvailabilityBadgeVariant, string> = {
  owned: 'availability-badge-owned',
  requested: 'availability-badge-requested',
};

/**
 * Homepage availability badge rendered on poster cards to signal that the user
 * either already owns a title (已有) or has requested it (已請求). Story 10-4
 * (P2-006). The requested state is stubbed to false until the request system
 * lands in Phase 3 — see Story 10-4 AC #5.
 */
export function AvailabilityBadge({ variant, className }: AvailabilityBadgeProps) {
  return (
    <span
      data-testid={variantTestIds[variant]}
      className={cn(
        'rounded px-1.5 py-0.5 text-[10px] font-bold',
        variantClasses[variant],
        className
      )}
      // Badge appears async after the ownership POST resolves — announce it
      // politely so screen-reader users hear the change without interrupting.
      role="status"
      aria-live="polite"
      aria-label={variantLabels[variant]}
    >
      {variantLabels[variant]}
    </span>
  );
}
