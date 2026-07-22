// Design ref: ux-design.pen Screen Navigation Shell v2 (CLo58)
/**
 * The application shell (UX Redesign Phase 2 — UX2-1; sole chassis since
 * ux3-cutover-4 deleted the legacy shell and the `new_shell_enabled` flag).
 * Mounted unconditionally by `__root.tsx`. Composes the collapsible desktop
 * sidebar + a slim top header (search) + the mobile bottom tab bar, and OWNS
 * the global `ScanProgress` overlay (AC #6 — moved in from __root so it is no
 * longer an unowned corner).
 */
import { useCallback, useEffect, useState } from 'react';
import { Link } from '@tanstack/react-router';
import { ArrowLeft, Search } from 'lucide-react';
import { AppSidebar } from './AppSidebar';
import { MobileTabBar } from './MobileTabBar';
import { InstantSearchBar } from '../search/InstantSearchBar';
import { ScanProgress } from '../scanner/ScanProgress';

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

  // The top header's bottom divider is chrome-only: hidden at rest so it never
  // floats under a sparse bar, shown once content scrolls under the sticky header
  // (applies to both the mobile and desktop header — it's one element).
  const [scrolled, setScrolled] = useState(false);
  useEffect(() => {
    const onScroll = () => setScrolled(window.scrollY > 4);
    onScroll();
    window.addEventListener('scroll', onScroll, { passive: true });
    return () => window.removeEventListener('scroll', onScroll);
  }, []);

  return (
    <div className="flex min-h-screen bg-[var(--bg-primary)]" data-testid="app-shell-v2">
      {/* Desktop sidebar (display:contents wrapper so the sticky aside is a direct flex child) */}
      <div className="hidden sm:contents">
        <AppSidebar collapsed={collapsed} onToggleCollapse={toggleCollapse} />
      </div>

      {/* Content column */}
      <div className="flex min-w-0 flex-1 flex-col">
        <header
          data-scrolled={scrolled}
          className={`sticky top-0 z-30 flex h-14 items-center gap-3 border-b bg-[var(--bg-primary)]/95 px-4 backdrop-blur-sm transition-colors ${
            scrolled ? 'border-[var(--border-subtle)]' : 'border-transparent'
          }`}
        >
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
          {children}
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

      {/* Global overlay owned by the shell (AC #6) */}
      <ScanProgress />
    </div>
  );
}
