import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import {
  createMemoryHistory,
  RouterProvider,
  createRootRoute,
  createRoute,
  createRouter,
  Outlet,
} from '@tanstack/react-router';
import React from 'react';
import { ExploreBlocksList } from './ExploreBlocksList';

vi.mock('../../hooks/useExploreBlocks', () => ({
  useExploreBlocks: vi.fn(),
  useExploreBlockContent: vi.fn(() => ({ data: undefined, isLoading: true, isError: false })),
}));

import { useExploreBlocks } from '../../hooks/useExploreBlocks';

const mockList = vi.mocked(useExploreBlocks);

function renderList() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const rootRoute = createRootRoute({
    component: () => React.createElement(React.Fragment, null, React.createElement(Outlet)),
  });
  const indexRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/',
    component: () => React.createElement(ExploreBlocksList),
  });
  const mediaRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/media/$type/$id',
    component: () => React.createElement('div', null, 'Media Detail'),
  });
  const searchRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/search',
    component: () => React.createElement('div', null, 'Search'),
  });
  const routeTree = rootRoute.addChildren([indexRoute, mediaRoute, searchRoute]);
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

describe('ExploreBlocksList', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders nothing while loading', async () => {
    mockList.mockReturnValue({ data: undefined, isLoading: true, isError: false } as any);
    const { container } = renderList();
    await Promise.resolve();
    expect(container.querySelector('[data-testid="explore-blocks-list"]')).toBeNull();
  });

  it('renders nothing on error', async () => {
    mockList.mockReturnValue({ data: undefined, isLoading: false, isError: true } as any);
    const { container } = renderList();
    await Promise.resolve();
    expect(container.querySelector('[data-testid="explore-blocks-list"]')).toBeNull();
  });

  it('renders nothing when blocks array is empty', async () => {
    mockList.mockReturnValue({
      data: { blocks: [] },
      isLoading: false,
      isError: false,
    } as any);
    const { container } = renderList();
    await Promise.resolve();
    expect(container.querySelector('[data-testid="explore-blocks-list"]')).toBeNull();
  });

  it('renders one section per block (AC #1)', async () => {
    mockList.mockReturnValue({
      data: {
        blocks: [
          {
            id: 'a',
            name: '熱門電影',
            contentType: 'movie',
            genreIds: '',
            language: '',
            region: '',
            sortBy: 'popularity.desc',
            maxItems: 20,
            sortOrder: 0,
            createdAt: '',
            updatedAt: '',
          },
          {
            id: 'b',
            name: '熱門影集',
            contentType: 'tv',
            genreIds: '',
            language: '',
            region: '',
            sortBy: 'popularity.desc',
            maxItems: 20,
            sortOrder: 1,
            createdAt: '',
            updatedAt: '',
          },
        ],
      },
      isLoading: false,
      isError: false,
    } as any);

    renderList();
    expect(await screen.findByTestId('explore-blocks-list')).toBeInTheDocument();
    expect(screen.getByTestId('explore-block-a')).toBeInTheDocument();
    expect(screen.getByTestId('explore-block-b')).toBeInTheDocument();
  });
});
