import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, fireEvent, act } from '@testing-library/react';
import React from 'react';
import type { LibraryItem } from '../../types/library';

vi.mock('../../hooks/useLibrary', () => ({
  useRecentlyAdded: vi.fn(),
  // ux3-1-8 made the window a shared constant; a wholesale module mock must
  // re-export it or every consumer of this mock dies at import time.
  RECENT_LIMIT: 20,
  RECENT_STALE_TIME_MS: 30_000,
}));
// The hero's title/CTA are router <Link>s; this spec has no router, so stub
// Link as a plain anchor that serialises its params for href assertions.
vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = (await importOriginal()) as Record<string, unknown>;
  return {
    ...actual,
    Link: ({
      to,
      params,
      children,
      ...rest
    }: {
      to: string;
      params?: Record<string, string>;
      children: React.ReactNode;
    }) => (
      <a href={params ? to.replace('$type', params.type).replace('$id', params.id) : to} {...rest}>
        {children}
      </a>
    ),
  };
});

import { useRecentlyAdded } from '../../hooks/useLibrary';
import { HeroBanner, toHeroItems } from './HeroBanner';

const mockUseRecentlyAdded = vi.mocked(useRecentlyAdded);

function movie(id: string, over: Record<string, unknown> = {}): LibraryItem {
  return {
    type: 'movie',
    movie: {
      id,
      title: `電影 ${id}`,
      posterPath: '/p.jpg',
      backdropPath: `/backdrop-${id}.jpg`,
      releaseDate: '2024-05-10',
      voteAverage: 8.2,
      parseStatus: 'success',
      subtitleStatus: 'found',
      subtitleLanguage: 'zh-Hant',
      createdAt: '2026-08-01T00:00:00Z',
      ...over,
    },
  } as unknown as LibraryItem;
}

function series(id: string, over: Record<string, unknown> = {}): LibraryItem {
  return {
    type: 'tv',
    series: {
      id,
      title: `影集 ${id}`,
      posterPath: '/p.jpg',
      backdropPath: `/backdrop-${id}.jpg`,
      firstAirDate: '2023-01-15',
      voteAverage: 9.0,
      parseStatus: 'success',
      subtitleStatus: 'not_found',
      createdAt: '2026-08-02T00:00:00Z',
      ...over,
    },
  } as unknown as LibraryItem;
}

function result(over: Record<string, unknown> = {}) {
  return {
    data: undefined,
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
    ...over,
  } as unknown as ReturnType<typeof useRecentlyAdded>;
}

const activeSlide = () =>
  document.querySelector<HTMLElement>('[data-testid="hero-banner-slide"][data-active="true"]');

describe('HeroBanner (Home v3 own-library static hero — ux3-1-8)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('[P1] renders the newest own item with 最新入庫 eyebrow, title link to the LIBRARY detail route, and gold 查看詳情 CTA', () => {
    mockUseRecentlyAdded.mockReturnValue(result({ data: [movie('m1'), movie('m2')] }));
    render(<HeroBanner />);

    expect(screen.getByTestId('hero-banner')).toBeInTheDocument();
    const slide = activeSlide()!;
    expect(slide.querySelector('[data-testid="hero-banner-eyebrow"]')).toHaveTextContent(
      '最新入庫'
    );
    expect(slide.querySelector('[data-testid="hero-banner-title"]')).toHaveTextContent('電影 m1');
    // LIBRARY id in the route — not a TMDb id.
    expect(slide.querySelector('[data-testid="hero-banner-title-link"]')).toHaveAttribute(
      'href',
      '/media/movie/m1'
    );
    expect(slide.querySelector('[data-testid="hero-banner-detail-link"]')).toHaveAttribute(
      'href',
      '/media/movie/m1'
    );
    expect(slide.querySelector('[data-testid="hero-banner-year"]')).toHaveTextContent('2024');
  });

  it('[P1] the subtitle badge celebrates the zh-Hant steady state (繁中字幕 ✓ 已就緒, success tint)', () => {
    mockUseRecentlyAdded.mockReturnValue(result({ data: [movie('m1')] }));
    render(<HeroBanner />);
    const badge = screen.getByTestId('hero-banner-subtitle-badge');
    expect(badge).toHaveTextContent('繁中字幕 ✓ 已就緒');
    expect(badge.className).toContain('success');
  });

  it('[P1] a missing-subtitle item wears the grid vocabulary (缺字幕) AND is a door (brief §3「缺字幕→門」)', () => {
    mockUseRecentlyAdded.mockReturnValue(result({ data: [series('s1')] }));
    render(<HeroBanner />);
    const badge = screen.getByTestId('hero-banner-subtitle-badge');
    expect(badge).toHaveTextContent('缺字幕');
    expect(badge.className).not.toContain('success');
    // A named problem with no way to act on it is the dead end this redesign
    // keeps deleting.
    expect(badge.tagName).toBe('A');
    expect(badge).toHaveAttribute('href', '/media/tv/s1');
  });

  it('[P0] 固定詞彙 — a LIFECYCLE exception outranks the subtitle verdict, so the hero cannot show green while the poster below shows amber', () => {
    // Same item, both states true: zh-Hant subtitle on record AND a re-parse in
    // flight. The grid's pickPosterBadge shows 整理中; the hero must agree.
    mockUseRecentlyAdded.mockReturnValue(
      result({ data: [movie('m1', { parseStatus: 'pending' })] })
    );
    render(<HeroBanner />);
    const badge = screen.getByTestId('hero-banner-subtitle-badge');
    expect(badge).toHaveTextContent('整理中');
    expect(badge).not.toHaveTextContent('已就緒');
    expect(badge.className).toContain('warning');
    // The app's own in-flight work is the Activity hub's business, not a door
    // into the item — so this one stays a statement.
    expect(badge.tagName).toBe('SPAN');
  });

  it('[P1] a failed parse shows 失敗 on the hero too (same ladder as the grid)', () => {
    mockUseRecentlyAdded.mockReturnValue(
      result({ data: [movie('m1', { parseStatus: 'failed' })] })
    );
    render(<HeroBanner />);
    expect(screen.getByTestId('hero-banner-subtitle-badge')).toHaveTextContent('失敗');
  });

  it('[P0] STATIC by ruling — no pause button, no play/trailer CTA, and no timer-driven rotation', () => {
    vi.useFakeTimers();
    try {
      mockUseRecentlyAdded.mockReturnValue(result({ data: [movie('m1'), movie('m2')] }));
      render(<HeroBanner />);

      expect(screen.queryByTestId('hero-banner-pause')).toBeNull();
      expect(screen.queryByTestId('hero-banner-play-trailer')).toBeNull();

      expect(activeSlide()!.textContent).toContain('電影 m1');
      // 8s used to rotate the old carousel; nothing may move now.
      act(() => {
        vi.advanceTimersByTime(30000);
      });
      expect(activeSlide()!.textContent).toContain('電影 m1');
    } finally {
      vi.useRealTimers();
    }
  });

  it('[P1] manual switching — dots and prev/next chevrons change the active slide', () => {
    mockUseRecentlyAdded.mockReturnValue(
      result({ data: [movie('m1'), movie('m2'), series('s1')] })
    );
    render(<HeroBanner />);

    fireEvent.click(screen.getByTestId('hero-banner-dot-1'));
    expect(activeSlide()!.textContent).toContain('電影 m2');

    fireEvent.click(screen.getByTestId('hero-banner-next'));
    expect(activeSlide()!.textContent).toContain('影集 s1');

    // prev wraps around from index 0.
    fireEvent.click(screen.getByTestId('hero-banner-dot-0'));
    fireEvent.click(screen.getByTestId('hero-banner-prev'));
    expect(activeSlide()!.textContent).toContain('影集 s1');
  });

  it('[P1] items without a backdrop are excluded; ONE dressed item → single slide, dots hidden', () => {
    mockUseRecentlyAdded.mockReturnValue(
      result({ data: [movie('m1'), movie('m2', { backdropPath: null })] })
    );
    render(<HeroBanner />);
    expect(screen.getAllByTestId('hero-banner-slide')).toHaveLength(1);
    expect(screen.queryByTestId('hero-banner-dots')).toBeNull();
  });

  it('[P1] caps the hero at 5 dressed items (newest first)', () => {
    const many = ['a', 'b', 'c', 'd', 'e', 'f', 'g'].map((id) => movie(id));
    expect(toHeroItems(many as LibraryItem[])).toHaveLength(5);
  });

  it('[P0] 例外訊號原則 — no backdrops at all → the hero renders NOTHING (absent, not an empty frame)', () => {
    mockUseRecentlyAdded.mockReturnValue(result({ data: [movie('m1', { backdropPath: null })] }));
    const { container } = render(<HeroBanner />);
    expect(container.firstChild).toBeNull();
  });

  it('[P1] error state stays quiet here — the 最近新增 row is the shared query’s one spokesman', () => {
    mockUseRecentlyAdded.mockReturnValue(result({ isError: true }));
    const { container } = render(<HeroBanner />);
    expect(container.firstChild).toBeNull();
  });

  it('[P0] the status badge sits on an OPAQUE underlay — a tint over the scrim is unmeasured', () => {
    // Reported from a phone in 日巡: the green read as invisible. The *-tint
    // tokens are ~12–20% alpha and styles-contrast.spec.ts measures them
    // composited over a --bg-* PAGE ground; the hero puts the badge on the
    // scrim instead, where light's ink-green --success-text measured 1.47:1.
    // The opaque chip restores the stack the gate actually guarantees.
    mockUseRecentlyAdded.mockReturnValue(result({ data: [movie('m1')] }));
    render(<HeroBanner />);
    const badge = screen.getByTestId('hero-banner-subtitle-badge');
    const underlay = badge.parentElement!;
    expect(underlay.className).toContain('bg-[var(--bg-secondary)]');
  });

  it('[P1] the content block reserves room for the controls — they used to touch on mobile', () => {
    // Measured on a phone: the CTA's bottom edge and the dots pill's top edge
    // were both y=437, zero clearance. The pill is absolutely positioned over
    // this same lower edge, so the padding has to know whether it exists.
    mockUseRecentlyAdded.mockReturnValue(result({ data: [movie('m1'), movie('m2')] }));
    const { container, unmount } = render(<HeroBanner />);
    const withControls = container.querySelector('.absolute.inset-x-0.bottom-0')!;
    expect(withControls.className).toContain('pb-16');
    unmount();

    // A single dressed item renders no pill, so it keeps the room.
    mockUseRecentlyAdded.mockReturnValue(result({ data: [movie('m1')] }));
    const { container: solo } = render(<HeroBanner />);
    expect(solo.querySelector('.absolute.inset-x-0.bottom-0')!.className).toContain('pb-12');
  });

  it('[P2] loading renders the hero-shaped skeleton', () => {
    mockUseRecentlyAdded.mockReturnValue(result({ isLoading: true }));
    render(<HeroBanner />);
    expect(screen.getByTestId('hero-banner-skeleton')).toBeInTheDocument();
  });

  it('[P0] the selection is keyed by IDENTITY — a 30s refetch that prepends a new item must NOT swap the slide the user chose', () => {
    // The whole point of ux3-1-8: nothing moves unless the user moves it. With
    // an index-keyed selection, useRecentlyAdded's 30s poll landing a new
    // backdrop-bearing item would silently re-point the chosen slide — a
    // self-moving hero wearing a different mechanism.
    mockUseRecentlyAdded.mockReturnValue(result({ data: [movie('m1'), movie('m2')] }));
    const { rerender } = render(<HeroBanner />);

    fireEvent.click(screen.getByTestId('hero-banner-dot-1'));
    expect(activeSlide()!.textContent).toContain('電影 m2');

    // A scan finishes; the poll returns with a NEWER item at the head.
    mockUseRecentlyAdded.mockReturnValue(result({ data: [movie('m0'), movie('m1'), movie('m2')] }));
    rerender(<HeroBanner />);

    // Still the movie the user picked — not whatever now sits at index 1.
    expect(activeSlide()!.textContent).toContain('電影 m2');
  });

  it('[P1] a shrinking list falls back to the newest DURING render — the hero never blinks to a slide-less frame', () => {
    mockUseRecentlyAdded.mockReturnValue(result({ data: [movie('m1'), movie('m2'), movie('m3')] }));
    const { rerender } = render(<HeroBanner />);
    fireEvent.click(screen.getByTestId('hero-banner-dot-2'));
    expect(activeSlide()!.textContent).toContain('電影 m3');

    // The row refetches every 30s; an item can vanish from under the selection.
    mockUseRecentlyAdded.mockReturnValue(result({ data: [movie('m1')] }));
    rerender(<HeroBanner />);

    // Exactly one slide is active on the very first frame after the shrink.
    expect(
      document.querySelectorAll('[data-testid="hero-banner-slide"][data-active="true"]')
    ).toHaveLength(1);
    expect(activeSlide()!.textContent).toContain('電影 m1');
  });
});
