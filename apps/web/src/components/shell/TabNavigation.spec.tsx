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
    path: '/settings',
    component: () => React.createElement('div', null, 'Settings'),
  });

  const settingsConnectionRoute = createRoute({
    getParentRoute: () => settingsRoute,
    path: '/connection',
    component: () => React.createElement('div', null, 'Connection'),
  });

  const indexRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/',
    component: () => React.createElement('div', null, 'Home'),
  });

  const routeTree = rootRoute.addChildren([
    indexRoute,
    libraryRoute,
    downloadsRoute,
    pendingRoute,
    settingsRoute.addChildren([settingsConnectionRoute]),
  ]);
  const memoryHistory = createMemoryHistory({ initialEntries: [initialPath] });

  return createRouter({ routeTree, history: memoryHistory });
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
    expect(screen.getByTestId('tab-待解析')).toBeInTheDocument();
    expect(screen.getByTestId('tab-設定')).toBeInTheDocument();
  });

  it('highlights the active tab for library route', async () => {
    renderWithRouter('/library');
    const libraryTab = await screen.findByTestId('tab-媒體庫');
    expect(libraryTab).toHaveClass('text-white');
    expect(libraryTab).toHaveClass('border-blue-400');
    expect(screen.getByTestId('tab-下載中')).toHaveClass('text-slate-500');
  });

  it('highlights the active tab for downloads route', async () => {
    renderWithRouter('/downloads');
    const downloadsTab = await screen.findByTestId('tab-下載中');
    expect(downloadsTab).toHaveClass('text-white');
    expect(screen.getByTestId('tab-媒體庫')).toHaveClass('text-slate-500');
  });

  it('highlights the active tab for settings route', async () => {
    renderWithRouter('/settings/connection');
    const settingsTab = await screen.findByTestId('tab-設定');
    expect(settingsTab).toHaveClass('text-white');
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

  it('[P1] highlights the active tab for pending route', async () => {
    renderWithRouter('/pending');

    // GIVEN/WHEN: User is on /pending route
    const pendingTab = await screen.findByTestId('tab-待解析');

    // THEN: Pending tab is active with correct styles
    expect(pendingTab).toHaveClass('text-white');
    expect(pendingTab).toHaveClass('border-blue-400');
    expect(screen.getByTestId('tab-媒體庫')).toHaveClass('text-slate-500');
  });

  it('[P2] shows no active tab on non-tab route', async () => {
    renderWithRouter('/');

    // GIVEN/WHEN: User is on root route (not matching any tab)
    await screen.findByTestId('tab-navigation');

    // THEN: No tab has active styling
    expect(screen.getByTestId('tab-媒體庫')).toHaveClass('text-slate-500');
    expect(screen.getByTestId('tab-下載中')).toHaveClass('text-slate-500');
    expect(screen.getByTestId('tab-待解析')).toHaveClass('text-slate-500');
    expect(screen.getByTestId('tab-設定')).toHaveClass('text-slate-500');
  });
});
