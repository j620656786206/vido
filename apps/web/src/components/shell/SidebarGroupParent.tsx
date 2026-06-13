// Implements: Component/SidebarGroupParent (imFBW)
/**
 * Two-level sidebar group parent (UX Redesign Phase 2 — UX2-1, ADR D2 / §6.1).
 * The label is itself a Link to the merged view (媒體庫 → /library); the chevron
 * toggles the children (電影/影集). The parent shows a *subtler* active-ancestor
 * treatment (label weight + accent icon, no full wash) when a child route is
 * active, so the leaf's `accent-subtle` wash stays the stronger signal. Active
 * matching is TanStack Router's (`data-status`), not hand-rolled.
 */
import { useState } from 'react';
import { Link } from '@tanstack/react-router';
import { ChevronDown } from 'lucide-react';
import type { LucideIcon } from 'lucide-react';
import { cn } from '../../lib/utils';

interface SidebarGroupParentProps {
  to: string;
  label: string;
  icon: LucideIcon;
  navKey: string;
  defaultOpen?: boolean;
  children: React.ReactNode;
}

export function SidebarGroupParent({
  to,
  label,
  icon: Icon,
  navKey,
  defaultOpen = true,
  children,
}: SidebarGroupParentProps) {
  const [open, setOpen] = useState(defaultOpen);

  return (
    <div>
      <div className="flex items-center gap-1">
        <Link
          to={to}
          activeOptions={{ exact: false, includeSearch: false }}
          data-testid={`nav-${navKey}`}
          aria-label={label}
          className="group/parent flex min-h-[44px] flex-1 items-center gap-3 rounded-[var(--radius-md)] px-2.5 py-2 text-sm font-medium text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)] data-[status=active]:font-semibold data-[status=active]:text-[var(--text-primary)]"
        >
          <Icon
            className="h-[18px] w-[18px] shrink-0 text-[var(--text-muted)] transition-colors group-hover/parent:text-[var(--text-secondary)] group-data-[status=active]/parent:text-[var(--accent-hover)]"
            aria-hidden="true"
          />
          <span className="truncate">{label}</span>
        </Link>
        <button
          type="button"
          onClick={() => setOpen((v) => !v)}
          aria-expanded={open}
          aria-label={open ? `收合${label}` : `展開${label}`}
          data-testid={`nav-${navKey}-toggle`}
          className="flex h-11 w-9 items-center justify-center rounded-[var(--radius-md)] text-[var(--text-muted)] transition-colors hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)]"
        >
          <ChevronDown
            className={cn('h-4 w-4 transition-transform', open && 'rotate-180')}
            aria-hidden="true"
          />
        </button>
      </div>
      {open && (
        <div role="group" aria-label={label} className="mt-0.5 space-y-0.5">
          {children}
        </div>
      )}
    </div>
  );
}
