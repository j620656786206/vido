import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect } from 'vitest';
import {
  createRootRoute,
  createRoute,
  createRouter,
  createMemoryHistory,
  RouterProvider,
  Outlet,
} from '@tanstack/react-router';
import { SettingsLayout, SETTINGS_CATEGORIES } from './SettingsLayout';

function createTestRouter(initialPath = '/settings/connection') {
  const rootRoute = createRootRoute({
    component: () => React.createElement(Outlet),
  });

  const settingsRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/settings',
    component: () => React.createElement(SettingsLayout, null, React.createElement(Outlet)),
  });

  const connectionRoute = createRoute({
    getParentRoute: () => settingsRoute,
    path: '/connection',
    component: () => React.createElement('div', { 'data-testid': 'connection-page' }, 'Connection'),
  });

  const cacheRoute = createRoute({
    getParentRoute: () => settingsRoute,
    path: '/cache',
    component: () => React.createElement('div', { 'data-testid': 'cache-page' }, 'Cache'),
  });

  const logsRoute = createRoute({
    getParentRoute: () => settingsRoute,
    path: '/logs',
    component: () => React.createElement('div', null, 'Logs'),
  });

  const statusRoute = createRoute({
    getParentRoute: () => settingsRoute,
    path: '/status',
    component: () => React.createElement('div', null, 'Status'),
  });

  const backupRoute = createRoute({
    getParentRoute: () => settingsRoute,
    path: '/backup',
    component: () => React.createElement('div', null, 'Backup'),
  });

  const exportRoute = createRoute({
    getParentRoute: () => settingsRoute,
    path: '/export',
    component: () => React.createElement('div', null, 'Export'),
  });

  const performanceRoute = createRoute({
    getParentRoute: () => settingsRoute,
    path: '/performance',
    component: () => React.createElement('div', null, 'Performance'),
  });

  const routeTree = rootRoute.addChildren([
    settingsRoute.addChildren([
      connectionRoute,
      cacheRoute,
      logsRoute,
      statusRoute,
      backupRoute,
      exportRoute,
      performanceRoute,
    ]),
  ]);
  const memoryHistory = createMemoryHistory({ initialEntries: [initialPath] });

  return createRouter({ routeTree, history: memoryHistory });
}

function renderWithRouter(initialPath = '/settings/connection') {
  const router = createTestRouter(initialPath);
  return render(React.createElement(RouterProvider, { router }));
}

describe('SettingsLayout', () => {
  it('renders the settings layout container', async () => {
    renderWithRouter();
    expect(await screen.findByTestId('settings-layout')).toBeInTheDocument();
    expect(screen.getByTestId('settings-content')).toBeInTheDocument();
  });

  // The ruling: rail 2 is a PAGE TOOL, not a second navigation level. Settings
  // was the one place that broke it — it navigated while wearing the costume.
  it('has no vertical rail at all', async () => {
    renderWithRouter();
    await screen.findByTestId('settings-layout');
    expect(screen.queryByTestId('settings-sidebar')).toBeNull();
    // The old rail rendered its own item testids; nothing may still use them.
    for (const cat of SETTINGS_CATEGORIES) {
      expect(screen.queryByTestId(`settings-nav-${cat.key}`)).toBeNull();
    }
  });

  // One strip at every width — the desktop rail and the mobile chip strip used to
  // be two different components with two different contracts.
  it('renders exactly one navigation, not a desktop one and a mobile one', async () => {
    renderWithRouter();
    await screen.findByTestId('settings-layout');
    expect(screen.getAllByTestId('settings-tabs')).toHaveLength(1);
    for (const cat of SETTINGS_CATEGORIES) {
      expect(screen.getAllByTestId(`settings-tab-${cat.key}`)).toHaveLength(1);
    }
  });

  it('is navigation, not an ARIA tablist — these change route, not panel', async () => {
    renderWithRouter();
    const nav = await screen.findByTestId('settings-tabs');
    expect(nav.tagName).toBe('NAV');
    expect(nav).toHaveAttribute('aria-label', '設定分類');
    expect(nav.querySelector('[role="tablist"]')).toBeNull();
    expect(nav.querySelector('[role="tab"]')).toBeNull();
  });

  describe('grouping', () => {
    it('orders the categories by meaning, not by insertion', async () => {
      renderWithRouter();
      await screen.findByTestId('settings-tabs');
      const order = SETTINGS_CATEGORIES.map((c) => c.key);
      expect(order).toEqual([
        // 外觀 leads (日巡 light-theme story): the only group that changes how
        // everything else LOOKS, and the one a new user may want before
        // anything is connected.
        'appearance',
        'connection',
        'keys',
        'status', // moved UP to sit with what it reports on
        'scanner',
        'homepage',
        'cache',
        'logs',
        'backup', // moved DOWN to sit with the other chores
        'export',
        'performance',
      ]);
    });

    it('keeps every group inside the ≤4-per-decision-point rule', () => {
      const sizes = ['appearance', 'connection', 'library', 'maintenance', 'unavailable'].map(
        (g) => SETTINGS_CATEGORIES.filter((c) => c.group === g).length
      );
      expect(Math.max(...sizes)).toBeLessThanOrEqual(4);
      // export graduated to maintenance (fix-settings-graduation) — the
      // exporter had been live inside 備份與還原 all along; maintenance hits
      // the ≤4 ceiling exactly and 尚未開放 shrinks to performance alone.
      // 外觀 is a group of one on purpose — a theme choice is not a settings
      // page's worth of options, and folding it into 連線 would put a display
      // preference among service credentials.
      expect(sizes).toEqual([1, 3, 2, 4, 1]);
    });

    it('draws a divider between groups but never before the first', async () => {
      renderWithRouter();
      await screen.findByTestId('settings-tabs');
      // 外觀 is first now, so IT is the group that must carry no divider.
      expect(screen.queryByTestId('settings-tabs-divider-appearance')).toBeNull();
      expect(screen.getByTestId('settings-tabs-divider-connection')).toBeInTheDocument();
      expect(screen.getByTestId('settings-tabs-divider-library')).toBeInTheDocument();
      expect(screen.getByTestId('settings-tabs-divider-maintenance')).toBeInTheDocument();
      expect(screen.getByTestId('settings-tabs-divider-unavailable')).toBeInTheDocument();
    });

    it('hides the dividers from AT and carries the group in the accessible name', async () => {
      renderWithRouter();
      await screen.findByTestId('settings-tabs');
      expect(screen.getByTestId('settings-tabs-divider-library')).toHaveAttribute(
        'aria-hidden',
        'true'
      );
      // A decorative rule conveys nothing to a screen reader; the name must.
      expect(screen.getByTestId('settings-tab-status')).toHaveAttribute(
        'aria-label',
        '連線：服務狀態'
      );
      expect(screen.getByTestId('settings-tab-backup')).toHaveAttribute(
        'aria-label',
        '維護：備份與還原'
      );
    });
  });

  describe('overflow is signposted', () => {
    it('renders an edge fade so a clipped tab looks clipped', async () => {
      renderWithRouter();
      await screen.findByTestId('settings-tabs');
      const fade = screen.getByTestId('settings-tabs-fade');
      expect(fade).toHaveAttribute('aria-hidden', 'true');
      expect(fade.className).toContain('pointer-events-none');
    });

    it('lets the strip scroll instead of clipping items away', async () => {
      renderWithRouter();
      const strip = await screen.findByTestId('settings-tabs-strip');
      expect(strip.className).toContain('overflow-x-auto');
    });
  });

  describe('touch targets', () => {
    it.each(SETTINGS_CATEGORIES.map((c) => c.key))(
      '%s meets the 44px minimum the old 30px chips missed',
      async (key) => {
        renderWithRouter();
        const tab = await screen.findByTestId(`settings-tab-${key}`);
        expect(tab.className).toContain('min-h-[44px]');
      }
    );
  });

  describe('active state', () => {
    it.each([
      ['/settings/connection', 'connection'],
      ['/settings/cache', 'cache'],
      ['/settings/logs', 'logs'],
      ['/settings/status', 'status'],
      ['/settings/backup', 'backup'],
    ])('marks %s active through the router, not a hand-rolled match', async (path, key) => {
      renderWithRouter(path);
      const tab = await screen.findByTestId(`settings-tab-${key}`);
      // Both of these come from TanStack Router. The old rail hand-rolled
      // `currentPath.startsWith()`, which the ADR in SidebarNavItem.tsx:5-6
      // forbids and which the tab strip briefly inherited.
      expect(tab).toHaveAttribute('data-status', 'active');
      expect(tab).toHaveAttribute('aria-current', 'page');
    });

    it('leaves every other tab inactive', async () => {
      renderWithRouter('/settings/cache');
      await screen.findByTestId('settings-tab-cache');
      const connection = screen.getByTestId('settings-tab-connection');
      expect(connection).not.toHaveAttribute('aria-current');
      expect(connection).not.toHaveAttribute('data-status', 'active');
    });

    // The styling now rides on a `data-[status=active]:` variant, which means the
    // class string is IDENTICAL on every tab — asserting `.className` contains it
    // would pass for an inactive tab too. Assert the recipe once, structurally,
    // and let the data-status assertions above carry which tab wears it.
    it('dresses the active state in the AA-safe recipe, not the 3.04:1 one', async () => {
      renderWithRouter('/settings/connection');
      const tab = await screen.findByTestId('settings-tab-connection');
      expect(tab.className).toContain('data-[status=active]:bg-[var(--accent-subtle)]');
      expect(tab.className).toContain('data-[status=active]:text-[var(--text-primary)]');
      // --accent-primary as a label colour measured 3.04:1 (PR #287).
      expect(tab.className).not.toContain('text-[var(--accent-primary)]');
    });
  });

  describe('unavailable categories', () => {
    it.each([['performance', '效能監控']])(
      '%s stays visible, reachable, and says why',
      async (key, label) => {
        renderWithRouter();
        const tab = await screen.findByTestId(`settings-tab-${key}`);
        expect(tab).toHaveTextContent(label);
        expect(tab).toHaveTextContent('尚未開放');
        expect(tab).not.toHaveTextContent('Coming Soon');
        expect(tab).toHaveAttribute('role', 'link');
        expect(tab).toHaveAttribute('aria-disabled', 'true');
        expect(tab).toHaveAttribute('tabindex', '0');
        expect(tab).toHaveAttribute('aria-label', `${label}：此功能尚未實作`);
      }
    );

    it('does not navigate when an unavailable tab is clicked', async () => {
      const user = userEvent.setup();
      renderWithRouter('/settings/connection');
      await user.click(await screen.findByTestId('settings-tab-performance'));
      expect(screen.getByTestId('connection-page')).toBeInTheDocument();
    });

    // The other side of the graduation: export is a REAL link now. A tab that
    // kept the disabled costume after its page went live would be the same
    // lie in the opposite direction.
    it('export graduated: enabled, navigable, no 尚未開放 badge', async () => {
      const user = userEvent.setup();
      renderWithRouter('/settings/connection');
      const tab = await screen.findByTestId('settings-tab-export');
      expect(tab).not.toHaveTextContent('尚未開放');
      expect(tab).not.toHaveAttribute('aria-disabled');
      await user.click(tab);
      expect(await screen.findByText('Export')).toBeInTheDocument();
    });
  });

  describe('navigation', () => {
    it('clicking a tab changes route and moves the active state', async () => {
      const user = userEvent.setup();
      renderWithRouter('/settings/connection');
      await screen.findByTestId('settings-tabs');

      await user.click(screen.getByTestId('settings-tab-cache'));

      expect(await screen.findByTestId('cache-page')).toBeInTheDocument();
      expect(screen.getByTestId('settings-tab-cache')).toHaveAttribute('aria-current', 'page');
      expect(screen.getByTestId('settings-tab-connection')).not.toHaveAttribute('aria-current');
    });

    it('renders the routed child inside the content area', async () => {
      renderWithRouter('/settings/connection');
      const content = await screen.findByTestId('settings-content');
      expect(content).toContainElement(screen.getByTestId('connection-page'));
    });
  });
});
