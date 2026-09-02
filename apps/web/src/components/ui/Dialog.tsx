// Implements: <utility — no .pen counterpart>
import * as React from 'react';
import * as DialogPrimitive from '@radix-ui/react-dialog';
import { X } from 'lucide-react';
import { cn } from '@/lib/utils';

const Dialog = DialogPrimitive.Root;
const DialogTrigger = DialogPrimitive.Trigger;
const DialogPortal = DialogPrimitive.Portal;
const DialogClose = DialogPrimitive.Close;

function DialogOverlay({
  className,
  ...props
}: React.ComponentProps<typeof DialogPrimitive.Overlay>) {
  return (
    <DialogPrimitive.Overlay
      className={cn(
        /* --overlay-scrim is the modal-backdrop token and stays DARK in both
           themes: a paper modal on paper ground needs the same boundary a dark
           one does. Was black/60; the token is 70%. */
        /* The animate-in / fade-in-0 family this replaced came from
           tailwindcss-animate, a plugin this project never installed — it
           emitted zero CSS, so every modal in the subtitle flow popped in and
           out on a single frame. Replaced with keyframes styles.css owns and
           the motion tokens drive. */
        'fixed inset-0 z-50 bg-[var(--overlay-scrim)] data-[state=open]:animate-overlay-enter data-[state=closed]:animate-overlay-exit',
        className
      )}
      {...props}
    />
  );
}

function DialogContent({
  className,
  overlayClassName,
  children,
  ...props
}: React.ComponentProps<typeof DialogPrimitive.Content> & {
  /**
   * Extra classes for this dialog's own scrim. Needed because a dialog opened
   * from INSIDE another layer must lift the whole pair: raising only the content
   * leaves the scrim under the host layer, so the thing behind the dialog stays
   * fully lit and the dialog reads as floating over a live UI rather than over a
   * blocked one. Sheet.tsx sits at z-[70]/z-[71], so a dialog opened from a sheet
   * passes both an overlay and a content z above that.
   */
  overlayClassName?: string;
}) {
  return (
    <DialogPortal>
      <DialogOverlay className={overlayClassName} />
      <DialogPrimitive.Content
        className={cn(
          /* `duration-[…]` is gone with them: it sets --tw-duration, which a
             `animation:` shorthand never reads — the timing lives in the
             --animate-dialog-* tokens instead. Enter and exit carry DIFFERENT
             animation-names on purpose; see the ⚠️ in styles.css. */
          'fixed left-1/2 top-1/2 z-50 w-full max-w-lg -translate-x-1/2 -translate-y-1/2 rounded-[var(--radius-xl)] bg-[var(--bg-secondary)] p-6 shadow-[var(--shadow-xl)] data-[state=open]:animate-dialog-enter data-[state=closed]:animate-dialog-exit',
          className
        )}
        {...props}
      >
        {children}
        <DialogPrimitive.Close className="absolute right-4 top-4 rounded-[var(--radius-sm)] text-[var(--text-muted)] opacity-70 transition-opacity hover:opacity-100 focus:outline-none">
          <X className="size-4" />
          <span className="sr-only">Close</span>
        </DialogPrimitive.Close>
      </DialogPrimitive.Content>
    </DialogPortal>
  );
}

function DialogHeader({ className, ...props }: React.ComponentProps<'div'>) {
  return (
    <div className={cn('flex flex-col gap-1.5 text-center sm:text-left', className)} {...props} />
  );
}

function DialogFooter({ className, ...props }: React.ComponentProps<'div'>) {
  return (
    <div
      className={cn('flex flex-col-reverse gap-2 sm:flex-row sm:justify-end', className)}
      {...props}
    />
  );
}

function DialogTitle({ className, ...props }: React.ComponentProps<typeof DialogPrimitive.Title>) {
  return (
    <DialogPrimitive.Title
      className={cn('text-lg font-semibold text-[var(--text-primary)]', className)}
      {...props}
    />
  );
}

function DialogDescription({
  className,
  ...props
}: React.ComponentProps<typeof DialogPrimitive.Description>) {
  return (
    <DialogPrimitive.Description
      className={cn('text-sm text-[var(--text-secondary)]', className)}
      {...props}
    />
  );
}

export {
  Dialog,
  DialogPortal,
  DialogOverlay,
  DialogClose,
  DialogTrigger,
  DialogContent,
  DialogHeader,
  DialogFooter,
  DialogTitle,
  DialogDescription,
};
