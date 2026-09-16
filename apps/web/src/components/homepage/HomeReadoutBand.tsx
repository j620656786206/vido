// Design ref: ux-design.pen Screen H1-D-v3 (k2Otv) · H2-M-v3 (uGCAU) · H8-SPEC-v3 (iWUSV)
/**
 * Home v3 readout band (ux3-1-7 / epic ux3-home-v3, home-v3-identity-brief §2).
 *
 * The Operate answer to「我不在的時候你做了什麼？」— four dense cells, three
 * seconds, every cell a door. A BAND, not dashboard cards: one row on desktop,
 * 2×2 on mobile, 11px labels over mono digits.
 *
 * ⚖️ The 11 is deliberate and ratified twice. dsr-7 first "corrected" it to the
 * .pen's $Type/Label/Size (12) and was overruled (Alexyu 2026-09-16): 11px is
 * what 44 other micro-labels across this app use, INCLUDING the poster badge
 * 40px below this band (PosterCardV2). Moving one of them makes the odd one
 * out, not a fix. Whether the design system should grow an 11px Micro rung is
 * tracked as disc-2026-09-11px-micro-label-not-on-type-scale — until it does,
 * do not "clean this up".
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
 *  - a whole-summary failure still renders one honest, actionable band. The
 *    endpoint cannot identify a trustworthy source or timestamp, so this state
 *    names neither; inventing either would break the contract it protects.
 */
import { Link } from '@tanstack/react-router';
import { Activity, AlertTriangle, Captions, CheckCheck } from 'lucide-react';
import type { LucideIcon } from 'lucide-react';
import { useHomeSummary } from '../../hooks/useHomeSummary';
import { formatUsdShort } from '../../utils/formatUsdShort';
import { cn } from '../../lib/utils';
import type { AttentionCell } from '../../services/homeSummaryService';

/**
 * The band's shell and its cell box, shared by the skeleton and the real band
 * so the two can never disagree about how tall the band is. Extracting them is
 * the fix for a 54px mobile layout jump — see the isLoading branch.
 *
 * ⚠️ The mobile dividers are NOT `divide-y`. Tailwind compiles divide-* to
 * `> :not(:last-child) { border-bottom }`, which assumes ONE axis. On a
 * 2-column grid that draws a rule under cells 1, 2 AND 3, so the third cell
 * gets a 179px line inside the LAST row with nothing under its neighbour — the
 * band ends in half a rule hanging off its own edge. Measured 1px/1px/1px/0px
 * at 390 and caught independently by both critique assessments (2026-08-27).
 * A 2×2 needs the cross drawn deliberately: a right edge on the left column, a
 * bottom edge on the top row. Desktop stays one row, so md:divide-x is correct.
 */
const BAND_SHELL =
  'grid grid-cols-2 divide-[var(--border-subtle)] rounded-[var(--radius-lg)] bg-[var(--bg-secondary)] py-1 max-md:[&>*:nth-child(-n+2)]:border-b max-md:[&>*:nth-child(odd)]:border-r max-md:[&>*]:border-[var(--border-subtle)] md:flex md:divide-x';

/**
 * One cell's box — everything except its content and its interaction.
 *
 * The height floor is NOT just the 44px touch target: below `lg` the 需要注意
 * cell can render two lines (failures + spend), and every other cell in its row
 * stretches to match. If only the live band knew that, the loading skeleton
 * would be a line short and the band would grow on mount — the 54px / 0.0832
 * CLS jump the skeleton branch exists to prevent. Putting the floor here means
 * skeleton and band agree by construction, not by two hand-matched strings.
 * The exact value is measured, not guessed: 84px is what a two-line cell comes
 * out at, verified by the CLS test in homepage-layout.spec.ts.
 */
const CELL_BOX =
  'flex min-h-[84px] flex-1 flex-col items-center justify-center px-3 py-2 text-center lg:min-h-[44px]';

/**
 * The attention cell's readout, as either ONE string or TWO halves.
 *
 * Only the failures-AND-spend case has halves worth breaking apart on a narrow
 * cell (H8-SPEC-v3 / ux3-1-7 AC #3). That case returns `parts` and NO `text`:
 * the joined desktop reading is assembled once, in the renderer, next to the
 * separator that produces it — a second copy here could silently disagree with
 * what is on screen and no assertion would notice.
 */
type AttentionReadout =
  /** One line: `value` is the whole reading. */
  | { exception: boolean; text: string; parts: null }
  /** Two halves: the renderer owns how they are joined, so there is no second
   *  copy of the joined string to fall out of sync with the DOM. */
  | { exception: true; text: null; parts: { failures: string; spend: string } };

function attentionText(cell: AttentionCell): AttentionReadout {
  const spend =
    cell.spentUsd !== undefined && cell.budgetUsd !== undefined
      ? `${formatUsdShort(cell.spentUsd)}/${formatUsdShort(cell.budgetUsd)}`
      : null;

  if (cell.failedCount > 0) {
    const failures = `${cell.failedCount} 部失敗`;
    // H8-SPEC-v3: only the failures-AND-spend case earns a second line. With
    // nothing above it (「若沒有失敗、只剩預算警示」) the amount rises to the
    // first line and the cell stays one line at every width.
    return spend
      ? { exception: true, text: null, parts: { failures, spend } }
      : { exception: true, text: failures, parts: null };
  }
  // Live-batch spend is current, actionable information even with 0 failures;
  // a LAST run's spend next to 一切正常 would just be noise (historical, not
  // an exception) — so only live_batch surfaces here.
  if (spend && cell.spendSource === 'live_batch') {
    return { exception: false, text: `一切正常 · ${spend}`, parts: null };
  }
  return { exception: false, text: '一切正常', parts: null };
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
  /** null = unmeasurable (cell shows label only, no number) — unless `valueParts` is set. */
  value: string | null;
  /**
   * A two-half readout, rendered as ONE DOM whose axis CSS picks: stacked while
   * the cell is narrow, one line with a 「·」 once it is wide enough. Stacking
   * holds until `lg`, NOT `md` — at `md` the band is already four cells across
   * (≈155px of content each, and by then the digits are `text-lg`), which is
   * marginally TIGHTER than the 2×2 phone cell this exists to escape. Measured
   * at 390/768/1024/1280 in homepage-layout.spec.ts.
   */
  valueParts?: { failures: string; spend: string } | null;
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
  valueParts,
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
      className={cn(
        CELL_BOX,
        'gap-1 rounded-[var(--radius-md)] transition-colors duration-[var(--motion-touch)] hover:bg-[var(--bg-tertiary)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)] active:bg-[var(--bg-tertiary)]'
      )}
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
      {(value !== null || valueParts) && (
        <span
          data-testid={`${testId}-value`}
          className={cn(
            'font-mono text-base font-semibold tabular-nums sm:text-lg',
            // Only a two-half readout needs a layout mode. The other three
            // cells render a bare string and keep the markup they always had.
            valueParts && 'flex flex-col items-center justify-center lg:flex-row',
            exception ? 'text-[var(--warning-text)]' : 'text-[var(--text-primary)]'
          )}
        >
          {valueParts ? (
            <>
              <span data-testid={`${testId}-failures`}>{valueParts.failures}</span>
              {/* Desktop-only punctuation: once the halves stack, a trailing
                  「·」 would point at the line below it. */}
              <span
                data-testid={`${testId}-separator`}
                aria-hidden="true"
                className="hidden lg:inline"
              >
                &nbsp;·&nbsp;
              </span>
              <span data-testid={`${testId}-spend`}>{valueParts.spend}</span>
            </>
          ) : (
            value
          )}
        </span>
      )}
    </Link>
  );
}

export function HomeReadoutBand() {
  const { data, isLoading, isError, isFetching, refetch } = useHomeSummary();

  // A whole-request failure means no value was measured, not that the readout
  // stopped mattering. Keep one compact band so the homepage remains a useful
  // operating surface: state, recovery, and a diagnostic destination all stay
  // present without inventing four unknown values or a false cause.
  if (isError) {
    return (
      <div className="mx-auto w-full max-w-7xl px-4 sm:px-6">
        <section
          data-testid="home-readout-error"
          role="status"
          aria-live="polite"
          aria-label="媒體庫讀數目前無法取得"
          className="flex flex-col gap-3 rounded-[var(--radius-lg)] bg-[var(--error-tint)] px-4 py-3 sm:flex-row sm:items-center sm:justify-between"
        >
          <div className="flex min-w-0 items-start gap-3">
            <AlertTriangle
              className="mt-0.5 h-5 w-5 shrink-0 text-[var(--error-text)]"
              aria-hidden="true"
            />
            <div className="min-w-0">
              <p className="text-sm font-medium text-[var(--error-text)]">媒體庫讀數目前無法取得</p>
              <p className="mt-0.5 text-sm text-[var(--text-secondary)]">
                請重新讀取；其他首頁內容仍可使用。
              </p>
            </div>
          </div>
          <div className="flex shrink-0 flex-wrap items-center gap-2">
            <button
              type="button"
              onClick={() => void refetch()}
              disabled={isFetching}
              data-testid="home-readout-retry"
              className="min-h-[44px] rounded-[var(--radius-md)] px-4 text-sm font-medium text-[var(--error-text)] transition-colors hover:bg-[var(--error)]/10 disabled:cursor-not-allowed disabled:opacity-60"
            >
              {isFetching ? '重新讀取中' : '重新讀取'}
            </button>
            <Link
              to="/activity"
              data-testid="home-readout-activity"
              className="flex min-h-[44px] items-center px-2 text-sm font-medium text-[var(--text-primary)] underline underline-offset-2 transition-colors hover:text-[var(--error-text)]"
            >
              查看活動中心
            </Link>
          </div>
        </section>
      </div>
    );
  }

  /**
   * The skeleton is built from the band's OWN shell and cell box, not from a
   * guessed height. `h-[76px] md:h-[68px]` was measured wrong in both
   * directions — real 73 on desktop and **130 on mobile**, a 54px jump on the
   * surface PRODUCT.md calls 同等重要, contributing 0.0832 to the page's CLS
   * (critique 2026-08-27 P1). Two hardcoded numbers cannot track a 2×2 that
   * grows with its own copy, so they are gone: the same container classes and
   * the same per-cell box mean the height is DERIVED and cannot drift again.
   * Only the text is replaced by bars.
   */
  if (isLoading) {
    return (
      <div className="mx-auto w-full max-w-7xl px-4 sm:px-6">
        <div
          data-testid="home-readout-skeleton"
          aria-busy="true"
          aria-label="載入中"
          className={cn(BAND_SHELL, 'animate-pulse motion-reduce:animate-none')}
        >
          {[0, 1, 2, 3].map((i) => (
            <div key={i} className={cn(CELL_BOX, 'gap-1')}>
              {/* The bars are REAL TEXT in the real type classes, painted out
                  with text-transparent over a tinted background. Fixed bar
                  heights (h-[11px] / h-5) were still 10–19px short, because a
                  line box is line-height tall, not font-size tall — the same
                  guess this branch was extracted to stop making. Borrowing the
                  type means the skeleton measures itself. */}
              <span className="flex items-center gap-1 text-[11px]">
                <span aria-hidden="true" className="h-3.5 w-3.5 rounded bg-[var(--bg-tertiary)]" />
                <span
                  aria-hidden="true"
                  className="rounded bg-[var(--bg-tertiary)] text-transparent"
                >
                  繁中字幕
                </span>
              </span>
              <span
                aria-hidden="true"
                className="rounded bg-[var(--bg-tertiary)] font-mono text-base font-semibold tabular-nums text-transparent sm:text-lg"
              >
                00/00
              </span>
            </div>
          ))}
        </div>
      </div>
    );
  }
  if (!data) return null;

  const { coverage, processedToday, attention, inFlight } = data;
  const firstRun = coverage.status === 'ok' && coverage.total === 0;
  /**
   * ⚖️ Alexyu 2026-08-27. 產生字幕 → (`/library?generate=true`) existed in
   * exactly TWO places, both inside a scan-complete toast that destroys itself
   * after ten seconds: ScanProgressCard.tsx:215 and ScanProgressSheet.tsx:132.
   * A grep of the app finds no third entry point. So the scan would announce
   *「N 部影片缺繁中字幕」 and then take away the only one-tap way to act on it,
   * seconds later — worst of all on a phone, where the toast cannot be paused.
   *
   * The door belongs on a surface that is always there. This cell already IS
   * that surface: it is the product's reason to exist as one number, and it
   * already carries an `action` half (H5-D-v3 draws 「繁中字幕 · 開始掃描」)
   * for the first-run case. Same mechanism, one more rung on the ladder.
   *
   * ⚠️ The NUMBER does not change, only the door. `coverage` counts TITLES
   * (movieCovered + seriesCovered, home_summary_service.go:157-167) while the
   * toast's missingSubtitleCount counts EPISODES too
   * (totalItemsIncludingEpisodes, ScanProgress.tsx:67). They are different
   * quantities; showing one under the other's label would be the kind of
   * almost-right number this product exists not to print.
   */
  // No `total > 0` guard: Covered ≤ Total by construction (the backend says so
  // at home_summary_service.go:99), so 0/0 already fails `covered < total` —
  // and `firstRun` wins the ternary below regardless. A redundant clause that
  // reads like a guard is worse than no clause: it invites the next reader to
  // trust a check that was never doing anything.
  const hasUncovered = coverage.status === 'ok' && coverage.covered < coverage.total;
  const attentionLine = attentionText(attention);

  return (
    <div className="mx-auto w-full max-w-7xl px-4 sm:px-6">
      <div
        data-testid="home-readout-band"
        role="group"
        aria-label="媒體庫讀數"
        className={BAND_SHELL}
      >
        {/* ① 繁中覆蓋率 — the product's reason to exist as ONE number, and the
            cell whose door depends on what the number says. The ladder:
              0/0        → 開始掃描 (there is nothing to cover yet)
              covered<total → 產生字幕 (the permanent home of the consent flow)
              全部覆蓋   → no action; the number is the whole message. */}
        <ReadoutCell
          icon={Captions}
          label="繁中字幕"
          action={firstRun ? '開始掃描' : hasUncovered ? '產生字幕' : undefined}
          to={firstRun ? '/settings/scanner' : '/library'}
          search={hasUncovered ? { generate: true } : undefined}
          ariaLabel={
            coverage.status === 'ok'
              ? firstRun
                ? '繁中字幕，尚無媒體，前往掃描設定'
                : hasUncovered
                  ? `繁中字幕，已覆蓋 ${coverage.covered} / ${coverage.total} 部，前往產生字幕`
                  : `繁中字幕，已覆蓋 ${coverage.covered} / ${coverage.total} 部，前往媒體庫`
              : '繁中字幕，覆蓋率目前無法取得，前往媒體庫'
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
              ? `今天處理 ${processedToday.count} 部，前往活動中心`
              : '今天處理，數目前無法取得，前往活動中心'
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
          // The spend figure rides in the label too. It is the first layer of
          // the thing this product sells (花費上限＋同意流程), and without it
          // here a screen-reader user is the ONE audience that never hears the
          // number — the aria-label replaces the cell's content, it does not
          // supplement it.
          ariaLabel={
            attention.status === 'ok'
              ? attentionLine.parts
                ? `需要注意，${attentionLine.parts.failures}待處理，已花費 ${attentionLine.parts.spend}，前往活動中心`
                : attention.failedCount > 0
                  ? `需要注意，${attention.failedCount} 部失敗待處理，前往活動中心`
                  : `需要注意，${attentionLine.text}，前往活動中心`
              : '需要注意，狀態目前無法取得，前往活動中心'
          }
          value={attention.status === 'ok' ? attentionLine.text : null}
          valueParts={attention.status === 'ok' ? attentionLine.parts : null}
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
              ? `進行中，${inFlight.count} 個任務，前往活動中心`
              : '進行中，任務數目前無法取得，前往活動中心'
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
