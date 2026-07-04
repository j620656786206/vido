// Design ref: ux-design.pen Screen L2-D-v2 (VH3Tq)
// Source: ux-design.pen (Pencil app)
/**
 * The one-click 想要 button (Story 13-1b, Epic 13 G-1/P3-001) — three honest
 * states per design L2: 可請求 (accent button「＋ 想要」) / 已請求·處理中
 * ($info-tint pill, non-actionable — no duplicate requests from the UI) /
 * 已入庫 ($success-tint pill, no action). Success feedback = the L8 toast
 * (已加入想要清單 + 查看清單 → the Discover-hosted 想要清單 view).
 */
import { useEffect, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import { useNavigate } from '@tanstack/react-router';
import { Check, Loader2, Plus } from 'lucide-react';
import { cn } from '../../lib/utils';
import { useRequestActions } from '../../hooks/useRequestActions';
import type { RequestMediaType } from '../../services/requestService';

export interface RequestButtonProps {
  tmdbId: number;
  mediaType: RequestMediaType;
  /** Display title for the optimistic row + toast (server re-resolves its own). */
  title: string;
  owned: boolean;
  requested: boolean;
  fullWidth?: boolean;
  className?: string;
}

type ToastState = { kind: 'success' } | { kind: 'error'; message: string } | null;

export function RequestButton({
  tmdbId,
  mediaType,
  title,
  owned,
  requested,
  fullWidth,
  className,
}: RequestButtonProps) {
  const navigate = useNavigate();
  const { create } = useRequestActions();
  const [toast, setToast] = useState<ToastState>(null);
  const toastTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    return () => {
      if (toastTimerRef.current) clearTimeout(toastTimerRef.current);
    };
  }, []);

  const showToast = (next: ToastState) => {
    if (toastTimerRef.current) clearTimeout(toastTimerRef.current);
    setToast(next);
    toastTimerRef.current = setTimeout(() => setToast(null), 4000);
  };

  // Card contexts render this inside a <Link> — every interactive element must
  // stop the navigation (PosterCard kebab precedent).
  const guard = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
  };

  const handleRequest = (e: React.MouseEvent) => {
    guard(e);
    if (create.isPending) return;
    create.mutate(
      { tmdbId, mediaType, title },
      {
        onSuccess: () => showToast({ kind: 'success' }),
        onError: (error) => {
          // REQUEST_DUPLICATE settles into the requested state upstream — the
          // optimistic row stands, so no error surface here (AC #4).
          if ((error as { code?: string }).code === 'REQUEST_DUPLICATE') {
            showToast({ kind: 'success' });
            return;
          }
          showToast({ kind: 'error', message: error.message });
        },
      }
    );
  };

  // 已入庫 — $success-tint pill, no action (design L2 states-strip).
  if (owned) {
    return (
      <span
        data-testid="request-pill-owned"
        role="status"
        aria-live="polite"
        className={cn(
          'inline-flex items-center gap-1.5 rounded-full bg-[var(--success-tint)] px-4 py-2.5 text-[13px] font-semibold text-[var(--success)]',
          fullWidth && 'w-full justify-center',
          className
        )}
      >
        <Check className="h-3.5 w-3.5" aria-hidden="true" />
        已入庫
      </span>
    );
  }

  // 已請求·處理中 — $info-tint pill with dot, non-actionable (no duplicates).
  // The toast renders as a sibling: after a successful create the optimistic
  // cache flips `requested` true, this branch takes over, and the success
  // toast must survive that flip.
  if (requested || create.isPending) {
    return (
      <>
        <span
          data-testid="request-pill-requested"
          role="status"
          aria-live="polite"
          className={cn(
            'inline-flex items-center gap-1.5 rounded-full bg-[var(--info-tint)] px-4 py-2.5 text-[13px] font-semibold text-[var(--info)]',
            fullWidth && 'w-full justify-center',
            className
          )}
        >
          {create.isPending ? (
            <Loader2
              className="h-3.5 w-3.5 animate-spin motion-reduce:animate-none"
              aria-hidden="true"
            />
          ) : (
            <span className="h-1.5 w-1.5 rounded-full bg-[var(--info)]" aria-hidden="true" />
          )}
          已請求 · 處理中
        </span>
        {toast && <RequestToast toast={toast} onView={navigate} guard={guard} />}
      </>
    );
  }

  // 可請求 — ButtonPrimary「＋ 想要」, 44px touch floor (design otvKh ref).
  return (
    <>
      <button
        type="button"
        data-testid="request-button"
        onClick={handleRequest}
        className={cn(
          'inline-flex h-11 items-center justify-center gap-2 rounded-[var(--radius-md)] bg-[var(--accent-primary)] px-4 text-sm font-semibold text-[var(--text-on-accent)] shadow-[var(--shadow-sm)] transition-colors hover:bg-[var(--accent-hover)] active:bg-[var(--accent-pressed)] focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)]',
          fullWidth && 'w-full',
          className
        )}
      >
        <Plus className="h-4 w-4" aria-hidden="true" />
        想要
      </button>
      {toast && <RequestToast toast={toast} onView={navigate} guard={guard} />}
    </>
  );
}

/**
 * L8 request-submitted feedback — inline transient toast (no global toast lib;
 * `$type.$id.tsx` role="status" precedent). 查看清單 deep-links to the
 * Discover-hosted 想要清單 (`?view=requests`).
 */
function RequestToast({
  toast,
  onView,
  guard,
}: {
  toast: NonNullable<ToastState>;
  onView: ReturnType<typeof useNavigate>;
  guard: (e: React.MouseEvent) => void;
}) {
  // CR H1: portal to <body> — card contexts mount this inside PosterCard's
  // clip-path + transform-gpu container, and BOTH establish a containing block
  // for fixed-position descendants (the toast would position against the card
  // and get clipped). A portal escapes any transformed/clipped ancestor.
  return createPortal(
    <div
      data-testid="request-toast"
      role={toast.kind === 'error' ? 'alert' : 'status'}
      aria-live="polite"
      className="fixed bottom-6 left-1/2 z-50 flex -translate-x-1/2 items-center gap-3 rounded-[var(--radius-lg)] bg-[var(--bg-tertiary)] px-[18px] py-3.5 shadow-[var(--shadow-xl)]"
    >
      {toast.kind === 'success' ? (
        <>
          <Check className="h-[18px] w-[18px] text-[var(--success)]" aria-hidden="true" />
          <span className="text-sm font-semibold text-[var(--text-primary)]">已加入想要清單</span>
          <button
            type="button"
            data-testid="request-toast-view"
            onClick={(e) => {
              guard(e);
              onView({ to: '/discover', search: { view: 'requests' } });
            }}
            className="flex h-11 items-center px-2.5 text-sm font-semibold text-[var(--accent-text)]"
          >
            查看清單
          </button>
        </>
      ) : (
        <span className="text-sm font-semibold text-[var(--error-text)]">{toast.message}</span>
      )}
    </div>,
    document.body
  );
}
