/**
 * FallbackStatusDisplay Tests (Story 3.7 - UX-4)
 */

import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { FallbackStatusDisplay } from './FallbackStatusDisplay';
import type { FallbackStatus } from './ManualSearchDialog';

describe('FallbackStatusDisplay', () => {
  it('renders nothing when no attempts', () => {
    const status: FallbackStatus = { attempts: [] };
    const { container } = render(<FallbackStatusDisplay status={status} />);

    expect(container.firstChild).toBeNull();
  });

  it('renders source names correctly', () => {
    const status: FallbackStatus = {
      attempts: [
        { source: 'tmdb', success: false },
        { source: 'douban', success: false },
        { source: 'wikipedia', success: false },
      ],
    };

    render(<FallbackStatusDisplay status={status} />);

    expect(screen.getByText('TMDb')).toBeInTheDocument();
    expect(screen.getByText('豆瓣')).toBeInTheDocument();
    expect(screen.getByText('Wikipedia')).toBeInTheDocument();
  });

  it('shows success indicator for successful sources', () => {
    const status: FallbackStatus = {
      attempts: [{ source: 'tmdb', success: true }],
    };

    render(<FallbackStatusDisplay status={status} />);

    // The span element contains the text and has the background class
    const tmdbBadge = screen.getByText('TMDb').closest('span');
    expect(tmdbBadge).toHaveClass('bg-green-500/20');
  });

  it('shows failure indicator for failed sources', () => {
    const status: FallbackStatus = {
      attempts: [{ source: 'tmdb', success: false }],
    };

    render(<FallbackStatusDisplay status={status} />);

    const tmdbBadge = screen.getByText('TMDb').closest('span');
    expect(tmdbBadge).toHaveClass('bg-red-500/20');
  });

  it('shows skipped indicator for skipped sources', () => {
    const status: FallbackStatus = {
      attempts: [{ source: 'douban', success: false, skipped: true, skipReason: 'circuit open' }],
    };

    render(<FallbackStatusDisplay status={status} />);

    const doubanBadge = screen.getByText('豆瓣').closest('span');
    expect(doubanBadge).toHaveClass('bg-slate-500/20');
  });

  it('shows guidance message when all failed (UX-4)', () => {
    const status: FallbackStatus = {
      attempts: [
        { source: 'tmdb', success: false },
        { source: 'douban', success: false },
      ],
    };

    render(<FallbackStatusDisplay status={status} />);

    expect(screen.getByText(/所有自動來源都無法找到匹配/)).toBeInTheDocument();
  });

  it('does not show guidance when at least one succeeded', () => {
    const status: FallbackStatus = {
      attempts: [
        { source: 'tmdb', success: false },
        { source: 'douban', success: true },
      ],
    };

    render(<FallbackStatusDisplay status={status} />);

    expect(screen.queryByText(/所有自動來源都無法找到匹配/)).not.toBeInTheDocument();
  });

  it('shows cancelled message when search was cancelled', () => {
    const status: FallbackStatus = {
      attempts: [{ source: 'tmdb', success: false }],
      cancelled: true,
    };

    render(<FallbackStatusDisplay status={status} />);

    expect(screen.getByText(/搜尋被取消/)).toBeInTheDocument();
  });
});
