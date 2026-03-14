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
      {/* Header */}
      <header className="border-b border-slate-800">
        <div className="mx-auto max-w-7xl px-4">
          {/* Top row: logo, search, settings */}
          <div className="flex h-14 items-center gap-4">
            {/* Logo */}
            <Link
              to="/"
              className="shrink-0 text-lg font-bold text-blue-400"
              data-testid="app-logo"
            >
              vido
            </Link>

            {/* Desktop search bar */}
            <form
              onSubmit={handleSearchSubmit}
              className="relative mx-auto hidden w-full max-w-md md:block"
            >
              <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
              <input
                type="text"
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder="搜尋電影或影集..."
                className="w-full rounded-full border border-slate-700 bg-slate-800 py-1.5 pl-10 pr-4 text-sm text-slate-100 placeholder-slate-500 transition-colors focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                autoComplete="off"
                data-testid="global-search-input"
              />
            </form>

            {/* Right icons */}
            <div className="ml-auto flex items-center gap-2 md:ml-0">
              {/* Mobile search toggle */}
              <button
                type="button"
                onClick={() => setMobileSearchOpen(!mobileSearchOpen)}
                className="rounded-lg p-2 text-slate-400 transition-colors hover:bg-slate-800 hover:text-slate-200 md:hidden"
                aria-label="搜尋"
                data-testid="mobile-search-toggle"
              >
                <Search className="h-5 w-5" />
              </button>

              {/* Settings gear */}
              <Link
                to="/settings/qbittorrent"
                className="rounded-lg p-2 text-slate-400 transition-colors hover:bg-slate-800 hover:text-slate-200"
                aria-label="設定"
                data-testid="settings-link"
              >
                <Settings className="h-5 w-5" />
              </Link>
            </div>
          </div>

          {/* Mobile search bar (expandable) */}
          {mobileSearchOpen && (
            <form onSubmit={handleSearchSubmit} className="pb-3 md:hidden">
              <div className="relative">
                <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
                <input
                  type="text"
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                  placeholder="搜尋電影或影集..."
                  className="w-full rounded-full border border-slate-700 bg-slate-800 py-1.5 pl-10 pr-4 text-sm text-slate-100 placeholder-slate-500 transition-colors focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                  autoComplete="off"
                  autoFocus
                  data-testid="mobile-search-input"
                />
              </div>
            </form>
          )}

          {/* Tab Navigation */}
          <TabNavigation />
        </div>
      </header>

      {/* Page Content */}
      <main className="flex-1">{children}</main>
    </div>
  );
}
