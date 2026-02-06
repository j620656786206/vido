/**
 * CountdownTimer Component Tests (Story 3.11 - Task 8.2)
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, act } from '@testing-library/react';
import { CountdownTimer } from './CountdownTimer';

describe('CountdownTimer', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('renders countdown timer', () => {
    const futureTime = new Date(Date.now() + 30000).toISOString(); // 30 seconds
    render(<CountdownTimer targetTime={futureTime} />);

    expect(screen.getByTestId('countdown-timer')).toBeInTheDocument();
    expect(screen.getByTestId('countdown-value')).toHaveTextContent('30s');
  });

  it('displays minutes and seconds for longer durations', () => {
    const futureTime = new Date(Date.now() + 90000).toISOString(); // 90 seconds
    render(<CountdownTimer targetTime={futureTime} />);

    expect(screen.getByTestId('countdown-value')).toHaveTextContent('1m 30s');
  });

  it('displays only minutes when no remaining seconds', () => {
    const futureTime = new Date(Date.now() + 120000).toISOString(); // 120 seconds
    render(<CountdownTimer targetTime={futureTime} />);

    expect(screen.getByTestId('countdown-value')).toHaveTextContent('2m');
  });

  it('updates countdown every second', () => {
    const futureTime = new Date(Date.now() + 5000).toISOString(); // 5 seconds
    render(<CountdownTimer targetTime={futureTime} />);

    expect(screen.getByTestId('countdown-value')).toHaveTextContent('5s');

    act(() => {
      vi.advanceTimersByTime(1000);
    });

    expect(screen.getByTestId('countdown-value')).toHaveTextContent('4s');

    act(() => {
      vi.advanceTimersByTime(2000);
    });

    expect(screen.getByTestId('countdown-value')).toHaveTextContent('2s');
  });

  it('displays "即將重試" when countdown reaches zero', () => {
    const futureTime = new Date(Date.now() + 1000).toISOString(); // 1 second
    render(<CountdownTimer targetTime={futureTime} />);

    expect(screen.getByTestId('countdown-value')).toHaveTextContent('1s');

    act(() => {
      vi.advanceTimersByTime(2000);
    });

    expect(screen.getByTestId('countdown-value')).toHaveTextContent('即將重試');
  });

  it('calls onComplete when countdown reaches zero', () => {
    const onComplete = vi.fn();
    const futureTime = new Date(Date.now() + 1000).toISOString();
    render(<CountdownTimer targetTime={futureTime} onComplete={onComplete} />);

    expect(onComplete).not.toHaveBeenCalled();

    act(() => {
      vi.advanceTimersByTime(2000);
    });

    expect(onComplete).toHaveBeenCalledTimes(1);
  });

  it('hides icon when showIcon is false', () => {
    const futureTime = new Date(Date.now() + 30000).toISOString();
    render(<CountdownTimer targetTime={futureTime} showIcon={false} />);

    const timer = screen.getByTestId('countdown-timer');
    expect(timer.querySelector('svg')).not.toBeInTheDocument();
  });

  it('applies custom className', () => {
    const futureTime = new Date(Date.now() + 30000).toISOString();
    render(<CountdownTimer targetTime={futureTime} className="custom-class" />);

    expect(screen.getByTestId('countdown-timer')).toHaveClass('custom-class');
  });

  it('handles past timestamps gracefully', () => {
    const pastTime = new Date(Date.now() - 5000).toISOString(); // 5 seconds ago
    render(<CountdownTimer targetTime={pastTime} />);

    expect(screen.getByTestId('countdown-value')).toHaveTextContent('即將重試');
  });
});
