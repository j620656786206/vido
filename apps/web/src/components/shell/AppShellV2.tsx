// Design ref: ux-design.pen Screen Navigation Shell v2 (CLo58)
/**
 * v2 application shell (UX Redesign Phase 2 — UX2-1). Selected by `__root.tsx`
 * when `new_shell_enabled` is ON (the single flag chokepoint, F4). Composes the
 * collapsible desktop sidebar + a slim top header (search) + the mobile bottom
 * tab bar, and OWNS the global `ScanProgress` overlay (AC #6 — moved in from
 * __root so it is no longer an unowned corner).
 *
 * Content routing: a migrated route opts into a full-bleed layout via
 * `staticData.shell === 'v2'`; every other (legacy) route renders inside
 * `LegacyContentContainer`, pixel-unchanged (strangler discipline P3 — this shell
 * never modifies legacy route content).
 */
import { useCallback, useState } from 'react';
import { Link, useRouterState } from '@tanstack/react-router';
import { ArrowLeft, Search } from 'lucide-react';
import { AppSidebar } from './AppSidebar';
import { MobileTabBar } from './MobileTabBar';
import { LegacyContentContainer } from './LegacyContentContainer';
import { InstantSearchBar } from '../search/InstantSearchBar';
import { ScanProgress } from '../scanner/ScanProgress';
import { ShellVersionProvider } from './shellVersion';

const COLLAPSE_KEY = 'vido:sidebar:collapsed';

function readCollapsed(): boolean {
  try {
    return localStorage.getItem(COLLAPSE_KEY) === 'true';
  } catch {
    return false;
  }
}

interface AppShellV2Props {
  children: React.ReactNode;
}

export function AppShellV2({ children }: AppShellV2Props) {
  const [collapsed, setCollapsed] = useState(readCollapsed);
  const [mobileSearchOpen, setMobileSearchOpen] = useState(false);

  const toggleCollapse = useCallback(() => {
    setCollapsed((v) => {
      const next = !v;
      try {
        localStorage.setItem(COLLAPSE_KEY, String(next));
      } catch {
        // ignore — localStorage unavailable
      }
      return next;
    });
  }, []);

  // A migrated route (UX2-2/UX2-3) marks itself with staticData.shell === 'v2' to
  // opt out of the legacy container and take the full content width.
  const isMigrated = useRouterState({
    select: (s) =>
      s.matches.some((m) => (m.staticData as { shell?: string } | undefined)?.shell === 'v2'),
  });

  return (
    <ShellVersionProvider value="v2">
      <div className="flex min-h-screen bg-[var(--bg-primary)]" data-testid="app-shell-v2">
        {/* Desktop sidebar (display:contents wrapper so the sticky aside is a direct flex child) */}
        <div className="hidden sm:contents">
          <AppSidebar collapsed={collapsed} onToggleCollapse={toggleCollapse} />
        </div>

        {/* Content column */}
        <div className="flex min-w-0 flex-1 flex-col">
          <header className="sticky top-0 z-30 flex h-14 items-center gap-3 border-b border-[var(--border-subtle)] bg-[var(--bg-primary)]/95 px-4 backdrop-blur-sm">
            {/* Mobile logo (the sidebar carries it on desktop) */}
            <Link
              to="/"
              aria-label="vido 首頁"
              className="text-lg font-bold text-[var(--accent-primary)] sm:hidden"
            >
              vido
            </Link>
            {/* Desktop omnisearch (pilot: existing InstantSearchBar, v2-styled container) */}
            <div className="ml-auto hidden sm:flex">
              <InstantSearchBar variant="desktop" className="w-72" />
            </div>
            {/* Mobile search toggle */}
            <button
              type="button"
              onClick={() => setMobileSearchOpen(true)}
              aria-label="搜尋"
              data-testid="mobile-search-toggle"
              className="ml-auto flex h-11 w-11 items-center justify-center rounded-[var(--radius-md)] text-[var(--text-secondary)] transition-colors hover:text-[var(--text-primary)] sm:hidden"
            >
              <Search className="h-5 w-5" aria-hidden="true" />
            </button>
          </header>

          <main className="flex flex-1 flex-col pb-[calc(84px+env(safe-area-inset-bottom))] sm:pb-0">
            {isMigrated ? children : <LegacyContentContainer>{children}</LegacyContentContainer>}
          </main>
        </div>

        {/* Mobile bottom tab bar */}
        <MobileTabBar />

        {/* Mobile full-screen search overlay */}
        {mobileSearchOpen && (
          <div
            className="fixed inset-0 z-[60] flex flex-col bg-[var(--bg-primary)] sm:hidden"
            data-testid="mobile-search-overlay"
          >
            <div className="flex items-center gap-2 border-b border-[var(--border-subtle)] px-4 py-3">
              <button
                type="button"
                onClick={() => setMobileSearchOpen(false)}
                aria-label="返回"
                data-testid="mobile-search-close"
                className="flex h-11 w-11 items-center justify-center rounded-[var(--radius-md)] text-[var(--text-secondary)] transition-colors hover:text-[var(--text-primary)]"
              >
                <ArrowLeft className="h-5 w-5" aria-hidden="true" />
              </button>
              <span className="text-sm font-medium text-[var(--text-primary)]">搜尋</span>
            </div>
            <div className="flex-1 overflow-hidden px-4 py-3">
              <InstantSearchBar
                variant="mobile"
                focusOnMount
                onClose={() => setMobileSearchOpen(false)}
              />
            </div>
          </div>
        )}

        {/* Global overlay owned by the v2 shell (AC #6) */}
        <ScanProgress />
      </div>
    </ShellVersionProvider>
  );
}
