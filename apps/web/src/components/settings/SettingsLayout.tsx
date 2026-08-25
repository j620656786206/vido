// Design ref: ux-design.pen Screen C4-D (6UCtX) · C4-M (2H4OM)
import { Link, useRouterState } from '@tanstack/react-router';
import {
  Plug,
  Database,
  FileText,
  Activity,
  HardDrive,
  ArrowUpDown,
  Gauge,
  ScanLine,
  LayoutGrid,
  KeyRound,
} from 'lucide-react';
import type { LucideIcon } from 'lucide-react';
import { cn } from '../../lib/utils';

interface SettingsCategory {
  key: string;
  label: string;
  shortLabel: string;
  icon: LucideIcon;
  to: string;
  enabled?: boolean;
}

const SETTINGS_CATEGORIES: SettingsCategory[] = [
  {
    key: 'connection',
    label: '連線設定',
    shortLabel: '連線',
    icon: Plug,
    to: '/settings/connection',
  },
  {
    // Story sub-2-1b (FR25) — sits next to 連線設定 because both configure how
    // Vido reaches an external service. Without this entry the route would be
    // orphaned: /settings/ redirects to /settings/connection, so the sidebar IS
    // the settings index.
    key: 'keys',
    label: '金鑰設定',
    shortLabel: '金鑰',
    icon: KeyRound,
    to: '/settings/keys',
  },
  {
    key: 'scanner',
    label: '媒體庫掃描',
    shortLabel: '掃描',
    icon: ScanLine,
    to: '/settings/scanner',
  },
  {
    key: 'homepage',
    label: '自訂首頁',
    shortLabel: '首頁',
    icon: LayoutGrid,
    to: '/settings/homepage',
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
    enabled: false,
  },
  {
    key: 'performance',
    label: '效能監控',
    shortLabel: '效能',
    icon: Gauge,
    to: '/settings/performance',
    enabled: false,
  },
];

/**
 * Shown on categories that are routed but not built. Chinese, because it is the
 * only string in this chrome a 繁中-first user would otherwise have to read in
 * English — and "Coming Soon" does not tell them whether it is unbuilt or broken.
 */
const UNAVAILABLE_BADGE = '尚未開放';
const UNAVAILABLE_REASON = '此功能尚未實作';

interface SettingsLayoutProps {
  children: React.ReactNode;
}

export function SettingsLayout({ children }: SettingsLayoutProps) {
  const routerState = useRouterState();
  const currentPath = routerState.location.pathname;

  return (
    <div className="flex w-full flex-col md:flex-row" data-testid="settings-layout">
      {/* Desktop sidebar */}
      <nav
        className="hidden w-56 shrink-0 border-r border-[var(--border-subtle)] md:block"
        aria-label="設定分類導航"
        data-testid="settings-sidebar"
      >
        <ul className="py-4">
          {SETTINGS_CATEGORIES.map((cat) => {
            const isActive = currentPath.startsWith(cat.to);
            const isEnabled = cat.enabled !== false;
            const Icon = cat.icon;
            return (
              <li key={cat.key}>
                {isEnabled ? (
                  <Link
                    to={cat.to}
                    aria-current={isActive ? 'page' : undefined}
                    className={cn(
                      'flex items-center gap-3 px-4 py-2.5 text-sm transition-colors',
                      // "You are here" used to be --accent-primary on --bg-tertiary:
                      // 3.04:1, below the 4.5:1 PRODUCT.md calls a hard gate, and
                      // BELOW the 3.55:1 that got --text-disabled rejected. The
                      // inactive rows passed at 7.47:1, so the one row that had to
                      // stand out was the only one you could not read. This is the
                      // recipe SidebarNavItem.tsx:80 already proves at 10.00:1.
                      // Note --accent-text alone is NOT enough here: 4.40:1 on
                      // --bg-tertiary, still short. The wash is what carries it.
                      isActive
                        ? 'bg-[var(--accent-subtle)] font-semibold text-[var(--text-primary)]'
                        : 'font-medium text-[var(--text-secondary)] hover:bg-[var(--bg-secondary)] hover:text-[var(--text-primary)]'
                    )}
                    data-testid={`settings-nav-${cat.key}`}
                  >
                    <Icon className="h-4 w-4 shrink-0" />
                    {cat.label}
                  </Link>
                ) : (
                  // Kept on screen with its reason attached, per the disabled rule.
                  // It was a plain <span title=…>: not focusable, not in the tab
                  // order, and the reason lived in a hover tooltip — so keyboard
                  // and touch users got a dead row with no explanation. role=link
                  // + aria-disabled + tabIndex keeps it reachable and announces
                  // why, without making it navigate.
                  <span
                    role="link"
                    aria-disabled="true"
                    tabIndex={0}
                    aria-label={`${cat.label}：${UNAVAILABLE_REASON}`}
                    className="flex cursor-not-allowed items-center gap-3 px-4 py-2.5 text-sm font-medium text-[var(--text-muted)]"
                    data-testid={`settings-nav-${cat.key}`}
                    title={UNAVAILABLE_REASON}
                  >
                    <Icon className="h-4 w-4 shrink-0" aria-hidden="true" />
                    {cat.label}
                    <span className="ml-auto rounded-[var(--radius-sm)] bg-[var(--bg-tertiary)] px-1.5 py-0.5 text-[11px] text-[var(--text-muted)]">
                      {UNAVAILABLE_BADGE}
                    </span>
                  </span>
                )}
              </li>
            );
          })}
        </ul>
      </nav>

      {/* Mobile horizontal tabs */}
      <nav
        className="overflow-x-auto border-b border-[var(--border-subtle)] md:hidden"
        aria-label="設定分類標籤"
        data-testid="settings-tabs"
      >
        <div className="flex gap-1 px-4 py-2">
          {SETTINGS_CATEGORIES.map((cat) => {
            const isActive = currentPath.startsWith(cat.to);
            const isEnabled = cat.enabled !== false;
            const Icon = cat.icon;
            return isEnabled ? (
              <Link
                key={cat.key}
                to={cat.to}
                aria-current={isActive ? 'page' : undefined}
                className={cn(
                  'flex shrink-0 items-center gap-1.5 rounded-full border border-transparent px-3 py-1.5 text-xs transition-colors',
                  // Same inversion as the desktop rail, worse for being 12px:
                  // --accent-primary on --bg-primary measured 4.26:1.
                  isActive
                    ? 'bg-[var(--accent-subtle)] font-semibold text-[var(--text-primary)]'
                    : 'font-medium text-[var(--text-secondary)] hover:bg-[var(--bg-secondary)] hover:text-[var(--text-primary)]'
                )}
                data-testid={`settings-tab-${cat.key}`}
              >
                <Icon className="h-3.5 w-3.5 shrink-0" />
                {cat.shortLabel}
              </Link>
            ) : (
              <span
                key={cat.key}
                role="link"
                aria-disabled="true"
                tabIndex={0}
                aria-label={`${cat.shortLabel}：${UNAVAILABLE_REASON}`}
                className="flex shrink-0 cursor-not-allowed items-center gap-1.5 rounded-full border border-transparent px-3 py-1.5 text-xs font-medium text-[var(--text-muted)]"
                data-testid={`settings-tab-${cat.key}`}
                title={UNAVAILABLE_REASON}
              >
                <Icon className="h-3.5 w-3.5 shrink-0" />
                {cat.shortLabel}
              </span>
            );
          })}
        </div>
      </nav>

      {/* Content area. The width cap lives HERE, not on the layout root: capping
          the root centred the sidebar too, detaching it from the app sidebar and
          leaving a dead vertical gap between the two navs. Left-aligned (no
          mx-auto) so all three panes read as one continuous left edge and the
          leftover width collects on the right as page margin. */}
      <div className="min-h-[calc(100vh-8rem)] flex-1 p-6" data-testid="settings-content">
        <div className="w-full max-w-5xl">{children}</div>
      </div>
    </div>
  );
}

export { SETTINGS_CATEGORIES };
export type { SettingsCategory };
