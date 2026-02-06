/**
 * CountdownTimer Component (Story 3.11 - Task 8.2)
 * Real-time countdown display for retry items
 */

import { useState, useEffect, useRef } from 'react';
import { Clock } from 'lucide-react';
import { cn } from '../../lib/utils';

export interface CountdownTimerProps {
  /** Target time for countdown */
  targetTime: string;
  /** Called when countdown reaches zero */
  onComplete?: () => void;
  /** Additional CSS classes */
  className?: string;
  /** Show icon */
  showIcon?: boolean;
}

/**
 * Formats remaining seconds into a human-readable string
 */
function formatTimeRemaining(seconds: number): string {
  if (seconds <= 0) {
    return '即將重試';
  }

  if (seconds < 60) {
    return `${seconds}s`;
  }

  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = seconds % 60;

  if (remainingSeconds > 0) {
    return `${minutes}m ${remainingSeconds}s`;
  }

  return `${minutes}m`;
}

/**
 * Real-time countdown timer component
 * Updates every second to show time remaining until retry
 */
export function CountdownTimer({
  targetTime,
  onComplete,
  className,
  showIcon = true,
}: CountdownTimerProps) {
  const [secondsRemaining, setSecondsRemaining] = useState<number>(() => {
    const target = new Date(targetTime).getTime();
    const now = Date.now();
    return Math.max(0, Math.floor((target - now) / 1000));
  });
  const completedRef = useRef(false);

  useEffect(() => {
    const target = new Date(targetTime).getTime();
    completedRef.current = false;

    const updateTimer = () => {
      const now = Date.now();
      const remaining = Math.max(0, Math.floor((target - now) / 1000));
      setSecondsRemaining(remaining);

      if (remaining === 0 && onComplete && !completedRef.current) {
        completedRef.current = true;
        onComplete();
      }
    };

    // Initial update
    updateTimer();

    // Update every second
    const interval = setInterval(updateTimer, 1000);

    return () => clearInterval(interval);
  }, [targetTime, onComplete]);

  const isImmediate = secondsRemaining <= 0;

  return (
    <div
      className={cn(
        'inline-flex items-center gap-1.5 text-sm font-mono',
        isImmediate ? 'text-blue-400' : 'text-slate-400',
        className
      )}
      data-testid="countdown-timer"
    >
      {showIcon && <Clock className="h-3.5 w-3.5" />}
      <span data-testid="countdown-value">{formatTimeRemaining(secondsRemaining)}</span>
    </div>
  );
}

export default CountdownTimer;
