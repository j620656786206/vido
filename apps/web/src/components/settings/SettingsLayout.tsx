import { Link, useRouterState } from '@tanstack/react-router';
import { Plug, Database, FileText, Activity, HardDrive, ArrowUpDown, Gauge } from 'lucide-react';
import type { LucideIcon } from 'lucide-react';
import { cn } from '../../lib/utils';

interface SettingsCategory {
  key: string;
  label: string;
  shortLabel: string;
  icon: LucideIcon;
  to: string;
}

const SETTINGS_CATEGORIES: SettingsCategory[] = [
  {
    key: 'connection',
    label: '連線設定',
    shortLabel: '連線',
    icon: Plug,
    to: '/settings/connection',
  },
  { key: 'cache', label: '快取管理', shortLabel: '快取', icon: Database, to: '/settings/cache' },
  { key: 'logs', label: '系統日誌', shortLabel: '日誌', icon: FileText, to: '/settings/logs' },
  { key: 'status', label: '服務狀態', shortLabel: '狀態', icon: Activity, to: '/settings/status' },
  {
    key: 'backup',
    label: '備份與還原',
    shortLabel: '備份',
    icon: HardDrive,
    to: '/settings/backup',
  },
  {
    key: 'export',
    label: '匯出/匯入',
    shortLabel: '匯出',
    icon: ArrowUpDown,
    to: '/settings/export',
  },
  {
    key: 'performance',
    label: '效能監控',
    shortLabel: '效能',
    icon: Gauge,
    to: '/settings/performance',
  },
];

interface SettingsLayoutProps {
  children: React.ReactNode;
}

export function SettingsLayout({ children }: SettingsLayoutProps) {
  const routerState = useRouterState();
  const currentPath = routerState.location.pathname;

  return (
    <div className="mx-auto flex max-w-7xl flex-col md:flex-row" data-testid="settings-layout">
      {/* Desktop sidebar */}
      <nav
        className="hidden w-56 shrink-0 border-r border-slate-700 md:block"
        aria-label="設定分類導航"
        data-testid="settings-sidebar"
      >
        <ul className="py-4">
          {SETTINGS_CATEGORIES.map((cat) => {
            const isActive = currentPath.startsWith(cat.to);
            const Icon = cat.icon;
            return (
              <li key={cat.key}>
                <Link
                  to={cat.to}
                  className={cn(
                    'flex items-center gap-3 border-l-2 px-4 py-2.5 text-sm font-medium transition-colors',
                    isActive
                      ? 'border-blue-400 bg-slate-700 text-blue-400'
                      : 'border-transparent text-slate-400 hover:bg-slate-800 hover:text-slate-200'
                  )}
                  data-testid={`settings-nav-${cat.key}`}
                >
                  <Icon className="h-4 w-4 shrink-0" />
                  {cat.label}
                </Link>
              </li>
            );
          })}
        </ul>
      </nav>

      {/* Mobile horizontal tabs */}
      <nav
        className="overflow-x-auto border-b border-slate-700 md:hidden"
        aria-label="設定分類標籤"
        data-testid="settings-tabs"
      >
        <div className="flex gap-1 px-4 py-2">
          {SETTINGS_CATEGORIES.map((cat) => {
            const isActive = currentPath.startsWith(cat.to);
            const Icon = cat.icon;
            return (
              <Link
                key={cat.key}
                to={cat.to}
                className={cn(
                  'flex shrink-0 items-center gap-1.5 rounded-full px-3 py-1.5 text-xs font-medium transition-colors',
                  isActive
                    ? 'border border-blue-400 text-blue-400'
                    : 'border border-transparent text-slate-400 hover:bg-slate-800 hover:text-slate-200'
                )}
                data-testid={`settings-tab-${cat.key}`}
              >
                <Icon className="h-3.5 w-3.5 shrink-0" />
                {cat.shortLabel}
              </Link>
            );
          })}
        </div>
      </nav>

      {/* Content area */}
      <div className="min-h-[calc(100vh-8rem)] flex-1 p-6" data-testid="settings-content">
        {children}
      </div>
    </div>
  );
}

export { SETTINGS_CATEGORIES };
export type { SettingsCategory };
