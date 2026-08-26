// Design ref: ux-design.pen Screen 4 Detail Panel Desktop (RgSxQ)
import { cn } from '../../lib/utils';

export type AvailabilityBadgeVariant = 'owned' | 'requested';

export interface AvailabilityBadgeProps {
  variant: AvailabilityBadgeVariant;
  className?: string;
}

// Critique R3 remake (2026-08-26) — the badge had three diseases in one:
// ① 已有 wore SOLID running-green, but ownership is a static FACT — 固定詞彙
//   forbids green for anything that is not happening right now, and this was
//   the page's most frequent status colour (12×). 已有 is a CLASSIFICATION,
//   so it wears the neutral scrim recipe (same as the type badge).
// ② 已請求 is genuinely amber grammar (asked-and-not-yet-happening) → the
//   V2 badge recipe: warning tint + AA text over an opaque backing.
// ③ Both were 10px functional text (detector floor 11px) and square-cornered
//   while every V2 badge is a pill — the Shapes amendment makes the pill the
//   lawful shape for poster-overlay micro-elements.
const variantClasses: Record<AvailabilityBadgeVariant, string> = {
  owned: 'bg-[var(--overlay-scrim)] text-[var(--text-primary)]',
  requested: 'bg-[var(--bg-secondary)]',
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
  // NO role="status"/aria-live (critique R3): a grid resolving ownership used
  // to fire 12 simultaneous「已有」announcements — SR spam for a static fact.
  if (variant === 'requested') {
    return (
      <span
        data-testid={variantTestIds[variant]}
        className={cn('rounded-full', variantClasses[variant], className)}
      >
        <span className="block rounded-full bg-[var(--warning-tint)] px-1.5 py-0.5 text-[11px] font-medium text-[var(--warning-text)]">
          {variantLabels[variant]}
        </span>
      </span>
    );
  }
  return (
    <span
      data-testid={variantTestIds[variant]}
      className={cn(
        'rounded-full px-1.5 py-0.5 text-[11px] font-medium',
        variantClasses[variant],
        className
      )}
    >
      {variantLabels[variant]}
    </span>
  );
}
