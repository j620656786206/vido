import { useState } from 'react';
import { Link, useNavigate } from '@tanstack/react-router';
import { Search, Settings } from 'lucide-react';
import { TabNavigation } from './TabNavigation';

interface AppShellProps {
  children: React.ReactNode;
}

export function AppShell({ children }: AppShellProps) {
  const navigate = useNavigate();
  const [query, setQuery] = useState('');
  const [mobileSearchOpen, setMobileSearchOpen] = useState(false);

  const handleSearchSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (query.trim()) {
      navigate({ to: '/search', search: { q: query.trim() } });
      setMobileSearchOpen(false);
    }
  };

  return (
    <div className="flex min-h-screen flex-col" data-testid="app-shell">
      <header className="sticky top-0 z-50 bg-slate-900/95 backdrop-blur-sm">
        <div className="mx-auto max-w-7xl px-4 sm:px-6">
          {/* Row 1: logo | search | gear */}
          <div className="flex h-12 items-center">
            {/* Logo */}
            <Link
              to="/"
              className="mr-6 shrink-0 text-lg font-bold text-blue-400"
              data-testid="app-logo"
            >
              vido
            </Link>

            {/* Desktop: search bar centered via flex-1 wrappers */}
            <div className="hidden flex-1 justify-center sm:flex">
              <form
                onSubmit={handleSearchSubmit}
                className="relative w-64"
                data-testid="search-form"
              >
                <Search className="pointer-events-none absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-slate-500" />
                <input
                  type="text"
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                  placeholder="搜尋媒體庫..."
                  className="w-full rounded-full border border-slate-600/50 bg-slate-800/60 py-1.5 pl-9 pr-4 text-sm text-slate-200 placeholder-slate-500 transition-colors focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                  autoComplete="off"
                  data-testid="global-search-input"
                />
              </form>
            </div>

            {/* Right: icons — ml-auto on mobile, natural position on desktop */}
            <div className="ml-auto flex shrink-0 items-center gap-0.5 sm:ml-0">
              {/* Mobile search toggle */}
              <button
                type="button"
                onClick={() => setMobileSearchOpen(!mobileSearchOpen)}
                className="rounded-lg p-1.5 text-slate-400 transition-colors hover:text-slate-200 sm:hidden"
                aria-label="搜尋"
                data-testid="mobile-search-toggle"
              >
                <Search className="h-5 w-5" />
              </button>

              {/* Settings gear */}
              <Link
                to="/settings"
                className="rounded-lg p-1.5 text-slate-400 transition-colors hover:text-slate-200"
                aria-label="設定"
                data-testid="settings-link"
              >
                <Settings className="h-5 w-5" />
              </Link>
            </div>
          </div>

          {/* Mobile search (expandable) */}
          {mobileSearchOpen && (
            <form onSubmit={handleSearchSubmit} className="pb-3 sm:hidden">
              <div className="relative">
                <Search className="pointer-events-none absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-slate-500" />
                <input
                  type="text"
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                  placeholder="搜尋媒體庫..."
                  className="w-full rounded-full border border-slate-600/50 bg-slate-800/60 py-1.5 pl-9 pr-4 text-sm text-slate-200 placeholder-slate-500 transition-colors focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                  autoComplete="off"
                  autoFocus
                  data-testid="mobile-search-input"
                />
              </div>
            </form>
          )}

          {/* Row 2: Tab navigation */}
          <TabNavigation />
        </div>
      </header>

      <main className="flex-1">{children}</main>
    </div>
  );
}
