// Design ref: ux-design.pen — no current screen frame; the AI 花費明細 page was specified in words (disc-2026-11-spend-history-c, ⚖️ Alexyu 2026-11-01: no new .pen frames) and built from the K5 spend card's tokens
/**
 * 每日花費 — one stacked bar per day of the month, coloured by 處理方式
 * (disc-2026-11-spend-history-d AC #2). Inline SVG, no chart library.
 *
 * - The bars live in an SVG whose viewBox is stretched to the container
 *   (`preserveAspectRatio="none"`) — rects scale cleanly; every word (axis
 *   days, the scale, the day readout) is HTML so text never stretches.
 * - A day with no spend still gets a stub on the baseline: "nothing that day"
 *   is drawn, not left blank (-c design).
 * - Hovering a bar (desktop) or tapping it (phone) shows that day's total and
 *   each category's amount in the readout under the chart. A tap pins the day;
 *   hover only previews it. Bars are focusable buttons for the keyboard.
 */
import { useState } from 'react';
import { divideUsd, gtUsd, usd } from '../../lib/currency';
import type { RunHistoryDay } from '../../services/subtitleRunHistoryService';
import {
  CATEGORY_FILL,
  CATEGORY_LABEL,
  CATEGORY_SWATCH,
  CHART_CATEGORIES,
  dayLabel,
} from './spendHistoryCopy';

/** viewBox units: each day is SLOT wide; bars sit inside with a gap. */
const SLOT = 10;
const BAR = 6;
const HEIGHT = 100;
/** Head-room above the tallest bar. */
const PLOT = 94;
/** A non-zero amount is never drawn thinner than this (a $0.01 day stays visible). */
const MIN_SEGMENT = 1.5;
/** The stub a $0 day gets. */
const ZERO_STUB = 1;

function dayNumber(date: string): number {
  return Number(date.slice(8, 10));
}

/**
 * Which day numbers get an axis label: the 1st, every 5th, and the last day —
 * unless the last sits right next to a 5th (30 · 31 collide on a phone).
 */
function showAxisLabel(index: number, count: number, day: number): boolean {
  if (index === 0 || day % 5 === 0) return true;
  return index === count - 1 && day % 5 >= 3;
}

export function SpendDailyChart({ series }: { series: RunHistoryDay[] }) {
  const [hovered, setHovered] = useState<string | null>(null);
  const [pinned, setPinned] = useState<string | null>(null);
  const activeDate = hovered ?? pinned;
  const active = series.find((d) => d.date === activeDate);

  const max = series.reduce<number>((m, d) => (gtUsd(d.spentUsd, m) ? d.spentUsd : m), 0);
  const scale = (v: number) => (max > 0 ? divideUsd(v, max) * PLOT : 0);
  const width = Math.max(series.length, 1) * SLOT;

  return (
    <div className="flex flex-col gap-3" data-testid="spend-history-chart">
      <div className="flex items-baseline justify-between text-xs text-[var(--text-muted)]">
        <span>每日花費</span>
        {max > 0 && (
          <span className="font-mono tabular-nums" data-testid="spend-history-chart-max">
            最高 {usd(max)}
          </span>
        )}
      </div>

      <svg
        viewBox={`0 0 ${width} ${HEIGHT}`}
        preserveAspectRatio="none"
        className="h-32 w-full sm:h-40"
        role="group"
        aria-label="每日花費長條圖"
        onMouseLeave={() => setHovered(null)}
      >
        <line
          x1={0}
          y1={HEIGHT - 0.5}
          x2={width}
          y2={HEIGHT - 0.5}
          vectorEffect="non-scaling-stroke"
          className="stroke-[var(--border-subtle)]"
        />
        {series.map((day, i) => {
          const x = i * SLOT;
          const isActive = day.date === activeDate;
          let y = HEIGHT;
          const segments = CHART_CATEGORIES.flatMap((cat) => {
            const amount = day.byCategory[cat] ?? 0;
            if (!gtUsd(amount, 0)) return [];
            const h = Math.max(scale(amount), MIN_SEGMENT);
            y -= h;
            return [
              <rect
                key={cat}
                x={x + (SLOT - BAR) / 2}
                y={y}
                width={BAR}
                height={h}
                className={CATEGORY_FILL[cat]}
                data-category={cat}
              />,
            ];
          });
          return (
            <g key={day.date} data-testid="spend-history-chart-day" data-date={day.date}>
              {isActive && (
                <rect
                  x={x}
                  y={0}
                  width={SLOT}
                  height={HEIGHT}
                  className="fill-[var(--bg-tertiary)]"
                />
              )}
              {segments.length > 0 ? (
                segments
              ) : (
                <rect
                  x={x + (SLOT - BAR) / 2}
                  y={HEIGHT - ZERO_STUB}
                  width={BAR}
                  height={ZERO_STUB}
                  className="fill-[var(--border-subtle)]"
                  data-testid="spend-history-chart-zero"
                />
              )}
              {/* The hit target: the whole day column, so a thin bar is still easy to hit. */}
              <rect
                x={x}
                y={0}
                width={SLOT}
                height={HEIGHT}
                fill="transparent"
                role="button"
                tabIndex={0}
                aria-pressed={pinned === day.date}
                aria-label={`${dayLabel(day.date)}，${usd(day.spentUsd)}，${day.runs} 次`}
                data-testid="spend-history-chart-hit"
                className="cursor-pointer outline-none focus-visible:fill-[var(--accent-tint)]"
                onMouseEnter={() => setHovered(day.date)}
                onFocus={() => setHovered(day.date)}
                onBlur={() => setHovered(null)}
                onClick={() => setPinned((p) => (p === day.date ? null : day.date))}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' || e.key === ' ') {
                    e.preventDefault();
                    setPinned((p) => (p === day.date ? null : day.date));
                  }
                }}
              />
            </g>
          );
        })}
      </svg>

      {/* Axis days — HTML on the same N-column grid, so they line up with the bars. */}
      <div
        className="-mt-2 grid font-mono text-xs tabular-nums text-[var(--text-muted)]"
        style={{ gridTemplateColumns: `repeat(${Math.max(series.length, 1)}, minmax(0, 1fr))` }}
        aria-hidden="true"
      >
        {series.map((day, i) => (
          <span key={day.date} className="overflow-visible whitespace-nowrap text-center">
            {showAxisLabel(i, series.length, dayNumber(day.date)) ? dayNumber(day.date) : ''}
          </span>
        ))}
      </div>

      {/* The day readout: hover (desktop) or tap (phone). A fixed min-height so
          the page does not jump as the pointer crosses the bars. */}
      <div
        className="min-h-[52px] rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-3 py-2 text-sm"
        data-testid="spend-history-chart-readout"
        aria-live="polite"
      >
        {active ? (
          <div className="flex flex-col gap-1">
            <p className="text-[var(--text-primary)]">
              {dayLabel(active.date)}
              <span className="text-[var(--text-secondary)]"> · 合計 </span>
              <span className="font-mono font-semibold tabular-nums">{usd(active.spentUsd)}</span>
              <span className="text-[var(--text-secondary)]"> · {active.runs} 次</span>
            </p>
            {gtUsd(active.spentUsd, 0) ? (
              <ul className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-[var(--text-secondary)]">
                {CHART_CATEGORIES.filter((c) => gtUsd(active.byCategory[c] ?? 0, 0)).map((c) => (
                  <li key={c} className="flex items-center gap-1.5">
                    <span
                      className={`h-2 w-2 rounded-sm ${CATEGORY_SWATCH[c]}`}
                      aria-hidden="true"
                    />
                    {CATEGORY_LABEL[c]}
                    <span className="font-mono tabular-nums text-[var(--text-primary)]">
                      {usd(active.byCategory[c])}
                    </span>
                  </li>
                ))}
              </ul>
            ) : (
              <p className="text-xs text-[var(--text-muted)]">這天沒有花費</p>
            )}
          </div>
        ) : (
          <p className="text-xs text-[var(--text-muted)]">滑過或點一下長條，看當天各類花費</p>
        )}
      </div>

      <ul
        className="flex flex-wrap gap-x-4 gap-y-1.5 text-xs text-[var(--text-secondary)]"
        data-testid="spend-history-chart-legend"
      >
        {CHART_CATEGORIES.map((c) => (
          <li key={c} className="flex items-center gap-1.5">
            <span className={`h-2.5 w-2.5 rounded-sm ${CATEGORY_SWATCH[c]}`} aria-hidden="true" />
            {CATEGORY_LABEL[c]}
          </li>
        ))}
      </ul>
    </div>
  );
}
