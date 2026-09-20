// Implements: Component/BottomSheet (SG1ln)
/**
 * The phone bottom-sheet shell shared by the subtitle flow's dialogs (dsr-6f-1).
 *
 * These dialogs are ONE component with two looks: a centred Radix dialog from
 * 640px up, a bottom sheet below. That is why they stay on `ui/Dialog` rather
 * than moving to `ui/Sheet` (Base UI), which is for panels that exist only on a
 * phone — splitting each into two components would mean two focus-management
 * stacks, and ManageSubtitleDialogV2 nests the glossary dialog inside itself.
 *
 * Until this file, the shell string below was pasted into four components and
 * the "sheet" scaled-and-faded in place at the bottom edge (the desktop
 * `dialog-enter`), which reads as the screen twitching rather than as something
 * arriving. Now it slides up — but ONLY under `max-sm:`: from `sm:` the dialog
 * is centred with `translate`, the very property the sheet animation drives.
 */
import type { ComponentProps } from 'react';
import { cn } from '../../lib/utils';

/** Pin to the bottom edge, round the top corners, slide up (phone only). */
export const MOBILE_SHEET_CONTENT =
  'bottom-0 left-0 right-0 top-auto w-full max-w-none translate-x-0 translate-y-0 rounded-b-none rounded-t-[var(--radius-xl)] max-sm:data-[state=open]:animate-sheet-enter max-sm:data-[state=closed]:animate-sheet-exit';

/**
 * Phone-only sizing for ui/Dialog's ✕ (pass as `closeClassName`): a 44×44 hit
 * area with the drawn 18px icon. On the F1-M sheet there is no footer and no
 * 關閉 button, so ✕ is the only button-shaped way out — 16×16 will not do.
 *
 * Deliberately NO `top`: it depends on the header under it (grabber 16px + a
 * 44px or a 56px title row), so each dialog adds its own `max-sm:top-…`.
 * `max-sm:right-1` + 44 = 48 = the headers' `pr-12`, so the title never runs
 * under the button.
 */
export const MOBILE_SHEET_CLOSE =
  'max-sm:right-1 max-sm:flex max-sm:h-11 max-sm:w-11 max-sm:items-center max-sm:justify-center max-sm:opacity-100 max-sm:text-[var(--text-secondary)] max-sm:[&_svg]:size-[18px]';

/** The drag-handle bar (36×4). Decorative — dragging is not implemented. */
export function SheetGrabber({ className, ...props }: ComponentProps<'div'>) {
  return (
    <div {...props} className={cn('flex shrink-0 justify-center pb-1 pt-2 sm:hidden', className)}>
      <span aria-hidden="true" className="h-1 w-9 rounded-full bg-[var(--bg-tertiary)]" />
    </div>
  );
}
