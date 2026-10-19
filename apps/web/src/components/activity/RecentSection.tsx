// Design ref: ux-design.pen Screen K1-D-v2 (kMeWS) · K1-M-v2 (QIwY1) — 活動記錄列 (LndN2) (EVrLx) (JLTBV) (akt4i)
// Clock-injected: component accepts `clock` prop (named `now`); no fixture-side mock needed
/**
 * 活動記錄 — the activity hub's recent terminal events (ux3-2-3), moved out of
 * ActivityHub by disc-2026-10-activity-subtitle-run-time-b so the gallery can
 * photograph it with a fixed clock.
 *
 * Two kinds, newest first as the backend merged them:
 * - `parse` (and any kind this build does not know): 解析完成／解析失敗 + the
 *   backend's `detail` + one relative-time line — exactly as before.
 * - `subtitle`: 字幕完成／字幕沒做成, a description built from the copy-free
 *   fields (recentEventCopy), and 「花多久 · 花多少錢」 as a meta line that sits
 *   above the time on desktop and under the description on a phone (⚖️ Sally
 *   2026-10-19 rulings 2, 5, 6). Only two states: a money stop is still 沒做成.
 *   The meta starts with a screen-reader-only 「花了」 (⚖️ Alexyu 2026-10-19 1A)
 *   so it is not heard as one more time next to the relative time; the screen
 *   is unchanged.
 */
import { AlertTriangle, CircleCheck } from 'lucide-react';
import type {
  RecentEvent,
  RecentSection as RecentSectionData,
} from '../../services/activityService';
import { formatRelativeTime } from '../../utils/relativeTime';
import { ActivityRow } from './ActivityRow';
import { ActivitySectionShell } from './ActivitySectionShell';
import { ActivitySectionError } from './ActivityStates';
import {
  recentSubtitleDetail,
  recentSubtitleMeta,
  recentSubtitleMetaLead,
} from './recentEventCopy';

interface RecentSectionProps {
  section: RecentSectionData;
  onRetry: () => void;
  /** Epoch ms the relative times count from; omitted → the wall clock. */
  now?: number;
}

function TimeLabel({ at, now }: { at: string; now?: number }) {
  return (
    <span
      data-testid="activity-recent-time"
      className="whitespace-nowrap text-xs text-[var(--text-muted)]"
    >
      {formatRelativeTime(at, now)}
    </span>
  );
}

function SubtitleRow({ ev, now }: { ev: RecentEvent; now?: number }) {
  const failed = ev.result === 'failed';
  const meta = recentSubtitleMeta(ev);
  const lead = recentSubtitleMetaLead(ev);
  return (
    <ActivityRow
      icon={failed ? AlertTriangle : CircleCheck}
      iconTone={failed ? 'error' : 'success'}
      title={failed ? '字幕沒做成' : '字幕完成'}
      detail={recentSubtitleDetail(ev)}
      testId="activity-recent-row"
      kind="subtitle"
      wrapDetailOnMobile
      meta={
        meta && (
          <span
            data-testid="activity-recent-meta"
            className="font-mono text-xs text-[var(--text-secondary)] sm:text-sm"
          >
            {lead && <span className="sr-only">{lead}</span>}
            {meta}
          </span>
        )
      }
      right={<TimeLabel at={ev.at} now={now} />}
    />
  );
}

export function RecentSection({ section, onRetry, now }: RecentSectionProps) {
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
        if (ev.kind === 'subtitle') {
          return <SubtitleRow key={`subtitle-${ev.at}-${i}`} ev={ev} now={now} />;
        }
        const failed = ev.result === 'failed';
        return (
          <ActivityRow
            key={`${ev.detail ?? 'event'}-${i}`}
            icon={failed ? AlertTriangle : CircleCheck}
            iconTone={failed ? 'error' : 'success'}
            title={failed ? '解析失敗' : '解析完成'}
            detail={ev.detail}
            testId="activity-recent-row"
            kind={ev.kind}
            right={<TimeLabel at={ev.at} now={now} />}
          />
        );
      })}
    </ActivitySectionShell>
  );
}
