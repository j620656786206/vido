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
    <div className="flex min-h-screen flex-col bg-slate-900" data-testid="app-shell">
      {/* Header */}
      <header className="bg-slate-900">
        <div className="mx-auto max-w-7xl px-4">
          {/* Top row: logo, search, settings */}
          <div className="flex h-12 items-center gap-4">
            {/* Logo */}
            <Link
              to="/"
              className="shrink-0 text-lg font-bold text-blue-400"
              data-testid="app-logo"
            >
              vido
            </Link>

            {/* Search bar — visible on desktop, hidden on mobile */}
            <form
              onSubmit={handleSearchSubmit}
              className="relative mx-auto w-full max-w-sm max-[639px]:hidden"
              data-testid="search-form"
            >
              <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-500" />
              <input
                type="text"
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder="搜尋電影或影集..."
                className="w-full rounded-full border border-slate-700 bg-slate-800/80 py-1.5 pl-9 pr-4 text-sm text-slate-200 placeholder-slate-500 transition-colors focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                autoComplete="off"
                data-testid="global-search-input"
              />
            </form>

            {/* Right icons */}
            <div className="ml-auto flex shrink-0 items-center gap-1">
              {/* Mobile search toggle — hidden on desktop */}
              <button
                type="button"
                onClick={() => setMobileSearchOpen(!mobileSearchOpen)}
                className="rounded-lg p-2 text-slate-400 transition-colors hover:text-slate-200 min-[640px]:hidden"
                aria-label="搜尋"
                data-testid="mobile-search-toggle"
              >
                <Search className="h-5 w-5" />
              </button>

              {/* Settings gear */}
              <Link
                to="/settings/qbittorrent"
                className="rounded-lg p-2 text-slate-400 transition-colors hover:text-slate-200"
                aria-label="設定"
                data-testid="settings-link"
              >
                <Settings className="h-5 w-5" />
              </Link>
            </div>
          </div>

          {/* Mobile search bar (expandable) */}
          {mobileSearchOpen && (
            <form onSubmit={handleSearchSubmit} className="pb-3 min-[640px]:hidden">
              <div className="relative">
                <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-500" />
                <input
                  type="text"
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                  placeholder="搜尋電影或影集..."
                  className="w-full rounded-full border border-slate-700 bg-slate-800/80 py-1.5 pl-9 pr-4 text-sm text-slate-200 placeholder-slate-500 transition-colors focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
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
      <main className="flex-1 bg-slate-900">{children}</main>
    </div>
  );
}
