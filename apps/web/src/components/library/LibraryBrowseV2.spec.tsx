import React from 'react';
import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import {
  createRootRoute,
  createRoute,
  createRouter,
  createMemoryHistory,
  RouterProvider,
} from '@tanstack/react-router';
import type { LibraryItem } from '../../types/library';

const h = vi.hoisted(() => ({
  infinite: {} as Record<string, unknown>,
  lastArgs: undefined as Record<string, unknown> | undefined,
}));

vi.mock('../../hooks/useLibraryInfinite', () => ({
  useLibraryInfinite: (args: Record<string, unknown>) => {
    h.lastArgs = args;
    return h.infinite;
  },
}));
vi.mock('../../hooks/useQBittorrent', () => ({
  useQBittorrentConfig: () => ({ data: { configured: true }, isLoading: false }),
}));
vi.mock('../../hooks/useMediaLibrary', () => ({
  useMediaLibraries: () => ({ data: { libraries: [{ id: '1' }] }, isLoading: false }),
}));
vi.mock('../../hooks/useLibrary', () => ({
  useMovieStats: () => ({ data: { unmatchedCount: 0 } }),
  useSeriesStats: () => ({ data: { unmatchedCount: 0 } }),
  useLibraryGenres: () => ({ data: [] }),
}));

import { LibraryBrowseV2 } from './LibraryBrowseV2';

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

function renderBrowse(initial = '/library', type?: 'all' | 'movie' | 'tv') {
  const rootRoute = createRootRoute();
  const libraryRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/library',
    validateSearch: (s: Record<string, unknown>) => s,
    component: type ? () => React.createElement(LibraryBrowseV2, { type }) : LibraryBrowseV2,
  });
  const detail = createRoute({
    getParentRoute: () => rootRoute,
    path: '/media/$type/$id',
    component: () => null,
  });
  const router = createRouter({
    routeTree: rootRoute.addChildren([libraryRoute, detail]),
    history: createMemoryHistory({ initialEntries: [initial] }),
  });
  return render(React.createElement(RouterProvider, { router }));
}

describe('LibraryBrowseV2', () => {
  beforeEach(() => {
    h.infinite = infinite();
  });

  it('renders the grid + result count for a populated library', async () => {
    h.infinite = infinite({ items: [movie('a', '電影甲'), movie('b', '電影乙')], totalItems: 2 });
    renderBrowse();
    expect(await screen.findByTestId('library-grid-v2')).toBeInTheDocument();
    expect(screen.getByTestId('library-result-count')).toHaveTextContent('2 項');
    expect(screen.getByTestId('poster-v2-a')).toBeInTheDocument();
  });

  it('shows the loading skeleton while loading', async () => {
    h.infinite = infinite({ isLoading: true });
    renderBrowse();
    expect(await screen.findByTestId('library-grid-skeleton')).toBeInTheDocument();
  });

  it('shows the fail-soft error state on a non-ok response', async () => {
    h.infinite = infinite({ isError: true, error: { code: 'DB_QUERY_FAILED' } });
    renderBrowse();
    expect(await screen.findByTestId('library-error')).toHaveTextContent('DB_QUERY_FAILED');
  });

  it('shows no-result (not empty) when a filter returns nothing', async () => {
    h.infinite = infinite({ items: [], totalItems: 0 });
    renderBrowse('/library?genres=動作');
    expect(await screen.findByTestId('library-no-result')).toBeInTheDocument();
  });

  it('renders the list view when ?view=list', async () => {
    h.infinite = infinite({ items: [movie('a', '電影甲')], totalItems: 1 });
    renderBrowse('/library?view=list');
    expect(await screen.findByTestId('library-list-v2')).toBeInTheDocument();
    expect(screen.getByTestId('list-row-v2-a')).toBeInTheDocument();
  });

  it('queries the type the layout passes (clean route /library/movies → ux3-0-5)', async () => {
    h.infinite = infinite({ items: [movie('a', '電影甲')], totalItems: 1 });
    renderBrowse('/library', 'movie');
    await screen.findByTestId('library-grid-v2');
    expect(h.lastArgs?.type).toBe('movie');
  });
});
