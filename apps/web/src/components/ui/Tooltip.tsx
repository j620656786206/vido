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
  /** Hover-open delay in ms (Base UI default 600). Icon-only STATUS marks pass
   *  0: the tooltip is their only label (J11-D rule 4 — no browser-title wait). */
  delay?: number;
  /** Also open on a press, so a phone (no hover) gets the tooltip on tap; it
   *  then closes on a press elsewhere or Esc (J11-D rule 4). Pressing the
   *  trigger again keeps it open — a toggle would close it the moment a mouse
   *  user clicks an icon the hover already opened. */
  openOnPress?: boolean;
}

export function Tooltip({ content, children, side = 'right', delay, openOnPress }: TooltipProps) {
  const [open, setOpen] = React.useState(false);
  // A TOUCH press opens it; Base UI's hover hook then sees the synthetic
  // mouseleave a tap ends with and closes it again at once. So a touch-opened
  // tooltip ignores hover-closes and stays until a press elsewhere, Esc or
  // blur. A mouse click is left to hover — it already opened the tooltip.
  const pointerType = React.useRef<string | undefined>(undefined);
  const openedByTouch = React.useRef(false);
  type ChildProps = {
    onClick?: React.MouseEventHandler;
    onPointerDown?: React.PointerEventHandler;
  };
  const childProps = children.props as ChildProps;
  const controlled = openOnPress
    ? {
        open,
        onOpenChange: (next: boolean, details: { reason: string }) => {
          if (!next && openedByTouch.current && details.reason === 'trigger-hover') return;
          if (!next) openedByTouch.current = false;
          setOpen(next);
        },
      }
    : {};
  const trigger = openOnPress
    ? React.cloneElement(children as React.ReactElement<ChildProps>, {
        onPointerDown: (event: React.PointerEvent) => {
          childProps.onPointerDown?.(event);
          pointerType.current = event.pointerType;
        },
        onClick: (event: React.MouseEvent) => {
          childProps.onClick?.(event);
          // detail 0 = a keyboard click (Enter/Space): no pointer was involved,
          // so a pointerType left over from an earlier tap must not count (CR M1).
          const touch = event.detail !== 0 && pointerType.current !== 'mouse';
          pointerType.current = undefined;
          if (touch) openedByTouch.current = true;
          setOpen(true);
        },
      })
    : children;
  return (
    <BaseTooltip.Root {...controlled}>
      <BaseTooltip.Trigger
        render={trigger}
        delay={delay}
        closeOnClick={openOnPress ? false : undefined}
      />
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
