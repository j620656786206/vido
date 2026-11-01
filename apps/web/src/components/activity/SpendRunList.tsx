// Design ref: ux-design.pen — no current screen frame; the AI 花費明細 page was specified in words (disc-2026-11-spend-history-c, ⚖️ Alexyu 2026-11-01: no new .pen frames) and built from the K5 spend card's tokens
/**
 * 逐筆列表 — every run of the month, newest first (disc-2026-11-spend-history-d
 * AC #2/#4). One table at every width: eight columns on the desktop (花了多久
 * since disc-2026-11-spend-history-totals-and-duration); on a phone
 * (disc-2026-11-spend-history-mobile-table, ⚖️ Alexyu 2026-11-01) only
 * 影片／實際／預估／狀態, with time · 處理方式 and the models in small type under
 * the title. The 處理方式 chips above it filter the list only (the summary and
 * chart are the month).
 *
 * Honesty (PRODUCT.md 原則 1, -c design):
 *  - an amount nobody recorded reads 「沒記錄」 in --text-muted, never $0.00 —
 *    a real $0 (直接用片內繁中) still reads $0.00;
 *  - an actual more than 50% over its estimate wears ⚠ 比預估高 N%;
 *  - failed and cancelled runs are listed like any other, with what they cost.
 */
import { Check } from 'lucide-react';
import { usd } from '../../lib/currency';
import { formatLocalDateTime } from '../../utils/formatLocalDateTime';
import { formatRunDuration } from '../../utils/formatRunDuration';
import type { ChartCategory, RunHistoryItem } from '../../services/subtitleRunHistoryService';
import {
  CATEGORY_LABEL,
  CHART_CATEGORIES,
  mediaTitle,
  overEstimatePercent,
  routeLabel,
  runModels,
  runStatus,
} from './spendHistoryCopy';

/** 「11-01 13:33」 in the viewer's zone — the month is in the page header already. */
function runTime(iso: string): string {
  return formatLocalDateTime(iso).slice(5);
}

/** How long the run took; 「沒記錄」 when it has no end time. */
function Duration({ item }: { item: RunHistoryItem }) {
  const d = formatRunDuration(item.startedAt, item.completedAt);
  if (!d) return <NotRecorded testId="spend-run-duration-unrecorded" />;
  return <span data-testid="spend-run-duration">{d}</span>;
}

function NotRecorded({ testId }: { testId?: string }) {
  return (
    <span className="text-[var(--text-muted)]" data-testid={testId}>
      沒記錄
    </span>
  );
}

function ActualAmount({ item }: { item: RunHistoryItem }) {
  if (item.spentUsd == null) return <NotRecorded testId="spend-run-actual-unrecorded" />;
  const over = overEstimatePercent(item);
  return (
    <span className="inline-flex flex-wrap items-baseline justify-end gap-x-1.5">
      <span className="font-mono tabular-nums text-[var(--text-primary)]">
        {usd(item.spentUsd)}
      </span>
      {over !== undefined && (
        <span className="text-xs text-[var(--warning-text)]" data-testid="spend-run-over-estimate">
          ⚠ 比預估高 {over}%
        </span>
      )}
    </span>
  );
}

function EstimateAmount({ item }: { item: RunHistoryItem }) {
  if (item.estimateUsd == null) return <NotRecorded testId="spend-run-estimate-unrecorded" />;
  return (
    <span className="font-mono tabular-nums text-[var(--text-secondary)]">
      {usd(item.estimateUsd)}
    </span>
  );
}

function Title({ item }: { item: RunHistoryItem }) {
  const title = mediaTitle(item);
  return title ? (
    <span className="text-[var(--text-primary)]">{title}</span>
  ) : (
    <span className="text-[var(--text-muted)]">影片已刪除</span>
  );
}

function Models({ item }: { item: RunHistoryItem }) {
  const models = runModels(item);
  if (models.length === 0) return <span className="text-[var(--text-muted)]">—</span>;
  return (
    <span className="flex flex-col gap-0.5">
      {models.map((m) => (
        <span key={m} className="font-mono text-xs text-[var(--text-secondary)]">
          {m}
        </span>
      ))}
    </span>
  );
}

function Status({ item }: { item: RunHistoryItem }) {
  const s = runStatus(item);
  return (
    <span className="flex flex-col gap-0.5" data-testid="spend-run-status">
      <span className={s.tone}>{s.label}</span>
      {s.detail && <span className="text-xs text-[var(--text-muted)]">{s.detail}</span>}
    </span>
  );
}

export function SpendCategoryChips({
  value,
  onChange,
}: {
  value: ChartCategory | undefined;
  onChange: (next: ChartCategory | undefined) => void;
}) {
  const options: Array<{ key: string; cat: ChartCategory | undefined; label: string }> = [
    { key: 'all', cat: undefined, label: '全部' },
    ...CHART_CATEGORIES.map((cat) => ({ key: cat, cat, label: CATEGORY_LABEL[cat] })),
  ];
  return (
    <div className="flex flex-wrap gap-1.5" role="group" aria-label="處理方式">
      {options.map((o) => {
        const selected = value === o.cat;
        return (
          <button
            key={o.key}
            type="button"
            onClick={() => onChange(o.cat)}
            aria-pressed={selected}
            data-testid={`spend-history-chip-${o.key}`}
            className={`inline-flex min-h-[44px] items-center gap-1 rounded-full px-3 py-1.5 text-sm transition-colors ${
              selected
                ? 'border border-[var(--accent-primary)] bg-[var(--accent-primary)]/15 text-[var(--accent-text)]'
                : 'border border-transparent bg-[var(--bg-tertiary)] text-[var(--text-secondary)] hover:text-[var(--text-primary)]'
            }`}
          >
            {selected && <Check className="h-3.5 w-3.5" aria-hidden="true" />}
            {o.label}
          </button>
        );
      })}
    </div>
  );
}

/** Phone-only lines under the title: what the hidden columns would have said. */
function CompactDetails({ item }: { item: RunHistoryItem }) {
  const models = runModels(item);
  return (
    <span className="mt-1 flex flex-col gap-0.5 text-xs md:hidden" data-testid="spend-run-compact">
      <span className="text-[var(--text-secondary)]">
        <span className="font-mono tabular-nums">{runTime(item.startedAt)}</span>
        {' · '}
        {routeLabel(item)}
        {' · 花了 '}
        <Duration item={item} />
      </span>
      {models.length > 0 && (
        <span className="break-words text-[var(--text-muted)]">{models.join(' ＋ ')}</span>
      )}
    </span>
  );
}

/** Columns a phone does not draw — their content moves under the title. */
const DESKTOP_ONLY = 'hidden md:table-cell';

export function SpendRunList({ items }: { items: RunHistoryItem[] }) {
  return (
    <div className="overflow-x-auto rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)]">
      <table className="w-full text-sm" data-testid="spend-history-table">
        <thead>
          <tr className="border-b border-[var(--border-subtle)] text-left text-xs text-[var(--text-muted)]">
            <th className={`${DESKTOP_ONLY} px-4 py-2.5 font-medium`}>時間</th>
            <th className="px-3 py-2.5 font-medium md:px-4">影片</th>
            <th className={`${DESKTOP_ONLY} px-4 py-2.5 font-medium`}>處理方式</th>
            <th className={`${DESKTOP_ONLY} px-4 py-2.5 font-medium`}>用了哪些模型</th>
            <th className="px-2 py-2.5 text-right font-medium md:px-4">實際</th>
            <th className="px-2 py-2.5 text-right font-medium md:px-4">預估</th>
            <th className="px-3 py-2.5 font-medium md:px-4">狀態</th>
            <th className={`${DESKTOP_ONLY} whitespace-nowrap px-4 py-2.5 font-medium`}>
              花了多久
            </th>
          </tr>
        </thead>
        <tbody>
          {items.map((item) => (
            <tr
              key={item.id}
              data-testid="spend-history-row"
              className="border-b border-[var(--border-subtle)] align-top last:border-b-0"
            >
              <td
                className={`${DESKTOP_ONLY} whitespace-nowrap px-4 py-3 font-mono text-xs tabular-nums text-[var(--text-secondary)]`}
              >
                {runTime(item.startedAt)}
              </td>
              <td className="min-w-0 px-3 py-3 md:px-4">
                <span className="break-words">
                  <Title item={item} />
                </span>
                <CompactDetails item={item} />
              </td>
              <td className={`${DESKTOP_ONLY} px-4 py-3 text-[var(--text-secondary)]`}>
                {routeLabel(item)}
              </td>
              <td className={`${DESKTOP_ONLY} px-4 py-3`}>
                <Models item={item} />
              </td>
              <td className="px-2 py-3 text-right md:px-4">
                <ActualAmount item={item} />
              </td>
              <td className="px-2 py-3 text-right md:px-4">
                <EstimateAmount item={item} />
              </td>
              <td className="whitespace-nowrap px-3 py-3 md:whitespace-normal md:px-4">
                <Status item={item} />
              </td>
              <td
                className={`${DESKTOP_ONLY} whitespace-nowrap px-4 py-3 text-[var(--text-secondary)]`}
              >
                <Duration item={item} />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
