import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { DualRatingDisplay } from './DualRatingDisplay';

describe('DualRatingDisplay', () => {
  it('renders TMDb rating with vote count', () => {
    render(<DualRatingDisplay tmdbRating={8.7} tmdbVoteCount={15000} />);
    const badge = screen.getByTestId('rating-TMDb');
    expect(badge).toHaveTextContent('TMDb');
    expect(badge).toHaveTextContent('8.7');
    expect(badge).toHaveTextContent('15k');
  });

  it('renders both TMDb and Douban ratings side-by-side', () => {
    render(
      <DualRatingDisplay
        tmdbRating={8.7}
        tmdbVoteCount={15000}
        doubanRating={9.6}
        doubanVoteCount={1500000}
      />
    );
    expect(screen.getByTestId('rating-TMDb')).toHaveTextContent('8.7');
    const douban = screen.getByTestId('rating-豆瓣');
    expect(douban).toHaveTextContent('豆瓣');
    expect(douban).toHaveTextContent('9.6');
    expect(douban).toHaveTextContent('1.5M');
  });

  it('shows a loading skeleton in the Douban slot while enrichment is in flight', () => {
    render(<DualRatingDisplay tmdbRating={8.7} tmdbVoteCount={15000} doubanLoading />);
    const skeleton = screen.getByTestId('rating-skeleton-豆瓣');
    expect(skeleton).toBeInTheDocument();
    expect(skeleton).toHaveAttribute('role', 'status');
    expect(skeleton).toHaveAttribute('aria-live', 'polite');
    // No actual Douban rating badge yet
    expect(screen.queryByTestId('rating-豆瓣')).not.toBeInTheDocument();
  });

  it('hides the Douban slot entirely when no Douban data and not loading', () => {
    render(<DualRatingDisplay tmdbRating={8.7} tmdbVoteCount={15000} />);
    expect(screen.queryByTestId('rating-豆瓣')).not.toBeInTheDocument();
    expect(screen.queryByTestId('rating-skeleton-豆瓣')).not.toBeInTheDocument();
  });

  it('prefers the resolved Douban rating over the skeleton when both rating and loading are set', () => {
    render(<DualRatingDisplay tmdbRating={8.7} doubanRating={9.6} doubanLoading />);
    expect(screen.getByTestId('rating-豆瓣')).toBeInTheDocument();
    expect(screen.queryByTestId('rating-skeleton-豆瓣')).not.toBeInTheDocument();
  });

  it('renders nothing when there is no rating and not loading', () => {
    const { container } = render(<DualRatingDisplay />);
    expect(container).toBeEmptyDOMElement();
  });

  it('omits the vote count when it is zero or missing', () => {
    render(<DualRatingDisplay tmdbRating={7.5} />);
    const badge = screen.getByTestId('rating-TMDb');
    expect(badge).toHaveTextContent('7.5');
    expect(badge).not.toHaveTextContent('(');
  });

  it('stacks on mobile and rows on desktop', () => {
    render(<DualRatingDisplay tmdbRating={8.0} />);
    const root = screen.getByTestId('dual-rating-display');
    expect(root.className).toContain('flex-col');
    expect(root.className).toContain('sm:flex-row');
  });
});
