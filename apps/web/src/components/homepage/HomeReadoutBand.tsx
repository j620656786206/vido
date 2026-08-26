// Design ref: ux-design.pen Screen H1-D-v3 (k2Otv)
// Companion frames: H2-M-v3 (uGCAU) mobile 2×2 · H8-SPEC-v3 (iWUSV) 金額顯示規則
/**
 * Home v3 readout band (ux3-1-7 / epic ux3-home-v3, home-v3-identity-brief §2).
 *
 * The Operate answer to「我不在的時候你做了什麼？」— four dense cells, three
 * seconds, every cell a door. A BAND, not dashboard cards: one row on desktop,
 * 2×2 on mobile, 11px labels over mono digits.
 *
 * Honesty rules (brief §2/§5, inherited from the site-wide 固定詞彙):
 *  - a cell whose backend source degraded shows its label with NO number
 *    (量不到的格不顯示數字) — siblings render on.
 *  - 0 renders as 0 (0 是資訊); coverage 0/0 (first run) becomes the 開始掃描
 *    door instead of a dead readout.
 *  - the attention cell NEVER disappears: 0 failures renders 「一切正常」 —
 *    no bad news is itself the good news this product sells. Amber
 *    (--warning-text, 要求了但沒發生) is worn ONLY when failures > 0.
 *  - the spend trio is shown only when the backend sent one (absent ≠ $0);
 *    amounts fold per H8-SPEC-v3 (formatUsdShort) so no cell ever truncates.
 */
import { Link } from '@tanstack/react-router';
import { Activity, AlertTriangle, Captions, CheckCheck } from 'lucide-react';
import type { LucideIcon } from 'lucide-react';
import { useHomeSummary } from '../../hooks/useHomeSummary';
import { formatUsdShort } from '../../utils/formatUsdShort';
import { cn } from '../../lib/utils';
import type { AttentionCell } from '../../services/homeSummaryService';

/** The attention cell's readout line, or null when the cell is unmeasurable. */
function attentionText(cell: AttentionCell): { text: string; exception: boolean } {
  const spend =
    cell.spentUsd !== undefined && cell.budgetUsd !== undefined
      ? `${formatUsdShort(cell.spentUsd)}/${formatUsdShort(cell.budgetUsd)}`
      : null;

  if (cell.failedCount > 0) {
    return {
      text: spend ? `${cell.failedCount} 部失敗 · ${spend}` : `${cell.failedCount} 部失敗`,
      exception: true,
    };
  }
  // Live-batch spend is current, actionable information even with 0 failures;
  // a LAST run's spend next to 一切正常 would just be noise (historical, not
  // an exception) — so only live_batch surfaces here.
  if (spend && cell.spendSource === 'live_batch') {
    return { text: `一切正常 · ${spend}`, exception: false };
  }
  return { text: '一切正常', exception: false };
}

interface ReadoutCellProps {
  icon: LucideIcon;
  label: string;
  /**
   * The call-to-action half of the label, set apart in accent (H5-D-v3 draws
   * 「繁中字幕 · 開始掃描」 with only the second half gold — accent is 你在這裡
   * / the action, so a flat muted label hides that the cell is a door). The
   * token is `--accent-text`, NOT the design's literal `--accent-primary`:
   * styles.css records that #c9a24b measures 4.40:1 as text and was ratified
   * out of text roles, so this honours the intent at the AA-safe value.
   */
  action?: string;
  to: string;
  search?: Record<string, unknown>;
  ariaLabel: string;
  /** null = unmeasurable (cell shows label only, no number). */
  value: string | null;
  exception?: boolean;
  /**
   * Motion licence ③: work is happening RIGHT NOW behind this cell, so its
   * icon breathes. Only 進行中 with a non-zero count may pass this — a static
   * fact that moves is the time-axis version of a green badge on a job that
   * never ran. See the motion block in styles.css.
   */
  live?: boolean;
  testId: string;
}

function ReadoutCell({
  icon: Icon,
  label,
  action,
  to,
  search,
  ariaLabel,
  value,
  exception,
  live,
  testId,
}: ReadoutCellProps) {
  return (
    <Link
      to={to}
      search={search}
      aria-label={ariaLabel}
      data-testid={testId}
      // Every cell is a door and none of them looked like one until you were
      // already on it. active: gives the tap somewhere to land — on a phone
      // there is no hover, so without it the only feedback for「我按到了嗎」
      // is the route change, which is exactly when the app is busiest.
      className="flex min-h-[44px] flex-1 flex-col items-center justify-center gap-1 rounded-[var(--radius-md)] px-3 py-2 text-center transition-colors duration-[var(--motion-touch)] hover:bg-[var(--bg-tertiary)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)] active:bg-[var(--bg-tertiary)]"
    >
      <span className="flex items-center gap-1 text-[11px] text-[var(--text-muted)]">
        <Icon
          className={cn('h-3.5 w-3.5', live && 'motion-safe:animate-breathe')}
          aria-hidden="true"
        />
        {label}
        {action && (
          <span data-testid={`${testId}-action`} className="text-[var(--accent-text)]">
            · {action}
          </span>
        )}
      </span>
      {value !== null && (
        <span
          data-testid={`${testId}-value`}
          className={cn(
            'font-mono text-base font-semibold tabular-nums sm:text-lg',
            exception ? 'text-[var(--warning-text)]' : 'text-[var(--text-primary)]'
          )}
        >
          {value}
        </span>
      )}
    </Link>
  );
}

export function HomeReadoutBand() {
  const { data, isLoading, isError } = useHomeSummary();

  // A whole-request failure means NOTHING was measured: the band absents
  // itself entirely rather than rendering four empty label stubs. The page's
  // sections below carry on (fail-soft, F3).
  if (isError) return null;

  if (isLoading) {
    return (
      <div className="mx-auto w-full max-w-7xl px-4 sm:px-6">
        <div
          data-testid="home-readout-skeleton"
          aria-busy="true"
          aria-label="載入中"
          className="grid h-[76px] animate-pulse grid-cols-2 gap-px rounded-[var(--radius-lg)] bg-[var(--bg-secondary)] motion-reduce:animate-none md:h-[68px] md:grid-cols-4"
        />
      </div>
    );
  }
  if (!data) return null;

  const { coverage, processedToday, attention, inFlight } = data;
  const firstRun = coverage.status === 'ok' && coverage.total === 0;
  const attentionLine = attentionText(attention);

  return (
    <div className="mx-auto w-full max-w-7xl px-4 sm:px-6">
      <div
        data-testid="home-readout-band"
        role="group"
        aria-label="媒體庫讀數"
        className="grid grid-cols-2 divide-[var(--border-subtle)] rounded-[var(--radius-lg)] bg-[var(--bg-secondary)] py-1 max-md:divide-y md:flex md:divide-x"
      >
        {/* ① 繁中覆蓋率 — the product's reason to exist as ONE number. On a
            fresh library (0/0) the same cell becomes the 開始掃描 door. */}
        <ReadoutCell
          icon={Captions}
          label="繁中字幕"
          action={firstRun ? '開始掃描' : undefined}
          to={firstRun ? '/settings/scanner' : '/library'}
          ariaLabel={
            coverage.status === 'ok'
              ? firstRun
                ? '尚無媒體，前往掃描設定'
                : `繁中字幕覆蓋 ${coverage.covered} / ${coverage.total} 部，前往媒體庫`
              : '繁中字幕覆蓋率目前無法取得，前往媒體庫'
          }
          value={coverage.status === 'ok' ? `${coverage.covered}/${coverage.total}` : null}
          testId="readout-coverage"
        />
        {/* ② 不在時處理報告 */}
        <ReadoutCell
          icon={CheckCheck}
          label="今天處理"
          to="/activity"
          ariaLabel={
            processedToday.status === 'ok'
              ? `今天處理了 ${processedToday.count} 部，前往活動中心`
              : '今天處理數目前無法取得，前往活動中心'
          }
          value={processedToday.status === 'ok' ? `${processedToday.count} 部` : null}
          testId="readout-processed"
        />
        {/* ③ 需要注意 — the only cell allowed to wear amber, and only when
            failures > 0. It never vanishes: 一切正常 is a readout too. */}
        <ReadoutCell
          icon={AlertTriangle}
          label="需要注意"
          to="/activity"
          ariaLabel={
            attention.status === 'ok'
              ? attention.failedCount > 0
                ? `${attention.failedCount} 部失敗待處理，前往活動中心`
                : '一切正常，前往活動中心'
              : '例外狀態目前無法取得，前往活動中心'
          }
          value={attention.status === 'ok' ? attentionLine.text : null}
          exception={attention.status === 'ok' && attentionLine.exception}
          testId="readout-attention"
        />
        {/* ④ 正在進行中 — same counting path as the nav badge (/activity). */}
        <ReadoutCell
          icon={Activity}
          label="進行中"
          to="/activity"
          ariaLabel={
            inFlight.status === 'ok'
              ? `${inFlight.count} 個任務進行中，前往活動中心`
              : '進行中任務數目前無法取得，前往活動中心'
          }
          value={inFlight.status === 'ok' ? `${inFlight.count} 個任務` : null}
          // The page's one moving thing, and only while the count is real and
          // above zero. 0 個任務 sits perfectly still — that stillness is the
          // readout.
          live={inFlight.status === 'ok' && inFlight.count > 0}
          testId="readout-inflight"
        />
      </div>
    </div>
  );
}
