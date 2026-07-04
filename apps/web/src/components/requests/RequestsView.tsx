// Design ref: ux-design.pen Screen L1-D-v2 (K7fiy)
// Source: ux-design.pen (Pencil app)
/**
 * The Discover-hosted 想要清單 view (Story 13-1b AC #5) — the lit PH3-R2
 * reserved entry lands here (`/discover?view=requests`, nav-ADR:630: NOT a new
 * destination). STATIC list this story (plain fetch + N4 states); 13-3b
 * upgrades it with request_progress SSE / live status / progress % per the
 * recorded SCOPE WALL. NO SSE here.
 */
import { useQuery } from '@tanstack/react-query';
import { RotateCcw } from 'lucide-react';
import { requestService } from '../../services/requestService';
import { requestKeys } from '../../hooks/useRequestedMedia';
import { RequestRow } from './RequestRow';

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

  const requests = query.data ?? [];

  // L6 — empty distinct from failure: calm 尚無請求 + quiet 前往探索.
  if (requests.length === 0) {
    return (
      <div
        data-testid="requests-empty"
        className="flex flex-col items-center gap-3 rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] px-6 py-12 text-center"
      >
        <p className="text-sm text-[var(--text-secondary)]">尚無請求</p>
        <button
          type="button"
          data-testid="requests-go-explore"
          onClick={onExplore}
          className="flex h-11 items-center px-3 text-sm font-medium text-[var(--accent-text)] transition-colors hover:text-[var(--accent-primary)]"
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
          <RequestRow key={request.id} request={request} />
        ))}
      </div>
    </div>
  );
}
