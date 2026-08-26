// Design ref: ux-design.pen Screen C4-D (6UCtX) · C4-M (2H4OM)
// ⚠️ The .pen still shows the RETIRED vertical rail. feat-settings-tabs-ia reshapes
// this surface in code; the design file is brought back into line in the same story.
import { useEffect, useRef } from 'react';
import { Link, useRouterState } from '@tanstack/react-router';
import { cn } from '../../lib/utils';

/**
 * Shown on categories that are routed but not built. Chinese, because it is the
 * only string in this chrome a 繁中-first user would otherwise have to read in
 * English — and "Coming Soon" does not tell them whether it is unbuilt or broken.
 */
const UNAVAILABLE_BADGE = '尚未開放';
const UNAVAILABLE_REASON = '此功能尚未實作';

/**
 * The three clusters the flat list always had and never showed. Ten ungrouped
 * options sat well past the ≤4-per-decision-point line; each of these is 2–3.
 * `unavailable` is not a fourth topic — it is where routed-but-unbuilt entries
 * park, so they stop interrupting a real group.
 */
type SettingsGroup = 'appearance' | 'connection' | 'library' | 'maintenance' | 'unavailable';

// 外觀 leads: it is the only group that changes how everything else LOOKS, and
// it is the one setting a new user is most likely to want before anything is
// even connected (ADR D4-3 — Settings holds preferences, System holds ops).
const GROUP_ORDER: SettingsGroup[] = [
  'appearance',
  'connection',
  'library',
  'maintenance',
  'unavailable',
];

const GROUP_LABEL: Record<SettingsGroup, string> = {
  appearance: '外觀',
  connection: '連線',
  library: '媒體庫',
  maintenance: '維護',
  unavailable: '尚未開放',
};

interface SettingsCategory {
  key: string;
  label: string;
  to: string;
  group: SettingsGroup;
  enabled?: boolean;
}

/**
 * Order is MEANING, not insertion order. 服務狀態 moved up to sit with the things
 * it reports on, and 備份與還原 moved down to sit with the other maintenance
 * chores. Both moves are the point of the grouping.
 */
const SETTINGS_CATEGORIES: SettingsCategory[] = [
  {
    key: 'appearance',
    label: '外觀',
    to: '/settings/appearance',
    group: 'appearance',
  },
  {
    key: 'connection',
    label: '連線設定',
    to: '/settings/connection',
    group: 'connection',
  },
  {
    // Story sub-2-1b (FR25) — sits next to 連線設定 because both configure how
    // Vido reaches an external service.
    key: 'keys',
    label: '金鑰設定',
    to: '/settings/keys',
    group: 'connection',
  },
  {
    key: 'status',
    label: '服務狀態',
    to: '/settings/status',
    group: 'connection',
  },
  {
    key: 'scanner',
    label: '媒體庫掃描',
    to: '/settings/scanner',
    group: 'library',
  },
  {
    key: 'homepage',
    label: '自訂首頁',
    to: '/settings/homepage',
    group: 'library',
  },
  {
    key: 'cache',
    label: '快取管理',
    to: '/settings/cache',
    group: 'maintenance',
  },
  {
    key: 'logs',
    label: '系統日誌',
    to: '/settings/logs',
    group: 'maintenance',
  },
  {
    key: 'backup',
    label: '備份與還原',
    to: '/settings/backup',
    group: 'maintenance',
  },
  {
    // 匯出 shipped inside 備份與還原 long before this tab was un-parked — the
    // strip claiming「尚未實作」while a working exporter lived one tab away was
    // the chrome's one outright false statement (critique R1, fixed R4). 匯入
    // is still honestly pending, stated on the page itself.
    key: 'export',
    label: '匯出/匯入',
    to: '/settings/export',
    group: 'maintenance',
  },
  {
    key: 'performance',
    label: '效能監控',
    to: '/settings/performance',
    group: 'unavailable',
    enabled: false,
  },
];

interface SettingsLayoutProps {
  children: React.ReactNode;
}

export function SettingsLayout({ children }: SettingsLayoutProps) {
  const routerState = useRouterState();
  const currentPath = routerState.location.pathname;

  const stripRef = useRef<HTMLDivElement>(null);

  // The retired mobile strip hid five of ten categories behind a swipe with no
  // fade, no arrow and no clipped tab — so it read as complete and half the
  // settings IA was unreachable in practice. Scrolling the active tab into view
  // means you at least always start from where you are, at any width.
  useEffect(() => {
    const strip = stripRef.current;
    if (!strip) return;
    if (strip.scrollWidth <= strip.clientWidth) return;
    // The router owns "active"; read its marker rather than keeping a second
    // opinion in a ref.
    strip
      .querySelector('[data-status="active"]')
      ?.scrollIntoView({ inline: 'center', block: 'nearest' });
  }, [currentPath]);

  return (
    // The strip and the content share ONE centered container, so their left
    // edges stay glued to each other while spare width falls on BOTH sides.
    // (Alexyu, 2026-08-25, from an ultra-wide screenshot: the old
    // spare-width-all-on-the-right ruling reads lopsided past ~1600px. The
    // historical dead-gap bug this replaces was about centering the LAYOUT
    // ROOT while a second rail existed — the rail is a tab strip now, so
    // centering the container cannot detach anything.)
    <div className="min-h-[calc(100vh-8rem)] p-6" data-testid="settings-layout">
      {/* 1152px, not 1024: the ten-tab strip measures 1072px, and at 1440 the
          content pane is exactly 1152 — so this width is invisible at 1440 and
          only buys balance on ultra-wide screens. */}
      <div className="mx-auto w-full max-w-6xl">
        {/* Visually tabs, semantically NAVIGATION. These change route, so there are
          no tabpanels in this document and role="tablist" would promise a widget
          the DOM does not implement. A nav of links carrying aria-current is the
          honest markup for what actually happens. */}
        <nav aria-label="設定分類" data-testid="settings-tabs" className="relative">
          <div
            ref={stripRef}
            data-testid="settings-tabs-strip"
            className="flex items-center gap-1 overflow-x-auto pb-3 pr-10 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden"
          >
            {GROUP_ORDER.map((group, groupIndex) => {
              const items = SETTINGS_CATEGORIES.filter((c) => c.group === group);
              if (items.length === 0) return null;

              return (
                <div key={group} className="flex shrink-0 items-center gap-1">
                  {groupIndex > 0 && (
                    // Presentational only. The grouping reaches assistive tech
                    // through the accessible names below, never through a rule.
                    <span
                      aria-hidden="true"
                      data-testid={`settings-tabs-divider-${group}`}
                      className="mx-2 h-5 w-px shrink-0 bg-[var(--border-subtle)]"
                    />
                  )}
                  {items.map((cat) => {
                    const isEnabled = cat.enabled !== false;

                    if (!isEnabled) {
                      // Kept on screen with its reason attached, per the disabled
                      // rule — and reachable, which the old <span title=…> was not.
                      return (
                        <span
                          key={cat.key}
                          role="link"
                          aria-disabled="true"
                          tabIndex={0}
                          aria-label={`${cat.label}：${UNAVAILABLE_REASON}`}
                          title={UNAVAILABLE_REASON}
                          data-testid={`settings-tab-${cat.key}`}
                          className="flex min-h-[44px] shrink-0 cursor-not-allowed items-center gap-2 rounded-[var(--radius-md)] px-3 text-sm font-medium text-[var(--text-muted)]"
                        >
                          {cat.label}
                          <span className="rounded-[var(--radius-sm)] bg-[var(--bg-tertiary)] px-1.5 py-0.5 text-[11px]">
                            {UNAVAILABLE_BADGE}
                          </span>
                        </span>
                      );
                    }

                    return (
                      <Link
                        key={cat.key}
                        to={cat.to}
                        // Active state is TanStack Router's `data-status`, NOT a
                        // hand-rolled startsWith — the ADR in SidebarNavItem.tsx:5-6
                        // mandates this, and the old rail was the one nav that
                        // disagreed. `aria-current="page"` comes from Link for free;
                        // setting it here was dead code.
                        activeOptions={{ exact: false, includeSearch: false }}
                        // The group rides in the accessible name so a screen-reader
                        // user gets the same structure a sighted user reads off the
                        // dividers.
                        aria-label={`${GROUP_LABEL[cat.group]}：${cat.label}`}
                        data-testid={`settings-tab-${cat.key}`}
                        className={cn(
                          // 44px is the system minimum the retired 30px chips missed.
                          'flex min-h-[44px] shrink-0 items-center gap-2 rounded-[var(--radius-md)] px-3 text-sm font-medium transition-colors',
                          'text-[var(--text-secondary)] hover:bg-[var(--bg-secondary)] hover:text-[var(--text-primary)]',
                          'data-[status=active]:bg-[var(--accent-subtle)] data-[status=active]:font-semibold data-[status=active]:text-[var(--text-primary)]'
                        )}
                      >
                        {cat.label}
                      </Link>
                    );
                  })}
                </div>
              );
            })}
          </div>

          {/* The clipped tab has to LOOK clipped. Without this the strip ends flush
            at the container edge and reads as the whole list — which is exactly
            how five categories went missing on mobile. */}
          <span
            aria-hidden="true"
            data-testid="settings-tabs-fade"
            className="pointer-events-none absolute inset-y-0 right-0 w-10 bg-gradient-to-l from-[var(--bg-primary)] to-transparent"
          />
        </nav>

        {/* The strip gets the full column; the CONTENT keeps its measure. Removing
          the rail freed ~224px, and spending all of it on longer log lines and
          wider form rows would be a regression dressed as a win. */}
        <div className="border-t border-[var(--border-subtle)] pt-6" data-testid="settings-content">
          {children}
        </div>
      </div>
    </div>
  );
}

export { SETTINGS_CATEGORIES, GROUP_ORDER, GROUP_LABEL, UNAVAILABLE_BADGE, UNAVAILABLE_REASON };
export type { SettingsCategory, SettingsGroup };
