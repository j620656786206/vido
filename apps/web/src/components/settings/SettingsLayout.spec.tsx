import React from 'react';
import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import {
  createRootRoute,
  createRoute,
  createRouter,
  createMemoryHistory,
  RouterProvider,
  Outlet,
} from '@tanstack/react-router';
import { SettingsLayout } from './SettingsLayout';

function createTestRouter(initialPath = '/settings/connection') {
  const rootRoute = createRootRoute({
    component: () => React.createElement(Outlet),
  });

  const settingsRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/settings',
    component: () =>
      React.createElement(SettingsLayout, null, React.createElement(Outlet)),
  });

  const connectionRoute = createRoute({
    getParentRoute: () => settingsRoute,
    path: '/connection',
    component: () =>
      React.createElement('div', { 'data-testid': 'connection-page' }, 'Connection'),
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
  });

  it('renders the content area', async () => {
    renderWithRouter();
    expect(await screen.findByTestId('settings-content')).toBeInTheDocument();
  });

  it('renders sidebar with aria-label for accessibility', async () => {
    renderWithRouter();
    const sidebar = await screen.findByTestId('settings-sidebar');
    expect(sidebar).toHaveAttribute('aria-label', '設定分類');
  });

  it('renders mobile tabs with aria-label for accessibility', async () => {
    renderWithRouter();
    const tabs = await screen.findByTestId('settings-tabs');
    expect(tabs).toHaveAttribute('aria-label', '設定分類');
  });
});
