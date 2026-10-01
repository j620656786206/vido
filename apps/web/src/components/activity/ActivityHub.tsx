// Design ref: ux-design.pen Screen K1-D-v2 (kMeWS) · K1-M-v2 (QIwY1)
// 代號在「活動中心從 Flow A 獨立成 Flow K」那次重組時改過，節點 ID 沒變、名字變了；
// 這一行原本寫 A1-D-v2，照它去翻會翻到瀏覽流程。（dsr-10）
/**
 * Activity hub page (UX Redesign Phase 3 — ux3-2-3 / D4-1). The v2 destination that
 * unifies the previously-invisible background journeys: 進行中 (live scan / batch-subtitle
 * jobs) → 待處理 (pending parse) → 下載 (summary row that LINKS OUT to the deep page,
 * D4-1 HYBRID) → 本月 AI 花費 (sub-7-6c, its own query — SpendSection) → 活動記錄 (recent
 * terminal events). Consumes the fail-soft aggregate
 * GET /api/v1/activity (ux3-2-2): a section reporting `unavailable` degrades to an inline
 * banner while the rest of the page renders (F3); the whole page only shows a single
 * error when the request itself fails. Four states (N4): loading / empty / per-section
 * fail-soft / data. Copy + icons live here — the backend sends copy-free enums.
 */
import { useState } from 'react';
import { Link, getRouteApi } from '@tanstack/react-router';
import {
  Radar,
  Captions,
  FileSearch,
  Download,
  CircleCheck,
  AlertTriangle,
  ChevronRight,
  Activity,
  Sparkles,
} from 'lucide-react';
import type { LucideIcon } from 'lucide-react';
import { useActivity } from '../../hooks/useActivity';
import { useSubtitleSpend } from '../../hooks/useSubtitleSpend';
import { GenerationBatchDialogV2 } from '../subtitle/GenerationBatchDialogV2';
import { GenerationWorkspace } from '../subtitle/GenerationWorkspaceV2';
import type {
  ActivitySummary,
  ActiveJobsSection,
  PendingSection,
  DownloadsSection,
  RecentSection,
} from '../../services/activityService';
import { formatRelativeTime } from '../../utils/relativeTime';
import { ActivityRow } from './ActivityRow';
import { ActivitySectionShell } from './ActivitySectionShell';
import { SpendSectionView, spendHasContent } from './SpendSection';
import { ActivitySkeleton, ActivityEmpty, ActivitySectionError } from './ActivityStates';

const ACTIVE_META: Record<string, { icon: LucideIcon; title: string }> = {
  scan: { icon: Radar, title: '媒體庫掃描' },
  // dsr-10: 「批次字幕」跟下面的「批次生成」分不出來。後端這個 kind 走的是
  // subtitle/batch.go，打 providers.SubtitleQuery、掃 NotSearched/NotFound —— 它是
  // 去**搜尋**已經存在的字幕，不是生成。
  subtitle_batch: { icon: Captions, title: '批次字幕搜尋' },
  // Story ux3-subtitle-v2-batch AC 4 — the 9R-16 generation-batch job row.
  generation_batch: { icon: Captions, title: '批次生成' },
  // disc-2026-07-transcription-active-jobs — a solo (non-batch) 生成字幕 click,
  // previously invisible once its progress modal closed.
  transcription: { icon: Sparkles, title: '字幕生成中' },
};

/** Jobs whose right-hand slot renders `current / total` instead of a percent. */
const COUNTED_KINDS = new Set(['subtitle_batch', 'generation_batch']);

/**
 * Jobs with no fractional/bounded progress to report — the backend deliberately
 * sends percentDone=0 rather than fabricate a number (disc-2026-07-
 * transcription-active-jobs: this service tracks discrete pipeline stages, not
 * a byte/item count). Showing a literal "0%" for a job's entire runtime would
 * read as stuck; these render a static "進行中" label and no progress bar
 * instead of `${percentDone}%`.
 */
const NO_PERCENT_KINDS = new Set(['transcription']);

/**
 * The four /activity sections say nothing. The spend card (sub-7-6c) is judged
 * beside this in ActivityHub, from its own query: a month with money in it is
 * content, and a spend endpoint that failed is a section to show, so neither
 * may be swept into the calm empty state.
 */
function isEmpty(d: ActivitySummary): boolean {
  return (
    d.activeJobs.status === 'ok' &&
    d.activeJobs.jobs.length === 0 &&
    d.pending.status === 'ok' &&
    d.pending.parseCount === 0 &&
    // Not set up yet counts as "no downloads", not as a failure.
    (d.downloads.status === 'not_configured' ||
      (d.downloads.status === 'ok' && d.downloads.total === 0)) &&
    d.recent.status === 'ok' &&
    d.recent.events.length === 0
  );
}

function ActiveSection({ section, onRetry }: { section: ActiveJobsSection; onRetry: () => void }) {
  if (section.status === 'unavailable') {
    return (
      <ActivitySectionShell title="進行中">
        <ActivitySectionError onRetry={onRetry} testId="activity-active-error" />
      </ActivitySectionShell>
    );
  }
  if (section.jobs.length === 0) return null;
  return (
    <ActivitySectionShell title="進行中" count={section.jobs.length}>
      {section.jobs.map((j, i) => {
        const meta = ACTIVE_META[j.kind] ?? { icon: Activity, title: j.kind };
        const right =
          COUNTED_KINDS.has(j.kind) && j.total ? (
            <span className="font-mono text-xs text-[var(--text-secondary)]">
              {j.current ?? 0} / {j.total}
            </span>
          ) : NO_PERCENT_KINDS.has(j.kind) ? (
            <span className="text-xs text-[var(--text-secondary)]">進行中</span>
          ) : (
            <span className="font-mono text-xs text-[var(--accent-text)]">{j.percentDone}%</span>
          );
        const row = (
          <ActivityRow
            icon={meta.icon}
            title={meta.title}
            detail={j.detail}
            right={right}
            progress={NO_PERCENT_KINDS.has(j.kind) ? undefined : j.percentDone}
            testId={`activity-job-${j.kind}`}
          />
        );
        // Story ux3-ai-2 — the generation-batch row LINKS to the immersive workspace
        // (the WATCHER); the header CTA stays the LAUNCHER (dialog). D4-1: one row, one link.
        if (j.kind === 'generation_batch') {
          return (
            <Link
              key={`${j.kind}-${i}`}
              to="/activity"
              search={{ view: 'generation' }}
              data-testid="activity-generation-batch-link"
              className="block rounded-[var(--radius-lg)] transition-colors hover:bg-[var(--bg-tertiary)]/40"
            >
              {row}
            </Link>
          );
        }
        return <div key={`${j.kind}-${i}`}>{row}</div>;
      })}
    </ActivitySectionShell>
  );
}

function PendingSectionView({
  section,
  onRetry,
}: {
  section: PendingSection;
  onRetry: () => void;
}) {
  if (section.status === 'unavailable') {
    return (
      <ActivitySectionShell title="待處理">
        <ActivitySectionError onRetry={onRetry} testId="activity-pending-error" />
      </ActivitySectionShell>
    );
  }
  if (section.parseCount === 0) return null;
  return (
    <ActivitySectionShell title="待處理">
      <ActivityRow
        icon={FileSearch}
        title="待解析項目"
        detail={`${section.parseCount} 個項目待處理`}
        testId="activity-pending-row"
        right={
          <Link
            to="/library"
            search={{ unmatched: true }}
            data-testid="activity-pending-cta"
            className="flex items-center gap-1 text-sm font-medium text-[var(--accent-text)] hover:underline"
          >
            前往處理
            <ChevronRight className="h-3.5 w-3.5" aria-hidden="true" />
          </Link>
        }
      />
    </ActivitySectionShell>
  );
}

function DownloadsSectionView({
  section,
  onRetry,
}: {
  section: DownloadsSection;
  onRetry: () => void;
}) {
  if (section.status === 'unavailable') {
    return (
      <ActivitySectionShell title="下載">
        <ActivitySectionError onRetry={onRetry} testId="activity-downloads-error" />
      </ActivitySectionShell>
    );
  }
  // ⚖️ Alexyu 2026-09-30 (disc-activity-downloads-unconfigured-copy, A): no
  // qBittorrent yet is not a load failure — hide the section like "no
  // downloads"; the downloads page (d11) and 服務狀態 do the nudging.
  if (section.status === 'not_configured' || section.total === 0) return null;
  return (
    <ActivitySectionShell title="下載">
      <ActivityRow
        icon={Download}
        title="下載中"
        detail={
          /* bugfix-e AC #2: errored/paused torrents used to be swept into 個排隊, so a
             library of 3,068 broken torrents read as a healthy queue. Both counts are
             appended only when non-zero — a healthy system's line is byte-unchanged.

             CR M3: 錯誤 leads the line. ActivityRow renders `detail` inside a
             `truncate` <p>, so on a narrow viewport the tail is ellipsed away —
             and the errored count IS the signal this story exists to surface.
             Alarm first, ambient counts after.

             CR L1: AC #2 names the `--error` token, but text uses the AA-safe
             `--error-text` variant per the DL-v2 §2.5 / TC-2 convention that
             `downloadStatus.ts` already follows. Deliberate deviation. */
          <>
            {section.errored > 0 && (
              <span className="text-[var(--error-text)]" data-testid="activity-downloads-errored">
                {`${section.errored} 個錯誤 · `}
              </span>
            )}
            {`${section.downloading} 個進行中 · ${section.queued} 個排隊`}
            {section.paused > 0 && (
              <span className="text-[var(--text-muted)]" data-testid="activity-downloads-paused">
                {` · ${section.paused} 個暫停`}
              </span>
            )}
          </>
        }
        testId="activity-downloads-row"
        right={
          <Link
            to="/downloads"
            data-testid="activity-downloads-cta"
            className="flex items-center gap-1 text-sm font-medium text-[var(--accent-text)] hover:underline"
          >
            開啟下載頁
            <ChevronRight className="h-3.5 w-3.5" aria-hidden="true" />
          </Link>
        }
      />
    </ActivitySectionShell>
  );
}

function RecentSectionView({ section, onRetry }: { section: RecentSection; onRetry: () => void }) {
  if (section.status === 'unavailable') {
    return (
      <ActivitySectionShell title="活動記錄">
        <ActivitySectionError onRetry={onRetry} testId="activity-recent-error" />
      </ActivitySectionShell>
    );
  }
  if (section.events.length === 0) return null;
  return (
    <ActivitySectionShell title="活動記錄">
      {section.events.map((ev, i) => {
        const failed = ev.result === 'failed';
        return (
          <ActivityRow
            key={`${ev.detail ?? 'event'}-${i}`}
            icon={failed ? AlertTriangle : CircleCheck}
            iconTone={failed ? 'error' : 'success'}
            title={failed ? '解析失敗' : '解析完成'}
            detail={ev.detail}
            testId="activity-recent-row"
            right={
              <span className="whitespace-nowrap text-xs text-[var(--text-muted)]">
                {formatRelativeTime(ev.at)}
              </span>
            }
          />
        );
      })}
    </ActivitySectionShell>
  );
}

const routeApi = getRouteApi('/activity');

export function ActivityHub() {
  const { data, isLoading, isError, refetch } = useActivity();
  // 本月 AI 花費 rides its own endpoint (sub-7-6b), so it is its own query —
  // a slow or broken ledger must not take the four live sections down with it.
  const spend = useSubtitleSpend();
  const search = routeApi.useSearch();
  const navigate = routeApi.useNavigate();
  // Story ux3-subtitle-v2-batch AC 4a — the hub's launch CTA opens the batch
  // dialog with scope=missing (the ONLY Activity-side entry; D4-1 boundary).
  const [generationBatchOpen, setGenerationBatchOpen] = useState(false);
  const goToKeySettings = () => {
    setGenerationBatchOpen(false);
    void navigate({ to: '/settings/keys' });
  };
  const retry = () => {
    void refetch();
    void spend.refetch();
  };
  const retrySpend = () => {
    void spend.refetch();
  };

  // Story ux3-ai-2 — `?view=generation` hosts the F11 generation workspace in place
  // of the hub body (the immersive WATCHER; the dialog stays the LAUNCHER). The
  // workspace's own SSE is visibility-gated; leaving the view unmounts it → streams close.
  if (search.view === 'generation') {
    return (
      <>
        <div data-testid="activity-root" className="h-full">
          <GenerationWorkspace
            active
            onLaunch={() => setGenerationBatchOpen(true)}
            // dsr-6f-4: the phone back button — drop ?view=generation (a push, so
            // the browser's own Back returns to the workspace).
            onBack={() => void navigate({ to: '/activity', search: {} })}
          />
        </div>
        <GenerationBatchDialogV2
          open={generationBatchOpen}
          onOpenChange={setGenerationBatchOpen}
          onGoToKeySettings={goToKeySettings}
        />
      </>
    );
  }

  const spendHasData = spend.data !== undefined && spendHasContent(spend.data);
  // ⚖️ Alexyu 2026-10-01: the card sits BELOW 進行中／待處理／下載 (you come here
  // to see what is running) and above 活動記錄. When nothing is running those
  // sections do not render, so the card is first by itself — no reordering.
  // Judged HERE, after the workspace branch: the hub body is the only reader.
  const pageEmpty = data !== undefined && isEmpty(data) && !spendHasData && !spend.isError;

  return (
    <div
      data-testid="activity-root"
      className="mx-auto flex w-full max-w-5xl flex-col gap-6 px-4 py-8 sm:px-6"
    >
      <header className="flex flex-wrap items-start justify-between gap-3">
        <div className="flex flex-col gap-1">
          <h1 className="text-xl sm:text-2xl font-bold text-[var(--text-primary)]">活動</h1>
          <p className="text-sm text-[var(--text-secondary)]">
            媒體庫的所有背景工作 — 掃描、字幕、解析與下載
          </p>
        </div>
        <button
          type="button"
          data-testid="activity-generation-batch-cta"
          onClick={() => setGenerationBatchOpen(true)}
          className="flex min-h-[44px] items-center gap-1.5 rounded-[var(--radius-md)] bg-[var(--accent-primary)] px-4 text-sm font-medium text-[var(--text-on-accent)] transition-colors hover:bg-[var(--accent-pressed)]"
        >
          <Captions className="h-4 w-4" aria-hidden="true" />
          批次生成字幕
        </button>
      </header>

      {isLoading || (spend.isLoading && !spend.isError) ? (
        <ActivitySkeleton />
      ) : isError || !data ? (
        <ActivitySectionError onRetry={retry} testId="activity-page-error" />
      ) : pageEmpty ? (
        <ActivityEmpty />
      ) : (
        <>
          <ActiveSection section={data.activeJobs} onRetry={retry} />
          <PendingSectionView section={data.pending} onRetry={retry} />
          <DownloadsSectionView section={data.downloads} onRetry={retry} />
          <SpendSectionView summary={spend.data} failed={spend.isError} onRetry={retrySpend} />
          <RecentSectionView section={data.recent} onRetry={retry} />
        </>
      )}

      {/* Batch dialog opens OVER the hub (F8/F9 backdrops render A1-D-v2). */}
      <GenerationBatchDialogV2
        open={generationBatchOpen}
        onOpenChange={setGenerationBatchOpen}
        onGoToKeySettings={goToKeySettings}
      />
    </div>
  );
}
