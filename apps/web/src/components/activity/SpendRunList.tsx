// Design ref: ux-design.pen — no current screen frame; the AI 花費明細 page was specified in words (disc-2026-11-spend-history-c, ⚖️ Alexyu 2026-11-01: no new .pen frames) and built from the K5 spend card's tokens
/**
 * 逐筆列表 — every run of the month, newest first (disc-2026-11-spend-history-d
 * AC #2/#4). A table on the desktop, a card per run on a phone; the 處理方式
 * chips above it filter the list only (the summary and chart are the month).
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

export function SpendRunList({ items }: { items: RunHistoryItem[] }) {
  return (
    <>
      {/* Desktop: the seven-column table. */}
      <div className="hidden overflow-x-auto rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] md:block">
        <table className="w-full text-sm" data-testid="spend-history-table">
          <thead>
            <tr className="border-b border-[var(--border-subtle)] text-left text-xs text-[var(--text-muted)]">
              <th className="px-4 py-2.5 font-medium">時間</th>
              <th className="px-4 py-2.5 font-medium">影片</th>
              <th className="px-4 py-2.5 font-medium">處理方式</th>
              <th className="px-4 py-2.5 font-medium">用了哪些模型</th>
              <th className="px-4 py-2.5 text-right font-medium">實際</th>
              <th className="px-4 py-2.5 text-right font-medium">預估</th>
              <th className="px-4 py-2.5 font-medium">狀態</th>
            </tr>
          </thead>
          <tbody>
            {items.map((item) => (
              <tr
                key={item.id}
                data-testid="spend-history-row"
                className="border-b border-[var(--border-subtle)] align-top last:border-b-0"
              >
                <td className="whitespace-nowrap px-4 py-3 font-mono text-xs tabular-nums text-[var(--text-secondary)]">
                  {runTime(item.startedAt)}
                </td>
                <td className="px-4 py-3">
                  <Title item={item} />
                </td>
                <td className="px-4 py-3 text-[var(--text-secondary)]">{routeLabel(item)}</td>
                <td className="px-4 py-3">
                  <Models item={item} />
                </td>
                <td className="px-4 py-3 text-right">
                  <ActualAmount item={item} />
                </td>
                <td className="px-4 py-3 text-right">
                  <EstimateAmount item={item} />
                </td>
                <td className="px-4 py-3">
                  <Status item={item} />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Phone: one card per run — title + actual / time · 處理方式 / models /
          estimate + status (-c design, four lines). */}
      <ul className="flex flex-col gap-3 md:hidden" data-testid="spend-history-cards">
        {items.map((item) => (
          <li
            key={item.id}
            data-testid="spend-history-card"
            className="flex flex-col gap-1.5 rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] p-4 text-sm"
          >
            <div className="flex items-start justify-between gap-3">
              <span className="min-w-0 break-words font-medium">
                <Title item={item} />
              </span>
              <span className="shrink-0 text-right">
                <ActualAmount item={item} />
              </span>
            </div>
            <p className="text-xs text-[var(--text-secondary)]">
              <span className="font-mono tabular-nums">{runTime(item.startedAt)}</span>
              {' · '}
              {routeLabel(item)}
            </p>
            <Models item={item} />
            <div className="flex items-start justify-between gap-3 text-xs">
              <span className="text-[var(--text-secondary)]">
                預估 <EstimateAmount item={item} />
              </span>
              <Status item={item} />
            </div>
          </li>
        ))}
      </ul>
    </>
  );
}
