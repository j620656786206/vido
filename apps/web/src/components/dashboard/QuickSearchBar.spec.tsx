import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import {
  createMemoryHistory,
  RouterProvider,
  createRootRoute,
  createRoute,
  createRouter,
} from '@tanstack/react-router';
import React from 'react';
import { QuickSearchBar } from './QuickSearchBar';

const mockNavigate = vi.fn();

vi.mock('@tanstack/react-router', async () => {
  const actual = await vi.importActual('@tanstack/react-router');
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

function renderSearchBar() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  const rootRoute = createRootRoute({
    component: () => React.createElement(QuickSearchBar),
  });
  const searchRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/search',
    component: () => React.createElement('div', null, 'Search Page'),
  });

  const routeTree = rootRoute.addChildren([searchRoute]);
  const router = createRouter({
    routeTree,
    history: createMemoryHistory({ initialEntries: ['/'] }),
  });

  return render(
    React.createElement(
      QueryClientProvider,
      { client: queryClient },
      React.createElement(RouterProvider, { router } as any)
    )
  );
}

describe('QuickSearchBar', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('[P1] renders search input with placeholder', async () => {
    renderSearchBar();
    const input = await screen.findByPlaceholderText('搜尋媒體庫...');
    expect(input).toBeTruthy();
  });

  it('[P1] accepts text input', async () => {
    const user = userEvent.setup();
    renderSearchBar();
    const input = await screen.findByPlaceholderText('搜尋媒體庫...');
    await user.type(input, '鬼滅之刃');
    expect((input as HTMLInputElement).value).toBe('鬼滅之刃');
  });

  it('[P1] has quick-search-bar test id', async () => {
    renderSearchBar();
    expect(await screen.findByTestId('quick-search-bar')).toBeTruthy();
  });

  it('[P2] has search icon', async () => {
    renderSearchBar();
    expect(await screen.findByTestId('search-icon')).toBeTruthy();
  });

  it('[P1] navigates to search page on form submit', async () => {
    // GIVEN: QuickSearchBar is rendered
    const user = userEvent.setup();
    renderSearchBar();
    const input = await screen.findByPlaceholderText('搜尋媒體庫...');

    // WHEN: User types a query and presses Enter
    await user.type(input, '鬼滅之刃');
    await user.keyboard('{Enter}');

    // THEN: navigate called with correct params
    expect(mockNavigate).toHaveBeenCalledWith({
      to: '/search',
      search: { q: '鬼滅之刃' },
    });
  });

  it('[P1] does not navigate with empty query', async () => {
    // GIVEN: QuickSearchBar is rendered
    const user = userEvent.setup();
    renderSearchBar();
    const input = await screen.findByPlaceholderText('搜尋媒體庫...');

    // WHEN: User presses Enter with empty input
    await user.click(input);
    await user.keyboard('{Enter}');

    // THEN: navigate not called
    expect(mockNavigate).not.toHaveBeenCalled();
  });

  it('[P2] does not navigate with whitespace-only query', async () => {
    // GIVEN: QuickSearchBar is rendered
    const user = userEvent.setup();
    renderSearchBar();
    const input = await screen.findByPlaceholderText('搜尋媒體庫...');

    // WHEN: User types whitespace and presses Enter
    await user.type(input, '   ');
    await user.keyboard('{Enter}');

    // THEN: navigate not called
    expect(mockNavigate).not.toHaveBeenCalled();
  });
});
