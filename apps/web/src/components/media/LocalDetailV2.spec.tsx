import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import {
  createRootRoute,
  createRoute,
  createRouter,
  createMemoryHistory,
  RouterProvider,
} from '@tanstack/react-router';

const h = vi.hoisted(() => ({
  local: {} as Record<string, unknown>,
  localSeries: undefined as Record<string, unknown> | undefined,
  movieCredits: { data: undefined } as { data: unknown },
  tvCredits: { data: undefined } as { data: unknown },
}));

vi.mock('../../hooks/useMediaDetails', async (importOriginal) => ({
  // keep the REAL detailKeys (AC 6 invalidation asserts against them)
  detailKeys: (await importOriginal<typeof import('../../hooks/useMediaDetails')>()).detailKeys,
  useLocalMovieDetails: () => h.local,
  useLocalSeriesDetails: () =>
    h.localSeries ?? { data: undefined, isLoading: false, isError: false, refetch: vi.fn() },
  useMovieCredits: () => h.movieCredits,
  useTVShowCredits: () => h.tvCredits,
  useSeriesSeasons: () => ({ data: [], isLoading: false, isError: false, refetch: vi.fn() }),
  useRecommendations: () => ({
    data: { results: [] },
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  }),
  useWatchProviders: () => ({
    data: { results: {} },
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  }),
}));
vi.mock('../../hooks/useDoubanRating', () => ({
  useDoubanRating: () => ({ data: null, isLoading: false }),
}));
vi.mock('../../hooks/useDoubanReviewSummary', () => ({
  useDoubanReviewSummary: () => ({ data: null, isLoading: false, isError: false }),
}));
// Stub the heavy / self-fetching section + dialog children.
vi.mock('./TrailerSection', () => ({ TrailerSection: () => null }));
vi.mock('./StreamingAvailability', () => ({ StreamingAvailability: () => null }));
vi.mock('./RelatedContent', () => ({ RelatedContent: () => null }));
vi.mock('./SeasonAccordion', () => ({ SeasonAccordion: () => null }));
vi.mock('./DoubanSection', () => ({ DoubanSection: () => null }));
vi.mock('./CreditsSection', () => ({
  CreditsSection: ({ cast }: { cast?: Array<{ name: string }> }) => (
    <div data-testid="stub-credits-cast">{(cast ?? []).map((c) => c.name).join(',')}</div>
  ),
}));
vi.mock('./DualRatingDisplay', () => ({ DualRatingDisplay: () => null }));
vi.mock('../metadata-editor', () => ({ MetadataEditorDialog: () => null }));
// v2 shell swap (ux3-subtitle-v2 Task 6): LocalDetailV2 renders the NEW
// ManageSubtitleDialogV2 (the v1 SubtitleSearchDialog file stays for the legacy shell).
vi.mock('../subtitle/ManageSubtitleDialogV2', () => ({
  ManageSubtitleDialogV2: ({
    open,
    productionCountry,
    onGenerationComplete,
  }: {
    open: boolean;
    productionCountry?: string;
    onGenerationComplete?: () => void;
  }) =>
    open ? (
      <div data-testid="subtitle-dialog">
        <span data-testid="stub-production-country">{productionCountry}</span>
        <button
          type="button"
          data-testid="stub-generation-complete"
          onClick={() => onGenerationComplete?.()}
        />
      </div>
    ) : null,
}));

import { LocalDetailV2 } from './LocalDetailV2';
import { detailKeys } from '../../hooks/useMediaDetails';
import { libraryKeys } from '../../hooks/useLibrary';

function movie(over: Record<string, unknown> = {}) {
  return {
    data: {
      id: 'abc',
      title: '你的名字',
      originalTitle: '君の名は。',
      releaseDate: '2016-08-26',
      runtime: 106,
      genres: ['動畫', '劇情'],
      voteAverage: 8.5,
      voteCount: 100,
      overview: '兩個陌生少年少女的奇妙相遇。',
      posterPath: null,
      backdropPath: null,
      tmdbId: 0,
      parseStatus: 'success',
      subtitleTracks: JSON.stringify([{ language: 'zh-Hant' }]),
      filePath: '/media/movies/yourname.mkv',
      videoResolution: '1080p',
      videoCodec: 'HEVC',
      audioCodec: 'DTS',
      fileSize: 3 * 1024 ** 3,
      createdAt: '',
      updatedAt: '',
    },
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
    ...over,
  };
}

function renderDetail(
  queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
) {
  const rootRoute = createRootRoute({
    component: () =>
      React.createElement(
        QueryClientProvider,
        { client: queryClient },
        React.createElement(LocalDetailV2, { type: 'movie', id: 'abc' })
      ),
  });
  const library = createRoute({
    getParentRoute: () => rootRoute,
    path: '/library',
    component: () => null,
  });
  const router = createRouter({
    routeTree: rootRoute.addChildren([library]),
    history: createMemoryHistory({ initialEntries: ['/'] }),
  });
  return render(React.createElement(RouterProvider, { router }));
}

function series(over: Record<string, unknown> = {}) {
  return {
    data: {
      id: 'sid',
      title: '進擊的巨人',
      originalTitle: '進撃の巨人',
      firstAirDate: '2013-04-07',
      genres: ['動畫'],
      numberOfSeasons: 4,
      numberOfEpisodes: 87,
      tmdbId: 0,
      parseStatus: 'success',
      filePath: '/media/series/aot',
      createdAt: '',
      updatedAt: '',
    },
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
    ...over,
  };
}

function renderSeriesDetail(
  queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
) {
  const rootRoute = createRootRoute({
    component: () =>
      React.createElement(
        QueryClientProvider,
        { client: queryClient },
        React.createElement(LocalDetailV2, { type: 'series', id: 'sid' })
      ),
  });
  const library = createRoute({
    getParentRoute: () => rootRoute,
    path: '/library',
    component: () => null,
  });
  const router = createRouter({
    routeTree: rootRoute.addChildren([library]),
    history: createMemoryHistory({ initialEntries: ['/'] }),
  });
  return render(React.createElement(RouterProvider, { router }));
}

describe('LocalDetailV2', () => {
  beforeEach(() => {
    h.local = movie();
    h.localSeries = undefined;
    h.movieCredits = { data: undefined };
    h.tvCredits = { data: undefined };
  });

  it('renders the hero, overview and tech-info for a library item', async () => {
    renderDetail();
    expect(await screen.findByTestId('local-detail-v2')).toBeInTheDocument();
    expect(screen.getByTestId('detail-hero-v2')).toHaveTextContent('你的名字');
    expect(screen.getByTestId('detail-overview')).toBeInTheDocument();
    expect(screen.getByTestId('detail-tech-info')).toHaveTextContent('1080p');
  });

  it('surfaces the resolved CTAs — 管理字幕 (primary) + 修改資訊 + 複製路徑, no 播放', async () => {
    renderDetail();
    await screen.findByTestId('local-detail-v2');
    expect(screen.getByTestId('action-manage-subtitle')).toHaveTextContent('管理字幕');
    expect(screen.getByTestId('action-edit-metadata')).toHaveTextContent('修改資訊');
    expect(screen.getByTestId('action-copy-path')).toBeInTheDocument();
    expect(screen.queryByText('播放')).not.toBeInTheDocument();
  });

  it('opens the v2 manage-subtitle dialog from 管理字幕', async () => {
    renderDetail();
    fireEvent.click(await screen.findByTestId('action-manage-subtitle'));
    expect(await screen.findByTestId('subtitle-dialog')).toBeInTheDocument();
  });

  it('passes production countries to the dialog as a comma-joined ISO string (§9b source)', async () => {
    h.local = movie({
      data: {
        ...movie().data,
        productionCountries: [
          { iso31661: 'CN', name: 'China' },
          { iso31661: 'US', name: 'United States of America' },
        ],
      },
    });
    renderDetail();
    fireEvent.click(await screen.findByTestId('action-manage-subtitle'));
    expect(await screen.findByTestId('stub-production-country')).toHaveTextContent('CN,US');
  });

  it('passes an empty production-country string when the movie has none', async () => {
    renderDetail();
    fireEvent.click(await screen.findByTestId('action-manage-subtitle'));
    expect(await screen.findByTestId('stub-production-country')).toHaveTextContent('');
  });

  it('invalidates media-detail + library caches on transcription_complete (AC 6)', async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries');
    renderDetail(queryClient);

    fireEvent.click(await screen.findByTestId('action-manage-subtitle'));
    fireEvent.click(await screen.findByTestId('stub-generation-complete'));

    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: detailKeys.localMovie('abc') });
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: libraryKeys.all });
  });

  it('hides 管理字幕 / 複製路徑 when the item has no local filePath', async () => {
    h.local = movie({ data: { ...movie().data, filePath: undefined } });
    renderDetail();
    await screen.findByTestId('local-detail-v2');
    expect(screen.queryByTestId('action-manage-subtitle')).not.toBeInTheDocument();
    expect(screen.queryByTestId('action-copy-path')).not.toBeInTheDocument();
    expect(screen.getByTestId('action-edit-metadata')).toBeInTheDocument();
  });

  it('shows the skeleton while loading and not-found on error', async () => {
    h.local = movie({ data: undefined, isLoading: true });
    const { unmount } = renderDetail();
    expect(await screen.findByTestId('detail-skeleton')).toBeInTheDocument();
    unmount();

    h.local = movie({ data: undefined, isLoading: false, isError: true });
    renderDetail();
    expect(await screen.findByTestId('detail-not-found')).toBeInTheDocument();
  });

  // disc-2026-07-credits-spoken-languages-persist: the cast display prefers the persisted
  // local credits when the item was manually edited, else falls back to live TMDb.
  it('prefers manually-edited local credits over live TMDb (movie, metadataSource=manual)', async () => {
    h.local = movie({
      data: {
        ...movie().data,
        metadataSource: 'manual',
        credits: { cast: [{ name: 'ManualActor' }] },
      },
    });
    h.movieCredits = { data: { cast: [{ name: 'TMDbActor' }] } };
    renderDetail();
    const el = await screen.findByTestId('stub-credits-cast');
    expect(el).toHaveTextContent('ManualActor');
    expect(el).not.toHaveTextContent('TMDbActor');
  });

  it('falls back to live TMDb credits when the movie is not manually edited', async () => {
    h.local = movie({ data: { ...movie().data, metadataSource: 'tmdb' } });
    h.movieCredits = { data: { cast: [{ name: 'TMDbActor' }] } };
    renderDetail();
    expect(await screen.findByTestId('stub-credits-cast')).toHaveTextContent('TMDbActor');
  });

  it('prefers manually-edited local credits over live TMDb (series, metadataSource=manual)', async () => {
    h.localSeries = series({
      data: {
        ...series().data,
        metadataSource: 'manual',
        credits: { cast: [{ name: 'ManualSeriesActor' }] },
      },
    });
    h.tvCredits = { data: { cast: [{ name: 'TMDbSeriesActor' }] } };
    renderSeriesDetail();
    const el = await screen.findByTestId('stub-credits-cast');
    expect(el).toHaveTextContent('ManualSeriesActor');
    expect(el).not.toHaveTextContent('TMDbSeriesActor');
  });

  it('falls back to live TMDb credits for a non-manual series', async () => {
    h.localSeries = series({ data: { ...series().data, metadataSource: 'tmdb' } });
    h.tvCredits = { data: { cast: [{ name: 'TMDbSeriesActor' }] } };
    renderSeriesDetail();
    expect(await screen.findByTestId('stub-credits-cast')).toHaveTextContent('TMDbSeriesActor');
  });
});
