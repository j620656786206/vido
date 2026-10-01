// Design ref: ux-design.pen Screen K5-D-v2 (xgYKA) · K5-M-v2 (ptNai)
/**
 * 本月 AI 花費 — the activity hub's monthly subtitle-spend card (sub-7-6c AC #1).
 *
 * The one place the user can SEE the moat paying for itself: what the two paid
 * lanes (翻譯 / 語音辨識) cost this month, what the free lanes avoided (直接交付
 * 舊字幕, the prompt cache) and which model the money went to. A CARD, not an
 * ActivityRow: the row is "one job, one line", this is one month's ledger.
 *
 * Honesty rules (AC #4) — the same vocabulary the readout band uses:
 *  - nothing recorded renders 「—」, never $0 (a lane with 0 runs, a cache no
 *    run measured);
 *  - every saving is an ESTIMATE and says so in words;
 *  - `≈` keeps its one meaning — the amount rests on an assumed runtime — so
 *    it appears on the skipped saving ONLY when the backend flagged
 *    `skipped_saved_runtime_assumed` (⚖️ Sally 2026-09-05, carried by 7-6 AA);
 *  - completed runs with no recorded amount and pre-ledger rows are listed in a
 *    footnote, kept out of the lane totals, so the headline never claims money
 *    nobody counted.
 *
 * The month label comes from the response's own `from`, not the browser clock:
 * the server defines the month it aggregated (its zone), and a wall-clock read
 * here could disagree with the data under it — and would rot the visual
 * baseline on the 1st (Rule 23). Fixtures pin `from`.
 */
import { usd, usdWithEstimate } from '../../lib/currency';
import type { SubtitleSpendSummary } from '../../services/subtitleSpendService';
import { ActivitySectionShell } from './ActivitySectionShell';
import { ActivitySectionError } from './ActivityStates';

/** 「10 月」 from the summary's `from` (RFC3339 text, server zone); '' if unreadable. */
export function spendMonthLabel(summary: Pick<SubtitleSpendSummary, 'from'>): string {
  const month = Number.parseInt(summary.from.slice(5, 7), 10);
  return Number.isInteger(month) && month >= 1 && month <= 12 ? `${month} 月` : '';
}

/**
 * Whether the month has anything worth a card. A paid run, a free delivery, a
 * measured cache split, or rows the footnote must own all count; an untouched
 * month hides the section like every other zero-count section does.
 */
export function spendHasContent(summary: SubtitleSpendSummary): boolean {
  return (
    summary.translatedRuns > 0 ||
    summary.asrRuns > 0 ||
    summary.unpricedRuns > 0 ||
    summary.unroutedRuns > 0 ||
    summary.skippedDeliverCount > 0 ||
    summary.cacheMeasuredRuns > 0
  );
}

function LaneStat({
  label,
  runs,
  amount,
  testId,
}: {
  label: string;
  runs: number;
  amount: number;
  testId: string;
}) {
  return (
    <div className="flex flex-col gap-0.5" data-testid={testId}>
      <dt className="text-sm text-[var(--text-secondary)]">{label}</dt>
      <dd className="font-mono text-base font-semibold tabular-nums text-[var(--text-primary)]">
        {/* 0 runs is "nothing recorded", not "$0.00 spent". */}
        {runs > 0 ? `${runs} 次 · ${usd(amount)}` : '—'}
      </dd>
    </div>
  );
}

/** Prop-driven card (gallery fixtures render this directly). */
export function SpendCard({ summary }: { summary: SubtitleSpendSummary }) {
  const byModel = summary.byModel ?? [];
  const cacheMeasured = summary.cacheMeasuredRuns > 0 && summary.cacheSavedUsdEstimate != null;
  const footnote: string[] = [];
  if (summary.unpricedRuns > 0) footnote.push(`另有 ${summary.unpricedRuns} 次沒記到金額`);
  if (summary.unroutedRuns > 0) {
    footnote.push(
      `舊紀錄 ${summary.unroutedRuns} 次 · ${usd(summary.unroutedUsd)}，未分翻譯／語音辨識`
    );
  }

  return (
    <div
      data-testid="activity-spend-card"
      className="flex flex-col gap-4 rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] p-4"
    >
      {/* Paid lanes — side by side on the desktop (K5-D), stacked on a phone (K5-M). */}
      <dl className="flex flex-col gap-3 sm:flex-row sm:gap-8">
        <LaneStat
          label="翻譯"
          runs={summary.translatedRuns}
          amount={summary.translatedUsd}
          testId="activity-spend-translated"
        />
        <LaneStat
          label="語音辨識"
          runs={summary.asrRuns}
          amount={summary.asrUsd}
          testId="activity-spend-asr"
        />
      </dl>

      {/* What the free lanes avoided. Both are estimates and say so; the
          skipped line also wears ≈ when a runtime had to be assumed. */}
      <p className="flex flex-col gap-1 text-sm text-[var(--text-secondary)] sm:flex-row sm:gap-8">
        {summary.skippedDeliverCount > 0 && (
          <span data-testid="activity-spend-skipped">
            略過 {summary.skippedDeliverCount} 集，估算省下{' '}
            <span className="font-mono tabular-nums">
              {usdWithEstimate(summary.skippedSavedUsdEstimate, summary.skippedSavedRuntimeAssumed)}
            </span>
          </span>
        )}
        <span data-testid="activity-spend-cache">
          快取估算省下{' '}
          <span className="font-mono tabular-nums">
            {cacheMeasured ? usd(summary.cacheSavedUsdEstimate as number) : '—'}
          </span>
        </span>
      </p>

      {footnote.length > 0 && (
        <p data-testid="activity-spend-footnote" className="text-xs text-[var(--text-muted)]">
          {footnote.join('；')}
        </p>
      )}

      {byModel.length > 0 && (
        <div className="flex flex-col gap-1.5">
          <h3 className="text-xs text-[var(--text-muted)]">依模型</h3>
          <ul className="flex flex-col gap-1" data-testid="activity-spend-by-model">
            {byModel.map((m) => (
              <li
                key={m.modelId}
                className="flex items-center justify-between gap-3 text-sm"
                data-testid="activity-spend-model-row"
              >
                <span className="truncate font-mono text-[var(--text-secondary)]">{m.modelId}</span>
                <span className="shrink-0 font-mono tabular-nums text-[var(--text-primary)]">
                  {m.runs} 次 · {usd(m.usd)}
                </span>
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}

/**
 * The hub section around the card: fail-soft like the other four (its own
 * 無法載入 + 重試, the page renders on), hidden when the month is untouched.
 */
export function SpendSectionView({
  summary,
  failed,
  onRetry,
}: {
  summary?: SubtitleSpendSummary;
  failed: boolean;
  onRetry: () => void;
}) {
  if (failed) {
    return (
      <ActivitySectionShell title="本月 AI 花費">
        <ActivitySectionError onRetry={onRetry} testId="activity-spend-error" />
      </ActivitySectionShell>
    );
  }
  if (!summary || !spendHasContent(summary)) return null;
  return (
    <ActivitySectionShell
      title="本月 AI 花費"
      trailing={<span data-testid="activity-spend-month">{spendMonthLabel(summary)}</span>}
    >
      <SpendCard summary={summary} />
    </ActivitySectionShell>
  );
}
