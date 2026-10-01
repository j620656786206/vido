// Design ref: ux-design.pen Screen K1-D-v2 (kMeWS) · K5-D-v2 (xgYKA)
import type { ReactNode } from 'react';

/**
 * One activity-hub section: a bold title, an optional count pill and an
 * optional trailing note (K5 draws the month —「10 月」— at the header's right
 * edge), over the section's rows. Shared by ActivityHub's four sections and
 * the sub-7-6c spend card so a new section cannot drift from the others.
 */
export function ActivitySectionShell({
  title,
  count,
  trailing,
  children,
}: {
  title: string;
  count?: number;
  trailing?: ReactNode;
  children: ReactNode;
}) {
  return (
    <section className="flex flex-col gap-3">
      <div className="flex items-center gap-2">
        <h2 className="text-base font-bold text-[var(--text-primary)]">{title}</h2>
        {typeof count === 'number' && count > 0 && (
          <span className="rounded-full bg-[var(--accent-tint)] px-2.5 py-0.5 font-mono text-xs text-[var(--accent-text)]">
            {count}
          </span>
        )}
        {trailing != null && (
          <span className="ml-auto text-sm text-[var(--text-muted)]">{trailing}</span>
        )}
      </div>
      {children}
    </section>
  );
}
