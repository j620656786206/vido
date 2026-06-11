import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { StreamingAvailability } from './StreamingAvailability';
import type { WatchProviderRegion } from '../../types/library';

const region: WatchProviderRegion = {
  link: 'https://www.themoviedb.org/movie/550/watch?locale=TW',
  flatrate: [
    { providerId: 8, providerName: 'Netflix', logoPath: '/netflix.jpg', displayPriority: 1 },
    { providerId: 337, providerName: 'Disney Plus', logoPath: '/disney.jpg', displayPriority: 0 },
  ],
  rent: [{ providerId: 2, providerName: 'Apple TV', logoPath: '/apple.jpg', displayPriority: 0 }],
  buy: [{ providerId: 3, providerName: 'Google Play', logoPath: null, displayPriority: 0 }],
};

describe('StreamingAvailability', () => {
  it('renders the section heading', () => {
    render(<StreamingAvailability region={region} />);
    expect(screen.getByText('可在哪裡觀看')).toBeInTheDocument();
  });

  it('renders flatrate / rent / buy group labels and provider logos', () => {
    render(<StreamingAvailability region={region} />);
    expect(screen.getByText('訂閱')).toBeInTheDocument();
    expect(screen.getByText('租借')).toBeInTheDocument();
    expect(screen.getByText('購買')).toBeInTheDocument();

    // Logo <img> uses providerName as alt (a11y).
    const netflix = screen.getByAltText('Netflix') as HTMLImageElement;
    expect(netflix).toBeInTheDocument();
    expect(netflix.src).toContain('w92/netflix.jpg');
    expect(screen.getByAltText('Disney Plus')).toBeInTheDocument();
    expect(screen.getByAltText('Apple TV')).toBeInTheDocument();
  });

  it('sorts each group by displayPriority (Disney before Netflix)', () => {
    render(<StreamingAvailability region={region} />);
    const imgs = screen.getAllByRole('img').map((el) => el.getAttribute('alt'));
    // flatrate group: Disney (priority 0) must precede Netflix (priority 1).
    expect(imgs.indexOf('Disney Plus')).toBeLessThan(imgs.indexOf('Netflix'));
  });

  it('falls back to a name chip when a provider logo is missing', () => {
    render(<StreamingAvailability region={region} />);
    // Google Play has logoPath null → rendered as a text chip, not an <img>.
    expect(screen.getByText('Google Play')).toBeInTheDocument();
    expect(screen.queryByAltText('Google Play')).not.toBeInTheDocument();
  });

  it('renders the outbound TMDB watch link with safe rel/target', () => {
    render(<StreamingAvailability region={region} />);
    const link = screen.getByTestId('streaming-availability-link');
    expect(link).toBeInTheDocument();
    expect(link).toHaveAttribute('href', region.link);
    expect(link).toHaveAttribute('target', '_blank');
    expect(link).toHaveAttribute('rel', 'noopener noreferrer');
  });

  it('renders the mandatory JustWatch attribution', () => {
    render(<StreamingAvailability region={region} />);
    expect(screen.getByText('資料來源：JustWatch')).toBeInTheDocument();
  });

  it('shows a skeleton while loading', () => {
    render(<StreamingAvailability isLoading />);
    expect(screen.getByTestId('streaming-availability-skeleton')).toBeInTheDocument();
    expect(screen.queryByText('資料來源：JustWatch')).not.toBeInTheDocument();
  });

  it('shows a quiet error state with a retry button (fail-soft, AC #5)', () => {
    const onRetry = vi.fn();
    render(<StreamingAvailability isError onRetry={onRetry} />);
    const alert = screen.getByTestId('streaming-availability-error');
    expect(alert).toHaveAttribute('role', 'alert');
    fireEvent.click(screen.getByText('重試'));
    expect(onRetry).toHaveBeenCalledTimes(1);
  });

  it('shows the muted empty-state when the region has no providers (AC #4)', () => {
    render(<StreamingAvailability region={{ link: '', flatrate: [], rent: [], buy: [] }} />);
    expect(screen.getByTestId('streaming-availability-empty')).toHaveTextContent(
      '此區域暫無串流資訊'
    );
    // No outbound link when there is no data.
    expect(screen.queryByTestId('streaming-availability-link')).not.toBeInTheDocument();
  });

  it('shows the empty-state when region is undefined (never throws)', () => {
    render(<StreamingAvailability region={undefined} />);
    expect(screen.getByTestId('streaming-availability-empty')).toBeInTheDocument();
  });
});
