import React from 'react';
import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import {
  createRootRoute,
  createRoute,
  createRouter,
  createMemoryHistory,
  RouterProvider,
} from '@tanstack/react-router';
import { TabNavigation } from './TabNavigation';

function createTestRouter(initialPath = '/library') {
  const rootRoute = createRootRoute({
    component: () => React.createElement('div', null, React.createElement(TabNavigation)),
  });

  const libraryRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/library',
    component: () => React.createElement('div', null, 'Library'),
  });

  const downloadsRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/downloads',
    component: () => React.createElement('div', null, 'Downloads'),
  });

  const pendingRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/pending',
    component: () => React.createElement('div', null, 'Pending'),
  });

  const settingsRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/settings/qbittorrent',
    component: () => React.createElement('div', null, 'Settings'),
  });

  const routeTree = rootRoute.addChildren([
    libraryRoute,
    downloadsRoute,
    pendingRoute,
    settingsRoute,
  ]);
  const memoryHistory = createMemoryHistory({ initialEntries: [initialPath] });

  return createRouter({
    routeTree,
    history: memoryHistory,
  });
}

function renderWithRouter(initialPath = '/library') {
  const router = createTestRouter(initialPath);
  return render(React.createElement(RouterProvider, { router }));
}

describe('TabNavigation', () => {
  it('renders all four navigation tabs', async () => {
    renderWithRouter();
    expect(await screen.findByTestId('tab-媒體庫')).toBeInTheDocument();
    expect(screen.getByTestId('tab-下載中')).toBeInTheDocument();
    expect(screen.getByTestId('tab-待處理')).toBeInTheDocument();
    expect(screen.getByTestId('tab-設定')).toBeInTheDocument();
  });

  it('highlights the active tab for library route', async () => {
    renderWithRouter('/library');
    const libraryTab = await screen.findByTestId('tab-媒體庫');
    expect(libraryTab).toHaveClass('text-blue-400');
    expect(screen.getByTestId('tab-下載中')).toHaveClass('text-slate-400');
  });

  it('highlights the active tab for downloads route', async () => {
    renderWithRouter('/downloads');
    const downloadsTab = await screen.findByTestId('tab-下載中');
    expect(downloadsTab).toHaveClass('text-blue-400');
    expect(screen.getByTestId('tab-媒體庫')).toHaveClass('text-slate-400');
  });

  it('highlights the active tab for settings route', async () => {
    renderWithRouter('/settings/qbittorrent');
    const settingsTab = await screen.findByTestId('tab-設定');
    expect(settingsTab).toHaveClass('text-blue-400');
  });

  it('shows active tab indicator for current route', async () => {
    renderWithRouter('/library');
    await screen.findByTestId('tab-媒體庫');
    expect(screen.getByTestId('active-tab-indicator')).toBeInTheDocument();
  });

  it('renders navigation with aria-label', async () => {
    renderWithRouter();
    const nav = await screen.findByTestId('tab-navigation');
    expect(nav).toHaveAttribute('aria-label', '主要導航');
  });
});
