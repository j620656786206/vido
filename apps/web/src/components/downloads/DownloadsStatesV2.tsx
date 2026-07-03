// Design ref: ux-design.pen Screen D5-D-v2 (dVPuY)
// (also renders D4-D-v2 skeleton (T95wy) + D6-D-v2 qBT-unreachable fail-soft (UNVRU))
import { Link } from '@tanstack/react-router';
import { Inbox, PlugZap } from 'lucide-react';
import { cn } from '../../lib/utils';
import { Button } from '../ui/Button';
import type { FilterStatus } from '../../services/downloadService';

const CARD =
  'rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)]';

/** D4 — card-shaped loading skeleton (toolbar pills + a few cards). */
export function DownloadsSkeletonV2() {
  return (
    <div data-testid="downloads-skeleton-v2" aria-busy="true" className="flex flex-col gap-3">
      <div className="flex flex-wrap gap-2">
        {Array.from({ length: 6 }).map((_, i) => (
          <div key={i} className="h-8 w-20 animate-pulse rounded-full bg-[var(--bg-tertiary)]" />
        ))}
      </div>
      {Array.from({ length: 4 }).map((_, i) => (
        <div key={i} className={cn(CARD, 'p-4')}>
          <div className="h-4 w-2/3 animate-pulse rounded bg-[var(--bg-tertiary)]" />
          <div className="mt-3 h-2 w-full animate-pulse rounded-full bg-[var(--bg-tertiary)]" />
          <div className="mt-2 h-3 w-1/3 animate-pulse rounded bg-[var(--bg-tertiary)]" />
        </div>
      ))}
    </div>
  );
}

/** ux3-4-4 — table-shaped loading skeleton (the variant ux3-4-1 deferred); a header row + a few rows. */
export function DownloadsTableSkeletonV2() {
  return (
    <div
      data-testid="downloads-table-skeleton-v2"
      aria-busy="true"
      className={cn(CARD, 'overflow-hidden')}
    >
      <div className="h-9 border-b border-[var(--border-subtle)] bg-[var(--bg-secondary)]" />
      {Array.from({ length: 6 }).map((_, i) => (
        <div
          key={i}
          className="flex items-center gap-3 border-b border-[var(--border-subtle)] px-3 py-3"
        >
          <div className="h-4 w-4 shrink-0 animate-pulse rounded bg-[var(--bg-tertiary)]" />
          <div className="h-3 flex-1 animate-pulse rounded bg-[var(--bg-tertiary)]" />
          <div className="h-3 w-24 animate-pulse rounded bg-[var(--bg-tertiary)]" />
          <div className="h-3 w-20 animate-pulse rounded bg-[var(--bg-tertiary)]" />
        </div>
      ))}
    </div>
  );
}

const EMPTY_MESSAGES: Record<FilterStatus, string> = {
  all: '目前沒有下載任務',
  downloading: '沒有正在下載的任務',
  paused: '沒有已暫停的任務',
  completed: '沒有已完成的任務',
  seeding: '沒有正在做種的任務',
  error: '沒有發生錯誤的任務',
};

/**
 * D5 — empty state. The `all` filter (genuinely no downloads) gets a distinct message + a quiet
 * 前往探索 affordance (never a bare blank, AC6); other filters get a switch-filter hint.
 */
export function DownloadsEmptyV2({ filter }: { filter: FilterStatus }) {
  return (
    <div
      data-testid="downloads-empty-v2"
      className={cn(CARD, 'flex flex-col items-center px-6 py-12 text-center')}
    >
      <Inbox className="h-10 w-10 text-[var(--text-muted)]" aria-hidden="true" />
      <p className="mt-4 text-base text-[var(--text-primary)]">{EMPTY_MESSAGES[filter]}</p>
      {filter === 'all' ? (
        <>
          <p className="mt-1 text-sm text-[var(--text-secondary)]">
            在 qBittorrent 中新增種子後會自動顯示
          </p>
          <Button asChild variant="outline" size="sm" className="mt-4">
            <Link to="/discover">前往探索</Link>
          </Button>
        </>
      ) : (
        <p className="mt-1 text-sm text-[var(--text-secondary)]">嘗試切換其他篩選條件</p>
      )}
    </div>
  );
}

/**
 * D6 — qBittorrent-unreachable per-section fail-soft. Covers BOTH "not configured" and "configured
 * but the poll errored": the shell + nav still render, the page never hard-fails, and the user gets
 * 重試 + 前往設定 (AC6). `onRetry` re-runs the query.
 */
export function DownloadsQbtErrorV2({
  onRetry,
  message,
}: {
  onRetry: () => void;
  message?: string;
}) {
  return (
    <div
      data-testid="downloads-qbt-error-v2"
      role="alert"
      className={cn(
        'flex flex-col items-center rounded-[var(--radius-lg)] border border-[var(--error)] bg-[var(--error-tint)] px-6 py-12 text-center'
      )}
    >
      <PlugZap className="h-10 w-10 text-[var(--error-text)]" aria-hidden="true" />
      <p className="mt-4 text-base text-[var(--text-primary)]">無法連線到 qBittorrent</p>
      <p className="mt-1 text-sm text-[var(--text-secondary)]">
        {message ?? '請確認 qBittorrent 正在執行，或前往設定檢查連線。'}
      </p>
      <div className="mt-4 flex items-center gap-3">
        <Button variant="default" size="sm" onClick={onRetry}>
          重試
        </Button>
        <Button asChild variant="outline" size="sm">
          <Link to="/settings">前往設定</Link>
        </Button>
      </div>
    </div>
  );
}
