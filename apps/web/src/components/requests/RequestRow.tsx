// Implements: Component/RequestRow-v2 (LkjRd)
// Source: ux-design.pen (Pencil app)
/**
 * One row of the 想要清單 (Story 13-1b, design L1-D-v2): 40×60 thumb + title +
 * type·date meta (Mono date per Rule TY-3) + status pill through the DL-v2
 * §2.5 shared token map — all five enum statuses wired (capability-honor: only
 * `pending` occurs until 13-3/13-4 land; no bespoke palette). The Mono
 * progress-% slot renders only when a progress value exists (13-3b's SSE
 * supplies it live). The design's cancel/retry action-area is deliberately NOT
 * built — no backend endpoint exists yet (Rule 24 lane ③: 13-7 in
 * sprint-status.yaml).
 */
import { Film } from 'lucide-react';
import type { MediaRequest, RequestStatus } from '../../services/requestService';

/** DL-v2 §2.5 status→token map — one state machine, no bespoke palette. */
const STATUS_TOKENS: Record<RequestStatus, { label: string; pillBg: string; fg: string }> = {
  pending: { label: '想要', pillBg: 'bg-[var(--info-tint)]', fg: 'text-[var(--info)]' },
  searching: { label: '搜尋中', pillBg: 'bg-[var(--warning-tint)]', fg: 'text-[var(--warning)]' },
  downloading: {
    label: '下載中',
    pillBg: 'bg-[var(--accent-tint)]',
    fg: 'text-[var(--accent-text)]',
  },
  completed: { label: '已入庫', pillBg: 'bg-[var(--success-tint)]', fg: 'text-[var(--success)]' },
  failed: { label: '失敗', pillBg: 'bg-[var(--error-tint)]', fg: 'text-[var(--error-text)]' },
};

const DOT_BG: Record<RequestStatus, string> = {
  pending: 'bg-[var(--info)]',
  searching: 'bg-[var(--warning)]',
  downloading: 'bg-[var(--accent-text)]',
  completed: 'bg-[var(--success)]',
  failed: 'bg-[var(--error-text)]',
};

export interface RequestRowProps {
  request: MediaRequest & { progress?: number };
}

export function RequestRow({ request }: RequestRowProps) {
  const token = STATUS_TOKENS[request.status] ?? STATUS_TOKENS.pending;
  const date = request.requestedAt?.slice(0, 10) ?? '';
  const pct =
    request.status === 'downloading' && typeof request.progress === 'number'
      ? `${Math.round(request.progress * 100)}%`
      : null;

  return (
    <div
      data-testid="request-row"
      className="flex items-center gap-3.5 rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] px-4 py-1.5"
    >
      {/* Thumb — 40×60 placeholder (design: film icon on $bg-tertiary) */}
      <div
        className="flex h-[60px] w-10 shrink-0 items-center justify-center overflow-hidden rounded-[var(--radius-md)] bg-[var(--bg-tertiary)]"
        aria-hidden="true"
      >
        <Film className="h-[18px] w-[18px] text-[var(--text-muted)]" />
      </div>

      {/* Title + meta */}
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-semibold text-[var(--text-primary)]">{request.title}</p>
        <div className="mt-1 flex items-center gap-1.5 text-xs text-[var(--text-secondary)]">
          <span>{request.mediaType === 'movie' ? '電影' : '影集'}</span>
          <span className="text-[var(--text-muted)]" aria-hidden="true">
            ·
          </span>
          <span className="font-mono tabular-nums">{date}</span>
        </div>
        {request.status === 'failed' && request.errorMessage && (
          <p className="mt-1 truncate text-xs text-[var(--error-text)]">{request.errorMessage}</p>
        )}
      </div>

      {/* Status pill — announced politely on async transitions (13-3b SSE) */}
      <span
        data-testid={`request-status-${request.status}`}
        role="status"
        aria-live="polite"
        className={`inline-flex shrink-0 items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-semibold ${token.pillBg} ${token.fg}`}
      >
        <span
          className={`h-1.5 w-1.5 rounded-full ${DOT_BG[request.status] ?? DOT_BG.pending}`}
          aria-hidden="true"
        />
        {token.label}
      </span>

      {/* Mono progress slot — populated by 13-3b's request_progress SSE */}
      {pct && (
        <span className="shrink-0 font-mono text-[13px] tabular-nums text-[var(--accent-text)]">
          {pct}
        </span>
      )}
    </div>
  );
}
