// Design ref: ux-design.pen Screen L2-D-v2 (VH3Tq) · L8-D-v2 (G0xib) · B4p-D (N2fmG6)
// L2 是三態按鈕本身，L8 是它送出後那顆「已加入想要清單」toast 所在的詳情頁。（dsr-11）
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
import { SeasonEpisodeTreeDialog } from './SeasonEpisodeTreeDialog';
import type { RequestPayload } from '../../utils/requestSelection';

export interface RequestButtonProps {
  tmdbId: number;
  mediaType: RequestMediaType;
  /** Display title for the optimistic row + toast (server re-resolves its own). */
  title: string;
  owned: boolean;
  requested: boolean;
  fullWidth?: boolean;
  className?: string;
  /**
   * 13-2b: for a tv show, 想要 opens the L3 season/episode tree instead of
   * requesting the whole title at once (the detail page sets this; card
   * contexts keep the one-click whole-title request).
   */
  pickEpisodes?: boolean;
  /**
   * 13-2c: the tree may NOT fall back to "no coverage = nothing owned" — on an
   * owned series that would send a whole-title request the backend refuses.
   */
  treeRequiresCoverage?: boolean;
  /** 13-2c: Secondary (YDPhc) for places where the page already has its one solid accent. */
  variant?: 'primary' | 'secondary';
  /** Button text; defaults to 想要. */
  label?: string;
  /**
   * The button sits on a poster's --overlay-scrim (PosterCard hover overlay).
   * The 已請求 pill's --info-tint is ~20% alpha, so over the dark veil 日巡's
   * dark --info-text measured 1.36–1.82:1. With this set the pill gets an
   * OPAQUE --bg-secondary underlay — the HeroBanner / PosterCardV2 precedent —
   * so the tint composites over the ground the contrast gate measures.
   */
  onScrim?: boolean;
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
  pickEpisodes = false,
  treeRequiresCoverage = false,
  variant = 'primary',
  label = '想要',
  onScrim = false,
}: RequestButtonProps) {
  const navigate = useNavigate();
  const { create } = useRequestActions();
  const [toast, setToast] = useState<ToastState>(null);
  const [treeOpen, setTreeOpen] = useState(false);
  const [treeError, setTreeError] = useState<string | null>(null);
  // After the tree closes the trigger has turned into the 已請求 pill — focus
  // goes there instead of falling to <body> (13-2b CR).
  const pillRef = useRef<HTMLElement | null>(null);
  // …or back to the 想要 button when a failed submit brought it back: the node
  // Radix remembered unmounted during the pending pill (13-2c CR).
  const buttonRef = useRef<HTMLButtonElement | null>(null);
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

  const opensTree = pickEpisodes && mediaType === 'tv';

  const submit = (
    selection?: { seasons?: number[]; episodes?: Record<string, number[]> },
    fromTree = false
  ) => {
    if (create.isPending) return;
    create.mutate(
      { tmdbId, mediaType, title, ...selection },
      {
        onSuccess: () => {
          if (fromTree) setTreeOpen(false);
          showToast({ kind: 'success' });
        },
        onError: (error) => {
          // REQUEST_DUPLICATE settles into the requested state upstream — the
          // optimistic row stands, so no error surface here (AC #4).
          if ((error as { code?: string }).code === 'REQUEST_DUPLICATE') {
            if (fromTree) setTreeOpen(false);
            showToast({ kind: 'success' });
            return;
          }
          // From the tree: keep it open with the picks intact and say why in
          // place, instead of closing and losing the selection (13-2b CR).
          if (fromTree) {
            setTreeError(error.message);
            return;
          }
          showToast({ kind: 'error', message: error.message });
        },
      }
    );
  };

  const handleRequest = (e: React.MouseEvent) => {
    guard(e);
    if (opensTree) {
      setTreeOpen(true);
      return;
    }
    submit();
  };

  // The tree's 確認請求: everything checked on a show with nothing owned is the
  // same whole-title request as one click (no selection on the wire, 13-2b).
  const handleTreeConfirm = (payload: RequestPayload) => {
    setTreeError(null);
    submit(
      payload.whole ? undefined : { seasons: payload.seasons, episodes: payload.episodes },
      true
    );
  };

  const tree = opensTree ? (
    <SeasonEpisodeTreeDialog
      open={treeOpen}
      onOpenChange={(open) => {
        setTreeOpen(open);
        if (!open) setTreeError(null);
      }}
      tmdbId={tmdbId}
      title={title}
      onConfirm={handleTreeConfirm}
      submitting={create.isPending}
      submitError={treeError}
      requireCoverage={treeRequiresCoverage}
      onCloseAutoFocus={(e) => {
        const target = pillRef.current ?? buttonRef.current;
        if (target) {
          e.preventDefault();
          target.focus();
        }
      }}
    />
  ) : null;

  // 已入庫 — $success-tint pill, no action (design L2 states-strip).
  if (owned) {
    return (
      <span
        data-testid="request-pill-owned"
        role="status"
        aria-live="polite"
        className={cn(
          'inline-flex items-center gap-1.5 rounded-full bg-[var(--success-tint)] px-4 py-2.5 text-xs font-semibold text-[var(--success-text)]',
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
    const requestedPill = (
      <span
        ref={pillRef}
        tabIndex={-1}
        data-testid="request-pill-requested"
        role="status"
        aria-live="polite"
        className={cn(
          'inline-flex items-center gap-1.5 rounded-full bg-[var(--info-tint)] px-4 py-2.5 text-xs font-semibold text-[var(--info-text)]',
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
    );
    return (
      <>
        {onScrim ? (
          <span
            data-testid="request-pill-underlay"
            className={cn(
              'inline-flex rounded-full bg-[var(--bg-secondary)]',
              fullWidth && 'w-full'
            )}
          >
            {requestedPill}
          </span>
        ) : (
          requestedPill
        )}
        {toast && <RequestToast toast={toast} onView={navigate} guard={guard} />}
        {tree}
      </>
    );
  }

  // 可請求 — ButtonPrimary「＋ 想要」, 44px touch floor (design otvKh ref).
  return (
    <>
      <button
        ref={buttonRef}
        type="button"
        data-testid="request-button"
        onClick={handleRequest}
        className={cn(
          'inline-flex h-11 items-center justify-center gap-2 rounded-[var(--radius-md)] text-sm font-semibold transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)]',
          variant === 'secondary'
            ? // ButtonSecondary YDPhc at 44px, padding [0, lg-plus=20], plus icon 18 (B4p-D xn9Tr)
              'bg-[var(--bg-tertiary)] px-5 text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)]/80'
            : 'bg-[var(--accent-primary)] px-4 text-[var(--text-on-accent)] hover:bg-[var(--accent-hover)] active:bg-[var(--accent-pressed)]',
          fullWidth && 'w-full',
          className
        )}
      >
        <Plus
          className={variant === 'secondary' ? 'h-[18px] w-[18px]' : 'h-4 w-4'}
          aria-hidden="true"
        />
        {label}
      </button>
      {toast && <RequestToast toast={toast} onView={navigate} guard={guard} />}
      {tree}
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
          <Check className="h-[18px] w-[18px] text-[var(--success-text)]" aria-hidden="true" />
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
