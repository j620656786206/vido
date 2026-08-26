// Implements: <screen-section — pending epic-19-8 mapping>
/**
 * 夜行 / 日巡 switch, in the shell rather than in Settings.
 *
 * ⚖️ Alexyu 2026-08-26:「放到設定裡面有點太深了」— a theme switch is the one
 * preference people expect to reach without navigating, so it lives on a
 * surface that is always on screen. The two shells split it cleanly and it is
 * never rendered twice at one breakpoint: DESKTOP puts it in the sidebar
 * footer beside the ambient status strip, MOBILE puts it in the sticky header
 * (there is no sidebar there, only the bottom tab bar).
 *
 * Settings ▸ 外觀 stays: it is where the state gets EXPLAINED (whether the app
 * is still following the OS, and why dark is the default). Both read the same
 * useTheme hook, so they cannot disagree.
 *
 * One button, not a radio pair: with exactly two themes the useful affordance
 * is "flip", and the label says where the flip GOES (「切換到日巡」) rather than
 * where you are — a control named after its current state makes the user guess
 * whether it is a readout or a button.
 */
import { Moon, Sun } from 'lucide-react';
import { useTheme } from '../../hooks/useTheme';
import { Tooltip } from '../ui/Tooltip';
import { cn } from '../../lib/utils';

interface ThemeToggleProps {
  /** 'rail' = icon only (collapsed sidebar, mobile header); 'row' = icon + label. */
  variant?: 'rail' | 'row';
  className?: string;
}

export function ThemeToggle({ variant = 'rail', className }: ThemeToggleProps) {
  const [theme, setTheme] = useTheme();
  const next = theme === 'dark' ? 'light' : 'dark';
  const nextName = next === 'light' ? '日巡' : '夜行';
  const label = `切換到${nextName}`;
  // The icon shows the DESTINATION, matching the label — a sun on the button
  // that turns the lights on.
  const Icon = next === 'light' ? Sun : Moon;

  if (variant === 'row') {
    return (
      <button
        type="button"
        onClick={() => setTheme(next)}
        data-testid="theme-toggle"
        data-theme-next={next}
        aria-label={label}
        className={cn(
          'flex min-h-[44px] w-full items-center gap-2 rounded-[var(--radius-md)] px-2 text-[11px] text-[var(--text-muted)] transition-colors hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-secondary)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)]',
          className
        )}
      >
        <Icon className="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
        <span className="truncate">{label}</span>
      </button>
    );
  }

  return (
    <Tooltip content={label}>
      <button
        type="button"
        onClick={() => setTheme(next)}
        data-testid="theme-toggle"
        data-theme-next={next}
        aria-label={label}
        className={cn(
          'flex h-11 w-11 items-center justify-center rounded-[var(--radius-md)] text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)]',
          className
        )}
      >
        <Icon className="h-5 w-5" aria-hidden="true" />
      </button>
    </Tooltip>
  );
}
