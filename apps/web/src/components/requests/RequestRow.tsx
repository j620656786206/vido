// Implements: Component/RequestRow-v2 (LkjRd)
// Source: ux-design.pen (Pencil app)
/**
 * One row of the 想要清單 (Story 13-1b, design L1-D-v2): 40×60 thumb + title +
 * type·date meta (Mono date per Rule TY-3) + status pill through the DL-v2
 * §2.5 shared token map — all five enum statuses wired (capability-honor: only
 * `pending` occurs until 13-3/13-4 land; no bespoke palette). The Mono
 * progress-% slot renders only when a progress value exists (13-3b's SSE
 * supplies it live). The trailing action-area (13-7b, `pbeYJ`) follows the
 * design matrix: 取消 only on pending, fail-caption + 重試 only on failed,
 * nothing interactive on searching/downloading/completed. Actions render only
 * when the parent passes the handler (gallery/static uses stay inert).
 */
import { Film } from 'lucide-react';
import { Button } from '../ui/Button';
import type { MediaRequest, RequestStatus } from '../../services/requestService';

/** DL-v2 §2.5 status→token map — one state machine, no bespoke palette. */
const STATUS_TOKENS: Record<RequestStatus, { label: string; pillBg: string; fg: string }> = {
  pending: { label: '想要', pillBg: 'bg-[var(--info-tint)]', fg: 'text-[var(--info-text)]' },
  // ⚖️ Alexyu 2026-09-11（dsr-11）：泥金，不是赭。搜尋中是「正在跑」；赭色說的是
  // 「你要求了，但它沒發生」。改完之後 searching 與 downloading 同為泥金——那是對的，
  // 兩者都在跑，區別由標籤與 downloading 才有的百分比承擔，不靠顏色。DESIGN.md 明寫
  // 色相上已無處可放第六個詞彙，所以不為了區分而發明新色。
  searching: {
    label: '搜尋中',
    pillBg: 'bg-[var(--accent-tint)]',
    fg: 'text-[var(--accent-text)]',
  },
  downloading: {
    label: '下載中',
    pillBg: 'bg-[var(--accent-tint)]',
    fg: 'text-[var(--accent-text)]',
  },
  completed: {
    label: '已入庫',
    pillBg: 'bg-[var(--success-tint)]',
    fg: 'text-[var(--success-text)]',
  },
  failed: { label: '失敗', pillBg: 'bg-[var(--error-tint)]', fg: 'text-[var(--error-text)]' },
};

// 圓點是填色不是文字，所以五個都用飽和階。dsr-11 之前 downloading 用 --accent-text、
// failed 用 --error-text，那是「給人讀」的那一階（DESIGN.md §兩種金規則）。
const DOT_BG: Record<RequestStatus, string> = {
  pending: 'bg-[var(--info)]',
  searching: 'bg-[var(--accent-primary)]',
  downloading: 'bg-[var(--accent-primary)]',
  completed: 'bg-[var(--success)]',
  failed: 'bg-[var(--error)]',
};

export interface RequestRowProps {
  request: MediaRequest & { progress?: number };
  /** Cancel a pending request (13-7b). Absent → no 取消 button. */
  onCancel?: () => void;
  /** Retry a failed request (13-7b). Absent → no 重試 button. */
  onRetry?: () => void;
  /** This row's cancel/retry is in flight — the button disables (no double-fire). */
  busy?: boolean;
}

/**
 * The request's calendar day on the viewer's clock. The API now stores every
 * timestamp in UTC (bugfix-h), so slicing the ISO text would show yesterday
 * for anything requested before 08:00 in Taipei.
 */
function localDay(iso?: string): string {
  if (!iso) return '';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso.slice(0, 10);
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

export function RequestRow({ request, onCancel, onRetry, busy = false }: RequestRowProps) {
  const token = STATUS_TOKENS[request.status] ?? STATUS_TOKENS.pending;
  const date = localDay(request.requestedAt);
  const caption = request.status === 'failed' ? request.errorMessage : null;
  const showCancel = request.status === 'pending' && !!onCancel;
  const showRetry = request.status === 'failed' && !!onRetry;
  const pctNum =
    request.status === 'downloading' && typeof request.progress === 'number'
      ? Math.round(request.progress * 100)
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

      {/* Title + meta. From md the 7rem floor makes a long fail-caption
          truncate instead of squeezing the title to nothing (768px, 13-7b);
          below md there is no caption beside the buttons, and a floor would
          push 重試 out of the card on a 320–360px phone. */}
      <div className="min-w-0 flex-1 md:min-w-[7rem]">
        <p className="truncate text-sm font-semibold text-[var(--text-primary)]">{request.title}</p>
        <div className="mt-1 flex min-w-0 items-center gap-1.5 overflow-hidden whitespace-nowrap text-xs text-[var(--text-secondary)]">
          <span>{request.mediaType === 'movie' ? '電影' : '影集'}</span>
          <span className="text-[var(--text-muted)]" aria-hidden="true">
            ·
          </span>
          <span className="truncate font-mono tabular-nums">{date}</span>
        </div>
        {/* Below md the trailing cluster has no room for the caption (L4-M-v2
            draws none) — it stays under the title there. */}
        {caption && (
          <p className="mt-1 truncate text-xs text-[var(--error-text)] md:hidden">{caption}</p>
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

      {/* Mono progress slot — populated live by 13-3b's request_progress SSE.
          Figure-only per design L1 (no bar); exposed as a progressbar for a11y
          (DownloadCardV2 pattern). "%" is not a CJK unit, so TY-3 keeps it in the
          number's Mono node — no split. */}
      {pctNum !== null && (
        <span
          role="progressbar"
          aria-valuenow={pctNum}
          aria-valuemin={0}
          aria-valuemax={100}
          aria-label={`${request.title} 下載進度`}
          className="shrink-0 font-mono text-xs tabular-nums text-[var(--accent-text)]"
        >
          {pctNum}%
        </span>
      )}

      {/* Action-area (pbeYJ). Failed override iyYqV: fail-caption XJ1hS + 重試
          qDo2F (ButtonSecondary YDPhc at 44px), gap 10. Pending: 取消 yTntT — a
          plain text button, $text-secondary, 44px, padding [0, md-plus]. */}
      {(caption || showRetry) && (
        <div className="flex min-w-0 shrink items-center gap-2.5">
          {caption && (
            <span
              data-testid="request-fail-caption"
              title={caption}
              className="hidden max-w-[16rem] truncate text-xs text-[var(--error-text)] md:inline"
            >
              {caption}
            </span>
          )}
          {showRetry && (
            <Button
              type="button"
              variant="secondary"
              data-testid="request-retry-btn"
              onClick={onRetry}
              disabled={busy}
              aria-label={`重試請求：${request.title}`}
              className="h-11 shrink-0 px-5 focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)]"
            >
              重試
            </Button>
          )}
        </div>
      )}
      {showCancel && (
        <button
          type="button"
          data-testid="request-cancel-btn"
          onClick={onCancel}
          disabled={busy}
          aria-label={`取消請求：${request.title}`}
          className="flex h-11 shrink-0 items-center justify-center rounded-[var(--radius-md)] px-3.5 text-sm text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)] focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)] disabled:pointer-events-none disabled:opacity-50"
        >
          取消
        </button>
      )}
    </div>
  );
}
