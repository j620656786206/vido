import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { InFlightBadge } from './InFlightBadge';

/**
 * This component is the app's only holder of motion licence ③ — the one thing
 * allowed to move without a gesture. These guards protect the two properties
 * that make that defensible: the figure stays readable while the halo moves,
 * and the movement is gated behind `motion-safe`.
 */
describe('InFlightBadge', () => {
  it('[P1] the WASH breathes and the DIGIT does not — a count you wait to read is not a readout', () => {
    render(<InFlightBadge count={7} variant="row" testId="b" />);
    const badge = screen.getByTestId('b');
    const halo = badge.querySelector('span[aria-hidden]')!;

    expect(halo.getAttribute('class')).toContain('motion-safe:animate-breathe');
    expect(halo.getAttribute('class')).toContain('bg-[var(--accent-subtle)]');

    const digit = [...badge.querySelectorAll('span')].find((s) => s.textContent === '7')!;
    expect(digit.getAttribute('class')).not.toContain('animate-breathe');
  });

  it('[P1] the breath is behind motion-safe, so reduced motion leaves a still count', () => {
    render(<InFlightBadge count={1} variant="rail" testId="b" />);
    // Bare `animate-breathe` (no variant) would run regardless of preference;
    // the global CSS net would clamp it to one iteration, but relying on the
    // net means the call site can no longer be reasoned about on its own.
    const badge = screen.getByTestId('b');
    const classes = [...badge.querySelectorAll('*')].flatMap((el) => [...el.classList]);
    expect(classes).toContain('motion-safe:animate-breathe');
    expect(classes).not.toContain('animate-breathe');
  });

  it('[P2] stays out of the accessible tree — the count is spoken by the nav item’s aria-label', () => {
    render(<InFlightBadge count={3} variant="row" testId="b" />);
    expect(screen.getByTestId('b')).toHaveAttribute('aria-hidden', 'true');
    expect(screen.getByTestId('b')).toHaveTextContent('3');
  });

  it('[P2] both variants keep the accent TEXT twin, never a solid accent fill', () => {
    for (const variant of ['rail', 'row'] as const) {
      const { unmount } = render(<InFlightBadge count={2} variant={variant} testId="b" />);
      const badge = screen.getByTestId('b');
      expect(badge.className).toContain('text-[var(--accent-text)]');
      expect(badge.outerHTML).not.toContain('bg-[var(--accent-primary)]');
      unmount();
    }
  });
});
