// Implements: <utility — no .pen counterpart>
/**
 * Base UI bottom-sheet wrapper (UX Redesign Phase 2 — UX2-1 / ADR D1-b, D1-d).
 *
 * Base UI ships no literal "Sheet"; a bottom sheet is its `Dialog` styled to
 * slide up from the bottom edge. Using Dialog gives focus-trap, Escape, scroll
 * lock and a scrim by construction — the exact a11y the ADR's Base UI decision
 * exists to outsource (P4 hand-rolled-dialog failures). The single wrap point
 * for Base UI Dialog; importing `@base-ui/react` outside `components/ui/` is
 * ESLint-banned (F2). Token classes only (N6).
 */
import * as React from 'react';
import { Dialog } from '@base-ui/react/dialog';
import { cn } from '../../lib/utils';

const POPUP_BASE =
  'fixed inset-x-0 bottom-0 z-[71] max-h-[85vh] overflow-y-auto rounded-t-[var(--radius-xl)] border-t border-[var(--border-subtle)] bg-[var(--bg-secondary)] p-4 pb-[max(1rem,env(safe-area-inset-bottom))] shadow-[var(--shadow-xl)] transition-transform duration-[var(--motion-move)] data-[ending-style]:translate-y-full data-[starting-style]:translate-y-full';

const TITLE_BASE = 'mb-3 text-base font-semibold text-[var(--text-primary)]';

const DESCRIPTION_BASE = 'text-sm text-[var(--text-secondary)]';

interface SheetProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Visible heading; when omitted, provide `ariaLabel` for an SR-only title. */
  title?: React.ReactNode;
  ariaLabel?: string;
  children: React.ReactNode;
  /** Defaults to `bottom-sheet` — the id every existing caller and test knows. */
  testId?: string;
  /**
   * Merged into the popup with `cn()`. Heads-up: `p-0` also drops the base
   * `pb-[max(1rem,env(safe-area-inset-bottom))]` (same tailwind-merge group), so
   * a caller that zeroes the padding must add its own safe-area bottom back.
   */
  className?: string;
  /** Merged into the visible title (not the sr-only fallback). */
  titleClassName?: string;
  /**
   * A line under the title, rendered as `Dialog.Description` so the sheet's
   * `aria-describedby` points at it — a plain `<p>` in `children` is never read
   * out when the sheet opens. (`Dialog.Description` cannot be imported outside ui/.)
   */
  description?: React.ReactNode;
  /** Merged into the description. */
  descriptionClassName?: string;
  /** Where focus goes on close — a ref, a function, or a boolean (Base UI default `true`). */
  finalFocus?: React.ComponentProps<typeof Dialog.Popup>['finalFocus'];
  /** Fires once the open/close transition has finished. */
  onOpenChangeComplete?: (open: boolean) => void;
}

export function Sheet({
  open,
  onOpenChange,
  title,
  ariaLabel,
  children,
  testId = 'bottom-sheet',
  className,
  titleClassName,
  description,
  descriptionClassName,
  finalFocus,
  onOpenChangeComplete,
}: SheetProps) {
  return (
    <Dialog.Root
      open={open}
      onOpenChange={onOpenChange}
      onOpenChangeComplete={onOpenChangeComplete}
    >
      <Dialog.Portal>
        <Dialog.Backdrop className="fixed inset-0 z-[70] bg-[var(--overlay-scrim)] transition-opacity duration-[var(--motion-move)] data-[ending-style]:opacity-0 data-[starting-style]:opacity-0" />
        <Dialog.Popup
          className={cn(POPUP_BASE, className)}
          data-testid={testId}
          finalFocus={finalFocus}
        >
          {/* Drag handle affordance */}
          <div
            aria-hidden="true"
            className="mx-auto mb-3 h-1 w-10 rounded-full bg-[var(--border-subtle)]"
          />
          {title ? (
            <Dialog.Title className={cn(TITLE_BASE, titleClassName)}>{title}</Dialog.Title>
          ) : (
            <Dialog.Title className="sr-only">{ariaLabel ?? '選單'}</Dialog.Title>
          )}
          {description && (
            <Dialog.Description className={cn(DESCRIPTION_BASE, descriptionClassName)}>
              {description}
            </Dialog.Description>
          )}
          {children}
        </Dialog.Popup>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
