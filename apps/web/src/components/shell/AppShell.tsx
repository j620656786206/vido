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
      <header className="bg-slate-900/95 backdrop-blur-sm">
        <div className="mx-auto max-w-7xl px-4 sm:px-6">
          {/* Top row: 3-column grid — logo | search | settings */}
          <div className="grid h-12 grid-cols-[auto_1fr_auto] items-center gap-4">
            {/* Left: Logo */}
            <Link to="/" className="text-lg font-bold text-blue-400" data-testid="app-logo">
              vido
            </Link>

            {/* Center: Search bar (desktop) */}
            <div className="flex justify-center max-[639px]:hidden">
              <form
                onSubmit={handleSearchSubmit}
                className="relative w-full max-w-xs"
                data-testid="search-form"
              >
                <Search className="pointer-events-none absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-slate-500" />
                <input
                  type="text"
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                  placeholder="搜尋電影或影集..."
                  className="w-full rounded-full border border-slate-600/50 bg-slate-800/60 py-1.5 pl-9 pr-4 text-sm text-slate-200 placeholder-slate-500 transition-colors focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                  autoComplete="off"
                  data-testid="global-search-input"
                />
              </form>
            </div>

            {/* Center: spacer for mobile */}
            <div className="min-[640px]:hidden" />

            {/* Right: Icons */}
            <div className="flex items-center gap-0.5">
              {/* Mobile search toggle */}
              <button
                type="button"
                onClick={() => setMobileSearchOpen(!mobileSearchOpen)}
                className="rounded-lg p-1.5 text-slate-400 transition-colors hover:text-slate-200 min-[640px]:hidden"
                aria-label="搜尋"
                data-testid="mobile-search-toggle"
              >
                <Search className="h-5 w-5" />
              </button>

              {/* Settings gear */}
              <Link
                to="/settings/qbittorrent"
                className="rounded-lg p-1.5 text-slate-400 transition-colors hover:text-slate-200"
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
                <Search className="pointer-events-none absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-slate-500" />
                <input
                  type="text"
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                  placeholder="搜尋電影或影集..."
                  className="w-full rounded-full border border-slate-600/50 bg-slate-800/60 py-1.5 pl-9 pr-4 text-sm text-slate-200 placeholder-slate-500 transition-colors focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
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
