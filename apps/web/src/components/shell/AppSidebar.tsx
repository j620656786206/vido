// Implements: Component/Sidebar-Expanded (b7CqJ0) + Component/Sidebar-Rail (H7eXAK)
/**
 * Desktop sidebar (UX Redesign Phase 2 — UX2-1, ADR D1-a / §6.1–§6.2).
 * Collapsible: 240px expanded (icon + label) ↔ 64px icon-rail. The collapse
 * state is owned by the parent shell (so the content width budget recomputes)
 * and persisted there. Groups encode N3 (內容 above 任務). Library counts come
 * from the existing stats query, fail-soft (no count while loading/errored).
 * The whole sidebar is wrapped in one Tooltip provider so the rail's per-icon
 * tooltips share open-delay timing.
 */
import { Link } from '@tanstack/react-router';
import { PanelLeftClose, PanelLeftOpen } from 'lucide-react';
import { useLibraryStats } from '../../hooks/useLibrary';
import { TooltipProvider, Tooltip } from '../ui/Tooltip';
import { SidebarNavItem } from './SidebarNavItem';
import { SidebarGroupLabel } from './SidebarGroupLabel';
import { SidebarGroupParent } from './SidebarGroupParent';
import { SidebarFooter } from './SidebarFooter';
import { HOME, LIBRARY, MOVIES, TV, DISCOVER, DOWNLOADS, SETTINGS, RAIL_DESTS } from './navModel';

interface AppSidebarProps {
  collapsed: boolean;
  onToggleCollapse: () => void;
}

export function AppSidebar({ collapsed, onToggleCollapse }: AppSidebarProps) {
  const { data: stats } = useLibraryStats();

  return (
    <TooltipProvider>
      <aside
        data-testid="app-sidebar"
        data-collapsed={collapsed}
        aria-label="主要導航"
        className={`sticky top-0 flex h-screen shrink-0 flex-col border-r border-[var(--border-subtle)] bg-[var(--bg-secondary)] ${
          collapsed ? 'w-16' : 'w-60'
        }`}
      >
        {/* Header: logo + collapse toggle */}
        <div
          className={`flex h-14 items-center border-b border-[var(--border-subtle)] ${
            collapsed ? 'justify-center px-0' : 'px-3'
          }`}
        >
          {collapsed ? (
            <Link
              to="/"
              aria-label="vido 首頁"
              className="flex h-9 w-9 items-center justify-center rounded-[var(--radius-md)] text-lg font-bold text-[var(--accent-primary)] hover:bg-[var(--bg-tertiary)]"
            >
              V
            </Link>
          ) : (
            <Link to="/" aria-label="vido 首頁" className="flex flex-col leading-tight">
              <span className="text-lg font-bold text-[var(--accent-primary)]">vido</span>
              <span className="text-[11px] text-[var(--text-secondary)]">NAS 媒體庫</span>
            </Link>
          )}
          {!collapsed && (
            <button
              type="button"
              onClick={onToggleCollapse}
              aria-label="收合側邊欄"
              data-testid="sidebar-collapse-toggle"
              className="ml-auto flex h-9 w-9 items-center justify-center rounded-[var(--radius-md)] text-[var(--text-muted)] transition-colors hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)]"
            >
              <PanelLeftClose className="h-5 w-5" aria-hidden="true" />
            </button>
          )}
        </div>

        {/* Nav */}
        <nav className="flex-1 overflow-y-auto px-2 py-2">
          {collapsed ? (
            <div className="flex flex-col items-center gap-1">
              <Tooltip content="展開側邊欄">
                <button
                  type="button"
                  onClick={onToggleCollapse}
                  aria-label="展開側邊欄"
                  data-testid="sidebar-expand-toggle"
                  className="flex h-11 w-11 items-center justify-center rounded-[var(--radius-md)] text-[var(--text-muted)] transition-colors hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)]"
                >
                  <PanelLeftOpen className="h-5 w-5" aria-hidden="true" />
                </button>
              </Tooltip>
              {RAIL_DESTS.map((d) => (
                <SidebarNavItem
                  key={d.key}
                  collapsed
                  to={d.to}
                  search={d.search}
                  label={d.label}
                  icon={d.icon}
                  navKey={d.key}
                  exact={d.exact}
                />
              ))}
            </div>
          ) : (
            <>
              <SidebarGroupLabel>內容</SidebarGroupLabel>
              <SidebarNavItem
                to={HOME.to}
                label={HOME.label}
                icon={HOME.icon}
                navKey={HOME.key}
                exact
              />
              <SidebarGroupParent
                to={LIBRARY.to}
                label={LIBRARY.label}
                icon={LIBRARY.icon}
                navKey={LIBRARY.key}
              >
                <SidebarNavItem
                  indent
                  to={MOVIES.to}
                  search={MOVIES.search}
                  label={MOVIES.label}
                  icon={MOVIES.icon}
                  navKey={MOVIES.key}
                  count={stats?.movieCount}
                />
                <SidebarNavItem
                  indent
                  to={TV.to}
                  search={TV.search}
                  label={TV.label}
                  icon={TV.icon}
                  navKey={TV.key}
                  count={stats?.tvCount}
                />
              </SidebarGroupParent>
              <SidebarNavItem
                to={DISCOVER.to}
                label={DISCOVER.label}
                icon={DISCOVER.icon}
                navKey={DISCOVER.key}
              />

              <SidebarGroupLabel>任務</SidebarGroupLabel>
              <SidebarNavItem
                to={DOWNLOADS.to}
                label={DOWNLOADS.label}
                icon={DOWNLOADS.icon}
                navKey={DOWNLOADS.key}
              />
              <SidebarNavItem
                to={SETTINGS.to}
                label={SETTINGS.label}
                icon={SETTINGS.icon}
                navKey={SETTINGS.key}
              />
            </>
          )}
        </nav>

        <SidebarFooter collapsed={collapsed} />
      </aside>
    </TooltipProvider>
  );
}
