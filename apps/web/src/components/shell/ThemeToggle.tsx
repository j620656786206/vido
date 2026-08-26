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

/**
 * ② 解釋改變, and the one interaction in this app that gets authored rather
 * than merely tokenised.
 *
 * Swapping the element outright made the destination icon TELEPORT: the button
 * changed identity between two frames, which is the one thing a toggle must
 * not do — you cannot tell a swap from a re-render. So both faces are always
 * mounted and they TRADE PLACES: sun and moon turn past each other on the same
 * axis, which is what 夜行⇄日巡 literally means. The rotation is the message,
 * so it comes from --motion-turn and lands at 0deg under reduced motion — the
 * opacity cross-fade survives, and with durations at 1ms it reads as a clean
 * instant swap rather than a suppressed spin.
 *
 * Both faces sit in one grid cell instead of absolute-positioning, so the
 * button reserves its own size with no magic numbers and nothing to keep in
 * sync when the icon scale changes between variants.
 */
function ThemeFaces({ next, size }: { next: 'light' | 'dark'; size: string }) {
  // ⚠️ `rotate`, NOT `transform`. Tailwind v4 compiles rotate-* to the
  // INDIVIDUAL `rotate` property (`.rotate-0 { rotate: none }`), which
  // `transition-property: transform` does not cover — v4's own
  // `transition-transform` expands to `transform, translate, scale, rotate`
  // precisely because of this. Writing the list by hand and naming `transform`
  // got it wrong in both directions at once: `transform` is never set on these
  // faces (dead weight) and `rotate` — the one thing that actually changes —
  // was excluded, so the turn snapped in a single frame and only the opacity
  // cross-faded. The whole point of --motion-turn never rendered.
  const face = 'col-start-1 row-start-1 transition-[opacity,rotate] ease-[var(--ease-settle)]';
  // Entering settles in over --motion-state; leaving clears out faster, the
  // same asymmetry the hero cross-fade uses.
  const shown = 'opacity-100 rotate-0 duration-[var(--motion-state)]';
  const hidden = 'opacity-0 duration-[var(--motion-leave)]';
  return (
    <span className={cn('grid shrink-0', size)} aria-hidden="true">
      <Sun
        className={cn(
          face,
          size,
          next === 'light' ? shown : cn(hidden, 'rotate-[var(--motion-turn)]')
        )}
      />
      <Moon
        className={cn(
          face,
          size,
          next === 'dark' ? shown : cn(hidden, 'rotate-[calc(var(--motion-turn)*-1)]')
        )}
      />
    </span>
  );
}

export function ThemeToggle({ variant = 'rail', className }: ThemeToggleProps) {
  const [theme, setTheme] = useTheme();
  const next = theme === 'dark' ? 'light' : 'dark';
  const nextName = next === 'light' ? '日巡' : '夜行';
  const label = `切換到${nextName}`;

  if (variant === 'row') {
    return (
      <button
        type="button"
        onClick={() => setTheme(next)}
        data-testid="theme-toggle"
        data-theme-next={next}
        aria-label={label}
        className={cn(
          'flex min-h-[44px] w-full items-center gap-2 rounded-[var(--radius-md)] px-2 text-[11px] text-[var(--text-muted)] transition-colors duration-[var(--motion-touch)] hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-secondary)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)]',
          className
        )}
      >
        <ThemeFaces next={next} size="h-3.5 w-3.5" />
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
          'flex h-11 w-11 items-center justify-center rounded-[var(--radius-md)] text-[var(--text-secondary)] transition-colors duration-[var(--motion-touch)] hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)]',
          className
        )}
      >
        <ThemeFaces next={next} size="h-5 w-5" />
      </button>
    </Tooltip>
  );
}
