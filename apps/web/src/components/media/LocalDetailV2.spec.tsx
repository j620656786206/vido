import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import {
  createRootRoute,
  createRoute,
  createRouter,
  createMemoryHistory,
  RouterProvider,
} from '@tanstack/react-router';

const h = vi.hoisted(() => ({ local: {} as Record<string, unknown> }));

vi.mock('../../hooks/useMediaDetails', () => ({
  useLocalMovieDetails: () => h.local,
  useLocalSeriesDetails: () => ({
    data: undefined,
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  }),
  useMovieCredits: () => ({ data: undefined }),
  useTVShowCredits: () => ({ data: undefined }),
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
vi.mock('./CreditsSection', () => ({ CreditsSection: () => null }));
vi.mock('./DualRatingDisplay', () => ({ DualRatingDisplay: () => null }));
vi.mock('../metadata-editor', () => ({ MetadataEditorDialog: () => null }));
vi.mock('../subtitle/SubtitleSearchDialog', () => ({
  SubtitleSearchDialog: ({ open }: { open: boolean }) =>
    open ? <div data-testid="subtitle-dialog" /> : null,
}));

import { LocalDetailV2 } from './LocalDetailV2';

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

function renderDetail() {
  const rootRoute = createRootRoute({
    component: () => React.createElement(LocalDetailV2, { type: 'movie', id: 'abc' }),
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

  it('opens the subtitle dialog from 管理字幕', async () => {
    renderDetail();
    fireEvent.click(await screen.findByTestId('action-manage-subtitle'));
    expect(await screen.findByTestId('subtitle-dialog')).toBeInTheDocument();
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
});
