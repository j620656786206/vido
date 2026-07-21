import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
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
  batchDelete: undefined as unknown as ReturnType<typeof vi.fn>,
  batchReparse: undefined as unknown as ReturnType<typeof vi.fn>,
  batchExport: undefined as unknown as ReturnType<typeof vi.fn>,
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
  // ux3-cutover-2: selection-mode batch mutations (spies live in `h` so tests
  // can assert the ids/type each batch call receives)
  useBatchDelete: () => ({ mutateAsync: h.batchDelete, isPending: false }),
  useBatchReparse: () => ({ mutateAsync: h.batchReparse, isPending: false }),
  useBatchExport: () => ({ mutateAsync: h.batchExport, isPending: false }),
}));
// ux3-cutover-2: the generation-batch dialog pulls SSE plumbing — stub it out.
vi.mock('../subtitle/GenerationBatchDialogV2', () => ({
  GenerationBatchDialogV2: ({ open }: { open: boolean }) =>
    open ? <div data-testid="generation-batch-dialog-stub" /> : null,
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
    h.batchDelete = vi.fn().mockResolvedValue({ successCount: 0, errors: [] });
    h.batchReparse = vi.fn().mockResolvedValue({ successCount: 0, errors: [] });
    h.batchExport = vi.fn().mockResolvedValue({ items: [] });
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

describe('LibraryBrowseV2 — desktop filter rail (ux3-0-7)', () => {
  beforeEach(() => {
    h.infinite = infinite({ items: [movie('a', '電影甲')], totalItems: 1 });
    try {
      localStorage.clear();
    } catch {
      /* ignore */
    }
  });

  it('renders the persistent filter rail by default', async () => {
    renderBrowse();
    expect(await screen.findByTestId('library-filter-rail')).toBeInTheDocument();
    // grid uses the rail-open column class (narrower lg columns)
    expect(screen.getByTestId('library-grid-v2').className).toContain('lg:grid-cols-3');
  });

  it('collapse → rail hidden, 篩選 re-open button shown, grid reflows wider', async () => {
    renderBrowse();
    await screen.findByTestId('library-filter-rail');
    await userEvent.click(screen.getByTestId('library-rail-collapse'));
    expect(screen.queryByTestId('library-filter-rail')).not.toBeInTheDocument();
    expect(screen.getByTestId('library-rail-expand')).toBeInTheDocument();
    expect(screen.getByTestId('library-grid-v2').className).toContain('lg:grid-cols-4');
  });

  it('re-expand restores the rail', async () => {
    renderBrowse();
    await userEvent.click(await screen.findByTestId('library-rail-collapse'));
    await userEvent.click(screen.getByTestId('library-rail-expand'));
    expect(await screen.findByTestId('library-filter-rail')).toBeInTheDocument();
  });

  it('rail active-count counts genres + decade-range as one, not type', async () => {
    renderBrowse('/library?genres=動作,科幻&yearMin=2020&yearMax=2029');
    // 2 genres + 1 decade range = 3 (type=全部 not counted)
    expect(await screen.findByTestId('library-rail-active-count')).toHaveTextContent('3');
  });
});

// ux3-cutover-2: v2 selection mode + batch ops (legacy-shell deletion gate)
describe('LibraryBrowseV2 — selection mode (ux3-cutover-2)', () => {
  beforeEach(() => {
    h.infinite = infinite({
      items: [movie('a', '電影甲'), movie('b', '電影乙'), movie('c', '電影丙')],
      totalItems: 3,
    });
    h.batchDelete = vi.fn().mockResolvedValue({ successCount: 2, errors: [] });
    h.batchReparse = vi.fn().mockResolvedValue({ successCount: 1, errors: [] });
    h.batchExport = vi.fn().mockResolvedValue({ items: [] });
  });

  it('選取 enters selection mode: toolbar swaps in, cards stop navigating and toggle', async () => {
    renderBrowse();
    await userEvent.click(await screen.findByTestId('enter-selection-btn'));
    expect(screen.getByTestId('selection-toolbar')).toBeInTheDocument();
    expect(screen.queryByTestId('library-result-count')).not.toBeInTheDocument();

    await userEvent.click(screen.getByTestId('poster-v2-a'));
    expect(screen.getByTestId('selected-count')).toHaveTextContent('已選取 1 項');
    expect(screen.getByTestId('poster-v2-a')).toHaveAttribute('aria-pressed', 'true');
    // still on /library — the card click did NOT navigate to the detail route
    expect(screen.getByTestId('library-grid-v2')).toBeInTheDocument();

    // toggle off
    await userEvent.click(screen.getByTestId('poster-v2-a'));
    expect(screen.getByTestId('selected-count')).toHaveTextContent('已選取 0 項');
  });

  it('全選 selects all loaded items; 取消 exits and clears', async () => {
    renderBrowse();
    await userEvent.click(await screen.findByTestId('enter-selection-btn'));
    await userEvent.click(screen.getByTestId('select-all-btn'));
    expect(screen.getByTestId('selected-count')).toHaveTextContent('已選取 3 項');

    await userEvent.click(screen.getByTestId('batch-cancel-btn'));
    expect(screen.queryByTestId('selection-toolbar')).not.toBeInTheDocument();
    expect(screen.getByTestId('enter-selection-btn')).toBeInTheDocument();
  });

  it('Escape exits selection mode', async () => {
    renderBrowse();
    await userEvent.click(await screen.findByTestId('enter-selection-btn'));
    await userEvent.keyboard('{Escape}');
    expect(screen.queryByTestId('selection-toolbar')).not.toBeInTheDocument();
  });

  it('batch delete: confirm dialog → mutation receives the selected ids + type', async () => {
    renderBrowse();
    await userEvent.click(await screen.findByTestId('enter-selection-btn'));
    await userEvent.click(screen.getByTestId('poster-v2-a'));
    await userEvent.click(screen.getByTestId('poster-v2-b'));
    await userEvent.click(screen.getByTestId('batch-delete-btn'));
    expect(screen.getByTestId('batch-confirm-dialog')).toBeInTheDocument();
    await userEvent.click(screen.getByTestId('confirm-action-btn'));
    expect(h.batchDelete).toHaveBeenCalledWith({ ids: ['a', 'b'], type: 'movie' });
  });

  it('批次生成字幕 opens the generation-batch dialog with the movie selection', async () => {
    renderBrowse();
    await userEvent.click(await screen.findByTestId('enter-selection-btn'));
    await userEvent.click(screen.getByTestId('poster-v2-c'));
    await userEvent.click(screen.getByTestId('batch-subtitle-btn'));
    expect(await screen.findByTestId('generation-batch-dialog-stub')).toBeInTheDocument();
  });

  it('list view rows toggle selection too', async () => {
    renderBrowse('/library?view=list');
    await userEvent.click(await screen.findByTestId('enter-selection-btn'));
    await userEvent.click(screen.getByTestId('list-row-v2-b'));
    expect(screen.getByTestId('selected-count')).toHaveTextContent('已選取 1 項');
    expect(screen.getByTestId('list-row-v2-b')).toHaveAttribute('aria-pressed', 'true');
  });
});
