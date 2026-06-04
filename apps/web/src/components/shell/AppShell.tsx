// Implements: <utility — no .pen counterpart>
import { useState } from 'react';
import { Link } from '@tanstack/react-router';
import { ArrowLeft, Search, Settings } from 'lucide-react';
import { TabNavigation } from './TabNavigation';
import { InstantSearchBar } from '../search/InstantSearchBar';

interface AppShellProps {
  children: React.ReactNode;
}

export function AppShell({ children }: AppShellProps) {
  const [mobileSearchOpen, setMobileSearchOpen] = useState(false);

  return (
    <div className="flex min-h-screen flex-col" data-testid="app-shell">
      <header className="sticky top-0 z-50 bg-[var(--bg-primary)]/95 backdrop-blur-sm">
        <div className="mx-auto max-w-7xl px-4 sm:px-6">
          {/* Row 1: logo | search | gear */}
          <div className="flex h-12 items-center">
            {/* Logo */}
            <Link
              to="/"
              className="mr-6 shrink-0 text-lg font-bold text-[var(--accent-primary)]"
              data-testid="app-logo"
            >
              vido
            </Link>

            {/* Desktop: instant search bar centered via flex-1 wrappers */}
            <div className="hidden flex-1 justify-center sm:flex">
              <InstantSearchBar variant="desktop" className="w-64" />
            </div>

            {/* Right: icons — ml-auto on mobile, natural position on desktop */}
            <div className="ml-auto flex shrink-0 items-center gap-0.5 sm:ml-0">
              {/* Mobile search toggle */}
              <button
                type="button"
                onClick={() => setMobileSearchOpen(true)}
                className="rounded-lg p-1.5 text-[var(--text-secondary)] transition-colors hover:text-[var(--text-primary)] sm:hidden"
                aria-label="搜尋"
                data-testid="mobile-search-toggle"
              >
                <Search className="h-5 w-5" />
              </button>

              {/* Settings gear */}
              <Link
                to="/settings"
                className="rounded-lg p-1.5 text-[var(--text-secondary)] transition-colors hover:text-[var(--text-primary)]"
                aria-label="設定"
                data-testid="settings-link"
              >
                <Settings className="h-5 w-5" />
              </Link>
            </div>
          </div>

          {/* Row 2: Tab navigation */}
          <TabNavigation />
        </div>
      </header>

      {/* Mobile: full-screen dedicated search view (AC #5) */}
      {mobileSearchOpen && (
        <div
          className="fixed inset-0 z-[60] flex flex-col bg-[var(--bg-primary)] sm:hidden"
          data-testid="mobile-search-overlay"
        >
          <div className="flex items-center gap-2 border-b border-[var(--border-subtle)] px-4 py-3">
            <button
              type="button"
              onClick={() => setMobileSearchOpen(false)}
              className="rounded-lg p-1 text-[var(--text-secondary)] transition-colors hover:text-[var(--text-primary)]"
              aria-label="返回"
              data-testid="mobile-search-close"
            >
              <ArrowLeft className="h-5 w-5" />
            </button>
            <span className="text-sm font-medium text-[var(--text-primary)]">搜尋</span>
          </div>
          <div className="flex-1 overflow-hidden px-4 py-3">
            <InstantSearchBar
              variant="mobile"
              autoFocus
              onClose={() => setMobileSearchOpen(false)}
            />
          </div>
        </div>
      )}

      <main className="flex-1">{children}</main>
    </div>
  );
}
