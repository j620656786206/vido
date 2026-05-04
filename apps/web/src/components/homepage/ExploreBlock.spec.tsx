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
import { ExploreBlock } from './ExploreBlock';
import type { ExploreBlock as ExploreBlockType } from '../../services/exploreBlockService';
import type { OwnedMediaState } from '../../hooks/useOwnedMedia';

vi.mock('../../hooks/useExploreBlocks', () => ({
  useExploreBlockContent: vi.fn(),
}));

import { useExploreBlockContent } from '../../hooks/useExploreBlocks';

const mockHook = vi.mocked(useExploreBlockContent);

// Story 10-4: ownership state is now injected by ExploreBlocksList. Tests
// supply a stub so the block renders independently of the parent.
const stubOwnership: OwnedMediaState = {
  owned: new Set<number>(),
  isOwned: () => false,
  isRequested: () => false,
  isLoading: false,
  error: null,
};

function testBlock(overrides: Partial<ExploreBlockType> = {}): ExploreBlockType {
  return {
    id: 'block-1',
    name: '熱門電影',
    contentType: 'movie',
    genreIds: '',
    language: '',
    region: '',
    sortBy: 'popularity.desc',
    maxItems: 20,
    sortOrder: 0,
    createdAt: '2026-04-15T00:00:00Z',
    updatedAt: '2026-04-15T00:00:00Z',
    ...overrides,
  };
}

function renderBlock(block: ExploreBlockType) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  const rootRoute = createRootRoute({
    component: () => React.createElement(React.Fragment, null, React.createElement(Outlet)),
  });
  const indexRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/',
    component: () => React.createElement(ExploreBlock, { block, ownership: stubOwnership }),
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

describe('ExploreBlock', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('shows skeleton placeholders while loading', async () => {
    mockHook.mockReturnValue({
      data: undefined,
      isLoading: true,
      isError: false,
    } as any);
    renderBlock(testBlock());

    const skeletons = await screen.findAllByTestId('explore-block-skeleton');
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it('renders title and see-more link', async () => {
    mockHook.mockReturnValue({
      data: { blockId: 'block-1', contentType: 'movie', movies: [], totalItems: 0 },
      isLoading: false,
      isError: false,
    } as any);
    renderBlock(testBlock({ name: '熱門韓劇' }));

    expect(await screen.findByTestId('explore-block-title')).toHaveTextContent('熱門韓劇');
    expect(screen.getByTestId('explore-block-see-more')).toBeInTheDocument();
  });

  it('renders movie poster cards when content_type is movie', async () => {
    mockHook.mockReturnValue({
      data: {
        blockId: 'block-1',
        contentType: 'movie',
        movies: [
          {
            id: 1,
            title: '電影 A',
            originalTitle: 'Movie A',
            overview: '',
            releaseDate: '2024-01-01',
            posterPath: '/p1.jpg',
            backdropPath: null,
            voteAverage: 8,
            voteCount: 100,
            genreIds: [28],
          },
          {
            id: 2,
            title: '電影 B',
            originalTitle: 'Movie B',
            overview: '',
            releaseDate: '2024-02-01',
            posterPath: '/p2.jpg',
            backdropPath: null,
            voteAverage: 7,
            voteCount: 80,
            genreIds: [12],
          },
        ],
        totalItems: 2,
      },
      isLoading: false,
      isError: false,
    } as any);

    renderBlock(testBlock());

    expect(await screen.findByText('電影 A')).toBeInTheDocument();
    expect(screen.getByText('電影 B')).toBeInTheDocument();
    expect(screen.getAllByTestId('poster-card')).toHaveLength(2);
  });

  it('renders tv show poster cards when content_type is tv', async () => {
    mockHook.mockReturnValue({
      data: {
        blockId: 'block-tv',
        contentType: 'tv',
        tvShows: [
          {
            id: 10,
            name: '劇集 X',
            originalName: 'Show X',
            overview: '',
            firstAirDate: '2023-01-01',
            posterPath: '/px.jpg',
            backdropPath: null,
            voteAverage: 9,
            voteCount: 500,
            genreIds: [18],
          },
        ],
        totalItems: 1,
      },
      isLoading: false,
      isError: false,
    } as any);

    renderBlock(testBlock({ id: 'block-tv', contentType: 'tv', name: '熱門劇集' }));

    expect(await screen.findByText('劇集 X')).toBeInTheDocument();
  });

  it('shows empty-state message when content is empty and not loading', async () => {
    mockHook.mockReturnValue({
      data: { blockId: 'block-1', contentType: 'movie', movies: [], totalItems: 0 },
      isLoading: false,
      isError: false,
    } as any);

    renderBlock(testBlock());

    expect(await screen.findByTestId('explore-block-empty')).toHaveTextContent(
      '沒有符合條件的內容'
    );
  });

  it('returns null (hides itself) when the content query errors', async () => {
    mockHook.mockReturnValue({
      data: undefined,
      isLoading: false,
      isError: true,
    } as any);

    const { container } = renderBlock(testBlock());

    // Wait a tick so the router mounts, then verify the section is not rendered.
    await Promise.resolve();
    expect(container.querySelector('[data-testid^="explore-block-"]')).toBeNull();
  });

  // bugfix-10-1 Task 5.7 — verify the regression locus stays correct: an
  // ExploreBlock poster's Link MUST encode the TMDb numeric id verbatim so the
  // route's classifyId() can detect it and dispatch to the TMDb detail branch.
  // Don't deep-render the detail page; assert the URL shape only.
  it('poster card link encodes TMDb numeric id in /media/$type/$id', async () => {
    mockHook.mockReturnValue({
      data: {
        blockId: 'block-1',
        contentType: 'movie',
        movies: [
          {
            id: 83533,
            title: '熱門電影',
            originalTitle: 'Trending Movie',
            overview: '',
            releaseDate: '2024-01-01',
            posterPath: '/p.jpg',
            backdropPath: null,
            voteAverage: 8,
            voteCount: 100,
            genreIds: [28],
          },
        ],
        totalItems: 1,
      },
      isLoading: false,
      isError: false,
    } as ReturnType<typeof useExploreBlockContent>);

    renderBlock(testBlock());

    const card = await screen.findByTestId('poster-card');
    expect(card).toHaveAttribute('href', '/media/movie/83533');
  });

  it('renders desktop scroll chevrons', async () => {
    mockHook.mockReturnValue({
      data: {
        blockId: 'block-1',
        contentType: 'movie',
        movies: [],
        totalItems: 0,
      },
      isLoading: false,
      isError: false,
    } as any);

    renderBlock(testBlock());

    expect(await screen.findByTestId('explore-block-scroll-left')).toBeInTheDocument();
    expect(screen.getByTestId('explore-block-scroll-right')).toBeInTheDocument();
  });
});
