// Implements: <utility — no .pen counterpart>
/**
 * Base UI Tooltip wrapper (UX Redesign Phase 2 — UX2-1 / ADR D1-d).
 *
 * The single wrap point for `@base-ui/react`'s Tooltip — an ESLint
 * `no-restricted-imports` rule (F2) bans importing Base UI anywhere except this
 * `components/ui/` dir, so swapping the primitive later is a one-dir change.
 *
 * Required by the 64px collapsed icon-rail (ADR D1-a / §6.2): rail items hide
 * their label, so each exposes its label + count through this tooltip. Token
 * classes only — zero hardcoded color/size values (N6).
 */
import * as React from 'react';
import { Tooltip as BaseTooltip } from '@base-ui/react/tooltip';

/** Wrap a region (e.g. the whole sidebar) to share hover-open delay timing. */
export const TooltipProvider = BaseTooltip.Provider;

interface TooltipProps {
  /** Tooltip content — a label, optionally with a count. */
  content: React.ReactNode;
  /** A single interactive trigger element (a Link/button). Base UI merges the
   * trigger behaviour onto it rather than nesting an extra control. */
  children: React.ReactElement;
  side?: 'top' | 'right' | 'bottom' | 'left';
}

export function Tooltip({ content, children, side = 'right' }: TooltipProps) {
  return (
    <BaseTooltip.Root>
      <BaseTooltip.Trigger render={children} />
      <BaseTooltip.Portal>
        <BaseTooltip.Positioner side={side} sideOffset={8}>
          <BaseTooltip.Popup className="z-[80] flex items-center gap-2 rounded-[var(--radius-md)] border border-[var(--border-subtle)] bg-[var(--bg-tertiary)] px-2.5 py-1.5 text-xs font-medium text-[var(--text-primary)] shadow-[var(--shadow-lg)] data-[starting-style]:opacity-0 data-[ending-style]:opacity-0">
            {content}
          </BaseTooltip.Popup>
        </BaseTooltip.Positioner>
      </BaseTooltip.Portal>
    </BaseTooltip.Root>
  );
}
