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

interface SheetProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Visible heading; when omitted, provide `ariaLabel` for an SR-only title. */
  title?: React.ReactNode;
  ariaLabel?: string;
  children: React.ReactNode;
}

export function Sheet({ open, onOpenChange, title, ariaLabel, children }: SheetProps) {
  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Backdrop className="fixed inset-0 z-[70] bg-[var(--overlay-scrim)] transition-opacity duration-200 data-[ending-style]:opacity-0 data-[starting-style]:opacity-0" />
        <Dialog.Popup
          className="fixed inset-x-0 bottom-0 z-[71] max-h-[85vh] overflow-y-auto rounded-t-[var(--radius-xl)] border-t border-[var(--border-subtle)] bg-[var(--bg-secondary)] p-4 pb-[max(1rem,env(safe-area-inset-bottom))] shadow-[var(--shadow-xl)] transition-transform duration-200 data-[ending-style]:translate-y-full data-[starting-style]:translate-y-full"
          data-testid="bottom-sheet"
        >
          {/* Drag handle affordance */}
          <div
            aria-hidden="true"
            className="mx-auto mb-3 h-1 w-10 rounded-full bg-[var(--border-subtle)]"
          />
          {title ? (
            <Dialog.Title className="mb-3 text-base font-semibold text-[var(--text-primary)]">
              {title}
            </Dialog.Title>
          ) : (
            <Dialog.Title className="sr-only">{ariaLabel ?? '選單'}</Dialog.Title>
          )}
          {children}
        </Dialog.Popup>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
