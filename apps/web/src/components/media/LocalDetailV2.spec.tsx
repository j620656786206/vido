import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { ApiError } from '../../lib/apiError';
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
  douban: { data: null, isLoading: false } as { data: unknown; isLoading: boolean },
  reparse: {} as Record<string, unknown>,
}));

// dsr-2b-b: the re-match mutation is the container's; stub it so the no-metadata
// block's states can be driven directly.
vi.mock('../../hooks/useLibrary', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../../hooks/useLibrary')>()),
  useReparseItem: () => h.reparse,
}));
vi.mock('./ManualMatchDialogV2', () => ({
  ManualMatchDialogV2: ({
    open,
    mediaType,
    initialQuery,
  }: {
    open: boolean;
    mediaType: string;
    initialQuery: string;
  }) =>
    open ? (
      <div data-testid="stub-manual-match" data-media-type={mediaType}>
        {initialQuery}
      </div>
    ) : null,
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
  useDoubanRating: () => h.douban,
}));
vi.mock('../../hooks/useDoubanReviewSummary', () => ({
  useDoubanReviewSummary: () => ({ data: null, isLoading: false, isError: false }),
}));
// Stub the heavy / self-fetching section + dialog children.
// Each stub leaves a marker so the section ORDER can be asserted (dsr-2 AC #9).
vi.mock('./TrailerSection', () => ({ TrailerSection: () => <div data-testid="stub-trailer" /> }));
vi.mock('./StreamingAvailability', () => ({
  StreamingAvailability: () => <div data-testid="stub-streaming" />,
}));
vi.mock('./RelatedContent', () => ({ RelatedContent: () => <div data-testid="stub-related" /> }));
vi.mock('./SeasonAccordion', () => ({ SeasonAccordion: () => <div data-testid="stub-seasons" /> }));
vi.mock('./DoubanSection', () => ({ DoubanSection: () => <div data-testid="stub-douban" /> }));
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
    h.douban = { data: null, isLoading: false };
    h.local = movie();
    h.localSeries = undefined;
    h.movieCredits = { data: undefined };
    h.tvCredits = { data: undefined };
    h.reparse = {
      mutate: vi.fn(),
      isPending: false,
      data: undefined,
      variables: undefined,
      error: null,
    };
  });

  // dsr-2b-b AC #2 / #3 / #5: the state is read from parseStatus, never from
  // tmdbId — a Douban/NFO match or a manual edit has no tmdb id and IS matched.
  describe('no metadata', () => {
    const withStatus = (parseStatus: string, extra: Record<string, unknown> = {}) =>
      movie({ data: { ...movie().data, parseStatus, overview: undefined, ...extra } });

    it('failed → the 比對失敗 block, first thing under the hero', async () => {
      h.local = withStatus('failed', { title: '[Leopard-Raws] Kimi no Na wa (BD).mkv' });
      renderDetail();
      const block = await screen.findByTestId('detail-no-metadata');
      expect(block).toHaveAttribute('data-variant', 'failed');
      expect(block).toHaveTextContent('沒有找到這部電影的資料');
      const body = block.parentElement!;
      expect(body.firstElementChild).toBe(block);
    });

    it('pending → the 資料整理中 block', async () => {
      h.local = withStatus('pending');
      renderDetail();
      expect(await screen.findByTestId('detail-no-metadata')).toHaveAttribute(
        'data-variant',
        'pending'
      );
    });

    it.each([
      ['empty status (API-created, usually already matched)', ''],
      ['success without a TMDb id (Douban / NFO / manual)', 'success'],
    ])('%s → no block', async (_label, status) => {
      h.local = withStatus(status, { tmdbId: 0 });
      renderDetail();
      await screen.findByTestId('local-detail-v2');
      expect(screen.queryByTestId('detail-no-metadata')).not.toBeInTheDocument();
    });

    it('hides 在地化資訊 (it would translate the file name into the NFO) and demotes 管理字幕', async () => {
      h.local = withStatus('failed'); // filePath present — the action would otherwise render
      renderDetail();
      await screen.findByTestId('detail-no-metadata');
      expect(screen.queryByTestId('action-localize-nfo')).not.toBeInTheDocument();
      expect(screen.getByTestId('action-manage-subtitle').className).not.toMatch(
        /--accent-primary/
      );
      const solid = Array.from(document.querySelectorAll('button')).filter((b) =>
        /bg-\[var\(--accent-primary\)\]/.test(b.className)
      );
      expect(solid.map((b) => b.getAttribute('data-testid'))).toEqual(['no-metadata-manual-match']);
    });

    it('a matched item keeps 在地化資訊 and the primary 管理字幕', async () => {
      renderDetail();
      await screen.findByTestId('local-detail-v2');
      expect(screen.getByTestId('action-localize-nfo')).toBeInTheDocument();
      expect(screen.getByTestId('action-manage-subtitle').className).toMatch(/--accent-primary/);
    });

    it('手動選片 opens the dialog, locked to this item, prefilled from the FILE name', async () => {
      h.local = withStatus('failed', {
        title: '使用者改過的片名',
        filePath: '/volume1/Movies/[Leopard-Raws] Kimi no Na wa (BD).mkv',
      });
      renderDetail();
      fireEvent.click(await screen.findByTestId('no-metadata-manual-match'));
      const dialog = await screen.findByTestId('stub-manual-match');
      expect(dialog).toHaveAttribute('data-media-type', 'movie');
      expect(dialog).toHaveTextContent(/^Kimi no Na wa$/);
    });

    it('a series is prefilled from its title — its file path is a folder, maybe the library root', async () => {
      h.localSeries = series({
        data: {
          ...series().data,
          parseStatus: 'failed',
          title: 'The Last of Us',
          filePath: '/volume1/TV',
        },
      });
      renderSeriesDetail();
      fireEvent.click(await screen.findByTestId('no-metadata-manual-match'));
      const dialog = await screen.findByTestId('stub-manual-match');
      expect(dialog).toHaveAttribute('data-media-type', 'series');
      expect(dialog).toHaveTextContent(/^The Last of Us$/);
    });

    it('a pending item whose re-match just came back failed does not yet say "still not found"', async () => {
      // The refetch has not landed: the page still reads pending.
      h.local = withStatus('pending');
      h.reparse = {
        mutate: vi.fn(),
        isPending: false,
        data: { id: 'abc', parseStatus: 'failed', title: 'x', tmdbId: 0 },
        variables: { type: 'movie', id: 'abc' },
        error: null,
      };
      renderDetail();
      expect(await screen.findByTestId('detail-no-metadata')).toHaveAttribute(
        'data-variant',
        'pending'
      );
      expect(screen.queryByRole('status')).not.toBeInTheDocument();
    });

    it('without a file path the prefill falls back to the title', async () => {
      h.local = withStatus('failed', { title: 'zzqx.nonsense.1080p', filePath: undefined });
      renderDetail();
      fireEvent.click(await screen.findByTestId('no-metadata-manual-match'));
      expect(await screen.findByTestId('stub-manual-match')).toHaveTextContent(/^zzqx nonsense$/);
    });

    it('re-match runs for THIS item and reports "still not found"', async () => {
      h.local = withStatus('failed');
      const mutate = vi.fn();
      h.reparse = {
        mutate,
        isPending: false,
        data: { id: 'abc', parseStatus: 'failed', title: 'x', tmdbId: 0 },
        variables: { type: 'movie', id: 'abc' },
        error: null,
      };
      renderDetail();
      fireEvent.click(await screen.findByTestId('no-metadata-rematch'));
      expect(mutate).toHaveBeenCalledWith({ type: 'movie', id: 'abc' });
      expect(screen.getByRole('status')).toHaveTextContent('重新比對完成，還是沒有找到。');
    });

    it('a re-match result for ANOTHER item says nothing here', async () => {
      h.local = withStatus('failed');
      h.reparse = {
        mutate: vi.fn(),
        isPending: false,
        data: { id: 'other', parseStatus: 'failed', title: 'x', tmdbId: 0 },
        variables: { type: 'movie', id: 'other' },
        error: new ApiError('busy', 409, 'ENRICHMENT_ALREADY_RUNNING'),
      };
      renderDetail();
      await screen.findByTestId('detail-no-metadata');
      expect(screen.queryByRole('status')).not.toBeInTheDocument();
      expect(screen.queryByRole('alert')).not.toBeInTheDocument();
    });

    it('while the re-match runs the block shows it', async () => {
      h.local = withStatus('pending');
      h.reparse = {
        mutate: vi.fn(),
        isPending: true,
        data: undefined,
        variables: { type: 'movie', id: 'abc' },
        error: null,
      };
      renderDetail();
      expect(await screen.findByTestId('no-metadata-rematch')).toHaveTextContent('比對中…');
    });
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

  it('shows the skeleton while loading and not-found on a 404', async () => {
    h.local = movie({ data: undefined, isLoading: true });
    const { unmount } = renderDetail();
    expect(await screen.findByTestId('detail-skeleton')).toBeInTheDocument();
    unmount();

    h.local = movie({
      data: undefined,
      isLoading: false,
      isError: true,
      error: new ApiError('Movie not found', 404, 'DB_NOT_FOUND'),
    });
    renderDetail();
    expect(await screen.findByTestId('detail-not-found')).toBeInTheDocument();
    expect(screen.queryByTestId('detail-load-error')).not.toBeInTheDocument();
  });

  // Review #2: React Query keeps cached data when a BACKGROUND refetch fails (e.g. the
  // refetch after subtitle generation hits a locked DB). The loaded page — and any
  // open dialog — must stay; only a load with nothing to show takes over the page.
  it('keeps the loaded page when a background refetch fails', async () => {
    h.local = movie({
      isError: true,
      error: new ApiError('Failed to load movie', 500, 'DB_QUERY_FAILED'),
    });
    renderDetail();
    expect(await screen.findByTestId('local-detail-v2')).toBeInTheDocument();
    expect(screen.queryByTestId('detail-load-error')).not.toBeInTheDocument();
    expect(screen.queryByTestId('detail-not-found')).not.toBeInTheDocument();
  });

  it('series: a failed load shows the load error and retries the SERIES query', async () => {
    const refetch = vi.fn();
    h.localSeries = series({
      data: undefined,
      isError: true,
      error: new ApiError('Failed to load series', 500, 'DB_QUERY_FAILED'),
      refetch,
    });
    renderSeriesDetail();
    await screen.findByTestId('detail-load-error');
    fireEvent.click(screen.getByTestId('detail-load-error-retry'));
    expect(refetch).toHaveBeenCalledTimes(1);
    expect(h.local.refetch).not.toHaveBeenCalled();
  });

  // dsr-2 AC #8: a server failure used to read 「找不到這部影片，可能已被移除」.
  it('shows a load error — not not-found — when the request fails for any other reason', async () => {
    const refetch = vi.fn();
    h.local = movie({
      data: undefined,
      isLoading: false,
      isError: true,
      error: new ApiError('Failed to load movie', 500, 'DB_QUERY_FAILED'),
      refetch,
    });
    renderDetail();
    const panel = await screen.findByTestId('detail-load-error');
    expect(screen.queryByTestId('detail-not-found')).not.toBeInTheDocument();
    // A library item: the files on disk were never touched, so say so.
    expect(panel).toHaveTextContent('你的檔案沒有受影響');
    expect(screen.getByTestId('detail-load-error-code')).toHaveTextContent('DB_QUERY_FAILED');
    fireEvent.click(screen.getByTestId('detail-load-error-retry'));
    expect(refetch).toHaveBeenCalledTimes(1);
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

  // sub-6-9 — TMDB API Terms of Use §3. StreamingAvailability is stubbed to
  // null in this file, which is exactly the point: the notice must not depend
  // on that section rendering.
  it('shows the TMDB attribution on a matched title', async () => {
    h.local = movie({ data: { ...movie().data, tmdbId: 550 } });
    renderDetail();
    expect(await screen.findByTestId('tmdb-attribution')).toHaveTextContent('資料來源：');
  });

  it('omits the TMDB attribution for an unmatched local file (no TMDB data shown)', async () => {
    h.local = movie(); // fixture default is tmdbId: 0
    renderDetail();
    await screen.findByTestId('local-detail-v2');
    expect(screen.queryByTestId('tmdb-attribution')).not.toBeInTheDocument();
  });

  // dsr-2 AC #9: section order follows B3p-D / B4p-D / B8p-D (and ux2-3 AC #4).
  describe('section order', () => {
    function order(ids: string[]) {
      const nodes = ids.map((id) => screen.getByTestId(id));
      for (let i = 1; i < nodes.length; i++) {
        // DOCUMENT_POSITION_FOLLOWING (4): nodes[i] comes after nodes[i - 1].
        expect(nodes[i - 1].compareDocumentPosition(nodes[i]) & 4).toBe(4);
      }
    }

    it('movie: 簡介 → 演員 → 檔案資訊 → 預告片 → 觀看平台 → 相關推薦 → 豆瓣', async () => {
      h.local = movie({ data: { ...movie().data, tmdbId: 27205 } });
      h.movieCredits = { data: { cast: [{ id: 1, name: '神木隆之介' }], crew: [] } };
      h.douban = { data: { doubanId: '26683290' }, isLoading: false };
      renderDetail();
      await screen.findByTestId('local-detail-v2');
      order([
        'detail-overview',
        'stub-credits-cast',
        'detail-tech-info',
        'stub-trailer',
        'stub-streaming',
        'stub-related',
        'stub-douban',
      ]);
    });

    it('series: 簡介 → 季與劇集 → 檔案資訊 → 演員 (tech info stays above cast)', async () => {
      h.localSeries = series({
        data: { ...series().data, tmdbId: 1429, overview: '人類與巨人。', videoCodec: 'HEVC' },
      });
      h.tvCredits = { data: { cast: [{ id: 1, name: '梶裕貴' }], crew: [] } };
      renderSeriesDetail();
      await screen.findByTestId('local-detail-v2');
      order(['detail-overview', 'stub-seasons', 'detail-tech-info', 'stub-credits-cast']);
    });
  });
});
