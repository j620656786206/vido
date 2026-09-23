// Design ref: ux-design.pen Screen C4-D (6UCtX) · C4-M (2H4OM)
// page-header（L7yCT）— 每個設定分頁同一個
import type { ReactNode } from 'react';
import { cn } from '../../lib/utils';

interface SettingsPageHeaderProps {
  title: string;
  description?: ReactNode;
  /** Lets a region on the page name itself after the heading (aria-labelledby). */
  titleId?: string;
  testId?: string;
}

/**
 * The one page heading every settings tab shares.
 *
 * Ten routes used to hand-write the same h1 + p, and two tabs (外觀, 效能監控)
 * had drifted to an 18px h2 — so the heading changed size as you moved along
 * the strip. The phone rule is DESIGN.md §Responsive: titles step down one
 * rung (24 → 20), body copy does not (the description stays 14 at every
 * width). ≥640 this renders exactly what the hand-written headers did.
 */
export function SettingsPageHeader({
  title,
  description,
  titleId,
  testId,
}: SettingsPageHeaderProps) {
  return (
    <>
      <h1
        id={titleId}
        data-testid={testId}
        className={cn(
          'text-xl font-bold text-[var(--text-primary)] sm:text-2xl',
          description ? 'mb-2' : 'mb-6'
        )}
      >
        {title}
      </h1>
      {description && <p className="mb-6 text-sm text-[var(--text-secondary)]">{description}</p>}
    </>
  );
}
