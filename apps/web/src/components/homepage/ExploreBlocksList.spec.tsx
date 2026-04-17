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

vi.mock('../../hooks/useExploreBlocks', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../hooks/useExploreBlocks')>();
  return {
    ...actual,
    useExploreBlocks: vi.fn(),
    useExploreBlockContent: vi.fn(() => ({ data: undefined, isLoading: true, isError: false })),
  };
});

// Story 10-4 AC #4: parent ExploreBlocksList fetches each block's content via
// useQueries so a single useOwnedMedia call can union the TMDb IDs across
// blocks. Stub both the content service and the ownership hook here so the
// existing ExploreBlocksList tests stay network-free.
vi.mock('../../services/exploreBlockService', () => ({
  exploreBlockService: {
    getContent: vi
      .fn()
      .mockResolvedValue({ blockId: '', contentType: 'movie', movies: [], totalItems: 0 }),
  },
}));

vi.mock('../../hooks/useOwnedMedia', () => ({
  useOwnedMedia: vi.fn(() => ({
    owned: new Set<number>(),
    isOwned: () => false,
    isRequested: () => false,
    isLoading: false,
    error: null,
  })),
}));

import { useExploreBlocks } from '../../hooks/useExploreBlocks';
import { exploreBlockService } from '../../services/exploreBlockService';
import { useOwnedMedia } from '../../hooks/useOwnedMedia';

const mockList = vi.mocked(useExploreBlocks);
const mockGetContent = vi.mocked(exploreBlockService.getContent);
const mockUseOwnedMedia = vi.mocked(useOwnedMedia);

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

  it('renders loading placeholder while loading (L2 fix)', async () => {
    mockList.mockReturnValue({ data: undefined, isLoading: true, isError: false } as any);
    renderList();
    await Promise.resolve();
    expect(await screen.findByTestId('explore-blocks-loading')).toBeInTheDocument();
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

  it('hoists availability lookup — one useOwnedMedia call unions TMDb IDs across blocks (Story 10-4 AC #4)', async () => {
    mockList.mockReturnValue({
      data: {
        blocks: [
          {
            id: 'mov',
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
            id: 'tv',
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

    // Each block returns different TMDb IDs. If ownership is lifted correctly,
    // a single useOwnedMedia call receives the union.
    mockGetContent.mockImplementation(async (id: string) => {
      if (id === 'mov') {
        return {
          blockId: 'mov',
          contentType: 'movie',
          movies: [
            { id: 603, title: 'Matrix', posterPath: null } as any,
            { id: 157336, title: 'Interstellar', posterPath: null } as any,
          ],
          totalItems: 2,
        } as any;
      }
      return {
        blockId: 'tv',
        contentType: 'tv',
        tvShows: [{ id: 1396, name: 'Breaking Bad', posterPath: null } as any],
        totalItems: 1,
      } as any;
    });

    renderList();
    // Wait for useQueries to resolve both mocked fetches.
    await screen.findByTestId('explore-blocks-list');
    await vi.waitFor(() => {
      const calls = mockUseOwnedMedia.mock.calls;
      // Find the call where tmdbIds has been unioned (all three ids present).
      const unioned = calls.find((c) => {
        const arg = (c[0] ?? []) as number[];
        return arg.includes(603) && arg.includes(157336) && arg.includes(1396);
      });
      expect(unioned).toBeDefined();
    });

    // Regression guard: the child ExploreBlock must NOT call useOwnedMedia on
    // its own. All observed calls come from the parent with the full union,
    // never with a single block's subset.
    const calls = mockUseOwnedMedia.mock.calls;
    const perBlockCalls = calls.filter((c) => {
      const arg = (c[0] ?? []) as number[];
      return arg.length > 0 && arg.length < 3; // subset of one block only
    });
    expect(perBlockCalls).toHaveLength(0);
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
