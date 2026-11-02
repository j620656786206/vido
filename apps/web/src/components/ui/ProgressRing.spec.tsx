import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { ProgressRing } from './ProgressRing';

describe('ProgressRing', () => {
  it('exposes its value as a progressbar and draws that share of the ring', () => {
    const { container } = render(<ProgressRing value={40} label="下載中 40%" />);
    const bar = screen.getByRole('progressbar', { name: '下載中 40%' });
    expect(bar).toHaveAttribute('aria-valuenow', '40');
    const arc = container.querySelectorAll('circle')[1];
    const c = 2 * Math.PI * 9;
    expect(Number(arc.getAttribute('stroke-dashoffset'))).toBeCloseTo(c * 0.6);
  });

  it('clamps out-of-range values', () => {
    render(<ProgressRing value={140} label="x" />);
    expect(screen.getByRole('progressbar')).toHaveAttribute('aria-valuenow', '100');
  });
});
