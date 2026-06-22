// Design ref: ux-design.pen Screen A1-D-v2 (kMeWS)
/**
 * Activity hub page (UX Redesign Phase 3 — ux3-2-3 / D4-1). The v2 destination that
 * unifies the previously-invisible background journeys: 進行中 (live scan / batch-subtitle
 * jobs) → 待處理 (pending parse) → 下載 (summary row that LINKS OUT to the deep page,
 * D4-1 HYBRID) → 活動記錄 (recent terminal events). Consumes the fail-soft aggregate
 * GET /api/v1/activity (ux3-2-2): a section reporting `unavailable` degrades to an inline
 * banner while the rest of the page renders (F3); the whole page only shows a single
 * error when the request itself fails. Four states (N4): loading / empty / per-section
 * fail-soft / data. Copy + icons live here — the backend sends copy-free enums.
 */
import { Link } from '@tanstack/react-router';
import {
  Radar,
  Captions,
  FileSearch,
  Download,
  CircleCheck,
  AlertTriangle,
  ChevronRight,
  Activity,
} from 'lucide-react';
import type { LucideIcon } from 'lucide-react';
import { useActivity } from '../../hooks/useActivity';
import type {
  ActivitySummary,
  ActiveJobsSection,
  PendingSection,
  DownloadsSection,
  RecentSection,
} from '../../services/activityService';
import { formatRelativeTime } from '../../utils/relativeTime';
import { ActivityRow } from './ActivityRow';
import { ActivitySkeleton, ActivityEmpty, ActivitySectionError } from './ActivityStates';

const ACTIVE_META: Record<string, { icon: LucideIcon; title: string }> = {
  scan: { icon: Radar, title: '媒體庫掃描' },
  subtitle_batch: { icon: Captions, title: '批次字幕' },
};

function isEmpty(d: ActivitySummary): boolean {
  return (
    d.activeJobs.status === 'ok' &&
    d.activeJobs.jobs.length === 0 &&
    d.pending.status === 'ok' &&
    d.pending.parseCount === 0 &&
    d.downloads.status === 'ok' &&
    d.downloads.total === 0 &&
    d.recent.status === 'ok' &&
    d.recent.events.length === 0
  );
}

function SectionShell({
  title,
  count,
  children,
}: {
  title: string;
  count?: number;
  children: React.ReactNode;
}) {
  return (
    <section className="flex flex-col gap-3">
      <div className="flex items-center gap-2">
        <h2 className="text-base font-bold text-[var(--text-primary)]">{title}</h2>
        {typeof count === 'number' && count > 0 && (
          <span className="rounded-full bg-[var(--accent-tint)] px-2.5 py-0.5 font-mono text-xs text-[var(--accent-text)]">
            {count}
          </span>
        )}
      </div>
      {children}
    </section>
  );
}

function ActiveSection({ section, onRetry }: { section: ActiveJobsSection; onRetry: () => void }) {
  if (section.status === 'unavailable') {
    return (
      <SectionShell title="進行中">
        <ActivitySectionError onRetry={onRetry} testId="activity-active-error" />
      </SectionShell>
    );
  }
  if (section.jobs.length === 0) return null;
  return (
    <SectionShell title="進行中" count={section.jobs.length}>
      {section.jobs.map((j, i) => {
        const meta = ACTIVE_META[j.kind] ?? { icon: Activity, title: j.kind };
        const right =
          j.kind === 'subtitle_batch' && j.total ? (
            <span className="font-mono text-[13px] text-[var(--text-secondary)]">
              {j.current ?? 0} / {j.total}
            </span>
          ) : (
            <span className="font-mono text-[13px] text-[var(--accent-text)]">
              {j.percentDone}%
            </span>
          );
        return (
          <ActivityRow
            key={`${j.kind}-${i}`}
            icon={meta.icon}
            title={meta.title}
            detail={j.detail}
            right={right}
            progress={j.percentDone}
            testId={`activity-job-${j.kind}`}
          />
        );
      })}
    </SectionShell>
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
      <SectionShell title="待處理">
        <ActivitySectionError onRetry={onRetry} testId="activity-pending-error" />
      </SectionShell>
    );
  }
  if (section.parseCount === 0) return null;
  return (
    <SectionShell title="待處理">
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
            className="flex items-center gap-1 text-[13px] font-medium text-[var(--accent-primary)] hover:text-[var(--accent-hover)]"
          >
            前往處理
            <ChevronRight className="h-3.5 w-3.5" aria-hidden="true" />
          </Link>
        }
      />
    </SectionShell>
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
      <SectionShell title="下載">
        <ActivitySectionError onRetry={onRetry} testId="activity-downloads-error" />
      </SectionShell>
    );
  }
  if (section.total === 0) return null;
  return (
    <SectionShell title="下載">
      <ActivityRow
        icon={Download}
        title="下載中"
        detail={`${section.downloading} 個進行中 · ${section.queued} 個排隊`}
        testId="activity-downloads-row"
        right={
          <Link
            to="/downloads"
            data-testid="activity-downloads-cta"
            className="flex items-center gap-1 text-[13px] font-medium text-[var(--accent-primary)] hover:text-[var(--accent-hover)]"
          >
            開啟下載頁
            <ChevronRight className="h-3.5 w-3.5" aria-hidden="true" />
          </Link>
        }
      />
    </SectionShell>
  );
}

function RecentSectionView({ section, onRetry }: { section: RecentSection; onRetry: () => void }) {
  if (section.status === 'unavailable') {
    return (
      <SectionShell title="活動記錄">
        <ActivitySectionError onRetry={onRetry} testId="activity-recent-error" />
      </SectionShell>
    );
  }
  if (section.events.length === 0) return null;
  return (
    <SectionShell title="活動記錄">
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
    </SectionShell>
  );
}

export function ActivityHub() {
  const { data, isLoading, isError, refetch } = useActivity();
  const retry = () => {
    void refetch();
  };

  return (
    <div
      data-testid="activity-root"
      className="mx-auto flex w-full max-w-5xl flex-col gap-6 px-4 py-8 sm:px-6"
    >
      <header className="flex flex-col gap-1">
        <h1 className="text-2xl font-bold text-[var(--text-primary)]">活動</h1>
        <p className="text-sm text-[var(--text-secondary)]">
          媒體庫的所有背景工作 — 掃描、字幕、解析與下載
        </p>
      </header>

      {isLoading ? (
        <ActivitySkeleton />
      ) : isError || !data ? (
        <ActivitySectionError onRetry={retry} testId="activity-page-error" />
      ) : isEmpty(data) ? (
        <ActivityEmpty />
      ) : (
        <>
          <ActiveSection section={data.activeJobs} onRetry={retry} />
          <PendingSectionView section={data.pending} onRetry={retry} />
          <DownloadsSectionView section={data.downloads} onRetry={retry} />
          <RecentSectionView section={data.recent} onRetry={retry} />
        </>
      )}
    </div>
  );
}
