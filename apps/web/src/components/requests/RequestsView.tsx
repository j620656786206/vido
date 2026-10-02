// Design ref: ux-design.pen Screen L1-D-v2 (K7fiy) · L5-D-v2 (ER39x) · L6-D-v2 (x4CNb) · L7-D-v2 (oopme)
// 這個檔案同時實作四張稿：清單、載入骨架、空狀態、fail-soft。原本只列了 L1。（dsr-11）
// Source: ux-design.pen (Pencil app)
/**
 * The Discover-hosted 想要清單 view (Story 13-1b AC #5) — the lit PH3-R2
 * reserved entry lands here (`/discover?view=requests`, nav-ADR:630: NOT a new
 * destination). STATIC list this story (plain fetch + N4 states); 13-3b
 * upgrades it with request_progress SSE / live status / progress % per the
 * recorded SCOPE WALL. NO SSE here.
 */
import { useEffect, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import { useQuery } from '@tanstack/react-query';
import { Check, RotateCcw } from 'lucide-react';
import { requestService } from '../../services/requestService';
import { requestKeys } from '../../hooks/useRequestedMedia';
import { useRequestActions, usePendingRequestActions } from '../../hooks/useRequestActions';
import { RequestRow } from './RequestRow';

type ActionToast = { kind: 'success' | 'error'; message: string } | null;

export interface RequestsViewProps {
  /** Quiet 前往探索 affordance on the empty state — clears `?view=requests`. */
  onExplore: () => void;
}

export function RequestsView({ onExplore }: RequestsViewProps) {
  // Same key + options as useRequestedMedia (TanStack dedupes the fetch); this
  // view holds its own useQuery because it needs error/refetch for the L7
  // fail-soft, which the lookup hook deliberately doesn't expose (CR L1).
  const query = useQuery({
    queryKey: requestKeys.list(),
    queryFn: () => requestService.listRequests(),
    staleTime: 30 * 1000,
    retry: 1,
  });

  // 13-7b — row actions + their feedback toast. The toast lives HERE, not in
  // the row: a cancelled row is removed optimistically, and a toast inside it
  // would unmount with it.
  const { cancel, retry } = useRequestActions();
  const [toast, setToast] = useState<ActionToast>(null);
  const toastTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  useEffect(
    () => () => {
      if (toastTimerRef.current) clearTimeout(toastTimerRef.current);
    },
    []
  );
  const showToast = (next: NonNullable<ActionToast>) => {
    if (toastTimerRef.current) clearTimeout(toastTimerRef.current);
    setToast(next);
    toastTimerRef.current = setTimeout(() => setToast(null), 4000);
  };
  // mutateAsync, not mutate(id, { onError }): per-call callbacks only fire for
  // the LAST mutate() on a hook, so a failure of an earlier overlapping
  // action would show no toast at all (13-7b CR). The promise is per call.
  const handleCancel = (id: string) =>
    cancel
      .mutateAsync(id)
      .then(() => showToast({ kind: 'success', message: '已取消請求' }))
      .catch((error: Error) => showToast({ kind: 'error', message: error.message }));
  const handleRetry = (id: string) =>
    retry
      .mutateAsync(id)
      .then(() => showToast({ kind: 'success', message: '已重新嘗試，開始搜尋來源' }))
      .catch((error: Error) => showToast({ kind: 'error', message: error.message }));
  const { cancelling, retrying } = usePendingRequestActions();

  // The toast renders beside EVERY state: cancelling the last row flips the
  // view to L6 empty, and the confirmation must survive that.
  return (
    <>
      {renderBody()}
      {toast && <RequestActionToast toast={toast} />}
    </>
  );

  function renderBody() {
    // L5 — list-shaped skeleton, reduced-motion respected.
    if (query.isLoading) {
      return (
        <div data-testid="requests-skeleton" aria-busy="true" className="space-y-3">
          {Array.from({ length: 4 }).map((_, i) => (
            <div
              key={i}
              className="h-[72px] animate-pulse rounded-[var(--radius-lg)] bg-[var(--bg-secondary)] motion-reduce:animate-none"
            />
          ))}
        </div>
      );
    }

    // L7 — status-source fail-soft: inline retry, shell/toolbar keep rendering.
    if (query.isError) {
      return (
        <div
          data-testid="requests-error"
          role="alert"
          className="flex flex-col items-center gap-3 rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] px-6 py-10 text-center"
        >
          <p className="text-sm text-[var(--text-secondary)]">無法載入請求狀態</p>
          {/* dsr-11: 主行說「發生了什麼」，這一行說「為什麼」。兩者不重複，L7-D 一直有。 */}
          <p className="text-xs text-[var(--text-muted)]">請求服務暫時無法連線</p>
          <button
            type="button"
            data-testid="requests-retry"
            onClick={() => query.refetch()}
            className="flex h-11 items-center gap-2 rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-4 text-sm text-[var(--text-primary)] transition-colors hover:bg-[var(--bg-tertiary)]/80"
          >
            <RotateCcw className="h-4 w-4" aria-hidden="true" />
            重試
          </button>
        </div>
      );
    }

    // A row whose cancel is in flight stays hidden even if an SSE snapshot
    // re-adds it before the DELETE lands (it is still active in the DB).
    const requests = (query.data ?? []).filter((row) => !cancelling.has(row.id));

    // L6 — empty distinct from failure: calm 尚無請求 + quiet 前往探索.
    if (requests.length === 0) {
      return (
        <div
          data-testid="requests-empty"
          className="flex flex-col items-center gap-3 rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] px-6 py-12 text-center"
        >
          <p className="text-sm text-[var(--text-secondary)]">尚無請求</p>
          {/* dsr-11: 「尚無請求」只說了發生什麼。L6-D 一直畫著這一行，它說的是怎麼開始
              ——空狀態的職責是「永遠給出下一步」，少了它使用者只知道這裡是空的。 */}
          <p className="text-xs text-[var(--text-muted)]">從探索或詳情頁按「想要」開始追蹤</p>
          <button
            type="button"
            data-testid="requests-go-explore"
            onClick={onExplore}
            className="flex h-11 items-center px-3 text-sm font-medium text-[var(--accent-text)] transition-colors hover:underline"
          >
            前往探索
          </button>
        </div>
      );
    }

    // L1 — the request list: header (count in Mono per TY-3) + rows.
    return (
      <div data-testid="requests-view">
        <div className="mb-3 flex items-center gap-2">
          <h2 className="text-base font-semibold text-[var(--text-primary)]">想要清單</h2>
          <span className="text-xs text-[var(--text-secondary)]">
            <span className="font-mono tabular-nums">{requests.length}</span> 筆
          </span>
        </div>
        <div className="space-y-3">
          {requests.map((request) => (
            <RequestRow
              key={request.id}
              request={request}
              onCancel={() => handleCancel(request.id)}
              onRetry={() => handleRetry(request.id)}
              busy={cancelling.has(request.id) || retrying.has(request.id)}
            />
          ))}
        </div>
      </div>
    );
  }
}

/**
 * Cancel/retry feedback — the 13-1b RequestToast portal pattern (no global
 * toast lib): portalled to <body>, polite status on success, alert on error,
 * 4s auto-dismiss owned by the caller.
 */
function RequestActionToast({ toast }: { toast: NonNullable<ActionToast> }) {
  return createPortal(
    <div
      data-testid="request-action-toast"
      role={toast.kind === 'error' ? 'alert' : 'status'}
      aria-live={toast.kind === 'error' ? 'assertive' : 'polite'}
      className="fixed bottom-6 left-1/2 z-50 flex -translate-x-1/2 items-center gap-3 rounded-[var(--radius-lg)] bg-[var(--bg-tertiary)] px-[18px] py-3.5 shadow-[var(--shadow-xl)]"
    >
      {toast.kind === 'success' && (
        <Check className="h-[18px] w-[18px] text-[var(--success-text)]" aria-hidden="true" />
      )}
      <span
        className={`text-sm font-semibold ${
          toast.kind === 'error' ? 'text-[var(--error-text)]' : 'text-[var(--text-primary)]'
        }`}
      >
        {toast.message}
      </span>
    </div>,
    document.body
  );
}
