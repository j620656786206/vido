// Implements: Component/Checkbox/Checked (4EHFN) + Component/Checkbox/Empty (Wd9AL) + Component/Checkbox/Indeterminate (NfHDL) + Component/Checkbox/DisabledChecked (Fn5MZ) + Component/Checkbox/DisabledEmpty (VSXl5)
// Source: ux-design.pen (Pencil app)
import { Check } from 'lucide-react';
import { cn } from '../../lib/utils';

export type CheckboxBoxState =
  | 'checked'
  | 'empty'
  | 'mixed'
  | 'disabled-checked'
  | 'disabled-empty';

/**
 * The drawn 20px checkbox, in the five states the .pen components define. It
 * is ONLY the picture (aria-hidden): pair it with a native
 * `<input type="checkbox" className="peer sr-only">` placed right before it,
 * which carries the real semantics (checked / indeterminate / disabled, Space
 * to toggle) — the RadioDot precedent. The focus ring follows that input.
 */
export function CheckboxBox({ state, className }: { state: CheckboxBoxState; className?: string }) {
  const disabled = state === 'disabled-checked' || state === 'disabled-empty';
  return (
    <span
      aria-hidden="true"
      data-state={state}
      className={cn(
        'flex size-5 shrink-0 items-center justify-center rounded-[var(--radius-sm)] transition-colors',
        'peer-focus-visible:ring-2 peer-focus-visible:ring-[var(--focus-ring)] peer-focus-visible:ring-offset-2 peer-focus-visible:ring-offset-transparent',
        (state === 'checked' || state === 'mixed') &&
          'bg-[var(--accent-primary)] text-[var(--text-on-accent)]',
        state === 'empty' && 'border-[1.5px] border-[var(--text-muted)]',
        disabled &&
          'border border-[var(--border-subtle)] bg-[var(--bg-tertiary)] text-[var(--text-muted)]',
        className
      )}
    >
      {state === 'checked' && <Check className="size-3.5" strokeWidth={3} />}
      {state === 'disabled-checked' && <Check className="size-[13px]" strokeWidth={3} />}
      {state === 'mixed' && (
        <span className="h-0.5 w-2.5 rounded-[var(--radius-sm)] bg-[var(--text-on-accent)]" />
      )}
    </span>
  );
}
