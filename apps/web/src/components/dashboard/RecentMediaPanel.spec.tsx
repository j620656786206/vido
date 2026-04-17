import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import {
  createMemoryHistory,
  RouterProvider,
  createRootRoute,
  createRoute,
  createRouter,
} from '@tanstack/react-router';
import React from 'react';
import { RecentMediaPanel } from './RecentMediaPanel';

vi.mock('../../hooks/useDashboardData', () => ({
  useRecentMedia: vi.fn(),
}));

import { useRecentMedia } from '../../hooks/useDashboardData';

const mockUseRecentMedia = vi.mocked(useRecentMedia);

function renderPanel(props: { hideWhenEmpty?: boolean } = {}) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  const rootRoute = createRootRoute({
    component: () => React.createElement(RecentMediaPanel, props),
  });
  const libraryRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/library',
    component: () => React.createElement('div', null, 'Library'),
  });
  const mediaRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/media/$id',
    component: () => React.createElement('div', null, 'Media Detail'),
  });

  const routeTree = rootRoute.addChildren([libraryRoute, mediaRoute]);
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

const mockMedia = [
  {
    id: 'movie-1',
    title: '測試電影',
    year: 2024,
    posterUrl: 'https://image.tmdb.org/t/p/w342/poster1.jpg',
    mediaType: 'movie' as const,
    justAdded: true,
    addedAt: '2026-02-10T10:00:00Z',
  },
  {
    id: 'series-1',
    title: '測試影集',
    year: 2023,
    posterUrl: 'https://image.tmdb.org/t/p/w342/poster2.jpg',
    mediaType: 'tv' as const,
    justAdded: false,
    addedAt: '2026-02-10T09:00:00Z',
  },
];

describe('RecentMediaPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('[P1] renders panel title', async () => {
    mockUseRecentMedia.mockReturnValue({
      data: mockMedia,
      isLoading: false,
      isSuccess: true,
      error: null,
    } as ReturnType<typeof useRecentMedia>);
    renderPanel();
    const heading = await screen.findByRole('heading', { name: '最近新增' });
    expect(heading).toBeTruthy();
  });

  it('[P1] renders media items', async () => {
    mockUseRecentMedia.mockReturnValue({
      data: mockMedia,
      isLoading: false,
      isSuccess: true,
      error: null,
    } as ReturnType<typeof useRecentMedia>);
    renderPanel();
    expect(await screen.findByText('測試電影')).toBeTruthy();
    expect(screen.getByText('測試影集')).toBeTruthy();
  });

  it('[P1] shows "剛剛新增" badge for justAdded items', async () => {
    mockUseRecentMedia.mockReturnValue({
      data: mockMedia,
      isLoading: false,
      isSuccess: true,
      error: null,
    } as ReturnType<typeof useRecentMedia>);
    renderPanel();
    await screen.findByText('測試電影');
    expect(screen.getByText('剛剛新增')).toBeTruthy();
  });

  it('[P1] shows loading state', async () => {
    mockUseRecentMedia.mockReturnValue({
      data: undefined,
      isLoading: true,
      isSuccess: false,
      error: null,
    } as ReturnType<typeof useRecentMedia>);
    renderPanel();
    expect(await screen.findByTestId('recent-media-loading')).toBeTruthy();
  });

  it('[P1] shows empty state when no media', async () => {
    mockUseRecentMedia.mockReturnValue({
      data: [],
      isLoading: false,
      isSuccess: true,
      error: null,
    } as ReturnType<typeof useRecentMedia>);
    renderPanel();
    expect(await screen.findByText('媒體庫中還沒有內容')).toBeTruthy();
  });

  it('[P1] shows "查看全部媒體庫" link', async () => {
    mockUseRecentMedia.mockReturnValue({
      data: mockMedia,
      isLoading: false,
      isSuccess: true,
      error: null,
    } as ReturnType<typeof useRecentMedia>);
    renderPanel();
    expect(await screen.findByText(/查看全部媒體庫/)).toBeTruthy();
  });

  it('[P1] has recent-media-panel test id', async () => {
    mockUseRecentMedia.mockReturnValue({
      data: mockMedia,
      isLoading: false,
      isSuccess: true,
      error: null,
    } as ReturnType<typeof useRecentMedia>);
    renderPanel();
    expect(await screen.findByTestId('recent-media-panel')).toBeTruthy();
  });

  it('[P2] shows year for media items', async () => {
    mockUseRecentMedia.mockReturnValue({
      data: mockMedia,
      isLoading: false,
      isSuccess: true,
      error: null,
    } as ReturnType<typeof useRecentMedia>);
    renderPanel();
    await screen.findByText('測試電影');
    expect(screen.getByText('2024')).toBeTruthy();
    expect(screen.getByText('2023')).toBeTruthy();
  });

  it('[P2] shows emoji fallback when media has no posterUrl', async () => {
    const mediaWithoutPoster = [
      {
        id: 'no-poster-1',
        title: '無海報電影',
        year: 2024,
        mediaType: 'movie' as const,
        justAdded: false,
        addedAt: '2026-02-10T10:00:00Z',
      },
    ];
    mockUseRecentMedia.mockReturnValue({
      data: mediaWithoutPoster,
      isLoading: false,
      isSuccess: true,
      error: null,
    } as ReturnType<typeof useRecentMedia>);
    renderPanel();
    await screen.findByText('無海報電影');
    expect(screen.getByText('🎬')).toBeTruthy();
  });

  it('[P2] does not show "剛剛新增" badge when justAdded is false', async () => {
    const mediaNotJustAdded = [
      {
        id: 'old-media-1',
        title: '舊電影',
        year: 2020,
        posterUrl: 'https://example.com/poster.jpg',
        mediaType: 'movie' as const,
        justAdded: false,
        addedAt: '2026-02-09T10:00:00Z',
      },
    ];
    mockUseRecentMedia.mockReturnValue({
      data: mediaNotJustAdded,
      isLoading: false,
      isSuccess: true,
      error: null,
    } as ReturnType<typeof useRecentMedia>);
    renderPanel();
    await screen.findByText('舊電影');
    expect(screen.queryByText('剛剛新增')).toBeNull();
  });

  it('[P2] renders poster image with lazy loading', async () => {
    mockUseRecentMedia.mockReturnValue({
      data: mockMedia,
      isLoading: false,
      isSuccess: true,
      error: null,
    } as ReturnType<typeof useRecentMedia>);
    renderPanel();
    const img = await screen.findByAltText('測試電影');
    expect(img).toBeTruthy();
    expect(img.getAttribute('loading')).toBe('lazy');
    expect((img as HTMLImageElement).src).toContain('poster1.jpg');
  });

  it('[P1] Story 10-5 AC #5 — hideWhenEmpty hides the panel entirely when media is empty', async () => {
    mockUseRecentMedia.mockReturnValue({
      data: [],
      isLoading: false,
      isSuccess: true,
      error: null,
    } as ReturnType<typeof useRecentMedia>);
    renderPanel({ hideWhenEmpty: true });
    expect(screen.queryByTestId('recent-media-panel')).toBeNull();
    expect(screen.queryByText('媒體庫中還沒有內容')).toBeNull();
  });

  it('[P1] Story 10-5 AC #5 — hideWhenEmpty still renders the panel during loading (no flash)', async () => {
    mockUseRecentMedia.mockReturnValue({
      data: undefined,
      isLoading: true,
      isSuccess: false,
      error: null,
    } as ReturnType<typeof useRecentMedia>);
    renderPanel({ hideWhenEmpty: true });
    expect(await screen.findByTestId('recent-media-loading')).toBeInTheDocument();
  });

  it('[P2] does not show year when year is absent', async () => {
    const mediaNoYear = [
      {
        id: 'no-year-1',
        title: '無年份電影',
        posterUrl: 'https://example.com/poster.jpg',
        mediaType: 'movie' as const,
        justAdded: false,
        addedAt: '2026-02-10T10:00:00Z',
      },
    ];
    mockUseRecentMedia.mockReturnValue({
      data: mediaNoYear,
      isLoading: false,
      isSuccess: true,
      error: null,
    } as ReturnType<typeof useRecentMedia>);
    renderPanel();
    await screen.findByText('無年份電影');
    // Should only have the title text, not any year text
    const panel = screen.getByTestId('recent-media-panel');
    const yearElements = panel.querySelectorAll('p');
    // No year paragraph should be rendered for this item
    const yearTexts = Array.from(yearElements)
      .map((el) => el.textContent)
      .filter(Boolean);
    expect(yearTexts.every((t) => !t?.match(/^\d{4}$/))).toBe(true);
  });
});
