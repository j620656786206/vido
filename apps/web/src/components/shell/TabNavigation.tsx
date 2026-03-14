import { Link, useRouterState } from '@tanstack/react-router';
import { cn } from '../../lib/utils';

interface NavTab {
  label: string;
  to: string;
  matchPaths: string[];
}

const TABS: NavTab[] = [
  { label: '媒體庫', to: '/library', matchPaths: ['/library'] },
  { label: '下載中', to: '/downloads', matchPaths: ['/downloads'] },
  { label: '待處理', to: '/pending', matchPaths: ['/pending'] },
  { label: '設定', to: '/settings/qbittorrent', matchPaths: ['/settings'] },
];

export function TabNavigation() {
  const routerState = useRouterState();
  const currentPath = routerState.location.pathname;

  return (
    <nav
      className="flex gap-1 overflow-x-auto scrollbar-none"
      aria-label="主要導航"
      data-testid="tab-navigation"
    >
      {TABS.map((tab) => {
        const isActive = tab.matchPaths.some((path) => currentPath.startsWith(path));

        return (
          <Link
            key={tab.to}
            to={tab.to}
            className={cn(
              'relative shrink-0 px-3 py-2.5 text-sm font-medium transition-colors',
              isActive ? 'text-blue-400' : 'text-slate-400 hover:text-slate-200'
            )}
            data-testid={`tab-${tab.label}`}
          >
            {tab.label}
            {isActive && (
              <span
                className="absolute bottom-0 left-0 right-0 h-0.5 rounded-full bg-blue-400"
                data-testid="active-tab-indicator"
              />
            )}
          </Link>
        );
      })}
    </nav>
  );
}
