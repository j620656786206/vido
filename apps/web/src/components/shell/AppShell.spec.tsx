import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import {
  createRootRoute,
  createRoute,
  createRouter,
  createMemoryHistory,
  RouterProvider,
} from '@tanstack/react-router';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { AppShell } from './AppShell';

function createTestRouter(initialPath = '/') {
  const rootRoute = createRootRoute({
    component: () =>
      React.createElement(
        AppShell,
        null,
        React.createElement('div', { 'data-testid': 'page-content' }, 'Page Content')
      ),
  });

  const indexRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/',
    component: () => React.createElement('div', null, 'Home'),
  });

  const libraryRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/library',
    component: () => React.createElement('div', null, 'Library'),
  });

  const searchRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/search',
    component: () => React.createElement('div', null, 'Search'),
  });

  const settingsRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/settings/qbittorrent',
    component: () => React.createElement('div', null, 'Settings'),
  });

  const routeTree = rootRoute.addChildren([indexRoute, libraryRoute, searchRoute, settingsRoute]);
  const memoryHistory = createMemoryHistory({ initialEntries: [initialPath] });

  return createRouter({
    routeTree,
    history: memoryHistory,
  });
}

function renderWithRouter(initialPath = '/') {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  const router = createTestRouter(initialPath);

  return render(
    React.createElement(
      QueryClientProvider,
      { client: queryClient },
      React.createElement(RouterProvider, { router })
    )
  );
}

describe('AppShell', () => {
  it('renders the app shell with logo', async () => {
    renderWithRouter();
    expect(await screen.findByTestId('app-logo')).toHaveTextContent('vido');
  });

  it('renders the app shell container', async () => {
    renderWithRouter();
    expect(await screen.findByTestId('app-shell')).toBeInTheDocument();
  });

  it('renders desktop search bar', async () => {
    renderWithRouter();
    expect(await screen.findByTestId('global-search-input')).toBeInTheDocument();
  });

  it('renders settings link', async () => {
    renderWithRouter();
    expect(await screen.findByTestId('settings-link')).toBeInTheDocument();
  });

  it('renders tab navigation', async () => {
    renderWithRouter();
    expect(await screen.findByTestId('tab-navigation')).toBeInTheDocument();
  });

  it('renders mobile search toggle button', async () => {
    renderWithRouter();
    expect(await screen.findByTestId('mobile-search-toggle')).toBeInTheDocument();
  });

  it('renders page content within shell', async () => {
    renderWithRouter();
    expect(await screen.findByTestId('page-content')).toBeInTheDocument();
  });

  it('shows search placeholder text', async () => {
    renderWithRouter();
    const searchInput = await screen.findByTestId('global-search-input');
    expect(searchInput).toHaveAttribute('placeholder', '搜尋電影或影集...');
  });

  it('[P0] navigates to /search with query on form submit', async () => {
    const router = createTestRouter('/');
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });

    render(
      React.createElement(
        QueryClientProvider,
        { client: queryClient },
        React.createElement(RouterProvider, { router })
      )
    );

    // GIVEN: Search input is available
    const searchInput = await screen.findByTestId('global-search-input');

    // WHEN: User types a query and submits the form
    fireEvent.change(searchInput, { target: { value: '蜘蛛人' } });
    fireEvent.submit(screen.getByTestId('search-form'));

    // THEN: Router navigates to /search with query param
    await vi.waitFor(() => {
      expect(router.state.location.pathname).toBe('/search');
      expect(router.state.location.search).toEqual({ q: '蜘蛛人' });
    });
  });

  it('[P1] does not navigate when search query is empty', async () => {
    const router = createTestRouter('/library');
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });

    render(
      React.createElement(
        QueryClientProvider,
        { client: queryClient },
        React.createElement(RouterProvider, { router })
      )
    );

    // GIVEN: Search input is empty
    const searchInput = await screen.findByTestId('global-search-input');
    expect(searchInput).toHaveValue('');

    // WHEN: User submits the form without typing
    fireEvent.submit(screen.getByTestId('search-form'));

    // THEN: Router stays on current page
    expect(router.state.location.pathname).toBe('/library');
  });

  it('[P1] mobile search toggle shows and hides search input', async () => {
    renderWithRouter();

    // GIVEN: Mobile search is initially hidden
    await screen.findByTestId('mobile-search-toggle');
    expect(screen.queryByTestId('mobile-search-input')).not.toBeInTheDocument();

    // WHEN: User clicks the mobile search toggle
    fireEvent.click(screen.getByTestId('mobile-search-toggle'));

    // THEN: Mobile search input appears
    expect(screen.getByTestId('mobile-search-input')).toBeInTheDocument();

    // WHEN: User clicks toggle again
    fireEvent.click(screen.getByTestId('mobile-search-toggle'));

    // THEN: Mobile search input disappears
    expect(screen.queryByTestId('mobile-search-input')).not.toBeInTheDocument();
  });

  it('[P1] app shell container has dark theme baseline', async () => {
    renderWithRouter();

    // GIVEN/WHEN: App shell renders
    const shell = await screen.findByTestId('app-shell');

    // THEN: Container has dark theme classes
    expect(shell).toHaveClass('bg-slate-900');
  });
});
