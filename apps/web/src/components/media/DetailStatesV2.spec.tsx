import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { DetailSkeletonV2, DetailNotFoundV2 } from './DetailStatesV2';

describe('DetailStatesV2', () => {
  it('skeleton is busy and renders no spinner', () => {
    render(<DetailSkeletonV2 />);
    expect(screen.getByTestId('detail-skeleton')).toHaveAttribute('aria-busy', 'true');
  });

  it('not-found shows the message and calls onBack from both affordances', () => {
    const onBack = vi.fn();
    render(<DetailNotFoundV2 onBack={onBack} />);
    expect(screen.getByTestId('detail-not-found')).toHaveTextContent('找不到這部影片');
    fireEvent.click(screen.getByTestId('detail-back'));
    fireEvent.click(screen.getByText('返回媒體庫'));
    expect(onBack).toHaveBeenCalledTimes(2);
  });
});
