import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { SeasonAccordion } from './SeasonAccordion';
import type { SeasonSummary, SeasonEpisodesResponse } from '../../types/library';

// useSeasonEpisodes is the lazy episode query; stub it so the accordion can be
// tested without a QueryClientProvider. The mock records the `enabled` arg so we
// can assert lazy-load gating (AC #3).
const useSeasonEpisodesMock = vi.fn();
vi.mock('../../hooks/useMediaDetails', () => ({
  useSeasonEpisodes: (seriesId: string, seasonNumber: number, enabled: boolean) =>
    useSeasonEpisodesMock(seriesId, seasonNumber, enabled),
}));

const seasons: SeasonSummary[] = [
  {
    id: 1,
    seasonNumber: 1,
    name: '第 1 季',
    episodeCount: 12,
    airDate: '2024-01-05',
    posterPath: '/p1.jpg',
  },
  { id: 2, seasonNumber: 2, name: '第 2 季', episodeCount: 10, airDate: '2025-01-05' },
];

function stubQuery(
  overrides: Partial<{ data: SeasonEpisodesResponse; isLoading: boolean; isError: boolean }> = {}
) {
  return {
    data: overrides.data,
    isLoading: overrides.isLoading ?? false,
    isError: overrides.isError ?? false,
    refetch: vi.fn(),
  };
}

describe('SeasonAccordion', () => {
  beforeEach(() => {
    useSeasonEpisodesMock.mockReset();
    useSeasonEpisodesMock.mockReturnValue(stubQuery());
  });

  it('renders nothing when tmdbId <= 0 (AC #1)', () => {
    const { container } = render(<SeasonAccordion seasons={seasons} seriesId="s1" tmdbId={0} />);
    expect(container.firstChild).toBeNull();
  });

  it('renders nothing when there are no seasons', () => {
    const { container } = render(<SeasonAccordion seasons={[]} seriesId="s1" tmdbId={123} />);
    expect(container.firstChild).toBeNull();
  });

  it('renders one collapsed header per season with name, episode count and air date (AC #2)', () => {
    render(<SeasonAccordion seasons={seasons} seriesId="s1" tmdbId={123} />);

    expect(screen.getByTestId('season-header-1')).toBeInTheDocument();
    expect(screen.getByTestId('season-header-2')).toBeInTheDocument();
    expect(screen.getByText('第 1 季')).toBeInTheDocument();
    expect(screen.getByText(/12 集/)).toBeInTheDocument();
    expect(screen.getByText(/2024-01-05/)).toBeInTheDocument();
  });

  it('keeps the episode query disabled until a season is expanded (AC #3 lazy-load)', () => {
    render(<SeasonAccordion seasons={seasons} seriesId="s1" tmdbId={123} />);

    // Both seasons collapsed → both queries enabled=false.
    expect(useSeasonEpisodesMock).toHaveBeenCalledWith('s1', 1, false);
    expect(useSeasonEpisodesMock).toHaveBeenCalledWith('s1', 2, false);

    fireEvent.click(screen.getByTestId('season-header-1'));

    // Season 1 now expanded → its query becomes enabled.
    expect(useSeasonEpisodesMock).toHaveBeenCalledWith('s1', 1, true);
  });

  it('renders the episode list when an expanded season resolves', () => {
    useSeasonEpisodesMock.mockReturnValue(
      stubQuery({
        data: {
          season: seasons[0],
          episodes: [
            { episodeNumber: 1, name: '第一集', hasLocalFile: true, subtitleStatus: 'found' },
          ],
        },
      })
    );

    render(<SeasonAccordion seasons={seasons} seriesId="s1" tmdbId={123} />);
    fireEvent.click(screen.getByTestId('season-header-1'));

    expect(screen.getByText('第一集')).toBeInTheDocument();
    expect(screen.getByText('S01E01')).toBeInTheDocument();
  });

  it('shows a skeleton while the season list is loading (M2)', () => {
    render(<SeasonAccordion seasons={[]} seriesId="s1" tmdbId={123} isLoading />);
    expect(screen.getByTestId('season-accordion-skeleton')).toBeInTheDocument();
    expect(screen.queryByTestId('season-header-1')).not.toBeInTheDocument();
  });

  it('shows a retry-able error when the season list fails (M2)', () => {
    const onRetry = vi.fn();
    render(<SeasonAccordion seasons={[]} seriesId="s1" tmdbId={123} isError onRetry={onRetry} />);

    expect(screen.getByTestId('season-accordion-error')).toBeInTheDocument();
    expect(screen.getByRole('alert')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: '重試' }));
    expect(onRetry).toHaveBeenCalledTimes(1);
  });

  it('still renders nothing when tmdbId <= 0 even while loading (AC #1 gate wins)', () => {
    const { container } = render(
      <SeasonAccordion seasons={[]} seriesId="s1" tmdbId={0} isLoading />
    );
    expect(container.firstChild).toBeNull();
  });

  it('toggles aria-expanded on the season header', () => {
    render(<SeasonAccordion seasons={seasons} seriesId="s1" tmdbId={123} />);
    const header = screen.getByTestId('season-header-1');

    expect(header).toHaveAttribute('aria-expanded', 'false');
    fireEvent.click(header);
    expect(header).toHaveAttribute('aria-expanded', 'true');
  });
});
