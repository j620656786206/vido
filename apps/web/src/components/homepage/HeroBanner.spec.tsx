import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, act } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import {
  createMemoryHistory,
  RouterProvider,
  createRootRoute,
  createRoute,
  createRouter,
} from '@tanstack/react-router';
import React from 'react';
import { HeroBanner } from './HeroBanner';
import type { HeroBannerItem } from '../../types/tmdb';

vi.mock('../../hooks/useTrending', () => ({
  useTrendingHero: vi.fn(),
}));

vi.mock('../../services/tmdb', () => ({
  default: {
    getMovieVideos: vi.fn(),
    getTVShowVideos: vi.fn(),
  },
}));

import { useTrendingHero } from '../../hooks/useTrending';
import tmdbService from '../../services/tmdb';

const mockUseTrendingHero = vi.mocked(useTrendingHero);
const mockGetMovieVideos = vi.mocked(tmdbService.getMovieVideos);

function item(overrides: Partial<HeroBannerItem> = {}): HeroBannerItem {
  return {
    id: 550,
    mediaType: 'movie',
    title: '鬥陣俱樂部',
    overview: '一段關於失眠者與肥皂商人的旅程。',
    backdropPath: '/backdrop.jpg',
    releaseDate: '1999-10-15',
    voteAverage: 8.4,
    ...overrides,
  };
}

function renderBanner() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  const rootRoute = createRootRoute({
    component: () => React.createElement(HeroBanner),
  });
  const mediaRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/media/$type/$id',
    component: () => React.createElement('div', null, 'Media Detail'),
  });

  const routeTree = rootRoute.addChildren([mediaRoute]);
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

function mockHook(overrides: Partial<ReturnType<typeof useTrendingHero>> = {}) {
  mockUseTrendingHero.mockReturnValue({
    data: [],
    isLoading: false,
    isError: false,
    isSuccess: true,
    error: null,
    ...overrides,
  } as ReturnType<typeof useTrendingHero>);
}

describe('HeroBanner', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.useRealTimers();
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.useRealTimers();
  });

  it('[P1] hides section when API returns empty (AC #5)', async () => {
    mockHook({ data: [], isSuccess: true });
    renderBanner();
    // Wait one tick for the router to mount, then assert nothing rendered.
    await Promise.resolve();
    expect(screen.queryByTestId('hero-banner')).toBeNull();
    expect(screen.queryByTestId('hero-banner-skeleton')).toBeNull();
  });

  it('[P1] hides section when API errors (AC #5)', async () => {
    mockHook({ data: undefined, isError: true, isSuccess: false, error: new Error('boom') });
    renderBanner();
    await Promise.resolve();
    expect(screen.queryByTestId('hero-banner')).toBeNull();
  });

  it('[P1] renders skeleton while loading', async () => {
    mockHook({ data: undefined, isLoading: true, isSuccess: false });
    renderBanner();
    expect(await screen.findByTestId('hero-banner-skeleton')).toBeInTheDocument();
  });

  it('[P1] renders backdrop, title, year, rating, overview (AC #1)', async () => {
    mockHook({ data: [item()] });
    renderBanner();

    expect(await screen.findByTestId('hero-banner-title')).toHaveTextContent('鬥陣俱樂部');
    expect(screen.getByTestId('hero-banner-year')).toHaveTextContent('1999');
    expect(screen.getByTestId('hero-banner-rating')).toHaveTextContent('8.4');
    expect(screen.getByTestId('hero-banner-overview')).toHaveTextContent('一段關於失眠者');
    const backdrop = screen.getByTestId('hero-banner-backdrop') as HTMLImageElement;
    expect(backdrop.src).toContain('/original/backdrop.jpg');
  });

  it('[P1] detail link points at /media/$type/$id with TMDb id (AC #3)', async () => {
    mockHook({ data: [item({ id: 1396, mediaType: 'tv', title: 'Breaking Bad' })] });
    renderBanner();

    const link = (await screen.findByTestId('hero-banner-detail-link')) as HTMLAnchorElement;
    expect(link.href).toMatch(/\/media\/tv\/1396$/);
  });

  it('[P1] auto-rotates every 8s (AC #2)', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    mockHook({
      data: [item({ id: 1, title: 'A' }), item({ id: 2, title: 'B' }), item({ id: 3, title: 'C' })],
    });
    renderBanner();
    await screen.findByTestId('hero-banner');

    const slidesActive = () =>
      screen.getAllByTestId('hero-banner-slide').map((el) => el.getAttribute('data-active'));

    expect(slidesActive()).toEqual(['true', 'false', 'false']);

    act(() => {
      vi.advanceTimersByTime(8000);
    });
    expect(slidesActive()).toEqual(['false', 'true', 'false']);

    act(() => {
      vi.advanceTimersByTime(8000);
    });
    expect(slidesActive()).toEqual(['false', 'false', 'true']);

    // Wraps around
    act(() => {
      vi.advanceTimersByTime(8000);
    });
    expect(slidesActive()).toEqual(['true', 'false', 'false']);
  });

  it('[P1] pauses rotation on hover (Task 1.5)', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    mockHook({
      data: [item({ id: 1 }), item({ id: 2 })],
    });
    renderBanner();
    const banner = await screen.findByTestId('hero-banner');

    fireEvent.mouseEnter(banner);

    act(() => {
      vi.advanceTimersByTime(20000);
    });
    expect(screen.getAllByTestId('hero-banner-slide')[0].getAttribute('data-active')).toBe('true');

    fireEvent.mouseLeave(banner);
    act(() => {
      vi.advanceTimersByTime(8000);
    });
    expect(screen.getAllByTestId('hero-banner-slide')[1].getAttribute('data-active')).toBe('true');
  });

  it('[P1] does not auto-rotate when only one item', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    mockHook({ data: [item()] });
    renderBanner();
    await screen.findByTestId('hero-banner');

    expect(screen.queryByTestId('hero-banner-dots')).toBeNull();
    act(() => {
      vi.advanceTimersByTime(30000);
    });
    expect(screen.getAllByTestId('hero-banner-slide')[0].getAttribute('data-active')).toBe('true');
  });

  it('[P1] dot click jumps to that slide (Task 1.6)', async () => {
    mockHook({
      data: [item({ id: 1 }), item({ id: 2 }), item({ id: 3 })],
    });
    renderBanner();
    await screen.findByTestId('hero-banner');

    fireEvent.click(screen.getByTestId('hero-banner-dot-2'));
    const slides = screen.getAllByTestId('hero-banner-slide');
    expect(slides[2].getAttribute('data-active')).toBe('true');
    expect(slides[0].getAttribute('data-active')).toBe('false');
  });

  it('[P1] play button opens trailer modal (AC #6)', async () => {
    mockGetMovieVideos.mockResolvedValue({ id: 1, results: [] });
    mockHook({ data: [item({ id: 1 })] });
    renderBanner();

    fireEvent.click(await screen.findByTestId('hero-banner-play-trailer'));
    expect(await screen.findByTestId('trailer-modal')).toBeInTheDocument();
  });

  it('[P2] does not render rating badge when voteAverage is 0', async () => {
    mockHook({ data: [item({ voteAverage: 0 })] });
    renderBanner();
    await screen.findByTestId('hero-banner');
    expect(screen.queryByTestId('hero-banner-rating')).toBeNull();
  });

  it('[P2] does not render year when releaseDate is missing', async () => {
    mockHook({ data: [item({ releaseDate: '' })] });
    renderBanner();
    await screen.findByTestId('hero-banner');
    expect(screen.queryByTestId('hero-banner-year')).toBeNull();
  });

  it('[P2] does not render backdrop image when backdropPath is null', async () => {
    mockHook({ data: [item({ backdropPath: null })] });
    renderBanner();
    await screen.findByTestId('hero-banner');
    expect(screen.queryByTestId('hero-banner-backdrop')).toBeNull();
  });

  it('[P2] uses TV Chinese label "影集" for TV mediaType', async () => {
    mockHook({ data: [item({ mediaType: 'tv' })] });
    renderBanner();
    await screen.findByTestId('hero-banner');
    expect(screen.getByText('影集')).toBeInTheDocument();
  });
});
