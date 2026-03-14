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
      className="-mb-px flex gap-6 overflow-x-auto border-b border-slate-800/50"
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
              'relative shrink-0 border-b-2 px-1 py-2 text-sm font-medium transition-colors',
              isActive
                ? 'border-blue-400 text-blue-400'
                : 'border-transparent text-slate-500 hover:text-slate-300'
            )}
            data-testid={`tab-${tab.label}`}
          >
            {tab.label}
            {isActive && (
              <span className="sr-only" data-testid="active-tab-indicator">
                (目前頁面)
              </span>
            )}
          </Link>
        );
      })}
    </nav>
  );
}
