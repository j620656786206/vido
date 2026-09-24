// Implements: <utility — no .pen counterpart>
// Drawn in C9-D (f7AZv / o6V6JS) and C13-D (W36o90): a 20px circle, gold with a
// check when selected.
import { Check } from 'lucide-react';
import { cn } from '../../lib/utils';

/**
 * The visible half of a custom radio. It draws nothing on its own: place it
 * IMMEDIATELY AFTER a native `<input type="radio" className="peer sr-only">`
 * (same parent) and it follows that input through `peer-*` — checked, keyboard
 * focus. The native input keeps doing the real work (arrow keys, the `name`
 * group, form semantics); never replace it with `div role="radio"`.
 *
 * One component for every settings radio (dsr-3f export formats, dsr-3d
 * localization levels), so the two cannot drift apart again (they were 18 vs 20).
 */
export function RadioDot({ className, ...props }: React.ComponentProps<'span'>) {
  return (
    <span
      aria-hidden="true"
      {...props}
      className={cn(
        'flex size-5 shrink-0 items-center justify-center rounded-full border-2 border-[var(--border-subtle)] text-[var(--text-on-accent)] transition-colors',
        'peer-checked:border-[var(--accent-primary)] peer-checked:bg-[var(--accent-primary)]',
        'peer-focus-visible:ring-2 peer-focus-visible:ring-[var(--focus-ring)] peer-focus-visible:ring-offset-2 peer-focus-visible:ring-offset-[var(--bg-secondary)]',
        '[&>svg]:hidden peer-checked:[&>svg]:block',
        className
      )}
    >
      <Check className="size-3" strokeWidth={3} />
    </span>
  );
}
