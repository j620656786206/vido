import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import {
  createRootRoute,
  createRoute,
  createRouter,
  createMemoryHistory,
  RouterProvider,
  Outlet,
} from '@tanstack/react-router';

// Heavy / network-y leaves are stubbed — this spec is about the shell's chrome
// (sidebar, header, ScanProgress ownership), not their internals.
vi.mock('../search/InstantSearchBar', () => ({ InstantSearchBar: () => null }));
vi.mock('../scanner/ScanProgress', () => ({
  ScanProgress: () => <div data-testid="scan-progress" />,
}));
vi.mock('../../hooks/useLibrary', () => ({
  useLibraryStats: () => ({ data: { movieCount: 1, tvCount: 1 } }),
}));
vi.mock('../../hooks/useStatusSummary', () => ({
  useStatusSummary: () => ({ data: undefined }),
}));

import { AppShellV2 } from './AppShellV2';

function renderShell(path: string) {
  const rootRoute = createRootRoute({
    component: () => React.createElement(AppShellV2, null, React.createElement(Outlet)),
  });
  const indexRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/',
    component: () => React.createElement('div', { 'data-testid': 'home-content' }, 'home'),
  });
  const libraryRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/library',
    component: () => React.createElement('div', { 'data-testid': 'browse-content' }, 'browse'),
  });
  const downloadsRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/downloads',
    component: () => null,
  });
  const discoverRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/discover',
    component: () => null,
  });
  const settingsRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/settings',
    component: () => null,
  });
  const routeTree = rootRoute.addChildren([
    indexRoute,
    libraryRoute,
    downloadsRoute,
    discoverRoute,
    settingsRoute,
  ]);
  const router = createRouter({
    routeTree,
    history: createMemoryHistory({ initialEntries: [path] }),
  });
  return render(React.createElement(RouterProvider, { router }));
}

describe('AppShellV2', () => {
  it('renders the shell with the sidebar and owns the ScanProgress overlay', async () => {
    renderShell('/');
    expect(await screen.findByTestId('app-shell-v2')).toBeInTheDocument();
    expect(screen.getByTestId('app-sidebar')).toBeInTheDocument();
    expect(screen.getByTestId('scan-progress')).toBeInTheDocument();
  });

  it('renders route content directly in the main column (no legacy wrapper — ux3-cutover-4)', async () => {
    renderShell('/');
    const content = await screen.findByTestId('home-content');
    expect(content).toBeInTheDocument();
    expect(screen.queryByTestId('legacy-content-container')).not.toBeInTheDocument();
    expect(content.closest('main')).not.toBeNull();
  });

  it('reveals the top header divider only once the page is scrolled', async () => {
    renderShell('/');
    const header = await screen.findByRole('banner');
    // At rest: divider hidden (no floating line under a sparse bar).
    expect(header).toHaveAttribute('data-scrolled', 'false');
    expect(header.className).toContain('border-transparent');

    Object.defineProperty(window, 'scrollY', { value: 120, configurable: true, writable: true });
    fireEvent.scroll(window);

    expect(header).toHaveAttribute('data-scrolled', 'true');
    expect(header.className).toContain('border-[var(--border-subtle)]');
  });
});
