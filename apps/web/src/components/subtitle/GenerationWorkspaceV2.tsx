// Design ref: ux-design.pen Screen F11-D-v2 (l8FsB) · F12-D-v2 (iH98f) · F13-D-v2 (F7ohe)
// Source: ux-design.pen (Pencil app)
/**
 * The AI generation WORKSPACE (Story ux3-ai-2, PH3-G1) — an immersive, 活動-hosted
 * full-page surface that WATCHES a running generation batch (or detail-triggered
 * single jobs) with a live event log, so a long AI batch has a home you can leave
 * and return to. The F8/F9 dialog stays the LAUNCHER; this is the WATCHER (design
 * IA ruling, ux3-ai-1). Vocabulary + tokens converge 1:1 with GenerationBatchDialogV2.
 *
 * Capability-honored (ux3-ai-1 amber note c4FIoB): NO pause/resume, cancel is
 * batch-wide only, no per-item retry, movies-only, cost from SSE spent_usd/budget_usd
 * only, budget non-editable, no history. The right pane is an EVENT LOG (not a
 * transcript — SSE carries no transcript content) with NO timestamps (Rule 23-clean).
 * Attach-degraded is drawn honestly: the status probe has no items[]
 * (disc-2026-07-generation-batch-status-items) → counters + in-flight card only.
 */
import { useEffect } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Check,
  CircleAlert,
  CirclePause,
  RotateCcw,
  Radio,
  Sparkles,
  Captions,
} from 'lucide-react';
import { cn } from '../../lib/utils';
import { GenerationProgressV2 } from './GenerationProgressV2';
import {
  deriveRowStates,
  generationBatchPreviewKey,
  generationBatchItemsKey,
} from './GenerationBatchDialogV2';
import { subtitleService, type GenerationBatchItem } from '../../services/subtitleService';
import {
  useGenerationBatchProgress,
  type GenerationBatchProgressState,
} from '../../hooks/useGenerationBatchProgress';
import {
  useGenerationProgress,
  type GenerationProgressState,
} from '../../hooks/useGenerationProgress';
import {
  useGenerationJobsFeed,
  type FeedRow,
  type FeedTone,
  type SingleJobState,
} from '../../hooks/useGenerationJobsFeed';
import { usePageVisibility } from '../../hooks/useDownloads';
import { deriveWorkspaceMode, modeShowsFeed, type WorkspaceMode } from './generationWorkspace';

function usd(v: number): string {
  return `$${v.toFixed(2)}`;
}

const FEED_TONE_CLASS: Record<FeedTone, string> = {
  active: 'text-[var(--accent-text)]',
  done: 'text-[var(--success)]',
  failed: 'text-[var(--error-text)]',
  info: 'text-[var(--text-secondary)]',
};

// --- Presentational pieces ---------------------------------------------------

/** Overall counts + progress bar (role=progressbar) + cost line + SSE chip. */
function OverallStrip({ progress }: { progress: GenerationBatchProgressState }) {
  const { successCount, failCount, totalItems, spentUsd, budgetUsd } = progress;
  const done = successCount + failCount;
  const pct = totalItems > 0 ? Math.round((done / totalItems) * 100) : 0;
  return (
    <div
      data-testid="workspace-overall"
      className="flex flex-wrap items-center gap-x-8 gap-y-3 rounded-[var(--radius-md)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] px-4 py-3"
    >
      <div className="flex flex-col gap-1.5">
        <span className="text-[11px] text-[var(--text-muted)]">整批進度</span>
        <div className="flex items-baseline gap-1">
          <span className="text-sm text-[var(--text-secondary)]">已完成</span>
          <span className="font-mono text-xl font-semibold tabular-nums text-[var(--text-primary)]">
            {done}
          </span>
          <span className="font-mono text-sm tabular-nums text-[var(--text-muted)]">/</span>
          <span className="font-mono text-sm tabular-nums text-[var(--text-secondary)]">
            {totalItems}
          </span>
          <span className="text-xs text-[var(--text-secondary)]">部</span>
        </div>
        <div
          className="h-1.5 w-44 overflow-hidden rounded-[var(--radius-sm)] bg-[var(--bg-tertiary)]"
          role="progressbar"
          aria-label="整批生成進度"
          aria-valuenow={pct}
          aria-valuemin={0}
          aria-valuemax={100}
        >
          <div
            className="h-full rounded-[var(--radius-sm)] bg-[var(--accent-primary)]"
            style={{ width: `${pct}%` }}
          />
        </div>
      </div>
      <div className="flex flex-col gap-1.5">
        <span className="text-[11px] text-[var(--text-muted)]">本次用量</span>
        <div className="flex items-baseline gap-1">
          <span className="font-mono text-base font-semibold tabular-nums text-[var(--text-primary)]">
            {usd(spentUsd)}
          </span>
          <span className="text-xs text-[var(--text-secondary)]">／上限</span>
          <span className="font-mono text-sm tabular-nums text-[var(--text-secondary)]">
            {usd(budgetUsd)}
          </span>
        </div>
      </div>
      <SseChip />
    </div>
  );
}

function SseChip() {
  return (
    <span className="inline-flex items-center gap-1.5 rounded-[var(--radius-sm)] bg-[var(--info-tint)] px-2 py-1 text-[11px] text-[var(--info)]">
      <Radio className="h-3 w-3" aria-hidden="true" />
      即時更新（SSE）
    </span>
  );
}

function QueueRowLabel({ state }: { state: string }) {
  switch (state) {
    case 'done':
      return (
        <span className="flex shrink-0 items-center gap-1.5 text-xs text-[var(--success)]">
          <Check className="h-4 w-4" aria-hidden="true" />
          完成
        </span>
      );
    case 'failed':
      return (
        <span className="flex shrink-0 items-center gap-1.5 text-xs text-[var(--error-text)]">
          <CircleAlert className="h-4 w-4" aria-hidden="true" />
          失敗
        </span>
      );
    case 'active':
      return (
        <span className="shrink-0 text-xs font-semibold text-[var(--accent-text)]">轉錄中</span>
      );
    case 'paused':
      return (
        <span className="flex shrink-0 items-center gap-1.5 text-xs text-[var(--text-muted)]">
          <CirclePause className="h-3.5 w-3.5" aria-hidden="true" />
          已暫停 — 下次繼續
        </span>
      );
    case 'stopped':
      return <span className="shrink-0 text-xs text-[var(--text-muted)]">已取消</span>;
    default:
      return <span className="shrink-0 text-xs text-[var(--text-muted)]">排隊中</span>;
  }
}

function QueueRow({
  mediaId,
  title,
  state,
  activeItemProgress,
}: {
  mediaId: string;
  title: string;
  state: string;
  activeItemProgress?: GenerationProgressState | null;
}) {
  return (
    <li
      data-testid={`workspace-queue-row-${mediaId}`}
      data-state={state}
      className="flex flex-col gap-3 rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] px-4 py-3.5"
    >
      <div className="flex items-center gap-3.5">
        <span
          aria-hidden="true"
          className="h-[54px] w-[38px] shrink-0 rounded-[var(--radius-sm)] bg-[var(--bg-tertiary)]"
        />
        <span className="min-w-0 flex-1 truncate text-sm text-[var(--text-primary)]">{title}</span>
        <QueueRowLabel state={state} />
      </div>
      {state === 'active' && (
        <div className="sm:pl-[52px]">
          <GenerationProgressV2
            phase={
              activeItemProgress && activeItemProgress.phase !== 'failed'
                ? activeItemProgress.phase
                : 'idle'
            }
            percentage={activeItemProgress?.percentage}
            message={activeItemProgress?.message}
          />
        </div>
      )}
    </li>
  );
}

/** Live event log (AC 4) — session-scoped, order-only, no timestamps. */
function EventLogPane({ feed }: { feed: FeedRow[] }) {
  return (
    <aside
      data-testid="workspace-event-log"
      className="flex w-full flex-col overflow-hidden rounded-[var(--radius-md)] border border-[var(--border-subtle)] bg-[var(--bg-primary)] lg:w-[380px]"
    >
      <div className="flex items-center gap-2.5 border-b border-[var(--border-subtle)] px-3.5 py-2.5">
        <span className="text-sm font-semibold text-[var(--text-primary)]">即時活動</span>
        <span className="ml-auto text-xs text-[var(--text-muted)]">自開啟本頁起累積</span>
      </div>
      <ol
        aria-live="polite"
        aria-label="生成事件日誌"
        className="flex-1 space-y-0.5 overflow-y-auto px-1.5 py-1.5"
      >
        {feed.map((row) => (
          <li
            key={row.seq}
            data-testid="workspace-feed-row"
            className="flex items-center gap-2 rounded-[var(--radius-sm)] px-2 py-1.5"
          >
            <span className={cn('text-xs font-semibold', FEED_TONE_CLASS[row.tone])}>
              {row.stage}
            </span>
            {row.message && (
              <span className="min-w-0 flex-1 truncate text-xs text-[var(--text-secondary)]">
                {row.message}
              </span>
            )}
            {row.trail && (
              <span
                className={cn(
                  'ml-auto shrink-0 font-mono text-xs tabular-nums',
                  FEED_TONE_CLASS[row.tone]
                )}
              >
                {row.trail}
              </span>
            )}
          </li>
        ))}
      </ol>
      <div className="flex items-center gap-2 border-t border-[var(--border-subtle)] px-3.5 py-2.5">
        <SseChip />
        <span className="ml-auto text-[11px] text-[var(--text-muted)]">
          僅狀態事件，不含逐字內容
        </span>
      </div>
    </aside>
  );
}

/** F9-verbatim budget-ceiling banner (success semantics, warning tokens). */
function BudgetBanner({ progress }: { progress: GenerationBatchProgressState }) {
  const { successCount, failCount, totalItems, pausedCount, budgetUsd } = progress;
  const done = successCount + failCount;
  const remaining = pausedCount || Math.max(0, totalItems - done);
  return (
    <div
      data-testid="workspace-budget-banner"
      role="status"
      className="flex items-center gap-2.5 rounded-[var(--radius-md)] bg-[var(--warning-tint)] px-4 py-3"
    >
      <CircleAlert
        className="h-[18px] w-[18px] shrink-0 text-[var(--warning)]"
        aria-hidden="true"
      />
      <p className="text-sm text-[var(--text-primary)]">
        已達本次預算上限（
        <span className="font-mono font-semibold tabular-nums text-[var(--warning)]">
          {usd(budgetUsd)}
        </span>
        ）— 已完成{' '}
        <span className="font-mono font-semibold tabular-nums text-[var(--warning)]">{done}</span>{' '}
        部，剩餘{' '}
        <span className="font-mono font-semibold tabular-nums text-[var(--warning)]">
          {remaining}
        </span>{' '}
        部下次繼續
      </p>
    </div>
  );
}

const TERMINAL_COPY: Record<string, { label: string; tone: string }> = {
  complete: { label: '全部完成', tone: 'text-[var(--success)]' },
  cancelled: { label: '已取消', tone: 'text-[var(--text-muted)]' },
  error: { label: '批次發生錯誤', tone: 'text-[var(--error-text)]' },
};

// --- Presentational workspace ------------------------------------------------

export interface GenerationWorkspaceV2Props {
  mode: WorkspaceMode;
  progress: GenerationBatchProgressState;
  /** Batch queue rows (launcher-cached start items[]); empty in attach mode. */
  items: GenerationBatchItem[];
  failedIds?: ReadonlySet<string>;
  activeItemProgress?: GenerationProgressState | null;
  /** In-flight detail-triggered single jobs (single mode). */
  singleJobs?: Record<string, SingleJobState>;
  feed: FeedRow[];
  /** 缺字幕 preview count for the idle launcher. */
  previewCount?: number;
  /** Data-source failure (activity/preview down) → fail-soft banner. */
  dataError?: boolean;
  onLaunch: () => void;
  onConfirmCancelAll: () => void;
  onResume: () => void;
  onRetryData: () => void;
}

const EMPTY_FAILED: ReadonlySet<string> = new Set();

export function GenerationWorkspaceV2({
  mode,
  progress,
  items,
  failedIds = EMPTY_FAILED,
  activeItemProgress = null,
  singleJobs = {},
  feed,
  previewCount,
  dataError = false,
  onLaunch,
  onConfirmCancelAll,
  onResume,
  onRetryData,
}: GenerationWorkspaceV2Props) {
  const rowStates = deriveRowStates(items, progress, failedIds);
  const showFeed = modeShowsFeed(mode);
  const isTerminal = mode === 'complete' || mode === 'cancelled' || mode === 'error';
  const singleList = Object.values(singleJobs);

  return (
    <div data-testid="generation-workspace" data-mode={mode} className="flex min-h-full flex-col">
      {/* Header */}
      <div className="flex flex-col gap-3 px-6 pb-3 pt-4 sm:px-8">
        <nav
          aria-label="麵包屑"
          className="flex items-center gap-1.5 text-[13px] text-[var(--text-secondary)]"
        >
          <span>活動</span>
          <span className="text-[var(--text-muted)]" aria-hidden="true">
            ／
          </span>
          <span className="text-[var(--text-primary)]">生成字幕</span>
        </nav>
        <h1 className="flex items-center gap-2 text-xl font-bold text-[var(--text-primary)]">
          <Captions className="h-5 w-5 text-[var(--accent-text)]" aria-hidden="true" />
          {mode === 'single' ? '生成字幕' : '批次生成字幕'}
        </h1>
        {mode === 'budget_ceiling' && <BudgetBanner progress={progress} />}
        {!['idle', 'loading'].includes(mode) && <OverallStrip progress={progress} />}
      </div>

      {/* Body */}
      <div className="flex min-h-0 flex-1 flex-col gap-4 px-6 pb-6 sm:px-8 lg:flex-row">
        <div className="flex min-h-0 flex-1 flex-col gap-3">
          {dataError && (
            <div
              data-testid="workspace-data-error"
              role="alert"
              className="flex items-center gap-2.5 rounded-[var(--radius-md)] bg-[var(--error-tint)] px-4 py-3.5"
            >
              <CircleAlert
                className="h-5 w-5 shrink-0 text-[var(--error-text)]"
                aria-hidden="true"
              />
              <span className="text-sm text-[var(--error-text)]">無法載入生成狀態</span>
              <button
                type="button"
                data-testid="workspace-data-retry"
                onClick={onRetryData}
                className="ml-auto flex h-11 items-center gap-1.5 rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-3 text-sm text-[var(--text-primary)]"
              >
                <RotateCcw className="h-4 w-4" aria-hidden="true" />
                重試
              </button>
            </div>
          )}

          {mode === 'idle' && (
            <div
              data-testid="workspace-idle"
              className="flex flex-col items-center justify-center gap-3 rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] px-6 py-14 text-center"
            >
              <Sparkles className="h-10 w-10 text-[var(--text-muted)]" aria-hidden="true" />
              <p className="text-sm text-[var(--text-secondary)]">目前沒有進行中的生成</p>
              {previewCount !== undefined && (
                <p className="flex items-baseline gap-1 text-sm text-[var(--text-secondary)]">
                  缺繁中字幕：
                  <span className="font-mono text-lg font-semibold tabular-nums text-[var(--text-primary)]">
                    {previewCount}
                  </span>
                  部
                </p>
              )}
              <button
                type="button"
                data-testid="workspace-launch"
                onClick={onLaunch}
                className="mt-1 flex h-11 items-center rounded-[var(--radius-md)] bg-[var(--accent-primary)] px-4 text-sm font-medium text-[var(--text-on-accent)]"
              >
                批次生成字幕
              </button>
            </div>
          )}

          {mode === 'loading' && (
            <div data-testid="workspace-skeleton" aria-busy="true" className="space-y-3">
              {Array.from({ length: 4 }).map((_, i) => (
                <div
                  key={i}
                  className="h-[72px] animate-pulse rounded-[var(--radius-lg)] bg-[var(--bg-secondary)] motion-reduce:animate-none"
                />
              ))}
            </div>
          )}

          {mode === 'attach' && (
            <div data-testid="workspace-attach" className="flex flex-col gap-3">
              {progress.currentItem && (
                <QueueRow
                  mediaId={progress.currentMediaId ?? 'current'}
                  title={progress.currentItem}
                  state="active"
                  activeItemProgress={activeItemProgress}
                />
              )}
              {[0, 1].map((i) => (
                <div
                  key={i}
                  aria-hidden="true"
                  className="h-14 animate-pulse rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] opacity-50 motion-reduce:animate-none"
                />
              ))}
              <p className="rounded-[var(--radius-md)] border border-[var(--border-subtle)] px-3 py-2 text-xs text-[var(--text-muted)]">
                佇列明細自本頁開啟起顯示（附加至進行中的批次時，僅重建進行中項目）
              </p>
            </div>
          )}

          {(mode === 'running' || mode === 'budget_ceiling' || isTerminal) && items.length > 0 && (
            <div className="flex flex-col gap-2.5">
              <div className="flex items-center justify-between">
                <h2 className="text-[15px] font-semibold text-[var(--text-primary)]">生成佇列</h2>
                {mode === 'running' && <CancelAll onConfirm={onConfirmCancelAll} />}
                {mode === 'budget_ceiling' && (
                  <button
                    type="button"
                    data-testid="workspace-resume"
                    onClick={onResume}
                    className="flex h-11 items-center rounded-[var(--radius-md)] bg-[var(--accent-primary)] px-4 text-sm font-medium text-[var(--text-on-accent)]"
                  >
                    下次繼續
                  </button>
                )}
                {isTerminal && (
                  <span
                    data-testid="workspace-terminal-label"
                    className={cn('text-sm font-semibold', TERMINAL_COPY[mode].tone)}
                  >
                    {TERMINAL_COPY[mode].label}
                  </span>
                )}
              </div>
              <ul className="space-y-2.5">
                {items.map((it, i) => (
                  <QueueRow
                    key={it.mediaId}
                    mediaId={it.mediaId}
                    title={it.title}
                    state={rowStates[i]}
                    activeItemProgress={rowStates[i] === 'active' ? activeItemProgress : null}
                  />
                ))}
              </ul>
            </div>
          )}

          {mode === 'single' && (
            <div data-testid="workspace-single" className="flex flex-col gap-2.5">
              <h2 className="text-[15px] font-semibold text-[var(--text-primary)]">進行中任務</h2>
              <ul className="space-y-2.5">
                {singleList.map((job) => (
                  <QueueRow
                    key={job.mediaId}
                    mediaId={job.mediaId}
                    title={job.message || job.mediaId}
                    state="active"
                    activeItemProgress={{
                      phase: job.phase,
                      failedPhase: null,
                      percentage: job.percentage,
                      message: job.message,
                      jobId: null,
                      error: null,
                      srtPath: null,
                      zhSrtPath: null,
                    }}
                  />
                ))}
              </ul>
            </div>
          )}
        </div>

        {showFeed && <EventLogPane feed={feed} />}
      </div>
    </div>
  );
}

function CancelAll({ onConfirm }: { onConfirm: () => void }) {
  return (
    <button
      type="button"
      data-testid="workspace-cancel-all"
      onClick={onConfirm}
      className="flex h-11 items-center rounded-[var(--radius-md)] border border-[var(--border-subtle)] px-4 text-sm text-[var(--text-secondary)] transition-colors hover:text-[var(--text-primary)]"
    >
      全部取消
    </button>
  );
}

// --- Container ---------------------------------------------------------------

export interface GenerationWorkspaceProps {
  /** True when this view is the active + visible surface (drives lazy SSE, §8). */
  active: boolean;
  /** Opens the F8 batch dialog (the launcher — the workspace never rebuilds scope). */
  onLaunch: () => void;
}

export function GenerationWorkspace({ active, onLaunch }: GenerationWorkspaceProps) {
  const queryClient = useQueryClient();
  const isVisible = usePageVisibility();
  const live = active && isVisible;

  const batch = useGenerationBatchProgress();
  const jobs = useGenerationJobsFeed();
  const activeItem = useGenerationProgress();

  // 缺字幕 preview count for the idle launcher (shared cache with the dialog).
  const previewQuery = useQuery({
    queryKey: generationBatchPreviewKey,
    queryFn: () => subtitleService.previewGenerationBatch(),
    enabled: live,
    staleTime: 30_000,
    retry: 1,
  });

  // Status probe on activation → attach to a running batch (seed) + open the feed.
  const statusQuery = useQuery({
    queryKey: ['subtitles', 'generation-batch', 'status'],
    queryFn: () => subtitleService.getGenerationBatchStatus(),
    enabled: live,
    retry: 1,
  });

  // Launcher-cached start items[] (present only when a batch was started this session).
  // REACTIVE subscription — a plain getQueryData read would not re-render when the
  // dialog caches items[] while the workspace is already mounted (idle → launch →
  // start, the dialog opening over the workspace). enabled:false: the dialog owns the
  // writes (setQueryData); this query only subscribes to that key's cache updates.
  const { data: items = [] } = useQuery<GenerationBatchItem[]>({
    queryKey: generationBatchItemsKey,
    queryFn: () => [],
    enabled: false,
    staleTime: Infinity,
  });

  useEffect(() => {
    if (!live) {
      batch.reset();
      jobs.stop();
      activeItem.reset();
      return;
    }
    jobs.startTracking();
    const probe = statusQuery.data;
    if (probe?.running && probe.progress) {
      batch.startTracking(probe.progress);
      if (probe.progress.currentMediaId) activeItem.startTracking(probe.progress.currentMediaId);
    }
    // Re-run when activation or the probe result changes.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [live, statusQuery.data]);

  const mode = deriveWorkspaceMode({
    probing: live && statusQuery.isLoading,
    batchStatus: batch.status,
    hasItems: items.length > 0,
    singleJobCount: Object.keys(jobs.singleJobs).length,
  });

  const dataError = live && !statusQuery.isLoading && (statusQuery.isError || previewQuery.isError);

  return (
    <GenerationWorkspaceV2
      mode={mode}
      progress={batch.progress}
      items={items}
      activeItemProgress={activeItem.progress}
      singleJobs={jobs.singleJobs}
      feed={jobs.feed}
      previewCount={previewQuery.data?.totalItems}
      dataError={dataError}
      onLaunch={onLaunch}
      onConfirmCancelAll={() => {
        void subtitleService.cancelGenerationBatch();
      }}
      onResume={() => {
        void subtitleService
          .startGenerationBatch({ scope: 'missing' })
          .then(() => queryClient.invalidateQueries({ queryKey: generationBatchPreviewKey }));
      }}
      onRetryData={() => {
        void statusQuery.refetch();
        void previewQuery.refetch();
      }}
    />
  );
}
