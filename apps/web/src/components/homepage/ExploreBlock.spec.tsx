import { describe, it, expect, vi, beforeEach } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
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
  // A settled verdict by default: most tests care about rendering, not about
  // the ux3-1-8 filter's gating.
  isSettled: true,
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

function renderBlock(block: ExploreBlockType, ownership: OwnedMediaState = stubOwnership) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  const rootRoute = createRootRoute({
    component: () => React.createElement(React.Fragment, null, React.createElement(Outlet)),
  });
  const indexRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/',
    component: () => React.createElement(ExploreBlock, { block, ownership }),
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
    } as ReturnType<typeof useExploreBlockContent>);
    renderBlock(testBlock());

    const skeletons = await screen.findAllByTestId('explore-block-skeleton');
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it('renders the title; 查看更多 stays OUT until Epic 11 lets it keep its promise', async () => {
    mockHook.mockReturnValue({
      data: { blockId: 'block-1', contentType: 'movie', movies: [], totalItems: 0 },
      isLoading: false,
      isError: false,
    } as ReturnType<typeof useExploreBlockContent>);
    renderBlock(testBlock({ name: '熱門韓劇' }));

    expect(await screen.findByTestId('explore-block-title')).toHaveTextContent('熱門韓劇');
    // Critique R2 P2: it promised「more of this row」and delivered unfiltered /search.
    expect(screen.queryByTestId('explore-block-see-more')).toBeNull();
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
    } as ReturnType<typeof useExploreBlockContent>);

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
    } as ReturnType<typeof useExploreBlockContent>);

    renderBlock(testBlock({ id: 'block-tv', contentType: 'tv', name: '熱門劇集' }));

    expect(await screen.findByText('劇集 X')).toBeInTheDocument();
  });

  it('shows empty-state message when content is empty and not loading', async () => {
    mockHook.mockReturnValue({
      data: { blockId: 'block-1', contentType: 'movie', movies: [], totalItems: 0 },
      isLoading: false,
      isError: false,
    } as ReturnType<typeof useExploreBlockContent>);

    renderBlock(testBlock());

    expect(await screen.findByTestId('explore-block-empty')).toHaveTextContent(
      '沒有符合條件的內容'
    );
  });

  // --- ux3-1-8: owned items leave the discovery row (frontend-only filter) ---

  it('[P1] ux3-1-8 — owned items are filtered OUT; the caption says so', async () => {
    mockHook.mockReturnValue({
      data: {
        blockId: 'block-1',
        contentType: 'movie',
        movies: [
          { id: 1, title: '已擁有的電影', posterPath: '/p1.jpg', voteAverage: 8, genreIds: [] },
          { id: 2, title: '還沒有的電影', posterPath: '/p2.jpg', voteAverage: 7, genreIds: [] },
        ],
        totalItems: 2,
      },
      isLoading: false,
      isError: false,
    } as unknown as ReturnType<typeof useExploreBlockContent>);

    renderBlock(testBlock(), {
      ...stubOwnership,
      owned: new Set<number>([1]),
      isOwned: (id) => id === 1,
    });

    expect(await screen.findByText('還沒有的電影')).toBeInTheDocument();
    expect(screen.queryByText('已擁有的電影')).toBeNull();
    expect(screen.getByTestId('explore-block-caption')).toHaveTextContent(
      '已擁有的作品不會出現在這裡'
    );
  });

  it('[P1] ux3-1-8 — while the ownership lookup is inflight the row renders UNFILTERED (no shrink flash)', async () => {
    mockHook.mockReturnValue({
      data: {
        blockId: 'block-1',
        contentType: 'movie',
        movies: [
          { id: 1, title: '已擁有的電影', posterPath: '/p1.jpg', voteAverage: 8, genreIds: [] },
          { id: 2, title: '還沒有的電影', posterPath: '/p2.jpg', voteAverage: 7, genreIds: [] },
        ],
        totalItems: 2,
      },
      isLoading: false,
      isError: false,
    } as unknown as ReturnType<typeof useExploreBlockContent>);

    renderBlock(testBlock(), {
      ...stubOwnership,
      isLoading: true,
      isSettled: false,
      owned: new Set<number>([1]),
      isOwned: (id) => id === 1,
    });

    expect(await screen.findByText('已擁有的電影')).toBeInTheDocument();
    expect(screen.getByText('還沒有的電影')).toBeInTheDocument();
    // No filter ran, so the row may not claim one did.
    expect(screen.queryByTestId('explore-block-caption')).toBeNull();
  });

  it('[P1] ux3-1-8 誠實 — a DISABLED ownership query is not a verdict: nothing filtered, no caption', async () => {
    mockHook.mockReturnValue({
      data: {
        blockId: 'block-1',
        contentType: 'movie',
        movies: [
          { id: 1, title: '已擁有的電影', posterPath: '/p1.jpg', voteAverage: 8, genreIds: [] },
          { id: 2, title: '還沒有的電影', posterPath: '/p2.jpg', voteAverage: 7, genreIds: [] },
        ],
        totalItems: 2,
      },
      isLoading: false,
      isError: false,
    } as unknown as ReturnType<typeof useExploreBlockContent>);

    // The shape a DISABLED useOwnedMedia returns while the parent's stability
    // gate blanks the id batch: not loading, no error, empty owned set. Reading
    // that as a settled verdict is the bug this guard pins (3 review lenses).
    renderBlock(testBlock(), {
      ...stubOwnership,
      isLoading: false,
      error: null,
      isSettled: false,
      owned: new Set<number>(),
      isOwned: () => false,
    });

    expect(await screen.findByText('已擁有的電影')).toBeInTheDocument();
    expect(screen.getByText('還沒有的電影')).toBeInTheDocument();
    expect(screen.queryByTestId('explore-block-caption')).toBeNull();
  });

  it('[P1] ux3-1-8 誠實 — a FAILED ownership lookup filters nothing, so the caption makes no claim', async () => {
    mockHook.mockReturnValue({
      data: {
        blockId: 'block-1',
        contentType: 'movie',
        movies: [
          { id: 1, title: '已擁有的電影', posterPath: '/p1.jpg', voteAverage: 8, genreIds: [] },
          { id: 2, title: '還沒有的電影', posterPath: '/p2.jpg', voteAverage: 7, genreIds: [] },
        ],
        totalItems: 2,
      },
      isLoading: false,
      isError: false,
    } as unknown as ReturnType<typeof useExploreBlockContent>);

    // A failed POST /media/check-owned leaves `owned` empty — isOwned answers
    // "no" for everything, so NOTHING is filtered. A standing caption would be
    // the page taking credit for work it did not do.
    renderBlock(testBlock(), {
      ...stubOwnership,
      error: new Error('check-owned failed'),
      isSettled: false,
    });

    expect(await screen.findByText('已擁有的電影')).toBeInTheDocument();
    expect(screen.getByText('還沒有的電影')).toBeInTheDocument();
    expect(screen.queryByTestId('explore-block-caption')).toBeNull();
  });

  it('[P2] ux3-1-8 — a row emptied BY the filter says so (你都已經擁有了), not 沒有符合條件的內容', async () => {
    mockHook.mockReturnValue({
      data: {
        blockId: 'block-1',
        contentType: 'movie',
        movies: [
          { id: 1, title: '已擁有的電影', posterPath: '/p1.jpg', voteAverage: 8, genreIds: [] },
        ],
        totalItems: 1,
      },
      isLoading: false,
      isError: false,
    } as unknown as ReturnType<typeof useExploreBlockContent>);

    renderBlock(testBlock(), {
      ...stubOwnership,
      owned: new Set<number>([1]),
      isOwned: (id) => id === 1,
    });

    expect(await screen.findByTestId('explore-block-empty')).toHaveTextContent(
      '這排的作品你都已經擁有了'
    );
  });

  it('returns null (hides itself) when the content query errors', async () => {
    mockHook.mockReturnValue({
      data: undefined,
      isLoading: false,
      isError: true,
    } as ReturnType<typeof useExploreBlockContent>);

    const { container } = renderBlock(testBlock());

    // Wait a tick so the router mounts, then verify the section is not rendered.
    await Promise.resolve();
    expect(container.querySelector('[data-testid^="explore-block-"]')).toBeNull();
  });

  // bugfix-10-1 Task 5.7 — verify the regression locus stays correct: an
  // ExploreBlock poster's Link MUST encode the TMDb numeric id verbatim so the
  // route's classifyId() can detect it and dispatch to the TMDb detail branch.
  //
  // CR M1 — beyond the static href shape, fire an actual click and assert the
  // router navigates AND mounts the registered media route stub. This covers
  // AC #9(e) "navigates ... AND resolves (smoke)" — the prior version only
  // verified the href attribute, which would still pass even if the Link were
  // disabled or the route were unregistered.
  it('poster card link encodes TMDb numeric id and resolves /media/$type/$id', async () => {
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

    // Smoke: actually navigate and confirm the route mounts.
    fireEvent.click(card);
    await waitFor(() => {
      expect(screen.getByText('Media Detail')).toBeInTheDocument();
    });
  });

  // bugfix-10-6 AC #2 — when the block has items, both scroll chevrons stay in
  // the DOM (hidden lg:block, new contrast styling). Touch-visibility and the
  // scroll() click handler are unchanged.
  it('renders desktop scroll chevrons when the block has items (AC #2)', async () => {
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
        ],
        totalItems: 1,
      },
      isLoading: false,
      isError: false,
    } as ReturnType<typeof useExploreBlockContent>);

    renderBlock(testBlock());

    expect(await screen.findByTestId('explore-block-scroll-left')).toBeInTheDocument();
    expect(screen.getByTestId('explore-block-scroll-right')).toBeInTheDocument();
  });

  // bugfix-10-6 AC #2 (TA pass) — beyond DOM presence, the chevron click
  // handler must still drive scrollerRef.scrollBy() with a smooth, viewport-
  // proportional delta (left = -80% width, right = +80% width). jsdom reports
  // clientWidth 0, so pin a width to make the delta deterministic.
  it('clicking a scroll chevron scrolls the scroller by 80% of its width (AC #2)', async () => {
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
        ],
        totalItems: 1,
      },
      isLoading: false,
      isError: false,
    } as ReturnType<typeof useExploreBlockContent>);

    renderBlock(testBlock());

    const scroller = await screen.findByTestId('explore-block-scroller');
    Object.defineProperty(scroller, 'clientWidth', { configurable: true, value: 800 });
    const scrollBy = vi.fn();
    Object.defineProperty(scroller, 'scrollBy', { configurable: true, value: scrollBy });

    fireEvent.click(screen.getByTestId('explore-block-scroll-left'));
    expect(scrollBy).toHaveBeenCalledWith({ left: -640, behavior: 'smooth' });

    fireEvent.click(screen.getByTestId('explore-block-scroll-right'));
    expect(scrollBy).toHaveBeenCalledWith({ left: 640, behavior: 'smooth' });
  });

  // bugfix-10-6 AC #5 — an empty block (fetched OK, zero matching results)
  // renders NO scroll chevrons, so the "沒有符合條件的內容" message at the left
  // edge can never be clipped by a chevron. The block itself still renders
  // (only isError makes ExploreBlock return null).
  it('does not render scroll chevrons when the block is empty (AC #5)', async () => {
    mockHook.mockReturnValue({
      data: { blockId: 'block-1', contentType: 'movie', movies: [], tvShows: [], totalItems: 0 },
      isLoading: false,
      isError: false,
    } as ReturnType<typeof useExploreBlockContent>);

    renderBlock(testBlock());

    expect(await screen.findByTestId('explore-block-empty')).toHaveTextContent(
      '沒有符合條件的內容'
    );
    expect(screen.queryByTestId('explore-block-scroll-left')).toBeNull();
    expect(screen.queryByTestId('explore-block-scroll-right')).toBeNull();
  });
});
