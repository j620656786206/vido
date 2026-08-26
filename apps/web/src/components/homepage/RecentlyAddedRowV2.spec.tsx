import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import React from 'react';
import type { LibraryItem } from '../../types/library';

vi.mock('../../hooks/useLibrary', () => ({
  useRecentlyAdded: vi.fn(),
  // ux3-1-8 made the window a shared constant; a wholesale module mock must
  // re-export it or every consumer of this mock dies at import time.
  RECENT_LIMIT: 20,
  RECENT_STALE_TIME_MS: 30_000,
}));
// Stub PosterCardV2 — it needs the router + image pipeline; here we only care that a
// card renders per item. Its own behaviour (badge, links) is covered by its spec.
// The chip is a real router <Link> now (its door to /activity); this spec has
// no router, so stub Link as a plain anchor.
vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = (await importOriginal()) as Record<string, unknown>;
  return {
    ...actual,
    Link: ({ to, children, ...rest }: { to: string; children: React.ReactNode }) => (
      <a href={to} {...rest}>
        {children}
      </a>
    ),
  };
});
vi.mock('../library/PosterCardV2', () => ({
  PosterCardV2: ({ id, title }: { id: string; title: string }) =>
    React.createElement('div', { 'data-testid': `card-${id}` }, title),
}));

import { useRecentlyAdded } from '../../hooks/useLibrary';
import { RecentlyAddedRowV2 } from './RecentlyAddedRowV2';

const mockUseRecentlyAdded = vi.mocked(useRecentlyAdded);

function movie(id: string, over: Record<string, unknown> = {}): LibraryItem {
  return {
    type: 'movie',
    movie: {
      id,
      title: `電影 ${id}`,
      posterPath: '/p.jpg',
      releaseDate: '2024-03-01',
      runtime: 128,
      voteAverage: 8.2,
      parseStatus: 'success',
      createdAt: '2026-06-14T00:00:00Z',
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

describe('RecentlyAddedRowV2 (own-content 最近新增 row — four states)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('[P1] Loading — renders the poster-shaped skeleton row (H4-D-v2)', () => {
    mockUseRecentlyAdded.mockReturnValue(result({ isLoading: true }));
    render(<RecentlyAddedRowV2 />);
    expect(screen.getByTestId('home-recent-skeleton')).toBeInTheDocument();
    expect(screen.queryByTestId('home-recent-row')).toBeNull();
  });

  it('[P1] Empty — graceful 「尚無最近新增」 hint, no error (H5-D-v2)', () => {
    mockUseRecentlyAdded.mockReturnValue(result({ data: [] }));
    render(<RecentlyAddedRowV2 />);
    const empty = screen.getByTestId('home-recent-empty');
    expect(empty).toHaveTextContent('尚無最近新增');
    expect(screen.queryByTestId('home-recent-error')).toBeNull();
  });

  it('[P1] Error — fail-soft inline banner + 重試 retries (H6-D-v2)', () => {
    const refetch = vi.fn();
    mockUseRecentlyAdded.mockReturnValue(result({ isError: true, refetch }));
    render(<RecentlyAddedRowV2 />);
    expect(screen.getByTestId('home-recent-error')).toHaveTextContent('無法載入，請稍後再試');
    fireEvent.click(screen.getByTestId('home-recent-retry'));
    expect(refetch).toHaveBeenCalledTimes(1);
  });

  it('[P1] Data — renders a PosterCardV2 per item', () => {
    mockUseRecentlyAdded.mockReturnValue(result({ data: [movie('a'), movie('b')] }));
    render(<RecentlyAddedRowV2 />);
    expect(screen.getByTestId('home-recent-row')).toBeInTheDocument();
    expect(screen.getByTestId('card-a')).toBeInTheDocument();
    expect(screen.getByTestId('card-b')).toBeInTheDocument();
  });

  it('[P2] 整理中 · N chip counts pending items, hidden when none', () => {
    mockUseRecentlyAdded.mockReturnValue(
      result({
        data: [
          movie('a', { parseStatus: 'pending' }),
          movie('b'),
          movie('c', { parseStatus: 'pending' }),
        ],
      })
    );
    const { rerender } = render(<RecentlyAddedRowV2 />);
    const chip = screen.getByTestId('home-recent-progress');
    expect(chip).toHaveTextContent('整理中');
    expect(chip).toHaveTextContent('2');

    // No pending items → chip is suppressed (exception-signal only).
    mockUseRecentlyAdded.mockReturnValue(result({ data: [movie('a'), movie('b')] }));
    rerender(<RecentlyAddedRowV2 />);
    expect(screen.queryByTestId('home-recent-progress')).toBeNull();
  });

  // ⚖️ R2 ruling: pending = QUEUED → amber, matching the poster badge exactly
  // (one screen, one truth, one colour). Supersedes R1's brief green.
  it('[P2] chip is a DOOR to /activity and wears the badge-matching amber', () => {
    mockUseRecentlyAdded.mockReturnValue(
      result({ data: [movie('a', { parseStatus: 'pending' }), movie('b')] })
    );
    render(<RecentlyAddedRowV2 />);
    const chip = screen.getByTestId('home-recent-progress');
    expect(chip).toHaveAttribute('href', '/activity');
    expect(chip).toHaveTextContent('整理中');
    expect(chip.className).toContain('bg-[var(--warning-tint)]');
    expect(chip.className).toContain('text-[var(--warning-text)]');
    expect(chip.className).not.toContain('success');
    expect(chip.className).not.toContain('accent');
  });
});
