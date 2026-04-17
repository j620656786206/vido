import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { ExploreBlockSkeleton } from './ExploreBlockSkeleton';

describe('ExploreBlockSkeleton', () => {
  it('[P1] renders 6 placeholders by default (desktop row)', () => {
    render(<ExploreBlockSkeleton />);
    expect(screen.getAllByTestId('explore-block-skeleton')).toHaveLength(6);
  });

  it('[P1] honors custom count prop', () => {
    render(<ExploreBlockSkeleton count={3} />);
    expect(screen.getAllByTestId('explore-block-skeleton')).toHaveLength(3);
  });

  it('[P1] wrapper is hidden from assistive tech (aria-hidden)', () => {
    render(<ExploreBlockSkeleton />);
    expect(screen.getByTestId('explore-block-skeleton-row')).toHaveAttribute('aria-hidden', 'true');
  });
});
