import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
// Story 13-1b: PosterCard's hover 想要 scrim mounts RequestButton (needs a
// QueryClient); stub it so these tests stay presentation-focused.
vi.mock('../requests/RequestButton', () => ({
  RequestButton: () => null,
}));

import { RelatedContent } from './RelatedContent';
import type { RecommendationItem } from '../../types/library';

// Mock TanStack Router Link (PosterCard renders a <Link>) — mirrors MediaGrid.spec.
vi.mock('@tanstack/react-router', () => ({
  Link: ({
    children,
    to,
    params,
    ...props
  }: {
    children: React.ReactNode;
    to: string;
    params: Record<string, string>;
  }) => (
    <a href={`${to.replace('$type', params.type).replace('$id', params.id)}`} {...props}>
      {children}
    </a>
  ),
}));

// Stub PosterCard's lazy TMDb-detail hooks so tiles don't need a QueryClientProvider.
vi.mock('../../hooks/useMediaDetails', () => ({
  useMovieDetails: (): { data?: { runtime?: number } } => ({ data: undefined }),
  useTVShowDetails: (): { data?: { numberOfSeasons?: number; numberOfEpisodes?: number } } => ({
    data: undefined,
  }),
}));

const items: RecommendationItem[] = [
  {
    id: 603,
    mediaType: 'movie',
    title: 'The Matrix',
    posterPath: '/matrix.jpg',
    releaseDate: '1999-03-31',
    voteAverage: 8.2,
    isOwned: true,
  },
  {
    id: 604,
    mediaType: 'movie',
    title: 'The Matrix Reloaded',
    posterPath: null,
    releaseDate: '2003-05-15',
    voteAverage: 7.0,
    isOwned: false,
  },
];

describe('RelatedContent', () => {
  it('renders the 相關推薦 heading and one tile per item', () => {
    render(<RelatedContent items={items} />);
    expect(screen.getByRole('heading', { name: '相關推薦' })).toBeInTheDocument();
    expect(screen.getByText('The Matrix')).toBeInTheDocument();
    expect(screen.getByText('The Matrix Reloaded')).toBeInTheDocument();
    expect(screen.getAllByTestId('poster-card')).toHaveLength(2);
  });

  it('shows the 已有 owned badge on owned tiles only', () => {
    render(<RelatedContent items={items} />);
    // AvailabilityBadge variant="owned" renders the 已有 label.
    const ownedBadges = screen.getAllByText('已有');
    expect(ownedBadges).toHaveLength(1);
  });

  it('renders skeletons while loading', () => {
    render(<RelatedContent items={[]} isLoading />);
    expect(screen.getByTestId('related-content-skeleton')).toBeInTheDocument();
    expect(screen.queryByTestId('poster-card')).not.toBeInTheDocument();
  });

  it('renders a quiet retry alert on error and calls onRetry', () => {
    const onRetry = vi.fn();
    render(<RelatedContent items={[]} isError onRetry={onRetry} />);
    const alert = screen.getByRole('alert');
    expect(alert).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: '重試' }));
    expect(onRetry).toHaveBeenCalledTimes(1);
  });

  it('renders nothing when there are no items and not loading/error (AC #5)', () => {
    const { container } = render(<RelatedContent items={[]} />);
    expect(container).toBeEmptyDOMElement();
  });
});
