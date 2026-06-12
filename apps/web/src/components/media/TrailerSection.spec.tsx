import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { TrailerSection } from './TrailerSection';
import type { Video } from '../../types/tmdb';

vi.mock('../../hooks/useMediaDetails', () => ({
  useMediaVideos: vi.fn(),
}));

import { useMediaVideos } from '../../hooks/useMediaDetails';

const mockUseMediaVideos = vi.mocked(useMediaVideos);

function video(overrides: Partial<Video> = {}): Video {
  return {
    key: 'SUXWAEX2jlg',
    name: 'Trailer',
    site: 'YouTube',
    type: 'Trailer',
    official: true,
    publishedAt: '2024-01-01T00:00:00.000Z',
    ...overrides,
  };
}

// Only `data` + `isError` are consumed by the component; cast the partial mock.
function mockVideos(data: { id: number; results: Video[] } | undefined, isError = false) {
  mockUseMediaVideos.mockReturnValue({
    data,
    isError,
    isLoading: false,
  } as unknown as ReturnType<typeof useMediaVideos>);
}

describe('TrailerSection', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('[P1] renders the TrailerEmbed button when an embeddable YouTube trailer exists (AC #1)', () => {
    mockVideos({ id: 550, results: [video({ key: 'SUXWAEX2jlg' })] });

    render(<TrailerSection tmdbId={550} type="movie" title="Fight Club" />);

    expect(screen.getByText('預告片')).toBeInTheDocument();
    // Pre-click state: TrailerEmbed shows its "▶ 觀看預告片" button, not the iframe.
    const button = screen.getByTestId('trailer-button');
    expect(button).toBeInTheDocument();
    expect(button).toHaveTextContent('觀看預告片');
    expect(screen.queryByTestId('trailer-player')).toBeNull();
    // Fallback link must NOT render when an embed is available.
    expect(screen.queryByTestId('trailer-section-fallback-link')).toBeNull();
  });

  it('[P1] activating the embed autoplays the iframe (AC #5)', () => {
    mockVideos({ id: 550, results: [video({ key: 'SUXWAEX2jlg' })] });

    render(<TrailerSection tmdbId={550} type="movie" title="Fight Club" />);

    fireEvent.click(screen.getByTestId('trailer-button'));

    const iframe = screen.getByTitle('Fight Club 預告片');
    expect(iframe.getAttribute('src')).toBe(
      'https://www.youtube-nocookie.com/embed/SUXWAEX2jlg?autoplay=1'
    );
  });

  it('[P1] falls back to a TMDB videos-page link when videos exist but none is a YouTube Trailer (AC #3)', () => {
    mockVideos({
      id: 550,
      results: [video({ key: 'vimeo1', site: 'Vimeo' }), video({ key: 'teaser1', type: 'Teaser' })],
    });

    render(<TrailerSection tmdbId={550} type="movie" title="Fight Club" />);

    expect(screen.getByText('預告片')).toBeInTheDocument();
    const link = screen.getByTestId('trailer-section-fallback-link');
    expect(link).toBeInTheDocument();
    expect(link.getAttribute('href')).toBe('https://www.themoviedb.org/movie/550/videos');
    expect(link.getAttribute('target')).toBe('_blank');
    expect(link.getAttribute('rel')).toBe('noopener noreferrer');
    // No embed button in the fallback path.
    expect(screen.queryByTestId('trailer-button')).toBeNull();
  });

  it('[P1] builds the tv fallback URL for series (AC #3, #7)', () => {
    mockVideos({ id: 1396, results: [video({ key: 'vimeo1', site: 'Vimeo' })] });

    render(<TrailerSection tmdbId={1396} type="tv" title="Breaking Bad" />);

    expect(screen.getByTestId('trailer-section-fallback-link').getAttribute('href')).toBe(
      'https://www.themoviedb.org/tv/1396/videos'
    );
  });

  it('[P1] renders nothing when TMDB returns no videos at all (AC #4)', () => {
    mockVideos({ id: 550, results: [] });

    render(<TrailerSection tmdbId={550} type="movie" title="Fight Club" />);

    expect(screen.queryByTestId('trailer-section')).toBeNull();
    expect(screen.queryByText('預告片')).toBeNull();
  });

  it('[P2] renders nothing on error — fail-soft, never breaks the page (AC #4)', () => {
    mockVideos(undefined, true);

    const { container } = render(<TrailerSection tmdbId={550} type="movie" title="Fight Club" />);

    expect(screen.queryByTestId('trailer-section')).toBeNull();
    expect(container).toBeEmptyDOMElement();
  });

  it('[P2] renders nothing while loading — silent, no skeleton flash (Task 3.3)', () => {
    mockVideos(undefined, false);

    const { container } = render(<TrailerSection tmdbId={550} type="movie" title="Fight Club" />);

    expect(container).toBeEmptyDOMElement();
  });
});
