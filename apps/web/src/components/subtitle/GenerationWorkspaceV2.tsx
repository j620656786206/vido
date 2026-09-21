// Implements: Component/GenQueueRow-v2 (aw4Qr)
// Design ref: ux-design.pen Screen F11-D-v2 (l8FsB) · F12-D-v2 (iH98f) · F13-D-v2 (F7ohe) · F11-M-v2 (PXB0z)
// Source: ux-design.pen (Pencil app)
/**
 * The AI generation WORKSPACE (Story ux3-ai-2, PH3-G1) — an immersive, 活動-hosted
 * full-page surface that WATCHES a running generation batch (or detail-triggered
 * single jobs) with a live event log, so a long AI batch has a home you can leave
 * and return to. The F8/F9 dialog stays the LAUNCHER; this is the WATCHER (design
 * IA ruling, ux3-ai-1). Vocabulary + tokens converge 1:1 with GenerationBatchDialogV2.
 *
 * Capability-honored (ux3-ai-1 amber note c4FIoB): NO pause/resume, cancel is
 * batch-wide only, no per-item retry, cost from SSE spent_usd/budget_usd only,
 * budget non-editable. The right pane is an EVENT LOG (not a transcript — SSE
 * carries no transcript content) with NO timestamps (Rule 23-clean).
 *
 * dsr-6d-c-2: the right pane records what happened (useGenerationJobsFeed) and
 * draws it through generationEventCopy — one row per film per stage, film names,
 * Chinese failure reasons, and a single live region that reads results only.
 *
 * dsr-6d-c-1: the queue is the BACKEND's (`progress.items[]` — dsr-6d-a AC #1),
 * never a guess. Row words come from the ONE shared vocabulary in
 * generationQueueRow.ts, so this page and the dialog can never say opposite things
 * about the same title. After a batch ends the queue comes from the status probe's
 * `last` (the dialog removes its own items[] cache on terminal), and 關閉 is the
 * only caller of dismiss — i.e. "I have seen this result, forget it".
 *
 * dsr-6f-4: the PHONE page (F11-M). This is a page, not a sheet — no grabber, no ✕;
 * the way out is a back button. Everything phone-only is `max-sm:` / `sm:hidden`, so
 * ≥640 is untouched. The overall strip re-flows IN PLACE (`max-sm:contents` + order —
 * never drawn twice), and the live log starts collapsed: the LIST is hidden with
 * `display:none`, never unmounted, because the page's only aria-live region lives in
 * that pane and a region mounted together with its text is never read.
 */
import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useId,
  useRef,
  useState,
  type ComponentRef,
} from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Activity,
  Check,
  CircleAlert,
  ChevronDown,
  ChevronLeft,
  CircleDashed,
  CirclePause,
  Hourglass,
  LoaderCircle,
  Pause,
  Radio,
  Sparkles,
  TriangleAlert,
} from 'lucide-react';
import { cn } from '../../lib/utils';
import { GenerationProgressV2 } from './GenerationProgressV2';
import { generationBatchPreviewKey, generationBatchStatusKey } from './GenerationBatchDialogV2';
import {
  queueRowLabel,
  queueRowTitle,
  queueRowBadge,
  queueRowSubStatus,
} from './generationQueueRow';
import {
  subtitleService,
  type GenerationBatchItemState,
  type GenerationBatchItemStatus,
} from '../../services/subtitleService';
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
  type SingleJobState,
} from '../../hooks/useGenerationJobsFeed';
import { feedRowView, type FeedGlyph } from './generationEventCopy';
import { usePageVisibility } from '../../hooks/useDownloads';
import { libraryKeys } from '../../hooks/useLibrary';
import { detailKeys } from '../../hooks/useMediaDetails';
import { activityKeys } from '../../hooks/useActivity';
import { transcriptionEstimateKeys } from '../../hooks/useTranscriptionEstimate';
import { deriveWorkspaceMode, modeShowsFeed, type WorkspaceMode } from './generationWorkspace';
import { usd } from '../../lib/currency';

// --- Presentational pieces ---------------------------------------------------

/** Overall counts + progress bar (role=progressbar) + cost line + SSE chip. */
function OverallStrip({
  progress,
  live,
}: {
  progress: GenerationBatchProgressState;
  /**
   * The batch is in a state where the stream is SUPPOSED to be open. It cannot
   * prove the socket is up (the hook exposes no connected flag and retries on a
   * 10s backoff) — it only guarantees the chip is gone once the batch is over.
   */
  live: boolean;
}) {
  const { successCount, failCount, totalItems, spentUsd, budgetUsd } = progress;
  const done = successCount + failCount;
  const pct = totalItems > 0 ? Math.round((done / totalItems) * 100) : 0;
  return (
    <div
      data-testid="workspace-overall"
      // Phone (F11-M `QNVCS`): three rows — count + chip / full-width bar / cost on
      // one line. Re-flowed in place: the first column dissolves (`contents`) so its
      // children can be ordered around the chip. Never a second copy of the strip.
      className="flex flex-wrap items-center gap-x-8 gap-y-3 rounded-[var(--radius-md)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] px-4 py-3 max-sm:gap-x-2 max-sm:gap-y-2 max-sm:px-3.5"
    >
      <div className="flex flex-col gap-1.5 max-sm:contents">
        <span className="text-[11px] text-[var(--text-muted)] max-sm:hidden">整批進度</span>
        <div className="flex items-baseline gap-1 max-sm:order-1">
          <span className="text-sm text-[var(--text-secondary)]">已完成</span>
          <span className="font-mono text-xl font-semibold tabular-nums text-[var(--text-primary)] max-sm:text-lg">
            {done}
          </span>
          <span className="font-mono text-base tabular-nums text-[var(--text-muted)] max-sm:text-sm">
            /
          </span>
          <span className="font-mono text-base tabular-nums text-[var(--text-secondary)] max-sm:text-sm">
            {totalItems}
          </span>
          <span className="text-xs text-[var(--text-secondary)] max-sm:text-sm">部</span>
        </div>
        <div
          className="h-1.5 w-[180px] overflow-hidden rounded-[var(--radius-sm)] bg-[var(--bg-tertiary)] max-sm:order-3 max-sm:w-full"
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
      <div className="flex flex-col gap-1.5 max-sm:order-4 max-sm:w-full max-sm:flex-row max-sm:items-baseline max-sm:gap-1">
        <span className="text-[11px] text-[var(--text-muted)]">本次用量</span>
        <div className="flex items-baseline gap-1">
          <span className="font-mono text-base font-semibold tabular-nums text-[var(--text-primary)] max-sm:text-sm">
            {usd(spentUsd)}
          </span>
          <span className="text-xs text-[var(--text-secondary)]">／上限</span>
          <span className="font-mono text-base tabular-nums text-[var(--text-secondary)] max-sm:text-sm">
            {usd(budgetUsd)}
          </span>
        </div>
      </div>
      {/* `ml-auto` MUST stay max-sm-only: on desktop the chip is the row's last
          item and an unprefixed one would throw it to the far right. No wrapper
          either — a blockified box around it can change the desktop row height. */}
      {live && <SseChip className="max-sm:order-2 max-sm:ml-auto" />}
    </div>
  );
}

function SseChip({ className }: { className?: string }) {
  return (
    <span
      data-testid="workspace-sse-chip"
      className={cn(
        'inline-flex items-center gap-1.5 rounded-[var(--radius-sm)] bg-[var(--info-tint)] px-2 py-1.5 text-[11px] text-[var(--info-text)]',
        className
      )}
    >
      <Radio className="h-3 w-3" aria-hidden="true" />
      即時更新（SSE）
    </span>
  );
}

/**
 * Status pill beside the page title (F11 `X22E1A` / F12 `FKDCf`). Ochre is the
 * BATCH-level statement "you asked and it did not finish" (⚖️ Alexyu Q1); the
 * rows and every number stay neutral.
 */
function StatusPill({ mode }: { mode: WorkspaceMode }) {
  if (mode === 'running') {
    return (
      <span
        data-testid="workspace-status-pill"
        className="inline-flex items-center gap-1.5 rounded-full bg-[var(--accent-tint)] px-2.5 py-1 text-xs font-medium text-[var(--accent-text)]"
      >
        <span
          aria-hidden="true"
          className="h-2 w-2 rounded-full bg-[var(--accent-text)] max-sm:h-1.5 max-sm:w-1.5"
        />
        進行中
      </span>
    );
  }
  if (mode === 'budget_ceiling') {
    return (
      <span
        data-testid="workspace-status-pill"
        className="inline-flex items-center gap-1.5 rounded-full bg-[var(--warning-tint)] px-2.5 py-1 text-xs font-medium text-[var(--warning-text)]"
      >
        <span
          aria-hidden="true"
          className="h-2 w-2 rounded-full bg-[var(--warning-text)] max-sm:h-1.5 max-sm:w-1.5"
        />
        已達上限
      </span>
    );
  }
  return null;
}

/**
 * Badge tint per backend status (`aw4Qr` → `IcPTo`). Paused/cancelled/queued are
 * NEUTRAL: an item nobody got to is not a warning, it is simply not done yet.
 */
const BADGE_TINT: Record<GenerationBatchItemStatus, string> = {
  done: 'bg-[var(--success-tint)]',
  failed: 'bg-[var(--error-tint)]',
  running: 'bg-[var(--accent-tint)]',
  paused: 'bg-[var(--bg-tertiary)]',
  cancelled: 'bg-[var(--bg-tertiary)]',
  queued: 'bg-[var(--bg-tertiary)]',
};

/**
 * Glyph per status (`aw4Qr` → `CQhw2`, 14×14). The WORDS come from the shared
 * generationQueueRow vocabulary; the glyph is this surface's own design (F8/F9
 * draw a different set for the same states, so it is a rendering choice — not
 * something to force into the shared module).
 */
function BadgeIcon({ status }: { status: GenerationBatchItemStatus }) {
  const cls = 'h-3.5 w-3.5 shrink-0';
  switch (status) {
    case 'done':
      return <Check className={cls} aria-hidden="true" />;
    case 'failed':
      return <TriangleAlert className={cls} aria-hidden="true" />;
    case 'running':
      return (
        <LoaderCircle
          className={cn(cls, 'animate-spin motion-reduce:animate-none')}
          aria-hidden="true"
        />
      );
    case 'queued':
      return <Hourglass className={cls} aria-hidden="true" />;
    case 'paused':
      return <Pause className={cls} aria-hidden="true" />;
    default:
      return null;
  }
}

function QueueRow({
  item,
  activeItemProgress,
}: {
  item: GenerationBatchItemState;
  activeItemProgress?: GenerationProgressState | null;
}) {
  const label = queueRowLabel(
    { status: item.status, reason: item.reason },
    activeItemProgress?.phase
  );
  const isRunning = item.status === 'running';
  // The percentage only exists in the translating stage (dsr-6d-a): claiming one
  // during extraction/transcription would be inventing a number.
  const pct =
    isRunning &&
    activeItemProgress?.phase === 'translating' &&
    activeItemProgress.percentage !== null &&
    activeItemProgress.percentage !== undefined
      ? `${Math.round(activeItemProgress.percentage)}%`
      : null;

  return (
    <li
      data-testid={`workspace-queue-row-${item.mediaId}`}
      data-state={item.status}
      className="flex flex-col gap-2.5 rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] px-4 py-3.5"
    >
      <div className="flex items-center gap-3.5">
        <span
          aria-hidden="true"
          className="h-[60px] w-10 shrink-0 rounded-[var(--radius-sm)] bg-[var(--bg-tertiary)]"
        />
        <div className="flex min-w-0 flex-1 flex-col gap-1">
          <span className="truncate text-base font-semibold text-[var(--text-primary)]">
            {queueRowTitle(item)}
          </span>
          <span
            className={cn(
              'truncate text-sm max-sm:whitespace-normal',
              isRunning ? label.className : 'text-[var(--text-secondary)]'
            )}
          >
            {queueRowSubStatus(item, label.text)}
          </span>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          {pct && (
            <span className="font-mono text-sm tabular-nums text-[var(--accent-text)]">{pct}</span>
          )}
          <span
            className={cn(
              'inline-flex items-center gap-1.5 rounded-full px-2.5 py-1.5 text-xs',
              BADGE_TINT[item.status],
              label.className
            )}
          >
            <BadgeIcon status={item.status} />
            {queueRowBadge(item, label.text)}
          </span>
        </div>
      </div>
      {isRunning && (
        <div className="sm:pl-[54px]">
          <GenerationProgressV2
            align="start"
            // A failed per-item event must render the FAILED stepper, not rewind
            // to 提取音訊 as if extraction had restarted.
            phase={activeItemProgress?.phase ?? 'idle'}
            failedPhase={activeItemProgress?.failedPhase ?? null}
            percentage={activeItemProgress?.percentage}
            message={activeItemProgress?.message}
            error={activeItemProgress?.error}
          />
        </div>
      )}
    </li>
  );
}

const FEED_GLYPH: Record<FeedGlyph, typeof Check> = {
  loader: LoaderCircle,
  check: Check,
  'triangle-alert': TriangleAlert,
  'circle-alert': CircleAlert,
  'circle-pause': CirclePause,
  'circle-dashed': CircleDashed,
};

/** One log row (F11 `evt-N`): glyph 16 · stage · `·` · film … trailing value. */
function FeedRowItem({ row }: { row: FeedRow }) {
  const v = feedRowView(row);
  const Glyph = FEED_GLYPH[v.glyph];
  return (
    <li data-testid="workspace-feed-row" className="flex items-center gap-2 px-3.5 py-2">
      <Glyph
        className={cn(
          'h-4 w-4 shrink-0',
          v.glyphClass,
          v.spin && 'animate-spin motion-reduce:animate-none'
        )}
        aria-hidden="true"
      />
      <span className={cn('shrink-0 text-sm font-semibold', v.stageClass)}>{v.stage}</span>
      {v.srState && <span className="sr-only">{v.srState}</span>}
      {v.parts.map((part, i) => (
        <span
          key={i}
          className={cn(
            'flex items-center gap-2 text-sm text-[var(--text-secondary)]',
            // The film name may truncate; the reason after it stays whole.
            i === 0 ? 'min-w-0 shrink' : 'shrink-0'
          )}
        >
          <span aria-hidden="true" className="text-[var(--text-muted)]">
            ·
          </span>
          <span className={cn(i === 0 && 'truncate')}>{part}</span>
        </span>
      ))}
      {v.trail && (
        <span className={cn('ml-auto shrink-0 font-mono text-sm tabular-nums', v.trailClass)}>
          {v.trail}
        </span>
      )}
    </li>
  );
}

/**
 * Live event log (F11 `DUvwI` / F12 `LTW74`) — session-scoped, order-only, no
 * timestamps. The list is NOT a live region (it used to read every translation
 * percentage aloud); one always-mounted sr-only region reads result and batch
 * rows only. It lives here, outside anything that remounts on a mode change,
 * because a region mounted together with its text is never read (dsr-6d-b M4).
 */
function EventLogPane({
  feed,
  connected,
  budgetStopped,
}: {
  feed: FeedRow[];
  /** The jobs stream is open — NOT "the batch is running" (it outlives the batch). */
  connected: boolean;
  budgetStopped: boolean;
}) {
  const listRef = useRef<ComponentRef<'ol'> | null>(null);
  /** The reader was at the bottom before the last change — keep following. */
  const pinnedRef = useRef(true);
  /**
   * Phone only (F11-M `rMD73`): the log starts collapsed. This state acts through
   * `max-sm:` classes alone, so ≥640 never sees it — no matchMedia needed.
   */
  const [open, setOpen] = useState(false);
  const listId = useId();

  const announcement = (() => {
    for (let i = feed.length - 1; i >= 0; i -= 1) {
      const text = feedRowView(feed[i]).announce;
      if (text) return { seq: feed[i].seq, text };
    }
    return null;
  })();

  // Newest is at the bottom: follow it, unless the reader scrolled up to read.
  const lastSeq = feed.length > 0 ? feed[feed.length - 1].seq : 0;
  useLayoutEffect(() => {
    const el = listRef.current;
    if (el && pinnedRef.current) el.scrollTop = el.scrollHeight;
    // `open`: while collapsed the list is display:none (scrollHeight 0) — the
    // moment it opens it must land on the newest row.
  }, [lastSeq, open]);

  return (
    <aside
      data-testid="workspace-event-log"
      // Capped to the viewport (below the 56px sticky app header) so the LIST
      // scrolls and can follow the newest row; unbounded, the pane grew with its
      // rows and the whole page scrolled instead (CR H3).
      // Phone: no outer frame — the header is its own card and the open list
      // carries its own border (F11-M `AfUUS`).
      className="flex w-full flex-col overflow-hidden rounded-[var(--radius-md)] border border-[var(--border-subtle)] bg-[var(--bg-primary)] max-sm:gap-2 max-sm:overflow-visible max-sm:rounded-none max-sm:border-0 max-sm:bg-transparent lg:sticky lg:top-[4.5rem] lg:max-h-[calc(100vh-6rem)] lg:w-[400px]"
    >
      {/* `relative` lives HERE, not on the button: the toggle's ::after stretches
          over the whole header so the entire bar is the hit area. */}
      <div className="relative flex h-11 shrink-0 items-center gap-2.5 border-b border-[var(--border-subtle)] px-3.5 max-sm:h-auto max-sm:min-h-[47px] max-sm:gap-1.5 max-sm:rounded-[var(--radius-md)] max-sm:border-b-0 max-sm:bg-[var(--bg-secondary)]">
        <Activity
          data-testid="workspace-log-header-icon"
          className="h-3.5 w-3.5 shrink-0 text-[var(--text-secondary)] max-sm:h-4 max-sm:w-4"
          aria-hidden="true"
        />
        <span className="text-sm font-semibold text-[var(--text-primary)]">即時活動</span>
        <span className="ml-auto text-xs text-[var(--text-muted)] max-sm:ml-0">
          <span aria-hidden="true" className="sm:hidden">
            ·{' '}
          </span>
          自開啟本頁起累積
        </span>
        <button
          type="button"
          data-testid="workspace-log-toggle"
          // A fixed name + aria-expanded: a name that flips to 收合… would be read
          // as "收合即時活動, expanded" (CollapsibleSection precedent). It is the
          // header's own visible words, so voice control can say what it sees.
          aria-label="即時活動"
          aria-expanded={open}
          aria-controls={listId}
          onClick={() => {
            // Re-pin on open: the reader may have scrolled up before collapsing,
            // and display:none can zero scrollTop without a scroll event.
            if (!open) pinnedRef.current = true;
            setOpen((v) => !v);
          }}
          className="-mr-3 ml-auto flex h-11 w-11 shrink-0 items-center justify-center rounded-[var(--radius-md)] text-[var(--text-muted)] after:absolute after:inset-0 after:content-[''] sm:hidden"
        >
          <ChevronDown
            className={cn(
              'h-5 w-5 transition-transform duration-[var(--motion-state)] motion-reduce:transition-none',
              open && 'rotate-180'
            )}
            aria-hidden="true"
          />
        </button>
      </div>
      <ol
        ref={listRef}
        id={listId}
        aria-label="生成事件日誌"
        onScroll={(e) => {
          const el = e.currentTarget;
          // Within one row (≈39px) of the bottom counts as "at the bottom".
          pinnedRef.current = el.scrollHeight - el.scrollTop - el.clientHeight <= 40;
        }}
        // Collapsed = hidden, NEVER unmounted (the announcer below must survive).
        // Open on a phone: capped, so the LIST scrolls and follows (CR H3 parity).
        className={cn(
          // `empty:hidden`: opened before any event arrived, the phone would draw an
          // empty bordered box (the list holds nothing but <li>s, so :empty is exact).
          'min-h-0 flex-1 overflow-y-auto py-1.5 max-sm:max-h-80 max-sm:flex-none max-sm:rounded-[var(--radius-md)] max-sm:border max-sm:border-[var(--border-subtle)] max-sm:empty:hidden',
          !open && 'max-sm:hidden'
        )}
      >
        {feed.map((row) => (
          <FeedRowItem key={row.seq} row={row} />
        ))}
      </ol>
      <p data-testid="workspace-log-announcer" aria-live="polite" className="sr-only">
        {/* Keyed by row: a second 批次完成 after the first is a NEW node, so it is
            read again — replacing identical text would be silent (CR M7). */}
        {announcement && <span key={announcement.seq}>{announcement.text}</span>}
      </p>
      <div className="flex flex-col gap-2 border-t border-[var(--border-subtle)] px-3.5 py-2.5 max-sm:border-t-0 max-sm:px-0 max-sm:py-0">
        {/* `hidden` ATTRIBUTE, not a class: jsdom loads no stylesheet. */}
        <p
          data-testid="workspace-log-hint"
          hidden={open}
          className="text-xs text-[var(--text-muted)] sm:hidden"
        >
          展開查看即時事件（不含逐字內容、無時間戳）
        </p>
        <div
          data-testid="workspace-log-footer-row"
          className={cn(
            'flex items-center gap-2',
            // Collapsed with no stream: the row would be empty yet keep its gap.
            !open && !connected && 'max-sm:hidden'
          )}
        >
          {connected && <SseChip />}
          <span
            data-testid="workspace-log-note"
            className={cn(
              'ml-auto text-[11px] text-[var(--text-muted)]',
              // The hint above already says it while collapsed.
              !open && 'max-sm:hidden'
            )}
          >
            僅狀態事件，不含逐字內容
          </span>
        </div>
        {budgetStopped && (
          <div className="flex items-center justify-end gap-1.5 max-sm:justify-start">
            <CirclePause
              className="h-[13px] w-[13px] shrink-0 text-[var(--warning-text)]"
              aria-hidden="true"
            />
            <span className="text-xs font-medium text-[var(--warning-text)]">
              已停止（達預算上限）
            </span>
          </div>
        )}
      </div>
    </aside>
  );
}

/** F9-verbatim budget-ceiling banner. Ochre is the batch-level claim; numbers stay neutral. */
function BudgetBanner({ progress }: { progress: GenerationBatchProgressState }) {
  const { successCount, failCount, pausedCount, budgetUsd } = progress;
  const done = successCount + failCount;
  return (
    <div
      data-testid="workspace-budget-banner"
      role="status"
      className="flex items-center gap-2.5 rounded-[var(--radius-md)] bg-[var(--warning-tint)] px-4 py-3"
    >
      <CircleAlert
        className="h-[18px] w-[18px] shrink-0 text-[var(--warning-text)]"
        aria-hidden="true"
      />
      <p className="text-sm text-[var(--text-primary)]">
        已達本次預算上限（
        {/* Money is a FACT, not a state (DESIGN.md:308) — never a status colour. */}
        <span className="font-mono font-semibold tabular-nums text-[var(--text-primary)]">
          {usd(budgetUsd)}
        </span>
        ）— 已完成{' '}
        <span className="font-mono font-semibold tabular-nums text-[var(--text-primary)]">
          {done}
        </span>{' '}
        部，剩餘{' '}
        <span className="font-mono font-semibold tabular-nums text-[var(--text-primary)]">
          {pausedCount}
        </span>{' '}
        部下次繼續
      </p>
    </div>
  );
}

/**
 * The batch's one-line verdict. Green is "there is an answer and it is the good
 * one" — a run with failures does not qualify, so it stays neutral (same rule and
 * same words as GenerationBatchDialogV2).
 */
function terminalVerdict(
  mode: WorkspaceMode,
  progress: GenerationBatchProgressState
): { label: string; tone: string } | null {
  if (mode === 'complete') {
    return progress.failCount > 0
      ? {
          label: `完成 ${progress.successCount} 部、失敗 ${progress.failCount} 部`,
          tone: 'text-[var(--text-secondary)]',
        }
      : {
          label: `全部完成（${progress.successCount} 部）`,
          tone: 'text-[var(--success-text)]',
        };
  }
  if (mode === 'cancelled') {
    return {
      label: `已取消：完成 ${progress.successCount} 部`,
      tone: 'text-[var(--text-secondary)]',
    };
  }
  if (mode === 'error') {
    return { label: '批次發生錯誤', tone: 'text-[var(--error-text)]' };
  }
  return null;
}

// --- Presentational workspace ------------------------------------------------

export interface GenerationWorkspaceV2Props {
  mode: WorkspaceMode;
  progress: GenerationBatchProgressState;
  activeItemProgress?: GenerationProgressState | null;
  /** In-flight detail-triggered single jobs (single mode). */
  singleJobs?: Record<string, SingleJobState>;
  feed: FeedRow[];
  /**
   * The live log's own stream is open (useGenerationJobsFeed `connected`).
   * Defaults to false: not knowing is not a reason to claim 即時更新.
   */
  feedConnected?: boolean;
  /** 缺字幕 preview count for the idle launcher. */
  previewCount?: number;
  /** Data-source failure (activity/preview down) → fail-soft banner. */
  dataError?: boolean;
  onLaunch: () => void;
  /** Resolves when the cancel succeeded; REJECTS when it failed (the panel says so). */
  onConfirmCancelAll: () => Promise<void>;
  onResume: () => void;
  onRetryData: () => void;
  /** F12 關閉 — forget the remembered terminal result (dismiss). */
  onDismiss?: () => Promise<void>;
  /**
   * Phone back button (F11-M `d86Ux`) — leave the workspace for 活動. A prop, not a
   * router <Link>: this component renders bare in its spec and in the gallery.
   * Without it no back button is drawn.
   */
  onBack?: () => void;
}

export function GenerationWorkspaceV2({
  mode,
  progress,
  activeItemProgress = null,
  singleJobs = {},
  feed,
  feedConnected = false,
  previewCount,
  dataError = false,
  onLaunch,
  onConfirmCancelAll,
  onResume,
  onRetryData,
  onDismiss,
  onBack,
}: GenerationWorkspaceV2Props) {
  // A single job's result retires its entry and the page falls back to idle —
  // keep the log that recorded it on screen (CR M5).
  const showFeed = modeShowsFeed(mode) || (mode === 'idle' && feed.length > 0);
  const isTerminal = mode === 'complete' || mode === 'cancelled' || mode === 'error';
  const singleList = Object.values(singleJobs);
  const rows = progress.items ?? [];
  const verdict = terminalVerdict(mode, progress);
  const [dismissFailed, setDismissFailed] = useState(false);

  const handleDismiss = async () => {
    if (!onDismiss) return;
    setDismissFailed(false);
    try {
      await onDismiss();
    } catch {
      setDismissFailed(true);
    }
  };

  return (
    <div data-testid="generation-workspace" data-mode={mode} className="flex min-h-full flex-col">
      {/* Header */}
      <div className="flex flex-col gap-3.5 px-6 pb-4 pt-6 max-sm:px-4 max-sm:pb-3.5 max-sm:pt-3 sm:px-8">
        {/* Phone (F11-M `e6nOTG`): the TITLE ROW leads (`order-first`) and the
            breadcrumb keeps order 0 — an order on the nav would drop it below the
            banner and the strip, which share this column. -mt-3 = 14 − 12 → 2px. */}
        <nav
          aria-label="麵包屑"
          className="flex items-center gap-1.5 text-sm text-[var(--text-secondary)] max-sm:-mt-3 max-sm:text-xs max-sm:text-[var(--text-muted)]"
        >
          <span>活動</span>
          <span className="text-[var(--text-muted)]" aria-hidden="true">
            ／
          </span>
          <span className="text-[var(--text-primary)] max-sm:text-[var(--text-muted)]">
            生成字幕
          </span>
        </nav>
        <div
          data-testid="workspace-title-row"
          className="flex items-center gap-2 max-sm:order-first max-sm:justify-between"
        >
          {/* Dissolves on desktop (`sm:contents`): h1 + pill stay siblings there. */}
          <div className="flex min-w-0 items-center sm:contents">
            {onBack && (
              <button
                type="button"
                data-testid="workspace-back"
                aria-label="返回活動"
                onClick={onBack}
                // 44×44 hit area; the negative margins give back the 10px it adds on
                // every side, so the glyph sits on the 16px gutter and the row is not
                // stretched to 44.
                className="-my-2.5 -ml-2.5 flex h-11 w-11 shrink-0 items-center justify-center rounded-[var(--radius-md)] text-[var(--text-secondary)] sm:hidden"
              >
                <ChevronLeft className="h-6 w-6" aria-hidden="true" />
              </button>
            )}
            <h1 className="text-xl font-bold text-[var(--text-primary)] max-sm:truncate max-sm:text-lg">
              {mode === 'idle' || mode === 'loading'
                ? '生成工作區'
                : mode === 'single'
                  ? '生成字幕'
                  : '批次生成字幕'}
            </h1>
          </div>
          <StatusPill mode={mode} />
        </div>
        {mode === 'budget_ceiling' && <BudgetBanner progress={progress} />}
        {/* `single` has no BATCH — a 0 / 0 strip with $0.00 would be inventing one. */}
        {!['idle', 'loading', 'single'].includes(mode) && (
          <OverallStrip progress={progress} live={mode === 'running' || mode === 'attach'} />
        )}
      </div>

      {/* Body */}
      <div className="flex min-h-0 flex-1 flex-col gap-4 px-6 pb-6 max-sm:gap-3.5 max-sm:px-4 sm:px-8 lg:flex-row">
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
              <span className="text-sm font-medium text-[var(--error-text)]">無法載入生成狀態</span>
              <button
                type="button"
                data-testid="workspace-data-retry"
                onClick={onRetryData}
                className="ml-auto flex h-11 items-center rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-5 text-sm font-medium text-[var(--text-primary)]"
              >
                重試
              </button>
            </div>
          )}

          {mode === 'idle' && (
            <div
              data-testid="workspace-idle"
              className="flex flex-col items-center justify-center gap-3 rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] px-6 py-6 text-center"
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
                className="mt-1 flex h-11 items-center rounded-[var(--radius-md)] bg-[var(--accent-primary)] px-5 text-sm font-semibold text-[var(--text-on-accent)]"
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
              <h2 className="text-base font-semibold text-[var(--text-primary)]">
                已接上進行中的批次
              </h2>
              {progress.currentItem && (
                <ul className="space-y-2.5">
                  <QueueRow
                    item={{
                      mediaId: progress.currentMediaId ?? 'current',
                      title: progress.currentItem,
                      mediaType: 'movie',
                      seriesTitle: '',
                      status: 'running',
                      reason: '',
                    }}
                    activeItemProgress={activeItemProgress}
                  />
                </ul>
              )}
              {[0, 1].map((i) => (
                <div
                  key={i}
                  aria-hidden="true"
                  className="h-14 rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] opacity-50"
                />
              ))}
              <p className="rounded-[var(--radius-md)] border border-[var(--border-subtle)] px-3 py-2 text-xs text-[var(--text-muted)]">
                佇列明細自本頁開啟起顯示（單部任務或舊版後端時只重建進行中項目）
              </p>
            </div>
          )}

          {/* The heading row is driven by MODE, never by whether we happen to know
              the queue — a cold attach at the ceiling used to get no verdict, and a
              running batch with no snapshot yet got no way to cancel. */}
          {(mode === 'running' || mode === 'budget_ceiling' || isTerminal) && (
            <div className="flex items-center justify-between gap-3 max-sm:flex-wrap">
              <h2 className="text-base font-semibold text-[var(--text-primary)]">生成佇列</h2>
              {mode === 'running' && <CancelAll onConfirm={onConfirmCancelAll} />}
              {verdict && (
                <span
                  data-testid="workspace-terminal-label"
                  className={cn('text-sm font-semibold', verdict.tone)}
                >
                  {verdict.label}
                </span>
              )}
            </div>
          )}

          {(mode === 'running' || mode === 'budget_ceiling' || isTerminal) && rows.length > 0 && (
            <div className="flex flex-col gap-3">
              <ul className="space-y-2.5">
                {rows.map((it) => (
                  <QueueRow
                    key={it.mediaId}
                    item={it}
                    activeItemProgress={
                      it.mediaId === progress.currentMediaId ? activeItemProgress : null
                    }
                  />
                ))}
              </ul>
            </div>
          )}

          {mode === 'single' && (
            <div data-testid="workspace-single" className="flex flex-col gap-3">
              <h2 className="text-base font-semibold text-[var(--text-primary)]">進行中任務</h2>
              <ul className="space-y-2.5">
                {singleList.map((job) => (
                  <QueueRow
                    key={job.mediaId}
                    item={{
                      mediaId: job.mediaId,
                      // dsr-6d-c-2 AC #8: transcription_* now carries the title the
                      // backend resolved for this run ('' when it could not). The
                      // message is a stage sentence (正在轉錄音訊), never a title —
                      // and a raw UUID never is either.
                      title: job.title || '處理中的項目',
                      mediaType: 'movie',
                      seriesTitle: '',
                      status: 'running',
                      reason: '',
                    }}
                    activeItemProgress={{
                      phase: job.phase,
                      failedPhase: null,
                      percentage: job.percentage,
                      message: job.message,
                      jobId: null,
                      error: null,
                      srtPath: null,
                      zhSrtPath: null,
                      partial: false,
                      englishKeptBlocks: null,
                    }}
                  />
                ))}
              </ul>
            </div>
          )}
        </div>

        {showFeed && (
          <EventLogPane
            feed={feed}
            connected={feedConnected}
            budgetStopped={mode === 'budget_ceiling'}
          />
        )}
      </div>

      {/* Footer bar (F12 `v9WuzZ`) — the way OUT of a finished batch. */}
      {(mode === 'budget_ceiling' || isTerminal) && (
        <div
          data-testid="workspace-footer"
          className="flex shrink-0 flex-wrap items-center justify-end gap-3 border-t border-[var(--border-subtle)] bg-[var(--bg-secondary)] px-6 py-3.5 max-sm:px-4 sm:px-8"
        >
          {dismissFailed && (
            <span
              role="alert"
              data-testid="workspace-dismiss-error"
              className="mr-auto text-sm text-[var(--error-text)] max-sm:w-full"
            >
              批次還在進行中，現在無法關閉。
            </span>
          )}
          {onDismiss && (
            <button
              type="button"
              data-testid="workspace-close"
              onClick={() => void handleDismiss()}
              className="flex min-h-[44px] items-center rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-5 text-sm font-medium text-[var(--text-primary)] max-sm:flex-1 max-sm:justify-center"
            >
              關閉
            </button>
          )}
          {mode === 'budget_ceiling' && (
            <button
              type="button"
              data-testid="workspace-resume"
              onClick={onResume}
              className="flex min-h-[44px] items-center rounded-[var(--radius-md)] bg-[var(--accent-primary)] px-5 text-sm font-semibold text-[var(--text-on-accent)] max-sm:flex-1 max-sm:justify-center"
            >
              下次繼續
            </button>
          )}
        </div>
      )}
    </div>
  );
}

/**
 * Batch-wide cancel with a two-step confirm (the dialog's shape). The request can
 * fail, and a cancel that looks like it worked while the batch keeps running and
 * keeps spending is the lie this guards against.
 */
function CancelAll({ onConfirm }: { onConfirm: () => Promise<void> }) {
  const [confirming, setConfirming] = useState(false);
  const [busy, setBusy] = useState(false);
  const [failed, setFailed] = useState(false);
  const cancelAllRef = useRef<HTMLButtonElement | null>(null);
  const keepGoingRef = useRef<HTMLButtonElement | null>(null);
  /** Where focus must land once that control exists (the dialog's pattern). */
  const pendingFocusRef = useRef<'keepGoing' | 'cancelAll' | null>(null);

  // Runs after EVERY render: the control we want focused may not exist yet.
  useEffect(() => {
    const target = pendingFocusRef.current;
    if (!target) return;
    const el = target === 'keepGoing' ? keepGoingRef.current : cancelAllRef.current;
    if (el) {
      el.focus();
      pendingFocusRef.current = null;
    }
  });

  if (!confirming) {
    return (
      <button
        type="button"
        ref={cancelAllRef}
        data-testid="workspace-cancel-all"
        onClick={() => {
          setFailed(false);
          setConfirming(true);
          pendingFocusRef.current = 'keepGoing';
        }}
        className="flex min-h-[44px] items-center rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-5 text-sm font-medium text-[var(--text-primary)]"
      >
        全部取消
      </button>
    );
  }

  return (
    <div
      data-testid="workspace-cancel-confirm"
      className="flex flex-wrap items-center gap-3 max-sm:w-full"
    >
      <span className="text-sm text-[var(--text-secondary)] max-sm:w-full">
        確定要取消整個批次嗎？已完成的字幕會保留。
      </span>
      {failed && (
        <span
          role="alert"
          data-testid="workspace-cancel-error"
          className="text-sm text-[var(--error-text)] max-sm:w-full"
        >
          取消失敗，批次仍在進行。請再試一次。
        </span>
      )}
      <button
        type="button"
        ref={keepGoingRef}
        onClick={() => {
          setConfirming(false);
          setFailed(false);
          pendingFocusRef.current = 'cancelAll';
        }}
        className="flex min-h-[44px] items-center rounded-[var(--radius-md)] px-4 text-sm text-[var(--text-secondary)] max-sm:flex-1 max-sm:justify-center"
      >
        繼續生成
      </button>
      <button
        type="button"
        // aria-disabled, not disabled: a disabled button drops focus to <body>.
        aria-disabled={busy || undefined}
        data-testid="workspace-cancel-confirm-btn"
        onClick={() => {
          if (busy) return;
          setBusy(true);
          setFailed(false);
          void onConfirm()
            .catch(() => {
              setFailed(true);
              setBusy(false);
            })
            .then(() => undefined);
        }}
        className={cn(
          'flex min-h-[44px] items-center rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-4 text-sm text-[var(--text-primary)] max-sm:flex-1 max-sm:justify-center',
          busy && 'opacity-60'
        )}
      >
        {busy && !failed ? '取消中…' : '確定取消'}
      </button>
    </div>
  );
}

// --- Container ---------------------------------------------------------------

export interface GenerationWorkspaceProps {
  /** True when this view is the active + visible surface (drives lazy SSE, §8). */
  active: boolean;
  /** Opens the F8 batch dialog (the launcher — the workspace never rebuilds scope). */
  onLaunch: () => void;
  /** Phone back button → 活動 (dsr-6f-4). Passed straight through. */
  onBack?: () => void;
}

export function GenerationWorkspace({ active, onLaunch, onBack }: GenerationWorkspaceProps) {
  const isVisible = usePageVisibility();
  const live = active && isVisible;
  const queryClient = useQueryClient();

  const batch = useGenerationBatchProgress();
  const jobs = useGenerationJobsFeed();
  const activeItem = useGenerationProgress();
  const { startTracking: startBatchTracking, attachSnapshot, connectionEpoch } = batch;
  const { startTracking: startItemTracking } = activeItem;
  const { seedBatch: seedFeedBatch, endBatch: endFeedBatch } = jobs;

  /**
   * The terminal result the USER has explicitly closed (關閉 → dismiss). It is
   * NOT "the probe already showed it once": the tab going hidden resets the hook,
   * and blocking the re-attach on return is how the result — and the only dismiss
   * exit — used to disappear for good (CR H1). The server forgets `last` on
   * dismiss anyway; this ref only covers the window before that refetch lands.
   */
  const dismissedLastBatchIdRef = useRef<string | null>(null);
  /** Terminal batch ids already seen, so a late probe cannot resurrect them. */
  const terminalBatchIdsRef = useRef<Set<string>>(new Set());
  /** Terminal batches whose cache refresh already ran (once per batch). */
  const handledTerminalRef = useRef<string | null>(null);

  // 缺字幕 preview count for the idle launcher (shared cache with the dialog).
  const previewQuery = useQuery({
    queryKey: generationBatchPreviewKey,
    queryFn: () => subtitleService.previewGenerationBatch(),
    enabled: live,
    staleTime: 30_000,
    retry: 1,
  });

  /**
   * Status probe. staleTime 0 + refetchOnWindowFocus 'always': coming back to the
   * tab MUST re-read, or a batch that finished while we were away leaves the page
   * claiming 進行中 for the rest of the cache window with no way out.
   */
  const statusQuery = useQuery({
    queryKey: generationBatchStatusKey,
    queryFn: () => subtitleService.getGenerationBatchStatus(),
    enabled: live,
    retry: 1,
    staleTime: 0,
    refetchOnWindowFocus: 'always',
  });

  // A backoff reconnect may have swallowed the terminal event — re-read status.
  const probeData = statusQuery.data;
  useEffect(() => {
    if (!live) return;
    void statusQuery.refetch();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [connectionEpoch]);

  // Feed + batch attachment. Deliberately NOT the per-item subscription (that has
  // its own effect below) — one effect doing both is what froze the stepper on the
  // first title and what re-seeded (and reset) the batch on every refetch.
  useEffect(() => {
    if (!live) {
      batch.reset();
      jobs.stop();
      activeItem.reset();
      return;
    }
    jobs.startTracking();
    if (!probeData) return;
    if (probeData.running && probeData.progress) {
      // A probe issued BEFORE the terminal event can resolve after it. Re-seeding
      // would force `status: 'running'` and reopen a stream that will never carry
      // another event for this batch — 進行中 for ever (CR H3).
      if (terminalBatchIdsRef.current.has(probeData.progress.batchId)) return;
      startBatchTracking(probeData.progress);
      // The log must know which films are this batch's, or a mid-batch attach
      // logs the running one as an untitled single job.
      seedFeedBatch(probeData.progress.batchId, probeData.progress.items ?? []);
      return;
    }
    // Nothing runs: whatever the log still counts as the batch's is left over
    // from a terminal it never saw (tab hidden, reconnect gap) — CR H1.
    endFeedBatch();
    const last = probeData.last;
    if (last?.batchId && dismissedLastBatchIdRef.current !== last.batchId) {
      attachSnapshot(last);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [live, probeData]);

  // Per-item join: follow the batch to the NEXT title. Without currentMediaId in
  // the deps the stepper stays subscribed to the first item for the whole batch.
  const currentMediaId = batch.progress.currentMediaId;
  const batchStatus = batch.status;
  useEffect(() => {
    if (live && batchStatus === 'running' && currentMediaId) startItemTracking(currentMediaId);
  }, [live, batchStatus, currentMediaId, startItemTracking]);

  // The batch hook saw the terminal (its own stream) — end the log's membership
  // too, in case the log's stream missed that frame (CR H1).
  useEffect(() => {
    if (batchStatus !== 'idle' && batchStatus !== 'running') endFeedBatch();
  }, [batchStatus, endFeedBatch]);

  // Terminal: the finished items wrote subtitle_status back. Refresh what is now
  // stale — ONCE per batch, so re-attaching `last` on a later visit is free.
  const batchId = batch.progress.batchId;
  useEffect(() => {
    if (batchStatus === 'idle' || batchStatus === 'running') return;
    if (batchId) terminalBatchIdsRef.current.add(batchId);
    if (handledTerminalRef.current === batchId) return;
    handledTerminalRef.current = batchId;
    void queryClient.invalidateQueries({ queryKey: libraryKeys.all });
    void queryClient.invalidateQueries({ queryKey: generationBatchPreviewKey });
    void queryClient.invalidateQueries({ queryKey: detailKeys.all });
    void queryClient.invalidateQueries({ queryKey: activityKeys.all });
    void queryClient.invalidateQueries({ queryKey: transcriptionEstimateKeys.all });
  }, [batchStatus, batchId, queryClient]);

  const mode = deriveWorkspaceMode({
    probing: live && statusQuery.isLoading,
    batchStatus: batch.status,
    hasItems: (batch.progress.items?.length ?? 0) > 0,
    singleJobCount: Object.keys(jobs.singleJobs).length,
  });

  const dataError = live && !statusQuery.isLoading && (statusQuery.isError || previewQuery.isError);

  const handleConfirmCancelAll = useCallback(async () => {
    const result = await subtitleService.cancelGenerationBatch();
    if (!result.cancelled && !result.running) {
      // Nothing to cancel — it had already finished and we missed the terminal.
      void statusQuery.refetch();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleDismiss = useCallback(async () => {
    const result = await subtitleService.dismissGenerationBatch();
    if (!result.dismissed && result.running) {
      // Still running — "closing" it would be a lie; surface it as a failure.
      throw new Error('batch still running');
    }
    dismissedLastBatchIdRef.current = batch.progress.batchId;
    batch.reset();
    void statusQuery.refetch();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [batch.progress.batchId]);

  return (
    <GenerationWorkspaceV2
      mode={mode}
      progress={batch.progress}
      activeItemProgress={activeItem.progress}
      singleJobs={jobs.singleJobs}
      feed={jobs.feed}
      feedConnected={jobs.connected}
      // sub-5-1 AC #7: the movies-only total_items is NOT what the consent list
      // will show — prefer the episode-inclusive count, fall back for old servers.
      previewCount={previewQuery.data?.totalItemsIncludingEpisodes ?? previewQuery.data?.totalItems}
      dataError={dataError}
      onLaunch={onLaunch}
      onConfirmCancelAll={handleConfirmCancelAll}
      // sub-4-3 CR H1: 下次繼續 must NEVER start an un-consented scope=missing
      // batch (the exact class the 2026-08-07 三件一體 ruling bans). A resume
      // is a NEW consent — route through the launch path, which opens the
      // dialog in its consent phase (F15 re-select → F16 confirm).
      onResume={onLaunch}
      onDismiss={handleDismiss}
      onBack={onBack}
      onRetryData={() => {
        void statusQuery.refetch();
        void previewQuery.refetch();
      }}
    />
  );
}
