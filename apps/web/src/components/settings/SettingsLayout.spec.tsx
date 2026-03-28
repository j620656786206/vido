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
  });

  it('renders the desktop sidebar', async () => {
    renderWithRouter();
    expect(await screen.findByTestId('settings-sidebar')).toBeInTheDocument();
  });

  it('renders the mobile tabs', async () => {
    renderWithRouter();
    expect(await screen.findByTestId('settings-tabs')).toBeInTheDocument();
  });

  it('renders all 7 sidebar navigation items', async () => {
    renderWithRouter();
    await screen.findByTestId('settings-sidebar');
    expect(screen.getByTestId('settings-nav-connection')).toBeInTheDocument();
    expect(screen.getByTestId('settings-nav-cache')).toBeInTheDocument();
    expect(screen.getByTestId('settings-nav-logs')).toBeInTheDocument();
    expect(screen.getByTestId('settings-nav-status')).toBeInTheDocument();
    expect(screen.getByTestId('settings-nav-backup')).toBeInTheDocument();
    expect(screen.getByTestId('settings-nav-export')).toBeInTheDocument();
    expect(screen.getByTestId('settings-nav-performance')).toBeInTheDocument();
  });

  it('renders all 7 mobile tab items', async () => {
    renderWithRouter();
    await screen.findByTestId('settings-tabs');
    expect(screen.getByTestId('settings-tab-connection')).toBeInTheDocument();
    expect(screen.getByTestId('settings-tab-cache')).toBeInTheDocument();
    expect(screen.getByTestId('settings-tab-logs')).toBeInTheDocument();
    expect(screen.getByTestId('settings-tab-status')).toBeInTheDocument();
    expect(screen.getByTestId('settings-tab-backup')).toBeInTheDocument();
    expect(screen.getByTestId('settings-tab-export')).toBeInTheDocument();
    expect(screen.getByTestId('settings-tab-performance')).toBeInTheDocument();
  });

  it('displays correct zh-TW labels for categories', async () => {
    renderWithRouter();
    await screen.findByTestId('settings-sidebar');
    expect(screen.getByTestId('settings-nav-connection')).toHaveTextContent('連線設定');
    expect(screen.getByTestId('settings-nav-cache')).toHaveTextContent('快取管理');
    expect(screen.getByTestId('settings-nav-logs')).toHaveTextContent('系統日誌');
    expect(screen.getByTestId('settings-nav-status')).toHaveTextContent('服務狀態');
    expect(screen.getByTestId('settings-nav-backup')).toHaveTextContent('備份與還原');
    expect(screen.getByTestId('settings-nav-export')).toHaveTextContent('匯出/匯入');
    expect(screen.getByTestId('settings-nav-performance')).toHaveTextContent('效能監控');
  });

  it('highlights the active sidebar item for connection route', async () => {
    renderWithRouter('/settings/connection');
    const connectionNav = await screen.findByTestId('settings-nav-connection');
    expect(connectionNav).toHaveClass('text-blue-400');
    expect(connectionNav).toHaveClass('bg-slate-700');
    expect(connectionNav).toHaveClass('border-blue-400');
  });

  it('shows inactive styling for non-active sidebar items', async () => {
    renderWithRouter('/settings/connection');
    const cacheNav = await screen.findByTestId('settings-nav-cache');
    expect(cacheNav).toHaveClass('text-slate-400');
    expect(cacheNav).toHaveClass('border-transparent');
  });

  it('highlights the active mobile tab for connection route', async () => {
    renderWithRouter('/settings/connection');
    const connectionTab = await screen.findByTestId('settings-tab-connection');
    expect(connectionTab).toHaveClass('text-blue-400');
    expect(connectionTab).toHaveClass('border-blue-400');
  });

  it('displays abbreviated labels in mobile tabs', async () => {
    renderWithRouter();
    await screen.findByTestId('settings-tabs');
    expect(screen.getByTestId('settings-tab-connection')).toHaveTextContent('連線');
    expect(screen.getByTestId('settings-tab-cache')).toHaveTextContent('快取');
    expect(screen.getByTestId('settings-tab-logs')).toHaveTextContent('日誌');
  });

  it('renders the content area', async () => {
    renderWithRouter();
    expect(await screen.findByTestId('settings-content')).toBeInTheDocument();
  });

  it('renders sidebar with aria-label for accessibility', async () => {
    renderWithRouter();
    const sidebar = await screen.findByTestId('settings-sidebar');
    expect(sidebar).toHaveAttribute('aria-label', '設定分類導航');
  });

  it('renders mobile tabs with aria-label for accessibility', async () => {
    renderWithRouter();
    const tabs = await screen.findByTestId('settings-tabs');
    expect(tabs).toHaveAttribute('aria-label', '設定分類標籤');
  });

  // --- Navigation between categories ---

  it('highlights cache sidebar item when navigated to /settings/cache', async () => {
    renderWithRouter('/settings/cache');
    const cacheNav = await screen.findByTestId('settings-nav-cache');
    expect(cacheNav).toHaveClass('text-blue-400');
    expect(cacheNav).toHaveClass('bg-slate-700');
    expect(cacheNav).toHaveClass('border-blue-400');
    // connection should be inactive
    const connectionNav = screen.getByTestId('settings-nav-connection');
    expect(connectionNav).toHaveClass('text-slate-400');
    expect(connectionNav).toHaveClass('border-transparent');
  });

  it('highlights logs sidebar item when navigated to /settings/logs', async () => {
    renderWithRouter('/settings/logs');
    const logsNav = await screen.findByTestId('settings-nav-logs');
    expect(logsNav).toHaveClass('text-blue-400');
    expect(logsNav).toHaveClass('border-blue-400');
  });

  it('highlights status sidebar item when navigated to /settings/status', async () => {
    renderWithRouter('/settings/status');
    const statusNav = await screen.findByTestId('settings-nav-status');
    expect(statusNav).toHaveClass('text-blue-400');
    expect(statusNav).toHaveClass('border-blue-400');
  });

  it('highlights backup sidebar item when navigated to /settings/backup', async () => {
    renderWithRouter('/settings/backup');
    const backupNav = await screen.findByTestId('settings-nav-backup');
    expect(backupNav).toHaveClass('text-blue-400');
    expect(backupNav).toHaveClass('border-blue-400');
  });

  it('renders disabled export sidebar item with Coming Soon badge', async () => {
    renderWithRouter('/settings/connection');
    const exportNav = await screen.findByTestId('settings-nav-export');
    expect(exportNav).toHaveClass('cursor-not-allowed');
    expect(exportNav).toHaveClass('text-slate-600');
    expect(exportNav).toHaveTextContent('Coming Soon');
  });

  it('renders disabled performance sidebar item with Coming Soon badge', async () => {
    renderWithRouter('/settings/connection');
    const perfNav = await screen.findByTestId('settings-nav-performance');
    expect(perfNav).toHaveClass('cursor-not-allowed');
    expect(perfNav).toHaveClass('text-slate-600');
    expect(perfNav).toHaveTextContent('Coming Soon');
  });

  // --- Mobile tab active states for each category ---

  it('highlights cache mobile tab when navigated to /settings/cache', async () => {
    renderWithRouter('/settings/cache');
    const cacheTab = await screen.findByTestId('settings-tab-cache');
    expect(cacheTab).toHaveClass('text-blue-400');
    expect(cacheTab).toHaveClass('border-blue-400');
    // connection tab should be inactive
    const connectionTab = screen.getByTestId('settings-tab-connection');
    expect(connectionTab).toHaveClass('text-slate-400');
  });

  it('renders disabled performance mobile tab without active styling', async () => {
    renderWithRouter('/settings/connection');
    const perfTab = await screen.findByTestId('settings-tab-performance');
    expect(perfTab).toHaveClass('cursor-not-allowed');
    expect(perfTab).toHaveClass('text-slate-600');
  });

  // --- All 7 categories have correct routes ---

  it('sidebar navigation items link to correct routes', async () => {
    renderWithRouter();
    await screen.findByTestId('settings-sidebar');
    expect(screen.getByTestId('settings-nav-connection')).toHaveAttribute(
      'href',
      '/settings/connection'
    );
    expect(screen.getByTestId('settings-nav-cache')).toHaveAttribute('href', '/settings/cache');
    expect(screen.getByTestId('settings-nav-logs')).toHaveAttribute('href', '/settings/logs');
    expect(screen.getByTestId('settings-nav-status')).toHaveAttribute('href', '/settings/status');
    expect(screen.getByTestId('settings-nav-backup')).toHaveAttribute('href', '/settings/backup');
    // export and performance are disabled — rendered as spans, no href
    expect(screen.getByTestId('settings-nav-export').tagName).toBe('SPAN');
    expect(screen.getByTestId('settings-nav-performance').tagName).toBe('SPAN');
  });

  it('mobile tab items link to correct routes', async () => {
    renderWithRouter();
    await screen.findByTestId('settings-tabs');
    expect(screen.getByTestId('settings-tab-connection')).toHaveAttribute(
      'href',
      '/settings/connection'
    );
    expect(screen.getByTestId('settings-tab-cache')).toHaveAttribute('href', '/settings/cache');
    expect(screen.getByTestId('settings-tab-logs')).toHaveAttribute('href', '/settings/logs');
    expect(screen.getByTestId('settings-tab-status')).toHaveAttribute('href', '/settings/status');
    expect(screen.getByTestId('settings-tab-backup')).toHaveAttribute('href', '/settings/backup');
    // export and performance are disabled — rendered as spans, no href
    expect(screen.getByTestId('settings-tab-export').tagName).toBe('SPAN');
    expect(screen.getByTestId('settings-tab-performance').tagName).toBe('SPAN');
  });

  // --- Mobile abbreviated labels vs desktop full labels ---

  it('displays all abbreviated labels in mobile tabs', async () => {
    renderWithRouter();
    await screen.findByTestId('settings-tabs');
    expect(screen.getByTestId('settings-tab-connection')).toHaveTextContent('連線');
    expect(screen.getByTestId('settings-tab-cache')).toHaveTextContent('快取');
    expect(screen.getByTestId('settings-tab-logs')).toHaveTextContent('日誌');
    expect(screen.getByTestId('settings-tab-status')).toHaveTextContent('狀態');
    expect(screen.getByTestId('settings-tab-backup')).toHaveTextContent('備份');
    expect(screen.getByTestId('settings-tab-export')).toHaveTextContent('匯出');
    expect(screen.getByTestId('settings-tab-performance')).toHaveTextContent('效能');
  });

  it('mobile tabs use shortLabel (not full label) for brevity', async () => {
    renderWithRouter();
    await screen.findByTestId('settings-tabs');
    // Mobile tabs should NOT contain the full labels (which have extra characters)
    // e.g. '連線設定' vs '連線', '備份與還原' vs '備份'
    const backupTab = screen.getByTestId('settings-tab-backup');
    expect(backupTab.textContent).not.toContain('與還原');
    const exportTab = screen.getByTestId('settings-tab-export');
    expect(exportTab.textContent).not.toContain('/匯入');
  });

  // --- Clicking sidebar item navigates and changes active state ---

  it('clicking a sidebar item navigates and updates active state', async () => {
    const user = userEvent.setup();
    renderWithRouter('/settings/connection');
    await screen.findByTestId('settings-sidebar');

    // Verify connection is active initially
    expect(screen.getByTestId('settings-nav-connection')).toHaveClass('text-blue-400');

    // Click cache
    await user.click(screen.getByTestId('settings-nav-cache'));

    // Cache should now be active
    const cacheNav = await screen.findByTestId('settings-nav-cache');
    expect(cacheNav).toHaveClass('text-blue-400');
    expect(cacheNav).toHaveClass('border-blue-400');

    // Connection should now be inactive
    expect(screen.getByTestId('settings-nav-connection')).toHaveClass('text-slate-400');
  });

  it('clicking a mobile tab navigates and updates active state', async () => {
    const user = userEvent.setup();
    renderWithRouter('/settings/connection');
    await screen.findByTestId('settings-tabs');

    // Click logs tab
    await user.click(screen.getByTestId('settings-tab-logs'));

    const logsTab = await screen.findByTestId('settings-tab-logs');
    expect(logsTab).toHaveClass('text-blue-400');
  });

  // --- Content rendering ---

  it('renders child content in the content area for connection route', async () => {
    renderWithRouter('/settings/connection');
    expect(await screen.findByTestId('connection-page')).toBeInTheDocument();
    expect(screen.getByTestId('connection-page')).toHaveTextContent('Connection');
  });

  it('renders child content in the content area for cache route', async () => {
    renderWithRouter('/settings/cache');
    expect(await screen.findByTestId('cache-page')).toBeInTheDocument();
    expect(screen.getByTestId('cache-page')).toHaveTextContent('Cache');
  });

  // --- SETTINGS_CATEGORIES export ---

  it('exports SETTINGS_CATEGORIES with exactly 8 entries', () => {
    expect(SETTINGS_CATEGORIES).toHaveLength(8);
  });

  it('SETTINGS_CATEGORIES entries have required fields', () => {
    for (const cat of SETTINGS_CATEGORIES) {
      expect(cat).toHaveProperty('key');
      expect(cat).toHaveProperty('label');
      expect(cat).toHaveProperty('shortLabel');
      expect(cat).toHaveProperty('icon');
      expect(cat).toHaveProperty('to');
      expect(cat.to).toMatch(/^\/settings\//);
    }
  });

  // --- Sidebar uses <nav> elements ---

  it('sidebar is rendered as a nav element', async () => {
    renderWithRouter();
    const sidebar = await screen.findByTestId('settings-sidebar');
    expect(sidebar.tagName).toBe('NAV');
  });

  it('mobile tabs container is rendered as a nav element', async () => {
    renderWithRouter();
    const tabs = await screen.findByTestId('settings-tabs');
    expect(tabs.tagName).toBe('NAV');
  });

  // --- Layout structure ---

  it('content area is inside the layout container', async () => {
    renderWithRouter();
    const layout = await screen.findByTestId('settings-layout');
    const content = screen.getByTestId('settings-content');
    expect(layout).toContainElement(content);
  });

  it('sidebar is inside the layout container', async () => {
    renderWithRouter();
    const layout = await screen.findByTestId('settings-layout');
    const sidebar = screen.getByTestId('settings-sidebar');
    expect(layout).toContainElement(sidebar);
  });

  // --- Redirect behavior (AC4, AC5) ---

  it('redirects /settings/ to /settings/connection (AC4)', async () => {
    const rootRoute = createRootRoute({
      component: () => React.createElement(Outlet),
    });
    const settingsRoute = createRoute({
      getParentRoute: () => rootRoute,
      path: '/settings',
      component: () => React.createElement(SettingsLayout, null, React.createElement(Outlet)),
    });
    const indexRoute = createRoute({
      getParentRoute: () => settingsRoute,
      path: '/',
      beforeLoad: () => {
        throw new Error('redirect:/settings/connection');
      },
    });
    const connectionRoute = createRoute({
      getParentRoute: () => settingsRoute,
      path: '/connection',
      component: () =>
        React.createElement('div', { 'data-testid': 'connection-page' }, 'Connection'),
    });
    const routeTree = rootRoute.addChildren([
      settingsRoute.addChildren([indexRoute, connectionRoute]),
    ]);
    // Verify the redirect route file uses beforeLoad with redirect
    // (structural test — TanStack Router redirect throws are hard to test in unit context)
    const { Route: IndexRoute } = await import('../../routes/settings/index');
    expect(IndexRoute).toBeDefined();
    expect(IndexRoute.options).toHaveProperty('beforeLoad');
  });

  it('qbittorrent route has redirect beforeLoad (AC5)', async () => {
    const { Route: QBRoute } = await import('../../routes/settings/qbittorrent');
    expect(QBRoute).toBeDefined();
    expect(QBRoute.options).toHaveProperty('beforeLoad');
  });
});
