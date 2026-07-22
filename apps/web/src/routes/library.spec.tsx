/**
 * Route-level spec for /library (ux3-cutover-3).
 *
 * Test-gap root cause paid down here: the old spec HAND-COPIED the route tree
 * (validateSearch + component only), so `beforeLoad` — the ?type= → clean-route
 * redirect — never existed in tests and the 2026-07-22 type-filter bug (URL
 * moves, UI + query stay 全部) was structurally invisible. This spec mounts the
 * REAL route options (validateSearch + beforeLoad + component) plus the real
 * child path markers, and asserts URL ↔ UI ↔ query agreement end to end.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {
  createMemoryHistory,
  RouterProvider,
  createRootRoute,
  createRoute,
  createRouter,
} from '@tanstack/react-router';
import React from 'react';
import type { LibraryItem } from '../types/library';

const h = vi.hoisted(() => ({
  infinite: {} as Record<string, unknown>,
  lastArgs: undefined as Record<string, unknown> | undefined,
}));

vi.mock('../hooks/useLibraryInfinite', () => ({
  useLibraryInfinite: (args: Record<string, unknown>) => {
    h.lastArgs = args;
    return h.infinite;
  },
}));
vi.mock('../hooks/useQBittorrent', () => ({
  useQBittorrentConfig: () => ({ data: { configured: true }, isLoading: false }),
}));
vi.mock('../hooks/useMediaLibrary', () => ({
  useMediaLibraries: () => ({ data: { libraries: [{ id: '1' }] }, isLoading: false }),
}));
vi.mock('../hooks/useLibrary', () => ({
  useMovieStats: () => ({ data: { unmatchedCount: 0 } }),
  useSeriesStats: () => ({ data: { unmatchedCount: 0 } }),
  useLibraryGenres: () => ({ data: [] }),
  useBatchDelete: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useBatchReparse: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useBatchExport: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));
vi.mock('../components/subtitle/GenerationBatchDialogV2', () => ({
  GenerationBatchDialogV2: () => null,
}));

import { Route as LibraryRoute } from './library';
import { Route as LibraryIndexRoute } from './library/index';
import { Route as LibraryMoviesRoute } from './library/movies';
import { Route as LibraryTvRoute } from './library/tv';

function infinite(over: Record<string, unknown> = {}) {
  return {
    items: [] as LibraryItem[],
    totalItems: 0,
    isLoading: false,
    isError: false,
    error: null,
    fetchNextPage: vi.fn(),
    hasNextPage: false,
    isFetchingNextPage: false,
    refetch: vi.fn(),
    ...over,
  };
}

const movie = (id: string, title: string): LibraryItem => ({
  type: 'movie',
  movie: {
    id,
    title,
    releaseDate: '2020-01-01',
    runtime: 120,
    genres: ['動作'],
    parseStatus: 'success',
    subtitleTracks: JSON.stringify([{ language: 'zh-Hant' }]),
    voteAverage: 7.5,
    posterPath: null,
    createdAt: '',
    updatedAt: '',
  },
});

/**
 * Mount the REAL /library layout (validateSearch + beforeLoad + component) with
 * the real child path-marker components — the closest test double to the
 * production tree that doesn't drag in __root's shell chassis.
 */
function renderLibrary(initial: string) {
  const rootRoute = createRootRoute();
  const libraryRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/library',
    validateSearch: LibraryRoute.options.validateSearch,
    beforeLoad: LibraryRoute.options.beforeLoad,
    component: LibraryRoute.options.component,
  });
  const libraryIndex = createRoute({
    getParentRoute: () => libraryRoute,
    path: '/',
    component: LibraryIndexRoute.options.component,
  });
  const libraryMovies = createRoute({
    getParentRoute: () => libraryRoute,
    path: '/movies',
    component: LibraryMoviesRoute.options.component,
  });
  const libraryTv = createRoute({
    getParentRoute: () => libraryRoute,
    path: '/tv',
    component: LibraryTvRoute.options.component,
  });
  const detail = createRoute({
    getParentRoute: () => rootRoute,
    path: '/media/$type/$id',
    component: () => null,
  });
  const router = createRouter({
    routeTree: rootRoute.addChildren([
      libraryRoute.addChildren([libraryIndex, libraryMovies, libraryTv]),
      detail,
    ]),
    history: createMemoryHistory({ initialEntries: [initial] }),
  });
  const utils = render(React.createElement(RouterProvider, { router } as never));
  return { router, ...utils };
}

describe('routes/library — clean-route redirect (real beforeLoad)', () => {
  beforeEach(() => {
    h.infinite = infinite({ items: [movie('a', '電影甲')], totalItems: 1 });
    h.lastArgs = undefined;
  });

  it('?type=movie deep link redirects to /library/movies with type stripped', async () => {
    const { router } = renderLibrary('/library?type=movie');
    await screen.findByTestId('library-grid-v2');
    await waitFor(() => expect(router.state.location.pathname).toBe('/library/movies'));
    // the query the UI issues agrees with the URL — the 2026-07-22 bug assertion
    expect(h.lastArgs?.type).toBe('movie');
  });

  it('?type=tv deep link redirects to /library/tv', async () => {
    const { router } = renderLibrary('/library?type=tv');
    await waitFor(() => expect(router.state.location.pathname).toBe('/library/tv'));
    await waitFor(() => expect(h.lastArgs?.type).toBe('tv'));
  });

  it('?type=all (and absent) stays on the merged /library view', async () => {
    const { router } = renderLibrary('/library?type=all');
    await screen.findByTestId('library-grid-v2');
    expect(router.state.location.pathname).toBe('/library');
    expect(h.lastArgs?.type).toBe('all');
  });
});

describe('routes/library — URL ↔ UI consistency (ux3-cutover-3)', () => {
  beforeEach(() => {
    h.infinite = infinite({ items: [movie('a', '電影甲')], totalItems: 1 });
    h.lastArgs = undefined;
  });

  it('deep link /library/movies: rail 電影 is pressed AND the query asks for movies', async () => {
    renderLibrary('/library/movies');
    await screen.findByTestId('library-filter-rail');
    expect(screen.getByTestId('filter-type-movie')).toHaveAttribute('aria-pressed', 'true');
    expect(screen.getByTestId('filter-type-all')).toHaveAttribute('aria-pressed', 'false');
    expect(h.lastArgs?.type).toBe('movie');
  });

  it('clicking 電影 navigates to /library/movies and URL, button, query all agree', async () => {
    const { router } = renderLibrary('/library');
    await screen.findByTestId('library-filter-rail');
    expect(screen.getByTestId('filter-type-all')).toHaveAttribute('aria-pressed', 'true');

    await userEvent.click(screen.getByTestId('filter-type-movie'));

    await waitFor(() => expect(router.state.location.pathname).toBe('/library/movies'));
    await waitFor(() =>
      expect(screen.getByTestId('filter-type-movie')).toHaveAttribute('aria-pressed', 'true')
    );
    expect(h.lastArgs?.type).toBe('movie');
  });

  it('clicking 全部 from /library/tv returns to the merged /library view', async () => {
    const { router } = renderLibrary('/library/tv');
    await screen.findByTestId('library-filter-rail');
    expect(screen.getByTestId('filter-type-tv')).toHaveAttribute('aria-pressed', 'true');

    await userEvent.click(screen.getByTestId('filter-type-all'));

    await waitFor(() => expect(router.state.location.pathname).toBe('/library'));
    await waitFor(() => expect(h.lastArgs?.type).toBe('all'));
  });

  it('?genres= deep link flows into the query and the rail active-count', async () => {
    renderLibrary('/library?genres=動作,科幻');
    await screen.findByTestId('library-filter-rail');
    expect(h.lastArgs?.genres).toBe('動作,科幻');
    expect(screen.getByTestId('library-rail-active-count')).toHaveTextContent('2');
  });
});
