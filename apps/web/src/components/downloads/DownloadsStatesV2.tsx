// Design ref: ux-design.pen Screen D5-D-v2 (dVPuY)
// (also renders D4-D-v2 skeleton (T95wy) + D6-D-v2 qBT-unreachable fail-soft (UNVRU)
//  + D11-D-v2 qBT-not-configured (tQex7))
import { Link } from '@tanstack/react-router';
import { Inbox, Plug, WifiOff } from 'lucide-react';
import { cn } from '../../lib/utils';
import { Button } from '../ui/Button';
import type { FilterStatus } from '../../services/downloadService';

const CARD =
  'rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)]';
const PULSE = 'animate-pulse bg-[var(--bg-tertiary)] motion-reduce:animate-none';

/** D4 — card-shaped loading skeleton. The page's real chips and toolbar stay above it. */
export function DownloadsSkeletonV2() {
  return (
    <div data-testid="downloads-skeleton-v2" aria-busy="true" className="flex flex-col gap-4">
      {[300, 260, 340, 280].map((width, i) => (
        <div key={i} className={cn(CARD, 'flex flex-col gap-3 p-4')}>
          <div className="flex items-center justify-between gap-3">
            <div
              className={cn(PULSE, 'h-4 max-w-[70%] rounded-[var(--radius-md)]')}
              style={{ width }}
            />
            <div className={cn(PULSE, 'h-6 w-[70px] rounded-full')} />
          </div>
          <div className={cn(PULSE, 'h-2 w-full rounded-[var(--radius-sm)]')} />
          <div className="flex gap-4">
            {[84, 72, 64].map((w, j) => (
              <div
                key={j}
                className={cn(PULSE, 'h-3 rounded-[var(--radius-sm)]')}
                style={{ width: w }}
              />
            ))}
          </div>
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
      <div className="h-11 border-b border-[var(--border-subtle)] bg-[var(--bg-tertiary)]" />
      {Array.from({ length: 6 }).map((_, i) => (
        <div
          key={i}
          className="flex items-center gap-3 border-b border-[var(--border-subtle)] px-3 py-4"
        >
          <div className={cn(PULSE, 'size-5 shrink-0 rounded')} />
          <div className={cn(PULSE, 'h-3 flex-1 rounded')} />
          <div className={cn(PULSE, 'h-3 w-24 rounded')} />
          <div className={cn(PULSE, 'h-3 w-20 rounded')} />
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
 * D5 — empty state. The `all` filter (genuinely no downloads) gets a distinct message + a
 * 前往探索 affordance (never a bare blank, AC6); other filters get a switch-filter hint.
 */
export function DownloadsEmptyV2({ filter }: { filter: FilterStatus }) {
  const isAll = filter === 'all';
  return (
    <div
      data-testid="downloads-empty-v2"
      className="flex flex-col items-center px-6 py-16 text-center"
    >
      <span className="flex size-18 items-center justify-center rounded-[var(--radius-xl)] bg-[var(--bg-tertiary)]">
        <Inbox className="size-8 text-[var(--text-secondary)]" aria-hidden="true" />
      </span>
      <p className="mt-4 text-xl font-semibold text-[var(--text-primary)]">
        {EMPTY_MESSAGES[filter]}
      </p>
      {/* The one sentence that is true whether or not *arr is set up: anything added to
          qBittorrent — by hand, by Sonarr/Radarr, by a request — shows up here. */}
      <p className="mt-2 text-sm text-[var(--text-secondary)]">
        {isAll ? '在 qBittorrent 新增種子後，下載進度會顯示在這裡。' : '嘗試切換其他篩選條件'}
      </p>
      {isAll && (
        <Button asChild className="mt-4 h-11 px-5 font-semibold">
          <Link to="/discover">前往探索</Link>
        </Button>
      )}
    </div>
  );
}

/**
 * D6 — qBittorrent is set up but the poll errored: per-section fail-soft. The shell + nav still
 * render, the page never hard-fails, and the user gets 重試 + 前往設定 (AC6). `onRetry` re-runs the
 * query. A qBittorrent that was never set up is not this card — see DownloadsQbtNotConfiguredV2.
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
      className="flex flex-col items-center gap-3 rounded-[var(--radius-lg)] border border-[var(--error)] bg-[var(--error-tint)] px-7 py-12 text-center"
    >
      <WifiOff className="size-10 text-[var(--error-text)]" aria-hidden="true" />
      <p className="text-base font-semibold text-[var(--error-text)]">無法連線到 qBittorrent</p>
      <p className="text-sm text-[var(--text-secondary)]">
        {message ?? '請確認下載器已啟動，或檢查連線設定。'}
      </p>
      <div className="mt-1 flex items-center gap-3">
        <Button className="h-11 px-5 font-semibold" onClick={onRetry}>
          重試
        </Button>
        {/* The qBittorrent page, not the settings index — that is the one thing to check. */}
        <Button asChild variant="outline" className="h-11 px-5 font-semibold">
          <Link to="/settings/qbittorrent">前往設定</Link>
        </Button>
      </div>
    </div>
  );
}

/**
 * D11 — qBittorrent has never been set up. Nothing is broken and there is nothing to reconnect to,
 * so no 硃砂 and no 重試 (it could never succeed) — just the one way forward. Before this card the
 * page said「無法連線到 qBittorrent，請確認下載器已啟動」to people who had no downloader to start
 * (disc-2026-09-downloads-not-configured-says-unreachable, Alexyu 2026-09-14).
 */
export function DownloadsQbtNotConfiguredV2() {
  return (
    <div
      data-testid="downloads-qbt-not-configured-v2"
      className="flex flex-col items-center gap-3 rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] px-7 py-12 text-center"
    >
      <Plug className="size-10 text-[var(--text-secondary)]" aria-hidden="true" />
      <p className="text-base font-semibold text-[var(--text-primary)]">還沒有設定 qBittorrent</p>
      <p className="text-sm text-[var(--text-secondary)]">設定好之後，下載進度會顯示在這裡。</p>
      <Button asChild className="mt-1 h-11 px-5 font-semibold">
        <Link to="/settings/qbittorrent">前往設定</Link>
      </Button>
    </div>
  );
}
