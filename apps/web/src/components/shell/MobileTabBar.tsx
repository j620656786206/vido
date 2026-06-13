// Implements: Component/MobileTabBar (u91vZI) + Component/MobileTabItem (S86VM)
/**
 * Mobile bottom tab bar (UX Redesign Phase 2 — UX2-1, ADR D1-b / §6.3). Four
 * thumb-reach destinations + a 5th "更多" slot opening the More sheet. Active state
 * from TanStack Router (`data-status`). 84px tall with safe-area inset; each tab's
 * hit area is full-height × equal-width (≥44px). Desktop-hidden (`sm:hidden`).
 */
import { useState } from 'react';
import { Link } from '@tanstack/react-router';
import { MoreHorizontal } from 'lucide-react';
import { MOBILE_TABS } from './navModel';
import { MobileMoreSheet } from './MobileMoreSheet';

export function MobileTabBar() {
  const [moreOpen, setMoreOpen] = useState(false);

  return (
    <>
      <nav
        aria-label="主要導航"
        data-testid="mobile-tab-bar"
        className="fixed inset-x-0 bottom-0 z-40 flex h-[84px] border-t border-[var(--border-subtle)] bg-[var(--bg-secondary)] pb-[env(safe-area-inset-bottom)] sm:hidden"
      >
        {MOBILE_TABS.map((d) => {
          const Icon = d.icon;
          return (
            <Link
              key={d.key}
              to={d.to}
              search={d.search}
              activeOptions={{ exact: !!d.exact, includeSearch: false }}
              data-testid={`nav-${d.key}`}
              aria-label={d.label}
              className="group/tab flex flex-1 flex-col items-center justify-center gap-1 pt-1 text-[var(--text-muted)] transition-colors data-[status=active]:text-[var(--accent-primary)]"
            >
              <Icon className="h-6 w-6" aria-hidden="true" />
              <span className="text-[11px] font-medium group-data-[status=active]/tab:font-bold">
                {d.label}
              </span>
            </Link>
          );
        })}
        <button
          type="button"
          onClick={() => setMoreOpen(true)}
          aria-label="更多"
          data-testid="nav-more"
          className="flex flex-1 flex-col items-center justify-center gap-1 pt-1 text-[var(--text-muted)] transition-colors hover:text-[var(--text-primary)]"
        >
          <MoreHorizontal className="h-6 w-6" aria-hidden="true" />
          <span className="text-[11px] font-medium">更多</span>
        </button>
      </nav>
      <MobileMoreSheet open={moreOpen} onOpenChange={setMoreOpen} />
    </>
  );
}
