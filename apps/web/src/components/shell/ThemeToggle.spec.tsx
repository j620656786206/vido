import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ThemeToggle } from './ThemeToggle';

vi.mock('../ui/Tooltip', () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => children,
}));

/**
 * ⚖️ Alexyu 2026-08-26:「放到設定裡面有點太深了」— these guard the two things
 * that make the control usable at a glance: it names its DESTINATION, and it
 * actually flips.
 */
describe('ThemeToggle', () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.removeAttribute('data-theme');
    vi.stubGlobal(
      'matchMedia',
      vi.fn(() => ({
        matches: false,
        media: '',
        onchange: null,
        addEventListener: () => {},
        removeEventListener: () => {},
        addListener: () => {},
        removeListener: () => {},
        dispatchEvent: () => false,
      })) as unknown as typeof window.matchMedia
    );
  });

  it('[P1] names where the flip GOES, not where you are', () => {
    // A control named after its current state makes the user guess whether it
    // is a readout or a button.
    render(<ThemeToggle />);
    expect(screen.getByTestId('theme-toggle')).toHaveAttribute('aria-label', '切換到日巡');
  });

  it('[P1] flips the theme, and then offers the way back', () => {
    render(<ThemeToggle />);
    const btn = screen.getByTestId('theme-toggle');

    fireEvent.click(btn);
    expect(document.documentElement.getAttribute('data-theme')).toBe('light');
    expect(localStorage.getItem('vido:theme')).toBe('light');
    expect(btn).toHaveAttribute('aria-label', '切換到夜行');

    fireEvent.click(btn);
    expect(document.documentElement.hasAttribute('data-theme')).toBe(false);
    expect(localStorage.getItem('vido:theme')).toBe('dark');
  });

  it('[P2] the row variant shows the label; the rail variant is icon-only', () => {
    const { unmount } = render(<ThemeToggle variant="row" />);
    expect(screen.getByTestId('theme-toggle')).toHaveTextContent('切換到日巡');
    unmount();

    render(<ThemeToggle variant="rail" />);
    // Icon-only: the accessible name still carries the meaning.
    expect(screen.getByTestId('theme-toggle')).toHaveTextContent('');
    expect(screen.getByTestId('theme-toggle')).toHaveAttribute('aria-label', '切換到日巡');
  });

  it('[P2] both variants keep the 44px touch floor', () => {
    const { unmount } = render(<ThemeToggle variant="row" />);
    expect(screen.getByTestId('theme-toggle').className).toContain('min-h-[44px]');
    unmount();
    render(<ThemeToggle variant="rail" />);
    expect(screen.getByTestId('theme-toggle').className).toContain('h-11');
  });
});
